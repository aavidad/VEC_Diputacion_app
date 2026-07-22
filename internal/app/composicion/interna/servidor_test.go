package interna

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConstruirServidorInternoCargaReferenciasYSellaAllowlist(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	llamadas := 0
	servidor, err := construirServidorInterno(material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidarServidorParaEscucha(servidor); err != nil {
		t.Fatalf("servidor preparado: %v", err)
	}
	if _, sellado := servidor.Handler.(*manejadorInternoVerificado); !sellado {
		t.Fatalf("manejador sin sello: %T", servidor.Handler)
	}
	permitida := httptest.NewRequest(http.MethodGet, "/api/vec/prueba", nil)
	permitida.RemoteAddr = "127.0.0.2:50000"
	respuesta := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(respuesta, permitida)
	if respuesta.Code != http.StatusNoContent || llamadas != 1 {
		t.Fatalf("API permitida = (%d, %d)", respuesta.Code, llamadas)
	}
	for _, ruta := range []string{"/", "/api/publico/prueba", "/bolsa/", "/administracion/"} {
		peticion := httptest.NewRequest(http.MethodGet, ruta, nil)
		peticion.RemoteAddr = "127.0.0.2:50000"
		respuesta = httptest.NewRecorder()
		servidor.Handler.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusNotFound {
			t.Errorf("ruta %q = %d", ruta, respuesta.Code)
		}
	}
	if llamadas != 1 {
		t.Fatalf("ruta ajena alcanzo API: %d", llamadas)
	}
}

func TestConstruirServidorInternoRechazaManejadoresNoAutorizadosAntesDeLeerTLS(t *testing.T) {
	cfg := configuracionInternaValidaPrueba()
	for _, manejador := range []http.Handler{nil, (*manejadorPunteroPrueba)(nil), http.DefaultServeMux} {
		servidor, err := construirServidorInterno(cfg, manejador)
		if servidor != nil || !errors.Is(err, ErrAPIInternaNoDisponible) {
			t.Fatalf("manejador %T = (%v, %v)", manejador, servidor, err)
		}
	}
}

func TestConstruirServidorInternoNoAceptaConfiguracionTLSAutosellada(t *testing.T) {
	cfg := configuracionInternaValidaPrueba()
	servidor, err := construirServidorInterno(cfg, http.NotFoundHandler())
	if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
		t.Fatalf("referencias inexistentes = (%v, %v)", servidor, err)
	}
}

func TestCargaTLSRechazaRutasYFicherosInseguros(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	pruebas := []struct {
		nombre string
		mutar  func(*testing.T, *Configuracion)
	}{
		{"certificado symlink", func(t *testing.T, cfg *Configuracion) {
			enlace := filepath.Join(t.TempDir(), "cert.pem")
			if err := os.Symlink(cfg.CertificadoServidorTLS, enlace); err != nil {
				t.Fatal(err)
			}
			cfg.CertificadoServidorTLS = enlace
		}},
		{"directorio symlink", func(t *testing.T, cfg *Configuracion) {
			raiz := t.TempDir()
			enlace := filepath.Join(raiz, "secretos")
			if err := os.Symlink(filepath.Dir(cfg.CertificadoServidorTLS), enlace); err != nil {
				t.Fatal(err)
			}
			cfg.CertificadoServidorTLS = filepath.Join(enlace, filepath.Base(cfg.CertificadoServidorTLS))
		}},
		{"clave escribible por grupo", func(t *testing.T, cfg *Configuracion) {
			if err := os.Chmod(cfg.ClaveServidorTLS, 0o660); err != nil {
				t.Fatal(err)
			}
		}},
		{"CA escribible por grupo", func(t *testing.T, cfg *Configuracion) {
			if err := os.Chmod(cfg.AutoridadClientesTLS, 0o664); err != nil {
				t.Fatal(err)
			}
		}},
		{"certificado no regular", func(t *testing.T, cfg *Configuracion) {
			directorio := t.TempDir()
			cfg.CertificadoServidorTLS = directorio
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			cfg := material.cfg
			prueba.mutar(t, &cfg)
			servidor, err := construirServidorInterno(cfg, http.NotFoundHandler())
			if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
				t.Fatalf("fichero inseguro = (%v, %v)", servidor, err)
			}
		})
	}
}

