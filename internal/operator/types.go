package operator

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	GroupVersion  = schema.GroupVersion{Group: "k3sc.abix.dev", Version: "v1"}
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&AgentJob{},
		&AgentJobList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}

// AgentJob represents one execution of work on a GitHub issue.
// Multiple AgentJobs can exist for the same issue (execution history).
type AgentJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AgentJobSpec   `json:"spec"`
	Status            AgentJobStatus `json:"status,omitempty"`
}

type AgentJobSpec struct {
	Repo        string `json:"repo"`        // e.g. "abix-/endless"
	RepoName    string `json:"repoName"`    // e.g. "endless"
	IssueNumber int    `json:"issueNumber"`
	RepoURL     string `json:"repoURL"`     // clone URL
	Slot        int    `json:"slot"`         // assigned by scanner, immutable
	Agent       string `json:"agent"`        // assigned by scanner, immutable (e.g. "claude-a")
	Family      string `json:"family"`       // "claude" or "codex"
	OriginState string `json:"originState"`  // "ready" or "needs-review" -- determines next state on completion
}

type TaskPhase string

const (
	TaskPhasePending   TaskPhase = "Pending"   // created, waiting for slot
	TaskPhaseAssigned  TaskPhase = "Assigned"  // slot + agent assigned, claiming on github
	TaskPhaseRunning   TaskPhase = "Running"   // job created, pod active
	TaskPhaseSucceeded TaskPhase = "Succeeded" // pod completed successfully
	TaskPhaseFailed    TaskPhase = "Failed"    // pod failed (may retry)
	TaskPhaseBlocked   TaskPhase = "Blocked"   // too many failures, needs human
)

func IsTerminal(phase TaskPhase) bool {
	return phase == TaskPhaseSucceeded || phase == TaskPhaseFailed || phase == TaskPhaseBlocked
}

type AgentJobStatus struct {
	Phase     TaskPhase    `json:"phase,omitempty"`
	Agent     string       `json:"agent,omitempty"`
	Slot      int          `json:"slot,omitempty"`
	JobName   string       `json:"jobName,omitempty"`
	LastError string       `json:"lastError,omitempty"`
	Reported   bool         `json:"reported,omitempty"`   // result comment posted to github
	LogTail    string       `json:"logTail,omitempty"`    // last meaningful output line
	NextAction string       `json:"nextAction,omitempty"` // needs-review, needs-human
	StartedAt  *metav1.Time `json:"startedAt,omitempty"`
	FinishedAt *metav1.Time `json:"finishedAt,omitempty"`
}

// AgentJobList contains a list of AgentJobs.
type AgentJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentJob `json:"items"`
}

// DeepCopyObject implementations for runtime.Object interface.
func (in *AgentJob) DeepCopyObject() runtime.Object {
	out := new(AgentJob)
	*out = *in
	out.ObjectMeta = *in.ObjectMeta.DeepCopy()
	if in.Status.StartedAt != nil {
		t := *in.Status.StartedAt
		out.Status.StartedAt = &t
	}
	if in.Status.FinishedAt != nil {
		t := *in.Status.FinishedAt
		out.Status.FinishedAt = &t
	}
	return out
}

func (in *AgentJobList) DeepCopyObject() runtime.Object {
	out := new(AgentJobList)
	out.TypeMeta = in.TypeMeta
	out.ListMeta = *in.ListMeta.DeepCopy()
	if in.Items != nil {
		out.Items = make([]AgentJob, len(in.Items))
		for i := range in.Items {
			out.Items[i] = *in.Items[i].DeepCopyObject().(*AgentJob)
		}
	}
	return out
}
