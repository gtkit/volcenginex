package volcenginex

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"time"
)

func (s *DocumentService) waitTask(ctx context.Context, projectID, taskID int32, policy PollPolicy) (*TaskDetail, error) {
	if ctx == nil || projectID <= 0 || taskID <= 0 {
		return nil, wrapSentinel("WaitTask", ErrInvalidArgument, "project ID and task ID must be positive")
	}
	if policy.Timeout <= 0 {
		policy = policy.withDefaults()
	}
	sleep := s.sleep
	if sleep == nil {
		sleep = waitDuration
	}
	waitCtx, cancel := context.WithTimeout(ctx, policy.Timeout)
	defer cancel()
	interval := policy.Interval
	if interval <= 0 {
		interval = 3 * time.Second
	}
	if policy.MaxInterval < interval {
		policy.MaxInterval = interval
	}
	failures := 0
	for {
		if err := waitCtx.Err(); err != nil {
			return nil, err
		}
		detail, err := s.TaskDetail(waitCtx, projectID, taskID, false)
		if err != nil {
			if waitCtx.Err() != nil {
				return nil, waitCtx.Err()
			}
			failures++
			if failures >= 5 {
				return nil, fmt.Errorf("%w: %v", ErrPollExhausted, err)
			}
			if err := sleep(waitCtx, jitterDuration(interval, policy.Jitter)); err != nil {
				return nil, err
			}
			interval = minDuration(policy.MaxInterval, interval*2)
			continue
		}
		failures = 0
		if s.progressHook != nil {
			s.progressHook(detail)
		}
		switch detail.State {
		case TaskDone:
			return detail, nil
		case TaskStopped, TaskDeleted:
			return nil, &Error{Op: "WaitTask", Code: "TaskAborted", Message: "document task reached an aborted state", Err: ErrTaskAborted}
		}
		if err := sleep(waitCtx, jitterDuration(interval, policy.Jitter)); err != nil {
			return nil, err
		}
		interval = minDuration(policy.MaxInterval, interval*2)
	}
}

func waitDuration(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func jitterDuration(base time.Duration, jitter float64) time.Duration {
	if base <= 0 || jitter <= 0 {
		return base
	}
	var randomBytes [8]byte
	if _, err := crand.Read(randomBytes[:]); err != nil {
		return base
	}
	random := float64(binary.LittleEndian.Uint64(randomBytes[:])) / float64(^uint64(0))
	factor := 1 + (random*2-1)*jitter
	if factor < 0 {
		factor = 0
	}
	return time.Duration(float64(base) * factor)
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
