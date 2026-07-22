package interna

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type conexionConEstadoCopiado struct {
	net.Conn
	estado tls.ConnectionState
}

func (c *conexionConEstadoCopiado) ConnectionState() tls.ConnectionState {
	return c.estado
}

type proxyConexionTLS struct {
	*tls.Conn
}

type contextoApagadoObservado struct {
	context.Context
	consultas atomic.Int32
	esperando chan struct{}
	unaVez    sync.Once
}

func (c *contextoApagadoObservado) Done() <-chan struct{} {
	if c.consultas.Add(1) >= 2 {
		c.unaVez.Do(func() { close(c.esperando) })
	}
	return c.Context.Done()
}

func TestPosesionTLSRechazaTokenConexionEstadoYProxiesAjenos(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	var llamadas atomic.Int32
	capturas := make(chan *http.Request, 4)
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llamadas.Add(1)
		capturas <- r
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	direccion := iniciarServidorTLSPrueba(t, servidor)
	conexion, err := tls.Dial(
		"tcp", direccion,
		material.configCliente([]string{protocoloALPNHTTPUno}, true),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conexion.Close()
	if _, err := io.WriteString(
		conexion,
		"GET /api/vec/prueba HTTP/1.1\r\nHost: interno.test\r\nConnection: keep-alive\r\n\r\n",
	); err != nil {
		t.Fatal(err)
	}
	respuesta, err := http.ReadResponse(
		bufio.NewReader(conexion),
		&http.Request{Method: http.MethodGet},
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusNoContent {
		t.Fatalf("peticion real = %d", respuesta.StatusCode)
	}
	peticionReal := <-capturas
	posesionReal, valida := peticionReal.Context().Value(
		claveContextoConexionTLS{},
	).(*posesionConexionTLS)
	if !valida || posesionReal == nil || posesionReal.token != servidor.token ||
		posesionReal.conexion == nil {
		t.Fatal("peticion real sin posesion exacta de la conexion")
	}

	ejecutar := func(ctx context.Context, estado *tls.ConnectionState) int {
		peticion := httptest.NewRequest(http.MethodGet, "/api/vec/prueba", nil)
		peticion.RemoteAddr = peticionReal.RemoteAddr
		peticion.TLS = estado
		peticion = peticion.WithContext(ctx)
		grabador := httptest.NewRecorder()
		servidor.manejador.ServeHTTP(grabador, peticion)
		return grabador.Code
	}
	ctxExacto := context.WithValue(
		context.Background(), claveContextoConexionTLS{}, posesionReal,
	)
	if codigo := ejecutar(ctxExacto, peticionReal.TLS); codigo != http.StatusNoContent {
		t.Fatalf("posesion exacta = %d", codigo)
	}
	<-capturas

	conexionB, err := tls.Dial(
		"tcp", direccion,
		material.configCliente([]string{protocoloALPNHTTPUno}, true),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer conexionB.Close()
	if _, err := io.WriteString(
		conexionB,
		"GET /api/vec/prueba HTTP/1.1\r\nHost: interno.test\r\nConnection: keep-alive\r\n\r\n",
	); err != nil {
		t.Fatal(err)
	}
	respuestaB, err := http.ReadResponse(
		bufio.NewReader(conexionB),
		&http.Request{Method: http.MethodGet},
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = respuestaB.Body.Close()
	peticionB := <-capturas
	posesionB, valida := peticionB.Context().Value(
		claveContextoConexionTLS{},
	).(*posesionConexionTLS)
	if !valida || posesionB == nil || posesionB.conexion == nil ||
		posesionB.conexion == posesionReal.conexion {
		t.Fatal("segunda peticion no uso otra conexion TLS real")
	}
	visibleB := posesionB.conexion.ConnectionState()
	estadoHibrido := *peticionReal.TLS
	estadoHibrido.Version = visibleB.Version
	estadoHibrido.HandshakeComplete = visibleB.HandshakeComplete
	estadoHibrido.DidResume = visibleB.DidResume
	estadoHibrido.CipherSuite = visibleB.CipherSuite
	estadoHibrido.CurveID = visibleB.CurveID
	estadoHibrido.NegotiatedProtocol = visibleB.NegotiatedProtocol
	estadoHibrido.NegotiatedProtocolIsMutual = visibleB.NegotiatedProtocolIsMutual
	estadoHibrido.ServerName = visibleB.ServerName
	estadoHibrido.PeerCertificates = visibleB.PeerCertificates
	estadoHibrido.VerifiedChains = visibleB.VerifiedChains
	estadoHibrido.SignedCertificateTimestamps = visibleB.SignedCertificateTimestamps
	estadoHibrido.OCSPResponse = visibleB.OCSPResponse
	estadoHibrido.TLSUnique = visibleB.TLSUnique
	estadoHibrido.ECHAccepted = visibleB.ECHAccepted
	vinculoA, errA := peticionReal.TLS.ExportKeyingMaterial(
		etiquetaExportadorConexion, nil, 32,
	)
	vinculoB, errB := visibleB.ExportKeyingMaterial(
		etiquetaExportadorConexion, nil, 32,
	)
	if errA != nil || errB != nil || len(vinculoA) != 32 || len(vinculoB) != 32 ||
		bytes.Equal(vinculoA, vinculoB) {
		t.Fatalf("exportadores A/B no acreditan conexiones distintas: (%v, %v)", errA, errB)
	}
	ctxConexionB := context.WithValue(
		context.Background(), claveContextoConexionTLS{}, posesionB,
	)
	if codigo := ejecutar(ctxConexionB, &estadoHibrido); codigo != http.StatusBadRequest {
		t.Fatalf("estado visible B con exporter de A = %d", codigo)
	}

	otroServidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	ctxTokenAjeno := context.WithValue(
		context.Background(), claveContextoConexionTLS{}, &posesionConexionTLS{
			token: otroServidor.token, conexion: posesionReal.conexion,
		},
	)
	if codigo := ejecutar(ctxTokenAjeno, peticionReal.TLS); codigo != http.StatusBadRequest {
		t.Fatalf("token de otra capsula = %d", codigo)
	}

	extremoLocal, extremoRemoto := net.Pipe()
	conexionTLSAjena := tls.Server(extremoLocal, servidor.configuracionTLS.Clone())
	t.Cleanup(func() {
		_ = conexionTLSAjena.Close()
		_ = extremoRemoto.Close()
	})
	ctxConexionAjena := context.WithValue(
		context.Background(), claveContextoConexionTLS{}, &posesionConexionTLS{
			token: servidor.token, conexion: conexionTLSAjena,
		},
	)
	if codigo := ejecutar(ctxConexionAjena, peticionReal.TLS); codigo != http.StatusBadRequest {
		t.Fatalf("conexion TLS distinta = %d", codigo)
	}

	pipeFalso, parFalso := net.Pipe()
	falsa := &conexionConEstadoCopiado{Conn: pipeFalso, estado: *peticionReal.TLS}
	t.Cleanup(func() {
		_ = falsa.Close()
		_ = parFalso.Close()
	})
	ctxFalso := servidor.contextoConexionTLS(context.Background(), falsa)
	if ctxFalso.Value(claveContextoConexionTLS{}) != nil {
		t.Fatal("net.Conn con ConnectionState copiado recibio token")
	}
	if codigo := ejecutar(ctxFalso, peticionReal.TLS); codigo != http.StatusBadRequest {
		t.Fatalf("ConnectionState copiado = %d", codigo)
	}

	proxy := &proxyConexionTLS{Conn: posesionReal.conexion}
	ctxProxy := servidor.contextoConexionTLS(context.Background(), proxy)
	if ctxProxy.Value(claveContextoConexionTLS{}) != nil {
		t.Fatal("proxy de *tls.Conn recibio token")
	}
	if codigo := ejecutar(ctxProxy, peticionReal.TLS); codigo != http.StatusBadRequest {
		t.Fatalf("proxy TLS = %d", codigo)
	}
	if llamadas.Load() != 3 {
		t.Fatalf("peticiones que alcanzaron API = %d", llamadas.Load())
	}
}

func TestServidorInternoRechazaHTTPPlanoYTLS12(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	var llamadas atomic.Int32
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	direccion := iniciarServidorTLSPrueba(t, servidor)

	plana, err := net.DialTimeout("tcp", direccion, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = plana.SetDeadline(time.Now().Add(time.Second))
	if _, err := io.WriteString(
		plana,
		"GET /api/vec/prueba HTTP/1.1\r\nHost: interno.test\r\nConnection: close\r\n\r\n",
	); err != nil {
		t.Fatal(err)
	}
	respuesta, err := http.ReadResponse(
		bufio.NewReader(plana),
		&http.Request{Method: http.MethodGet},
	)
	_ = plana.Close()
	if err != nil {
		t.Fatal(err)
	}
	_ = respuesta.Body.Close()
	if respuesta.StatusCode == http.StatusNoContent {
		t.Fatal("HTTP plano alcanzo API")
	}

	clienteTLS12 := material.configCliente([]string{protocoloALPNHTTPUno}, true)
	clienteTLS12.MinVersion = tls.VersionTLS12
	clienteTLS12.MaxVersion = tls.VersionTLS12
	if conexion, err := tls.Dial("tcp", direccion, clienteTLS12); err == nil {
		_ = conexion.Close()
		t.Fatal("TLS 1.2 completo handshake")
	}
	if llamadas.Load() != 0 {
		t.Fatalf("protocolos no permitidos alcanzaron API: %d", llamadas.Load())
	}
}

func TestTLSListenerConCertificadoServidorAjenoNoObtieneTokenDeCapsula(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	otro := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	var llamadas atomic.Int32
	capsula, err := construirServidorInternoPrueba(t, material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	configuracionAjena := otro.configServidor.Clone()
	configuracionAjena.MinVersion = tls.VersionTLS13
	configuracionAjena.MaxVersion = tls.VersionTLS13
	configuracionAjena.NextProtos = []string{protocoloALPNHTTPUno}
	configuracionAjena.ClientAuth = tls.RequireAndVerifyClientCert
	configuracionAjena.ClientCAs = material.raicesClientes
	escuchaTCP, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	escuchaAjena := tls.NewListener(escuchaTCP, configuracionAjena)
	servidorAjeno := &http.Server{
		Handler:  capsula.manejador,
		ErrorLog: log.New(io.Discard, "", 0),
	}
	terminado := make(chan error, 1)
	go func() { terminado <- servidorAjeno.Serve(escuchaAjena) }()
	t.Cleanup(func() {
		_ = servidorAjeno.Close()
		<-terminado
	})
	cliente := material.configCliente([]string{protocoloALPNHTTPUno}, true)
	cliente.RootCAs = otro.raicesServidor
	conexion, err := tls.Dial("tcp", escuchaTCP.Addr().String(), cliente)
	if err != nil {
		t.Fatal(err)
	}
	defer conexion.Close()
	if _, err := io.WriteString(
		conexion,
		"GET /api/vec/prueba HTTP/1.1\r\nHost: interno.test\r\nConnection: close\r\n\r\n",
	); err != nil {
		t.Fatal(err)
	}
	respuesta, err := http.ReadResponse(
		bufio.NewReader(conexion),
		&http.Request{Method: http.MethodGet},
	)
	if err != nil {
		t.Fatal(err)
	}
	_ = respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusBadRequest || llamadas.Load() != 0 {
		t.Fatalf("tls.Listener ajeno = (%d, %d)", respuesta.StatusCode, llamadas.Load())
	}
}

func TestServidorInternoAceptaSNIRealDNSMixtoEIPSinSNI(t *testing.T) {
	casos := []struct {
		nombre        string
		opciones      opcionesCertificadoServidor
		nombreCliente string
	}{
		{
			nombre:        "DNS ASCII mezcla mayusculas",
			nombreCliente: "SeRvIdOr.InTeRnA.TeSt",
		},
		{
			nombre:        "IP literal omite SNI",
			opciones:      opcionesCertificadoServidor{sanIP: true},
			nombreCliente: "127.0.0.1",
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			material := materialTLSMutuoPrueba(t, caso.opciones)
			var llamadas atomic.Int32
			servidor, err := construirServidorInternoPrueba(t, material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				llamadas.Add(1)
				w.WriteHeader(http.StatusNoContent)
			}))
			if err != nil {
				t.Fatal(err)
			}
			direccion := iniciarServidorTLSPrueba(t, servidor)
			cliente := material.configCliente([]string{protocoloALPNHTTPUno}, true)
			cliente.ServerName = caso.nombreCliente
			conexion, err := tls.Dial("tcp", direccion, cliente)
			if err != nil {
				t.Fatal(err)
			}
			defer conexion.Close()
			if _, err := io.WriteString(
				conexion,
				"GET /api/vec/prueba HTTP/1.1\r\nHost: interno.test\r\nConnection: close\r\n\r\n",
			); err != nil {
				t.Fatal(err)
			}
			respuesta, err := http.ReadResponse(
				bufio.NewReader(conexion),
				&http.Request{Method: http.MethodGet},
			)
			if err != nil {
				t.Fatal(err)
			}
			_ = respuesta.Body.Close()
			if respuesta.StatusCode != http.StatusNoContent || llamadas.Load() != 1 {
				t.Fatalf("SNI real = (%d, %d)", respuesta.StatusCode, llamadas.Load())
			}
		})
	}
}

func TestHandshakeExigePosesionDeClavePrivadaCliente(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	otro := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	var llamadas atomic.Int32
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	direccion := iniciarServidorTLSPrueba(t, servidor)
	cliente := material.configCliente([]string{protocoloALPNHTTPUno}, true)
	cliente.Certificates[0].PrivateKey = otro.cliente.PrivateKey
	conexion, err := tls.Dial("tcp", direccion, cliente)
	if err == nil {
		_, errEscritura := io.WriteString(
			conexion,
			"GET /api/vec/prueba HTTP/1.1\r\nHost: interno.test\r\nConnection: close\r\n\r\n",
		)
		_, errLectura := http.ReadResponse(
			bufio.NewReader(conexion),
			&http.Request{Method: http.MethodGet},
		)
		_ = conexion.Close()
		if errEscritura == nil && errLectura == nil {
			t.Fatal("certificado cliente sin su clave privada alcanzo HTTP")
		}
	}
	if llamadas.Load() != 0 {
		t.Fatalf("clave cliente ajena alcanzo API: %d", llamadas.Load())
	}
}

func TestServidorInternoUsoUnicoYApagadoIdempotente(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	_ = iniciarServidorTLSPrueba(t, servidor)
	if err := servidor.EscucharYServir(); err != ErrServidorInternoInvalido {
		t.Fatalf("segunda escucha = %v", err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelar()
	if err := servidor.Apagar(ctx); err != nil {
		t.Fatal(err)
	}
	if err := servidor.Apagar(ctx); err != nil {
		t.Fatalf("apagado repetido = %v", err)
	}
}

func TestApagarAntesDeEscucharCancelaSinAbrirRed(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	if err := servidor.Apagar(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := servidor.EscucharYServir(); err != ErrServidorInternoInvalido {
		t.Fatalf("escucha tras apagado = %v", err)
	}
}

func TestApagarEsperaVentanaEntreServeYTermino(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	servidor.ejecucion.estado.Store(estadoServidorInternoEscuchando)
	servidor.notificarListo()
	liberar := make(chan struct{})
	go func() {
		<-liberar
		servidor.ejecucion.estado.Store(estadoServidorInternoTerminado)
		servidor.notificarTerminado()
	}()
	base, cancelar := context.WithTimeout(context.Background(), time.Second)
	defer cancelar()
	ctx := &contextoApagadoObservado{
		Context:   base,
		esperando: make(chan struct{}),
	}
	resultado := make(chan error, 1)
	go func() { resultado <- servidor.Apagar(ctx) }()
	select {
	case <-ctx.esperando:
	case <-base.Done():
		t.Fatal("Apagar no alcanzo espera de termino")
	}
	select {
	case err := <-resultado:
		t.Fatalf("Apagar no espero termino: %v", err)
	default:
	}
	close(liberar)
	if err := <-resultado; err != nil {
		t.Fatalf("Apagar tras termino = %v", err)
	}
}
