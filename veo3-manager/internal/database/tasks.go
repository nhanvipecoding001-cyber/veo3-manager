package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (db *DB) CreateTask(prompt, aspectRatio, model string, outputCount int) (*Task, error) {
	task := &Task{
		ID:          uuid.New().String(),
		Prompt:      prompt,
		Status:      "pending",
		AspectRatio: aspectRatio,
		Model:       model,
		OutputCount: outputCount,
		MediaIDs:    StringSlice{},
		VideoPaths:  StringSlice{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mediaIDsVal, _ := task.MediaIDs.Value()
	videoPathsVal, _ := task.VideoPaths.Value()

	_, err := db.conn.Exec(
		`INSERT INTO tasks (id, prompt, status, aspect_ratio, model, output_count, media_ids, video_paths, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		task.ID, task.Prompt, task.Status, task.AspectRatio, task.Model, task.OutputCount,
		mediaIDsVal, videoPathsVal, task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (db *DB) CreateTasksBatch(prompts []string, aspectRatio, model string, outputCount int) ([]Task, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO tasks (id, prompt, status, aspect_ratio, model, output_count, media_ids, video_paths, created_at, updated_at)
		 VALUES (?, ?, 'pending', ?, ?, ?, '[]', '[]', ?, ?)`,
	)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	now := time.Now()
	tasks := make([]Task, 0, len(prompts))
	for _, prompt := range prompts {
		prompt = strings.TrimSpace(prompt)
		if prompt == "" {
			continue
		}
		id := uuid.New().String()
		if _, err := stmt.Exec(id, prompt, aspectRatio, model, outputCount, now, now); err != nil {
			return nil, err
		}
		tasks = append(tasks, Task{
			ID:          id,
			Prompt:      prompt,
			Status:      "pending",
			AspectRatio: aspectRatio,
			Model:       model,
			OutputCount: outputCount,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	return tasks, tx.Commit()
}

func (db *DB) GetTask(id string) (*Task, error) {
	row := db.conn.QueryRow(
		`SELECT id, prompt, status, aspect_ratio, model, output_count, media_ids, video_paths,
		        error_message, seed, created_at, updated_at, completed_at
		 FROM tasks WHERE id = ?`, id,
	)
	return scanTaskFrom(row)
}

func (db *DB) ListTasks(filter TaskFilter) ([]Task, error) {
	query := `SELECT id, prompt, status, aspect_ratio, model, output_count, media_ids, video_paths,
	                  error_message, seed, created_at, updated_at, completed_at
	           FROM tasks WHERE 1=1`
	args := []interface{}{}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.Search != "" {
		query += " AND prompt LIKE ?"
		args = append(args, "%"+filter.Search+"%")
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		t, err := scanTaskFrom(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *t)
	}
	return tasks, rows.Err()
}

func (db *DB) UpdateTaskStatus(id, status string) error {
	_, err := db.conn.Exec(
		`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now(), id,
	)
	return err
}

func (db *DB) UpdateTaskMediaIDs(id string, mediaIDs []string) error {
	val, _ := StringSlice(mediaIDs).Value()
	_, err := db.conn.Exec(
		`UPDATE tasks SET media_ids = ?, status = 'polling', updated_at = ? WHERE id = ?`,
		val, time.Now(), id,
	)
	return err
}

func (db *DB) UpdateTaskVideoPaths(id string, paths []string) error {
	val, _ := StringSlice(paths).Value()
	_, err := db.conn.Exec(
		`UPDATE tasks SET video_paths = ?, status = 'completed', updated_at = ?, completed_at = ? WHERE id = ?`,
		val, time.Now(), time.Now(), id,
	)
	return err
}

// ResetStuckTasks resets any tasks stuck in processing/polling/downloading back to pending
func (db *DB) ResetStuckTasks() {
	db.conn.Exec(
		`UPDATE tasks SET status = 'pending', updated_at = ? WHERE status IN ('processing', 'polling', 'downloading')`,
		time.Now(),
	)
}

func (db *DB) UpdateTaskError(id, errMsg string) error {
	_, err := db.conn.Exec(
		`UPDATE tasks SET status = 'failed', error_message = ?, updated_at = ? WHERE id = ?`,
		errMsg, time.Now(), id,
	)
	return err
}

func (db *DB) GetTaskStats() (*TaskStats, error) {
	stats := &TaskStats{}
	err := db.conn.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status IN ('processing','polling','downloading') THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0)
		FROM tasks
	`).Scan(&stats.Total, &stats.Pending, &stats.Processing, &stats.Completed, &stats.Failed)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (db *DB) GetPendingTasks() ([]Task, error) {
	return db.ListTasks(TaskFilter{Status: "pending"})
}

func (db *DB) DeleteTask(id string) error {
	_, err := db.conn.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	return err
}

func (db *DB) CancelPendingTasks() error {
	_, err := db.conn.Exec(
		`UPDATE tasks SET status = 'cancelled', updated_at = ? WHERE status = 'pending'`,
		time.Now(),
	)
	return err
}

// Scanner helper

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTaskFrom(s scanner) (*Task, error) {
	t := &Task{}
	var mediaIDs, videoPaths, errMsg, seed string
	err := s.Scan(
		&t.ID, &t.Prompt, &t.Status, &t.AspectRatio, &t.Model, &t.OutputCount,
		&mediaIDs, &videoPaths, &errMsg, &seed,
		&t.CreatedAt, &t.UpdatedAt, &t.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	t.ErrorMessage = errMsg
	t.Seed = seed
	t.MediaIDs.Scan(mediaIDs)
	t.VideoPaths.Scan(videoPaths)
	return t, nil
}
