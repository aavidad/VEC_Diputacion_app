package postgrespublico

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPrepararConfiguracionPoolExigeVerifyFull(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		modo   string
		valido bool
	}{
		{nombre: "verify-full", modo: "verify-full", valido: true},
		{nombre: "verify-ca", modo: "verify-ca"},
		{nombre: "require", modo: "require"},
		{nombre: "prefer", modo: "prefer"},
		{nombre: "disable", modo: "disable"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			configuracion, err := prepararConfiguracionPool(
				"postgres://lector:secreto-no-visible@db-publica.example/vec?sslmode=" + caso.modo,
			)
			if caso.valido {
				if err != nil || configuracion == nil ||
					configuracion.ConnConfig.RuntimeParams["default_transaction_read_only"] != "on" {
					t.Fatalf("verify-full rechazado: %v", err)
				}
				return
			}
			if configuracion != nil || !errors.Is(err, ErrTLSPostgreSQLPublicaInseguro) {
				t.Fatalf("modo %s aceptado: configuracion=%v error=%v", caso.modo, configuracion, err)
			}
			if strings.Contains(err.Error(), "secreto-no-visible") || strings.Contains(err.Error(), "db-publica.example") {
				t.Fatalf("error filtro el DSN: %v", err)
			}
		})
	}
}

func TestValidarTLSPostgreSQLPublicoRevisaTodosLosFallbacksYVersiones(t *testing.T) {
	raices := x509.NewCertPool()
	raices.AddCert(certificadoPruebaTLS(t))
	segura := func(nombre string) *tls.Config {
		return &tls.Config{ServerName: nombre, RootCAs: raices, MinVersion: tls.VersionTLS12}
	}
	pruebas := []struct {
		nombre        string
		configuracion *pgconn.Config
		valida        bool
	}{
		{"principal y fallback seguros", &pgconn.Config{
			Host: "db.example", TLSConfig: segura("db.example"),
			Fallbacks: []*pgconn.FallbackConfig{{Host: "replica.example", TLSConfig: segura("replica.example")}},
		}, true},
		{"fallback sin TLS", &pgconn.Config{
			Host: "db.example", TLSConfig: segura("db.example"),
			Fallbacks: []*pgconn.FallbackConfig{{Host: "replica.example"}},
		}, false},
		{"nombre distinto", &pgconn.Config{Host: "db.example", TLSConfig: segura("otro.example")}, false},
		{"TLS obsoleto", &pgconn.Config{Host: "db.example", TLSConfig: &tls.Config{
			ServerName: "db.example", RootCAs: raices, MinVersion: tls.VersionTLS10,
		}}, false},
		{"pool explicito vacio", &pgconn.Config{Host: "db.example", TLSConfig: &tls.Config{
			ServerName: "db.example", RootCAs: x509.NewCertPool(), MinVersion: tls.VersionTLS12,
		}}, false},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			err := validarTLSPostgreSQLPublico(prueba.configuracion)
			if prueba.valida && err != nil || !prueba.valida && !errors.Is(err, ErrTLSPostgreSQLPublicaInseguro) {
				t.Fatalf("validacion TLS = %v", err)
			}
		})
	}
}

func TestPrepararConfiguracionPoolRechazaParametrosDeSesionDelDSN(t *testing.T) {
	for _, consulta := range []string{
		"options=-c%20statement_timeout%3D0%20-c%20default_transaction_read_only%3Doff",
		"application_name=aplicacion-ajena",
		"search_path=public",
	} {
		configuracion, err := prepararConfiguracionPool(
			"postgres://lector:secreto-no-visible@db-publica.example/vec?sslmode=verify-full&" + consulta,
		)
		if configuracion != nil || !errors.Is(err, ErrConfiguracionPostgreSQLPublicaInvalida) {
			t.Fatalf("parametro de sesion admitido (%s): configuracion=%v error=%v", consulta, configuracion, err)
		}
		if strings.Contains(err.Error(), "secreto-no-visible") {
			t.Fatalf("el error filtro el DSN: %v", err)
		}
	}
}

func TestNuevaFuenteRechazaTestigoDeInvalidacionComoManifiesto(t *testing.T) {
	fuente, err := NuevaFuente(
		&pgxpool.Pool{}, "categorias-profesionales", 1,
		strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("0", 64),
	)
	if fuente != nil || !errors.Is(err, ErrConfiguracionPostgreSQLPublicaInvalida) {
		t.Fatalf("testigo reservado aceptado: fuente=%v error=%v", fuente, err)
	}
}

func certificadoPruebaTLS(t *testing.T) *x509.Certificate {
	t.Helper()
	// Un certificado parseable basta para acreditar que el pool no esta vacio;
	// crypto/tls realizara la verificacion criptografica real al conectar.
	certificado, err := x509.ParseCertificate([]byte{})
	if err == nil {
		return certificado
	}
	return &x509.Certificate{RawSubject: []byte{1}, Raw: []byte{1}}
}
