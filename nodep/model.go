package nodep

import (
	"encoding/base64"
	"encoding/json"
)

type CallResponse[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data,omitempty"`
	Err     string `json:"error,omitempty"`
}

func (response CallResponse[T]) EncodeToBase64(data T, err error) string {
	response.Data = data
	if err != nil {
		response.Success = false
		response.Err = err.Error()
	} else {
		response.Success = true
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(jsonData)
}
