package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

var (
	OptionOutput        string
	OptionFilter        string
	OptionRestfulURL    string
	OptionRestfulMethod string
	OptionRestfulHeader string
	OptionInterval      int
	OptionHelp          bool
)

type Interface struct {
	Name  string   `json:"name"`
	Index int      `json:"index"`
	Flag  string   `json:"flag"`
	Mac   string   `json:"mac"`
	Mtu   int      `json:"mtu"`
	IP    []string `json:"ip"`
}

func init() {
	flag.StringVar(&OptionRestfulURL, "restful-url", "", "call the system's IP address changes to a specified restful API in JSON format.")
	flag.StringVar(&OptionRestfulMethod, "restful-method", "POST", "call specified restful API with the specified method.")
	flag.StringVar(&OptionRestfulHeader, "restful-header", "", "call specified restful API with the specified header, example: \"key:value\".")
	flag.StringVar(&OptionOutput, "output", "ip_info.json", "Save the system's IP address changes to a specified file in JSON format.")
	flag.StringVar(&OptionFilter, "filter", "", "Filter the output by the specified interface name with case-insensitive comparison.")
	flag.IntVar(&OptionInterval, "interval", 60, "The interval between each check, in seconds.")
	flag.BoolVar(&OptionHelp, "help", false, "Usage help")
}

func CallRestFul(jsonData []byte) {
	if OptionRestfulURL == "" {
		return
	}

	req, err := http.NewRequest(OptionRestfulMethod, OptionRestfulURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("new restful request fail, ", err)
		return
	}

	if OptionRestfulHeader != "" {
		header := strings.Split(OptionRestfulHeader, ":")
		if len(header) == 2 {
			req.Header.Set(header[0], header[1])
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", runtime.GOOS+"/"+runtime.GOARCH)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("do restful request fail, ", err)
		return
	}
	defer resp.Body.Close()

	log.Println("restful response status:", resp.Status)
}

func main() {
	flag.Parse()
	if OptionHelp {
		flag.Usage()
		return
	}

	log.SetFlags(log.Lshortfile | log.Ldate | log.Ltime | log.Lmicroseconds)

	var latest []byte
	var count int64

	for {
		if count > 0 {
			time.Sleep(time.Second * time.Duration(OptionInterval))
		}
		count++

		ifaces, err := net.Interfaces()
		if err != nil {
			log.Println(err.Error())
			continue
		}

		lookup := make([]Interface, 0)

		for _, iface := range ifaces {
			var info Interface

			if OptionFilter != "" && !strings.EqualFold(iface.Name, OptionFilter) {
				continue
			}

			info.Name = iface.Name
			info.Index = iface.Index
			info.Mac = iface.HardwareAddr.String()
			info.Mtu = iface.MTU
			info.Flag = iface.Flags.String()

			address, err := iface.Addrs()
			if err != nil {
				log.Println(err.Error())
			} else {
				for _, ip := range address {
					info.IP = append(info.IP, ip.String())
				}
			}
			lookup = append(lookup, info)
		}

		body, err := json.MarshalIndent(lookup, "", "\t")
		if err != nil {
			log.Println(err.Error())
			continue
		}

		if bytes.Equal(body, latest) {
			continue
		}

		latest = body
		err = os.WriteFile(OptionOutput, body, 0644)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		CallRestFul(latest)

		log.Println("ip lookup success")
	}
}
