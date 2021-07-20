package xgin

import (
	"net/http"
)

// IRoutes defines all router handle interface.
type IRoutes interface {
	Use(...HandlerFunc) IRoutes

	// Handle(string, string, ...HandlerFunc) IRoutes
	// Any(string, ...HandlerFunc) IRoutes
	GET(string, ...HandlerFunc) IRoutes
	// POST(string, ...HandlerFunc) IRoutes
	// DELETE(string, ...HandlerFunc) IRoutes
	// PATCH(string, ...HandlerFunc) IRoutes
	// PUT(string, ...HandlerFunc) IRoutes
	// OPTIONS(string, ...HandlerFunc) IRoutes
	// HEAD(string, ...HandlerFunc) IRoutes

	// StaticFile(string, string) IRoutes
	// Static(string, string) IRoutes
	// StaticFS(string, http.FileSystem) IRoutes
}

// RouterGroup is used internally to configure router, a RouterGroup is associated with
// a prefix and an array of handlers (middleware).
type RouterGroup struct {
	Handlers HandlersChain
	basePath string
	engine   *Engine
	root     bool
}

func (group *RouterGroup) handle(httpMethod, relativePath string, handlers HandlersChain) IRoutes {
	absolutePath := group.calculateAbsolutePath(relativePath)
	handlers = group.combineHandlers(handlers)
	// 添加handlers
	group.engine.addRoute(httpMethod, absolutePath, handlers)
	return group.returnObj()
}

// GET is a shortcut for router.Handle("GET", path, handle).
func (group *RouterGroup) GET(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodGet, relativePath, handlers)
}

// Use adds middleware to the group, see example code in GitHub.
func (group *RouterGroup) Use(middleware ...HandlerFunc) IRoutes {
	group.Handlers = append(group.Handlers, middleware...) // 修改group.Handlers
	return group.returnObj()
}

func (group *RouterGroup) combineHandlers(handlers HandlersChain) HandlersChain {
	finalSize := len(group.Handlers) + len(handlers)
	mergedHandlers := make(HandlersChain, finalSize)
	copy(mergedHandlers, group.Handlers)
	copy(mergedHandlers[len(group.Handlers):], handlers) // mergedHandlers = group.Handlers + handlers
	return mergedHandlers
}

func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return joinPaths(group.basePath, relativePath)
}

func (group *RouterGroup) returnObj() IRoutes {
	if group.root {
		return group.engine
	}
	return group
}
