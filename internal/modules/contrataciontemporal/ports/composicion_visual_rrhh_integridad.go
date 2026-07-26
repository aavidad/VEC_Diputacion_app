package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const dominioHuellaComposicionVisualRRHH = "VEC-CT-COMPOSICION-VISUAL-RRHH-V1"

// CalcularHuellaComposicionVisualRRHH permite que el adaptador durable selle
// exactamente el DTO que va a publicar. No incluye el recibo interno.
func CalcularHuellaComposicionVisualRRHH(
	composicion ComposicionVisualRRHH,
) (string, error) {
	if composicion.validarContenido() != nil {
		return "", ErrResultadoComposicionVisualRRHHNoConfiable
	}
	contenido, err := json.Marshal(composicion)
	if err != nil {
		return "", ErrResultadoComposicionVisualRRHHNoConfiable
	}
	material := make([]byte, 0, len(dominioHuellaComposicionVisualRRHH)+1+len(contenido))
	material = append(material, dominioHuellaComposicionVisualRRHH...)
	material = append(material, 0)
	material = append(material, contenido...)
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

// ReciboComposicionVisualRRHH confirma que lectura, ámbito y registro de
// acceso se consolidaron antes de publicar el DTO. Sus campos nunca se
// serializan.
type ReciboComposicionVisualRRHH struct {
	lecturaRef      string
	auditoriaRef    string
	decisionRef     string
	correlacionRef  string
	sesionRef       string
	organizacionRef string
	claseAmbito     ClaseAmbitoConsultaRRHH
	ambitoRef       string
	accion          string
	finalidad       string
	flujoRef        string
	flujoVersion    uint64
	huellaContenido string
	registradaEn    time.Time
}

func NuevoReciboComposicionVisualRRHH(
	lecturaRef, auditoriaRef string,
	orden OrdenConsultaComposicionVisualRRHH,
	huellaContenido string,
	registradaEn time.Time,
) (ReciboComposicionVisualRRHH, error) {
	capacidad := orden.capacidad
	recibo := ReciboComposicionVisualRRHH{
		lecturaRef: lecturaRef, auditoriaRef: auditoriaRef,
		decisionRef:     capacidad.decisionRef,
		correlacionRef:  capacidad.correlacionRef,
		sesionRef:       orden.contexto.sesionRef,
		organizacionRef: orden.contexto.organizacionRef,
		claseAmbito:     capacidad.claseAmbito, ambitoRef: capacidad.ambitoRef,
		accion:          orden.vocabulario.accion,
		finalidad:       orden.vocabulario.finalidad,
		flujoRef:        orden.solicitud.flujoRef,
		flujoVersion:    orden.solicitud.flujoVersion,
		huellaContenido: huellaContenido, registradaEn: registradaEn,
	}
	if capacidad.validaPara(
		orden.contexto, orden.vocabulario, orden.solicitud, registradaEn,
	) != nil || recibo.validar() != nil {
		return ReciboComposicionVisualRRHH{},
			ErrResultadoComposicionVisualRRHHNoConfiable
	}
	return recibo, nil
}

func (r ReciboComposicionVisualRRHH) validar() error {
	if !domain.ReferenciaOpacaValida(r.lecturaRef) ||
		!domain.ReferenciaOpacaValida(r.auditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.decisionRef) ||
		!domain.ReferenciaOpacaValida(r.correlacionRef) ||
		!domain.ReferenciaOpacaValida(r.sesionRef) ||
		!domain.ReferenciaOpacaValida(r.organizacionRef) ||
		!r.claseAmbito.valida() ||
		!domain.ReferenciaOpacaValida(r.ambitoRef) ||
		(r.claseAmbito == AmbitoOrganizacionRRHH &&
			r.ambitoRef != r.organizacionRef) ||
		!claveVocabularioComposicionVisualValida(r.accion) ||
		!claveVocabularioComposicionVisualValida(r.finalidad) ||
		!domain.ReferenciaOpacaValida(r.flujoRef) ||
		r.flujoVersion < 1 || r.flujoVersion > versionMaximaJSONSegura ||
		!patronHuellaRRHH.MatchString(r.huellaContenido) ||
		r.huellaContenido == strings.Repeat("0", 64) ||
		!domain.InstanteUTCCanonico(r.registradaEn) {
		return ErrResultadoComposicionVisualRRHHNoConfiable
	}
	return nil
}

func (r ReciboComposicionVisualRRHH) coincideCon(
	orden OrdenConsultaComposicionVisualRRHH,
	huellaContenido string,
) bool {
	capacidad := orden.capacidad
	return r.validar() == nil &&
		capacidad.validaPara(
			orden.contexto, orden.vocabulario, orden.solicitud, r.registradaEn,
		) == nil &&
		r.decisionRef == capacidad.decisionRef &&
		r.correlacionRef == capacidad.correlacionRef &&
		r.sesionRef == orden.contexto.sesionRef &&
		r.organizacionRef == orden.contexto.organizacionRef &&
		r.claseAmbito == capacidad.claseAmbito &&
		r.ambitoRef == capacidad.ambitoRef &&
		r.accion == orden.vocabulario.accion &&
		r.finalidad == orden.vocabulario.finalidad &&
		r.flujoRef == orden.solicitud.flujoRef &&
		r.flujoVersion == orden.solicitud.flujoVersion &&
		r.huellaContenido == huellaContenido
}

func (r ReciboComposicionVisualRRHH) RegistradaEn() time.Time {
	return r.registradaEn
}
func (ReciboComposicionVisualRRHH) String() string {
	return "[recibo-composicion-visual-rrhh-redactado]"
}
func (ReciboComposicionVisualRRHH) GoString() string {
	return "[recibo-composicion-visual-rrhh-redactado]"
}
func (ReciboComposicionVisualRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialComposicionVisualRRHHSensible
}

func (c ComposicionVisualRRHH) ValidarPara(
	orden OrdenConsultaComposicionVisualRRHH,
) error {
	if orden.capacidad.validaPara(
		orden.contexto, orden.vocabulario, orden.solicitud, orden.instante,
	) != nil || c.validarContenido() != nil ||
		c.Flujo.Referencia != orden.solicitud.flujoRef ||
		c.Flujo.Version != orden.solicitud.flujoVersion ||
		c.GeneradaEn.Before(orden.instante) ||
		!c.GeneradaEn.Before(orden.capacidad.validaHasta) {
		return ErrResultadoComposicionVisualRRHHNoConfiable
	}
	huella, err := CalcularHuellaComposicionVisualRRHH(c)
	if err != nil || !c.Lectura.coincideCon(orden, huella) ||
		c.Lectura.registradaEn.Before(c.GeneradaEn) {
		return ErrResultadoComposicionVisualRRHHNoConfiable
	}
	return nil
}

func (c ComposicionVisualRRHH) Clonar() (ComposicionVisualRRHH, error) {
	if validarLimitesComposicionVisualRRHH(c) != nil {
		return ComposicionVisualRRHH{},
			ErrResultadoComposicionVisualRRHHNoConfiable
	}
	copia := c
	copia.Flujo.Fases = make([]FaseVisualRRHH, len(c.Flujo.Fases))
	copy(copia.Flujo.Fases, c.Flujo.Fases)
	copia.Flujo.Tareas = make([]TareaVisualRRHH, len(c.Flujo.Tareas))
	for i, tarea := range c.Flujo.Tareas {
		copia.Flujo.Tareas[i] = tarea
		copia.Flujo.Tareas[i].Paneles = make([]string, len(tarea.Paneles))
		copy(copia.Flujo.Tareas[i].Paneles, tarea.Paneles)
		copia.Flujo.Tareas[i].Operaciones = make(
			[]OperacionVisualRRHH, len(tarea.Operaciones),
		)
		copy(copia.Flujo.Tareas[i].Operaciones, tarea.Operaciones)
	}
	copia.Flujo.Paneles = make([]PanelVisualRRHH, len(c.Flujo.Paneles))
	for i, panel := range c.Flujo.Paneles {
		copia.Flujo.Paneles[i] = panel
		copia.Flujo.Paneles[i].Campos = make(
			[]CampoVisualRRHH, len(panel.Campos),
		)
		copy(copia.Flujo.Paneles[i].Campos, panel.Campos)
	}
	copia.Catalogos = make([]CatalogoVisualRRHH, len(c.Catalogos))
	for i, catalogo := range c.Catalogos {
		copia.Catalogos[i] = catalogo
		copia.Catalogos[i].Opciones = make(
			[]OpcionCatalogoVisualRRHH, len(catalogo.Opciones),
		)
		copy(copia.Catalogos[i].Opciones, catalogo.Opciones)
	}
	copia.Capacidades = make(
		[]CapacidadVisualConcedidaRRHH, len(c.Capacidades),
	)
	copy(copia.Capacidades, c.Capacidades)
	return copia, nil
}

func (ComposicionVisualRRHH) String() string {
	return "[composicion-visual-rrhh-redactada]"
}

func (ComposicionVisualRRHH) GoString() string {
	return "[composicion-visual-rrhh-redactada]"
}

var _ json.Marshaler = ReciboComposicionVisualRRHH{}
