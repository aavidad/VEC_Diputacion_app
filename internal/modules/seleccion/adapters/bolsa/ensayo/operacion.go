package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	selauth "vec-diputacion-granada/internal/modules/seleccion/adapters/autorizacion"
	selpg "vec-diputacion-granada/internal/modules/seleccion/adapters/bolsa"
	selapp "vec-diputacion-granada/internal/modules/seleccion/application"
	selports "vec-diputacion-granada/internal/modules/seleccion/ports"
	ctxpg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	idpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	authpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func execute(ctx context.Context, c cryptoConfig, a actorConfig, o operation) (operationResult, error) {
	var output operationResult
	err := withRuntime(ctx, c, a, o, func(reader selports.LectorConvocatoriaExacta, solicitud selports.SolicitudConsultaConvocatoria) error {
		var err error
		output.Lectura, err = selapp.ConsultarConvocatoria(ctx, solicitud, reader)
		output.Codigo = "obtenida"
		switch err {
		case nil:
		case selports.ErrConsultaConvocatoriaDenegada:
			output.Codigo = "denegada"
		case selports.ErrConvocatoriaNoEncontrada:
			output.Codigo = "no_encontrada"
		default:
			return err
		}
		return nil
	})
	if err != nil {
		return operationResult{}, err
	}
	return output, nil
}

func withRuntime(ctx context.Context, c cryptoConfig, a actorConfig, o operation, use func(selports.LectorConvocatoriaExacta, selports.SolicitudConsultaConvocatoria) error) error {
	pools := make([]*pgxpool.Pool, 0, 7)
	defer func() {
		for _, p := range pools {
			p.Close()
		}
	}()
	open := func(login string) (*pgxpool.Pool, error) {
		p, err := pool(ctx, login)
		if err == nil {
			pools = append(pools, p)
		}
		return p, err
	}
	cp, err := open(a.ContextLogin)
	if err != nil {
		return fmt.Errorf("context_pool: %w", err)
	}
	rp, err := open(a.RevalidationLogin)
	if err != nil {
		return fmt.Errorf("revalidation_pool: %w", err)
	}
	sp, err := open(a.SourceLogin)
	if err != nil {
		return fmt.Errorf("source_pool: %w", err)
	}
	dp, err := open(a.RegisterLogin)
	if err != nil {
		return fmt.Errorf("register_pool: %w", err)
	}
	mp, err := open(a.ReasonLogin)
	if err != nil {
		return fmt.Errorf("reason_pool: %w", err)
	}
	ep, err := open(a.RuntimeLogin)
	if err != nil {
		return fmt.Errorf("effect_pool: %w", err)
	}
	resolver, err := ctxpg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, cp)
	if err != nil {
		return fmt.Errorf("context_runtime: %w", err)
	}
	contextService, err := app.NuevoServicioContextoActorProductivoV2(resolver, ctxpg.NuevoGeneradorOperacionContextoActorV2Criptografico(), clock{})
	if err != nil {
		return err
	}
	authority, err := app.NuevaAutoridadContextoActorRegistradoV2(contextService)
	if err != nil {
		return err
	}
	revalidator, err := idpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, rp)
	if err != nil {
		return fmt.Errorf("revalidation_runtime: %w", err)
	}
	link, result, err := vd.CrearVinculoAutenticacionActorV2ConResultado(ctx, revalidator, vd.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: a.Authentication, SesionRef: a.Session}, authority, vd.SolicitudContextoActor{Cuenta: vd.CuentaAutenticadaContextoActor{CuentaRef: a.Account, Metodo: vd.AuthMethodCertificate, Garantia: vd.AuthAssuranceHigh}, PerfilActivoRef: a.Profile}, clock{})
	if err != nil {
		return fmt.Errorf("actor_link: %w", err)
	}
	correlation, err := vd.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return err
	}
	solicitud := selports.SolicitudConsultaConvocatoria{Selector: o.Selector, Actor: result.Contexto, Correlacion: correlation}
	audience := selapp.AudienciaConsultaConvocatoriaV3
	source, err := authpg.NuevoAlmacenAutorizacion(sp)
	if err != nil {
		return err
	}
	registration, err := authpg.NuevoAlmacenAutorizacion(dp)
	if err != nil {
		return err
	}
	reasons, err := authpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(mp, o.Motivo.CatalogoID)
	if err != nil {
		return err
	}
	authorization, err := app.NuevoServicioAutorizacionSolicitudLigadaV3(source, registration, registration, reasons, clock{}, seguridad.GeneradorReferenciasCriptograficas{}, app.ConfiguracionServicioAutorizacion{})
	if err != nil {
		return err
	}
	cfg, key, err := trust(c)
	if err != nil {
		return err
	}
	defer clear(key)
	verification, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(cfg, clock{})
	if err != nil {
		return err
	}
	header := vd.CabeceraAtestacionAutorizacionV3{FormatoVersion: vd.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: c.RootID, Audiencia: c.Deployment}
	attestor, err := app.NuevoServicioAtestacionesAutorizacionV3(header, signer{key, header})
	if err != nil {
		return err
	}
	hmacKey, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(c.KeyID, c.KeyVersion, c.HMAC, c.Issuer, audience, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, c.KeyFrom, c.KeyUntil, time.Time{}, c.GovernmentRevision, c.GovernmentSHA)
	if err != nil {
		return err
	}
	emitter, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(hmacKey, clock{})
	if err != nil {
		return err
	}
	common, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(authorization, attestor, verification, emitter)
	if err != nil {
		return err
	}
	provider, err := selauth.NuevoProveedor(identidadActual{link, result}, common, o.Motivo)
	if err != nil {
		return err
	}
	repository, err := selpg.NuevoRepositorioVersiones(ep)
	if err != nil {
		return err
	}
	reader, err := selapp.NuevoLectorConvocatoriaV3(provider, repository)
	if err != nil {
		return err
	}
	return use(reader, solicitud)
}

type operationResult struct {
	Codigo  string                       `json:"codigo"`
	Lectura selports.LecturaConvocatoria `json:"lectura"`
}

// Solo conserva el resultado recién obtenido de los adaptadores reales de PG.
type identidadActual struct {
	vinculo   vd.VinculoAutenticacionActorV2
	resultado vd.ResultadoContextoActorRegistradoV2
}

func (i identidadActual) ResolverIdentidadConvocatoria(ctx context.Context) (selauth.IdentidadRegistrada, error) {
	if err := ctx.Err(); err != nil {
		return selauth.IdentidadRegistrada{}, err
	}
	return selauth.IdentidadRegistrada{Vinculo: i.vinculo, Resultado: i.resultado}, nil
}
