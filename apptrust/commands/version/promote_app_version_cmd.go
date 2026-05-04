package version

//go:generate ${PROJECT_DIR}/scripts/mockgen.sh ${GOFILE}

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/jfrog/jfrog-cli-application/apptrust/app"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands"
	"github.com/jfrog/jfrog-cli-application/apptrust/commands/utils"
	"github.com/jfrog/jfrog-cli-application/apptrust/common"
	"github.com/jfrog/jfrog-cli-application/apptrust/model"
	"github.com/jfrog/jfrog-cli-application/apptrust/service"
	"github.com/jfrog/jfrog-cli-application/apptrust/service/versions"
	commonCLiCommands "github.com/jfrog/jfrog-cli-core/v2/common/commands"
	coreformat "github.com/jfrog/jfrog-cli-core/v2/common/format"
	pluginsCommon "github.com/jfrog/jfrog-cli-core/v2/plugins/common"
	"github.com/jfrog/jfrog-cli-core/v2/plugins/components"
	coreConfig "github.com/jfrog/jfrog-cli-core/v2/utils/config"
	clientUtils "github.com/jfrog/jfrog-client-go/utils"
	"github.com/jfrog/jfrog-client-go/utils/errorutils"
	"github.com/jfrog/jfrog-client-go/utils/log"
)

type promoteAppVersionCommand struct {
	versionService versions.VersionService
	serverDetails  *coreConfig.ServerDetails
	applicationKey string
	version        string
	requestPayload *model.PromoteAppVersionRequest
	sync           bool
	responseBody   []byte
}

func (pv *promoteAppVersionCommand) Run() error {
	ctx, err := service.NewContext(*pv.serverDetails)
	if err != nil {
		return err
	}

	pv.responseBody, err = pv.versionService.PromoteAppVersion(ctx, pv.applicationKey, pv.version, pv.requestPayload, pv.sync)
	return err
}

func (pv *promoteAppVersionCommand) ServerDetails() (*coreConfig.ServerDetails, error) {
	return pv.serverDetails, nil
}

func (pv *promoteAppVersionCommand) CommandName() string {
	return commands.VersionPromote
}

func (pv *promoteAppVersionCommand) prepareAndRunCommand(ctx *components.Context) error {
	if len(ctx.Arguments) != 3 {
		return pluginsCommon.WrongNumberOfArgumentsHandler(ctx)
	}

	// Extract from arguments
	pv.applicationKey = ctx.Arguments[0]
	pv.version = ctx.Arguments[1]

	// Extract sync flag value
	pv.sync = ctx.GetBoolTFlagValue(commands.SyncFlag)

	serverDetails, err := utils.ServerDetailsByFlags(ctx)
	if err != nil {
		return err
	}
	pv.serverDetails = serverDetails
	pv.requestPayload, err = pv.buildRequestPayload(ctx)
	if errorutils.CheckError(err) != nil {
		return err
	}

	outputFormat, err := ctx.GetOutputFormat()
	if err != nil {
		return err
	}

	if err = commonCLiCommands.Exec(pv); err != nil {
		return err
	}

	return printPromoteAppVersionResponse(pv.responseBody, outputFormat, os.Stdout)
}

// orderedPromoteAppVersionKeys defines the display order for version-promote table output.
var orderedPromoteAppVersionKeys = []string{
	"application_key",
	"version",
	"target_stage",
	"status",
	"current_stage",
}

// printPromoteAppVersionResponse formats and prints the promote-app-version response.
// When outputFormat is Table it renders a FIELD/VALUE table; when Json it
// pretty-prints the raw JSON; when None (flag absent) it falls back to the
// previous log.Output behaviour for backward-compatibility.
func printPromoteAppVersionResponse(data []byte, outputFormat coreformat.OutputFormat, w io.Writer) error {
	switch outputFormat {
	case coreformat.Json:
		log.Output(clientUtils.IndentJson(data))
		return nil
	case coreformat.Table:
		return printPromoteAppVersionTable(data, w)
	default:
		// No --format flag provided: preserve the original output behaviour.
		log.Output(string(data))
		return nil
	}
}

// printPromoteAppVersionTable renders the promote-app-version response as a FIELD/VALUE table.
func printPromoteAppVersionTable(data []byte, w io.Writer) error {
	var fields map[string]interface{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return fmt.Errorf("failed to parse promote response: %w", err)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FIELD\tVALUE")
	for _, key := range orderedPromoteAppVersionKeys {
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

func (pv *promoteAppVersionCommand) buildRequestPayload(ctx *components.Context) (*model.PromoteAppVersionRequest, error) {
	stage := ctx.Arguments[2]

	promotionType, includedRepos, excludedRepos, err := BuildPromotionParams(ctx)
	if err != nil {
		return nil, err
	}

	artifactProps, err := ParseArtifactProps(ctx)
	if err != nil {
		return nil, err
	}

	overwriteStrategy, err := ParseOverwriteStrategy(ctx)
	if err != nil {
		return nil, err
	}

	return &model.PromoteAppVersionRequest{
		Stage: stage,
		CommonPromoteAppVersion: model.CommonPromoteAppVersion{
			PromotionType:                promotionType,
			IncludedRepositoryKeys:       includedRepos,
			ExcludedRepositoryKeys:       excludedRepos,
			ArtifactAdditionalProperties: artifactProps,
			OverwriteStrategy:            overwriteStrategy,
		},
	}, nil
}

func GetPromoteAppVersionCommand(appContext app.Context) components.Command {
	cmd := &promoteAppVersionCommand{versionService: appContext.GetVersionService()}
	return components.Command{
		Name:        commands.VersionPromote,
		Description: "Promote application version.",
		Category:    common.CategoryVersion,
		Aliases:     []string{"vp"},
		Arguments: []components.Argument{
			{
				Name:        "application-key",
				Description: "The application key.",
				Optional:    false,
			},
			{
				Name:        "version",
				Description: "The version to promote.",
				Optional:    false,
			},
			{
				Name:        "target-stage",
				Description: "The target stage to which the application version should be promoted.",
				Optional:    false,
			},
		},
		Flags:            commands.GetCommandFlags(commands.VersionPromote),
		SupportedFormats: []coreformat.OutputFormat{coreformat.Table, coreformat.Json},
		Action:           cmd.prepareAndRunCommand,
	}
}
