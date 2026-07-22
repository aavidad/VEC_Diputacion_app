package interna

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestConstruirServidorInternoUsaListaPositivaYSellaManejador(t *testing.T) {
	tlsMutuo := configuracionTLSMutuoValidaPrueba(t)
	llamadasAPI := 0
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadasAPI++
		w.WriteHeader(http.StatusNoContent)
	})

	servidor, err := construirServidorInterno(configuracionInternaValidaPrueba(), api, tlsMutuo)
	if err != nil {
		t.Fatalf("construir servidor interno: %v", err)
	}
	if err := ValidarServidorParaEscucha(servidor); err != nil {
		t.Fatalf("servidor preparado: %v", err)
	}
	if _, sellado := servidor.Handler.(*manejadorInternoVerificado); !sellado {
		t.Fatalf("manejador sin sello de procedencia: %T", servidor.Handler)
	}
	if servidor.TLSConfig == tlsMutuo {
		t.Fatal("el servidor conservo la configuracion TLS mutable del llamador")
	}

	permitida := peticionInternaPrueba(http.MethodGet, "/api/vec/prueba")
	respuesta := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(respuesta, permitida)
	if respuesta.Code != http.StatusNoContent || llamadasAPI != 1 {
		t.Fatalf("API interna = codigo %d, llamadas %d", respuesta.Code, llamadasAPI)
	}

	for _, ruta := range []string{"/", "/api/publico/prueba", "/bolsa/", "/administracion/"} {
		respuesta = httptest.NewRecorder()
		servidor.Handler.ServeHTTP(respuesta, peticionInternaPrueba(http.MethodGet, ruta))
		if respuesta.Code != http.StatusNotFound {
			t.Errorf("ruta fuera de lista positiva %q = %d; se esperaba 404", ruta, respuesta.Code)
		}
	}
	if llamadasAPI != 1 {
		t.Fatalf("una ruta ajena alcanzo la API interna: %d llamadas", llamadasAPI)
	}
}

func TestConstruirServidorInternoRechazaManejadoresNoAutorizados(t *testing.T) {
	configuracionTLS := configuracionTLSMutuoValidaPrueba(t)
	for _, prueba := range []struct {
		nombre    string
		manejador http.Handler
	}{
		{nombre: "nulo", manejador: nil},
		{nombre: "nulo tipado", manejador: (*manejadorPunteroPrueba)(nil)},
		{nombre: "mux global", manejador: http.DefaultServeMux},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			servidor, err := construirServidorInterno(
				configuracionInternaValidaPrueba(), prueba.manejador, configuracionTLS,
			)
			if servidor != nil || !errors.Is(err, ErrAPIInternaNoDisponible) {
				t.Fatalf("resultado = (%v, %v); se esperaba API no disponible", servidor, err)
			}
		})
	}
}

func TestConstruirServidorInternoAceptaManejadorNoComparable(t *testing.T) {
	manejador := manejadorNoComparablePrueba{"prueba"}
	servidor, err := construirServidorInterno(
		configuracionInternaValidaPrueba(), manejador, configuracionTLSMutuoValidaPrueba(t),
	)
	if err != nil || servidor == nil {
		t.Fatalf("manejador no comparable = (%v, %v)", servidor, err)
	}
	respuesta := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(
		respuesta, peticionInternaPrueba(http.MethodGet, "/api/vec/prueba"),
	)
	if respuesta.Code != http.StatusNoContent {
		t.Fatalf("codigo = %d; se esperaba 204", respuesta.Code)
	}
}

