package migration

import (
	"context"

	"github.com/m-mizutani/fireconf"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/domain/model"
)

// MigrateFirestore creates or updates Firestore indexes using fireconf
func MigrateFirestore(ctx context.Context, projectID, databaseID string) error {
	// Define Firestore configuration
	config := &fireconf.Config{
		Collections: []fireconf.Collection{
			{
				Name: "iocs",
				Indexes: []fireconf.Index{
					{
						// Vector index for semantic search
						Fields: []fireconf.IndexField{
							{
								Path: "Embedding",
								Vector: &fireconf.VectorConfig{
									Dimension: model.EmbeddingDimension, // 128
								},
							},
						},
						QueryScope: fireconf.QueryScopeCollection,
					},
				},
			},
		},
	}

	// Create fireconf client
	client, err := fireconf.NewClient(ctx, projectID, databaseID)
	if err != nil {
		return goerr.Wrap(err, "failed to create fireconf client",
			goerr.V("project_id", projectID),
			goerr.V("database_id", databaseID))
	}
	defer client.Close()

	// Apply configuration
	if err := client.Migrate(ctx, config); err != nil {
		return goerr.Wrap(err, "failed to migrate firestore",
			goerr.V("project_id", projectID),
			goerr.V("database_id", databaseID))
	}

	return nil
}
