package postgres

import (
	"crypto/tls"
	"crypto/x509"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func configuracionPoolAcreditacionO405Valida(
	configuracion *pgxpool.Config,
	modo modoTLSAcreditacionPoolO405,
) bool {
	if configuracion == nil || configuracion.ConnConfig == nil ||
		configuracion.BeforeConnect != nil ||
		configuracion.AfterConnect != nil ||
		configuracion.BeforeAcquire != nil ||
		configuracion.PrepareConn != nil ||
		configuracion.AfterRelease != nil ||
		configuracion.BeforeClose != nil ||
		configuracion.ShouldPing != nil ||
		configuracion.ConnConfig.OnNotification != nil ||
		!onPgErrorAcreditacionO405PredeterminadoONulo(
			configuracion.ConnConfig.OnPgError,
		) {
		return false
	}
	return configuracionConexionAcreditacionO405Valida(
		configuracion.ConnConfig,
		modo,
	)
}

func configuracionConexionAcreditacionO405Valida(
	configuracion *pgx.ConnConfig,
	modo modoTLSAcreditacionPoolO405,
) bool {
	if configuracion == nil || configuracion.Tracer != nil ||
		configuracion.DialFunc == nil ||
		configuracion.LookupFunc == nil ||
		configuracion.BuildFrontend == nil ||
		configuracion.BuildContextWatcherHandler == nil ||
		configuracion.AfterNetConnect != nil ||
		configuracion.ValidateConnect != nil ||
		configuracion.AfterConnect != nil ||
		configuracion.OnNotice != nil ||
		configuracion.OAuthTokenProvider != nil ||
		len(configuracion.RuntimeParams) != 0 ||
		!onPgErrorAcreditacionO405PredeterminadoONulo(
			configuracion.OnPgError,
		) {
		return false
	}
	return configuracionTLSAcreditacionPoolO405Valida(
		&configuracion.Config,
		modo,
	)
}

func configuracionTLSAcreditacionPoolO405Valida(
	configuracion *pgconn.Config,
	modo modoTLSAcreditacionPoolO405,
) bool {
	if configuracion == nil {
		return false
	}
	switch modo {
	case modoTLSAcreditacionPoolO405Produccion:
		if !tlsAcreditacionPoolO405VerificaIdentidad(
			configuracion.TLSConfig,
			configuracion.Host,
		) {
			return false
		}
		for _, alternativa := range configuracion.Fallbacks {
			if alternativa == nil ||
				!tlsAcreditacionPoolO405VerificaIdentidad(
					alternativa.TLSConfig,
					alternativa.Host,
				) {
				return false
			}
		}
		return true
	case modoTLSAcreditacionPoolO405SocketUnixPrueba:
		if configuracion.TLSConfig != nil ||
			!destinoSocketUnixAcreditacionPoolO405(
				configuracion.Host,
				configuracion.Port,
			) {
			return false
		}
		for _, alternativa := range configuracion.Fallbacks {
			if alternativa == nil || alternativa.TLSConfig != nil ||
				!destinoSocketUnixAcreditacionPoolO405(
					alternativa.Host,
					alternativa.Port,
				) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func destinoSocketUnixAcreditacionPoolO405(host string, puerto uint16) bool {
	red, _ := pgconn.NetworkAddress(strings.TrimSpace(host), puerto)
	return red == "unix"
}

func tlsAcreditacionPoolO405VerificaIdentidad(
	configuracion *tls.Config,
	host string,
) bool {
	if configuracion == nil || configuracion.InsecureSkipVerify ||
		strings.TrimSpace(configuracion.ServerName) == "" ||
		!strings.EqualFold(
			strings.TrimSpace(configuracion.ServerName),
			strings.TrimSpace(host),
		) ||
		versionesTLSAcreditacionPoolO405Inseguras(configuracion) ||
		suitesTLSAcreditacionPoolO405Inseguras(configuracion) ||
		callbacksTLSAcreditacionPoolO405Inseguros(configuracion) {
		return false
	}
	return configuracion.RootCAs == nil ||
		poolCertificadosAcreditacionPoolO405NoVacio(configuracion.RootCAs)
}

func versionesTLSAcreditacionPoolO405Inseguras(
	configuracion *tls.Config,
) bool {
	if configuracion == nil ||
		configuracion.MinVersion < tls.VersionTLS12 ||
		(configuracion.MaxVersion != 0 &&
			configuracion.MaxVersion < tls.VersionTLS12) {
		return true
	}
	return configuracion.MinVersion != 0 &&
		configuracion.MaxVersion != 0 &&
		configuracion.MinVersion > configuracion.MaxVersion
}

func callbacksTLSAcreditacionPoolO405Inseguros(
	configuracion *tls.Config,
) bool {
	return configuracion == nil ||
		configuracion.KeyLogWriter != nil ||
		configuracion.Rand != nil ||
		configuracion.Time != nil ||
		configuracion.Renegotiation != tls.RenegotiateNever ||
		configuracion.VerifyPeerCertificate != nil ||
		configuracion.VerifyConnection != nil ||
		configuracion.GetClientCertificate != nil ||
		configuracion.ClientSessionCache != nil ||
		configuracion.EncryptedClientHelloRejectionVerify != nil
}

func suitesTLSAcreditacionPoolO405Inseguras(
	configuracion *tls.Config,
) bool {
	if configuracion == nil {
		return true
	}
	if len(configuracion.CipherSuites) == 0 {
		return configuracion.MinVersion < tls.VersionTLS13
	}
	seguras := make(map[uint16]struct{}, 6)
	for _, identificador := range cipherSuitesTLS12AcreditacionPoolO405() {
		seguras[identificador] = struct{}{}
	}
	for _, identificador := range configuracion.CipherSuites {
		if _, segura := seguras[identificador]; !segura {
			return true
		}
	}
	return false
}

func cipherSuitesTLS12AcreditacionPoolO405() []uint16 {
	return []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	}
}

var punteroOnPgErrorPredeterminadoAcreditacionO405 = func() uintptr {
	configuracion, err := pgconn.ParseConfig(
		"host=localhost user=vec_o405 dbname=postgres " +
			"password='' sslmode=disable",
	)
	if err != nil || configuracion == nil ||
		configuracion.OnPgError == nil {
		return 0
	}
	return reflect.ValueOf(configuracion.OnPgError).Pointer()
}()

func onPgErrorAcreditacionO405PredeterminadoONulo(
	callback pgconn.PgErrorHandler,
) bool {
	if callback == nil {
		return true
	}
	return punteroOnPgErrorPredeterminadoAcreditacionO405 != 0 &&
		reflect.ValueOf(callback).Pointer() ==
			punteroOnPgErrorPredeterminadoAcreditacionO405
}

func poolCertificadosAcreditacionPoolO405NoVacio(
	pool *x509.CertPool,
) bool {
	return pool != nil && len(pool.Subjects()) > 0
}
