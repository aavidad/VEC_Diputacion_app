package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"errors"
	"net/http"
	"slices"
	"time"
	"vec-diputacion-granada/config"

	seleccioninterno "vec-diputacion-granada/internal/modules/seleccion/adapters/httpinterno"
	seleccionapp "vec-diputacion-granada/internal/modules/seleccion/application"
	seleccionports "vec-diputacion-granada/internal/modules/seleccion/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// RRHH consulta las solicitudes de Selección con un perfil propio del mismo
// certificado corporativo de desarrollo (discriminador distinto del de CT y
// del de Bolsa): su asignación y su rol no tocan los de otros módulos.
const clavePoliticaSeleccionRRHHDesarrollo = "politica-seleccion-rrhh"

func discriminadorContextoSinteticoSeleccionRRHHDesarrollo() discriminadorContextoSinteticoDesarrollo {
	return discriminadorContextoSinteticoDesarrollo{
		perfil: "perfil-seleccion-rrhh-v1", vinculo: "vinculo-seleccion-rrhh-v1",
		// Cuenta y persona son las del técnico de RRHH bajo la acreditación CT.
		procedencia: "procedencia", registro: "registro-contexto-seleccion-rrhh-v1",
		autenticacion: "autenticacion-seleccion-rrhh-v1", asercion: "asercion-seleccion-rrhh-v1",
		sesion: "sesion-seleccion-rrhh-v1", controlSesion: "control-sesion-seleccion-rrhh-v1",
		politicaGarantia: "politica-garantia-seleccion-rrhh-v1",
	}
}

// soporteSeleccionRRHHDesarrollo fija el canal de RRHH de Selección.
type soporteSeleccionRRHHDesarrollo struct {
	canal *soporteAltaContratacionTemporalDesarrollo
}

func (s *soporteSeleccionRRHHDesarrollo) perfilRef() string {
	if s == nil || s.canal == nil {
		return ""
	}
	return s.canal.contexto.Resultado.Contexto.PerfilActivoRef
}

// nuevoSoporteSeleccionRRHHDesarrollo construye el contexto sintético del
// perfil de RRHH de Selección a partir de la identidad del técnico de CT.
func nuevoSoporteSeleccionRRHHDesarrollo(soporteCT *soporteAltaContratacionTemporalDesarrollo, ahora time.Time) (*soporteSeleccionRRHHDesarrollo, error) {
	if soporteCT == nil {
		return nil, errSeleccionNoDisponible
	}
	soporteCT.mu.Lock()
	principalID, certificado, contextoCT, sello, reloj := soporteCT.principalID, soporteCT.certificadoSHA256, soporteCT.contexto, soporteCT.sello, soporteCT.reloj
	soporteCT.mu.Unlock()
	principal := dominiovec.Principal{ID: principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: dominiovec.AuthMethodCertificate, AuthAssurance: dominiovec.AuthAssuranceHigh,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment, "certificate_sha256": certificado}}
	contexto, err := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(principal, ahora, discriminadorContextoSinteticoSeleccionRRHHDesarrollo())
	if err != nil || sello == nil || !contextoSinteticoBolsaSeparadoDeCT(contextoCT, contexto) {
		return nil, errors.Join(errSeleccionNoDisponible, err)
	}
	return &soporteSeleccionRRHHDesarrollo{canal: &soporteAltaContratacionTemporalDesarrollo{
		sello: sello, principalID: principalID, certificadoSHA256: certificado, contexto: contexto, reloj: reloj}}, nil
}

