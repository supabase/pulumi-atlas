package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/supabase/atlasctl/pkg/plan"
)

// Measurement is the Pulumi resource for a single RIPE Atlas measurement.
// Each instance maps to one (name, round) pair and one RIPE Atlas measurement ID.
type Measurement struct{}

// MeasurementArgs are the inputs declared by the Pulumi program.
type MeasurementArgs struct {
	Name     string `pulumi:"name"`
	Round    string `pulumi:"round"`
	Target   string `pulumi:"target"`
	Type     string `pulumi:"type"`          // dns, ping, tls, traceroute
	AF       int    `pulumi:"af,optional"`   // 4 or 6; default 4
	Interval int    `pulumi:"intervalSeconds"`
	ProbeIDs []int  `pulumi:"probeIds"`
}

// MeasurementState is the full resource state stored in Pulumi state.
type MeasurementState struct {
	MeasurementArgs
	MsmID string `pulumi:"msmId"`
}

func (m *Measurement) Annotate(a infer.Annotator) {
	a.SetToken("index", "Measurement")
}

var _ infer.CustomResource[MeasurementArgs, MeasurementState] = (*Measurement)(nil)
var _ infer.CustomUpdate[MeasurementArgs, MeasurementState] = (*Measurement)(nil)
var _ infer.CustomRead[MeasurementArgs, MeasurementState] = (*Measurement)(nil)
var _ infer.CustomDiff[MeasurementArgs, MeasurementState] = (*Measurement)(nil)
var _ infer.CustomDelete[MeasurementState] = (*Measurement)(nil)

func (m *Measurement) Create(
	ctx context.Context,
	req infer.CreateRequest[MeasurementArgs],
) (infer.CreateResponse[MeasurementState], error) {
	if req.DryRun {
		return infer.CreateResponse[MeasurementState]{
			Output: MeasurementState{MeasurementArgs: req.Inputs},
		}, nil
	}

	cfg := infer.GetConfig[*AtlasConfig](ctx)
	af := req.Inputs.AF
	if af == 0 {
		af = 4
	}

	spec := plan.MsmSpec{
		Key:      plan.MsmKey{Name: req.Inputs.Name, Round: req.Inputs.Round},
		Target:   req.Inputs.Target,
		Type:     plan.MsmType(req.Inputs.Type),
		AF:       af,
		Interval: req.Inputs.Interval,
		ProbeIDs: uint32Slice(req.Inputs.ProbeIDs),
	}

	msmID, err := cfg.client.CreateMeasurement(ctx, spec)
	if err != nil {
		return infer.CreateResponse[MeasurementState]{}, fmt.Errorf(
			"create measurement %s/%s: %w", req.Inputs.Name, req.Inputs.Round, err)
	}

	idStr := strconv.FormatUint(msmID, 10)
	return infer.CreateResponse[MeasurementState]{
		ID:     idStr,
		Output: MeasurementState{MeasurementArgs: req.Inputs, MsmID: idStr},
	}, nil
}

func (m *Measurement) Read(
	ctx context.Context,
	req infer.ReadRequest[MeasurementArgs, MeasurementState],
) (infer.ReadResponse[MeasurementArgs, MeasurementState], error) {
	cfg := infer.GetConfig[*AtlasConfig](ctx)

	msmID, err := parseMsmID(req.State.MsmID)
	if err != nil {
		return infer.ReadResponse[MeasurementArgs, MeasurementState]{}, err
	}
	info, err := cfg.client.GetMeasurement(ctx, msmID)
	if err != nil {
		if errors.Is(err, plan.ErrMsmNotFound) {
			// Return empty state so Pulumi marks the resource for recreation.
			return infer.ReadResponse[MeasurementArgs, MeasurementState]{
				ID: req.ID,
			}, nil
		}
		return infer.ReadResponse[MeasurementArgs, MeasurementState]{}, fmt.Errorf(
			"read measurement %s: %w", req.State.MsmID, err)
	}

	state := req.State
	state.Target = info.Target
	state.Type = info.Type
	state.Interval = info.Interval
	state.ProbeIDs = intSlice(info.ProbeIDs)

	return infer.ReadResponse[MeasurementArgs, MeasurementState]{
		ID:     req.ID,
		Inputs: req.Inputs,
		State:  state,
	}, nil
}

