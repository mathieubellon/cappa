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
	"github.com/spf13/cobra"
	"github.com/ttacon/chalk"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

var anonymisationQueries = []string{
	`update pilot_users_pilotuser set email='pilotuser' || id || '@pilot.pm';`,
	`update pilot_users_pilotuser set password='pbkdf2_sha256$10000$s1w0UXDd00XB$+4ORmyvVWAQvoAEWlDgN34vlaJx1ZTZpa1pCSRey2Yk=';`,
	`update pilot_users_pilotuser set password='pbkdf2_sha256$10000$s1w0UXDd00XB$+4ORmyvVWAQvoAEWlDgN34vlaJx1ZTZpa1pCSRey2Yk=';`,
	`update pilot_users_pilotuser set first_name = initcap(lower(translate(first_name, 'aeiouyAEIOUYctsrnm', 'eaeaeiEAEAAdsmtbv')));`,
	`update pilot_users_pilotuser set last_name = initcap(lower(translate(last_name, 'aeiouyAEIOUYctsrnm', 'eaeaeiEAEAAdsmtbv')));`,
	`update pilot_users_pilotuser set username='pilotuser' || id;`,
	`update pilot_users_pilotuser set phone='01.23.45.67.89';`,
	`update channels_twittercredential set access_token_key=random(), access_token_secret=random();`,
	`update channels_facebookcredential set access_token=random(), facebook_page_token=random();`,
}

// executeCmd represents the execute command
var executeCmd = &cobra.Command{
	Use:   "execute",
	Short: "Execute sql from file (default '.cappa/execute.sql')",
	Long: `This can be useful if you need to alter data in your working database after you restored
a dump file from production

Write sql statements in a file (default '.cappa.sql') separate by newline and run this command.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Command `execute` called")
		sqlFileName := "execute.sql"

		// open sql file
		dat, err := ioutil.ReadFile(path.Join(".cappa", sqlFileName))
		if os.IsNotExist(err) {
			fmt.Println(chalk.Bold.TextStyle("File does not exists, create a .cappa/execute.sql file"))
			os.Exit(0)
		}
		if err != nil {
			fmt.Println("Error : Could not open sql file")
			log.Fatal(err)
		}

		//Split content at newline
		Sqls := strings.Split(string(dat), "\n")

		// Create connection to primary database
		primaryConn := createConnection(trackedDbUrl)
		defer primaryConn.Close(context.Background())

		executionCount := 0
		for _, sql := range Sqls {
			if sql == "" {
				continue
			}
			log.Printf("Execute : %s", sql)

			_, err = primaryConn.Exec(context.Background(), sql)
			if err != nil {
				fmt.Printf("Error executing the statement : %s", err)
			}
			executionCount += 1
		}
		if executionCount > 0 {
			fmt.Println(chalk.Green.Color("All statements executed"))
		} else {
			fmt.Println(chalk.Yellow.Color("Nothing done, file empty ?"))
		}

	},
}

func init() {
	rootCmd.AddCommand(executeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// executeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// executeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
