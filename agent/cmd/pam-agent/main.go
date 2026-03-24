//go:build windows

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"pam-platform/agent/service"

	"golang.org/x/sys/windows/svc"
)

func main() {
	baseDir, err := service.BaseDirFromExecutable()
	if err != nil {
		log.Fatal(err)
	}

	isSvc, err := svc.IsWindowsService()
	if err != nil {
		log.Fatal(err)
	}

	prog := &service.Program{
		BaseDir: baseDir,
		Run:     service.RunAgent,
	}

	if isSvc {
		if err := svc.Run(service.Name, prog); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Console mode for local development / manual testing.
	if len(os.Args) > 1 && os.Args[1] == "-h" {
		fmt.Fprintf(os.Stderr, "usage: %s [run]\n  (no args) run named pipe server in foreground\n", os.Args[0])
		os.Exit(2)
	}
	fmt.Println("pam-agent: running in console mode (not installed as a service)")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := service.RunAgent(ctx, baseDir); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
