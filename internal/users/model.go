package users

type UpdateProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
