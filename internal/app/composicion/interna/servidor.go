package interna

import (
	"bytes"
	"crypto"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"net"
	"net/http"
	"reflect"
	"slices"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/server"
)

var (
	ErrAPIInternaNoDisponible  = errors.New("composicion interna: API interna no disponible")
	ErrTLSMutuoNoVerificado    = errors.New("composicion interna: TLS mutuo no verificado")
	ErrServidorInternoInvalido = errors.New("composicion interna: servidor no construido por la raiz interna")
)

const protocoloALPNHTTPUno = "http/1.1"

// manejadorInternoVerificado sella la procedencia del handler. Sus campos son
// privados: el cmd no puede sustituirlo por DefaultServeMux ni por un handler
// que evite la lista positiva de server.NewHTTPServerInterno.
type manejadorInternoVerificado struct {
	siguiente            http.Handler
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
	if r == nil || r.TLS == nil || r.ProtoMajor != 1 || r.ProtoMinor != 1 {
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

func sniTLSCoherente(nombreConfigurado, sni string) bool {
	if net.ParseIP(nombreConfigurado) != nil {
		return sni == ""
	}
	return nombreConfigurado != "" && sni == nombreConfigurado
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
) (*http.Server, error) {
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
) (*http.Server, error) {
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
	servidorHTTP.DisableGeneralOptionsHandler = true
	servidorHTTP.TLSNextProto = nil
	servidorHTTP.HTTP2 = nil
	servidorHTTP.Protocols = &http.Protocols{}
	servidorHTTP.Protocols.SetHTTP1(true)
	configuracionTLS := clonarConfiguracionTLSMutuo(materialCargado.configuracion)
	materialTLS, err := aprobarMaterialTLS(configuracionTLS, materialCargado)
	if err != nil {
		return nil, ErrTLSMutuoNoVerificado
	}
	servidorHTTP.Handler = &manejadorInternoVerificado{
		siguiente:            servidorHTTP.Handler,
		direccionEscucha:     servidorHTTP.Addr,
		tiempoCabeceras:      servidorHTTP.ReadHeaderTimeout,
		tiempoLectura:        servidorHTTP.ReadTimeout,
		tiempoEscritura:      servidorHTTP.WriteTimeout,
		tiempoInactividad:    servidorHTTP.IdleTimeout,
		maximoBytesCabeceras: servidorHTTP.MaxHeaderBytes,
		materialTLS:          materialTLS,
		protocolosTLS:        append([]string(nil), configuracionTLS.NextProtos...),
		protocolosHTTP:       obtenerProtocolosHTTP(servidorHTTP.Protocols),
		desactivarOPTIONS:    servidorHTTP.DisableGeneralOptionsHandler,
	}
	servidorHTTP.TLSConfig = configuracionTLS
	if err := ValidarServidorParaEscucha(servidorHTTP); err != nil {
		return nil, err
	}
	return servidorHTTP, nil
}

// ValidarServidorParaEscucha es la ultima barrera usada por cmd/vec-interno.
// Revalida el material mutable y la procedencia del handler inmediatamente
// antes de ListenAndServeTLS.
func ValidarServidorParaEscucha(servidorHTTP *http.Server) error {
	if servidorHTTP == nil || servidorHTTP.Handler == nil {
		return ErrServidorInternoInvalido
	}
	if err := validarTLSMutuo(servidorHTTP.TLSConfig); err != nil {
		return err
	}
	manejador, valido := servidorHTTP.Handler.(*manejadorInternoVerificado)
	if !valido || manejador == nil || manejadorNulo(manejador.siguiente) ||
		manejador.direccionEscucha == "" || servidorHTTP.Addr != manejador.direccionEscucha ||
		servidorHTTP.ReadHeaderTimeout != manejador.tiempoCabeceras ||
		servidorHTTP.ReadTimeout != manejador.tiempoLectura ||
		servidorHTTP.WriteTimeout != manejador.tiempoEscritura ||
		servidorHTTP.IdleTimeout != manejador.tiempoInactividad ||
		servidorHTTP.MaxHeaderBytes != manejador.maximoBytesCabeceras ||
		servidorHTTP.DisableGeneralOptionsHandler != manejador.desactivarOPTIONS ||
		!servidorHTTP.DisableGeneralOptionsHandler || servidorHTTP.TLSNextProto != nil ||
		servidorHTTP.HTTP2 != nil || servidorHTTP.Protocols == nil ||
		obtenerProtocolosHTTP(servidorHTTP.Protocols) != manejador.protocolosHTTP ||
		manejador.protocolosHTTP != (protocolosHTTPAprobados{httpUno: true}) ||
		!slices.Equal(servidorHTTP.TLSConfig.NextProtos, manejador.protocolosTLS) {
		return ErrServidorInternoInvalido
	}
	if !manejador.materialTLS.coincide(servidorHTTP.TLSConfig) {
		return ErrTLSMutuoNoVerificado
	}
	return nil
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

// clonarConfiguracionTLSMutuo separa todas las referencias mutables entre la
// carga, el servidor y el sello. La clave se recodifica como PKCS#8.
func clonarConfiguracionTLSMutuo(origen *tls.Config) *tls.Config {
	clon := origen.Clone()
	clon.ClientCAs = origen.ClientCAs.Clone()
	clon.NextProtos = append([]string(nil), origen.NextProtos...)
	clon.Certificates = make([]tls.Certificate, len(origen.Certificates))
	for indice := range origen.Certificates {
		clon.Certificates[indice] = clonarCertificadoTLS(origen.Certificates[indice])
		clave, err := clonarClavePrivada(origen.Certificates[indice].PrivateKey)
		if err == nil {
			clon.Certificates[indice].PrivateKey = clave
		}
	}
	return clon
}

func clonarCertificadoTLS(origen tls.Certificate) tls.Certificate {
	clon := origen
	clon.Certificate = clonarBytesBidimensionales(origen.Certificate)
	clon.SupportedSignatureAlgorithms = append(
		[]tls.SignatureScheme(nil), origen.SupportedSignatureAlgorithms...,
	)
	clon.OCSPStaple = append([]byte(nil), origen.OCSPStaple...)
	clon.SignedCertificateTimestamps = clonarBytesBidimensionales(
		origen.SignedCertificateTimestamps,
	)
	if len(clon.Certificate) != 0 {
		// validarTLSMutuo ya ha comprobado la cadena; esta nueva instancia evita
		// compartir el puntero mutable Leaf con el proveedor.
		clon.Leaf, _ = x509.ParseCertificate(clon.Certificate[0])
	}
	return clon
}

func clonarBytesBidimensionales(origen [][]byte) [][]byte {
	if origen == nil {
		return nil
	}
	clon := make([][]byte, len(origen))
	for indice := range origen {
		clon[indice] = append([]byte(nil), origen[indice]...)
	}
	return clon
}

func aprobarMaterialTLS(
	configuracion *tls.Config,
	cargado materialTLSCargado,
) (materialTLSAprobado, error) {
	huellaCadena, huellaClave, huellaPrivada, err := resumirCertificadoServidor(configuracion.Certificates[0])
	if err != nil {
		return materialTLSAprobado{}, err
	}
	certificado := clonarCertificadoTLS(configuracion.Certificates[0])
	// La aprobacion conserva material publico y metadatos, no duplica la clave.
	certificado.PrivateKey = nil
	return materialTLSAprobado{
		autoridadesClientes:        configuracion.ClientCAs.Clone(),
		certificadosAutoridades:    clonarCertificadosX509(cargado.autoridadesClientes),
		nombreServidor:             cargado.nombreServidor,
		certificadoServidor:        certificado,
		huellaCadenaServidor:       huellaCadena,
		huellaClavePublicaServidor: huellaClave,
		huellaClavePrivadaServidor: huellaPrivada,
		huellaCertPEM:              cargado.huellaCertPEM,
		huellaClavePEM:             cargado.huellaClavePEM,
		huellaCAPEM:                cargado.huellaCAPEM,
	}, nil
}

func (aprobado materialTLSAprobado) coincide(configuracion *tls.Config) bool {
	if configuracion == nil || configuracion.ClientCAs == nil ||
		aprobado.autoridadesClientes == nil ||
		!configuracion.ClientCAs.Equal(aprobado.autoridadesClientes) ||
		len(configuracion.Certificates) != 1 {
		return false
	}
	actual := configuracion.Certificates[0]
	ahora := time.Now()
	for _, autoridad := range aprobado.certificadosAutoridades {
		if validarAutoridad(autoridad, ahora) != nil {
			return false
		}
	}
	cadena := make([]*x509.Certificate, len(actual.Certificate))
	for indice, der := range actual.Certificate {
		parseado, err := x509.ParseCertificate(der)
		if err != nil {
			return false
		}
		cadena[indice] = parseado
	}
	if validarCadenaServidor(
		Configuracion{NombreServidorTLS: aprobado.nombreServidor}, cadena, actual,
	) != nil {
		return false
	}
	huellaCadena, huellaClave, huellaPrivada, err := resumirCertificadoServidor(actual)
	if err != nil || huellaCadena != aprobado.huellaCadenaServidor ||
		huellaClave != aprobado.huellaClavePublicaServidor ||
		huellaPrivada != aprobado.huellaClavePrivadaServidor ||
		aprobado.huellaCertPEM == ([sha256.Size]byte{}) ||
		aprobado.huellaClavePEM == ([sha256.Size]byte{}) ||
		aprobado.huellaCAPEM == ([sha256.Size]byte{}) {
		return false
	}
	return certificadoTLSEquivalente(actual, aprobado.certificadoServidor)
}

func clonarCertificadosX509(origen []*x509.Certificate) []*x509.Certificate {
	clon := make([]*x509.Certificate, 0, len(origen))
	for _, certificado := range origen {
		if certificado == nil {
			clon = append(clon, nil)
			continue
		}
		parseado, err := x509.ParseCertificate(append([]byte(nil), certificado.Raw...))
		if err != nil {
			clon = append(clon, nil)
			continue
		}
		clon = append(clon, parseado)
	}
	return clon
}

func resumirCertificadoServidor(
	certificado tls.Certificate,
) ([sha256.Size]byte, [sha256.Size]byte, [sha256.Size]byte, error) {
	var vacia [sha256.Size]byte
	if len(certificado.Certificate) == 0 {
		return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
	}
	certificadosParseados := make([]*x509.Certificate, len(certificado.Certificate))
	for indice, der := range certificado.Certificate {
		parseado, err := x509.ParseCertificate(der)
		if err != nil {
			return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
		}
		certificadosParseados[indice] = parseado
	}
	if certificado.Leaf != nil && !certificado.Leaf.Equal(certificadosParseados[0]) {
		return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
	}
	firmante, valido := certificado.PrivateKey.(crypto.Signer)
	if !valido || firmante == nil {
		return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
	}
	clavePublica, err := x509.MarshalPKIXPublicKey(firmante.Public())
	if err != nil || !bytes.Equal(clavePublica, certificadosParseados[0].RawSubjectPublicKeyInfo) {
		return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
	}
	clavePrivada, err := x509.MarshalPKCS8PrivateKey(certificado.PrivateKey)
	if err != nil {
		return vacia, vacia, vacia, ErrTLSMutuoNoVerificado
	}

	hashCadena := sha256.New()
	var longitud [8]byte
	for _, der := range certificado.Certificate {
		binary.BigEndian.PutUint64(longitud[:], uint64(len(der)))
		_, _ = hashCadena.Write(longitud[:])
		_, _ = hashCadena.Write(der)
	}
	var huellaCadena [sha256.Size]byte
	copy(huellaCadena[:], hashCadena.Sum(nil))
	return huellaCadena, sha256.Sum256(clavePublica), sha256.Sum256(clavePrivada), nil
}

func certificadoTLSEquivalente(actual, aprobado tls.Certificate) bool {
	if !bytesBidimensionalesIguales(actual.Certificate, aprobado.Certificate) ||
		!slices.Equal(actual.SupportedSignatureAlgorithms, aprobado.SupportedSignatureAlgorithms) ||
		!bytes.Equal(actual.OCSPStaple, aprobado.OCSPStaple) ||
		!bytesBidimensionalesIguales(
			actual.SignedCertificateTimestamps, aprobado.SignedCertificateTimestamps,
		) || (actual.Leaf == nil) != (aprobado.Leaf == nil) {
		return false
	}
	return actual.Leaf == nil || actual.Leaf.Equal(aprobado.Leaf)
}

func bytesBidimensionalesIguales(izquierda, derecha [][]byte) bool {
	if len(izquierda) != len(derecha) {
		return false
	}
	for indice := range izquierda {
		if !bytes.Equal(izquierda[indice], derecha[indice]) {
			return false
		}
	}
	return true
}

func obtenerProtocolosHTTP(protocolos *http.Protocols) protocolosHTTPAprobados {
	if protocolos == nil {
		return protocolosHTTPAprobados{}
	}
	return protocolosHTTPAprobados{
		httpUno:          protocolos.HTTP1(),
		httpDos:          protocolos.HTTP2(),
		httpDosSinCifrar: protocolos.UnencryptedHTTP2(),
	}
}
