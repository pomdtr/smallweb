package server

type LoginParams struct {
	Email string
}

type SignupParams struct {
	Email    string
	Username string
}

type SignupResponse struct {
	Username string
}

type LoginResponse struct {
	Username string
}

type VerifyEmailParams struct {
	Code string
}

type UserResponse struct {
	ID    string
	Name  string
	Email string
}

type ErrorPayload struct {
	Message string
}

type ListAppsResponse struct {
	Apps []string
}

type ErrorResponse struct {
	Message string `json:"message"`
}
