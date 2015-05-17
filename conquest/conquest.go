package conquest

import (
	"errors"
	"time"
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
	FETCH_HTML
)

type Conquest struct {
	Proto, Host, PemFilePath  string
	TlsInsecure, Sequential   bool
	TotalUsers, TotalRequests int64
	Initials                  map[string]map[string]string
	Duration                  *time.Duration
	Track                     *TransactionContext
}

type Transaction struct {
	conquest         *Conquest
	ReqOptions       uint8
	Verb, Path       string
	Headers, Cookies map[string]string
	Body             map[string]interface{}
	ResConditions    map[string]interface{}
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
		return nil, errors.New("map can not be converted as FetchNotation")
	}

	return &FetchNotation{
		Type: src["type"].(uint8),
		Args: src["args"].([]string),
	}, nil
}
