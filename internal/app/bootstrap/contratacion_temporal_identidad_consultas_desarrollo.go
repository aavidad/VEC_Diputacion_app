package bootstrap

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	postgrescontexto "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	postgresidentidad "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
)

const (
	rolRegistroIdentidadConsultasDesarrollo     = "vec_identidad_sesiones_v1_registrador"
	rolRevalidacionIdentidadConsultasDesarrollo = "vec_identidad_sesiones_v1_revalidador"
	rolContextoActorConsultasDesarrollo         = "vec_contexto_actor_v1_runtime"
)

func nuevasDependenciasIdentidadConsultasDesarrollo(
	ctx context.Context, c config.ConfiguracionPostgreSQLContratacionTemporal,
	alta *dependenciasAltaContratacionTemporalDesarrollo, derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*proveedorSesionConsultaRRHHDesarrollo, func(), error) {
	fallo := ports.ErrConsultaRRHHNoDisponible
	registroDSN, revalidacionDSN, err := c.DSNIdentidadConsultasSeparados()
	if err != nil || alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil || derivador == nil || !derivador.valido() {
		return nil, nil, fallo
	}
	consultaDSN, motivosDSN, _ := c.DSNConsultasRRHHSeparados()
	contextoDSN, err := c.DSNContextoActorConsultasSeparado()
	if err != nil {
		return nil, nil, fallo
	}
	ejecutorDSN, gobiernoDSN, _ := c.DSNSeparados()
	confirmadorDSN, lectorDSN, _ := c.DSNCoberturaSeparados()
	autorizacionDSN, _ := c.DSNRegistroAutorizacionSeparado()
	dsns := []string{registroDSN, revalidacionDSN, contextoDSN, consultaDSN, motivosDSN, ejecutorDSN, gobiernoDSN, confirmadorDSN, lectorDSN, autorizacionDSN}
	if c.BolsaLlamamientosConfigurada() {
		bolsaDSN, err := c.DSNBolsaLlamamientosSeparado()
		if err != nil {
			return nil, nil, fallo
		}
		dsns = append(dsns, bolsaDSN)
	}
	usuarios := map[string]struct{}{}
	for _, dsn := range dsns {
		configuracion, err := pgxpool.ParseConfig(dsn)
		if err != nil || configuracion.ConnConfig.User == "" {
			return nil, nil, fallo
		}
		if _, repetido := usuarios[configuracion.ConnConfig.User]; repetido {
			return nil, nil, fallo
		}
		usuarios[configuracion.ConnConfig.User] = struct{}{}
	}
	poolRegistro, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, registroDSN, "vec-ct-desarrollo-identidad-registro", rolRegistroIdentidadConsultasDesarrollo)
	if err != nil {
		return nil, nil, fallo
	}
	poolRevalidacion, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, revalidacionDSN, "vec-ct-desarrollo-identidad-revalidacion", rolRevalidacionIdentidadConsultasDesarrollo)
	if err != nil {
		poolRegistro.Close()
		return nil, nil, fallo
	}
	poolContexto, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, contextoDSN, "vec-ct-desarrollo-contexto-actor", rolContextoActorConsultasDesarrollo)
	if err != nil {
		poolRevalidacion.Close()
		poolRegistro.Close()
		return nil, nil, fallo
	}
	var unaVez sync.Once
	cerrar := func() { unaVez.Do(func() { poolContexto.Close(); poolRevalidacion.Close(); poolRegistro.Close() }) }
	completa := false
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	seudonimizador := &seudonimizadorSesionDesarrollo{derivador: derivador}
	registro, err := postgresidentidad.NuevoRegistroSesionesPostgreSQL(ctx, poolRegistro, poolRevalidacion, seudonimizador, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if err != nil {
		return nil, nil, fallo
	}
	revalidador, err := postgresidentidad.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, poolRevalidacion)
	if err != nil {
		return nil, nil, fallo
	}
	vinculo, err := alta.soporte.contexto.Vinculo.Datos()
	if err != nil {
		return nil, nil, fallo
	}
	// Solo se usan los digests de cuenta/sujeto para incorporar el alias; no
	// se registra una autenticación ni se consume una aserción al arrancar.
	seudonimos, err := seudonimizador.SeudonimizarAlta(ctx, postgresidentidad.IdentificadoresAlta{
		EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		AsercionID:       "preparacion-cuenta", SesionID: "preparacion-alias",
		CuentaID: "desarrollo:" + vinculo.CuentaRef, SujetoID: alta.soporte.principalID,
	})
	if err != nil {
		return nil, nil, fallo
	}
	if err = prepararCuentaNominalConsultasDesarrollo(ctx, alta.postgresql.gobierno, alta.soporte, seudonimos); err != nil {
		return nil, nil, fallo
	}
	resolutor, err := postgrescontexto.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, poolContexto)
	if err != nil {
		return nil, nil, fallo
	}
	servicioContexto, err := aplicacionvec.NuevoServicioContextoActorProductivoV2(resolutor, postgrescontexto.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
	if err != nil {
		return nil, nil, fallo
	}
	autoridadContexto, err := aplicacionvec.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if err != nil {
		return nil, nil, fallo
	}
	proveedor, err := nuevoProveedorSesionConsultaRRHHDesarrollo(alta.soporte, registro, revalidador, reloj, autoridadContexto)
	if err != nil {
		return nil, nil, fallo
	}
	completa = true
	return proveedor, cerrar, nil
}
