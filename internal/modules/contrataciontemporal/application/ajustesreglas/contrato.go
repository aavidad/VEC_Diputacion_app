// Package ajustesreglas coordina la edición gobernada de las reglas de
// Contratación temporal. El repositorio conserva la autoridad V3 y la historia CT.
package ajustesreglas

import (
	"context"
	"errors"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

var (
	ErrNoDisponible    = errors.New("ajustes de reglas no disponibles")
	ErrEntradaInvalida = errors.New("entrada de ajustes de reglas invalida")
	ErrConflicto       = errors.New("version o clave de ajustes en conflicto")
)

// Repositorio separa la lectura con sesión certificada y auditoría común del
// ajuste V3. Operar consume la decisión nominal en la misma transacción
// serializable que el cambio o su replay. El ámbito procede del servidor.
type Repositorio interface {
	Consultar(context.Context, vecdomain.ContextoActor, int, *int64) (Lectura, error)
	LeerActivacion(context.Context) (ActivacionBase, error)
	Operar(context.Context, vecdomain.ContextoActor, Material) (Resultado, error)
}

// ActivacionBase es la proyección nominal de CT191. Una lectura fallida nunca
// se representa como sin_publicar: el repositorio devuelve ErrNoDisponible.
type ActivacionBase struct {
	Estado        string `json:"estado"`
	Secuencia     int64  `json:"secuencia,omitempty"`
	CatalogoID    string `json:"catalogo_id,omitempty"`
	Version       int    `json:"version,omitempty"`
	HuellaSHA256  string `json:"huella_sha256,omitempty"`
	AprobacionRef string `json:"aprobacion_ref,omitempty"`
}

// Lectura contiene la cabeza autorizada y su historia paginada. PuedeAjustar
// es una proyección para la interfaz; Operar vuelve a autorizar el material.
type Lectura struct {
	Vigente            *reglas.VersionAjustes `json:"vigente"`
	VigenteBaseVersion int                    `json:"-"`
	VigenteBaseHuella  string                 `json:"-"`
	Historial          []CambioHistorico      `json:"historial"`
	HayMas             bool                   `json:"hay_mas"`
	PuedeAjustar       bool                   `json:"puede_ajustar"`
	Activacion         ActivacionBase         `json:"activacion"`
	Reglas             []reglas.Regla         `json:"-"`
}

type CambioHistorico struct {
	Version      int       `json:"version"`
	VigenteDesde time.Time `json:"vigente_desde"`
	MotivoClave  string    `json:"motivo_clave"`
	Referencia   *string   `json:"referencia,omitempty"`
	Nota         *string   `json:"nota,omitempty"`
	BaseVersion  int       `json:"base_version"`
	ReciboRef    string    `json:"recibo_ref"`
	Cambios      []Cambio  `json:"cambios"`
}

// Cambio usa el nombre exacto que CT148 comprueba en el material JSON.
type Cambio struct {
	ReglaClave string `json:"regla_clave"`
	Campo      string `json:"campo"`
	Anterior   string `json:"anterior"`
	Nuevo      string `json:"nuevo"`
}

// Material es la petición completa para CT148. OrganizacionRef la liga el
// adaptador PostgreSQL a partir de configuración, nunca del cuerpo HTTP.
type Material struct {
	Operacion           string   `json:"operacion"`
	OrganizacionRef     string   `json:"organizacion_ref"`
	CatalogoID          string   `json:"catalogo_id"`
	ClaveIdempotencia   string   `json:"clave_idempotencia"`
	VersionEsperada     int      `json:"version_esperada"`
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
