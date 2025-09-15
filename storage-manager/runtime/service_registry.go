package runtime

import (
	"fmt"
	"reflect"

	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/util"
)

var log = util.NewLogger("registry")

type Service interface {
	Start()
	Stop() error
}

type ServiceRegistry struct {
	services     map[reflect.Type]Service
	serviceTypes []reflect.Type
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[reflect.Type]Service),
	}
}

func (s *ServiceRegistry) StartAll() {
	log.Debugf("Starting all %d services: %v", len(s.serviceTypes), s.serviceTypes)
	for idx, kind := range s.serviceTypes {
		serviceName := kind.String()
		log.Debugf("Starting service [%d/%d]: %s", idx+1, len(s.serviceTypes), serviceName)
		go s.services[kind].Start()
	}
}

func (s *ServiceRegistry) StopAll() {
	for i := len(s.serviceTypes) - 1; i >= 0; i-- {
		kind := s.serviceTypes[i]
		service := s.services[kind]
		if err := service.Stop(); err != nil {
			log.WithError(err).Errorf("Could not stop the following service: %v", kind)
		}
	}
}

func (s *ServiceRegistry) RegisterService(service Service) error {
	kind := reflect.TypeOf(service)
	if _, exists := s.services[kind]; exists {
		return fmt.Errorf("service already exists: %v", kind)
	}
	s.services[kind] = service
	s.serviceTypes = append(s.serviceTypes, kind)
	return nil
}

func (s *ServiceRegistry) FetchService(service interface{}) error {
	if reflect.TypeOf(service).Kind() != reflect.Ptr {
		return fmt.Errorf("input must be of pointer type, received value type instead: %T", service)
	}
	element := reflect.ValueOf(service).Elem()
	if running, ok := s.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return fmt.Errorf("unknown service: %T", service)
}
