package xgin

import (
	"log"
	"net/http"
	"sync"
)

// HandlerFunc defines the handler used by gin middleware as return value.
type HandlerFunc func(*Context)

// HandlersChain defines a HandlerFunc array.
type HandlersChain []HandlerFunc

// Last returns the last handler in the chain. ie. the last handler is the main one.
func (c HandlersChain) Last() HandlerFunc {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}

// Engine is the framework's instance, it contains the muxer, middleware and configuration settings.
// Create an instance of Engine, by using New() or Default()
type Engine struct {
	RouterGroup

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 307 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the router tries to fix the current request path, if no
	// handle is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a handle can be found for this route, the router makes a redirection
	// to the corrected path with status code 301 for GET requests and 307 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.
	RedirectFixedPath bool

	// If enabled, the router checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool

	// If enabled, client IP will be parsed from the request's headers that
	// match those stored at `(*gin.Engine).RemoteIPHeaders`. If no IP was
	// fetched, it falls back to the IP obtained from
	// `(*gin.Context).Request.RemoteAddr`.
	ForwardedByClientIP bool

	// DEPRECATED: USE `TrustedPlatform` WITH VALUE `gin.GoogleAppEngine` INSTEAD
	// #726 #755 If enabled, it will trust some headers starting with
	// 'X-AppEngine...' for better integration with that PaaS.
	AppEngine bool

	// If enabled, the url.RawPath will be used to find parameters.
	UseRawPath bool

	// If true, the path value will be unescaped.
	// If UseRawPath is false (by default), the UnescapePathValues effectively is true,
	// as url.Path gonna be used, which is already unescaped.
	UnescapePathValues bool

	// RemoveExtraSlash a parameter can be parsed from the URL even with extra slashes.
	// See the PR #1817 and issue #1644
	RemoveExtraSlash bool

	// List of headers used to obtain the client IP when
	// `(*gin.Engine).ForwardedByClientIP` is `true` and
	// `(*gin.Context).Request.RemoteAddr` is matched by at least one of the
	// network origins of `(*gin.Engine).TrustedProxies`.
	RemoteIPHeaders []string

	// List of network origins (IPv4 addresses, IPv4 CIDRs, IPv6 addresses or
	// IPv6 CIDRs) from which to trust request's headers that contain
	// alternative client IP when `(*gin.Engine).ForwardedByClientIP` is
	// `true`.
	TrustedProxies []string

	// If set to a constant of value gin.Platform*, trusts the headers set by
	// that platform, for example to determine the client IP
	TrustedPlatform string

	// Value of 'maxMemory' param that is given to http.Request's ParseMultipartForm
	// method call.
	MaxMultipartMemory int64

	// delims           render.Delims
	// secureJSONPrefix string
	// HTMLRender       render.HTMLRender
	// FuncMap          template.FuncMap
	// allNoRoute       HandlersChain
	// allNoMethod      HandlersChain
	// noRoute          HandlersChain
	// noMethod         HandlersChain
	pool      sync.Pool
	trees     methodTrees
	maxParams uint16
	// trustedCIDRs     []*net.IPNet
}

var _ IRoutes = &Engine{}

func New() *Engine {
	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
	}
	engine.RouterGroup.engine = engine
	engine.pool.New = func() interface{} {
		// 复用Context对象
		return engine.allocateContext()
	}
	return engine
}

func Default() *Engine {
	engine := New()
	// engine.Use(Logger(), Recovery())
	return engine
}

func (engine *Engine) allocateContext() *Context {
	v := make(Params, 0, engine.maxParams) // maxParams值是会变的
	return &Context{engine: engine, params: &v}
}

func (engine *Engine) addRoute(method, path string, handlers HandlersChain) {
	root := engine.trees.get(method)
	if root == nil {
		root = new(node)
		root.fullPath = "/"
		engine.trees = append(engine.trees, methodTree{method: method, root: root})
	}
	root.addRoute(path, handlers)

	// Update maxParams
	// if paramsCount := countParams(path); paramsCount > engine.maxParams {
	// 	engine.maxParams = paramsCount
	// }
}

// Run attaches the router to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) Run(addr ...string) (err error) {
	address := ":8080"
	log.Printf("Listening and serving HTTP on %s\n", address)
	err = http.ListenAndServe(address, engine)
	return
}

// ServeHTTP conforms to the http.Handler interface.
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := engine.pool.Get().(*Context)
	c.writermem.reset(w)
	c.Request = req
	c.reset()

	engine.handleHTTPRequest(c)

	engine.pool.Put(c)
}

func (engine *Engine) handleHTTPRequest(c *Context) {
	httpMethod := c.Request.Method
	rPath := c.Request.URL.Path

	// Find root of the tree for the given HTTP method
	t := engine.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method != httpMethod {
			continue
		}
		root := t[i].root
		// Find route in tree
		value := root.getValue(rPath, c.params)
		if value.params != nil {
			c.Params = *value.params
		}
		if value.handlers != nil {
			// 找到handlers, 赋给context, 执行context.Next
			c.handlers = value.handlers
			c.fullPath = value.fullPath
			c.Next()
			c.writermem.WriteHeaderNow()
			return
		}
	}

}
