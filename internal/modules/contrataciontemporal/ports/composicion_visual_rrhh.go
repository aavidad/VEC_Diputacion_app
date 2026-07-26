package ports

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	EsquemaComposicionVisualRRHH = "vec.contratacion_temporal.composicion_visual.v1"

	MaximoFasesComposicionVisualRRHH       = 32
	MaximoTareasComposicionVisualRRHH      = 128
	MaximoPanelesComposicionVisualRRHH     = 64
	MaximoCamposComposicionVisualRRHH      = 512
	MaximoCatalogosComposicionVisualRRHH   = 64
	MaximoOpcionesCatalogoVisualRRHH       = 1_000
	MaximoOpcionesComposicionVisualRRHH    = 5_000
	MaximoOperacionesComposicionVisualRRHH = 128
	MaximoCapacidadesVisualesRRHH          = 128
	MaximoPanelesPorTareaVisualRRHH        = 64
	MaximoReferenciasPanelesVisualRRHH     = 1_024
)

var patronClaveVisualRRHH = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)

type TipoPanelVisualRRHH string

const (
	PanelVisualDatos          TipoPanelVisualRRHH = "datos"
	PanelVisualFormulario     TipoPanelVisualRRHH = "formulario"
	PanelVisualComprobaciones TipoPanelVisualRRHH = "comprobaciones"
	PanelVisualTabla          TipoPanelVisualRRHH = "tabla"
	PanelVisualDocumentos     TipoPanelVisualRRHH = "documentos"
	PanelVisualAviso          TipoPanelVisualRRHH = "aviso"
)

func (t TipoPanelVisualRRHH) valido() bool {
	switch t {
	case PanelVisualDatos, PanelVisualFormulario,
		PanelVisualComprobaciones, PanelVisualTabla,
		PanelVisualDocumentos, PanelVisualAviso:
		return true
	default:
		return false
	}
}

type ControlCampoVisualRRHH string

const (
	ControlVisualSoloLectura ControlCampoVisualRRHH = "solo_lectura"
	ControlVisualTexto       ControlCampoVisualRRHH = "texto"
	ControlVisualArea        ControlCampoVisualRRHH = "area"
	ControlVisualFecha       ControlCampoVisualRRHH = "fecha"
	ControlVisualSeleccion   ControlCampoVisualRRHH = "seleccion"
	ControlVisualRadio       ControlCampoVisualRRHH = "radio"
	ControlVisualImporte     ControlCampoVisualRRHH = "importe"
)

func (c ControlCampoVisualRRHH) valido() bool {
	switch c {
	case ControlVisualSoloLectura, ControlVisualTexto, ControlVisualArea,
		ControlVisualFecha, ControlVisualSeleccion, ControlVisualRadio,
		ControlVisualImporte:
		return true
	default:
		return false
	}
}

type OpcionCatalogoVisualRRHH struct {
	Clave     domain.ClaveCatalogo `json:"clave"`
	ClaveI18n string               `json:"clave_i18n"`
}

type CatalogoVisualRRHH struct {
	Referencia   string                     `json:"referencia"`
	Version      uint64                     `json:"version"`
	Huella       string                     `json:"huella_sha256"`
	ClaveI18n    string                     `json:"clave_i18n"`
	PublicadoEn  time.Time                  `json:"publicado_en"`
	VigenteDesde time.Time                  `json:"vigente_desde"`
	VigenteHasta time.Time                  `json:"vigente_hasta"`
	Opciones     []OpcionCatalogoVisualRRHH `json:"opciones"`
}

type CampoVisualRRHH struct {
	Clave           string                 `json:"clave"`
	Orden           uint16                 `json:"orden"`
	ClaveI18n       string                 `json:"clave_i18n"`
	Control         ControlCampoVisualRRHH `json:"control"`
	Obligatorio     bool                   `json:"obligatorio"`
	CatalogoRef     string                 `json:"catalogo_ref,omitempty"`
	CatalogoVersion uint64                 `json:"catalogo_version,omitempty"`
}

type PanelVisualRRHH struct {
	Referencia string              `json:"referencia"`
	Orden      uint16              `json:"orden"`
	Tipo       TipoPanelVisualRRHH `json:"tipo"`
	ClaveI18n  string              `json:"clave_i18n"`
	Campos     []CampoVisualRRHH   `json:"campos"`
}

