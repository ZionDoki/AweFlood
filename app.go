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
	"time"
)

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
		fmt.Fprintf(os.Stderr, "\nERR: %s", err.Error())
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
	count, secondCount, counted := 0, 0, 0
	nextTime := time.Now().UnixNano()
	durationEnd := nextTime + 1e9

	content := make([]byte, 1024)
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

func startClient(IP string, port string, speed float64, duration int64, maxTries int) {

	startSig := []byte("QOS")
	endSig := []byte("END")
	startTries, endTries := maxTries, maxTries
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
	logPrintln(speed)

	conn, err := net.Dial("udp", IP+":"+port)
	defer conn.Close()
	if err != nil {
		os.Exit(1)
	}

	// define a channel storage bool, size one

	// send start
	// Loop:
	for {
		conn.Write(startSig)
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(time.Second))
		len, err := conn.Read(buf)
		startTries--
		if startTries < 0 {
			checkError(errors.New("Maxtries exceed"))
		}
		if err != nil {
			logPrintln("Retry!")
		} else {
			if string(re.FindAll(buf[:len], 1)[0]) == "OK" {
				logPrintln("Start..")
				endTime = time.Now().UnixNano() + (duration * 1e9)
				logPrintln(speed)
				break
			}
		}
	}

	logPrintln("Start Send Test Packets!")

	if duration != 0 {
		// go sendUntil(conn, endTime, 1e9/speed)
		sendUntil(conn, endTime, 1e9/speed)
	}
	logPrintln("OK")

	for {
		conn.Write(endSig)

		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(time.Second))
		len, err := conn.Read(buf)

		endTries--
		if endTries < 0 {
			checkError(errors.New("Maxtries exceed"))
		}

		if err != nil {
			logPrintln("Retry!")
		} else {
			if string(re.FindAll(buf[:len], 1)[0]) == "OK" {
				logPrintln("END!")
				break
			}
		}
	}
}

func listenPort(port string, keepAlive bool, maxTries int) {

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

	for {
		data := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(time.Second * 2))
		_, remoteAddr, err := conn.ReadFromUDP(data)

		if err != nil {
			listenTries--
			if listenTries < 0 {
				checkError(errors.New("Maxtries exceed"))
			}
			logPrint("=")
		} else {
			if string(re.FindAll(data[:], 1)[0]) == "END" {
				retResult(time.Now(), count-counted)
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
						retResult(time.Now(), secondCount)
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
		}
	}
}

func main() {

	var operation string
	var v string
	var duration int64
	var IP string
	var port string
	var maxTries int
	var keepAlive bool

	flag.StringVar(&operation, "o", "sever", "operation you want to call! [ sever | client ]")
	flag.IntVar(&maxTries, "t", 10, "maxTries when send start signal or end siganal! [ Client Only ]")
	flag.StringVar(&v, "v", "100.0", "Test Bandwith KB/s [ Client Only ]")
	flag.Int64Var(&duration, "d", 10, "Duration of test [ Client Only ]")
	flag.StringVar(&IP, "i", "127.0.0.1", "target IP [ Client Only ]")
	flag.StringVar(&port, "p", "2333", "target port")
	flag.BoolVar(&keepAlive, "a", false, "Keep sever end alive!")
	flag.BoolVar(&showLog, "l", false, "Show log")
	flag.Parse()

	float64V, errFloat := strconv.ParseFloat(v, 64)

	logPrint(`
	******************************************************
	* Welcome to AwesomeQoS UDP Bandwidth testing tools! *
	******************************************************
	`)

	if operation == "sever" {
		listenPort(port, keepAlive, maxTries)
	} else if operation == "client" {
		if errFloat == nil {
			startClient(IP, port, float64V, duration, maxTries)
		} else {
			logPrintln(errFloat)
		}
	} else {
		logPrintln("Please Enter Correct Param Before You Start Test!")
	}
}
