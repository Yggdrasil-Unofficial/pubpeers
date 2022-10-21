package main

import (
	"os"
	"os/exec"
	"fmt"
	"sort"
	"regexp"
	"bufio"
	"net/url"
	"strings"
	"encoding/json"
	"path/filepath"
	log "github.com/sirupsen/logrus"
)

const (
	SRC_REPO = "https://github.com/yggdrasil-network/public-peers.git"
	TXT_FILE = "./peers.txt"
	JSON_FILE = "./peers.json"
)

func clear(){
	os.RemoveAll("./public-peers")
	os.RemoveAll(TXT_FILE)
	os.RemoveAll(JSON_FILE)
}

func clone() {
	clear()
	exec.Command("git", "clone", SRC_REPO).Output()
}

func comit(){
	log.Error(exec.Command("git", "add", "*").Output())
	log.Error(exec.Command("git", "comit", "-n", "UpdatePeers").Output())
	log.Error(exec.Command("git", "push").Output())
}

type Peer = []string

type Region struct {
	name string
	Peers []Peer
}

func NewRegion(name string) Region {
	return Region {
		name,
		[]Peer{},
	}
}

func (r *Region) Add(peer Peer) {
	r.Peers = append(r.Peers, peer)
}

type Total struct {
	Regions map[string][]Peer
}

func NewTotal() Total {
	return Total{
		make(map[string][]Peer),
	}
}

func (t *Total) AddRegion(region Region) {
	t.Regions[region.name] = region.Peers
}

func (t *Total) ToString() string {
	added := make(map[string]struct{})
	for name := range t.Regions {
		region := t.Regions[name]
		for _, peer := range region {
			for _, p := range peer {
				added[p] = struct{}{}
			}
		}
	}
	peers := []string{}
	for peer := range added {
		peers = append(peers, peer)
	}
	sort.Strings(peers)
	ret := ""
	first := true
	for _, peer := range peers {
		if !first { ret += "\n" }
		ret+=peer
		first = false
	}
	return ret
}

func (t *Total) ToJson() string {
	b, _ := json.MarshalIndent(t.Regions, "", "  ")
	return string(b)
}

func getUri(s string) (string, error) {
	blacklisthosts := []string{
		"localhost",
		"127.0.0.",
		"0.0.0.0",
		"1.2.3.4",
		"peers: []",
	}
	re := regexp.MustCompile("`(.*?)`")
	matches := re.FindStringSubmatch(s)
	outer: for _, m := range matches {
		u, err := url.Parse(m)
		if err != nil { continue }
		if u.Scheme == "socks" {
			nu := "tls://"+u.String()[len("socks://")+len(u.Host)+1:]
			//log.Warn(nu)
			u, err = url.Parse(nu)
			if err != nil { continue }
		}
		ret := u.String()
		for _, b := range blacklisthosts {
			if strings.Contains(ret, b) {
				continue outer
			}
		}
		return ret, nil
	}
	return "", fmt.Errorf("No valid uri here")
}

func write(file, text string) {
	f, _ := os.Create(file)
  	w := bufio.NewWriter(f)
    w.WriteString(text)
    w.Flush()
    f.Close()
}

func main() {
	clone()
	log.Info("Peers cloned")
	//
	total := NewTotal()
	filepath.Walk("./public-peers",
	    func(path string, info os.FileInfo, err error) error {
	    	if err != nil {
	        	return err
	    	}
	    	if info.IsDir() { return nil }
	    	if strings.Contains(path, ".git") { return nil }
	    	if strings.Contains(path, "README") { return nil }
	    	if !strings.Contains(path, ".md") { return nil }
	    	p, _ := filepath.Abs(path)
	    	name := strings.TrimSuffix(filepath.Base(p), ".md")
	  	 	log.Info("[Found peers region] ", name)
	  	 	region := NewRegion(name)
			b, _ := os.ReadFile(path)
			peer := Peer{}
			scanner := bufio.NewScanner(strings.NewReader(string(b)))
			for scanner.Scan() {
				text := scanner.Text()
			    u, e := getUri(text)
			    if e != nil{
			    	if len(peer) > 0 {
			    		//log.Warn(peer)
			    		region.Add(peer)
			    		peer = Peer{}
			    	}
			    	continue
			    }
			    //log.Warn(u)
			    peer = append(peer, u)
			}
			if len(peer) > 0 {
				//log.Warn(peer)
			    region.Add(peer)
			}
			if len(region.Peers) > 0 {
				total.AddRegion(region)
			}
	    	return nil
		},
	)
	clear()
	log.Info("Peers list builded")
	write(TXT_FILE, total.ToString())
	log.Info("peers.txt writed")
	write(JSON_FILE, total.ToJson())
	log.Info("peers.json writed")
	//comit()
	//log.Info("Comited")
}
