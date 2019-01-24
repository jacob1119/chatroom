package main

import (
	"fmt"
	"net"
)

type Client struct {
	C    chan string // 发送数据的管道
	Name string      // 用户名
	Addr string      // 网络地址
}

var onlineMap map[string]Client
var message = make(chan string)

func Manager() {
	// 给map分配空间
	onlineMap = make(map[string]Client)
	for {
		msg := <-message // 没有消息时，阻塞

		// 遍历map，给每个成员发消息
		for _, cli := range onlineMap {
			cli.C <- msg
		}

	}
}

func MakeMsg(cli Client, msg string) (buf string) {
	buf = "{" + cli.Addr + "}" + cli.Name + " : " + msg

	return buf
}

func HandleConn(conn net.Conn) {
	defer conn.Close()
	// 获取客户端的网络地址
	cliAddr := conn.RemoteAddr().String()

	// 创建一个结构体
	cli := Client{make(chan string), cliAddr, cliAddr,}

	// 把结构体添加到map
	onlineMap[cliAddr] = cli

	//新开一个协程，给当前客户端发送信息
	go WriteMsgToClient(cli, conn)

	// 广播某人在线
	//message <- "{" + cli.Addr + "}" + cli.Name + ": login "
	message <- MakeMsg(cli, "login")

	for {

	}

}

func WriteMsgToClient(cli Client, conn net.Conn) {
	// 给当前客户端发信息
	for msg := range cli.C {
		conn.Write([]byte(msg + "\n"))
	}
}

func main() {
	// 监听
	listener, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		fmt.Println("net.Listen err = ", err)
		return
	}
	defer listener.Close()

	// 新开协程，用于转发消息,有消息时遍历map，给map每个成员发送消息
	go Manager()

	// 主协程
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener.Accept err = ", err)
			continue
		}

		go HandleConn(conn) // 处理用户连接
	}

}
