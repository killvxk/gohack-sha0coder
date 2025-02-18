package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
	"sync"
)

var TIMEOUT *time.Duration
var VERBOSE *bool
var mtx sync.Mutex

func end(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}

func check(err error, msg string) {
	if err != nil {
		fmt.Println(msg)
		os.Exit(1)
	}
}

func wait() {
	var i int
	fmt.Println("Press enter to stop")
	fmt.Scanf("%d", &i)
}

func checkPort(hostport string) (bool, string) {
	conn, err := net.DialTimeout("tcp", hostport, *TIMEOUT)
	if err != nil {
		//panic(err)
		return false, ""
	}
	defer conn.Close()

	fmt.Fprintf(conn, "TEST\n\n")
	buff := make([]byte, 1024)
	conn.Read(buff)
	banner := string(buff)

	if  buff[0] != 0 {
		return true, banner
	}
	return false, banner
}

func expandPorts(port *string) *[]string {
	ports := &[]string{}

	for _, p := range strings.Split(*port, ",") {
		if strings.Contains(p, "-") {
			spl := strings.Split(p, "-")
			start, _ := strconv.Atoi(spl[0])
			end, _ := strconv.Atoi(spl[1])
			for i := start; i <= end; i++ {
				*ports = append(*ports, strconv.Itoa(i))
			}
		} else {
			*ports = append(*ports, p)
		}
	}

	return ports
}

func expandHosts(host *string) *[]string {
	hosts := &[]string{}

	if !strings.Contains(*host, "-") {
		*hosts = append(*hosts, *host)
		return hosts
	}

	spl := strings.Split(*host, "-")

	sIP := ip2octet(spl[0])
	eIP := ip2octet(spl[1])

	for {

		ip := octet2ip(sIP)
		*hosts = append(*hosts, ip)

		if equalOctets(sIP, eIP) {
			break
		}

		sIP = incOctet(sIP)
	}
	return hosts
}

func scan(c <-chan string) {
	for hostport := range c {
		if hostport == "EOF" {
			time.Sleep(*TIMEOUT)
			os.Exit(1)
		}
		isOpen, banner := checkPort(hostport)
		if isOpen {
			fmt.Printf("%s Open [%s]\n", hostport, banner)
		} /*else {
			fmt.Printf("%s Closed [%s]\n", hostport, banner)
		}*/
	}
}

func main() {
	port := flag.String("p", "", "ports ex: -p 80,81  ex: -p 0-80")
	fullMode := flag.Bool("full", false, "scan all the 65535 ports")
	lowMode := flag.Bool("low", false, "scan the 1024 lower ports")
	host := flag.String("h", "127.0.0.1", "hosts ex: -h 192.168.1.16-192.168.1.17")
	gor := flag.Int("go",5,"goroutines")
	TIMEOUT = flag.Duration("t", 4*time.Second, "timeout in seconds")
	flag.Parse()

	if len(*host) <= 0 ||
		(len(*port) <= 0 && !*fullMode && !*lowMode) {
		fmt.Println("try -h")
		return
	}

	fmt.Println("buffering ...")
	ports := expandPorts(port)
	hosts := expandHosts(host)
	fmt.Println("scanning ...")

	if *lowMode {
		for i := 0; i < 1024; i++ {
			port := strconv.Itoa(i)
			*ports = append(*ports, port)
		}
	} else if *fullMode {
		for i := 0; i < 65535; i++ {
			port := strconv.Itoa(i)
			*ports = append(*ports, port)
		}
	}

	c := make(chan string, 1)

	// feed the channel
	go func (hosts []string, ports []string, c chan string, gor int) {
		for _, h := range hosts {
			for _, p := range ports {
				c <- fmt.Sprintf("%s:%s",h,p)
			}
		}
		c <- "EOF"
		close(c)
	}(*hosts,*ports,c,*gor)

	// launch goroutines
	for i:=0; i<*gor; i++ {
		go scan(c)
	}

	fmt.Println("waiting ...")
	wait()
}
