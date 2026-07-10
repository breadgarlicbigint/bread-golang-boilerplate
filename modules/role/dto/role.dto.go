package dto

import (
	"github.com/google/uuid"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
)

// RoleResponse is the shape returned by role listing endpoints — enough for
// a client to render a role-selection dropdown without exposing permissions.
type RoleResponse struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Slug        entity.RoleSlug `json:"slug"`
	Description string          `json:"description"`
}

// FromEntity maps a role entity to its response DTO.
func FromEntity(r entity.Role) RoleResponse {
	return RoleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Slug:        r.Slug,
		Description: r.Description,
	}
}
