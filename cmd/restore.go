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
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"strings"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

var directory string

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:     "restore",
	Aliases: []string{"r"},
	Short:   "Restore from dump file",
	Run: func(cmd *cobra.Command, args []string) {
		err := restoreFromDir(directory)
		if err != nil {
			fmt.Printf("Error while restoring from dir : %s", err)
		}
	},
}

func restoreFromDir(dir string) error {

	_, err := os.Stat(dir)
	if err != nil {
		log.Printf("Error while retriving os.Stat infos : %s", err)
		return fmt.Errorf("Directory %s does not exists", dir)
	}

	defaultDbConn := createConnection(defaultDbUrl)
	defer defaultDbConn.Close(context.Background())

	backupSelected, err := PickFileIn(dir)
	if err != nil {
		return err
	}

	dumpPath := filepath.Join(dir, backupSelected)

	fmt.Printf("Start restore from dump file %v\nPlease wait...", dumpPath)

	if DatabaseExists(defaultDbConn, getProjectName()) {
		TerminateDatabaseConnections(defaultDbConn, getProjectName())
		DropDatabase(defaultDbConn, getProjectName())
	}
	CreateDatabase(defaultDbConn, getProjectName())
	restoreDatabase(dumpPath, trackedDbUrl)

	return nil
}

func DatabaseExists(conn *pgx.Conn, database string) bool {
	var exists bool

	query := fmt.Sprintf("select exists(SELECT datname FROM pg_catalog.pg_database WHERE lower(datname) = lower('%s'));", database)
	log.Print(query)

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

func PickFileIn(dir string) (string, error) {
	var Selector []string

	completeList, err := ioutil.ReadDir(dir)
	for i := len(completeList)/2 - 1; i >= 0; i-- {
		opp := len(completeList) - 1 - i
		completeList[i], completeList[opp] = completeList[opp], completeList[i]
	}
	if err != nil {
		log.Fatal(err)
	}
	if len(completeList) == 0 {
		return "", fmt.Errorf("Directory exists but is empty")
	}

	for _, localbackup := range completeList {
		//fmt.Printf("%s | %d\n", backup.key, backup.size)
		Selector = append(Selector, localbackup.Name())
	}
	backupSelected := ""
	prompt := &survey.Select{
		Message: fmt.Sprintf("Select local backup file in %s/:", viper.GetString("from-dir")),
		Options: Selector,
	}
	survey.AskOne(prompt, &backupSelected, nil)
	if backupSelected == "" {
		log.Fatal("No backup selected")
	}
	return backupSelected, nil
}

// TerminateDatabaseConnections force cuts all connections to database before drop or create operations
func TerminateDatabaseConnections(conn *pgx.Conn, database string) error {

	//server_version = raw_conn.execute('SHOW server_version;').first()[0]
	//version_string, _, _ = server_version.partition(' ')
	//version = [int(x) for x in version_string.split('.')]
	//return 'pid' if version >= [9, 2] else 'procpid'

	sqlTerminate := fmt.Sprintf(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%[1]s';`, database)
	log.Println(sqlTerminate)

	_, err := conn.Exec(context.Background(), sqlTerminate)
	if err != nil {
		return err
	}
	return nil
}

func restoreDatabase(dumpPath string, connUrl string) {

	// Check command is available
	ensurepath("pg_restore")
	c, err := url.Parse(connUrl)
	if err != nil {
		log.Printf("Unable to parse conn url : %s", err)
	}
	host, port, _ := net.SplitHostPort(c.Host)

	log.Printf("Start restore dump %v into database %v\nPlease wait ...\n", dumpPath, config.Database)
	args := fmt.Sprintf("--host=%s --port=%s --username=%s --verbose --clean --disable-triggers --no-acl --no-owner -d %s %s", host, port, c.User.Username(), getProjectName(), dumpPath)
	cmd := exec.Command("pg_restore", strings.Split(args, " ")...)

	stderr, _ := cmd.StderrPipe()

	err = cmd.Start()
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
	restoreCmd.PersistentFlags().StringVar(&directory, "dir", ".cappa", "Directory to look dumps files for")
}
