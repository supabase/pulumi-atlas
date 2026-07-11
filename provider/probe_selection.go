package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/supabase/atlasctl/pkg/config"
	"github.com/supabase/atlasctl/pkg/selection"
	"github.com/supabase/atlasctl/pkg/snapshot"
)

// ProbeSelection runs the atlasctl probe selection algorithm and stores the
// resulting probe ID lists in Pulumi state. Measurement resources reference
// these outputs so that probe assignments flow automatically from
// atlasctl refresh → pulumi up.
//
// Round definitions, scoring weights, exclude tags, and geo diversity are all
// read from the atlasctl YAML config file — atlasctl.yaml is the single source
// of truth for selection parameters.
type ProbeSelection struct{}

func (ps *ProbeSelection) Annotate(a infer.Annotator) {
	a.SetToken("index", "ProbeSelection")
}

// ProbeSelectionArgs are the inputs to a ProbeSelection resource.
type ProbeSelectionArgs struct {
	// ConfigPath is the path to the atlasctl YAML config file (e.g. atlasctl.yaml).
	// Round definitions, scoring, excludeTags, and geoDiversity are all read from here.
	ConfigPath string `pulumi:"configPath"`
	// SnapshotPath overrides the snapshot path specified in the config file.
	// Useful when the Pulumi stack runs from a different working directory.
	// If empty, the path from the config file is used.
	SnapshotPath string `pulumi:"snapshotPath,optional"`
}

// ProbeSelectionState is the full state stored in Pulumi after selection.
type ProbeSelectionState struct {
	ProbeSelectionArgs
	// ConfigHash is the SHA-256 of the config file at last selection.
	// A change here (scoring weights, round definitions, etc.) triggers re-selection.
	ConfigHash string `pulumi:"configHash"`
	// SnapshotHash is the SHA-256 of the snapshot file at last selection.
	// A change here (from atlasctl refresh) triggers re-selection.
	SnapshotHash string `pulumi:"snapshotHash"`
	// RoundProbeIds maps round name to the selected probe IDs for that round.
	RoundProbeIds map[string][]int `pulumi:"roundProbeIds"`
	// SelectedAt is the RFC3339 timestamp of the last selection run.
	SelectedAt string `pulumi:"selectedAt"`
}

var _ infer.CustomResource[ProbeSelectionArgs, ProbeSelectionState] = (*ProbeSelection)(nil)
var _ infer.CustomRead[ProbeSelectionArgs, ProbeSelectionState] = (*ProbeSelection)(nil)
var _ infer.CustomDiff[ProbeSelectionArgs, ProbeSelectionState] = (*ProbeSelection)(nil)
var _ infer.CustomUpdate[ProbeSelectionArgs, ProbeSelectionState] = (*ProbeSelection)(nil)

func (ps *ProbeSelection) Create(
	ctx context.Context,
	req infer.CreateRequest[ProbeSelectionArgs],
) (infer.CreateResponse[ProbeSelectionState], error) {
	if req.DryRun {
		return infer.CreateResponse[ProbeSelectionState]{
			ID:     "probe-selection",
			Output: ProbeSelectionState{ProbeSelectionArgs: req.Inputs},
		}, nil
	}

	state, err := runSelection(ctx, req.Inputs)
	if err != nil {
		return infer.CreateResponse[ProbeSelectionState]{}, err
	}
	return infer.CreateResponse[ProbeSelectionState]{ID: "probe-selection", Output: state}, nil
}

func (ps *ProbeSelection) Read(
	ctx context.Context,
	req infer.ReadRequest[ProbeSelectionArgs, ProbeSelectionState],
) (infer.ReadResponse[ProbeSelectionArgs, ProbeSelectionState], error) {
	// Re-hash both files to detect out-of-band changes (new snapshot, edited config).
	// On any read error, return current state unchanged and let Diff sort it out.
	configHash, err := hashFile(req.State.ConfigPath)
	if err != nil {
		return infer.ReadResponse[ProbeSelectionArgs, ProbeSelectionState]{
			ID: req.ID, Inputs: req.Inputs, State: req.State,
		}, nil
	}
	snapPath := effectiveSnapshotPath(req.State.ProbeSelectionArgs)
	snapHash, err := hashFile(snapPath)
	if err != nil {
		return infer.ReadResponse[ProbeSelectionArgs, ProbeSelectionState]{
			ID: req.ID, Inputs: req.Inputs, State: req.State,
		}, nil
	}

	state := req.State
	state.ConfigHash = configHash
	state.SnapshotHash = snapHash
	return infer.ReadResponse[ProbeSelectionArgs, ProbeSelectionState]{
		ID: req.ID, Inputs: req.Inputs, State: state,
	}, nil
}

