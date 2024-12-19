package utils

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type InstructionInfoEnvelope struct {
	AsString          string                   `json:"string,omitempty"`
	AsInstructionInfo *ParsedInstructionInfoV2 `json:"-"`
}

func (wrap *InstructionInfoEnvelope) MarshalJSON() ([]byte, error) {
	if wrap.AsString != "" {
		return json.Marshal(wrap.AsString)
	}
	return json.Marshal(wrap.AsInstructionInfo)
}

func (wrap *InstructionInfoEnvelope) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || (len(data) == 4 && string(data) == "null") {
		// TODO: is this an error?
		return nil
	}
	firstChar := data[0]
	switch firstChar {
	// Check if first character is [, standing for a JSON array.
	case '"':
		// It's base64 (or similar)
		err := json.Unmarshal(data, &wrap.AsString)
		if err != nil {
			return err
		}
	case '{':
		// It's JSON, most likely.
		return json.Unmarshal(data, &wrap.AsInstructionInfo)

	default:
		return errors.Errorf("unknown kind: %v", data)
	}
	return nil
}
