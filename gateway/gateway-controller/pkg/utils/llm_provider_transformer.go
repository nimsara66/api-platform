package utils

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v3"

	api "github.com/wso2/api-platform/gateway/gateway-controller/pkg/api/generated"
	"github.com/wso2/api-platform/gateway/gateway-controller/pkg/storage"
)

// LLMProviderTransformer implements Transformer for LLM Provider -> APIConfiguration
// Responsibilities:
// 1) Retrieve referenced template from in-memory store
// 2) Convert template's OpenAPI spec to base APIConfiguration (name, version, context, operations)
// 3) Merge LLM Provider specifics (upstreams, access control, policies) into APIConfiguration
// 4) Return final APIConfiguration for deployment
//
// Notes:
// - We reuse the high-level logic from JS convertOpenAPIToWSO2YAML, but build Go structs instead of YAML
// - Context derivation: use provider name/version to build a stable context if not provided by servers
// - Operations: extracted from OpenAPI.paths[*][method]
// - Upstreams: from provider config
// - Access control: filter operations by allow_all / deny_all with exceptions
// - Policies: attach to matching operations by (path, methods)

type LLMProviderTransformer struct {
	store *storage.ConfigStore
}

func NewLLMProviderTransformer(store *storage.ConfigStore) *LLMProviderTransformer {
	return &LLMProviderTransformer{store: store}
}

func (t *LLMProviderTransformer) Transform(input any, output *api.APIConfiguration) *api.APIConfiguration {
	providerCfg, ok := input.(*api.LLMProviderConfiguration)
	if !ok || providerCfg == nil {
		return nil
	}

	// provider.Spec is a pointer to LLMProviderSpec
	if providerCfg.Spec == nil {
		return nil
	}

	provider := providerCfg.Spec

	// 1) Retrieve the referenced template from in-memory
	tmpl, err := t.store.GetTemplateByName(provider.Template)
	if err != nil {
		return nil
	}

	// 2) Convert template's OpenAPI spec into a base APIConfiguration
	apiCfg, err := buildAPIConfigFromOpenAPI(
		tmpl.Configuration.Data.Openapi,
		provider.Name,
		provider.Version,
		provider.Name, // use name as seed for context derivation if servers missing
	)
	if err != nil {
		return nil
	}

	// 3) Merge LLM Provider specifics
	// 3a) Set upstreams from provider
	apiCfg.Spec.Upstreams = make([]api.Upstream, 0, len(provider.Upstreams))
	for _, u := range provider.Upstreams {
		apiCfg.Spec.Upstreams = append(apiCfg.Spec.Upstreams, api.Upstream{Url: u.Url})
	}

	// 3b) Apply access control (filter operations)
	applyAccessControl(provider.AccessControl, &apiCfg.Spec.Operations)

	// 3c) Attach policies from provider to relevant operations
	attachPolicies(provider.Policies, &apiCfg.Spec.Operations)

	return apiCfg
}

// buildAPIConfigFromOpenAPI parses an OpenAPI (YAML or JSON) and constructs an APIConfiguration
// Similar to convertOpenAPIToWSO2YAML but returns Go structs and derives context sensibly.
func buildAPIConfigFromOpenAPI(openapiSpec string, userProvidedName, userProvidedVersion, contextSeed string) (*api.APIConfiguration, error) {
	// Parse YAML (or JSON-as-YAML)
	var spec map[string]interface{}
	if err := yaml.Unmarshal([]byte(openapiSpec), &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI: %w", err)
	}

	// Extract info fields
	info := getMap(spec, "info")
	apiName := firstNonEmpty(userProvidedName, getString(info, "title"), "Untitled API")
	apiVersion := firstNonEmpty(userProvidedVersion, getString(info, "version"), "1.0.0")
	apiDescription := getString(info, "description")

	// Extract operations from paths
	ops := extractOperations(spec)

	// Derive context from servers[0].url or from title/name
	contextPath := deriveContext(spec, contextSeed)

	cfg := &api.APIConfiguration{
		Version: api.APIConfigurationVersion("api-platform.wso2.com/v1"),
		Kind:    api.APIConfigurationKind("http/rest"),
		Spec: api.APIConfigData{
			Name:       apiName,
			Version:    apiVersion,
			Context:    contextPath,
			Operations: ops,
			Upstreams:  []api.Upstream{},
			Policies:   nil,
		},
	}

	// Attach description as a policy-friendly metadata if needed in future; skip for now to keep abstract
	_ = apiDescription

	return cfg, nil
}

// extractOperations walks spec.paths and gathers (method, path) pairs
func extractOperations(spec map[string]interface{}) []api.Operation {
	paths := getMap(spec, "paths")
	if len(paths) == 0 {
		return nil
	}
	methods := []string{"get", "post", "put", "delete", "patch", "options", "head", "trace"}
	ops := make([]api.Operation, 0)
	for pathKey, raw := range paths {
		pmap, _ := raw.(map[string]interface{})
		if pmap == nil {
			continue
		}
		for _, m := range methods {
			if _, ok := pmap[m]; ok {
				ops = append(ops, api.Operation{
					Method: api.OperationMethod(strings.ToUpper(m)),
					Path:   pathKey,
				})
			}
		}
	}
	// stable order for deterministic results
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path == ops[j].Path {
			return ops[i].Method < ops[j].Method
		}
		return ops[i].Path < ops[j].Path
	})
	return ops
}

