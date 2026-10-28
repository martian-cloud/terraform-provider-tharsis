package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RunnerAgentModel is the model for a runner agent.
// Fields intentionally omitted: Type (always "group" for a resource-created runner agent),
// CreatedBy, and the session/activity collections.
type RunnerAgentModel struct {
	ID              types.String `tfsdk:"id"`
	ResourcePath    types.String `tfsdk:"resource_path"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	GroupID         types.String `tfsdk:"group_id"`
	Disabled        types.Bool   `tfsdk:"disabled"`
	RunUntaggedJobs types.Bool   `tfsdk:"run_untagged_jobs"`
	Tags            types.Set    `tfsdk:"tags"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*runnerAgentResource)(nil)
	_ resource.ResourceWithConfigure   = (*runnerAgentResource)(nil)
	_ resource.ResourceWithImportState = (*runnerAgentResource)(nil)
)

// NewRunnerAgentResource is a helper function to simplify the provider implementation.
func NewRunnerAgentResource() resource.Resource {
	return &runnerAgentResource{}
}

type runnerAgentResource struct {
	client *client.GRPCClient
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *runnerAgentResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = "tharsis_runner_agent"
}

func (t *runnerAgentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a group runner agent."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the runner agent.",
				Description:         "String identifier of the runner agent.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_path": schema.StringAttribute{
				MarkdownDescription: "The path of the parent group plus the name of the runner agent.",
				Description:         "The path of the parent group plus the name of the runner agent.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the runner agent.",
				Description:         "The name of the runner agent.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the runner agent.",
				Description:         "A description of the runner agent.",
				Optional:            true,
				Computed:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The ID or TRN of the parent group. Runner agents are group-scoped.",
				Description:         "The ID or TRN of the parent group. Runner agents are group-scoped.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					// The API has no update for the group, so a change forces replacement.
					// UseStateForUnknown keeps the as-returned value so a TRN input does not
					// plan a perpetual diff against the Global ID the API returns.
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the runner agent is prevented from claiming jobs. Defaults to false.",
				Description:         "Whether the runner agent is prevented from claiming jobs. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"run_untagged_jobs": schema.BoolAttribute{
				MarkdownDescription: "Whether the runner agent claims jobs that carry no tags. Defaults to false.",
				Description:         "Whether the runner agent claims jobs that carry no tags. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"tags": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Tags that route tagged jobs to this runner agent. Changing a non-empty set to empty forces replacement, because the API cannot clear tags through an update.",
				Description:         "Tags that route tagged jobs to this runner agent. Changing a non-empty set to empty forces replacement, because the API cannot clear tags through an update.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *runnerAgentResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*client.GRPCClient)
}

// ModifyPlan forces replacement when a non-empty tag set is being emptied.
// UpdateRunner ignores an empty tags list (if len(req.Tags) > 0), so the only way
// to actually clear tags is to recreate the runner agent.
func (t *runnerAgentResource) ModifyPlan(ctx context.Context,
	req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse,
) {
	// Only relevant on update: no prior state on create, nothing to do on destroy.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan RunnerAgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A known, empty plan set replacing a known, non-empty state set cannot be applied
	// in place, so require replacement.
	if !state.Tags.IsNull() && !state.Tags.IsUnknown() &&
		!plan.Tags.IsNull() && !plan.Tags.IsUnknown() &&
		len(state.Tags.Elements()) > 0 && len(plan.Tags.Elements()) == 0 {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("tags"))
	}
}

func (t *runnerAgentResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	var runnerAgent RunnerAgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &runnerAgent)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tags []string
	if !runnerAgent.Tags.IsNull() && !runnerAgent.Tags.IsUnknown() {
		resp.Diagnostics.Append(runnerAgent.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	created, err := t.client.RunnersClient.CreateRunner(ctx,
		&pb.CreateRunnerRequest{
			Name:            runnerAgent.Name.ValueString(),
			Description:     runnerAgent.Description.ValueString(),
			GroupId:         runnerAgent.GroupID.ValueString(),
			Disabled:        new(runnerAgent.Disabled.ValueBool()),
			RunUntaggedJobs: runnerAgent.RunUntaggedJobs.ValueBool(),
			Tags:            tags,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating runner agent",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(t.copyRunnerAgent(ctx, created, &runnerAgent)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, runnerAgent)...)
}

func (t *runnerAgentResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	var state RunnerAgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := t.client.RunnersClient.GetRunnerByID(ctx,
		&pb.GetRunnerByIDRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading runner agent",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(t.copyRunnerAgent(ctx, found, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *runnerAgentResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	var plan RunnerAgentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tags []string
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Name and group_id both force replacement, so they are not sent here. An empty
	// tag set cannot reach this point: ModifyPlan forces replacement for that case.
	updated, err := t.client.RunnersClient.UpdateRunner(ctx,
		&pb.UpdateRunnerRequest{
			Id:              plan.ID.ValueString(),
			Description:     new(plan.Description.ValueString()),
			Disabled:        new(plan.Disabled.ValueBool()),
			RunUntaggedJobs: new(plan.RunUntaggedJobs.ValueBool()),
			Tags:            tags,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating runner agent",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(t.copyRunnerAgent(ctx, updated, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *runnerAgentResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	var state RunnerAgentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := t.client.RunnersClient.DeleteRunner(ctx,
		&pb.DeleteRunnerRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting runner agent",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *runnerAgentResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyRunnerAgent copies the contents of a runner agent returned by Tharsis into a
// Terraform plan or state.
func (t *runnerAgentResource) copyRunnerAgent(ctx context.Context,
	src *pb.Runner, dest *RunnerAgentModel,
) (diags diag.Diagnostics) {
	parsed := trn.MustParseAny(src.Metadata.Trn)
	dest.ID = types.StringValue(src.Metadata.Id)
	dest.ResourcePath = types.StringValue(parsed.Path())
	dest.Name = types.StringValue(src.Name)
	dest.Description = types.StringValue(src.Description)
	// GroupId is a pointer on the response; fall back to the parent path if the API
	// does not populate it, so the attribute never goes null.
	if src.GroupId != nil && *src.GroupId != "" {
		dest.GroupID = types.StringValue(*src.GroupId)
	} else {
		dest.GroupID = types.StringValue(trn.TypeGroup.Build(parsed.ParentPath()))
	}
	dest.Disabled = types.BoolValue(src.Disabled)
	dest.RunUntaggedJobs = types.BoolValue(src.RunUntaggedJobs)

	tagSet, d := types.SetValueFrom(ctx, types.StringType, src.Tags)
	diags = append(diags, d...)
	dest.Tags = tagSet

	return diags
}
