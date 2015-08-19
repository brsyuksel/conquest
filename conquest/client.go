package conquest

import (
	"crypto/tls"
	"net/http"
)

// creates specified http.Client with provided parameters
func buildHttpClient(scheme string) (*http.Client, error) {
	c := &http.Client{}
	if scheme == "https" {
		c.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		return c, nil
	}
	return c, nil
}
