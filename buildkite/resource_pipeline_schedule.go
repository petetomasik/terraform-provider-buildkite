package buildkite

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	resource_schema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type pipelineSchedule struct {
	client *Client
}

type pipelineScheduleResourceModel struct {
	Id         types.String `tfsdk:"id"`
	Uuid       types.String `tfsdk:"uuid"`
	Label      types.String `tfsdk:"label"`
	Cronline   types.String `tfsdk:"cronline"`
	Commit     types.String `tfsdk:"commit"`
	Branch     types.String `tfsdk:"branch"`
	Message    types.String `tfsdk:"message"`
	Env        types.Map    `tfsdk:"env"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	PipelineId types.String `tfsdk:"pipeline_id"`
}

func newPipelineScheduleResource() resource.Resource {
	return &pipelineSchedule{}
}

func (ps *pipelineSchedule) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline_schedule"
}

func (ps *pipelineSchedule) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	ps.client = req.ProviderData.(*Client)
}

func (ps *pipelineSchedule) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_schema.Schema{
		MarkdownDescription: heredoc.Doc(`
			A pipeline schedule is a schedule that triggers a pipeline to run at a specific time.

			You can find more information in the [documentation](https://buildkite.com/docs/pipelines/scheduled-builds).
		`),
		Attributes: map[string]resource_schema.Attribute{
			"id": resource_schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The GraphQL ID of the schedule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": resource_schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The UUID of the schedule.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pipeline_id": resource_schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The GraphQL ID of the pipeline that this schedule belongs to.",
			},
			"label": resource_schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A label to describe the schedule.",
			},
			"cronline": resource_schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The cronline that describes when the schedule should run. See" +
					"[here](https://buildkite.com/docs/pipelines/scheduled-builds#schedule-intervals) for supported syntax.",
			},
			"branch": resource_schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The branch that the schedule should run on.",
			},
			"commit": resource_schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The commit that the schedule should run on.",
				Default:             stringdefault.StaticString("HEAD"),
			},
			"message": resource_schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The message the builds show for builds created by this schedule.",
			},
			"env": resource_schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "The environment variables that scheduled builds should use.",
			},
			"enabled": resource_schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the schedule is enabled or not.",
				Default:             booldefault.StaticBool(true),
			},
		},
	}
}

func (ps *pipelineSchedule) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan, state pipelineScheduleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	timeouts, diags := ps.client.timeouts.Create(ctx, DefaultTimeout)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	envVars := envVarsMapFromTfToString(ctx, plan.Env)
	var apiResponse *createPipelineScheduleResponse

	err := retry.RetryContext(ctx, timeouts, func() *retry.RetryError {
		var err error

		apiResponse, err = createPipelineSchedule(ctx,
			ps.client.genqlient,
			plan.PipelineId.ValueString(),
			plan.Label.ValueStringPointer(),
			plan.Cronline.ValueStringPointer(),
			plan.Message.ValueStringPointer(),
			plan.Commit.ValueStringPointer(),
			plan.Branch.ValueStringPointer(),
			&envVars,
			plan.Enabled.ValueBool())

		return retryContextError(err)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Pipeline schedule",
			fmt.Sprintf("Unable to create Pipeline schedule: %s", err.Error()),
		)
		return
	}

	state.PipelineId = types.StringValue(apiResponse.PipelineScheduleCreate.Pipeline.Id)
	state.Id = types.StringValue(apiResponse.PipelineScheduleCreate.PipelineScheduleEdge.Node.Id)
	state.Uuid = types.StringValue(apiResponse.PipelineScheduleCreate.PipelineScheduleEdge.Node.Uuid)
	state.Label = types.StringPointerValue(apiResponse.PipelineScheduleCreate.PipelineScheduleEdge.Node.Label)
	state.Branch = types.StringPointerValue(apiResponse.PipelineScheduleCreate.PipelineScheduleEdge.Node.Branch)
	state.Commit = types.StringPointerValue(apiResponse.PipelineScheduleCreate.PipelineScheduleEdge.Node.Commit)
	state.Cronline = types.StringPointerValue(apiResponse.PipelineScheduleCreate.PipelineScheduleEdge.Node.Cronline)
	state.Message = types.StringPointerValue(apiResponse.PipelineScheduleCreate.PipelineScheduleEdge.Node.Message)
	state.Enabled = types.BoolValue(apiResponse.PipelineScheduleCreate.PipelineScheduleEdge.Node.Enabled)
	state.Env = plan.Env

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (ps *pipelineSchedule) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pipelineScheduleResourceModel

	diagsState := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diagsState...)

	if resp.Diagnostics.HasError() {
		return
	}

	timeouts, diags := ps.client.timeouts.Read(ctx, DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResponse *getPipelineScheduleResponse
	err := retry.RetryContext(ctx, timeouts, func() *retry.RetryError {
		var err error

		apiResponse, err = getPipelineSchedule(ctx,
			ps.client.genqlient,
			state.Id.ValueString(),
		)

		return retryContextError(err)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read Pipeline schedule",
			fmt.Sprintf("Unable to read Pipeline schedule: %s", err.Error()),
		)
		return
	}

	if pipelineScheduleNode, ok := apiResponse.GetNode().(*getPipelineScheduleNodePipelineSchedule); ok {
		if pipelineScheduleNode == nil {
			resp.Diagnostics.AddError(
				"Unable to read Pipeline schedule",
				"Error getting Pipeline schedule: nil response",
			)
			return
		}
		updatePipelineScheduleNode(ctx, &state, *pipelineScheduleNode)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	} else {
		resp.Diagnostics.AddWarning(
			"Pipeline schedule not found",
			"Removing Pipeline schedule from state...",
		)
		resp.State.RemoveResource(ctx)
	}
}

func (ps *pipelineSchedule) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan pipelineScheduleResourceModel

	diagsState := req.State.Get(ctx, &state)
	diagsPlan := req.Plan.Get(ctx, &plan)

	resp.Diagnostics.Append(diagsState...)
	resp.Diagnostics.Append(diagsPlan...)

	if resp.Diagnostics.HasError() {
		return
	}

	timeouts, diags := ps.client.timeouts.Update(ctx, DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	envVars := envVarsMapFromTfToString(ctx, plan.Env)
	input := PipelineScheduleUpdateInput{
		Id:       state.Id.ValueString(),
		Label:    plan.Label.ValueStringPointer(),
		Cronline: plan.Cronline.ValueStringPointer(),
		Branch:   plan.Branch.ValueStringPointer(),
		Commit:   plan.Commit.ValueStringPointer(),
		Message:  plan.Message.ValueStringPointer(),
		Env:      &envVars,
		Enabled:  plan.Enabled.ValueBoolPointer(),
	}

	err := retry.RetryContext(ctx, timeouts, func() *retry.RetryError {
		var err error
		_, err = updatePipelineSchedule(ctx,
			ps.client.genqlient,
			input,
		)

		return retryContextError(err)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update Pipeline schedule",
			fmt.Sprintf("Unable to update Pipeline schedule: %s", err.Error()),
		)
	}

	plan.Id = state.Id
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (ps *pipelineSchedule) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan pipelineScheduleResourceModel

	diagsPlan := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diagsPlan...)

	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := ps.client.timeouts.Delete(ctx, DefaultTimeout)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		_, err := deletePipelineSchedule(ctx, ps.client.genqlient, plan.Id.ValueString())

		if err != nil && isResourceNotFoundError(err) {
			return nil
		}

		return retryContextError(err)
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete Pipeline schedule",
			fmt.Sprintf("Unable to delete Pipeline schedule: %s", err.Error()),
		)
		return
	}
}

func (ps *pipelineSchedule) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func envVarsArrayToMap(ctx context.Context, envVars []*string) types.Map {
	envVarsMap := make(map[string]string)
	for _, envVar := range envVars {
		if envVar != nil {
			key, value, _ := strings.Cut(*envVar, "=")
			envVarsMap[key] = value
		}
	}
	if len(envVarsMap) == 0 {
		return types.MapNull(types.StringType)
	}

	m, _ := types.MapValueFrom(ctx, types.StringType, envVarsMap)
	return m
}

func envVarsMapFromTfToString(ctx context.Context, m types.Map) string {
	b := new(bytes.Buffer)

	for key, value := range m.Elements() {
		fmt.Fprintf(b, "%s=%s\n", key, value)
	}
	return b.String()
}

func updatePipelineScheduleNode(ctx context.Context, psState *pipelineScheduleResourceModel, psNode getPipelineScheduleNodePipelineSchedule) {
	psState.Uuid = types.StringValue(psNode.Uuid)
	psState.Label = types.StringPointerValue(psNode.Label)
	psState.Branch = types.StringPointerValue(psNode.Branch)
	psState.Commit = types.StringPointerValue(psNode.Commit)
	psState.Cronline = types.StringPointerValue(psNode.Cronline)
	psState.Message = types.StringPointerValue(psNode.Message)
	psState.Enabled = types.BoolValue(psNode.Enabled)
	psState.Env = envVarsArrayToMap(ctx, psNode.Env)
	psState.PipelineId = types.StringValue(psNode.Pipeline.Id)
}
