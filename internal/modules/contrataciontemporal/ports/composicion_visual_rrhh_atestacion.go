package ports

import (
	"encoding/json"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type ClasePublicacionVisualRRHH string

const (
	PublicacionFlujoVisualRRHH    ClasePublicacionVisualRRHH = "flujo"
	PublicacionCatalogoVisualRRHH ClasePublicacionVisualRRHH = "catalogo"
)

type PublicacionAtestableVisualRRHH struct {
	clase      ClasePublicacionVisualRRHH
	referencia string
	version    uint64
	huella     string
}

func (p PublicacionAtestableVisualRRHH) Clase() ClasePublicacionVisualRRHH {
	return p.clase
}
func (p PublicacionAtestableVisualRRHH) Referencia() string { return p.referencia }
func (p PublicacionAtestableVisualRRHH) Version() uint64    { return p.version }
func (p PublicacionAtestableVisualRRHH) Huella() string     { return p.huella }

// SolicitudAtestacionPublicacionesVisualesRRHH solo puede derivarse de una
// composición ligada a orden y recibo durable. No es una prueba: obliga a
// consultar una autoridad independiente de la fuente de composición.
type SolicitudAtestacionPublicacionesVisualesRRHH struct {
	publicaciones        []PublicacionAtestableVisualRRHH
	autenticacionRef     string
	decisionRef          string
	correlacionRef       string
	lecturaRef           string
	auditoriaRef         string
	sesionRef            string
	actorRef             string
	perfilRef            string
	organizacionRef      string
	claseAmbito          ClaseAmbitoConsultaRRHH
	ambitoRef            string
	accion               string
	finalidad            string
	flujoRef             string
	flujoVersion         uint64
	huellaComposicion    string
	registradaEn         time.Time
	capacidadValidaHasta time.Time
}

func NuevaSolicitudAtestacionPublicacionesVisualesRRHH(
	orden OrdenConsultaComposicionVisualRRHH,
	composicion ComposicionVisualRRHH,
) (SolicitudAtestacionPublicacionesVisualesRRHH, error) {
	if composicion.ValidarPara(orden) != nil {
		return SolicitudAtestacionPublicacionesVisualesRRHH{},
			ErrPublicacionesVisualesRRHHNoAtestadas
	}
	huella, err := CalcularHuellaComposicionVisualRRHH(composicion)
	if err != nil {
		return SolicitudAtestacionPublicacionesVisualesRRHH{},
			ErrPublicacionesVisualesRRHHNoAtestadas
	}
	publicaciones := make(
		[]PublicacionAtestableVisualRRHH, 1+len(composicion.Catalogos),
	)
	publicaciones[0] = PublicacionAtestableVisualRRHH{
		clase: PublicacionFlujoVisualRRHH, referencia: composicion.Flujo.Referencia,
		version: composicion.Flujo.Version, huella: composicion.Flujo.Huella,
	}
	for indice, catalogo := range composicion.Catalogos {
		publicaciones[indice+1] = PublicacionAtestableVisualRRHH{
			clase: PublicacionCatalogoVisualRRHH, referencia: catalogo.Referencia,
			version: catalogo.Version, huella: catalogo.Huella,
		}
	}
	capacidad, contexto, recibo := orden.capacidad, orden.contexto, composicion.Lectura
	solicitud := SolicitudAtestacionPublicacionesVisualesRRHH{
		publicaciones:    publicaciones,
		autenticacionRef: contexto.autenticacionRef,
		decisionRef:      capacidad.decisionRef, correlacionRef: capacidad.correlacionRef,
		lecturaRef: recibo.lecturaRef, auditoriaRef: recibo.auditoriaRef,
		sesionRef: contexto.sesionRef, actorRef: contexto.actorRef,
		perfilRef: contexto.perfilRef, organizacionRef: contexto.organizacionRef,
		claseAmbito: capacidad.claseAmbito, ambitoRef: capacidad.ambitoRef,
		accion: orden.vocabulario.accion, finalidad: orden.vocabulario.finalidad,
		flujoRef:          orden.solicitud.flujoRef,
		flujoVersion:      orden.solicitud.flujoVersion,
		huellaComposicion: huella, registradaEn: recibo.registradaEn,
		capacidadValidaHasta: capacidad.validaHasta,
	}
	if solicitud.Validar() != nil {
		return SolicitudAtestacionPublicacionesVisualesRRHH{},
			ErrPublicacionesVisualesRRHHNoAtestadas
	}
	return solicitud, nil
}

func (s SolicitudAtestacionPublicacionesVisualesRRHH) Validar() error {
	if len(s.publicaciones) < 1 ||
		len(s.publicaciones) > 1+MaximoCatalogosComposicionVisualRRHH ||
		!domain.ReferenciaOpacaValida(s.autenticacionRef) ||
		!domain.ReferenciaOpacaValida(s.decisionRef) ||
		!domain.ReferenciaOpacaValida(s.correlacionRef) ||
		!domain.ReferenciaOpacaValida(s.lecturaRef) ||
		!domain.ReferenciaOpacaValida(s.auditoriaRef) ||
		!domain.ReferenciaOpacaValida(s.sesionRef) ||
		!domain.ReferenciaOpacaValida(s.actorRef) ||
		!domain.ReferenciaOpacaValida(s.perfilRef) ||
		!domain.ReferenciaOpacaValida(s.organizacionRef) ||
		!s.claseAmbito.valida() || !domain.ReferenciaOpacaValida(s.ambitoRef) ||
		!claveVocabularioComposicionVisualValida(s.accion) ||
		!claveVocabularioComposicionVisualValida(s.finalidad) ||
		!domain.ReferenciaOpacaValida(s.flujoRef) ||
		s.flujoVersion < 1 || s.flujoVersion > versionMaximaJSONSegura ||
		!patronHuellaRRHH.MatchString(s.huellaComposicion) ||
		!domain.InstanteUTCCanonico(s.registradaEn) ||
		!domain.InstanteUTCCanonico(s.capacidadValidaHasta) ||
		!s.registradaEn.Before(s.capacidadValidaHasta) {
		return ErrPublicacionesVisualesRRHHNoAtestadas
	}
	vistas := make(map[string]struct{}, len(s.publicaciones))
	for indice, publicacion := range s.publicaciones {
		if (publicacion.clase != PublicacionFlujoVisualRRHH &&
			publicacion.clase != PublicacionCatalogoVisualRRHH) ||
			!domain.ReferenciaOpacaValida(publicacion.referencia) ||
			publicacion.version < 1 ||
			publicacion.version > versionMaximaJSONSegura ||
			!patronHuellaRRHH.MatchString(publicacion.huella) ||
			(indice == 0 && (publicacion.clase != PublicacionFlujoVisualRRHH ||
				publicacion.referencia != s.flujoRef ||
				publicacion.version != s.flujoVersion)) {
			return ErrPublicacionesVisualesRRHHNoAtestadas
		}
		identidad := string(publicacion.clase) + "\x00" +
			identidadCatalogoVisualRRHH(publicacion.referencia, publicacion.version)
		if _, repetida := vistas[identidad]; repetida {
			return ErrPublicacionesVisualesRRHHNoAtestadas
		}
		vistas[identidad] = struct{}{}
	}
	return nil
}

func (s SolicitudAtestacionPublicacionesVisualesRRHH) Publicaciones() []PublicacionAtestableVisualRRHH {
	copia := make([]PublicacionAtestableVisualRRHH, len(s.publicaciones))
	copy(copia, s.publicaciones)
	return copia
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) AutenticacionRef() string {
	return s.autenticacionRef
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) DecisionRef() string {
	return s.decisionRef
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) CorrelacionRef() string {
	return s.correlacionRef
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) LecturaRef() string {
	return s.lecturaRef
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) AuditoriaRef() string {
	return s.auditoriaRef
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) SesionRef() string {
	return s.sesionRef
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) ActorRef() string {
	return s.actorRef
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) PerfilRef() string {
	return s.perfilRef
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) OrganizacionRef() string {
	return s.organizacionRef
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) ClaseAmbito() ClaseAmbitoConsultaRRHH {
	return s.claseAmbito
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) AmbitoRef() string {
	return s.ambitoRef
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) Accion() string {
	return s.accion
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) Finalidad() string {
	return s.finalidad
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) FlujoRef() string {
	return s.flujoRef
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) FlujoVersion() uint64 {
	return s.flujoVersion
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) HuellaComposicion() string {
	return s.huellaComposicion
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) RegistradaEn() time.Time {
	return s.registradaEn
}
func (s SolicitudAtestacionPublicacionesVisualesRRHH) CapacidadValidaHasta() time.Time {
	return s.capacidadValidaHasta
}
func (SolicitudAtestacionPublicacionesVisualesRRHH) String() string {
	return "[solicitud-atestacion-publicaciones-visuales-rrhh-redactada]"
}
func (SolicitudAtestacionPublicacionesVisualesRRHH) GoString() string {
	return "[solicitud-atestacion-publicaciones-visuales-rrhh-redactada]"
}
func (SolicitudAtestacionPublicacionesVisualesRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialComposicionVisualRRHHSensible
}

var _ json.Marshaler = SolicitudAtestacionPublicacionesVisualesRRHH{}
