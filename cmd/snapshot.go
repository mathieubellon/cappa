package cmd

import (
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/jackc/pgx/v4"
	"github.com/lithammer/shortuuid/v3"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// snapshotCmd represents the snapshot command
var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Snapshot database",
	RunE: func(cmd *cobra.Command, args []string) error {
		var snapshotName string
		snapUuid := shortuuid.New()
		toDatabase := fmt.Sprintf("%s_%s", cliName, strings.ToLower(snapUuid))

		// Ask user for snapshot name
		prompt := &survey.Input{
			Message: "Name of snapshot",
		}
		err := survey.AskOne(prompt, &snapshotName, survey.WithValidator(survey.Required))
		if err == terminal.InterruptErr {
			fmt.Println("User terminated prompt")
			return nil
		} else if err != nil {
			log.Fatal(err)
		}

		// Create raw db connexion for copy operation
		rawConn := createConnection(config, "")
		defer rawConn.Close(context.Background())

		// terminate connexion of source DB before copy
		err = TerminateDatabaseConnections(rawConn, config.Database)
		if err != nil {
			log.Fatalf("Impossible to terminate DB connexion : %s", err)
		}

		// Copy source DB to snapshot DB
		copy_database(rawConn, config.Database, toDatabase)

		// After (and only after) snapshot DB is created we create tracked db informations
		trackerConn := createConnection(config, cliName)
		defer trackerConn.Close(context.Background())
		insertSql := fmt.Sprintf("INSERT INTO snapshots (hash, name, project) VALUES ('%s', '%s', '%s');", strings.ToLower(snapUuid), snapshotName, config.Project)
		log.Print(insertSql)

		_, err = trackerConn.Exec(context.Background(), insertSql)
		if err != nil {
			log.Fatalf("Error inserting snapshot infos : %s", err)
		}

		fmt.Printf("Snapshot created from %s to %s\n", config.Database, snapshotName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(snapshotCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// snapshotCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// snapshotCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func copy_database(conn *pgx.Conn, from_database string, to_database string) {
	query := fmt.Sprintf(`CREATE DATABASE "%s" WITH TEMPLATE "%s";`, to_database, from_database)
	log.Print(query)

	_, err := conn.Exec(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Copy database failed: %v\n", err)
		log.Fatal(err)
	}
}