func TestCargaTLSRechazaMaterialServidorNoProductivo(t *testing.T) {
	casos := []struct {
		nombre   string
		opciones opcionesCertificadoServidor
	}{
		{"expirado", opcionesCertificadoServidor{expirado: true}},
		{"aun no vigente", opcionesCertificadoServidor{futuro: true}},
		{"sin serverAuth", opcionesCertificadoServidor{sinServerAuth: true}},
		{"sin SAN", opcionesCertificadoServidor{sinSAN: true}},
		{"SAN ajeno", opcionesCertificadoServidor{sanDNSAjeno: true}},
		{"CA usada como leaf", opcionesCertificadoServidor{caComoHoja: true}},
		{"cadena sin raiz", opcionesCertificadoServidor{sinRaiz: true}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			material := materialTLSMutuoPrueba(t, caso.opciones)
			servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
			if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
				t.Fatalf("material invalido = (%v, %v)", servidor, err)
			}
		})
	}
}

func TestCargaTLSAdmiteFullchainConIntermediaComoAnclaExplicita(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{anclaIntermedia: true})
	servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
	if err != nil || servidor == nil {
		t.Fatalf("fullchain leaf+intermedia = (%v, %v)", servidor, err)
	}
}

func TestCargaTLSRechazaClaveYAutoridadInvalidas(t *testing.T) {
	t.Run("clave ajena", func(t *testing.T) {
		material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
		otro := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
		contenido, err := os.ReadFile(otro.cfg.ClaveServidorTLS)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(material.cfg.ClaveServidorTLS, contenido, 0o600); err != nil {
			t.Fatal(err)
		}
		servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
		if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
			t.Fatalf("clave ajena = (%v, %v)", servidor, err)
		}
	})
	t.Run("cadena firmada por CA ajena", func(t *testing.T) {
		material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
		otro := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
		hoja := material.configServidor.Certificates[0].Certificate[0]
		caAjena := otro.configServidor.Certificates[0].Certificate[1]
		contenido := append(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: hoja}), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caAjena})...)
		if err := os.WriteFile(material.cfg.CertificadoServidorTLS, contenido, 0o644); err != nil {
			t.Fatal(err)
		}
		servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
		if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
			t.Fatalf("cadena ajena = (%v, %v)", servidor, err)
		}
	})
	t.Run("certificado duplicado en cadena", func(t *testing.T) {
		material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
		contenido, err := os.ReadFile(material.cfg.CertificadoServidorTLS)
		if err != nil {
			t.Fatal(err)
		}
		contenido = append(contenido, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: material.configServidor.Certificates[0].Certificate[1]})...)
		if err := os.WriteFile(material.cfg.CertificadoServidorTLS, contenido, 0o644); err != nil {
			t.Fatal(err)
		}
		servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
		if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
			t.Fatalf("duplicado = (%v, %v)", servidor, err)
		}
	})
	t.Run("leaf como autoridad de clientes", func(t *testing.T) {
		material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
		if err := os.WriteFile(material.cfg.AutoridadClientesTLS, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: material.cliente.Certificate[0]}), 0o644); err != nil {
			t.Fatal(err)
		}
		servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
		if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
			t.Fatalf("leaf CA = (%v, %v)", servidor, err)
		}
	})
	t.Run("bundle del sistema", func(t *testing.T) {
		material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
		for _, ruta := range []string{"/etc/ssl/certs/ca-certificates.crt", "/etc/pki/tls/certs/ca-bundle.crt"} {
			if _, err := os.Stat(ruta); err == nil {
				material.cfg.AutoridadClientesTLS = ruta
				servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
				if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
					t.Fatalf("CA sistema = (%v, %v)", servidor, err)
				}
				return
			}
		}
		t.Skip("sistema sin bundle CA conocido")
	})
}