type OperacionVisualRRHH struct {
	Clave          string `json:"clave"`
	ClaveI18n      string `json:"clave_i18n"`
	CapacidadClave string `json:"capacidad_clave"`
}

type TareaVisualRRHH struct {
	Referencia  string                `json:"referencia"`
	FaseClave   domain.ClaveFase      `json:"fase_clave"`
	Orden       uint16                `json:"orden"`
	ClaveI18n   string                `json:"clave_i18n"`
	Paneles     []string              `json:"paneles"`
	Operaciones []OperacionVisualRRHH `json:"operaciones"`
}

type FaseVisualRRHH struct {
	Clave     domain.ClaveFase `json:"clave"`
	Orden     uint16           `json:"orden"`
	ClaveI18n string           `json:"clave_i18n"`
}

type DefinicionFlujoVisualRRHH struct {
	Referencia   string            `json:"referencia"`
	Version      uint64            `json:"version"`
	Huella       string            `json:"huella_sha256"`
	ClaveI18n    string            `json:"clave_i18n"`
	PublicadoEn  time.Time         `json:"publicado_en"`
	VigenteDesde time.Time         `json:"vigente_desde"`
	VigenteHasta time.Time         `json:"vigente_hasta"`
	Fases        []FaseVisualRRHH  `json:"fases"`
	Tareas       []TareaVisualRRHH `json:"tareas"`
	Paneles      []PanelVisualRRHH `json:"paneles"`
}

// CapacidadVisualConcedidaRRHH es solo una pista de presentación. El efecto
// asociado se vuelve a autorizar siempre en su caso de uso.
type CapacidadVisualConcedidaRRHH struct {
	OperacionClave string `json:"operacion_clave"`
	CapacidadClave string `json:"capacidad_clave"`
}

type ComposicionVisualRRHH struct {
	Esquema     string                         `json:"esquema"`
	GeneradaEn  time.Time                      `json:"generada_en"`
	Flujo       DefinicionFlujoVisualRRHH      `json:"flujo"`
	Catalogos   []CatalogoVisualRRHH           `json:"catalogos"`
	Capacidades []CapacidadVisualConcedidaRRHH `json:"capacidades_visuales"`
	Lectura     ReciboComposicionVisualRRHH    `json:"-"`
}

// MarshalJSON normaliza todas las colecciones del DTO público a [] antes de
// codificarlas. El recibo interno permanece excluido por contrato.
func (c ComposicionVisualRRHH) MarshalJSON() ([]byte, error) {
	copia, err := c.Clonar()
	if err != nil {
		return nil, ErrResultadoComposicionVisualRRHHNoConfiable
	}
	type composicionVisualPublica ComposicionVisualRRHH
	return json.Marshal(composicionVisualPublica(copia))
}

