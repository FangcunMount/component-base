package processruntime

import "sync"

// ServiceRunner 描述一个长跑服务。
type ServiceRunner struct {
	Name string
	Run  func() error
}

// RunGroup 启动一组长跑服务，并返回首个服务错误。
type RunGroup struct {
	StartShutdown func() error
	Services      []ServiceRunner
}

// Run 先启动关闭管理器，再并发运行所有服务。
func (g RunGroup) Run() error {
	if g.StartShutdown != nil {
		if err := g.StartShutdown(); err != nil {
			return err
		}
	}

	errCh := make(chan error, len(g.Services))
	done := make(chan struct{})
	var wg sync.WaitGroup

	active := 0
	for _, service := range g.Services {
		if service.Run == nil {
			continue
		}
		active++
		wg.Add(1)
		go func(run func() error) {
			defer wg.Done()
			if err := run(); err != nil {
				errCh <- err
			}
		}(service.Run)
	}

	if active == 0 {
		return nil
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errCh:
		return err
	case <-done:
		return nil
	}
}
