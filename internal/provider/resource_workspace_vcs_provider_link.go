package provider

import (
	"context"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
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
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *workspaceVCSProviderLinkResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_workspace_vcs_provider_link"
}

func (t *workspaceVCSProviderLinkResource) Schema(_ context.Context, _ resource.SchemaRequest,
	resp *resource.SchemaResponse) {
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
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"workspace_path": schema.StringAttribute{
				MarkdownDescription: "The resource path of the workspace.",
				Description:         "The resource path of the workspace.",
				Required:            true,
			},
			"vcs_provider_id": schema.StringAttribute{
				MarkdownDescription: "The string identifier of the  VCS provider.",
				Description:         "The string identifier of the  VCS provider.",
				Required:            true,
			},
			"repository_path": schema.StringAttribute{
				MarkdownDescription: "The path portion of the repository URL.",
				Description:         "The path portion of the repository URL.",
				Required:            true,
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
			},
			"branch": schema.StringAttribute{
				MarkdownDescription: "The repository branch.",
				Description:         "The repository branch.",
				Optional:            true,
				Computed:            true, // API sets a default value if not specified.
			},
			"tag_regex": schema.StringAttribute{
				MarkdownDescription: "A regular expression that specifies which tags trigger runs.",
				Description:         "A regular expression that specifies which tags trigger runs.",
				Optional:            true,
				Computed:            true, // API sets a default value of nil if not specified.
			},
			"glob_patterns": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Glob patterns to use for monitoring changes.",
				Description:         "Glob patterns to use for monitoring changes.",
				Required:            true,
			},
			"auto_speculative_plan": schema.BoolAttribute{
				MarkdownDescription: "Whether to create speculative plans automatically for PRs.",
				Description:         "Whether to create speculative plans automatically for PRs.",
				Required:            true,
			},
			"webhook_disabled": schema.BoolAttribute{
				MarkdownDescription: "Whether to disable the webhook.",
				Description:         "Whether to disable the webhook.",
				Required:            true,
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
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *workspaceVCSProviderLinkResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from workspace VCS provider link.
	var workspaceVCSProviderLink WorkspaceVCSProviderLinkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &workspaceVCSProviderLink)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the workspace VCS provider link.
	var moduleDirectory *string
	if workspaceVCSProviderLink.ModuleDirectory.ValueString() != "" {
		moduleDirectory = ptr.String(workspaceVCSProviderLink.ModuleDirectory.ValueString())
	}
	var branch *string
	if workspaceVCSProviderLink.Branch.ValueString() != "" {
		branch = ptr.String(workspaceVCSProviderLink.Branch.ValueString())
	}
	var tagRegex *string
	if workspaceVCSProviderLink.TagRegex.ValueString() != "" {
		tagRegex = ptr.String(workspaceVCSProviderLink.TagRegex.ValueString())
	}
	globPatterns := []string{}
	for _, gp := range workspaceVCSProviderLink.GlobPatterns {
		globPatterns = append(globPatterns, gp.ValueString())
	}
	createResponse, err := t.client.WorkspaceVCSProviderLink.CreateLink(ctx,
		&ttypes.CreateWorkspaceVCSProviderLinkInput{
			ModuleDirectory:     moduleDirectory,
			RepositoryPath:      workspaceVCSProviderLink.RepositoryPath.ValueString(),
			WorkspacePath:       workspaceVCSProviderLink.WorkspacePath.ValueString(),
			ProviderID:          workspaceVCSProviderLink.VCSProviderID.ValueString(),
			Branch:              branch,
			TagRegex:            tagRegex,
			GlobPatterns:        globPatterns,
			AutoSpeculativePlan: workspaceVCSProviderLink.AutoSpeculativePlan.ValueBool(),
			WebhookDisabled:     workspaceVCSProviderLink.WebhookDisabled.ValueBool(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating workspace VCS provider link",
			err.Error(),
		)
		return
	}
	created := createResponse.VCSProviderLink

	// Map the response body to the schema and update the plan with the computed attribute values.
	t.copyWorkspaceVCSProviderLink(created, &workspaceVCSProviderLink)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, workspaceVCSProviderLink)...)
}

func (t *workspaceVCSProviderLinkResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state WorkspaceVCSProviderLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the workspace VCS provider link from Tharsis.
	found, err := t.client.WorkspaceVCSProviderLink.GetLink(ctx, &ttypes.GetWorkspaceVCSProviderLinkInput{
		ID: state.ID.ValueString(),
	})
	if err != nil {
		if tharsis.IsNotFoundError(err) {
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
	t.copyWorkspaceVCSProviderLink(*found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *workspaceVCSProviderLinkResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan WorkspaceVCSProviderLinkModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the workspace VCS provider link via Tharsis.
	// The ID is used to find the record to update.
	var moduleDirectory *string
	if plan.ModuleDirectory.ValueString() != "" {
		moduleDirectory = ptr.String(plan.ModuleDirectory.ValueString())
	}
	var branch *string
	if plan.Branch.ValueString() != "" {
		branch = ptr.String(plan.Branch.ValueString())
	}
	var tagRegex *string
	if plan.TagRegex.ValueString() != "" {
		tagRegex = ptr.String(plan.TagRegex.ValueString())
	}
	globPatterns := []string{}
	for _, gp := range plan.GlobPatterns {
		globPatterns = append(globPatterns, gp.ValueString())
	}
	updated, err := t.client.WorkspaceVCSProviderLink.UpdateLink(ctx,
		&ttypes.UpdateWorkspaceVCSProviderLinkInput{
			ID:                  plan.ID.ValueString(),
			ModuleDirectory:     moduleDirectory,
			Branch:              branch,
			TagRegex:            tagRegex,
			GlobPatterns:        globPatterns,
			AutoSpeculativePlan: plan.AutoSpeculativePlan.ValueBool(),
			WebhookDisabled:     plan.WebhookDisabled.ValueBool(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating workspace VCS provider link",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	t.copyWorkspaceVCSProviderLink(*updated, &plan)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *workspaceVCSProviderLinkResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state WorkspaceVCSProviderLinkModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the workspace VCS provider link via Tharsis.
	_, err := t.client.WorkspaceVCSProviderLink.DeleteLink(ctx,
		&ttypes.DeleteWorkspaceVCSProviderLinkInput{
			ID: state.ID.ValueString(),
		})
	if err != nil {

		// Handle the case that the workspace VCS provider link no longer exists.
		if tharsis.IsNotFoundError(err) {
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
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Get the workspace VCS provider link by full path from Tharsis.
	found, err := t.client.WorkspaceVCSProviderLink.GetLink(ctx, &ttypes.GetWorkspaceVCSProviderLinkInput{
		ID: req.ID, // FIXME: What should this be searching for?
	})
	if err != nil {
		if tharsis.IsNotFoundError(err) {
			resp.Diagnostics.AddError(
				"Import workspace VCS provider link not found: "+req.ID,
				"",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Import workspace VCS provider link not found: "+req.ID,
			err.Error(),
		)
		return
	}

	// Import by full path.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.Metadata.ID)...)
}

// copyWorkspaceVCSProviderLink copies the contents of a workspace VCS provider link.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *workspaceVCSProviderLinkResource) copyWorkspaceVCSProviderLink(src ttypes.WorkspaceVCSProviderLink,
	dest *WorkspaceVCSProviderLinkModel) {
	dest.ID = types.StringValue(src.Metadata.ID)
	dest.WorkspaceID = types.StringValue(src.WorkspaceID)
	dest.WorkspacePath = types.StringValue(src.WorkspacePath)
	dest.VCSProviderID = types.StringValue(src.VCSProviderID)
	dest.RepositoryPath = types.StringValue(src.RepositoryPath)
	if src.WebhookID != nil {
		dest.WebhookID = types.StringValue(*src.WebhookID)
	}
	if src.ModuleDirectory != nil {
		dest.ModuleDirectory = types.StringValue(*src.ModuleDirectory)
	}
	dest.Branch = types.StringValue(src.Branch)
	if src.TagRegex != nil {
		dest.TagRegex = types.StringValue(*src.TagRegex)
	}
	dest.GlobPatterns = []types.String{}
	for _, gp := range src.GlobPatterns {
		dest.GlobPatterns = append(dest.GlobPatterns, types.StringValue(gp))
	}
	dest.AutoSpeculativePlan = types.BoolValue(src.AutoSpeculativePlan)
	dest.WebhookDisabled = types.BoolValue(src.WebhookDisabled)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.LastUpdatedTimestamp.Format(time.RFC850))
}

// The End.
