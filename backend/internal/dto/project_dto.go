package dto

// CreateProjectRequest is the payload for creating a mock project.
type CreateProjectRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=128"`
	Description string `json:"description" validate:"max=512"`
}

// UpdateProjectRequest is the payload for updating a mock project.
type UpdateProjectRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=128"`
	Description string `json:"description" validate:"max=512"`
}
