package postgres

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

type filaAcreditacionPoolO405Prueba struct {
	valores []any
	err     error
}

func (f filaAcreditacionPoolO405Prueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return errors.New("columnas inesperadas")
	}
	for indice, destino := range destinos {
		switch puntero := destino.(type) {
		case *string:
			valor, ok := f.valores[indice].(string)
			if !ok {
				return errors.New("texto inesperado")
			}
			*puntero = valor
		case *bool:
			valor, ok := f.valores[indice].(bool)
			if !ok {
				return errors.New("booleano inesperado")
			}
			*puntero = valor
		case *uint32:
			valor, ok := f.valores[indice].(uint32)
			if !ok {
				return errors.New("oid inesperado")
			}
			*puntero = valor
		default:
			return errors.New("destino inesperado")
		}
	}
	return nil
}

type conexionAcreditacionPoolO405Prueba struct {
	configuracion          *pgx.ConnConfig
	sello                  *selloFabricaPoolO405
	sinSello               bool
	fila                   pgx.Row
	consulta               string
	argumentos             []any
	liberaciones           int
	panico                 bool
	tx                     pgx.Tx
	errBegin               error
	inicios                int
	acreditadaAntesDeBegin bool
	panicoBegin            bool
	panicoLiberar          bool
}

var (
	selloProduccionAcreditacionO405Prueba = nuevoSelloAcreditacionO405Prueba(
		modoTLSAcreditacionPoolO405Produccion,
	)
	selloSocketAcreditacionO405Prueba = nuevoSelloAcreditacionO405Prueba(
		modoTLSAcreditacionPoolO405SocketUnixPrueba,
	)
)

func nuevoSelloAcreditacionO405Prueba(
	modo modoTLSAcreditacionPoolO405,
) *selloFabricaPoolO405 {
	dependencia := &PoolRecuperacionCoberturaO405PostgreSQL{}
	sello := &selloFabricaPoolO405{
		dependencia:              dependencia,
		modo:                     modo,
		callbacksPredeterminados: true,
	}
	dependencia.sello = sello
	return sello
}

func (c *conexionAcreditacionPoolO405Prueba) Configuracion() *pgx.ConnConfig {
	if c.configuracion != nil {
		return c.configuracion
	}
	return configuracionConexionAcreditacionPoolO405Prueba()
}

func (c *conexionAcreditacionPoolO405Prueba) Sello() *selloFabricaPoolO405 {
	if c == nil || c.sinSello {
		return nil
	}
	if c.sello != nil {
		return c.sello
	}
	if configuracionConexionPruebaEsSocketO405(c.Configuracion()) {
		return selloSocketAcreditacionO405Prueba
	}
	return selloProduccionAcreditacionO405Prueba
}

func (c *conexionAcreditacionPoolO405Prueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	if c.panico {
		panic("detalle privado de conexión")
	}
	c.consulta = consulta
	c.argumentos = append([]any(nil), argumentos...)
	return c.fila
}

func (c *conexionAcreditacionPoolO405Prueba) BeginTx(
	_ context.Context,
	_ pgx.TxOptions,
) (pgx.Tx, error) {
	if c.panicoBegin {
		panic("detalle privado al iniciar")
	}
	c.inicios++
	c.acreditadaAntesDeBegin = c.consulta != ""
	return c.tx, c.errBegin
}

func (c *conexionAcreditacionPoolO405Prueba) Liberar() {
	c.liberaciones++
	if c.panicoLiberar {
		panic("detalle privado al liberar")
	}
}

type origenAcreditacionPoolO405Prueba struct {
	configuracion *pgxpool.Config
	sello         *selloFabricaPoolO405
	sinSello      bool
	conexion      conexionAcreditacionPoolO405
	conexiones    []conexionAcreditacionPoolO405
	err           error
	adquisiciones int
}

func (o *origenAcreditacionPoolO405Prueba) Configuracion() *pgxpool.Config {
	return o.configuracion
}

func (o *origenAcreditacionPoolO405Prueba) Sello() *selloFabricaPoolO405 {
	if o == nil || o.sinSello {
		return nil
	}
	if o.sello != nil {
		return o.sello
	}
	if o.configuracion != nil &&
		configuracionConexionPruebaEsSocketO405(o.configuracion.ConnConfig) {
		return selloSocketAcreditacionO405Prueba
	}
	return selloProduccionAcreditacionO405Prueba
}