func TestCargaTLSPermiteSecretoRootAppSoloLectura(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	if err := os.Chmod(material.cfg.ClaveServidorTLS, 0o440); err != nil {
		t.Fatal(err)
	}
	servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
	if err != nil || servidor == nil {
		t.Fatalf("secreto 0440 = (%v, %v)", servidor, err)
	}
}

func TestConfiguracionTLSDenyByDefaultRechazaTodoCampoActivo(t *testing.T) {
	pruebas := []struct {
		nombre string
		mutar  func(*tls.Config)
	}{
		{"Rand", func(c *tls.Config) { c.Rand = bytes.NewReader(make([]byte, 256)) }},
		{"Time", func(c *tls.Config) { c.Time = time.Now }},
		{"NameToCertificate", func(c *tls.Config) { c.NameToCertificate = map[string]*tls.Certificate{} }},
		{"GetCertificate", func(c *tls.Config) {
			c.GetCertificate = func(*tls.ClientHelloInfo) (*tls.Certificate, error) { return nil, nil }
		}},
		{"GetClientCertificate", func(c *tls.Config) {
			c.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) { return nil, nil }
		}},
		{"GetConfigForClient", func(c *tls.Config) {
			c.GetConfigForClient = func(*tls.ClientHelloInfo) (*tls.Config, error) { return c, nil }
		}},
		{"VerifyPeerCertificate", func(c *tls.Config) {
			c.VerifyPeerCertificate = func([][]byte, [][]*x509.Certificate) error { return nil }
		}},
		{"VerifyConnection", func(c *tls.Config) { c.VerifyConnection = func(tls.ConnectionState) error { return nil } }},
		{"RootCAs", func(c *tls.Config) { c.RootCAs = x509.NewCertPool() }},
		{"ServerName", func(c *tls.Config) { c.ServerName = "ajeno.test" }},
		{"ClientAuth", func(c *tls.Config) { c.ClientAuth = tls.NoClientCert }},
		{"ClientCAs", func(c *tls.Config) { c.ClientCAs = x509.NewCertPool() }},
		{"InsecureSkipVerify", func(c *tls.Config) { c.InsecureSkipVerify = true }},
		{"CipherSuites", func(c *tls.Config) { c.CipherSuites = []uint16{tls.TLS_AES_128_GCM_SHA256} }},
		{"PreferServerCipherSuites", func(c *tls.Config) { c.PreferServerCipherSuites = true }},
		{"SessionTicketsDisabled", func(c *tls.Config) { c.SessionTicketsDisabled = false }},
		{"SessionTicketKey", func(c *tls.Config) { c.SessionTicketKey[0] = 1 }},
		{"ClientSessionCache", func(c *tls.Config) { c.ClientSessionCache = tls.NewLRUClientSessionCache(1) }},
		{"UnwrapSession", func(c *tls.Config) {
			c.UnwrapSession = func([]byte, tls.ConnectionState) (*tls.SessionState, error) { return nil, nil }
		}},
		{"WrapSession", func(c *tls.Config) {
			c.WrapSession = func(tls.ConnectionState, *tls.SessionState) ([]byte, error) { return nil, nil }
		}},
		{"MinVersion", func(c *tls.Config) { c.MinVersion = tls.VersionTLS12 }},
		{"MaxVersion", func(c *tls.Config) { c.MaxVersion = 0 }},
		{"CurvePreferences", func(c *tls.Config) { c.CurvePreferences = []tls.CurveID{tls.X25519} }},
		{"DynamicRecordSizingDisabled", func(c *tls.Config) { c.DynamicRecordSizingDisabled = true }},
		{"Renegotiation", func(c *tls.Config) { c.Renegotiation = tls.RenegotiateOnceAsClient }},
		{"KeyLogWriter", func(c *tls.Config) { c.KeyLogWriter = io.Discard }},
		{"EncryptedClientHelloConfigList", func(c *tls.Config) { c.EncryptedClientHelloConfigList = []byte{1} }},
		{"EncryptedClientHelloRejectionVerify", func(c *tls.Config) {
			c.EncryptedClientHelloRejectionVerify = func(tls.ConnectionState) error { return nil }
		}},
		{"GetEncryptedClientHelloKeys", func(c *tls.Config) {
			c.GetEncryptedClientHelloKeys = func(*tls.ClientHelloInfo) ([]tls.EncryptedClientHelloKey, error) { return nil, nil }
		}},
		{"EncryptedClientHelloKeys", func(c *tls.Config) { c.EncryptedClientHelloKeys = []tls.EncryptedClientHelloKey{{}} }},
		{"NextProtos", func(c *tls.Config) { c.NextProtos = []string{"h2"} }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
			servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
			if err != nil {
				t.Fatal(err)
			}
			prueba.mutar(servidor.TLSConfig)
			if err := ValidarServidorParaEscucha(servidor); !errors.Is(err, ErrTLSMutuoNoVerificado) {
				t.Fatalf("campo activo aceptado: %v", err)
			}
		})
	}
}

