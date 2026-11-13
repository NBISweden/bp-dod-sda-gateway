package postgres

import (
	"context"
	"database/sql"
	"time"
)

const getOriginDatasetSyncCursorQuery = "gettOriginDatasetSyncCursor"

func init() {
	queries[getOriginDatasetSyncCursorQuery] = `SELECT cursor 
FROM origin_dataset_sync_cursor;`
}
func (db *pgDb) getOriginDatasetSyncCursor(ctx context.Context, tx *sql.Tx) (time.Time, error) {
	stmt, err := db.getPreparedStmt(tx, getOriginDatasetSyncCursorQuery)
	if err != nil {
		return time.Time{}, err
	}

	var cursor time.Time
	if err := stmt.QueryRow().Scan(&cursor); err != nil {
		return time.Time{}, err
	}

	return cursor, nil
}
