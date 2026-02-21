package api

type ErrorResponse struct {
	Error string `json:"error"`
}

type RenderResponse struct {
	Content string `json:"content"`
}
