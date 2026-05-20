//go:build !integration
// +build !integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dto "github.com/vega-trello/trello-back/internal/dto/column"
	"github.com/vega-trello/trello-back/internal/model"
	"github.com/vega-trello/trello-back/internal/service"
)

type mockColumnService struct {
	createFunc func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto.CreateColumnRequest) (*model.Column, error)
	listFunc   func(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Column, error)
	getFunc    func(ctx context.Context, columnID int, userUUID uuid.UUID) (*model.Column, error)
	updateFunc func(ctx context.Context, columnID int, userUUID uuid.UUID, req dto.UpdateColumnRequest) (*model.Column, error)
	deleteFunc func(ctx context.Context, columnID int, userUUID uuid.UUID) error
	moveFunc   func(ctx context.Context, columnID int, userUUID uuid.UUID, direction string) (*model.Column, error)
}

func (m *mockColumnService) CreateColumn(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID, req dto.CreateColumnRequest) (*model.Column, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, projectUUID, userUUID, req)
	}
	return nil, nil
}
func (m *mockColumnService) GetProjectColumns(ctx context.Context, projectUUID uuid.UUID, userUUID uuid.UUID) ([]*model.Column, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, projectUUID, userUUID)
	}
	return nil, nil
}
func (m *mockColumnService) GetColumn(ctx context.Context, columnID int, userUUID uuid.UUID) (*model.Column, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, columnID, userUUID)
	}
	return nil, nil
}
func (m *mockColumnService) UpdateColumn(ctx context.Context, columnID int, userUUID uuid.UUID, req dto.UpdateColumnRequest) (*model.Column, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, columnID, userUUID, req)
	}
	return nil, nil
}
func (m *mockColumnService) DeleteColumn(ctx context.Context, columnID int, userUUID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, columnID, userUUID)
	}
	return nil
}
func (m *mockColumnService) MoveColumn(ctx context.Context, columnID int, userUUID uuid.UUID, direction string) (*model.Column, error) {
	if m.moveFunc != nil {
		return m.moveFunc(ctx, columnID, userUUID, direction)
	}
	return nil, nil
}

func setupColumnRouter(t *testing.T, colSvc *mockColumnService, jwtSecret string) *gin.Engine {
	t.Helper()

	h := NewColumnHandler(colSvc)

	return SetupTestRouterWithAuth(t, jwtSecret, func(rg *gin.RouterGroup) {
		rg.GET("/projects/:projectUUID/columns", h.ListProjectColumns)
		rg.POST("/projects/:projectUUID/columns", h.CreateColumn)

		// Column-scoped endpoints
		rg.GET("/columns/:columnID", h.GetColumn)
		rg.PATCH("/columns/:columnID", h.UpdateColumn)
		rg.DELETE("/columns/:columnID", h.DeleteColumn)
		rg.POST("/columns/:columnID/move", h.MoveColumn)
	})
}

func TestColumnHandler_ListProjectColumns_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	colID := 42

	colSvc := &mockColumnService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) ([]*model.Column, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			return []*model.Column{
				{
					ID:          colID,
					ProjectUUID: projectUUID,
					Name:        "To Do",
					Position:    0,
					CreatedAt:   time.Now(),
				},
			}, nil
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/columns", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var columns []dto.ColumnResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &columns))
	require.Len(t, columns, 1)
	assert.Equal(t, "To Do", columns[0].Name)
	assert.Equal(t, colID, columns[0].ID)
}

func TestColumnHandler_ListProjectColumns_InvalidUUID(t *testing.T) {
	userUUID := uuid.New()
	colSvc := &mockColumnService{}
	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/not-a-uuid/columns", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_uuid", http.StatusBadRequest)
}

