package metrics

// Label keys used for metrics label cardinality enforcement.
const (
	LabelSignal     = "signal"
	LabelWorkerType = "worker_type"
	LabelStrategy   = "strategy"
	LabelSupervisor = "supervisor"
	LabelChild      = "child"
	LabelSource     = "source"
	LabelTopic      = "topic"
	LabelImage      = "image"
)

const (
	LabelValueOther    = "other"
	LabelValueUnknown  = "unknown"
	DefaultLabelMaxLen = 64
)

// LabelPolicy controls how metric labels are sanitized or rejected.
type LabelPolicy struct {
	MaxLen    int
	Strict    bool
	Allowlist map[string]map[string]struct{}
}

// DefaultLabelPolicy provides a conservative allowlist for low-cardinality labels.
func DefaultLabelPolicy() *LabelPolicy {
	return &LabelPolicy{
		MaxLen: DefaultLabelMaxLen,
		Allowlist: map[string]map[string]struct{}{
			LabelSignal: toSet(
				"SIGINT",
				"SIGTERM",
				"SIGQUIT",
				"SIGHUP",
				"SIGKILL",
				"SIGABRT",
				"SIGSTOP",
				"SIGCONT",
			),
			LabelStrategy: toSet(
				"OneForOne",
				"OneForAll",
			),
			LabelWorkerType: toSet(
				"process",
				"container",
				"func",
				"supervisor",
				"goroutine",
				"supervisor_child",
			),
		},
	}
}

// SetLabelPolicy configures label sanitization for all metrics providers.
//
// Passing nil disables label enforcement and uses raw values.
func SetLabelPolicy(p *LabelPolicy) {
	providerMu.Lock()
	defer providerMu.Unlock()
	if p != nil && p.MaxLen == 0 {
		p.MaxLen = DefaultLabelMaxLen
	}
	labelPolicy = p
	provider = wrapProvider(baseProvider, labelPolicy)
}

// GetLabelPolicy returns the active label policy, if any.
func GetLabelPolicy() *LabelPolicy {
	providerMu.RLock()
	defer providerMu.RUnlock()
	return labelPolicy
}

func toSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, v := range values {
		set[v] = struct{}{}
	}
	return set
}