func TestValidarServidorParaEscuchaFallaCerrado(t *testing.T) {
	configuracionTLSValida := configuracionTLSMutuoValidaPrueba(t)
	manejadorSellado := &manejadorInternoVerificado{siguiente: http.NotFoundHandler()}

	pruebas := []struct {
		nombre   string
		servidor func() *http.Server
		error    error
	}{
		{"servidor nulo", func() *http.Server { return nil }, ErrServidorInternoInvalido},
		{"manejador nulo", func() *http.Server {
			return &http.Server{TLSConfig: configuracionTLSValida.Clone()}
		}, ErrServidorInternoInvalido},
		{"TLS nulo", func() *http.Server {
			return &http.Server{Handler: manejadorSellado}
		}, ErrTLSMutuoNoVerificado},
		{"TLS 1.2", func() *http.Server {
			cfg := configuracionTLSValida.Clone()
			cfg.MinVersion = tls.VersionTLS12
			return &http.Server{Handler: manejadorSellado, TLSConfig: cfg}
		}, ErrTLSMutuoNoVerificado},
		{"cliente sin certificado", func() *http.Server {
			cfg := configuracionTLSValida.Clone()
			cfg.ClientAuth = tls.NoClientCert
			return &http.Server{Handler: manejadorSellado, TLSConfig: cfg}
		}, ErrTLSMutuoNoVerificado},
		{"CA nula", func() *http.Server {
			cfg := configuracionTLSValida.Clone()
			cfg.ClientCAs = nil
			return &http.Server{Handler: manejadorSellado, TLSConfig: cfg}
		}, ErrTLSMutuoNoVerificado},
		{"CA vacia", func() *http.Server {
			cfg := configuracionTLSValida.Clone()
			cfg.ClientCAs = x509.NewCertPool()
			return &http.Server{Handler: manejadorSellado, TLSConfig: cfg}
		}, ErrTLSMutuoNoVerificado},
		{"sin certificado servidor", func() *http.Server {
			cfg := configuracionTLSValida.Clone()
			cfg.Certificates = nil
			return &http.Server{Handler: manejadorSellado, TLSConfig: cfg}
		}, ErrTLSMutuoNoVerificado},
		{"certificado sin cadena", func() *http.Server {
			cfg := configuracionTLSValida.Clone()
			cfg.Certificates = []tls.Certificate{{PrivateKey: configuracionTLSValida.Certificates[0].PrivateKey}}
			return &http.Server{Handler: manejadorSellado, TLSConfig: cfg}
		}, ErrTLSMutuoNoVerificado},
		{"certificado sin clave", func() *http.Server {
			cfg := configuracionTLSValida.Clone()
			cfg.Certificates = []tls.Certificate{{Certificate: configuracionTLSValida.Certificates[0].Certificate}}
			return &http.Server{Handler: manejadorSellado, TLSConfig: cfg}
		}, ErrTLSMutuoNoVerificado},
		{"configuracion TLS dinamica", func() *http.Server {
			cfg := configuracionTLSValida.Clone()
			cfg.GetConfigForClient = func(*tls.ClientHelloInfo) (*tls.Config, error) {
				return &tls.Config{}, nil
			}
			return &http.Server{Handler: manejadorSellado, TLSConfig: cfg}
		}, ErrTLSMutuoNoVerificado},
		{"certificado TLS dinamico", func() *http.Server {
			cfg := configuracionTLSValida.Clone()
			cfg.GetCertificate = func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
				return &cfg.Certificates[0], nil
			}
			return &http.Server{Handler: manejadorSellado, TLSConfig: cfg}
		}, ErrTLSMutuoNoVerificado},
		{"verificacion insegura", func() *http.Server {
			cfg := configuracionTLSValida.Clone()
			cfg.InsecureSkipVerify = true
			return &http.Server{Handler: manejadorSellado, TLSConfig: cfg}
		}, ErrTLSMutuoNoVerificado},
		{"version maxima incoherente", func() *http.Server {
			cfg := configuracionTLSValida.Clone()
			cfg.MaxVersion = tls.VersionTLS12
			return &http.Server{Handler: manejadorSellado, TLSConfig: cfg}
		}, ErrTLSMutuoNoVerificado},
		{"manejador arbitrario", func() *http.Server {
			return &http.Server{Handler: http.NotFoundHandler(), TLSConfig: configuracionTLSValida.Clone()}
		}, ErrServidorInternoInvalido},
		{"sello sin manejador", func() *http.Server {
			return &http.Server{Handler: &manejadorInternoVerificado{}, TLSConfig: configuracionTLSValida.Clone()}
		}, ErrServidorInternoInvalido},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			if err := ValidarServidorParaEscucha(prueba.servidor()); !errors.Is(err, prueba.error) {
				t.Fatalf("error = %v; se esperaba %v", err, prueba.error)
			}
		})
	}
}

