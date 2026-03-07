// Package auth provides authentication middleware for the MediLink API.
package auth

import (
"net/http"
"strings"

"github.com/gin-gonic/gin"
"github.com/google/uuid"
)

const (
ActorIDKey           = "actor_id"
ActorRoleKey         = "actor_role"
ActorEmailHashKey    = "actor_email_hash"
ActorOrgIDKey        = "actor_org_id"
ActorTOTPVerifiedKey = "actor_totp_verified"
RequestJTIKey        = "request_jti"
)

// AuthMiddleware validates JWT and sets actor context.
// Replaces the AuthStub from Week 1.
func AuthMiddleware(jwtSvc JWTService) gin.HandlerFunc {
return func(c *gin.Context) {
// Extract token from Authorization: Bearer <token> or X-Auth-Token header
tokenString := ""
authHeader := c.GetHeader("Authorization")
if strings.HasPrefix(authHeader, "Bearer ") {
tokenString = strings.TrimPrefix(authHeader, "Bearer ")
}
if tokenString == "" {
tokenString = c.GetHeader("X-Auth-Token")
}

if tokenString == "" {
c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
"resourceType": "OperationOutcome",
"issue": []gin.H{{"severity": "error", "code": "login", "diagnostics": "Authentication required"}},
})
return
}

// Parse and validate — algorithm MUST be HS256
claims, err := jwtSvc.ValidateAccessToken(tokenString)
if err != nil {
c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
"resourceType": "OperationOutcome",
"issue": []gin.H{{"severity": "error", "code": "login", "diagnostics": "Invalid or expired token"}},
})
return
}

// Check jti blacklist
blacklisted, err := jwtSvc.IsBlacklisted(c.Request.Context(), claims.ID)
if err != nil || blacklisted {
c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
"resourceType": "OperationOutcome",
"issue": []gin.H{{"severity": "error", "code": "login", "diagnostics": "Token has been revoked"}},
})
return
}

// For physicians: check TOTPVerified
if (claims.Role == "physician" || claims.Role == "admin") && !claims.TOTPVerified {
path := c.FullPath()
allowedPaths := []string{
"/auth/login/verify-totp",
"/auth/logout",
"/auth/totp/setup",
"/auth/totp/verify-setup",
}
allowed := false
for _, p := range allowedPaths {
if path == p {
allowed = true
break
}
}
if !allowed {
c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
"resourceType": "OperationOutcome",
"issue": []gin.H{{"severity": "error", "code": "forbidden", "diagnostics": "MFA verification required before accessing this resource"}},
})
return
}
}

// Set actor context — never trust client-supplied IDs
uid, _ := uuid.Parse(claims.UserID)
c.Set(ActorIDKey, uid)
c.Set(ActorRoleKey, claims.Role)
c.Set(ActorOrgIDKey, claims.OrgID)
c.Set(ActorTOTPVerifiedKey, claims.TOTPVerified)
c.Set(RequestJTIKey, claims.ID)

c.Next()
}
}

// RequireRole checks that the authenticated user has one of the specified roles.
func RequireRole(roles ...string) gin.HandlerFunc {
return func(c *gin.Context) {
actorRole := c.GetString(ActorRoleKey)
for _, role := range roles {
if actorRole == role {
c.Next()
return
}
}
c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
"resourceType": "OperationOutcome",
"issue": []gin.H{{"severity": "error", "code": "forbidden", "diagnostics": "Insufficient permissions"}},
})
}
}

// RequirePhysician allows physician and admin roles.
func RequirePhysician() gin.HandlerFunc {
return RequireRole("physician", "admin")
}

// RequireMFA ensures the physician has completed TOTP verification.
func RequireMFA() gin.HandlerFunc {
return func(c *gin.Context) {
actorRole := c.GetString(ActorRoleKey)
if actorRole == "patient" {
c.Next()
return
}
totpVerified, _ := c.Get(ActorTOTPVerifiedKey)
if verified, ok := totpVerified.(bool); ok && verified {
c.Next()
return
}
c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
"resourceType": "OperationOutcome",
"issue": []gin.H{{"severity": "error", "code": "forbidden", "diagnostics": "MFA verification required"}},
})
}
}

// AuthTestMiddleware provides a stub middleware for tests that need
// an authenticated context without a real JWT service.
func AuthTestMiddleware() gin.HandlerFunc {
return func(c *gin.Context) {
c.Set(ActorIDKey, uuid.MustParse("00000000-0000-0000-0000-000000000001"))
c.Set(ActorRoleKey, "admin")
c.Set(ActorEmailHashKey, "stub-email-hash")
c.Set(ActorOrgIDKey, "")
c.Set(ActorTOTPVerifiedKey, true)
c.Set(RequestJTIKey, "test-jti")
c.Next()
}
}

// GetActorID extracts the actor ID from the Gin context.
func GetActorID(c *gin.Context) uuid.UUID {
if id, exists := c.Get(ActorIDKey); exists {
if uid, ok := id.(uuid.UUID); ok {
return uid
}
}
return uuid.Nil
}

// GetActorRole extracts the actor role from the Gin context.
func GetActorRole(c *gin.Context) string {
if role, exists := c.Get(ActorRoleKey); exists {
if r, ok := role.(string); ok {
return r
}
}
return ""
}

// GetActorEmailHash extracts the actor email hash from the Gin context.
func GetActorEmailHash(c *gin.Context) string {
if hash, exists := c.Get(ActorEmailHashKey); exists {
if h, ok := hash.(string); ok {
return h
}
}
return ""
}
