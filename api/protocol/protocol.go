package protocol

type AuthRequest struct {
	Address   string
	Challenge string
	Signature string
}

type SendRequest struct {
	From    string
	To      string
	Content string
}
