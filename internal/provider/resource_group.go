package provider

import (
	"context"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	ttypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// GroupModel is the model for a group.
type GroupModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ParentPath  types.String `tfsdk:"parent_path"`
	FullPath    types.String `tfsdk:"full_path"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &groupResource{}
	_ resource.ResourceWithConfigure   = &groupResource{}
	_ resource.ResourceWithImportState = &groupResource{}
)

// NewGroupResource is a helper function to simplify the provider implementation.
func NewGroupResource() resource.Resource {
	return &groupResource{}
}

type groupResource struct {
	client *tharsis.Client
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *groupResource) Metadata(ctx context.Context,
	req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "tharsis_group"
}

// The diagnostics return value is required by the interface even though this function returns only nil.
func (t *groupResource) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	description := "Defines and manages a group."

	return tfsdk.Schema{
		Version: 1,

		MarkdownDescription: description,
		Description:         description,

		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Type:                types.StringType,
				MarkdownDescription: "String identifier of the group.",
				Description:         "String identifier of the group.",
				Computed:            true,
				PlanModifiers: tfsdk.AttributePlanModifiers{
					resource.UseStateForUnknown(),
				},
			},
			"name": {
				Type:                types.StringType,
				MarkdownDescription: "The name of the group.",
				Description:         "The name of the group.",
				Required:            true,
			},
			"description": {
				Type:                types.StringType,
				MarkdownDescription: "A description of the group.",
				Description:         "A description of the group.",
				Optional:            true,
			},
			"parent_path": {
				Type:                types.StringType,
				MarkdownDescription: "Full path of the parent namespace.",
				Description:         "Full path of the parent namespace.",
				Required:            true,
			},
			"full_path": {
				Type:                types.StringType,
				MarkdownDescription: "The path of the parent namespace plus the name of the group.",
				Description:         "The path of the parent namespace plus the name of the group.",
				Computed:            true,
			},
			"last_updated": {
				Type:                types.StringType,
				MarkdownDescription: "Timestamp when this group was most recently updated.",
				Description:         "Timestamp when this group was most recently updated.",
				Computed:            true,
			},
		},
	}, nil
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *groupResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*tharsis.Client)
}

func (t *groupResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse) {

	// Retrieve values from group.
	var group GroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the group.
	created, err := t.client.Group.CreateGroup(ctx,
		&ttypes.CreateGroupInput{
			Name:        group.Name.ValueString(),
			Description: group.Description.ValueString(),
			ParentPath:  ptr.String(group.ParentPath.ValueString()),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating group",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	// Because the schema uses the Set type rather than the List type, make sure to set all fields.
	t.copyGroup(*created, &group)

	// Set the response state to the fully-populated plan.
	resp.Diagnostics.Append(resp.State.Set(ctx, group)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (t *groupResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse) {

	// Get the current state.
	var state GroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the group from Tharsis.
	found, err := t.client.Group.GetGroup(ctx, &ttypes.GetGroupInput{
		ID: ptr.String(state.ID.ValueString()),
	})
	if err != nil {

		// Handle the case that the group no longer exists if that fact is reported as an error.
		if t.isErrorGroupNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error reading group",
			err.Error(),
		)
		return
	}

	if found == nil {
		// Handle the case that the group no longer exists if that fact is reported by returning nil.
		resp.State.RemoveResource(ctx)
	}

	// Copy the from-Tharsis struct to the state.
	t.copyGroup(*found, &state)

	// Set the refreshed state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (t *groupResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse) {

	// Retrieve values from plan.
	var plan GroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the group via Tharsis.
	// The ID is used to find the record to update.
	// The description is modified.
	updated, err := t.client.Group.UpdateGroup(ctx,
		&ttypes.UpdateGroupInput{
			ID:          ptr.String(plan.ID.ValueString()),
			Description: plan.Description.ValueString(),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating group",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	t.copyGroup(*updated, &plan)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *groupResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse) {

	// Get the current state.
	var state GroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the group via Tharsis.
	err := t.client.Group.DeleteGroup(ctx,
		&ttypes.DeleteGroupInput{
			ID: ptr.String(state.ID.ValueString()),
		})
	if err != nil {

		// Handle the case that the group no longer exists.
		if t.isErrorGroupNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting group",
			err.Error(),
		)
		return
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *groupResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse) {

	// Retrieve import ID and save to id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// copyGroup copies the contents of a group.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *groupResource) copyGroup(src ttypes.Group, dest *GroupModel) error {

	dest.ID = types.StringValue(src.Metadata.ID)
	dest.Name = types.StringValue(src.Name)
	dest.Description = types.StringValue(src.Description)
	dest.ParentPath = types.StringValue(t.getParentPath(src.FullPath))
	dest.FullPath = types.StringValue(src.FullPath)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.LastUpdatedTimestamp.Format(time.RFC850))

	return nil
}

// getParentPath returns the parent path
func (t *groupResource) getParentPath(fullPath string) string {
	return fullPath[:strings.LastIndex(fullPath, "/")]
}

// isErrorGroupNotFound returns true iff the error message is that a group was not found.
// In theory, we should never see a message that some other ID was not found.
func (t *groupResource) isErrorGroupNotFound(e error) bool {
	lowerError := strings.ToLower(e.Error())
	return strings.Contains(lowerError, "group with id ") &&
		strings.Contains(lowerError, " not found")
}

// The End.
