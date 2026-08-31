package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	rolAjenoSeleccionO6Integracion = "vec_ct_o6_ajeno_prueba"
	sesionesSeleccionO6Integracion = 8
)

// TestEjecucionesSeleccionO6PostgreSQLGoATerminal se prepara para el runner
// PostgreSQL 18.4. Las puertas Go ordinarias lo compilan, pero nunca abren una
// conexion si el runner no declara expresamente una base desechable.
func TestEjecucionesSeleccionO6PostgreSQLGoATerminal(t *testing.T) {
	if os.Getenv("VEC_CT_O6_INTEGRACION_PG") != "SI" ||
		os.Getenv("VEC_CT_O6_BD_DESECHABLE") != "SI" {
		t.Skip("integracion PostgreSQL O6 no solicitada")
	}
	if os.Getenv("VEC_CT_O6_SESIONES_CONCURRENTES") != "8" {
		t.Fatal("el runner no fijo las ocho sesiones concurrentes exigidas")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	admin, err := pgxpool.New(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	var version string
	if err = admin.QueryRow(ctx, "SELECT current_setting('server_version_num')").Scan(&version); err != nil || version != "180004" {
		t.Fatalf("PostgreSQL no fijado en 18.4: version=%q err=%v", version, err)
	}
	prepararRolAjenoSeleccionO6Integracion(t, ctx, admin)
	probarACLSeleccionO6Integracion(t, ctx, admin)

	solicitud, recibo, artefacto := materialesEjecucionSeleccionO6Prueba(t)
	borrarEjecucionSeleccionO6Integracion(t, context.Background(), admin, solicitud.ClaveIdempotencia)
	probarConcurrenciaSeleccionO6Integracion(t, ctx, admin, solicitud)
	probarHuellasNulasSolicitudSeleccionO6Integracion(t, ctx, admin, solicitud)
	if os.Getenv("VEC_CT_O6_CONSERVAR_HISTORIA") != "SI" {
		t.Cleanup(func() {
			borrarEjecucionSeleccionO6Integracion(t, context.Background(), admin, solicitud.ClaveIdempotencia)
		})
	}
	poolEjecutor := nuevoPoolEjecutorSeleccionO6Integracion(t, ctx)
	adaptador, err := NuevasEjecucionesSeleccionLlamamientoPostgreSQL(poolEjecutor)
	if err != nil {
		t.Fatal(err)
	}
	probarHuellaNoAutoritativaSeleccionO6Integracion(t, ctx, poolEjecutor, solicitud)
	ausente, confirmada, err := adaptador.ResolverTerminal(ctx, solicitud.ClaveIdempotencia)
	if err != nil || confirmada || ausente != (ports.EstadoEjecucionSeleccionLlamamiento{}) {
		t.Fatalf("la huella falsa dejo estado: estado=%#v confirmada=%v err=%v",
			ausente, confirmada, err)
	}
	estado, err := adaptador.Reservar(ctx, solicitud)
	if err != nil || estado.Situacion != ports.EjecucionSeleccionLlamamientoPropietaria ||
		!referenciaReservaSeleccionO6Valida(estado.ReservaRef) {
		t.Fatalf("reserva real no propietaria: estado=%#v err=%v", estado, err)
	}
	reserva := ports.ReservaEjecucionSeleccionLlamamiento{
		Solicitud: solicitud, ReservaRef: estado.ReservaRef,
	}
	if err = adaptador.LiberarAntesDeEfectos(ctx, reserva); err != nil {
		t.Fatal(err)
	}
	ausente, confirmada, err = adaptador.ResolverTerminal(ctx, solicitud.ClaveIdempotencia)
	if err != nil || confirmada || ausente != (ports.EstadoEjecucionSeleccionLlamamiento{}) {
		t.Fatalf("liberacion previa dejo estado: estado=%#v confirmada=%v err=%v",
			ausente, confirmada, err)
	}
	estado, err = adaptador.Reservar(ctx, solicitud)
	if err != nil || estado.Situacion != ports.EjecucionSeleccionLlamamientoPropietaria {
		t.Fatalf("segunda reserva no propietaria: estado=%#v err=%v", estado, err)
	}
	reserva.ReservaRef = estado.ReservaRef
	if err = adaptador.AbrirVentanaEfecto(ctx, reserva,
		ports.EfectoPrepararOrdenSeleccionLlamamiento); err != nil {
		t.Fatal(err)
	}
	if err = adaptador.AbrirVentanaEfecto(ctx, reserva,
		ports.EfectoPrepararOrdenSeleccionLlamamiento); err == nil {
		t.Fatal("la ventana repetida fue aceptada")
	}
	if err = adaptador.MarcarIndeterminada(ctx, reserva,
		ports.EfectoPrepararOrdenSeleccionLlamamiento); err != nil {
		t.Fatal(err)
	}
	indeterminada, confirmada, err := adaptador.ResolverTerminal(ctx, solicitud.ClaveIdempotencia)
	if err != nil || confirmada ||
		indeterminada.Situacion != ports.EjecucionSeleccionLlamamientoIndeterminada ||
		indeterminada.EfectoPosible != ports.EfectoPrepararOrdenSeleccionLlamamiento {
		t.Fatalf("terminal indeterminado divergente: estado=%#v confirmada=%v err=%v",
			indeterminada, confirmada, err)
	}
	borrarEjecucionSeleccionO6Integracion(t, context.Background(), admin, solicitud.ClaveIdempotencia)
	estado, err = adaptador.Reservar(ctx, solicitud)
	if err != nil || estado.Situacion != ports.EjecucionSeleccionLlamamientoPropietaria {
		t.Fatalf("reserva final no propietaria: estado=%#v err=%v", estado, err)
	}
	reserva.ReservaRef = estado.ReservaRef
	if err = adaptador.AbrirVentanaEfecto(ctx, reserva,
		ports.EfectoPrepararOrdenSeleccionLlamamiento); err != nil {
		t.Fatal(err)
	}
	if err = adaptador.AbrirVentanaEfecto(ctx, reserva,
		ports.EfectoSolicitarSeleccionLlamamiento); err != nil {
		t.Fatal(err)
	}
	probarEncuadresNoCanonicosSeleccionO6Integracion(
		t, ctx, admin, poolEjecutor, reserva, recibo, artefacto,
	)
	if err = adaptador.Confirmar(ctx, reserva, recibo, artefacto); err != nil {
		t.Fatal(err)
	}
	poolEjecutor.Close()

	poolReinicio := nuevoPoolEjecutorSeleccionO6Integracion(t, ctx)
	defer poolReinicio.Close()
	reiniciado, err := NuevasEjecucionesSeleccionLlamamientoPostgreSQL(poolReinicio)
	if err != nil {
		t.Fatal(err)
	}
	terminal, confirmada, err := reiniciado.ResolverTerminal(ctx, solicitud.ClaveIdempotencia)
	if err != nil || !confirmada || terminal.Solicitud != solicitud ||
		terminal.ReciboConfirmado != recibo || terminal.ArtefactoConfirmado != artefacto {
		t.Fatalf("terminal Go-PG-Go divergente: estado=%#v confirmada=%v err=%v",
			terminal, confirmada, err)
	}
	replay, err := reiniciado.Reservar(ctx, solicitud)
	if err != nil || replay.Situacion != ports.EjecucionSeleccionLlamamientoConfirmada ||
		replay.ReciboConfirmado != recibo || replay.ArtefactoConfirmado != artefacto {
		t.Fatalf("replay terminal divergente: estado=%#v err=%v", replay, err)
	}
	if err = reiniciado.Confirmar(ctx, reserva, recibo, artefacto); err != nil {
		t.Fatalf("replay exacto de confirmacion rechazado: %v", err)
	}
}

func probarHuellaNoAutoritativaSeleccionO6Integracion(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
) {
	t.Helper()
	var vista map[string]any
	if err := json.Unmarshal(debeCodificarSolicitudSeleccionO6Prueba(t, solicitud), &vista); err != nil {
		t.Fatal(err)
	}
	vista["huella_semantica"] = strings.Repeat("0", 64)
	contenido := debeJSONSeleccionO6Prueba(t, vista)
	var fila filaEjecucionSeleccionO6
	err := pool.QueryRow(ctx, `SELECT situacion, solicitud_json, reserva_ref,
		efecto, recibo_json, artefacto_json FROM `+funcionReservarSeleccionO6+`(
		$1::uuid,$2::text,$3::text)`, solicitud.ClaveIdempotencia,
		strings.Repeat("0", 64), contenido).Scan(
		&fila.Situacion, &fila.SolicitudJSON, &fila.ReservaRef,
		&fila.Efecto, &fila.ReciboJSON, &fila.ArtefactoJSON,
	)
	var falloPG *pgconn.PgError
	if !errors.As(err, &falloPG) || falloPG.Code != "22023" {
		t.Fatalf("huella no autoritativa aceptada: fila=%#v err=%v", fila, err)
	}
}

func probarConcurrenciaSeleccionO6Integracion(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
) {
	t.Helper()
	poolPropietario := nuevoPoolEjecutorSeleccionO6Integracion(t, ctx)
	propietario, err := NuevasEjecucionesSeleccionLlamamientoPostgreSQL(poolPropietario)
	if err != nil {
		t.Fatal(err)
	}
	estadoPropietario, err := propietario.Reservar(ctx, solicitud)
	if err != nil || estadoPropietario.Situacion != ports.EjecucionSeleccionLlamamientoPropietaria {
		t.Fatalf("no se preparo propietario para la contencion: estado=%#v err=%v",
			estadoPropietario, err)
	}
	poolPropietario.Close()
	type resultado struct {
		estado ports.EstadoEjecucionSeleccionLlamamiento
		err    error
	}
	adaptadores := make([]*EjecucionesSeleccionLlamamientoPostgreSQL, sesionesSeleccionO6Integracion)
	pools := make([]*pgxpool.Pool, sesionesSeleccionO6Integracion)
	pids := make(map[int32]struct{}, sesionesSeleccionO6Integracion)
	for indice := range sesionesSeleccionO6Integracion {
		pools[indice] = nuevoPoolEjecutorSeleccionO6Integracion(t, ctx)
		defer pools[indice].Close()
		var pid int32
		if err := pools[indice].QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
			t.Fatal(err)
		}
		pids[pid] = struct{}{}
		adaptador, err := NuevasEjecucionesSeleccionLlamamientoPostgreSQL(pools[indice])
		if err != nil {
			t.Fatal(err)
		}
		adaptadores[indice] = adaptador
	}
	if len(pids) != sesionesSeleccionO6Integracion {
		t.Fatalf("la carrera no preparo ocho sesiones distintas: %d", len(pids))
	}
	inicio := make(chan struct{})
	resultados := make(chan resultado, sesionesSeleccionO6Integracion)
	var grupo sync.WaitGroup
	for _, adaptador := range adaptadores {
		grupo.Add(1)
		go func(adaptador *EjecucionesSeleccionLlamamientoPostgreSQL) {
			defer grupo.Done()
			<-inicio
			estado, err := adaptador.Reservar(ctx, solicitud)
			resultados <- resultado{estado: estado, err: err}
		}(adaptador)
	}
	close(inicio)
	grupo.Wait()
	close(resultados)
	ocupadas := 0
	for resultado := range resultados {
		if resultado.err != nil {
			t.Fatalf("sesion concurrente fallo: %v", resultado.err)
		}
		switch resultado.estado.Situacion {
		case ports.EjecucionSeleccionLlamamientoOcupada:
			ocupadas++
		default:
			t.Fatalf("estado concurrente inesperado: %#v", resultado.estado)
		}
	}
	if ocupadas != sesionesSeleccionO6Integracion {
		t.Fatalf("contencion de ocho sesiones divergente: ocupadas=%d", ocupadas)
	}
	borrarEjecucionSeleccionO6Integracion(t, context.Background(), admin, solicitud.ClaveIdempotencia)
}

func probarHuellasNulasSolicitudSeleccionO6Integracion(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	solicitud ports.SolicitudReservaEjecucionSeleccionLlamamiento,
) {
	t.Helper()
	canonico := debeCodificarSolicitudSeleccionO6Prueba(t, solicitud)
	pool := nuevoPoolEjecutorSeleccionO6Integracion(t, ctx)
	defer pool.Close()
	for _, caso := range []struct{ campo, huella string }{
		{"accion_orden", solicitud.AccionOrden.HuellaSHA256},
		{"finalidad", solicitud.Finalidad.HuellaSHA256},
		{"necesidad", solicitud.Necesidad.HuellaSHA256},
		{"bolsa", solicitud.Bolsa.HuellaSHA256},
		{"politica", solicitud.Politica.HuellaSHA256},
	} {
		t.Run("huella nula solicitud "+caso.campo, func(t *testing.T) {
			mutada := reemplazarUnaVezSeleccionO6Integracion(t, string(canonico),
				`"huella_sha256":"`+caso.huella+`"`,
				`"huella_sha256":"`+strings.Repeat("0", 64)+`"`)
			_, err := pool.Exec(ctx, `SELECT * FROM `+funcionReservarSeleccionO6+`(
				$1::uuid,$2::text,$3::text)`, solicitud.ClaveIdempotencia,
				solicitud.HuellaSemantica, mutada)
			var falloPG *pgconn.PgError
			if !errors.As(err, &falloPG) || falloPG.Code != "22023" {
				t.Fatalf("la fachada acepto huella nula %s: %v", caso.campo, err)
			}
			var rechazada bool
			if err := admin.QueryRow(ctx, `SELECT
				vec_contratacion_temporal.huella_solicitud_seleccion_llamamiento_o6_v1(
				$1::jsonb) IS NULL`, mutada).Scan(&rechazada); err != nil || !rechazada {
				t.Fatalf("el recomputador acepto huella nula %s: rechazada=%v err=%v",
					caso.campo, rechazada, err)
			}
		})
	}
}

func nuevoPoolEjecutorSeleccionO6Integracion(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	configuracion, err := pgxpool.ParseConfig("")
	if err != nil {
		t.Fatal(err)
	}
	configuracion.MaxConns = 1
	configuracion.AfterConnect = func(ctx context.Context, conexion *pgx.Conn) error {
		_, err := conexion.Exec(ctx,
			"SET SESSION AUTHORIZATION vec_contratacion_temporal_ejecutor")
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil {
		t.Fatal(err)
	}
	return pool
}

func prepararRolAjenoSeleccionO6Integracion(
	t *testing.T, ctx context.Context, admin *pgxpool.Pool,
) {
	t.Helper()
	var existe bool
	if err := admin.QueryRow(ctx,
		"SELECT to_regrole($1) IS NOT NULL", rolAjenoSeleccionO6Integracion,
	).Scan(&existe); err != nil || existe {
		t.Fatalf("rol ajeno de prueba no exclusivo: existe=%v err=%v", existe, err)
	}
	if _, err := admin.Exec(ctx, "CREATE ROLE "+rolAjenoSeleccionO6Integracion+" NOLOGIN"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctxLimpieza, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		if _, err := admin.Exec(ctxLimpieza, "DROP ROLE "+rolAjenoSeleccionO6Integracion); err != nil {
			t.Errorf("no se retiro el rol ajeno de prueba: %v", err)
		}
	})
}

func probarACLSeleccionO6Integracion(t *testing.T, ctx context.Context, admin *pgxpool.Pool) {
	t.Helper()
	principales := map[string]bool{
		"resolver_terminal_seleccion_llamamiento_o6_v1":    true,
		"reservar_seleccion_llamamiento_o6_v1":             true,
		"abrir_ventana_seleccion_llamamiento_o6_v1":        true,
		"marcar_indeterminada_seleccion_llamamiento_o6_v1": true,
		"liberar_seleccion_llamamiento_o6_v1":              true,
		"confirmar_seleccion_llamamiento_o6_v1":            true,
		"consultar_seleccion_llamamiento_o6_v1":            true,
	}
	filas, err := admin.Query(ctx, `
		SELECT funcion.proname, funcion.prosecdef, propietario.rolname,
		       pg_catalog.coalesce(funcion.proconfig, ARRAY[]::text[]),
		       pg_catalog.has_function_privilege(
		           'vec_contratacion_temporal_ejecutor', funcion.oid, 'EXECUTE'),
		       pg_catalog.has_function_privilege($1, funcion.oid, 'EXECUTE'),
		       EXISTS (
		           SELECT 1 FROM pg_catalog.aclexplode(pg_catalog.coalesce(
		               funcion.proacl, pg_catalog.acldefault('f', funcion.proowner)
		           )) permiso
		           WHERE permiso.grantee = 0 AND permiso.privilege_type = 'EXECUTE'
		       )
		  FROM pg_catalog.pg_proc funcion
		  JOIN pg_catalog.pg_roles propietario ON propietario.oid = funcion.proowner
		 WHERE funcion.pronamespace = 'vec_contratacion_temporal'::regnamespace
		   AND funcion.proname LIKE '%seleccion_llamamiento_o6_v1'`,
		rolAjenoSeleccionO6Integracion)
	if err != nil {
		t.Fatal(err)
	}
	defer filas.Close()
	conteo := 0
	for filas.Next() {
		var nombre, propietario string
		var definidora, ejecutor, ajeno, publico bool
		var configuracion []string
		if err = filas.Scan(&nombre, &definidora, &propietario, &configuracion,
			&ejecutor, &ajeno, &publico); err != nil {
			t.Fatal(err)
		}
		conteo++
		principal := principales[nombre]
		if propietario != "vec_contratacion_temporal_propietario" || publico || ajeno ||
			principal != definidora || principal != ejecutor ||
			(principal && (!contieneConfiguracionSeleccionO6(configuracion, "search_path=pg_catalog") ||
				!contieneConfiguracionSeleccionO6(configuracion, "row_security=on"))) {
			t.Fatalf("ACL/SECURITY DEFINER divergente en %s: owner=%s definer=%v ejecutor=%v PUBLIC=%v ajeno=%v config=%v",
				nombre, propietario, definidora, ejecutor, publico, ajeno, configuracion)
		}
	}
	if err = filas.Err(); err != nil || conteo != 21 {
		t.Fatalf("inventario de funciones O6 incompleto: conteo=%d err=%v", conteo, err)
	}
	var tablaEjecutor, tablaAjena, tablaPublica bool
	if err = admin.QueryRow(ctx, `SELECT
		pg_catalog.has_table_privilege('vec_contratacion_temporal_ejecutor',
		 'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6', 'SELECT'),
		pg_catalog.has_table_privilege($1,
		 'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6', 'SELECT'),
		EXISTS (
		    SELECT 1 FROM pg_catalog.pg_class tabla,
		         LATERAL pg_catalog.aclexplode(pg_catalog.coalesce(
		             tabla.relacl, pg_catalog.acldefault('r', tabla.relowner)
		         )) permiso
		    WHERE tabla.oid =
		          'vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6'::regclass
		      AND permiso.grantee = 0
		)`, rolAjenoSeleccionO6Integracion).Scan(
		&tablaEjecutor, &tablaAjena, &tablaPublica,
	); err != nil || tablaEjecutor || tablaAjena || tablaPublica {
		t.Fatalf("ACL de tabla abierta: ejecutor=%v PUBLIC=%v ajeno=%v err=%v",
			tablaEjecutor, tablaPublica, tablaAjena, err)
	}
	conexion, err := pgx.Connect(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	defer conexion.Close(context.Background())
	if _, err = conexion.Exec(ctx, "SET SESSION AUTHORIZATION "+rolAjenoSeleccionO6Integracion); err != nil {
		t.Fatal(err)
	}
	_, err = conexion.Exec(ctx, `SELECT
		vec_contratacion_temporal.resolver_terminal_seleccion_llamamiento_o6_v1(
		'30000000-0000-4000-8000-000000000001'::uuid)`)
	var falloPG *pgconn.PgError
	if !errors.As(err, &falloPG) || falloPG.Code != "42501" {
		t.Fatalf("rol ajeno cruzo la fachada: %v", err)
	}
}

func contieneConfiguracionSeleccionO6(configuracion []string, buscada string) bool {
	for _, valor := range configuracion {
		if valor == buscada {
			return true
		}
	}
	return false
}

func probarEncuadresNoCanonicosSeleccionO6Integracion(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	pool *pgxpool.Pool,
	reserva ports.ReservaEjecucionSeleccionLlamamiento,
	recibo ports.ReciboSolicitudLlamamientoBolsa,
	artefacto ports.ArtefactoProbatorioLlamamientoBolsa,
) {
	t.Helper()
	solicitudJSON := string(debeCodificarSolicitudSeleccionO6Prueba(t, reserva.Solicitud))
	reciboJSON := string(debeJSONSeleccionO6Prueba(t, recibo))
	canonico := string(debeJSONSeleccionO6Prueba(t, artefacto))
	var vista map[string]any
	if err := json.Unmarshal([]byte(canonico), &vista); err != nil {
		t.Fatal(err)
	}
	reordenado := string(debeJSONSeleccionO6Prueba(t, vista))
	duplicado := reemplazarUnaVezSeleccionO6Integracion(t, canonico, `{"esquema":`,
		`{"esquema":"sustituido","esquema":`)
	versionTexto := reemplazarUnaVezSeleccionO6Integracion(t, canonico,
		`"version":1`, `"version":"1"`)
	contratoTexto := reemplazarNSeleccionO6Integracion(t, canonico,
		`"contrato_version":1`, `"contrato_version":"1"`, 2)
	decimalArtefacto := reemplazarUnaVezSeleccionO6Integracion(t, canonico,
		`"total_posiciones_orden":3`, `"total_posiciones_orden":3.0`)
	decimalArtefacto = rehuellarArtefactoSeleccionO6Integracion(
		t, decimalArtefacto, artefacto.HuellaArtefactoSHA256,
	)
	exponenteArtefacto := reemplazarUnaVezSeleccionO6Integracion(t, canonico,
		`"total_posiciones_orden":3`, `"total_posiciones_orden":3e0`)
	exponenteArtefacto = rehuellarArtefactoSeleccionO6Integracion(
		t, exponenteArtefacto, artefacto.HuellaArtefactoSHA256,
	)
	referenciaPropuesta := `"propuesta":{"referencia":"` + recibo.Propuesta.Referencia + `"`
	referenciaPropuestaDistinta := `"propuesta":{"referencia":"propuesta:bolsa:otra"`
	artefactoDesligado := reemplazarUnaVezSeleccionO6Integracion(
		t, canonico, referenciaPropuesta, referenciaPropuestaDistinta,
	)
	artefactoDesligado = rehuellarArtefactoSeleccionO6Integracion(
		t, artefactoDesligado, artefacto.HuellaArtefactoSHA256,
	)
	reciboDesligado := reemplazarUnaVezSeleccionO6Integracion(
		t, reciboJSON, referenciaPropuesta, referenciaPropuestaDistinta,
	)
	type casoEncuadre struct {
		nombre    string
		solicitud string
		artefacto string
		recibo    string
	}
	nuevoCaso := func(nombre, solicitud, reciboMutado, artefactoMutado string) casoEncuadre {
		return casoEncuadre{nombre: nombre, solicitud: solicitud,
			recibo: reciboMutado, artefacto: artefactoMutado}
	}
	casos := []casoEncuadre{
		nuevoCaso("orden de claves artefacto", solicitudJSON, reciboJSON, reordenado),
		nuevoCaso("clave duplicada artefacto", solicitudJSON, reciboJSON, duplicado),
		nuevoCaso("version textual artefacto", solicitudJSON, reciboJSON, versionTexto),
		nuevoCaso("contrato textual artefacto", solicitudJSON, reciboJSON, contratoTexto),
		nuevoCaso("decimal artefacto", solicitudJSON, reciboJSON, decimalArtefacto),
		nuevoCaso("exponente artefacto", solicitudJSON, reciboJSON, exponenteArtefacto),
		nuevoCaso("material desligado", solicitudJSON, reciboDesligado, artefactoDesligado),
		nuevoCaso("decimal solicitud", reemplazarUnaVezSeleccionO6Integracion(
			t, solicitudJSON, `"version_expediente":7`, `"version_expediente":7.0`),
			reciboJSON, canonico),
		nuevoCaso("exponente solicitud", reemplazarUnaVezSeleccionO6Integracion(
			t, solicitudJSON, `"version_expediente":7`, `"version_expediente":7e0`),
			reciboJSON, canonico),
		nuevoCaso("duplicada solicitud", reemplazarUnaVezSeleccionO6Integracion(
			t, solicitudJSON, `{"clave_idempotencia":`,
			`{"clave_idempotencia":"`+reserva.Solicitud.ClaveIdempotencia+`","clave_idempotencia":`),
			reciboJSON, canonico),
		nuevoCaso("decimal recibo", solicitudJSON, reemplazarUnaVezSeleccionO6Integracion(
			t, reciboJSON, `"orden_seleccionado":2`, `"orden_seleccionado":2.0`), canonico),
		nuevoCaso("exponente recibo", solicitudJSON, reemplazarUnaVezSeleccionO6Integracion(
			t, reciboJSON, `"orden_seleccionado":2`, `"orden_seleccionado":2e0`), canonico),
		nuevoCaso("duplicada recibo", solicitudJSON, reemplazarUnaVezSeleccionO6Integracion(
			t, reciboJSON, `{"operacion_ref":`,
			`{"operacion_ref":"`+recibo.OperacionRef+`","operacion_ref":`), canonico),
	}
	for _, mutacion := range []struct{ nombre, anterior, posterior string }{
		{"accion", `"accion:evento:001"`, `"accion:evento:mutada"`},
		{"evento", `"evento:llamamiento:001"`, `"evento:llamamiento:mutado"`},
		{"llamamiento", `"llamamiento:seleccion:001"`, `"llamamiento:seleccion:mutado"`},
		{"seleccion", strings.Repeat("9", 64), strings.Repeat("8", 64)},
		{"retencion", `"retencion:seleccion:001"`, `"retencion:seleccion:mutada"`},
		{"orden", `"orden_seleccionado":2`, `"orden_seleccionado":3`},
	} {
		reciboMutado := reemplazarUnaVezSeleccionO6Integracion(
			t, reciboJSON, mutacion.anterior, mutacion.posterior,
		)
		artefactoMutado := reemplazarUnaVezSeleccionO6Integracion(
			t, canonico, mutacion.anterior, mutacion.posterior,
		)
		artefactoMutado = rehuellarArtefactoSeleccionO6Integracion(
			t, artefactoMutado, artefacto.HuellaArtefactoSHA256,
		)
		casos = append(casos, nuevoCaso(
			"ligadura "+mutacion.nombre, solicitudJSON, reciboMutado, artefactoMutado,
		))
	}
	casos = append(casos,
		mutacionCronologicaSeleccionO6Integracion(t, ctx, admin, reserva, recibo,
			artefacto, "evidencia emitida fuera de contexto", "2026-08-31T09:02:00Z",
			"2026-08-31T09:10:00Z", "2026-08-31T09:08:00Z", "2026-08-31T09:11:00Z"),
		mutacionCronologicaSeleccionO6Integracion(t, ctx, admin, reserva, recibo,
			artefacto, "evidencia valida fuera de contexto", "2026-08-31T09:02:00Z",
			"2026-08-31T09:02:00Z", "2026-08-31T09:08:00Z", "2026-08-31T09:11:00Z"),
	)
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if caso.solicitud == solicitudJSON && caso.recibo == reciboJSON &&
				caso.artefacto == canonico {
				t.Fatal("la mutacion no produjo cambio efectivo")
			}
			var aplicada bool
			err := pool.QueryRow(ctx, `SELECT `+funcionConfirmarSeleccionO6+`(
				$1::uuid,$2::text,$3::text,$4::text,$5::text,$6::text)`,
				reserva.Solicitud.ClaveIdempotencia, reserva.Solicitud.HuellaSemantica,
				reserva.ReservaRef, caso.solicitud, caso.recibo, caso.artefacto).Scan(&aplicada)
			var falloPG *pgconn.PgError
			if !errors.As(err, &falloPG) || falloPG.Code != "22023" {
				t.Fatalf("encuadre no canonico aceptado: aplicada=%v err=%v", aplicada, err)
			}
		})
	}
}

