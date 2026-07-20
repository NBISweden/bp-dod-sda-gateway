package on_demand_dataset_metadata_file_handler

import (
	"context"
	"fmt"
)

// fileInfo based on https://github.com/neicnordic/sensitive-data-archive/blob/main/sda/cmd/api/swagger_v1.yml#L505
// only relevant fields are specified
type fileInfo struct {
	FileId string `json:"fileID"`

	InboxPath string `json:"inboxPath"`

	Status string `json:"fileStatus"`
}

func paginate[T any](
	ctx context.Context,
	fetch func(ctx context.Context, cursor string) ([]T, string, error),
) ([]T, error) {
	var all []T
	var nextCursor string // nil = first call
	seen := make(map[string]struct{})
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch, next, err := fetch(ctx, nextCursor)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if next == "" {
			return all, nil
		}
		if _, dup := seen[next]; dup {
			return nil, fmt.Errorf("pagination aborted: server returned repeated pageToken %q", next)
		}
		seen[next] = struct{}{}
		nextCursor = next
	}
}
