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

// RunnerModel is the model for a runner.
// Fields intentionally omitted: Type (always "group" for a resource-created runner),
// CreatedBy, and the session/activity collections.
type RunnerModel struct {
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
	_ resource.Resource                = (*runnerResource)(nil)
	_ resource.ResourceWithConfigure   = (*runnerResource)(nil)
	_ resource.ResourceWithImportState = (*runnerResource)(nil)
	_ resource.ResourceWithModifyPlan  = (*runnerResource)(nil)
)

// NewRunnerResource is a helper function to simplify the provider implementation.
func NewRunnerResource() resource.Resource {
	return &runnerResource{}
}

type runnerResource struct {
	client *client.GRPCClient
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *runnerResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = "tharsis_runner"
}

func (t *runnerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a group runner."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the runner.",
				Description:         "String identifier of the runner.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"resource_path": schema.StringAttribute{
				MarkdownDescription: "The path of the parent group plus the name of the runner.",
				Description:         "The path of the parent group plus the name of the runner.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the runner.",
				Description:         "The name of the runner.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the runner.",
				Description:         "A description of the runner.",
				Optional:            true,
				Computed:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"group_id": schema.StringAttribute{
				MarkdownDescription: "The ID or TRN of the parent group. Runners are group-scoped.",
				Description:         "The ID or TRN of the parent group. Runners are group-scoped.",
				Required:            true,
				// The API has no update for the group. ModifyPlan forces a replacement when
				// group_id moves to a different group; a switch between the group's Global
				// ID and its TRN is an in-place update that only changes the stored form.
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the runner is prevented from claiming jobs. Defaults to false.",
				Description:         "Whether the runner is prevented from claiming jobs. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"run_untagged_jobs": schema.BoolAttribute{
				MarkdownDescription: "Whether the runner claims jobs that carry no tags. Defaults to false.",
				Description:         "Whether the runner claims jobs that carry no tags. Defaults to false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"tags": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Tags that route tagged jobs to this runner. Changing a non-empty set to empty forces replacement, because the API cannot clear tags through an update.",
				Description:         "Tags that route tagged jobs to this runner. Changing a non-empty set to empty forces replacement, because the API cannot clear tags through an update.",
				Optional:            true,
				Computed:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *runnerResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*client.GRPCClient)
}

// ModifyPlan forces a replacement when group_id moves to a different group, and when a
// non-empty tag set is being emptied: UpdateRunner ignores an empty tags list
// (if len(req.Tags) > 0), so the only way to actually clear tags is to recreate the
// runner.
func (t *runnerResource) ModifyPlan(ctx context.Context,
	req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse,
) {
	// Only relevant on update: no prior state on create, nothing to do on destroy.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	requireReplaceIfDifferentObject(ctx, t.client, req, resp, path.Root("group_id"), trn.TypeGroup)

	var state, plan RunnerModel
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

func (t *runnerResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	var runner RunnerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &runner)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tags []string
	if !runner.Tags.IsNull() && !runner.Tags.IsUnknown() {
		resp.Diagnostics.Append(runner.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	created, err := t.client.RunnersClient.CreateRunner(ctx,
		&pb.CreateRunnerRequest{
			Name:            runner.Name.ValueString(),
			Description:     runner.Description.ValueString(),
			GroupId:         runner.GroupID.ValueString(),
			Disabled:        new(runner.Disabled.ValueBool()),
			RunUntaggedJobs: runner.RunUntaggedJobs.ValueBool(),
			Tags:            tags,
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating runner",
			err.Error(),
		)
		return
	}

	plannedGroupID := runner.GroupID
	resp.Diagnostics.Append(t.copyRunner(ctx, created, &runner)...)
	// The API acted on the planned reference, so the planned form is kept.
	runner.GroupID = keepPlanned(plannedGroupID, runnerGroupGID(created))
	resp.Diagnostics.Append(resp.State.Set(ctx, runner)...)
}

func (t *runnerResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	var state RunnerModel
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
			"Error reading runner",
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(t.copyRunner(ctx, found, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *runnerResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {

	var plan RunnerModel
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
			"Error updating runner",
			err.Error(),
		)
		return
	}

	plannedGroupID := plan.GroupID
	resp.Diagnostics.Append(t.copyRunner(ctx, updated, &plan)...)
	// The API acted on the planned reference, so the planned form is kept.
	plan.GroupID = keepPlanned(plannedGroupID, runnerGroupGID(updated))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *runnerResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	var state RunnerModel
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
			"Error deleting runner",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *runnerResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyRunner copies the contents of a runner returned by Tharsis into a
// Terraform plan or state.
func (t *runnerResource) copyRunner(ctx context.Context,
	src *pb.Runner, dest *RunnerModel,
) (diags diag.Diagnostics) {
	parsed := trn.MustParseAny(src.Metadata.Trn)
	dest.ID = types.StringValue(src.Metadata.Id)
	dest.ResourcePath = types.StringValue(parsed.Path())
	dest.Name = types.StringValue(src.Name)
	dest.Description = types.StringValue(src.Description)
	// Preserve the caller's group_id input (a TRN or a Global ID); the API returns the
	// Global ID, which would otherwise overwrite a TRN input and fail Terraform's
	// consistency check. On import the field is null and is populated from the API,
	// falling back to the parent path so the attribute never goes null.
	{
		apiGID := trn.TypeGroup.Build(parsed.ParentPath())
		if src.GroupId != nil && *src.GroupId != "" {
			apiGID = *src.GroupId
		}
		dest.GroupID = keepOrRefresh(dest.GroupID, apiGID, trn.TypeGroup.Build(parsed.ParentPath()))
	}
	dest.Disabled = types.BoolValue(src.Disabled)
	dest.RunUntaggedJobs = types.BoolValue(src.RunUntaggedJobs)

	tagSet, d := types.SetValueFrom(ctx, types.StringType, src.Tags)
	diags = append(diags, d...)
	dest.Tags = tagSet

	return diags
}

// runnerGroupGID returns the runner's parent group as the API reports it: its Global
// ID, or the group TRN when the response carries no Global ID.
func runnerGroupGID(src *pb.Runner) string {
	if src.GroupId != nil && *src.GroupId != "" {
		return *src.GroupId
	}
	return trn.TypeGroup.Build(trn.MustParseAny(src.Metadata.Trn).ParentPath())
}
