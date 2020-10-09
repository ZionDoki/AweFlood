package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PACKAGESIZE : A const of package size
const PACKAGESIZE int = 992

var showLog bool

func logPrint(format string, a ...interface{}) {
	if showLog {
		fmt.Printf(format, a...)
	}
}

func retResult(a ...interface{}) {
	fmt.Printf("{ \"timestamp\": %v, \"value\": %v}\n", a...)
}

func checkError(err error) {
	if err != nil {

		x := fmt.Sprintf("%s", err)
		if strings.Contains(x, "Only one usage of each socket address") {
			logPrint("Error: Port occupied")
		} else {
			fmt.Printf("Error: %s \n", err.Error())
		}
		os.Exit(1)
	}
}

func logPrintln(a ...interface{}) {
	if showLog {
		fmt.Println(a...)
	}
}

func removeSpace(content []byte) string {
	pattern := "([^\u0000]*)"
	re, _ := regexp.Compile(pattern)
	return string(re.FindAll(content[:], 1)[0])
}

func sendUntil(udpConn net.Conn, endTime int64, interval float64) {
	count, secondCount, counted := 0, 0, 0
	nextTime := time.Now().UnixNano()
	durationEnd := nextTime + 1e9

	content := make([]byte, PACKAGESIZE)
	rand.Read(content)

	// start test
	for endTime >= time.Now().UnixNano() {
		if time.Now().UnixNano() >= nextTime {
			nextTime += int64(interval)
			udpConn.Write(content)
			count++
			if durationEnd >= time.Now().UnixNano() {
				secondCount++
			} else {
				retResult(time.Now(), secondCount)
				counted += secondCount
				durationEnd += 1e9
				secondCount = 0
			}
		}
	}

	retResult(time.Now(), count-counted)
	logPrint("Total send number is: %v \n", count)
}

func replyUntil(udpConn *net.UDPConn, remoteAddr *net.UDPAddr, endTime int64, interval float64) {
	count, secondCount, counted := 0, 0, 0
	nextTime := time.Now().UnixNano()
	durationEnd := nextTime + 1e9

	content := make([]byte, PACKAGESIZE)
	rand.Read(content)

	// start test
	for endTime >= time.Now().UnixNano() {
		if time.Now().UnixNano() >= nextTime {
			nextTime += int64(interval)
			udpConn.WriteToUDP(content, remoteAddr)
			count++
			if durationEnd >= time.Now().UnixNano() {
				secondCount++
			} else {
				retResult(time.Now(), secondCount)
				counted += secondCount
				durationEnd += 1e9
				secondCount = 0
			}
		}
	}

	retResult(time.Now(), count-counted)
	logPrint("Total send number is: %v \n", count)
}

func sendSignal(signal []byte, maxTries int, udpConn net.Conn) {
	for {
		udpConn.Write(signal)
		buf := make([]byte, PACKAGESIZE)
		udpConn.SetReadDeadline(time.Now().Add(time.Second * 2))
		_, err := udpConn.Read(buf)
		maxTries--
		if maxTries < 0 {
			checkError(errors.New("Maxtries exceed"))
		}
		if err != nil {
			logPrintln("Retry!")
		} else {
			if removeSpace(buf) == "OK" {
				break
			}
		}
	}
}

func startClient(IP string, port string, speed float64, duration int64, special bool, maxTries int) {

	startSig := []byte("QOS")
	specialStartSig := []byte(fmt.Sprintf("QOS,%v,%v", speed, duration))
	endSig := []byte("END")

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

	logPrintln(speed)

	conn, err := net.Dial("udp", IP+":"+port)
	defer conn.Close()
	if err != nil {
		os.Exit(1)
	}

	// define a channel storage bool, size one

	if special {
		// 内-外网模式
		listenTries := maxTries
		count, counted := 0, 0
		secondCount := 0
		firstTime := true
		var durationEnd int64

		logPrintln("Starting")
		// send start
		sendSignal(specialStartSig, maxTries, conn)
		logPrint("Started")
		logPrintln("Start Send Test Packets!")

		for {

			data := make([]byte, PACKAGESIZE)
			conn.SetReadDeadline(time.Now().Add(time.Second * 2))
			_, err := conn.Read(data)

			if err != nil {
				listenTries--
				if listenTries < 0 {
					checkError(errors.New("Maxtries exceed"))
				}
				logPrint("*")
			} else {
				if removeSpace(data) == "END" {
					retResult(time.Now(), count-counted)
					break
				} else {
					count++
					if firstTime {
						durationEnd = time.Now().UnixNano() + 1e9
						firstTime = false
					}
					if durationEnd >= time.Now().UnixNano() {
						secondCount++
					} else {
						retResult(time.Now(), secondCount)
						counted += secondCount
						durationEnd += 1e9
						secondCount = 0
					}
				}
			}
		}
	} else {
		logPrint("Starting")
		// send start
		sendSignal(startSig, maxTries, conn)
		logPrint("Started")
		logPrintln("Start Send Test Packets!")
		// 非内-外网模式
		endTime := time.Now().UnixNano() + (duration * 1e9)
		if duration != 0 {
			sendUntil(conn, endTime, 1e9/speed)
		}

		logPrintln("OK")
		logPrint("Ending")

		sendSignal(endSig, maxTries, conn)
		logPrint("Ended!")
	}
}

