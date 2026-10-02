package executor

import (
	"context"

	"latere.ai/x/wallfacer/internal/harness"
)

// launchCursor execs cursor-agent and forwards its stream-json output directly.
// Permission is carried by the canonical harness.Request; ordinary launches
// default to Full while restricted callers can request ReadOnly/Edit.
func (b *HostBackend) launchCursor(ctx context.Context, spec ContainerSpec) (Handle, error) {
	return b.launchPlainHostAgent(ctx, spec, plainHostLaunch{
		id:            harness.Cursor,
		requirePrompt: true,
	})
}
