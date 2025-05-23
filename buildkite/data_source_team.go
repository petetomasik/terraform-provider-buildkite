package buildkite

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type teamDatasourceModel struct {
	ID                        types.String `tfsdk:"id"`
	UUID                      types.String `tfsdk:"uuid"`
	Slug                      types.String `tfsdk:"slug"`
	Name                      types.String `tfsdk:"name"`
	Privacy                   types.String `tfsdk:"privacy"`
	Description               types.String `tfsdk:"description"`
	IsDefaultTeam             types.Bool   `tfsdk:"default_team"`
	DefaultMemberRole         types.String `tfsdk:"default_member_role"`
	MembersCanCreatePipelines types.Bool   `tfsdk:"members_can_create_pipelines"`
}

type teamDatasource struct {
	client *Client
}

func newTeamDatasource() datasource.DataSource {
	return &teamDatasource{}
}

func (t *teamDatasource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	t.client = req.ProviderData.(*Client)
}

func (t *teamDatasource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (t *teamDatasource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: heredoc.Doc(`
			Use this data source to retrieve a team by slug or id. You can find out more about teams in the Buildkite
			[documentation](https://buildkite.com/docs/pipelines/permissions).
		`),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The GraphQL ID of the team to find.",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("slug"),
						path.MatchRoot("id"),
					}...),
				},
			},
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the team.",
				Computed:            true,
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "The slug of the team to find.",
				Computed:            true,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the team.",
				Computed:            true,
			},
			"privacy": schema.StringAttribute{
				MarkdownDescription: "The privacy setting of the team.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "The description of the team.",
				Computed:            true,
			},
			"default_team": schema.BoolAttribute{
				MarkdownDescription: "Whether the team is the default team.",
				Computed:            true,
			},
			"default_member_role": schema.StringAttribute{
				MarkdownDescription: "The default member role of the team.",
				Computed:            true,
			},
			"members_can_create_pipelines": schema.BoolAttribute{
				MarkdownDescription: "Whether members can create pipelines.",
				Computed:            true,
			},
		},
	}
}

func (t *teamDatasource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state teamDatasourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !state.Slug.IsNull() {
		res, err := GetTeamFromSlug(ctx, t.client.genqlient, fmt.Sprintf("%s/%s", t.client.organization, state.Slug.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to get team",
				fmt.Sprintf("Error getting team: %s", err.Error()),
			)
			return
		}
		updateTeamDatasourceStateFromSlug(&state, *res)
	} else if !state.ID.IsNull() {
		res, err := getNode(ctx, t.client.genqlient, state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to get team",
				fmt.Sprintf("Error getting team: %s", err.Error()),
			)
			return
		}
		if teamNode, ok := res.GetNode().(*getNodeNodeTeam); ok {
			if !ok {
				resp.Diagnostics.AddError(
					"Unable to get team",
					"Error getting team. Please create a new issue with any log output from the error here: https://github.com/buildkite/terraform-provider-buildkite/issues/new",
				)
				return
			}
			updateTeamDatasourceState(&state, *teamNode)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func updateTeamDatasourceStateFromSlug(state *teamDatasourceModel, data GetTeamFromSlugResponse) {
	state.ID = types.StringValue(data.Team.Id)
	state.UUID = types.StringValue(data.Team.Uuid)
	state.Slug = types.StringValue(data.Team.Slug)
	state.Name = types.StringValue(data.Team.Name)
	state.Privacy = types.StringValue(data.Team.Privacy)
	state.Description = types.StringValue(data.Team.Description)
	state.IsDefaultTeam = types.BoolValue(data.Team.IsDefaultTeam)
	state.DefaultMemberRole = types.StringValue(data.Team.DefaultMemberRole)
	state.MembersCanCreatePipelines = types.BoolValue(data.Team.MembersCanCreatePipelines)
}

func updateTeamDatasourceState(state *teamDatasourceModel, data getNodeNodeTeam) {
	state.ID = types.StringValue(data.Id)
	state.UUID = types.StringValue(data.Uuid)
	state.Slug = types.StringValue(data.Slug)
	state.Name = types.StringValue(data.Name)
	state.Privacy = types.StringValue(string(data.GetPrivacy()))
	state.Description = types.StringValue(data.Description)
	state.IsDefaultTeam = types.BoolValue(data.IsDefaultTeam)
	state.DefaultMemberRole = types.StringValue(string(data.GetDefaultMemberRole()))
	state.MembersCanCreatePipelines = types.BoolValue(data.MembersCanCreatePipelines)
}
