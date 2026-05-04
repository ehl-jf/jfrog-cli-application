package application

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"testing"

	"github.com/urfave/cli"

	"github.com/jfrog/jfrog-cli-application/apptrust/model"
	mockapps "github.com/jfrog/jfrog-cli-application/apptrust/service/applications/mocks"
	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	"github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCreateAppCommand_Run_Flags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	description := "Test application"
	businessCriticality := "high"
	maturityLevel := "production"

	ctx := &components.Context{
		Arguments: []string{"app-key"},
	}
	ctx.AddStringFlag("application-name", "test-app")
	ctx.AddStringFlag("project", "test-project")
	ctx.AddStringFlag("desc", description)
	ctx.AddStringFlag("business-criticality", "high")
	ctx.AddStringFlag("maturity-level", maturityLevel)
	ctx.AddStringFlag("labels", "env=prod;region=us-east")
	ctx.AddStringFlag("user-owners", "john.doe;jane.smith")
	ctx.AddStringFlag("group-owners", "devops;security")
	ctx.AddStringFlag("url", "https://example.com")

	requestPayload := &model.AppDescriptor{
		ApplicationKey:      "app-key",
		ApplicationName:     "test-app",
		ProjectKey:          "test-project",
		Description:         &description,
		BusinessCriticality: &businessCriticality,
		MaturityLevel:       &maturityLevel,
		Labels: &[]model.LabelEntry{
			{Key: "env", Value: "prod"},
			{Key: "region", Value: "us-east"},
		},
		UserOwners:  &[]string{"john.doe", "jane.smith"},
		GroupOwners: &[]string{"devops", "security"},
	}

	mockAppService := mockapps.NewMockApplicationService(ctrl)
	mockAppService.EXPECT().CreateApplication(gomock.Any(), requestPayload).Return(nil, nil).Times(1)

	cmd := &createAppCommand{
		applicationService: mockAppService,
		requestBody:        requestPayload,
	}

	err := cmd.prepareAndRunCommand(ctx)
	assert.NoError(t, err)
}

func TestCreateAppCommand_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	serverDetails := &config.ServerDetails{Url: "https://example.com"}
	requestPayload := &model.AppDescriptor{
		ApplicationKey:  "app-key",
		ApplicationName: "app-name",
		ProjectKey:      "proj-key",
	}

	mockAppService := mockapps.NewMockApplicationService(ctrl)
	mockAppService.EXPECT().CreateApplication(gomock.Any(), requestPayload).Return(nil, errors.New("failed to create an application. Status code: 500")).Times(1)

	cmd := &createAppCommand{
		applicationService: mockAppService,
		serverDetails:      serverDetails,
		requestBody:        requestPayload,
	}

	err := cmd.Run()
	assert.Error(t, err)
	assert.Equal(t, "failed to create an application. Status code: 500", err.Error())
}

func TestCreateAppCommand_WrongNumberOfArguments(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)

	mockAppService := mockapps.NewMockApplicationService(ctrl)
	cmd := &createAppCommand{
		applicationService: mockAppService,
	}

	// Test with no arguments
	context, err := components.ConvertContext(ctx)
	assert.NoError(t, err)

	err = cmd.prepareAndRunCommand(context)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Wrong number of arguments")
}

func TestCreateAppCommand_MissingProjectFlag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := &components.Context{
		Arguments: []string{"app-key"},
	}
	ctx.AddStringFlag("application-name", "test-app")
	ctx.AddStringFlag("url", "https://example.com")
	mockAppService := mockapps.NewMockApplicationService(ctrl)

	cmd := &createAppCommand{
		applicationService: mockAppService,
	}

	err := cmd.prepareAndRunCommand(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--project is mandatory")
}

func TestCreateAppCommand_Run_FullSpecFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := &components.Context{
		Arguments: []string{"app-full"},
	}
	ctx.AddStringFlag("url", "https://example.com")
	ctx.AddStringFlag("spec", "./testfiles/full-spec.json")

	expectedDescription := "A comprehensive test application"
	expectedMaturityLevel := "production"
	expectedBusinessCriticality := "high"
	expectedPayload := &model.AppDescriptor{
		ApplicationKey:      "app-full",
		ApplicationName:     "test-app-full",
		ProjectKey:          "test-project",
		Description:         &expectedDescription,
		MaturityLevel:       &expectedMaturityLevel,
		BusinessCriticality: &expectedBusinessCriticality,
		Labels: &[]model.LabelEntry{
			{Key: "environment", Value: "production"},
			{Key: "environment", Value: "staging"},
			{Key: "region", Value: "us-east-1"},
			{Key: "team", Value: "devops"},
		},
		UserOwners:  &[]string{"john.doe", "jane.smith"},
		GroupOwners: &[]string{"devops-team", "security-team"},
	}

	var actualPayload *model.AppDescriptor
	mockAppService := mockapps.NewMockApplicationService(ctrl)
	mockAppService.EXPECT().CreateApplication(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, req *model.AppDescriptor) ([]byte, error) {
			actualPayload = req
			return nil, nil
		}).Times(1)

	cmd := &createAppCommand{
		applicationService: mockAppService,
	}

	err := cmd.prepareAndRunCommand(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedPayload, actualPayload)
}

