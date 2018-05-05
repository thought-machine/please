// +build !bootstrap

// Contains routines to marshal between internal structures and
// proto-generated equivalents.
// The duplication is unfortunate but it's preferable to needing
// to run proto / gRPC compilers at bootstrap time.

package follow

import (
	"errors"
	"time"

	"core"
	pb "follow/proto/build_event"
)

// toProto converts an internal test result into a proto type.
func toProto(r *core.BuildResult) *pb.BuildEventResponse {
	t := &r.Tests
	return &pb.BuildEventResponse{
		ThreadId:    int32(r.ThreadID),
		Timestamp:   r.Time.UnixNano(),
		BuildLabel:  toProtoBuildLabel(r.Label),
		Status:      pb.BuildResultStatus(r.Status),
		Error:       toProtoError(r.Err),
		Description: r.Description,
		TestResults: &pb.TestResults{
			NumTests:         int32(t.NumTests),
			Passed:           int32(t.Passed),
			Failed:           int32(t.Failed),
			ExpectedFailures: int32(t.ExpectedFailures),
			Skipped:          int32(t.Skipped),
			Flakes:           int32(t.Flakes),
			Results:          toProtoTestResults(t.Results),
			Output:           t.Output,
			Duration:         int64(t.Duration),
			Cached:           t.Cached,
			TimedOut:         t.TimedOut,
		},
	}
}

// toProtos converts a slice of internal test results to a slice of protos.
func toProtos(results []*core.BuildResult, active, done int) []*pb.BuildEventResponse {
	ret := make([]*pb.BuildEventResponse, 0, len(results))
	for _, r := range results {
		if r != nil {
			p := toProto(r)
			p.NumActive = int64(active)
			p.NumDone = int64(done)
			ret = append(ret, p)
		}
	}
	return ret
}

// toProtoTestResults converts a slice of test failures to the proto equivalent.
func toProtoTestResults(results []core.TestResult) []*pb.TestResult {
	ret := make([]*pb.TestResult, len(results))
	for i, r := range results {
		ret[i] = &pb.TestResult{
			Name:      r.Name,
			Type:      r.Type,
			Traceback: r.Traceback,
			Stdout:    r.Stdout,
			Stderr:    r.Stderr,
			Duration:  int64(r.Duration),
			Success:   r.Success,
			Skipped:   r.Skipped,
		}
	}
	return ret
}

// toProtoBuildLabel converts the internal build label type to a proto equivalent.
func toProtoBuildLabel(label core.BuildLabel) *pb.BuildLabel {
	return &pb.BuildLabel{PackageName: label.PackageName, Name: label.Name}
}

// toProtoError converts an error to a string if the error is non-nil.
func toProtoError(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

// fromProto converts from a proto type into the internal equivalent.
func fromProto(r *pb.BuildEventResponse) *core.BuildResult {
	t := r.TestResults
	return &core.BuildResult{
		ThreadID:    int(r.ThreadId),
		Time:        time.Unix(0, r.Timestamp),
		Label:       fromProtoBuildLabel(r.BuildLabel),
		Status:      core.BuildResultStatus(r.Status),
		Err:         fromProtoError(r.Error),
		Description: r.Description,
		Tests: core.TestResults{
			NumTests:         int(t.NumTests),
			Passed:           int(t.Passed),
			Failed:           int(t.Failed),
			ExpectedFailures: int(t.ExpectedFailures),
			Skipped:          int(t.Skipped),
			Flakes:           int(t.Flakes),
			Results:          fromProtoTestResults(t.Results),
			Output:           t.Output,
			Duration:         time.Duration(t.Duration),
			Cached:           t.Cached,
			TimedOut:         t.TimedOut,
		},
	}
}

// fromProtoTestResults converts a slice of proto test failures to the internal equivalent.
func fromProtoTestResults(results []*pb.TestResult) []core.TestResult {
	ret := make([]core.TestResult, len(results))
	for i, r := range results {
		ret[i] = core.TestResult{
			Name:      r.Name,
			Type:      r.Type,
			Traceback: r.Traceback,
			Stdout:    r.Stdout,
			Stderr:    r.Stderr,
			Duration:  time.Duration(r.Duration),
			Success:   r.Success,
			Skipped:   r.Skipped,
		}
	}
	return ret
}

// fromProtoBuildLabel converts a proto build label to the internal version.
func fromProtoBuildLabel(label *pb.BuildLabel) core.BuildLabel {
	return core.BuildLabel{PackageName: label.PackageName, Name: label.Name}
}

// fromProtoBuildLabels converts a series of proto build labels to a slice of internal ones.
func fromProtoBuildLabels(labels []*pb.BuildLabel) []core.BuildLabel {
	ret := make([]core.BuildLabel, len(labels))
	for i, l := range labels {
		ret[i] = fromProtoBuildLabel(l)
	}
	return ret
}

// fromProtoError converts a proto string into an error if it's non-empty.
func fromProtoError(s string) error {
	if s != "" {
		return errors.New(s)
	}
	return nil
}

// resourceToProto converts the internal resource stats to a proto message.
func resourceToProto(stats *core.SystemStats) *pb.ResourceUsageResponse {
	return &pb.ResourceUsageResponse{
		NumCpus:  int32(stats.CPU.Count),
		CpuUse:   stats.CPU.Used,
		IoWait:   stats.CPU.IOWait,
		MemTotal: stats.Memory.Total,
		MemUsed:  stats.Memory.Used,
	}
}

// resourceFromProto converts the proto message back to the internal type.
func resourceFromProto(r *pb.ResourceUsageResponse) *core.SystemStats {
	s := &core.SystemStats{}
	s.CPU.Count = int(r.NumCpus)
	s.CPU.Used = r.CpuUse
	s.CPU.IOWait = r.IoWait
	s.Memory.Total = r.MemTotal
	s.Memory.Used = r.MemUsed
	s.Memory.UsedPercent = 100.0 * float64(r.MemUsed) / float64(r.MemTotal)
	return s
}
