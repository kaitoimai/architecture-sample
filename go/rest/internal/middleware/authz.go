package middleware

import (
	"github.com/kaitoimai/go-sample/rest/internal/auth"
	"github.com/kaitoimai/go-sample/rest/internal/pkg/myerrors"
	"github.com/ogen-go/ogen/middleware"
)

// authorizeRoleMap はoperationIdに対する許可ロールのマッピング
// 新しいエンドポイントを追加する際は、ここに必ずマッピングを追加する
var authorizeRoleMap = map[string][]string{
	"v1GetHello": {auth.RoleUser, auth.RoleAdmin}, // user または admin が必要

	// 将来的なエンドポイント追加例:
	// "v1PostItems":   {auth.RoleUser, auth.RoleAdmin},
	// "v1DeleteUsers": {auth.RoleAdmin}, // admin のみ
}

// AuthzMiddleware は Role-Based Access Control (RBAC) による認可を行うミドルウェア
type AuthzMiddleware struct{}

// NewAuthzMiddleware creates a new authorization middleware
func NewAuthzMiddleware() *AuthzMiddleware {
	return &AuthzMiddleware{}
}

// Handle processes the authorization middleware
func (m *AuthzMiddleware) Handle(req middleware.Request, next middleware.Next) (middleware.Response, error) {
	// 全てのAPIリクエストで認可チェックを行う
	// ロールマッピングが定義されていない場合はデフォルト拒否（セキュアバイデフォルト）
	allowedRoles, exists := authorizeRoleMap[req.OperationID]
	if !exists {
		// マッピングがない = エンドポイント追加時のマッピング漏れを防ぐため拒否
		return middleware.Response{}, myerrors.NewForbidden("この操作を実行する権限がありません（ロールマッピング未定義）")
	}

	// Contextからclaimsを取得
	claims := auth.FromContext(req.Context)
	if claims == nil {
		// 認証ミドルウェアを通過していない場合（本来ありえない）
		return middleware.Response{}, myerrors.NewUnauthorized("認証情報が見つかりません")
	}

	// ロールチェック
	if !claims.HasAnyRole(allowedRoles...) {
		return middleware.Response{}, myerrors.NewForbidden("この操作を実行する権限がありません")
	}

	return next(req)
}
