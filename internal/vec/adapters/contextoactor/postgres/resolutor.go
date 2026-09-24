// Package postgres implementa la resolucion y el registro atomicos del
// contexto de actor V2. No abre conexiones, conserva DSN ni ejecuta migraciones.
package postgres

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Con alcance vacio se usan las firmas heredadas de siete argumentos, que
// ContextoActor 000007 define como equivalentes a '{}': mismo SQL, mismos bytes
// y ningun requisito de migracion nuevo para CT y el resto. Solo un alcance
// con proyecciones usa las firmas de 000007 y, sin ellas, falla cerrado.
const (
	consultaResolverContextoActorV2 = `
		SELECT operacion_ref, registro_contexto_ref,
		       representacion_canonica, huella_sha256,
		       manifiesto_procedencia_canonico,
		       manifiesto_procedencia_huella_sha256,
		       autoridad_efectiva, resuelto_en
		  FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
		       $1, $2, $3, $4, $5, $6, $7)`
	consultaReconciliarContextoActorV2 = `
		SELECT operacion_ref, registro_contexto_ref,
		       representacion_canonica, huella_sha256,
		       manifiesto_procedencia_canonico,
		       manifiesto_procedencia_huella_sha256,
		       autoridad_efectiva, resuelto_en
		  FROM vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
		       $1, $2, $3, $4, $5, $6, $7)`
	consultaResolverContextoActorV2Alcance = `
		SELECT operacion_ref, registro_contexto_ref,
		       representacion_canonica, huella_sha256,
		       manifiesto_procedencia_canonico,
		       manifiesto_procedencia_huella_sha256,
		       autoridad_efectiva, resuelto_en
		  FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2(
		       $1, $2, $3, $4, $5, $6, $7, $8::text[])`
	consultaReconciliarContextoActorV2Alcance = `
		SELECT operacion_ref, registro_contexto_ref,
		       representacion_canonica, huella_sha256,
		       manifiesto_procedencia_canonico,
		       manifiesto_procedencia_huella_sha256,
		       autoridad_efectiva, resuelto_en
		  FROM vec_contexto_actor_v1.reconciliar_contexto_actor_v2(
		       $1, $2, $3, $4, $5, $6, $7, $8::text[])`
)

type iniciadorContextoActorPostgreSQL interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// ResolutorRegistroContextoActorPostgreSQLV2 usa un pool cuyo LOGIN solo puede
// ejecutar las funciones cerradas del modulo. El recibo usa material CSPRNG
// distinto del token de operacion creado por el servicio.
type ResolutorRegistroContextoActorPostgreSQLV2 struct {
	pool      iniciadorContextoActorPostgreSQL
	aleatorio io.Reader
}

