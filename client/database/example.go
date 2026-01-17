package database

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/zoff-music/cibes/example"
	"github.com/zoff-music/cibes/monitoring/opentracing"
	"github.com/jmoiron/sqlx"
	"time"
)

func (c *Client) prepareRecordExampleDataStmt() error {
	stmt, err := c.DB.Preparex(`
		INSERT INTO 
			examples (is_fake, creation_date)
	    VALUES ($1, $2)
	`)
	if err != nil {
		return fmt.Errorf("error preparing record example data statement: %w", err)
	}
	c.RecordExampleDataStatement = stmt
	return nil
}

func (c *Client) RecordExampleData(ctx context.Context, exampleData example.Data) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RecordExampleData")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	_, err := c.RecordExampleDataStatement.QueryxContext(
		cctx,
		exampleData.IsFake,
		exampleData.Date.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("record example data failed: %w", err)
	}

	return nil
}

func (c *Client) prepareGetExampleDataStmt() error {
	stmt, err := c.DB.Preparex(`
		SELECT
			is_fake,
		    creation_date
	    FROM examples
		WHERE id = $1
	`)
	if err != nil {
		return fmt.Errorf("error preparing get example data statement: %w", err)
	}
	c.GetExampleDataStatement = stmt
	return nil
}

func (c *Client) GetAllExampleData(ctx context.Context) ([]example.Data, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GetAllExampleData")
	defer span.Finish()

	cctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	rows, err := c.GetExampleDataStatement.QueryxContext(cctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("error getting all example data from database, no rows exist: %w", err)
		}
		return nil, fmt.Errorf("error getting all example data from database: %w", err)
	}
	defer rows.Close()

	var allExampleData []example.Data
	for rows.Next() {
		var row exampleDataRow

		err = row.scan(rows)
		if err != nil {
			return nil, fmt.Errorf("error scanning example data row: %w", err)
		}

		exData := row.toExampleData()
		allExampleData = append(allExampleData, exData)
	}

	return allExampleData, nil
}

type exampleDataRow struct {
	IsFake sql.NullBool
	Date   sql.NullTime
}

func (e *exampleDataRow) scan(rows *sqlx.Rows) error {
	return rows.Scan(
		&e.IsFake,
		&e.Date,
	)
}

func (e *exampleDataRow) toExampleData() example.Data {
	return example.Data{
		IsFake: e.IsFake.Bool,
		Date:   e.Date.Time,
	}
}
