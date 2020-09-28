package cmd

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	return rootCmd
}

func Test_ExecuteCommand(t *testing.T) {
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