// NuevoResolutorRegistroContextoActorPostgreSQLV2 acredita inmediatamente el
// LOGIN efectivo y su membresia exclusiva. Un pool con SET ROLE, privilegios
// administrativos o grupos adicionales se rechaza cerrado.
func NuevoResolutorRegistroContextoActorPostgreSQLV2(
	ctx context.Context,
	pool *pgxpool.Pool,
) (*ResolutorRegistroContextoActorPostgreSQLV2, error) {
	if ctx == nil || pool == nil {
		return nil, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	var identidad string
	var acreditada bool
	if err := pool.QueryRow(ctx, `
		SELECT identidad_login, acreditada
		  FROM vec_contexto_actor_v1.acreditar_runtime_contexto_actor_v1()`,
	).Scan(&identidad, &acreditada); err != nil || !acreditada || identidad == "" {
		return nil, errorResolutorContextoActorPostgreSQL(ctx)
	}
	return nuevoResolutorRegistroContextoActorPostgreSQLV2(pool, rand.Reader)
}

func nuevoResolutorRegistroContextoActorPostgreSQLV2(
	pool iniciadorContextoActorPostgreSQL,
	aleatorio io.Reader,
) (*ResolutorRegistroContextoActorPostgreSQLV2, error) {
	if valorNuloContextoActorPostgreSQL(pool) || valorNuloContextoActorPostgreSQL(aleatorio) {
		return nil, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	return &ResolutorRegistroContextoActorPostgreSQLV2{pool: pool, aleatorio: aleatorio}, nil
}

func (r *ResolutorRegistroContextoActorPostgreSQLV2) ResolverYRegistrarContextoActorV2(
	ctx context.Context,
	solicitud ports.SolicitudResolucionRegistroContextoActorV2,
) (ports.ConfirmacionRegistroContextoActorV2, error) {
	if ctx == nil || r == nil || valorNuloContextoActorPostgreSQL(r.pool) ||
		valorNuloContextoActorPostgreSQL(r.aleatorio) || solicitud.Validar() != nil {
		return ports.ConfirmacionRegistroContextoActorV2{},
			ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ConfirmacionRegistroContextoActorV2{}, err
	}
	reciboRef, err := nuevaReferenciaContextoActorV2(
		ctx, r.aleatorio, "rca_", ports.ErrResolutorRegistroContextoActorNoDisponible,
	)
	if err != nil {
		return ports.ConfirmacionRegistroContextoActorV2{}, errorResolutorContextoActorPostgreSQL(ctx)
	}
	argumentos := argumentosContextoActorPostgreSQL(solicitud, reciboRef)
	consultaResolver, consultaReconciliar := consultaResolverContextoActorV2, consultaReconciliarContextoActorV2
	if !solicitud.Proyecciones.Vacio() {
		consultaResolver, consultaReconciliar = consultaResolverContextoActorV2Alcance, consultaReconciliarContextoActorV2Alcance
	}

	// La unica repeticion permitida conserva operacion_ref y rca_. Se usa cuando
	// la reconciliacion confirma ausencia tras un COMMIT fallido.
	for intento := 0; intento < 2; intento++ {
		respuesta, estado, denegacion := r.ejecutar(ctx, consultaResolver, argumentos)
		if estado == estadoContextoActorConfirmado {
			return confirmarRespuestaContextoActor(solicitud, respuesta)
		}
		if estado == estadoContextoActorDenegado {
			return ports.ConfirmacionRegistroContextoActorV2{}, errors.Join(
				ports.ErrResolutorRegistroContextoActorNoDisponible, denegacion,
			)
		}
		if estado == estadoContextoActorReintentable {
			continue
		}
		if estado != estadoContextoActorCommitIncierto {
			return ports.ConfirmacionRegistroContextoActorV2{}, errorResolutorContextoActorPostgreSQL(ctx)
		}
		reconciliada, estadoReconciliacion := r.reconciliar(ctx, consultaReconciliar, argumentos)
		switch estadoReconciliacion {
		case estadoContextoActorConfirmado:
			if !respuestasContextoActorIguales(respuesta, reconciliada) {
				return ports.ConfirmacionRegistroContextoActorV2{}, ports.ErrResolutorRegistroContextoActorNoDisponible
			}
			return confirmarRespuestaContextoActor(solicitud, reconciliada)
		case estadoContextoActorAusente:
			continue
		default:
			return ports.ConfirmacionRegistroContextoActorV2{}, errorResolutorContextoActorPostgreSQL(ctx)
		}
	}
	return ports.ConfirmacionRegistroContextoActorV2{}, ports.ErrResolutorRegistroContextoActorNoDisponible
}

type estadoEjecucionContextoActor uint8

const (
	estadoContextoActorFallido estadoEjecucionContextoActor = iota
	estadoContextoActorConfirmado
	estadoContextoActorCommitIncierto
	estadoContextoActorAusente
	estadoContextoActorReintentable
	estadoContextoActorDenegado
)

type respuestaContextoActorPostgreSQL struct {
	operacionRef, reciboRef string
	representacion          []byte
	huella                  string
	manifiesto              []byte
	huellaManifiesto        string
	autoridadEfectiva       string
	resueltoEn              time.Time
}

func (r *ResolutorRegistroContextoActorPostgreSQLV2) ejecutar(
	ctx context.Context,
	consulta string,
	argumentos []any,
) (respuestaContextoActorPostgreSQL, estadoEjecucionContextoActor, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido, nil
	}
	defer revertirContextoActorPostgreSQL(tx)
	if prepararTransaccionContextoActorPostgreSQL(ctx, tx) != nil {
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido, nil
	}
	respuesta, err := consultarRespuestaContextoActor(ctx, tx, consulta, argumentos)
	if err != nil {
		if denegacion := denegacionProyeccionContextoActorPostgreSQL(err); denegacion != nil {
			return respuestaContextoActorPostgreSQL{}, estadoContextoActorDenegado, denegacion
		}
		if errorContextoActorPostgreSQLReintentable(err) {
			return respuestaContextoActorPostgreSQL{}, estadoContextoActorReintentable, nil
		}
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido, nil
	}
	if err = tx.Commit(ctx); err != nil {
		return respuesta, estadoContextoActorCommitIncierto, nil
	}
	return respuesta, estadoContextoActorConfirmado, nil
}

// denegacionProyeccionContextoActorPostgreSQL traduce los dos motivos cerrados
// de 000007. Cualquier otro error conserva el tratamiento opaco.
func denegacionProyeccionContextoActorPostgreSQL(err error) error {
	var postgres *pgconn.PgError
	if !errors.As(err, &postgres) {
		return nil
	}
	switch postgres.Code {
	case "PCA01":
		return ports.ErrProyeccionEmpleadoContextoActorAusente
	case "PCA02":
		return ports.ErrProyeccionEmpleadoContextoActorAmbigua
	default:
		return nil
	}
}

func errorContextoActorPostgreSQLReintentable(err error) bool {
	var postgres *pgconn.PgError
	if !errors.As(err, &postgres) {
		return false
	}
	switch postgres.Code {
	case "23505", "40001", "40P01":
		return true
	default:
		return false
	}
}

func (r *ResolutorRegistroContextoActorPostgreSQLV2) reconciliar(
	ctx context.Context,
	consulta string,
	argumentos []any,
) (respuestaContextoActorPostgreSQL, estadoEjecucionContextoActor) {
	ctxReconciliacion, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	tx, err := r.pool.BeginTx(ctxReconciliacion, pgx.TxOptions{
		// READ COMMITTED permite renovar snapshot despues de esperar el mismo
		// advisory lock que la escritura. SERIALIZABLE podria observar ausencia
		// con un snapshot adquirido antes de la finalizacion concurrente.
		IsoLevel: pgx.ReadCommitted, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido
	}
	defer revertirContextoActorPostgreSQL(tx)
	if prepararTransaccionContextoActorPostgreSQL(ctxReconciliacion, tx) != nil {
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido
	}
	respuesta, err := consultarRespuestaContextoActor(
		ctxReconciliacion, tx, consulta, argumentos,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		if tx.Commit(ctxReconciliacion) != nil {
			return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido
		}
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorAusente
	}
	if err != nil || tx.Commit(ctxReconciliacion) != nil {
		return respuestaContextoActorPostgreSQL{}, estadoContextoActorFallido
	}
	return respuesta, estadoContextoActorConfirmado
}

func argumentosContextoActorPostgreSQL(
	s ports.SolicitudResolucionRegistroContextoActorV2,
	reciboRef string,
) []any {
	argumentos := []any{
		s.OperacionRef, reciboRef, s.Contexto.Cuenta.CuentaRef, s.Contexto.PerfilActivoRef,
		string(s.Contexto.Cuenta.Metodo), string(s.Contexto.Cuenta.Garantia), s.SolicitadoEn,
	}
	if s.Proyecciones.Vacio() {
		return argumentos
	}
	proyecciones := make([]string, 0, 1)
	for _, proyeccion := range s.Proyecciones.Proyecciones() {
		proyecciones = append(proyecciones, string(proyeccion))
	}
	return append(argumentos, proyecciones)
}

func consultarRespuestaContextoActor(
	ctx context.Context,
	tx pgx.Tx,
	consulta string,
	argumentos []any,
) (respuestaContextoActorPostgreSQL, error) {
	var respuesta respuestaContextoActorPostgreSQL
	err := tx.QueryRow(ctx, consulta, argumentos...).Scan(
		&respuesta.operacionRef, &respuesta.reciboRef, &respuesta.representacion,
		&respuesta.huella, &respuesta.manifiesto, &respuesta.huellaManifiesto,
		&respuesta.autoridadEfectiva, &respuesta.resueltoEn,
	)
	if err == nil {
		respuesta.representacion = append([]byte(nil), respuesta.representacion...)
		respuesta.manifiesto = append([]byte(nil), respuesta.manifiesto...)
		respuesta.resueltoEn = respuesta.resueltoEn.UTC().Truncate(time.Microsecond)
	}
	return respuesta, err
}

func confirmarRespuestaContextoActor(
	solicitud ports.SolicitudResolucionRegistroContextoActorV2,
	respuesta respuestaContextoActorPostgreSQL,
) (ports.ConfirmacionRegistroContextoActorV2, error) {
	contexto, err := domain.RehidratarContextoActorVinculadoV2(respuesta.representacion)
	if err != nil {
		return ports.ConfirmacionRegistroContextoActorV2{}, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	confirmacion := ports.ConfirmacionRegistroContextoActorV2{
		OperacionRef: respuesta.operacionRef, RegistroContextoRef: respuesta.reciboRef,
		Contexto: contexto, RepresentacionCanonica: append([]byte(nil), respuesta.representacion...),
		HuellaSHA256:                      respuesta.huella,
		ManifiestoProcedenciaCanonico:     append([]byte(nil), respuesta.manifiesto...),
		ManifiestoProcedenciaHuellaSHA256: respuesta.huellaManifiesto,
		AutoridadEfectiva:                 domain.AutoridadProcedenciaContextoActorV1(respuesta.autoridadEfectiva),
		ResueltoEnAutoritativo:            respuesta.resueltoEn,
	}
	if confirmacion.ValidarParaProductiva(solicitud) != nil {
		return ports.ConfirmacionRegistroContextoActorV2{}, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	return confirmacion, nil
}

func respuestasContextoActorIguales(a, b respuestaContextoActorPostgreSQL) bool {
	return a.operacionRef == b.operacionRef && a.reciboRef == b.reciboRef &&
		bytes.Equal(a.representacion, b.representacion) && a.huella == b.huella &&
		bytes.Equal(a.manifiesto, b.manifiesto) && a.huellaManifiesto == b.huellaManifiesto &&
		a.autoridadEfectiva == b.autoridadEfectiva &&
		a.resueltoEn.Equal(b.resueltoEn)
}

func prepararTransaccionContextoActorPostgreSQL(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '4s', true),
		       set_config('statement_timeout', '8s', true),
		       set_config('idle_in_transaction_session_timeout', '10s', true)`)
	return err
}

func revertirContextoActorPostgreSQL(tx pgx.Tx) {
	if tx == nil {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelar()
	_ = tx.Rollback(ctx)
}

func errorResolutorContextoActorPostgreSQL(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return ports.ErrResolutorRegistroContextoActorNoDisponible
}

func valorNuloContextoActorPostgreSQL(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}

var _ ports.ResolutorRegistroContextoActorV2 = (*ResolutorRegistroContextoActorPostgreSQLV2)(nil)
