package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/config"
)

func TestPrepararConfiguracionPoolPostgreSQLBorradoresImponeLimites(t *testing.T) {
	dsn := "postgres://cuenta:secreto-no-visible@127.0.0.1:5432/vec" +
		"?sslmode=disable&pool_max_conns=99&pool_min_conns=50" +
		"&pool_max_conn_lifetime=99h&connect_timeout=99" +
		"&application_name=intruso&statement_timeout=99min" +
		"&default_transaction_read_only=off"
	for _, perfil := range perfilesPoolPostgreSQLBorradores {
		configuracion, err := prepararConfiguracionPoolPostgreSQLBorradores(
			dsn, perfil, politicaTLSPostgreSQLBorradoresDesarrolloValidado,
		)
		if err != nil {
			t.Fatalf("perfil %s: %v", perfil.rolEsperado, err)
		}
		if configuracion.MaxConns != perfil.maxConexiones || configuracion.MinConns != 0 ||
			configuracion.MinIdleConns != 0 ||
			configuracion.MaxConnLifetime != duracionVidaPostgreSQLBorradores ||
			configuracion.MaxConnLifetimeJitter != duracionJitterPostgreSQLBorradores ||
			configuracion.MaxConnIdleTime != duracionInactividadPostgreSQLBorradores ||
			configuracion.HealthCheckPeriod != periodoSaludPostgreSQLBorradores ||
			configuracion.PingTimeout != duracionSondaPostgreSQLBorradores ||
			configuracion.ConnConfig.ConnectTimeout != duracionConexionPostgreSQLBorradores {
			t.Fatalf("limites no endurecidos para %s", perfil.rolEsperado)
		}
		parametros := configuracion.ConnConfig.RuntimeParams
		esperados := map[string]string{
			"application_name":                    perfil.aplicacion,
			"timezone":                            "UTC",
			"search_path":                         "pg_catalog,pg_temp",
			"default_transaction_isolation":       "serializable",
			"statement_timeout":                   "15s",
			"lock_timeout":                        "3s",
			"idle_in_transaction_session_timeout": "15s",
			"default_transaction_read_only":       map[bool]string{true: "on", false: "off"}[perfil.soloLectura],
		}
		for clave, esperado := range esperados {
			if parametros[clave] != esperado {
				t.Fatalf("%s de %s = %q, esperado %q", clave, perfil.rolEsperado, parametros[clave], esperado)
			}
		}
		if configuracion.AfterConnect == nil {
			t.Fatalf("%s no revalida conexiones nuevas", perfil.rolEsperado)
		}
	}
}

func TestEjecutorConsultaPostgreSQLBorradoresAdmiteAuditoriaGobernada(t *testing.T) {
	configuracion, err := prepararConfiguracionPoolPostgreSQLBorradores(
		"postgres://ejecutor:secreto@127.0.0.1:5432/vec?sslmode=disable",
		perfilesPoolPostgreSQLBorradores[0],
		politicaTLSPostgreSQLBorradoresDesarrolloValidado,
	)
	if err != nil {
		t.Fatal(err)
	}
	// Las funciones listar_borradores_v1 y obtener_borrador_v1 son VOLATILE:
	// escriben consumo, auditoria y cursor aunque la operacion sea una lectura
	// funcional. Forzar read-only impediria la vertical real.
	if obtenido := configuracion.ConnConfig.RuntimeParams["default_transaction_read_only"]; obtenido != "off" {
		t.Fatalf("ejecutor de consulta configurado como read-only: %q", obtenido)
	}
}

func TestPrepararConfiguracionPoolPostgreSQLBorradoresNoFiltraDSNInvalido(t *testing.T) {
	_, err := prepararConfiguracionPoolPostgreSQLBorradores(
		"postgres://cuenta:secreto-super-sensible@[::1/vec",
		perfilesPoolPostgreSQLBorradores[0],
		politicaTLSPostgreSQLBorradoresProduccion,
	)
	if !errors.Is(err, ErrConfiguracionPoolPostgreSQLBorradoresInvalida) {
		t.Fatalf("error no cerrado: %v", err)
	}
	if strings.Contains(err.Error(), "secreto-super-sensible") {
		t.Fatalf("el error filtro el DSN: %v", err)
	}
}

