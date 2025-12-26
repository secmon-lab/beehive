package config

import (
	"github.com/urfave/cli/v3"
)

// Firestore represents Firestore configuration
type Firestore struct {
	ProjectID  string
	DatabaseID string
}

// Flags returns CLI flags for Firestore configuration
func (f *Firestore) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "firestore-project-id",
			Usage:       "Google Cloud project ID for Firestore",
			Destination: &f.ProjectID,
			Sources:     cli.EnvVars("BEEHIVE_FIRESTORE_PROJECT_ID", "GOOGLE_CLOUD_PROJECT"),
		},
		&cli.StringFlag{
			Name:        "firestore-database-id",
			Usage:       "Firestore database ID (default database if not specified)",
			Destination: &f.DatabaseID,
			Sources:     cli.EnvVars("BEEHIVE_FIRESTORE_DATABASE_ID"),
		},
	}
}
