package xgin

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the router.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

type methodTree struct {
	method string
	root   *node
}

type methodTrees []methodTree

func (trees methodTrees) get(method string) *node {
	for _, tree := range trees {
		if tree.method == method {
			return tree.root
		}
	}
	return nil
}

type node struct {
	path      string
	indices   string
	wildChild bool
	// nType     nodeType
	priority uint32
	children []*node // child nodes, at most 1 :param style node at the end of the array
	handlers HandlersChain
	fullPath string
}

func (n *node) addRoute(path string, handlers HandlersChain) {
	// 沿根节点添加子阶段
	n.fullPath = path
	n.handlers = handlers
}

func (n *node) getValue(path string, params *Params) (value nodeValue) {
	// 从根结点向下查找
	value = nodeValue{handlers: n.handlers, fullPath: n.fullPath}
	return
}

type nodeValue struct {
	handlers HandlersChain
	params   *Params
	tsr      bool
	fullPath string
}
