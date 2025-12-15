package utils

import (
	"fmt"

	api "github.com/wso2/api-platform/gateway/gateway-controller/pkg/api/generated"
	"github.com/wso2/api-platform/gateway/gateway-controller/pkg/config"
	"github.com/wso2/api-platform/gateway/gateway-controller/pkg/constants"
	"github.com/wso2/api-platform/gateway/gateway-controller/pkg/storage"
	"gopkg.in/yaml.v3"
)

type LLMProviderTransformer struct {
	store        *storage.ConfigStore
	routerConfig *config.RouterConfig
}

func NewLLMProviderTransformer(store *storage.ConfigStore, routerConfig *config.RouterConfig) *LLMProviderTransformer {
	return &LLMProviderTransformer{store: store, routerConfig: routerConfig}
}

func (t *LLMProviderTransformer) Transform(input any, output *api.APIConfiguration) (*api.APIConfiguration, error) {
	switch v := input.(type) {
	case *api.LLMProviderConfiguration:
		return t.transformProvider(v, output)
	case *api.LLMProxyConfiguration:
		return t.transformProxy(v, output)
	default:
		return nil, fmt.Errorf("invalid input type: expected *api.LLMProviderConfiguration or *api.LLMProxyConfiguration")
	}
}

func (t *LLMProviderTransformer) transformProxy(proxy *api.LLMProxyConfiguration,
	output *api.APIConfiguration) (*api.APIConfiguration, error) {
	// @TODO: Step 1) Configure token based rate-limiting policy based on template configs
	provider := t.store.GetByKindAndHandle(string(api.LlmProvider), proxy.Spec.Provider)
	if provider == nil {
		return nil, fmt.Errorf("failed to retrieve provider by id '%s'", proxy.Spec.Provider)
	}

	output.Kind = api.RestApi
	output.ApiVersion = api.GatewayApiPlatformWso2Comv1alpha1

	spec := api.APIConfigData{}
	spec.DisplayName = proxy.Spec.DisplayName
	spec.Version = proxy.Spec.Version
	spec.Context = constants.BASE_PATH
	if proxy.Spec.Context != nil {
		spec.Context = *proxy.Spec.Context
	}

	// Step 2) Upstreams: map provider.Spec.Upstreams to api.Upstreams
	// Map provider upstream and vhost to API main upstream and vhost
	providerData, err := provider.Configuration.Spec.AsAPIConfigData()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provider spec '%s'", proxy.Spec.DisplayName)
	}
	// If provider does not have vhost for main, fall back to gateway configs
	effectiveVhost := providerData.Vhosts.Main
	if effectiveVhost == "" {
		effectiveVhost = t.routerConfig.VHosts.Main.Default
	}
	upstream := constants.HTTP + effectiveVhost + provider.GetContext()
	spec.Upstream.Main = api.Upstream{
		Url: &upstream,
	}
	if proxy.Spec.Vhost != nil {
		spec.Vhosts = &struct {
			Main    string  `json:"main" yaml:"main"`
			Sandbox *string `json:"sandbox,omitempty" yaml:"sandbox,omitempty"`
		}{
			Main: *proxy.Spec.Vhost,
		}
	}

	var ops []api.Operation
	// Add catch-all operation '/' to allow all requests
	// TODO: Add support for custom OpenAPI specification to override this behaviour
	ops = append(ops, api.Operation{Method: constants.WILD_CARD, Path: constants.BASE_PATH + constants.WILD_CARD})

	spec.Operations = ops

	// Step 5) Attach policies from proxy.Spec.Policies to matching operations
	if proxy.Spec.Policies != nil {
		// Policies are now a simple array of LLMPolicy
		for _, llmPol := range *proxy.Spec.Policies {
			// For each path entry in the policy
			for _, pathEntry := range llmPol.Paths {
				// For each method in the path entry
				for _, method := range pathEntry.Methods {
					// Find matching operations (same path and same method)
					found := false
					for i := range spec.Operations {
						op := &spec.Operations[i]
						if op.Path == pathEntry.Path && string(op.Method) == string(method) {
							// Convert LLMPolicy to API Policy using path-specific params
							pol := api.Policy{
								Name:    llmPol.Name,
								Version: llmPol.Version,
								Params:  &pathEntry.Params,
							}
							if op.Policies == nil {
								op.Policies = &[]api.Policy{pol}
							} else {
								// Append to existing slice
								existing := *op.Policies
								existing = append(existing, pol)
								op.Policies = &existing
							}
							found = true
						}
					}

					// If no matching operation was found, create a new one
					// TODO: This should not be applied in upcoming feature providing custom OpenAPI to override
					if !found {
						pol := api.Policy{
							Name:    llmPol.Name,
							Version: llmPol.Version,
							Params:  &pathEntry.Params,
						}
						newOp := api.Operation{
							Path:     pathEntry.Path,
							Method:   api.OperationMethod(method),
							Policies: &[]api.Policy{pol},
						}
						spec.Operations = append(spec.Operations, newOp)
					}
				}
			}
		}
	}

	// finalize output
	var specUnion api.APIConfiguration_Spec
	if err := specUnion.FromAPIConfigData(spec); err != nil {
		return nil, err
	}
	output.Spec = specUnion
	return output, nil
}

