package di

import "fmt"

type ServiceFactoryFn = func(*Container) any

type Container struct {
	services map[string]ServiceFactoryFn
}

func NewContainer(services map[string]ServiceFactoryFn) *Container {
	return &Container{
		services: services,
	}
}

func (c *Container) Get(serviceName string) any {
	fn, ok := c.services[serviceName]
	if !ok {
		panic(fmt.Sprintf("service %s not found", serviceName))
	}

	return fn(c)
}
