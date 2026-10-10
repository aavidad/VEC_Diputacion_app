package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	claveFronteraVinculoEmisionBolsaCT = "ct-bolsa-emision-vincular"
)

var errVinculoEmisionBolsaCTNoDisponible = errors.New("bootstrap: vinculo de emision Bolsa CT no disponible")

// La función de efecto y sus dos autoridades deben existir juntas. Una base
// anterior conserva CT operativo pero no publica esta ruta ni su audiencia.
func detectarVinculoEmisionBolsaCTDesarrollo(ctx context.Context, ejecucion *pgxpool.Pool) (bool, error) {
	if ctx == nil || ejecucion == nil {
		return false, errVinculoEmisionBolsaCTNoDisponible
	}
	const sql = `SELECT pg_catalog.to_regprocedure($1) IS NOT NULL
	  AND coalesce(pg_catalog.has_function_privilege(session_user,pg_catalog.to_regprocedure($1),'EXECUTE'),false)
	  AND pg_catalog.to_regprocedure($2) IS NOT NULL
	  AND pg_catalog.to_regprocedure($3) IS NOT NULL
	  AND coalesce(pg_catalog.has_function_privilege(session_user,pg_catalog.to_regprocedure($3),'EXECUTE'),false)`
	var disponible bool
	err := ejecucion.QueryRow(ctx, sql,
		"vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
		"vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_emision_bolsa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
		"vec_contratacion_temporal.leer_ambitos_vinculo_emision_bolsa_ct_v1(text,text,numeric,text,text,text,text)",
	).Scan(&disponible)
	if err != nil {
		return false, errVinculoEmisionBolsaCTNoDisponible
	}
	if !disponible {
		slog.Warn("vinculo de emision Bolsa CT cerrado: faltan CT201 o AD233, o el ejecutor carece de EXECUTE")
	}
	return disponible, nil
}

func descriptorMaterialVinculoEmisionBolsaDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        ports.AudienciaVincularEmisionBolsa,
		Dominio:          "vec.ct.vinculo-emision-bolsa.capacidad-v3",
		Prefijo:          "clave:capacidad:ct:vinculo-bolsa:",
		ProveedorNominal: proveedorMaterialContratacionTemporal,
	}
}

func fronteraVinculoEmisionBolsaDesarrollo(perfil string) descriptorFronteraComunDesarrollo {
	return fronteraContratacionTemporalDesarrollo(claveFronteraVinculoEmisionBolsaCT,
		ports.AccionVincularEmisionBolsa, httpinterno.RutaVinculosEmisionBolsa, []string{perfil})
}

func motivoVinculoEmisionBolsaDesarrollo() core.ReferenciaEntradaCatalogo {
	return core.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_vinculo_emision_bolsa_ct", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("vinculo-emision-bolsa-ct-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "vinculo-emision-bolsa-ct"),
	}
}

func prepararPerfilVinculoEmisionBolsaCTDesarrollo(ctx context.Context, cfg config.Config,
	alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo,
	origen *origenConsultasContratacionTemporalDesarrollo) error {
	if alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil ||
		!alta.postgresql.vinculoEmisionBolsa || ctx == nil {
		return errVinculoEmisionBolsaCTNoDisponible
	}
	s := alta.soporte
	// La asignación inicial del perfil de alta ya existe. Esta ampliación no
	// publica permisos nuevos por ausencia de fila: sustituye la preimagen
	// exacta sólo cuando la administración aporta su huella y aprobación CAS.
	perfil := s.perfilFijoParaRuta(httpinterno.RutaAltaSolicitudes)
	if perfil == nil || perfil.clave != clavePerfilFijoAltaCTDesarrollo {
		return errVinculoEmisionBolsaCTNoDisponible
	}
	centrosPeticion, err := origen.centrosOrganizacionPeticion()
	if err != nil {
		return err
	}
	s.mu.Lock()
	previa, err := plantillaAltaConVinculoBolsaCT(perfil.plantilla, nil)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	plantilla, err := plantillaAltaConVinculoBolsaCT(perfil.plantilla, centrosPeticion)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	perfil.plantillaVinculoBolsa = &plantilla
	perfil.plantillaVinculoBolsaPrevia = &previa
	perfil.rutas[httpinterno.RutaVinculosEmisionBolsa] = struct{}{}
	s.mu.Unlock()
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if !vigente {
		return errVinculoEmisionBolsaCTNoDisponible
	}
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
		[]core.ReferenciaEntradaCatalogo{motivoVinculoEmisionBolsaDesarrollo()}, desde); err != nil {
		return err
	}
	perfilActualizado := *perfil
	perfilActualizado.plantilla = plantilla
	return asegurarPerfilesFijosCTDesarrollo(ctx, alta.postgresql.gobierno, s,
		aprobacionProvisionPerfilesRRHHDesdeConfig(cfg), &perfilActualizado)
}

