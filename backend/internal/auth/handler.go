package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const cookie = "livepoll_session"

type Handler struct {
	Store  *Store
	Secure bool
}

func (h Handler) cookie(c *gin.Context, token string, age int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cookie, token, age, "/", "", h.Secure, true)
}

func (h Handler) signIn(c *gin.Context, signup bool) {
	var input Credentials
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(400, gin.H{"error": "Enter your email and password."})
		return
	}
	ctx := c.Request.Context()
	var user User
	var err error
	if signup {
		user, err = h.Store.signup(ctx, input)
	} else {
		user, err = h.Store.login(ctx, input)
	}
	if err != nil {
		switch {
		case errors.Is(err, errInvalid):
			c.JSON(400, gin.H{"error": err.Error()})
		case errors.Is(err, errCredentials):
			c.JSON(401, gin.H{"error": err.Error()})
		case errors.Is(err, errEmailTaken):
			c.JSON(409, gin.H{"error": err.Error()})
		default:
			h.internalError(c, err)
		}
		return
	}
	previous, _ := c.Cookie(cookie)
	token, err := h.Store.startSession(ctx, user, previous)
	if err != nil {
		h.internalError(c, err)
		return
	}
	h.cookie(c, token, int(lifetime.Seconds()))
	status := 200
	if signup {
		status = 201
	}
	c.JSON(status, user)
}

func (h Handler) Signup(c *gin.Context) { h.signIn(c, true) }
func (h Handler) Login(c *gin.Context)  { h.signIn(c, false) }

func (h Handler) Logout(c *gin.Context) {
	if token, _ := c.Cookie(cookie); token != "" {
		if _, err := h.Store.sessions.DeleteOne(c.Request.Context(), bson.M{"_id": Fingerprint(token)}); err != nil {
			h.internalError(c, err)
			return
		}
	}
	h.cookie(c, "", -1)
	c.Status(204)
}

func (h Handler) Me(c *gin.Context) { c.JSON(200, c.MustGet("creator")) }

func (h Handler) internalError(c *gin.Context, err error) {
	slog.Error("authentication failed", "error", err)
	c.AbortWithStatusJSON(503, gin.H{"error": "Account service unavailable. Try again shortly."})
}