func TestPrepararConfiguracionPoolPostgreSQLBorradoresExigeVerifyFullEnProduccion(t *testing.T) {
	casos := []struct {
		nombre string
		modo   string
		valido bool
	}{
		{nombre: "modo ausente equivale a prefer"},
		{nombre: "disable", modo: "disable"},
		{nombre: "allow", modo: "allow"},
		{nombre: "prefer", modo: "prefer"},
		{nombre: "require sin CA", modo: "require"},
		{nombre: "verify-ca", modo: "verify-ca"},
		{nombre: "verify-full", modo: "verify-full", valido: true},
		{nombre: "verify-full multihost", modo: "verify-full", valido: true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			dsn := "postgres://cuenta:password-no-visible@bd-no-visible.example/vec"
			if caso.nombre == "verify-full multihost" {
				dsn = "postgres://cuenta:password-no-visible@bd-uno.example,bd-dos.example/vec"
			}
			if caso.modo != "" {
				dsn += "?sslmode=" + caso.modo
			}
			configuracion, err := prepararConfiguracionPoolPostgreSQLBorradores(
				dsn, perfilesPoolPostgreSQLBorradores[0], politicaTLSPostgreSQLBorradoresProduccion,
			)
			if caso.valido {
				if err != nil || configuracion == nil {
					t.Fatalf("modo seguro rechazado: %v", err)
				}
				if caso.nombre == "verify-full multihost" && len(configuracion.ConnConfig.Fallbacks) != 1 {
					t.Fatalf("fallbacks multihost=%d", len(configuracion.ConnConfig.Fallbacks))
				}
				return
			}
			if configuracion != nil || !errors.Is(err, ErrTLSPostgreSQLBorradoresInseguro) {
				t.Fatalf("modo inseguro aceptado: configuracion_nula=%t error=%v", configuracion == nil, err)
			}
			for _, sensible := range []string{"password-no-visible", "bd-no-visible.example"} {
				if strings.Contains(err.Error(), sensible) {
					t.Fatalf("error TLS filtro %q: %v", sensible, err)
				}
			}
		})
	}
}

