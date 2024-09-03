package rcon

import (
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/gorcon/rcon"
)

type RconInfo struct {
	Map  string `json:"map"`
	IP   string `json:"ip"`
	Port string `json:"port"`
	Pass string `json:"pass"`
}

func RconCommand(m string, c string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\s]+`)
	res := re.ReplaceAllString(c, "")
	cl := strings.ToLower(res)

	data, err := os.ReadFile("config/rcon_config.json")
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}

	var rdata []RconInfo
	err = json.Unmarshal(data, &rdata)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
	}

	for _, rinfo := range rdata {
		if rinfo.Map == m {
			log.Printf("Map: %s\nCommands: %s", rinfo.Map, cl)
			ip := rinfo.IP + ":" + rinfo.Port
			return doRcon(cl, ip, rinfo.Pass)
		}
	}
	return ""
}

func doRcon(c string, s string, p string) string {
	conn, err := rcon.Dial(s, p)
	if err != nil {
		log.Printf("Could not connect: %v", err)
	}
	defer conn.Close()

	response, err := conn.Execute(c)
	if err != nil {
		log.Printf("Error executing: %v", err)
	}

	return response
}

func dummyRcon(m string, c string) string {
	if c == "doexit" {
		return "Exiting... \n "
	}

	if c == "saveworld" {
		return "World Saved \n "
	}

	return ""
}