func (o *origenAcreditacionPoolO405Prueba) Adquirir(
	_ context.Context,
) (conexionAcreditacionPoolO405, error) {
	o.adquisiciones++
	if len(o.conexiones) >= o.adquisiciones {
		return o.conexiones[o.adquisiciones-1], o.err
	}
	return o.conexion, o.err
}

func valoresAcreditacionPoolO405Prueba(tlsActivo bool) []any {
	return []any{
		uint32(405), "vec_o405_lector", "vec_o405_lector", tlsActivo,
		true, true, true, true, true, true, true, true,
		true, true, true, true, true, true, true, true,
	}
}

func configuracionTLSAcreditacionPoolO405Prueba() *pgconn.Config {
	return &pgconn.Config{
		Host: "db-o405.example",
		TLSConfig: &tls.Config{
			ServerName: "db-o405.example",
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		},
	}
}

func configuracionConexionAcreditacionPoolO405Prueba() *pgx.ConnConfig {
	configuracion, err := pgx.ParseConfig(
		"host=db-o405.example user=vec_o405 dbname=postgres " +
			"password='' sslmode=disable",
	)
	if err != nil {
		panic(err)
	}
	tlsSegura := configuracionTLSAcreditacionPoolO405Prueba()
	configuracion.Host = tlsSegura.Host
	configuracion.TLSConfig = tlsSegura.TLSConfig
	configuracion.Fallbacks = tlsSegura.Fallbacks
	return configuracion
}

func configuracionPoolAcreditacionO405Prueba() *pgxpool.Config {
	return &pgxpool.Config{
		ConnConfig: configuracionConexionAcreditacionPoolO405Prueba(),
	}
}

func configuracionConexionPruebaEsSocketO405(
	configuracion *pgx.ConnConfig,
) bool {
	return configuracion != nil &&
		destinoSocketUnixAcreditacionPoolO405(
			configuracion.Host,
			configuracion.Port,
		) &&
		configuracion.TLSConfig == nil
}

