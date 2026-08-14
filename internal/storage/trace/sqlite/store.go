package sqlite

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	domain "github.com/panda/tracy/internal/trace"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }
func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS spans (project_id TEXT NOT NULL, trace_id TEXT NOT NULL, span_id TEXT NOT NULL, parent_span_id TEXT NOT NULL DEFAULT '', name TEXT NOT NULL, span_kind TEXT NOT NULL DEFAULT '', start_time INTEGER NOT NULL, received_at INTEGER NOT NULL, duration INTEGER NOT NULL, status TEXT NOT NULL DEFAULT '', status_message TEXT NOT NULL DEFAULT '', input TEXT NOT NULL DEFAULT '', output TEXT NOT NULL DEFAULT '', input_tokens INTEGER NOT NULL DEFAULT 0, output_tokens INTEGER NOT NULL DEFAULT 0, attributes_json TEXT NOT NULL DEFAULT '{}', PRIMARY KEY(project_id,trace_id,span_id)); CREATE INDEX IF NOT EXISTS idx_spans_project_start ON spans(project_id,start_time); CREATE INDEX IF NOT EXISTS idx_spans_project_trace ON spans(project_id,trace_id); CREATE INDEX IF NOT EXISTS idx_spans_project_kind_start ON spans(project_id,span_kind,start_time); CREATE TABLE IF NOT EXISTS trace_summaries (project_id TEXT NOT NULL, trace_id TEXT NOT NULL, start_time INTEGER NOT NULL, end_time INTEGER NOT NULL, span_count INTEGER NOT NULL, status TEXT NOT NULL, input_tokens INTEGER NOT NULL, output_tokens INTEGER NOT NULL, PRIMARY KEY(project_id,trace_id)); CREATE INDEX IF NOT EXISTS idx_trace_summaries_project_start ON trace_summaries(project_id,start_time,trace_id); CREATE INDEX IF NOT EXISTS idx_trace_summaries_project_status ON trace_summaries(project_id,status,start_time,trace_id);`)
	if err != nil {
		return err
	}
	var summaryCount int
	if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM trace_summaries`).Scan(&summaryCount); err != nil {
		return err
	}
	if summaryCount == 0 {
		_, err = s.db.ExecContext(ctx, `INSERT OR REPLACE INTO trace_summaries(project_id,trace_id,start_time,end_time,span_count,status,input_tokens,output_tokens) SELECT project_id,trace_id,MIN(start_time),MAX(start_time + duration / 1000),COUNT(*),CASE WHEN SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) > 0 THEN 'error' ELSE 'ok' END,COALESCE(SUM(input_tokens),0),COALESCE(SUM(output_tokens),0) FROM spans GROUP BY project_id,trace_id`)
	}
	return err
}
func (s *Store) Append(ctx context.Context, spans []domain.Span) error {
	if len(spans) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO spans(project_id,trace_id,span_id,parent_span_id,name,span_kind,start_time,received_at,duration,status,status_message,input,output,input_tokens,output_tokens,attributes_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(project_id,trace_id,span_id) DO UPDATE SET parent_span_id=excluded.parent_span_id,name=excluded.name,span_kind=excluded.span_kind,start_time=excluded.start_time,received_at=excluded.received_at,duration=excluded.duration,status=excluded.status,status_message=excluded.status_message,input=excluded.input,output=excluded.output,input_tokens=excluded.input_tokens,output_tokens=excluded.output_tokens,attributes_json=excluded.attributes_json`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, sp := range spans {
		attrs, _ := json.Marshal(sp.Attributes)
		if _, err = stmt.ExecContext(ctx, sp.ProjectID, sp.TraceID, sp.SpanID, sp.ParentSpanID, sp.Name, sp.Kind, sp.StartTime.UTC().UnixMicro(), sp.ReceivedAt.UTC().UnixMicro(), sp.Duration.Nanoseconds(), sp.Status, sp.StatusMessage, sp.Input, sp.Output, sp.InputTokens, sp.OutputTokens, string(attrs)); err != nil {
			return err
		}
	}
	traceIDs := make(map[string]string, len(spans))
	for _, sp := range spans {
		traceIDs[sp.ProjectID+"\x00"+sp.TraceID] = sp.ProjectID + "\x00" + sp.TraceID
	}
	for _, composite := range traceIDs {
		parts := strings.SplitN(composite, "\x00", 2)
		if _, err = tx.ExecContext(ctx, `INSERT INTO trace_summaries(project_id,trace_id,start_time,end_time,span_count,status,input_tokens,output_tokens) SELECT project_id,trace_id,MIN(start_time),MAX(start_time + duration / 1000),COUNT(*),CASE WHEN SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) > 0 THEN 'error' ELSE 'ok' END,COALESCE(SUM(input_tokens),0),COALESCE(SUM(output_tokens),0) FROM spans WHERE project_id=? AND trace_id=? GROUP BY project_id,trace_id ON CONFLICT(project_id,trace_id) DO UPDATE SET start_time=excluded.start_time,end_time=excluded.end_time,span_count=excluded.span_count,status=excluded.status,input_tokens=excluded.input_tokens,output_tokens=excluded.output_tokens`, parts[0], parts[1]); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) GetTrace(ctx context.Context, projectID, traceID string) ([]domain.Span, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT project_id,trace_id,span_id,parent_span_id,name,span_kind,start_time,received_at,duration,status,status_message,input,output,input_tokens,output_tokens,attributes_json FROM spans WHERE project_id=? AND trace_id=? ORDER BY start_time,span_id`, projectID, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Span
	for rows.Next() {
		var sp domain.Span
		var start, received, duration int64
		var attrs string
		if err = rows.Scan(&sp.ProjectID, &sp.TraceID, &sp.SpanID, &sp.ParentSpanID, &sp.Name, &sp.Kind, &start, &received, &duration, &sp.Status, &sp.StatusMessage, &sp.Input, &sp.Output, &sp.InputTokens, &sp.OutputTokens, &attrs); err != nil {
			return nil, err
		}
		sp.StartTime = time.UnixMicro(start).UTC()
		sp.ReceivedAt = time.UnixMicro(received).UTC()
		sp.Duration = time.Duration(duration)
		_ = json.Unmarshal([]byte(attrs), &sp.Attributes)
		result = append(result, sp)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) ListTraces(ctx context.Context, q domain.Query) (domain.Page, error) {
	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	where := []string{"project_id = ?"}
	args := []any{q.ProjectID}
	if q.Status != "" {
		where = append(where, "status = ?")
		args = append(args, q.Status)
	}
	if q.Kind != "" {
		where = append(where, "EXISTS (SELECT 1 FROM spans s_kind WHERE s_kind.project_id=trace_summaries.project_id AND s_kind.trace_id=trace_summaries.trace_id AND s_kind.span_kind=?)")
		args = append(args, q.Kind)
	}
	if q.Name != "" {
		where = append(where, "EXISTS (SELECT 1 FROM spans s_name WHERE s_name.project_id=trace_summaries.project_id AND s_name.trace_id=trace_summaries.trace_id AND s_name.name=?)")
		args = append(args, q.Name)
	}
	if q.TraceID != "" {
		where = append(where, "trace_id = ?")
		args = append(args, q.TraceID)
	}
	if q.StartTime != nil {
		where = append(where, "start_time >= ?")
		args = append(args, q.StartTime.UTC().UnixMicro())
	}
	if q.EndTime != nil {
		where = append(where, "start_time <= ?")
		args = append(args, q.EndTime.UTC().UnixMicro())
	}
	if q.MinDuration > 0 {
		where = append(where, "end_time - start_time >= ?")
		args = append(args, q.MinDuration.Microseconds())
	}
	if q.MaxDuration > 0 {
		where = append(where, "end_time - start_time <= ?")
		args = append(args, q.MaxDuration.Microseconds())
	}
	if q.MinTokens > 0 {
		where = append(where, "input_tokens + output_tokens >= ?")
		args = append(args, q.MinTokens)
	}
	if q.MaxTokens > 0 {
		where = append(where, "input_tokens + output_tokens <= ?")
		args = append(args, q.MaxTokens)
	}
	if q.Cursor != "" {
		start, id, err := decodeCursor(q.Cursor)
		if err != nil {
			return domain.Page{}, fmt.Errorf("invalid cursor: %w", err)
		}
		where = append(where, "(start_time < ? OR (start_time = ? AND trace_id < ?))")
		args = append(args, start, start, id)
	}
	args = append(args, limit+1)
	query := `SELECT trace_id, start_time, end_time, span_count, status, input_tokens, output_tokens FROM trace_summaries WHERE ` + strings.Join(where, " AND ") + ` ORDER BY start_time DESC, trace_id DESC LIMIT ?`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return domain.Page{}, err
	}
	defer rows.Close()
	page := domain.Page{}
	for rows.Next() {
		var item domain.Summary
		var start, end sql.NullInt64
		if err := rows.Scan(&item.TraceID, &start, &end, &item.SpanCount, &item.Status, &item.InputTokens, &item.OutputTokens); err != nil {
			return domain.Page{}, err
		}
		if start.Valid {
			item.StartTime = time.UnixMicro(start.Int64).UTC()
		}
		if end.Valid {
			item.EndTime = time.UnixMicro(end.Int64).UTC()
		}
		item.ProjectID = q.ProjectID
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.Page{}, err
	}
	if len(page.Items) > limit {
		last := page.Items[limit-1]
		page.Items = page.Items[:limit]
		page.NextCursor = encodeCursor(last.StartTime.UnixMicro(), last.TraceID)
	}
	return page, nil
}

func encodeCursor(start int64, traceID string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(start, 10) + "\x00" + traceID))
}
func decodeCursor(cursor string) (int64, string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, "", err
	}
	parts := strings.SplitN(string(raw), "\x00", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("malformed cursor")
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	return start, parts[1], err
}
