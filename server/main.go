package main

import (
	"fmt"
	"net"
	"strings"
	"time"
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

	// 提示我是谁
	cli.C <- MakeMsg(cli, "I am here")

	isQuit := make(chan bool) //对方是否主动退出
	hasData := make(chan bool) // 判断对方在某段时间内是否有数据

	// 新建协程，接收用户发送来的数据
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if n == 0 { //对方断开，或者出现问题
				isQuit <- true
				fmt.Println("conn read err = ", err)
				return
			}

			msg := string(buf[:n-1])

			if len(msg) == 3 && msg == "who" {
				// 遍历map 给当前用户发送所有成员
				conn.Write([]byte("user list :\n"))

				for _, tmp := range onlineMap {
					msg = tmp.Addr + ":" + tmp.Name + "\n"
					conn.Write([]byte(msg))
				}

			} else if len(msg) >= 8 && msg[:6] == "rename" {
				// rename | mike
				name := strings.Split(msg, "|")[1]
				cli.Name = name
				onlineMap[cliAddr] = cli
				conn.Write([]byte("rename ok\n"))
			} else {
				// 转发此内容
				message <- MakeMsg(cli, msg)
			}

			hasData <- true

		}
	}()

	for {
		// 通过select检测channel流动
		select {
		case <-isQuit:
			delete(onlineMap, cliAddr) // 当前用户从map中移除
			message <- MakeMsg(cli, "login out")
			return

		case <- hasData:

		case <- time.After(3600*time.Second): // 超时时间
			delete(onlineMap, cliAddr) // 当前用户从map中移除
			message <- MakeMsg(cli, "has been leave!!!  because time out!! please be more behavior")
			return

		}
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
	listener, err := net.Listen("tcp", "0.0.0.0:80")
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
