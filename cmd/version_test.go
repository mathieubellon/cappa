package cmd

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"
)

func Test_ExecuteVersionCommand(t *testing.T) {
	cmd := NewRootCmd()
	b := bytes.NewBufferString("")
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{"version"})
	cmd.Execute()
	out, err := ioutil.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}
	subString := "Cappa"
	if !strings.Contains(string(out), subString) {
		t.Fatalf("expected output to contain \"%s\" in \"%s\"", subString, string(out))
	}
}
