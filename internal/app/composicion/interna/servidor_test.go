package interna

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

func TestConstruirServidorInternoCreaCapsulaYRechazaTLSFabricado(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	llamadas := 0
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	if err := validarServidorInterno(servidor); err != nil {
		t.Fatalf("servidor preparado: %v", err)
	}
	if servidor.manejador == nil || servidor.manejador.token != servidor.token {
		t.Fatal("manejador sin token de capsula")
	}
	permitida := httptest.NewRequest(http.MethodGet, "/api/vec/prueba", nil)
	permitida.RemoteAddr = "127.0.0.2:50000"
	permitida.TLS = estadoTLSMutuoValidoPrueba(t, material)
	respuesta := httptest.NewRecorder()
	servidor.manejador.ServeHTTP(respuesta, permitida)
	if respuesta.Code != http.StatusBadRequest || llamadas != 0 {
		t.Fatalf("TLS fabricado = (%d, %d)", respuesta.Code, llamadas)
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

func TestCargaTLSRechazaDirectoriosModificablesPorRuntime(t *testing.T) {
	raiz, err := syscall.Open("/", syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Close(raiz)
	if err := validarDirectorioTLS(raiz); err != nil {
		t.Fatalf("raiz del sistema rechazada: %v", err)
	}
	for _, modo := range []os.FileMode{0o777, 0o770, 0o550, 0o500} {
		t.Run(modo.String(), func(t *testing.T) {
			directorio := filepath.Join(t.TempDir(), "secretos")
			if err := os.Mkdir(directorio, modo); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(directorio, modo); err != nil {
				t.Fatal(err)
			}
			fd, err := syscall.Open(directorio, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer syscall.Close(fd)
			if err := validarDirectorioTLS(fd); !errors.Is(err, ErrTLSMutuoNoVerificado) {
				t.Fatalf("directorio app-owned %04o aceptado: %v", modo.Perm(), err)
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
			servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
			if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
				t.Fatalf("material invalido = (%v, %v)", servidor, err)
			}
		})
	}
}

func TestCargaTLSAdmiteFullchainConIntermediaComoAnclaExplicita(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{anclaIntermedia: true})
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
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
		servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
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
		servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
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
		servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
		if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
			t.Fatalf("duplicado = (%v, %v)", servidor, err)
		}
	})
	t.Run("leaf como autoridad de clientes", func(t *testing.T) {
		material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
		if err := os.WriteFile(material.cfg.AutoridadClientesTLS, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: material.cliente.Certificate[0]}), 0o644); err != nil {
			t.Fatal(err)
		}
		servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
		if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
			t.Fatalf("leaf CA = (%v, %v)", servidor, err)
		}
	})
	t.Run("bundle del sistema", func(t *testing.T) {
		material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
		for _, ruta := range []string{"/etc/ssl/certs/ca-certificates.crt", "/etc/pki/tls/certs/ca-bundle.crt"} {
			if _, err := os.Stat(ruta); err == nil {
				material.cfg.AutoridadClientesTLS = ruta
				servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
				if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
					t.Fatalf("CA sistema = (%v, %v)", servidor, err)
				}
				return
			}
		}
		t.Skip("sistema sin bundle CA conocido")
	})
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
			servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
			if err != nil {
				t.Fatal(err)
			}
			prueba.mutar(servidor.configuracionTLS)
			if err := validarServidorInterno(servidor); !errors.Is(err, ErrTLSMutuoNoVerificado) {
				t.Fatalf("campo activo aceptado: %v", err)
			}
		})
	}
}

