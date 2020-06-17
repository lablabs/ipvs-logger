package main

import (
	"bufio"
	"crypto/sha256"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

type connectionEntry struct {
	params connectionParams
	expiry int64
}

type connectionParams struct {
	protocol string
	fromIP   string
	fromPort string
	toIP     string
	toPort   string
	destIP   string
	destPort string
	state    string
}

func newConnectionParams(protocol, fromIP, fromPort, toIP, toPort, destIP, destPort, state string) *connectionParams {
	return &connectionParams{
		protocol: protocol,
		fromIP:   fromIP,
		fromPort: fromPort,
		toIP:     toIP,
		toPort:   toPort,
		destIP:   destIP,
		destPort: destPort,
		state:    state,
	}
}

func asSha256(o interface{}) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", o)))

	return fmt.Sprintf("%x", h.Sum(nil))
}

func newConnectionEntry(params *connectionParams, expiry int64) *connectionEntry {
	return &connectionEntry{
		params: *params,
		expiry: expiry,
	}
}

func parseList(data string, prevState map[string]*connectionEntry, filter string) map[string]*connectionEntry {
	connections := make(map[string]*connectionEntry)
	i := 0

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		words := strings.Fields(scanner.Text())
		e := parseRow(words, filter)

		if e == nil {
			continue
		}

		connections[asSha256(e.params)] = e
		if prevConnectionEntry, ok := prevState[asSha256(e.params)]; ok {
			// Conn was reused
			if prevConnectionEntry.expiry < e.expiry {
				i++
				now := time.Now()

				fmt.Printf("%s - from: %s:%s - to %s:%s, dest: %s:%s %d-%d\n", now, e.params.fromIP, e.params.fromPort, e.params.toIP, e.params.toPort, e.params.destIP, e.params.destPort, prevConnectionEntry.expiry, e.expiry)
			}
		}
	}
	return connections
}

func parseRow(words []string, filter string) *connectionEntry {

	// Skip row if it has incorrect length or UDP
	if len(words) != 9 {
		return nil
	}

	// Discard UDP entries
	if words[0] != "TCP" {
		return nil
	}

	// Parse string into params structure
	params := newConnectionParams(words[0], hexToIP(words[1]), hexToPort(words[2]), hexToIP(words[3]), hexToPort(words[4]), hexToIP(words[5]), hexToPort(words[6]), words[7])
	if filter != "" && params.toIP != filter {
		return nil
	}

	expiry, err := strconv.ParseInt(words[8], 10, 64)
	if err != nil {
		panic(err)
	}

	if expiry >= 120 {
		return nil
	}

	return newConnectionEntry(params, expiry)
}

// Convert IP from hex string to string
func hexToIP(s string) string {
	first, _ := strconv.ParseUint(s[:2], 16, 64)
	second, _ := strconv.ParseUint(s[2:4], 16, 64)
	third, _ := strconv.ParseUint(s[4:6], 16, 64)
	fourth, _ := strconv.ParseUint(s[6:8], 16, 64)

	return strconv.Itoa(int(first)) + "." + strconv.Itoa(int(second)) + "." + strconv.Itoa(int(third)) + "." + strconv.Itoa(int(fourth))
}

func hexToPort(s string) string {
	p, _ := strconv.ParseInt(s, 16, 64)
	return strconv.Itoa(int(p))
}

func main() {

	sleepTime := flag.Int("interval", 5, "evaluation interval in seconds")
	filterIP := flag.String("service-ip", "", "IP address of the target service")
	ipVsConnFile := flag.String("path", "/proc/net/ip_vs_conn", "Path to ip_vs_conn file")
	flag.Parse()

	// create a map of entries
	prevState := make(map[string]*connectionEntry)

	for {
		file, err := ioutil.ReadFile(*ipVsConnFile)
		fileContent := string(file)
		if err != nil {
			panic(err)
		}

		prevState = parseList(fileContent, prevState, *filterIP)
		time.Sleep(time.Duration(*sleepTime) * time.Second)
	}

}
