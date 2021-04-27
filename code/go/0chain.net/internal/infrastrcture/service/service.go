package service

import (
	"os"
	"os/signal"
	"syscall"
)

type Service interface {
	Name() string
	Init(opts ...Option)
	Options() Options
	Run() error
	Start() error
	Stop() error
	String() string
}

type service struct {
	opts Options
}

func (s *service) Init(opts ...Option) {

	for _, o := range opts {
		o(&s.opts)
	}

}

func (s *service) Start() error {

	return nil
}

func (s *service) Stop() error {

	s.Options().Logger.Infof("Stopping [%s] %s", s.String(), s.Name())

	//if err := s.opts.Logger.Flush(); err != nil {
	//	return err
	//}

	return nil
}

func (s *service) Name() string {
	return s.opts.Name
}

func (s *service) Options() Options {
	return s.opts
}

func (s *service) Run() error {

	s.Options().Logger.Infof("Starting [%s] %s", s.String(), s.Name())

	if err := s.Start(); err != nil {
		return err
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ch:
	case <-s.opts.Context.Done():
	}

	return s.Stop()
}

func (s *service) String() string {
	return "service"
}

// New new instance of service
func New(opts ...Option) Service {

	service := &service{}
	service.opts = newOptions(opts...)

	return service
}
