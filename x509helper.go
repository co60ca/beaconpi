package beaconpi

import (
	"crypto/x509"
	"io/ioutil"
)

func LoadFileToCert(file string) *x509.CertPool {
	certs := x509.NewCertPool()
	cert, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}
	if !certs.AppendCertsFromPEM(cert) {
		return nil
	}
	return certs
}