func TestAcreditacionPoolO405ExigeManifiestoExactoEnUnaConexion(
	t *testing.T,
) {
	filaValida := valoresAcreditacionPoolO405Prueba(true)
	casos := []struct {
		nombre  string
		cambiar func([]any)
	}{
		{"usuario efectivo sustituido", func(fila []any) {
			fila[2] = "rol_sustituido"
		}},
		{"OID no resuelto", func(fila []any) { fila[0] = uint32(0) }},
		{"TLS no usado", func(fila []any) { fila[3] = false }},
		{"réplica", func(fila []any) { fila[4] = false }},
		{"LOGIN inseguro", func(fila []any) { fila[5] = false }},
		{"grupo inseguro", func(fila []any) { fila[6] = false }},
		{"membresía directa insegura", func(fila []any) { fila[7] = false }},
		{"membresía adicional", func(fila []any) { fila[8] = false }},
		{"LOGIN con autoridad", func(fila []any) { fila[9] = false }},
		{"grupo con ACL adicional", func(fila []any) { fila[10] = false }},
		{"privilegios efectivos adicionales", func(fila []any) {
			fila[11] = false
		}},
		{"OID/esquema/nombre/prokind", func(fila []any) { fila[12] = false }},
		{"propietario", func(fila []any) { fila[13] = false }},
		{"security definer o stable", func(fila []any) { fila[14] = false }},
		{"proconfig exacto", func(fila []any) { fila[15] = false }},
		{"firma retorno argumentos", func(fila []any) { fila[16] = false }},
		{"lenguaje o probin", func(fila []any) { fila[17] = false }},
		{"prosrc", func(fila []any) { fila[18] = false }},
		{"definición canónica", func(fila []any) { fila[19] = false }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fila := append([]any(nil), filaValida...)
			caso.cambiar(fila)
			conexion := &conexionAcreditacionPoolO405Prueba{
				fila: filaAcreditacionPoolO405Prueba{valores: fila},
			}
			origen := &origenAcreditacionPoolO405Prueba{
				configuracion: configuracionPoolAcreditacionO405Prueba(),
				conexion:      conexion,
			}
			err := acreditarPoolRecuperacionCoberturaO405(
				context.Background(),
				origen,
				modoTLSAcreditacionPoolO405Produccion,
			)
			if !errors.Is(
				err,
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
			) {
				t.Fatalf("manifiesto hostil aceptado: %v", err)
			}
			if origen.adquisiciones != 1 || conexion.liberaciones != 1 {
				t.Fatalf(
					"ciclo físico incompleto: adquirir=%d liberar=%d",
					origen.adquisiciones,
					conexion.liberaciones,
				)
			}
		})
	}

	conexion := &conexionAcreditacionPoolO405Prueba{
		fila: filaAcreditacionPoolO405Prueba{valores: filaValida},
	}
	origen := &origenAcreditacionPoolO405Prueba{
		configuracion: configuracionPoolAcreditacionO405Prueba(),
		conexion:      conexion,
	}
	if err := acreditarPoolRecuperacionCoberturaO405(
		context.Background(),
		origen,
		modoTLSAcreditacionPoolO405Produccion,
	); err != nil {
		t.Fatalf("manifiesto válido rechazado: %v", err)
	}
	if origen.adquisiciones != 1 || conexion.liberaciones != 1 {
		t.Fatalf(
			"conexión no liberada: adquirir=%d liberar=%d",
			origen.adquisiciones,
			conexion.liberaciones,
		)
	}
	if len(conexion.argumentos) != 11 {
		t.Fatalf("argumentos inesperados: %#v", conexion.argumentos)
	}
	configuracionEsperada, configuracionValida :=
		conexion.argumentos[5].([]string)
	if conexion.argumentos[0] !=
		firmaFuncionRecuperacionResultadoCoberturaO405 ||
		conexion.argumentos[1] != rolLectorResultadoCoberturaO405 ||
		conexion.argumentos[2] !=
			esquemaFuncionRecuperacionResultadoCoberturaO405 ||
		conexion.argumentos[3] !=
			nombreFuncionRecuperacionResultadoCoberturaO405 ||
		conexion.argumentos[4] !=
			propietarioFuncionRecuperacionResultadoCoberturaO405 ||
		conexion.argumentos[6] !=
			argumentosFuncionRecuperacionResultadoCoberturaO405 ||
		conexion.argumentos[7] !=
			retornoFuncionRecuperacionResultadoCoberturaO405 ||
		conexion.argumentos[8] !=
			lenguajeFuncionRecuperacionResultadoCoberturaO405 ||
		conexion.argumentos[9] !=
			huellaProsrcFuncionRecuperacionResultadoCoberturaO405 ||
		conexion.argumentos[10] !=
			huellaDefinicionFuncionRecuperacionResultadoCoberturaO405 ||
		!configuracionValida ||
		strings.Join(configuracionEsperada, "\x00") != strings.Join(
			configuracionFuncionRecuperacionResultadoCoberturaO405(),
			"\x00",
		) {
		t.Fatalf("manifiesto parametrizado inesperado: %#v", conexion.argumentos)
	}
	for _, fragmento := range []string{
		"pg_catalog.pg_stat_ssl",
		"version='TLSv1.2'",
		"'ECDHE-ECDSA-AES128-GCM-SHA256'",
		"'ECDHE-ECDSA-AES256-GCM-SHA384'",
		"'ECDHE-RSA-AES128-GCM-SHA256'",
		"'ECDHE-RSA-AES256-GCM-SHA384'",
		"'ECDHE-ECDSA-CHACHA20-POLY1305'",
		"'ECDHE-RSA-CHACHA20-POLY1305'",
		"version='TLSv1.3'",
		"'TLS_AES_128_GCM_SHA256'",
		"'TLS_AES_256_GCM_SHA384'",
		"'TLS_CHACHA20_POLY1305_SHA256'",
		"pg_catalog.pg_is_in_recovery",
		"session_user::text,current_user::text",
		"NOT login.rolsuper",
		"NOT login.rolcreatedb",
		"NOT login.rolcreaterole",
		"NOT login.rolreplication",
		"NOT login.rolbypassrls",
		"NOT grupo.rolcanlogin",
		"pg_catalog.pg_auth_members",
		"pg_catalog.pg_has_role",
		"'MEMBER'",
		"pg_catalog.pg_shdepend",
		"count(*)=3",
		"acl.privilege_type='CONNECT'",
		"acl.privilege_type='USAGE'",
		"acl.privilege_type='EXECUTE'",
		"NOT acl.is_grantable",
		"pg_catalog.has_database_privilege",
		"pg_catalog.has_schema_privilege",
		"pg_catalog.has_table_privilege",
		"pg_catalog.has_sequence_privilege",
		"pg_catalog.to_regprocedure($1::text)",
		"esquema.nspname=$3",
		"procedimiento.proname=$4",
		"procedimiento.prokind='f'",
		"procedimiento.proowner=propietario.oid",
		"propietario.rolname=$5",
		"procedimiento.prosecdef",
		"procedimiento.provolatile='s'",
		"procedimiento.proconfig",
		"ORDER BY valor COLLATE \"C\"",
		"IS NOT DISTINCT FROM $6::text[]",
		"pg_get_function_identity_arguments",
		"pg_get_function_arguments",
		"pg_get_function_result",
		"procedimiento.proargtypes[0]",
		"procedimiento.proallargtypes",
		"procedimiento.proargmodes",
		"procedimiento.proargnames",
		"lenguaje.lanname=$9",
		"procedimiento.probin IS NULL",
		"procedimiento.prosrc",
		"pg_catalog.pg_get_functiondef",
		"pg_catalog.sha256",
		"count(*)=1",
	} {
		if !strings.Contains(conexion.consulta, fragmento) {
			t.Fatalf("acreditación incompleta: falta %q", fragmento)
		}
	}
	if strings.Contains(
		conexion.consulta,
		"recuperar_resultado_propio_decision_cobertura_o405_v1(",
	) {
		t.Fatal("la acreditación intentó ejecutar la función O4-05")
	}
}

