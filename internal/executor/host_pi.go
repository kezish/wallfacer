package executor

import (
	"context"

	"latere.ai/x/wallfacer/internal/harness"
)

// launchPi execs the pi CLI. Pi emits JSON natively under --mode json, so its
// stdout can be forwarded directly. Permission arrives through the canonical
// harness.Request: ordinary launches default to Full in requestFromClaudeSpec,
// while restricted callers can explicitly request ReadOnly/Edit.
func (b *HostBackend) launchPi(ctx context.Context, spec ContainerSpec) (Handle, error) {
	return b.launchPlainHostAgent(ctx, spec, plainHostLaunch{
		id:            harness.Pi,
		requirePrompt: true,
	})
}