func (ps *ProbeSelection) Diff(
	ctx context.Context,
	req infer.DiffRequest[ProbeSelectionArgs, ProbeSelectionState],
) (infer.DiffResponse, error) {
	configHash, err := hashFile(req.Inputs.ConfigPath)
	if err != nil {
		return infer.DiffResponse{}, fmt.Errorf("hashing config %s: %w", req.Inputs.ConfigPath, err)
	}

	snapPath := effectiveSnapshotPath(req.Inputs)
	snapHash, err := hashFile(snapPath)
	if err != nil {
		return infer.DiffResponse{}, fmt.Errorf("hashing snapshot %s: %w", snapPath, err)
	}

	hasChanges := configHash != req.State.ConfigHash ||
		snapHash != req.State.SnapshotHash ||
		req.Inputs.ConfigPath != req.State.ConfigPath ||
		req.Inputs.SnapshotPath != req.State.SnapshotPath

	return infer.DiffResponse{HasChanges: hasChanges}, nil
}

func (ps *ProbeSelection) Update(
	ctx context.Context,
	req infer.UpdateRequest[ProbeSelectionArgs, ProbeSelectionState],
) (infer.UpdateResponse[ProbeSelectionState], error) {
	if req.DryRun {
		return infer.UpdateResponse[ProbeSelectionState]{Output: req.State}, nil
	}

	state, err := runSelection(ctx, req.Inputs)
	if err != nil {
		return infer.UpdateResponse[ProbeSelectionState]{}, err
	}
	return infer.UpdateResponse[ProbeSelectionState]{Output: state}, nil
}

// runSelection loads the config and snapshot, runs the selection algorithm,
// and returns a fully populated ProbeSelectionState.
func runSelection(ctx context.Context, args ProbeSelectionArgs) (ProbeSelectionState, error) {
	cfg, err := config.Load(args.ConfigPath)
	if err != nil {
		return ProbeSelectionState{}, fmt.Errorf("loading config %s: %w", args.ConfigPath, err)
	}

	snapPath := effectiveSnapshotPath(args)
	if snapPath == "" {
		return ProbeSelectionState{}, fmt.Errorf("snapshot path not set in config or args")
	}

	configHash, err := hashFile(args.ConfigPath)
	if err != nil {
		return ProbeSelectionState{}, fmt.Errorf("hashing config: %w", err)
	}
	snapHash, err := hashFile(snapPath)
	if err != nil {
		return ProbeSelectionState{}, fmt.Errorf("hashing snapshot %s: %w", snapPath, err)
	}

	snap, err := snapshot.Load(snapPath)
	if err != nil {
		return ProbeSelectionState{}, fmt.Errorf("loading snapshot %s: %w", snapPath, err)
	}

	rounds, err := selection.Select(ctx, snap, *cfg)
	if err != nil {
		return ProbeSelectionState{}, fmt.Errorf("probe selection: %w", err)
	}

	roundProbeIds := make(map[string][]int, len(rounds))
	for _, r := range rounds {
		ids := make([]int, len(r.Probes))
		for i, p := range r.Probes {
			ids[i] = int(p.ID)
		}
		roundProbeIds[r.Round.Name] = ids
	}

	return ProbeSelectionState{
		ProbeSelectionArgs: args,
		ConfigHash:         configHash,
		SnapshotHash:       snapHash,
		RoundProbeIds:      roundProbeIds,
		SelectedAt:         time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// effectiveSnapshotPath returns the snapshot path to use: the explicit override
// in args if set, otherwise the path from the loaded config file.
func effectiveSnapshotPath(args ProbeSelectionArgs) string {
	if args.SnapshotPath != "" {
		return args.SnapshotPath
	}
	// Load config just to read the snapshot field. Errors here surface in runSelection.
	cfg, err := config.Load(args.ConfigPath)
	if err != nil || cfg.Snapshot == "" {
		return ""
	}
	return cfg.Snapshot
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
