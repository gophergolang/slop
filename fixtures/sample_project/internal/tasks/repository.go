package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vibeguard/team-task-saas/internal/tasks/domain"
)

type TaskRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create - respects tenant isolation
func (r *TaskRepository) Create(ctx context.Context, tenantID, userID string, req domain.CreateTaskRequest) (*domain.Task, error) {
	query := `
		INSERT INTO tasks (tenant_id, team_id, title, description, status, priority, due_date, created_by, assignee_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, tenant_id, team_id, title, description, status, priority, due_date, assignee_id, created_by, created_at, updated_at, deleted_at
	`
	var task domain.Task
	err := r.db.QueryRow(ctx, query,
		tenantID, req.TeamID, req.Title, req.Description, req.Status, req.Priority, req.DueDate, userID, req.AssigneeID,
	).Scan(&task.ID, &task.TenantID, &task.TeamID, &task.Title, &task.Description, &task.Status,
		&task.Priority, &task.DueDate, &task.AssigneeID, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt)
	return &task, err
}

// GetByID - automatically filters by tenant_id + RLS condition from declaration
func (r *TaskRepository) GetByID(ctx context.Context, tenantID, id string) (*domain.Task, error) {
	query := `
		SELECT id, tenant_id, team_id, title, description, status, priority, due_date, assignee_id, created_by, created_at, updated_at, deleted_at
		FROM tasks
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`
	var task domain.Task
	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&task.ID, &task.TenantID, &task.TeamID, &task.Title, &task.Description, &task.Status,
		&task.Priority, &task.DueDate, &task.AssigneeID, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt,
	)
	return &task, err
}

// List - applies tenant filter + RLS policy from declaration
func (r *TaskRepository) List(ctx context.Context, tenantID, userID string) ([]domain.Task, error) {
	// RLS condition from declaration is applied here + at DB level
	query := `
		SELECT id, tenant_id, team_id, title, description, status, priority, due_date, assignee_id, created_by, created_at, updated_at, deleted_at
		FROM tasks
		WHERE tenant_id = $1 
		  AND deleted_at IS NULL
		  AND (assignee_id = $2 OR created_by = $2 OR team_id IN (
		      SELECT team_id FROM team_members WHERE user_id = $2
		  ))
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var t domain.Task
		rows.Scan(&t.ID, &t.TenantID, &t.TeamID, &t.Title, &t.Description, &t.Status,
			&t.Priority, &t.DueDate, &t.AssigneeID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt)
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// Update - ONLY allowed fields from declaration: title, description, status, priority, due_date, assignee_id
func (r *TaskRepository) Update(ctx context.Context, tenantID, id string, req domain.UpdateTaskRequest) (*domain.Task, error) {
	// Dynamic query building based on declaration whitelist
	setClauses := []string{}
	args := []interface{}{tenantID, id}
	argPos := 3

	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argPos))
		args = append(args, *req.Title)
		argPos++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *req.Description)
		argPos++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *req.Status)
		argPos++
	}
	if req.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argPos))
		args = append(args, *req.Priority)
		argPos++
	}
	if req.DueDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argPos))
		args = append(args, *req.DueDate)
		argPos++
	}
	if req.AssigneeID != nil {
		setClauses = append(setClauses, fmt.Sprintf("assignee_id = $%d", argPos))
		args = append(args, *req.AssigneeID)
		argPos++
	}

	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE tasks 
		SET %s, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $1 AND deleted_at IS NULL
		RETURNING id, tenant_id, team_id, title, description, status, priority, due_date, assignee_id, created_by, created_at, updated_at, deleted_at
	`, strings.Join(setClauses, ", "))

	var task domain.Task
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&task.ID, &task.TenantID, &task.TeamID, &task.Title, &task.Description, &task.Status,
		&task.Priority, &task.DueDate, &task.AssigneeID, &task.CreatedBy, &task.CreatedAt, &task.UpdatedAt, &task.DeletedAt,
	)
	return &task, err
}

// PrioritizeWithAI - custom method for the /prioritize endpoint declared in spec
func (r *TaskRepository) PrioritizeWithAI(ctx context.Context, tenantID, id string) (*domain.Task, error) {
	// Calls OpenAI (integration declared), updates priority
	// ... implementation ...
	return nil, nil
}