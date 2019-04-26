package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	post   message
	metric openfalconMetric

	// pid    = fmt.Sprintf("%d", os.Getpid())
)

type SockTabEntry struct {
	ino        string
	LocalAddr  string
	LocalPort  string
	RemoteAddr string
	RemotePort string
	Process    string
}
type message struct {
	Item []openfalconMetric `json:"item"`
}
type addresses map[string]int

type ports map[string]int

type openfalconMetric struct {
	Metric      string `json:"metric"`
	Endpoint    string `json:"endpoint"`
	Timestamp   int64  `json:"timestamp"`
	Value       int    `json:"value"`
	Step        int    `json:"step"`
	CounterType string `json:"counterType"`
	Tags        string `json:"tags"`
}

// type Lines struct {
// 	inConn    map[string]int
// 	outConn   map[string]int
// 	localConn map[string]int
// }

type inConLines struct {
	lines map[string]int
}
type outConLines struct {
	lines map[string]int
}
type localConLines struct {
	lines map[string]int
}

func (this *inConLines) init() {
	this.lines = make(map[string]int)
}
func (this *outConLines) init() {
	this.lines = make(map[string]int)
}
func (this *localConLines) init() {
	this.lines = make(map[string]int)
}

// func dothings() (inConLines, outConLines, localConLines, error) {
func dothings() (outConLines, error) {

	// s, err := netstat.TCPSocks(netstat.NoopFilter)
	s := getStruct()

	// var in inConLines
	var out outConLines
	// var lo localConLines
	// in.init()
	out.init()
	// lo.init()
	result := GetETCDKeyValues()
	for k, v := range result {
		fmt.Println(k, v)
	}
	for _, socket := range s {

		raddr := socket.RemoteAddr
		rport := socket.RemotePort
		process := socket.Process
		if process == "" {
			continue
		}

		// // fmt.Printf("rport:%s raddr:%s lport:%s process:%s\n", rport, raddr, lport, process)
		// // fmt.Println(processName)
		// // process := socket.Process.String()
		// if result[raddr] == "localhost" {
		// 	// if strconv.ParseInt(rport, 16, 32) > 10000 {
		// 	// 	continue
		// 	// }
		// 	// 如果远程端口号已知，则是已知服务，否则为未知端口服务
		// 	if _, ok := result[rport]; ok {
		// 		info := "localPort=" + result[rport]
		// 		lo.lines[info]++
		// 		continue
		// 	}
		// 	info := "localPort=" + rport
		// 	lo.lines[info]++
		// 	continue
		// }

		// //如果本地端口已知，且远程地址已知，则是公司内部的服务访问，否则是外部服务的访问请求
		// if _, ok := result[lport]; ok {
		// 	if _, ok := result[raddr]; ok {
		// 		info := "srcIP=" + result[raddr] + "," + "localPort=" + result[lport]
		// 		in.lines[info]++
		// 		continue
		// 	}
		// 	info := "srcIP=" + raddr + "," + "localPort=" + result[lport]
		// 	inConut := "localPort=" + result[lport]
		// 	in.lines[inConut]++
		// 	in.lines[info]++
		// 	continue
		// }

		// 如果远程端口已知，且远程地址已知，则是本机向公司内部已知的服务发器的请求，否则是向未知的服务发起请求
		if _, ok := result[rport]; ok {
			if _, ok := result[raddr]; ok {
				info := "dstIP=" + result[raddr] + "," + "dstPort=" + result[rport] + "," + "process=" + process
				out.lines[info]++
				continue
			}
			// if strings.Contains()
			info := "dstIP=" + raddr + "," + "dstPort=" + result[rport] + "," + "process=" + process
			out.lines[info]++
			// continue
		}
	}
	// return in, out, lo, nil
	return out, nil

}

func (local localConLines) genMetrics(msg *message) *message {
	// m := newOpenFalconMetric(&metric)
	var metric openfalconMetric
	newOpenFalconMetric(&metric)
	metric.Metric = "LocalConnection"
	for key, value := range local.lines {
		if value < 10 {
			continue
		}
		metric.Tags = key
		metric.Value = value - 1
		msg.Item = append(msg.Item, metric)
	}
	return msg
}

func (out outConLines) genMetrics(msg *message) *message {
	var metric openfalconMetric
	newOpenFalconMetric(&metric)
	metric.Metric = "OutConnection"
	for key, value := range out.lines {
		if value < 10 {
			continue
		}
		metric.Tags = key
		metric.Value = value - 1
		msg.Item = append(msg.Item, metric)
	}
	return msg
}
func (in inConLines) genMetrics(msg *message) *message {
	var metric openfalconMetric
	newOpenFalconMetric(&metric)
	metric.Metric = "InConnection"
	for key, value := range in.lines {
		if value < 10 {
			continue
		}
		metric.Tags = key
		metric.Value = value - 1
		msg.Item = append(msg.Item, metric)
	}
	return msg
}

func newOpenFalconMetric(metric *openfalconMetric) *openfalconMetric {
	hostname, _ := os.Hostname()
	// metric.Metric = "Connected_ports"
	metric.Timestamp = time.Now().Unix()
	metric.Step = 10
	metric.CounterType = Gauge
	metric.Endpoint = hostname
	// return &openfalconMetric{
	return &openfalconMetric{}
	// }
}

func getHexInode(hexline string) string {
	inode := strings.Fields(hexline)[11]
	return inode
}

func putMetric(msg *message) {
	jsonStr, _ := json.Marshal(msg.Item)
	fmt.Printf("%s", jsonStr)
	req, err := http.NewRequest("POST", APIURL, bytes.NewBuffer([]byte(jsonStr)))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}

func putMetricToFalcon() error {
	for {
		// in, out, lo, err := dothings()
		out, err := dothings()

		if err != nil {
			return err
		}
		var msg message
		// in.genMetrics(&msg)
		out.genMetrics(&msg)
		// lo.genMetrics(&msg)
		// for k, v := range lo.lines {
		// 	fmt.Println(k, v)
		// }
		// fmt.Printf("%v", msg)
		putMetric(&msg)
		fmt.Println("put")
		time.Sleep(time.Second * 10)
	}
}

func getStruct() []SockTabEntry {
	var socket SockTabEntry
	var sockets []SockTabEntry
	for _, v := range getFiled() {
		filed1 := strings.Split(v[3], ":")
		socket.LocalAddr = filed1[0]
		socket.LocalPort = filed1[1]
		filed2 := strings.Split(v[4], ":")
		socket.RemoteAddr = filed2[0]
		socket.RemotePort = filed2[1]
		socket.Process = v[6]
		sockets = append(sockets, socket)
	}
	return sockets
}

func getFiled() [][]string {
	var str [][]string
	var buf bytes.Buffer
	cmd := exec.Command("netstat", "-anp")
	cmd.Stdout = &buf
	err := cmd.Run()
	if err != nil {
		log.Println(err)
	}
	cmd.Wait()
	for {
		var bt byte
		bt = 10
		b, err := buf.ReadString(bt)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if !strings.Contains(b, "ESTABLISHED") {
			continue
		}
		if strings.Contains(b, "tcp6") {
			continue
		}
		if strings.Contains(b, "tcp") {
			str = append(str, strings.Fields(b))
		}
	}
	return str

}
