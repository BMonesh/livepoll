package polls

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"livepoll/internal/auth"
)

const voterCookie = "livepoll_voter"

type Handler struct {
	Store  *Store
	Secure bool
	Notify func(context.Context, string)
}

func (h Handler) Create(c *gin.Context) {
	var input CreateInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(400, gin.H{"error": "Enter a valid question and options."})
		return
	}
	question, options, err := validate(input)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	poll, err := h.Store.create(c.Request.Context(), auth.CreatorID(c), question, options)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(201, poll)
}

func (h Handler) List(c *gin.Context) {
	var before time.Time
	if raw := c.Query("before"); raw != "" {
		var err error
		before, err = time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid page cursor."})
			return
		}
	}
	result, err := h.Store.list(c.Request.Context(), auth.CreatorID(c), before)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, result)
}

func (h Handler) Close(c *gin.Context) {
	ctx := c.Request.Context()
	poll, err := h.Store.closeOwned(ctx, c.Param("slug"), auth.CreatorID(c))
	if err != nil {
		respondError(c, err)
		return
	}
	h.Notify(ctx, poll.Slug)
	c.JSON(200, poll)
}

func voter(c *gin.Context) string {
	raw, _ := c.Cookie(voterCookie)
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(decoded) != 32 {
		return ""
	}
	return raw
}

func (h Handler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	poll, err := h.Store.Find(ctx, c.Param("slug"))
	if err != nil {
		respondError(c, err)
		return
	}
	token := voter(c)
	var selected string
	if token == "" {
		token, err = auth.Token(32)
		if err != nil {
			respondError(c, err)
			return
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(voterCookie, token, 365*24*60*60, "/", "", h.Secure, true)
	} else {
		selected, err = h.Store.selected(ctx, poll.ID, auth.Fingerprint(token))
		if err != nil {
			respondError(c, err)
			return
		}
	}
	c.JSON(200, View{poll, selected})
}

func (h Handler) Vote(c *gin.Context) {
	token := voter(c)
	if token == "" {
		c.JSON(428, gin.H{"error": "Allow cookies and reopen the poll before voting."})
		return
	}
	var input struct {
		OptionID string `json:"optionId"`
	}
	if c.ShouldBindJSON(&input) != nil || len(input.OptionID) != 1 {
		c.JSON(400, gin.H{"error": errChoice.Error()})
		return
	}
	ctx := c.Request.Context()
	result, err := h.Store.vote(ctx, c.Param("slug"), auth.Fingerprint(token), input.OptionID)
	if err != nil {
		respondError(c, err)
		return
	}
	if result.changed {
		h.Notify(ctx, result.Slug)
	}
	c.JSON(200, result.View)
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		c.JSON(404, gin.H{"error": "This poll could not be found."})
	case errors.Is(err, errForbidden):
		c.JSON(403, gin.H{"error": err.Error()})
	case errors.Is(err, errClosed):
		c.JSON(409, gin.H{"error": err.Error()})
	case errors.Is(err, errChoice):
		c.JSON(400, gin.H{"error": err.Error()})
	default:
		slog.Error("poll request failed", "error", err)
		c.JSON(503, gin.H{"error": "Poll service unavailable. Try again shortly."})
	}
}
