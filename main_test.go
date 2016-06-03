package main

import (
	"errors"
	"log"
	"testing"
)

func MockUpDiscoverPeer() error {
	return errors.New("DiscoverPeer function called")
}

func Test_nodeEventHandler(t *testing.T) {
	eventType1 := "node"
	eventType2 := "service"
	state1 := "Deployed"
	state2 := "Deploying"
	state3 := "Terminated"
	action1 := "create"
	action2 := "update"
	Msg := "DiscoverPeer function called"

	err := nodeEventHandler(eventType1, state1, action2, MockUpDiscoverPeer)
	if err != nil {
		log.Println(err)
	}
	err2 := nodeEventHandler(eventType2, state1, action2, MockUpDiscoverPeer)
	if err2 != nil {
		log.Println(err2)
	}

	err3 := nodeEventHandler(eventType2, state3, action2, MockUpDiscoverPeer)
	if err3 != nil {
		log.Println(err3)
	}

	err4 := nodeEventHandler(eventType1, state2, action2, MockUpDiscoverPeer)
	if err4 != nil {
		log.Println(err4)
	}

	err5 := nodeEventHandler(eventType1, state3, action1, MockUpDiscoverPeer)
	if err5 != nil {
		log.Println(err5)
	}

	if err2 != nil {
		t.Error("Expected empty error message, got ", err2.Error())
	}
	if err3 != nil {
		t.Error("Expected empty error message, got ", err3.Error())
	}
	if err4 != nil {
		t.Error("Expected empty error message, got ", err4.Error())
	}

	if err != nil && err5 != nil {
		if err.Error() != Msg || err5.Error() != Msg {
			t.Error("Expected error, got ", err.Error())
		}
	}
}
