package metrics

import (
	"strings"
	"time"
)

type guardedProvider struct {
	provider Provider
	policy   *LabelPolicy
}

func wrapProvider(p Provider, policy *LabelPolicy) Provider {
	if policy == nil {
		return p
	}
	return &guardedProvider{provider: p, policy: policy}
}

func (g *guardedProvider) sanitize(key, value string) (string, bool) {
	if g.policy == nil {
		return value, true
	}
	v := strings.TrimSpace(value)
	if g.policy.MaxLen > 0 && len(v) > g.policy.MaxLen {
		v = v[:g.policy.MaxLen]
	}
	if allow, ok := g.policy.Allowlist[key]; ok && len(allow) > 0 {
		if v == "" {
			if g.policy.Strict {
				return "", false
			}
			return LabelValueUnknown, true
		}
		if _, ok := allow[v]; !ok {
			if g.policy.Strict {
				return "", false
			}
			return LabelValueOther, true
		}
	}
	return v, true
}

func (g *guardedProvider) IncSignalReceived(sig string) {
	if v, ok := g.sanitize(LabelSignal, sig); ok {
		g.provider.IncSignalReceived(v)
	}
}

func (g *guardedProvider) IncProcessStarted() {
	g.provider.IncProcessStarted()
}

func (g *guardedProvider) IncProcessFailed() {
	g.provider.IncProcessFailed()
}

func (g *guardedProvider) IncTerminalUpgrade(success bool) {
	g.provider.IncTerminalUpgrade(success)
}

func (g *guardedProvider) IncHookExecuted() {
	g.provider.IncHookExecuted()
}

func (g *guardedProvider) IncHookPanicked() {
	g.provider.IncHookPanicked()
}

func (g *guardedProvider) ObserveHookDuration(d time.Duration) {
	g.provider.ObserveHookDuration(d)
}

func (g *guardedProvider) IncWorkerStarted(workerType string) {
	if v, ok := g.sanitize(LabelWorkerType, workerType); ok {
		g.provider.IncWorkerStarted(v)
	}
}

func (g *guardedProvider) IncWorkerStopped(workerType string) {
	if v, ok := g.sanitize(LabelWorkerType, workerType); ok {
		g.provider.IncWorkerStopped(v)
	}
}

func (g *guardedProvider) IncWorkerFailed(workerType string) {
	if v, ok := g.sanitize(LabelWorkerType, workerType); ok {
		g.provider.IncWorkerFailed(v)
	}
}

func (g *guardedProvider) ObserveWorkerDuration(workerType string, d time.Duration) {
	if v, ok := g.sanitize(LabelWorkerType, workerType); ok {
		g.provider.ObserveWorkerDuration(v, d)
	}
}

func (g *guardedProvider) IncSupervisorRestart(supervisorName, strategy string) {
	name, okName := g.sanitize(LabelSupervisor, supervisorName)
	strat, okStrat := g.sanitize(LabelStrategy, strategy)
	if okName && okStrat {
		g.provider.IncSupervisorRestart(name, strat)
	}
}

func (g *guardedProvider) IncChildRestart(supervisorName, childName string) {
	sup, okSup := g.sanitize(LabelSupervisor, supervisorName)
	child, okChild := g.sanitize(LabelChild, childName)
	if okSup && okChild {
		g.provider.IncChildRestart(sup, child)
	}
}

func (g *guardedProvider) IncSupervisorAdd(supervisorName string) {
	if v, ok := g.sanitize(LabelSupervisor, supervisorName); ok {
		g.provider.IncSupervisorAdd(v)
	}
}

func (g *guardedProvider) IncSupervisorRemove(supervisorName string) {
	if v, ok := g.sanitize(LabelSupervisor, supervisorName); ok {
		g.provider.IncSupervisorRemove(v)
	}
}

func (g *guardedProvider) IncBackoffTriggered(childName string, d time.Duration) {
	if v, ok := g.sanitize(LabelChild, childName); ok {
		g.provider.IncBackoffTriggered(v, d)
	}
}

