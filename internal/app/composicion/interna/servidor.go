package interna

import (
	"bytes"
	"crypto"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
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
}

type materialTLSAprobado struct {
	autoridadesClientes        *x509.CertPool
	certificadoServidor        tls.Certificate
	huellaCadenaServidor       [sha256.Size]byte
	huellaClavePublicaServidor [sha256.Size]byte
}

func (m *manejadorInternoVerificado) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.siguiente.ServeHTTP(w, r)
}

// construirServidorInterno es el unico puente futuro entre C5/C6 y net/http.
// Permanece no exportado y C4 no lo invoca: solo acepta una API no nula y una
// configuracion mTLS ya materializada, y siempre delega la allowlist a
// server.NewHTTPServerInterno.
func construirServidorInterno(
	cfg Configuracion,
	api http.Handler,
	tlsMutuo *tls.Config,
) (*http.Server, error) {
	if err := cfg.Validar(); err != nil {
		return nil, err
	}
	if manejadorNulo(api) || esMuxPredeterminado(api) {
		return nil, ErrAPIInternaNoDisponible
	}
	if err := validarTLSMutuo(tlsMutuo); err != nil {
		return nil, err
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
	configuracionTLS := clonarConfiguracionTLSMutuo(tlsMutuo)
	materialTLS, err := aprobarMaterialTLS(configuracionTLS)
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
		servidorHTTP.MaxHeaderBytes != manejador.maximoBytesCabeceras {
		return ErrServidorInternoInvalido
	}
	if !manejador.materialTLS.coincide(servidorHTTP.TLSConfig) {
		return ErrTLSMutuoNoVerificado
	}
	return nil
}

func validarTLSMutuo(configuracion *tls.Config) error {
	if configuracion == nil || configuracion.MinVersion != tls.VersionTLS13 ||
		configuracion.MaxVersion != tls.VersionTLS13 ||
		configuracion.ClientAuth != tls.RequireAndVerifyClientCert ||
		!poolCertificadosConAutoridades(configuracion.ClientCAs) ||
		len(configuracion.Certificates) != 1 ||
		configuracion.GetConfigForClient != nil || configuracion.GetCertificate != nil ||
		configuracion.NameToCertificate != nil || configuracion.InsecureSkipVerify ||
		configuracion.Renegotiation != tls.RenegotiateNever ||
		configuracion.GetClientCertificate != nil {
		return ErrTLSMutuoNoVerificado
	}
	certificado := configuracion.Certificates[0]
	if len(certificado.Certificate) == 0 || certificado.PrivateKey == nil {
		return ErrTLSMutuoNoVerificado
	}
	if _, _, err := resumirCertificadoServidor(certificado); err != nil {
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

// clonarConfiguracionTLSMutuo evita que el llamador degrade indirectamente la
// politica ya validada mutando slices o el pool de autoridades compartidos.
// La clave privada se conserva como crypto.PrivateKey opaca, igual que hace
// crypto/tls; su proveedor debe tratarla como inmutable.
func clonarConfiguracionTLSMutuo(origen *tls.Config) *tls.Config {
	clon := origen.Clone()
	clon.ClientCAs = origen.ClientCAs.Clone()
	clon.Certificates = make([]tls.Certificate, len(origen.Certificates))
	for indice := range origen.Certificates {
		clon.Certificates[indice] = clonarCertificadoTLS(origen.Certificates[indice])
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

func aprobarMaterialTLS(configuracion *tls.Config) (materialTLSAprobado, error) {
	huellaCadena, huellaClave, err := resumirCertificadoServidor(configuracion.Certificates[0])
	if err != nil {
		return materialTLSAprobado{}, err
	}
	certificado := clonarCertificadoTLS(configuracion.Certificates[0])
	// La aprobacion conserva material publico y metadatos, no duplica la clave.
	certificado.PrivateKey = nil
	return materialTLSAprobado{
		autoridadesClientes:        configuracion.ClientCAs.Clone(),
		certificadoServidor:        certificado,
		huellaCadenaServidor:       huellaCadena,
		huellaClavePublicaServidor: huellaClave,
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
	huellaCadena, huellaClave, err := resumirCertificadoServidor(actual)
	if err != nil || huellaCadena != aprobado.huellaCadenaServidor ||
		huellaClave != aprobado.huellaClavePublicaServidor {
		return false
	}
	return certificadoTLSEquivalente(actual, aprobado.certificadoServidor)
}

func resumirCertificadoServidor(
	certificado tls.Certificate,
) ([sha256.Size]byte, [sha256.Size]byte, error) {
	var vacia [sha256.Size]byte
	if len(certificado.Certificate) == 0 {
		return vacia, vacia, ErrTLSMutuoNoVerificado
	}
	certificadosParseados := make([]*x509.Certificate, len(certificado.Certificate))
	for indice, der := range certificado.Certificate {
		parseado, err := x509.ParseCertificate(der)
		if err != nil {
			return vacia, vacia, ErrTLSMutuoNoVerificado
		}
		certificadosParseados[indice] = parseado
	}
	if certificado.Leaf != nil && !certificado.Leaf.Equal(certificadosParseados[0]) {
		return vacia, vacia, ErrTLSMutuoNoVerificado
	}
	firmante, valido := certificado.PrivateKey.(crypto.Signer)
	if !valido || firmante == nil {
		return vacia, vacia, ErrTLSMutuoNoVerificado
	}
	clavePublica, err := x509.MarshalPKIXPublicKey(firmante.Public())
	if err != nil || !bytes.Equal(clavePublica, certificadosParseados[0].RawSubjectPublicKeyInfo) {
		return vacia, vacia, ErrTLSMutuoNoVerificado
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
	return huellaCadena, sha256.Sum256(clavePublica), nil
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
