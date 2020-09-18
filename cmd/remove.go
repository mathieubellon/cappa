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
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"log"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove snapshot",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("remove called")

		trackerConn := createConnection(config, cliName)
		defer trackerConn.Close(context.Background())

		snapshots, err := listSnapshots(trackerConn)
		if err != nil {
			log.Fatalf("Error while listing snasphot for removal", err)
		}

		templates := &promptui.SelectTemplates{
			Label:    "{{ . | red }}",
			Active:   "\U0001F336 {{ .Name | cyan }} ({{ .Hash | yellow }})",
			Inactive: "  {{ .Name | cyan }} ({{ .Hash | yellow }})",
			Selected: "\U0001F336 {{ .Name | red | cyan }}",
			Details: `
--------- Snapshot ----------
{{ "Name:" | faint }}	{{ .Name }}
{{ "Hash:" | faint }}	{{ .Hash }}
{{ "Created at:" | faint }}	{{ .CreatedAt }}`,
		}

		searcher := func(input string, index int) bool {
			snapshot := snapshots[index]
			name := strings.Replace(strings.ToLower(snapshot.Name), " ", "", -1)
			input = strings.Replace(strings.ToLower(input), " ", "", -1)

			return strings.Contains(name, input)
		}

		prompt := promptui.Select{
			Label:     "Select snapshot to delete",
			Items:     snapshots,
			Templates: templates,
			Size:      5,
			Searcher:  searcher,
		}

		i, _, err := prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		fmt.Printf("You choose number %d: %s %v\n", i+1, snapshots[i].Name, snapshots[i])

		snap := snapshots[i]

		rawConn := createConnection(config, cliName)
		defer rawConn.Close(context.Background())

		databaseToDrop := fmt.Sprintf("%s_%s", cliName, snap.Hash)

		TerminateDatabaseConnections(rawConn, databaseToDrop)
		DropDatabase(rawConn, databaseToDrop)

		deleteSql := fmt.Sprintf("DELETE FROM snapshots WHERE id=%d;", snap.Id)
		log.Print(deleteSql)
		_, err = trackerConn.Exec(context.Background(), deleteSql)
		if err != nil {
			log.Fatalf("Error deleting snapshot infos : %s", err)
		}
		log.Printf("Successfully deleted snapshot %v", snap)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// removeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// removeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
