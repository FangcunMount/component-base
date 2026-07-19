package messaging

import (
	"hash/fnv"
	"strconv"
	"time"
)

// RetryDelay returns a deterministic exponentially increasing delay. The
// deterministic jitter keeps tests repeatable and prevents a redelivery herd.
func RetryDelay(options RetryBackoffOptions, failedAttempt int, messageID string) time.Duration {
	if options.BaseDelay <= 0 {
		return 0
	}
	if failedAttempt < 1 {
		failedAttempt = 1
	}
	maxDelay := options.MaxDelay
	if maxDelay <= 0 || maxDelay < options.BaseDelay {
		maxDelay = options.BaseDelay
	}
	delay := options.BaseDelay
	for step := 1; step < failedAttempt && delay < maxDelay; step++ {
		if delay > maxDelay/2 {
			delay = maxDelay
			break
		}
		delay *= 2
	}
	if delay > maxDelay {
		delay = maxDelay
	}
	jitter := options.JitterFraction
	if jitter <= 0 {
		return delay
	}
	if jitter > 1 {
		jitter = 1
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(messageID + ":" + strconv.Itoa(failedAttempt)))
	unit := float64(hash.Sum32()%20001)/10000 - 1
	return delay + time.Duration(float64(delay)*jitter*unit)
}