// descriptoresFronterasSeleccionRRHHDesarrollo declara cada operación de RRHH
// en la frontera interna común con el perfil de Selección.
func descriptoresFronterasSeleccionRRHHDesarrollo(perfilRef string) []descriptorFronteraComunDesarrollo {
	nombres := map[string]string{seleccioninterno.RutaConvocatorias: "convocatorias", seleccioninterno.RutaConsultas: "listado", seleccioninterno.RutaDetalleConsultas: "detalle"}
	var descriptores []descriptorFronteraComunDesarrollo
	for _, o := range seleccioninterno.Operaciones() {
		clave := "seleccion-rrhh-" + nombres[o.Ruta]
		descriptores = append(descriptores, descriptorFronteraComunDesarrollo{Clave: clave, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: o.Metodo, Ruta: o.Ruta, PerfilesActivosRef: []string{perfilRef}, ClavePolitica: clavePoliticaSeleccionRRHHDesarrollo, ClaveCapacidad: "capacidad-" + clave})
	}
	return descriptores
}

func ambitosSeleccionRRHHDesarrollo() map[string]string {
	return map[string]string{"organizacion_ref": organizacionAltaContratacionTemporalDesarrollo}
}

// nuevaInstantaneaSeleccionRRHHDesarrollo: rol propio con las dos consultas
// de AD3-90, acotado a la organización.
func nuevaInstantaneaSeleccionRRHHDesarrollo(principalID, perfilRef string, ahora time.Time) (dominiovec.InstantaneaAutorizacion, error) {
	desde, hasta, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente || principalID == "" || perfilRef == "" {
		return dominiovec.InstantaneaAutorizacion{}, errSeleccionNoDisponible
	}
	concesiones := make([]dominiovec.ConcesionRol, 0, 2)
	for _, par := range seleccionports.AccionesRRHH() {
		concesiones = append(concesiones, dominiovec.ConcesionRol{Accion: par[0], ModuloID: seleccionports.ModuloSeleccion,
			TipoRecurso: seleccionports.TipoRecursoSolicitudes, Finalidades: []string{seleccionports.FinalidadConsultaSolicitudes},
			GarantiaMinima: dominiovec.AuthAssuranceHigh})
	}
	rol := dominiovec.VersionRol{RolID: "tecnico_rrhh_seleccion_desarrollo", Version: 1, Nombre: "Técnico de RRHH de Selección en desarrollo",
		Estado: dominiovec.EstadoVersionRolPublicada, Concesiones: concesiones, PublicadaPor: "seguridad:desarrollo:no-autoritativa", PublicadaEn: desde}
	asignacion := dominiovec.AsignacionPerfil{
		AsignacionID: referenciaAltaContratacionTemporalDesarrollo("asg_", principalID+"\x00"+perfilRef+"\x00seleccion-rrhh-v1"),
		Version:      1, PerfilActivoRef: perfilRef, PrincipalID: principalID, VersionRolRef: rol.Referencia(),
		Estado:       dominiovec.EstadoAsignacionPerfilActiva,
		Ambitos:      []dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}},
		VigenteDesde: desde, VigenteHasta: hasta, EmitidaPor: "identidad:desarrollo:no-autoritativa", EmitidaEn: desde,
	}
	huella, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, errors.Join(errSeleccionNoDisponible, err)
	}
	i := dominiovec.InstantaneaAutorizacion{AsignacionPerfil: asignacion, VersionRol: rol,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1,
			Estado: dominiovec.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: desde},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella}
	return i, i.Validar()
}

// politicaSeleccionRRHHDesarrollo es la fuente de autorización del perfil de
// RRHH de Selección: instantánea publicada al arrancar y registro durable de
// decisiones.
type politicaSeleccionRRHHDesarrollo struct {
	instantanea dominiovec.InstantaneaAutorizacion
	registro    registroDecisionesAnalisisContratacionTemporalDesarrollo
	motivo      dominiovec.ReferenciaEntradaCatalogo
}

