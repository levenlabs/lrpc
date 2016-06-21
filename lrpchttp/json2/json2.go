// Package json2 implements an lrpchttp.Codec interface for using the JSON RPC2
// protocol
package json2

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/levenlabs/lrpc"

	"golang.org/x/net/context"
)

// ErrorCode is an integer used to identify errors over JSON RPC2
type ErrorCode int

const (
	// ErrParse is used when invalid JSON was received by the server
	ErrParse ErrorCode = -32700
	// ErrInvalidRequest is used when the JSON sent is not a valid Request
	// object
	ErrInvalidRequest ErrorCode = -32600
	// ErrNoMethod is used when the method does not exist / is not available
	ErrNoMethod ErrorCode = -32601
	// ErrInvalidParams is used when invalid method parameters were sent
	ErrInvalidParams ErrorCode = -32602
	// ErrInternal is used when  an internal JSON-rpc error is encountered
	ErrInternal ErrorCode = -32603
	// ErrServer is reserved for implementation-defined server-errors
	ErrServer ErrorCode = -32000
)

// Error is an error implementation which contains additional information used
// by JSON RPC2
type Error struct {
	// Required. A Number that indicates the error type that occurred.
	Code ErrorCode `json:"code"`

	// Required. A string providing a short description of the error. The
	// message SHOULD be limited to a concise single sentence.
	Message string `json:"message"`

	// Optional. A primitive or structured value that contains additional
	// information about the error.
	Data interface{} `json:"data"`
}

func (e *Error) Error() string {
	return e.Message
}

// Response implements a response object for the JSON RPC2 protocol
//
// If being used as part of a client request/response, before unmarshalling the
// response into this struct, be sure to fill Result with a pointer type of the
// expected return.
type Response struct {
	Version string `json:"jsonrpc"`

	// The object that was returned by the invoked method
	Result interface{} `json:"result,omitempty"`

	// An Error object if there was an error invoking the method
	Error *Error `json:"error,omitempty"`

	// This must be the same id as the request it is responding to.
	ID *json.RawMessage `json:"id"`
}

// Request implements a request object for the JSON RPC2 protocol
type Request struct {
	Version string `json:"jsonrpc"`

	// A string containing the name of the method to be invoked.
	Method string `json:"method"`

	// A structured value to pass as arguments to the method.
	Params *json.RawMessage `json:"params"`

	// The request id. MUST be a string, number or null.
	// Our implementation will not do type checking for id.
	// It will be copied as it is.
	ID *json.RawMessage `json:"id"`
}

// Call is an implementation of the lrpc.Call interface for the JSON RPC2
// protocol
type Call struct {
	W http.ResponseWriter `json:"-"`
	R *http.Request       `json:"-"`
	Request
}

// GetContext implements the Call interface
func (c *Call) GetContext() context.Context {
	// TODO use the http.Request's context when 1.7 is stable
	return context.Background()
}

// GetMethod implements the Call interface
func (c *Call) GetMethod() string {
	return c.Method
}

// UnmarshalArgs implements the Call interface
func (c *Call) UnmarshalArgs(i interface{}) error {
	return json.Unmarshal(*c.Params, i)
}

// MarshalResponse implements the Call interface
func (c *Call) MarshalResponse(w io.Writer, i interface{}) error {
	var res Response
	if err, ok := i.(error); ok {
		jerr, ok := i.(*Error)
		if !ok {
			jerr = &Error{Code: ErrServer, Message: err.Error()}
		}
		res.Error = jerr
	} else {
		res.Result = i
	}
	res.Version = "2.0"
	res.ID = c.ID

	c.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(&res)
}

// Codec implements the lrpchttp.Codec interface
//
//	httpHandler := lrpchttp.HTTPHandler(json2.Codec{}, h)
//
type Codec struct{}

// NewCall implements the lrpchttp.Codec interface
func (Codec) NewCall(w http.ResponseWriter, r *http.Request) (lrpc.Call, error) {
	c := &Call{W: w, R: r}
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		return nil, err
	}
	return c, nil
}
