package bootstrap

import (
	"context"
	"net/http"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

type identidadSesionAuditoriaConsultaDesarrollo struct {
	proveedor *proveedorSesionConsultaRRHHDesarrollo
	fuente    auditoria.FuenteConsulta
	ruta      string
	metodo    string
}

func (i identidadSesionAuditoriaConsultaDesarrollo) ResolverIdentidadConsulta(ctx context.Context, r *http.Request, fuente auditoria.FuenteConsulta) (auditoria.IdentidadResuelta, error) {
	if ctx == nil || ctx.Err() != nil || r == nil || r.URL == nil || i.proveedor == nil ||
		fuente != i.fuente || r.URL.Path != i.ruta || r.Method != i.metodo {
		return auditoria.IdentidadResuelta{}, auditoria.ErrDenegada
	}
	resuelta, err := i.proveedor.ResolverContexto(ctx)
	if err != nil || resuelta.Resultado.Validar() != nil || resuelta.Vinculo.ValidarPara(resuelta.Resultado) != nil {
		return auditoria.IdentidadResuelta{}, auditoria.ErrDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return auditoria.IdentidadResuelta{}, auditoria.ErrNoDisponible
	}
	return auditoria.IdentidadResuelta{Vinculo: resuelta.Vinculo, Resultado: resuelta.Resultado, Correlacion: correlacion}, nil
}

func nuevasRutasAuditoriaConsultaDesarrollo(
	ctx context.Context, cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo,
	soportes soportesAuditoriaConsultaDesarrollo, identidadBase *proveedorSesionConsultaRRHHDesarrollo,
	fronteras catalogoFronterasComunDesarrollo, reloj relojContratacionTemporalDesarrollo,
) ([]vechttp.RutaExacta, func(), error) {
	fallo := func(cerrar func()) ([]vechttp.RutaExacta, func(), error) {
		if cerrar != nil {
			cerrar()
		}
		return nil, nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	if ctx == nil || ctx.Err() != nil || alta == nil || alta.postgresql.gobierno == nil ||
		alta.postgresql.registroAutorizacion == nil || alta.postgresql.bolsa == nil ||
		alta.postgresql.proveedorMaterialAuditoriaCT == nil || alta.postgresql.proveedorMaterialAuditoriaBolsa == nil ||
		soportes.CT == nil || soportes.Bolsa == nil || identidadBase == nil || fronteras.identidad == nil {
		return fallo(nil)
	}
	ctRef, bolsaRef, err := cfg.ExpedientesAuditoriaConsultaDesarrollo()
	if err != nil {
		return fallo(nil)
	}
	dsnCT, dsnMotivos, err := cfg.ContratacionTemporalPostgreSQL.DSNConsultasRRHHSeparados()
	if err != nil {
		return fallo(nil)
	}
	dsnFuente, err := cfg.DSNFuenteAutorizacionAuditoriaDesarrollo()
	if err != nil {
		return fallo(nil)
	}
	sonda, cancelar := context.WithTimeout(ctx, 60*time.Second)
	defer cancelar()
	poolCT, err := abrirPoolConsultaAuditoriaCTDesarrollo(sonda, dsnCT)
	if err != nil {
		return fallo(nil)
	}
	poolFuente, err := abrirPoolAutoridadAuditoriaDesarrollo(sonda, dsnFuente, config.RolFuenteAutorizacionAuditoriaDesarrollo, "vec-auditoria-fuente")
	if err != nil {
		poolCT.Close()
		return fallo(nil)
	}
	poolMotivos, err := abrirPoolAutoridadAuditoriaDesarrollo(sonda, dsnMotivos, "vec_autorizacion_motivos_evaluador", "vec-auditoria-motivos")
	if err != nil {
		poolFuente.Close()
		poolCT.Close()
		return fallo(nil)
	}
	cerrar := func() { poolMotivos.Close(); poolFuente.Close(); poolCT.Close() }
	if preflightAuditoriaConsultaDesarrollo(sonda, poolFuente, poolMotivos, alta.postgresql.bolsa) != nil {
		return fallo(cerrar)
	}
	for _, soporte := range []struct {
		base      *soporteAltaContratacionTemporalDesarrollo
		operacion string
	}{
		{soportes.CT, soportes.OperacionContextoCT}, {soportes.Bolsa, soportes.OperacionContextoBolsa},
	} {
		if publicarResultadoContextoPostgreSQLDesarrollo(sonda, alta.postgresql.gobierno, soporte.base.contexto.Resultado, soporte.operacion) != nil {
			return fallo(cerrar)
		}
		esperado, err := contextoEsperadoRegistradoDesarrollo(sonda, identidadBase.resolutor, soporte.base)
		if err != nil {
			return fallo(cerrar)
		}
		soporte.base.contextoEsperadoRegistrado = esperado
	}
	validador, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(poolMotivos, catalogoMotivosAuditoriaConsultaDesarrollo)
	if err != nil {
		return fallo(cerrar)
	}
	opciones, err := configurarOpcionesAuditoriaConsultaDesarrollo(sonda, cfg, alta.postgresql.gobierno, validador, reloj)
	if err != nil {
		return fallo(cerrar)
	}
	for _, s := range []struct {
		base                       *soporteAltaContratacionTemporalDesarrollo
		fuente, perfil, expediente string
	}{
		{soportes.CT, "ct", soportes.PerfilCT, ctRef},
		{soportes.Bolsa, "bolsa", soportes.PerfilBolsa, bolsaRef},
	} {
		semilla, err := instantaneaAuditoriaConsultaNominalDesarrollo(s.base.principalID, s.perfil, s.fuente, s.expediente, opciones.FinalidadRef, reloj.Ahora())
		if err != nil || publicarInstantaneaAuditoriaConsultaDesarrollo(sonda, alta.postgresql.gobierno, s.base, semilla) != nil {
			return fallo(cerrar)
		}
	}
	identidadCT, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soportes.CT,
		identidadBase.registro, identidadBase.revalidador, reloj, identidadBase.resolutor, fronteras)
	if err != nil {
		return fallo(cerrar)
	}
	identidadBolsa, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soportes.Bolsa,
		identidadBase.registro, identidadBase.revalidador, reloj, identidadBase.resolutor, fronteras)
	if err != nil {
		return fallo(cerrar)
	}
	emisorCT, emisorBolsa, err := nuevosEmisoresAuditoriaConsultaDesarrollo(dependenciasAutoridadesAuditoriaConsultaDesarrollo{
		FuenteCT: poolFuente, FuenteBolsa: poolFuente,
		RegistroCT: alta.postgresql.registroAutorizacion, RegistroBolsa: alta.postgresql.registroAutorizacion,
		MotivosCT: poolMotivos, MotivosBolsa: poolMotivos,
		MaterialCT: alta.postgresql.proveedorMaterialAuditoriaCT, MaterialBolsa: alta.postgresql.proveedorMaterialAuditoriaBolsa,
		PerfilCT: soportes.PerfilCT, PerfilBolsa: soportes.PerfilBolsa,
		ExpedienteCT: ctRef, ExpedienteBolsa: bolsaRef, Opciones: opciones, Reloj: reloj,
	})
	if err != nil {
		return fallo(cerrar)
	}
	proveedorOpciones, _, err := nuevoProveedorOpcionesAuditoriaConsultaDesarrollo(sonda, cfg, reloj, validador)
	if err != nil {
		return fallo(cerrar)
	}
	rutas, err := nuevasRutasAuditoriaConsultaConIdentidadesRRHH(dependenciasIdentidadAuditoriaConsultaRRHH{
		PoolCT: poolCT, PoolBolsa: alta.postgresql.bolsa,
		EmisorCT: emisorCT, EmisorBolsa: emisorBolsa,
		IdentidadOpciones: identidadSesionAuditoriaConsultaDesarrollo{identidadCT, auditoria.FuenteConsultaGeneral, auditoria.RutaOpciones, http.MethodGet},
		IdentidadCT:       identidadSesionAuditoriaConsultaDesarrollo{identidadCT, auditoria.FuenteConsultaCT, auditoria.RutaConsulta, http.MethodPost},
		IdentidadBolsa:    identidadSesionAuditoriaConsultaDesarrollo{identidadBolsa, auditoria.FuenteConsultaBolsa, auditoria.RutaConsulta, http.MethodPost},
		Opciones:          proveedorOpciones,
	})
	if err != nil {
		return fallo(cerrar)
	}
	return rutas, cerrar, nil
}

