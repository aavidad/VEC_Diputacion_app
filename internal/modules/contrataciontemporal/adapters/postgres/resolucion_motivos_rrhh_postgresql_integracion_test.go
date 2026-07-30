package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	variableDSNM23           = "VEC_M23_RESOLUTOR_DSN"
	variableDSNUnicoM23      = "VEC_M23_RESOLUTOR_UNICO_DSN"
	variableDSNAdminM23      = "VEC_M23_ADMIN_DSN"
	variableDSNProyectorM23  = "VEC_M23_PROYECTOR_DSN"
	variableLoginM23         = "VEC_M23_RESOLUTOR_LOGIN"
	variableAislamientoM23   = "VEC_M23_PG18_AISLADO"
	variableBarreraM23       = "VEC_M23_BARRERA"
	catalogoM23              = "motivos_rrhh_m23"
	claveCuadroM23           = "motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	claveDetalleM23          = "motivo_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	claveDetalleCaducableM23 = "motivo_cccccccccccccccccccccccccccccccc"
)

type entornoResolucionMotivosRRHHM23 struct {
	dsn           string
	dsnUnico      string
	login         string
	ctx           context.Context
	cancelar      context.CancelFunc
	administrador *pgxpool.Pool
	proyector     *pgxpool.Pool
}

func cargarEntornoResolucionMotivosRRHHM23(
	t *testing.T,
) *entornoResolucionMotivosRRHHM23 {
	t.Helper()
	dsn := os.Getenv(variableDSNM23)
	dsnUnico := os.Getenv(variableDSNUnicoM23)
	dsnAdmin := os.Getenv(variableDSNAdminM23)
	dsnProyector := os.Getenv(variableDSNProyectorM23)
	login := os.Getenv(variableLoginM23)
	if dsn == "" || dsnUnico == "" || dsnAdmin == "" ||
		dsnProyector == "" || login == "" {
		t.Skip("PostgreSQL 18.4 efímero M2.3 no solicitado")
	}
	if os.Getenv(variableAislamientoM23) != "1" {
		t.Fatal("M2.3 exige un PostgreSQL 18.4 efímero aislado")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 45*time.Second)
	administrador, err := pgxpool.New(ctx, dsnAdmin)
	if err != nil {
		cancelar()
		t.Fatal("no se pudo abrir la administración efímera M2.3")
	}
	proyector, err := pgxpool.New(ctx, dsnProyector)
	if err != nil {
		administrador.Close()
		cancelar()
		t.Fatal("no se pudo abrir la proyección efímera M2.3")
	}
	entorno := &entornoResolucionMotivosRRHHM23{
		dsn: dsn, dsnUnico: dsnUnico, login: login,
		ctx: ctx, cancelar: cancelar, administrador: administrador,
		proyector: proyector,
	}
	t.Cleanup(func() {
		proyector.Close()
		administrador.Close()
		cancelar()
	})
	var version int
	if err := administrador.QueryRow(
		ctx,
		"SELECT current_setting('server_version_num')::integer",
	).Scan(&version); err != nil || version != 180004 {
		t.Fatal("M2.3 no está ejecutándose sobre PostgreSQL 18.4 exacto")
	}
	return entorno
}

func (e *entornoResolucionMotivosRRHHM23) nuevoResolutor(
	t *testing.T,
) (*PoolResolucionMotivosRRHHPostgreSQL, *ResolutorMotivosRRHHPostgreSQL) {
	return e.nuevoResolutorConDSN(t, e.dsn)
}

func (e *entornoResolucionMotivosRRHHM23) nuevoResolutorConDSN(
	t *testing.T,
	dsn string,
) (*PoolResolucionMotivosRRHHPostgreSQL, *ResolutorMotivosRRHHPostgreSQL) {
	t.Helper()
	pool, err := nuevoPoolResolucionMotivosRRHHPostgreSQL(
		e.ctx, dsn, e.login,
		modoTLSAcreditacionPoolO405SocketUnixPrueba,
	)
	if err != nil {
		t.Fatal("el pool nominal real M2.3 no superó la acreditación")
	}
	t.Cleanup(pool.Cerrar)
	resolutor, err := NuevoResolutorMotivoConsultaRRHHPostgreSQL(pool)
	if err != nil {
		t.Fatal("no se pudo componer el resolutor real M2.3")
	}
	return pool, resolutor
}

