package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"hockey_season/backend/hub"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 45 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WSHandler struct {
	hub    *hub.Hub
	secret string
}

func NewWS(h *hub.Hub, secret string) *WSHandler {
	return &WSHandler{hub: h, secret: secret}
}

func (h *WSHandler) Handle(w http.ResponseWriter, r *http.Request) {
	if _, err := ParseToken(ExtractToken(r), h.secret); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	draftID := r.PathValue("id")
	if _, err := strconv.Atoi(draftID); err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade: %v", err)
		return
	}

	client := &hub.Client{
		DraftID: draftID,
		Conn:    conn,
		Send:    make(chan []byte, 64),
	}
	h.hub.Register(client)

	go writePump(client)
	readPump(client, h.hub)
}

func writePump(c *hub.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func readPump(c *hub.Client, h *hub.Hub) {
	defer func() {
		h.Unregister(c)
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		if _, _, err := c.Conn.ReadMessage(); err != nil {
			return
		}
	}
}