func TestValidarServidorParaEscuchaDetectaMutacionPosterior(t *testing.T) {
	servidor, err := construirServidorInterno(
		configuracionInternaValidaPrueba(), http.NotFoundHandler(), configuracionTLSMutuoValidaPrueba(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	servidor.Handler = http.NotFoundHandler()
	if err := ValidarServidorParaEscucha(servidor); !errors.Is(err, ErrServidorInternoInvalido) {
		t.Fatalf("manejador sustituido aceptado: %v", err)
	}

	servidor, err = construirServidorInterno(
		configuracionInternaValidaPrueba(), http.NotFoundHandler(), configuracionTLSMutuoValidaPrueba(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	servidor.TLSConfig.ClientAuth = tls.NoClientCert
	if err := ValidarServidorParaEscucha(servidor); !errors.Is(err, ErrTLSMutuoNoVerificado) {
		t.Fatalf("TLS degradado aceptado: %v", err)
	}

	servidor, err = construirServidorInterno(
		configuracionInternaValidaPrueba(), http.NotFoundHandler(), configuracionTLSMutuoValidaPrueba(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	servidor.Addr = "0.0.0.0:8443"
	if err := ValidarServidorParaEscucha(servidor); !errors.Is(err, ErrServidorInternoInvalido) {
		t.Fatalf("direccion degradada aceptada: %v", err)
	}

	servidor, err = construirServidorInterno(
		configuracionInternaValidaPrueba(), http.NotFoundHandler(), configuracionTLSMutuoValidaPrueba(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	servidor.ReadHeaderTimeout = 0
	if err := ValidarServidorParaEscucha(servidor); !errors.Is(err, ErrServidorInternoInvalido) {
		t.Fatalf("limites HTTP degradados aceptados: %v", err)
	}
}

func TestConstruirServidorInternoClonaMaterialTLSMutable(t *testing.T) {
	origen := configuracionTLSMutuoValidaPrueba(t)
	servidor, err := construirServidorInterno(
		configuracionInternaValidaPrueba(), http.NotFoundHandler(), origen,
	)
	if err != nil {
		t.Fatal(err)
	}
	if servidor.TLSConfig.ClientCAs == origen.ClientCAs {
		t.Fatal("el servidor compartio el pool de autoridades con el llamador")
	}
	sello := servidor.Handler.(*manejadorInternoVerificado).materialTLS
	if sello.autoridadesClientes == servidor.TLSConfig.ClientCAs {
		t.Fatal("el sello compartio el pool de autoridades con el servidor mutable")
	}
	if sello.certificadoServidor.PrivateKey != nil {
		t.Fatal("el sello duplico una referencia a la clave privada")
	}
	byteServidor := servidor.TLSConfig.Certificates[0].Certificate[0][0]
	origen.Certificates[0].Certificate[0][0] ^= 0xff
	origen.Certificates[0].Certificate = nil
	if servidor.TLSConfig.Certificates[0].Certificate[0][0] != byteServidor {
		t.Fatal("el llamador pudo mutar la cadena TLS del servidor")
	}
	if err := ValidarServidorParaEscucha(servidor); err != nil {
		t.Fatalf("la copia defensiva quedo invalida: %v", err)
	}
}

func TestValidarServidorParaEscuchaRechazaSustitucionDeMaterialTLS(t *testing.T) {
	pruebas := []struct {
		nombre string
		mutar  func(*testing.T, *http.Server, *tls.Config)
	}{
		{
			nombre: "otra CA no vacia",
			mutar: func(t *testing.T, servidor *http.Server, _ *tls.Config) {
				servidor.TLSConfig.ClientCAs = materialTLSMutuoPrueba(t).raices.Clone()
			},
		},
		{
			nombre: "CA anadida al pool aprobado",
			mutar: func(t *testing.T, servidor *http.Server, _ *tls.Config) {
				otro := materialTLSMutuoPrueba(t)
				ca, err := x509.ParseCertificate(otro.servidor.Certificates[0].Certificate[1])
				if err != nil {
					t.Fatal(err)
				}
				servidor.TLSConfig.ClientCAs.AddCert(ca)
			},
		},
		{
			nombre: "otro par servidor valido",
			mutar: func(t *testing.T, servidor *http.Server, _ *tls.Config) {
				servidor.TLSConfig.Certificates = materialTLSMutuoPrueba(t).servidor.Certificates
			},
		},
		{
			nombre: "cadena modificada directamente",
			mutar: func(_ *testing.T, servidor *http.Server, _ *tls.Config) {
				servidor.TLSConfig.Certificates[0].Certificate[0][0] ^= 0xff
			},
		},
		{
			nombre: "clave incoherente",
			mutar: func(t *testing.T, servidor *http.Server, _ *tls.Config) {
				servidor.TLSConfig.Certificates[0].PrivateKey =
					materialTLSMutuoPrueba(t).servidor.Certificates[0].PrivateKey
			},
		},
		{
			nombre: "Leaf eliminado",
			mutar: func(_ *testing.T, servidor *http.Server, _ *tls.Config) {
				servidor.TLSConfig.Certificates[0].Leaf = nil
			},
		},
		{
			nombre: "algoritmos de firma alterados",
			mutar: func(_ *testing.T, servidor *http.Server, _ *tls.Config) {
				servidor.TLSConfig.Certificates[0].SupportedSignatureAlgorithms =
					[]tls.SignatureScheme{tls.PSSWithSHA256}
			},
		},
		{
			nombre: "clave del proveedor mutada despues de construir",
			mutar: func(t *testing.T, _ *http.Server, origen *tls.Config) {
				clave, valida := origen.Certificates[0].PrivateKey.(ed25519.PrivateKey)
				if !valida || len(clave) == 0 {
					t.Fatal("la clave de prueba no es Ed25519")
				}
				clave[len(clave)-1] ^= 0xff
			},
		},
	}

	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			origen := configuracionTLSMutuoValidaPrueba(t)
			servidor, err := construirServidorInterno(
				configuracionInternaValidaPrueba(), http.NotFoundHandler(), origen,
			)
			if err != nil {
				t.Fatal(err)
			}
			prueba.mutar(t, servidor, origen)
			if err := ValidarServidorParaEscucha(servidor); !errors.Is(err, ErrTLSMutuoNoVerificado) {
				t.Fatalf("material TLS sustituido aceptado: %v", err)
			}
		})
	}
}

func TestServidorInternoExigeCertificadoClienteEnHandshakeReal(t *testing.T) {
	material := materialTLSMutuoPrueba(t)
	cfg := configuracionInternaValidaPrueba()
	cfg.DireccionEscucha = "127.0.0.1:8443"
	cfg.RedesPermitidas = []string{"127.0.0.0/8"}
	servidor, err := construirServidorInterno(cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), material.servidor)
	if err != nil {
		t.Fatal(err)
	}
	servidor.ErrorLog = log.New(io.Discard, "", 0)
	escucha, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	terminado := make(chan error, 1)
	go func() {
		terminado <- servidor.Serve(tls.NewListener(escucha, servidor.TLSConfig))
	}()
	t.Cleanup(func() {
		_ = servidor.Close()
		<-terminado
	})

	nuevaConexion := func(certificados []tls.Certificate) *http.Client {
		transporte := &http.Transport{TLSClientConfig: &tls.Config{
			MinVersion:   tls.VersionTLS13,
			MaxVersion:   tls.VersionTLS13,
			RootCAs:      material.raices,
			ServerName:   "servidor.interna.test",
			Certificates: certificados,
		}}
		t.Cleanup(transporte.CloseIdleConnections)
		return &http.Client{Transport: transporte, Timeout: 3 * time.Second}
	}
	url := "https://" + escucha.Addr().String() + "/api/vec/prueba"
	respuesta, err := nuevaConexion(nil).Get(url)
	if err == nil {
		_ = respuesta.Body.Close()
		t.Fatal("el handshake acepto un cliente sin certificado")
	}

	respuesta, err = nuevaConexion([]tls.Certificate{material.cliente}).Get(url)
	if err != nil {
		t.Fatalf("handshake mTLS valido: %v", err)
	}
	defer respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusNoContent {
		t.Fatalf("codigo mTLS = %d; se esperaba 204", respuesta.StatusCode)
	}
}

func peticionInternaPrueba(metodo, ruta string) *http.Request {
	peticion := httptest.NewRequest(metodo, ruta, nil)
	peticion.RemoteAddr = "10.7.1.1:50000"
	return peticion
}

func configuracionTLSMutuoValidaPrueba(t *testing.T) *tls.Config {
	t.Helper()
	return materialTLSMutuoPrueba(t).servidor
}

type materialTLSMutuo struct {
	servidor *tls.Config
	cliente  tls.Certificate
	raices   *x509.CertPool
}

func materialTLSMutuoPrueba(t *testing.T) materialTLSMutuo {
	t.Helper()
	publicaCA, privadaCA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now()
	plantilla := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "CA mTLS interna prueba"},
		NotBefore:             ahora.Add(-time.Minute),
		NotAfter:              ahora.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	derCA, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, publicaCA, privadaCA)
	if err != nil {
		t.Fatal(err)
	}
	certificadoCA, err := x509.ParseCertificate(derCA)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(certificadoCA)
	crearCertificado := func(serial int64, nombre string, dns []string, usos []x509.ExtKeyUsage) tls.Certificate {
		t.Helper()
		publica, privada, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		plantillaFinal := &x509.Certificate{
			SerialNumber: big.NewInt(serial),
			Subject:      pkix.Name{CommonName: nombre},
			DNSNames:     dns,
			NotBefore:    ahora.Add(-time.Minute),
			NotAfter:     ahora.Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  usos,
		}
		der, err := x509.CreateCertificate(
			rand.Reader, plantillaFinal, certificadoCA, publica, privadaCA,
		)
		if err != nil {
			t.Fatal(err)
		}
		return tls.Certificate{Certificate: [][]byte{der, derCA}, PrivateKey: privada}
	}
	servidor := crearCertificado(
		2, "servidor.interna.test", []string{"servidor.interna.test"},
		[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	)
	cliente := crearCertificado(
		3, "cliente.interna.test", nil, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	)
	return materialTLSMutuo{
		servidor: &tls.Config{
			MinVersion:   tls.VersionTLS13,
			MaxVersion:   tls.VersionTLS13,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    pool,
			Certificates: []tls.Certificate{servidor},
		},
		cliente: cliente,
		raices:  pool,
	}
}

type manejadorNoComparablePrueba []string

func (manejadorNoComparablePrueba) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

type manejadorPunteroPrueba struct{}

func (*manejadorPunteroPrueba) ServeHTTP(http.ResponseWriter, *http.Request) {}
