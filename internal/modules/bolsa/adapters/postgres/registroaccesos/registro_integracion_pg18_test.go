package registroaccesos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	registroaplicacion "vec-diputacion-granada/internal/modules/bolsa/application/registroaccesos"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	vecpruebas "vec-diputacion-granada/internal/vec/pruebas"
)

const (
	variableDSNRegistroAccesosPG18      = "VEC_BOLSA_ACCESOS_PG18_DSN"
	variableDSNAdminRegistroAccesosPG18 = "VEC_BOLSA_ACCESOS_PG18_ADMIN_DSN"
)

type generadorCorrelacionAccesosPG18 string

func (g generadorCorrelacionAccesosPG18) NuevaReferenciaCorrelacionAutorizacionV2(
	context.Context,
) (string, error) {
	return string(g), nil
}

type seudonimizadorAccesosPG18 string

func (s seudonimizadorAccesosPG18) SeudonimizarSujetoAlmacen(
	context.Context,
	vecports.SolicitudSeudonimizarSujetoAlmacen,
) (string, error) {
	return string(s), nil
}

type iniciadorCommitObservadoPG18 struct {
	pool      *pgxpool.Pool
	alcanzado chan struct{}
	liberar   chan struct{}
}

func (i iniciadorCommitObservadoPG18) BeginTx(
	ctx context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	tx, err := i.pool.BeginTx(ctx, opciones)
	if err != nil {
		return nil, err
	}
	return &transaccionCommitObservadoPG18{
		Tx: tx, alcanzado: i.alcanzado, liberar: i.liberar,
	}, nil
}

type transaccionCommitObservadoPG18 struct {
	pgx.Tx
	alcanzado chan struct{}
	liberar   chan struct{}
	unaVez    sync.Once
}

