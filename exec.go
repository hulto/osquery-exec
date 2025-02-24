package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	osquery "github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func main() {
	tries := 10
	for i := 0; i < tries; i++ {
		info, err := os.Stat("/opt/orbit/orbit-osquery.em")
		if os.IsNotExist(err) {
			time.Sleep(3 * time.Second)
			continue
		}
		if !info.IsDir() {
			break
		}
	}

	server, err := osquery.NewExtensionManagerServer("exec", "/opt/orbit/orbit-osquery.em")
	if err != nil {
		log.Fatalf("Error creating extension: %s\n", err)
	}

	// Create and register a new table plugin with the server.
	// table.NewPlugin requires the table plugin name,
	// a slice of Columns and a Generate function.
	server.RegisterPlugin(table.NewPlugin("exec", ExecColumns(), ExecGenerate))
	if err := server.Run(); err != nil {
		log.Fatalln(err)
	}
}

// ExecColumns returns the columns that our table will return.
func ExecColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("cmd"),
		table.TextColumn("stdout"),
		table.TextColumn("stderr"),
		table.TextColumn("code"),
	}
}

// ExecGenerate will be called whenever the table is queried. It should return
// a full table scan.
func ExecGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	if cnstList, present := queryContext.Constraints["cmd"]; present {
		// If 'cmd' is present in queryContext.Contraints's keys
		// translate: if 'cmd' is in the WHERE clause

		for _, cnst := range cnstList.Constraints {
			if cnst.Operator == table.OperatorEquals {
				cmdArr := strings.Split(cnst.Expression, " ")
				out, err, code := execute(cmdArr[0], cmdArr[1:]...)
				return []map[string]string{
					{
						"cmd":    cnst.Expression,
						"stdout": out,
						"stderr": err,
						"code":   strconv.Itoa(code),
					},
				}, nil
			}
		}
	}
	return nil, errors.New("Query to table exec must have a WHERE clause on 'cmd'")
}

func execute(bin string, args ...string) (string, string, int) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return strings.Trim(stdout.String(), " \n"), strings.Trim(stderr.String(), " \n"), exitError.ExitCode()
		}
		return "", err.Error(), -1
	}
	return stdout.String(), stderr.String(), 0
}
