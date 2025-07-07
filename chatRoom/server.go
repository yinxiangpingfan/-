// Server直接连接浏览器，收到的信息传给hub，之后再由hub广播给所有客户端
package chatRoom

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// 加入心跳检测
const (
	// 心跳检测时间间隔
	pongWait = 5 * time.Second
	// 写消息超时时间
	pingPeriod = 3 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有跨域请求，实际开发中需要根据情况进行安全校验
	},
}

// Server用于直接连接浏览器，之后再和hub相互通讯
type Server struct {
	hub *Hub
	//用于存储每个连接的客户端
	conn      *websocket.Conn
	msg       chan []byte
	frontName []byte
}

// read用于读取客户端发来的消息
func (s *Server) Read() {
	defer func() {
		s.hub.Unregister <- s
		s.conn.Close()
	}()
	for {
		//读取消息
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			//如果是异常关闭，打印日志
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			//将用户名从hub中注销
			s.hub.Unregister <- s
			break
		}
		//约定：从浏览器读到的第一条消息代表前端的身份标识，该信息不进行广播
		if len(s.frontName) == 0 {
			s.frontName = message
		} else {
			s.hub.Broadcast <- bytes.Join([][]byte{s.frontName, message}, []byte(":"))
		}
	}
}

func (s *Server) Write() {
	defer func() {
		s.conn.Close()
	}()
	for {
		message, ok := <-s.msg
		if !ok {
			//如果消息通道关闭，发送关闭消息
			s.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		//将消息写入连接
		err := s.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("error: %v", err)
			return
		}
	}
}

func heartBeat(conn *websocket.Conn, pongWait time.Duration, pingPeriod time.Duration) {
	//设置心跳检测时间
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	//定时发送心跳消息
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		conn.Close()
	}()
LOOP:
	for {
		select {
		case <-ticker.C:
			//发送心跳消息
			err := conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				log.Printf("error: %v", err)
				break LOOP
			}
		}
	}
}

func MakeWebsocket(hub *Hub, c *gin.Context) {
	//升级协议
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}
	//加入心跳检测
	go heartBeat(conn, pongWait, pingPeriod)
	//前端每连接一次，就创建一个server
	server := &Server{
		hub:       hub,
		conn:      conn,
		msg:       make(chan []byte),
		frontName: []byte{},
	}
	//将server注册到hub
	hub.Register <- server
	//开启协程，用于读取消息
	go server.Read()
	//开启协程，用于写入消息
	go server.Write()
}
