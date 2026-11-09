/*
Copyright 2026 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metricshandler

import (
	"context"
	"errors"
	"time"

	"k8s.io/klog/v2"

	ksmtypes "k8s.io/kube-state-metrics/v2/pkg/builder/types"
)

type syncWaiter struct {
	revision uint64
	done     chan struct{}
}

func (m *MetricsHandler) startRebuildWorker() {
	m.workerWG.Add(1)
	go func() {
		defer m.workerWG.Done()
		m.rebuildWorker()
	}()
}

func (m *MetricsHandler) stopRebuildWorker() {
	m.workerStopped.Do(func() {
		close(m.workerStop)
		m.cancelActiveCandidate()
		m.releaseAllSyncWaiters()
		m.workerWG.Wait()
	})
}

func (m *MetricsHandler) releaseAllSyncWaiters() {
	m.rebuildMu.Lock()
	defer m.rebuildMu.Unlock()
	for _, w := range m.syncWaiters {
		closeWaiter(w)
	}
	m.syncWaiters = m.syncWaiters[:0]
}

func (m *MetricsHandler) signalRebuildWorker() {
	select {
	case m.workerWake <- struct{}{}:
	default:
	}
}

// runContext returns the context passed to Run, or Background before Run.
func (m *MetricsHandler) runContext() context.Context {
	m.runCtxMu.Lock()
	defer m.runCtxMu.Unlock()
	if m.workerParentCtx == nil {
		return context.Background()
	}
	return m.workerParentCtx
}

func (m *MetricsHandler) registerRequestOwnerLocked(callerCtx context.Context, revision uint64) {
	if callerCtx == nil {
		callerCtx = context.Background()
	}
	m.requestOwnerByRev[revision] = callerCtx
}

// prepareCandidateGenerationLocked must be called with rebuildMu held. It rechecks that attemptRevision
// is still the head config (supersession) before creating the generation context and registering
// activeCancel so a concurrent enqueue cannot miss a cancel handle for an in-flight obsolete attempt.
func (m *MetricsHandler) prepareCandidateGenerationLocked(attemptRevision uint64) (context.Context, context.CancelFunc, func(), bool) {
	if m.configRevision.Load() != attemptRevision {
		return nil, nil, nil, false
	}
	runCtx := m.runContext()
	genCtx, genCancel := context.WithCancel(runCtx)
	owner := m.requestOwnerByRev[attemptRevision]

	var stopBridge func()
	if owner == nil {
		stopBridge = func() {}
	} else {
		stopCh := make(chan struct{})
		go func() {
			select {
			case <-owner.Done():
				genCancel()
			case <-stopCh:
			}
		}()
		stopBridge = func() { close(stopCh) }
	}

	m.activeCandidateGen = attemptRevision
	if m.storeSyncTimeout > 0 {
		m.activeCancel = genCancel
	}
	return genCtx, genCancel, stopBridge, true
}

func (m *MetricsHandler) removeRequestOwner(revision uint64) {
	m.rebuildMu.Lock()
	delete(m.requestOwnerByRev, revision)
	m.rebuildMu.Unlock()
}

// requestOwnerCanceled reports whether any registered caller context for this revision is done.
// Checked synchronously on the worker goroutine to avoid racing async owner→genCtx bridging.
func (m *MetricsHandler) requestOwnerCanceled(revision uint64) bool {
	m.rebuildMu.Lock()
	owner := m.requestOwnerByRev[revision]
	m.rebuildMu.Unlock()
	if owner == nil {
		return false
	}
	select {
	case <-owner.Done():
		return true
	default:
		return false
	}
}

func (m *MetricsHandler) enqueueRebuild(callerCtx context.Context, wait bool) {
	if m.storeSyncTimeout < 0 {
		return
	}

	if m.workerStopping() {
		return
	}

	m.rebuildMu.Lock()
	revision := m.configRevision.Add(1)
	m.registerRequestOwnerLocked(callerCtx, revision)
	if m.activeCancel != nil && m.cancelActiveGeneration() {
		m.activeCancel()
	}
	var waiter *syncWaiter
	if wait {
		waiter = &syncWaiter{revision: revision, done: make(chan struct{})}
		m.syncWaiters = append(m.syncWaiters, waiter)
	}
	m.rebuildMu.Unlock()

	m.signalRebuildWorker()

	if !wait || waiter == nil {
		return
	}
	select {
	case <-waiter.done:
	case <-callerCtx.Done():
	case <-m.workerStop:
	}
}

func (m *MetricsHandler) releaseSyncWaitersThrough(revision uint64) {
	m.rebuildMu.Lock()
	defer m.rebuildMu.Unlock()
	remaining := m.syncWaiters[:0]
	for _, w := range m.syncWaiters {
		if w.revision <= revision {
			closeWaiter(w)
			continue
		}
		remaining = append(remaining, w)
	}
	m.syncWaiters = remaining
}

func closeWaiter(w *syncWaiter) {
	select {
	case <-w.done:
	default:
		close(w.done)
	}
}

func (m *MetricsHandler) rebuildWorker() {
	for {
		select {
		case <-m.workerStop:
			m.cancelActiveCandidate()
			m.releaseAllSyncWaiters()
			return
		case <-m.workerWake:
		}

		for {
			if m.workerStopping() {
				return
			}
			head := m.configRevision.Load()
			lh := m.lastHandledRev.Load()
			if lh >= head {
				break
			}
			next := lh + 1

			// Coalesce: newer revision(s) may have registered after head was read — reload and
			// skip sync/build for stale work (no WaitForStoresSync for superseded revisions).
			head = m.configRevision.Load()
			if next < head {
				m.releaseSyncWaitersThrough(next)
				m.removeRequestOwner(next)
				m.lastHandledRev.Store(next)
				continue
			}

			_, timedOut, advance := m.runOneCandidate(next)
			// Unblock synchronous BuildWriters/ConfigureStore waiters after each candidate
			// attempt (success, discard, or sync timeout); mirrors prior worker behavior.
			m.releaseSyncWaitersThrough(next)

			if m.workerStopping() {
				return
			}

			if !advance && timedOut {
				if !m.retriesAllowedForRevision(next) {
					m.removeRequestOwner(next)
					m.lastHandledRev.Store(next)
					continue
				}
				if !m.cooldown(next) {
					return
				}
				if !m.retriesAllowedForRevision(next) {
					m.removeRequestOwner(next)
					m.lastHandledRev.Store(next)
					continue
				}
				m.signalRebuildWorker()
				continue
			}

			if advance {
				m.removeRequestOwner(next)
				m.lastHandledRev.Store(next)
			}
		}
	}
}

func (m *MetricsHandler) workerStopping() bool {
	select {
	case <-m.workerStop:
		return true
	default:
		return false
	}
}

func (m *MetricsHandler) cooldown(waitRevision uint64) bool {
	m.cooldownActive.Store(true)
	m.candidateActive.Store(false)
	defer m.cooldownActive.Store(false)

	timer := time.NewTimer(m.storeSyncTimeout)
	defer timer.Stop()
	for {
		m.rebuildMu.Lock()
		owner := m.requestOwnerByRev[waitRevision]
		m.rebuildMu.Unlock()
		runDone := m.runContext().Done()
		if owner == nil {
			select {
			case <-m.workerStop:
				return false
			case <-m.workerWake:
				if m.configRevision.Load() > waitRevision {
					return true
				}
			case <-runDone:
				return true
			case <-timer.C:
				return true
			}
			continue
		}
		select {
		case <-m.workerStop:
			return false
		case <-m.workerWake:
			if m.configRevision.Load() > waitRevision {
				return true
			}
		case <-runDone:
			return true
		case <-owner.Done():
			return true
		case <-timer.C:
			return true
		}
	}
}

func (m *MetricsHandler) retriesAllowedForRevision(rev uint64) bool {
	runDone := m.runContext().Done()
	m.rebuildMu.Lock()
	owner := m.requestOwnerByRev[rev]
	m.rebuildMu.Unlock()
	if owner == nil {
		select {
		case <-runDone:
			return false
		default:
			return true
		}
	}
	select {
	case <-runDone:
		return false
	case <-owner.Done():
		return false
	default:
		return true
	}
}

func (m *MetricsHandler) cancelActiveCandidate() {
	m.rebuildMu.Lock()
	cancel := m.activeCancel
	m.activeCancel = nil
	m.activeCandidateGen = 0
	m.rebuildMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (m *MetricsHandler) clearActiveCandidateCancel(gen uint64) {
	m.rebuildMu.Lock()
	if m.activeCandidateGen == gen {
		m.activeCancel = nil
		m.activeCandidateGen = 0
	}
	m.rebuildMu.Unlock()
}

func (m *MetricsHandler) runOneCandidate(attemptRevision uint64) (published bool, timedOut bool, advance bool) {
	if m.requestOwnerCanceled(attemptRevision) {
		return false, false, true
	}

	m.rebuildMu.Lock()
	genCtx, genCancel, stopOwnerBridge, ok := m.prepareCandidateGenerationLocked(attemptRevision)
	m.rebuildMu.Unlock()
	if !ok {
		return false, false, true
	}

	m.candidateActive.Store(true)
	defer m.candidateActive.Store(false)

	writers, waitErr := m.doRebuild(genCtx)
	if m.beforeSwap != nil {
		m.beforeSwap()
	}

	if m.requestOwnerCanceled(attemptRevision) {
		discardCandidate := func() {
			m.clearActiveCandidateCancel(attemptRevision)
			genCancel()
			stopOwnerBridge()
		}
		m.logSkippedInstall(genCtx, waitErr, len(writers))
		discardCandidate()
		return false, false, true
	}

	discardCandidate := func() {
		m.clearActiveCandidateCancel(attemptRevision)
		genCancel()
		stopOwnerBridge()
	}

	if genCtx.Err() != nil {
		m.logSkippedInstall(genCtx, waitErr, len(writers))
		discardCandidate()
		return false, false, true
	}
	if m.configRevision.Load() != attemptRevision {
		m.logSkippedInstall(genCtx, waitErr, len(writers))
		discardCandidate()
		return false, false, true
	}
	if !installableSync(waitErr) {
		if errors.Is(waitErr, ksmtypes.ErrStoreSyncTimeout) {
			klog.InfoS("Store sync observation window expired; canceling candidate and scheduling retry",
				"writerCount", len(writers),
			)
			discardCandidate()
			return false, true, false
		}
		m.logSkippedInstall(genCtx, waitErr, len(writers))
		discardCandidate()
		return false, false, true
	}
	if m.installWriters(genCtx, attemptRevision, writers, genCancel, stopOwnerBridge) {
		klog.InfoS("Installed metrics writers after store sync", "writerCount", len(writers))
		return true, false, true
	}
	if m.onCandidateCancel != nil {
		m.onCandidateCancel()
	}
	discardCandidate()
	return false, false, true
}

// workerQuiescent reports whether the worker is not building, syncing, or in retry cooldown.
func (m *MetricsHandler) workerQuiescent() bool {
	if m.candidateActive.Load() || m.cooldownActive.Load() {
		return false
	}
	head := m.configRevision.Load()
	lh := m.lastHandledRev.Load()
	if head > lh {
		return false
	}
	m.rebuildMu.Lock()
	defer m.rebuildMu.Unlock()
	return len(m.syncWaiters) == 0
}
