package adminperfiles

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	ca "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	h "vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const registrarContexto = `SELECT vec_contexto_actor_v1.registrar_contexto_admin_v1($1,$2,$3,$4,$5,$6,$7,$8,$9::numeric,$10,$11,$12,$13,$14,$15)`
const recuperarContexto = `SELECT vec_contexto_actor_v1.recuperar_contexto_admin_v1($1,$2,$3,$4,$5,$6,$7,$8,$9::numeric,$10,$11,$12,$13,$14,$15,$16::jsonb)`

type ConfiguracionContextoADMIN struct{ Proceso string }

var procesoContextoADMIN = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)

type resolutorContexto struct {
	base    *PostgreSQL
	proceso string
	login   string
}

// La firma heredada no acredita un pool CA segregado ni un proceso AD192.
// La composición usa el constructor tipado cuando esas dependencias existan.
func NuevoContextoRegistradoPostgreSQL(ctx context.Context, pool *pgxpool.Pool, reloj h.Reloj) (*application.AutoridadContextoActorRegistradoV2, error) {
	return nil, api.ErrConfiguracionIncompleta
}

func NuevoContextoRegistradoADMINPostgreSQL(ctx context.Context, pool *pgxpool.Pool, reloj h.Reloj,
	config ConfiguracionContextoADMIN) (*application.AutoridadContextoActorRegistradoV2, error) {
	return nuevoContextoRegistradoPostgreSQL(ctx, pool, reloj, config)
}

