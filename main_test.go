package main

import (
	"log"
	"testing"
)

func Test_nodeEventHandler(t *testing.T) {
	eventType1 := "node"
	eventType2 := "service"
	state1 := "Deployed"
	state2 := "Deploying"
	state3 := "Terminated"
	action1 := "create"
	action2 := "update"
	Msg := "Failed API call: 404 NOT FOUND"

	err := nodeEventHandler(eventType1, state1, action2)
	err2 := nodeEventHandler(eventType2, state1, action2)
	if err2 != nil {
		log.Println(err)
		t.Error("Expected empty error message, got ", err2.Error())
	}

	err3 := nodeEventHandler(eventType2, state3, action2)
	if err3 != nil {
		t.Error("Expected empty error message, got ", err3.Error())
	}

	err4 := nodeEventHandler(eventType1, state2, action2)
	if err4 != nil {
		t.Error("Expected empty error message, got ", err4.Error())
	}

	err5 := nodeEventHandler(eventType1, state3, action1)

	if err != nil && err5 != nil {
		if err.Error() != Msg || err5.Error() != Msg {
			t.Error("Expected error, got ", err.Error())
		}
	}
}
