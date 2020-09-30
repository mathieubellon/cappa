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
	"github.com/ory/dockertest/v3"
	"github.com/spf13/cobra"
)

var db *sql.DB
var pool *dockertest.Pool
var resource *dockertest.Resource
var database = "postgres"
var err error
var testDir string

var testConfig = Config{
	Username: "postgres",
	Password: "secret",
	Host:     "localhost",
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

	//setupDockerDb()
	code := m.Run()
	//teardown()
	os.Exit(code)
}

func setupDockerDb() {
	pool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	resource, err = pool.Run("postgres", "9.6", []string{"POSTGRES_PASSWORD=" + testConfig.Password, "POSTGRES_DB=" + database})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	if err = pool.Retry(func() error {
		var err error
		db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", testConfig.Username, testConfig.Password, testConfig.Host, resource.GetPort("5432/tcp"), database))
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	fmt.Printf("\033[1;36m%s\033[0m", "> Setup completed\n")
}

func teardown() {
	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}
	err = os.RemoveAll(testDir)
	if err != nil {
		log.Fatalf("error removing testDir")
	}
	fmt.Printf("\033[1;36m%s\033[0m", "> Teardown completed\n")
}

func NewRootCmd() *cobra.Command {
	return rootCmd
}

type fakeConfig struct {
	Config
}

func (f *fakeConfig) create() error {
	_, err := os.Create(".cappa.toml")
	if err != nil {
		return err
	}
	return nil
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