func mutacionCronologicaSeleccionO6Integracion(
	t *testing.T, ctx context.Context, admin *pgxpool.Pool,
	reserva ports.ReservaEjecucionSeleccionLlamamiento,
	recibo ports.ReciboSolicitudLlamamientoBolsa,
	artefacto ports.ArtefactoProbatorioLlamamientoBolsa,
	nombre, emitidaAnterior, emitidaNueva, validaAnterior, validaNueva string,
) struct{ nombre, solicitud, artefacto, recibo string } {
	t.Helper()
	reciboMutado := string(debeJSONSeleccionO6Prueba(t, recibo))
	if emitidaAnterior != emitidaNueva {
		reciboMutado = reemplazarNSeleccionO6Integracion(t, reciboMutado,
			`"emitida_en":"`+emitidaAnterior+`"`, `"emitida_en":"`+emitidaNueva+`"`, 1)
	}
	reciboMutado = reemplazarNSeleccionO6Integracion(t, reciboMutado,
		`"valida_hasta":"`+validaAnterior+`"`, `"valida_hasta":"`+validaNueva+`"`, 1)
	artefactoMutado := string(debeJSONSeleccionO6Prueba(t, artefacto))
	if emitidaAnterior != emitidaNueva {
		artefactoMutado = reemplazarNSeleccionO6Integracion(t, artefactoMutado,
			`"emitida_en":"`+emitidaAnterior+`"`, `"emitida_en":"`+emitidaNueva+`"`, 2)
	}
	artefactoMutado = reemplazarNSeleccionO6Integracion(t, artefactoMutado,
		`"valida_hasta":"`+validaAnterior+`"`, `"valida_hasta":"`+validaNueva+`"`, 2)
	artefactoMutado = rehuellarMaterialesSeleccionO6Integracion(t, ctx, admin,
		artefactoMutado, artefacto)
	artefactoMutado = rehuellarArtefactoSeleccionO6Integracion(
		t, artefactoMutado, artefacto.HuellaArtefactoSHA256,
	)
	return struct{ nombre, solicitud, artefacto, recibo string }{
		nombre: nombre, solicitud: string(debeCodificarSolicitudSeleccionO6Prueba(t, reserva.Solicitud)),
		artefacto: artefactoMutado, recibo: reciboMutado,
	}
}

