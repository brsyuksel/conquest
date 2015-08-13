package conquest

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"sync"
)

// routine of crew members
type dutyRoutine func(chan<- *Success, chan<- *Fail, *sync.WaitGroup)

func buildDutyRoutine(c *http.Client, conquest *Conquest,
	t *Transaction) (dutyRoutine, error) {

	target := conquest.scheme + "://" + conquest.Host + t.Path + "?"
	body := &bytes.Buffer{}

	var carrier *bytes.Buffer
	switch t.Verb {
	case "POST":
		fallthrough
	case "PUT":
		fallthrough
	case "PATCH":
		fallthrough
	case "DELETE":
		if t.isMultiPart {
			// multipart ops
			// ...
			break
		}
		carrier = body
		fallthrough
	case "GET":
		fallthrough
	case "HEAD":
		fallthrough
	case "OPTIONS":
		if t.isMultiPart {
			return nil, errors.New(t.Verb + " can not contain multipart data.")
		}

		v := url.Values{}
		for k, d := range t.Body {
			if data, ok := d.(string); ok {
				v.Add(k, data)
			}
		}
		// form values for falled through cases
		if carrier != nil {
			carrier.Write([]byte(v.Encode()))
			break
		}
		// url values
		target += v.Encode()
	}

	req, err := http.NewRequest(t.Verb, target, body)
	if err != nil {
		return nil, err
	}

	// initial headers
	if t.ReqOptions&CLEAR_HEADERS == 0 {
		for k, v := range conquest.Initials["Headers"] {
			req.Header.Add(k, v.(string))
		}
	}

	for k, v := range t.Headers {
		if val, ok := v.(string); ok {
			req.Header.Set(k, val)
			continue
		}
		// FetchNotation Works
	}

	// initial cookies
	if t.ReqOptions&CLEAR_COOKIES == 0 {
		for k, v := range conquest.Initials["Cookies"] {
			c := &http.Cookie{
				Name:  k,
				Value: v.(string),
			}
			req.AddCookie(c)
		}
	}

	for k, v := range t.Cookies {
		if val, ok := v.(string); ok {
			c := &http.Cookie{
				Name:  k,
				Value: val,
			}
			req.AddCookie(c)
		}
		// FetchNotation Works
	}

	// routine func
	routine := func(s chan<- *Success, f chan<- *Fail, d *sync.WaitGroup) {
		defer d.Done()
		// recover panics and generate stats about transactions
		defer func() {
			if r := recover(); r != nil {
				switch r.(type) {
				case *Success:
					s <- r.(*Success)
				case *Fail:
					f <- r.(*Fail)
				}
			}
		}()
		//fmt.Println("routine for:", req.URL.Path)
		start := time.Now()
		res, err := c.Do(req)
		elapsed := time.Since(start)
		if err != nil {
			panic(NewFail(REASON_TRANSACTION, req.URL.Path, err, elapsed, req))
		}
		defer res.Body.Close()

		resCookies := res.Cookies()
		if t.ReqOptions&REJECT_COOKIES == 0 {
			// store cookies
			// ...
		}

		if len(t.ResConditions) == 0 {
			goto SUCCESS_STAT
		}

		// check response conditions
		for k, v := range t.ResConditions {
			switch k {
			case "StatusCode":
				if res.StatusCode != v.(int) {
					err := errors.New(
						fmt.Sprintf(
							"Expected status code is %d but it returned as %d.",
							v.(int), res.StatusCode))
					panic(NewFail(REASON_RESPONSE, req.URL.Path, err, elapsed, req))
				}
			case "Header":
				for name, val := range v.(map[string]string) {
					h := res.Header.Get(name)
					if h != val {
						err := errors.New(
							fmt.Sprintf(
								"Expected %s header value is %s but it returned as %s.",
								name, val, h))
						panic(NewFail(REASON_RESPONSE, req.URL.Path, err, elapsed, req))
					}
				}
			case "Cookie":
				eCookies := v.(map[string]string)
				for _, cookie := range resCookies {
					if _, ok := eCookies[cookie.Name]; !ok {
						continue
					}

					if eCookies[cookie.Name] != cookie.Value {
						err := errors.New(
							fmt.Sprintf("Expected %s cookie value is %s but it returned as %s",
								cookie.Name, eCookies[cookie.Name], cookie.Value))
						panic(NewFail(REASON_RESPONSE, req.URL.Path, err, elapsed, req))
					}
					delete(eCookies, cookie.Name)
				}

				for n, _ := range eCookies {
					err := errors.New(fmt.Sprintf("No cookie named as %s", n))
					panic(NewFail(REASON_RESPONSE, req.URL.Path, err, elapsed, req))
				}
			case "Contains":
				body, err := ioutil.ReadAll(res.Body)
				if err != nil {
					panic(NewFail(REASON_TRANSACTION, req.URL.Path, err, elapsed, req))
				}

				if !strings.Contains(string(body), v.(string)) {
					err := errors.New(fmt.Sprintf("Response does not contain %s.", v.(string)))
					panic(NewFail(REASON_RESPONSE, req.URL.Path, err, elapsed, req))
				}
			}
		}
	SUCCESS_STAT:
		panic(NewSuccess(req.URL.Path, elapsed))
	}
	return routine, nil
}
