package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"livepoll/internal/polls"
)

func (h *Hub) Stream(store *polls.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		slug := c.Param("slug")
		updates, leave := h.watch(slug)
		defer leave()

		initial, err := store.Find(ctx, slug)
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(404, gin.H{"error": "This poll could not be found."})
			return
		}
		if err != nil {
			c.JSON(503, gin.H{"error": "Results are temporarily unavailable."})
			return
		}

		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache, no-transform")
		c.Header("X-Accel-Buffering", "no")
		controller := http.NewResponseController(c.Writer)
		write := func(frame string) bool {
			_ = controller.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if _, err := fmt.Fprint(c.Writer, frame); err != nil {
				return false
			}
			err := controller.Flush()
			_ = controller.SetWriteDeadline(time.Time{})
			return err == nil
		}
		send := func(poll polls.Poll) bool {
			data, err := json.Marshal(poll)
			if err != nil {
				return false
			}
			return write(fmt.Sprintf("id: %d\nevent: results\ndata: %s\n\n", poll.Revision, data))
		}
		if !send(initial) {
			return
		}
		revision := initial.Revision
		heartbeat := time.NewTicker(12 * time.Second)
		defer heartbeat.Stop()

		for {
			tick := false
			select {
			case <-ctx.Done():
				return
			case <-updates:
			case <-heartbeat.C:
				tick = true
			}
			readCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			snapshot, err := store.Find(readCtx, slug)
			cancel()
			if err != nil {
				return
			}
			if snapshot.Revision > revision {
				if !send(snapshot) {
					return
				}
				revision = snapshot.Revision
			} else if tick && !write(": keepalive\n\n") {
				return
			}
		}
	}
}
