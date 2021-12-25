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

func (m Message) Participants() map[string]bool {
	ps := make(map[string]bool)
	ps[m.From] = true
	for _, addr := range m.To {
		ps[addr] = true
	}
	return ps
}
