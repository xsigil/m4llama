package sqlite3

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/xsigil/m4llama/internal/domain/entity"
)

type HistoryRepository struct {
	db *sqlx.DB
}

func NewHistoryRepository(db *sqlx.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) Save(ctx context.Context, entry *entity.HistoryEntry) error {
	varsJSON, err := json.Marshal(entry.Variables)
	if err != nil {
		return fmt.Errorf("marshal variables: %w", err)
	}
	entry.VariablesJSON = string(varsJSON)

	query := `
	INSERT INTO history (
		template_name, variables_json, expanded_prompt, completion,
		prompt_tokens, completion_tokens, total_tokens, pinned, created_at
	) VALUES (
		:template_name, :variables_json, :expanded_prompt, :completion,
		:prompt_tokens, :completion_tokens, :total_tokens, :pinned, :created_at
	)
	`
	res, err := r.db.NamedExecContext(ctx, query, entry)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		entry.ID = id
	}
	return nil
}

func (r *HistoryRepository) Update(ctx context.Context, entry *entity.HistoryEntry) error {
	varsJSON, err := json.Marshal(entry.Variables)
	if err != nil {
		return fmt.Errorf("marshal variables: %w", err)
	}
	entry.VariablesJSON = string(varsJSON)

	query := `
	UPDATE history SET
		template_name = :template_name,
		variables_json = :variables_json,
		expanded_prompt = :expanded_prompt,
		completion = :completion,
		prompt_tokens = :prompt_tokens,
		completion_tokens = :completion_tokens,
		total_tokens = :total_tokens,
		pinned = :pinned
	WHERE id = :id
	`
	_, err = r.db.NamedExecContext(ctx, query, entry)
	return err
}

func (r *HistoryRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM history WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *HistoryRepository) FindByID(ctx context.Context, id int64) (*entity.HistoryEntry, error) {
	var entry entity.HistoryEntry
	query := `SELECT * FROM history WHERE id = ? LIMIT 1`
	err := r.db.GetContext(ctx, &entry, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if entry.VariablesJSON != "" {
		_ = json.Unmarshal([]byte(entry.VariablesJSON), &entry.Variables)
	}
	return &entry, nil
}

func (r *HistoryRepository) ListRecent(ctx context.Context, limit int) ([]*entity.HistoryEntry, error) {
	var list []*entity.HistoryEntry
	query := `SELECT * FROM history ORDER BY id DESC LIMIT ?`
	err := r.db.SelectContext(ctx, &list, query, limit)
	if err != nil {
		return nil, err
	}
	for _, e := range list {
		if e.VariablesJSON != "" {
			_ = json.Unmarshal([]byte(e.VariablesJSON), &e.Variables)
		}
	}
	return list, nil
}

func (r *HistoryRepository) ListPinned(ctx context.Context) ([]*entity.HistoryEntry, error) {
	var list []*entity.HistoryEntry
	query := `SELECT * FROM history WHERE pinned = 1 ORDER BY id DESC`
	err := r.db.SelectContext(ctx, &list, query)
	if err != nil {
		return nil, err
	}
	for _, e := range list {
		if e.VariablesJSON != "" {
			_ = json.Unmarshal([]byte(e.VariablesJSON), &e.Variables)
		}
	}
	return list, nil
}

func (r *HistoryRepository) SetPinned(ctx context.Context, id int64, pinned bool) error {
	query := `UPDATE history SET pinned = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, pinned, id)
	return err
}
