package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"livepoll/internal/auth"
	"livepoll/internal/config"
	"livepoll/internal/polls"
	"livepoll/internal/realtime"
)

var slugPattern = regexp.MustCompile("^[A-Za-z0-9_-]{22}$")

var incrementLimit = redis.NewScript(`
 local count = redis.call('INCR', KEYS[1])
 if count == 1 then
  redis.call('EXPIRE', KEYS[1], ARGV[1])
 end
 return count
`)

func rateLimit(client *redis.Client, group string, limit int64, seconds int, requireRedis bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Store a short-lived digest rather than an IP string.
		hashed := sha256.Sum256([]byte(c.ClientIP()))
		key := "livepoll:limit:" + group + ":" + hex.EncodeToString(hashed[:])
		ctx, cancel := context.WithTimeout(c.Request.Context(), 400*time.Millisecond)
		count, err := incrementLimit.Run(ctx, client, []string{key}, seconds).Int64()
		cancel()
		if err != nil && requireRedis {
			c.AbortWithStatusJSON(503, gin.H{"error": "Try again in a moment."})
			return
		}
		if err == nil && count > limit {
			c.Header("Retry-After", fmt.Sprint(seconds))
			c.AbortWithStatusJSON(429, gin.H{"error": "Too many requests. Try again in a moment."})
			return
		}
		c.Next()
	}
}

func apiPolicy(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if c.Request.Method == http.MethodPost {
			requestOrigin := c.GetHeader("Origin")
			if (requestOrigin != "" && requestOrigin != origin) || c.GetHeader("Sec-Fetch-Site") == "cross-site" {
				c.AbortWithStatusJSON(403, gin.H{"error": "This request must come from the polling app."})
				return
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
		}
		c.Next()
	}
}

func pollSlug(c *gin.Context) {
	if !slugPattern.MatchString(c.Param("slug")) {
		c.AbortWithStatusJSON(404, gin.H{"error": "This poll could not be found."})
		return
	}
	c.Next()
}

func Router(cfg config.Config, accounts *auth.Store, pollStore *polls.Store, hub *realtime.Hub, redisClient *redis.Client, mongoClient *mongo.Client) (*gin.Engine, error) {
	router := gin.New()
	router.Use(gin.Recovery(), func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("X-Robots-Tag", "noindex, nofollow")
		c.Next()
	})
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	a := auth.Handler{Store: accounts, Secure: cfg.Secure}
	p := polls.Handler{Store: pollStore, Secure: cfg.Secure, Notify: hub.Notify}
	router.GET("/healthz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if mongoClient.Ping(ctx, nil) != nil || redisClient.Ping(ctx).Err() != nil {
			c.JSON(503, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})
	api := router.Group("/api", apiPolicy(cfg.Origin))
	api.POST("/auth/signup", rateLimit(redisClient, "signup", 15, 60, true), a.Signup)
	api.POST("/auth/login", rateLimit(redisClient, "login", 25, 60, true), a.Login)
	api.POST("/auth/logout", a.Logout)
	api.GET("/auth/me", a.Require, a.Me)

	creator := api.Group("/creator/polls", a.Require)
	creator.GET("", p.List)
	creator.POST("", rateLimit(redisClient, "create", 20, 60, true), p.Create)
	creator.POST("/:slug/close", pollSlug, p.Close)

	public := api.Group("/polls/:slug", pollSlug)
	public.GET("", p.Get)
	public.POST("/vote", rateLimit(redisClient, "vote", 180, 60, false), p.Vote)
	public.GET("/events", hub.Stream(pollStore))

	if cfg.StaticDir != "" {
		router.Static("/assets", filepath.Join(cfg.StaticDir, "assets"))
	}
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if cfg.StaticDir != "" && c.Request.Method == http.MethodGet &&
			(path == "/" || path == "/login" || path == "/signup" || path == "/dashboard" || path == "/polls/new" || strings.HasPrefix(path, "/p/")) {
			c.Header("Cache-Control", "no-cache")
			c.File(filepath.Join(cfg.StaticDir, "index.html"))
			return
		}
		c.JSON(404, gin.H{"error": "Page not found."})
	})
	return router, nil
}
