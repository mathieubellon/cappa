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
	Run: func(cmd *cobra.Command, args []string) {
		var snapshotName string

		prompt := &survey.Input{
			Message: "Name of snapshot",
		}

		err := survey.AskOne(prompt, &snapshotName, survey.WithValidator(survey.Required))
		if err == terminal.InterruptErr {
			fmt.Println("User terminated prompt")
			os.Exit(0)
		} else if err != nil {
			panic(err)
		}

		if snapshotName != "" {
			rawConn := createConnection(config, "")
			defer rawConn.Close(context.Background())

			log.Printf("Will create snapshot from %s to %s", config.Database, snapshotName)

			snapuuid := shortuuid.New()

			trackerConn := createConnection(config, cliName)
			defer trackerConn.Close(context.Background())
			insertSql := fmt.Sprintf("INSERT INTO snapshots (hash, name) VALUES ('%s', '%s');", strings.ToLower(snapuuid), snapshotName)
			log.Print(insertSql)

			inserted, err := trackerConn.Exec(context.Background(), insertSql)
			if err != nil {
				log.Fatalf("Error inserting snapshot infos : %s", err)
			}
			log.Printf("%v", inserted)

			TerminateDatabaseConnections(rawConn, config.Database)
			toDatabase := fmt.Sprintf("%s_%s", cliName, strings.ToLower(snapuuid))
			copy_database(rawConn, config.Database, toDatabase)
		}
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
	log.Printf("execute this query %s", query)

	_, err := conn.Exec(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Copy database failed: %v\n", err)
		log.Fatal(err)
	}
}
