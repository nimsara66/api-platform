package transformer

import (
	api "github.com/wso2/api-platform/gateway/gateway-controller/pkg/api/generated"
)

// Transformer converts higher-level entities (like LLM Provider) into deployable APIConfiguration
// Implementations should be pure and side-effect free.
type Transformer interface {
	// Transform converts the provided configuration object into an APIConfiguration.
	// The input is a generated API type (e.g., *api.LLMProvider) and the output is a fully
	// formed API configuration ready for validation and deployment.
	Transform(config interface{}) (*api.APIConfiguration, error)
}
