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

func TestPromoteAppVersionCommand_Run(t *testing.T) {
	tests := []struct {
		name              string
		sync              bool
		overwriteStrategy string
	}{
		{
			name: "sync flag true",
			sync: true,
		},
		{
			name: "sync flag false",
			sync: false,
		},
		{
			name:              "with overwrite strategy disabled (sent as DISABLED)",
			sync:              true,
			overwriteStrategy: "DISABLED",
		},
		{
			name:              "with overwrite strategy latest (sent as LATEST)",
			sync:              true,
			overwriteStrategy: "LATEST",
		},
		{
			name:              "with overwrite strategy all (sent as ALL)",
			sync:              true,
			overwriteStrategy: "ALL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			serverDetails := &config.ServerDetails{Url: "https://example.com"}
			applicationKey := "app-key"
			version := "1.0.0"
			requestPayload := &model.PromoteAppVersionRequest{
				Stage: "prod",
				CommonPromoteAppVersion: model.CommonPromoteAppVersion{
					OverwriteStrategy: tt.overwriteStrategy,
				},
			}

			mockVersionService := mockversions.NewMockVersionService(ctrl)
			mockVersionService.EXPECT().PromoteAppVersion(gomock.Any(), applicationKey, version, requestPayload, tt.sync).
				Return(nil, nil).Times(1)

			cmd := &promoteAppVersionCommand{
				versionService: mockVersionService,
				serverDetails:  serverDetails,
				applicationKey: applicationKey,
				version:        version,
				requestPayload: requestPayload,
				sync:           tt.sync,
			}

			err := cmd.Run()
			assert.NoError(t, err)
		})
	}
}

func TestPromoteAppVersionCommand_Run_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serverDetails := &config.ServerDetails{Url: "https://example.com"}
	applicationKey := "app-key"
	version := "1.0.0"
	requestPayload := &model.PromoteAppVersionRequest{
		Stage: "prod",
	}
	sync := true
	expectedError := errors.New("service error occurred")

	mockVersionService := mockversions.NewMockVersionService(ctrl)
	mockVersionService.EXPECT().PromoteAppVersion(gomock.Any(), applicationKey, version, requestPayload, sync).
		Return(nil, expectedError).Times(1)

	cmd := &promoteAppVersionCommand{
		versionService: mockVersionService,
		serverDetails:  serverDetails,
		applicationKey: applicationKey,
		version:        version,
		requestPayload: requestPayload,
		sync:           sync,
	}

	err := cmd.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service error occurred")
}

// --- printPromoteAppVersionResponse tests ---

const samplePromoteAppVersionJSON = `{"application_key":"my-app","version":"1.0.0","target_stage":"prod","status":"COMPLETED","current_stage":"prod"}`

func TestPrintPromoteAppVersionResponse_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := printPromoteAppVersionResponse([]byte(samplePromoteAppVersionJSON), coreformat.Json, &buf)
	assert.NoError(t, err)
	// The JSON path goes through log.Output, not the writer — assert no error and valid JSON.
	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(samplePromoteAppVersionJSON), &parsed))
}

func TestPrintPromoteAppVersionResponse_Table(t *testing.T) {
	var buf bytes.Buffer
	err := printPromoteAppVersionResponse([]byte(samplePromoteAppVersionJSON), coreformat.Table, &buf)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "FIELD")
	assert.Contains(t, output, "VALUE")
	assert.Contains(t, output, "application_key")
	assert.Contains(t, output, "my-app")
	assert.Contains(t, output, "version")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "target_stage")
	assert.Contains(t, output, "prod")
	assert.Contains(t, output, "status")
	assert.Contains(t, output, "COMPLETED")
	assert.Contains(t, output, "current_stage")
}

func TestPrintPromoteAppVersionResponse_Table_AbsentFieldsOmitted(t *testing.T) {
	// Only application_key and version are present — other fields must be absent from output.
	payload := `{"application_key":"my-app","version":"1.0.0"}`
	var buf bytes.Buffer
	err := printPromoteAppVersionResponse([]byte(payload), coreformat.Table, &buf)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "application_key")
	assert.Contains(t, output, "my-app")
	assert.Contains(t, output, "version")
	assert.Contains(t, output, "1.0.0")
	assert.NotContains(t, output, "target_stage")
	assert.NotContains(t, output, "status")
	assert.NotContains(t, output, "current_stage")
}

func TestPrintPromoteAppVersionResponse_None_BackwardCompat(t *testing.T) {
	// When outputFormat is None (no flag set), the function must not error.
	var buf bytes.Buffer
	err := printPromoteAppVersionResponse([]byte(samplePromoteAppVersionJSON), coreformat.None, &buf)
	assert.NoError(t, err)
}

func TestPrintPromoteAppVersionResponse_Table_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := printPromoteAppVersionResponse([]byte("not-json"), coreformat.Table, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse promote response")
}