func TestCreateAppCommand_Run_SpecFile(t *testing.T) {
	tests := []struct {
		name           string
		specPath       string
		args           []string
		expectsError   bool
		errorContains  string
		expectsPayload *model.AppDescriptor
	}{
		{
			name:     "minimal spec file",
			specPath: "./testfiles/minimal-spec.json",
			args:     []string{"app-min"},
			expectsPayload: &model.AppDescriptor{
				ApplicationKey:  "app-min",
				ApplicationName: "app-min",
				ProjectKey:      "test-project",
			},
		},
		{
			name:          "invalid spec file",
			specPath:      "./testfiles/invalid-spec.json",
			args:          []string{"app-invalid"},
			expectsError:  true,
			errorContains: "unexpected end of JSON input",
		},
		{
			name:          "missing project key",
			specPath:      "./testfiles/missing-project-spec.json",
			args:          []string{"app-no-project"},
			expectsError:  true,
			errorContains: "project_key is mandatory in spec file",
		},
		{
			name:          "non-existent spec file",
			specPath:      "./testfiles/non-existent.json",
			args:          []string{"app-nonexistent"},
			expectsError:  true,
			errorContains: "no such file or directory",
		},
		{
			name:     "spec with application_key that should be ignored",
			specPath: "./testfiles/spec-with-app-key.json",
			args:     []string{"command-line-app-key"},
			expectsPayload: &model.AppDescriptor{
				ApplicationKey:  "command-line-app-key",
				ApplicationName: "test-app",
				ProjectKey:      "test-project",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := &components.Context{
				Arguments: tt.args,
			}
			ctx.AddStringFlag("url", "https://example.com")
			ctx.AddStringFlag("spec", tt.specPath)

			var actualPayload *model.AppDescriptor
			mockAppService := mockapps.NewMockApplicationService(ctrl)
			if !tt.expectsError {
				mockAppService.EXPECT().CreateApplication(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ interface{}, req *model.AppDescriptor) ([]byte, error) {
						actualPayload = req
						return nil, nil
					}).Times(1)
			}

			cmd := &createAppCommand{
				applicationService: mockAppService,
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

func TestCreateAppCommand_Run_SpecVars(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	expectedDescription := "A test application for production"
	expectedMaturityLevel := "production"
	expectedBusinessCriticality := "high"

	expectedPayload := &model.AppDescriptor{
		ApplicationKey:      "app-with-vars",
		ApplicationName:     "test-app",
		ProjectKey:          "test-project",
		Description:         &expectedDescription,
		MaturityLevel:       &expectedMaturityLevel,
		BusinessCriticality: &expectedBusinessCriticality,
		Labels: &[]model.LabelEntry{
			{Key: "environment", Value: "production"},
			{Key: "region", Value: "us-east-1"},
		},
	}

	ctx := &components.Context{
		Arguments: []string{"app-with-vars"},
	}
	ctx.AddStringFlag("spec", "./testfiles/with-vars-spec.json")
	ctx.AddStringFlag("spec-vars", "PROJECT_KEY=test-project;APP_NAME=test-app;ENVIRONMENT=production;MATURITY_LEVEL=production;CRITICALITY=high;REGION=us-east-1")
	ctx.AddStringFlag("url", "https://example.com")

	var actualPayload *model.AppDescriptor
	mockAppService := mockapps.NewMockApplicationService(ctrl)
	mockAppService.EXPECT().CreateApplication(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, req *model.AppDescriptor) ([]byte, error) {
			actualPayload = req
			return nil, nil
		}).Times(1)

	cmd := &createAppCommand{
		applicationService: mockAppService,
	}

	err := cmd.prepareAndRunCommand(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expectedPayload, actualPayload)
}

func TestCreateAppCommand_Error_SpecAndFlags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSpecPath := "./testfiles/minimal-spec.json"
	ctx := &components.Context{
		Arguments: []string{"app-key"},
	}
	ctx.AddStringFlag("spec", testSpecPath)
	ctx.AddStringFlag("project", "test-project")
	ctx.AddStringFlag("url", "https://example.com")

	mockAppService := mockapps.NewMockApplicationService(ctrl)

	cmd := &createAppCommand{
		applicationService: mockAppService,
	}

	err := cmd.prepareAndRunCommand(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "the flag --project is not allowed when --spec is provided")
}

// --- printCreateAppResponse tests ---

const sampleAppJSON = `{"application_key":"my-app","application_name":"My App","project_key":"proj1","criticality":"high","maturity_level":"production"}`

func TestPrintCreateAppResponse_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := printCreateAppResponse([]byte(sampleAppJSON), coreformat.Json, &buf)
	assert.NoError(t, err)
	// The JSON path goes through log.Output, not the writer — assert no error and valid JSON.
	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal([]byte(sampleAppJSON), &parsed))
}

func TestPrintCreateAppResponse_Table(t *testing.T) {
	var buf bytes.Buffer
	err := printCreateAppResponse([]byte(sampleAppJSON), coreformat.Table, &buf)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "FIELD")
	assert.Contains(t, output, "VALUE")
	assert.Contains(t, output, "application_key")
	assert.Contains(t, output, "my-app")
	assert.Contains(t, output, "application_name")
	assert.Contains(t, output, "My App")
	assert.Contains(t, output, "project_key")
	assert.Contains(t, output, "proj1")
}

func TestPrintCreateAppResponse_Table_AbsentFieldsOmitted(t *testing.T) {
	// Only application_key is present — other fields must be absent from output.
	payload := `{"application_key":"only-key"}`
	var buf bytes.Buffer
	err := printCreateAppResponse([]byte(payload), coreformat.Table, &buf)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "application_key")
	assert.Contains(t, output, "only-key")
	assert.NotContains(t, output, "application_name")
	assert.NotContains(t, output, "project_key")
}

func TestPrintCreateAppResponse_None_BackwardCompat(t *testing.T) {
	// When outputFormat is None (no flag set), the function must not error.
	var buf bytes.Buffer
	err := printCreateAppResponse([]byte(sampleAppJSON), coreformat.None, &buf)
	assert.NoError(t, err)
}

func TestPrintCreateAppResponse_Table_InvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := printCreateAppResponse([]byte("not-json"), coreformat.Table, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse application response")
}
