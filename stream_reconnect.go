package hyperliquid

import (
	"context"
	"math/rand"
	"time"
)

// handleDisconnect tears down the current connection and schedules a
// reconnect. Safe to call repeatedly; no-op after Close.
func (w *Stream) handleDisconnect() {
	if w.closed.Load() {
		return
	}

	w.connected.Store(false)

	w.connMu.Lock()
	if w.conn != nil {
		_ = w.conn.Close()
		w.conn = nil
	}
	w.connMu.Unlock()

	w.scheduleReconnect()
}

// scheduleReconnect schedules an asynchronous reconnection attempt with
// exponential backoff and jitter. Subsequent failures double the delay
// until maxReconnectWait.
func (w *Stream) scheduleReconnect() {
	if w.closed.Load() {
		return
	}

	w.reconnectMu.Lock()
	defer w.reconnectMu.Unlock()

	if w.reconnectTimer != nil {
		w.reconnectTimer.Stop()
	}

	w.reconnectAttempts++
	attempts := w.reconnectAttempts

	if w.maxReconnectAttempts > 0 && attempts > w.maxReconnectAttempts {
		w.warnf("Max reconnection attempts (%d) reached, giving up", w.maxReconnectAttempts)
		return
	}

	backoff := w.reconnectWait * time.Duration(1<<(attempts-1))
	if backoff > maxReconnectWait {
		backoff = maxReconnectWait
	}
	jitter := time.Duration(float64(backoff) * 0.2 * (2*rand.Float64() - 1))
	delay := backoff + jitter
	if delay < time.Second {
		delay = time.Second
	}

	w.warnf("Reconnection attempt %d in %v...", attempts, delay)

	w.reconnectTimer = time.AfterFunc(delay, func() {
		if w.closed.Load() || w.connected.Load() {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
		err := w.Connect(ctx)
		cancel()

		if err != nil {
			w.warnf("Reconnection attempt %d failed: %v", attempts, err)
			w.scheduleReconnect()
		} else {
			w.warnf("Reconnection successful after %d attempts", attempts)
		}
	})
}