func referenciaCuadroM23() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: catalogoM23, CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("2", 64),
		EntradaClave:         claveCuadroM23,
	}
}

func referenciaDetalleM23() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: catalogoM23, CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("2", 64),
		EntradaClave:         claveDetalleM23,
	}
}

func exigirResolucionM23(
	t *testing.T,
	resolutor *ResolutorMotivosRRHHPostgreSQL,
	instante time.Time,
	esperadaCuadro dominiovec.ReferenciaEntradaCatalogo,
	esperadaDetalle dominiovec.ReferenciaEntradaCatalogo,
) {
	t.Helper()
	cuadro, err := resolutor.ResolverMotivoCuadroRRHH(
		context.Background(), instante,
	)
	if err != nil || cuadro != esperadaCuadro {
		t.Fatalf("resolución real de cuadro divergente: %+v", cuadro)
	}
	detalle, err := resolutor.ResolverMotivoDetalleRRHH(
		context.Background(), instante,
	)
	if err != nil || detalle != esperadaDetalle {
		t.Fatalf("resolución real de detalle divergente: %+v", detalle)
	}
}

func exigirNoDisponibleM23(
	t *testing.T,
	obtener func() (dominiovec.ReferenciaEntradaCatalogo, error),
) {
	t.Helper()
	referencia, err := obtener()
	if referencia != (dominiovec.ReferenciaEntradaCatalogo{}) ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) ||
		err.Error() != ports.ErrMotivoConsultaRRHHNoDisponible.Error() {
		t.Fatalf("la ausencia real no falló cerrada: %+v", referencia)
	}
}

func TestIntegracionResolutorMotivosRRHHPostgreSQLSinMembresia(t *testing.T) {
	entorno := cargarEntornoResolucionMotivosRRHHM23(t)
	pool, err := nuevoPoolResolucionMotivosRRHHPostgreSQL(
		entorno.ctx, entorno.dsn, entorno.login,
		modoTLSAcreditacionPoolO405SocketUnixPrueba,
	)
	if pool != nil || !errors.Is(
		err, ports.ErrMotivoConsultaRRHHNoDisponible,
	) {
		t.Fatal("un LOGIN sin la membresía nominal superó la acreditación")
	}
}

func TestIntegracionResolutorMotivosRRHHPostgreSQLAusencia(t *testing.T) {
	entorno := cargarEntornoResolucionMotivosRRHHM23(t)
	_, resolutor := entorno.nuevoResolutor(t)
	instante := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	exigirNoDisponibleM23(t, func() (
		dominiovec.ReferenciaEntradaCatalogo, error,
	) {
		return resolutor.ResolverMotivoCuadroRRHH(context.Background(), instante)
	})
	exigirNoDisponibleM23(t, func() (
		dominiovec.ReferenciaEntradaCatalogo, error,
	) {
		return resolutor.ResolverMotivoDetalleRRHH(context.Background(), instante)
	})
}

func TestIntegracionResolutorMotivosRRHHPostgreSQLPositivo(t *testing.T) {
	entorno := cargarEntornoResolucionMotivosRRHHM23(t)
	pool, resolutor := entorno.nuevoResolutor(t)
	instante := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	exigirResolucionM23(
		t, resolutor, instante, referenciaCuadroM23(), referenciaDetalleM23(),
	)
	probarAislamientoDirectoResolucionMotivosRRHHM23(t, entorno)
	probarSearchPathHostilResolucionMotivosRRHHM23(t, entorno, instante)
	probarReconexionResolucionMotivosRRHHM23(
		t, entorno, pool, resolutor, instante,
	)

	login := pgx.Identifier{entorno.login}.Sanitize()
	probarDerivaResolucionMotivosRRHHM23(
		t, entorno, resolutor, instante,
		"GRANT vec_m23_autoridad_extra TO "+login+
			" WITH ADMIN FALSE, INHERIT TRUE, SET FALSE",
		"REVOKE vec_m23_autoridad_extra FROM "+login,
		false,
	)
	probarDerivaResolucionMotivosRRHHM23(
		t, entorno, resolutor, instante,
		"GRANT SELECT ON TABLE "+
			"vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 TO "+login,
		"REVOKE SELECT ON TABLE "+
			"vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 FROM "+login,
		true,
	)
	probarDerivaResolucionMotivosRRHHM23(
		t, entorno, resolutor, instante,
		"REVOKE EXECUTE ON FUNCTION "+
			"vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz) "+
			"FROM vec_autorizacion_motivos_rrhh_resolutor",
		"GRANT EXECUTE ON FUNCTION "+
			"vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz) "+
			"TO vec_autorizacion_motivos_rrhh_resolutor",
		true,
	)

	pool.Cerrar()
	_, resolutorReconectado := entorno.nuevoResolutor(t)
	exigirResolucionM23(
		t, resolutorReconectado, instante,
		referenciaCuadroM23(), referenciaDetalleM23(),
	)
}

