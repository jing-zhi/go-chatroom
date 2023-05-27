package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

//命令大全：
/*
1. online           查看在线用户
2. rename-新名字     改名字
3. 关掉客服端，其他客户端会收到他下线的消息
4. 超过60秒没发消息   自动下线
*/

//用户结构体类型
type Client struct {
	C    chan string
	Name string
	Addr string
}

//全局map，存储在线用户,map型切片
var onlineMap map[string]Client

//全局channel 传递用户消息
var message = make(chan string)

func main() {
	//go是多返回值，
	//listener监听用户
	listener, err := net.Listen("tcp", "127.0.0.1:8000")
	if err != nil {
		fmt.Println("监听错误", err)
		return
	}
	defer listener.Close()
	//go程
	go Manager()
	//循环监听客户端连接请求
	//for循环接受客户端是否有行动
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept错误", err)
			return
		}
		//启动go程处理客户端数据请求
		go HandlerConnect(conn)
	}

}

//go程终端，传给用户消息
func WriteMsgToClient(clnt Client, conn net.Conn) {
	for msg := range clnt.C {
		//用户的chan监听是否有消息，
		conn.Write([]byte(msg + "\n"))
	}

}

//客户端写消息消息，這是一個函數，参数是客户端和消息，返回一个消息
func MakeMsg(clnt Client, msg string, isWhisper bool) (buf string) {
	//buf = "[" + clnt.Addr + "]" + clnt.Name + ":" + msg
	if isWhisper {
		buf = "whisper:" + clnt.Name + ":  " + msg
	} else {
		buf = clnt.Name + ":  " + msg
	}

	return
}

func HandlerConnect(conn net.Conn) {
	defer conn.Close()
	//创建hasDate 判断是否活跃，是否还在线
	hasDate := make(chan bool) //双向通道
	//获取新用户 网络地址 IP+port
	netAddr := conn.RemoteAddr().String()
	//创建链接新用户的 结构体 默认用户是ip+port
	//简单说就是初始化
	clnt := Client{
		make(chan string),
		netAddr,
		netAddr,
	}
	//将新连接用户，添加到在线用户map，key clnt.Name
	onlineMap[clnt.Name] = clnt

	//go程终端，传给用户消息
	go WriteMsgToClient(clnt, conn)

	//用户上线，发送到全局map
	//message听取每一个在线用户的消息
	message <- MakeMsg(clnt, "login!", false)

	//创建一个channel,用来判断退出状态
	isQuit := make(chan bool)
	//匿名go程，专门处理用户发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			//conn.Read函数有两个返回值，懂吧？
			n, err := conn.Read(buf)
			if n == 0 { //没有消息的情况，试试发一个空
				isQuit <- true
				fmt.Printf("检测到客户端:%s退出\n", clnt.Name)
				return
			}
			if err != nil { //出错误的情况  说实话因为不懂，感觉这没用
				fmt.Println("conn.Read err;", err)
				return
			}
			//读入用户消息存入msg
			//buf是一个字节切片，这里就是调用string方法，把这个字节棋牌你转化为字符串形式
			msg := string(buf[:n-1])

			//查看用户在线列表
			if msg == "online-" && len(msg) == 7 {
				conn.Write([]byte("online user list:\n"))
				//遍历当前map，获取在线用户
				for _, user := range onlineMap { //clnt就是value，就是具体的客户端
					userInfo := user.Addr + ":" + user.Name + "\n"
					conn.Write([]byte(userInfo))
				}
				//改名字
			} else if len(msg) >= 8 && msg[:7] == "rename-" {
				newName := strings.Split(msg, "-")[1]
				clnt.Name = newName
				onlineMap[clnt.Name] = clnt
				conn.Write([]byte("rename sucess!\n"))
			} else if strings.TrimSpace(msg) == "help-" {
				command := `
命令提示:
1. online-           View online users
2. rename-           Change name
3. to-newmessage-    Private chat
3. help-             View All commands
3. 关掉客服端，其他客户端会收到他下线的消息
4. 超过60秒没发消息    自动下线
`
				conn.Write([]byte(command))
				//私发功能,用用户名字指定对象
			} else if len(msg) >= 5 && msg[:3] == "to-" {
				aimClientName := strings.Split(msg, "-")
				name := aimClientName[1]
				message := aimClientName[2] //这个是局部变量
				clnt, ok := onlineMap[name]
				if !ok {
					conn.Write([]byte("User does not exist!\n"))
				} else {
					//msg := MakeMsg(clnt, clnt.Name, true)
					//clnt.C <- msg
					clnt.C <- message
				}
			} else {
				//将读到的用户信息，写入到message
				message <- MakeMsg(clnt, msg, false)
			}
			//判断是否超时没发消息
			hasDate <- true

		}
	}()

	//保证不退出
	for {
		//判断用户是否退出
		select {
		case <-isQuit:
			delete(onlineMap, clnt.Addr)
			message <- MakeMsg(clnt, "退出", false) //给其他用户广播
			return
		case <-hasDate: //重置活跃时间
		case <-time.After(time.Second * 20000):
			delete(onlineMap, clnt.Addr)
			message <- MakeMsg(clnt, "退出", false) //给其他用户广播
			return
		}

	}

}

func Manager() {
	//初始化
	onlineMap = make(map[string]Client)
	//监听全局channel中是否有数据，有数据存到msg,没数据就阻塞
	for {
		//msg读chann的消息
		msg := <-message
		//循环发送消息给所有在线用户，
		for _, clnt := range onlineMap {
			clnt.C <- msg
		}
	}

}