func TestConfiguracionTLSAcreditacionPoolO405CierraProduccionYPrueba(
	t *testing.T,
) {
	segura := configuracionTLSAcreditacionPoolO405Prueba()
	seguraConFallback := configuracionTLSAcreditacionPoolO405Prueba()
	seguraConFallback.Fallbacks = []*pgconn.FallbackConfig{{
		Host: "replica-o405.example",
		TLSConfig: &tls.Config{
			ServerName: "replica-o405.example",
			MinVersion: tls.VersionTLS13,
		},
	}}
	raicesVacias := configuracionTLSAcreditacionPoolO405Prueba()
	raicesVacias.TLSConfig.RootCAs = x509.NewCertPool()
	cipherInseguro := configuracionTLSAcreditacionPoolO405Prueba()
	cipherInseguro.TLSConfig.CipherSuites = []uint16{
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	}
	cipherSeguro := configuracionTLSAcreditacionPoolO405Prueba()
	cipherSeguro.TLSConfig.CipherSuites = []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	}
	cipherNilTLS12 := configuracionTLSAcreditacionPoolO405Prueba()
	cipherNilTLS12.TLSConfig.CipherSuites = nil
	cipherNilTLS13 := configuracionTLSAcreditacionPoolO405Prueba()
	cipherNilTLS13.TLSConfig.MinVersion = tls.VersionTLS13
	cipherNilTLS13.TLSConfig.CipherSuites = nil
	cipherCBC := configuracionTLSAcreditacionPoolO405Prueba()
	cipherCBC.TLSConfig.CipherSuites = []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	}
	keyLog := configuracionTLSAcreditacionPoolO405Prueba()
	keyLog.TLSConfig.KeyLogWriter = &strings.Builder{}
	randPersonalizado := configuracionTLSAcreditacionPoolO405Prueba()
	randPersonalizado.TLSConfig.Rand = strings.NewReader("azar-no-acreditable")
	relojPersonalizado := configuracionTLSAcreditacionPoolO405Prueba()
	relojPersonalizado.TLSConfig.Time = func() time.Time {
		return time.Unix(1, 0)
	}
	renegociacion := configuracionTLSAcreditacionPoolO405Prueba()
	renegociacion.TLSConfig.Renegotiation =
		tls.RenegotiateOnceAsClient
	verificarPar := configuracionTLSAcreditacionPoolO405Prueba()
	verificarPar.TLSConfig.VerifyPeerCertificate = func(
		[][]byte,
		[][]*x509.Certificate,
	) error {
		return nil
	}
	verificarConexion := configuracionTLSAcreditacionPoolO405Prueba()
	verificarConexion.TLSConfig.VerifyConnection = func(
		tls.ConnectionState,
	) error {
		return nil
	}
	certificadoDinamico := configuracionTLSAcreditacionPoolO405Prueba()
	certificadoDinamico.TLSConfig.GetClientCertificate = func(
		*tls.CertificateRequestInfo,
	) (*tls.Certificate, error) {
		return &tls.Certificate{}, nil
	}
	cacheSesion := configuracionTLSAcreditacionPoolO405Prueba()
	cacheSesion.TLSConfig.ClientSessionCache =
		tls.NewLRUClientSessionCache(1)
	verificacionECH := configuracionTLSAcreditacionPoolO405Prueba()
	verificacionECH.TLSConfig.EncryptedClientHelloRejectionVerify = func(
		tls.ConnectionState,
	) error {
		return nil
	}
	fallbackConCallback := configuracionTLSAcreditacionPoolO405Prueba()
	fallbackConCallback.Fallbacks = []*pgconn.FallbackConfig{{
		Host: "alternativa-o405.example",
		TLSConfig: &tls.Config{
			ServerName: "alternativa-o405.example",
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
			Time: func() time.Time { return time.Unix(1, 0) },
		},
	}}
	casos := []struct {
		nombre        string
		configuracion *pgconn.Config
		modo          modoTLSAcreditacionPoolO405
		valida        bool
	}{
		{"verify-full", segura, modoTLSAcreditacionPoolO405Produccion, true},
		{
			"verify-full cipher seguro",
			cipherSeguro,
			modoTLSAcreditacionPoolO405Produccion,
			true,
		},
		{
			"cipher inseguro",
			cipherInseguro,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"TLS 1.2 sin allowlist explícita",
			cipherNilTLS12,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"TLS 1.3 usa allowlist fija",
			cipherNilTLS13,
			modoTLSAcreditacionPoolO405Produccion,
			true,
		},
		{
			"CBC fuera de allowlist",
			cipherCBC,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{"KeyLogWriter", keyLog, modoTLSAcreditacionPoolO405Produccion, false},
		{
			"Rand personalizado",
			randPersonalizado,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"Time personalizado",
			relojPersonalizado,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"renegociación",
			renegociacion,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"VerifyPeerCertificate",
			verificarPar,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"VerifyConnection",
			verificarConexion,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"GetClientCertificate",
			certificadoDinamico,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"ClientSessionCache",
			cacheSesion,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"verificación ECH custom",
			verificacionECH,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"fallback con callback",
			fallbackConCallback,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"verify-full con fallback",
			seguraConFallback,
			modoTLSAcreditacionPoolO405Produccion,
			true,
		},
		{"configuración nula", nil, modoTLSAcreditacionPoolO405Produccion, false},
		{
			"producción sin TLS",
			&pgconn.Config{Host: "db-o405.example"},
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"insecure",
			&pgconn.Config{Host: "db-o405.example", TLSConfig: &tls.Config{
				ServerName: "db-o405.example", InsecureSkipVerify: true,
			}},
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"nombre distinto",
			&pgconn.Config{Host: "db-o405.example", TLSConfig: &tls.Config{
				ServerName: "otra.example",
			}},
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"TLS obsoleto",
			&pgconn.Config{Host: "db-o405.example", TLSConfig: &tls.Config{
				ServerName: "db-o405.example", MinVersion: tls.VersionTLS11,
			}},
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"raíces explícitas vacías",
			raicesVacias,
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"fallback nil",
			&pgconn.Config{
				Host:      "db-o405.example",
				TLSConfig: segura.TLSConfig,
				Fallbacks: []*pgconn.FallbackConfig{nil},
			},
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"fallback sin TLS",
			&pgconn.Config{
				Host:      "db-o405.example",
				TLSConfig: segura.TLSConfig,
				Fallbacks: []*pgconn.FallbackConfig{{
					Host: "replica-o405.example",
				}},
			},
			modoTLSAcreditacionPoolO405Produccion,
			false,
		},
		{
			"socket Unix de prueba",
			&pgconn.Config{Host: "/var/run/postgresql"},
			modoTLSAcreditacionPoolO405SocketUnixPrueba,
			true,
		},
		{
			"loopback no es socket",
			&pgconn.Config{Host: "127.0.0.1"},
			modoTLSAcreditacionPoolO405SocketUnixPrueba,
			false,
		},
		{
			"socket con fallback Unix",
			&pgconn.Config{
				Host: "/var/run/postgresql",
				Fallbacks: []*pgconn.FallbackConfig{{
					Host: "/run/postgresql",
				}},
			},
			modoTLSAcreditacionPoolO405SocketUnixPrueba,
			true,
		},
		{
			"socket con fallback TCP",
			&pgconn.Config{
				Host: "/var/run/postgresql",
				Fallbacks: []*pgconn.FallbackConfig{{
					Host: "127.0.0.1",
				}},
			},
			modoTLSAcreditacionPoolO405SocketUnixPrueba,
			false,
		},
		{
			"socket de prueba con TLS",
			&pgconn.Config{
				Host:      "/var/run/postgresql",
				TLSConfig: &tls.Config{ServerName: "localhost"},
			},
			modoTLSAcreditacionPoolO405SocketUnixPrueba,
			false,
		},
		{
			"modo desconocido",
			segura,
			modoTLSAcreditacionPoolO405(99),
			false,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if obtenida := configuracionTLSAcreditacionPoolO405Valida(
				caso.configuracion,
				caso.modo,
			); obtenida != caso.valida {
				t.Fatalf("validez=%t; quiere %t", obtenida, caso.valida)
			}
		})
	}
}

