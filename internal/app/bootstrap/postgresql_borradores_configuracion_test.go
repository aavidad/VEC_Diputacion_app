package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPrepararConfiguracionPoolPostgreSQLBorradoresImponeLimites(t *testing.T) {
	dsn := "postgres://cuenta:secreto-no-visible@127.0.0.1:5432/vec" +
		"?sslmode=disable&pool_max_conns=99&pool_min_conns=50" +
		"&pool_max_conn_lifetime=99h&connect_timeout=99" +
		"&application_name=intruso&statement_timeout=99min" +
		"&default_transaction_read_only=off"
	for _, perfil := range perfilesPoolPostgreSQLBorradores {
		configuracion, err := prepararConfiguracionPoolPostgreSQLBorradores(dsn, perfil)
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
	)
	if !errors.Is(err, ErrConfiguracionPoolPostgreSQLBorradoresInvalida) {
		t.Fatalf("error no cerrado: %v", err)
	}
	if strings.Contains(err.Error(), "secreto-super-sensible") {
		t.Fatalf("el error filtro el DSN: %v", err)
	}
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
