package conquest

import (
	"conquest/utils"
	"encoding/json"
)

func (c *TransactionContext) MarshalJSON() ([]byte, error) {
	var ctx string
	switch c.CtxType {
	case CTX_FINALLY:
		ctx = "FINALLY"
	case CTX_EVERY:
		ctx = "EVERY"
	case CTX_THEN:
		ctx = "THEN"
	}

	return json.Marshal(struct {
		Type         string
		Transactions []*Transaction
		Next         *TransactionContext
	}{
		Type:         ctx,
		Transactions: c.Transactions,
		Next:         c.Next,
	})
}

func (t *Transaction) MarshalJSON() ([]byte, error) {
	var topts string

	reqopts := t.ReqOptions
	if reqopts&CLEAR_COOKIES > 0 {
		topts += "CLEAR_COOKIES "
	} else {
		t.Cookies = utils.MapMerge(t.Cookies, t.conquest.Initials["Cookies"],
			false)
	}

	if reqopts&CLEAR_HEADERS > 0 {
		topts += "CLEAR_HEADERS "
	} else {
		t.Headers = utils.MapMerge(t.Headers, t.conquest.Initials["Headers"],
			false)
	}
	if reqopts&REJECT_COOKIES > 0 {
		topts += "REJECT_COOKIES "
	}

	res := struct {
		Options, Header  string
		Conditions, Body map[string]interface{}
	}{
		Options:    topts,
		Conditions: t.ResConditions,
		Body:       t.Body,
	}

	res.Header = t.Verb + " " + t.Path + " " + t.conquest.Proto + "\r\n"
	res.Header += "Host: " + t.conquest.Host + "\r\n"

	if len(t.Headers) > 0 {
		for k, v := range t.Headers {
			res.Header += k + ": " + v + "\r\n"
		}
	}

	if len(t.Cookies) > 0 {
		res.Header += "Cookie: "
		for k, v := range t.Cookies {
			res.Header += k + "=" + v + "; "
		}
		res.Header += "\r\n"
	}
	return json.Marshal(&res)
}

func (f *FetchNotation) MarshalJSON() ([]byte, error) {
	var kind string
	switch f.Type {
	case FETCH_COOKIE:
		kind = "FROM_COOKIE"
	case FETCH_DISK:
		kind = "FROM_DISK"
	case FETCH_HEADER:
		kind = "FROM_HEADER"
	case FETCH_HTML:
		kind = "FROM_HTML"
	}

	return json.Marshal(struct {
		Type string
		Args []string
	}{
		Type: kind,
		Args: f.Args,
	})
}
