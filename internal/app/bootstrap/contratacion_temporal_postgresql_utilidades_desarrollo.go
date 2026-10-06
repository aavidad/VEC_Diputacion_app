package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"time"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/auditoria"
)

func registrarFalloPostgreSQLContratacionTemporalDesarrollo(etapa, causa string) {
	slog.Error(
		"fallo al componer PostgreSQL de contratacion temporal en desarrollo",
		"etapa", etapa,
		"causa", causa,
	)
}

func codigoFalloGobiernoPostgreSQLContratacionTemporalDesarrollo(err error) string {
	switch {
	case errors.Is(err, errGobiernoPostgreSQLContratacionTemporalDesarrolloAjeno):
		return "gobierno_actual_ajeno"
	case errors.Is(err, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente):
		return "gobierno_sintetico_incoherente"
	case errors.Is(err, errGobiernoPostgreSQLContratacionTemporalDesarrolloAgotado):
		return "versiones_agotadas"
	default:
		return "publicacion_no_disponible"
	}
}

// jitSesionPostgreSQLContratacionTemporalDesarrollo apaga la compilación JIT
// en las sesiones de Contratación temporal y Bolsa. Son lecturas cortas y
// repetidas: con estimaciones infladas (una función que el planificador cree
// que devuelve 1000 filas por participación) PostgreSQL compilaba con LLVM en
// cada llamada y el orden vigente de una bolsa pasaba de ~30 ms a ~350 ms.
// No cambia resultados, permisos ni transacciones; solo el plan de ejecución.
const jitSesionPostgreSQLContratacionTemporalDesarrollo = "off"

// aplicarParametrosSesionPostgreSQLContratacionTemporalDesarrollo fija los
// parámetros de sesión comunes de los pools de Contratación temporal y Bolsa.
func aplicarParametrosSesionPostgreSQLContratacionTemporalDesarrollo(parametros map[string]string, aplicacion string) {
	parametros["application_name"] = aplicacion
	parametros["timezone"] = "UTC"
	parametros["search_path"] = "pg_catalog,pg_temp"
	parametros["default_transaction_isolation"] = "serializable"
	parametros["default_transaction_read_only"] = "off"
	parametros["statement_timeout"] = "15s"
	parametros["lock_timeout"] = "3s"
	parametros["idle_in_transaction_session_timeout"] = "20s"
	parametros["jit"] = jitSesionPostgreSQLContratacionTemporalDesarrollo
}

func abrirPoolPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	dsn string,
	aplicacion string,
	rolEsperado string,
) (*pgxpool.Pool, string, error) {
	configuracion, err := pgxpool.ParseConfig(dsn)
	if err != nil || configuracion == nil || configuracion.ConnConfig == nil ||
		validarTLSPostgreSQLBorradores(&configuracion.ConnConfig.Config, true) != nil {
		return nil, "", falloPostgreSQLCTDesarrollo(err)
	}
	configuracion.MaxConns = 4
	if rolPoolIncorporacionV2(rolEsperado) {
		configuracion.AfterConnect = func(_ context.Context, c *pgx.Conn) error {
			c.TypeMap().RegisterType(&pgtype.Type{Name: "timestamptz", OID: pgtype.TimestamptzOID, Codec: &pgtype.TimestamptzCodec{ScanLocation: time.UTC}})
			c.TypeMap().RegisterType(&pgtype.Type{Name: "timestamp", OID: pgtype.TimestampOID, Codec: &pgtype.TimestampCodec{ScanLocation: time.UTC}})
			return nil
		}
	}
	configuracion.MinConns = 0
	configuracion.ConnConfig.ConnectTimeout = 5 * time.Second
	if configuracion.ConnConfig.RuntimeParams == nil {
		configuracion.ConnConfig.RuntimeParams = make(map[string]string)
	}
	aplicarParametrosSesionPostgreSQLContratacionTemporalDesarrollo(configuracion.ConnConfig.RuntimeParams, aplicacion)
	configurarVerificacionPorConexionAuditoriaFronteraBolsaDesarrollo(configuracion, rolEsperado)
	pool, err := pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil {
		return nil, "", falloPostgreSQLCTDesarrollo(err)
	}
	usuario, err := comprobarIdentidadPostgreSQLContratacionTemporalDesarrollo(
		ctx, pool, rolEsperado,
	)
	if err != nil {
		pool.Close()
		return nil, "", err
	}
	return pool, usuario, nil
}

func comprobarIdentidadPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	consultador interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	rolEsperado string,
) (string, error) {
	if ctx == nil || consultador == nil ||
		(rolEsperado != rolEjecucionPostgreSQLContratacionTemporalDesarrollo &&
			rolEsperado != rolGobiernoPostgreSQLContratacionTemporalDesarrollo &&
			rolEsperado != rolRegistroAutorizacionPostgreSQLContratacionTemporalDesarrollo &&
			rolEsperado != rolConfirmadorPostgreSQLContratacionTemporalDesarrollo &&
			rolEsperado != rolEjecucionBolsaLlamamientosDesarrollo &&
			rolEsperado != rolRegistroIdentidadConsultasDesarrollo &&
			rolEsperado != rolRevalidacionIdentidadConsultasDesarrollo &&
			rolEsperado != rolContextoActorConsultasDesarrollo &&
			rolEsperado != rolLectorPostgreSQLContratacionTemporalDesarrollo &&
			rolEsperado != rolAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo &&
			rolEsperado != rolAuditoriaFronteraPostgreSQLBolsaDesarrollo &&
			!rolPoolIncorporacionV2(rolEsperado)) {
		return "", falloPostgreSQLCTDesarrollo(nil)
	}
	if rolEsperado == rolAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo {
		return comprobarIdentidadAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo(ctx, consultador)
	}
	if rolEsperado == rolAuditoriaFronteraPostgreSQLBolsaDesarrollo {
		return comprobarIdentidadAuditoriaFronteraPostgreSQLBolsaDesarrollo(ctx, consultador)
	}
	var usuario string
	var valido bool
	err := consultador.QueryRow(ctx, `
		SELECT session_user::text,
		       session_user = current_user
		       AND identidad.rolcanlogin
		       AND identidad.rolinherit
		       AND NOT identidad.rolsuper
		       AND NOT identidad.rolcreatedb
		       AND NOT identidad.rolcreaterole
		       AND NOT identidad.rolreplication
		       AND NOT identidad.rolbypassrls
		       AND pg_catalog.pg_has_role(session_user, $1, 'MEMBER')
		  FROM pg_catalog.pg_roles AS identidad
		 WHERE identidad.rolname = session_user`, rolEsperado).Scan(&usuario, &valido)
	if err != nil || !valido || usuario == "" {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	return usuario, nil
}

// comprobarIdentidadAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo
// admite exclusivamente el grupo registrador. La CTE recorre las membresías
// directas y transitivas del LOGIN; cualquier segundo rol es incompatible,
// incluso si no hereda privilegios automáticamente.
func comprobarIdentidadAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	consultador interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
) (string, error) {
	if ctx == nil || consultador == nil {
		return "", falloPostgreSQLCTDesarrollo(nil)
	}
	var usuario string
	var valido bool
	err := consultador.QueryRow(ctx, `
		WITH RECURSIVE membresias_efectivas(rol_id) AS (
			SELECT directa.roleid
			  FROM pg_catalog.pg_auth_members AS directa
			 WHERE directa.member = session_user::regrole
			UNION
			SELECT siguiente.roleid
			  FROM pg_catalog.pg_auth_members AS siguiente
			  JOIN membresias_efectivas AS previa ON previa.rol_id = siguiente.member
		)
		SELECT session_user::text,
		       session_user = current_user
		       AND identidad.rolcanlogin
		       AND identidad.rolinherit
		       AND NOT identidad.rolsuper
		       AND NOT identidad.rolcreatedb
		       AND NOT identidad.rolcreaterole
		       AND NOT identidad.rolreplication
		       AND NOT identidad.rolbypassrls
		       AND pg_catalog.pg_has_role(
		           session_user,
		           $1::regrole,
		           'MEMBER'
		       )
		       AND NOT EXISTS (
		           SELECT 1
		             FROM membresias_efectivas
		            WHERE rol_id <> $1::regrole
		       )
		  FROM pg_catalog.pg_roles AS identidad
		 WHERE identidad.rolname = session_user`,
		rolAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo,
	).Scan(&usuario, &valido)
	if err != nil || !valido || usuario == "" {
		return "", falloPostgreSQLCTDesarrollo(err)
	}
	return usuario, nil
}
func nuevoMaterialAtestacionContratacionTemporalDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	ahora time.Time,
) (materialAtestacionContratacionTemporalDesarrollo, error) {
	vacio := materialAtestacionContratacionTemporalDesarrollo{}
	if derivador == nil || !derivador.valido() {
		return vacio, falloPostgreSQLCTDesarrollo(nil)
	}
	resultados, err := derivador.calcularHMAC(
		[]byte("vec.ct.desarrollo.atestacion-v3.ed25519.v1"),
		[]byte("vec.ct.desarrollo.atestacion-v3.capacidad-hmac.v1"),
	)
	if err != nil || len(resultados) == 0 {
		return vacio, falloPostgreSQLCTDesarrollo(err)
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	activo := resultados[0]
	semilla := append([]byte(nil), activo.localizador[:]...)
	defer borrarBytes(semilla)
	privada := ed25519.NewKeyFromSeed(semilla)
	publica := privada.Public().(ed25519.PublicKey)
	claveID := "clave:atestacion:ct:desarrollo:v" + numeroDecimal(activo.generacion)
	claveHMACID := "clave:capacidad:ct:desarrollo:v" + numeroDecimal(activo.generacion)
	validaDesde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validaHasta := time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC)
	ahora = ahora.UTC().Truncate(time.Microsecond)
	if ahora.Before(validaDesde) || !ahora.Before(validaHasta) {
		return vacio, falloPostgreSQLCTDesarrollo(nil)
	}
	publicadaEn := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, time.UTC)
	expiraEn := publicadaEn.Add(24 * time.Hour)
	secuencia := uint64(ahora.Year()*10000 + int(ahora.Month())*100 + ahora.Day())
	configuracionRef := "confianza:atestacion:ct:desarrollo:" + publicadaEn.Format("2006-01-02")
	raiz, err := confianzaatestacion.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		claveID, 1, publica, audienciaAtestacionContratacionTemporalDesarrollo,
		confianzaatestacion.EstadoClaveAtestacionAutorizacionV3Activa,
		validaDesde, validaHasta, time.Time{},
	)
	if err != nil {
		return vacio, falloPostgreSQLCTDesarrollo(err)
	}
	configuracion, err := confianzaatestacion.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		configuracionRef, secuencia, publicadaEn, expiraEn, raiz,
	)
	if err != nil {
		return vacio, falloPostgreSQLCTDesarrollo(err)
	}
	huellaConfiguracion, err := configuracion.HuellaSHA256ParaGobierno()
	if err != nil {
		return vacio, falloPostgreSQLCTDesarrollo(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		return vacio, falloPostgreSQLCTDesarrollo(err)
	}
	huellaSPKI := sha256.Sum256(spki)
	claveHMAC := append([]byte(nil), activo.huellaSolicitud[:]...)
	huellaSecreto := sha256.Sum256(claveHMAC)
	calculadorGobierno := sha256.New()
	_, _ = calculadorGobierno.Write([]byte("vec.ct.desarrollo.capacidad-v3.gobierno.v1\x00"))
	_, _ = calculadorGobierno.Write(claveHMAC)
	huellaGobierno := hex.EncodeToString(calculadorGobierno.Sum(nil))
	emisorID := "emisor:ct:desarrollo:v" + numeroDecimal(activo.generacion)
	capacidad, err := confianzaatestacion.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(
		claveHMACID, 1, claveHMAC, emisorID,
		audienciaConsumoAltaContratacionTemporal,
		confianzaatestacion.EstadoClaveHMACCapacidadAtestacionV3Emision,
		validaDesde, validaHasta, time.Time{}, 1, huellaGobierno,
	)
	if err != nil {
		borrarBytes(claveHMAC)
		return vacio, falloPostgreSQLCTDesarrollo(err)
	}
	return materialAtestacionContratacionTemporalDesarrollo{
		claveID: claveID, claveVersion: 1, privada: privada,
		raiz: raiz, configuracion: configuracion,
		configuracionRef: configuracionRef, configuracionOrden: secuencia,
		configuracionHuella: huellaConfiguracion,
		publicadaEn:         publicadaEn, expiraEn: expiraEn,
		validaDesde: validaDesde, validaHasta: validaHasta,
		spki: spki, spkiHuella: hex.EncodeToString(huellaSPKI[:]),
		claveHMACID: claveHMACID, claveHMACVersion: 1, claveHMAC: claveHMAC,
		claveHMACOrden: 1, claveHMACRevision: 1, claveHMACHuella: huellaGobierno,
		claveHMACSecreto: hex.EncodeToString(huellaSecreto[:]),
		emisorID:         emisorID, capacidad: capacidad,
		audienciaConsumo: audienciaConsumoAltaContratacionTemporal,
	}, nil
}