func probarReconexionResolucionMotivosRRHHM23(
	t *testing.T,
	entorno *entornoResolucionMotivosRRHHM23,
	pool *PoolResolucionMotivosRRHHPostgreSQL,
	resolutor *ResolutorMotivosRRHHPostgreSQL,
	instante time.Time,
) {
	t.Helper()
	conexion, err := pool.adquirirOperacion(entorno.ctx)
	if err != nil {
		t.Fatal("no se pudo fijar una conexión física M2.3")
	}
	liberada := false
	defer func() {
		if !liberada {
			conexion.Liberar()
		}
	}()
	var terminadas int64
	if err = entorno.administrador.QueryRow(
		entorno.ctx,
		`WITH objetivos AS MATERIALIZED (
			SELECT pid FROM pg_stat_activity
			WHERE usename=$1 AND pid<>pg_backend_pid()
		)
		SELECT count(*)::bigint FROM objetivos
		WHERE pg_terminate_backend(pid)`,
		entorno.login,
	).Scan(&terminadas); err != nil {
		t.Fatal("falló la terminación administrativa M2.3")
	}
	if terminadas < 1 {
		t.Fatal("no se encontró la conexión física M2.3 fijada")
	}
	conexion.Liberar()
	liberada = true
	for {
		ctxIntento, cancelar := context.WithTimeout(entorno.ctx, 3*time.Second)
		referencia, err := resolutor.ResolverMotivoCuadroRRHH(
			ctxIntento, instante,
		)
		cancelar()
		switch {
		case err == nil && referencia == referenciaCuadroM23():
			return
		case referencia == (dominiovec.ReferenciaEntradaCatalogo{}) &&
			errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible):
			select {
			case <-entorno.ctx.Done():
				t.Fatal("el pool no se recuperó tras terminar sus conexiones")
			case <-time.After(50 * time.Millisecond):
			}
		default:
			t.Fatalf("la reconexión produjo una referencia no canónica: %+v",
				referencia)
		}
	}
}

func TestIntegracionResolutorMotivosRRHHPostgreSQLReinicioVivo(t *testing.T) {
	entorno := cargarEntornoResolucionMotivosRRHHM23(t)
	barrera := os.Getenv(variableBarreraM23)
	if barrera == "" {
		t.Skip("reinicio vivo M2.3 no solicitado")
	}
	_, resolutor := entorno.nuevoResolutor(t)
	instante := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	exigirResolucionM23(
		t, resolutor, instante,
		referenciaCuadroM23(), referenciaDetalleM23(),
	)
	if err := os.WriteFile(
		filepath.Join(barrera, "listo"), []byte("listo\n"), 0o600,
	); err != nil {
		t.Fatal("no se pudo publicar la barrera de reinicio M2.3")
	}
	for {
		if _, err := os.Stat(filepath.Join(barrera, "continuar")); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatal("falló la barrera de continuación M2.3")
		}
		select {
		case <-entorno.ctx.Done():
			t.Fatal("el runner no completó el reinicio M2.3")
		case <-time.After(20 * time.Millisecond):
		}
	}
	recuperada := false
	for !recuperada {
		referencia, err := resolutor.ResolverMotivoCuadroRRHH(
			context.Background(), instante,
		)
		switch {
		case err == nil && referencia == referenciaCuadroM23():
			recuperada = true
		case referencia == (dominiovec.ReferenciaEntradaCatalogo{}) &&
			errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible):
			select {
			case <-entorno.ctx.Done():
				t.Fatal("el mismo pool no se recuperó tras el reinicio")
			case <-time.After(50 * time.Millisecond):
			}
		default:
			t.Fatalf("el reinicio produjo una referencia no canónica: %+v",
				referencia)
		}
	}
	detalle, err := resolutor.ResolverMotivoDetalleRRHH(
		context.Background(), instante,
	)
	if err != nil || detalle != referenciaDetalleM23() {
		t.Fatal("el mismo pool no reacreditó detalle tras el reinicio")
	}
}

