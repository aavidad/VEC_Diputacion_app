package bootstrap

import (
	"context"
	"reflect"
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
	reloj relojContratacionTemporalDesarrollo, soporte *soporteAltaContratacionTemporalDesarrollo,
	fronteras catalogoFronterasComunDesarrollo,
) (*proveedorSesionConsultaRRHHDesarrollo, func(), error) {
	fallo := ports.ErrConsultaRRHHNoDisponible
	registroDSN, revalidacionDSN, err := c.DSNIdentidadConsultasSeparados()
	if err != nil || alta == nil || soporte == nil || alta.postgresql.gobierno == nil || derivador == nil || !derivador.valido() {
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
	vinculo, err := soporte.contexto.Vinculo.Datos()
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
	cuentaNominal, err := soporteCuentaNominalConsultasRRHHDesarrollo(alta, soporte)
	if err != nil {
		return nil, nil, fallo
	}
	if err = prepararCuentaNominalConsultasDesarrollo(ctx, alta.postgresql.gobierno, cuentaNominal, seudonimos); err != nil {
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
	esperado, err := contextoEsperadoRegistradoDesarrollo(ctx, autoridadContexto, soporte)
	if err != nil {
		return nil, nil, fallo
	}
	soporte.mu.Lock()
	if soporte.contextoEsperadoRegistrado.Validar() == nil &&
		!mismoContextoEsperadoRegistradoDesarrollo(soporte.contextoEsperadoRegistrado, esperado) {
		soporte.mu.Unlock()
		return nil, nil, fallo
	}
	soporte.contextoEsperadoRegistrado = esperado
	soporte.mu.Unlock()
	proveedor, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soporte, registro, revalidador, reloj, autoridadContexto, fronteras)
	if err != nil {
		return nil, nil, fallo
	}
	soporte.mu.Lock()
	if soporte.sesionOperativa != nil {
		soporte.mu.Unlock()
		return nil, nil, fallo
	}
	soporte.sesionOperativa = proveedor
	soporte.mu.Unlock()
	// El error ya está saneado (sin datos de la base); se propaga tal cual.
	if err := configurarSesionesPerfilesFijosCTDesarrollo(ctx, soporte, proveedor); err != nil {
		return nil, nil, err
	}
	completa = true
	return proveedor, cerrar, nil
}

// El técnico lector comparte la cuenta y el certificado del alta de CT,
// aunque consulta con su perfil fijo. El preparador nominal genérico acredita
// el replay de la cuenta base; nunca se le presenta el perfil fijo como si
// fuese el perfil dinámico ni se registra otro alias o cuenta.
func soporteCuentaNominalConsultasRRHHDesarrollo(
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	lector *soporteAltaContratacionTemporalDesarrollo,
) (*soporteAltaContratacionTemporalDesarrollo, error) {
	if alta == nil || lector == nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	if !lector.tecnicoConsultaRRHH {
		return lector, nil
	}
	base := alta.soporte
	if base == nil || !lector.lectorConsultasRRHH || lector.principalID == "" ||
		lector.principalID != base.principalID || lector.certificadoSHA256 == "" ||
		lector.certificadoSHA256 != base.certificadoSHA256 ||
		lector.contexto.Vinculo.ValidarPara(lector.contexto.Resultado) != nil {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	baseVinculo, errBase := base.contexto.Vinculo.Datos()
	lectorVinculo, errLector := lector.contexto.Vinculo.Datos()
	identidadCertificada := lector.principalID + "\x00" + lector.certificadoSHA256
	perfilFijo := referenciaAltaContratacionTemporalDesarrollo("prf_", identidadCertificada+"\x00"+
		discriminadorPerfilFijoCTDesarrollo(clavePerfilFijoAltaCTDesarrollo).perfil)
	if errBase != nil || errLector != nil || baseVinculo.CuentaRef == "" ||
		baseVinculo.CuentaRef != lectorVinculo.CuentaRef ||
		baseVinculo.CuentaOrdinariaRef != lectorVinculo.CuentaOrdinariaRef ||
		baseVinculo.PrincipalID != lectorVinculo.PrincipalID ||
		base.contexto.Resultado.Contexto.PersonaRef != lector.contexto.Resultado.Contexto.PersonaRef ||
		lectorVinculo.PerfilActivoRef != perfilFijo ||
		lector.contexto.Resultado.Contexto.PerfilActivoRef != perfilFijo ||
		baseVinculo.PerfilActivoRef == perfilFijo {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	coincidencias := 0
	for _, fijo := range base.perfilesFijosRegistrados() {
		if fijo != nil && fijo.clave == clavePerfilFijoAltaCTDesarrollo &&
			fijo.perfilRef() == perfilFijo && reflect.DeepEqual(fijo.contexto, lector.contexto) {
			coincidencias++
		}
	}
	if coincidencias != 1 {
		return nil, ports.ErrConsultaRRHHNoDisponible
	}
	return base, nil
}
