package common

import (
	"bytes"
	"encoding/json"
	"testing"

	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	"github.com/jfrog/jfrog-client-go/utils/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleOutputJSON = `{"key":"alpha","extra":"beta"}`

var orderedOutputKeys = []string{"key", "extra", "missing"}

func TestPrintJsonOrTableResponse_JSON(t *testing.T) {
	var logBuf bytes.Buffer
	prevLogger := log.GetLogger()
	log.SetLogger(log.NewLogger(log.INFO, &logBuf))
	t.Cleanup(func() { log.SetLogger(prevLogger) })

	var buf bytes.Buffer
	err := PrintJsonOrTableResponse([]byte(sampleOutputJSON), coreformat.Json, &buf, orderedOutputKeys)
	assert.NoError(t, err)
	assert.Empty(t, buf.String(), "Json branch must not write to the io.Writer")

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(logBuf.Bytes(), &parsed))
	assert.Equal(t, "alpha", parsed["key"])
	assert.Equal(t, "beta", parsed["extra"])
}

func TestPrintJsonOrTableResponse_Table(t *testing.T) {
	var buf bytes.Buffer
	err := PrintJsonOrTableResponse([]byte(sampleOutputJSON), coreformat.Table, &buf, orderedOutputKeys)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "FIELD")
	assert.Contains(t, output, "VALUE")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "alpha")
	assert.Contains(t, output, "extra")
	assert.Contains(t, output, "beta")
}

func TestPrintJsonOrTableResponse_Table_AbsentFieldsOmitted(t *testing.T) {
	payload := `{"key":"only-key"}`
	var buf bytes.Buffer
	err := PrintJsonOrTableResponse([]byte(payload), coreformat.Table, &buf, orderedOutputKeys)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "only-key")
	assert.NotContains(t, output, "extra")
	assert.NotContains(t, output, "missing")
}

func TestPrintJsonOrTableResponse_None_BackwardCompat(t *testing.T) {
	var buf bytes.Buffer
	err := PrintJsonOrTableResponse([]byte(sampleOutputJSON), coreformat.None, &buf, orderedOutputKeys)
	assert.NoError(t, err)
}

func TestPrintJsonOrTableResponse_Table_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := PrintJsonOrTableResponse([]byte("not-json"), coreformat.Table, &buf, orderedOutputKeys)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}
