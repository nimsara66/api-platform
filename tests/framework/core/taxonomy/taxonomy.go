/*
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package taxonomy

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/cucumber/gherkin/go/v26"
	"github.com/cucumber/messages/go/v21"
	"gopkg.in/yaml.v3"
)

// Map is the closed capability and feature vocabulary used by suite annotations.
type Map struct {
	Capabilities map[string]Capability `yaml:"capabilities"`
}

// Capability describes one capability and its features.
type Capability struct {
	Name     string             `yaml:"name"`
	Features map[string]Feature `yaml:"features"`
}

// Feature describes one feature in a capability.
type Feature struct {
	Name string `yaml:"name"`
}

// Scenario contains the annotation metadata needed for validation and rendering.
type Scenario struct {
	Path string
	Name string
	Line int64
	Tags []string
}

var identifierPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// LoadMap parses and validates a capability map.
func LoadMap(r io.Reader) (*Map, error) {
	var out Map
	if err := yaml.NewDecoder(r).Decode(&out); err != nil {
		return nil, fmt.Errorf("taxonomy: decode capability map: %w", err)
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return &out, nil
}

// LoadMapFile loads and validates a capability map from disk.
func LoadMapFile(path string) (*Map, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("taxonomy: read capability map %q: %w", path, err)
	}
	return LoadMap(bytes.NewReader(raw))
}

// Validate checks the map structure and identifier vocabulary.
func (m *Map) Validate() error {
	if m == nil || len(m.Capabilities) == 0 {
		return fmt.Errorf("taxonomy: capability map is empty")
	}
	for capabilityID, capability := range m.Capabilities {
		if err := validateIdentifier("capability", capabilityID); err != nil {
			return err
		}
		if strings.TrimSpace(capability.Name) == "" {
			return fmt.Errorf("taxonomy: capability %q has no display name", capabilityID)
		}
		if len(capability.Features) == 0 {
			return fmt.Errorf("taxonomy: capability %q has no features", capabilityID)
		}
		for featureID, feature := range capability.Features {
			if err := validateIdentifier("feature", featureID); err != nil {
				return fmt.Errorf("taxonomy: capability %q: %w", capabilityID, err)
			}
			if strings.TrimSpace(feature.Name) == "" {
				return fmt.Errorf("taxonomy: feature %q/%q has no display name", capabilityID, featureID)
			}
		}
	}
	return nil
}

func validateIdentifier(kind, value string) error {
	if !identifierPattern.MatchString(value) {
		return fmt.Errorf("taxonomy: invalid %s identifier %q", kind, value)
	}
	return nil
}

// ParseFeature parses scenarios and their effective tags from one Gherkin file.
func ParseFeature(path string) ([]Scenario, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("taxonomy: read feature %q: %w", path, err)
	}
	var ids atomic.Uint64
	doc, err := gherkin.ParseGherkinDocument(bytes.NewReader(raw), func() string {
		return fmt.Sprintf("taxonomy-%d", ids.Add(1))
	})
	if err != nil {
		return nil, fmt.Errorf("taxonomy: parse feature %q: %w", path, err)
	}
	if doc == nil || doc.Feature == nil {
		return nil, fmt.Errorf("taxonomy: feature %q has no Feature declaration", path)
	}

	featureTags := tagNames(doc.Feature.Tags)
	var scenarios []Scenario
	for _, child := range doc.Feature.Children {
		if child.Scenario != nil {
			scenarios = append(scenarios, scenarioInfo(path, child.Scenario, featureTags))
		}
		if child.Rule != nil {
			ruleTags := append(append([]string(nil), featureTags...), tagNames(child.Rule.Tags)...)
			for _, ruleChild := range child.Rule.Children {
				if ruleChild.Scenario != nil {
					scenarios = append(scenarios, scenarioInfo(path, ruleChild.Scenario, ruleTags))
				}
			}
		}
	}
	return scenarios, nil
}

func scenarioInfo(path string, scenario *messages.Scenario, inherited []string) Scenario {
	tags := append(append([]string(nil), inherited...), tagNames(scenario.Tags)...)
	return Scenario{Path: path, Name: scenario.Name, Line: scenario.Location.Line, Tags: tags}
}

func tagNames(tags []*messages.Tag) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag != nil {
			out = append(out, tag.Name)
		}
	}
	return out
}

// ValidateScenarios validates annotations against the capability map.
func (m *Map) ValidateScenarios(scenarios []Scenario) error {
	if err := m.Validate(); err != nil {
		return err
	}
	for _, scenario := range scenarios {
		if err := m.validateScenario(scenario); err != nil {
			return err
		}
	}
	return nil
}

func (m *Map) validateScenario(scenario Scenario) error {
	byPrefix := collectTags(scenario.Tags)
	exclusions := byPrefix["setup"]
	exclusions = append(exclusions, byPrefix["infra"]...)
	exclusions = append(exclusions, byPrefix["framework"]...)
	exclusions = append(exclusions, byPrefix["migration"]...)
	if len(exclusions) > 1 {
		return scenarioError(scenario, "multiple exclusion annotations")
	}

	isSetupFile := strings.HasPrefix(filepath.Base(scenario.Path), "_setup_")
	if isSetupFile != (len(byPrefix["setup"]) == 1) {
		return scenarioError(scenario, "setup feature name and @setup annotation must agree")
	}
	if len(exclusions) == 1 {
		if len(byPrefix["cap"]) != 0 || len(byPrefix["feat"]) != 0 {
			return scenarioError(scenario, "exclusion annotations cannot be combined with @cap or @feat")
		}
		return nil
	}

	if len(byPrefix["cap"]) != 1 {
		return scenarioError(scenario, "exactly one @cap annotation is required")
	}
	if len(byPrefix["feat"]) != 1 {
		return scenarioError(scenario, "exactly one @feat annotation is required")
	}
	capID := byPrefix["cap"][0]
	featID := byPrefix["feat"][0]
	capability, ok := m.Capabilities[capID]
	if !ok {
		return scenarioError(scenario, fmt.Sprintf("unknown capability %q", capID))
	}
	if _, ok := capability.Features[featID]; !ok {
		return scenarioError(scenario, fmt.Sprintf("unknown feature %q under capability %q", featID, capID))
	}
	for _, kind := range []string{"rule", "legacy"} {
		for _, value := range byPrefix[kind] {
			if strings.TrimSpace(value) == "" {
				return scenarioError(scenario, fmt.Sprintf("@%s cannot have an empty value", kind))
			}
		}
	}
	for _, testType := range byPrefix["type"] {
		switch testType {
		case "smoke", "negative", "regression":
		default:
			return scenarioError(scenario, fmt.Sprintf("unsupported test type %q", testType))
		}
	}
	for _, dependency := range byPrefix["dep"] {
		if _, ok := m.Capabilities[dependency]; !ok {
			return scenarioError(scenario, fmt.Sprintf("unknown dependency capability %q", dependency))
		}
	}
	return nil
}

func collectTags(tags []string) map[string][]string {
	out := make(map[string][]string)
	for _, tag := range tags {
		if !strings.HasPrefix(tag, "@") {
			continue
		}
		name, value, ok := strings.Cut(strings.TrimPrefix(tag, "@"), ":")
		if name == "" {
			continue
		}
		if !ok {
			value = ""
		}
		out[name] = append(out[name], value)
	}
	return out
}

func scenarioError(scenario Scenario, message string) error {
	return fmt.Errorf("taxonomy: %s:%d scenario %q: %s", scenario.Path, scenario.Line, scenario.Name, message)
}

// DiscoverFeatureFiles returns all feature files below root in stable order.
func DiscoverFeatureFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".feature") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("taxonomy: discover features under %q: %w", root, err)
	}
	sort.Strings(paths)
	return paths, nil
}

// ValidateBindings verifies that every discovered feature is assigned to exactly one runner.
// Paths must use the same normalized representation in featureFiles and runnerFeatures.
func ValidateBindings(featureFiles []string, runnerFeatures map[string][]string) error {
	discovered := make(map[string]struct{}, len(featureFiles))
	for _, path := range featureFiles {
		path = filepath.Clean(path)
		if _, exists := discovered[path]; exists {
			return fmt.Errorf("taxonomy: feature %q was discovered more than once", path)
		}
		discovered[path] = struct{}{}
	}
	owners := make(map[string]string, len(featureFiles))
	for runner, paths := range runnerFeatures {
		if strings.TrimSpace(runner) == "" {
			return fmt.Errorf("taxonomy: runner name cannot be empty")
		}
		for _, path := range paths {
			path = filepath.Clean(path)
			if _, exists := discovered[path]; !exists {
				return fmt.Errorf("taxonomy: runner %q references unknown feature %q", runner, path)
			}
			if previous, exists := owners[path]; exists {
				return fmt.Errorf("taxonomy: feature %q is bound to runners %q and %q", path, previous, runner)
			}
			owners[path] = runner
		}
	}
	var orphaned []string
	for path := range discovered {
		if _, exists := owners[path]; !exists {
			orphaned = append(orphaned, path)
		}
	}
	if len(orphaned) > 0 {
		sort.Strings(orphaned)
		return fmt.Errorf("taxonomy: feature files are not bound to a runner: %s", strings.Join(orphaned, ", "))
	}
	return nil
}

// ParseAndValidate parses and validates a feature file.
func (m *Map) ParseAndValidate(path string) ([]Scenario, error) {
	scenarios, err := ParseFeature(path)
	if err != nil {
		return nil, err
	}
	return scenarios, m.ValidateScenarios(scenarios)
}

// RenderCoverageTree writes a deterministic Markdown tree for the capability map.
func (m *Map) RenderCoverageTree(w io.Writer, scenarios []Scenario) error {
	if err := m.Validate(); err != nil {
		return err
	}
	if err := m.ValidateScenarios(scenarios); err != nil {
		return err
	}
	type scenarioRef struct {
		name string
		path string
		line int64
	}
	refs := make(map[string][]scenarioRef)
	for _, scenario := range scenarios {
		tags := collectTags(scenario.Tags)
		if len(tags["cap"]) == 1 && len(tags["feat"]) == 1 {
			key := tags["cap"][0] + "/" + tags["feat"][0]
			refs[key] = append(refs[key], scenarioRef{scenario.Name, scenario.Path, scenario.Line})
		}
	}

	if _, err := fmt.Fprintln(w, "# Coverage Tree"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	capabilities := sortedKeys(m.Capabilities)
	for _, capabilityID := range capabilities {
		capability := m.Capabilities[capabilityID]
		if _, err := fmt.Fprintf(w, "- **%s** (`@cap:%s`)\n", capability.Name, capabilityID); err != nil {
			return err
		}
		for _, featureID := range sortedKeys(capability.Features) {
			feature := capability.Features[featureID]
			key := capabilityID + "/" + featureID
			items := refs[key]
			if _, err := fmt.Fprintf(w, "  - %s (`@feat:%s`) — %d scenario(s)\n", feature.Name, featureID, len(items)); err != nil {
				return err
			}
			sort.Slice(items, func(i, j int) bool {
				if items[i].path != items[j].path {
					return items[i].path < items[j].path
				}
				return items[i].line < items[j].line
			})
			for _, item := range items {
				if _, err := fmt.Fprintf(w, "    - `%s:%d` — %s\n", item.path, item.line, item.name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