func TestIntegracionResolutorMotivosRRHHPostgreSQLRetiradas(t *testing.T) {
	entorno := cargarEntornoResolucionMotivosRRHHM23(t)
	_, resolutor := entorno.nuevoResolutor(t)
	instante := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	exigirResolucionM23(
		t, resolutor, instante, referenciaCuadroM23(), referenciaDetalleM23(),
	)
	probarRetiradaConcurrenteDetalleM23(t, entorno, resolutor, instante)
	probarCaducidadDetalleM23(t, entorno, resolutor)
	if !consultarBooleanoM23(
		t, entorno,
		`SELECT vec_autorizacion.retirar_motivos_autorizacion_v2(
			$1,$2,$3,$4,$5,$6,$7,$8)`,
		"evento_33333333333333333333333333333333", int64(3),
		strings.Repeat("d", 64), catalogoM23, 1,
		strings.Repeat("2", 64), strings.Repeat("b", 64),
		time.Now().UTC().Truncate(time.Microsecond),
	) {
		t.Fatal("la retirada V2 sintética fue rechazada")
	}
	exigirNoDisponibleM23(t, func() (
		dominiovec.ReferenciaEntradaCatalogo, error,
	) {
		return resolutor.ResolverMotivoCuadroRRHH(context.Background(), instante)
	})
}

func probarRetiradaConcurrenteDetalleM23(
	t *testing.T,
	entorno *entornoResolucionMotivosRRHHM23,
	resolutor *ResolutorMotivosRRHHPostgreSQL,
	instante time.Time,
) {
	t.Helper()
	bloqueador, err := entorno.administrador.Acquire(entorno.ctx)
	if err != nil {
		t.Fatal("no se pudo abrir la barrera causal M2.3")
	}
	defer bloqueador.Release()
	transaccion, err := bloqueador.Begin(entorno.ctx)
	if err != nil {
		t.Fatal("no se pudo iniciar la barrera causal M2.3")
	}
	cerrada := false
	defer func() {
		if !cerrada {
			_ = transaccion.Rollback(context.Background())
		}
	}()
	if _, err = transaccion.Exec(
		entorno.ctx,
		`SELECT 1
		   FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_checkpoint_v1
		  WHERE clase_consulta='detalle'
		  FOR UPDATE`,
	); err != nil {
		t.Fatal("no se pudo fijar el checkpoint de detalle M2.3")
	}
	canalResolucion := make(chan bool, 1)
	go func() {
		referencia, errResolucion := resolutor.ResolverMotivoDetalleRRHH(
			entorno.ctx, instante,
		)
		canalResolucion <- errResolucion == nil &&
			referencia == referenciaDetalleM23()
	}()
	pidResolutor := esperarPIDBloqueadoM23(
		t, entorno, entorno.login, "",
	)
	conexionProyector, err := entorno.proyector.Acquire(entorno.ctx)
	if err != nil {
		t.Fatal("no se pudo abrir la retirada causal M2.3")
	}
	defer conexionProyector.Release()
	const aplicacionRetirada = "vec_m23_retirada_detalle"
	if _, err = conexionProyector.Exec(
		entorno.ctx, "SET application_name='"+aplicacionRetirada+"'",
	); err != nil {
		t.Fatal("no se pudo identificar la retirada causal M2.3")
	}
	canalRetirada := make(chan bool, 1)
	go func() {
		var aceptada bool
		errRetirada := conexionProyector.QueryRow(
			entorno.ctx,
			`SELECT vec_autorizacion.retirar_vinculacion_motivo_detalle_rrhh_v1(
				$1,$2,$3,$4,$5,$6)`,
			"evento_vinculacion_motivo_rrhh_22222222222222222222222222222223",
			strings.Repeat("7", 64), int64(1),
			"publicacion_motivo_rrhh_22222222222222222222222222222222",
			strings.Repeat("6", 64),
			time.Now().UTC().Truncate(time.Microsecond),
		).Scan(&aceptada)
		canalRetirada <- errRetirada == nil && aceptada
	}()
	pidRetirada := esperarPIDBloqueadoM23(
		t, entorno, "", aplicacionRetirada,
	)
	var ordenCausal bool
	if err = entorno.administrador.QueryRow(
		entorno.ctx,
		"SELECT $1::integer=ANY(pg_blocking_pids($2::integer))",
		pidResolutor, pidRetirada,
	).Scan(&ordenCausal); err != nil || !ordenCausal {
		t.Fatal("la retirada no quedó encolada tras la resolución M2.3")
	}
	if err = transaccion.Commit(entorno.ctx); err != nil {
		t.Fatal("no se pudo liberar la barrera causal M2.3")
	}
	cerrada = true

	select {
	case correcta := <-canalResolucion:
		if !correcta {
			t.Fatal("la resolución anterior a la retirada no fue positiva")
		}
	case <-entorno.ctx.Done():
		t.Fatal("la resolución causal M2.3 no concluyó")
	}
	select {
	case aceptada := <-canalRetirada:
		if !aceptada {
			t.Fatal("la retirada causal M2.3 no fue confirmada")
		}
	case <-entorno.ctx.Done():
		t.Fatal("la retirada causal M2.3 no concluyó")
	}
	exigirNoDisponibleM23(t, func() (
		dominiovec.ReferenciaEntradaCatalogo, error,
	) {
		return resolutor.ResolverMotivoDetalleRRHH(entorno.ctx, instante)
	})
}

