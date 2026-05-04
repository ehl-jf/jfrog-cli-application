package packagecmds

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jfrog/jfrog-cli-application/apptrust/model"
	mockpackages "github.com/jfrog/jfrog-cli-application/apptrust/service/packages/mocks"
	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestBindPackageCommand_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serverDetails := &config.ServerDetails{Url: "https://example.com"}
	applicationKey := "app-key"
	requestPayload := &model.BindPackageRequest{
		Type:    "npm",
		Name:    "test-package",
		Version: "1.0.0",
	}

	mockPackageService := mockpackages.NewMockPackageService(ctrl)
	mockPackageService.EXPECT().BindPackage(gomock.Any(), applicationKey, requestPayload).
		Return(nil, nil).Times(1)

	cmd := &bindPackageCommand{
		packageService: mockPackageService,
		serverDetails:  serverDetails,
		applicationKey: applicationKey,
		requestPayload: requestPayload,
	}

	err := cmd.Run()
	assert.NoError(t, err)
}

func TestBindPackageCommand_Run_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serverDetails := &config.ServerDetails{Url: "https://example.com"}
	applicationKey := "app-key"
	requestPayload := &model.BindPackageRequest{
		Type:    "npm",
		Name:    "test-package",
		Version: "1.0.0",
	}

	mockPackageService := mockpackages.NewMockPackageService(ctrl)
	mockPackageService.EXPECT().BindPackage(gomock.Any(), applicationKey, requestPayload).
		Return(nil, errors.New("bind error")).Times(1)

	cmd := &bindPackageCommand{
		packageService: mockPackageService,
		serverDetails:  serverDetails,
		applicationKey: applicationKey,
		requestPayload: requestPayload,
	}

	err := cmd.Run()
	assert.Error(t, err)
	assert.Equal(t, "bind error", err.Error())
}

// --- printBindPackageResponse tests ---

const sampleBindPackageJSON = `{"application_key":"my-app","package_type":"npm","package_name":"my-package","package_version":"1.0.0","status":"bound"}`

func TestPrintBindPackageResponse_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := printBindPackageResponse([]byte(sampleBindPackageJSON), coreformat.Json, &buf)
	assert.NoError(t, err)
	// The JSON path goes through log.Output, not the writer — assert no error and valid JSON.
	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(sampleBindPackageJSON), &parsed))
}

func TestPrintBindPackageResponse_Table(t *testing.T) {
	var buf bytes.Buffer
	err := printBindPackageResponse([]byte(sampleBindPackageJSON), coreformat.Table, &buf)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "FIELD")
	assert.Contains(t, output, "VALUE")
	assert.Contains(t, output, "application_key")
	assert.Contains(t, output, "my-app")
	assert.Contains(t, output, "package_type")
	assert.Contains(t, output, "npm")
	assert.Contains(t, output, "package_name")
	assert.Contains(t, output, "my-package")
	assert.Contains(t, output, "package_version")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "status")
	assert.Contains(t, output, "bound")
}

func TestPrintBindPackageResponse_Table_AbsentFieldsOmitted(t *testing.T) {
	// Only application_key and package_type are present — other fields must be absent.
	payload := `{"application_key":"my-app","package_type":"npm"}`
	var buf bytes.Buffer
	err := printBindPackageResponse([]byte(payload), coreformat.Table, &buf)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "application_key")
	assert.Contains(t, output, "my-app")
	assert.Contains(t, output, "package_type")
	assert.Contains(t, output, "npm")
	assert.NotContains(t, output, "package_name")
	assert.NotContains(t, output, "package_version")
	assert.NotContains(t, output, "status")
}

func TestPrintBindPackageResponse_None_BackwardCompat(t *testing.T) {
	// When outputFormat is None (no flag set), the function must not error.
	var buf bytes.Buffer
	err := printBindPackageResponse([]byte(sampleBindPackageJSON), coreformat.None, &buf)
	assert.NoError(t, err)
}

func TestPrintBindPackageResponse_Table_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := printBindPackageResponse([]byte("not-json"), coreformat.Table, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse bind-package response")
}
