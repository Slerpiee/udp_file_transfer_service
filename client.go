package main

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

type packet_file struct{
    byte_file []byte
    packet_count []byte
	md5 []byte
	name []byte
	splitted_file [][]byte
	unique []byte
}


var global_file packet_file

var client net.PacketConn
var remoteAddr *net.UDPAddr


func init_file(){
	path := os.Args[1]
	file, err := os.Open(path)
    if err != nil{
        log.Fatal(err)
    }
    stat, _ := file.Stat()
    global_file.byte_file = make([]byte, stat.Size())
    file.Read(global_file.byte_file)
	global_file.name = []byte(file.Name())
	massive := []byte{}
	for ind, el := range global_file.byte_file{
		if ind % 6000 == 0 && ind != 0{
			global_file.splitted_file = append(global_file.splitted_file, massive)
			massive = []byte{}
		}
		massive = append(massive, el)
	}
	if len(massive) != 0{
		global_file.splitted_file = append(global_file.splitted_file, massive)
	}
	binary.BigEndian.PutUint64(global_file.packet_count, uint64(len(global_file.splitted_file)))
	global_file.md5 = get_md5(global_file.byte_file)
}


func get_md5(data[]byte) (total []byte){
	for _, el := range md5.Sum(data){
		total = append(total, el)
	}
	fmt.Println(hex.EncodeToString(total))
	return 
}


func wait_for_answer(err chan struct{}, answer chan []byte) {
	time.AfterFunc(3*time.Second, func(){
		err <- struct{}{}
	})
	buffer := make([]byte, 65535)
	byt, adr, er := client.ReadFrom(buffer)
	if er != nil{
		log.Fatal(er)
	}
	log.Printf("Accept: %v bytes from %v", byt, adr)
	data := make([]byte, byt)
	copy(data, buffer[:byt])
	answer <- data

}


func send_start_package(){
	startpackage := []byte{0}
	startpackage = append(startpackage, global_file.packet_count...)
	startpackage = append(startpackage, global_file.md5...)
	startpackage = append(startpackage, global_file.name...)
	err_chan := make(chan struct{})
	data_chan := make(chan []byte)
	client.WriteTo(startpackage, remoteAddr)
	go wait_for_answer(err_chan, data_chan)
	for{
		select{
		case <- err_chan:
			log.Fatal("Time Expired, Server is not Responding")
			return
		case data := <- data_chan:
			fmt.Println("OK - uuid")
			global_file.unique = append(global_file.unique, data...)
			return
		}
	}
	
}

func send_packets(){
	fmt.Println("Packets for file:", len(global_file.splitted_file))
	for ind, el := range global_file.splitted_file{
		i := ind
		pack := []byte{1}
		pack = append(pack, global_file.unique...)
		num := make([]byte, 16)
		binary.BigEndian.PutUint64(num, uint64(i))
		pack = append(pack, num...)
		pack = append(pack, el...)
		client.WriteTo(pack, remoteAddr)
		time.Sleep(1*time.Millisecond)
		
	}
}



func init(){
	var port string 
	fmt.Printf("Введите порт сервера: ")
    fmt.Scan(&port)
	addr := "127.0.0.1:" + port
	remoteAddr, _ = net.ResolveUDPAddr("udp", addr)
	global_file.md5 = make([]byte, 16)
	global_file.packet_count = make([]byte, 16)
}



func main() {
	if len(os.Args) != 2{
		log.Fatal("ERROR: File path did not provided")
	}
	con, err := net.ListenPacket("udp", "")
	client = con
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	init_file()
	send_start_package()
	send_packets()
}