func rehuellarMaterialesSeleccionO6Integracion(
	t *testing.T, ctx context.Context, admin *pgxpool.Pool, contenido string,
	artefacto ports.ArtefactoProbatorioLlamamientoBolsa,
) string {
	t.Helper()
	var peticion, respuesta string
	err := admin.QueryRow(ctx, `SELECT huellas[1], huellas[2] FROM (
		SELECT vec_contratacion_temporal.huellas_materiales_seleccion_llamamiento_o6_v1(
			$1::jsonb) AS huellas
	) calculo`, contenido).Scan(&peticion, &respuesta)
	if err != nil {
		t.Fatal(err)
	}
	contenido = reemplazarUnaVezSeleccionO6Integracion(t, contenido,
		`"huella_peticion_sha256":"`+artefacto.Evidencia.HuellaPeticionSHA256+`"`,
		`"huella_peticion_sha256":"`+peticion+`"`)
	return reemplazarUnaVezSeleccionO6Integracion(t, contenido,
		`"huella_respuesta_sha256":"`+artefacto.Evidencia.HuellaRespuestaSHA256+`"`,
		`"huella_respuesta_sha256":"`+respuesta+`"`)
}

func reemplazarUnaVezSeleccionO6Integracion(t *testing.T, texto, anterior, posterior string) string {
	t.Helper()
	return reemplazarNSeleccionO6Integracion(t, texto, anterior, posterior, 1)
}

