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
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
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

	conn  *tls.Conn
	r     *bufio.Reader
	seqNo int // instead of UUIDs
}

type Request struct {
	CommuniqueType string
	Header         RequestHeader
	Body           interface{} `json:",omitempty"`
}

type RequestHeader struct {
	ClientTag   string `json:",omitempty"`
	RequestType string `json:",omitempty"`
	URL         string `json:"Url,omitempty"`
}

type Response struct {
	CommuniqueType string
	Header         ResponseHeader
	Body           map[string]any
}

type ResponseHeader struct {
	ClientTag       string
	MessageBodyType string
	StatusCode      string
	URL             string `json:"Url"`
}

type HrefObject struct {
	Href string `json:"href"`
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

func (c *Client) dial() error {
	cert, err := c.loadClientCertificate()
	if err != nil {
		return err
	}

	c.conn, err = tls.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, controlPort), &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	})
	if err != nil {
		return err
	}

	c.r = bufio.NewReader(c.conn)

	return nil
}

func (c *Client) dialPairing() error {
	cert, err := c.loadPairingCertificate()
	if err != nil {
		return err
	}

	c.conn, err = tls.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, pairingPort), &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
	})
	if err != nil {
		return err
	}

	c.r = bufio.NewReader(c.conn)

	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) generateClientTag() string {
	return uuid.NewString()
}

