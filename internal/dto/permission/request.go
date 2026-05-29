package dto

// ListPermissionsRequest -параметры запроса для GET /projects/permissions
type ListPermissionsRequest struct{}

// Validate возвращает nil, так как валидировать нечего.
func (r *ListPermissionsRequest) Validate() error {
	return nil
}
