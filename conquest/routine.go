package conquest

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type mUser struct {
	M       *sync.Mutex
	Cookies map[string]string
	Headers map[string]map[string]string
}

var (
	metaUser mUser = mUser{
		M:       &sync.Mutex{},
		Cookies: map[string]string{},
		Headers: map[string]map[string]string{},
	}
)

// stores caching headers
func storeHeaders(p string, h http.Header) {
	metaUser.M.Lock()
	defer metaUser.M.Unlock()

	for name, values := range h {
		switch name {
		case "Etag", "Last-Modified":
			if _, ok := metaUser.Headers[p]; !ok {
				metaUser.Headers[p] = map[string]string{}
			}

			metaUser.Headers[p][name] = values[0]
		}
	}
}

func storeCookies(cs []*http.Cookie) {
	metaUser.M.Lock()
	defer metaUser.M.Unlock()

	for _, c := range cs {
		// delete cookie
		if c.Value == "" {
			delete(metaUser.Cookies, c.Name)
			continue
		}
		metaUser.Cookies[c.Name] = c.Value
	}
}

// routine of crew members
type dutyRoutine func(chan<- *Success, chan<- *Fail, *sync.WaitGroup)

func buildDutyRoutine(c *http.Client, conquest *Conquest,
	t *Transaction) (dutyRoutine, error) {

	target := conquest.scheme + "://" + conquest.Host + t.Path + "?"
	body := &bytes.Buffer{}

	var carrier *bytes.Buffer
	var boundary string

	switch t.Verb {
	case "POST", "PUT", "PATCH", "DELETE":
		if t.isMultiPart {
			mwriter := multipart.NewWriter(body)
			boundary = mwriter.Boundary()

			for k, d := range t.Body {
				if data, ok := d.(string); ok {
					mwriter.WriteField(k, data)
					continue
				}

				f := d.(*FetchNotation)
				val, err := FetchFrom(f, t.Path, &metaUser)
				if err != nil {
					return nil, errors.New(t.Verb + " " + t.Path + " Error:" + err.Error())
				}

				if f.Type == FETCH_DISK {

					part, err := mwriter.CreateFormFile(k,
						filepath.Base(f.Args[0]))
					if err != nil {
						return nil, err
					}

					if _, err := part.Write(val); err != nil {
						return nil, errors.New(t.Verb + " " + t.Path + " Error:" + err.Error())
					}

				} else {
					mwriter.WriteField(k, string(val))
				}
			}

			if err := mwriter.Close(); err != nil {
				return nil, err
			}
			break
		}
		carrier = body
		fallthrough
	case "GET", "HEAD", "OPTIONS":
		if t.isMultiPart {
			return nil, errors.New(t.Verb + " can not contain multipart data.")
		}

		v := url.Values{}
		for k, d := range t.Body {
			if data, ok := d.(string); ok {
				v.Add(k, data)
				continue
			}

			f := d.(*FetchNotation)
			if strKind, ok := CorrectFetch(FETCH_COOKIE|FETCH_HEADER, f); !ok {
				return nil, errors.New(strKind + " fetch can not be used with " +
					t.Verb + " " + t.Path)
			}

			val, err := FetchFrom(f, t.Path, &metaUser)
			if err != nil {
				return nil, errors.New(t.Verb + " " + t.Path + " Error:" + err.Error())
			}

			v.Add(k, string(val))
		}
		// form values for falled through cases
		if carrier != nil {
			carrier.Write([]byte(v.Encode()))
			break
		}
		// url values
		target += v.Encode()
	}

	manreq, err := http.NewRequest(t.Verb, target, body)
	if err != nil {
		return nil, err
	}

	if t.isMultiPart {
		manreq.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	}
	if carrier != nil {
		manreq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// initial conquest headers
	if t.ReqOptions&CLEAR_HEADERS == 0 {
		for k, v := range conquest.Initials["Headers"] {
			manreq.Header.Add(k, v.(string))
		}
	}

	for k, d := range t.Headers {
		if val, ok := d.(string); ok {
			manreq.Header.Set(k, val)
			continue
		}

		f := d.(*FetchNotation)
		if strKind, ok := CorrectFetch(FETCH_COOKIE|FETCH_HEADER, f); !ok {
			return nil, errors.New(strKind + " fetch can not be used with " +
				t.Verb + " " + t.Path)
		}

		val, err := FetchFrom(f, t.Path, &metaUser)
		if err != nil {
			return nil, errors.New(t.Verb + " " + t.Path + " Error:" + err.Error())
		}
		manreq.Header.Set(k, string(val))
	}

	// initial and stored cookies
	if t.ReqOptions&CLEAR_COOKIES == 0 {
		for k, v := range conquest.Initials["Cookies"] {
			c := &http.Cookie{
				Name:  k,
				Value: v.(string),
			}
			manreq.AddCookie(c)
		}

		for k, v := range metaUser.Cookies {
			c := &http.Cookie{
				Name:  k,
				Value: v,
			}
			manreq.AddCookie(c)
		}
	}

	for k, v := range t.Cookies {
		if val, ok := v.(string); ok {
			c := &http.Cookie{
				Name:  k,
				Value: val,
			}
			manreq.AddCookie(c)
		}

		f := v.(*FetchNotation)
		if strKind, ok := CorrectFetch(FETCH_COOKIE|FETCH_HEADER, f); !ok {
			return nil, errors.New(strKind + " fetch can not be used with " +
				t.Verb + " " + t.Path)
		}

		val, err := FetchFrom(f, t.Path, &metaUser)
		if err != nil {
			return nil, errors.New(t.Verb + " " + t.Path + " Error:" + err.Error())
		}
		manreq.AddCookie(&http.Cookie{Name: k, Value: string(val)})
	}
	
	bodyByte := body.Bytes()

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
		
		req, _ := http.NewRequest(t.Verb, target, bytes.NewBuffer(bodyByte))
		req.Header = manreq.Header
		
		start := time.Now()
		res, err := c.Do(req)
		elapsed := time.Since(start)
		if err != nil {
			panic(NewFail(REASON_TRANSACTION, req.URL.Path, err, elapsed, req))
		}
		defer res.Body.Close()

		// store caching headers
		storeHeaders(req.URL.Path, res.Header)

		resCookies := res.Cookies()
		// store cookies
		if t.ReqOptions&REJECT_COOKIES == 0 {
			storeCookies(resCookies)
		}

		if len(t.ResConditions) == 0 {
			goto SUCCESS_STAT
		}

		// check response conditions
		for k, v := range t.ResConditions {
			switch k {
			case "StatusCode":
				if int64(res.StatusCode) != v.(int64) {
					err := errors.New(
						fmt.Sprintf(
							"Expected status code is %d but it returned as %d.",
							v.(int64), res.StatusCode))
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
