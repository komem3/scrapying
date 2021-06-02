package main

import (
	"fmt"
	"hash/fnv"
)

type Comic struct {
	Link string
	ID   string
}

func NewCommic(link string) *Comic {
	h := fnv.New32()
	h.Write([]byte(link))
	return &Comic{
		ID:   fmt.Sprintf("%x", h.Sum(nil)),
		Link: link,
	}
}
