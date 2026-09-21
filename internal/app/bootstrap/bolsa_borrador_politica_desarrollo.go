package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominioct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible = errors.New(
	"bolsa: politica de borrador de llamamiento de desarrollo no disponible",
)

// autoridadInicialBorradorLlamamientoBolsaDesarrollo conserva la preparación
// inicial separada. Sólo la composición puede aportar la autoridad PostgreSQL
// común; la política no abre conexiones ni publica por su cuenta al construirse.
type autoridadInicialBorradorLlamamientoBolsaDesarrollo interface {
	prepararInstantanea(context.Context, dominiovec.InstantaneaAutorizacion, bool) (dominiovec.InstantaneaAutorizacion, error)
	publicarInstantaneaDesdePreimagen(context.Context, dominiovec.InstantaneaAutorizacion, dominiovec.InstantaneaAutorizacion) error
}

// politicaBorradorLlamamientoBolsaDesarrollo es la fuente nominal exclusiva de
// B-BACK. No contiene concesiones de Contratación ni interpreta datos del
// transporte: principal, perfil y ámbitos proceden del contexto Bolsa creado
// antes de atender peticiones.
type politicaBorradorLlamamientoBolsaDesarrollo struct {
	soporte     *soporteSesionBorradorBolsaDesarrollo
	autoridad   autoridadInicialBorradorLlamamientoBolsaDesarrollo
	registro    registroDecisionesAnalisisContratacionTemporalDesarrollo
	reloj       relojContratacionTemporalDesarrollo
	mu          sync.RWMutex
	publicada   bool
	instantanea dominiovec.InstantaneaAutorizacion
}

