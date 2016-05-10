package main //import "github.com/docker/dockercloud-network-daemon"

import (
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/dockercloud-network-daemon/nodes"
	"github.com/docker/dockercloud-network-daemon/tools"
	"github.com/docker/go-dockercloud/dockercloud"
)

//Event type from dockercloud API
type Event struct {
	Node       string `json:"node,omitempty"`
	Status     string `json:"status"`
	ID         string `json:"id"`
	From       string `json:"from"`
	Time       int64  `json:"time"`
	HandleTime int64  `json:"handletime"`
	ExitCode   string `json:"exitcode"`
}

//DiscoverPeer type to mock during tests
type DiscoverPeer func() error

func nodeEventHandler(eventType string, state string, action string, discoverFunc DiscoverPeer) error {
	if (eventType == "node" && state == "Deployed" && action == "update") || (eventType == "node" && state == "Terminated") {
		err := discoverFunc()
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
			err := nodeEventHandler(event.Type, event.State, event.Action, nodes.DiscoverPeers)
			if err != nil {
				log.Println(err)
			}
			break
		case err := <-e:
			if err.Error() == "401" {
				log.Println("Not authorized")
				time.Sleep(1 * time.Hour)
				break Loop
			}
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
				if strings.TrimSpace(strings.ToLower(err.Error())) == "failed api call: 401 unauthorized" {
					log.Println("Not authorized. Retry in 1 hour")
					time.Sleep(1 * time.Hour)
					break
				}
				log.Print(strings.ToLower(err.Error()))
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