func nuevoContextoRegistradoPostgreSQL(ctx context.Context, pool transactor, reloj h.Reloj,
	config ConfiguracionContextoADMIN) (*application.AutoridadContextoActorRegistradoV2, error) {
	if ctx == nil || ctx.Err() != nil || nulo(pool) || nulo(reloj) || !procesoContextoADMIN.MatchString(config.Proceso) {
		return nil, api.ErrConfiguracionIncompleta
	}
	var login string
	var acreditada bool
	if err := pool.QueryRow(ctx, `SELECT identidad_login,acreditada FROM vec_contexto_actor_v1.acreditar_runtime_contexto_admin_v1()`).Scan(&login, &acreditada); err != nil || !acreditada || login == "" {
		return nil, api.ErrConfiguracionIncompleta
	}
	base := &PostgreSQL{pool: pool, reloj: reloj}
	// El pool de contexto acredita exclusivamente CA36; el constructor de IS
	// acredita su propio LOGIN y no se presta entre autoridades.
	servicio, err := application.NuevoServicioContextoActorProductivoV2(
		&resolutorContexto{base: base, proceso: config.Proceso, login: login}, ca.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
	if err != nil {
		return nil, err
	}
	return application.NuevaAutoridadContextoActorRegistradoV2(servicio)
}

func (r *resolutorContexto) ResolverYRegistrarContextoActorV2(ctx context.Context, solicitud ports.SolicitudResolucionRegistroContextoActorV2) (ports.ConfirmacionRegistroContextoActorV2, error) {
	var vacia ports.ConfirmacionRegistroContextoActorV2
	if r == nil || r.base == nil || nulo(r.base.pool) || nulo(r.base.reloj) ||
		r.login == "" || !procesoContextoADMIN.MatchString(r.proceso) ||
		ctx == nil || ctx.Err() != nil || solicitud.Validar() != nil ||
		!solicitud.Proyecciones.Vacio() ||
		(solicitud.Contexto.Cuenta.Metodo != domain.AuthMethodCertificate && solicitud.Contexto.Cuenta.Metodo != domain.AuthMethodDNIe) ||
		solicitud.Contexto.Cuenta.Garantia != domain.AuthAssuranceHigh {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	vinculo, err := VinculoSesionADMINDeContexto(ctx)
	ahora := r.base.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err != nil || vinculo.CuentaRef != solicitud.Contexto.Cuenta.CuentaRef ||
		vinculo.PerfilActivoRef != solicitud.Contexto.PerfilActivoRef ||
		vinculo.VinculadaEn.After(ahora) || !ahora.Before(vinculo.VigenteHasta) {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	var entropia [24]byte
	if _, err := rand.Read(entropia[:]); err != nil {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	recibo := "rca_" + base64.RawURLEncoding.EncodeToString(entropia[:])
	var aleatorioEvento [16]byte
	if _, err := rand.Read(aleatorioEvento[:]); err != nil {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	evento := "evento_" + hex.EncodeToString(aleatorioEvento[:])
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	args := []any{solicitud.OperacionRef, recibo, solicitud.Contexto.Cuenta.CuentaRef,
		solicitud.Contexto.PerfilActivoRef, string(solicitud.Contexto.Cuenta.Metodo),
		string(solicitud.Contexto.Cuenta.Garantia), solicitud.SolicitadoEn, vinculo.Referencia,
		strconv.FormatUint(vinculo.Version, 10), vinculo.HuellaSHA256, vinculo.AutenticacionRef,
		vinculo.SesionRef, evento, correlacionRef, r.proceso}
	respuesta, confirmada, incierto, err := r.ejecutar(ctx, registrarContexto, args, solicitud, vinculo,
		recibo, evento, correlacionRef)
	if err == nil && respuesta.Estado == "permitido" {
		return confirmada, nil
	}
	if !incierto {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	// La respuesta previa conserva el evento15 original sólo durante esta
	// llamada. Recuperar nunca añade otro evento ni devuelve un V2 ausente.
	recCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	argsRecuperar := append(append([]any(nil), args...), string(respuesta.Evento))
	argsRecuperar[12] = respuesta.EventoDTO.EventoRef
	recuperada, confirmacionRecuperada, _, recErr := r.ejecutar(recCtx, recuperarContexto,
		argsRecuperar, solicitud, vinculo, recibo, respuesta.EventoDTO.EventoRef, correlacionRef)
	if recErr != nil || !mismaRespuestaContextoADMIN(respuesta, recuperada, confirmada, confirmacionRecuperada) {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	if recuperada.Estado != "permitido" {
		return vacia, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	return confirmacionRecuperada, nil
}

func (r *resolutorContexto) ejecutar(ctx context.Context, consulta string, args []any,
	solicitud ports.SolicitudResolucionRegistroContextoActorV2, vinculo VinculoSesionADMIN,
	recibo, evento, correlacion string) (respuestaContextoADMIN, ports.ConfirmacionRegistroContextoActorV2, bool, error) {
	var respuesta respuestaContextoADMIN
	var resultado ports.ConfirmacionRegistroContextoActorV2
	iso := pgx.Serializable
	if consulta == recuperarContexto {
		iso = pgx.ReadCommitted
	}
	tx, err := r.base.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: iso, AccessMode: pgx.ReadWrite})
	if err != nil {
		return respuesta, resultado, false, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(c)
	}()
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','4s',true),set_config('statement_timeout','8s',true)`); err != nil {
		return respuesta, resultado, false, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	var bruto []byte
	err = tx.QueryRow(ctx, consulta, args...).Scan(&bruto)
	if err != nil {
		return respuesta, resultado, false, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	resultado, err = respuesta.validar(bruto, solicitud, vinculo, solicitud.OperacionRef, recibo,
		evento, correlacion, r.proceso, r.login)
	if err != nil || ctx.Err() != nil {
		return respuestaContextoADMIN{}, ports.ConfirmacionRegistroContextoActorV2{}, false, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	err = tx.Commit(ctx)
	if err != nil {
		return respuesta, resultado, true, ports.ErrResolutorRegistroContextoActorNoDisponible
	}
	return respuesta, resultado, false, nil
}

func mismaRespuestaContextoADMIN(a, b respuestaContextoADMIN,
	primera, recuperada ports.ConfirmacionRegistroContextoActorV2) bool {
	return a.Estado == b.Estado && a.MotivoRef == b.MotivoRef &&
		bytes.Equal(a.Evento, b.Evento) && bytes.Equal(a.Acuse, b.Acuse) &&
		(a.Estado != "permitido" || mismaConfirmacion(primera, recuperada))
}

func mismaConfirmacion(a, b ports.ConfirmacionRegistroContextoActorV2) bool {
	return a.OperacionRef == b.OperacionRef && a.RegistroContextoRef == b.RegistroContextoRef &&
		bytes.Equal(a.RepresentacionCanonica, b.RepresentacionCanonica) && a.HuellaSHA256 == b.HuellaSHA256 &&
		bytes.Equal(a.ManifiestoProcedenciaCanonico, b.ManifiestoProcedenciaCanonico) &&
		a.ManifiestoProcedenciaHuellaSHA256 == b.ManifiestoProcedenciaHuellaSHA256 &&
		a.AutoridadEfectiva == b.AutoridadEfectiva && a.ResueltoEnAutoritativo.Equal(b.ResueltoEnAutoritativo)
}

var _ ports.ResolutorRegistroContextoActorV2 = (*resolutorContexto)(nil)
