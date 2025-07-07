package main

import (
	"ChatRoom/chatRoom"
	"net/http"

	"github.com/gin-gonic/gin"
)

//用gin+websocket实现聊天室

func main() {
	r := gin.Default()
	// 加载HTML模板
	r.LoadHTMLGlob("*.html")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	//开启聊天室
	hub := chatRoom.Hub{
		Broadcast:  make(chan []byte),
		Register:   make(chan *chatRoom.Server),
		Unregister: make(chan *chatRoom.Server),
		Servers:    make(map[*chatRoom.Server]struct{}),
	}
	go hub.Run()

	//开启websocket路由
	r.GET("/ws", func(c *gin.Context) {
		chatRoom.MakeWebsocket(&hub, c)
	})

	r.Run(":8080")
}
