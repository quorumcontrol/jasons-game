package handlers

import (
	"github.com/golang/protobuf/proto"
)

type HandlerMessageList []string

func (l HandlerMessageList) Contains(msg proto.Message) bool {
	msgType := proto.MessageName(msg)
	return l.ContainsType(msgType)
}

func (l HandlerMessageList) ContainsType(msgType string) bool {
	for _, msg := range l {
		if msg == msgType {
			return true
		}
	}
	return false
}