func nuevaPoliticaBorradorLlamamientoBolsaDesarrollo(
	soporte *soporteSesionBorradorBolsaDesarrollo,
	autoridad autoridadInicialBorradorLlamamientoBolsaDesarrollo,
	registro registroDecisionesAnalisisContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*politicaBorradorLlamamientoBolsaDesarrollo, error) {
	if soporte == nil || soporte.soporteCanal == nil || dependenciaAutorizacionComunDesarrolloNula(autoridad) || dependenciaAutorizacionComunDesarrolloNula(registro) ||
		soporte.unidadRef == "" || soporte.ambitoRef == "" ||
		soporte.soporteCanal.contexto.Resultado.Validar() != nil ||
		soporte.soporteCanal.contexto.Vinculo.ValidarPara(soporte.soporteCanal.contexto.Resultado) != nil {
		return nil, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	return &politicaBorradorLlamamientoBolsaDesarrollo{
		soporte: soporte, autoridad: autoridad, registro: registro, reloj: reloj,
	}, nil
}

// PublicarInicial prepara con permitirInicial=true y sólo conserva la copia
// después de que la autoridad durable la acepte. Debe invocarse una vez durante
// la composición, antes de exponer rutas.
func (p *politicaBorradorLlamamientoBolsaDesarrollo) PublicarInicial(ctx context.Context) error {
	if p == nil || ctx == nil || ctx.Err() != nil || p.soporte == nil || p.soporte.soporteCanal == nil ||
		dependenciaAutorizacionComunDesarrolloNula(p.autoridad) {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.publicada {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	datos, err := p.soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	ahora := p.reloj.Ahora()
	semilla, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrollo(
		datos.PrincipalID, datos.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef, ahora,
	)
	if err != nil {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	preimagen, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
		datos.PrincipalID, datos.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef, ahora, 1, false,
	)
	if err != nil {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	preparada, err := p.autoridad.prepararInstantanea(ctx, semilla, true)
	esperada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	esperada.AsignacionPerfil.Version = preparada.AsignacionPerfil.Version
	versionAdmitida := preparada.AsignacionPerfil.Version == 1 || preparada.AsignacionPerfil.Version == 2
	if err != nil || preparada.Validar() != nil || !versionAdmitida || !reflect.DeepEqual(preparada, esperada) ||
		p.autoridad.publicarInstantaneaDesdePreimagen(ctx, preparada, preimagen) != nil {
		return errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	p.instantanea = clonarInstantaneaAutorizacionPostgreSQLDesarrollo(preparada)
	p.publicada = true
	return nil
}

func nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrollo(
	principalID, perfilRef, unidadRef, ambitoRef string, ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, error) {
	return nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(principalID, perfilRef, unidadRef, ambitoRef, ahora, 2, true)
}

func nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
	principalID, perfilRef, unidadRef, ambitoRef string, ahora time.Time, versionRol int, incluirSituacion bool,
) (dominiovec.InstantaneaAutorizacion, error) {
	desde, hasta, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente || principalID == "" || perfilRef == "" || unidadRef == "" || ambitoRef == "" {
		return dominiovec.InstantaneaAutorizacion{}, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	concesion := func(accion, finalidad string) dominiovec.ConcesionRol {
		tipoRecurso := puertosbolsa.TipoRecursoBorradorLlamamiento
		if accion == puertosbolsa.AccionCambiarSituacionParticipacion {
			tipoRecurso = puertosbolsa.TipoRecursoSituacionParticipacion
		}
		return dominiovec.ConcesionRol{
			Accion: accion, ModuloID: puertosbolsa.ModuloBorradorLlamamiento,
			TipoRecurso: tipoRecurso,
			Finalidades: []string{finalidad}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}
	}
	concesiones := []dominiovec.ConcesionRol{
		concesion(puertosbolsa.AccionCrearBorradorLlamamientoInterno, puertosbolsa.FinalidadCrearBorradorLlamamientoInterno),
		concesion(puertosbolsa.AccionConsultarBorradorLlamamientoInterno, puertosbolsa.FinalidadConsultarBorradorLlamamientoInterno),
	}
	if incluirSituacion {
		concesiones = append(concesiones, concesion(puertosbolsa.AccionCambiarSituacionParticipacion, puertosbolsa.FinalidadCambiarSituacionParticipacion))
	}
	version := dominiovec.VersionRol{
		RolID: "tecnico_rrhh_borrador_llamamiento_bolsa_desarrollo", Version: versionRol,
		Nombre:       "Tecnico RRHH de borradores de llamamiento de desarrollo",
		Estado:       dominiovec.EstadoVersionRolPublicada,
		Concesiones:  concesiones,
		PublicadaPor: "seguridad:desarrollo:no-autoritativa", PublicadaEn: desde,
	}
	asignacion := dominiovec.AsignacionPerfil{
		AsignacionID: referenciaAltaContratacionTemporalDesarrollo("asg_", principalID+"\x00"+perfilRef+"\x00bolsa-bback-v1"),
		Version:      1, PerfilActivoRef: perfilRef, PrincipalID: principalID, VersionRolRef: version.Referencia(),
		Estado:       dominiovec.EstadoAsignacionPerfilActiva,
		Ambitos:      []dominiovec.AmbitoPerfil{{Clave: "unidad_ref", Valores: []string{unidadRef}}, {Clave: "ambito_ref", Valores: []string{ambitoRef}}},
		VigenteDesde: desde, VigenteHasta: hasta,
		EmitidaPor: "identidad:desarrollo:no-autoritativa", EmitidaEn: desde,
	}
	huella, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: asignacion, VersionRol: version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1, Estado: dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: desde,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella,
	}
	if instantanea.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, errPoliticaBorradorLlamamientoBolsaDesarrolloNoDisponible
	}
	return instantanea, nil
}

func (p *politicaBorradorLlamamientoBolsaDesarrollo) ObtenerInstantaneaAutorizacion(
	ctx context.Context, principalID, perfilRef string,
) (dominiovec.InstantaneaAutorizacion, error) {
	if p == nil || ctx == nil || ctx.Err() != nil {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.publicada || p.instantanea.Validar() != nil ||
		principalID != p.instantanea.AsignacionPerfil.PrincipalID || perfilRef != p.instantanea.AsignacionPerfil.PerfilActivoRef {
		return dominiovec.InstantaneaAutorizacion{}, puertosvec.ErrFuenteAutorizacionNoDisponible
	}
	return clonarInstantaneaAutorizacionPostgreSQLDesarrollo(p.instantanea), nil
}

func (p *politicaBorradorLlamamientoBolsaDesarrollo) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context, referencia dominiovec.ReferenciaEntradaCatalogo, instante time.Time,
) error {
	if p == nil || ctx == nil || ctx.Err() != nil || !dominioct.InstanteUTCCanonico(instante) {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	p.mu.RLock()
	publicada, instantanea := p.publicada, p.instantanea
	p.mu.RUnlock()
	if !publicada || instantanea.Validar() != nil || !instantanea.AsignacionPerfil.VigenteEn(instante) ||
		(referencia != motivoCrearBorradorLlamamientoBolsaDesarrollo() && referencia != motivoConsultarBorradorLlamamientoBolsaDesarrollo() && referencia != motivoCambiarSituacionParticipacionBolsaDesarrollo()) {
		return dominiovec.ErrSolicitudAutorizacionInvalida
	}
	return nil
}

func (p *politicaBorradorLlamamientoBolsaDesarrollo) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	ctx context.Context, orden puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	if !p.ordenRegistroBorradorLlamamientoValida(ctx, orden, true) ||
		dependenciaAutorizacionComunDesarrolloNula(p.registro) {
		return time.Time{}, puertosvec.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	return p.registro.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, orden)
}

func (p *politicaBorradorLlamamientoBolsaDesarrollo) RegistrarDenegacionAutorizacionLigadaV3(
	ctx context.Context, orden puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	if !p.ordenRegistroBorradorLlamamientoValida(ctx, orden, false) ||
		dependenciaAutorizacionComunDesarrolloNula(p.registro) {
		return puertosvec.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	return p.registro.RegistrarDenegacionAutorizacionLigadaV3(ctx, orden)
}

func (p *politicaBorradorLlamamientoBolsaDesarrollo) ordenRegistroBorradorLlamamientoValida(
	ctx context.Context,
	orden interface {
		Datos() (puertosvec.DatosOrdenRegistroAutorizacionLigadaV3, error)
	},
	concesion bool,
) bool {
	if p == nil || ctx == nil || ctx.Err() != nil || orden == nil {
		return false
	}
	datos, err := orden.Datos()
	if err != nil || datos.ResultadoContexto.Validar() != nil {
		return false
	}
	p.mu.RLock()
	publicada, instantanea := p.publicada, p.instantanea
	p.mu.RUnlock()
	if !publicada || instantanea.Validar() != nil ||
		datos.ResultadoContexto.Contexto.Principal.ID != instantanea.AsignacionPerfil.PrincipalID ||
		datos.ResultadoContexto.Contexto.PerfilActivoRef != instantanea.AsignacionPerfil.PerfilActivoRef {
		return false
	}
	solicitud, err := datos.Solicitud.Datos()
	if err != nil || !motivoBorradorLlamamientoCorresponde(solicitud.Accion, datos.ReferenciaMotivo) ||
		solicitud.ReferenciaMotivo != datos.ReferenciaMotivo {
		return false
	}
	return !concesion || datos.Decision.ValidarPara(datos.Solicitud) == nil
}

func motivoBorradorLlamamientoCorresponde(accion string, motivo dominiovec.ReferenciaEntradaCatalogo) bool {
	switch accion {
	case puertosbolsa.AccionCrearBorradorLlamamientoInterno:
		return motivo == motivoCrearBorradorLlamamientoBolsaDesarrollo()
	case puertosbolsa.AccionConsultarBorradorLlamamientoInterno:
		return motivo == motivoConsultarBorradorLlamamientoBolsaDesarrollo()
	case puertosbolsa.AccionCambiarSituacionParticipacion:
		return motivo == motivoCambiarSituacionParticipacionBolsaDesarrollo()
	default:
		return false
	}
}

func motivoCambiarSituacionParticipacionBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_situacion_participacion_bolsa", CatalogoVersion: 1, CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-b2-v1"), EntradaClave: referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-b2-situacion-cambiar")}
}

func motivoCrearBorradorLlamamientoBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_borrador_llamamiento_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-bback-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-bback-crear")}
}

func motivoConsultarBorradorLlamamientoBolsaDesarrollo() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_borrador_llamamiento_bolsa", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-bolsa-bback-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "bolsa-bback-consultar")}
}

var _ puertosvec.FuenteAutorizacion = (*politicaBorradorLlamamientoBolsaDesarrollo)(nil)
var _ puertosvec.RegistroConcesionesCandidatasAutorizacionLigadaV3 = (*politicaBorradorLlamamientoBolsaDesarrollo)(nil)
var _ puertosvec.RegistroDenegacionesAutorizacionLigadaV3 = (*politicaBorradorLlamamientoBolsaDesarrollo)(nil)
var _ puertosvec.ValidadorReferenciaMotivoAutorizacionV2 = (*politicaBorradorLlamamientoBolsaDesarrollo)(nil)
