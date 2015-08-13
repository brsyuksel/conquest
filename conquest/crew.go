package conquest

import (
	"errors"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// transaction getter func builder
// returns a func that it can return a transaction as randomly
// sequential.
func transactionGetter(s bool, t []*Transaction) func() *Transaction {
	c := len(t)

	if s {
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

func chargeCrew(c *[]dutyRoutine, C *reportChannels) {
	var done sync.WaitGroup
	done.Add(len(*c))

	for _, r := range *c {
		go r(C.Success, C.Fail, &done)
	}

	done.Wait()
	c = nil
}

// adds a duty assigned member to crew and performs it with using
// chargeCrew func
func createCrew(client *http.Client, s bool, c *Conquest,
	t []*Transaction, C *reportChannels) error {

	var routines []dutyRoutine
	var timerC <-chan time.Time
	getTransaction := transactionGetter(s, t)

DUTY_LOOP:
	for d := getTransaction(); d != nil; d = getTransaction() {
		if routines == nil {
			routines = []dutyRoutine{}
		}

		routine, err := buildDutyRoutine(client, c, d)
		if err != nil {
			return err
		}

		if s {
			for i := uint64(0); i < c.TotalUsers; i++ {
				routines = append(routines, routine)
			}
			chargeCrew(&routines, C)
			continue
		}

		routines = append(routines, routine)
		if timerC == nil {
			timerC = time.After(c.Duration)
		}

		select {
		case <-timerC:
			routines = nil
		default:
			if uint64(len(routines)) < c.TotalUsers {
				continue DUTY_LOOP
			}
			chargeCrew(&routines, C)
		}

	}

	return nil
}

// creates a crew which contains members with assigned routines
// and runs all.
func Perform(conquest *Conquest, reporter *report) error {
	if conquest.Track == nil {
		return errors.New("Empty transaction stack.")
	}

	httpClient, err := buildHttpClient(conquest.scheme, conquest.PemFilePath,
		conquest.TlsInsecure)

	if err != nil {
		return err
	}

	for track := conquest.Track; track != nil; track = track.Next {
		seq := conquest.Sequential || track.CtxType&(CTX_FINALLY|CTX_EVERY) > 0

		if err := createCrew(httpClient, seq,
			conquest, track.Transactions, reporter.C); err != nil {
			return err
		}
	}
	reporter.C.Done <- true
	return nil
}
