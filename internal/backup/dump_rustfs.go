package backup

import (
	"context"
	"fmt"
)

// rustfsDataDir is the on-disk object store root in the Helm chart.
const rustfsDataDir = "/data"

func rustfsDumpK8s(ctx context.Context, ns, pod string) ([]byte, error) {
	inner := fmt.Sprintf(`tar -cf - -C %s .`, rustfsDataDir)
	out, err := kubectlStdout(ctx, "exec", "-n", ns, pod, "-c", "rustfs", "--", "sh", "-ec", inner)
	if err != nil {
		return nil, fmt.Errorf("rustfs volume tar: %w", err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("rustfs volume tar produced empty output")
	}
	return out, nil
}