func TestAcreditacionPoolO405SocketPruebaExigeTLSInactivo(
	t *testing.T,
) {
	for _, tlsActivo := range []bool{false, true} {
		t.Run(map[bool]string{false: "sin TLS", true: "TLS inesperado"}[tlsActivo],
			func(t *testing.T) {
				conexion := &conexionAcreditacionPoolO405Prueba{
					fila: filaAcreditacionPoolO405Prueba{
						valores: valoresAcreditacionPoolO405Prueba(tlsActivo),
					},
				}
				configuracionSocket :=
					configuracionConexionAcreditacionPoolO405Prueba()
				configuracionSocket.Host = "/var/run/postgresql"
				configuracionSocket.TLSConfig = nil
				configuracionSocket.Fallbacks = nil
				origen := &origenAcreditacionPoolO405Prueba{
					configuracion: &pgxpool.Config{
						ConnConfig: configuracionSocket,
					},
					conexion: conexion,
				}
				conexion.configuracion = origen.configuracion.ConnConfig
				err := acreditarPoolRecuperacionCoberturaO405(
					context.Background(),
					origen,
					modoTLSAcreditacionPoolO405SocketUnixPrueba,
				)
				if tlsActivo && !errors.Is(
					err,
					cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
				) {
					t.Fatalf("TLS inesperado aceptado: %v", err)
				}
				if !tlsActivo && err != nil {
					t.Fatalf("socket Unix de prueba rechazado: %v", err)
				}
			},
		)
	}
}

type contextoAcreditacionPoolO405Nulo struct{}

func (*contextoAcreditacionPoolO405Nulo) Deadline() (time.Time, bool) {
	return time.Time{}, false
}
func (*contextoAcreditacionPoolO405Nulo) Done() <-chan struct{} { return nil }
func (*contextoAcreditacionPoolO405Nulo) Err() error            { return nil }
func (*contextoAcreditacionPoolO405Nulo) Value(any) any         { return nil }

func TestAcreditacionPoolO405RechazaContextoNuloTipado(t *testing.T) {
	var ctxNulo *contextoAcreditacionPoolO405Nulo
	origen := &origenAcreditacionPoolO405Prueba{
		configuracion: configuracionPoolAcreditacionO405Prueba(),
	}
	if err := acreditarPoolRecuperacionCoberturaO405(
		ctxNulo,
		origen,
		modoTLSAcreditacionPoolO405Produccion,
	); !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) || origen.adquisiciones != 0 {
		t.Fatalf("contexto nulo tipado aceptado: %v", err)
	}
}
