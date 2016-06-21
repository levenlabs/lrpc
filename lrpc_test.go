package lrpc

import (
	. "testing"

	"github.com/stretchr/testify/assert"
)

type echoRes struct {
	Method string
	Args   interface{}
}

// echo is a simple rpc method which simply returns its arguments. It doesn't
// even care about the method name
var echo = HandlerFunc(func(c Call) interface{} {
	var in interface{}
	if err := c.UnmarshalArgs(&in); err != nil {
		return err
	}
	return echoRes{
		Method: c.GetMethod(),
		Args:   in,
	}
})

var mux = ServeMux{
	"Echo": echo,
}

func TestDirectCall(t *T) {
	assertEcho := func(args interface{}) {
		dc := DirectCall{Method: "Echo", Args: args}
		res := echo.ServeRPC(dc)
		assert.Equal(t, echoRes{Method: "Echo", Args: args}, res)
	}

	assertEcho(true)
	assertEcho(1)
	assertEcho("foo")
	assertEcho([]int{1, 2, 3})
	assertEcho(map[int]string{1: "one", 2: "two"})
	assertEcho(struct{ a, b int }{1, 2})
	assertEcho(&struct{ a, b int }{1, 2})
}

func TestServeMux(t *T) {
	dc := DirectCall{Method: "Echo", Args: true}
	assert.Equal(t, echoRes{Method: "Echo", Args: true}, mux.ServeRPC(dc))
	dc.Method = "wat"
	assert.Equal(t, ErrMethodNotFound, mux.ServeRPC(dc))
}

// Test that rpc works over HTTP
//func TestHTTPRPC(t *T) {
//	httpHandler := HTTPHandler(JSONrpc2Codec{}, mux)
//
//	body := bytes.NewBufferString(`{"method":"Echo","params":{"hello":"world"}} `)
//	r, err := http.NewRequest("GET", "/", body)
//	require.Nil(t, err)
//	w := httptest.NewRecorder()
//
//	httpHandler.ServeHTTP(w, r)
//	var res JSONrpc2Response
//	res.Result = &json.RawMessage{}
//	require.Nil(t, json.NewDecoder(w.Body).Decode(&res))
//	assert.Equal(t, `{"hello":"world"}`, string(*(res.Result.(*json.RawMessage))))
//}