func TestColumnHandler_ListProjectColumns_AccessDenied(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	colSvc := &mockColumnService{
		listFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID) ([]*model.Column, error) {
			return nil, service.ErrAccessDenied
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/projects/"+projectUUID.String()+"/columns", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "access_denied", http.StatusForbidden)
}

func TestColumnHandler_CreateColumn_Success(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	colID := 42
	position := 1

	colSvc := &mockColumnService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateColumnRequest) (*model.Column, error) {
			assert.Equal(t, projectUUID, pUUID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, "New Column", req.Name)
			assert.NotNil(t, req.Position)
			assert.Equal(t, position, *req.Position)
			return &model.Column{
				ID:          colID,
				ProjectUUID: projectUUID,
				Name:        req.Name,
				Position:    *req.Position,
				CreatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"New Column","position":1}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/columns", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var col dto.ColumnResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &col))
	assert.Equal(t, "New Column", col.Name)
	assert.Equal(t, position, col.Position)
}

func TestColumnHandler_CreateColumn_WithoutPosition(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	colID := 42

	colSvc := &mockColumnService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateColumnRequest) (*model.Column, error) {
			assert.Nil(t, req.Position)
			return &model.Column{
				ID:          colID,
				ProjectUUID: projectUUID,
				Name:        req.Name,
				Position:    0,
				CreatedAt:   time.Now(),
			}, nil
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Auto Position"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/columns", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var col dto.ColumnResponse
	json.Unmarshal(w.Body.Bytes(), &col)
	assert.Equal(t, "Auto Position", col.Name)
}

func TestColumnHandler_CreateColumn_InvalidName(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	colSvc := &mockColumnService{}
	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":""}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/columns", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_request", http.StatusBadRequest)
}

func TestColumnHandler_CreateColumn_AccessDenied(t *testing.T) {
	userUUID := uuid.New()
	projectUUID := uuid.New()
	colSvc := &mockColumnService{
		createFunc: func(ctx context.Context, pUUID uuid.UUID, uUUID uuid.UUID, req dto.CreateColumnRequest) (*model.Column, error) {
			return nil, service.ErrAccessDenied
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Test"}`)
	req := httptest.NewRequest("POST", "/projects/"+projectUUID.String()+"/columns", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "access_denied", http.StatusForbidden)
}

func TestColumnHandler_GetColumn_Success(t *testing.T) {
	userUUID := uuid.New()
	columnID := 42

	colSvc := &mockColumnService{
		getFunc: func(ctx context.Context, cID int, uUUID uuid.UUID) (*model.Column, error) {
			assert.Equal(t, columnID, cID)
			assert.Equal(t, userUUID, uUUID)
			return &model.Column{
				ID:        columnID,
				Name:      "Done",
				Position:  2,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/columns/"+strconv.Itoa(columnID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var col dto.ColumnResponse
	json.Unmarshal(w.Body.Bytes(), &col)
	assert.Equal(t, "Done", col.Name)
	assert.Equal(t, columnID, col.ID)
}

func TestColumnHandler_GetColumn_InvalidID(t *testing.T) {
	userUUID := uuid.New()
	colSvc := &mockColumnService{}
	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/columns/abc", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_column_id", http.StatusBadRequest)
}

func TestColumnHandler_GetColumn_NotFound(t *testing.T) {
	userUUID := uuid.New()
	columnID := 999
	colSvc := &mockColumnService{
		getFunc: func(ctx context.Context, cID int, uUUID uuid.UUID) (*model.Column, error) {
			return nil, service.ErrColumnNotFound
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("GET", "/columns/"+strconv.Itoa(columnID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "column_not_found", http.StatusNotFound)
}

func TestColumnHandler_UpdateColumn_Success(t *testing.T) {
	userUUID := uuid.New()
	columnID := 42
	newName := "Updated Name"
	newPosition := 5

	colSvc := &mockColumnService{
		updateFunc: func(ctx context.Context, cID int, uUUID uuid.UUID, req dto.UpdateColumnRequest) (*model.Column, error) {
			assert.Equal(t, columnID, cID)
			assert.Equal(t, userUUID, uUUID)
			assert.Equal(t, newName, req.Name)
			assert.NotNil(t, req.Position)
			assert.Equal(t, newPosition, *req.Position)
			return &model.Column{
				ID:        columnID,
				Name:      newName,
				Position:  newPosition,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Updated Name","position":5}`)
	req := httptest.NewRequest("PATCH", "/columns/"+strconv.Itoa(columnID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var col dto.ColumnResponse
	json.Unmarshal(w.Body.Bytes(), &col)
	assert.Equal(t, newName, col.Name)
	assert.Equal(t, newPosition, col.Position)
}

func TestColumnHandler_UpdateColumn_PartialUpdate_NameOnly(t *testing.T) {
	userUUID := uuid.New()
	columnID := 42
	newName := "Only Name Changed"

	colSvc := &mockColumnService{
		updateFunc: func(ctx context.Context, cID int, uUUID uuid.UUID, req dto.UpdateColumnRequest) (*model.Column, error) {
			assert.Equal(t, newName, req.Name)
			assert.Nil(t, req.Position)
			return &model.Column{
				ID:        columnID,
				Name:      newName,
				Position:  2,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"name":"Only Name Changed"}`)
	req := httptest.NewRequest("PATCH", "/columns/"+strconv.Itoa(columnID), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var col dto.ColumnResponse
	json.Unmarshal(w.Body.Bytes(), &col)
	assert.Equal(t, newName, col.Name)
}

func TestColumnHandler_DeleteColumn_Success(t *testing.T) {
	userUUID := uuid.New()
	columnID := 42
	called := false

	colSvc := &mockColumnService{
		deleteFunc: func(ctx context.Context, cID int, uUUID uuid.UUID) error {
			assert.Equal(t, columnID, cID)
			assert.Equal(t, userUUID, uUUID)
			called = true
			return nil
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/columns/"+strconv.Itoa(columnID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
	assert.True(t, called)
}

func TestColumnHandler_DeleteColumn_HasTasks(t *testing.T) {
	userUUID := uuid.New()
	columnID := 42
	colSvc := &mockColumnService{
		deleteFunc: func(ctx context.Context, cID int, uUUID uuid.UUID) error {
			return service.ErrColumnHasTasks
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	req := httptest.NewRequest("DELETE", "/columns/"+strconv.Itoa(columnID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "column_has_tasks", http.StatusConflict)
}

func TestColumnHandler_MoveColumn_Success_Left(t *testing.T) {
	userUUID := uuid.New()
	columnID := 42

	colSvc := &mockColumnService{
		moveFunc: func(ctx context.Context, cID int, uUUID uuid.UUID, direction string) (*model.Column, error) {
			assert.Equal(t, columnID, cID)
			assert.Equal(t, "left", direction)
			return &model.Column{
				ID:        columnID,
				Name:      "Moved",
				Position:  1,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"direction":"left"}`)
	req := httptest.NewRequest("POST", "/columns/"+strconv.Itoa(columnID)+"/move", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var col dto.ColumnResponse
	json.Unmarshal(w.Body.Bytes(), &col)
	assert.Equal(t, 1, col.Position)
}

func TestColumnHandler_MoveColumn_InvalidDirection(t *testing.T) {
	userUUID := uuid.New()
	columnID := 42
	colSvc := &mockColumnService{}
	r := setupColumnRouter(t, colSvc, "test-secret")
	token := GenerateTestToken(t, userUUID, "test-secret")

	body := bytes.NewBufferString(`{"direction":"up"}`)
	req := httptest.NewRequest("POST", "/columns/"+strconv.Itoa(columnID)+"/move", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	AssertErrorResponse(t, w.Body.Bytes(), "invalid_request", http.StatusBadRequest)
}

func TestColumnHandler_Unauthorized_AllEndpoints(t *testing.T) {
	colSvc := &mockColumnService{}
	r := setupColumnRouter(t, colSvc, "test-secret")

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/projects/" + uuid.New().String() + "/columns", ""},
		{"POST", "/projects/" + uuid.New().String() + "/columns", `{"name":"test"}`},
		{"GET", "/columns/42", ""},
		{"PATCH", "/columns/42", `{"name":"new"}`},
		{"DELETE", "/columns/42", ""},
		{"POST", "/columns/42/move", `{"direction":"left"}`},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			var req *http.Request
			if ep.body != "" {
				req = httptest.NewRequest(ep.method, ep.path, bytes.NewBufferString(ep.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(ep.method, ep.path, nil)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
