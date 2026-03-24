package service

import (
	"context"
	"sync"

	"golang.org/x/sys/windows/svc"
)

const Name = "PamElevationAgent"

// Program implements svc.Handler and runs the elevation broker.
type Program struct {
	BaseDir string
	Run     func(ctx context.Context, baseDir string) error
}

// Execute runs the service until stop/shutdown.
func (p *Program) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	var runErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		if p.Run != nil {
			runErr = p.Run(ctx, p.BaseDir)
		}
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				cancel()
				changes <- svc.Status{State: svc.StopPending}
				break loop
			default:
			}
		}
	}

	wg.Wait()
	if runErr != nil && runErr != context.Canceled {
		// SCM does not surface Go errors; log file captures details.
	}
	changes <- svc.Status{State: svc.Stopped}
	return false, 0
}
