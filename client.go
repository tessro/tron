package main

import (
	"bufio"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"time"
)

const controlPort = 8081
const pairingPort = 8083

// Client is a Lutron Caséta LEAP API client.
type Client struct {
	Host string

	CACertPath     string
	ClientCertPath string
	ClientKeyPath  string

	Verbose bool

	client http.Client
	seqNo  int // instead of UUIDs
}

type RequestHeader struct {
	ClientTag   string
	RequestType string
	Url         string
}

func (c Client) loadClientCertificate() (tls.Certificate, error) {
	clientCert, err := os.ReadFile(c.ClientCertPath)
	if err != nil {
		return tls.Certificate{}, err
	}
	clientKey, err := os.ReadFile(c.ClientKeyPath)
	if err != nil {
		return tls.Certificate{}, err
	}
	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}

func (c *Client) loadPairingCertificate() (tls.Certificate, error) {
	const clientCert = `-----BEGIN CERTIFICATE-----
MIIECjCCAvKgAwIBAgIBAzANBgkqhkiG9w0BAQ0FADCBlzELMAkGA1UEBhMCVVMx
FTATBgNVBAgTDFBlbm5zeWx2YW5pYTElMCMGA1UEChMcTHV0cm9uIEVsZWN0cm9u
aWNzIENvLiwgSW5jLjEUMBIGA1UEBxMLQ29vcGVyc2J1cmcxNDAyBgNVBAMTK0Nh
c2V0YSBMb2NhbCBBY2Nlc3MgUHJvdG9jb2wgQ2VydCBBdXRob3JpdHkwHhcNMTUx
MDMxMDAwMDAwWhcNMzUxMDMxMDAwMDAwWjB+MQswCQYDVQQGEwJVUzEVMBMGA1UE
CBMMUGVubnN5bHZhbmlhMSUwIwYDVQQKExxMdXRyb24gRWxlY3Ryb25pY3MgQ28u
LCBJbmMuMRQwEgYDVQQHEwtDb29wZXJzYnVyZzEbMBkGA1UEAxMSQ2FzZXRhIEFw
cGxpY2F0aW9uMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAyAOELqTw
WNkF8ofSYJ9QkOHAYMmkVSRjVvZU2AqFfaZYCfWLoors7EBeQrsuGyojqxCbtRUd
l2NQrkPrGVw9cp4qsK54H8ntVadNsYi7KAfDW8bHQNf3hzfcpe8ycXcdVPZram6W
pM9P7oS36jV2DLU59A/OGkcO5AkC0v5ESqzab3qaV3ZvELP6qSt5K4MaJmm8lZT2
6deHU7Nw3kR8fv41qAFe/B0NV7IT+hN+cn6uJBxG5IdAimr4Kl+vTW9tb+/Hh+f+
pQ8EzzyWyEELRp2C72MsmONarnomei0W7dVYbsgxUNFXLZiXBdtNjPCMv1u6Znhm
QMIu9Fhjtz18LwIDAQABo3kwdzAJBgNVHRMEAjAAMB0GA1UdDgQWBBTiN03yqw/B
WK/jgf6FNCZ8D+SgwDAfBgNVHSMEGDAWgBSB7qznOajKywOtZypVvV7ECAsgZjAL
BgNVHQ8EBAMCBaAwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMA0GCSqG
SIb3DQEBDQUAA4IBAQABdgPkGvuSBCwWVGO/uzFEIyRius/BF/EOZ7hMuZluaF05
/FT5PYPWg+UFPORUevB6EHyfezv+XLLpcHkj37sxhXdDKB4rrQPNDY8wzS9DAqF4
WQtGMdY8W9z0gDzajrXRbXkYLDEXnouUWA8+AblROl1Jr2GlUsVujI6NE6Yz5JcJ
zDLVYx7pNZkhYcmEnKZ30+ICq6+0GNKMW+irogm1WkyFp4NHiMCQ6D2UMAIMfeI4
xsamcaGquzVMxmb+Py8gmgtjbpnO8ZAHV6x3BG04zcaHRDOqyA4g+Xhhbxp291c8
B31ZKg0R+JaGyy6ZpE5UPLVyUtLlN93V2V8n66kR
-----END CERTIFICATE-----`

	const clientKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAyAOELqTwWNkF8ofSYJ9QkOHAYMmkVSRjVvZU2AqFfaZYCfWL
oors7EBeQrsuGyojqxCbtRUdl2NQrkPrGVw9cp4qsK54H8ntVadNsYi7KAfDW8bH
QNf3hzfcpe8ycXcdVPZram6WpM9P7oS36jV2DLU59A/OGkcO5AkC0v5ESqzab3qa
V3ZvELP6qSt5K4MaJmm8lZT26deHU7Nw3kR8fv41qAFe/B0NV7IT+hN+cn6uJBxG
5IdAimr4Kl+vTW9tb+/Hh+f+pQ8EzzyWyEELRp2C72MsmONarnomei0W7dVYbsgx
UNFXLZiXBdtNjPCMv1u6ZnhmQMIu9Fhjtz18LwIDAQABAoIBAQCXDtDNyZQcBgwP
17RzdN8MDPOWJbQO+aRtES2S3J9k/jSPkPscj3/QDe0iyOtRaMn3cFuor4HhzAgr
FPCB/sAJyJrFRX9DwuWUQv7SjkmLOhG5Rq9FsdYoMXBbggO+3g8xE8qcX1k2r7vW
kDW2lRnLDzPtt+IYxoHgh02yvIYnPn1VLuryM0+7eUrTVmdHQ1IGS5RRAGvtoFjf
4QhkkwLzZzCBly/iUDtNiincwRx7wUG60c4ZYu/uBbdJKT+8NcDLnh6lZyJIpGns
jjZvvYA9kgCB2QgQ0sdvm0rA31cbc72Y2lNdtE30DJHCQz/K3X7T0PlfR191NMiX
E7h2I/oBAoGBAPor1TqsQK0tT5CftdN6j49gtHcPXVoJQNhPyQldKXADIy8PVGnn
upG3y6wrKEb0w8BwaZgLAtqOO/TGPuLLFQ7Ln00nEVsCfWYs13IzXjCCR0daOvcF
3FCb0IT/HHym3ebtk9gvFY8Y9AcV/GMH5WkAufWxAbB7J82M//afSghPAoGBAMys
g9D0FYO/BDimcBbUBpGh7ec+XLPaB2cPM6PtXzMDmkqy858sTNBLLEDLl+B9yINi
FYcxpR7viNDAWtilVGKwkU3hM514k+xrEr7jJraLzd0j5mjp55dnmH0MH0APjEV0
qum+mIJmWXlkfKKIiIDgr6+FwIiF5ttSbX1NwnYhAoGAMRvjqrXfqF8prEk9xzra
7ZldM7YHbEI+wXfADh+En+FtybInrvZ3UF2VFMIQEQXBW4h1ogwfTkn3iRBVje2x
v4rHRbzykjwF48XPsTJWPg2E8oPK6Wz0F7rOjx0JOYsEKm3exORRRhru5Gkzdzk4
lok29/z8SOmUIayZHo+cV88CgYEAgPsmhoOLG19A9cJNWNV83kHBfryaBu0bRSMb
U+6+05MtpG1pgaGVNp5o4NxsdZhOyB0DnBL5D6m7+nF9zpFBwH+s0ftdX5sg/Rfs
1Eapmtg3f2ikRvFAdPVf7024U9J4fzyqiGsICQUe1ZUxxetsumrdzCrpzh80AHrN
bO2X4oECgYEAxoVXNMdFH5vaTo3X/mOaCi0/j7tOgThvGh0bWcRVIm/6ho1HXk+o
+kY8ld0vCa7VvqT+iwPt+7x96qesVPyWQN3+uLz9oL3hMOaXCpo+5w8U2Qxjinod
uHnNjMTXCVxNy4tkARwLRwI+1aV5PMzFSi+HyuWmBaWOe19uz3SFbYs=
-----END RSA PRIVATE KEY-----`

	cert, err := tls.X509KeyPair([]byte(clientCert), []byte(clientKey))
	if err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}

// Pair pairs with a Lutron Caséta LEAP controller. This requires the user to
// press the pairing button on the controller. After pairing, the client
// certificate is written to the config file.
func (c *Client) Pair() error {
	cert, err := c.loadPairingCertificate()
	if err != nil {
		return err
	}

	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, pairingPort), &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.SetDeadline(time.Now().Add(2 * time.Minute))
	if err != nil {
		return err
	}

	type PairRequestParameters struct {
		CSR         string
		DeviceUID   string
		DisplayName string
		Role        string
	}

	type PairRequestBody struct {
		CommandType string
		Parameters  PairRequestParameters
	}

	type PairRequest struct {
		Body   PairRequestBody
		Header RequestHeader
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// TODO: configure file path
	w, err := os.OpenFile(c.ClientKeyPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = pem.Encode(w, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	if err != nil {
		return err
	}

	csrCert, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		SignatureAlgorithm: x509.SHA256WithRSA,
		Subject: pkix.Name{
			CommonName: "tron",
		},
	}, priv)
	if err != nil {
		return err
	}

	csr := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrCert,
	})

	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return err
	}
	fmt.Printf("response = %s\n", line)

	req := PairRequest{
		Header: RequestHeader{
			RequestType: "Execute",
			Url:         "/pair",
			ClientTag:   "pair",
		},
		Body: PairRequestBody{
			CommandType: "CSR",
			Parameters: PairRequestParameters{
				CSR:         string(csr),
				DeviceUID:   "000000000000",
				DisplayName: "tron",
				Role:        "Admin",
			},
		},
	}

	msg, err := json.Marshal(req)
	fmt.Printf("request = %s\n", msg)
	if err != nil {
		return err
	}

	_, err = conn.Write(msg)
	if err != nil {
		return err
	}

	_, err = conn.Write([]byte("\n"))
	if err != nil {
		return err
	}

	line, err = r.ReadString('\n')
	if err != nil {
		return err
	}
	fmt.Printf("response = %s\n", line)

	type PairResponse struct {
		Body struct {
			SigningResult struct {
				Certificate     string
				RootCertificate string
			}
		}
	}

	var res PairResponse
	err = json.Unmarshal([]byte(line), &res)
	if err != nil {
		return err
	}

	err = os.WriteFile(c.ClientCertPath, []byte(res.Body.SigningResult.Certificate), 0644)
	if err != nil {
		return err
	}

	err = os.WriteFile(c.CACertPath, []byte(res.Body.SigningResult.RootCertificate), 0644)
	if err != nil {
		return err
	}

	return nil
}

// Ping sends a `ping` request to the controller.
func (c *Client) Ping() error {
	cert, err := c.loadClientCertificate()
	if err != nil {
		return err
	}

	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, controlPort), &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	type PingRequest struct {
		CommuniqueType string
		Header         RequestHeader
	}

	req := PingRequest{
		CommuniqueType: "ReadRequest",
		Header: RequestHeader{
			ClientTag: "ping-1",
			Url:       "/server/1/status/ping",
		},
	}

	json, err := json.Marshal(req)
	fmt.Printf("request = %s\n", json)
	if err != nil {
		return err
	}

	conn.Write(json)
	conn.Write([]byte("\n"))

	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return err
		}
		if c.Verbose {
			fmt.Println("<===", line)
			//fmt.Println("<===", res.Status)
			//if len(responseBody) > 0 {
			//	fmt.Println("<===", string(responseBody))
			//}
			//fmt.Println()
		}
		// TODO: parse client tag and close when it matches
	}
	return nil
}

// Devices gets the list of devices this controller knows about.
func (c *Client) Devices() (string, error) {
	cert, err := c.loadClientCertificate()
	if err != nil {
		return "", err
	}

	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, controlPort), &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	})
	if err != nil {
		return "", err
	}
	defer conn.Close()

	type PingRequest struct {
		CommuniqueType string
		Header         RequestHeader
	}

	req := PingRequest{
		CommuniqueType: "ReadRequest",
		Header: RequestHeader{
			Url: "/device",
		},
	}

	json, err := json.Marshal(req)
	if c.Verbose {
		fmt.Println("ReadRequest", "/device")
		fmt.Println("===>", string(json))
	}
	if err != nil {
		return "", err
	}

	conn.Write(json)
	conn.Write([]byte("\n"))

	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return "", err
		}

		if c.Verbose {
			fmt.Println("<===", line)
			//fmt.Println("<===", res.Status)
			//if len(responseBody) > 0 {
			//	fmt.Println("<===", string(responseBody))
			//}
			//fmt.Println()
		}
	}
	return "result", nil
}
