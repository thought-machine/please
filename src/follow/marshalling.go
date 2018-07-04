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
		TestResults: &pb.TestSuite{
			Package:    t.Package,
			Name:       t.Name,
			TestCases:  toProtoTestCases(t.TestCases),
			Duration:   int64(t.Duration),
			Cached:     t.Cached,
			TimedOut:   t.TimedOut,
			Properties: t.Properties,
			Timestamp:  t.Timestamp,
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

// toProtoTestCases converts a slice of test failures to the proto equivalent.
func toProtoTestCases(results []core.TestCase) []*pb.TestCase {
	ret := make([]*pb.TestCase, len(results))
	for i, r := range results {
		ret[i] = &pb.TestCase{
			ClassName:      r.ClassName,
			Name:           r.Name,
			TestExecutions: toProtoTestExecutions(r.Executions),
		}
	}
	return ret
}

func toProtoTestExecutions(executions []core.TestExecution) []*pb.TestExecution {
	ret := make([]*pb.TestExecution, len(executions))
	for i, r := range executions {
		ret[i] = &pb.TestExecution{
			Failure: toTestFailure(r.Failure),
			Error:   toTestFailure(r.Error),
			Skip:    toTestSkip(r.Skip),
			Stdout:  r.Stdout,
			Stderr:  r.Stderr,
		}
	}
	return ret
}

func toTestFailure(f *core.TestResultFailure) *pb.TestFailure {
	if f == nil {
		return nil
	}
	return &pb.TestFailure{
		Type:      f.Type,
		Message:   f.Message,
		Traceback: f.Traceback,
	}
}

func toTestSkip(s *core.TestResultSkip) *pb.TestSkip {
	if s == nil {
		return nil
	}
	return &pb.TestSkip{
		Message: s.Message,
	}
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
		Tests: core.TestSuite{
			Package:    t.Package,
			Duration:   time.Duration(t.Duration),
			Properties: t.Properties,
			Timestamp:  t.Timestamp,
			Name:       t.Name,
			TestCases:  fromProtoTestCases(t.TestCases),
			Cached:     t.Cached,
			TimedOut:   t.TimedOut,
		},
	}
}

// fromProtoTestResults converts a slice of proto test failures to the internal equivalent.
func fromProtoTestCases(results []*pb.TestCase) []core.TestCase {
	ret := make([]core.TestCase, len(results))
	for i, r := range results {
		ret[i] = core.TestCase{
			ClassName:  r.ClassName,
			Name:       r.Name,
			Executions: fromProtoTestExecutions(r.TestExecutions),
		}
	}
	return ret
}

func fromProtoTestExecutions(executions []*pb.TestExecution) []core.TestExecution {
	ret := make([]core.TestExecution, len(executions))
	for i, r := range executions {
		duration := time.Duration(r.Duration)
		ret[i] = core.TestExecution{
			Failure:  fromProtoTestFailure(r.Failure),
			Error:    fromProtoTestFailure(r.Error),
			Skip:     fromProtoTestSkip(r.Skip),
			Stdout:   r.Stdout,
			Stderr:   r.Stderr,
			Duration: &duration,
		}
	}
	return ret
}

func fromProtoTestFailure(f *pb.TestFailure) *core.TestResultFailure {
	if f == nil {
		return nil
	}
	return &core.TestResultFailure{
		Type:      f.Type,
		Message:   f.Message,
		Traceback: f.Traceback,
	}
}

func fromProtoTestSkip(s *pb.TestSkip) *core.TestResultSkip {
	if s == nil {
		return nil
	}
	return &core.TestResultSkip{
		Message: s.Message,
	}
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
		NumCpus:            int32(stats.CPU.Count),
		CpuUse:             stats.CPU.Used,
		IoWait:             stats.CPU.IOWait,
		MemTotal:           stats.Memory.Total,
		MemUsed:            stats.Memory.Used,
		NumWorkerProcesses: int32(stats.NumWorkerProcesses),
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
	s.NumWorkerProcesses = int(r.NumWorkerProcesses)
	return s
}