func numeroDecimal(valor uint32) string {
	const digitos = "0123456789"
	if valor == 0 {
		return "0"
	}
	var buffer [10]byte
	indice := len(buffer)
	for valor > 0 {
		indice--
		buffer[indice] = digitos[valor%10]
		valor /= 10
	}
	return string(buffer[indice:])
}

func (m *materialAtestacionContratacionTemporalDesarrollo) borrarCopiasEfimeras() {
	if m == nil {
		return
	}
	borrarBytes(m.claveHMAC)
	borrarBytes(m.spki)
	borrarBytes(m.privada)
}

const maximoVersionGobiernoPostgreSQLContratacionTemporalDesarrollo int64 = 9007199254740991

const rolConsultaAuditoriaCTDesarrollo = "vec_contratacion_temporal_consultor_rrhh"

const funcionConsultaAuditoriaCTDesarrollo = "vec_contratacion_temporal.consultar_auditoria_ct_atestada_v1(text,text,text,timestamptz,timestamptz,integer,timestamptz,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
const funcionConsultaAuditoriaBolsaDesarrollo = "vec_bolsa_llamamientos.consultar_auditoria_participacion_v1(text,text,timestamptz,timestamptz,timestamptz,text,text,integer,text,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"

// La consulta CT usa el LOGIN consultor nominal ya configurado. Abre un pool
// exclusivo para la superficie Auditoría y coteja su membresía en cada conexión.
func abrirPoolConsultaAuditoriaCTDesarrollo(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if ctx == nil || ctx.Err() != nil || dsn == "" {
		return nil, auditoria.ErrNoDisponible
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c == nil || c.ConnConfig == nil || c.ConnConfig.User == "" ||
		validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, auditoria.ErrNoDisponible
	}
	c.MaxConns, c.MinConns = 2, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = make(map[string]string)
	}
	p := c.ConnConfig.RuntimeParams
	p["application_name"] = "vec-auditoria-ct-rrhh"
	p["timezone"] = "UTC"
	p["search_path"] = "pg_catalog,pg_temp"
	p["default_transaction_read_only"] = "off"
	p["default_transaction_isolation"] = "serializable"
	p["statement_timeout"] = "15s"
	p["lock_timeout"] = "2s"
	p["idle_in_transaction_session_timeout"] = "20s"
	login := c.ConnConfig.User
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return comprobarPoolConsultaAuditoriaCTDesarrollo(ctx, conn, login)
	}
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, auditoria.ErrNoDisponible
	}
	if err := comprobarPoolConsultaAuditoriaCTDesarrollo(ctx, pool, login); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func comprobarPoolConsultaAuditoriaCTDesarrollo(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, login string) error {
	if ctx == nil || ctx.Err() != nil || q == nil || login == "" {
		return auditoria.ErrNoDisponible
	}
	const sonda = `SELECT session_user::text,
	 session_user=current_user AND l.rolcanlogin AND l.rolinherit
	 AND NOT l.rolsuper AND NOT l.rolcreatedb AND NOT l.rolcreaterole
	 AND NOT l.rolreplication AND NOT l.rolbypassrls
	 AND NOT g.rolcanlogin AND NOT g.rolbypassrls
	 AND pg_has_role(session_user,g.oid,'MEMBER')
	 AND pg_has_role(session_user,g.oid,'USAGE')
	 AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=l.oid)=1
	 AND EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
	            AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
	 AND NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.roleid=l.oid)
	 AND coalesce(has_function_privilege(session_user,to_regprocedure($2)::oid,'EXECUTE'),false)
	 AND NOT coalesce(has_function_privilege(session_user,to_regprocedure($3)::oid,'EXECUTE'),false)
	 FROM pg_roles l JOIN pg_roles g ON g.rolname=$1 WHERE l.rolname=session_user`
	var usuario string
	var valido bool
	sondaCtx, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	if err := q.QueryRow(sondaCtx, sonda, rolConsultaAuditoriaCTDesarrollo,
		funcionConsultaAuditoriaCTDesarrollo, funcionConsultaAuditoriaBolsaDesarrollo).Scan(&usuario, &valido); err != nil || !valido || usuario != login {
		return auditoria.ErrNoDisponible
	}
	return nil
}

