package backup

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/hsoni92/obelisk/internal/backup/lineage"
	"github.com/hsoni92/obelisk/pkg/obelisk"
)

// CloneBackupOutcome is a new duplicate backup catalog entry (separate forest).
type CloneBackupOutcome struct {
	ServiceType          string
	InstanceID           string
	LineageID            string
	NodeCount            int
	Scope                string
	SourceKey            string
	UploadedDestinations int
}

// CloneBackupEntry duplicates a backup forest without changing the source lineage tree.
// scope is lineage.CloneScopeSnapshot (one snapshot) or lineage.CloneScopeLineage (full tree).
func CloneBackupEntry(
	ctx context.Context,
	configPath string,
	spec obelisk.Spec,
	sourceType, sourceInstanceID, scope, snapshotBackupID, label string,
) (CloneBackupOutcome, error) {
	var out CloneBackupOutcome
	sourceType = strings.ToLower(strings.TrimSpace(sourceType))
	sourceInstanceID = strings.TrimSpace(sourceInstanceID)
	if sourceType == "" || sourceInstanceID == "" {
		return out, fmt.Errorf("source_type and source_instance_id are required")
	}

	idx, err := lineage.Load(configPath)
	if err != nil {
		return out, err
	}

	plan, err := lineage.PlanCloneForest(
		idx,
		sourceType,
		sourceInstanceID,
		scope,
		snapshotBackupID,
		label,
		clonedObjectKey,
	)
	if err != nil {
		return out, err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	machineID := spec.BackupBucketName()
	var pairs []ObjectPair
	for _, job := range plan.Jobs {
		body, err := FetchObjectFromDestinations(ctx, configPath, spec, job.SourceObjectKey)
		if err != nil {
			return out, fmt.Errorf("fetch %s: %w", job.SourceObjectKey, err)
		}
		size := int64(len(body))
		if job.Node.Size > 0 {
			size = job.Node.Size
		}
		node := job.Node
		node.Size = size
		manifest := lineage.NewManifest(node, plan.TargetForest, machineID)
		manifestBody, err := manifest.JSON()
		if err != nil {
			return out, err
		}
		pairs = append(pairs, ObjectPair{
			Key:          node.ObjectKey,
			Body:         body,
			ManifestKey:  node.ManifestKey,
			ManifestBody: manifestBody,
		})
		plan.TargetForest.Nodes[node.ID] = node
	}

	uploaded, err := ReplicateObjects(ctx, configPath, spec, pairs)
	out.UploadedDestinations = uploaded
	if err != nil {
		return out, err
	}
	if err := lineage.PutForest(configPath, plan.TargetForest); err != nil {
		return out, err
	}

	out.ServiceType = plan.TargetForest.ServiceType
	out.InstanceID = plan.TargetForest.InstanceID
	out.LineageID = plan.TargetForest.LineageID
	out.NodeCount = len(plan.TargetForest.Nodes)
	out.Scope = scope
	out.SourceKey = plan.SourceKey
	return out, nil
}

func clonedObjectKey(serviceType, cloneInstanceID, sourceObjectKey string, seq int) (string, error) {
	base := path.Base(strings.TrimSpace(sourceObjectKey))
	if base == "" || base == "." {
		return "", fmt.Errorf("invalid source object key %q", sourceObjectKey)
	}
	ext := ""
	if i := strings.LastIndex(base, "."); i >= 0 {
		ext = base[i:]
	} else {
		ext = dataExt(serviceType)
	}
	ts := time.Now().UTC().Format("20060102T150405Z")
	name := ts + ext
	if seq > 0 {
		name = fmt.Sprintf("%s-%04d%s", ts, seq, ext)
	}
	return obelisk.BackupDataObjectKey(serviceType, cloneInstanceID, name), nil
}