func TestSelloTLSDetectaSustitucionYConservaCopiasDefensivas(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	original := servidor.TLSConfig.Certificates[0].Certificate[0][0]
	if err := os.WriteFile(material.cfg.CertificadoServidorTLS, []byte("sustituido"), 0o644); err != nil {
		t.Fatal(err)
	}
	if servidor.TLSConfig.Certificates[0].Certificate[0][0] != original {
		t.Fatal("fichero compartio bytes con servidor")
	}
	if err := ValidarServidorParaEscucha(servidor); err != nil {
		t.Fatalf("copia defensiva: %v", err)
	}

	mutaciones := []struct {
		nombre string
		mutar  func(*testing.T, *http.Server)
	}{
		{"CA ajena", func(t *testing.T, s *http.Server) {
			s.TLSConfig.ClientCAs = materialTLSMutuoPrueba(t, opcionesCertificadoServidor{}).raicesClientes
		}},
		{"CA anadida", func(t *testing.T, s *http.Server) {
			otro := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
			s.TLSConfig.ClientCAs.AddCert(otro.caClientes)
		}},
		{"certificado ajeno", func(t *testing.T, s *http.Server) {
			otro := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
			s.TLSConfig.Certificates = otro.configServidor.Certificates
		}},
		{"cadena mutada", func(_ *testing.T, s *http.Server) { s.TLSConfig.Certificates[0].Certificate[0][0] ^= 0xff }},
		{"clave ajena", func(t *testing.T, s *http.Server) {
			s.TLSConfig.Certificates[0].PrivateKey = materialTLSMutuoPrueba(t, opcionesCertificadoServidor{}).configServidor.Certificates[0].PrivateKey
		}},
		{"clave mutada", func(t *testing.T, s *http.Server) {
			clave := s.TLSConfig.Certificates[0].PrivateKey.(ed25519.PrivateKey)
			clave[0] ^= 0xff
		}},
		{"Leaf eliminado", func(_ *testing.T, s *http.Server) { s.TLSConfig.Certificates[0].Leaf = nil }},
		{"algoritmos firma", func(_ *testing.T, s *http.Server) {
			s.TLSConfig.Certificates[0].SupportedSignatureAlgorithms = []tls.SignatureScheme{tls.Ed25519}
		}},
		{"OCSP", func(_ *testing.T, s *http.Server) { s.TLSConfig.Certificates[0].OCSPStaple = []byte{1} }},
		{"SCT", func(_ *testing.T, s *http.Server) {
			s.TLSConfig.Certificates[0].SignedCertificateTimestamps = [][]byte{{1}}
		}},
	}
	for _, mutacion := range mutaciones {
		t.Run(mutacion.nombre, func(t *testing.T) {
			material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
			servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
			if err != nil {
				t.Fatal(err)
			}
			mutacion.mutar(t, servidor)
			if err := ValidarServidorParaEscucha(servidor); !errors.Is(err, ErrTLSMutuoNoVerificado) {
				t.Fatalf("mutacion aceptada: %v", err)
			}
		})
	}
}

