package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	post   message
	metric openfalconMetric
	file   = "/proc/net/tcp"
)

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

func getAddressAndPortWithoutHex(s string) (address string, port string) {
	if !strings.Contains(s, ":") {
		return "", ""
	}
	address = strings.Split(s, ":")[0]
	port = strings.Split(s, ":")[1]
	return convHex(address), convHex(port)
}

func (local localConLines) genMetrics(msg *message) *message {
	// m := newOpenFalconMetric(&metric)
	var metric openfalconMetric
	metric.Metric = "LocalConnection"
	for key, value := range local.lines {
		metric.Tags = key
		metric.Value = value - 1
		msg.Item = append(msg.Item, metric)
	}
	return msg
}

func (out outConLines) genMetrics(msg *message) *message {
	var metric openfalconMetric
	metric.Metric = "OutConnection"
	for key, value := range out.lines {
		metric.Tags = key
		metric.Value = value - 1
		msg.Item = append(msg.Item, metric)
	}
	return msg
}
func (in inConLines) genMetrics(msg *message) *message {
	var metric openfalconMetric
	metric.Metric = "InConnection"
	for key, value := range in.lines {
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
func getLines(file string) (inConLines, outConLines, localConLines, int, error) {
	var in inConLines
	var out outConLines
	var lo localConLines
	in.init()
	out.init()
	lo.init()

	f, err := ioutil.ReadFile(file)
	if err != nil {
		return in, out, lo, 0, err
	}
	result := GetETCDKeyValues()
	for k, v := range result {
		fmt.Println(k, v)
	}
	buf := bufio.NewReader(bytes.NewBuffer(f))
	var sumConn = 0
	for {
		bline, _, err := buf.ReadLine()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return in, out, lo, 0, err
		}
		sumConn++
		sline := fmt.Sprintf("%s", bline)
		if strings.Contains(sline, "local_address") {
			continue
		}
		s := strings.Fields(sline)
		local := s[1]
		remote := s[2]
		// local := strings.Split(s[1], ":")
		// remote := strings.Split(s[2], ":")
		// fmt.Println(local, remote)
		_, lport := getAddressAndPortWithoutHex(local)
		raddr, rport := getAddressAndPortWithoutHex(remote)
		// fmt.Printf("localPort=%s\nremoteAddr=%s,remotePort=%s\n", lport, raddr, rport)
		// fmt.Println(lo.lines[result[rport]])

		if result[raddr] == "localhost" {
			port := result[rport]
			lo.lines[port]++
		}

		if _, ok := result[lport]; !ok {
			if _, ok := result[raddr]; !ok {
				fmt.Println(lport, raddr, "1")
				continue
			}
			fmt.Println(rport, raddr, "2")
			remoteInfo := result[raddr] + ":" + rport
			remoteinfo := result[remoteInfo]
			out.lines[remoteinfo]++
			continue
		}
		if _, ok := result[rport]; !ok {
			if _, ok := result[raddr]; !ok {
				continue
			}
			addr := result[raddr]
			out.lines[addr]++
			continue
		}
		in.lines[result[lport]]++

	}
	for k, v := range lo.lines {
		fmt.Printf("key:%s,value:%d\n", k, v)
	}
	return in, out, lo, sumConn, nil
}
func convHex(hex string) string {
	switch len(hex) {
	case 13:
		if !strings.Contains(hex, ":") {
			return ""
		}
		s1 := strings.Split(hex, ":")
		s := convHex(s1[0]) + convHex(s1[1])
		return s
	case 2:
		netstat, _ := strconv.ParseUint(hex, 16, 32)
		// if err != nil {
		// 	return "", err
		// }
		return fmt.Sprintf("%d", uint32(netstat))
	case 4:
		port, _ := strconv.ParseUint(hex, 16, 32)
		// if err != nil {
		// 	return "", err
		// }
		return fmt.Sprintf("%d", uint32(port))
	case 8:
		// 获取到的16进制字符串转换后的十进制字符串与一般的ip地址互为反转
		// 例如 127.0.0.1 的16进制字符串转换后的ip地址为1.0.0.127
		d, _ := strconv.ParseUint(hex[0:2], 16, 32)
		c, _ := strconv.ParseUint(hex[2:4], 16, 32)
		b, _ := strconv.ParseUint(hex[4:6], 16, 32)
		a, _ := strconv.ParseUint(hex[6:8], 16, 32)
		// if err != nil {
		// 	return "", err
		// }
		ipad := fmt.Sprintf("%d.%d.%d.%d", uint32(a), uint32(b), uint32(c), uint32(d))
		return ipad
	default:
		return ""
	}
}
func putMetric(msg *message) {
	jsonStr, _ := json.Marshal(msg.Item)
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
		in, out, lo, _, err := getLines(File)
		if err != nil {
			return err
		}
		var msg message
		in.genMetrics(&msg)
		out.genMetrics(&msg)
		lo.genMetrics(&msg)
		putMetric(&msg)
		// for k, v := range in.lines {
		// 	fmt.Println(k, v)
		// }

		// log.Println(msg)
		fmt.Println("put")
		time.Sleep(time.Second * 10)
	}
}
