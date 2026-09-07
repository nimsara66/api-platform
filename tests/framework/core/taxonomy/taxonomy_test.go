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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func testMap() *Map {
	return &Map{Capabilities: map[string]Capability{
		"gateway": {
			Name: "Gateway",
			Features: map[string]Feature{
				"routing": {Name: "Routing"},
				"health":  {Name: "Health"},
			},
		},
		"admin": {
			Name:     "Admin",
			Features: map[string]Feature{"users": {Name: "Users"}},
		},
	}}
}

func writeFeature(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestParseAndValidateFeature(t *testing.T) {
	path := writeFeature(t, "routing.feature", `Feature: Routing

  @cap:gateway @feat:routing @type:smoke @dep:admin
  Scenario: routes a request
    When a request is sent

  Rule: fallback
    @cap:gateway @feat:routing @rule:fallback
    Scenario: uses a fallback
      When a request is sent
`)

	scenarios, err := testMap().ParseAndValidate(path)
	require.NoError(t, err)
	require.Len(t, scenarios, 2)
	require.Contains(t, scenarios[0].Tags, "@cap:gateway")
	require.Contains(t, scenarios[1].Tags, "@feat:routing")
}

func TestValidateScenarioRejectsInvalidAnnotations(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want string
	}{
		{"missing capability", []string{"@feat:routing"}, "exactly one @cap"},
		{"duplicate feature", []string{"@cap:gateway", "@feat:routing", "@feat:health"}, "exactly one @feat"},
		{"unknown feature", []string{"@cap:gateway", "@feat:missing"}, "unknown feature"},
		{"unknown type", []string{"@cap:gateway", "@feat:routing", "@type:load"}, "unsupported test type"},
		{"unknown dependency", []string{"@cap:gateway", "@feat:routing", "@dep:missing"}, "unknown dependency"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := testMap().ValidateScenarios([]Scenario{{Path: "test.feature", Line: 2, Name: tt.name, Tags: tt.tags}})
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestValidateSetupAnnotations(t *testing.T) {
	m := testMap()
	require.NoError(t, m.ValidateScenarios([]Scenario{
		{Path: "_setup_gateway.feature", Line: 1, Name: "setup", Tags: []string{"@setup"}},
		{Path: "helper.feature", Line: 1, Name: "helper", Tags: []string{"@infra"}},
	}))
	require.ErrorContains(t, m.ValidateScenarios([]Scenario{
		{Path: "_setup_gateway.feature", Line: 1, Name: "setup", Tags: []string{"@infra"}},
	}), "setup feature name")
	require.ErrorContains(t, m.ValidateScenarios([]Scenario{
		{Path: "helper.feature", Line: 1, Name: "helper", Tags: []string{"@setup"}},
	}), "setup feature name")
}

func TestRenderCoverageTreeIncludesEmptyFeatures(t *testing.T) {
	var output bytes.Buffer
	err := testMap().RenderCoverageTree(&output, []Scenario{{
		Path: "routing.feature", Line: 3, Name: "routes", Tags: []string{"@cap:gateway", "@feat:routing"},
	}})
	require.NoError(t, err)
	text := output.String()
	require.Contains(t, text, "Gateway")
	require.Contains(t, text, "Routing")
	require.Contains(t, text, "1 scenario(s)")
	require.Contains(t, text, "Health")
	require.Contains(t, text, "0 scenario(s)")
	first := strings.Index(text, "Users")
	second := strings.Index(text, "Health")
	require.GreaterOrEqual(t, first, 0)
	require.GreaterOrEqual(t, second, 0)
	require.Less(t, first, second)
}

func TestLoadMapRejectsInvalidMap(t *testing.T) {
	_, err := LoadMap(strings.NewReader("capabilities:\n  Bad_ID:\n    name: Bad\n    features:\n      test: {name: Test}\n"))
	require.ErrorContains(t, err, "invalid capability identifier")
	_, err = LoadMap(strings.NewReader("capabilities: {}\n"))
	require.ErrorContains(t, err, "capability map is empty")
}

func TestValidateBindings(t *testing.T) {
	files := []string{"a.feature", "b.feature"}
	require.NoError(t, ValidateBindings(files, map[string][]string{
		"runner-a": {"a.feature"}, "runner-b": {"b.feature"},
	}))
	require.ErrorContains(t, ValidateBindings(files, map[string][]string{
		"runner-a": {"a.feature"},
	}), "not bound")
	require.ErrorContains(t, ValidateBindings(files, map[string][]string{
		"runner-a": {"a.feature"}, "runner-b": {"a.feature", "b.feature"},
	}), "bound to runners")
	require.ErrorContains(t, ValidateBindings(files, map[string][]string{
		"runner-a": {"missing.feature"}, "runner-b": {"b.feature"},
	}), "unknown feature")
}
