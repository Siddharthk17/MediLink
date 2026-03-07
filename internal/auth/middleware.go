// Package auth provides authentication middleware for the MediLink API.
package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ActorIDKey is the context key for the authenticated user's ID.
const ActorIDKey = "actor_id"

// ActorRoleKey is the context key for the authenticated user's role.
const ActorRoleKey = "actor_role"

// ActorEmailHashKey is the context key for the authenticated user's email hash.
const ActorEmailHashKey = "actor_email_hash"

// stubActorID is used during Week 1 before real JWT validation is implemented.
var stubActorID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// AuthMiddleware is a scaffold for JWT validation.
// In Week 1, it sets a stub actor ID on every request.
// Real JWT validation will be implemented in Week 3.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Week 1 stub: set a fixed actor ID for all requests.
		// This ensures the service layer always has an actorID available.
		c.Set(ActorIDKey, stubActorID)
		c.Set(ActorRoleKey, "admin")
		c.Set(ActorEmailHashKey, "stub-email-hash")
		c.Next()
	}
}

// GetActorID extracts the actor ID from the Gin context.
func GetActorID(c *gin.Context) uuid.UUID {
	if id, exists := c.Get(ActorIDKey); exists {
		return id.(uuid.UUID)
	}
	return uuid.Nil
}

// GetActorRole extracts the actor role from the Gin context.
func GetActorRole(c *gin.Context) string {
	if role, exists := c.Get(ActorRoleKey); exists {
		return role.(string)
	}
	return ""
}

// GetActorEmailHash extracts the actor email hash from the Gin context.
func GetActorEmailHash(c *gin.Context) string {
	if hash, exists := c.Get(ActorEmailHashKey); exists {
		return hash.(string)
	}
	return ""
}
