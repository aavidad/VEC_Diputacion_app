package domain

import (
	"errors"
	"sort"

	meritos "vec-diputacion-granada/internal/modules/meritos/domain"
)

var ErrRevisionLista = errors.New("seleccion.revision_lista.entrada_invalida")

const EsquemaRevisionListaProvisional = "vec.seleccion.lista-admision-revision.v1"

// RevisionListaProvisional enlaza una nueva revisión de la provisional con la
// anterior. La lista va completa y sin el enlace dentro, de modo que su huella
// se sigue obteniendo de su propio material (la definitiva la usa así).
type RevisionListaProvisional struct {
	Esquema      string                   `json:"esquema"`
	Lista        ListaAdmisionProvisional `json:"lista"`
	Anterior     AntecedenteLista         `json:"anterior"`
	Incorporadas []string                 `json:"incorporadas"`
	Pendientes   []string                 `json:"pendientes"`
}

// ValidarListaProvisional comprueba la coherencia interna de una provisional
// leída de un archivo: forma, recuentos y que siga siendo un borrador.
func ValidarListaProvisional(l ListaAdmisionProvisional) error {
	if l.Esquema != EsquemaListaAdmisionProvisional || l.Alcance != AlcanceAdmisionPreparacion || l.Estado != "borrador_pendiente_aprobacion" ||
		l.Aprobada || l.Publicada || l.Persistida || !meritos.ReferenciaValida(l.ListaRef) || l.Revision < 1 || l.Revision > 1_000_000 ||
		ValidarBasesAdmision(l.Bases, nil) != nil || l.Vencimiento != "pendiente_publicacion" ||
		len(l.Admitidas)+len(l.Excluidas) > MaximoSolicitudesLista {
		return ErrRevisionLista
	}
	refs := map[string]bool{}
	subsanables := 0
	for _, a := range l.Admitidas {
		if refs[a.Antecedente.PreparacionRef] || !meritos.ReferenciaValida(a.Antecedente.PreparacionRef) {
			return ErrRevisionLista
		}
		refs[a.Antecedente.PreparacionRef] = true
	}
	for _, e := range l.Excluidas {
		if refs[e.Antecedente.PreparacionRef] || !meritos.ReferenciaValida(e.Antecedente.PreparacionRef) || len(e.Motivos) == 0 {
			return ErrRevisionLista
		}
		refs[e.Antecedente.PreparacionRef] = true
		todos := true
		for _, m := range e.Motivos {
			todos = todos && m.Subsanable
		}
		if todos != e.Subsanable {
			return ErrRevisionLista
		}
		if e.Subsanable {
			subsanables++
		}
	}
	r := l.Resumen
	if r.Admitidas != len(l.Admitidas) || r.Excluidas != len(l.Excluidas) || r.Solicitudes != len(refs) || r.ExcluidasSubsanables != subsanables {
		return ErrRevisionLista
	}
	return nil
}

// ComprobarRevisionProvisional exige que la nueva revisión conserve la anterior
// tal cual (mismas bases, catálogo y plazo, y cada decisión con sus motivos) y
// que solo añada solicitudes. Devuelve las incorporadas, ordenadas. Corregir
// una decisión ya tomada es otro acto y no cabe aquí.
func ComprobarRevisionProvisional(anterior, nueva ListaAdmisionProvisional) ([]string, error) {
	if ValidarListaProvisional(anterior) != nil || ValidarListaProvisional(nueva) != nil ||
		nueva.ListaRef != anterior.ListaRef || nueva.Revision != anterior.Revision+1 ||
		nueva.Bases != anterior.Bases || nueva.Catalogo != anterior.Catalogo || nueva.PlazoSubsanacion != anterior.PlazoSubsanacion {
		return nil, ErrRevisionLista
	}
	admitidas := make(map[string]SolicitudAdmitida, len(nueva.Admitidas))
	for _, a := range nueva.Admitidas {
		admitidas[a.Antecedente.PreparacionRef] = a
	}
	excluidas := make(map[string]SolicitudExcluida, len(nueva.Excluidas))
	for _, e := range nueva.Excluidas {
		excluidas[e.Antecedente.PreparacionRef] = e
	}
	previas := map[string]bool{}
	for _, a := range anterior.Admitidas {
		if actual, ok := admitidas[a.Antecedente.PreparacionRef]; !ok || actual != a {
			return nil, ErrRevisionLista
		}
		previas[a.Antecedente.PreparacionRef] = true
	}
	for _, e := range anterior.Excluidas {
		if actual, ok := excluidas[e.Antecedente.PreparacionRef]; !ok || !mismaExclusion(actual, e) {
			return nil, ErrRevisionLista
		}
		previas[e.Antecedente.PreparacionRef] = true
	}
	incorporadas := []string{}
	for ref := range admitidas {
		if !previas[ref] {
			incorporadas = append(incorporadas, ref)
		}
	}
	for ref := range excluidas {
		if !previas[ref] {
			incorporadas = append(incorporadas, ref)
		}
	}
	if len(incorporadas) == 0 {
		return nil, ErrRevisionLista
	}
	sort.Strings(incorporadas)
	return incorporadas, nil
}

// mismaExclusion compara antecedente, subsanabilidad y motivos como conjunto:
// reordenar los motivos en el material no cambia la decisión.
func mismaExclusion(a, b SolicitudExcluida) bool {
	if a.Antecedente != b.Antecedente || a.Subsanable != b.Subsanable || len(a.Motivos) != len(b.Motivos) {
		return false
	}
	previos := make(map[MotivoAplicado]bool, len(b.Motivos))
	for _, m := range b.Motivos {
		previos[m] = true
	}
	for _, m := range a.Motivos {
		if !previos[m] {
			return false
		}
	}
	return true
}