func (p *politicaSeleccionRRHHDesarrollo) ObtenerInstantaneaAutorizacion(ctx context.Context, principal, perfil string) (dominiovec.InstantaneaAutorizacion, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || p.instantanea.Validar() != nil ||
		principal != p.instantanea.AsignacionPerfil.PrincipalID || perfil != p.instantanea.AsignacionPerfil.PerfilActivoRef {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	return clonarInstantaneaAutorizacionPostgreSQLDesarrollo(p.instantanea), nil
}

func (p *politicaSeleccionRRHHDesarrollo) ValidarReferenciaMotivoAutorizacionV2(ctx context.Context, motivo dominiovec.ReferenciaEntradaCatalogo, ahora time.Time) error {
	if p == nil || ctx == nil || ctx.Err() != nil || motivo != p.motivo || !p.instantanea.AsignacionPerfil.VigenteEn(ahora) {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

var errOrdenSeleccionRRHHDesarrollo = errors.New("bootstrap: orden de autorización de Selección RRHH no admitida")

// comprobarOrden exige que la orden sea del perfil de RRHH de Selección, de
// una de sus acciones, con su motivo y el ámbito de la organización.
func (p *politicaSeleccionRRHHDesarrollo) comprobarOrden(ctx context.Context, orden interface {
	Datos() (puertosvec.DatosOrdenRegistroAutorizacionLigadaV3, error)
}) error {
	if p == nil || ctx == nil || ctx.Err() != nil || orden == nil || p.instantanea.Validar() != nil {
		return errOrdenSeleccionRRHHDesarrollo
	}
	datos, err := orden.Datos()
	if err != nil {
		return err
	}
	if datos.ResultadoContexto.Validar() != nil || datos.ResultadoContexto.Contexto.Principal.ID != p.instantanea.AsignacionPerfil.PrincipalID ||
		datos.ResultadoContexto.Contexto.PerfilActivoRef != p.instantanea.AsignacionPerfil.PerfilActivoRef {
		return errOrdenSeleccionRRHHDesarrollo
	}
	solicitud, err := datos.Solicitud.Datos()
	if err != nil {
		return err
	}
	if _, esSeleccion := audienciaSeleccion(solicitud.Accion); !esSeleccion || esAccionSeleccionPropia(solicitud.Accion) ||
		datos.ReferenciaMotivo != p.motivo || solicitud.ReferenciaMotivo != p.motivo ||
		solicitud.Recurso.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo {
		return errOrdenSeleccionRRHHDesarrollo
	}
	return nil
}

func (p *politicaSeleccionRRHHDesarrollo) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx context.Context, orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	if err := p.comprobarOrden(ctx, orden); err != nil || p.registro == nil {
		return time.Time{}, errors.Join(puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible, err)
	}
	return p.registro.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
}

func (p *politicaSeleccionRRHHDesarrollo) RegistrarDenegacionAutorizacionLigadaV3(ctx context.Context, orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	if err := p.comprobarOrden(ctx, orden); err != nil || p.registro == nil {
		return errors.Join(puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible, err)
	}
	return p.registro.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
}

// autorizadorSeleccionRRHHDesarrollo liga cada decisión a la capacidad mTLS
// de la ruta de RRHH: acción de esa ruta, recurso de Selección y ámbito de la
// organización.
type autorizadorSeleccionRRHHDesarrollo struct {
	delegado *aplicacionvec.ServicioAutorizacionSolicitudLigadaV3
	soporte  *soporteSeleccionRRHHDesarrollo
}

func (a *autorizadorSeleccionRRHHDesarrollo) ExigirSolicitudLigadaV3(ctx context.Context, s dominiovec.SolicitudAutorizacionLigadaV3, r dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	var d dominiovec.DecisionAutorizacionLigadaV3
	var c puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	if a == nil || a.delegado == nil || a.soporte == nil || a.soporte.canal == nil || ctx == nil {
		return d, c, dominiovec.ErrAutorizacionDenegada
	}
	capacidad, ok := a.soporte.canal.capacidadValida(ctx)
	datos, err := s.Datos()
	var admitidas []string
	for _, o := range seleccioninterno.Operaciones() {
		if o.Ruta == capacidad.ruta && o.Accion != "" {
			admitidas = append(admitidas, o.Accion)
		}
	}
	if !ok || err != nil || !seleccioninterno.EsRuta(capacidad.ruta) || !slices.Contains(admitidas, datos.Accion) ||
		datos.Recurso.ModuloID != seleccionports.ModuloSeleccion || datos.Recurso.Tipo != seleccionports.TipoRecursoSolicitudes ||
		len(datos.Recurso.Ambitos) != 1 || datos.Recurso.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo ||
		r.Contexto.PerfilActivoRef != a.soporte.perfilRef() {
		return d, c, dominiovec.ErrAutorizacionDenegada
	}
	return a.delegado.ExigirSolicitudLigadaV3(ctx, s, r)
}

// preparadorSeleccionRRHHDesarrollo resuelve la sesión corporativa revalidada
// en la misma petición (puerto de identidad de RRHH).
type preparadorSeleccionRRHHDesarrollo struct {
	seguridad *seguridadComunDesarrollo
	perfilRef string
}

func (p *preparadorSeleccionRRHHDesarrollo) PrepararOrdenRRHH(r *http.Request) (seleccionapp.Orden, error) {
	if p == nil || p.seguridad == nil || r == nil {
		return seleccionapp.Orden{}, seleccionports.ErrNoDisponible
	}
	contexto, err := p.seguridad.ResolverContexto(r.Context())
	if err != nil || contexto.Resultado.Contexto.PerfilActivoRef != p.perfilRef {
		return seleccionapp.Orden{}, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(r.Context(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return seleccionapp.Orden{}, errors.Join(seleccionports.ErrNoDisponible, err)
	}
	return seleccionapp.Orden{ResultadoContexto: contexto.Resultado, Vinculo: contexto.Vinculo, Motivo: motivoSeleccionRRHHDesarrollo(),
		Correlacion: correlacion, Ambitos: ambitosSeleccionRRHHDesarrollo()}, nil
}

// publicarAutoridadSeleccionRRHHDesarrollo registra el contexto del perfil de
// RRHH de Selección, prepara y publica su rol y su asignación (la versión del
// rol se elige por contenido: un arranque repetido reutiliza la publicada) y
// publica su motivo con el instante fijo de la ventana sintética. Es
// idempotente: el segundo arranque no crea versiones.
func publicarAutoridadSeleccionRRHHDesarrollo(ctx context.Context, gobierno *pgxpool.Pool, soporte *soporteSeleccionRRHHDesarrollo, reloj relojContratacionTemporalDesarrollo) (dominiovec.InstantaneaAutorizacion, error) {
	var vacia dominiovec.InstantaneaAutorizacion
	if ctx == nil || gobierno == nil || soporte == nil || soporte.canal == nil {
		return vacia, errSeleccionNoDisponible
	}
	if err := publicarResultadoContextoPostgreSQLDesarrollo(ctx, gobierno, soporte.canal.contexto.Resultado,
		referenciaAltaContratacionTemporalDesarrollo("oca_", "seleccion-rrhh:registro-contexto:v1")); err != nil {
		return vacia, errors.Join(errSeleccionNoDisponible, err)
	}
	vinculo, err := soporte.canal.contexto.Vinculo.Datos()
	if err != nil {
		return vacia, errors.Join(errSeleccionNoDisponible, err)
	}
	autoridad := autoridadPostgreSQLDesarrollo{pool: gobierno, vinculo: soporte.canal.contexto.Vinculo,
		prefijoBloqueo: "vec:seleccion:rrhh:autorizacion:", actoControlRol: "acto:seleccion:rrhh:control-rol:v1",
		actoAsignacion: "acto:seleccion:rrhh:asignacion:v1", actoSesion: "acto:seleccion:rrhh:sesion:v1"}
	semilla, err := nuevaInstantaneaSeleccionRRHHDesarrollo(vinculo.PrincipalID, vinculo.PerfilActivoRef, reloj.Ahora())
	if err != nil || !autoridad.validaConfiguracion() {
		return vacia, errors.Join(errSeleccionNoDisponible, err)
	}
	preparada, err := autoridad.prepararInstantanea(ctx, semilla, true)
	if err != nil || preparada.Validar() != nil {
		return vacia, errors.Join(errSeleccionNoDisponible, err)
	}
	if err := autoridad.publicarInstantanea(ctx, preparada); err != nil {
		return vacia, errors.Join(errSeleccionNoDisponible, err)
	}
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if !vigente {
		return vacia, errSeleccionNoDisponible
	}
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, gobierno, []dominiovec.ReferenciaEntradaCatalogo{motivoSeleccionRRHHDesarrollo()}, desde); err != nil {
		return vacia, errors.Join(errSeleccionNoDisponible, err)
	}
	return preparada, nil
}

// componerSeleccionRRHHDesarrollo registra el contexto del perfil de RRHH de
// Selección, publica su rol (idempotente: reutiliza la versión si coincide) y
// su motivo con el instante fijo de la ventana sintética, y compone las rutas.
func componerSeleccionRRHHDesarrollo(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo, identidadCT *proveedorSesionConsultaRRHHDesarrollo,
	fronteras catalogoFronterasComunDesarrollo, soporte *soporteSeleccionRRHHDesarrollo, d *dependenciasSeleccionDesarrollo, reloj relojContratacionTemporalDesarrollo) ([]vechttp.RutaExacta, error) {
	if ctx == nil || alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil || identidadCT == nil || identidadCT.resolutor == nil ||
		soporte == nil || soporte.canal == nil || !d.valida() || alta.soporte.registroDecisionesAnalisis == nil {
		return nil, errSeleccionNoDisponible
	}
	gobierno := alta.postgresql.gobierno
	preparada, err := publicarAutoridadSeleccionRRHHDesarrollo(ctx, gobierno, soporte, reloj)
	if err != nil {
		return nil, err
	}
	esperado, err := contextoEsperadoRegistradoDesarrollo(ctx, identidadCT.resolutor, soporte.canal)
	if err != nil {
		return nil, errors.Join(errSeleccionNoDisponible, err)
	}
	soporte.canal.contextoEsperadoRegistrado = esperado
	sesion, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soporte.canal, identidadCT.registro, identidadCT.revalidador, identidadCT.reloj, identidadCT.resolutor, fronteras)
	if err != nil {
		return nil, errors.Join(errSeleccionNoDisponible, err)
	}
	seguridad, err := nuevaSeguridadComunDesarrollo(sesion, reloj)
	if err != nil {
		return nil, errors.Join(errSeleccionNoDisponible, err)
	}
	politica := &politicaSeleccionRRHHDesarrollo{instantanea: preparada, registro: alta.soporte.registroDecisionesAnalisis, motivo: motivoSeleccionRRHHDesarrollo()}
	autorizador, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(politica, politica, politica, politica, reloj,
		seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return nil, errors.Join(errSeleccionNoDisponible, err)
	}
	emisor, err := nuevoEmisorSeleccionDesarrollo(&autorizadorSeleccionRRHHDesarrollo{delegado: autorizador, soporte: soporte}, d.proveedores, seleccionports.AccionesRRHH())
	if err != nil {
		return nil, errors.Join(errSeleccionNoDisponible, err)
	}
	servicio, err := seleccionapp.NuevoServicioConsultaRRHH(d.repositorio, d.repositorio, emisor, d.kms, reloj)
	if err != nil {
		return nil, errors.Join(errSeleccionNoDisponible, err)
	}
	handler, err := seleccioninterno.Nuevo(&preparadorSeleccionRRHHDesarrollo{seguridad: seguridad, perfilRef: soporte.perfilRef()}, servicio)
	if err != nil {
		return nil, errors.Join(errSeleccionNoDisponible, err)
	}
	rutas := make([]vechttp.RutaExacta, 0, 3)
	for _, ruta := range seleccioninterno.Rutas() {
		rutas = append(rutas, vechttp.RutaExacta{Ruta: ruta, Manejador: handler})
	}
	return rutas, nil
}
