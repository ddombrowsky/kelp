package kelpdb

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// insertParts splits an "INSERT INTO t (cols) VALUES (vals)" template into its column names and value placeholders
func insertParts(t *testing.T, template string) ([]string, []string) {
	m := regexp.MustCompile(`^INSERT INTO \w+ \(([^)]*)\) VALUES \(([^)]*)\)$`).FindStringSubmatch(template)
	if !assert.NotNil(t, m, "template is not a single-row insert: %s", template) {
		return nil, nil
	}
	return strings.Split(m[1], ", "), strings.Split(m[2], ", ")
}

// the insert templates are filled with fmt.Sprintf, so the number of placeholders must match the number of columns
func TestInsertTemplatesHaveOnePlaceholderPerColumn(t *testing.T) {
	for name, template := range map[string]string{
		"SqlMarketsInsertTemplate": SqlMarketsInsertTemplate,
		"SqlTradesInsertTemplate":  SqlTradesInsertTemplate,
	} {
		columns, values := insertParts(t, template)
		assert.Equal(t, len(columns), len(values), name)
		for _, v := range values {
			assert.Regexp(t, `^'?%(s|\.15f)'?$`, v, "%s has an unexpected placeholder %q", name, v)
		}
	}
}

// the trades insert must write every column created by the table definition and its alter statements
func TestTradesInsertCoversAllColumns(t *testing.T) {
	m := regexp.MustCompile(`\(([^()]*), PRIMARY KEY`).FindStringSubmatch(SqlTradesTableCreate)
	if !assert.NotNil(t, m) {
		return
	}

	tableColumns := []string{}
	for _, def := range strings.Split(m[1], ", ") {
		tableColumns = append(tableColumns, strings.Fields(def)[0])
	}
	for _, alter := range []string{SqlTradesTableAlter1, SqlTradesTableAlter2} {
		tableColumns = append(tableColumns, strings.Fields(strings.TrimPrefix(alter, "ALTER TABLE trades ADD COLUMN "))[0])
	}

	insertColumns, _ := insertParts(t, SqlTradesInsertTemplate)
	assert.Equal(t, tableColumns, insertColumns)
}
