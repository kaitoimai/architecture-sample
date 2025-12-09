package routing

import (
	"strings"
)

// nodeType はノードの種類を表す
type nodeType int

const (
	staticNode nodeType = iota // 静的パス
	paramNode                   // パラメータ（:id）
	wildcardNode                // ワイルドカード（*）
)

// node はTrie構造のノード
type node struct {
	nodeType nodeType
	segment  string            // パスセグメント（例: "users", ":id"）
	children map[string]*node  // 子ノード
	route    *Route            // このノードに対応するルート
	paramName string           // パラメータ名（:id の場合 "id"）
}

// newNode は新しいノードを作成する
func newNode(segment string) *node {
	n := &node{
		segment:  segment,
		children: make(map[string]*node),
	}

	// ノードタイプを判定
	if strings.HasPrefix(segment, ":") {
		n.nodeType = paramNode
		n.paramName = strings.TrimPrefix(segment, ":")
	} else if segment == "*" || segment == "**" {
		n.nodeType = wildcardNode
	} else {
		n.nodeType = staticNode
	}

	return n
}

// addChild は子ノードを追加する
func (n *node) addChild(segment string) *node {
	if child, exists := n.children[segment]; exists {
		return child
	}

	child := newNode(segment)
	n.children[segment] = child
	return child
}

// getChild は子ノードを取得する（静的マッチング）
func (n *node) getChild(segment string) *node {
	return n.children[segment]
}

// findMatchingChild はセグメントにマッチする子ノードを検索する
// 優先順位: 静的マッチ > パラメータマッチ > ワイルドカード
func (n *node) findMatchingChild(segment string) (*node, bool) {
	// 1. 静的マッチを試行
	if child := n.getChild(segment); child != nil {
		return child, true
	}

	// 2. パラメータノードを検索
	for _, child := range n.children {
		if child.nodeType == paramNode {
			return child, true
		}
	}

	// 3. ワイルドカードノードを検索
	for _, child := range n.children {
		if child.nodeType == wildcardNode {
			return child, true
		}
	}

	return nil, false
}

// SplitPath はパスをセグメントに分割する
func SplitPath(path string) []string {
	// 先頭と末尾のスラッシュを除去
	path = strings.Trim(path, "/")

	if path == "" {
		return []string{}
	}

	return strings.Split(path, "/")
}

// JoinPath はセグメントをパスに結合する
func JoinPath(segments []string) string {
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}