func listenPort(port string, keepAlive bool, special bool, maxTries int) {

	count := 0
	testing := false
	firstTime := true

	listenTries := maxTries
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

	logPrint("Started")

	// 内-外网模式
	if special {

		var speed float64
		var duration int64

		// 等待握手
		for {
			data := make([]byte, PACKAGESIZE)
			conn.SetReadDeadline(time.Now().Add(time.Second * 2))
			_, remoteAddr, err := conn.ReadFromUDP(data)

			if err != nil {
				listenTries--
				if listenTries < 0 {
					checkError(errors.New("Maxtries exceed"))
				}
				logPrint("*")
			} else {
				missionStr := removeSpace(data)
				if strings.Index(missionStr, "QOS") != -1 {
					if testing == false {

						params := strings.Split(missionStr, ",")
						speed, err = strconv.ParseFloat(params[1], 64)
						checkError(err)
						duration, err = strconv.ParseInt(params[2], 10, 64)
						checkError(err)

						_, err = conn.WriteToUDP([]byte("OK"), remoteAddr)
						checkError(err)
						firstTime = true
						testing = true
						replyUntil(conn, remoteAddr, time.Now().UnixNano()+(duration*1e9), 1e9/speed)

						// end test
						_, err = conn.WriteToUDP([]byte("END"), remoteAddr)
						checkError(err)

						if keepAlive {
							testing, firstTime = false, true
							count, counted, secondCount = 0, 0, 0
							continue
						} else {
							break
						}

					} else {
						_, err = conn.WriteToUDP([]byte("Testing Anothor Mission, Please Wait!"), remoteAddr)
						checkError(err)
					}
				}
			}
		}

	} else {
		// 非内-外网模式
		for {
			data := make([]byte, PACKAGESIZE)
			conn.SetReadDeadline(time.Now().Add(time.Second * 2))
			_, remoteAddr, err := conn.ReadFromUDP(data)

			if err != nil {
				listenTries--
				if listenTries < 0 {
					checkError(errors.New("Maxtries exceed"))
				}
				logPrint("*")
			} else {
				if string(re.FindAll(data[:], 1)[0]) == "END" {
					retResult(time.Now(), count-counted)
					_, err = conn.WriteToUDP([]byte("OK"), remoteAddr)
					if keepAlive {
						testing, firstTime = false, true
						count, counted, secondCount = 0, 0, 0
						continue
					} else {
						break
					}
				}

				if string(re.FindAll(data[:], 1)[0]) == "QOS" {

					if testing == false {
						_, err = conn.WriteToUDP([]byte("OK"), remoteAddr)
						checkError(err)
						firstTime = true
						testing = true
					} else {
						_, err = conn.WriteToUDP([]byte("Testing Anothor Mission, Please Wait!"), remoteAddr)
						checkError(err)
					}
				}

				if testing && (string(re.FindAll(data[:], 1)[0]) != "QOS") {
					count++
					if firstTime {
						durationEnd = time.Now().UnixNano() + 1e9
						firstTime = false
					}
					if durationEnd >= time.Now().UnixNano() {
						secondCount++
					} else {
						retResult(time.Now(), secondCount)
						counted += secondCount
						durationEnd += 1e9
						secondCount = 0
					}
				}
			}
		}
	}
}

// 穿局域网时，客户端发送局域网穿透模式信号，并附带参数信息

func main() {

	var operation string
	var v string
	var duration int64
	var IP string
	var port string
	var maxTries int
	var keepAlive bool
	var special bool

	flag.StringVar(&operation, "o", "server", "operation you want to call! [ server | client ]")
	flag.IntVar(&maxTries, "t", 10, "maxTries when send start signal or end siganal! [ Client Only ]")
	flag.StringVar(&v, "v", "100.0", "Test Bandwith KB/s [ Client Only ]")
	flag.Int64Var(&duration, "d", 10, "Duration of test [ Client Only ]")
	flag.StringVar(&IP, "i", "127.0.0.1", "target IP [ Client Only ]")
	flag.StringVar(&port, "p", "2333", "target port")
	flag.BoolVar(&keepAlive, "a", false, "Keep sever end alive!")
	flag.BoolVar(&showLog, "l", false, "Show log")
	flag.BoolVar(&special, "s", false, "Special mode for accross the local network")
	flag.Parse()

	float64V, errFloat := strconv.ParseFloat(v, 64)

	logPrint(`
	******************************************************
	* Welcome to AwesomeQoS UDP Bandwidth testing tools! *
	******************************************************
	`)

	if operation == "server" {
		listenPort(port, keepAlive, special, maxTries)
	} else if operation == "client" {
		if errFloat == nil {
			startClient(IP, port, float64V, duration, special, maxTries)
		} else {
			logPrintln(errFloat)
		}
	} else {
		logPrintln("Please Enter Correct Param Before You Start Test!")
	}
}
