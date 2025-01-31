package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"

	"github.com/celestiaorg/knuu/internal/database/models"
)

const (
	UserTokenDuration = 24 * time.Hour
	UserContextKey    = "user"

	authTokenPrefix         = "Bearer "
	userTokenClaimsUserID   = "user_id"
	userTokenClaimsUsername = "username"
	userTokenClaimsRole     = "role"
	userTokenClaimsExp      = "exp"
)

type Auth struct {
	secretKey string
}

func NewAuth(secretKey string) *Auth {
	return &Auth{
		secretKey: secretKey,
	}
}

func (a *Auth) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := a.getAuthToken(c)
		if token == "" || !a.isValidToken(token) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		user, err := a.getUserFromToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token err: " + err.Error()})
			c.Abort()
			return
		}
		c.Set(UserContextKey, user)
		c.Next()
	}
}

func (a *Auth) RequireRole(requiredRole models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := a.getUserFromToken(a.getAuthToken(c))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		if user.Role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *Auth) GenerateToken(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		userTokenClaimsUserID:   user.ID,
		userTokenClaimsUsername: user.Username,
		userTokenClaimsRole:     user.Role,
		userTokenClaimsExp:      time.Now().Add(UserTokenDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.secretKey))
}

func (a *Auth) getUserFromToken(token string) (*models.User, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.secretKey), nil
	})
	if err != nil {
		return nil, err
	}

	userID, ok := claims[userTokenClaimsUserID].(float64)
	if !ok {
		return nil, errors.New("invalid user ID")
	}
	username, ok := claims[userTokenClaimsUsername].(string)
	if !ok {
		return nil, errors.New("invalid username")
	}
	role, ok := claims[userTokenClaimsRole].(float64)
	if !ok {
		return nil, errors.New("invalid role")
	}

	return &models.User{ID: uint(userID), Username: username, Role: models.UserRole(role)}, nil
}

func (a *Auth) isValidToken(token string) bool {
	claims := &jwt.MapClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(a.secretKey), nil
	})

	if err != nil {
		return false
	}

	return parsedToken.Valid
}

func (a *Auth) getAuthToken(c *gin.Context) string {
	token := c.GetHeader("Authorization")
	if len(token) > len(authTokenPrefix) && token[:len(authTokenPrefix)] == authTokenPrefix {
		token = token[len(authTokenPrefix):]
	}
	return token
}
