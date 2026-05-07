package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GroupModel is the model for a group.
type GroupModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	ParentPath  types.String `tfsdk:"parent_path"`
	ParentID    types.String `tfsdk:"parent_id"`
	FullPath    types.String `tfsdk:"full_path"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = (*groupResource)(nil)
	_ resource.ResourceWithConfigure   = (*groupResource)(nil)
	_ resource.ResourceWithImportState = (*groupResource)(nil)
)

// NewGroupResource is a helper function to simplify the provider implementation.
func NewGroupResource() resource.Resource {
	return &groupResource{}
}

type groupResource struct {
	client *client.GRPCClient
}

// Metadata returns the full name of the resource, including prefix, underscore, instance name.
func (t *groupResource) Metadata(_ context.Context,
	_ resource.MetadataRequest, resp *resource.MetadataResponse,
) {
	resp.TypeName = "tharsis_group"
}

func (t *groupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	description := "Defines and manages a group."

	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: description,
		Description:         description,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "String identifier of the group.",
				Description:         "String identifier of the group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the group.",
				Description:         "The name of the group.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the group.",
				Description:         "A description of the group.",
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Computed:            true,
				// Description can be updated in place, so no RequiresReplace plan modifier.
			},
			"parent_path": schema.StringAttribute{
				MarkdownDescription: "Full path of the parent namespace.",
				Description:         "Full path of the parent namespace.",
				Optional:            true, // A root group has no parent path.
				DeprecationMessage:  "Use parent_id instead. This field will be removed in a future version.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the parent group.",
				Description:         "The ID of the parent group.",
				Optional:            true, // A root group has no parent.
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"full_path": schema.StringAttribute{
				MarkdownDescription: "The path of the parent namespace plus the name of the group.",
				Description:         "The path of the parent namespace plus the name of the group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "Timestamp when this group was most recently updated.",
				Description:         "Timestamp when this group was most recently updated.",
				Computed:            true,
			},
		},
	}
}

// Configure lets the provider implement the ResourceWithConfigure interface.
func (t *groupResource) Configure(_ context.Context,
	req resource.ConfigureRequest, _ *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}
	t.client = req.ProviderData.(*client.GRPCClient)
}

func (t *groupResource) Create(ctx context.Context,
	req resource.CreateRequest, resp *resource.CreateResponse,
) {
	// Retrieve values from group.
	var group GroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the group.
	input := &pb.CreateGroupRequest{
		Name:        group.Name.ValueString(),
		Description: group.Description.ValueString(),
	}
	if v := group.ParentID.ValueString(); v != "" {
		input.ParentId = &v
	} else if v := group.ParentPath.ValueString(); v != "" {
		parentID := trn.TypeGroup.Build(v)
		input.ParentId = &parentID
	}

	created, err := t.client.GroupsClient.CreateGroup(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating group",
			err.Error(),
		)
		return
	}

	// Map the response body to the schema and update the plan with the computed attribute values.
	// Because the schema uses the Set type rather than the List type, make sure to set all fields.
	t.copyGroup(created, &group)

	// Set the response state to the fully-populated plan, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, group)...)
}

func (t *groupResource) Read(ctx context.Context,
	req resource.ReadRequest, resp *resource.ReadResponse,
) {
	// Get the current state.
	var state GroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get the group from Tharsis.
	found, err := t.client.GroupsClient.GetGroupByID(ctx, &pb.GetGroupByIDRequest{
		Id: state.ID.ValueString(),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading group",
			err.Error(),
		)
		return
	}

	// Copy the from-Tharsis struct to the state.
	t.copyGroup(found, &state)

	// Set the refreshed state, whether or not there is an error.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (t *groupResource) Update(ctx context.Context,
	req resource.UpdateRequest, resp *resource.UpdateResponse,
) {
	// Retrieve values from plan.
	var plan GroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the group via Tharsis.
	// The ID is used to find the record to update.
	// The description is modified.
	updated, err := t.client.GroupsClient.UpdateGroup(ctx,
		&pb.UpdateGroupRequest{
			Id:          plan.ID.ValueString(),
			Description: new(plan.Description.ValueString()),
		})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating group",
			err.Error(),
		)
		return
	}

	// Copy all fields returned by Tharsis back into the plan.
	t.copyGroup(updated, &plan)

	// Set the response state to the fully-populated plan, with or without error.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (t *groupResource) Delete(ctx context.Context,
	req resource.DeleteRequest, resp *resource.DeleteResponse,
) {
	// Get the current state.
	var state GroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the group via Tharsis.
	_, err := t.client.GroupsClient.DeleteGroup(ctx,
		&pb.DeleteGroupRequest{
			Id: state.ID.ValueString(),
		})
	if err != nil {
		// Handle the case that the group no longer exists.
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Error deleting group",
			err.Error(),
		)
	}
}

// ImportState helps the provider implement the ResourceWithImportState interface.
func (t *groupResource) ImportState(ctx context.Context,
	req resource.ImportStateRequest, resp *resource.ImportStateResponse,
) {
	// Get the group by full path from Tharsis.
	found, err := t.client.GroupsClient.GetGroupByID(ctx, &pb.GetGroupByIDRequest{
		Id: trn.TypeGroup.Normalize(req.ID),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			resp.Diagnostics.AddError(
				"Import group not found: "+req.ID,
				"",
			)
			return
		}
		resp.Diagnostics.AddError(
			"Import group not found: "+req.ID,
			err.Error(),
		)
		return
	}

	// Import by full path.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), found.Metadata.Id)...)
}

// copyGroup copies the contents of a group.
// It is intended to copy from a struct returned by Tharsis to a Terraform plan or state.
func (t *groupResource) copyGroup(src *pb.Group, dest *GroupModel) {
	dest.ID = types.StringValue(src.Metadata.Id)
	dest.Name = types.StringValue(src.Name)
	dest.Description = types.StringValue(src.Description)
	parsed := trn.MustParseAny(src.Metadata.Trn)
	if parsed.HasParent() {
		dest.ParentPath = types.StringValue(parsed.ParentPath())
	}
	if src.ParentId != "" {
		dest.ParentID = types.StringValue(src.ParentId)
	} else {
		dest.ParentID = types.StringNull()
	}
	dest.FullPath = types.StringValue(src.FullPath)

	// Must use time value from SDK/API.  Using time.Now() is not reliable.
	dest.LastUpdated = types.StringValue(src.Metadata.UpdatedAt.AsTime().Format(time.RFC850))
}
