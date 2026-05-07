package common

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	clientUtils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

// PrintJsonOrTableResponse formats and prints a JSON response body.
// Json: pretty-prints the JSON. Table: renders a FIELD/VALUE table using orderedKeys.
// None/default: falls back to raw string output for backward compatibility.
func PrintJsonOrTableResponse(data []byte, outputFormat coreformat.OutputFormat, w io.Writer, orderedKeys []string) error {
	switch outputFormat {
	case coreformat.Json:
		log.Output(clientUtils.IndentJson(data))
		return nil
	case coreformat.Table:
		return PrintTable(data, w, orderedKeys)
	default:
		log.Output(string(data))
		return nil
	}
}

// PrintTable renders data as a FIELD/VALUE table using orderedKeys for display order.
// Fields absent from data or with empty/nil values are omitted.
func PrintTable(data []byte, w io.Writer, orderedKeys []string) error {
	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FIELD\tVALUE")
	for _, key := range orderedKeys {
		val, ok := fields[key]
		if !ok || val == nil {
			continue
		}
		var strVal string
		switch v := val.(type) {
		case string:
			strVal = v
		case []interface{}, map[string]interface{}:
			b, err := json.Marshal(v)
			if err != nil {
				return err
			}
			strVal = string(b)
		default:
			strVal = fmt.Sprintf("%v", v)
		}
		if strVal == "" {
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\n", key, strVal)
	}
	return tw.Flush()
}