func TestServidorInternoRechazaMutacionesHTTP(t *testing.T) {
	pruebas := []struct {
		nombre string
		mutar  func(*http.Server)
	}{
		{"Addr", func(s *http.Server) { s.Addr = "0.0.0.0:8443" }},
		{"Handler", func(s *http.Server) { s.Handler = http.NotFoundHandler() }},
		{"ReadHeaderTimeout", func(s *http.Server) { s.ReadHeaderTimeout = 0 }},
		{"OPTIONS", func(s *http.Server) { s.DisableGeneralOptionsHandler = false }},
		{"TLSNextProto", func(s *http.Server) { s.TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){} }},
		{"HTTP2 config", func(s *http.Server) { s.HTTP2 = &http.HTTP2Config{} }},
		{"protocolos nil", func(s *http.Server) { s.Protocols = nil }},
		{"HTTP2", func(s *http.Server) { s.Protocols.SetHTTP2(true) }},
		{"h2c", func(s *http.Server) { s.Protocols.SetUnencryptedHTTP2(true) }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
			servidor, err := construirServidorInterno(material.cfg, http.NotFoundHandler())
			if err != nil {
				t.Fatal(err)
			}
			prueba.mutar(servidor)
			if err := ValidarServidorParaEscucha(servidor); !errors.Is(err, ErrServidorInternoInvalido) {
				t.Fatalf("mutacion aceptada: %v", err)
			}
		})
	}
}

