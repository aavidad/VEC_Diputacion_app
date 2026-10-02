package adminperfiles

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	ca "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	h "vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const registrarContexto = `SELECT operacion_ref,registro_contexto_ref,representacion_canonica,huella_sha256,
 manifiesto_procedencia_canonico,manifiesto_procedencia_huella_sha256,autoridad_efectiva,resuelto_en
 FROM vec_contexto_actor_v1.resolver_y_registrar_contexto_admin_perfiles_v1($1,$2,$3,$4)`
const recuperarContexto = `SELECT operacion_ref,registro_contexto_ref,representacion_canonica,huella_sha256,
 manifiesto_procedencia_canonico,manifiesto_procedencia_huella_sha256,autoridad_efectiva,resuelto_en
 FROM vec_contexto_actor_v1.reconciliar_contexto_admin_perfiles_v1($1,$2,$3,$4)`

type resolutorContexto struct{ base *PostgreSQL }

func NuevoContextoRegistradoPostgreSQL(ctx context.Context, pool *pgxpool.Pool, reloj h.Reloj) (*application.AutoridadContextoActorRegistradoV2, error) {
	base, err := NuevoPostgreSQL(ctx, pool, reloj)
	if err != nil {
		return nil, err
	}
	servicio, err := application.NuevoServicioContextoActorProductivoV2(
		&resolutorContexto{base: base}, ca.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
	if err != nil {
		return nil, err
	}
	return application.NuevaAutoridadContextoActorRegistradoV2(servicio)
}

func (r *resolutorContexto) ResolverYRegistrarContextoActorV2(ctx context.Context, solicitud ports.SolicitudResolucionRegistroContextoActorV2) (ports.ConfirmacionRegistroContextoActorV2, error) {
	var vacia ports.ConfirmacionRegistroContextoActorV2
	if r == nil || r.base == nil || ctx == nil || ctx.Err() != nil || solicitud.Validar() != nil ||
		!solicitud.Proyecciones.Vacio() || solicitud.Contexto.Cuenta.Metodo != domain.AuthMethodCertificate ||
		solicitud.Contexto.Cuenta.Garantia != domain.AuthAssuranceHigh {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	var entropia [24]byte
	if _, err := rand.Read(entropia[:]); err != nil {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	recibo := "rca_" + base64.RawURLEncoding.EncodeToString(entropia[:])
	args := []any{solicitud.OperacionRef, recibo, solicitud.Contexto.Cuenta.CuentaRef, solicitud.SolicitadoEn}
	confirmada, incierto, err := r.ejecutar(ctx, registrarContexto, args, solicitud)
	if err == nil {
		return confirmada, nil
	}
	if !incierto {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	// Una respuesta de COMMIT incierta solo admite recuperar exactamente el
	// registro anterior con la misma operación y el mismo recibo generado.
	recCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	recuperada, _, recErr := r.ejecutar(recCtx, recuperarContexto, args, solicitud)
	if recErr != nil || !mismaConfirmacion(confirmada, recuperada) {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	return recuperada, nil
}

func (r *resolutorContexto) ejecutar(ctx context.Context, consulta string, args []any,
	solicitud ports.SolicitudResolucionRegistroContextoActorV2) (ports.ConfirmacionRegistroContextoActorV2, bool, error) {
	var resultado ports.ConfirmacionRegistroContextoActorV2
	tx, err := r.base.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return resultado, false, err
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(c)
	}()
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true)`); err != nil {
		return resultado, false, err
	}
	var autoridad string
	err = tx.QueryRow(ctx, consulta, args...).Scan(&resultado.OperacionRef, &resultado.RegistroContextoRef,
		&resultado.RepresentacionCanonica, &resultado.HuellaSHA256,
		&resultado.ManifiestoProcedenciaCanonico, &resultado.ManifiestoProcedenciaHuellaSHA256,
		&autoridad, &resultado.ResueltoEnAutoritativo)
	if err != nil {
		return resultado, false, err
	}
	resultado.ResueltoEnAutoritativo = resultado.ResueltoEnAutoritativo.UTC().Truncate(time.Microsecond)
	resultado.AutoridadEfectiva = domain.AutoridadProcedenciaContextoActorV1(autoridad)
	resultado.Contexto, err = domain.RehidratarContextoActorVinculadoV2(resultado.RepresentacionCanonica)
	if err != nil || resultado.RegistroContextoRef != args[1] || resultado.ValidarParaProductiva(solicitud) != nil {
		return ports.ConfirmacionRegistroContextoActorV2{}, false, ports.ErrConfirmacionRegistroContextoActorV2Invalida
	}
	err = tx.Commit(ctx)
	return resultado, err != nil, err
}

func mismaConfirmacion(a, b ports.ConfirmacionRegistroContextoActorV2) bool {
	return a.OperacionRef == b.OperacionRef && a.RegistroContextoRef == b.RegistroContextoRef &&
		bytes.Equal(a.RepresentacionCanonica, b.RepresentacionCanonica) && a.HuellaSHA256 == b.HuellaSHA256 &&
		bytes.Equal(a.ManifiestoProcedenciaCanonico, b.ManifiestoProcedenciaCanonico) &&
		a.ManifiestoProcedenciaHuellaSHA256 == b.ManifiestoProcedenciaHuellaSHA256 &&
		a.AutoridadEfectiva == b.AutoridadEfectiva && a.ResueltoEnAutoritativo.Equal(b.ResueltoEnAutoritativo)
}

var _ ports.ResolutorRegistroContextoActorV2 = (*resolutorContexto)(nil)
