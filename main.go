// Copyright wasilibs authors
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"strconv"
	"sync"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"

	_ "github.com/wasilibs/nottinygc"
)

func main() {
	proxywasm.SetVMContext(&vm{})
}

type vm struct {
	types.DefaultVMContext
}

func (v *vm) NewPluginContext(contextID uint32) types.PluginContext {
	return &plugin{}
}

type plugin struct {
	// Embed the default plugin context here,
	// so that we don't need to reimplement all the methods.
	types.DefaultPluginContext

	size int
}

var bufPool sync.Pool

// OnPluginStart Override types.DefaultPluginContext.
func (h *plugin) OnPluginStart(_ int) types.OnPluginStartStatus {
	data, err := proxywasm.GetPluginConfiguration()
	if err != nil {
		panic(err)
	}
	sz, err := strconv.Atoi(string(bytes.TrimSpace(data)))
	if err != nil {
		panic(err)
	}
	h.size = sz

	bufPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, sz)
		},
	}

	return types.OnPluginStartStatusOK
}

// NewHttpContext Override types.DefaultPluginContext to allow us to declare a request handler for each
// intercepted request the Envoy Sidecar sends us
func (h *plugin) NewHttpContext(_ uint32) types.HttpContext {
	return &tester{size: h.size}
}

type tester struct {
	types.DefaultHttpContext
	size int
}

func (c *tester) OnHttpRequestHeaders(numHeaders int, endOfStream bool) types.Action {
	b := bufPool.Get().([]byte)
	proxywasm.LogInfof("alloc success, point address: %p", b)
	proxywasm.SendHttpResponse(200, nil, nil, -1)

	// Re-slice to maximum capacity and return it  for re-use. This is important to guarantee that
	// all calls to Get() will return a buffer of length c.size.
	bufPool.Put(b[:c.size])

	return types.ActionContinue
}
