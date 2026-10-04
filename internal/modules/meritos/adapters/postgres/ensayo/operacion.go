package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"strconv"
	"time"
	merpg "vec-diputacion-granada/internal/modules/meritos/adapters/postgres"
	merapp "vec-diputacion-granada/internal/modules/meritos/application"
	merports "vec-diputacion-granada/internal/modules/meritos/ports"
	ctxpg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	idpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	authpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func execute(ctx context.Context, c cryptoConfig, a actorConfig, o operation) (merports.ResultadoOperacion, error) {
	var output merports.ResultadoOperacion
	pools := make([]*pgxpool.Pool, 0, 7)
	defer func() {
		for _, p := range pools {
			p.Close()
		}
	}()
	open := func(login string) (*pgxpool.Pool, error) {
		p, err := pool(ctx, login, c.Passwords[login])
		if err == nil {
			pools = append(pools, p)
		}
		return p, err
	}
	cp, err := open(a.ContextLogin)
	if err != nil {
		return output, fmt.Errorf("context_pool: %w", err)
	}
	rp, err := open(a.RevalidationLogin)
	if err != nil {
		return output, fmt.Errorf("revalidation_pool: %w", err)
	}
	sp, err := open(a.SourceLogin)
	if err != nil {
		return output, fmt.Errorf("source_pool: %w", err)
	}
	dp, err := open(a.RegisterLogin)
	if err != nil {
		return output, fmt.Errorf("register_pool: %w", err)
	}
	mp, err := open(a.ReasonLogin)
	if err != nil {
		return output, fmt.Errorf("reason_pool: %w", err)
	}
	ep, err := open(a.RuntimeLogin)
	if err != nil {
		return output, fmt.Errorf("effect_pool: %w", err)
	}
	resolver, err := ctxpg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, cp)
	if err != nil {
		return output, fmt.Errorf("context_runtime: %w", err)
	}
	contextService, err := app.NuevoServicioContextoActorProductivoV2(resolver, ctxpg.NuevoGeneradorOperacionContextoActorV2Criptografico(), clock{})
	if err != nil {
		return output, err
	}
	authority, err := app.NuevaAutoridadContextoActorRegistradoV2(contextService)
	if err != nil {
		return output, err
	}
	revalidator, err := idpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, rp)
	if err != nil {
		return output, fmt.Errorf("revalidation_runtime: %w", err)
	}
	link, result, err := vd.CrearVinculoAutenticacionActorV2ConResultado(ctx, revalidator, vd.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: a.Authentication, SesionRef: a.Session}, authority, vd.SolicitudContextoActor{Cuenta: vd.CuentaAutenticadaContextoActor{CuentaRef: a.Account, Metodo: vd.AuthMethodCertificate, Garantia: vd.AuthAssuranceHigh}, PerfilActivoRef: a.Profile}, clock{})
	if err != nil {
		return output, fmt.Errorf("actor_link: %w", err)
	}
	command := o.Command
	if command.ActorRef != result.Contexto.PersonaRef {
		return output, errors.New("actor_mismatch")
	}
	canonical, err := command.RepresentacionCanonica()
	if err != nil {
		return output, fmt.Errorf("command: %w", err)
	}
	hash := sha256.Sum256(canonical)
	hashText := hex.EncodeToString(hash[:])
	correlation, err := vd.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return output, err
	}
	purpose, audience := merapp.EspecificacionAutorizacion(command.Accion)
	request, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: link, ReferenciaMotivo: command.Motivo, Accion: command.Accion, Finalidad: purpose, Correlacion: correlation, Recurso: vd.RecursoAutorizable{Referencia: command.Hecho.Referencia, ModuloID: "meritos", Tipo: "hecho", Ambitos: map[string]string{"persona_ref": command.Hecho.PersonaRef, "version_esperada": strconv.Itoa(command.VersionEsperada), "huella_comando_sha256": hashText}}})
	if err != nil {
		return output, err
	}
	source, err := authpg.NuevoAlmacenAutorizacion(sp)
	if err != nil {
		return output, err
	}
	registration, err := authpg.NuevoAlmacenAutorizacion(dp)
	if err != nil {
		return output, err
	}
	reasons, err := authpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(mp, command.Motivo.CatalogoID)
	if err != nil {
		return output, err
	}
	authorization, err := app.NuevoServicioAutorizacionSolicitudLigadaV3(source, registration, registration, reasons, clock{}, seguridad.GeneradorReferenciasCriptograficas{}, app.ConfiguracionServicioAutorizacion{})
	if err != nil {
		return output, err
	}
	cfg, key, err := trust(c)
	if err != nil {
		return output, err
	}
	defer clear(key)
	verification, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(cfg, clock{})
	if err != nil {
		return output, err
	}
	header := vd.CabeceraAtestacionAutorizacionV3{FormatoVersion: vd.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: c.RootID, Audiencia: c.Deployment}
	attestor, err := app.NuevoServicioAtestacionesAutorizacionV3(header, signer{key, header})
	if err != nil {
		return output, err
	}
	hmacKey, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(c.KeyID, c.KeyVersion, c.HMAC, c.Issuer, audience, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, c.KeyFrom, c.KeyUntil, time.Time{}, c.GovernmentRevision, c.GovernmentSHA)
	if err != nil {
		return output, err
	}
	emitter, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(hmacKey, clock{})
	if err != nil {
		return output, err
	}
	common, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(authorization, attestor, verification, emitter)
	if err != nil {
		return output, err
	}
	registry, err := merpg.NuevoRegistro(ep)
	if err != nil {
		return output, err
	}
	if command.Accion == "meritos.hecho.declarar" {
		observado := &registroDeclaracionObservado{registro: registry}
		service, err := merapp.NuevoServicio(common, observado, auditoriaIntentosPendiente{}, clock{})
		if err != nil {
			return output, err
		}
		receipt, err := service.Declarar(ctx, merapp.Solicitud{Vinculo: link, Contexto: result,
			Correlacion: correlation, Motivo: command.Motivo, ClaveIdempotencia: command.ClaveIdempotencia,
			VersionEsperada: command.VersionEsperada, FechaCorte: command.FechaCorte, Hecho: command.Hecho})
		if err != nil {
			if observado.confirmado && observado.resultado.Recibo == nil &&
				(errors.Is(err, merports.ErrConflictoVersion) || errors.Is(err, merports.ErrClaveReutilizada) || errors.Is(err, vd.ErrAutorizacionDenegada)) {
				return observado.resultado, nil
			}
			return output, err
		}
		if !observado.confirmado || observado.resultado.Recibo == nil || observado.resultado.Recibo.Referencia != receipt.Referencia {
			return output, merports.ErrRegistroNoDisponible
		}
		return observado.resultado, nil
	}
	decision, confirmation, exporter, err := common.EmitirMaterialAutorizacionAtestadaV3(ctx, request, result)
	if err != nil {
		return output, fmt.Errorf("real_v3_emission: %w", err)
	}
	material, err := exporter.ExportarMaterialParaConsumidor()
	if err != nil {
		return output, err
	}
	if command.Accion == "meritos.hecho.verificar" {
		return output, errors.New("verification_disabled")
	}
	order := merports.OrdenOperacion{Accion: command.Accion, ActorRef: command.ActorRef, ClaveIdempotencia: command.ClaveIdempotencia, HuellaComando: hashText, VersionEsperada: command.VersionEsperada, Hecho: command.Hecho, Motivo: command.Motivo, FechaCorte: command.FechaCorte, Autorizacion: merports.AutorizacionOperacion{Contexto: result, Solicitud: request, Decision: decision, Confirmacion: confirmation, Material: material}}
	return registry.EjecutarOperacion(ctx, order)
}

// Conserva el DTO del repositorio real después de COMMIT para la salida
// histórica del ensayo. No construye recibos ni ejecuta otra operación.
type registroDeclaracionObservado struct {
	registro   merports.Registro
	resultado  merports.ResultadoOperacion
	confirmado bool
}

func (r *registroDeclaracionObservado) EjecutarOperacion(ctx context.Context, orden merports.OrdenOperacion) (merports.ResultadoOperacion, error) {
	resultado, err := r.registro.EjecutarOperacion(ctx, orden)
	if err == nil {
		r.resultado, r.confirmado = resultado, true
	}
	return resultado, err
}

// El éxito de M1 incluye su auditoría en la transacción SQL. El ensayo no
// simula una auditoría común de fallos: si ésta se necesita, falla sin recibo.
type auditoriaIntentosPendiente struct{}

func (auditoriaIntentosPendiente) AppendAudit(context.Context, vd.AuditEntry) (vd.AuditEntry, error) {
	return vd.AuditEntry{}, merports.ErrRegistroNoDisponible
}