func TestValidarTLSPostgreSQLBorradoresRevisaConfiguracionEfectivaYFallbacks(t *testing.T) {
	raices := poolRaicesPostgreSQLBorradoresPrueba(t)
	segura := func(nombre string) *tls.Config {
		return &tls.Config{ServerName: nombre, RootCAs: raices}
	}
	casos := []struct {
		nombre        string
		configuracion *pgconn.Config
		valida        bool
	}{
		{nombre: "configuracion nil"},
		{nombre: "TLS nil", configuracion: &pgconn.Config{}},
		{
			nombre: "InsecureSkipVerify", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: &tls.Config{
					ServerName: "db.example", RootCAs: raices, InsecureSkipVerify: true,
				},
			},
		},
		{
			nombre: "ServerName vacio", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: &tls.Config{RootCAs: raices},
			},
		},
		{
			nombre: "ServerName no corresponde al host", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: segura("otro.example"),
			},
		},
		{
			nombre: "RootCAs explicitas vacias", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: &tls.Config{
					ServerName: "db.example", RootCAs: x509.NewCertPool(),
				},
			},
		},
		{
			nombre: "minimo TLS obsoleto", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: &tls.Config{
					ServerName: "db.example", RootCAs: raices, MinVersion: tls.VersionTLS10,
				},
			},
		},
		{
			nombre: "maximo TLS obsoleto", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: &tls.Config{
					ServerName: "db.example", RootCAs: raices, MaxVersion: tls.VersionTLS11,
				},
			},
		},
		{
			nombre: "intervalo TLS imposible", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: &tls.Config{
					ServerName: "db.example", RootCAs: raices,
					MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS12,
				},
			},
		},
		{
			nombre: "fallback nil", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: segura("db.example"),
				Fallbacks: []*pgconn.FallbackConfig{nil},
			},
		},
		{
			nombre: "fallback sin TLS", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: segura("db.example"),
				Fallbacks: []*pgconn.FallbackConfig{{Host: "replica.example"}},
			},
		},
		{
			nombre: "fallback insecure", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: segura("db.example"),
				Fallbacks: []*pgconn.FallbackConfig{{Host: "replica.example", TLSConfig: &tls.Config{
					ServerName: "replica.example", RootCAs: raices, InsecureSkipVerify: true,
				}}},
			},
		},
		{
			nombre: "fallback sin ServerName", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: segura("db.example"),
				Fallbacks: []*pgconn.FallbackConfig{{
					Host: "replica.example", TLSConfig: &tls.Config{RootCAs: raices},
				}},
			},
		},
		{
			nombre: "fallback ServerName no corresponde al host", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: segura("db.example"),
				Fallbacks: []*pgconn.FallbackConfig{{
					Host: "replica.example", TLSConfig: segura("otro.example"),
				}},
			},
		},
		{
			nombre: "fallback RootCAs vacias", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: segura("db.example"),
				Fallbacks: []*pgconn.FallbackConfig{{Host: "replica.example", TLSConfig: &tls.Config{
					ServerName: "replica.example", RootCAs: x509.NewCertPool(),
				}}},
			},
		},
		{
			nombre: "almacen del sistema", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: &tls.Config{ServerName: "db.example"},
			}, valida: true,
		},
		{
			nombre: "TLS 1.2 explicito", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: &tls.Config{
					ServerName: "db.example", RootCAs: raices,
					MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12,
				},
			}, valida: true,
		},
		{
			nombre: "primaria y fallback seguros", configuracion: &pgconn.Config{
				Host: "db.example", TLSConfig: segura("db.example"),
				Fallbacks: []*pgconn.FallbackConfig{{
					Host: "replica.example", TLSConfig: segura("replica.example"),
				}},
			}, valida: true,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			err := validarTLSPostgreSQLBorradores(caso.configuracion, false)
			if caso.valida && err != nil {
				t.Fatalf("configuracion segura rechazada: %v", err)
			}
			if !caso.valida && !errors.Is(err, ErrTLSPostgreSQLBorradoresInseguro) {
				t.Fatalf("configuracion insegura aceptada: %v", err)
			}
		})
	}
}

func TestValidarTLSPostgreSQLBorradoresSoloAdmitePlaintextDeDesarrolloLocal(t *testing.T) {
	casos := []struct {
		nombre        string
		configuracion *pgconn.Config
		valida        bool
	}{
		{
			nombre:        "IPv4 loopback",
			configuracion: &pgconn.Config{Host: "127.0.0.1", Port: 5432},
			valida:        true,
		},
		{
			nombre:        "IPv6 loopback",
			configuracion: &pgconn.Config{Host: "::1", Port: 5432},
			valida:        true,
		},
		{
			nombre:        "socket Unix",
			configuracion: &pgconn.Config{Host: "/var/run/postgresql", Port: 5432},
			valida:        true,
		},
		{
			nombre:        "host remoto",
			configuracion: &pgconn.Config{Host: "db.example", Port: 5432},
		},
		{
			nombre: "fallback remoto",
			configuracion: &pgconn.Config{
				Host: "127.0.0.1", Port: 5432,
				Fallbacks: []*pgconn.FallbackConfig{{Host: "db.example", Port: 5432}},
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			err := validarTLSPostgreSQLBorradores(caso.configuracion, true)
			if caso.valida && err != nil {
				t.Fatalf("plaintext local rechazado: %v", err)
			}
			if !caso.valida && !errors.Is(err, ErrTLSPostgreSQLBorradoresInseguro) {
				t.Fatalf("plaintext no local aceptado: %v", err)
			}
		})
	}
}

