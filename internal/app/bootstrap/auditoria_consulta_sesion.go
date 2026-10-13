package bootstrap

import (
	"context"
	"net/http"
	"time"

	"vec-diputacion-granada/config"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func descriptoresFronterasAuditoriaSesionRRHH(perfil string) ([]descriptorFronteraComunDesarrollo, error) {
	if !perfilActivoSeguridadComunValido(perfil) {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	return []descriptorFronteraComunDesarrollo{
		{Clave: claveFronteraOpcionesAuditoriaRRHH, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodGet, Ruta: auditoria.RutaOpciones, PerfilesActivosRef: []string{perfil},
			ClavePolitica: clavePoliticaAuditoriaRRHH, ClaveCapacidad: claveCapacidadOpcionesAuditoria},
		{Clave: claveFronteraConsultaAuditoriaRRHH, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodPost, Ruta: auditoria.RutaConsulta, PerfilesActivosRef: []string{perfil},
			ClavePolitica: clavePoliticaAuditoriaRRHH, ClaveCapacidad: claveCapacidadConsultaAuditoria},
	}, nil
}

// Conserva los LOGIN y las fronteras nominales existentes. La asignación de
// consulta_auditoria_rrhh procede de ADMIN; el arranque sólo publica identidad
// y motivos, y no publica ni sustituye asignaciones o versiones de rol.
func nuevasRutasAuditoriaConsultaDesarrollo(ctx context.Context, cfg config.Config,
	alta *dependenciasAltaContratacionTemporalDesarrollo, soportes soportesAuditoriaConsultaDesarrollo,
	base *proveedorSesionConsultaRRHHDesarrollo, fronteras catalogoFronterasComunDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) ([]vechttp.RutaExacta, func(), error) {
	if ctx == nil || ctx.Err() != nil || alta == nil || soportes.CT == nil || base == nil ||
		alta.postgresql.gobierno == nil || dependenciaAuditoriaConsultaNula(reloj) {
		return nil, nil, auditoria.ErrNoDisponible
	}
	dsnCT, _, err := cfg.ContratacionTemporalPostgreSQL.DSNConsultasRRHHSeparados()
	if err != nil {
		return nil, nil, err
	}
	dsnFuente, err := cfg.DSNFuenteAutorizacionAuditoriaDesarrollo()
	if err != nil {
		return nil, nil, err
	}
	dsnMotivos, err := cfg.DSNMotivosAuditoriaDesarrollo()
	if err != nil {
		return nil, nil, err
	}
	sonda, cancelar := context.WithTimeout(ctx, 20*time.Second)
	defer cancelar()
	poolCT, err := abrirPoolConsultaAuditoriaCTDesarrollo(sonda, dsnCT)
	if err != nil {
		return nil, nil, err
	}
	poolFuente, err := abrirPoolAutoridadAuditoriaDesarrollo(sonda, dsnFuente, config.RolFuenteAutorizacionAuditoriaDesarrollo, "vec-auditoria-fuente")
	if err != nil {
		poolCT.Close()
		return nil, nil, err
	}
	poolMotivos, err := abrirPoolAutoridadAuditoriaDesarrollo(sonda, dsnMotivos, config.RolMotivosAuditoriaDesarrollo, "vec-auditoria-motivos")
	if err != nil {
		poolFuente.Close()
		poolCT.Close()
		return nil, nil, err
	}
	cerrar := func() { poolMotivos.Close(); poolFuente.Close(); poolCT.Close() }
	completa := false
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	fuente, err := postgresvec.NuevaFuenteAuditoriaRRHHPostgreSQL(poolCT)
	if err != nil {
		return nil, nil, err
	}
	autoridad, err := postgresvec.NuevoAlmacenAutorizacion(poolFuente)
	if err != nil {
		return nil, nil, err
	}
	validador, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(poolMotivos, catalogoMotivosAuditoriaConsultaDesarrollo)
	if err != nil {
		return nil, nil, err
	}
	opciones, err := configurarOpcionesAuditoriaConsultaDesarrollo(sonda, cfg, alta.postgresql.gobierno, validador, reloj)
	if err != nil || !opciones.SeleccionObjetivo {
		return nil, nil, auditoria.ErrNoDisponible
	}
	if err := publicarResultadoContextoPostgreSQLDesarrollo(sonda, alta.postgresql.gobierno,
		soportes.CT.contexto.Resultado, soportes.OperacionContextoCT); err != nil {
		return nil, nil, err
	}
	esperado, err := contextoEsperadoRegistradoDesarrollo(sonda, base.resolutor, soportes.CT)
	if err != nil {
		return nil, nil, err
	}
	soportes.CT.contextoEsperadoRegistrado = esperado
	sesion, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soportes.CT,
		base.registro, base.revalidador, reloj, base.resolutor, fronteras)
	if err != nil {
		return nil, nil, err
	}
	servicio, err := auditoria.NuevoServicioLecturaSesion(capturadorAuditoriaLecturaRRHH{fuente: autoridad}, fuente)
	if err != nil {
		return nil, nil, err
	}
	if err := servicio.ConfigurarIntentos(auditoria.ConfiguracionIntentos{
		Registrador: alta.auditoriaLecturasCT, Proceso: alta.procesoAuditoriaLecturasCT,
		Canal: string(vecdomain.SuperficieAutenticacionInternaCorporativaV1), Finalidad: opciones.FinalidadRef, Motivo: opciones.Motivo,
	}); err != nil {
		return nil, nil, err
	}
	proveedor, _, err := nuevoProveedorOpcionesAuditoriaConsultaDesarrollo(sonda, cfg, reloj, validador)
	if err != nil {
		return nil, nil, err
	}
	manejador, err := auditoria.NuevoManejador(servicio, proveedor, identidadAuditoriaSesionRRHH{proveedor: sesion})
	if err != nil {
		return nil, nil, err
	}
	completa = true
	return []vechttp.RutaExacta{{Ruta: auditoria.RutaOpciones, Manejador: manejador},
		{Ruta: auditoria.RutaConsulta, Manejador: manejador}}, cerrar, nil
}

type identidadAuditoriaSesionRRHH struct {
	proveedor *proveedorSesionConsultaRRHHDesarrollo
}

func (i identidadAuditoriaSesionRRHH) ResolverIdentidadConsulta(ctx context.Context, r *http.Request,
	fuente auditoria.FuenteConsulta,
) (auditoria.IdentidadResuelta, error) {
	if i.proveedor == nil || r == nil || r.URL == nil || ctx == nil || ctx.Err() != nil ||
		!((r.Method == http.MethodGet && r.URL.Path == auditoria.RutaOpciones && fuente == auditoria.FuenteConsultaGeneral) ||
			(r.Method == http.MethodPost && r.URL.Path == auditoria.RutaConsulta &&
				(fuente == auditoria.FuenteConsultaCT || fuente == auditoria.FuenteConsultaBolsa || fuente == auditoria.FuenteConsultaPersona || fuente == auditoria.FuenteConsultaGeneral))) {
		return auditoria.IdentidadResuelta{}, auditoria.ErrDenegada
	}
	resuelta, err := i.proveedor.ResolverContexto(ctx)
	if err != nil {
		return auditoria.IdentidadResuelta{}, err
	}
	if holder, ok := ctx.Value(claveActorAuditoriaLocal{}).(*actorAuditoriaLocal); ok {
		holder.fijar(resuelta.Resultado.Contexto.Principal.ID)
	}
	correlacion, ok := ctx.Value(claveCorrelacionAuditoriaLocal{}).(vecdomain.ReferenciaCorrelacionAutorizacionV2)
	if !ok || correlacion.Validar() != nil {
		correlacion, err = vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
		if err != nil {
			return auditoria.IdentidadResuelta{}, err
		}
	}
	return auditoria.IdentidadResuelta{Vinculo: resuelta.Vinculo, Resultado: resuelta.Resultado, Correlacion: correlacion}, nil
}
