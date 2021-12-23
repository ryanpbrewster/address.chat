package protocol

type AuthRequest struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}
type AuthPayload struct {
	Address   string `json:"address"`
	ExpiresAt int    `json:"expiresAt"`
}
type AuthResponse struct {
	AuthenticatedUntil int `json:"authenticatedUntil"`
}

type SendRequest struct {
	To      []string `json:"to"`
	Content string   `json:"content"`
}

type Message struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Content string   `json:"content"`
}
