//go:build windows

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cupbot/cupbot/internal/service"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func main() {
	// Parse command line flags
	serviceFlag := flag.Bool("service", false, "Run as Windows service")
	installService := flag.Bool("install", false, "Install as Windows service")
	uninstallService := flag.Bool("uninstall", false, "Uninstall Windows service")
	debugService := flag.Bool("debug", false, "Run service in debug mode")
	flag.Parse()

	const serviceName = "CupBot"

	// Handle service installation/uninstallation
	if *installService {
		err := installSvc(serviceName)
		if err != nil {
			log.Fatalf("Failed to install service: %v", err)
		}
		fmt.Println("Service installed successfully")
		return
	}

	if *uninstallService {
		err := removeSvc(serviceName)
		if err != nil {
			log.Fatalf("Failed to uninstall service: %v", err)
		}
		fmt.Println("Service uninstalled successfully")
		return
	}

	// Check if running as Windows service
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Failed to determine if running as service: %v", err)
	}

	if isService || *serviceFlag {
		// Run as Windows service
		err := service.RunService(serviceName, *debugService)
		if err != nil {
			log.Fatalf("Service failed: %v", err)
		}
	} else {
		// Run in interactive mode
		err := service.RunInteractive()
		if err != nil {
			log.Fatalf("Failed to run bot: %v", err)
		}
	}
}

func installSvc(name string) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}

	s, err = m.CreateService(name, exePath, mgr.Config{
		DisplayName:      "CupBot Telegram Bot",
		Description:      "Telegram bot for remote computer management",
		StartType:        mgr.StartAutomatic,
		ServiceStartName: "", // Run as LocalSystem
	})
	if err != nil {
		return err
	}
	defer s.Close()

	return nil
}

func removeSvc(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()

	// Stop the service if it's running
	status, err := s.Query()
	if err != nil {
		return err
	}
	if status.State != svc.Stopped {
		_, err = s.Control(svc.Stop)
		if err != nil {
			return err
		}
	}

	err = s.Delete()
	if err != nil {
		return err
	}

	return nil
}
