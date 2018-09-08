package goa

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

const (
	addResultNoCommonPrefix uint8 = 0
	addResultSuccess              = 1
	addResultConflict             = 2
)

type node struct {
	static   string
	dynamic  *regexp.Regexp
	handlers []handlerFunc
	children []*node
}

// 新建节点
func newNode(path string, static bool, handlers []handlerFunc) *node {
	var n = &node{handlers: handlers}
	if static {
		n.static = path
	} else if _, complete := regexp.MustCompile(path).LiteralPrefix(); complete {
		n.static = path
	} else {
		n.dynamic = regexp.MustCompile("^" + path)
	}
	return n
}

// 添加到节点
func (n *node) add(path string, static bool, handlers []handlerFunc) uint8 {
	commonPrefix := n.commonPrefix(path, static)
	if len(commonPrefix) == 0 {
		return addResultNoCommonPrefix
	} else
	// 公共前缀比当前节点路径短，则分裂
	if len(commonPrefix) < len(n.static) ||
		n.dynamic != nil && len(commonPrefix) < len(n.dynamic.String())-1 {
		n.split(commonPrefix)
	}
	childPath := path[len(commonPrefix):]
	// 子节点路径为空
	if len(childPath) == 0 {
		if n.handlers == nil {
			n.handlers = handlers
			return addResultSuccess
		} else {
			return addResultConflict
		}
	}
	return n.addToChildren(childPath, static, handlers)
}

func (n *node) addToChildren(path string, static bool, handlers []handlerFunc) uint8 {
	for _, child := range n.children {
		if result := child.add(path, static, handlers); result != addResultNoCommonPrefix {
			return result
		}
	}
	child := newNode(path, static, handlers)
	// 静态路径优先匹配，所以将静态子节点放在动态子节点前边
	if l := len(n.children); l > 0 && len(child.static) > 0 && n.children[l-1].dynamic != nil {
		i := 0
		for ; i < l && len(n.children[i].static) > 0; i++ {
		}
		children := append(make([]*node, 0, l+1), n.children[:i]...)
		children = append(children, child)
		n.children = append(children, n.children[i:]...)
	} else {
		n.children = append(n.children, child)
	}
	return addResultSuccess
}

// 分裂为父节点和子节点
func (n *node) split(path string) {
	var child *node
	if len(n.static) > 0 {
		child = newNode(n.static[len(path):], true, n.handlers)
		n.static = path
	} else {
		child = newNode(n.dynamic.String()[len(path)+1:], false, n.handlers)
		if _, complete := regexp.MustCompile(path).LiteralPrefix(); complete {
			n.static = path
			n.dynamic = nil
		} else {
			n.dynamic = regexp.MustCompile("^" + path)
		}
	}
	child.children = n.children

	n.handlers = nil
	n.children = []*node{child}
}

func (n *node) lookup(path string) (bool, []handlerFunc, []string) {
	commonPrefix, captures := n.lookupCommonPrefix(path)
	if len(commonPrefix) == 0 {
		return false, nil, nil
	}

	childPath := path[len(commonPrefix):]
	if len(childPath) == 0 {
		if len(n.handlers) > 0 {
			return true, n.handlers, captures
		}
	} else if handlers, childCaptures := n.lookupChildren(childPath); len(handlers) > 0 {
		if len(childCaptures) > 0 {
			captures = append(captures, childCaptures...)
		}
		return true, handlers, captures
	}
	return true, nil, nil
}

func (n *node) lookupChildren(childPath string) ([]handlerFunc, []string) {
	for _, child := range n.children {
		if ok, handlers, captures := child.lookup(childPath); ok {
			return handlers, captures
		}
	}
	return nil, nil
}

func (n *node) String() string {
	return n.string("")
}

func (n *node) string(indent string) string {
	var fields []string
	if n.static != "" {
		fields = append(fields, "static: "+n.static)
	}
	if n.dynamic != nil {
		fields = append(fields, "dynamic: "+n.dynamic.String())
	}
	if len(n.handlers) > 0 {
		fields = append(fields, "handlers: "+fmt.Sprint(n.handlers))
	}
	if len(n.children) > 0 {
		var children bytes.Buffer
		for _, child := range n.children {
			children.WriteString(child.string(indent+"  ") + "\n")
		}
		fields = append(fields, fmt.Sprintf("children: [\n%s%s]", children.String(), indent))
	}

	return indent + "{ " + strings.Join(fields, ", ") + " }"
}
