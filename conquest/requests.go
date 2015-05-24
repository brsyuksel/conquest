package conquest

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type transactionRoutine func()

type userCookies struct {
	mutex   *sync.Mutex
	Cookies map[string]string
}

var (
	user = userCookies{
		mutex:   &sync.Mutex{},
		Cookies: map[string]string{},
	}
)

type failedTransaction struct {
	Cookie, Body, Header []string
	Response             []string
}

type Results struct {
	Path    string
	Hits    uint64
	AvgTime float64
	Rate    float64
	Fails   []*failedTransaction
}

type responseError struct {
	s string
}

func (r *responseError) Error() string {
	return r.s
}

func buildTransactionRoutine(client *http.Client, conquest *Conquest,
	t *Transaction) (transactionRoutine, error) {

	target := conquest.scheme + "://" + conquest.Host + "/" + t.Path + "?"
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
		/*if !t.isMultiPart {
			fallthrough
		}*/
		carrier = body //
		fallthrough    //
	case "GET":
		fallthrough
	case "HEAD":
		fallthrough
	case "OPTIONS":
		if t.isMultiPart {
			return nil, errors.New(t.Verb + " transaction cannot contain multipart data.")
		}
		v := url.Values{}
		for key, d := range t.Body {
			if data, ok := d.(string); ok {
				v.Add(key, data)
				continue
			}
		}
		if carrier != nil {
			carrier.Write([]byte(v.Encode()))
		} else {
			target += v.Encode()
		}
	}

	req, err := http.NewRequest(t.Verb, target, body)
	if err != nil {
		return nil, err
	}

	if t.ReqOptions&CLEAR_HEADERS == 0 {
		for k, v := range conquest.Initials["Headers"] {
			req.Header.Add(k, v.(string))
		}
	}

	if t.ReqOptions&CLEAR_COOKIES == 0 {
		for k, v := range conquest.Initials["Cookies"] {
			c := &http.Cookie{
				Name:  k,
				Value: v.(string),
			}

			req.AddCookie(c)
		}
	}

	for k, v := range user.Cookies {
		c := &http.Cookie{Name: k, Value: v}
		req.AddCookie(c)
	}

	routine := func() {
		defer func() {
			if r := recover(); r != nil {
				switch r.(type) {
				case error:
					// push failed
				case responseError:
					// push failed
				}
			}
		}()

		// timeit
		res, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer res.Body.Close()

		cookies := res.Cookies()

		if t.ReqOptions&REJECT_COOKIES == 0 {
			user.mutex.Lock()
			for _, c := range cookies {
				user.Cookies[c.Name] = c.Value
			}
			user.mutex.Unlock()
		}

		if len(t.ResConditions) == 0 {
			return
		}
		for k, v := range t.ResConditions {
			switch k {
			case "StatusCode":
				if res.StatusCode != v.(int) {
					panic(responseError{
						s: fmt.Sprintf("Expected StatusCode is %d but it returned as %d.",
							v.(int), res.StatusCode),
					})
				}
			case "Header":
				for header, value := range v.(map[string]string) {
					h := res.Header.Get(header)
					if h == value {
						panic(responseError{
							s: fmt.Sprintf("Expected %s header is %s but it returned as %s.",
								header, value, h),
						})
					}

				}
			case "Cookie":
				expecteds := v.(map[string]string)
				for _, c := range cookies {
					e, ok := expecteds[c.Name]
					if !ok {
						continue
					}

					if e != c.Value {
						panic(responseError{
							s: fmt.Sprintf("Expected %s cookie is %s but it returned as %s.",
								c.Name, e, c.Value),
						})
					}
					delete(expecteds, c.Name)
				}
				for n, _ := range expecteds {
					panic(responseError{
						s: fmt.Sprintf("No cookie named as %s.", n),
					})
				}
			case "Contains":
				body, err := ioutil.ReadAll(res.Body)
				if err != nil {
					panic(err)
				}

				if !strings.Contains(string(body), v.(string)) {
					panic(responseError{
						s: fmt.Sprintf("Response does not contains %s.", v.(string)),
					})
				}

			}
		}

	}

	return routine, nil
}

// returns a func which returns a Transaction pointer
func getTransactionFn(seq bool, t []*Transaction) func() *Transaction {
	c := len(t)

	if seq {
		i := 0
		return func() *Transaction {
			if i == c {
				return nil
			}
			r := t[i]
			i++
			return r
		}
	}

	rand.Seed(time.Now().Unix())
	return func() *Transaction {
		r := rand.Intn(c)
		return t[r]
	}
}

func chargeCrew(crew *[]transactionRoutine) {
	fmt.Println(crew)
	*crew = nil
}

func createCrew(client *http.Client, seq bool, conquest *Conquest,
	transactions []*Transaction) error {

	var crew []transactionRoutine
	var timerC <-chan time.Time
	getTransaction := getTransactionFn(seq, transactions)

CREW_LOOP:
	for t := getTransaction(); t != nil; t = getTransaction() {
		if crew == nil {
			crew = []transactionRoutine{}
		}

		if seq {
			for i := uint64(0); i < conquest.TotalUsers; i++ {
				routine, err := buildTransactionRoutine(client, conquest, t)
				if err != nil {
					return err
				}
				crew = append(crew, routine)
			}
			chargeCrew(&crew)
			continue
		}

		routine, err := buildTransactionRoutine(client, conquest, t)
		if err != nil {
			return err
		}
		crew = append(crew, routine)

		if timerC == nil {
			timerC = time.After(conquest.Duration)
		}

		select {
		case <-timerC:
			crew = nil
		default:
			if uint64(len(crew)) < conquest.TotalUsers {
				continue CREW_LOOP
			}
			chargeCrew(&crew)
		}
	}

	return nil
}

func WalkTrack(conquest *Conquest) error {
	if conquest.Track == nil {
		return errors.New("Empty transaction stack.")
	}

	httpClient, err := buildHttpClient(conquest.scheme,
		conquest.PemFilePath, conquest.TlsInsecure)
	if err != nil {
		return err
	}

	for track := conquest.Track; track != nil; track = track.Next {
		seq := conquest.Sequential || track.CtxType&(CTX_EVERY|CTX_FINALLY) > 0

		err := createCrew(httpClient, seq, conquest, track.Transactions)
		if err != nil {
			return err
		}
	}
	return nil
}
