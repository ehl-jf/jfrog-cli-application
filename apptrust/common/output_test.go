package common

import (
	"bytes"
	"encoding/json"
	"testing"

	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleOutputJSON = `{"key":"alpha","extra":"beta"}`

var orderedOutputKeys = []string{"key", "extra", "missing"}

func TestPrintResponse_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := PrintResponse([]byte(sampleOutputJSON), coreformat.Json, &buf, orderedOutputKeys)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, "alpha", parsed["key"])
	assert.Equal(t, "beta", parsed["extra"])
}

func TestPrintResponse_Table(t *testing.T) {
	var buf bytes.Buffer
	err := PrintResponse([]byte(sampleOutputJSON), coreformat.Table, &buf, orderedOutputKeys)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "FIELD")
	assert.Contains(t, output, "VALUE")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "alpha")
	assert.Contains(t, output, "extra")
	assert.Contains(t, output, "beta")
}

func TestPrintResponse_Table_AbsentFieldsOmitted(t *testing.T) {
	payload := `{"key":"only-key"}`
	var buf bytes.Buffer
	err := PrintResponse([]byte(payload), coreformat.Table, &buf, orderedOutputKeys)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "only-key")
	assert.NotContains(t, output, "extra")
	assert.NotContains(t, output, "missing")
}

func TestPrintResponse_None_NoOutput(t *testing.T) {
	var buf bytes.Buffer
	err := PrintResponse([]byte(sampleOutputJSON), coreformat.None, &buf, orderedOutputKeys)
	assert.NoError(t, err)
	assert.Empty(t, buf.String(), "None format must produce no output")
}

func TestPrintResponse_Table_EmptyData(t *testing.T) {
	var buf bytes.Buffer
	err := PrintResponse([]byte("   \n"), coreformat.Table, &buf, orderedOutputKeys)
	assert.NoError(t, err)
	assert.Empty(t, buf.String(), "empty data must produce no output")
}

func TestPrintResponse_Table_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := PrintResponse([]byte("not-json"), coreformat.Table, &buf, orderedOutputKeys)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}