func TestPrepararConfiguracionPoolPostgreSQLBorradoresExcepcionSinTLSSoloDesarrolloCerrado(t *testing.T) {
	dsn := "postgres://cuenta:password-no-visible@127.0.0.1/vec?sslmode=disable"
	desarrolloValidado, _ := generarMaterialDesarrolloPrueba(t)
	produccionConSelectores := desarrolloValidado
	produccionConSelectores.ExecutionProfile = config.ExecutionProfileProduction
	casos := []struct {
		nombre        string
		cfg           config.Config
		valido        bool
		errorPolitica error
	}{
		{nombre: "produccion", cfg: config.Config{ExecutionProfile: config.ExecutionProfileProduction}},
		{nombre: "produccion no se convierte con selectores locales", cfg: produccionConSelectores},
		{
			nombre: "solo perfil", cfg: config.Config{
				ExecutionProfile: config.ExecutionProfileDevelopment,
			},
		},
		{
			nombre: "perfil y auth", cfg: config.Config{
				ExecutionProfile: config.ExecutionProfileDevelopment,
				AuthMode:         config.AuthModeDevelopment,
			},
		},
		{
			nombre: "guarda incorrecta", cfg: config.Config{
				ExecutionProfile: config.ExecutionProfileDevelopment,
				AuthMode:         config.AuthModeDevelopment,
				DevelopmentGuard: "SI",
			},
		},
		{
			nombre: "doble llave sin material T21",
			cfg: config.Config{
				ExecutionProfile: config.ExecutionProfileDevelopment,
				AuthMode:         config.AuthModeDevelopment,
				DevelopmentGuard: config.DevelopmentGuardAcknowledgement,
			},
			errorPolitica: ErrActivacionDesarrolloInvalida,
		},
		{nombre: "doble llave y material T21 validados", cfg: desarrolloValidado, valido: true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			politica, err := politicaTLSPostgreSQLBorradoresDesdeConfiguracion(caso.cfg)
			if caso.errorPolitica != nil {
				if !errors.Is(err, caso.errorPolitica) {
					t.Fatalf("error de politica=%v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolver politica TLS: %v", err)
			}
			configuracion, err := prepararConfiguracionPoolPostgreSQLBorradores(
				dsn, perfilesPoolPostgreSQLBorradores[0], politica,
			)
			if caso.valido {
				if err != nil || configuracion == nil {
					t.Fatalf("desarrollo cerrado rechazado: %v", err)
				}
				return
			}
			if configuracion != nil || !errors.Is(err, ErrTLSPostgreSQLBorradoresInseguro) {
				t.Fatalf("excepcion de desarrollo abierta: configuracion_nula=%t error=%v", configuracion == nil, err)
			}
		})
	}
}

func TestPrepararConfiguracionPoolPostgreSQLBorradoresRequireConRootCert(t *testing.T) {
	rutaCA := t.TempDir() + "/ca.crt"
	if err := os.WriteFile(rutaCA, certificadoRaizPostgreSQLBorradoresPrueba(t), 0o600); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		dsn    string
		valido bool
	}{
		{
			nombre: "require con CA de fichero solo verifica cadena",
			dsn:    "postgres://cuenta:secreto@db.example/vec?sslmode=require&sslrootcert=" + rutaCA,
		},
		{
			nombre: "verify-full con CA de fichero",
			dsn:    "postgres://cuenta:secreto@db.example/vec?sslmode=verify-full&sslrootcert=" + rutaCA,
			valido: true,
		},
		{
			nombre: "require con raices del sistema se eleva a verify-full",
			dsn:    "postgres://cuenta:secreto@db.example/vec?sslmode=require&sslrootcert=system",
			valido: true,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			configuracion, err := prepararConfiguracionPoolPostgreSQLBorradores(
				caso.dsn, perfilesPoolPostgreSQLBorradores[0], politicaTLSPostgreSQLBorradoresProduccion,
			)
			if !caso.valido {
				if configuracion != nil || !errors.Is(err, ErrTLSPostgreSQLBorradoresInseguro) {
					t.Fatalf("require sin verificacion de nombre aceptado: configuracion_nula=%t error=%v", configuracion == nil, err)
				}
				return
			}
			if err != nil || configuracion == nil || configuracion.ConnConfig.TLSConfig == nil ||
				configuracion.ConnConfig.TLSConfig.InsecureSkipVerify ||
				configuracion.ConnConfig.TLSConfig.ServerName != "db.example" {
				t.Fatalf("verify-full efectivo rechazado: configuracion_nula=%t error=%v", configuracion == nil, err)
			}
		})
	}
}

