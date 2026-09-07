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

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunGeneratesAndChecksTree(t *testing.T) {
	root := t.TempDir()
	mapPath := filepath.Join(root, "map.yml")
	features := filepath.Join(root, "features")
	output := filepath.Join(root, "coverage-tree.md")
	require.NoError(t, os.Mkdir(features, 0o755))
	require.NoError(t, os.WriteFile(mapPath, []byte("capabilities:\n  gateway:\n    name: Gateway\n    features:\n      routing: {name: Routing}\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(features, "routing.feature"), []byte(`Feature: Routing

  @cap:gateway @feat:routing
  Scenario: routes a request
    When a request is sent
`), 0o600))

	var stdout bytes.Buffer
	require.NoError(t, run([]string{"-map", mapPath, "-features", features, "-output", output}, &stdout, &stdout))
	content, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Contains(t, string(content), "1 scenario(s)")
	require.NoError(t, run([]string{"-map", mapPath, "-features", features, "-output", output, "-check"}, &stdout, &stdout))

	require.NoError(t, os.WriteFile(output, []byte("stale\n"), 0o600))
	require.ErrorContains(t, run([]string{"-map", mapPath, "-features", features, "-output", output, "-check"}, &stdout, &stdout), "stale")
}

func TestRunRejectsCheckWithoutOutput(t *testing.T) {
	var output bytes.Buffer
	require.ErrorContains(t, run([]string{"-check"}, &output, &output), "requires -output")
}
