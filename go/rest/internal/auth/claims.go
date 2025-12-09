package auth

import "github.com/golang-jwt/jwt/v5"

// Claims represents JWT payload structure
type Claims struct {
	UserID               string `json:"user_id"` // ユーザー識別子
	Role                 string `json:"role"`    // ユーザーロール (admin, user)
	jwt.RegisteredClaims        // 標準クレーム (iss, sub, aud, exp, nbf, iat, jti)
}

// ロール定義
const (
	RoleAdmin = "admin" // 管理者: すべてのAPIにアクセス可能
	RoleUser  = "user"  // 一般ユーザー: 基本的なAPIにアクセス可能
)

// IsValidRole checks if the role is valid
func IsValidRole(role string) bool {
	return role == RoleAdmin || role == RoleUser
}

// HasRole checks if the claims have the specified role
func (c *Claims) HasRole(role string) bool {
	return c.Role == role
}

// HasAnyRole checks if the claims have any of the specified roles
func (c *Claims) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if c.Role == role {
			return true
		}
	}
	return false
}
