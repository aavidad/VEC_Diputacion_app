package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
)

// La activación propia no altera los tres perfiles de oferta. Solo cuando la
// fuente confirma rol, motivo y ámbito se publica la clave técnica del hijo.
func nuevaAutoridadDerivacionNombramientoDesarrollo(ctx context.Context, cfg config.Config,
	alta *dependenciasAltaContratacionTemporalDesarrollo, base *preparadorBorradorLlamamientoDesarrollo,
	publico catalogoFronterasComunDesarrollo, derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*autoridadPerfilOfertaCTDesarrollo, func(), bool) {
	vacio := func() (*autoridadPerfilOfertaCTDesarrollo, func(), bool) { return nil, func() {}, false }
	if ctx == nil || alta == nil || alta.soporte == nil || base == nil || base.sesion == nil ||
		base.soporte == nil || base.soporte.soporteCanal != alta.soporte ||
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
	if !existe || !frontera.admitePerfil(alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef) ||
		!ok || proveedor == nil || proveedor.soporte != base.soporte.soporteCanal ||
		!proveedor.fronteras.mismaInstancia(publico) ||
		c.Derivacion.PerfilRef == alta.soporte.contexto.Resultado.Contexto.PerfilActivoRef ||
		c.Derivacion.PerfilRef == proveedor.base.Contexto.PerfilActivoRef {
		return vacio()
	}
	var sqlListo bool
	err = alta.postgresql.ejecucion.QueryRow(ctx, `SELECT
		to_regprocedure('vec_contratacion_temporal.derivar_nombramiento_ct234_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
		AND to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_derivacion_nombramiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL`).Scan(&sqlListo)
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
			"unidad_ref": ""}, obligaciones: []string{"auditar"}, ruta: httpinterno.RutaDerivacionNombramiento}
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

// La configuración y la autoridad de la oferta son optativas. Un fallo deja
// esta función cerrada y conserva el arranque y la selección singular.
func configurarSeleccionOfertaOpcionalCT(cfg config.Config,
	alta *dependenciasAltaContratacionTemporalDesarrollo,
	seleccion *ejecutorSeleccionLlamamientoDesarrollo,
	base *preparadorBorradorLlamamientoDesarrollo, fronteras catalogoFronterasComunDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo, reloj relojContratacionTemporalDesarrollo,
) func() {
	if seleccion == nil || seleccion.puente == nil || alta == nil {
		return func() {}
	}
	if !alta.postgresql.seleccionOferta {
		_, presente, err := leerActivacionOfertaCTDesarrollo(cfg)
		if presente || err != nil {
			slog.Warn("oferta CT cerrada", "etapa", "sonda")
		}
		return func() {}
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelar()
	autoridad, cerrar, disponible := nuevaAutoridadSeleccionOfertaCTDesarrollo(ctx, cfg,
		alta, base, fronteras, derivador, reloj)
	if cerrar == nil {
		cerrar = func() {}
	}
	if !disponible {
		cerrar()
		return func() {}
	}
	registro, err := postgresct.NuevoRegistroSeleccionOfertaPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		cerrar()
		slog.Warn("seleccion de oferta de CT no compuesta", "error", err)
		return func() {}
	}
	resolucion, err := postgresbolsa.NuevoRepositorioResolucionIntencionOfertaCTPostgreSQL(alta.postgresql.bolsa)
	if err != nil {
		cerrar()
		slog.Warn("recuperacion de oferta de CT no compuesta", "error", err)
		return func() {}
	}
	publicador := publicadorOfertaBolsaCTDesarrollo{puente: seleccion.puente,
		resolucion: resolucion, autoridadCierre: autoridad}
	descarte, err := postgresct.NuevoRegistroDescarteSeleccionOfertaPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		cerrar()
		slog.Warn("descarte de oferta de CT no compuesto", "error", err)
		return func() {}
	}
	servicio, err := application.NuevoServicioSeleccionOfertaCTRecuperable(
		preparadorSeleccionOfertaCTDesarrollo{puente: seleccion.puente, autoridad: autoridad},
		autoridad, registro, publicador, publicador, autoridad, descarte, reloj)
	if err != nil {
		cerrar()
		slog.Warn("seleccion de oferta de CT no compuesta", "error", err)
		return func() {}
	}
	estado, err := application.NuevoServicioConsultaEstadoSeleccionOfertaCT(
		preparadorConsultaEstadoSeleccionOfertaCTDesarrollo{autoridad: autoridad, lector: registro},
		autoridad, registro, seleccion.lector)
	if err != nil {
		cerrar()
		slog.Warn("consulta de oferta de CT no compuesta", "error", err)
		return func() {}
	}
	seleccion.ofertas = servicio
	seleccion.estadoOfertas = estado
	return cerrar
}
