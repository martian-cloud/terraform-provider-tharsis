package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
)

var _ client.LeveledLogger = (*tflogAdapter)(nil)

// tflogAdapter adapts terraform-plugin-log/tflog to the client.LeveledLogger interface.
type tflogAdapter struct {
	ctx context.Context
}

func (l *tflogAdapter) Error(msg string, keysAndValues ...any) {
	tflog.Error(l.ctx, msg, kvToMap(keysAndValues))
}

func (l *tflogAdapter) Warn(msg string, keysAndValues ...any) {
	tflog.Warn(l.ctx, msg, kvToMap(keysAndValues))
}

func (l *tflogAdapter) Info(msg string, keysAndValues ...any) {
	tflog.Info(l.ctx, msg, kvToMap(keysAndValues))
}

func (l *tflogAdapter) Debug(msg string, keysAndValues ...any) {
	tflog.Debug(l.ctx, msg, kvToMap(keysAndValues))
}

func kvToMap(keysAndValues []any) map[string]any {
	m := make(map[string]any, len(keysAndValues)/2)
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		if k, ok := keysAndValues[i].(string); ok {
			m[k] = keysAndValues[i+1]
		}
	}
	return m
}
