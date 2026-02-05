package worker

// Standard environment variables for the Handover Protocol.
const (
	// EnvResumeID is the unique session identifier for a worker.
	// It remains constant across restarts within the same supervisor lifecycle.
	EnvResumeID = "LIFECYCLE_RESUME_ID"

	// EnvPrevExit is the exit code of the previous execution of this worker.
	// It is injected by the supervisor upon restart.
	EnvPrevExit = "LIFECYCLE_PREV_EXIT"

	// EnvResumeToken is the opaque token used to resume a worker session.
	// It is injected by the supervisor upon restart/handover.
	EnvResumeToken = "LIFECYCLE_RESUME_TOKEN"
)

// EnvInjector is an optional interface for workers that support environment variable injection.
type EnvInjector interface {
	SetEnv(key, value string)
}



