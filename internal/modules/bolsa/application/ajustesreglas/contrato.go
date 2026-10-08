package ajustesreglas

import (
	"context"
	"errors"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

const CatalogoBase = "vec.bolsa.reglas"
const CatalogoAjustes = CatalogoBase + reglas.SufijoCatalogoAjustes
const CatalogoMotivosID = CatalogoBase + ".motivos_ajuste"

var (
	ErrNoDisponible    = errors.New("ajustes_bolsa_no_disponibles")
	ErrEntradaInvalida = errors.New("ajustes_bolsa_entrada_invalida")
	ErrConflicto       = errors.New("ajustes_bolsa_conflicto")
	ErrProhibido       = errors.New("ajustes_bolsa_prohibido")
	ErrNoAutenticado   = errors.New("ajustes_bolsa_no_autenticado")
)

// Repositorio no inventa una autorización de lectura: Consultar y LeerPreimagen
// requieren el lector corporativo certificado y auditado del propietario.
// Operar obtiene ACTO V3 fresco y B88 lo consume en la misma transacción.
type Repositorio interface {
	Consultar(context.Context, vecdomain.ContextoActor, int, *int64) (Lectura, error)
	LeerCabeza(context.Context) (*reglas.VersionAjustes, error)
	LeerPreimagen(context.Context, vecdomain.ContextoActor, string) (Material, bool, error)
	Operar(context.Context, vecdomain.ContextoActor, Material) (Resultado, error)
}

type Lectura struct {
	Cabeza             *reglas.VersionAjustes `json:"cabeza"`
	VigenteHoy         *reglas.VersionAjustes `json:"vigente_hoy"`
	CabezaPublicadaEn  time.Time              `json:"-"`
	VigentePublicadaEn time.Time              `json:"-"`
	Reglas             []reglas.Regla         `json:"-"`
	Programados        []VersionProgramada    `json:"programados"`
	Historial          []CambioHistorico      `json:"historial"`
	HayMas             bool                   `json:"hay_mas"`
	PuedeAjustar       bool                   `json:"puede_ajustar"`
	ConsultadaEn       time.Time              `json:"consultada_en"`
}

type VersionProgramada struct {
	Version      int                          `json:"version"`
	HuellaSHA256 string                       `json:"huella_sha256"`
	Ajustes      map[string]map[string]string `json:"ajustes"`
	VigenteDesde time.Time                    `json:"vigente_desde"`
	PublicadaEn  time.Time                    `json:"publicada_en"`
}

type CambioHistorico struct {
	Version      int       `json:"version"`
	VigenteDesde time.Time `json:"vigente_desde"`
	PublicadaEn  time.Time `json:"publicada_en"`
	ActorNombre  *string   `json:"actor_nombre,omitempty"`
	MotivoClave  string    `json:"motivo_clave"`
	Referencia   *string   `json:"referencia,omitempty"`
	Nota         *string   `json:"nota,omitempty"`
	BaseVersion  int       `json:"base_version"`
	ReciboRef    string    `json:"recibo_ref"`
	Cambios      []Cambio  `json:"cambios"`
}

type Cambio struct {
	ReglaClave string `json:"regla_clave"`
	Campo      string `json:"campo"`
	Anterior   string `json:"anterior"`
	Nuevo      string `json:"nuevo"`
}

// Material tiene la forma exacta de B88; OrganizacionRef la fija el
// consumidor confiable y nunca se acepta de HTTP.
type Material struct {
	Operacion           string   `json:"operacion"`
	OrganizacionRef     string   `json:"organizacion_ref"`
	CatalogoID          string   `json:"catalogo_id"`
	ClaveIdempotencia   string   `json:"clave_idempotencia"`
	VersionEsperada     int      `json:"version_esperada"`
	VigenteDesde        *string  `json:"vigente_desde,omitempty"`
	BaseVersion         int      `json:"base_version"`
	BaseHuellaSHA256    string   `json:"base_huella_sha256"`
	AjustesCanonico     string   `json:"ajustes_canonico"`
	AjustesHuellaSHA256 string   `json:"ajustes_huella_sha256"`
	Cambios             []Cambio `json:"cambios"`
	MotivoClave         string   `json:"motivo_clave"`
	Referencia          *string  `json:"referencia,omitempty"`
	Nota                *string  `json:"nota,omitempty"`
}

type Recibo struct {
	ReciboRef           string    `json:"recibo_ref"`
	ClaveIdempotencia   string    `json:"clave_idempotencia"`
	Version             int       `json:"version"`
	HuellaSHA256        string    `json:"huella_sha256"`
	PublicadaEn         time.Time `json:"publicada_en"`
	VigenteDesde        time.Time `json:"vigente_desde"`
	DecisionRef         string    `json:"decision_ref"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsumoHuellaSHA256 string    `json:"consumo_huella_sha256"`
}

type Resultado struct {
	Ajustes map[string]map[string]string `json:"ajustes"`
	Replay  bool                         `json:"replay"`
	Recibo  Recibo                       `json:"recibo"`
}
