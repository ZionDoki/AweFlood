package main

import (
	"bytes"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"
)

var showLog bool

func logPrint(format string, a ...interface{}) {
	if showLog {
		fmt.Printf(format, a...)
	}
}

func checkError(err error) {
	if err != nil {
		logPrint("Error: %s \n", err.Error())
		os.Exit(1)
	}
}

func logPrintln(a ...interface{}) {
	if showLog {
		fmt.Println(a...)
	}
}

func sendUntil(udpConn net.Conn, endTime int64, interval float64) {
	count := 0
	nextTime := time.Now().UnixNano()
	content := make([]byte, 1024)

	rand.Read(content)

	// start test
	for endTime >= time.Now().UnixNano() {
		if time.Now().UnixNano() >= nextTime {
			nextTime += int64(interval)
			udpConn.Write(content)
			count++
		}
	}

	fmt.Println(count)
}

func startClient(IP string, port string, speed float64, duration int64) {

	startSig := []byte("QOS")
	endSig := []byte("END")

	pattern := "([^\u0000]*)"
	re, _ := regexp.Compile(pattern)

	portLen := bytes.Count([]byte(port+IP), nil)
	addStarts := ""

	for i := 0; i < portLen-2; i++ {
		addStarts += "*"
	}

	logPrint(`
	********************%v
	* Start Test to %v *
	********************%v
	`, addStarts, IP+":"+port, addStarts)

	endTime := time.Now().UnixNano() + (duration * 1e9)
	fmt.Println(speed)

	conn, err := net.Dial("udp", IP+":"+port)
	defer conn.Close()
	if err != nil {
		os.Exit(1)
	}

	// send start
	for {
		conn.Write(startSig)

		buf := make([]byte, 1024)
		len, err := conn.Read(buf)
		checkError(err)

		if string(re.FindAll(buf[:len], 1)[0]) == "OK" {
			fmt.Println("xixi", buf[:len])
			break
		} else {
			fmt.Println("RETRY!", buf[:len])
			time.Sleep(time.Second)
		}
	}

	fmt.Println("Start Send Test Packets!")
	if duration != 0 {
		sendUntil(conn, endTime, 1e9/speed)
	}
	fmt.Println("OK")

	for {
		conn.Write(endSig)

		buf := make([]byte, 1024)
		len, err := conn.Read(buf)
		checkError(err)

		if string(re.FindAll(buf[:len], 1)[0]) == "OK" {
			fmt.Println("END!")
			break
		} else {
			fmt.Println("RETRY!", buf[:len], endSig)
		}
	}

}

func listenPort(port string, keepAlive bool) {

	count := 0
	testing := false
	firstTime := true

	counted := 0
	secondCount := 0
	var durationEnd int64

	pattern := "([^\u0000]*)"
	re, _ := regexp.Compile(pattern)

	portLen := bytes.Count([]byte(port), nil) - 1
	addStarts := ""

	for i := 0; i < portLen-2; i++ {
		addStarts += "*"
	}

	udpAddr, err := net.ResolveUDPAddr("udp", ":"+port)
	checkError(err)

	logPrint(`
	**************************%v
	* Start sever at %v port *
	**************************%v
	`, addStarts, port, addStarts)

	conn, err := net.ListenUDP("udp", udpAddr)
	checkError(err)

	for {
		data := make([]byte, 1024)
		n, remoteAddr, err := conn.ReadFromUDP(data)

		if string(re.FindAll(data[:], 1)[0]) == "END" {
			fmt.Println(count - counted)
			_, err = conn.WriteToUDP([]byte("OK"), remoteAddr)
			if keepAlive {
				testing, firstTime = false, true
				count, counted, secondCount = 0, 0, 0
			} else {
				break
			}
		} else {
			if testing {
				count++
				if firstTime {
					durationEnd = time.Now().UnixNano() + 1e9
					firstTime = false
				}
				if durationEnd >= time.Now().UnixNano() {
					secondCount++
				} else {
					fmt.Println(secondCount)
					counted += secondCount
					durationEnd += 1e9
					secondCount = 0
				}
			} else if string(re.FindAll(data[:], 1)[0]) == "QOS" {
				_, err = conn.WriteToUDP([]byte("OK"), remoteAddr)
				checkError(err)
				firstTime = true
				testing = true
			} else if testing {
				_, err = conn.WriteToUDP([]byte("Testing Anothor Mission, Please Wait!"), remoteAddr)
				checkError(err)
			}
		}

		if err != nil {
			fmt.Println(string(data[:]), n, remoteAddr)
			logPrint("error during read: %s", err)
		}
	}
}

func main() {

	var operation string
	var v string
	var duration string
	var IP string
	var port string
	var keepAlive bool

	flag.StringVar(&operation, "o", "sever", "operation you want to call! [ sever | client ]")
	flag.StringVar(&v, "v", "100.0", "Test Bandwith KB/s")
	flag.StringVar(&duration, "t", "10", "Duration of test")
	flag.StringVar(&IP, "i", "127.0.0.1", "target IP")
	flag.StringVar(&port, "p", "2333", "target port")
	flag.BoolVar(&keepAlive, "a", false, "Keep sever end alive!")
	flag.BoolVar(&showLog, "l", false, "Show log")
	flag.Parse()

	float64V, errFloat := strconv.ParseFloat(v, 64)
	int64Duration, errInt := strconv.ParseInt(duration, 10, 64)

	logPrint(`
	******************************************************
	* Welcome to AwesomeQoS UDP Bandwidth testing tools! *
	******************************************************
	`)

	if operation == "sever" {
		listenPort(port, keepAlive)
	} else if operation == "client" {
		if errInt == nil && errFloat == nil {
			fmt.Println(int64Duration, duration)
			startClient(IP, port, float64V, int64Duration)
		} else {
			fmt.Println(errInt, errFloat)
		}
	} else {
		fmt.Println("Please Enter Correct Param Before You Start Test!")
	}
}
