package nodep

type CallResponse struct {
	Success bool   `json:"success,omitempty"`
	Result  string `json:"result,omitempty"`
	Err     string `json:"error,omitempty"`
}
