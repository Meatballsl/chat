package main

import (
	"golang.org/x/net/websocket"
	"iGong/util/log"
	"net/http"
	"iGong/json"
)

// 定义一个conn
type Client struct {
	id     string
	socket *websocket.Conn
	send   chan []byte
}

// 定义一个客户端管理器,管理所有conn ，为什么*Client 要有个*，因为是client的地址。
type ClientManager struct {
	//为什么*Client 要有个*
	clients map[*Client]bool

	//用于广播的配置？
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
			//有哪些是通的conn,把通的conn置true，并发送消息
			m.clients[conn] = true
			jsonMessage, _ := json.Marshal(&Message{Content: "a new socket"})
			m.send(jsonMessage, conn)
		case conn := <-m.unregister:
			//有哪些是不通的,把不通的先关闭
				if _,ok := m.clients[conn];ok{
					//为什么是这样关闭
					close(conn.send)
					//根据key 删除该map的值
					delete(m.clients,conn)
					jsonMessage,_ := json.Marshal()
				}

		}
	}
}
