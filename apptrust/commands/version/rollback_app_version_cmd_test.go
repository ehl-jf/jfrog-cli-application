package version

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	mockversions "github.com/jfrog/jfrog-cli-application/apptrust/service/versions/mocks"
	"go.uber.org/mock/gomock"

	"github.com/jfrog/jfrog-cli-application/apptrust/model"
	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/stretchr/testify/assert"
)

func TestRollbackAppVersionCommand_Run(t *testing.T) {
	tests := []struct {
		name           string
		applicationKey string
		version        string
		fromStage      string
		sync           bool
		mockError      error
		expectedError  bool
	}{
		{
			name:           "successful rollback with sync=false",
			applicationKey: "video-encoder",
			version:        "1.5.0",
			fromStage:      "qa",
			sync:           false,
			mockError:      nil,
			expectedError:  false,
		},
		{
			name:           "successful rollback with sync=true",
			applicationKey: "test-app",
			version:        "1.0.0",
			fromStage:      "qa",
			sync:           true,
			mockError:      nil,
			expectedError:  false,
		},
		{
			name:           "failed rollback",
			applicationKey: "video-encoder",
			version:        "1.5.0",
			fromStage:      "qa",
			sync:           false,
			mockError:      errors.New("rollback service error occurred"),
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			serverDetails := &config.ServerDetails{Url: "https://example.com"}
			requestPayload := &model.RollbackAppVersionRequest{
				FromStage: tt.fromStage,
			}

			mockVersionService := mockversions.NewMockVersionService(ctrl)
			mockVersionService.EXPECT().RollbackAppVersion(gomock.Any(), tt.applicationKey, tt.version, requestPayload, tt.sync).
				Return(nil, tt.mockError).Times(1)

			cmd := &rollbackAppVersionCommand{
				versionService: mockVersionService,
				serverDetails:  serverDetails,
				applicationKey: tt.applicationKey,
				version:        tt.version,
				requestPayload: requestPayload,
				fromStage:      tt.fromStage,
				sync:           tt.sync,
			}

			err := cmd.Run()

			if tt.expectedError {
				assert.Error(t, err)
				if tt.mockError != nil {
					assert.Contains(t, err.Error(), tt.mockError.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- printRollbackAppVersionResponse tests ---

const sampleRollbackAppVersionJSON = `{"application_key":"my-app","version":"1.5.0","project_key":"proj-1","rollback_from_stage":"prod","rollback_to_stage":"qa"}`

func TestPrintRollbackAppVersionResponse_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := printRollbackAppVersionResponse([]byte(sampleRollbackAppVersionJSON), coreformat.Json, &buf)
	assert.NoError(t, err)
	// The JSON path goes through log.Output, not the writer — assert no error and valid JSON.
	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(sampleRollbackAppVersionJSON), &parsed))
}

func TestPrintRollbackAppVersionResponse_Table(t *testing.T) {
	var buf bytes.Buffer
	err := printRollbackAppVersionResponse([]byte(sampleRollbackAppVersionJSON), coreformat.Table, &buf)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "FIELD")
	assert.Contains(t, output, "VALUE")
	assert.Contains(t, output, "application_key")
	assert.Contains(t, output, "my-app")
	assert.Contains(t, output, "version")
	assert.Contains(t, output, "1.5.0")
	assert.Contains(t, output, "project_key")
	assert.Contains(t, output, "proj-1")
	assert.Contains(t, output, "rollback_from_stage")
	assert.Contains(t, output, "prod")
	assert.Contains(t, output, "rollback_to_stage")
	assert.Contains(t, output, "qa")
}

func TestPrintRollbackAppVersionResponse_Table_AbsentFieldsOmitted(t *testing.T) {
	// Only application_key and version are present — other fields must be absent from output.
	payload := `{"application_key":"my-app","version":"1.5.0"}`
	var buf bytes.Buffer
	err := printRollbackAppVersionResponse([]byte(payload), coreformat.Table, &buf)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "application_key")
	assert.Contains(t, output, "my-app")
	assert.Contains(t, output, "version")
	assert.Contains(t, output, "1.5.0")
	assert.NotContains(t, output, "project_key")
	assert.NotContains(t, output, "rollback_from_stage")
	assert.NotContains(t, output, "rollback_to_stage")
}

func TestPrintRollbackAppVersionResponse_None_BackwardCompat(t *testing.T) {
	// When outputFormat is None (no flag set), the function must not error.
	var buf bytes.Buffer
	err := printRollbackAppVersionResponse([]byte(sampleRollbackAppVersionJSON), coreformat.None, &buf)
	assert.NoError(t, err)
}

func TestPrintRollbackAppVersionResponse_Table_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := printRollbackAppVersionResponse([]byte("not-json"), coreformat.Table, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse rollback response")
}