func TestServidorInternoProtocolosRealesNoAlcanzanAPISinHTTPUnoALPN(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	llamadas := 0
	servidor, err := construirServidorInterno(material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	direccion := iniciarServidorTLSPrueba(t, servidor)

	t.Run("HTTP2 TLS", func(t *testing.T) {
		conexion, err := tls.Dial("tcp", direccion, material.configCliente([]string{"h2"}, true))
		if err == nil {
			conexion.Close()
			t.Fatal("HTTP/2 negocio TLS")
		}
	})
	t.Run("h2c", func(t *testing.T) {
		conexion, err := net.DialTimeout("tcp", direccion, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer conexion.Close()
		_ = conexion.SetDeadline(time.Now().Add(time.Second))
		_, _ = io.WriteString(conexion, "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")
		respuesta := make([]byte, 1)
		if n, _ := conexion.Read(respuesta); n != 0 {
			t.Fatal("listener TLS respondio como h2c")
		}
	})
	t.Run("sin ALPN", func(t *testing.T) {
		conexion, err := tls.Dial("tcp", direccion, material.configCliente(nil, true))
		if err != nil {
			t.Fatal(err)
		}
		defer conexion.Close()
		if _, err := io.WriteString(conexion, "GET /api/vec/prueba HTTP/1.1\r\nHost: interno.test\r\nConnection: close\r\n\r\n"); err != nil {
			t.Fatal(err)
		}
		respuesta, err := http.ReadResponse(bufio.NewReader(conexion), &http.Request{Method: http.MethodGet})
		if err != nil {
			t.Fatal(err)
		}
		defer respuesta.Body.Close()
		if respuesta.StatusCode == http.StatusNoContent {
			t.Fatal("cliente sin ALPN alcanzo API")
		}
	})
	if llamadas != 0 {
		t.Fatalf("protocolos alternativos alcanzaron API: %d", llamadas)
	}
}

func TestServidorInternoMTLSYOPTIONSGeneralReales(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	llamadas := 0
	servidor, err := construirServidorInterno(material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	direccion := iniciarServidorTLSPrueba(t, servidor)
	if conexionSinCertificado, err := tls.Dial("tcp", direccion, material.configCliente([]string{protocoloALPNHTTPUno}, false)); err == nil {
		_, errEscritura := io.WriteString(conexionSinCertificado, "GET /api/vec/prueba HTTP/1.1\r\nHost: interno.test\r\nConnection: close\r\n\r\n")
		_, errLectura := http.ReadResponse(bufio.NewReader(conexionSinCertificado), &http.Request{Method: http.MethodGet})
		_ = conexionSinCertificado.Close()
		if errEscritura == nil && errLectura == nil {
			t.Fatal("cliente sin certificado alcanzo HTTP")
		}
	}
	conexion, err := tls.Dial("tcp", direccion, material.configCliente([]string{protocoloALPNHTTPUno}, true))
	if err != nil {
		t.Fatal(err)
	}
	defer conexion.Close()
	if _, err := fmt.Fprint(conexion, "OPTIONS * HTTP/1.1\r\nHost: interno.test\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	respuesta, err := http.ReadResponse(bufio.NewReader(conexion), &http.Request{Method: http.MethodOptions})
	if err != nil {
		t.Fatal(err)
	}
	defer respuesta.Body.Close()
	if respuesta.StatusCode == http.StatusOK || llamadas != 0 {
		t.Fatalf("OPTIONS bypass = codigo %d, API %d", respuesta.StatusCode, llamadas)
	}
}

func iniciarServidorTLSPrueba(t *testing.T, servidor *http.Server) string {
	t.Helper()
	servidor.ErrorLog = log.New(io.Discard, "", 0)
	escucha, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	terminado := make(chan error, 1)
	go func() { terminado <- servidor.ServeTLS(escucha, "", "") }()
	t.Cleanup(func() { _ = servidor.Close(); <-terminado })
	return escucha.Addr().String()
}

type opcionesCertificadoServidor struct {
	expirado, futuro, sinServerAuth, sinSAN, sanDNSAjeno, caComoHoja, sinRaiz, anclaIntermedia bool
}

type materialTLSMutuo struct {
	cfg            Configuracion
	cliente        tls.Certificate
	raicesServidor *x509.CertPool
	raicesClientes *x509.CertPool
	caClientes     *x509.Certificate
	configServidor *tls.Config
}

func materialTLSMutuoPrueba(t *testing.T, opciones opcionesCertificadoServidor) materialTLSMutuo {
	t.Helper()
	ahora := time.Now()
	crearCA := func(serial int64, nombre string) (*x509.Certificate, ed25519.PrivateKey, []byte) {
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, publica, privada)
		if err != nil {
			t.Fatal(err)
		}
		certificado, err := x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}
		return certificado, privada, der
	}
	caServidor, claveCAServidor, derCAServidor := crearCA(1, "CA servidor interna")
	caClientes, claveCAClientes, derCAClientes := crearCA(2, "CA clientes interna")
	emisorServidor, claveEmisorServidor, derEmisorServidor := caServidor, claveCAServidor, derCAServidor
	if opciones.anclaIntermedia {
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(20), Subject: pkix.Name{CommonName: "CA intermedia servidor"}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(12 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, caServidor, publica, claveCAServidor)
		if err != nil {
			t.Fatal(err)
		}
		emisorServidor, err = x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}
		claveEmisorServidor, derEmisorServidor = privada, der
	}

	publicaServidor, claveServidor, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	noAntes, noDespues := ahora.Add(-time.Hour), ahora.Add(time.Hour)
	if opciones.expirado {
		noAntes, noDespues = ahora.Add(-2*time.Hour), ahora.Add(-time.Hour)
	}
	if opciones.futuro {
		noAntes, noDespues = ahora.Add(time.Hour), ahora.Add(2*time.Hour)
	}
	plantillaServidor := &x509.Certificate{SerialNumber: big.NewInt(3), Subject: pkix.Name{CommonName: "servidor interno"}, NotBefore: noAntes, NotAfter: noDespues, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, DNSNames: []string{"servidor.interna.test"}}
	if opciones.sinServerAuth {
		plantillaServidor.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	if opciones.sinSAN {
		plantillaServidor.DNSNames = nil
	}
	if opciones.sanDNSAjeno {
		plantillaServidor.DNSNames = []string{"ajeno.test"}
	}
	padre, clavePadre := emisorServidor, claveEmisorServidor
	if opciones.caComoHoja {
		plantillaServidor.IsCA = true
		plantillaServidor.BasicConstraintsValid = true
		plantillaServidor.KeyUsage |= x509.KeyUsageCertSign
	}
	derServidor, err := x509.CreateCertificate(rand.Reader, plantillaServidor, padre, publicaServidor, clavePadre)
	if err != nil {
		t.Fatal(err)
	}

	crearFinal := func(serial int64, nombre string, ca *x509.Certificate, claveCA ed25519.PrivateKey, derCA []byte) tls.Certificate {
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantilla := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: nombre}, NotBefore: ahora.Add(-time.Hour), NotAfter: ahora.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
		der, err := x509.CreateCertificate(rand.Reader, plantilla, ca, publica, claveCA)
		if err != nil {
			t.Fatal(err)
		}
		return tls.Certificate{Certificate: [][]byte{der, derCA}, PrivateKey: privada}
	}
	cliente := crearFinal(4, "cliente interno", caClientes, claveCAClientes, derCAClientes)

	directorio := t.TempDir()
	cfg := configuracionInternaValidaPrueba()
	cfg.DireccionEscucha = "127.0.0.1:8443"
	cfg.RedesPermitidas = []string{"127.0.0.0/8"}
	cfg.NombreServidorTLS = "servidor.interna.test"
	cfg.CertificadoServidorTLS = filepath.Join(directorio, "servidor.crt")
	cfg.ClaveServidorTLS = filepath.Join(directorio, "servidor.key")
	cfg.AutoridadClientesTLS = filepath.Join(directorio, "clientes-ca.crt")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derServidor})
	if !opciones.sinRaiz {
		certPEM = append(certPEM, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derEmisorServidor})...)
	}
	claveDER, err := x509.MarshalPKCS8PrivateKey(claveServidor)
	if err != nil {
		t.Fatal(err)
	}
	clavePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: claveDER})
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derCAClientes})
	if err := os.WriteFile(cfg.CertificadoServidorTLS, certPEM, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.ClaveServidorTLS, clavePEM, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.AutoridadClientesTLS, caPEM, 0o644); err != nil {
		t.Fatal(err)
	}
	raicesServidor := x509.NewCertPool()
	raicesServidor.AddCert(caServidor)
	raicesClientes := x509.NewCertPool()
	raicesClientes.AddCert(caClientes)
	parServidor, err := tls.X509KeyPair(certPEM, clavePEM)
	if err != nil && !opciones.sinRaiz {
		t.Fatal(err)
	}
	return materialTLSMutuo{
		cfg: cfg, cliente: cliente, raicesServidor: raicesServidor, raicesClientes: raicesClientes, caClientes: caClientes,
		configServidor: &tls.Config{Certificates: []tls.Certificate{parServidor}},
	}
}

func (m materialTLSMutuo) configCliente(protocolos []string, conCertificado bool) *tls.Config {
	cfg := &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS13, RootCAs: m.raicesServidor, ServerName: "servidor.interna.test", NextProtos: protocolos}
	if conCertificado {
		cfg.Certificates = []tls.Certificate{m.cliente}
	}
	return cfg
}

type manejadorPunteroPrueba struct{}

func (*manejadorPunteroPrueba) ServeHTTP(http.ResponseWriter, *http.Request) {}
