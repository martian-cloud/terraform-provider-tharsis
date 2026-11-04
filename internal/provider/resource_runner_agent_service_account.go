package provider

import (
	"context"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RunnerAgentServiceAccountModel is the model for a runner-agent service-account
// assignment. The two input IDs are stored exactly as the caller supplied them (a TRN
// or a Global ID); they are resolved to Global IDs only transiently for the API calls,
// so the state value of a Required attribute never changes during apply.
type RunnerAgentServiceAccountModel struct {
	ID               types.String `tfsdk:"id"`
	RunnerAgentID    types.String `tfsdk:"runner_agent_id"`
	ServiceAccountID types.String `tfsdk:"service_account_id"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource               = (*runnerAgentServiceAccountResource)(nil)
	_ resource.ResourceWithConfigure  = (*runnerAgentServiceAccountResource)(nil)
	_ resource.ResourceWithModifyPlan = (*runnerAgentServiceAccountResource)(nil)
)

// NewRunnerAgentServiceAccountResource is a helper function to simplify the provider implementation.
func NewRunnerAgentServiceAccountResource() resource.Resource {
	return &runnerAgentServiceAccountResource{}
}

type runnerAgentServiceAccountResource struct {
	client *client.GRPCClient
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *runnerAgentServiceAccountResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = "tharsis_runner_agent_service_account"
}

func (t *runnerAgentServiceAccountResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Assigns a service account to a runner agent, letting the runner agent authenticate as it."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "An ID for this tharsis_runner_agent_service_account resource.",
				Description:         "An ID for this tharsis_runner_agent_service_account resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(), // set once during create, kept in state thereafter
				},
			},
			"runner_agent_id": schema.StringAttribute{
				MarkdownDescription: "The ID or TRN of the runner agent.",
				Description:         "The ID or TRN of the runner agent.",
				Required:            true,
			},
			"service_account_id": schema.StringAttribute{
				MarkdownDescription: "The ID or TRN of the service account to assign.",
				Description:         "The ID or TRN of the service account to assign.",
				Required:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *runnerAgentServiceAccountResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*client.GRPCClient)
}

// resolveIDs turns a runner agent and service account input (each a TRN or a Global ID)
// into their Global IDs, and returns the runner agent's group namespace path. The IDs
// are used for the API calls; the namespace path scopes the Read presence lookup. None
// of these are written back to state.
func (t *runnerAgentServiceAccountResource) resolveIDs(ctx context.Context,
	runnerInput, serviceAccountInput string,
) (runnerAgentID string, serviceAccountID string, runnerNamespacePath string, notFound bool, diags diag.Diagnostics) {
	runner, err := t.client.RunnersClient.GetRunnerByID(ctx, &pb.GetRunnerByIDRequest{Id: runnerInput})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", "", "", true, diags
		}
		diags.AddError("Error resolving runner agent", err.Error())
		return
	}
	serviceAccount, err := t.client.ServiceAccountsClient.GetServiceAccountByID(ctx, &pb.GetServiceAccountByIDRequest{Id: serviceAccountInput})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", "", "", true, diags
		}
		diags.AddError("Error resolving service account", err.Error())
		return
	}
	// The runner agent's TRN is trn:runner:<group-path>/<name>; its parent path is the
	// group namespace the assigned service accounts live in.
	namespacePath := trn.MustParseAny(runner.Metadata.Trn).ParentPath()
	return runner.Metadata.Id, serviceAccount.Metadata.Id, namespacePath, false, diags
}

func (t *runnerAgentServiceAccountResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	var assignment RunnerAgentServiceAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &assignment)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the assignment. AssignServiceAccountToRunner resolves a TRN or a Global ID
	// internally, so the caller's inputs are passed through as-is.
	_, err := t.client.RunnersClient.AssignServiceAccountToRunner(ctx,
		&pb.AssignServiceAccountToRunnerRequest{
			RunnerId:         assignment.RunnerAgentID.ValueString(),
			ServiceAccountId: assignment.ServiceAccountID.ValueString(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning service account to runner agent",
			err.Error(),
		)
		return
	}

	// Preserve the caller's input values; only the computed id is set.
	assignment.ID = types.StringValue(uuid.New().String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &assignment)...)
}

func (t *runnerAgentServiceAccountResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	var state RunnerAgentServiceAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// state.RunnerAgentID / ServiceAccountID hold the caller's input (TRN or Global ID).
	// Resolve both to Global IDs so the presence check compares like with like.
	runnerAgentID, wantID, namespacePath, notFound, diags := t.resolveIDs(ctx,
		state.RunnerAgentID.ValueString(), state.ServiceAccountID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// A deleted runner agent or service account means the assignment is gone; drop it
	// from state so a plan can recreate rather than hard-failing every run.
	if notFound {
		resp.State.RemoveResource(ctx)
		return
	}

	// The API always applies a namespace filter, so scope the lookup to the runner
	// agent's group and its ancestors (IncludeInherited) where assigned service accounts
	// may live. RunnerId then narrows the result to this runner agent's assignments.
	// Page through all results: a match on a later page must not be missed, or Read
	// would wrongly drop the resource and plan a recreate.
	var present bool
	var after *string
	// The API requires first or last on a paginated request.
	pageSize := int32(100)
	for {
		found, err := t.client.ServiceAccountsClient.GetServiceAccounts(ctx,
			&pb.GetServiceAccountsRequest{
				RunnerId:          &runnerAgentID,
				NamespacePath:     namespacePath,
				IncludeInherited:  true,
				PaginationOptions: &pb.PaginationOptions{First: &pageSize, After: after},
			})
		if err != nil {
			if status.Code(err) == codes.NotFound {
				resp.State.RemoveResource(ctx)
				return
			}

			resp.Diagnostics.AddError(
				"Error reading runner agent service account assignments",
				err.Error(),
			)
			return
		}

		for _, candidate := range found.ServiceAccounts {
			if candidate.Metadata.Id == wantID {
				present = true
				break
			}
		}
		if present || found.PageInfo == nil || !found.PageInfo.HasNextPage {
			break
		}
		// Guard against a server that claims another page but does not advance the
		// cursor: an empty or unchanged EndCursor would otherwise refetch page one
		// forever. Stop instead of looping.
		if found.PageInfo.EndCursor == nil || *found.PageInfo.EndCursor == "" ||
			(after != nil && *found.PageInfo.EndCursor == *after) {
			break
		}
		after = found.PageInfo.EndCursor
	}
	if !present {
		// The assignment is gone; drop it from state so a re-apply recreates it.
		resp.State.RemoveResource(ctx)
		return
	}

	// Nothing to refresh beyond confirming presence: the two IDs are immutable inputs
	// (RequiresReplace) and the manufactured ID is kept as-is.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *runnerAgentServiceAccountResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	// Reached only when runner_agent_id or service_account_id switches between an object's Global ID and its TRN:
	// ModifyPlan forces a replacement for any other change. Nothing changes in Tharsis, so
	// only the stored form is updated.
	var plan, state RunnerAgentServiceAccountModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.RunnerAgentID = plan.RunnerAgentID
	state.ServiceAccountID = plan.ServiceAccountID
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// ModifyPlan forces a replacement only when either runner_agent_id or service_account_id moves to a
// different object. A switch between a Global ID and a TRN of the same object is an
// in-place update.
func (t *runnerAgentServiceAccountResource) ModifyPlan(ctx context.Context,
	req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse,
) {
	requireReplaceIfDifferentObject(ctx, t.client, req, resp, path.Root("runner_agent_id"), trn.TypeRunner)
	requireReplaceIfDifferentObject(ctx, t.client, req, resp, path.Root("service_account_id"), trn.TypeServiceAccount)
}

func (t *runnerAgentServiceAccountResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	var state RunnerAgentServiceAccountModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := t.client.RunnersClient.UnassignServiceAccountFromRunner(ctx,
		&pb.UnassignServiceAccountFromRunnerRequest{
			RunnerId:         state.RunnerAgentID.ValueString(),
			ServiceAccountId: state.ServiceAccountID.ValueString(),
		})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error unassigning service account from runner agent",
			err.Error(),
		)
	}
}