func esperarPIDBloqueadoM23(
	t *testing.T,
	entorno *entornoResolucionMotivosRRHHM23,
	login string,
	aplicacion string,
) int32 {
	t.Helper()
	for {
		var pid int32
		err := entorno.administrador.QueryRow(
			entorno.ctx,
			`SELECT COALESCE((
				SELECT pid FROM pg_stat_activity
				WHERE wait_event_type='Lock'
				  AND ($1='' OR usename=$1)
				  AND ($2='' OR application_name=$2)
				ORDER BY query_start DESC LIMIT 1
			),0)::integer`,
			login, aplicacion,
		).Scan(&pid)
		if err != nil {
			t.Fatal("no se pudo observar la barrera causal M2.3")
		}
		if pid != 0 {
			return pid
		}
		select {
		case <-entorno.ctx.Done():
			t.Fatal("no se observó el bloqueo causal M2.3")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func probarAislamientoDirectoResolucionMotivosRRHHM23(
	t *testing.T,
	entorno *entornoResolucionMotivosRRHHM23,
) {
	t.Helper()
	conexion, err := pgx.Connect(entorno.ctx, entorno.dsn)
	if err != nil {
		t.Fatal("no se pudo abrir la conexión de aislamiento M2.3")
	}
	defer func() { _ = conexion.Close(context.Background()) }()
	for _, consulta := range []string{
		"SELECT * FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1",
		"INSERT INTO vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1" +
			"(clase_consulta,publicacion_version) VALUES('cuadro',99)",
		"UPDATE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1 " +
			"SET catalogo_id=catalogo_id",
		"DELETE FROM vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1",
		"TRUNCATE vec_autorizacion.vinculacion_motivo_consulta_rrhh_v1",
		"SET ROLE vec_autorizacion_propietario",
		"CREATE TEMP TABLE vec_m23_temporal_hostil(i integer)",
	} {
		_, err = conexion.Exec(entorno.ctx, consulta)
		var errorPostgreSQL *pgconn.PgError
		if !errors.As(err, &errorPostgreSQL) ||
			errorPostgreSQL.Code != "42501" {
			t.Fatal("el LOGIN resolutor superó el aislamiento directo M2.3")
		}
	}
}

func probarSearchPathHostilResolucionMotivosRRHHM23(
	t *testing.T,
	entorno *entornoResolucionMotivosRRHHM23,
	instante time.Time,
) {
	t.Helper()
	pool, resolutor := entorno.nuevoResolutorConDSN(t, entorno.dsnUnico)
	conexion, err := pool.adquirirOperacion(entorno.ctx)
	if err != nil {
		t.Fatal("no se pudo aislar la sesión hostil M2.3")
	}
	liberada := false
	defer func() {
		if !liberada {
			conexion.Liberar()
		}
	}()
	login := pgx.Identifier{entorno.login}.Sanitize()
	_, err = entorno.administrador.Exec(entorno.ctx, `
		CREATE FUNCTION vec_autorizacion.vec_m23_preparar_homonimo()
		RETURNS boolean LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog
		AS $funcion$
		BEGIN
		  EXECUTE $ddl$
		    CREATE FUNCTION pg_temp.resolver_motivo_cuadro_rrhh_v1(timestamptz)
		    RETURNS TABLE(catalogo_id text,catalogo_version integer,
		      catalogo_huella_sha256 text,entrada_clave text)
		    LANGUAGE sql AS $cuerpo$ SELECT 'hostil',1,repeat('f',64),
		      'motivo_ffffffffffffffffffffffffffffffff' $cuerpo$
		  $ddl$;
		  RETURN true;
		END $funcion$;
		REVOKE ALL ON FUNCTION vec_autorizacion.vec_m23_preparar_homonimo()
		  FROM PUBLIC;
		GRANT EXECUTE ON FUNCTION vec_autorizacion.vec_m23_preparar_homonimo()
		  TO `+login)
	if err != nil {
		t.Fatal("no se pudo preparar el homónimo temporal M2.3")
	}
	defer func() {
		_, _ = entorno.administrador.Exec(
			context.Background(),
			"DROP FUNCTION IF EXISTS vec_autorizacion.vec_m23_preparar_homonimo()",
		)
	}()
	var preparada bool
	var pidAntes int32
	if err = conexion.QueryRow(
		entorno.ctx,
		"SELECT vec_autorizacion.vec_m23_preparar_homonimo()",
	).Scan(&preparada); err != nil || !preparada {
		t.Fatal("no se pudo crear el homónimo temporal M2.3")
	}
	if err = conexion.QueryRow(
		entorno.ctx,
		"SELECT pg_backend_pid(),set_config('search_path','pg_temp,public',false)",
	).Scan(&pidAntes, new(string)); err != nil {
		t.Fatal("no se pudo fijar el search_path hostil M2.3")
	}
	if _, err = entorno.administrador.Exec(
		entorno.ctx,
		"DROP FUNCTION vec_autorizacion.vec_m23_preparar_homonimo()",
	); err != nil {
		t.Fatal("no se pudo retirar el helper temporal M2.3")
	}
	conexion.Liberar()
	liberada = true
	exigirResolucionM23(
		t, resolutor, instante, referenciaCuadroM23(), referenciaDetalleM23(),
	)
	conexion, err = pool.adquirirOperacion(entorno.ctx)
	if err != nil {
		t.Fatal("la sesión hostil M2.3 no permaneció disponible")
	}
	defer conexion.Liberar()
	var pidDespues int32
	if err = conexion.QueryRow(
		entorno.ctx, "SELECT pg_backend_pid()",
	).Scan(&pidDespues); err != nil || pidDespues != pidAntes {
		t.Fatal("la prueba hostil M2.3 no usó la misma conexión física")
	}
}

func probarDerivaResolucionMotivosRRHHM23(
	t *testing.T,
	entorno *entornoResolucionMotivosRRHHM23,
	resolutor *ResolutorMotivosRRHHPostgreSQL,
	instante time.Time,
	mutacion string,
	restauracion string,
	comoPropietario bool,
) {
	t.Helper()
	ejecutarMutacionM23(t, entorno, mutacion, comoPropietario)
	restaurada := false
	defer func() {
		if !restaurada {
			ejecutarMutacionM23(t, entorno, restauracion, comoPropietario)
		}
	}()
	exigirNoDisponibleM23(t, func() (
		dominiovec.ReferenciaEntradaCatalogo, error,
	) {
		return resolutor.ResolverMotivoCuadroRRHH(context.Background(), instante)
	})
	ejecutarMutacionM23(t, entorno, restauracion, comoPropietario)
	restaurada = true
	exigirResolucionM23(
		t, resolutor, instante, referenciaCuadroM23(), referenciaDetalleM23(),
	)
}

func ejecutarMutacionM23(
	t *testing.T,
	entorno *entornoResolucionMotivosRRHHM23,
	consulta string,
	comoPropietario bool,
) {
	t.Helper()
	if !comoPropietario {
		if _, err := entorno.administrador.Exec(entorno.ctx, consulta); err != nil {
			t.Fatal("falló una mutación administrativa sintética M2.3")
		}
		return
	}
	tx, err := entorno.administrador.Begin(entorno.ctx)
	if err != nil {
		t.Fatal("no se pudo abrir una mutación propietaria M2.3")
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err = tx.Exec(
		entorno.ctx, "SET LOCAL ROLE vec_autorizacion_propietario",
	); err != nil {
		t.Fatal("no se pudo aislar la mutación propietaria M2.3")
	}
	if _, err = tx.Exec(entorno.ctx, consulta); err != nil {
		t.Fatal("falló una mutación propietaria sintética M2.3")
	}
	if err = tx.Commit(entorno.ctx); err != nil {
		t.Fatal("no se pudo confirmar la mutación propietaria M2.3")
	}
}

func consultarBooleanoM23(
	t *testing.T,
	entorno *entornoResolucionMotivosRRHHM23,
	consulta string,
	argumentos ...any,
) bool {
	t.Helper()
	var resultado bool
	if err := entorno.proyector.QueryRow(
		entorno.ctx, consulta, argumentos...,
	).Scan(&resultado); err != nil {
		t.Fatal("falló una operación gobernada sintética M2.3")
	}
	return resultado
}

func probarCaducidadDetalleM23(
	t *testing.T,
	entorno *entornoResolucionMotivosRRHHM23,
	resolutor *ResolutorMotivosRRHHPostgreSQL,
) {
	t.Helper()
	var ahora time.Time
	if err := entorno.administrador.QueryRow(
		entorno.ctx, "SELECT clock_timestamp()",
	).Scan(&ahora); err != nil {
		t.Fatal("no se pudo obtener el reloj PostgreSQL M2.3")
	}
	ahora = ahora.UTC().Truncate(time.Microsecond)
	caduca := ahora.Add(5 * time.Second)
	formato := "2006-01-02T15:04:05.000000Z"
	entradas, err := json.Marshal([]map[string]any{{
		"clave":         claveDetalleCaducableM23,
		"vigente_desde": ahora.Add(-time.Minute).Format(formato),
		"vigente_hasta": caduca.Format(formato),
	}})
	if err != nil {
		t.Fatal("no se pudo construir el catálogo sintético caducable")
	}
	if !consultarBooleanoM23(
		t, entorno,
		`SELECT vec_autorizacion.publicar_motivos_autorizacion_v2(
			$1,$2,$3,$4,$5,$6,$7,$8::jsonb)`,
		"evento_22222222222222222222222222222222", int64(2),
		strings.Repeat("a", 64), catalogoM23, 2,
		strings.Repeat("c", 64), ahora.Add(-time.Minute), string(entradas),
	) {
		t.Fatal("el catálogo sintético caducable fue rechazado")
	}
	if !consultarBooleanoM23(
		t, entorno,
		`SELECT vec_autorizacion.publicar_vinculacion_motivo_detalle_rrhh_v1(
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		"evento_vinculacion_motivo_rrhh_22222222222222222222222222222224",
		strings.Repeat("8", 64), int64(2),
		"publicacion_motivo_rrhh_22222222222222222222222222222224",
		strings.Repeat("9", 64), catalogoM23, 2,
		strings.Repeat("c", 64), claveDetalleCaducableM23, ahora,
	) {
		t.Fatal("la vinculación sintética caducable fue rechazada")
	}
	instante := ahora
	esperada := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: catalogoM23, CatalogoVersion: 2,
		CatalogoHuellaSHA256: strings.Repeat("c", 64),
		EntradaClave:         claveDetalleCaducableM23,
	}
	obtenida, err := resolutor.ResolverMotivoDetalleRRHH(
		context.Background(), instante,
	)
	if err != nil || obtenida != esperada {
		t.Fatal("la referencia caducable no estuvo disponible")
	}
	for {
		var caducada bool
		if err = entorno.administrador.QueryRow(
			entorno.ctx, "SELECT clock_timestamp()>=$1", caduca,
		).Scan(&caducada); err != nil {
			t.Fatal("no se pudo observar la caducidad PostgreSQL M2.3")
		}
		if caducada {
			break
		}
		select {
		case <-entorno.ctx.Done():
			t.Fatal("la vigencia sintética M2.3 no caducó")
		case <-time.After(20 * time.Millisecond):
		}
	}
	exigirNoDisponibleM23(t, func() (
		dominiovec.ReferenciaEntradaCatalogo, error,
	) {
		return resolutor.ResolverMotivoDetalleRRHH(
			context.Background(), instante,
		)
	})
}
