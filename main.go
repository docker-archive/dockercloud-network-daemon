package main //import "github.com/docker/dockercloud-network-daemon"

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/docker/dockercloud-network-daemon/nodes"
	"github.com/docker/dockercloud-network-daemon/tools"
	"github.com/docker/go-dockercloud/dockercloud"
)

const (
	Version    = "1.0.3"
	DockerPath = "/usr/local/bin/docker"
)

type Event struct {
	Node       string `json:"node,omitempty"`
	Status     string `json:"status"`
	ID         string `json:"id"`
	From       string `json:"from"`
	Time       int64  `json:"time"`
	HandleTime int64  `json:"handletime"`
	ExitCode   string `json:"exitcode"`
}

func nodeEventHandler(eventType string, state string, action string) error {
	if (eventType == "node" && state == "Deployed" && action == "update") || (eventType == "node" && state == "Terminated") {
		err := nodes.DiscoverPeers()
		if err != nil {
			return err
		}
	}
	return nil
}

func tutumEventHandler(wg *sync.WaitGroup, c chan dockercloud.Event, e chan error) {
Loop:
	for {
		select {
		case event := <-c:
			err := nodeEventHandler(event.Type, event.State, event.Action)
			if err != nil {
				log.Println(err)
			}
			break
		case err := <-e:
			log.Println("[NODE DISCOVERY ERROR]: " + err.Error())
			time.Sleep(10 * time.Second)
			wg.Add(1)
			go discovering(wg)
			break Loop
		}
	}
}

func discovering(wg *sync.WaitGroup) {
	defer wg.Done()
	c := make(chan dockercloud.Event)
	e := make(chan error)

	nodes.DiscoverPeers()

	go dockercloud.Events(c, e)
	tutumEventHandler(wg, c, e)
}

func main() {
	log.Println("===> Start running daemon")
	wg := &sync.WaitGroup{}
	wg.Add(1)

	counter := 0
	if nodes.NodeAPIURI != "" {
		dockercloud.SetUserAgent("network-daemon/" + tools.Version)
	Loop:
		for {
			node, err := dockercloud.GetNode(nodes.NodeAPIURI)
			if err != nil {
				if counter > 100 {
					time.Sleep(time.Duration(counter) * time.Second)
					counter = 0
				} else {
					counter *= 2
					log.Println(err)
					time.Sleep(5 * time.Second)
				}
			} else {
				nodes.Region = node.Region
				nodes.NodePublicIP = node.Public_ip
				nodes.NodeUUID = node.Uuid

				log.Println("===> Posting interface data to database")
				nodes.PostInterfaceData(os.Getenv("DOCKERCLOUD_REST_HOST") + nodes.NodeAPIURI)

				log.Printf("This node IP is %s", nodes.NodePublicIP)
				if os.Getenv("DOCKERCLOUD_AUTH") != "" {
					log.Println("===> Detected Dockecloud API access - starting peer discovery goroutine")
					go discovering(wg)
					break Loop
				}
			}
		}
	}
	wg.Wait()
}
