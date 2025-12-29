package cli

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/beehive/pkg/cli/config"
	"github.com/secmon-lab/beehive/pkg/infra/migration"
	"github.com/secmon-lab/beehive/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func cmdMigrate() *cli.Command {
	var firestoreCfg config.Firestore

	return &cli.Command{
		Name:  "migrate",
		Usage: "Migrate Firestore schema (create vector indexes)",
		Flags: firestoreCfg.Flags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			logger := logging.From(ctx)

			if firestoreCfg.ProjectID == "" {
				return goerr.New("firestore-project-id is required")
			}

			logger.Info("Starting Firestore migration",
				"project_id", firestoreCfg.ProjectID,
				"database_id", firestoreCfg.DatabaseID)

			if err := migration.MigrateFirestore(ctx, firestoreCfg.ProjectID, firestoreCfg.DatabaseID); err != nil {
				return goerr.Wrap(err, "migration failed")
			}

			logger.Info("Migration completed successfully")
			return nil
		},
	}
}