func poolRaicesPostgreSQLBorradoresPrueba(t *testing.T) *x509.CertPool {
	t.Helper()
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certificadoRaizPostgreSQLBorradoresPrueba(t)) {
		t.Fatal("no se pudo construir el pool CA de prueba")
	}
	return pool
}

func certificadoRaizPostgreSQLBorradoresPrueba(t *testing.T) []byte {
	t.Helper()
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	plantilla := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "CA borradores prueba"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, publica, privada)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestComprobarIdentidadPoolPostgreSQLBorradoresFallaCerrado(t *testing.T) {
	casos := []struct {
		nombre          string
		fila            filaIdentidadPostgreSQLBorradoresPrueba
		rol             string
		usuarioEsperado string
		errorEsperado   error
	}{
		{
			nombre: "valida", rol: rolEjecutorConsultaPostgreSQLBorradores,
			fila: filaIdentidadPostgreSQLBorradoresPrueba{
				usuarioSesion: "login-ejecutor", usuarioEfectivo: "login-ejecutor", valida: true,
			},
			usuarioEsperado: "login-ejecutor",
		},
		{
			nombre: "set role", rol: rolEjecutorConsultaPostgreSQLBorradores,
			fila: filaIdentidadPostgreSQLBorradoresPrueba{
				usuarioSesion: "login-ejecutor", usuarioEfectivo: "rol-elevado", valida: true,
			},
			errorEsperado: ErrIdentidadPostgreSQLBorradoresInvalida,
		},
		{
			nombre: "membresia no exclusiva", rol: rolProyectorGobiernoPostgreSQLBorradores,
			fila: filaIdentidadPostgreSQLBorradoresPrueba{
				usuarioSesion: "login-proyector", usuarioEfectivo: "login-proyector",
			},
			errorEsperado: ErrIdentidadPostgreSQLBorradoresInvalida,
		},
		{
			nombre: "consulta falla", rol: rolVerificadorReciboPostgreSQLBorradores,
			fila:          filaIdentidadPostgreSQLBorradoresPrueba{err: errors.New("fallo secreto de PostgreSQL")},
			errorEsperado: ErrIdentidadPostgreSQLBorradoresInvalida,
		},
		{
			nombre: "rol no previsto", rol: "vec_propietario",
			fila:          filaIdentidadPostgreSQLBorradoresPrueba{valida: true},
			errorEsperado: ErrIdentidadPostgreSQLBorradoresInvalida,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			consultador := &consultadorIdentidadPostgreSQLBorradoresPrueba{fila: caso.fila}
			usuario, err := comprobarIdentidadPoolPostgreSQLBorradores(
				context.Background(), consultador, caso.rol,
			)
			if !errors.Is(err, caso.errorEsperado) || usuario != caso.usuarioEsperado {
				t.Fatalf("usuario=%q error=%v", usuario, err)
			}
			if err != nil && strings.Contains(err.Error(), "secreto") {
				t.Fatalf("detalle interno filtrado: %v", err)
			}
			if caso.errorEsperado == nil {
				if consultador.rolConsultado != caso.rol ||
					!strings.Contains(consultador.consulta, "pg_catalog.pg_auth_members") {
					t.Fatal("la sonda no uso el rol nominal y la consulta de membresia cerrada")
				}
			}
		})
	}
}

type filaIdentidadPostgreSQLBorradoresPrueba struct {
	usuarioSesion   string
	usuarioEfectivo string
	valida          bool
	err             error
}

func (f filaIdentidadPostgreSQLBorradoresPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*(destinos[0].(*string)) = f.usuarioSesion
	*(destinos[1].(*string)) = f.usuarioEfectivo
	*(destinos[2].(*bool)) = f.valida
	return nil
}

type consultadorIdentidadPostgreSQLBorradoresPrueba struct {
	fila          filaIdentidadPostgreSQLBorradoresPrueba
	consulta      string
	rolConsultado string
}

func (c *consultadorIdentidadPostgreSQLBorradoresPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	c.consulta = consulta
	if len(argumentos) == 1 {
		c.rolConsultado, _ = argumentos[0].(string)
	}
	return c.fila
}
