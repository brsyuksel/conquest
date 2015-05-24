package conquest

import (
	"net/http"
	"crypto/tls"
	"os"
	"io/ioutil"
	"crypto/x509"
	"errors"
)

// creates specified http.Client with provided parameters
func buildHttpClient(scheme, pem string, insecure bool) (*http.Client, error) {
	c := &http.Client{}
	if scheme == "https" {
		if insecure {
			c.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}

			return c, nil
		}

		if _, err := os.Stat(pem); err != nil {
			return nil, err
		}

		pemData, err := ioutil.ReadFile(pem)
		if err != nil {
			return nil, err
		}

		tlsConfig := &tls.Config{RootCAs: x509.NewCertPool()}
		if ok := tlsConfig.RootCAs.AppendCertsFromPEM(pemData); !ok {
			return nil, errors.New("Could not added certs from pem data.")
		}
		c.Transport = &http.Transport{TLSClientConfig: tlsConfig}
	}
	return c, nil
}