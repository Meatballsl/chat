package main

import (

	"iGong/util/log"
	"net/http"
	"encoding/json"
    "github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

// 定义一个conn
type Client struct {
	id     string
	socket *websocket.Conn
	send   chan []byte
}

// 定义一个客户端管理器,管理所有conn ，为什么*Client 要有个*，因为是client的地址。
type ClientManager struct {
	//为什么*Client 要有个*,如何给clients 赋值
	clients map[*Client]bool

	//里面装广播的内容
	broadcast chan []byte

	//chan 里面的都是client的地址
	register   chan *Client
	unregister chan *Client
}

//为什么消息体里面的变量要用大写
type Message struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Content   string `json:"content"`
}

var manager = ClientManager{
	clients:   make(map[*Client]bool),
	broadcast: make(chan []byte),


	register:   make(chan *Client),
	unregister: make(chan *Client),
}

func main() {

	log.Info("start server")

	//这是干嘛
	go manager.start()

	//客户端调用ws，会走wsPage
	http.HandleFunc("ws", wsPage)
	http.ListenAndServe(":9999", nil)
}

func (m *ClientManager) start() {

	for {
		select {
		case conn := <-m.register:
			//有新的conn来了，给其他人发通知说有新朋友来了(manager.clients[该conn]为true
			m.clients[conn] = true
			//这边为什么要用&
			jsonMessage, _ := json.Marshal(&Message{Content: "a new friend come"})
			m.send(jsonMessage, conn)
		case conn := <-m.unregister:
			//有人走了
			if _, ok := m.clients[conn]; ok {
				//关闭chan通道
				close(conn.send)
				//根据key 删除该map的值
				delete(m.clients, conn)
				jsonMessage, _ := json.Marshal(&Message{Content: "socket disconnect"})
				m.send(jsonMessage,conn)
			}
		case message := <-m.broadcast:
			//如果某个chan 已经关闭了，再往里写数据，会panic,不会发生这种情况
			for conn := range m.clients {
				select {
				case conn.send <- message:
				default:
					close(conn.send)
					delete(m.clients, conn)
				}

			}

		}
	}
}



func wsPage(res http.ResponseWriter,req *http.Request)  {
	//这个代码看不懂
	conn,error := (&websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		}}).Upgrade(res,req,nil)
	if error != nil {
		http.NotFound(res,req)
		log.Info(error)
	}
	idr,_ := uuid.NewV4()
	//有conn来，new client，并取该对象的地址赋值给manager.register
	client := &Client{id:idr.String(),socket:conn,send:make(chan []byte)}
	manager.register <- client

	go client.read()
	go client.write()


}



//给manager 里面除了自己的conn 的send chan 里装信息
func (m *ClientManager) send(msg []byte,conn *Client) {
	for conn := range m.clients {
		if conn != conn {
			conn.send <- msg
		}
	}
}


//从 conn里面读出数据,放到broadcast,再放到各个client的send chan 里面
func (c *Client) read (){
	defer func() {
		manager.unregister<-c
		c.socket.Close()
	}()
	for {
		_,msg,err := c.socket.ReadMessage()
		if err != nil {
			manager.unregister <- c
			c.socket.Close()
			break
		}

		json.Marshal(Message{Sender:c.id,Content:string(msg)})
		manager.broadcast<-msg
	}

}


//write 从 conn的send 里面获取数据 并发送
func (c *Client) write() {

	defer func() {
		c.socket.Close()
	}()

	for{
		select {
		case msg,ok := <-c.send:
			//这边为啥要用select呢，是怕当chan已经关闭了导致的阻塞吗。如果chan关闭了，ok 就为 false 吗？
			if !ok{
				c.socket.WriteMessage(websocket.CloseMessage,[]byte{})
			}
			c.socket.WriteMessage(websocket.TextMessage,msg)

		}

	}
}