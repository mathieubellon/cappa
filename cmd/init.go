/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/spf13/viper"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize cappa application",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("init called")
		conn := createConnection(config, "")
		defer conn.Close(context.Background())
		createTrackerDb(conn)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.PersistentFlags().String("host", "", "database server host")
	initCmd.PersistentFlags().String("port", "", "database server port")
	initCmd.PersistentFlags().String("username", "", "database user name")
	initCmd.PersistentFlags().String("password", "", "database user password")
	initCmd.PersistentFlags().String("database", "", "database name")
	viper.BindPFlag("host", initCmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag("port", initCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("username", initCmd.PersistentFlags().Lookup("username"))
	viper.BindPFlag("password", initCmd.PersistentFlags().Lookup("password"))
	viper.BindPFlag("database", initCmd.PersistentFlags().Lookup("database"))
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
}

// This function create the database for tracking snapshots
func createTrackerDb(conn *pgx.Conn) {
	structureSql := `CREATE TABLE snapshots (id SERIAL PRIMARY KEY, hash TEXT UNIQUE NOT NULL, name TEXT NOT NULL,project TEXT NOT NULL, created_at timestamp not null default CURRENT_TIMESTAMP);`
	if !DatabaseExists(conn, cliName) {
		CreateDatabase(conn, cliName)

		trackerConn := createConnection(config, cliName)
		defer trackerConn.Close(context.Background())

		log.Print(structureSql)
		_, err := trackerConn.Exec(context.Background(), structureSql)
		if err != nil {
			log.Fatalf("Failed to created cli database: %v\n", err)
		}
		fmt.Printf("Database %s successfully created", cliName)
	}
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
