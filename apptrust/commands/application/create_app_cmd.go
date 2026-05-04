package application

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	pluginsCommon "github.com/jfrog/jfrog-cli-core/v2/plugins/common"

	"github.com/jfrog/jfrog-cli-application/apptrust/commands/utils"
	"github.com/jfrog/jfrog-cli-application/apptrust/model"
	"github.com/jfrog/jfrog-cli-application/apptrust/service"
	commonCLiCommands "github.com/jfrog/jfrog-cli-core/v2/common/commands"
	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	coreConfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	"github.com/jfrog/jfrog-cli-core/v2/utils/coreutils"
	clientUtils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/io/fileutils"
	"github.com/jfrog/jfrog-client-go/utils/log"

	"github.com/jfrog/jfrog-cli-application/apptrust/app"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands"
	"github.com/jfrog/jfrog-cli-application/apptrust/common"
	"github.com/jfrog/jfrog-cli-application/apptrust/service/applications"
)

type createAppCommand struct {
	serverDetails      *coreConfig.ServerDetails
	applicationService applications.ApplicationService
	requestBody        *model.AppDescriptor
	responseBody       []byte
}

func (cac *createAppCommand) Run() error {
	ctx, err := service.NewContext(*cac.serverDetails)
	if err != nil {
		return err
	}

	cac.responseBody, err = cac.applicationService.CreateApplication(ctx, cac.requestBody)
	return err
}

func (cac *createAppCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return cac.serverDetails, nil
}

func (cac *createAppCommand) CommandName() string {
	return commands.AppCreate
}

func (cac *createAppCommand) buildRequestPayload(ctx *components.Context) (*model.AppDescriptor, error) {
	applicationKey := ctx.Arguments[0]

	var appDescriptor *model.AppDescriptor
	var err error

	if ctx.IsFlagSet(commands.SpecFlag) {
		appDescriptor, err = cac.loadFromSpec(ctx)
	} else {
		appDescriptor, err = cac.buildFromFlags(ctx)
	}

	if err != nil {
		return nil, err
	}

	appDescriptor.ApplicationKey = applicationKey
	if appDescriptor.ApplicationName == "" {
		appDescriptor.ApplicationName = applicationKey
	}

	return appDescriptor, nil
}

func (cac *createAppCommand) buildFromFlags(ctx *components.Context) (*model.AppDescriptor, error) {
	project := ctx.GetStringFlagValue(commands.ProjectFlag)
	if project == "" {
		return nil, errorutils.CheckErrorf("--%s is mandatory", commands.ProjectFlag)
	}

	descriptor := &model.AppDescriptor{
		ProjectKey: project,
	}

	err := populateApplicationFromFlags(ctx, descriptor)
	if err != nil {
		return nil, err
	}

	return descriptor, nil
}

func (cac *createAppCommand) loadFromSpec(ctx *components.Context) (*model.AppDescriptor, error) {
	specFilePath := ctx.GetStringFlagValue(commands.SpecFlag)
	spec := new(model.AppDescriptor)
	specVars := coreutils.SpecVarsStringToMap(ctx.GetStringFlagValue(commands.SpecVarsFlag))
	content, err := fileutils.ReadFile(specFilePath)
	if errorutils.CheckError(err) != nil {
		return nil, err
	}

	if len(specVars) > 0 {
		content = coreutils.ReplaceVars(content, specVars)
	}

	err = json.Unmarshal(content, spec)
	if errorutils.CheckError(err) != nil {
		return nil, err
	}

	if spec.ProjectKey == "" {
		return nil, errorutils.CheckErrorf("project_key is mandatory in spec file")
	}

	return spec, nil
}

func (cac *createAppCommand) prepareAndRunCommand(ctx *components.Context) error {
	if err := validateCreateAppContext(ctx); err != nil {
		return err
	}

	var err error
	cac.requestBody, err = cac.buildRequestPayload(ctx)
	if err != nil {
		return err
	}

	cac.serverDetails, err = utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}

	outputFormat, err := ctx.GetOutputFormat()
	if err != nil {
		return err
	}

	if err = commonCLiCommands.Exec(cac); err != nil {
		return err
	}

	return printCreateAppResponse(cac.responseBody, outputFormat, os.Stdout)
}

func validateCreateAppContext(ctx *components.Context) error {
	if err := validateNoSpecAndFlagsTogether(ctx); err != nil {
		return err
	}
	if len(ctx.Arguments) != 1 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}
	return nil
}

func validateNoSpecAndFlagsTogether(ctx *components.Context) error {
	if ctx.IsFlagSet(commands.SpecFlag) {
		otherAppFlags := []string{
			commands.ApplicationNameFlag,
			commands.ProjectFlag,
			commands.DescriptionFlag,
			commands.BusinessCriticalityFlag,
			commands.MaturityLevelFlag,
			commands.LabelsFlag,
			commands.UserOwnersFlag,
			commands.GroupOwnersFlag,
		}
		for _, flag := range otherAppFlags {
			if ctx.IsFlagSet(flag) {
				return errorutils.CheckErrorf("the flag --%s is not allowed when --spec is provided.", flag)
			}
		}
	}
	return nil
}

// orderedCreateAppKeys defines the display order for app-create table output.
var orderedCreateAppKeys = []string{
	"application_key",
	"application_name",
	"project_key",
	"description",
	"criticality",
	"maturity_level",
}

// printCreateAppResponse formats and prints the create-application response.
// When outputFormat is Table it renders a FIELD/VALUE table; when Json it
// pretty-prints the raw JSON; when None (flag absent) it falls back to the
// previous log.Output behaviour for backward-compatibility.
func printCreateAppResponse(data []byte, outputFormat coreformat.OutputFormat, w io.Writer) error {
	switch outputFormat {
	case coreformat.Json:
		log.Output(clientUtils.IndentJson(data))
		return nil
	case coreformat.Table:
		return printCreateAppTable(data, w)
	default:
		// No --format flag provided: preserve the original output behaviour.
		log.Output(string(data))
		return nil
	}
}

// printCreateAppTable renders the create-application response as a FIELD/VALUE table.
func printCreateAppTable(data []byte, w io.Writer) error {
	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to parse application response: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FIELD\tVALUE")
	for _, key := range orderedCreateAppKeys {
		val, ok := fields[key]
		if !ok || val == nil {
			continue
		}
		strVal := fmt.Sprintf("%v", val)
		if strVal == "" {
			continue
		}
		fmt.Fprintf(tw, "%s\t%s\n", key, strVal)
	}
	return tw.Flush()
}

func GetCreateAppCommand(appContext app.Context) components.Command {
	cmd := &createAppCommand{
		applicationService: appContext.GetApplicationService(),
	}
	return components.Command{
		Name:             commands.AppCreate,
		Description:      "Create a new application.",
		Category:         common.CategoryApplication,
		Aliases:          []string{"ac"},
		SupportedFormats: []coreformat.OutputFormat{coreformat.Table, coreformat.Json},
		Arguments: []components.Argument{
			{
				Name:        "application-key",
				Description: "The key of the application to create.",
				Optional:    false,
			},
		},
		Flags:  commands.GetCommandFlags(commands.AppCreate),
		Action: cmd.prepareAndRunCommand,
	}
}