func (t *LLMProviderTransformer) transformProvider(provider *api.LLMProviderConfiguration,
	output *api.APIConfiguration) (*api.APIConfiguration, error) {
	// @TODO: Step 1) Configure token based rate-limiting policy based on template configs
	_, err := t.store.GetTemplateByHandle(provider.Spec.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve template '%s': %w", provider.Spec.Template, err)
	}

	output.Kind = api.RestApi
	output.ApiVersion = api.GatewayApiPlatformWso2Comv1alpha1
	output.Metadata = provider.Metadata

	spec := api.APIConfigData{}
	spec.DisplayName = provider.Spec.DisplayName
	spec.Version = provider.Spec.Version
	spec.Context = constants.BASE_PATH
	if provider.Spec.Context != nil {
		spec.Context = *provider.Spec.Context
	}

	// Step 2) Upstreams: map provider.Spec.Upstreams to api.Upstreams
	// Map provider upstream and vhost to API main upstream and vhost
	spec.Upstream.Main = api.Upstream{
		Url: provider.Spec.Upstream.Url,
	}
	if provider.Spec.Vhost != nil {
		spec.Vhosts = &struct {
			Main    string  `json:"main" yaml:"main"`
			Sandbox *string `json:"sandbox,omitempty" yaml:"sandbox,omitempty"`
		}{
			Main: *provider.Spec.Vhost,
		}
	}

	// Step 3) Map upstream auth to corresponding api policy
	upstream := provider.Spec.Upstream
	if upstream.Auth != nil {
		switch upstream.Auth.Type {
		case api.LLMProviderConfigDataUpstreamAuthTypeApiKey:
			// Add API Key auth policy at API level
			params, err := GetUpstreamAuthApikeyPolicyParams(*upstream.Auth.Header, *upstream.Auth.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to build upstream auth params: %w", err)
			}
			mh := api.Policy{
				Name:    constants.UPSTREAM_AUTH_APIKEY_POLICY_NAME,
				Version: constants.UPSTREAM_AUTH_APIKEY_POLICY_VERSION, Params: &params}
			spec.Policies = &[]api.Policy{mh}
		default:
			return nil, fmt.Errorf("unsupported upstream auth type: %s", upstream.Auth.Type)
		}
	}

	// Step 4) Apply access control
	mode := provider.Spec.AccessControl.Mode
	var exceptions []api.RouteException
	if provider.Spec.AccessControl.Exceptions != nil {
		exceptions = *provider.Spec.AccessControl.Exceptions
	}

	var ops []api.Operation
	switch mode {
	case api.AllowAll:
		// for each exception, add an operation and attach a Response policy that returns 404
		for _, ex := range exceptions {
			// for each declared method on the exception
			for _, m := range ex.Methods {
				op := api.Operation{Method: api.OperationMethod(m), Path: ex.Path}

				// Build Respond policy params as requested
				var policyParams map[string]interface{}
				if err := yaml.Unmarshal([]byte(constants.ACCESS_CONTROL_DENY_POLICY_PARAMS), &policyParams); err != nil {
					return nil, err
				}
				pol := api.Policy{
					Name:    constants.ACCESS_CONTROL_DENY_POLICY_NAME,
					Version: constants.ACCESS_CONTROL_DENY_POLICY_VERSION, Params: &policyParams}
				op.Policies = &[]api.Policy{pol}
				ops = append(ops, op)
			}
		}

		// add catch-all operation '/'
		ops = append(ops, api.Operation{Method: constants.WILD_CARD, Path: constants.BASE_PATH + constants.WILD_CARD})

	case api.DenyAll:
		// Only include exception operations (allowed paths)
		for _, ex := range exceptions {
			for _, m := range ex.Methods {
				op := api.Operation{Method: api.OperationMethod(m), Path: ex.Path}
				ops = append(ops, op)
			}
		}

	default:
		return nil, fmt.Errorf("unsupported access control mode: %s", mode)
	}

	spec.Operations = ops

	// Step 5) Attach policies from provider.Spec.Policies to matching operations
	if provider.Spec.Policies != nil {
		// Policies are now a simple array of LLMPolicy
		for _, llmPol := range *provider.Spec.Policies {
			// For each path entry in the policy
			for _, pathEntry := range llmPol.Paths {
				// For each method in the path entry
				for _, method := range pathEntry.Methods {
					// Find matching operations (same path and same method)
					found := false
					for i := range spec.Operations {
						op := &spec.Operations[i]
						if op.Path == pathEntry.Path && string(op.Method) == string(method) {
							// Convert LLMPolicy to API Policy using path-specific params
							pol := api.Policy{
								Name:    llmPol.Name,
								Version: llmPol.Version,
								Params:  &pathEntry.Params,
							}
							if op.Policies == nil {
								op.Policies = &[]api.Policy{pol}
							} else {
								// Append to existing slice
								existing := *op.Policies
								existing = append(existing, pol)
								op.Policies = &existing
							}
							found = true
						}
					}

					// If no matching operation was found, create a new one only for access control allow_all mode
					if provider.Spec.AccessControl.Mode == api.AllowAll && !found {
						pol := api.Policy{
							Name:    llmPol.Name,
							Version: llmPol.Version,
							Params:  &pathEntry.Params,
						}
						newOp := api.Operation{
							Path:     pathEntry.Path,
							Method:   api.OperationMethod(method),
							Policies: &[]api.Policy{pol},
						}
						spec.Operations = append(spec.Operations, newOp)
					}
				}
			}
		}
	}

	// finalize output
	var specUnion api.APIConfiguration_Spec
	if err := specUnion.FromAPIConfigData(spec); err != nil {
		return nil, err
	}
	output.Spec = specUnion
	return output, nil
}

// GetUpstreamAuthApikeyPolicyParams renders the policy params with given header and value
func GetUpstreamAuthApikeyPolicyParams(header, value string) (map[string]interface{}, error) {
	rendered := fmt.Sprintf(constants.UPSTREAM_AUTH_APIKEY_POLICY_PARAMS, header, value)
	var m map[string]interface{}
	if err := yaml.Unmarshal([]byte(rendered), &m); err != nil {
		return nil, err
	}
	return m, nil
}
