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
	"bufio"
	"context"
	"fmt"
	"github.com/AlecAivazis/survey/v2"
	"github.com/jackc/pgx/v4"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	_ "github.com/lib/pq"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:     "restore",
	Aliases: []string{"r"},
	Short:   "Restore a backup into dev DB",
	Long:    ``,
	Run: func(cmd *cobra.Command, args []string) {
		conn := createConnection(config, "")
		defer conn.Close(context.Background())

		if DatabaseExists(conn, config.Database) {
			TerminateDatabaseConnections(conn, config.Database)
			DropDatabase(conn, config.Database)
		}

		CreateDatabase(conn, config.Database)

		backupSelected := PickFileIn(config.BackupDir)

		dumpPath := filepath.Join(config.BackupDir, backupSelected)

		log.Printf("Selected backup to restore : %v", dumpPath)

		restoreDatabase(dumpPath, config, config.Database)

	},
}

func DatabaseExists(conn *pgx.Conn, database string) bool {
	var exists bool

	query := fmt.Sprintf("select exists(SELECT datname FROM pg_catalog.pg_database WHERE lower(datname) = lower('%s'));", database)
	log.Printf("execute this query %s", query)

	err := conn.QueryRow(context.Background(), query).Scan(&exists)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to check if database exists: %v\n", err)
	}

	log.Printf("Database %s exists? : %v", database, exists)

	return exists
}

func ensurepath(command string) string {
	_, err := exec.LookPath(command)
	if err != nil {
		log.Fatalf("%v is not found", command)
		panic(err)
	}
	return command
}

func PickFileIn(dir string) string {
	var Selector []string

	completeList, err := ioutil.ReadDir(dir)
	for i := len(completeList)/2 - 1; i >= 0; i-- {
		opp := len(completeList) - 1 - i
		completeList[i], completeList[opp] = completeList[opp], completeList[i]
	}
	if err != nil {
		log.Fatal(err)
	}

	for _, localbackup := range completeList {
		//fmt.Printf("%s | %d\n", backup.key, backup.size)
		Selector = append(Selector, localbackup.Name())
	}
	backupSelected := ""
	prompt := &survey.Select{
		Message: "Select local backup file:",
		Options: Selector,
	}
	survey.AskOne(prompt, &backupSelected, nil)
	if backupSelected == "" {
		log.Fatal("No backup selected")
	}
	return backupSelected
}

// TerminateDatabaseConnections force cuts all connections to database before drop or create operations
func TerminateDatabaseConnections(conn *pgx.Conn, database string) {

	//server_version = raw_conn.execute('SHOW server_version;').first()[0]
	//version_string, _, _ = server_version.partition(' ')
	//version = [int(x) for x in version_string.split('.')]
	//return 'pid' if version >= [9, 2] else 'procpid'

	log.Printf("Terminate connexions for %s", database)

	sqlTerminate := fmt.Sprintf(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = %[1]s;`, database)

	log.Println(sqlTerminate)

	_, err := conn.Exec(context.Background(), sqlTerminate)
	if err != nil {
		log.Printf("Failed to terminate database connections: %v\n", err)
	}

}

func restoreDatabase(dumpPath string, config Config, database string) {

	// Check command is available
	ensurepath("pg_restore")

	log.Printf("Start restore dump %v into database %v\nPlease wait ...\n", dumpPath, config.Database)
	args := fmt.Sprintf("--host=%s --port=%s --username=%s --verbose --clean --disable-triggers --no-acl --no-owner -d %s %s", config.Host, config.Port, config.Username, config.Database, dumpPath)
	cmd := exec.Command("pg_restore", strings.Split(args, " ")...)

	stderr, _ := cmd.StderrPipe()

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Could not start command : %s", err)
	}

	scanner := bufio.NewScanner(stderr)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}

	cmd.Wait()
	if err != nil {
		log.Fatalf("Could not wait for command : %s", err)
	}

}

func DropDatabase(conn *pgx.Conn, database string) {
	query := fmt.Sprintf("DROP DATABASE %s;", database)
	log.Print(query)

	_, err := conn.Exec(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Drop database failed: %v\n", err)
		log.Fatal(err)
	}
}

func CreateDatabase(conn *pgx.Conn, database string) {
	query := fmt.Sprintf("CREATE DATABASE %s", database)
	log.Print(query)
	_, err := conn.Exec(context.Background(), query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Create database failed: %v\n", err)
		log.Fatal(err)
	}
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.PersistentFlags().String("host", "", "database server host")
	restoreCmd.PersistentFlags().String("port", "", "database server port")
	restoreCmd.PersistentFlags().String("username", "", "database user name")
	restoreCmd.PersistentFlags().String("password", "", "database user password")
	restoreCmd.PersistentFlags().String("database", "", "database name")
	restoreCmd.PersistentFlags().String("backup_dir", "", "Local directory where to look for local backup files")
	viper.BindPFlag("host", restoreCmd.PersistentFlags().Lookup("host"))
	viper.BindPFlag("port", restoreCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("username", restoreCmd.PersistentFlags().Lookup("username"))
	viper.BindPFlag("password", restoreCmd.PersistentFlags().Lookup("password"))
	viper.BindPFlag("database", restoreCmd.PersistentFlags().Lookup("database"))
	viper.BindPFlag("backup_dir", restoreCmd.PersistentFlags().Lookup("backup_dir"))
}
