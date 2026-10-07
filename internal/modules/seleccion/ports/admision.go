package ports

import (
	"context"

	meritos "vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

// MaterialAdmisionPreparacion solo recibe datos sintéticos locales. Hechos
// reutiliza el contrato de preparación RUM; no es el lector nominal RUM05.
type MaterialAdmisionPreparacion struct {
	PreparacionRef    string                         `json:"preparacion_ref"`
	Revision          int                            `json:"revision"`
	Alcance           string                         `json:"alcance"`
	Bases             domain.BasesAdmision           `json:"bases"`
	SolicitudContexto *domain.ContextoSolicitudLocal `json:"solicitud_contexto,omitempty"`
	Requisitos        []domain.RequisitoAdmision     `json:"requisitos"`
	Hechos            *meritos.HechosPreparados      `json:"hechos,omitempty"`
}

// CatalogoAdmision entrega la versión exacta del catálogo de motivos de
// exclusión y plazo de subsanación de una convocatoria. Debe fallar si no la
// conoce; nunca sustituye otra versión.
type CatalogoAdmision interface {
	CatalogoAdmision(ctx context.Context, referencia, version string) (domain.CatalogoAdmision, error)
}

// MaterialListaAdmision reúne las revisiones S4 de una convocatoria y la
// decisión que RRHH propone para cada una.
type MaterialListaAdmision struct {
	ListaRef        string                        `json:"lista_ref"`
	Revision        int                           `json:"revision"`
	Alcance         string                        `json:"alcance"`
	Bases           domain.BasesAdmision          `json:"bases"`
	CatalogoRef     string                        `json:"catalogo_ref"`
	CatalogoVersion string                        `json:"catalogo_version"`
	RevisionesS4    []MaterialAdmisionPreparacion `json:"revisiones_s4"`
	Decisiones      []domain.DecisionAdmision     `json:"decisiones"`
}

// MaterialListaDefinitiva parte del material exacto de la provisional, que se
// recompone, y de una resolución por cada exclusión de esa provisional.
type MaterialListaDefinitiva struct {
	ListaRef     string                         `json:"lista_ref"`
	Revision     int                            `json:"revision"`
	Alcance      string                         `json:"alcance"`
	Provisional  MaterialListaAdmision          `json:"provisional"`
	Antecedente  domain.AntecedenteLista        `json:"antecedente_provisional"`
	Resoluciones []domain.ResolucionSubsanacion `json:"resoluciones"`
}

// MaterialRevisionProvisional reúne el material completo de la nueva revisión
// de la provisional, la provisional anterior tal como se preparó y la huella
// que RRHH declara de ella (la de --salida antecedente-lista o la del
// envoltorio de la revisión previa). Sin esa huella la anterior no está anclada.
type MaterialRevisionProvisional struct {
	Material    MaterialListaAdmision           `json:"material"`
	Anterior    domain.ListaAdmisionProvisional `json:"anterior"`
	Antecedente domain.AntecedenteLista         `json:"antecedente_anterior"`
}
