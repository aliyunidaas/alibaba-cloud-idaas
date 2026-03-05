package openclaw

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// specification: https://docs.openclaw.ai/gateway/secrets

const ProtocolVersion1 = 1

// OpenClawSecretProviderRequest
// stdin:
// { "protocolVersion": 1, "provider": "vault", "ids": ["providers/openai/apiKey"] }
type OpenClawSecretProviderRequest struct {
	ProtocolVersion int      `json:"protocolVersion"`
	Provider        string   `json:"provider"`
	Ids             []string `json:"ids"`
}

type OpenClawSecretProviderResponseErrorMessage struct {
	Message string `json:"message"`
}

// OpenClawSecretProviderResponse
// stdout:
// { "protocolVersion": 1, "values": { "providers/openai/apiKey": "sk-..." } }
// with error:
// {
//  "protocolVersion": 1,
//  "values": {},
//  "errors": { "providers/openai/apiKey": { "message": "not found" } }
// }
type OpenClawSecretProviderResponse struct {
	ProtocolVersion int                                                     `json:"protocolVersion"`
	Values          *map[string]string                                      `json:"values"`
	Errors          *map[string]*OpenClawSecretProviderResponseErrorMessage `json:"errors,omitempty"`
}

func UnmarshalRequest(request string) (*OpenClawSecretProviderRequest, error) {
	var openClawSecretProviderRequest OpenClawSecretProviderRequest
	err := json.Unmarshal([]byte(request), &openClawSecretProviderRequest)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal OpenClaw secret request: %s failed", request)
	}
	return &openClawSecretProviderRequest, nil
}

func (t *OpenClawSecretProviderResponse) Marshal() (string, error) {
	if t == nil {
		return "null", nil
	}
	tokenBytes, err := json.Marshal(t)
	if err != nil {
		return "", errors.Wrap(err, "marshal OpenClaw secret response failed")
	}
	return string(tokenBytes), nil
}