// Los dos pools de autoridad comparten el protocolo central, con LOGIN y
// membresía nominal distintos del ejecutor CT/Bolsa y entre sí.
func abrirPoolAutoridadAuditoriaDesarrollo(ctx context.Context, dsn, rol, aplicacion string) (*pgxpool.Pool, error) {
	if ctx == nil || ctx.Err() != nil || dsn == "" || rol == "" || aplicacion == "" {
		return nil, auditoria.ErrNoDisponible
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c == nil || c.ConnConfig == nil || c.ConnConfig.User == "" ||
		validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, auditoria.ErrNoDisponible
	}
	c.MaxConns, c.MinConns = 2, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = make(map[string]string)
	}
	p := c.ConnConfig.RuntimeParams
	p["application_name"] = aplicacion
	p["timezone"] = "UTC"
	p["search_path"] = "pg_catalog,pg_temp"
	p["default_transaction_read_only"] = "off"
	p["statement_timeout"] = "15s"
	p["lock_timeout"] = "2s"
	p["idle_in_transaction_session_timeout"] = "20s"
	login := c.ConnConfig.User
	c.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if err := comprobarPoolAutoridadAuditoriaDesarrollo(ctx, conn, login, rol); err != nil {
			return err
		}
		if rol == "vec_contratacion_temporal_registrador_auditoria" {
			return preflightRegistradorFronteraAuditoriaConsultaDesarrollo(ctx, conn)
		}
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, auditoria.ErrNoDisponible
	}
	if err := comprobarPoolAutoridadAuditoriaDesarrollo(ctx, pool, login, rol); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func comprobarPoolAutoridadAuditoriaDesarrollo(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, login, rol string) error {
	if ctx == nil || ctx.Err() != nil || q == nil || login == "" ||
		(rol != "vec_autorizacion_fuente" && rol != "vec_autorizacion_motivos_evaluador" && rol != "vec_contratacion_temporal_registrador_auditoria") {
		return auditoria.ErrNoDisponible
	}
	const sonda = `SELECT session_user::text,
	 session_user=current_user AND l.rolcanlogin AND l.rolinherit
	 AND NOT l.rolsuper AND NOT l.rolcreatedb AND NOT l.rolcreaterole
	 AND NOT l.rolreplication AND NOT l.rolbypassrls
	 AND NOT g.rolcanlogin
	 AND g.rolinherit = (g.rolname IN ('vec_autorizacion_fuente','vec_contratacion_temporal_registrador_auditoria'))
	 AND NOT g.rolsuper
	 AND NOT g.rolcreatedb AND NOT g.rolcreaterole AND NOT g.rolreplication AND NOT g.rolbypassrls
	 AND pg_has_role(session_user,g.oid,'MEMBER') AND pg_has_role(session_user,g.oid,'USAGE')
	 AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=l.oid)=1
	 AND EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
	            AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
	 AND NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.roleid=l.oid)
	 AND NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=g.oid)
	 FROM pg_roles l JOIN pg_roles g ON g.rolname=$1 WHERE l.rolname=session_user`
	var usuario string
	var valido bool
	sondaCtx, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	if err := q.QueryRow(sondaCtx, sonda, rol).Scan(&usuario, &valido); err != nil || !valido || usuario != login {
		return auditoria.ErrNoDisponible
	}
	return nil
}