func (c *Client) send(message []byte) error {
	if c.Verbose {
		os.Stderr.WriteString(fmt.Sprintln("===>", string(message)))
	}

	_, err := c.conn.Write(message)
	if err != nil {
		return err
	}

	_, err = c.conn.Write([]byte("\n"))
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) readLine() (string, error) {
	line, err := c.r.ReadString('\n')
	if err != nil {
		return line, err
	}

	if c.Verbose {
		os.Stderr.WriteString(fmt.Sprintln("<===", strings.TrimRight(line, "\n")))
	}

	return line, nil
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
	err := c.dialPairing()
	if err != nil {
		return err
	}
	// May as well clean up, since the connection can't be reused due to
	// the deadline
	defer c.Close()

	// NOTE(ptr): Setting a deadline prevents the connection from being
	// reused
	err = c.conn.SetDeadline(time.Now().Add(2 * time.Minute))
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

	fmt.Println("Push the button on the back of your controller...")

	line, err := c.readLine()
	if err != nil {
		return err
	}

	req := PairRequest{
		Header: RequestHeader{
			RequestType: "Execute",
			URL:         "/pair",
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
	if err != nil {
		return err
	}

	err = c.send(msg)
	if err != nil {
		return err
	}

	line, err = c.readLine()
	if err != nil {
		return err
	}

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

// Get sends a `ReadRequest` communique to the controller.
func (c *Client) Get(path string) (map[string]any, error) {
	fail := func(err error) (map[string]any, error) { return map[string]any{}, err }

	err := c.dial()
	if err != nil {
		return fail(err)
	}
	defer c.Close()

	tag := c.generateClientTag()

	req := Request{
		CommuniqueType: "ReadRequest",
		Header: RequestHeader{
			ClientTag: tag,
			URL:       path,
		},
	}

	msg, err := json.Marshal(req)
	if err != nil {
		return fail(err)
	}

	err = c.send(msg)
	if err != nil {
		return fail(err)
	}

	for {
		line, err := c.readLine()
		if err != nil {
			return fail(err)
		}

		var res Response
		err = json.Unmarshal([]byte(line), &res)
		if err != nil {
			return fail(err)
		}

		if res.CommuniqueType == "ExceptionResponse" && res.Header.ClientTag == tag {
			return fail(fmt.Errorf("received %s: %s", res.Header.StatusCode, res.Body["Message"]))
		}
		if res.CommuniqueType == "ReadResponse" && res.Header.ClientTag == tag {
			if res.Header.StatusCode == "200 OK" {
				return res.Body, nil
			} else {
				return fail(fmt.Errorf("received %s status", res.Header.StatusCode))
			}
		}
	}
}

// Post sends a `CreateRequest` communique to the controller.
func (c *Client) Post(path string, payload any) (map[string]any, error) {
	fail := func(err error) (map[string]any, error) { return map[string]any{}, err }

	err := c.dial()
	if err != nil {
		return fail(err)
	}
	defer c.Close()

	tag := c.generateClientTag()

	req := Request{
		CommuniqueType: "CreateRequest",
		Header: RequestHeader{
			ClientTag: tag,
			URL:       path,
		},
		Body: payload,
	}

	msg, err := json.Marshal(req)
	if err != nil {
		return fail(err)
	}

	err = c.send(msg)
	if err != nil {
		return fail(err)
	}

	for {
		line, err := c.readLine()
		if err != nil {
			return fail(err)
		}

		var res Response
		err = json.Unmarshal([]byte(line), &res)
		if err != nil {
			return fail(err)
		}

		if res.CommuniqueType == "ExceptionResponse" && res.Header.ClientTag == tag {
			return fail(fmt.Errorf("received %s: %s", res.Header.StatusCode, res.Body["Message"]))
		}
		if res.CommuniqueType == "CreateResponse" && res.Header.ClientTag == tag {
			if res.Header.StatusCode == "201 Created" {
				return res.Body, nil
			} else {
				return fail(fmt.Errorf("received %s status", res.Header.StatusCode))
			}
		}
	}
}

type PingResponseBody struct {
	PingResponse PingResponse
}

type PingResponse struct {
	LEAPVersion float32
}

// Ping sends a `ping` request to the controller. If no error is returned, the
// controller responded with a 200 OK status.
func (c *Client) Ping() (PingResponse, error) {
	body, err := c.Get("/server/1/status/ping")
	if err != nil {
		return PingResponse{}, err
	}

	var res PingResponseBody
	err = mapstructure.Decode(body, &res)
	if err != nil {
		return PingResponse{}, err
	}

	return res.PingResponse, nil
}

type DeviceDefinition struct {
	Href string `json:"href"`

	DeviceType   string
	ModelNumber  string
	SerialNumber int

	Name               string
	FullyQualifiedName []string

	AddressedState string

	AssociatedArea HrefObject
	ButtonGroups   []HrefObject
	DeviceRules    []HrefObject
	LinkNodes      []HrefObject
	Parent         HrefObject
}

type OneDeviceDefinition struct {
	Device DeviceDefinition
}

type MultipleDeviceDefinition struct {
	Devices []DeviceDefinition
}

// Devices gets the list of devices this controller knows about.
func (c *Client) Device(id string) (DeviceDefinition, error) {
	body, err := c.Get(fmt.Sprintf("/device/%s", id))
	if err != nil {
		return DeviceDefinition{}, err
	}

	var res OneDeviceDefinition
	err = mapstructure.Decode(body, &res)
	if err != nil {
		return DeviceDefinition{}, err
	}

	return res.Device, nil
}

// Devices gets the list of devices this controller knows about.
func (c *Client) Devices() ([]DeviceDefinition, error) {
	body, err := c.Get("/device")
	if err != nil {
		return []DeviceDefinition{}, err
	}

	var res MultipleDeviceDefinition
	err = mapstructure.Decode(body, &res)
	if err != nil {
		return []DeviceDefinition{}, err
	}

	return res.Devices, nil
}

type MultipleServerDefinition struct {
	Servers []ServerDefinition
}

type ServerDefinition struct {
	Type string
	Href string `json:"href"`

	ProtocolVersion string
	EnableState     string

	Endpoints []struct {
		Port     int
		Protocol string

		AssociatedNetworkInterfaces any
	}
	LEAPProperties struct {
		PairingList HrefObject
	}
	NetworkInterfaces []HrefObject
}

// Servers gets the list of servers this controller knows about. Typically,
// this will just return a single entry for the controller we are connected to.
func (c *Client) Servers() ([]ServerDefinition, error) {
	body, err := c.Get("/server")
	if err != nil {
		return []ServerDefinition{}, err
	}

	var res MultipleServerDefinition
	err = mapstructure.Decode(body, &res)
	if err != nil {
		return []ServerDefinition{}, err
	}

	return res.Servers, nil
}

type OneServerDefinition struct {
	Server ServerDefinition
}

// Server gets information about the specified server.
func (c *Client) Server(id string) (ServerDefinition, error) {
	body, err := c.Get(fmt.Sprintf("/server/%s", id))
	if err != nil {
		return ServerDefinition{}, err
	}

	var res OneServerDefinition
	err = mapstructure.Decode(body, &res)
	if err != nil {
		return ServerDefinition{}, err
	}

	return res.Server, nil
}

type MultipleServiceDefinition struct {
	Services []ServiceDefinition
}

type ServiceProperties struct {
	// Common properties
	DataSummary HrefObject
	Errors      []struct {
		ErrorCode int
		Details   string
	}

	// AutoProgrammer-specific
	EnabledState string

	// HomeKit-specific
	BonjourServiceName string
	MaxAssociations    int

	// Sonos-specific
	FavoriteHousehold SonosHousehold
	Households        []SonosHousehold
}

type SonosHousehold HrefObject

type ServiceDefinition struct {
	Href string `json:"href"`
	Type string

	AlexaProperties          ServiceProperties
	AutoProgrammerProperties ServiceProperties
	GoogleHomeProperties     ServiceProperties
	HomeKitProperties        ServiceProperties
	IFTTTProperties          ServiceProperties
	NestProperties           ServiceProperties
	SonosProperties          ServiceProperties
}

// Services gets the list of 3rd-party services this controller can interface
// with.
func (c *Client) Services() ([]ServiceDefinition, error) {
	body, err := c.Get("/service")
	if err != nil {
		return []ServiceDefinition{}, err
	}

	var res MultipleServiceDefinition
	err = mapstructure.Decode(body, &res)
	if err != nil {
		return []ServiceDefinition{}, err
	}

	return res.Services, nil
}

type ZoneDefinition struct {
	Name        string
	Href        string `json:"href"`
	ControlType string

	Category struct {
		IsLight bool
		Type    string
	}
	Device HrefObject
}

type MultipleZoneDefinition struct {
	Zones []ZoneDefinition
}

type OneZoneDefinition struct {
	Zone ZoneDefinition
}

// Zones gets the list of zones defined on this controller.
func (c *Client) Zones() ([]ZoneDefinition, error) {
	body, err := c.Get("/zone")
	if err != nil {
		return []ZoneDefinition{}, err
	}

	var res MultipleZoneDefinition
	err = mapstructure.Decode(body, &res)
	if err != nil {
		return []ZoneDefinition{}, err
	}

	return res.Zones, nil
}

// Zone gets information about the specified zone.
func (c *Client) Zone(id string) (ZoneDefinition, error) {
	body, err := c.Get(fmt.Sprintf("/zone/%s", id))
	if err != nil {
		return ZoneDefinition{}, err
	}

	var res OneZoneDefinition
	err = mapstructure.Decode(body, &res)
	if err != nil {
		return ZoneDefinition{}, err
	}

	return res.Zone, nil
}

type DimCommand struct {
	CommandType string
	Parameter   []DimCommandParameter
}

type DimCommandParameter struct {
	Type  string
	Value int
}

type DimCommandBody struct {
	Command DimCommand
}

// ZoneDim dims the zone to the provided level.
func (c *Client) ZoneDim(id string, level int) (ZoneDefinition, error) {
	body := DimCommandBody{
		Command: DimCommand{
			CommandType: "GoToLevel",
			Parameter: []DimCommandParameter{
				{
					Type:  "Level",
					Value: level,
				},
			},
		},
	}

	raw, err := c.Post(fmt.Sprintf("/zone/%s/commandprocessor", id), body)
	if err != nil {
		return ZoneDefinition{}, err
	}

	var res OneZoneDefinition
	err = mapstructure.Decode(raw, &res)
	if err != nil {
		return ZoneDefinition{}, err
	}

	return res.Zone, nil
}

type ZoneStatus struct {
	Href string `json:"href"`

	Zone           HrefObject
	Level          int
	StatusAccuracy string
}

type OneZoneStatus struct {
	ZoneStatus ZoneStatus
}

// ZoneStatus gets the current status of the zone.
func (c *Client) ZoneStatus(id string) (ZoneStatus, error) {
	raw, err := c.Get(fmt.Sprintf("/zone/%s/status", id))
	if err != nil {
		return ZoneStatus{}, err
	}

	var res OneZoneStatus
	err = mapstructure.Decode(raw, &res)
	if err != nil {
		return ZoneStatus{}, err
	}

	return res.ZoneStatus, nil
}
