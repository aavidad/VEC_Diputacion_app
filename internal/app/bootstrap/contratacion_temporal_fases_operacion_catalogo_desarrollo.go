package bootstrap

import (
	"errors"
	"slices"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Fases en que RRHH puede hacer cada operación (reglas c23 del catálogo de
// Contratación temporal). El permiso fijo de una operación cubre la
// organización y estas fases y este estado previos, nunca un expediente
// concreto: el expediente va en la referencia del recurso y lo cotejan el caso
// de uso y la base de datos. Cambiar la fase de una operación es publicar otra
// entrada del catálogo; el perfil fijo afectado queda entonces pendiente de
// provisión hasta que el operador la apruebe.
const (
	prefijoFaseOperacionCT        = reglas.CTPrefijoFaseOperacion
	operacionFaseAnalisisCT       = "analisis"
	atributoEstadoFaseOperacionCT = "estado"
	maximoFasesOperacionCT        = 20
)

var errFasesOperacionNoValidas = errors.New("bootstrap: fases de operación del catálogo de reglas no válidas")

// faseOperacionCT son las fases y el estado previos admitidos para una
// operación. Las fases van ordenadas y sin repetir.
type faseOperacionCT struct {
	fases  []domain.ClaveFase
	estado domain.EstadoOperativo
}

func (f faseOperacionCT) valida() bool {
	if len(f.fases) == 0 || len(f.fases) > maximoFasesOperacionCT || !f.estado.Valido() {
		return false
	}
	for i, fase := range f.fases {
		if !fase.Valida() || (i > 0 && f.fases[i-1] >= fase) {
			return false
		}
	}
	return true
}

// admite dice si un expediente en esa fase y ese estado admite la operación.
func (f faseOperacionCT) admite(fase domain.ClaveFase, estado domain.EstadoOperativo) bool {
	return f.valida() && estado == f.estado && slices.Contains(f.fases, fase)
}

// ambitosPerfil son los ámbitos del permiso fijo de la operación en la
// organización dada.
func (f faseOperacionCT) ambitosPerfil(organizacionRef string) []dominiovec.AmbitoPerfil {
	fases := make([]string, len(f.fases))
	for i, fase := range f.fases {
		fases[i] = string(fase)
	}
	return []dominiovec.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{organizacionRef}},
		{Clave: "fase_previa", Valores: fases},
		{Clave: "estado_previo", Valores: []string{string(f.estado)}},
	}
}

// fasesOperacionPredeterminadasCT son las de siempre, sin catálogo: el
// análisis se hace con la solicitud en curso.
func fasesOperacionPredeterminadasCT() map[string]faseOperacionCT {
	return map[string]faseOperacionCT{
		operacionFaseAnalisisCT: {fases: []domain.ClaveFase{"solicitud"}, estado: domain.EstadoEnCurso},
	}
}

// fasesOperacionDesdeReglasCT lee las entradas c23. Una entrada mal formada
// impide arrancar en lugar de dejar un permiso a medias; sin entrada para una
// operación rige la de siempre.
func fasesOperacionDesdeReglasCT(vigentes []reglas.Regla) (map[string]faseOperacionCT, error) {
	resultado := fasesOperacionPredeterminadasCT()
	vistas := make(map[string]bool)
	for _, regla := range vigentes {
		if !strings.HasPrefix(regla.Clave, prefijoFaseOperacionCT) {
			continue
		}
		operacion := strings.TrimPrefix(regla.Clave, prefijoFaseOperacionCT)
		if !domain.ClaveCatalogo(operacion).Valida() || vistas[operacion] || regla.Unidad != reglas.UnidadLista {
			return nil, errFasesOperacionNoValidas
		}
		vistas[operacion] = true
		f := faseOperacionCT{estado: domain.EstadoOperativo(regla.Atributos[atributoEstadoFaseOperacionCT])}
		for _, fase := range regla.Elementos() {
			f.fases = append(f.fases, domain.ClaveFase(fase))
		}
		slices.Sort(f.fases)
		if !f.valida() {
			return nil, errFasesOperacionNoValidas
		}
		resultado[operacion] = f
	}
	return resultado, nil
}

// faseOperacionVigente es la del catálogo o, sin él, la de siempre.
func (o *opcionesAnalisisCTDesarrollo) faseOperacionVigente(operacion string) (faseOperacionCT, bool) {
	fases := fasesOperacionPredeterminadasCT()
	if o != nil && o.fasesOperacion != nil {
		fases = o.fasesOperacion
	}
	f, existe := fases[operacion]
	if !existe || !f.valida() {
		return faseOperacionCT{}, false
	}
	return faseOperacionCT{fases: slices.Clone(f.fases), estado: f.estado}, true
}
