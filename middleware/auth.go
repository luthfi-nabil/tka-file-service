package middleware

import (
	"crypto/rsa"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	RoleID    int16  `json:"role_id"`
	RoleName  string `json:"role_name"`
	SchoolID  string `json:"school_id,omitempty"`
	TeacherID string `json:"teacher_id,omitempty"`
	StudentID string `json:"student_id,omitempty"`
	jwt.RegisteredClaims
}

const claimsKey = "claims"

func GenerateToken(claims *Claims, privateKey *rsa.PrivateKey) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privateKey)
}

func BuildClaims(userID, username string, roleID int16, roleName, schoolID, teacherID, studentID string, expiry time.Duration) *Claims {
	return &Claims{
		UserID:    userID,
		Username:  username,
		RoleID:    roleID,
		RoleName:  roleName,
		SchoolID:  schoolID,
		TeacherID: teacherID,
		StudentID: studentID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		},
	}
}

func RequireAuth(publicKey *rsa.PublicKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization token required"})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return publicKey, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(claimsKey, claims)
		c.Next()
	}
}

func RequireRole(allowedRoles ...int16) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := c.MustGet(claimsKey).(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing auth claims"})
			return
		}

		for _, role := range allowedRoles {
			if claims.RoleID == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

func GetClaims(c *gin.Context) (*Claims, bool) {
	v, exists := c.Get(claimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := v.(*Claims)
	return claims, ok
}