func (g *guardedProvider) ObserveShutdownDuration(workerType string, d time.Duration) {
	if v, ok := g.sanitize(LabelWorkerType, workerType); ok {
		g.provider.ObserveShutdownDuration(v, d)
	}
}

func (g *guardedProvider) IncForceExitTriggered() {
	g.provider.IncForceExitTriggered()
}

func (g *guardedProvider) IncCircuitBreakerTriggered(childName string) {
	if v, ok := g.sanitize(LabelChild, childName); ok {
		g.provider.IncCircuitBreakerTriggered(v)
	}
}

func (g *guardedProvider) IncCriticalSectionStarted() {
	g.provider.IncCriticalSectionStarted()
}

func (g *guardedProvider) IncCriticalSectionFinished(success bool) {
	g.provider.IncCriticalSectionFinished(success)
}

func (g *guardedProvider) ObserveCriticalSectionDuration(d time.Duration) {
	g.provider.ObserveCriticalSectionDuration(d)
}

func (g *guardedProvider) IncContainerStarted(image string) {
	if v, ok := g.sanitize(LabelImage, image); ok {
		g.provider.IncContainerStarted(v)
	}
}

func (g *guardedProvider) IncContainerStopped(image string) {
	if v, ok := g.sanitize(LabelImage, image); ok {
		g.provider.IncContainerStopped(v)
	}
}

func (g *guardedProvider) IncContainerFailed(image string) {
	if v, ok := g.sanitize(LabelImage, image); ok {
		g.provider.IncContainerFailed(v)
	}
}

func (g *guardedProvider) ObserveContainerDuration(image string, d time.Duration) {
	if v, ok := g.sanitize(LabelImage, image); ok {
		g.provider.ObserveContainerDuration(v, d)
	}
}

func (g *guardedProvider) IncGoroutineStarted() {
	g.provider.IncGoroutineStarted()
}

func (g *guardedProvider) IncGoroutineFinished() {
	g.provider.IncGoroutineFinished()
}

func (g *guardedProvider) IncGoroutinePanicked() {
	g.provider.IncGoroutinePanicked()
}

func (g *guardedProvider) ObserveGoroutineBlockDuration(d time.Duration) {
	g.provider.ObserveGoroutineBlockDuration(d)
}

func (g *guardedProvider) IncGoroutineWaiting() {
	g.provider.IncGoroutineWaiting()
}

func (g *guardedProvider) DecGoroutineWaiting() {
	g.provider.DecGoroutineWaiting()
}

func (g *guardedProvider) IncEventEmitted(source string) {
	if v, ok := g.sanitize(LabelSource, source); ok {
		g.provider.IncEventEmitted(v)
	}
}

func (g *guardedProvider) IncEventRouted(topic string) {
	if v, ok := g.sanitize(LabelTopic, topic); ok {
		g.provider.IncEventRouted(v)
	}
}

func (g *guardedProvider) IncHandlerExecuted(topic string) {
	if v, ok := g.sanitize(LabelTopic, topic); ok {
		g.provider.IncHandlerExecuted(v)
	}
}

func (g *guardedProvider) IncHandlerError(topic string, err error) {
	if v, ok := g.sanitize(LabelTopic, topic); ok {
		g.provider.IncHandlerError(v, err)
	}
}

func (g *guardedProvider) ObserveHandlerDuration(topic string, d time.Duration) {
	if v, ok := g.sanitize(LabelTopic, topic); ok {
		g.provider.ObserveHandlerDuration(v, d)
	}
}

func (g *guardedProvider) ObserveEventBlockDuration(source string, d time.Duration) {
	if v, ok := g.sanitize(LabelSource, source); ok {
		g.provider.ObserveEventBlockDuration(v, d)
	}
}

func (g *guardedProvider) IncEventWaiting(source string) {
	if v, ok := g.sanitize(LabelSource, source); ok {
		g.provider.IncEventWaiting(v)
	}
}

func (g *guardedProvider) DecEventWaiting(source string) {
	if v, ok := g.sanitize(LabelSource, source); ok {
		g.provider.DecEventWaiting(v)
	}
}