func configurarOpcionesAuditoriaConsultaDesarrollo(ctx context.Context, cfg config.Config, gobierno *pgxpool.Pool,
	validador *postgresvec.ValidadorReferenciaMotivoPostgreSQLV2, reloj relojContratacionTemporalDesarrollo) (auditoria.Opciones, error) {
	ruta, err := cfg.RutaCatalogoAuditoriaConsultaDesarrollo()
	if err != nil || gobierno == nil || validador == nil {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	lector, err := reglas.NuevoResolutor(reglas.Configuracion{Consulta: consulta, Metadatos: consulta,
		CatalogoID: catalogoMotivosAuditoriaConsultaDesarrollo, ModuloID: auditoria.ModuloAutorizacion, Reloj: reloj})
	if err != nil {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	p, err := auditoria.NuevoProveedorOpcionesCatalogo(lector, validador)
	if err != nil {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	declaradas, err := p.Configuradas(ctx)
	if err != nil || !declaradas.EsEjemplo ||
		publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno,
			[]vecdomain.ReferenciaEntradaCatalogo{declaradas.Motivo}, reloj.Ahora()) != nil {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	_, actuales, err := nuevoProveedorOpcionesAuditoriaConsultaDesarrollo(ctx, cfg, reloj, validador)
	if err != nil || actuales.Motivo != declaradas.Motivo || actuales.FinalidadRef != declaradas.FinalidadRef {
		return auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	return actuales, nil
}

func publicarInstantaneaAuditoriaConsultaDesarrollo(ctx context.Context, gobierno *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo, semilla vecdomain.InstantaneaAutorizacion) error {
	if gobierno == nil || soporte == nil || semilla.Validar() != nil {
		return errAutoridadesAuditoriaConsultaDesarrollo
	}
	a := autoridadPostgreSQLDesarrollo{pool: gobierno, vinculo: soporte.contexto.Vinculo,
		prefijoBloqueo: "vec:auditoria:desarrollo:autorizacion:",
		actoControlRol: "acto:auditoria:desarrollo:control-rol:v1",
		actoAsignacion: "acto:auditoria:desarrollo:asignacion:v1",
		actoSesion:     "acto:auditoria:desarrollo:sesion:v1"}
	preparada, err := a.prepararInstantanea(ctx, semilla, true)
	if err != nil || preparada.Validar() != nil || preparada.AsignacionPerfil.PrincipalID != semilla.AsignacionPerfil.PrincipalID ||
		preparada.AsignacionPerfil.PerfilActivoRef != semilla.AsignacionPerfil.PerfilActivoRef ||
		!reflect.DeepEqual(preparada.VersionRol.Concesiones, semilla.VersionRol.Concesiones) ||
		!reflect.DeepEqual(preparada.AsignacionPerfil.Ambitos, semilla.AsignacionPerfil.Ambitos) {
		return errAutoridadesAuditoriaConsultaDesarrollo
	}
	if err := a.publicarInstantanea(ctx, preparada); err != nil {
		return errAutoridadesAuditoriaConsultaDesarrollo
	}
	return nil
}

func preflightAuditoriaConsultaDesarrollo(ctx context.Context, fuente, motivos, bolsa *pgxpool.Pool) error {
	if ctx == nil || fuente == nil || motivos == nil || bolsa == nil {
		return errAutoridadesAuditoriaConsultaDesarrollo
	}
	const sonda = `SELECT
	 to_regprocedure($1) IS NOT NULL AND to_regprocedure($2) IS NOT NULL
	 AND coalesce(has_function_privilege(session_user,to_regprocedure($1)::oid,'EXECUTE'),false)
	 AND NOT coalesce(has_function_privilege(session_user,to_regprocedure($2)::oid,'EXECUTE'),false)`
	const funcionFuente = "vec_autorizacion.obtener_instantanea(text,text)"
	const funcionMotivos = "vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)"
	for _, caso := range []struct {
		pool          *pgxpool.Pool
		propia, ajena string
	}{
		{fuente, funcionFuente, funcionMotivos},
		{motivos, funcionMotivos, funcionFuente},
		{bolsa, funcionConsultaAuditoriaBolsaDesarrollo, funcionConsultaAuditoriaCTDesarrollo},
	} {
		var valida bool
		if err := caso.pool.QueryRow(ctx, sonda, caso.propia, caso.ajena).Scan(&valida); err != nil || !valida {
			return errAutoridadesAuditoriaConsultaDesarrollo
		}
	}
	return nil
}
