package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	runpkg "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/run"
)

// runObserver is the provider's runpkg.Observer. Terraform resources don't render to a terminal,
// so run progress — stage transitions, policy-check verdicts, what the run is waiting on — goes to
// the provider's debug log (visible with TF_LOG=DEBUG). Full job-log output can contain plan/apply
// transcript (and anything a module prints), so it's emitted at TRACE, not DEBUG; DEBUG gets only a
// stage/length marker. The run-stage state machine itself lives in the shared follower.
type runObserver struct {
	ctx context.Context
}

var _ runpkg.Observer = (*runObserver)(nil)

// Logs records a chunk of a job's logs.
func (o *runObserver) Logs(stage runpkg.StageRef, chunk string) error {
	if chunk != "" {
		tflog.Debug(o.ctx, "tharsis run job logs", map[string]any{"stage": stage.Name, "bytes": len(chunk)})
		tflog.Trace(o.ctx, "tharsis run job logs", map[string]any{"stage": stage.Name, "logs": chunk})
	}
	return nil
}

// Event records an event.
func (o *runObserver) Event(e runpkg.Event) {
	switch e := e.(type) {
	case runpkg.StageStarted:
		tflog.Debug(o.ctx, "tharsis run stage started", map[string]any{"stage": e.Stage.Name, "retry": e.Retry})
	case runpkg.Waiting:
		tflog.Debug(o.ctx, "tharsis run waiting", map[string]any{"stage": e.Stage.Name, "detail": e.Detail})
	case runpkg.PolicyCheckFinished:
		tflog.Debug(o.ctx, "tharsis policy check finished", map[string]any{
			"stage":  e.Stage.Name,
			"type":   runpkg.PolicyCheckTypeName(e.Results.GetPolicyCheck().GetCheckType()),
			"status": e.Results.GetPolicyCheck().GetStatus().String(),
		})
	case runpkg.GatesPending:
		tflog.Debug(o.ctx, "tharsis run blocked on policy gate; approve or discard it in the Tharsis UI",
			map[string]any{"stage": e.Stage.Name})
	case runpkg.NoChanges:
		tflog.Debug(o.ctx, "tharsis run has no changes to apply", nil)
	case runpkg.AutoApproving:
		tflog.Debug(o.ctx, "tharsis run auto-approving", nil)
	case runpkg.DecisionDeferred:
		tflog.Debug(o.ctx, "tharsis run waiting on a decision made elsewhere",
			map[string]any{"stage": e.Stage.Name, "reason": e.Reason})
	case runpkg.Interrupted:
		tflog.Debug(o.ctx, "tharsis run following interrupted", nil)
	case runpkg.InterruptFailed:
		tflog.Debug(o.ctx, "tharsis run interrupt action failed",
			map[string]any{"action": e.Action, "error": e.Err.Error()})
	}
}