func TestSelloTLSDetectaSustitucionYConservaCopiasDefensivas(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	original := servidor.configuracionTLS.Certificates[0].Certificate[0][0]
	if err := os.WriteFile(material.cfg.CertificadoServidorTLS, []byte("sustituido"), 0o644); err != nil {
		t.Fatal(err)
	}
	if servidor.configuracionTLS.Certificates[0].Certificate[0][0] != original {
		t.Fatal("fichero compartio bytes con servidor")
	}
	if err := validarServidorInterno(servidor); err != nil {
		t.Fatalf("copia defensiva: %v", err)
	}

	mutaciones := []struct {
		nombre string
		mutar  func(*testing.T, *ServidorInterno)
	}{
		{"CA ajena", func(t *testing.T, s *ServidorInterno) {
			s.configuracionTLS.ClientCAs = materialTLSMutuoPrueba(t, opcionesCertificadoServidor{}).raicesClientes
		}},
		{"CA anadida", func(t *testing.T, s *ServidorInterno) {
			otro := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
			s.configuracionTLS.ClientCAs.AddCert(otro.caClientes)
		}},
		{"certificado ajeno", func(t *testing.T, s *ServidorInterno) {
			otro := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
			s.configuracionTLS.Certificates = otro.configServidor.Certificates
		}},
		{"cadena mutada", func(_ *testing.T, s *ServidorInterno) { s.configuracionTLS.Certificates[0].Certificate[0][0] ^= 0xff }},
		{"clave ajena", func(t *testing.T, s *ServidorInterno) {
			s.configuracionTLS.Certificates[0].PrivateKey = materialTLSMutuoPrueba(t, opcionesCertificadoServidor{}).configServidor.Certificates[0].PrivateKey
		}},
		{"clave mutada", func(t *testing.T, s *ServidorInterno) {
			clave := s.configuracionTLS.Certificates[0].PrivateKey.(ed25519.PrivateKey)
			clave[0] ^= 0xff
		}},
		{"Leaf eliminado", func(_ *testing.T, s *ServidorInterno) { s.configuracionTLS.Certificates[0].Leaf = nil }},
		{"algoritmos firma", func(_ *testing.T, s *ServidorInterno) {
			s.configuracionTLS.Certificates[0].SupportedSignatureAlgorithms = []tls.SignatureScheme{tls.Ed25519}
		}},
		{"OCSP", func(_ *testing.T, s *ServidorInterno) { s.configuracionTLS.Certificates[0].OCSPStaple = []byte{1} }},
		{"SCT", func(_ *testing.T, s *ServidorInterno) {
			s.configuracionTLS.Certificates[0].SignedCertificateTimestamps = [][]byte{{1}}
		}},
	}
	for _, mutacion := range mutaciones {
		t.Run(mutacion.nombre, func(t *testing.T) {
			material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
			servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
			if err != nil {
				t.Fatal(err)
			}
			mutacion.mutar(t, servidor)
			if err := validarServidorInterno(servidor); !errors.Is(err, ErrTLSMutuoNoVerificado) {
				t.Fatalf("mutacion aceptada: %v", err)
			}
		})
	}
}

func TestServidorInternoRechazaMutacionesHTTP(t *testing.T) {
	pruebas := []struct {
		nombre string
		mutar  func(*ServidorInterno)
	}{
		{"direccion", func(s *ServidorInterno) { s.direccionEscucha = "0.0.0.0:8443" }},
		{"manejador", func(s *ServidorInterno) { s.manejador = nil }},
		{"timeout", func(s *ServidorInterno) { s.tiempoCabeceras = 0 }},
		{"token", func(s *ServidorInterno) { s.token = &tokenServidorInterno{marca: 2} }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
			servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
			if err != nil {
				t.Fatal(err)
			}
			prueba.mutar(servidor)
			if err := validarServidorInterno(servidor); !errors.Is(err, ErrServidorInternoInvalido) {
				t.Fatalf("mutacion aceptada: %v", err)
			}
		})
	}
}

func TestServidorInternoNoExponeTransporteNiCampos(t *testing.T) {
	tipoValor := reflect.TypeOf(ServidorInterno{})
	for indice := range tipoValor.NumField() {
		campo := tipoValor.Field(indice)
		if campo.IsExported() || campo.Anonymous {
			t.Fatalf("campo expuesto: %s", campo.Name)
		}
	}
	tipoPuntero := reflect.TypeOf((*ServidorInterno)(nil))
	for _, nombre := range []string{
		"Serve", "ServeTLS", "ListenAndServe", "ListenAndServeTLS",
		"Handler", "Listener", "TLSConfig", "GetCertificate", "GetClientCAs",
	} {
		if _, existe := tipoPuntero.MethodByName(nombre); existe {
			t.Fatalf("metodo de transporte expuesto: %s", nombre)
		}
	}
	if tipoPuntero.NumMethod() != 2 {
		t.Fatalf("metodos publicos = %d; se esperaban EscucharYServir y Apagar", tipoPuntero.NumMethod())
	}
	if metodo, existe := tipoPuntero.MethodByName("EscucharYServir"); !existe || metodo.Type.NumIn() != 1 {
		t.Fatal("EscucharYServir ausente o acepta parametros")
	}
	if _, existe := tipoPuntero.MethodByName("Apagar"); !existe {
		t.Fatal("Apagar opaco ausente")
	}
}

func TestServidorInternoConstruyeHTTPLocalCerrado(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	local := servidor.nuevoServidorHTTP()
	if local == nil || local.Handler != servidor.manejador ||
		!local.DisableGeneralOptionsHandler || local.TLSConfig == nil ||
		local.TLSConfig == servidor.configuracionTLS || local.TLSNextProto != nil ||
		local.HTTP2 != nil || local.Protocols == nil || !local.Protocols.HTTP1() ||
		local.Protocols.HTTP2() || local.Protocols.UnencryptedHTTP2() ||
		local.BaseContext == nil || local.ConnContext == nil || local.ConnState == nil ||
		local.ErrorLog == nil {
		t.Fatal("http.Server local no conserva el perfil cerrado")
	}
}