func (m *Measurement) Update(
	ctx context.Context,
	req infer.UpdateRequest[MeasurementArgs, MeasurementState],
) (infer.UpdateResponse[MeasurementState], error) {
	if req.DryRun {
		return infer.UpdateResponse[MeasurementState]{
			Output: MeasurementState{MeasurementArgs: req.Inputs, MsmID: req.State.MsmID},
		}, nil
	}

	cfg := infer.GetConfig[*AtlasConfig](ctx)

	msmID, err := parseMsmID(req.State.MsmID)
	if err != nil {
		return infer.UpdateResponse[MeasurementState]{}, err
	}

	oldSet := uint32Set(req.State.ProbeIDs)
	newSet := uint32Set(req.Inputs.ProbeIDs)

	var toAdd, toRemove []uint32
	for pid := range newSet {
		if !oldSet[pid] {
			toAdd = append(toAdd, pid)
		}
	}
	for pid := range oldSet {
		if !newSet[pid] {
			toRemove = append(toRemove, pid)
		}
	}

	if len(toAdd) > 0 {
		if err := cfg.client.AddParticipants(ctx, msmID, toAdd); err != nil {
			return infer.UpdateResponse[MeasurementState]{}, fmt.Errorf(
				"add participants to %s: %w", req.State.MsmID, err)
		}
	}
	if len(toRemove) > 0 {
		if err := cfg.client.RemoveParticipants(ctx, msmID, toRemove); err != nil {
			return infer.UpdateResponse[MeasurementState]{}, fmt.Errorf(
				"remove participants from %s: %w", req.State.MsmID, err)
		}
	}

	return infer.UpdateResponse[MeasurementState]{
		Output: MeasurementState{MeasurementArgs: req.Inputs, MsmID: req.State.MsmID},
	}, nil
}

func (m *Measurement) Delete(
	ctx context.Context,
	req infer.DeleteRequest[MeasurementState],
) (infer.DeleteResponse, error) {
	cfg := infer.GetConfig[*AtlasConfig](ctx)
	msmID, err := parseMsmID(req.State.MsmID)
	if err != nil {
		return infer.DeleteResponse{}, err
	}
	if err := cfg.client.StopMeasurement(ctx, msmID); err != nil {
		return infer.DeleteResponse{}, fmt.Errorf("stop measurement %s: %w", req.State.MsmID, err)
	}
	return infer.DeleteResponse{}, nil
}

func (m *Measurement) Diff(
	ctx context.Context,
	req infer.DiffRequest[MeasurementArgs, MeasurementState],
) (infer.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}

	replaceFields := []struct {
		key  string
		same bool
	}{
		{"name", req.State.Name == req.Inputs.Name},
		{"round", req.State.Round == req.Inputs.Round},
		{"target", req.State.Target == req.Inputs.Target},
		{"type", req.State.Type == req.Inputs.Type},
		{"af", req.State.AF == req.Inputs.AF},
		{"intervalSeconds", req.State.Interval == req.Inputs.Interval},
	}
	for _, f := range replaceFields {
		if !f.same {
			diff[f.key] = p.PropertyDiff{Kind: p.UpdateReplace}
		}
	}

	if !probeIDsEqual(req.State.ProbeIDs, req.Inputs.ProbeIDs) {
		diff["probeIds"] = p.PropertyDiff{Kind: p.Update}
	}

	return infer.DiffResponse{
		DeleteBeforeReplace: false,
		HasChanges:          len(diff) > 0,
		DetailedDiff:        diff,
	}, nil
}

// parseMsmID converts the string-typed MsmID stored in state back to a uint64
// for API calls. An invalid value indicates corrupted state.
func parseMsmID(s string) (uint64, error) {
	id, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid msmId %q in state: %w", s, err)
	}
	return id, nil
}

// helpers

func uint32Slice(in []int) []uint32 {
	out := make([]uint32, len(in))
	for i, v := range in {
		out[i] = uint32(v)
	}
	return out
}

func intSlice(in []uint32) []int {
	out := make([]int, len(in))
	for i, v := range in {
		out[i] = int(v)
	}
	return out
}

func uint32Set(in []int) map[uint32]bool {
	s := make(map[uint32]bool, len(in))
	for _, v := range in {
		s[uint32(v)] = true
	}
	return s
}

func probeIDsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	sa, sb := uint32Set(a), uint32Set(b)
	for id := range sa {
		if !sb[id] {
			return false
		}
	}
	return true
}
