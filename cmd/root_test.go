package cmd

import (
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
)

var db *sql.DB
var database = "postgres"
var err error
var testDir string

var testConfig = Config{
	Username: "postgres",
	Password: "secret",
	Host:     "127.0.0.1",
	Port:     "5432",
	Database: "devproject",
	Project:  "testproject",
}

func TestMain(m *testing.M) {

	// Go test has path relative to package instead of root package
	// So we alter this behavior here
	_, filename, _, _ := runtime.Caller(0)
	rootdir := path.Join(path.Dir(filename), "..")
	err := os.Chdir(rootdir)
	if err != nil {
		panic(err)
	}

	// We create a test dir like a real project situation
	testDir, err = ioutil.TempDir(".", "testdir")
	if err != nil {
		panic(err)
	}
	os.Chdir(testDir)

	// Setup docker with database
	setup()

	// Run all tests
	code := m.Run()

	// Teardown
	teardown()

	os.Exit(code)
}

func setup() {
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", testConfig.Username, testConfig.Password, testConfig.Host, "5432", "postgres")
	fmt.Printf("connexion url %s", url)
	db, err = sql.Open("postgres", url)
	if err != nil {
		log.Fatalf("Cannot open connection to database : %s\n", err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalf("Cannot ping the database : %s\n", err)
	}
	fmt.Printf("\033[1;36m%s\033[0m", "> Setup completed\n")
}

// GetDeletionQueries select all databases to delete following a pattern and return
// a list of sql statements ready to be executed
func GetDeletionQueries() ([]string, error) {
	var deletionQueries []string
	var deleteNonTemplateDat = `select 'drop database "'||datname||'";' from pg_database where datistemplate=false AND datname!='postgres';`

	rows, err := db.Query(deleteNonTemplateDat)
	if err != nil {
		return deletionQueries, err
	}
	defer rows.Close()

	for rows.Next() {
		var deleteQuery string
		err := rows.Scan(&deleteQuery)
		if err != nil {
			return deletionQueries, err
		}
		deletionQueries = append(deletionQueries, deleteQuery)
	}
	fmt.Printf("rows %s", deletionQueries)
	return deletionQueries, nil
}

func teardown() {
	//
	queries, err := GetDeletionQueries()
	if err != nil {
		log.Printf("Error while getting all delete statements : %s", err)
	}
	// Execute each deletion query
	for _, query := range queries {
		row := db.QueryRow(query)
		switch err := row.Scan(); err {
		case sql.ErrNoRows:
			fmt.Println("No rows were returned!")
		case nil:
			fmt.Printf("Execute OK %s", query)
		default:
			panic(err)
		}
	}

	err = os.RemoveAll(testDir)
	if err != nil {
		fmt.Printf("Cannot delete dir : %s", err)
	}
	fmt.Printf("Delet this dir %s", testDir)

	fmt.Printf("\033[1;36m%s\033[0m", "> Teardown completed\n")
}

func NewRootCmd() *cobra.Command {
	return rootCmd
}

type fakeConfig struct {
	Config
}

func (f *fakeConfig) create() {
	_, err := os.Create(".cappa.toml")
	if err != nil {
		log.Fatal("Error creating empty fake config file")
	}
}

func (f *fakeConfig) createReal() {
	_, err := os.Create(".cappa.toml")
	if err != nil {
		log.Fatal("Error creating empty fake config file")
	}
}

func (f *fakeConfig) remove() error {
	err := os.Remove(".cappa.toml")
	if err != nil {
		return err
	}
	return nil
}

func Test_ExecuteRootCommand(t *testing.T) {
	cmd := NewRootCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.Execute()
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}
	subString := "Cappa allows you to take fast snapshots / restore of your development database"
	if !strings.Contains(string(out), subString) {
		t.Fatalf("expected output to contain \"%s\" in \"%s\"", subString, string(out))
	}
}
