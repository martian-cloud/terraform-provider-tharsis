package provider

import (
	"context"
	"time"

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

// WorkspaceVCSProviderLinkModel is the model for a workspace VCS provider link.
// Fields WebhookID, ModuleDirectory, and TagRegex are pointers in the SDK type but strings here.
type WorkspaceVCSProviderLinkModel struct {
	ID                  types.String   `tfsdk:"id"`
	LastUpdated         types.String   `tfsdk:"last_updated"`
	WorkspaceID         types.String   `tfsdk:"workspace_id"`
	WorkspacePath       types.String   `tfsdk:"workspace_path"`
	VCSProviderID       types.String   `tfsdk:"vcs_provider_id"`
	RepositoryPath      types.String   `tfsdk:"repository_path"`
	WebhookID           types.String   `tfsdk:"webhook_id"`
	ModuleDirectory     types.String   `tfsdk:"module_directory"`
	Branch              types.String   `tfsdk:"branch"`
	TagRegex            types.String   `tfsdk:"tag_regex"`
	GlobPatterns        []types.String `tfsdk:"glob_patterns"`
	AutoSpeculativePlan types.Bool     `tfsdk:"auto_speculative_plan"`
	WebhookDisabled     types.Bool     `tfsdk:"webhook_disabled"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*workspaceVCSProviderLinkResource)(nil)
	_ resource.ResourceWithConfigure   = (*workspaceVCSProviderLinkResource)(nil)
	_ resource.ResourceWithImportState = (*workspaceVCSProviderLinkResource)(nil)
)

// NewWorkspaceVCSProviderLinkResource is a helper function to simplify the provider implementation.
func NewWorkspaceVCSProviderLinkResource() resource.Resource {
	return &workspaceVCSProviderLinkResource{}
}

type workspaceVCSProviderLinkResource struct {
	client *client.GRPCClient
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *workspaceVCSProviderLinkResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = "tharsis_workspace_vcs_provider_link"
}

func (t *workspaceVCSProviderLinkResource) Schema(_ context.Context, _ resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	description := "Defines and manages a workspace VCS provider link."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the workspace VCS provider link.",
				Description:         "String identifier of the workspace VCS provider link.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the workspace.",
				Description:         "The ID of the workspace.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"workspace_path": schema.StringAttribute{
				MarkdownDescription: "The resource path of the workspace.",
				Description:         "The resource path of the workspace.",
				Optional:            true,
				DeprecationMessage:  "Use workspace_id instead. This field will be removed in a future version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vcs_provider_id": schema.StringAttribute{
				MarkdownDescription: "The string identifier of the  VCS provider.",
				Description:         "The string identifier of the  VCS provider.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"repository_path": schema.StringAttribute{
				MarkdownDescription: "The path portion of the repository URL.",
				Description:         "The path portion of the repository URL.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"webhook_id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the webhook.",
				Description:         "String identifier of the webhook.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"module_directory": schema.StringAttribute{
				MarkdownDescription: "The module's directory path.",
				Description:         "The module's directory path.",
				Required:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"branch": schema.StringAttribute{
				MarkdownDescription: "The repository branch.",
				Description:         "The repository branch.",
				Optional:            true,
				Computed:            true, // API sets a default value if not specified.
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"tag_regex": schema.StringAttribute{
				MarkdownDescription: "A regular expression that specifies which tags trigger runs.",
				Description:         "A regular expression that specifies which tags trigger runs.",
				Optional:            true,
				Computed:            true, // API sets a default value of nil if not specified.
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"glob_patterns": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Glob patterns to use for monitoring changes.",
				Description:         "Glob patterns to use for monitoring changes.",
				Required:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"auto_speculative_plan": schema.BoolAttribute{
				MarkdownDescription: "Whether to create speculative plans automatically for PRs.",
				Description:         "Whether to create speculative plans automatically for PRs.",
				Required:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"webhook_disabled": schema.BoolAttribute{
				MarkdownDescription: "Whether to disable the webhook.",
				Description:         "Whether to disable the webhook.",
				Required:            true,
				// Can be updated in place, so no RequiresReplace plan modifier.
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp when this workspace VCS provider link was most recently updated.",
				Description:         "Timestamp when this workspace VCS provider link was most recently updated.",
				Computed:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *workspaceVCSProviderLinkResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*client.GRPCClient)
}

func (t *workspaceVCSProviderLinkResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	// Retrieve values from workspace VCS provider link.
	var link WorkspaceVCSProviderLinkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &link)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve workspace ID: prefer workspace_id, fall back to workspace_path.
	var workspaceID string
	if v := link.WorkspaceID.ValueString(); v != "" {
		workspaceID = v
	} else if v := link.WorkspacePath.ValueString(); v != "" {
		workspaceID = trn.TypeWorkspace.Build(v)
	} else {
		resp.Diagnostics.AddError("Either workspace_id or workspace_path must be specified", "")
		return
	}

	// Create the workspace VCS provider link.
	input := &pb.CreateWorkspaceVCSProviderLinkRequest{
		WorkspaceId:         workspaceID,
		ProviderId:          link.VCSProviderID.ValueString(),
		RepositoryPath:      link.RepositoryPath.ValueString(),
		AutoSpeculativePlan: link.AutoSpeculativePlan.ValueBool(),
		WebhookDisabled:     link.WebhookDisabled.ValueBool(),
	}
	if v := link.ModuleDirectory.ValueString(); v != "" {
		input.ModuleDirectory = &v
	}

	if v := link.Branch.ValueString(); v != "" {
		input.Branch = &v
	}

	if v := link.TagRegex.ValueString(); v != "" {
		input.TagRegex = &v
	}

	for _, gp := range link.GlobPatterns {
		input.GlobPatterns = append(input.GlobPatterns, gp.ValueString())
	}

	createResponse, err := t.client.VCSProvidersClient.CreateWorkspaceVCSProviderLink(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating workspace VCS provider link",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	t.copyWorkspaceVCSProviderLink(createResponse.VcsProviderLink, &link)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, link)...)
}

func (t *workspaceVCSProviderLinkResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state.
	var state WorkspaceVCSProviderLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the workspace VCS provider link from Tharsis.
	found, err := t.client.VCSProvidersClient.GetWorkspaceVCSProviderLinkByID(ctx,
		&pb.GetWorkspaceVCSProviderLinkByIDRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading workspace VCS provider link",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	t.copyWorkspaceVCSProviderLink(found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *workspaceVCSProviderLinkResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	// Retrieve values from plan.
	var plan WorkspaceVCSProviderLinkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the workspace VCS provider link via Tharsis.
	// The ID is used to find the record to update.
	input := &pb.UpdateWorkspaceVCSProviderLinkRequest{
		Id:                  plan.ID.ValueString(),
		AutoSpeculativePlan: new(plan.AutoSpeculativePlan.ValueBool()),
		WebhookDisabled:     new(plan.WebhookDisabled.ValueBool()),
	}

	if v := plan.ModuleDirectory.ValueString(); v != "" {
		input.ModuleDirectory = &v
	}

	if v := plan.Branch.ValueString(); v != "" {
		input.Branch = &v
	}

	if v := plan.TagRegex.ValueString(); v != "" {
		input.TagRegex = &v
	}

	for _, gp := range plan.GlobPatterns {
		input.GlobPatterns = append(input.GlobPatterns, gp.ValueString())
	}

	updated, err := t.client.VCSProvidersClient.UpdateWorkspaceVCSProviderLink(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating workspace VCS provider link",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	t.copyWorkspaceVCSProviderLink(updated, &plan)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *workspaceVCSProviderLinkResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	// Get the current state.
	var state WorkspaceVCSProviderLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the workspace VCS provider link via Tharsis.
	_, err := t.client.VCSProvidersClient.DeleteWorkspaceVCSProviderLink(ctx,
		&pb.DeleteWorkspaceVCSProviderLinkRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the workspace VCS provider link no longer exists.
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting workspace VCS provider link",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *workspaceVCSProviderLinkResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyWorkspaceVCSProviderLink copies the contents of a workspace VCS provider link.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *workspaceVCSProviderLinkResource) copyWorkspaceVCSProviderLink(src *pb.WorkspaceVCSProviderLink,
	dest *WorkspaceVCSProviderLinkModel,
) {
	parsed := trn.MustParseAny(src.Metadata.Trn)
	dest.ID = types.StringValue(src.Metadata.Id)
	dest.WorkspaceID = types.StringValue(src.WorkspaceId)
	dest.WorkspacePath = types.StringValue(parsed.ParentPath())
	dest.VCSProviderID = types.StringValue(src.VcsProviderId)
	dest.RepositoryPath = types.StringValue(src.RepositoryPath)
	dest.WebhookID = types.StringValue(src.WebhookId)
	if src.ModuleDirectory != nil {
		dest.ModuleDirectory = types.StringValue(*src.ModuleDirectory)
	} else {
		dest.ModuleDirectory = types.StringNull()
	}
	dest.Branch = types.StringValue(src.Branch)
	if src.TagRegex != nil {
		dest.TagRegex = types.StringValue(*src.TagRegex)
	} else {
		dest.TagRegex = types.StringNull()
	}
	dest.GlobPatterns = []types.String{}
	for _, gp := range src.GlobPatterns {
		dest.GlobPatterns = append(dest.GlobPatterns, types.StringValue(gp))
	}
	dest.AutoSpeculativePlan = types.BoolValue(src.AutoSpeculativePlan)
	dest.WebhookDisabled = types.BoolValue(src.WebhookDisabled)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.UpdatedAt.AsTime().Format(time.RFC850))
}
