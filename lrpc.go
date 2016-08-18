// Package lrpc is an alternative to the standard rpc paradigm used by go
// projects. It's better than normal rpc, because it allows for saner chaining
// of rpc handlers in a similar way to how http.Handler can be chained.
package lrpc

import (
	"errors"
	"fmt"
	"reflect"

	"context"
)

// Errors applicable to rpc calling/handling
var (
	ErrMethodNotFound = errors.New("method not found")
)

// Call represents an rpc call currently being processed.
type Call interface {
	// Context returns a context object for the rpc call. The context may
	// already have a deadline set on it, or other key/value information,
	// depending on the underlying implementation. The same Call instance should
	// always return the same Context.
	Context() context.Context

	// Method returns the name of the method being called. The same Call
	// instance should always return the same method name.
	Method() string

	// UnmarshalArgs takes in an interface pointer and unmarshals the Call's
	// arguments to the call into it. This should only be called once on any
	// Call instance.
	UnmarshalArgs(interface{}) error
}

// Handler describes a type which can process incoming rpc requests and return a
// response to them
type Handler interface {
	ServeRPC(Call) interface{}
}

// HandlerFunc is a wrapper around a simple ServeRPC style function to make it
// actually implement the interface
type HandlerFunc func(Call) interface{}

// ServeRPC implements the Handler interface, it simply calls the callee
// HandleFunc
func (hf HandlerFunc) ServeRPC(c Call) interface{} {
	return hf(c)
}

// DirectCall implements the Call interface, and can be used to call a Handler
// directly
//
//	res := existingHandler.ServeRPC(lrpc.NewDirectCall(
//		context.WithTimeout(context.Background(), 5 * time.Second),
//		"Method.Name",
//		map[string]string{"foo":"bar"},
//	))
//
type DirectCall struct {
	ctx    context.Context
	method string
	args   interface{}
}

// NewDirectCall returns an initialized DirectCall which can be used as an
// lrpc.Call. If ctx is nil then context.Background() will be used. args must be
// a pointer or reference type
func NewDirectCall(ctx context.Context, method string, args interface{}) DirectCall {
	return DirectCall{
		ctx:    ctx,
		method: method,
		args:   args,
	}
}

// Context implements the Call interface
func (dc DirectCall) Context() context.Context {
	if dc.ctx == nil {
		return context.Background()
	}
	return dc.ctx
}

// Method implements the Call interface
func (dc DirectCall) Method() string {
	return dc.method
}

// UnmarshalArgs implements the Call interface. It sets the value of the pointer
// type passed in to the value of the Args field on the struct, effectively
// copying it into the pointer. The type of the Args field must be assignable to
// the passed in type.
func (dc DirectCall) UnmarshalArgs(i interface{}) error {
	thisV := reflect.ValueOf(dc.args)
	iV := reflect.Indirect(reflect.ValueOf(i))
	if !iV.CanSet() {
		return fmt.Errorf("type isn't setable: %T", i)
	}
	iV.Set(thisV)
	return nil
}

// ServeMux wraps multiple method name/Handler pairs and implements a Handler
// which will call the appropriate Handler for the called method, or returns
// ErrMethodNotFound
type ServeMux map[string]Handler

// ServeRPC implements the Handler interface. See the ServeMux type's docstring
func (sm ServeMux) ServeRPC(c Call) interface{} {
	if h, ok := sm[c.Method()]; ok {
		return h.ServeRPC(c)
	}
	return ErrMethodNotFound
}

// Handle registers the given Handler for the given method name. It returns the
// ServeMux so multiple calls can be easily chained
func (sm ServeMux) Handle(method string, h Handler) ServeMux {
	sm[method] = h
	return sm
}

// HandleFunc is like Handle, but it takes in a function which will be
// automatically wrapped with HandlerFunc
func (sm ServeMux) HandleFunc(method string, fn func(Call) interface{}) ServeMux {
	return sm.Handle(method, HandlerFunc(fn))
}
