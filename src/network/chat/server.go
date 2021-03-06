package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// 创建用户结构体类型
type Client struct {
	C chan string
	Name string
	Addr string
}

// 创建全局map在线用户, key:string, value:Client
var onlineMap map[string]Client


// 创建全局 channel，用于传递用户消息
var message = make(chan string)

// 错误输出信息
func outputError(msg string) {
	fmt.Println("msg")
}

func Manager() {
	// 初始化 onlineMap
	onlineMap = make(map[string]Client)

	// 监听全局channel是否有数据
	for {
		// 有数据存储到 msg，无数据阻塞
		msg := <- message

		// 循环发送消息给所有在线用户
		for _, client := range onlineMap {
			client.C <- msg
		}
	}
}

func WriteMsgToClient(client Client, conn net.Conn) {
	// 监听用户自带的channel上是否有消息
	for msg := range client.C {
		// 发送消息给当前客户端
		conn.Write([]byte(msg + "\n"))
	}
}

func MakeMsg(client Client, msg string)(buf string) {
	buf = "[" + client.Addr + "]" + client.Name + ": " + msg
	return
}

func HandlerConnect(conn net.Conn) {
	// 关闭客户读连接请求
	defer conn.Close()

	// 创建channel判断，用户是否活跃
	isOnline := make(chan bool)

	// 获取用户网络地址 IP+PORT
	netAddr := conn.RemoteAddr().String()

	// 创建新连接用户的结构体，默认用户名是 IP+PORT
	client := Client {
		make(chan string),
		netAddr,
		netAddr,
	}

	// 新连接用户添加到在线用户 map 中，key=IP+PORT, value:client
	onlineMap[netAddr] = client

	// 创建专门用户给当前用户发送消息的协程
	go WriteMsgToClient(client, conn)

	// 发送用户上线消息到全局 channel 中
	//message <- "[" + netAddr + "]" + client.Name + " login"
	message <- MakeMsg(client, "login")

	// 创建一个channel，判断退出状态
	isQuit := make(chan bool)

	// 创建匿名子协程，处理用户发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				isQuit <- true
				fmt.Printf("监测到客户端: %s退出\n", client.Name)
				return
			}
			if err != nil {
				fmt.Println("读取错误：", err)
				return
			}
			// 读到的用户消息，保存到msg中
			msg := string(buf[:n])

			// 提取在线用户列表
			if (msg == "who\n" && len(msg) == 4) || (msg == "who\r\n" && len(msg) == 5) {
				conn.Write([]byte("online  user list: \n"))
				// 遍历map在线用户
				for _, user := range onlineMap {
					userInfo := user.Addr + ":" + user.Name + "\n"
					conn.Write([]byte(userInfo))
				}
			} else if len(msg) >= 8 && msg[:6] == "rename" { // rename|
				newName := strings.Split(msg, "|")[1]
				fmt.Println("new Name:", newName)
				client.Name = newName // 修改结构体
				onlineMap[netAddr] = client // 更新用户列表 onlineMap
				conn.Write([]byte("rename sucessfull.\n"))
			} else {
				// 将读到的用户消息，写入到message中进行广播
				message <- MakeMsg(client, msg)
			}

			isOnline <- true // 活跃用户

			//fmt.Println([]byte(msg))
			//fmt.Println(msg + "...")

		}
	}()

	// 保证不退出
	for {
		// 监听channel上的流动
		select {
			case <-isQuit:
				close(client.C)
				delete(onlineMap, client.Addr) // 将用户从online移除
				message <- MakeMsg(client, "logout") // 写入用户退出消息到全局channel
				return
			case <-isOnline:
				// 重置下面的计时器
			case <-time.After(time.Second * 10):
				delete(onlineMap, client.Addr) // 将用户从online移除
				message <- MakeMsg(client, "logout") // 写入用户退出消息到全局channel
				return
		}
	}
}

func main() {
	server_addr_port := "192.168.3.12:9898"

	// 1. 创建监听套接字
	listener, err := net.Listen("tcp", server_addr_port)
	if err != nil {
		outputError("创建监听套接字错误")
		return
	}

	// 2. 关闭套接字
	defer listener.Close()

	// 3. 创建管理者协程，管理map和全局channel
	go Manager()

	// 4. 循环监听客户端连接请求
	for {
		conn, err := listener.Accept()
		if err != nil {
			outputError("监听客户端连接错误")
			return
		}

		// 4. 启动协程处理客户端数据请求
		go HandlerConnect(conn)
	}



}
