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
// organización y estas fases y estados previos, nunca un expediente
// concreto: el expediente va en la referencia del recurso y lo cotejan el caso
// de uso y la base de datos. Cambiar la fase de una operación es publicar otra
// entrada del catálogo; el perfil fijo afectado queda entonces pendiente de
// provisión hasta que el operador la apruebe.
//
// Formato de la entrada «c23.fase_operacion.<operación>»: «valor» es la lista
// de fases; «estado» el estado previo de todas ellas, salvo las que declaren
// el suyo en «estado_<fase>». Cada fase admite así un único estado.
const (
	prefijoFaseOperacionCT          = reglas.CTPrefijoFaseOperacion
	operacionFaseAnalisisCT         = "analisis"
	operacionFaseAsignacionCT       = "asignacion"
	operacionFaseInformeJuridicoCT  = "informe_juridico"
	atributoEstadoFaseOperacionCT   = "estado"
	prefijoAtributoEstadoFaseOperCT = "estado_"
	maximoFasesOperacionCT          = 20
)

var errFasesOperacionNoValidas = errors.New("bootstrap: fases de operación del catálogo de reglas no válidas")

// faseEstadoOperacionCT es un par fase previa y estado previo admitido.
type faseEstadoOperacionCT struct {
	fase   domain.ClaveFase
	estado domain.EstadoOperativo
}

// faseOperacionCT son los pares fase y estado previos admitidos para una
// operación, ordenados por fase y sin fases repetidas.
type faseOperacionCT struct {
	pares []faseEstadoOperacionCT
}

func (f faseOperacionCT) valida() bool {
	if len(f.pares) == 0 || len(f.pares) > maximoFasesOperacionCT {
		return false
	}
	for i, par := range f.pares {
		if !par.fase.Valida() || !par.estado.Valido() || (i > 0 && f.pares[i-1].fase >= par.fase) {
			return false
		}
	}
	return true
}

// admite dice si un expediente en esa fase y ese estado admite la operación:
// el par exacto debe estar declarado (no cualquier combinación de fases y
// estados sueltos).
func (f faseOperacionCT) admite(fase domain.ClaveFase, estado domain.EstadoOperativo) bool {
	return f.valida() && slices.Contains(f.pares, faseEstadoOperacionCT{fase: fase, estado: estado})
}

// ambitosPerfil son los ámbitos del permiso fijo de la operación en la
// organización dada, más los adicionales de la operación. Un ámbito V3 no
// expresa pares: lleva las fases y los estados declarados, y la validación de
// la petición (admite) exige además el par exacto antes de llegar al PDP.
func (f faseOperacionCT) ambitosPerfil(organizacionRef string, adicionales ...dominiovec.AmbitoPerfil) []dominiovec.AmbitoPerfil {
	fases := make([]string, 0, len(f.pares))
	estados := make([]string, 0, len(f.pares))
	for _, par := range f.pares {
		fases = append(fases, string(par.fase))
		if !slices.Contains(estados, string(par.estado)) {
			estados = append(estados, string(par.estado))
		}
	}
	slices.Sort(estados)
	ambitos := []dominiovec.AmbitoPerfil{
		{Clave: "organizacion_ref", Valores: []string{organizacionRef}},
		{Clave: "fase_previa", Valores: fases},
		{Clave: "estado_previo", Valores: estados},
	}
	return append(ambitos, adicionales...)
}

func faseOperacionDePares(pares ...faseEstadoOperacionCT) faseOperacionCT {
	f := faseOperacionCT{pares: slices.Clone(pares)}
	slices.SortFunc(f.pares, func(a, b faseEstadoOperacionCT) int { return strings.Compare(string(a.fase), string(b.fase)) })
	return f
}

// fasesOperacionPredeterminadasCT son las de siempre, sin catálogo.
func fasesOperacionPredeterminadasCT() map[string]faseOperacionCT {
	return map[string]faseOperacionCT{
		operacionFaseAnalisisCT:   faseOperacionDePares(faseEstadoOperacionCT{"solicitud", domain.EstadoEnCurso}),
		operacionFaseAsignacionCT: faseOperacionDePares(faseEstadoOperacionCT{"asignacion_unidad", domain.EstadoEnCurso}),
		// El informe inicial, tras la asignación, y el informe nuevo tras
		// subsanar un reparo.
		operacionFaseInformeJuridicoCT: faseOperacionDePares(
			faseEstadoOperacionCT{"asignacion_unidad", domain.EstadoEnCurso},
			faseEstadoOperacionCT{domain.FaseSubsanacionUnidad, domain.EstadoIncidencia}),
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
		var pares []faseEstadoOperacionCT
		usadas := map[string]bool{}
		for _, fase := range regla.Elementos() {
			estado, propio := regla.Atributos[prefijoAtributoEstadoFaseOperCT+fase]
			if !propio {
				estado = regla.Atributos[atributoEstadoFaseOperacionCT]
			}
			usadas[fase] = true
			pares = append(pares, faseEstadoOperacionCT{fase: domain.ClaveFase(fase), estado: domain.EstadoOperativo(estado)})
		}
		// Un «estado_<fase>» de una fase que no está en la lista es un error
		// de redacción, no algo que ignorar.
		for atributo := range regla.Atributos {
			if fase, ok := strings.CutPrefix(atributo, prefijoAtributoEstadoFaseOperCT); ok && !usadas[fase] {
				return nil, errFasesOperacionNoValidas
			}
		}
		f := faseOperacionDePares(pares...)
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
	return faseOperacionCT{pares: slices.Clone(f.pares)}, true
}

// faseDeOperacionCTDesarrollo es faseOperacionVigente sin el indicador: una
// operación sin fase válida devuelve la fase vacía, que ninguna plantilla ni
// validación admite.
func faseDeOperacionCTDesarrollo(o *opcionesAnalisisCTDesarrollo, operacion string) faseOperacionCT {
	f, _ := o.faseOperacionVigente(operacion)
	return f
}
