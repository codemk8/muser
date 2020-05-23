package verify

type VerifyRequest struct {
	UserName   string `json:"user_name,omitempty"`
	To         string `json:"to,omitempty"`
	VerifyCode string `json:"verify_code,omitempty"`
}