func reemplazarNSeleccionO6Integracion(
	t *testing.T, texto, anterior, posterior string, esperadas int,
) string {
	t.Helper()
	if anterior == posterior || strings.Count(texto, anterior) != esperadas {
		t.Fatalf("mutacion no efectiva o cardinalidad inesperada: %q", anterior)
	}
	return strings.Replace(texto, anterior, posterior, esperadas)
}

func rehuellarArtefactoSeleccionO6Integracion(
	t *testing.T, contenido, huellaAnterior string,
) string {
	t.Helper()
	marca := `"huella_artefacto_sha256":"` + huellaAnterior + `"`
	if strings.Count(contenido, marca) != 1 {
		t.Fatal("el artefacto de integracion no tiene una huella exterior unica")
	}
	preimagen := strings.Replace(contenido, marca, `"huella_artefacto_sha256":""`, 1)
	huella := sha256.Sum256([]byte(preimagen))
	nuevaMarca := `"huella_artefacto_sha256":"` + hex.EncodeToString(huella[:]) + `"`
	return strings.Replace(contenido, marca, nuevaMarca, 1)
}

func borrarEjecucionSeleccionO6Integracion(
	t *testing.T, ctx context.Context, admin *pgxpool.Pool, clave string,
) {
	t.Helper()
	ctxLimpieza, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	tx, err := admin.Begin(ctxLimpieza)
	if err == nil {
		_, err = tx.Exec(ctxLimpieza, "SET LOCAL ROLE vec_contratacion_temporal_propietario")
	}
	if err == nil {
		_, err = tx.Exec(ctxLimpieza, `DELETE FROM
			vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6
			WHERE clave_idempotencia=$1::uuid`, clave)
	}
	if err == nil {
		err = tx.Commit(ctxLimpieza)
	} else if tx != nil {
		_ = tx.Rollback(ctxLimpieza)
	}
	if err != nil {
		t.Errorf("no se limpio la ejecucion Go-PG-Go: %v", err)
	}
}
