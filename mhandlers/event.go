package mhandlers

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pritunl/pritunl-zero/event"
	"time"
)

const (
	writeTimeout = 10 * time.Second
	pingInterval = 30 * time.Second
	pingWait     = 40 * time.Second
)

func eventGet(c *gin.Context) {
	socket := &event.WebSocket{}

	defer func() {
		socket.Close()
		event.WebSocketsLock.Lock()
		event.WebSockets.Remove(socket)
		event.WebSocketsLock.Unlock()
	}()

	event.WebSocketsLock.Lock()
	event.WebSockets.Add(socket)
	event.WebSocketsLock.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	socket.Cancel = cancel

	conn, err := event.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	socket.Conn = conn

	conn.SetReadDeadline(time.Now().Add(pingWait))
	conn.SetPongHandler(func(x string) (err error) {
		conn.SetReadDeadline(time.Now().Add(pingWait))
		return
	})

	lst, err := event.SubscribeListener([]string{"dispatch"})
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	socket.Listener = lst

	ticker := time.NewTicker(pingInterval)
	socket.Ticker = ticker
	sub := lst.Listen()

	go func() {
		for {
			if _, _, err := conn.NextReader(); err != nil {
				conn.Close()
				break
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-sub:
			if !ok {
				conn.WriteControl(websocket.CloseMessage, []byte{},
					time.Now().Add(writeTimeout))
				return
			}

			conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			err = conn.WriteJSON(msg)
			if err != nil {
				return
			}
		case <-ticker.C:
			err = conn.WriteControl(websocket.PingMessage, []byte{},
				time.Now().Add(writeTimeout))
			if err != nil {
				return
			}
		}
	}
}
