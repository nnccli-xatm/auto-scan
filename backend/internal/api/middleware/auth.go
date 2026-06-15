// JWT认证中间件
// 提供Token生成、验证、刷新功能

package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWT配置
const (
	TokenExpireDuration = time.Hour * 24 * 7 // Token有效期7天
	TokenIssuer         = "auto-scan"
	TokenAudience       = "auto-scan-api"
)

// 自定义Claims
type CustomClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// JWTAuth JWT认证中间件
func JWTAuth(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Header获取Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401001,
				"message": "missing authorization header",
			})
			c.Abort()
			return
		}

		// 解析Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401002,
				"message": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		// 验证Token
		claims, err := ParseToken(parts[1], secretKey)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401003,
				"message": "invalid or expired token",
				"error":   err.Error(),
			})
			c.Abort()
			return
		}

		// 将用户信息存入Context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("claims", claims)

		c.Next()
	}
}

// GenerateToken 生成JWT Token
func GenerateToken(userID, username, role, secretKey string) (string, error) {
	claims := CustomClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpireDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    TokenIssuer,
			Audience:  jwt.ClaimStrings{TokenAudience},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// ParseToken 解析JWT Token
func ParseToken(tokenString, secretKey string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// RefreshToken 刷新Token
func RefreshToken(tokenString, secretKey string) (string, error) {
	claims, err := ParseToken(tokenString, secretKey)
	if err != nil {
		return "", err
	}

	// 生成新Token
	return GenerateToken(claims.UserID, claims.Username, claims.Role, secretKey)
}

// GetCurrentUser 获取当前登录用户
func GetCurrentUser(c *gin.Context) (*CustomClaims, bool) {
	claims, exists := c.Get("claims")
	if !exists {
		return nil, false
	}

	customClaims, ok := claims.(*CustomClaims)
	return customClaims, ok
}

// OptionalAuth 可选认证（未登录也可以访问，但登录后可以获取用户信息）
func OptionalAuth(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		claims, err := ParseToken(parts[1], secretKey)
		if err != nil {
			c.Next()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("claims", claims)

		c.Next()
	}
}