package bootstrap

import (
	"context"
	"net/http"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
)

// nuevaRutaDerivacionNombramientoOpcionalDesarrollo compone la capacidad para
// el router principal. Una dependencia ausente devuelve un manejador 503 y
// permite que el resto del portal arranque.
func nuevaRutaDerivacionNombramientoOpcionalDesarrollo(cfg config.Config,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	base *preparadorBorradorLlamamientoDesarrollo,
	detalle httpinterno.ConsultorDetalleRRHH,
	fronteras catalogoFronterasComunDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (http.Handler, func(), error) {
	cerrarNada := func() {}
	cerrada := func() (http.Handler, func(), error) {
		h, err := nuevaRutaDerivacionNombramientoDesarrollo(nil)
		return h, cerrarNada, err
	}
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(detalle) || fronteras.identidad == nil {
		return cerrada()
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelar()
	perfil, cerrar, disponible := nuevaAutoridadDerivacionNombramientoDesarrollo(ctx, cfg,
		alta, base, fronteras, derivador, reloj)
	if !disponible {
		return cerrada()
	}
	registro, err := postgresct.NuevoRegistroDerivacionNombramientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		cerrar()
		return cerrada()
	}
	baseAutoridad := autoridadSeleccionOfertaCTDesarrollo{alta: alta, publico: fronteras,
		vinculo: perfil, reloj: reloj}
	servicio, err := application.NuevoServicioDerivacionNombramiento(
		preparadorDerivacionNombramientoDesarrollo{soporte: alta.soporte, fronteras: fronteras, detalle: detalle},
		autorizadorDerivacionNombramientoDesarrollo{perfil: perfil, base: baseAutoridad}, registro)
	if err != nil {
		cerrar()
		return cerrada()
	}
	h, err := nuevaRutaDerivacionNombramientoDesarrollo(servicio)
	if err != nil {
		cerrar()
		return cerrada()
	}
	return h, cerrar, nil
}

func nuevaAutoridadDerivacionNombramientoDesarrollo(ctx context.Context, cfg config.Config,
	alta *dependenciasAltaContratacionTemporalDesarrollo, base *preparadorBorradorLlamamientoDesarrollo,
	publico catalogoFronterasComunDesarrollo, derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*autoridadPerfilOfertaCTDesarrollo, func(), bool) {
	vacio := func() (*autoridadPerfilOfertaCTDesarrollo, func(), bool) { return nil, func() {}, false }
	if ctx == nil || alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil ||
		base == nil || base.sesion == nil || base.soporte == nil ||
		base.soporte.soporteCanal != alta.soporte ||
		alta.postgresql.gobierno == nil || alta.postgresql.registroAutorizacion == nil ||
		derivador == nil || !derivador.valido() {
		return vacio()
	}
	c, presente, err := leerActivacionNombramientoCTDesarrollo(cfg)
	if err != nil || !presente {
		return vacio()
	}
	frontera, existe := publico.resolver(http.MethodPost, httpinterno.RutaDerivacionNombramiento)
	proveedor, ok := base.sesion.sesion.(*proveedorSesionConsultaRRHHDesarrollo)
	if !existe || frontera.Clave != "ct-nombramiento-derivar" ||
		frontera.ClaveCapacidad != string(domain.AccionDerivarNombramiento) ||
		!frontera.admitePerfil(alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef) ||
		!ok || proveedor == nil || proveedor.soporte != base.soporte.soporteCanal ||
		!proveedor.fronteras.mismaInstancia(publico) ||
		c.Derivacion.PerfilRef == alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef ||
		c.Derivacion.PerfilRef == proveedor.base.Contexto.PerfilActivoRef {
		return vacio()
	}
	var sqlListo bool
	err = alta.postgresql.ejecucion.QueryRow(ctx, `SELECT
		EXISTS(SELECT 1 FROM pg_catalog.pg_attribute a
			JOIN pg_catalog.pg_type t ON t.typrelid=a.attrelid
			WHERE t.oid=to_regtype('vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1')
			AND a.attname='instruccion_compartida' AND a.atttypid='jsonb'::regtype AND NOT a.attisdropped)
		AND EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
			WHERE p.oid=to_regprocedure('vec_contratacion_temporal.rechazar_nombramiento_derivado_ct241()')
			AND position('CT247: publicación inicial con ámbito propio' IN p.prosrc)>0)
		AND
		coalesce(pg_catalog.has_function_privilege(session_user,
			to_regprocedure('vec_contratacion_temporal.derivar_nombramiento_ct234_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'), false)
		AND coalesce(pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',
			to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'), false)`).Scan(&sqlListo)
	if err != nil || !sqlListo {
		return vacio()
	}
	dsnFuente, err := leerDSNOfertaCTDesarrollo(cfg, c.FuenteAutorizacionDSNFile)
	if err != nil {
		return vacio()
	}
	dsnMotivos, err := leerDSNOfertaCTDesarrollo(cfg, c.MotivosAutorizacionDSNFile)
	if err != nil {
		return vacio()
	}
	fuentePool, err := abrirPoolAutoridadAuditoriaDesarrollo(ctx, dsnFuente,
		config.RolFuenteAutorizacionAuditoriaDesarrollo, "vec-ct-nombramientos-fuente")
	if err != nil {
		return vacio()
	}
	motivosPool, err := abrirPoolAutoridadAuditoriaDesarrollo(ctx, dsnMotivos,
		config.RolMotivosAuditoriaDesarrollo, "vec-ct-nombramientos-motivos")
	if err != nil {
		fuentePool.Close()
		return vacio()
	}
	cerrar := func() { motivosPool.Close(); fuentePool.Close() }
	if !loginPoolsOfertaCTSeparados(fuentePool, motivosPool,
		alta.postgresql.ejecucion, alta.postgresql.gobierno, alta.postgresql.registroAutorizacion, alta.postgresql.bolsa) {
		cerrar()
		return vacio()
	}
	fuente, err := pgvec.NuevoAlmacenAutorizacion(fuentePool)
	if err != nil {
		cerrar()
		return vacio()
	}
	registro, err := pgvec.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if err != nil {
		cerrar()
		return vacio()
	}
	politica := politicaPerfilOfertaCTDesarrollo{accion: string(domain.AccionDerivarNombramiento),
		modulo: "contratacion_temporal", tipo: ports.TipoRecursoDerivacionNombramiento,
		finalidad: "gestionar_contratacion_temporal", audiencia: ports.AudienciaDerivacionNombramiento,
		ambitos: map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo,
			"unidad_ref": ""}, obligaciones: []string{"auditar"},
		ruta: httpinterno.RutaDerivacionNombramiento, claveFrontera: "ct-nombramiento-derivar"}
	perfil, err := componerPerfilAutoridadOfertaCTDesarrollo(ctx, c.Derivacion, politica,
		false, alta, proveedor, fuente, registro, motivosPool, reloj)
	if err != nil {
		cerrar()
		return vacio()
	}
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil || alta.postgresql.proveedorMaterial == nil ||
		alta.postgresql.proveedorMaterial.fuenteConfianza == nil {
		cerrar()
		return vacio()
	}
	defer material.borrarCopiasEfimeras()
	material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
	perfil.material, err = nuevoProveedorMaterialConsumidorConDescriptorDesarrollo(ctx,
		alta.postgresql.gobierno, material, alta.soporte, reloj, descriptorMaterialDerivacionNombramientoCTDesarrollo())
	if err != nil {
		cerrar()
		return vacio()
	}
	return perfil, cerrar, true
}

// La composición aporta el caso de uso con su autoridad propia. Mientras no
// esté disponible, la ruta permanece cerrada sin tomar permisos del padre.
func nuevaRutaDerivacionNombramientoDesarrollo(e httpinterno.EjecutorDerivacionNombramiento) (http.Handler, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(e) {
		e = derivacionNombramientoNoCompuesta{}
	}
	return httpinterno.NuevoManejadorDerivacionNombramiento(e)
}

type derivacionNombramientoNoCompuesta struct{}

func (derivacionNombramientoNoCompuesta) Derivar(context.Context,
	ports.EntradaDerivacionNombramiento) (ports.ResultadoDerivacionNombramiento, error) {
	return ports.ResultadoDerivacionNombramiento{}, ports.ErrDerivacionNombramientoNoDisponible
}
