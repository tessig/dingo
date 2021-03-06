package dingo

import (
	"reflect"
	"sync"
)

type (
	// Scope defines a scope's behaviour
	Scope interface {
		ResolveType(t reflect.Type, annotation string, unscoped func(t reflect.Type, annotation string, optional bool) reflect.Value) reflect.Value
	}

	identifier struct {
		t reflect.Type
		a string
	}

	// SingletonScope is our Scope to handle Singletons
	// todo use RWMutex for proper locking
	SingletonScope struct {
		mu           sync.Mutex                   // lock guarding instaceLocks
		instanceLock map[identifier]*sync.RWMutex // lock guarding instances
		instances    sync.Map
	}

	// ChildSingletonScope manages child-specific singleton
	ChildSingletonScope SingletonScope
)

var (
	// Singleton is the default SingletonScope for dingo
	Singleton Scope = NewSingletonScope()

	// ChildSingleton is a per-child singleton, means singletons are scoped and local to an injector instance
	ChildSingleton Scope = NewChildSingletonScope()
)

// NewSingletonScope creates a new singleton scope
func NewSingletonScope() *SingletonScope {
	return &SingletonScope{instanceLock: make(map[identifier]*sync.RWMutex)}
}

// NewChildSingletonScope creates a new child singleton scope
func NewChildSingletonScope() *ChildSingletonScope {
	return &ChildSingletonScope{instanceLock: make(map[identifier]*sync.RWMutex)}
}

// ResolveType resolves a request in this scope
func (s *SingletonScope) ResolveType(t reflect.Type, annotation string, unscoped func(t reflect.Type, annotation string, optional bool) reflect.Value) reflect.Value {
	ident := identifier{t, annotation}

	// try to get the instance type lock
	s.mu.Lock()

	if l, ok := s.instanceLock[ident]; ok {
		// we have the instance lock
		s.mu.Unlock()
		l.RLock()
		defer l.RUnlock()

		instance, _ := s.instances.Load(ident)
		return instance.(reflect.Value)
	}

	s.instanceLock[ident] = new(sync.RWMutex)
	l := s.instanceLock[ident]
	l.Lock()
	s.mu.Unlock()

	instance := unscoped(t, annotation, false)
	s.instances.Store(ident, instance)

	defer l.Unlock()

	return instance
}

// ResolveType delegates to SingletonScope.ResolveType
func (c *ChildSingletonScope) ResolveType(t reflect.Type, annotation string, unscoped func(t reflect.Type, annotation string, optional bool) reflect.Value) reflect.Value {
	return (*SingletonScope)(c).ResolveType(t, annotation, unscoped)
}
