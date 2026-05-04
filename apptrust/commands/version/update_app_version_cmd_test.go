package version

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	mockversions "github.com/jfrog/jfrog-cli-application/apptrust/service/versions/mocks"
	"go.uber.org/mock/gomock"

	"github.com/jfrog/jfrog-cli-application/apptrust/commands"
	"github.com/jfrog/jfrog-cli-application/apptrust/model"
	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/stretchr/testify/assert"
)

func TestUpdateAppVersionCommand_Run(t *testing.T) {
	tests := []struct {
		name         string
		request      *model.UpdateAppVersionRequest
		shouldError  bool
		errorMessage string
	}{
		{
			name: "success",
			request: &model.UpdateAppVersionRequest{
				Tag: "release/1.2.3",
				Properties: map[string][]string{
					"status": {"rc", "validated"},
				},
			},
		},
		{
			name: "context error",
			request: &model.UpdateAppVersionRequest{
				Tag: "test-tag",
			},
			shouldError:  true,
			errorMessage: "context error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockVersionService := mockversions.NewMockVersionService(ctrl)
			if tt.shouldError {
				mockVersionService.EXPECT().UpdateAppVersion(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New(tt.errorMessage)).Times(1)
			} else {
				mockVersionService.EXPECT().UpdateAppVersion(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, nil).Times(1)
			}

			cmd := &updateAppVersionCommand{
				versionService: mockVersionService,
				serverDetails:  &config.ServerDetails{Url: "https://example.com"},
				applicationKey: "app-key",
				version:        "1.0.0",
				requestPayload: tt.request,
			}

			err := cmd.Run()
			if tt.shouldError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMessage, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUpdateAppVersionCommand_FlagsSuite(t *testing.T) {
	tests := []struct {
		name           string
		ctxSetup       func(*components.Context)
		expectsError   bool
		errorContains  string
		expectsPayload *model.UpdateAppVersionRequest
	}{
		{
			name: "tag only",
			ctxSetup: func(ctx *components.Context) {
				ctx.Arguments = []string{"app-key", "1.0.0"}
				ctx.AddStringFlag(commands.TagFlag, "release/1.2.3")
			},
			expectsPayload: &model.UpdateAppVersionRequest{
				Tag: "release/1.2.3",
			},
		},
		{
			name: "properties only - single value",
			ctxSetup: func(ctx *components.Context) {
				ctx.Arguments = []string{"app-key", "1.0.0"}
				ctx.AddStringFlag(commands.PropertiesFlag, "status=rc")
			},
			expectsPayload: &model.UpdateAppVersionRequest{
				Properties: map[string][]string{
					"status": {"rc"},
				},
			},
		},
		{
			name: "properties only - multiple values",
			ctxSetup: func(ctx *components.Context) {
				ctx.Arguments = []string{"app-key", "1.0.0"}
				ctx.AddStringFlag(commands.PropertiesFlag, "status=rc,validated")
			},
			expectsPayload: &model.UpdateAppVersionRequest{
				Properties: map[string][]string{
					"status": {"rc", "validated"},
				},
			},
		},
		{
			name: "properties only - multiple properties",
			ctxSetup: func(ctx *components.Context) {
				ctx.Arguments = []string{"app-key", "1.0.0"}
				ctx.AddStringFlag(commands.PropertiesFlag, "status=rc,validated;deployed_to=staging-A,staging-B")
			},
			expectsPayload: &model.UpdateAppVersionRequest{
				Properties: map[string][]string{
					"status":      {"rc", "validated"},
					"deployed_to": {"staging-A", "staging-B"},
				},
			},
		},
		{
			name: "delete properties only",
			ctxSetup: func(ctx *components.Context) {
				ctx.Arguments = []string{"app-key", "1.0.0"}
				ctx.AddStringFlag(commands.DeletePropertiesFlag, "legacy_param;toBeDeleted")
			},
			expectsPayload: &model.UpdateAppVersionRequest{
				DeleteProperties: []string{"legacy_param", "toBeDeleted"},
			},
		},
		{
			name: "empty properties (clears values)",
			ctxSetup: func(ctx *components.Context) {
				ctx.Arguments = []string{"app-key", "1.0.0"}
				ctx.AddStringFlag(commands.PropertiesFlag, "old_feature_flag=")
			},
			expectsPayload: &model.UpdateAppVersionRequest{
				Properties: map[string][]string{
					"old_feature_flag": {},
				},
			},
		},
		{
			name: "combined update",
			ctxSetup: func(ctx *components.Context) {
				ctx.Arguments = []string{"app-key", "1.0.0"}
				ctx.AddStringFlag(commands.TagFlag, "release/1.2.3")
				ctx.AddStringFlag(commands.PropertiesFlag, "status=rc,validated")
				ctx.AddStringFlag(commands.DeletePropertiesFlag, "old_param")
			},
			expectsPayload: &model.UpdateAppVersionRequest{
				Tag: "release/1.2.3",
				Properties: map[string][]string{
					"status": {"rc", "validated"},
				},
				DeleteProperties: []string{"old_param"},
			},
		},
		{
			name: "empty tag (removes tag)",
			ctxSetup: func(ctx *components.Context) {
				ctx.Arguments = []string{"app-key", "1.0.0"}
				ctx.AddStringFlag(commands.TagFlag, "")
			},
			expectsPayload: &model.UpdateAppVersionRequest{
				Tag: "",
			},
		},
		{
			name: "invalid property format",
			ctxSetup: func(ctx *components.Context) {
				ctx.Arguments = []string{"app-key", "1.0.0"}
				ctx.AddStringFlag(commands.PropertiesFlag, "invalid-format")
			},
			expectsError:  true,
			errorContains: "invalid property format",
		},
		{
			name: "empty property key",
			ctxSetup: func(ctx *components.Context) {
				ctx.Arguments = []string{"app-key", "1.0.0"}
				ctx.AddStringFlag(commands.PropertiesFlag, "=value")
			},
			expectsError:  true,
			errorContains: "property key cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := &components.Context{}
			tt.ctxSetup(ctx)
			ctx.AddStringFlag("url", "https://example.com")

			var actualPayload *model.UpdateAppVersionRequest
			mockVersionService := mockversions.NewMockVersionService(ctrl)
			if !tt.expectsError {
				mockVersionService.EXPECT().UpdateAppVersion(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, _ string, _ string, req *model.UpdateAppVersionRequest) ([]byte, error) {
						actualPayload = req
						return nil, nil
					}).Times(1)
			}

			cmd := &updateAppVersionCommand{
				versionService: mockVersionService,
			}

			err := cmd.prepareAndRunCommand(ctx)
			if tt.expectsError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectsPayload, actualPayload)
			}
		})
	}
}

