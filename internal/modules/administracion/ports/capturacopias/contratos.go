// Package capturacopias define capacidades de captura, sin comandos ni rutas.
package capturacopias

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

type Exclusor interface {
	Adquirir(context.Context, string) (func() error, error)
}

// ControlEscritores debe cerrar todas las admisiones, drenar y comprobar los
// escritores inventariados, incluidos configuración, migradores y despachos.
// Un error parcial de cierre también exige Reabrir; nunca equivale a exclusión.
type ControlEscritores interface {
	CerrarAdmision(context.Context) error
	Drenar(context.Context) error
	ComprobarExclusion(context.Context) error
	Reabrir(context.Context) error
}

type Inventariador interface {
	Observar(context.Context) (copias.Inventario, error)
}

type CapturadorLogico interface {
	Capturar(context.Context, copias.Inventario) ([]copias.Artefacto, error)
}

// CapturadorComponentes continúa la captura física/ficheros dentro de la MISMA
// ventana. Su implementación exige y acredita parada limpia antes de copiar PG.
type CapturadorComponentes interface {
	Capturar(context.Context, copias.Inventario) ([]copias.Artefacto, error)
}

type Peticion struct {
	OrigenRef     string
	OperacionRef  string
	Esperado      copias.Inventario
	InicioVentana time.Time
	FinVentana    time.Time
}

// Parcial nunca constituye un manifiesto CS01 restaurable ni evidencia CS06.
// Los artefactos se guardan privados; no se publica contenido ni rutas.
type Parcial struct {
	FormatoVersion   int                `json:"formato_version"`
	OperacionRef     string             `json:"operacion_ref"`
	Estado           string             `json:"estado"`
	Completa         bool               `json:"completa"`
	Valida           bool               `json:"valida"`
	Publicable       bool               `json:"publicable"`
	Inicio           time.Time          `json:"inicio"`
	Fin              time.Time          `json:"fin"`
	InventarioSHA256 string             `json:"inventario_sha256"`
	Componentes      []copias.Artefacto `json:"componentes"`
}