// plantillaAltaConVinculoBolsaCT amplía el rol de alta con la concesión de
// vínculo. centrosPeticion añade al ámbito de centro las claves originales de
// la organización (centro-520): un expediente que nace de una petición del
// centro conserva esa clave, no la adaptada del alta (centro:rpt:520), y el
// vínculo autoriza sobre el centro guardado en el expediente.
func plantillaAltaConVinculoBolsaCT(anterior core.InstantaneaAutorizacion, centrosPeticion []string) (core.InstantaneaAutorizacion, error) {
	plantilla := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(anterior)
	if plantilla.Validar() != nil || plantilla.VersionRol.Version != 1 || len(plantilla.VersionRol.Concesiones) != 1 ||
		plantilla.VersionRol.Concesiones[0].Accion != ports.AccionCrearSolicitud {
		return core.InstantaneaAutorizacion{}, errVinculoEmisionBolsaCTNoDisponible
	}
	if len(centrosPeticion) > 0 {
		ampliado := false
		ambitos := make([]core.AmbitoPerfil, len(plantilla.AsignacionPerfil.Ambitos))
		for i, ambito := range plantilla.AsignacionPerfil.Ambitos {
			ambito.Valores = append([]string(nil), ambito.Valores...)
			if ambito.Clave == "centro_ref" {
				ampliado = true
				for _, centro := range centrosPeticion {
					if !slices.Contains(ambito.Valores, centro) {
						ambito.Valores = append(ambito.Valores, centro)
					}
				}
			}
			ambitos[i] = ambito
		}
		if !ampliado {
			return core.InstantaneaAutorizacion{}, errVinculoEmisionBolsaCTNoDisponible
		}
		plantilla.AsignacionPerfil.Ambitos = ambitos
	}
	plantilla.VersionRol.Version = 2
	plantilla.VersionRol.Concesiones = append(plantilla.VersionRol.Concesiones, core.ConcesionRol{
		Accion: ports.AccionVincularEmisionBolsa, ModuloID: ports.ModuloContratacion,
		TipoRecurso: ports.TipoRecursoVinculoEmisionBolsa,
		Finalidades: []string{ports.FinalidadVinculoEmisionBolsa}, GarantiaMinima: core.AuthAssuranceHigh,
	})
	plantilla.AsignacionPerfil.VersionRolRef = plantilla.VersionRol.Referencia()
	plantilla.ControlVigenciaVersionRol.VersionRolRef = plantilla.VersionRol.Referencia()
	if plantilla.Validar() != nil {
		return core.InstantaneaAutorizacion{}, errVinculoEmisionBolsaCTNoDisponible
	}
	return plantilla, nil
}

