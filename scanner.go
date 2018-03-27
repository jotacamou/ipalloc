package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	xCAT "github.com/jotacamou/xCAT-go"

	"gopkg.in/mgo.v2/bson"
)

type IP struct {
	Id       uint32 `bson:"_id,omitempty"`
	Name     string
	Addr     string
	CIDR     string
	Alive    bool
	Reserved bool
	Lock     bool
	Vlan     string
	PTR      string
}

var waitGroup sync.WaitGroup

func StartScan() {
	client := &xCAT.Client{
		Master:   os.Getenv("XCAT_API_SERVER"),
		Token:    os.Getenv("XCAT_TOKEN"),
		Insecure: true,
	}

	networks, err := client.GetNetworkObjects()
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	var objects interface{}
	err = json.Unmarshal(networks, &objects)
	if err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	var IPs []*IP

	for name, obj := range objects.(map[string]interface{}) {
		vlan := obj.(map[string]interface{})["net"]
		mask := obj.(map[string]interface{})["mask"]
		cidr := GetCIDR(vlan.(string), mask.(string))
		ips, err := GetCIDRIps(cidr)
		if err != nil {
			fmt.Printf("%v\n", err)
			continue
		}
		for _, ip := range ips {
			ip := &IP{Id: ip2int(ip), Name: name, Addr: ip, Vlan: vlan.(string), CIDR: cidr}
			IPs = append(IPs, ip)
		}
	}

	waitGroup.Add(len(IPs))
	concurMax := 100
	scan := make(chan *IP, concurMax)
	scanResult := make(chan *IP, len(IPs))

	for i := 0; i < concurMax; i++ {
		go scanner(scan, scanResult)
	}
	go storeResult(len(IPs), scanResult)
	log.Printf("Scanning %v IP addresses %v at a time ...\n", len(IPs), concurMax)
	start := time.Now()
	for _, ip := range IPs {
		scan <- ip
	}
	waitGroup.Wait()
	elapsed := time.Since(start)
	log.Printf("Finished scan of %v IP addresses in %v\n", len(IPs), elapsed)
}

func scanner(scan <-chan *IP, scanResult chan<- *IP) {
	for ip := range scan {
		_, err := exec.Command("ping", "-c1", "-W1", ip.Addr).Output()
		if err == nil {
			ip.Alive = true
		}
		ptr, _ := net.LookupAddr(ip.Addr)
		if len(ptr) != 0 {
			ip.PTR = ptr[0]
		}
		scanResult <- ip
	}
}

func storeResult(resultNum int, scanResult <-chan *IP) {
	var results []*IP
	var err error
	c := GetCollection()
	for i := 0; i < resultNum; i++ {
		res := <-scanResult
		query := c.FindId(res.Id)
		if queryCount, _ := query.Count(); queryCount == 0 {
			err = c.Insert(res)
		} else {
			// HERE: if alive and has ptr, then reserve?
			selector := bson.M{"_id": res.Id}
			update := bson.M{"$set": bson.M{"alive": res.Alive, "ptr": res.PTR}}
			err = c.Update(selector, update)
		}
		if err != nil {
			log.Println(err)
		}
		results = append(results, res)
		waitGroup.Done()
	}
}

func ip2int(ip interface{}) uint32 {
	ip = net.ParseIP(ip.(string))
	var bin uint32
	binary.BigEndian.PutUint32(ip.(net.IP), bin)
	return binary.BigEndian.Uint32(ip.(net.IP)[12:16])
}

func int2ip(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}

func GetCIDR(vlan string, netmask string) (cidr string) {
	size, _ := net.IPMask(net.ParseIP(netmask).To4()).Size()
	cidr = fmt.Sprintf("%s/%d", vlan, size)
	return
}

func GetCIDRIps(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	//var ips []string
	ips := make([]string, 0)
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	return ips[1 : len(ips)-1], nil
}

//  http://play.golang.org/p/m8TNTtygK0
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