// deriveContext determines a base context path from servers[0].url or a cleaned name seed
func deriveContext(spec map[string]interface{}, seed string) string {
	servers := getSlice(spec, "servers")
	if len(servers) > 0 {
		first := toMap(servers[0])
		if u := getString(first, "url"); u != "" {
			// try to parse as absolute URL to reuse path
			if parsed, err := url.Parse(u); err == nil && parsed.Path != "" {
				return ensureContext(parsed.Path)
			}
			// fallback if url is a path-only
			if strings.HasPrefix(u, "/") {
				return ensureContext(u)
			}
		}
	}
	// fallback: derive from seed
	clean := strings.ToLower(strings.TrimSpace(seed))
	clean = strings.ReplaceAll(clean, " ", "-")
	clean = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return -1
	}, clean)
	if clean == "" {
		clean = "api"
	}
	return ensureContext("/" + clean)
}

func ensureContext(p string) string {
	if p == "" {
		return "/api"
	}
	// must start with /
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	// remove trailing slash (except root)
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimRight(p, "/")
	}
	return p
}

// applyAccessControl filters operations based on allow_all/deny_all and exceptions
func applyAccessControl(ac *api.LLMAccessControl, ops *[]api.Operation) {
	if ac == nil || ac.Mode == nil {
		return
	}
	mode := string(*ac.Mode)
	if ac.Exceptions == nil || len(*ac.Exceptions) == 0 {
		// no exceptions; allow_all keeps all, deny_all removes all
		if mode == "deny_all" {
			*ops = nil
		}
		return
	}
	exceptions := *ac.Exceptions

	keep := make([]api.Operation, 0, len(*ops))
	switch mode {
	case "deny_all":
		// Keep only operations matching exceptions
		for _, op := range *ops {
			if matchesAnyException(op, exceptions) {
				keep = append(keep, op)
			}
		}
		*ops = keep
	case "allow_all":
		// Keep all except those matching exceptions
		for _, op := range *ops {
			if !matchesAnyException(op, exceptions) {
				keep = append(keep, op)
			}
		}
		*ops = keep
	}
}

func matchesAnyException(op api.Operation, exceptions []api.LLMAccessException) bool {
	for _, ex := range exceptions {
		if pathMatches(op.Path, ex.Path) && methodMatches(op.Method, ex.Methods) {
			return true
		}
	}
	return false
}

func pathMatches(opPath, exPath string) bool {
	// simple prefix or exact match for now; can be extended to proper path templating
	if exPath == opPath {
		return true
	}
	if strings.HasSuffix(exPath, "/*") {
		prefix := strings.TrimSuffix(exPath, "/*")
		return strings.HasPrefix(opPath, prefix)
	}
	return false
}

func methodMatches(opMethod api.OperationMethod, methods []api.LLMAccessExceptionMethods) bool {
	if len(methods) == 0 {
		return true
	}
	m := string(opMethod)
	for _, mm := range methods {
		if strings.EqualFold(m, string(mm)) {
			return true
		}
	}
	return false
}

// attachPolicies maps provider policies onto API operations by path+methods
func attachPolicies(p *api.LLMPolicies, ops *[]api.Operation) {
	if p == nil || ops == nil || len(*ops) == 0 {
		return
	}
	// BudgetControl policies
	if p.BudgetControl != nil {
		for _, pol := range *p.BudgetControl {
			attachPolicyToOps("budget-control", "1.0.0", pol.Path, pol.Methods, nil, ops)
		}
	}
	// PII policies
	if p.PII != nil {
		for _, pol := range *p.PII {
			params := map[string]interface{}{}
			for k, v := range pol.Params {
				params[k] = v.Value
			}
			attachPolicyToOps("pii", "1.0.0", pol.Path, pol.Methods, &params, ops)
		}
	}
	// SemanticPromptGuardrail policies
	if p.SemanticPromptGuardrail != nil {
		for _, pol := range *p.SemanticPromptGuardrail {
			params := map[string]interface{}{}
			for k, v := range pol.Params {
				params[k] = v.Value
			}
			attachPolicyToOps("semantic-guardrail", "1.0.0", pol.Path, pol.Methods, &params, ops)
		}
	}
}

func attachPolicyToOps(name, version, path string, methods []string, params *map[string]interface{}, ops *[]api.Operation) {
	for i := range *ops {
		op := &(*ops)[i]
		if !pathMatches(op.Path, path) {
			continue
		}
		if len(methods) > 0 && !stringInSlice(string(op.Method), methods) {
			continue
		}
		// append policy
		if op.Policies == nil {
			op.Policies = &[]api.Policy{}
		}
		pols := append(*op.Policies, api.Policy{
			Name:    name,
			Version: version,
			Params:  params,
		})
		op.Policies = &pols
	}
}

// helpers
func stringInSlice(s string, list []string) bool {
	for _, v := range list {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if m == nil {
		return nil
	}
	v, _ := m[key].(map[string]interface{})
	return v
}

func getSlice(m map[string]interface{}, key string) []interface{} {
	if m == nil {
		return nil
	}
	v, _ := m[key].([]interface{})
	return v
}

func toMap(v interface{}) map[string]interface{} {
	m, _ := v.(map[string]interface{})
	return m
}

func getString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// Validate that the provider input is adequate before transformation
func validateProviderForTransform(p *api.LLMProviderConfiguration) error {
	if p == nil {
		return errors.New("provider cannot be nil")
	}
	if p.Spec == nil || strings.TrimSpace(p.Spec.Name) == "" || strings.TrimSpace(p.Spec.Version) == "" {
		return errors.New("provider name and version are required")
	}
	if p.Spec == nil || strings.TrimSpace(p.Spec.Template) == "" {
		return errors.New("provider template is required")
	}
	return nil
}
