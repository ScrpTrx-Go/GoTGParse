package database

import (
	"context"
	"fmt"
	"time"

	"github.com/ScrpTrx-Go/GoTGParse/internal/config"
	"github.com/ScrpTrx-Go/GoTGParse/internal/domain/model"
	pkg "github.com/ScrpTrx-Go/GoTGParse/pkg/logger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
	Log  pkg.Logger
}

func MockPostgresPool(pkg.Logger) (d *Database) {
	return &Database{}
}

func NewPostgresPool(log pkg.Logger, cfg config.DatabaseConfig) (d *Database, err error) {
	dsn := cfg.DSN
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	return &Database{
		Pool: pool,
		Log:  log,
	}, nil
}

func (d *Database) SaveBatch(ctx context.Context, in <-chan *model.Post) error {
	posts := make([]*model.Post, 0, 1000)

	for post := range in {
		posts = append(posts, post)
	}

	if len(posts) == 0 {
		d.Log.Info("No posts to save")
		return nil
	}

	rows := make([][]interface{}, 0, len(posts))
	for _, p := range posts {
		rows = append(rows, []interface{}{
			p.ID,
			p.Link,
			p.Text,
			p.Timestamp,
			p.Username,
			p.Regions,
			p.ErrandType,
			p.ErrorType,
		})
	}

	_, err := d.Pool.CopyFrom(
		ctx,
		pgx.Identifier{"posts"},
		[]string{"id", "link", "text", "timestamp", "username", "regions", "errand_type", "error_type"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		d.Log.Error("CopyFrom failed", "err", err)
		return err
	}

	d.Log.Info("Saved posts to database", "count", len(posts))
	return nil
}

func (d *Database) GetMinMaxTimestamps(ctx context.Context) (min time.Time, max time.Time, ok bool, err error) {
	query := `SELECT MIN(timestamp), MAX(timestamp) FROM posts`
	row := d.Pool.QueryRow(ctx, query)

	var minPtr, maxPtr *time.Time
	if err := row.Scan(&minPtr, &maxPtr); err != nil {
		return time.Time{}, time.Time{}, false, err
	}

	if minPtr == nil || maxPtr == nil {
		return time.Time{}, time.Time{}, false, nil
	}

	return *minPtr, *maxPtr, true, nil
}

func (d *Database) GetPostsByPeriod(ctx context.Context, from, to time.Time) ([]*model.Post, error) {
	query := `SELECT id, link, text, timestamp, username, regions, errand_type, error_type
			  FROM posts
			  WHERE timestamp BETWEEN $1 AND $2
			  ORDER BY timestamp ASC`

	rows, err := d.Pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query posts by period: %w", err)
	}
	defer rows.Close()

	var posts []*model.Post
	for rows.Next() {
		var post model.Post
		err := rows.Scan(
			&post.ID,
			&post.Link,
			&post.Text,
			&post.Timestamp,
			&post.Username,
			&post.Regions,
			&post.ErrandType,
			&post.ErrorType,
		)
		if err != nil {
			d.Log.Warn("Failed to scan post", "err", err)
			continue
		}
		posts = append(posts, &post)
	}
	return posts, nil
}
