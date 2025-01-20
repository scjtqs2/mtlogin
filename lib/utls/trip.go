package utls

import (
	"bufio"
	"fmt"
	"github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
	"net"
	"net/http"
	"time"
)

type UTransport struct {
	Tr1 *http.Transport
	Tr2 *http2.Transport
}

func (u *UTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme == "http" {
		return u.Tr1.RoundTrip(req)
	} else if req.URL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme: %s", req.URL.Scheme)
	}

	port := req.URL.Port()
	if port == "" {
		port = "443"
	}

	// TCP connection
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", req.URL.Hostname(), port), 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("net.DialTimeout error: %+v", err)
	}

	// TLS connection
	uConn := tls.UClient(conn, &tls.Config{ServerName: req.URL.Hostname()}, tls.HelloCustom)
	if err = uConn.ApplyPreset(u.newSpec()); err != nil {
		return nil, fmt.Errorf("uConn.ApplyPreset() error: %+v", err)
	}
	if err = uConn.Handshake(); err != nil {
		return nil, fmt.Errorf("uConn.Handshake() error: %+v", err)
	}

	alpn := uConn.ConnectionState().NegotiatedProtocol
	switch alpn {
	case "h2":
		req.Proto = "HTTP/2.0"
		req.ProtoMajor = 2
		req.ProtoMinor = 0

		if c, err := u.Tr2.NewClientConn(uConn); err == nil {
			return c.RoundTrip(req)
		} else {
			return nil, fmt.Errorf("http2.Transport.NewClientConn() error: %+v", err)
		}

	case "http/1.1", "":
		req.Proto = "HTTP/1.1"
		req.ProtoMajor = 1
		req.ProtoMinor = 1

		if err := req.Write(uConn); err == nil {
			return http.ReadResponse(bufio.NewReader(uConn), req)
		} else {
			return nil, fmt.Errorf("http.Request.Write() error: %+v", err)
		}

	default:
		return nil, fmt.Errorf("unsupported ALPN: %v", alpn)
	}
}

//	func (u *UTransport) newSpec() *tls.ClientHelloSpec {
//		spec, _ := tls.UTLSIdToSpec(tls.HelloChrome_Auto)
//		return &spec
//	}
func (*UTransport) newSpec() *tls.ClientHelloSpec {
	return &tls.ClientHelloSpec{
		TLSVersMax:         tls.VersionTLS13,
		TLSVersMin:         tls.VersionTLS12,
		CipherSuites:       []uint16{tls.GREASE_PLACEHOLDER, 0x1301, 0x1302, 0x1303, 0xc02b, 0xc02f, 0xc02c, 0xc030, 0xcca9, 0xcca8, 0xc013, 0xc014, 0x009c, 0x009d, 0x002f, 0x0035},
		CompressionMethods: []uint8{0x0}, // no compression
		Extensions: []tls.TLSExtension{
			&tls.UtlsGREASEExtension{},
			&tls.SNIExtension{},
			&tls.UtlsExtendedMasterSecretExtension{},
			&tls.RenegotiationInfoExtension{},
			&tls.SupportedCurvesExtension{Curves: []tls.CurveID{tls.GREASE_PLACEHOLDER, tls.X25519, tls.CurveP256, tls.CurveP384}},
			&tls.SupportedPointsExtension{SupportedPoints: []byte{0x0}}, // uncompressed
			&tls.SessionTicketExtension{},
			&tls.ALPNExtension{AlpnProtocols: []string{"http/1.1"}},
			&tls.StatusRequestExtension{},
			&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{0x0403, 0x0804, 0x0401, 0x0503, 0x0805, 0x0501, 0x0806, 0x0601}},
			&tls.SCTExtension{},
			&tls.KeyShareExtension{KeyShares: []tls.KeyShare{
				{Group: tls.CurveID(tls.GREASE_PLACEHOLDER), Data: []byte{0}},
				{Group: tls.X25519},
			}},
			&tls.PSKKeyExchangeModesExtension{Modes: []uint8{tls.PskModeDHE}}, // pskModeDHE
			&tls.SupportedVersionsExtension{Versions: []uint16{tls.GREASE_PLACEHOLDER, tls.VersionTLS13, tls.VersionTLS12}},
			&tls.UtlsCompressCertExtension{Algorithms: []tls.CertCompressionAlgo{tls.CertCompressionBrotli}},
			&tls.ApplicationSettingsExtension{SupportedProtocols: []string{"h2"}},
			&tls.UtlsGREASEExtension{},
			&tls.UtlsPaddingExtension{GetPaddingLen: tls.BoringPaddingStyle},
		},
		GetSessionID: nil,
	}
}
