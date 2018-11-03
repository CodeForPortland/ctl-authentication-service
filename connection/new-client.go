/*
Package newclient provides a function to create a new client with the socket
type and serialization specified by command like arguments.  This is used for
all the sample clients.

*/
package connection

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/gammazero/nexus/client"
)

type Addresses struct {
	WsAddr   string
	WssAddr  string
	TcpAddr  string
	TcpsAddr string
	UnixAddr string
}

func NewClient(addrs Addresses, cfg client.Config, scheme string) (*client.Client, error) {

	logger := log.New(os.Stdout, "NewClient> ", log.LstdFlags)

	var skipVerify, compress bool
	var  serType, caFile, certFile, keyFile string
	flag.StringVar(&scheme, "scheme", "ws",
		"-scheme=[ws, wss, tcp, tcps, unix].  Default is ws (websocket no tls)")
	flag.StringVar(&serType, "serialize", "json",
		"-serialize[json, msgpack] or none for socket default")
	flag.BoolVar(&skipVerify, "skipverify", false,
		"accept any certificate presented by the server")
	flag.StringVar(&caFile, "trust", "",
		"CA or self-signed certificate to trust in PEM encoded file")
	flag.StringVar(&certFile, "cert", "",
		"certificate file with PEM encoded data")
	flag.StringVar(&keyFile, "key", "",
		"private key file with PEM encoded data")
	flag.BoolVar(&compress, "compress", false, "enable websocket compression")
	flag.Parse()

	// Get requested serialization.
	serialization := client.JSON
	switch serType {
	case "json":
	case "msgpack":
		serialization = client.MSGPACK
	default:
		return nil, errors.New(
			"invalid serialization, muse be one of: json, msgpack")
	}

	cfg.Serialization = serialization

	if scheme == "https" || scheme == "wss" || scheme == "tcps" {
		// If TLS requested, then set up TLS configuration.
		tlscfg := &tls.Config{
			InsecureSkipVerify: skipVerify,
		}
		// If asked to load a client certificate to present to server.
		if certFile != "" || keyFile != "" {
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err != nil {
				return nil, fmt.Errorf("error loading X509 key pair: %s", err)
			}
			tlscfg.Certificates = append(tlscfg.Certificates, cert)
		}
		// If not skipping verification and told to trust a certificate.
		if !skipVerify && caFile != "" {
			// Load PEM-encoded certificate to trust.
			certPEM, err := ioutil.ReadFile(caFile)
			if err != nil {
				return nil, err
			}
			// Create CertPool containing the certificate to trust.
			roots := x509.NewCertPool()
			if !roots.AppendCertsFromPEM(certPEM) {
				return nil, errors.New("failed to import certificate to trust")
			}
			// Trust the certificate by putting it into the pool of root CAs.
			tlscfg.RootCAs = roots

			// Decode and parse the server cert to extract the subject info.
			block, _ := pem.Decode(certPEM)
			if block == nil {
				return nil, errors.New("failed to decode certificate to trust")
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, err
			}
			log.Println("Trusting certificate", caFile, "with CN:",
				cert.Subject.CommonName)

			// Set ServerName in TLS config to CN from trusted cert so that
			// certificate will validate if CN does not match DNS name.
			tlscfg.ServerName = cert.Subject.CommonName
		}

		cfg.TlsCfg = tlscfg
	}
	if compress {
		cfg.WsCfg.EnableCompression = true
	}
	cfg.WsCfg.EnableTrackingCookie = true

	// Create client with requested transport type.
	var cli *client.Client
	var addr string
	var err error
	switch scheme {
	case "http", "ws":
		addr = fmt.Sprintf("ws://%s/", addrs.WsAddr)
	case "https", "wss":
		addr = fmt.Sprintf("wss://%s/", addrs.WssAddr)
	case "tcp":
		addr = fmt.Sprintf("tcp://%s/", addrs.TcpAddr)
	case "tcps":
		addr = fmt.Sprintf("tcps://%s/", addrs.TcpsAddr)
	case "unix":
		addr = fmt.Sprintf("unix://%s", addrs.UnixAddr)
	default:
		return nil, errors.New("scheme must be one of: http, https, ws, wss, tcp, tcps, unix")
	}

	addr = fmt.Sprintf("ws://%s/", addrs.WsAddr) // added SSL

	cli, err = client.ConnectNet(addr, cfg)
	if err != nil {
		return nil, err
	}

	logger.Println("Connected to", addr, "using", serType, "serialization")
	return cli, nil
}
