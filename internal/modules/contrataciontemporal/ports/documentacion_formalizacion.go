package ports

import (
	"context"
	"errors"
	"time"
)

// Documentación y plazos de la formalización tras aceptar una oferta
// (dudas 9 y 18). Qué documentos se exigen, en qué plazo y con qué margen
// hasta la incorporación procede de un catálogo de reglas versionado, nunca
// de constantes del código: este puerto solo transporta la regla resuelta.
var (
	// ErrReglasFormalizacionNoConfiguradas: la composición no tiene catálogo
	// de reglas. La interfaz lo dice; no se supone ningún plazo.
	ErrReglasFormalizacionNoConfiguradas = errors.New("contratacion temporal: reglas de formalizacion no configuradas")
	// ErrReglasFormalizacionNoDisponibles: el catálogo, la regla o el cálculo
	// de plazos no están disponibles o no cumplen el contrato.
	ErrReglasFormalizacionNoDisponibles    = errors.New("contratacion temporal: reglas de formalizacion no disponibles")
	ErrSolicitudDocumentacionFormalizacion = errors.New("contratacion temporal: solicitud de documentacion de formalizacion invalida")
)

// ReglaFormalizacion es la copia neutral de una regla del catálogo con su
// procedencia exacta. Ejemplo es cierto si la regla, o parte de ella, no
// procede de una norma aprobada: la interfaz debe rotularla.
type ReglaFormalizacion struct {
	Clave     string
	Unidad    string
	Cantidad  int
	Computo   string
	Elementos []string
	// ElementosPorModalidad solo existe si el catálogo distingue modalidades
	// (atributo valor_<modalidad> de una regla de lista).
	ElementosPorModalidad map[string][]string
	Origen                string
	Ejemplo               bool
	Articulo              string
	Norma                 string
	Duda                  string
	ParteEjemplo          string
	Referencia            string
	HuellaCatalogo        string
}

// VencimientoFormalizacion es el último día del plazo y el primer instante
// en que ya ha vencido (00:00 del día siguiente en hora peninsular).
type VencimientoFormalizacion struct {
	UltimoDia    string
	VenceAntesDe time.Time
	Prorrogado   bool
}

// ReglasFormalizacion es el puerto hacia el resolutor común de reglas.
type ReglasFormalizacion interface {
	ReglaFormalizacion(ctx context.Context, clave string) (ReglaFormalizacion, error)
	VencimientoFormalizacion(ctx context.Context, clave string, inicio time.Time) (ReglaFormalizacion, VencimientoFormalizacion, error)
}

// CatalogoTiposDocumentalesFormalizacion indica si un tipo documental tiene
// política de conservación y, por tanto, puede anotarse en Documentos.
type CatalogoTiposDocumentalesFormalizacion interface {
	TipoDocumentalCatalogado(tipo string) bool
}
