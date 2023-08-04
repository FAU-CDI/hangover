// Package htmlx provides facilitirs
package htmlx

import (
	"strings"

	"golang.org/x/net/html"
)

// ReplaceLinks parses source as a html fragment, and then replaces all paths
// inside <a> and <img> elements with the replace function.
func ReplaceLinks(source string, replace func(string) string) string {
	// NOTE(twiesing): we should better define what exactly this means
	// and parse all sorts of other elements
	var builder strings.Builder
	builder.Grow(len(source))

	nodes, err := html.ParseFragment(strings.NewReader(source), nil)
	if err != nil {
		panic(err)
	}

	for _, node := range nodes {
		IterTree(node, func(node *html.Node) {
			// TODO: Do we just want to use the 'href' and 'src' attributes on everything?
			if node.Type == html.ElementNode && node.Data == "a" {
				replaceAttr(node.Attr, "href", replace)
			}
			if node.Type == html.ElementNode && node.Data == "img" {
				replaceAttr(node.Attr, "src", replace)
			}
		})
		html.Render(&builder, node)
	}
	return builder.String()
}

func replaceAttr(attr []html.Attribute, key string, replace func(string) string) {
	for i, a := range attr {
		if a.Key == key {
			attr[i].Val = replace(a.Val)
			break
		}
	}
}

// IterTree calls f recursively for all nodes contained in the tree starting at node.
//
// f is called first recursively for all children (in order), and then for the node itself.
// The argument to f is guaranteed never to be nil.
func IterTree(node *html.Node, f func(node *html.Node)) {
	if node == nil {
		return
	}

	// iterate over all the children
	child := node.FirstChild
	for child != nil {
		IterTree(child, f)
		child = child.NextSibling
	}

	// and then the node itself
	f(node)
}
