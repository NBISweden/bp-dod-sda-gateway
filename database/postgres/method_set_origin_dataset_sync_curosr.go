package postgres

import (
	"context"
	"database/sql"
	"time"
)

const setOriginDatasetSyncCursorQuery = "setOriginDatasetSyncCursor"

func init() {
	queries[setOriginDatasetSyncCursorQuery] = `UPDATE origin_dataset_sync_cursor 
SET cursor = $1;`
}
func (db *pgDb) setOriginDatasetSyncCursor(ctx context.Context, tx *sql.Tx, cursor time.Time) error {
	stmt, err := db.getPreparedStmt(tx, setOriginDatasetSyncCursorQuery)
	if err != nil {
		return err
	}

	if _, err := stmt.ExecContext(ctx, cursor); err != nil {
		return err
	}

	return nil
}