// Alta conserva su permiso anterior hasta la provisión aprobada. Vínculo
// consume exclusivamente la versión nueva. La comparación se hace contra la
// asignación publicada actual; una revocación o fuente fallida deniega ambos.
func (s *soporteAltaContratacionTemporalDesarrollo) consumirPerfilAltaConVinculoBolsa(
	ctx context.Context, p *perfilFijoCTDesarrollo, ruta string,
) (core.InstantaneaAutorizacion, bool) {
	if s == nil || p == nil || ctx == nil || p.plantillaVinculoBolsa == nil {
		return core.InstantaneaAutorizacion{}, false
	}
	s.mu.Lock()
	lector, ok := s.autoridadAsignaciones.(lectorAsignacionPublicadaCTDesarrollo)
	s.mu.Unlock()
	if !ok || dependenciaEsNulaContratacionTemporalDesarrollo(lector) {
		return core.InstantaneaAutorizacion{}, false
	}
	publicada, encontrada, err := lector.leerAsignacionPublicada(ctx, p.perfilRef())
	if err != nil {
		slog.WarnContext(ctx, "ct_vinculo_perfil_no_disponible",
			"causa", causaFalloPostgreSQLCTDesarrollo(err))
		return core.InstantaneaAutorizacion{}, false
	}
	if !encontrada || publicada.actoAsignacion != actoAsignacionPerfilFijoCTDesarrollo {
		return core.InstantaneaAutorizacion{}, false
	}
	if actual, valida := instantaneaConsumible(publicada, *p.plantillaVinculoBolsa, s.reloj.Ahora()); valida {
		return actual, true
	}
	// La versión aprobada antes de ampliar los centros de petición sigue
	// sirviendo, con sus mismos permisos, hasta que la administración apruebe
	// la nueva por CAS: ni el alta ni el vínculo de un centro del alta se caen.
	if p.plantillaVinculoBolsaPrevia != nil {
		if actual, valida := instantaneaConsumible(publicada, *p.plantillaVinculoBolsaPrevia, s.reloj.Ahora()); valida {
			return actual, true
		}
	}
	if ruta == httpinterno.RutaAltaSolicitudes {
		return instantaneaConsumible(publicada, p.plantilla, s.reloj.Ahora())
	}
	return core.InstantaneaAutorizacion{}, false
}

func solicitudVinculoEmisionBolsaCTValida(d core.DatosSolicitudAutorizacionLigadaV3) bool {
	r := d.Recurso
	return d.Accion == ports.AccionVincularEmisionBolsa &&
		d.Finalidad == ports.FinalidadVinculoEmisionBolsa &&
		d.ReferenciaMotivo == motivoVinculoEmisionBolsaDesarrollo() &&
		r.ModuloID == ports.ModuloContratacion && r.Tipo == ports.TipoRecursoVinculoEmisionBolsa &&
		r.Referencia != "" &&
		len(r.Ambitos) == 3 && r.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		r.Ambitos["centro_ref"] != "" && r.Ambitos["categoria_ref"] != "" &&
		len(r.Atributos) == 1 && huellaSHA256ValidaContratacionTemporalDesarrollo(r.Atributos["material_sha256"])
}

type autoridadVinculoEmisionBolsaCTDesarrollo struct {
	alta      *dependenciasAltaContratacionTemporalDesarrollo
	proveedor *proveedorMaterialAltaContratacionTemporalDesarrollo
	lector    *postgresct.RepositorioVinculoEmisionBolsaPostgreSQL
	reloj     relojContratacionTemporalDesarrollo
}

var _ ports.AutorizadorVinculoEmisionBolsa = (*autoridadVinculoEmisionBolsaCTDesarrollo)(nil)
var _ httpinterno.AutoridadCanalSeguimiento = (*autoridadVinculoEmisionBolsaCTDesarrollo)(nil)

