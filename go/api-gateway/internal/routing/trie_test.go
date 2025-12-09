package routing

import (
	"reflect"
	"testing"
)

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "simple path",
			path: "/api/v1/users",
			want: []string{"api", "v1", "users"},
		},
		{
			name: "root path",
			path: "/",
			want: []string{},
		},
		{
			name: "path with trailing slash",
			path: "/api/v1/users/",
			want: []string{"api", "v1", "users"},
		},
		{
			name: "path without leading slash",
			path: "api/v1/users",
			want: []string{"api", "v1", "users"},
		},
		{
			name: "empty path",
			path: "",
			want: []string{},
		},
		{
			name: "path with parameter",
			path: "/api/v1/users/:id",
			want: []string{"api", "v1", "users", ":id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitPath(tt.path)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name     string
		segments []string
		want     string
	}{
		{
			name:     "simple segments",
			segments: []string{"api", "v1", "users"},
			want:     "/api/v1/users",
		},
		{
			name:     "empty segments",
			segments: []string{},
			want:     "/",
		},
		{
			name:     "single segment",
			segments: []string{"users"},
			want:     "/users",
		},
		{
			name:     "segments with parameter",
			segments: []string{"api", "v1", "users", ":id"},
			want:     "/api/v1/users/:id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinPath(tt.segments)
			if got != tt.want {
				t.Errorf("JoinPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewNode(t *testing.T) {
	tests := []struct {
		name         string
		segment      string
		wantNodeType nodeType
		wantParam    string
	}{
		{
			name:         "static node",
			segment:      "users",
			wantNodeType: staticNode,
			wantParam:    "",
		},
		{
			name:         "parameter node",
			segment:      ":id",
			wantNodeType: paramNode,
			wantParam:    "id",
		},
		{
			name:         "wildcard node",
			segment:      "*",
			wantNodeType: wildcardNode,
			wantParam:    "",
		},
		{
			name:         "double wildcard node",
			segment:      "**",
			wantNodeType: wildcardNode,
			wantParam:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := newNode(tt.segment)

			if node.nodeType != tt.wantNodeType {
				t.Errorf("nodeType = %v, want %v", node.nodeType, tt.wantNodeType)
			}

			if node.paramName != tt.wantParam {
				t.Errorf("paramName = %s, want %s", node.paramName, tt.wantParam)
			}

			if node.segment != tt.segment {
				t.Errorf("segment = %s, want %s", node.segment, tt.segment)
			}
		})
	}
}

func TestNodeAddChild(t *testing.T) {
	parent := newNode("api")

	// 子ノードを追加
	child1 := parent.addChild("v1")
	if child1 == nil {
		t.Fatal("addChild() returned nil")
	}

	// 同じセグメントで再度追加すると、同じノードが返る
	child2 := parent.addChild("v1")
	if child1 != child2 {
		t.Error("addChild() should return the same node for duplicate segment")
	}

	// 異なるセグメントを追加
	child3 := parent.addChild("v2")
	if child3 == nil {
		t.Fatal("addChild() returned nil")
	}

	if child1 == child3 {
		t.Error("addChild() should return different nodes for different segments")
	}

	// 子ノードの数を確認
	if len(parent.children) != 2 {
		t.Errorf("parent has %d children, want 2", len(parent.children))
	}
}

func TestNodeGetChild(t *testing.T) {
	parent := newNode("api")
	child := parent.addChild("v1")

	tests := []struct {
		name    string
		segment string
		wantNil bool
	}{
		{
			name:    "existing child",
			segment: "v1",
			wantNil: false,
		},
		{
			name:    "non-existing child",
			segment: "v2",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parent.getChild(tt.segment)

			if tt.wantNil {
				if got != nil {
					t.Error("getChild() should return nil for non-existing child")
				}
			} else {
				if got == nil {
					t.Fatal("getChild() returned nil for existing child")
				}
				if got != child {
					t.Error("getChild() returned wrong child")
				}
			}
		})
	}
}

func TestNodeFindMatchingChild(t *testing.T) {
	parent := newNode("api")

	// 静的ノードを追加
	staticChild := parent.addChild("users")

	// パラメータノードを追加
	paramChild := parent.addChild(":id")

	// ワイルドカードノードを追加
	_ = parent.addChild("*")

	tests := []struct {
		name      string
		segment   string
		wantChild *node
		wantFound bool
	}{
		{
			name:      "static match",
			segment:   "users",
			wantChild: staticChild,
			wantFound: true,
		},
		{
			name:      "parameter match",
			segment:   "123",
			wantChild: paramChild,
			wantFound: true,
		},
		{
			name:      "no match without param/wildcard",
			segment:   "orders",
			wantChild: paramChild, // パラメータノードがマッチする
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotChild, gotFound := parent.findMatchingChild(tt.segment)

			if gotFound != tt.wantFound {
				t.Errorf("findMatchingChild() found = %v, want %v", gotFound, tt.wantFound)
			}

			if tt.wantFound {
				if gotChild == nil {
					t.Fatal("findMatchingChild() returned nil child")
				}

				// 静的マッチの場合は完全一致を確認
				if tt.segment == "users" && gotChild != staticChild {
					t.Error("findMatchingChild() should prioritize static match")
				}

				// パラメータマッチの確認（静的マッチがない場合）
				if tt.segment != "users" && gotChild.nodeType != paramNode && gotChild.nodeType != wildcardNode {
					t.Errorf("findMatchingChild() returned node type %v, want paramNode or wildcardNode", gotChild.nodeType)
				}
			}
		})
	}
}

func TestNodeFindMatchingChild_Priority(t *testing.T) {
	// 優先順位のテスト: 静的 > パラメータ > ワイルドカード
	parent := newNode("api")

	// すべての種類のノードを追加
	paramChild := parent.addChild(":id")
	_ = parent.addChild("*") // wildcardChild
	staticChild := parent.addChild("users")

	// 静的マッチが最優先
	child, found := parent.findMatchingChild("users")
	if !found {
		t.Fatal("findMatchingChild() should find static match")
	}
	if child != staticChild {
		t.Error("findMatchingChild() should prioritize static match over param and wildcard")
	}

	// 静的マッチがない場合はパラメータマッチ
	child, found = parent.findMatchingChild("123")
	if !found {
		t.Fatal("findMatchingChild() should find param match")
	}
	if child != paramChild {
		t.Error("findMatchingChild() should prioritize param match over wildcard")
	}

	// パラメータノードがない場合のテスト
	parent2 := newNode("api")
	wildcard2 := parent2.addChild("*")

	child2, found2 := parent2.findMatchingChild("anything")
	if !found2 {
		t.Fatal("findMatchingChild() should find wildcard match")
	}
	if child2 != wildcard2 {
		t.Error("findMatchingChild() should match wildcard when no static or param")
	}
}
