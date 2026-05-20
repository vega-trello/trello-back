package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	dto "github.com/vega-trello/trello-back/internal/dto/member"
	"github.com/vega-trello/trello-back/internal/middleware"
)

type MemberHandler struct {
	memberService MemberServiceInterface
}

func NewMemberHandler(memberService MemberServiceInterface) *MemberHandler {
	return &MemberHandler{memberService: memberService}
}

// ListProjectMembers GET /projects/{projectUUID}/members
// Возвращает всех участников проекта с деталями
func (h *MemberHandler) ListProjectMembers(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	projectUUID, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	members, err := h.memberService.GetProjectMembers(c.Request.Context(), projectUUID, userUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	membersList := make([]dto.MemberResponse, 0, len(members))
	for _, m := range members {
		if m != nil {
			membersList = append(membersList, *m)
		}
	}

	c.JSON(http.StatusOK, dto.MemberListResponse{
		Members: membersList,
		Total:   len(membersList),
	})
}

// GetMember GET /projects/{projectUUID}/member?userUUID={uuid}
func (h *MemberHandler) GetMember(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	projectUUID, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	targetUserUUIDStr := c.Query("userUUID")
	if targetUserUUIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_user_uuid", "userUUID query parameter is required")
		return
	}

	targetUserUUID, err := uuid.Parse(targetUserUUIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_user_uuid", "Invalid user UUID format")
		return
	}

	member, err := h.memberService.GetMember(c.Request.Context(), projectUUID, userUUID, targetUserUUID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, member)
}

// AddMember POST /projects/{projectUUID}/members
// Добавляет пользователя в проект с указанной ролью
func (h *MemberHandler) AddMember(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	projectUUID, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	var req dto.CreateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	member, err := h.memberService.AddMember(c.Request.Context(), projectUUID, userUUID, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, member)
}

// UpdateMemberRole PATCH /projects/{projectUUID}/member?userUUID={uuid}
// Изменяет роль участника проекта
func (h *MemberHandler) UpdateMemberRole(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	projectUUID, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	targetUserUUIDStr := c.Query("userUUID")
	if targetUserUUIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_user_uuid", "userUUID query parameter is required")
		return
	}

	targetUserUUID, err := uuid.Parse(targetUserUUIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_user_uuid", "Invalid user UUID format")
		return
	}

	var req dto.UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := req.Validate(); err != nil {
		respondError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	updated, err := h.memberService.UpdateMemberRole(c.Request.Context(), projectUUID, userUUID, targetUserUUID, req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, updated)
}

// RemoveMember DELETE /projects/{projectUUID}/member?userUUID={uuid}
// Удаляет участника из проекта
func (h *MemberHandler) RemoveMember(c *gin.Context) {
	userUUID, ok := middleware.GetUserUUID(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "context_error", "User UUID not found in context")
		return
	}

	projectUUID, err := uuid.Parse(c.Param("projectUUID"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_uuid", "Invalid project UUID format")
		return
	}

	targetUserUUIDStr := c.Query("userUUID")
	if targetUserUUIDStr == "" {
		respondError(c, http.StatusBadRequest, "missing_user_uuid", "userUUID query parameter is required")
		return
	}

	targetUserUUID, err := uuid.Parse(targetUserUUIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid_user_uuid", "Invalid user UUID format")
		return
	}

	if err := h.memberService.RemoveMember(c.Request.Context(), projectUUID, userUUID, targetUserUUID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
