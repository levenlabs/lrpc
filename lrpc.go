// Package lrpc is an alternative to the standard rpc paradigm used by go
// projects. It's better than normal rpc, because it allows for saner chaining
// of rpc handlers in a similar way to how http.Handler can be chained.
package lrpc

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	"golang.org/x/net/context"
)

// TODO the documentation in here really doesn't need to mention codecs. Might
// not even need to mention transports

// Errors applicable to rpc calling/handling
var (
	ErrMethodNotFound = errors.New("method not found")
)

// Call represents an rpc call currently being processed.
type Call interface {
	// GetContext returns a context object for the rpc call. The context may
	// already have a deadline set on it, or other key/value information,
	// depending on the underlying transport/codec. The same Call instance
	// should always return the same Context.
	GetContext() context.Context

	// GetMethod returns the name of the method being called. The same Call
	// instance should always return the same method name.
	GetMethod() string

	// UnmarshalArgs takes in an interface pointer and unmarshals the Call's
	// arguments into it using the underlying codec. This should only be called
	// once on any Call instance.
	UnmarshalArgs(interface{}) error

	// MarshalResponse takes in an interface pointer and writes it to the given
	// io.Writer. This should only be called once on any Call instance.
	//
	// This is generally only used by the underlying transport for a Call, it's
	// not usually necessary to call this from a Handler
	//
	// TODO still not convinced this needs to be here
	MarshalResponse(io.Writer, interface{}) error
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
//	res := existingHandler.ServeRPC(lrpc.DirectCall{
//		Method: "Method.Name",
//		Args: map[string]string{"foo":"bar"},
//	})
//
// The MarshalResponse method on DirectCall will panic, since it should never
// actually be used. The Context method will return a new background context if
// none is set in the struct.
type DirectCall struct {
	Context context.Context
	Method  string

	// Args must be a pointer or reference type
	Args interface{}
}

// GetContext implements the Call interface. Returns context.Background() if one
// isn't set in the struct.
func (dc DirectCall) GetContext() context.Context {
	if dc.Context == nil {
		return context.Background()
	}
	return dc.Context
}

// GetMethod implements the Call interface. Returns the Method field of the
// struct directly
func (dc DirectCall) GetMethod() string {
	return dc.Method
}

// UnmarshalArgs implements the Call interface. It sets the value of the pointer
// type passed in to the value of the Args field on the struct, effectively
// copying it into the pointer. The type of the Args field must be assignable to
// the passed in type.
func (dc DirectCall) UnmarshalArgs(i interface{}) error {
	thisV := reflect.ValueOf(dc.Args)
	iV := reflect.Indirect(reflect.ValueOf(i))
	if !iV.CanSet() {
		return fmt.Errorf("type isn't setable: %T", i)
	}
	iV.Set(thisV)
	return nil
}

// MarshalResponse implements the Call interface. In reality, it panics since it
// should never be called.
func (dc DirectCall) MarshalResponse(io.Writer, interface{}) error {
	panic("MarshalResponse should never be called on a DirectCall")
}

// ServeMux wraps multiple method name/Handler pairs and implements a Handler
// which will call the appropriate Handler for the called method, or returns
// ErrMethodNotFound
type ServeMux map[string]Handler

// ServeRPC implements the Handler interface. See the ServeMux type's docstring
func (sm ServeMux) ServeRPC(c Call) interface{} {
	if h, ok := sm[c.GetMethod()]; ok {
		return h.ServeRPC(c)
	}
	return ErrMethodNotFound
}

// TODO some way of creating copies of a Call
