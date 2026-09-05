package bootstrap

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

type dependenciasConsultasRRHHDesarrollo struct {
	cuadro  httpinterno.ConsultorCuadroRRHH
	detalle httpinterno.ConsultorDetalleRRHH
	cerrar  func()
}

// Compone las rutas, el guardián y los dos adaptadores existentes. No aplica
// migraciones ni construye otra publicación o una bandeja en memoria.
func nuevasDependenciasConsultasRRHHDesarrollo(
	cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo, reloj relojContratacionTemporalDesarrollo,
) (dependenciasConsultasRRHHDesarrollo, error) {
	vacio := dependenciasConsultasRRHHDesarrollo{}
	c := cfg.ContratacionTemporalPostgreSQL
	consultaDSN, motivosDSN, err := c.DSNConsultasRRHHSeparados()
	if err != nil || alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	consultaConfig, errConsulta := pgxpool.ParseConfig(consultaDSN)
	motivosConfig, errMotivos := pgxpool.ParseConfig(motivosDSN)
	if errConsulta != nil || errMotivos != nil || consultaConfig.ConnConfig.User == motivosConfig.ConnConfig.User {
		return vacio, ports.ErrConsultaRRHHNoDisponible
	}
	ejecucion, gobierno, _ := c.DSNSeparados()
	confirmador, lector, _ := c.DSNCoberturaSeparados()
	registro, _ := c.DSNRegistroAutorizacionSeparado()
	anteriores := []string{ejecucion, gobierno, confirmador, lector, registro}
	if c.BolsaLlamamientosConfigurada() {
		bolsa, err := c.DSNBolsaLlamamientosSeparado()
		if err != nil {
			return vacio, ports.ErrConsultaRRHHNoDisponible
		}
		anteriores = append(anteriores, bolsa)
	}
	for _, dsn := range anteriores {
		anterior, err := pgxpool.ParseConfig(dsn)
		if err != nil || anterior.ConnConfig.User == consultaConfig.ConnConfig.User ||
			anterior.ConnConfig.User == motivosConfig.ConnConfig.User {
			return vacio, ports.ErrConsultaRRHHNoDisponible
		}
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	identidad, cerrarIdentidad, err := nuevasDependenciasIdentidadConsultasDesarrollo(ctx, c, alta, derivador, reloj)
	if err != nil {
		return vacio, err
	}
	identidadCompuesta := false
	defer func() {
		if !identidadCompuesta {
			cerrarIdentidad()
		}
	}()
	poolMotivos, err := postgresct.NuevoPoolResolucionMotivosRRHHPostgreSQL(ctx, motivosDSN, motivosConfig.ConnConfig.User)
	if err != nil {
		return vacio, err
	}
	poolConsultas, err := postgresct.NuevoPoolConsultasRRHHPostgreSQL(ctx, consultaDSN, consultaConfig.ConnConfig.User)
	if err != nil {
		poolMotivos.Cerrar()
		return vacio, err
	}
	var unaVez sync.Once
	cerrar := func() { unaVez.Do(func() { poolConsultas.Cerrar(); poolMotivos.Cerrar(); cerrarIdentidad() }) }
	completa := false
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	motivos, err := postgresct.NuevoResolutorMotivoConsultaRRHHPostgreSQL(poolMotivos)
	if err != nil {
		return vacio, err
	}
	motivoCuadro, err := motivos.ResolverMotivoCuadroRRHH(ctx, reloj.Ahora())
	if err != nil {
		return vacio, err
	}
	motivoDetalle, err := motivos.ResolverMotivoDetalleRRHH(ctx, reloj.Ahora())
	if err != nil {
		return vacio, err
	}
	autoridad, err := configurarAutoridadConsultasRRHHDesarrollo(alta, reloj, motivoCuadro, motivoDetalle)
	if err != nil {
		return vacio, err
	}
	if err = autoridad.configurarProveedorContextoConsultaRRHHDesarrollo(identidad); err != nil {
		return vacio, err
	}
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return vacio, err
	}
	defer material.borrarCopiasEfimeras()
	proveedorCuadro, err := nuevoProveedorMaterialConsumidorDesarrollo(ctx, alta.postgresql.gobierno, material, alta.soporte, reloj, ports.AudienciaConsumoConsultaCuadroRRHHV3)
	if err != nil {
		return vacio, err
	}
	proveedorDetalle, err := nuevoProveedorMaterialConsumidorDesarrollo(ctx, alta.postgresql.gobierno, material, alta.soporte, reloj, ports.AudienciaConsumoConsultaDetalleRRHHV3)
	if err != nil {
		return vacio, err
	}
	emisorCuadro, err := confianzaatestacion.NuevoEmisorMaterialAutorizacionAtestadaV3(
		autoridad, proveedorCuadro.atestador, proveedorCuadro.confianza, proveedorCuadro.emisor)
	if err != nil {
		return vacio, err
	}
	emisorDetalle, err := confianzaatestacion.NuevoEmisorMaterialAutorizacionAtestadaV3(
		autoridad, proveedorDetalle.atestador, proveedorDetalle.confianza, proveedorDetalle.emisor)
	if err != nil {
		return vacio, err
	}
	emisor, err := ports.NuevoEmisorMaterialConsultaRRHH(motivos, seguridadvec.GeneradorReferenciasCriptograficas{}, reloj, emisorCuadro, emisorDetalle)
	if err != nil {
		return vacio, err
	}
	sesion, err := postgresct.NuevaSesionConsultaRRHHPostgreSQL(poolConsultas)
	if err != nil {
		return vacio, err
	}
	cuadro, err := application.NuevoServicioConsultaCuadroRRHH(autoridad, emisor, sesion, reloj)
	if err != nil {
		return vacio, err
	}
	detalle, err := application.NuevoServicioConsultaDetalleRRHH(autoridad, emisor, sesion, reloj)
	if err != nil {
		return vacio, err
	}
	completa = true
	identidadCompuesta = true
	return dependenciasConsultasRRHHDesarrollo{cuadro: cuadro, detalle: detalle, cerrar: cerrar}, nil
}
