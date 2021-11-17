package lint

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/thought-machine/please/src/core"
)

func parseLintLines(linter *core.Linter, linterName, out string) []core.LintResult {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	results := make([]core.LintResult, len(lines))
	for i, line := range lines {
		results[i] = parseLintLine(linter, linterName, line)
	}
	return results
}

func parseLintLine(linter *core.Linter, linterName, line string) core.LintResult {
	matches := linter.OutputFormat.FindStringSubmatch(line)
	if matches == nil {
		// Failed to match, turn this into a linter error itself.
		return core.LintResult{
			Linter:  linterName,
			Message: "Failed to parse line: " + line,
		}
	}
	result := core.LintResult{Linter: linterName}
	v := reflect.ValueOf(&result)
	subfields := linter.OutputFormat.SubexpNames()
	// Recall that the first entry is the complete match, hence all the off-by-one stuff.
	for i, match := range matches[1:] {
		name := subfields[i+1]
		field := v.Elem().FieldByName(strings.ToUpper(name[:1]) + name[1:])
		switch field.Kind() {
		case reflect.Invalid:
			panic(fmt.Sprintf("Invalid field %s in linter %s", name, linterName))
		case reflect.String:
			field.SetString(match)
		case reflect.Int:
			i, err := strconv.ParseInt(match, 10, 64)
			if err != nil {
				result.Message = fmt.Sprintf("Invalid value for int field %s: %s", name, match)
				return result
			}
			field.SetInt(i)
		}
	}
	return result
}
