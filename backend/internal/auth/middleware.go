package auth

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func (h Handler) Require(c *gin.Context) {
	token, _ := c.Cookie(cookie)
	if token == "" {
		c.AbortWithStatusJSON(401, gin.H{"error": "Sign in to continue."})
		return
	}
	user, err := h.Store.sessionUser(c.Request.Context(), Fingerprint(token))
	if errors.Is(err, mongo.ErrNoDocuments) {
		c.AbortWithStatusJSON(401, gin.H{"error": "Your session has expired. Sign in again."})
		return
	}
	if err != nil {
		h.internalError(c, err)
		return
	}
	c.Set("creator", user)
	c.Next()
}

func CreatorID(c *gin.Context) bson.ObjectID {
	return c.MustGet("creator").(User).ID
}
