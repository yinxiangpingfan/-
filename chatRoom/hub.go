package chatRoom

// hub用于接收server传来的信息与消息,通过管道与server进行相互通讯
type Hub struct {
	//用于广播消息
	Broadcast chan []byte
	//用于注册客户端
	Register chan *Server
	//用于注销客户端
	Unregister chan *Server
	//用于存储客户端
	Servers map[*Server]struct{}
}

// 开启hub
func (h *Hub) Run() {
	for {
		select {
		case server := <-h.Register:
			h.Servers[server] = struct{}{}
		case server := <-h.Unregister:
			if _, ok := h.Servers[server]; ok {
				delete(h.Servers, server)
				close(server.msg)
			}
		case message := <-h.Broadcast:
			for server := range h.Servers {
				select {
				case server.msg <- message:
				default:
					close(server.msg)
					delete(h.Servers, server)
				}
			}
		}
	}
}
