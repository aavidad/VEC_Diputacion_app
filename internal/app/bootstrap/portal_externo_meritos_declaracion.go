package bootstrap

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
	meritospg "vec-diputacion-granada/internal/modules/meritos/adapters/postgres"
	meritosapp "vec-diputacion-granada/internal/modules/meritos/application"
	meritosports "vec-diputacion-granada/internal/modules/meritos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func (p preparadorCarteraExterna) PrepararDeclaracionCartera(r *http.Request) (meritosapp.Solicitud, error) {
	c, correlacion, err := p.contexto(r)
	if err != nil {
		return meritosapp.Solicitud{}, err
	}
	if p.motivoAUT.Validar() != nil || p.motivoAUT.CatalogoID != "motivos_meritos_declaracion" {
		return meritosapp.Solicitud{}, meritosports.ErrCarteraNoDisponible
	}
	return meritosapp.Solicitud{Contexto: c.resultado, Vinculo: c.vinculo, Correlacion: correlacion, Motivo: p.motivoAUT}, nil
}

// El registro de intentos queda ligado a la solicitud nominal de esta petición.
// Ningún estado de un actor se guarda en la instancia compartida del servicio.
type declaradorMeritosCartera struct {
	registro  meritosports.Registro
	emisor    meritosports.Autorizador
	auditoria *pgxpool.Pool
	reloj     vecports.Reloj
}

func (d *declaradorMeritosCartera) Declarar(ctx context.Context, s meritosapp.Solicitud) (meritosports.Recibo, error) {
	auditor, err := meritospg.NuevaAuditoriaDeclaracionCartera(d.auditoria, s)
	if err != nil {
		return meritosports.Recibo{}, err
	}
	servicio, err := meritosapp.NuevoServicio(d.emisor, d.registro, auditor, d.reloj)
	if err != nil {
		return meritosports.Recibo{}, err
	}
	return servicio.Declarar(ctx, s)
}

func nuevoDeclaradorMeritosCartera(ctx context.Context, cfg config.Config,
	topologia topologiaPostgreSQLPreferenciasUsuarios, auditoria *pgxpool.Pool,
	autorizador vecports.AutorizadorSolicitudLigadaV3,
	proveedor *proveedorMaterialAltaContratacionTemporalDesarrollo,
) (*declaradorMeritosCartera, *pgxpool.Pool, error) {
	dsn, err := cfg.ExternoMeritosOperacionPostgreSQL.DSN()
	if err != nil {
		return nil, nil, falloMontajeCartera("declaracion_dsn")
	}
	pool, login, err := abrirPoolNominalPortalExterno(ctx, dsn, "vec_meritos_ejecutor", "vec-portal-externo-meritos-operacion", "vec_meritos_externo")
	if err != nil {
		return nil, nil, falloMontajeCartera("declaracion_pool")
	}
	correcto := false
	defer func() {
		if !correcto {
			pool.Close()
		}
	}()
	if login != "vec_externo_meritos_operacion" || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil {
		return nil, nil, falloMontajeCartera("declaracion_login_topologia")
	}
	registro, err := meritospg.NuevoRegistro(pool)
	if err != nil {
		return nil, nil, falloMontajeCartera("declaracion_registro")
	}
	emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, proveedor)
	if err != nil {
		return nil, nil, falloMontajeCartera("declaracion_emisor")
	}
	correcto = true
	return &declaradorMeritosCartera{registro: registro, emisor: emisor, auditoria: auditoria, reloj: relojRutasDietas{}}, pool, nil
}

func montarDeclaracionMeritosCartera(ctx context.Context, cfg config.Config,
	externa *autoridadPreferenciasUsuariosDesarrollo, privada configuracionMotivoCarteraExterna,
	topologia topologiaPostgreSQLPreferenciasUsuarios, auditoria *pgxpool.Pool,
	autorizador vecports.AutorizadorSolicitudLigadaV3, proveedor *proveedorMaterialAltaContratacionTemporalDesarrollo,
) (*pgxpool.Pool, error) {
	declarador, pool, err := nuevoDeclaradorMeritosCartera(ctx, cfg, topologia, auditoria, autorizador, proveedor)
	if err != nil {
		return nil, err
	}
	externa.manejador, err = meritoshttp.NuevoHandlerDeclaracionesCartera(preparadorCarteraExterna{
		autoridad: externa, ruta: meritoshttp.RutaDeclaracionesCartera, motivoAUT: privada.MotivoDeclaracion}, declarador, relojRutasDietas{})
	if err != nil {
		pool.Close()
		return nil, falloMontajeCartera("declaracion_handler")
	}
	return pool, nil
}
