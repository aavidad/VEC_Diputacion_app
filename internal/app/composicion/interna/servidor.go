package interna

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/server"
)

var (
	ErrAPIInternaNoDisponible     = errors.New("composicion interna: API interna no disponible")
	ErrTLSMutuoNoVerificado       = errors.New("composicion interna: TLS mutuo no verificado")
	ErrServidorInternoInvalido    = errors.New("composicion interna: servidor no construido por la raiz interna")
	ErrEscuchaInternaNoDisponible = errors.New(
		"composicion interna: escucha TLS no disponible",
	)
)

const protocoloALPNHTTPUno = "http/1.1"

const (
	estadoServidorInternoNuevo uint32 = iota
	estadoServidorInternoEscuchando
	estadoServidorInternoTerminado
)

const etiquetaExportadorConexion = "EXPORTER-VEC-INTERNAL-CONNECTION-BINDING-v1"

type claveContextoConexionTLS struct{}

type tokenServidorInterno struct {
	// El tipo no puede ser de tamano cero: dos punteros a valores de tamano
	// cero no tienen por que ser distintos segun el lenguaje.
	marca byte
}

type posesionConexionTLS struct {
	token    *tokenServidorInterno
	conexion *tls.Conn
}

// ServidorInterno es una capsula opaca y de un solo uso. No incrusta ni
// publica http.Server, Handler, Listener, tls.Config, certificados o pools.
// El unico camino de escucha construye localmente el transporte sellado.
type ServidorInterno struct {
	direccionEscucha     string
	tiempoCabeceras      time.Duration
	tiempoLectura        time.Duration
	tiempoEscritura      time.Duration
	tiempoInactividad    time.Duration
	maximoBytesCabeceras int
	manejador            *manejadorInternoVerificado
	configuracionTLS     *tls.Config
	token                *tokenServidorInterno
	propietario          *ServidorInterno
	ejecucion            *ejecucionServidorInterno
	propiedadAplicacion  *atomic.Bool
}

type ejecucionServidorInterno struct {
	estado         atomic.Uint32
	mu             sync.Mutex
	servidorActivo *http.Server
	listo          chan struct{}
	marcarListo    sync.Once
	terminado      chan struct{}
	marcarTermino  sync.Once
}

// manejadorInternoVerificado sella la procedencia del handler. Sus campos son
// privados: el cmd no puede sustituirlo por DefaultServeMux ni por un handler
// que evite la lista positiva de server.NewHTTPServerInterno.
type manejadorInternoVerificado struct {
	siguiente            http.Handler
	token                *tokenServidorInterno
	direccionEscucha     string
	tiempoCabeceras      time.Duration
	tiempoLectura        time.Duration
	tiempoEscritura      time.Duration
	tiempoInactividad    time.Duration
	maximoBytesCabeceras int
	materialTLS          materialTLSAprobado
	protocolosTLS        []string
	protocolosHTTP       protocolosHTTPAprobados
	desactivarOPTIONS    bool
}

type materialTLSAprobado struct {
	autoridadesClientes        *x509.CertPool
	certificadosAutoridades    []*x509.Certificate
	nombreServidor             string
	certificadoServidor        tls.Certificate
	huellaCadenaServidor       [sha256.Size]byte
	huellaClavePublicaServidor [sha256.Size]byte
	huellaClavePrivadaServidor [sha256.Size]byte
	huellaCertPEM              [sha256.Size]byte
	huellaClavePEM             [sha256.Size]byte
	huellaCAPEM                [sha256.Size]byte
}

type protocolosHTTPAprobados struct {
	httpUno          bool
	httpDos          bool
	httpDosSinCifrar bool
}

func (m *manejadorInternoVerificado) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !m.peticionTLSMutuaVerificada(r) {
		w.Header().Set("Connection", "close")
		http.Error(w, "solicitud no disponible", http.StatusBadRequest)
		return
	}
	m.siguiente.ServeHTTP(w, r)
}