// --- printUpdateAppVersionResponse tests ---

const sampleUpdateAppVersionJSON = `{"application_key":"my-app","version":"1.0.0","status":"COMPLETED","current_stage":"dev","tag":"release-tag"}`

func TestPrintUpdateAppVersionResponse_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := printUpdateAppVersionResponse([]byte(sampleUpdateAppVersionJSON), coreformat.Json, &buf)
	assert.NoError(t, err)
	// The JSON path goes through log.Output, not the writer — assert no error and valid JSON.
	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(sampleUpdateAppVersionJSON), &parsed))
}

func TestPrintUpdateAppVersionResponse_Table(t *testing.T) {
	var buf bytes.Buffer
	err := printUpdateAppVersionResponse([]byte(sampleUpdateAppVersionJSON), coreformat.Table, &buf)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "FIELD")
	assert.Contains(t, output, "VALUE")
	assert.Contains(t, output, "application_key")
	assert.Contains(t, output, "my-app")
	assert.Contains(t, output, "version")
	assert.Contains(t, output, "1.0.0")
	assert.Contains(t, output, "status")
	assert.Contains(t, output, "COMPLETED")
	assert.Contains(t, output, "current_stage")
	assert.Contains(t, output, "dev")
	assert.Contains(t, output, "tag")
	assert.Contains(t, output, "release-tag")
}

func TestPrintUpdateAppVersionResponse_Table_AbsentFieldsOmitted(t *testing.T) {
	// Only application_key and version are present — other fields must be absent from output.
	payload := `{"application_key":"my-app","version":"1.0.0"}`
	var buf bytes.Buffer
	err := printUpdateAppVersionResponse([]byte(payload), coreformat.Table, &buf)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "application_key")
	assert.Contains(t, output, "my-app")
	assert.Contains(t, output, "version")
	assert.Contains(t, output, "1.0.0")
	assert.NotContains(t, output, "status")
	assert.NotContains(t, output, "current_stage")
	assert.NotContains(t, output, "tag")
}

func TestPrintUpdateAppVersionResponse_None_BackwardCompat(t *testing.T) {
	// When outputFormat is None (no flag set), the function must not error.
	var buf bytes.Buffer
	err := printUpdateAppVersionResponse([]byte(sampleUpdateAppVersionJSON), coreformat.None, &buf)
	assert.NoError(t, err)
}

func TestPrintUpdateAppVersionResponse_Table_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := printUpdateAppVersionResponse([]byte("not-json"), coreformat.Table, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse update response")
}
