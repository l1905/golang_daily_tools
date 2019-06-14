package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"log"
	"net"
)

const buf_size=8192
func main() {
	log.SetFlags(log.LstdFlags|log.Lshortfile)
	l, err := net.Listen("tcp", ":8082")
	if err != nil {
		log.Panic(err)
	}

	for {
		client, err := l.Accept()
		if err != nil {
			log.Panic(err)
		}
		log.Println("server 建立连接")

		//新起独立连接到 seq 正向代理
		server, _ := net.Dial("tcp", "127.0.0.1:3128")

		//接受新请求， 将包发给squid
		go handleClientRequest(client, server)

		//监听正向代理返回的包内容， 打包resp
		go handleClientResp(client, server)
	}
}

//解析client包内容， 将包发给正向代理
func handleClientRequest(client net.Conn, desc_client net.Conn) {
	if client == nil {
		return
	}
	defer client.Close()
	defer desc_client.Close()

	var stream_buf bytes.Buffer

	for {
		buf := make([]byte, buf_size)

		n, err := client.Read(buf)
		if err != nil {
			log.Println(err)
			goto RESULT
		}
		data := buf[:n]

		//打到流buf中
		stream_buf.Write(data)

		for {
			//获取全部数据
			all_data := stream_buf.Bytes()
			if len(all_data) > 8 {
				header := all_data[:8]

				data_len := binary.BigEndian.Uint32(header[4:])
				real_buf := all_data[8:]

				if len(real_buf) >= int(data_len) {
					encode_data := real_buf[:int(data_len)]
					data , _:= base64.StdEncoding.DecodeString(string(encode_data))

					desc_client.Write(data)

					stream_buf.Read(all_data[:(8+int(data_len))])
				} else {
					break
				}
			} else {
				break
			}

		}

	}
RESULT:


}

//todo 解析正向代理返回的包， 并且将包发给client
func handleClientResp(client net.Conn, desc_client net.Conn) {
	if client == nil {
		return
	}
	defer client.Close()
	defer desc_client.Close()


	for {
		buf := make([]byte, buf_size)

		//接收正向代理包内容
		n, err := desc_client.Read(buf)
		if err != nil {
			log.Println(err)
			break
		}
		data := buf[:n]
		//client.Write(data)
		encode_data := base64.StdEncoding.EncodeToString(data)

		//压缩成二进制包
		first := uint32(1)
		//第二个字节
		second := uint32(len(encode_data))
		buf_02 := make([]byte, 8)
		binary.BigEndian.PutUint32(buf_02[0:], first)
		binary.BigEndian.PutUint32(buf_02[4:], second)

		send_data := string(buf_02) + encode_data
		//建立隧道，加包
		log.Println("从正向代理接收到包")

		//同等加密方式，回写给client端
		client.Write([]byte(send_data))

	}

}

