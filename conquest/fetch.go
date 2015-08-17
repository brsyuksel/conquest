package conquest

import (
	"errors"
	"io/ioutil"
	"os"
)

var (
	fcache = map[string][]byte{}
)

func fromCookie(args []string, p string, u *mUser) ([]byte, error) {
	key := args[0]
	if key == "" {
		return nil, errors.New("Invalid cookie name: " + key)
	}
	if val, ok := u.Cookies[key]; ok {
		return []byte(val), nil
	}
	return nil, errors.New("Non-exists cookie: " + key)
}

func fromHeader(args []string, p string, u *mUser) ([]byte, error) {
	headers, ok := u.Headers[p]
	if !ok {
		return nil, errors.New("No headers for " + p)
	}
	key := args[0]
	if val, ok := headers[key]; ok {
		return []byte(val), nil
	}
	return nil, errors.New("No " + key + " cached header for " + p)
}

/* FIXME: file caching */
func fromDisk(args []string, p string, u *mUser) ([]byte, error) {
	fpath := args[0]

	if data, ok := fcache[fpath]; ok {
		return data, nil
	}
	
	finfo, err := os.Stat(fpath)
	if err != nil {
		return nil, err
	}

	if finfo.IsDir() {
		return nil, errors.New("You must provide a regular file")
	}

	content, err := ioutil.ReadFile(fpath)
	if err != nil {
		return nil, err
	}
	fcache[fpath] = content

	return content, nil
}

func FetchFrom(f *FetchNotation, path string, u *mUser) (b []byte, e error) {
	switch f.Type {
	case FETCH_COOKIE:
		b, e = fromCookie(f.Args, path, u)
	case FETCH_HEADER:
		b, e = fromHeader(f.Args, path, u)
	case FETCH_DISK:
		b, e = fromDisk(f.Args, path, u)
	}
	return
}

func CorrectFetch(s uint8, f *FetchNotation) (string, bool) {
	var strKind string
	switch f.Type {
	case FETCH_COOKIE:
		strKind = "Cookie"
	case FETCH_HEADER:
		strKind = "Header"
	case FETCH_DISK:
		strKind = "Disk"
	}
	return strKind, s&f.Type != 0
}