func (a *autoridadVinculoEmisionBolsaCTDesarrollo) contexto(ctx context.Context) (ports.ContextoAutorizacionAltaV3, error) {
	if a == nil || a.alta == nil || a.alta.soporte == nil || ctx == nil || ctx.Err() != nil {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	capacidad, valida := a.alta.soporte.capacidadValida(ctx)
	if !valida || capacidad.ruta != httpinterno.RutaVinculosEmisionBolsa || capacidad.metodo != http.MethodPost ||
		!certificadoConsultaReciboRespuestaVigente(capacidad, a.reloj.Ahora()) {
		return ports.ContextoAutorizacionAltaV3{}, ports.ErrAutorizacionDenegada
	}
	return a.alta.soporte.contextoOperativoDesarrollo(ctx)
}

func (a *autoridadVinculoEmisionBolsaCTDesarrollo) ResolverContextoCanalSeguimiento(ctx context.Context) (application.ContextoCanalSeguimiento, error) {
	operativo, err := a.contexto(ctx)
	if err != nil {
		return application.ContextoCanalSeguimiento{}, ports.ErrAutorizacionDenegada
	}
	v, err := operativo.Vinculo.Datos()
	if err != nil {
		return application.ContextoCanalSeguimiento{}, ports.ErrAutorizacionDenegada
	}
	return application.ContextoCanalSeguimiento{AutenticacionRef: v.AutenticacionRef,
		SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef,
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo}, nil
}

func (a *autoridadVinculoEmisionBolsaCTDesarrollo) AutorizarVinculoEmisionBolsa(ctx context.Context,
	s ports.SolicitudVinculoEmisionBolsa, huella string) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacio vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if a == nil || a.proveedor == nil || a.lector == nil || a.alta == nil || a.alta.autorizador == nil || application.ValidarSolicitudVinculoEmisionBolsa(s) != nil ||
		s.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo || !huellaSHA256ValidaContratacionTemporalDesarrollo(huella) {
		return vacio, ports.ErrAutorizacionDenegada
	}
	operativo, err := a.contexto(ctx)
	if err != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	centro, categoria, err := a.lector.LeerAmbitosVinculoEmisionBolsa(ctx, s)
	if err != nil {
		return vacio, err
	}
	recurso, err := application.NuevoRecursoVinculoEmisionBolsa(s, huella, centro, categoria)
	if err != nil ||
		!solicitudVinculoEmisionBolsaCTValida(core.DatosSolicitudAutorizacionLigadaV3{
			Accion: ports.AccionVincularEmisionBolsa, Finalidad: ports.FinalidadVinculoEmisionBolsa,
			ReferenciaMotivo: motivoVinculoEmisionBolsaDesarrollo(), Recurso: recurso,
		}) {
		return vacio, ports.ErrAutorizacionDenegada
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, err
	}
	datos := core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: operativo.Vinculo,
		ReferenciaMotivo: motivoVinculoEmisionBolsaDesarrollo(), Accion: ports.AccionVincularEmisionBolsa,
		Recurso: recurso, Finalidad: ports.FinalidadVinculoEmisionBolsa, Correlacion: correlacion}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := a.alta.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		return vacio, errorAutorizacionVinculoEmisionBolsa(ctx, err)
	}
	return a.proveedor.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion,
		motivoVinculoEmisionBolsaDesarrollo(), operativo.Resultado)
}

// errorAutorizacionVinculoEmisionBolsa separa la denegación del PDP (403) de
// la indisponibilidad de sus fuentes y registros (503). Antes toda denegación
// V3 salía como 503 y ocultaba, por ejemplo, un centro fuera del ámbito.
func errorAutorizacionVinculoEmisionBolsa(ctx context.Context, err error) error {
	switch {
	case ctx != nil && ctx.Err() != nil:
		return ctx.Err()
	case errors.Is(err, vecports.ErrFuenteAutorizacionNoDisponible),
		errors.Is(err, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible),
		errors.Is(err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible):
		return ports.ErrVinculoEmisionBolsaNoDisponible
	default:
		return ports.ErrAutorizacionDenegada
	}
}

func nuevaRutaVinculoEmisionBolsaCTDesarrollo(ctx context.Context,
	alta *dependenciasAltaContratacionTemporalDesarrollo, derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo) (http.Handler, error) {
	if alta == nil || alta.soporte == nil || !alta.postgresql.vinculoEmisionBolsa ||
		alta.postgresql.ejecucion == nil || alta.postgresql.gobierno == nil ||
		alta.postgresql.proveedorMaterial == nil || derivador == nil || !derivador.valido() {
		return nil, errVinculoEmisionBolsaCTNoDisponible
	}
	sonda, cancelar := context.WithTimeout(ctx, plazoarranque.Ampliar(20*time.Second))
	defer cancelar()
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return nil, err
	}
	defer material.borrarCopiasEfimeras()
	material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
	proveedor, err := nuevoProveedorMaterialConsumidorDesarrollo(sonda, alta.postgresql.gobierno,
		material, alta.soporte, reloj, alta.postgresql.catalogoMaterial, ports.AudienciaVincularEmisionBolsa)
	if err != nil {
		return nil, err
	}
	repositorio, err := postgresct.NuevoRepositorioVinculoEmisionBolsaPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, err
	}
	autoridad := &autoridadVinculoEmisionBolsaCTDesarrollo{alta: alta, proveedor: proveedor, lector: repositorio, reloj: reloj}
	servicio, err := application.NuevoServicioVinculoEmisionBolsa(autoridad, repositorio)
	if err != nil {
		return nil, err
	}
	return httpinterno.NuevoManejadorVinculoEmisionBolsa(autoridad, servicio)
}
