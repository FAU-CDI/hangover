// Package htmlx provides facilities for various html editing
package htmlx

import (
	"fmt"
	"iter"
	"strings"

	"golang.org/x/net/html"
)

// cspell:words htmlx twiesing

// ReplaceLinks parses source as a html fragment, and then replaces all paths
// inside <a> and <img> elements with the replace function.
func ReplaceLinks(source string, replace func(string) string) (string, error) {
	// NOTE(twiesing): we should better define what exactly this means
	// and parse all sorts of other elements
	var builder strings.Builder
	builder.Grow(len(source))

	nodes, err := html.ParseFragment(strings.NewReader(source), nil)
	if err != nil {
		return "", fmt.Errorf("failed to parse html fragment: %w", err)
	}

	for _, node := range nodes {
		for node := range IterTree(node) {
			// TODO: Do we just want to use the 'href' and 'src' attributes on everything?
			if node.Type == html.ElementNode && node.Data == "a" {
				replaceAttr(node.Attr, "href", replace)
			}
			if node.Type == html.ElementNode && node.Data == "img" {
				replaceAttr(node.Attr, "src", replace)
			}
		}
		if err := html.Render(&builder, node); err != nil {
			return "", fmt.Errorf("failed to render node: %w", err)
		}
	}
	return builder.String(), nil
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
func IterTree(node *html.Node) iter.Seq[*html.Node] {
	return func(yield func(*html.Node) bool) {
		iterTree(node, yield)
	}
}

func iterTree(node *html.Node, f func(node *html.Node) bool) bool {
	if node == nil {
		return true
	}

	// iterate over all the children
	child := node.FirstChild
	for child != nil {
		if !iterTree(child, f) {
			return false
		}
		child = child.NextSibling
	}

	// and then the node itself
	return f(node)
}
