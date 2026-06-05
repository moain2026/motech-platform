package agent

import (
	"log"

	"github.com/kardianos/service"
)

// svcProgram adapts the Agent to the kardianos service interface.
type svcProgram struct {
	a    *Agent
	stop chan struct{}
}

// Start is called by the service manager; it launches the heartbeat loop.
func (p *svcProgram) Start(s service.Service) error {
	p.stop = make(chan struct{})
	go p.a.loop(p.stop)
	return nil
}

// Stop is called by the service manager on shutdown.
func (p *svcProgram) Stop(s service.Service) error {
	close(p.stop)
	return nil
}

// serviceConfig defines how the agent registers itself as a service.
func serviceConfig() *service.Config {
	return &service.Config{
		Name:        "MotechConnect",
		DisplayName: "Motech Connect Agent",
		Description: "Secure remote access agent (Motech / Al-Abbasi Soft).",
		Arguments:   []string{"run"},
	}
}

func (a *Agent) newService() (service.Service, error) {
	return service.New(&svcProgram{a: a}, serviceConfig())
}

// RunService runs the agent under the service manager (blocking).
func (a *Agent) RunService() error {
	s, err := a.newService()
	if err != nil {
		return err
	}
	return s.Run()
}

// InstallService registers the agent as an OS service and starts it.
func (a *Agent) InstallService() error {
	s, err := a.newService()
	if err != nil {
		return err
	}
	if err := s.Install(); err != nil {
		return err
	}
	log.Println("service installed")
	return s.Start()
}

// UninstallService stops and removes the OS service.
func (a *Agent) UninstallService() error {
	s, err := a.newService()
	if err != nil {
		return err
	}
	_ = s.Stop()
	return s.Uninstall()
}