func (m *manejadorInternoVerificado) peticionTLSMutuaVerificada(r *http.Request) bool {
	if m == nil || m.token == nil || r == nil || r.TLS == nil ||
		r.ProtoMajor != 1 || r.ProtoMinor != 1 {
		return false
	}
	posesion, valida := r.Context().Value(claveContextoConexionTLS{}).(*posesionConexionTLS)
	if !valida || posesion == nil || posesion.token != m.token || posesion.conexion == nil ||
		!estadoTLSCoherenteConConexion(r.TLS, posesion.conexion) {
		return false
	}
	estado := r.TLS
	if !estado.HandshakeComplete || estado.Version != tls.VersionTLS13 || estado.DidResume ||
		estado.NegotiatedProtocol != protocoloALPNHTTPUno ||
		!estado.NegotiatedProtocolIsMutual ||
		!sniTLSCoherente(m.materialTLS.nombreServidor, estado.ServerName) ||
		!cifradoTLS13Aprobado(estado.CipherSuite) || estado.CurveID == 0 ||
		len(estado.TLSUnique) != 0 || len(estado.PeerCertificates) == 0 ||
		len(estado.VerifiedChains) == 0 || len(estado.VerifiedChains[0]) == 0 ||
		estado.VerifiedChains[0][0] == nil || estado.PeerCertificates[0] == nil ||
		!estado.VerifiedChains[0][0].Equal(estado.PeerCertificates[0]) {
		return false
	}
	hoja := estado.PeerCertificates[0]
	if hoja.IsCA || !contieneUsoExtendido(hoja.ExtKeyUsage, x509.ExtKeyUsageClientAuth) {
		return false
	}
	intermedias := x509.NewCertPool()
	for _, certificado := range estado.PeerCertificates[1:] {
		if certificado == nil {
			return false
		}
		intermedias.AddCert(certificado)
	}
	cadenas, err := hoja.Verify(x509.VerifyOptions{
		Roots:         m.materialTLS.autoridadesClientes,
		Intermediates: intermedias,
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	return err == nil && len(cadenas) != 0
}

func estadoTLSCoherenteConConexion(estado *tls.ConnectionState, conexion *tls.Conn) bool {
	if estado == nil || conexion == nil {
		return false
	}
	actual := conexion.ConnectionState()
	if estado.Version != actual.Version ||
		estado.HandshakeComplete != actual.HandshakeComplete ||
		estado.DidResume != actual.DidResume ||
		estado.CipherSuite != actual.CipherSuite ||
		estado.CurveID != actual.CurveID ||
		estado.NegotiatedProtocol != actual.NegotiatedProtocol ||
		estado.NegotiatedProtocolIsMutual != actual.NegotiatedProtocolIsMutual ||
		estado.ServerName != actual.ServerName ||
		estado.ECHAccepted != actual.ECHAccepted ||
		!certificadosMismaConexion(estado.PeerCertificates, actual.PeerCertificates) ||
		!cadenasMismaConexion(estado.VerifiedChains, actual.VerifiedChains) ||
		!bytesBidimensionalesIguales(
			estado.SignedCertificateTimestamps,
			actual.SignedCertificateTimestamps,
		) || !bytes.Equal(estado.OCSPResponse, actual.OCSPResponse) ||
		!bytes.Equal(estado.TLSUnique, actual.TLSUnique) {
		return false
	}
	vinculoPeticion, errPeticion := estado.ExportKeyingMaterial(
		etiquetaExportadorConexion, nil, sha256.Size,
	)
	vinculoConexion, errConexion := actual.ExportKeyingMaterial(
		etiquetaExportadorConexion, nil, sha256.Size,
	)
	return errPeticion == nil && errConexion == nil &&
		len(vinculoPeticion) == sha256.Size && len(vinculoConexion) == sha256.Size &&
		subtle.ConstantTimeCompare(vinculoPeticion, vinculoConexion) == 1
}

func certificadosMismaConexion(izquierda, derecha []*x509.Certificate) bool {
	if len(izquierda) != len(derecha) {
		return false
	}
	for indice := range izquierda {
		if izquierda[indice] != derecha[indice] {
			return false
		}
	}
	return true
}

func cadenasMismaConexion(izquierda, derecha [][]*x509.Certificate) bool {
	if len(izquierda) != len(derecha) {
		return false
	}
	for indice := range izquierda {
		if !certificadosMismaConexion(izquierda[indice], derecha[indice]) {
			return false
		}
	}
	return true
}

func sniTLSCoherente(nombreConfigurado, sni string) bool {
	if net.ParseIP(nombreConfigurado) != nil {
		return sni == ""
	}
	return igualesASCIIIgnorandoMayusculas(nombreConfigurado, sni)
}

func igualesASCIIIgnorandoMayusculas(izquierda, derecha string) bool {
	if izquierda == "" || len(izquierda) != len(derecha) {
		return false
	}
	for indice := range len(izquierda) {
		a, b := izquierda[indice], derecha[indice]
		if a > 0x7f || b > 0x7f {
			return false
		}
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

func cifradoTLS13Aprobado(cifrado uint16) bool {
	switch cifrado {
	case tls.TLS_AES_128_GCM_SHA256, tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256:
		return true
	default:
		return false
	}
}

// construirServidorInterno es el unico puente futuro entre C5/C6 y net/http.
// Permanece no exportado y C4 no lo invoca: solo acepta una API no nula y una
// configuracion que contiene las tres referencias TLS autoritativas. Carga ese
// material directamente y siempre delega la allowlist a server.NewHTTPServerInterno.
func construirServidorInterno(
	cfg Configuracion,
	api http.Handler,
) (*ServidorInterno, error) {
	if err := cfg.Validar(); err != nil {
		return nil, err
	}
	if manejadorNulo(api) || esMuxPredeterminado(api) {
		return nil, ErrAPIInternaNoDisponible
	}
	materialCargado, err := cargarMaterialTLS(cfg)
	if err != nil {
		return nil, ErrTLSMutuoNoVerificado
	}
	return construirServidorInternoConMaterial(cfg, api, materialCargado)
}

func construirServidorInternoConMaterial(
	cfg Configuracion,
	api http.Handler,
	materialCargado materialTLSCargado,
) (*ServidorInterno, error) {
	if err := cfg.Validar(); err != nil {
		return nil, err
	}
	if manejadorNulo(api) || esMuxPredeterminado(api) {
		return nil, ErrAPIInternaNoDisponible
	}
	if err := validarTLSMutuo(materialCargado.configuracion); err != nil {
		return nil, ErrTLSMutuoNoVerificado
	}

	servidorHTTP, err := server.NewHTTPServerInterno(config.Config{
		Address:             cfg.DireccionEscucha,
		ReadHeaderTimeout:   cfg.TiempoCabeceras,
		ReadTimeout:         cfg.TiempoLectura,
		WriteTimeout:        cfg.TiempoEscritura,
		IdleTimeout:         cfg.TiempoInactividad,
		MaxHeaderBytes:      cfg.MaximoBytesCabeceras,
		MaxRequestBodyBytes: cfg.MaximoBytesPeticion,
		HTTPAllowedCIDRs:    append([]string(nil), cfg.RedesPermitidas...),
		ExecutionProfile:    config.ExecutionProfileProduction,
		AuthMode:            config.AuthModeDisabled,
		StorageMode:         config.StorageModeLocalDurable,
	}, api)
	if err != nil || servidorHTTP == nil || servidorHTTP.Handler == nil {
		return nil, ErrServidorInternoInvalido
	}
	configuracionTLS := clonarConfiguracionTLSMutuo(materialCargado.configuracion)
	materialTLS, err := aprobarMaterialTLS(configuracionTLS, materialCargado)
	if err != nil {
		return nil, ErrTLSMutuoNoVerificado
	}
	token := &tokenServidorInterno{marca: 1}
	manejador := &manejadorInternoVerificado{
		siguiente:            servidorHTTP.Handler,
		token:                token,
		direccionEscucha:     servidorHTTP.Addr,
		tiempoCabeceras:      servidorHTTP.ReadHeaderTimeout,
		tiempoLectura:        servidorHTTP.ReadTimeout,
		tiempoEscritura:      servidorHTTP.WriteTimeout,
		tiempoInactividad:    servidorHTTP.IdleTimeout,
		maximoBytesCabeceras: servidorHTTP.MaxHeaderBytes,
		materialTLS:          materialTLS,
		protocolosTLS:        append([]string(nil), configuracionTLS.NextProtos...),
		protocolosHTTP:       protocolosHTTPAprobados{httpUno: true},
		desactivarOPTIONS:    true,
	}
	servidor := &ServidorInterno{
		direccionEscucha:     servidorHTTP.Addr,
		tiempoCabeceras:      servidorHTTP.ReadHeaderTimeout,
		tiempoLectura:        servidorHTTP.ReadTimeout,
		tiempoEscritura:      servidorHTTP.WriteTimeout,
		tiempoInactividad:    servidorHTTP.IdleTimeout,
		maximoBytesCabeceras: servidorHTTP.MaxHeaderBytes,
		manejador:            manejador,
		configuracionTLS:     configuracionTLS,
		token:                token,
		propiedadAplicacion:  &atomic.Bool{},
		ejecucion: &ejecucionServidorInterno{
			listo:     make(chan struct{}),
			terminado: make(chan struct{}),
		},
	}
	servidor.propietario = servidor
	if err := validarServidorInterno(servidor); err != nil {
		return nil, err
	}
	return servidor, nil
}

func validarServidorInterno(servidor *ServidorInterno) error {
	if servidor == nil || servidor.propietario != servidor || servidor.ejecucion == nil ||
		servidor.manejador == nil || servidor.token == nil || servidor.configuracionTLS == nil ||
		servidor.propiedadAplicacion == nil ||
		servidor.ejecucion.listo == nil || servidor.ejecucion.terminado == nil {
		return ErrServidorInternoInvalido
	}
	if err := validarTLSMutuo(servidor.configuracionTLS); err != nil {
		return err
	}
	manejador := servidor.manejador
	if manejadorNulo(manejador.siguiente) || manejador.token != servidor.token ||
		manejador.direccionEscucha == "" || servidor.direccionEscucha != manejador.direccionEscucha ||
		servidor.tiempoCabeceras != manejador.tiempoCabeceras ||
		servidor.tiempoLectura != manejador.tiempoLectura ||
		servidor.tiempoEscritura != manejador.tiempoEscritura ||
		servidor.tiempoInactividad != manejador.tiempoInactividad ||
		servidor.maximoBytesCabeceras != manejador.maximoBytesCabeceras ||
		!manejador.desactivarOPTIONS ||
		manejador.protocolosHTTP != (protocolosHTTPAprobados{httpUno: true}) ||
		!slices.Equal(servidor.configuracionTLS.NextProtos, manejador.protocolosTLS) {
		return ErrServidorInternoInvalido
	}
	if !manejador.materialTLS.coincide(servidor.configuracionTLS) {
		return ErrTLSMutuoNoVerificado
	}
	return nil
}

// EscucharYServir es el unico punto publico que abre red. Es atomico y de un
// solo uso: no admite listeners, handlers ni material TLS proporcionados por
// el llamador.
func (servidor *ServidorInterno) EscucharYServir() error {
	if servidor == nil || servidor.propietario != servidor || servidor.ejecucion == nil ||
		!servidor.ejecucion.estado.CompareAndSwap(
			estadoServidorInternoNuevo,
			estadoServidorInternoEscuchando,
		) {
		return ErrServidorInternoInvalido
	}
	defer func() {
		servidor.ejecucion.estado.Store(estadoServidorInternoTerminado)
		servidor.notificarListo()
		servidor.notificarTerminado()
	}()
	if err := validarServidorInterno(servidor); err != nil {
		return err
	}

	escucha, err := net.Listen("tcp", servidor.direccionEscucha)
	if err != nil {
		return ErrEscuchaInternaNoDisponible
	}
	servidorHTTP := servidor.nuevoServidorHTTP()
	servidor.ejecucion.mu.Lock()
	servidor.ejecucion.servidorActivo = servidorHTTP
	servidor.ejecucion.mu.Unlock()
	servidor.notificarListo()
	err = servidorHTTP.ServeTLS(escucha, "", "")
	servidor.ejecucion.mu.Lock()
	if servidor.ejecucion.servidorActivo == servidorHTTP {
		servidor.ejecucion.servidorActivo = nil
	}
	servidor.ejecucion.mu.Unlock()
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return ErrEscuchaInternaNoDisponible
}

// Apagar detiene ordenadamente la capsula sin revelar el servidor HTTP ni el
// listener. Si se llama antes de que arranque, cancela de forma atomica su
// unico intento de escucha.
func (servidor *ServidorInterno) Apagar(ctx context.Context) error {
	if servidor == nil || servidor.propietario != servidor || servidor.ejecucion == nil || ctx == nil {
		return ErrServidorInternoInvalido
	}
	for {
		switch servidor.ejecucion.estado.Load() {
		case estadoServidorInternoNuevo:
			if servidor.ejecucion.estado.CompareAndSwap(
				estadoServidorInternoNuevo,
				estadoServidorInternoTerminado,
			) {
				servidor.notificarListo()
				servidor.notificarTerminado()
				return nil
			}
		case estadoServidorInternoEscuchando:
			select {
			case <-servidor.ejecucion.listo:
			case <-ctx.Done():
				return ErrEscuchaInternaNoDisponible
			}
			servidor.ejecucion.mu.Lock()
			activo := servidor.ejecucion.servidorActivo
			servidor.ejecucion.mu.Unlock()
			if activo == nil {
				select {
				case <-servidor.ejecucion.terminado:
					return nil
				case <-ctx.Done():
					return ErrEscuchaInternaNoDisponible
				}
			}
			if err := activo.Shutdown(ctx); err != nil {
				return ErrEscuchaInternaNoDisponible
			}
			select {
			case <-servidor.ejecucion.terminado:
				return nil
			case <-ctx.Done():
				return ErrEscuchaInternaNoDisponible
			}
		case estadoServidorInternoTerminado:
			return nil
		default:
			return ErrServidorInternoInvalido
		}
	}
}

func (servidor *ServidorInterno) notificarListo() {
	if servidor != nil && servidor.ejecucion != nil && servidor.ejecucion.listo != nil {
		servidor.ejecucion.marcarListo.Do(func() { close(servidor.ejecucion.listo) })
	}
}

func (servidor *ServidorInterno) notificarTerminado() {
	if servidor != nil && servidor.ejecucion != nil && servidor.ejecucion.terminado != nil {
		servidor.ejecucion.marcarTermino.Do(func() { close(servidor.ejecucion.terminado) })
	}
}

func (servidor *ServidorInterno) nuevoServidorHTTP() *http.Server {
	protocolos := &http.Protocols{}
	protocolos.SetHTTP1(true)
	return &http.Server{
		Addr:                         servidor.direccionEscucha,
		Handler:                      servidor.manejador,
		DisableGeneralOptionsHandler: true,
		TLSConfig:                    clonarConfiguracionTLSMutuo(servidor.configuracionTLS),
		ReadHeaderTimeout:            servidor.tiempoCabeceras,
		ReadTimeout:                  servidor.tiempoLectura,
		WriteTimeout:                 servidor.tiempoEscritura,
		IdleTimeout:                  servidor.tiempoInactividad,
		MaxHeaderBytes:               servidor.maximoBytesCabeceras,
		TLSNextProto:                 nil,
		ConnState:                    controlarEstadoConexionTLS,
		ErrorLog:                     log.New(nuevoEscritorEventosHTTPSaneados(os.Stderr), "", 0),
		BaseContext:                  contextoBaseServidorInterno,
		ConnContext:                  servidor.contextoConexionTLS,
		HTTP2:                        nil,
		Protocols:                    protocolos,
	}
}

func contextoBaseServidorInterno(net.Listener) context.Context {
	return context.Background()
}

func (servidor *ServidorInterno) contextoConexionTLS(
	ctx context.Context,
	conexion net.Conn,
) context.Context {
	conexionTLS, valida := conexion.(*tls.Conn)
	if !valida || conexionTLS == nil || servidor == nil || servidor.token == nil {
		return ctx
	}
	return context.WithValue(ctx, claveContextoConexionTLS{}, &posesionConexionTLS{
		token:    servidor.token,
		conexion: conexionTLS,
	})
}

func controlarEstadoConexionTLS(conexion net.Conn, _ http.ConnState) {
	if _, valida := conexion.(*tls.Conn); !valida && conexion != nil {
		_ = conexion.Close()
	}
}

func validarTLSMutuo(configuracion *tls.Config) error {
	if configuracion == nil || configuracion.Rand != nil || configuracion.Time != nil ||
		configuracion.MinVersion != tls.VersionTLS13 ||
		configuracion.MaxVersion != tls.VersionTLS13 ||
		configuracion.ClientAuth != tls.RequireAndVerifyClientCert ||
		!poolCertificadosConAutoridades(configuracion.ClientCAs) ||
		len(configuracion.Certificates) != 1 ||
		configuracion.GetConfigForClient != nil || configuracion.GetCertificate != nil ||
		configuracion.GetClientCertificate != nil || configuracion.NameToCertificate != nil ||
		configuracion.VerifyPeerCertificate != nil || configuracion.VerifyConnection != nil ||
		configuracion.RootCAs != nil || configuracion.ServerName != "" ||
		configuracion.InsecureSkipVerify || len(configuracion.CipherSuites) != 0 ||
		configuracion.PreferServerCipherSuites || !configuracion.SessionTicketsDisabled ||
		configuracion.SessionTicketKey != ([32]byte{}) || configuracion.ClientSessionCache != nil ||
		configuracion.UnwrapSession != nil || configuracion.WrapSession != nil ||
		len(configuracion.CurvePreferences) != 0 || configuracion.DynamicRecordSizingDisabled ||
		configuracion.Renegotiation != tls.RenegotiateNever ||
		configuracion.KeyLogWriter != nil || len(configuracion.EncryptedClientHelloConfigList) != 0 ||
		configuracion.EncryptedClientHelloRejectionVerify != nil ||
		configuracion.GetEncryptedClientHelloKeys != nil ||
		len(configuracion.EncryptedClientHelloKeys) != 0 ||
		!slices.Equal(configuracion.NextProtos, []string{protocoloALPNHTTPUno}) {
		return ErrTLSMutuoNoVerificado
	}
	certificado := configuracion.Certificates[0]
	if len(certificado.Certificate) == 0 || certificado.PrivateKey == nil {
		return ErrTLSMutuoNoVerificado
	}
	if _, _, _, err := resumirCertificadoServidor(certificado); err != nil {
		return ErrTLSMutuoNoVerificado
	}
	return nil
}

func poolCertificadosConAutoridades(pool *x509.CertPool) bool {
	return pool != nil && len(pool.Subjects()) != 0
}

func manejadorNulo(manejador http.Handler) bool {
	if manejador == nil {
		return true
	}
	valor := reflect.ValueOf(manejador)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func esMuxPredeterminado(manejador http.Handler) bool {
	mux, valido := manejador.(*http.ServeMux)
	return valido && mux == http.DefaultServeMux
}
