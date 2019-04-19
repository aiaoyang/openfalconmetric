package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"
)

const (
	Gauge   = "GAUGE"
	Counter = "COUNTER"
	APIURL  = "http://127.0.0.1:1988/v1/push"

	ETCDHOST   = "10.105.33.173:2379"
	ETCDPREFIX = "/zonst/conf/"
	File       = "/proc/net/tcp"
)

type vars interface {
}

func GetETCDKeyValues() map[string]string {
conn:
	cfg := clientv3.Config{
		Endpoints:   []string{ETCDHOST},
		DialTimeout: time.Second * 3,
	}
	cli, err := clientv3.New(cfg)
	if err != nil {
		log.Println(err)
		time.Sleep(time.Second * 3)
		goto conn
	}
	defer cli.Close()
getkey:
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	resp, err := cli.Get(ctx, ETCDPREFIX, clientv3.WithPrefix())
	cancel()
	if err != nil {
		log.Println(err)
		time.Sleep(time.Second * 3)
		goto getkey
	}

	var keyvalues = make(map[string]string)
	for _, ev := range resp.Kvs {
		// 返回 etcd key value的反转键值对 value-key
		keyvalues[fmt.Sprintf("%s", ev.Value)] = singleKey(fmt.Sprintf("%s", ev.Key))
	}
	return keyvalues
}
func singleKey(s string) string {
	result := strings.Split(s, "/")
	return result[len(result)-1]
}
