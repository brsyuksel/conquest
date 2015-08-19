package conquest

import (
	"errors"
	"time"
)

const (
	VERSION string = "0.1.0b"
)

const (
	CLEAR_COOKIES uint8 = 1 << iota
	CLEAR_HEADERS
	REJECT_COOKIES
)
const (
	CTX_EVERY uint8 = 1 << iota
	CTX_THEN
	CTX_FINALLY
)

const (
	FETCH_HEADER uint8 = 1 << iota
	FETCH_COOKIE
	FETCH_DISK
	//FETCH_HTML
)

type Conquest struct {
	Proto, Host, scheme string
	Sequential          bool
	TotalUsers          uint64
	Initials            map[string]map[string]interface{}
	Duration            time.Duration
	Track               *TransactionContext
}

func NewConquest() *Conquest {
	c := &Conquest{
		Proto:    "HTTP/1.1",
		Initials: map[string]map[string]interface{}{},
		Duration: time.Duration(time.Minute * 1),
	}
	return c
}

type Transaction struct {
	conquest                              *Conquest
	ReqOptions                            uint8
	isMultiPart, Skip                     bool
	Verb, Path                            string
	Headers, Cookies, Body, ResConditions map[string]interface{}
}

type TransactionContext struct {
	CtxType      uint8
	Transactions []*Transaction
	Next         *TransactionContext
}

type FetchNotation struct {
	Type uint8
	Args []string
}

func mapToFetchNotation(src map[string]interface{}) (*FetchNotation, error) {
	_, e1 := src["type"]
	_, e2 := src["args"]

	if !e1 || !e2 {
		return nil, errors.New("map can not be converted to FetchNotation")
	}

	return &FetchNotation{
		Type: src["type"].(uint8),
		Args: src["args"].([]string),
	}, nil
}
