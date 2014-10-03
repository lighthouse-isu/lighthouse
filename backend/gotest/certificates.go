package main

import (
    "crypto/tls"
    "net/http"
    "crypto/x509"
    "io/ioutil"
)

type CertificateList struct {
    CertPem string
    KeyPem string
    CaPem string
}

func CreateDockerTLS(certs CertificateList) *http.Transport {
    cert, _ := tls.LoadX509KeyPair(certs.CertPem, certs.KeyPem)
    ca, _ := ioutil.ReadFile(certs.CaPem)

    cpool := x509.NewCertPool()
    cpool.AppendCertsFromPEM(ca)

    tlsConfig := &tls.Config {
        Certificates: []tls.Certificate{cert},
        RootCAs: cpool,
    }

    return &http.Transport {TLSClientConfig: tlsConfig}
}