func TestServidorInternoProtocolosRealesNoAlcanzanAPISinHTTPUnoALPN(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	llamadas := 0
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	otroMaterial := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	llamadas := 0
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
	configuracionCAAjena := material.configCliente([]string{protocoloALPNHTTPUno}, false)
	configuracionCAAjena.Certificates = []tls.Certificate{otroMaterial.cliente}
	if conexionCAAjena, err := tls.Dial("tcp", direccion, configuracionCAAjena); err == nil {
		_, errEscritura := io.WriteString(conexionCAAjena, "GET /api/vec/prueba HTTP/1.1\r\nHost: interno.test\r\nConnection: close\r\n\r\n")
		_, errLectura := http.ReadResponse(bufio.NewReader(conexionCAAjena), &http.Request{Method: http.MethodGet})
		_ = conexionCAAjena.Close()
		if errEscritura == nil && errLectura == nil {
			t.Fatal("cliente de CA ajena alcanzo HTTP")
		}
	}
	conexionGET, err := tls.Dial("tcp", direccion, material.configCliente([]string{protocoloALPNHTTPUno}, true))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprint(conexionGET, "GET /api/vec/prueba HTTP/1.1\r\nHost: interno.test\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	respuestaGET, err := http.ReadResponse(bufio.NewReader(conexionGET), &http.Request{Method: http.MethodGet})
	if err != nil {
		t.Fatal(err)
	}
	_ = respuestaGET.Body.Close()
	_ = conexionGET.Close()
	if respuestaGET.StatusCode != http.StatusNoContent || llamadas != 1 {
		t.Fatalf("peticion mTLS real = codigo %d, API %d", respuestaGET.StatusCode, llamadas)
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
	if respuesta.StatusCode == http.StatusOK || llamadas != 1 {
		t.Fatalf("OPTIONS bypass = codigo %d, API %d", respuesta.StatusCode, llamadas)
	}
}

func TestServidorInternoNoReanudaSesionesTLS(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	llamadas := 0
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	direccion := iniciarServidorTLSPrueba(t, servidor)
	configuracionCliente := material.configCliente([]string{protocoloALPNHTTPUno}, true)
	configuracionCliente.ClientSessionCache = tls.NewLRUClientSessionCache(2)
	for intento := 1; intento <= 2; intento++ {
		conexion, err := tls.Dial("tcp", direccion, configuracionCliente)
		if err != nil {
			t.Fatal(err)
		}
		if conexion.ConnectionState().DidResume {
			_ = conexion.Close()
			t.Fatalf("conexion %d reanudada", intento)
		}
		if _, err := fmt.Fprint(conexion, "GET /api/vec/prueba HTTP/1.1\r\nHost: interno.test\r\nConnection: close\r\n\r\n"); err != nil {
			t.Fatal(err)
		}
		respuesta, err := http.ReadResponse(bufio.NewReader(conexion), &http.Request{Method: http.MethodGet})
		if err != nil {
			t.Fatal(err)
		}
		_ = respuesta.Body.Close()
		_ = conexion.Close()
		if respuesta.StatusCode != http.StatusNoContent {
			t.Fatalf("conexion %d = %d", intento, respuesta.StatusCode)
		}
	}
	if llamadas != 2 {
		t.Fatalf("peticiones mTLS = %d", llamadas)
	}
}

func iniciarServidorTLSPrueba(t *testing.T, servidor *ServidorInterno) string {
	t.Helper()
	escucha, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	direccion := escucha.Addr().String()
	if err := escucha.Close(); err != nil {
		t.Fatal(err)
	}
	servidor.direccionEscucha = direccion
	servidor.manejador.direccionEscucha = direccion
	terminado := make(chan error, 1)
	go func() { terminado <- servidor.EscucharYServir() }()
	limite := time.Now().Add(2 * time.Second)
	for {
		servidor.ejecucion.mu.Lock()
		activo := servidor.ejecucion.servidorActivo != nil
		servidor.ejecucion.mu.Unlock()
		if activo {
			break
		}
		select {
		case err := <-terminado:
			t.Fatalf("escucha termino antes de arrancar: %v", err)
		default:
		}
		if time.Now().After(limite) {
			t.Fatal("escucha no arranco")
		}
		time.Sleep(time.Millisecond)
	}
	t.Cleanup(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		if err := servidor.Apagar(ctx); err != nil {
			t.Errorf("cerrar servidor de prueba: %v", err)
		}
		select {
		case err := <-terminado:
			if err != nil {
				t.Errorf("terminar servidor de prueba: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("servidor de prueba no termino")
		}
	})
	return direccion
}
