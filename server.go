package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
	"os"
	"crypto/md5"
	"github.com/google/uuid"
	"encoding/hex"
)


var server net.PacketConn
var mutex sync.Mutex


type user struct{
	file_name string
	packet_count int
	packets map[int][]byte
	md5 []byte
}
var users map[string]user


var time_channels map[string] chan int


func handle_request(data []byte, addr net.Addr){
	switch data[0]{
	case 0:
		fmt.Println("Start package arrived from: " + addr.String())
		uuid := uuid.New()
		send := make([]byte, 36)
		copy(send, []byte(uuid.String()))
		server.WriteTo(send, addr)
		newuser := user{}
		newuser.md5 = data[17:33]
		newuser.packet_count = int(binary.BigEndian.Uint64(data[1:16]))
		newuser.file_name = string(data[33:])
		newuser.packets = map[int][]byte{}
		fmt.Println(newuser)
		mutex.Lock() 
		users[uuid.String()] = newuser
		time_channels[uuid.String()] = make(chan int)
		mutex.Unlock()
		go timer(uuid.String())
	case 1:
		uid := string(data[1:37])

		
		mutex.Lock()
		_, ok := users[uid]
		if !ok{
			fmt.Println("No such user auth")
			return
		}
		mutex.Unlock()


		num := int(binary.BigEndian.Uint64(data[37:53]))
		pack := data[53:]

		fmt.Println("Package:", num+1, "for: ", uid)

		mutex.Lock()
		channel := time_channels[uid]
		mutex.Unlock()

		channel <- 1

		mutex.Lock()
		users[uid].packets[num] = pack
		user := users[uid]
		mutex.Unlock()
		if len(user.packets) == user.packet_count{
			mutex.Lock()
			c := time_channels[uid]
			mutex.Unlock()
			c <- 0
			mutex.Lock()
			delete(time_channels, uid)
			mutex.Unlock()
			file := []byte{}
			for i:=0; i<user.packet_count;i++{
				num = i
				file=append(file, user.packets[num]...)
			}
			if hex.EncodeToString(user.md5) == hex.EncodeToString(get_md5(file)){
				fmt.Println("BUILD FILE FOR: "+uid)
				build_file(file, user.file_name, uid)
			} else{
				fmt.Println("hash sum compare failed")
			}
			mutex.Lock()
			delete(users, uid)
			mutex.Unlock()
		}
	}
	
}


func get_md5(data[]byte) (total []byte){
	for _, el := range md5.Sum(data){
		total = append(total, el)
	}
	return 
}


func build_file(file []byte, name string, hex string) {
	_, err := os.Stat(name)
	if err == nil {
		name = hex + name
	}
	f, err := os.Create(name)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	_, e := f.Write(file)
	if e != nil {
		fmt.Println(e)
		return
	}

}


func timer(id string){
	for{
		mutex.Lock()
		channel := time_channels[id]
		mutex.Unlock()

		select{
		case x := <- channel:
			if x == 1{
				go timer(id)
				fmt.Println("timer resetted for: "+id)
			}
			return
		case <- time.After(5*time.Second):
			fmt.Println("Timer ended for: "+id)
			mutex.Lock()
			delete(users, id)
			mutex.Unlock()
			return
		}
	}
}


func init(){
	users = map[string]user{}
	time_channels = map[string]chan int{}
	err := os.Mkdir("storage", 0750)
	if err != nil && !os.IsExist(err) {
		fmt.Println("Already exists")
	}
	os.Chdir("storage")
}


func main(){
	var port string 
	fmt.Printf("Введите порт сервера, который будет слушать пакеты: ")
    fmt.Scan(&port) 
	addr := "127.0.0.1:" + port
	con, err := net.ListenPacket("udp", addr)
	server = con
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Server started on:", server.LocalAddr().String())
	defer server.Close()

	buffer := make([]byte, 65535)

	for {
		bytesRead, remoteAddr, err := server.ReadFrom(buffer)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Read: %v bytes from %v\n", bytesRead, remoteAddr)
		data := make([]byte, bytesRead)
		copy(data, buffer[:bytesRead])
		go handle_request(data, remoteAddr)
	}
}
