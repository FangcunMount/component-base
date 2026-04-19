// Copyright 2020 Lingfei Kong <colin404@foxmail.com>. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

/*
Package posixsignal provides a listener for a posix signal. By default
it listens for SIGINT and SIGTERM, but others can be chosen in NewPosixSignalManager.
When ShutdownFinish is called it exits with os.Exit(0)
*/
package posixsignal

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/FangcunMount/component-base/pkg/log"
	"github.com/FangcunMount/component-base/pkg/shutdown"
)

// Name defines shutdown manager name.
const Name = "PosixSignalManager"

// PosixSignalManager implements ShutdownManager interface that is added
// to GracefulShutdown. Initialize with NewPosixSignalManager.
type PosixSignalManager struct {
	signals []os.Signal
	flushFn func()
	exitFn  func(int)

	mu         sync.RWMutex
	lastSignal os.Signal
}

// NewPosixSignalManager initializes the PosixSignalManager.
// As arguments you can provide os.Signal-s to listen to, if none are given,
// it will default to SIGINT and SIGTERM.
func NewPosixSignalManager(sig ...os.Signal) *PosixSignalManager {
	if len(sig) == 0 {
		sig = make([]os.Signal, 2)
		sig[0] = os.Interrupt
		sig[1] = syscall.SIGTERM
	}

	return &PosixSignalManager{
		signals: sig,
		flushFn: log.Flush,
		exitFn:  os.Exit,
	}
}

// GetName returns name of this ShutdownManager.
func (posixSignalManager *PosixSignalManager) GetName() string {
	return Name
}

// Start starts listening for posix signals.
func (posixSignalManager *PosixSignalManager) Start(gs shutdown.GSInterface) error {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, posixSignalManager.signals...)

		// Block until a signal is received.
		sig := <-c
		signal.Stop(c)

		posixSignalManager.setLastSignal(sig)
		log.Infow("received shutdown signal",
			"shutdown_manager", posixSignalManager.GetName(),
			"signal", posixSignalManager.SignalName(),
		)

		gs.StartShutdown(posixSignalManager)
	}()

	return nil
}

// ShutdownStart logs the beginning of graceful shutdown.
func (posixSignalManager *PosixSignalManager) ShutdownStart() error {
	log.Infow("starting graceful shutdown",
		"shutdown_manager", posixSignalManager.GetName(),
		"signal", posixSignalManager.SignalName(),
	)
	return nil
}

// ShutdownFinish flushes logs and exits the app with status 0.
func (posixSignalManager *PosixSignalManager) ShutdownFinish() error {
	log.Infow("graceful shutdown completed, flushing logs before exit",
		"shutdown_manager", posixSignalManager.GetName(),
		"signal", posixSignalManager.SignalName(),
	)
	if posixSignalManager.flushFn != nil {
		posixSignalManager.flushFn()
	}
	if posixSignalManager.exitFn != nil {
		posixSignalManager.exitFn(0)
	}

	return nil
}

// SignalName returns the last observed signal for logging/debugging.
func (posixSignalManager *PosixSignalManager) SignalName() string {
	posixSignalManager.mu.RLock()
	defer posixSignalManager.mu.RUnlock()

	if posixSignalManager.lastSignal == nil {
		return "unknown"
	}
	return signalName(posixSignalManager.lastSignal)
}

func (posixSignalManager *PosixSignalManager) setLastSignal(sig os.Signal) {
	posixSignalManager.mu.Lock()
	defer posixSignalManager.mu.Unlock()
	posixSignalManager.lastSignal = sig
}

func signalName(sig os.Signal) string {
	if sig == nil {
		return "unknown"
	}
	if stringer, ok := sig.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%v", sig)
}
