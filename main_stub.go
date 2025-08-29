//go:build !windows

package main

import (
	"flag"
	"log"

	"github.com/cupbot/cupbot/internal/service"
)

func main() {
	// Parse command line flags (for compatibility)
	serviceFlag := flag.Bool("service", false, "Run as Windows service (not supported on this platform)")
	installService := flag.Bool("install", false, "Install as Windows service (not supported on this platform)")
	uninstallService := flag.Bool("uninstall", false, "Uninstall Windows service (not supported on this platform)")
	debugService := flag.Bool("debug", false, "Run service in debug mode")
	flag.Parse()

	// Windows service operations are not supported on non-Windows platforms
	if *installService {
		log.Println("Service installation is only supported on Windows")
		return
	}

	if *uninstallService {
		log.Println("Service uninstallation is only supported on Windows")
		return
	}

	if *serviceFlag {
		log.Println("Windows service mode is not supported on this platform, running in interactive mode")
	}

	if *debugService {
		log.Println("Running in debug mode")
	}

	// Always run in interactive mode on non-Windows platforms
	err := service.RunInteractive()
	if err != nil {
		log.Fatalf("Failed to run bot: %v", err)
	}
}