func (t *transaccionCommitObservadoPG18) Commit(ctx context.Context) error {
	t.unaVez.Do(func() { close(t.alcanzado) })
	select {
	case <-t.liberar:
		return t.Tx.Commit(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
}

type resultadoConsultaAccesosPG18 struct {
	pagina registroaplicacion.PaginaConsultaAdministrativaAccesos
	err    error
}

func TestIntegracionConsultarAccesosAdministrativosPG18(t *testing.T) {
	dsn := os.Getenv(variableDSNRegistroAccesosPG18)
	dsnAdmin := os.Getenv(variableDSNAdminRegistroAccesosPG18)
	if dsn == "" && dsnAdmin == "" {
		t.Skip("integración T13 omitida; use probar_pg18.sh")
	}
	if dsn == "" || dsnAdmin == "" {
		t.Fatal("configuración PG18 T13 parcial")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	pool := abrirPoolAccesosPG18(t, ctx, dsn)
	defer pool.Close()
	admin := abrirPoolAccesosPG18(t, ctx, dsnAdmin)
	defer admin.Close()

	ahora := relojPostgreSQLAccesosPG18(t, ctx, admin)
	solicitud := solicitudConsultaAccesosPG18(
		t, ahora, "decision:go:t13:exito",
		"correlacion_"+strings.Repeat("9", 32),
	)
	alcanzado := make(chan struct{})
	liberar := make(chan struct{})
	registro := &RegistroPostgreSQL{
		pool: iniciadorCommitObservadoPG18{
			pool: pool, alcanzado: alcanzado, liberar: liberar,
		},
		ahora: func() time.Time { return ahora },
	}
	resultado := make(chan resultadoConsultaAccesosPG18, 1)
	go func() {
		pagina, err := registro.ConsultarAccesosAdministrativos(ctx, solicitud)
		resultado <- resultadoConsultaAccesosPG18{pagina: pagina, err: err}
	}()

	select {
	case <-alcanzado:
	case <-time.After(10 * time.Second):
		t.Fatal("el adaptador no alcanzó COMMIT")
	}
	select {
	case previo := <-resultado:
		t.Fatalf("el adaptador devolvió datos antes de COMMIT: %+v", previo)
	default:
	}
	if filas := contarCorrelacionAccesosPG18(
		t, ctx, admin, "correlacion_"+strings.Repeat("9", 32),
	); filas != 0 {
		t.Fatalf("auditoría visible antes de COMMIT: %d", filas)
	}
	close(liberar)
	var confirmado resultadoConsultaAccesosPG18
	select {
	case confirmado = <-resultado:
	case <-time.After(10 * time.Second):
		t.Fatal("el adaptador no devolvió después de COMMIT")
	}
	if confirmado.err != nil ||
		confirmado.pagina.AuditoriaConfirmada.CorrelationRef !=
			"correlacion_"+strings.Repeat("9", 32) ||
		confirmado.pagina.AuditoriaConfirmada.Seq == 0 {
		t.Fatalf("consulta Go→pgx→PG18 no confirmada: %+v", confirmado)
	}
	if filas := contarCorrelacionAccesosPG18(
		t, ctx, admin, "correlacion_"+strings.Repeat("9", 32),
	); filas != 1 {
		t.Fatalf("COMMIT no hizo visible una auditoría: %d", filas)
	}
	probarFiltroLigadoGoPG18(t, ctx, pool, ahora)
	if len(confirmado.pagina.Registros) == 0 {
		t.Fatal("la prueba Go no recibió una fila para verificar el filtro")
	}
	fuera := append(
		[]registroaplicacion.ResumenAccesoAdministrativo(nil),
		confirmado.pagina.Registros...,
	)
	fuera[0].ActorSeudonimizado =
		"hmac-sha256:bolsa_accesos_v1:" + strings.Repeat("b", 64)
	if _, err := registroaplicacion.NuevaPaginaConsultaAdministrativaAccesos(
		solicitud, fuera, confirmado.pagina.Siguiente,
		confirmado.pagina.AuditoriaConfirmada,
	); !errors.Is(err, registroaplicacion.ErrRegistroAccesosInvalido) {
		t.Fatalf("Go aceptó respuesta fuera del filtro autorizado: %v", err)
	}

	registroNormal, err := NuevoRegistroPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	if pagina, err := registroNormal.ConsultarAccesosAdministrativos(
		ctx, solicitud,
	); !errors.Is(
		err, registroaplicacion.ErrConsultaAdministrativaAccesosDenegada,
	) || len(pagina.Registros) != 0 {
		t.Fatalf("replay no revirtió ni falló cerrado: pagina=%+v err=%v", pagina, err)
	}
	if filas := contarCorrelacionAccesosPG18(
		t, ctx, admin, "correlacion_"+strings.Repeat("9", 32),
	); filas != 1 {
		t.Fatalf("rollback del replay alteró la auditoría: %d", filas)
	}

	denegada := solicitudConsultaAccesosPG18(
		t, ahora, "decision:go:t13:denegada",
		"correlacion_"+strings.Repeat("8", 32),
	)
	if pagina, err := registroNormal.ConsultarAccesosAdministrativos(
		ctx, denegada,
	); !errors.Is(
		err, registroaplicacion.ErrConsultaAdministrativaAccesosDenegada,
	) || len(pagina.Registros) != 0 {
		t.Fatalf("decisión denegada filtró datos: pagina=%+v err=%v", pagina, err)
	}
	if filas := contarCorrelacionAccesosPG18(
		t, ctx, admin, "correlacion_"+strings.Repeat("8", 32),
	); filas != 0 {
		t.Fatalf("denegación dejó efecto parcial: %d", filas)
	}

	canceladaCommit := solicitudConsultaAccesosPG18(
		t, ahora, "decision:go:t13:cancelacion_commit",
		"correlacion_"+strings.Repeat("7", 32),
	)
	commitCanceladoAlcanzado := make(chan struct{})
	commitCanceladoLiberar := make(chan struct{})
	registroCommitCancelado := &RegistroPostgreSQL{
		pool: iniciadorCommitObservadoPG18{
			pool: pool, alcanzado: commitCanceladoAlcanzado,
			liberar: commitCanceladoLiberar,
		},
		ahora: func() time.Time { return ahora },
	}
	ctxCommitCancelado, cancelarCommit := context.WithCancel(context.Background())
	resultadoCancelado := make(chan resultadoConsultaAccesosPG18, 1)
	go func() {
		pagina, err := registroCommitCancelado.ConsultarAccesosAdministrativos(
			ctxCommitCancelado, canceladaCommit,
		)
		resultadoCancelado <- resultadoConsultaAccesosPG18{
			pagina: pagina, err: err,
		}
	}()
	select {
	case <-commitCanceladoAlcanzado:
	case <-time.After(10 * time.Second):
		t.Fatal("la cancelación no alcanzó la frontera de COMMIT")
	}
	if filas := contarCorrelacionAccesosPG18(
		t, ctx, admin, "correlacion_"+strings.Repeat("7", 32),
	); filas != 0 {
		t.Fatalf("auditoría cancelable visible antes del COMMIT: %d", filas)
	}
	cancelarCommit()
	select {
	case cancelado := <-resultadoCancelado:
		if !errors.Is(cancelado.err, context.Canceled) ||
			len(cancelado.pagina.Registros) != 0 {
			t.Fatalf(
				"cancelación en COMMIT filtró datos: pagina=%+v err=%v",
				cancelado.pagina, cancelado.err,
			)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("cancelar en COMMIT no desbloqueó el adaptador")
	}
	if filas := contarCorrelacionAccesosPG18(
		t, ctx, admin, "correlacion_"+strings.Repeat("7", 32),
	); filas != 0 {
		t.Fatalf("cancelación en COMMIT no revirtió la auditoría: %d", filas)
	}

	ctxCancelado, cancelarAhora := context.WithCancel(context.Background())
	cancelarAhora()
	if pagina, err := registroNormal.ConsultarAccesosAdministrativos(
		ctxCancelado, denegada,
	); !errors.Is(err, context.Canceled) || len(pagina.Registros) != 0 {
		t.Fatalf("cancelación no falló cerrada: pagina=%+v err=%v", pagina, err)
	}
}

func probarFiltroLigadoGoPG18(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	ahora time.Time,
) {
	t.Helper()
	mutaciones := []struct {
		campo string
		valor any
	}{
		{"version", 2},
		{"actor_seudonimizado", "hmac-sha256:bolsa_accesos_v1:" +
			strings.Repeat("b", 64)},
		{"module_id", "otro.modulo"},
		{"accion", "otra.accion"},
		{"finalidad_acceso", "otra-finalidad"},
		{"recurso_ref", "expediente:otro"},
		{"expediente_ref", "expediente:distinto"},
		{"resultado", "denegado"},
		{"desde_inclusive", formatearInstanteFiltro(ahora.Add(-30 * time.Minute))},
		{"hasta_exclusive", formatearInstanteFiltro(ahora.Add(30 * time.Minute))},
		{"version_objeto", 8},
		{"limite", 9},
		{"cursor", "cursor:v1:" + strings.Repeat("e", 64)},
		{"finalidad_consulta", "otra-finalidad"},
	}
	for indice, mutacion := range mutaciones {
		solicitud := solicitudConsultaAccesosPG18(
			t, ahora,
			"decision:go:t13:mutacion:"+mutacion.campo,
			"correlacion_"+fmt.Sprintf("%032x", indice+1),
		)
		for nombre, valor := range map[string]any{
			"valor": mutacion.valor, "null": nil, "tipo": true,
		} {
			t.Run("pgx_filtro_"+mutacion.campo+"_"+nombre, func(t *testing.T) {
				exigirCargaConsultaDenegadaPG18(
					t, ctx, pool, solicitud, ahora, mutacion.campo, valor,
				)
			})
		}
	}
	for indice, caso := range []struct {
		campo string
		valor string
	}{
		{"actor_seudonimizado", "hmac-sha256:bolsa_accesos_v1:" +
			strings.Repeat("A", 64)},
		{"cursor", "cursor:v1:" + strings.Repeat("A", 64)},
	} {
		solicitud := solicitudConsultaAccesosPG18(
			t, ahora,
			"decision:go:t13:mutacion:hex:"+caso.campo,
			"correlacion_"+fmt.Sprintf("%032x", len(mutaciones)+indice+1),
		)
		exigirCargaConsultaDenegadaPG18(
			t, ctx, pool, solicitud, ahora, caso.campo, caso.valor,
		)
	}
}

func exigirCargaConsultaDenegadaPG18(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	solicitud registroaplicacion.SolicitudConsultaAdministrativaAccesos,
	ahora time.Time,
	campo string,
	valor any,
) {
	t.Helper()
	datos, err := solicitud.RevalidarAutorizacion(ahora)
	if err != nil {
		t.Fatal(err)
	}
	carga, err := serializarConsulta(solicitud, datos, ahora)
	if err != nil {
		t.Fatal(err)
	}
	var documento map[string]any
	if err = json.Unmarshal(carga, &documento); err != nil {
		t.Fatal(err)
	}
	filtro, correcto := documento["filtro"].(map[string]any)
	if !correcto {
		t.Fatal("serialización Go perdió el filtro")
	}
	filtro[campo] = valor
	carga, err = json.Marshal(documento)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var respuesta []byte
	err = tx.QueryRow(ctx, sqlConsultar, carga).Scan(&respuesta)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) ||
		(pgErr.Code != "22023" && pgErr.Code != "42501") {
		t.Fatalf(
			"mutación %s no fue denegada: respuesta=%s error=%v",
			campo, respuesta, err,
		)
	}
}

func solicitudConsultaAccesosPG18(
	t *testing.T,
	ahora time.Time,
	decisionRef string,
	correlacionRef string,
) registroaplicacion.SolicitudConsultaAdministrativaAccesos {
	t.Helper()
	const principal = "per_0123456789abcdefghijkl"
	seudonimo := "hmac-sha256:bolsa_accesos_v1:" + strings.Repeat("a", 64)
	evidenciaActor, err := registroaplicacion.NuevaEvidenciaActorConsultaAccesos(
		context.Background(), principal, seudonimizadorAccesosPG18(seudonimo),
	)
	if err != nil {
		t.Fatal(err)
	}
	filtro := registroaplicacion.FiltroConsultaAdministrativaAccesos{
		Version: 1, ActorSeudonimizado: seudonimo,
		DesdeInclusive: ahora.Add(-time.Hour),
		HastaExclusive: ahora.Add(time.Hour),
		Limite:         10, FinalidadDeLaConsulta: "control-interno",
	}
	recurso, err := registroaplicacion.RecursoAutorizableConsultaAdministrativaAccesos(
		filtro, evidenciaActor,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor, vinculo, err := vecpruebas.NuevoContextoYVinculo(
		ahora, principal, "prf_0123456789abcdefghijkl",
		vecdomain.AuthMethodSSO, vecdomain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(), generadorCorrelacionAccesosPG18(correlacionRef),
	)
	if err != nil {
		t.Fatal(err)
	}
	motivo := vecdomain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("e", 64),
		EntradaClave:         "motivo_" + strings.Repeat("d", 32),
	}
	solicitudPDP, err := vecdomain.NuevaSolicitudAutorizacionLigadaV2(
		vecdomain.DatosSolicitudAutorizacionLigadaV2{
			ContextoActor: actor, VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: motivo,
			Accion:           registroaplicacion.AccionAuditoriaConsultaAccesos,
			Recurso:          recurso, Finalidad: filtro.FinalidadDeLaConsulta,
			Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaContexto, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaSolicitud, err := vecdomain.HuellaSHA256SolicitudAutorizacionV2(
		solicitudPDP,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaMotivo, err := vecdomain.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := vecdomain.HuellaEvidenciasCatalogoPoliticasAutorizacion(
		nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	decision := vecdomain.DecisionAutorizacion{
		DecisionRef: decisionRef, Concedida: true, Codigo: "concedida",
		PrincipalID: actor.Principal.ID, PerfilActivoRef: actor.PerfilActivoRef,
		Accion:     registroaplicacion.AccionAuditoriaConsultaAccesos,
		RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID,
		TipoRecurso: recurso.Tipo, ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad: filtro.FinalidadDeLaConsulta, CorrelacionRef: correlacionRef,
		EsquemaHuellaSolicitud:                vecdomain.EsquemaHuellaSolicitudAutorizacionV2,
		SolicitudHuellaSHA256:                 huellaSolicitud,
		EsquemaHuellaMotivo:                   vecdomain.EsquemaHuellaMotivoAutorizacionV2,
		MotivoHuellaSHA256:                    huellaMotivo,
		VinculoAutenticacionActor:             vinculo,
		AsignacionRef:                         "asignacion:registro_accesos:go:v1",
		AsignacionHuellaSHA256:                strings.Repeat("1", 64),
		VersionRolRef:                         "rol:auditor_t13:v1",
		VersionRolHuellaSHA256:                strings.Repeat("2", 64),
		ControlVigenciaVersionRolRef:          "rol:auditor_t13:v1",
		ControlVigenciaVersionRolRevision:     1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("3", 64),
		RevisionCatalogoPoliticas:             1,
		CatalogoPoliticasHuellaSHA256:         huellaCatalogo,
		GarantiaMinima:                        vecdomain.AuthAssuranceHigh,
		CamposPermitidos: registroaplicacion.
			CamposPermitidosConsultaAdministrativaAccesos(),
		EmitidaEn: ahora.Add(-time.Second), ValidaHasta: ahora.Add(time.Minute),
	}
	evidencia, err := vecports.NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		decision, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaFiltro, err := registroaplicacion.HuellaFiltroConsultaAdministrativaAccesos(
		filtro,
	)
	if err != nil {
		t.Fatal(err)
	}
	auditoria := vecdomain.AuditEntry{
		ActorID: seudonimo, ActorProfile: "auditoria-interna",
		ActorRoles: []string{"auditor"}, AuthMethod: vecdomain.AuthMethodSSO,
		AuthAssurance:    vecdomain.AuthAssuranceHigh,
		AuthorizationRef: decisionRef, Purpose: filtro.FinalidadDeLaConsulta,
		Action:        registroaplicacion.AccionAuditoriaConsultaAccesos,
		ModuleID:      registroaplicacion.ModuloAuditoriaConsultaAccesos,
		SubjectRef:    "consulta-accesos:sha256:" + huellaFiltro,
		ObjectVersion: 1, Result: "permitido",
		CorrelationRef: correlacionRef, Metadata: map[string]string{},
		OccurredAt: ahora,
	}
	solicitud, err := registroaplicacion.NuevaSolicitudConsultaAdministrativaAccesos(
		filtro, auditoria, evidenciaActor, evidencia,
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud
}

func abrirPoolAccesosPG18(
	t *testing.T,
	ctx context.Context,
	dsn string,
) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	return pool
}

func relojPostgreSQLAccesosPG18(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
) time.Time {
	t.Helper()
	var ahora time.Time
	if err := admin.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&ahora); err != nil {
		t.Fatal(err)
	}
	return ahora.UTC().Truncate(time.Microsecond)
}

func contarCorrelacionAccesosPG18(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	correlacion string,
) int {
	t.Helper()
	var filas int
	if err := admin.QueryRow(
		ctx,
		`SELECT count(*) FROM vec_bolsa_registro_accesos.registro_acceso
		  WHERE correlation_ref = $1`,
		correlacion,
	).Scan(&filas); err != nil {
		t.Fatal(err)
	}
	return filas
}
