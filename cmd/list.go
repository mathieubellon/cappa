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
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ttacon/chalk"
	"github.com/xeonx/timeago"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

type Snapshot struct {
	Id        int
	Hash      string
	Name      string
	CreatedAt time.Time
}

func (s *Snapshot) TimeAgo() string {
	return timeago.English.Format(s.CreatedAt)
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use: "list" +
		"",
	Short: "List your snapshots",
	Run: func(cmd *cobra.Command, args []string) {

		cliDbConn := createConnection(cliDbUrl)
		defer cliDbConn.Close(context.Background())

		list, err := listSnapshots(cliDbConn)
		if err != nil {
			log.Printf("Could not list snapshots : %s", err)
		}

		if len(list) == 0 {
			fmt.Println(chalk.Yellow.Color("No snapshots, run 'cappa snapshot'"))
			return
		}

		sort.Slice(list, func(i, j int) bool {
			return list[j].CreatedAt.Before(list[i].CreatedAt)
		})

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Created"})
		table.SetBorder(false)

		for _, snap := range list {
			value := []string{
				snap.Name,
				timeago.English.Format(snap.CreatedAt),
			}
			table.Append(value)
		}
		table.Render() // Send output
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getProjectName() string {

	databaseURL := viper.GetString("database_url")
	u, err := url.Parse(databaseURL)
	if err != nil {
		log.Printf("error parsing database_url : %s", err)
	}
	project := strings.Split(u.Path, "/")[1]
	log.Printf("projectName sets to : %s\n", project)
	if project == "" {
		fmt.Println("Error trying to get project name from config (did you set project value?)")
	}
	return project
}

func listSnapshots(conn *pgx.Conn) ([]Snapshot, error) {

	selectQuery := fmt.Sprintf("SELECT id, hash, name, created_at FROM snapshots WHERE project = '%s';", getProjectName())
	log.Println(selectQuery)

	rows, err := conn.Query(context.Background(), selectQuery)
	if err != nil {
		log.Printf("Select Query Error : %s", err)
	}

	var list []Snapshot

	for rows.Next() {
		var id int
		var hash string
		var name string
		var createdAt time.Time
		err := rows.Scan(&id, &hash, &name, &createdAt)
		if err != nil {
			return nil, err
		}
		list = append(list, Snapshot{Id: id, Hash: hash, Name: name, CreatedAt: createdAt})
	}

	return list, rows.Err()
}