func (c ComposicionVisualRRHH) validarContenido() error {
	if c.Esquema != EsquemaComposicionVisualRRHH ||
		!domain.InstanteUTCCanonico(c.GeneradaEn) ||
		validarLimitesComposicionVisualRRHH(c) != nil ||
		validarDefinicionFlujoVisualRRHH(c.Flujo, c.GeneradaEn) != nil {
		return ErrResultadoComposicionVisualRRHHNoConfiable
	}
	catalogos := make(map[string]CatalogoVisualRRHH, len(c.Catalogos))
	for _, catalogo := range c.Catalogos {
		if validarCatalogoVisualRRHH(catalogo, c.GeneradaEn) != nil {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		identidad := identidadCatalogoVisualRRHH(catalogo.Referencia, catalogo.Version)
		if _, repetido := catalogos[identidad]; repetido {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		catalogos[identidad] = catalogo
	}
	return validarRelacionesComposicionVisualRRHH(c, catalogos)
}

func validarLimitesComposicionVisualRRHH(c ComposicionVisualRRHH) error {
	if validarLimitesDefinicionFlujoVisualRRHH(c.Flujo) != nil ||
		len(c.Catalogos) > MaximoCatalogosComposicionVisualRRHH ||
		len(c.Capacidades) > MaximoCapacidadesVisualesRRHH {
		return ErrResultadoComposicionVisualRRHHNoConfiable
	}
	totalOpciones := 0
	for _, catalogo := range c.Catalogos {
		if len(catalogo.Opciones) > MaximoOpcionesCatalogoVisualRRHH ||
			len(catalogo.Opciones) >
				MaximoOpcionesComposicionVisualRRHH-totalOpciones {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		totalOpciones += len(catalogo.Opciones)
	}
	return nil
}

func validarDefinicionFlujoVisualRRHH(
	flujo DefinicionFlujoVisualRRHH,
	instante time.Time,
) error {
	if !identidadPublicacionVisualValida(
		flujo.Referencia, flujo.Version, flujo.Huella, flujo.ClaveI18n,
		flujo.PublicadoEn, flujo.VigenteDesde, flujo.VigenteHasta, instante,
	) {
		return ErrResultadoComposicionVisualRRHHNoConfiable
	}
	huella, err := CalcularHuellaDefinicionFlujoVisualRRHH(flujo)
	if err != nil || huella != flujo.Huella {
		return ErrResultadoComposicionVisualRRHHNoConfiable
	}
	fases := make(map[domain.ClaveFase]struct{}, len(flujo.Fases))
	ordenes := make(map[uint16]struct{}, len(flujo.Fases))
	for _, fase := range flujo.Fases {
		if !fase.Clave.Valida() || fase.Orden < 1 ||
			!claveI18nVisualRRHHValida(fase.ClaveI18n) {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		if _, repetida := fases[fase.Clave]; repetida {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		if _, repetido := ordenes[fase.Orden]; repetido {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		fases[fase.Clave], ordenes[fase.Orden] = struct{}{}, struct{}{}
	}
	return nil
}

func validarCatalogoVisualRRHH(
	catalogo CatalogoVisualRRHH,
	instante time.Time,
) error {
	if !identidadPublicacionVisualValida(
		catalogo.Referencia, catalogo.Version, catalogo.Huella,
		catalogo.ClaveI18n, catalogo.PublicadoEn, catalogo.VigenteDesde,
		catalogo.VigenteHasta, instante,
	) || len(catalogo.Opciones) < 1 {
		return ErrResultadoComposicionVisualRRHHNoConfiable
	}
	huella, err := CalcularHuellaCatalogoVisualRRHH(catalogo)
	if err != nil || huella != catalogo.Huella {
		return ErrResultadoComposicionVisualRRHHNoConfiable
	}
	opciones := make(map[domain.ClaveCatalogo]struct{}, len(catalogo.Opciones))
	for _, opcion := range catalogo.Opciones {
		if !opcion.Clave.Valida() || !claveI18nVisualRRHHValida(opcion.ClaveI18n) {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		if _, repetida := opciones[opcion.Clave]; repetida {
			return ErrResultadoComposicionVisualRRHHNoConfiable
		}
		opciones[opcion.Clave] = struct{}{}
	}
	return nil
}

func identidadPublicacionVisualValida(
	referencia string,
	version uint64,
	huella, claveI18n string,
	publicadoEn, vigenteDesde, vigenteHasta, instante time.Time,
) bool {
	return domain.ReferenciaOpacaValida(referencia) &&
		version >= 1 && version <= versionMaximaJSONSegura &&
		patronHuellaRRHH.MatchString(huella) &&
		huella != strings.Repeat("0", 64) &&
		claveI18nVisualRRHHValida(claveI18n) &&
		domain.InstanteUTCCanonico(publicadoEn) &&
		domain.InstanteUTCCanonico(vigenteDesde) &&
		domain.InstanteUTCCanonico(vigenteHasta) &&
		!publicadoEn.After(vigenteDesde) &&
		vigenteHasta.After(vigenteDesde) &&
		!instante.Before(vigenteDesde) && instante.Before(vigenteHasta)
}

func claveVisualRRHHValida(valor string) bool {
	return valor == strings.TrimSpace(valor) &&
		patronClaveVisualRRHH.MatchString(valor)
}

func claveI18nVisualRRHHValida(valor string) bool {
	return claveVisualRRHHValida(valor) && strings.ContainsRune(valor, '.')
}

func identidadCatalogoVisualRRHH(referencia string, version uint64) string {
	return referencia + "\x00" + strconv.FormatUint(version, 10)
}
