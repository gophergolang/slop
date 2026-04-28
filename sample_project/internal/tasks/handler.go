package tasks

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/vibeguard/platform/db"
	"github.com/vibeguard/platform/events"
	"github.com/vibeguard/team-task-saas/internal/tasks/repository"
)

type Handler struct {
	db       db.DB
	events   events.Publisher
	repo     *repository.TaskRepository
	validate *validator.Validate
}

func NewHandler(database db.DB, eventPublisher events.Publisher) *Handler {
	return &Handler{
		db:       database,
		events:   eventPublisher,
		repo:     repository.NewTaskRepository(database),
		validate: validator.New(),
	}
}

// Create - ONLY because create: true in declaration
func (h *Handler) Create(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	task, err := h.repo.Create(c.Request.Context(), tenantID, userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create task"})
		return
	}
	c.JSON(http.StatusCreated, task)
}

// Get - ONLY because read: true
func (h *Handler) Get(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenant_id")

	task, err := h.repo.GetByID(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// List - ONLY because list: true
func (h *Handler) List(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")

	tasks, err := h.repo.List(c.Request.Context(), tenantID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tasks"})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

// Update - ONLY fields allowed in declaration: [title, description, status, priority, due_date, assignee_id]
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenant_id")

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// The repository enforces the whitelist - only these fields are updatable
	task, err := h.repo.Update(c.Request.Context(), tenantID, id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update task"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// Prioritize - Custom endpoint from declaration (AI integration)
func (h *Handler) Prioritize(c *gin.Context) {
	id := c.Param("id")
	tenantID := c.GetString("tenant_id")

	task, err := h.repo.PrioritizeWithAI(c.Request.Context(), tenantID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "prioritization failed"})
		return
	}

	// Use Platform SDK to emit event (NATS-ready)
	h.events.Publish(c.Request.Context(), "tasks.prioritized", events.Event{
		Type:     "TaskPrioritized",
		TenantID: tenantID,
		Data:     []byte(`{"task_id":"` + id + `"}`),
	})

	c.JSON(http.StatusOK, task)
}

// NOTE: No Delete handler is generated because delete: false in the declaration
// NOTE: All queries automatically include tenant_id filter (multi-tenancy from declaration)
// NOTE: RLS condition from declaration is applied at repository layer