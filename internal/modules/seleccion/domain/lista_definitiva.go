package domain

import (
	"errors"
	"sort"

	meritos "vec-diputacion-granada/internal/modules/meritos/domain"
)

var ErrListaDefinitiva = errors.New("seleccion.lista_definitiva.entrada_invalida")

const (
	EsquemaListaAdmisionDefinitiva = "vec.seleccion.lista-admision-definitiva.v1"
	EsquemaAntecedenteLista        = "seleccion.admision.lista-provisional.v1"

	// Resultado de lo que la persona excluida presentó en el plazo.
	ResolucionEstimada     = "estimada"
	ResolucionDesestimada  = "desestimada"
	ResolucionNoPresentada = "no_presentada"

	// Vía del escrito. La subsanación solo cabe en motivos subsanables; una
	// reclamación puede corregir cualquier motivo, también un error de RRHH.
	ViaSubsanacion = "subsanacion"
	ViaReclamacion = "reclamacion"

	OrigenProvisional = "provisional"
)

// AntecedenteLista identifica la lista provisional exacta por la huella de su
// serialización. No acredita aprobación ni publicación de esa lista.
type AntecedenteLista struct {
	Esquema      string `json:"esquema"`
	ListaRef     string `json:"lista_ref"`
	Revision     int    `json:"revision"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (a AntecedenteLista) Validar() error {
	if a.Esquema != EsquemaAntecedenteLista || !meritos.ReferenciaValida(a.ListaRef) || a.Revision < 1 || !shaAdmision(a.HuellaSHA256) {
		return ErrListaDefinitiva
	}
	return nil
}

// ResolucionSubsanacion es la propuesta de RRHH para una exclusión de la
// provisional. MotivosPersistentes solo se usa al desestimar.
type ResolucionSubsanacion struct {
	Antecedente         AntecedenteAdmision `json:"antecedente"`
	Resultado           string              `json:"resultado"`
	Via                 string              `json:"via,omitempty"`
	MotivosPersistentes []string            `json:"motivos_persistentes,omitempty"`
}

type SolicitudAdmitidaDefinitiva struct {
	Antecedente AntecedenteAdmision `json:"antecedente"`
	// Origen es "provisional" o la vía por la que se estimó el escrito.
	Origen string `json:"origen"`
}

type SolicitudExcluidaDefinitiva struct {
	Antecedente AntecedenteAdmision `json:"antecedente"`
	Motivos     []MotivoAplicado    `json:"motivos"`
	Resolucion  string              `json:"resolucion"`
	Via         string              `json:"via,omitempty"`
}

type ResumenListaDefinitiva struct {
	Solicitudes          int `json:"solicitudes"`
	Admitidas            int `json:"admitidas"`
	AdmitidasTrasEscrito int `json:"admitidas_tras_escrito"`
	Excluidas            int `json:"excluidas"`
	ExcluidasSinEscrito  int `json:"excluidas_sin_escrito"`
}

// ListaAdmisionDefinitiva es un borrador: no está aprobada, publicada ni
// persistida y no identifica a las personas.
type ListaAdmisionDefinitiva struct {
	Esquema     string                        `json:"esquema"`
	ListaRef    string                        `json:"lista_ref"`
	Revision    int                           `json:"revision"`
	Alcance     string                        `json:"alcance"`
	Estado      string                        `json:"estado"`
	Bases       BasesAdmision                 `json:"bases"`
	Catalogo    ReferenciaCatalogoAdmision    `json:"catalogo"`
	Provisional AntecedenteLista              `json:"provisional"`
	Admitidas   []SolicitudAdmitidaDefinitiva `json:"admitidas"`
	Excluidas   []SolicitudExcluidaDefinitiva `json:"excluidas"`
	Resumen     ResumenListaDefinitiva        `json:"resumen"`
	Pendientes  []string                      `json:"pendientes"`
	Aprobada    bool                          `json:"aprobada"`
	Publicada   bool                          `json:"publicada"`
	Persistida  bool                          `json:"persistida"`
}

// ComponerListaDefinitiva parte de la provisional ya comprobada por el
// llamador. Las admitidas se conservan: excluir a quien ya estaba admitido
// exige audiencia y no es un paso de este borrador. Cada excluida necesita
// exactamente una resolución de su antecedente.
func ComponerListaDefinitiva(listaRef string, revision int, provisional ListaAdmisionProvisional, antecedente AntecedenteLista, resoluciones []ResolucionSubsanacion) (ListaAdmisionDefinitiva, error) {
	cero := ListaAdmisionDefinitiva{}
	if !meritos.ReferenciaValida(listaRef) || revision < 1 || revision > 1_000_000 || antecedente.Validar() != nil ||
		antecedente.ListaRef != provisional.ListaRef || antecedente.Revision != provisional.Revision ||
		provisional.Esquema != EsquemaListaAdmisionProvisional || listaRef == provisional.ListaRef ||
		len(resoluciones) != len(provisional.Excluidas) {
		return cero, ErrListaDefinitiva
	}
	porRef := make(map[string]ResolucionSubsanacion, len(resoluciones))
	for _, r := range resoluciones {
		if _, repetida := porRef[r.Antecedente.PreparacionRef]; repetida || len(r.MotivosPersistentes) > maximoMotivosDecision {
			return cero, ErrListaDefinitiva
		}
		porRef[r.Antecedente.PreparacionRef] = r
	}
	out := ListaAdmisionDefinitiva{
		Esquema: EsquemaListaAdmisionDefinitiva, ListaRef: listaRef, Revision: revision, Alcance: AlcanceAdmisionPreparacion,
		Estado: "borrador_pendiente_aprobacion", Bases: provisional.Bases, Catalogo: provisional.Catalogo, Provisional: antecedente,
		Admitidas: make([]SolicitudAdmitidaDefinitiva, 0, len(provisional.Admitidas)+len(provisional.Excluidas)),
		Excluidas: []SolicitudExcluidaDefinitiva{},
		Pendientes: []string{"seleccion.lista_definitiva.pendiente.aprobacion_competente",
			"seleccion.lista_definitiva.pendiente.pie_recursos",
			// Sin registro de revisiones publicadas, nadie puede comprobar aquí que
			// no exista una revisión posterior de la provisional: lo comprueba RRHH.
			"seleccion.lista_definitiva.pendiente.ultima_revision",
			"seleccion.lista_definitiva.pendiente.registro_escritos", "seleccion.lista_definitiva.pendiente.identidad_publicacion",
			"seleccion.lista_definitiva.pendiente.publicacion_oficial"},
	}
	if provisional.Catalogo.PaqueteEjemplo {
		out.Pendientes = append(out.Pendientes, "seleccion.lista_admision.pendiente.catalogo_ejemplo")
	}
	for _, a := range provisional.Admitidas {
		out.Admitidas = append(out.Admitidas, SolicitudAdmitidaDefinitiva{Antecedente: a.Antecedente, Origen: OrigenProvisional})
	}
	// Recorrer la provisional, ya ordenada, mantiene el orden por referencia.
	for _, e := range provisional.Excluidas {
		r, ok := porRef[e.Antecedente.PreparacionRef]
		if !ok || r.Antecedente != e.Antecedente {
			return cero, ErrListaDefinitiva
		}
		switch r.Resultado {
		case ResolucionEstimada:
			if len(r.MotivosPersistentes) != 0 || (r.Via != ViaReclamacion && (r.Via != ViaSubsanacion || !e.Subsanable)) {
				return cero, ErrListaDefinitiva
			}
			out.Admitidas = append(out.Admitidas, SolicitudAdmitidaDefinitiva{Antecedente: e.Antecedente, Origen: r.Via})
			out.Resumen.AdmitidasTrasEscrito++
		case ResolucionDesestimada:
			if r.Via != ViaSubsanacion && r.Via != ViaReclamacion {
				return cero, ErrListaDefinitiva
			}
			motivos, err := motivosPersistentes(e.Motivos, r.MotivosPersistentes)
			if err != nil {
				return cero, err
			}
			out.Excluidas = append(out.Excluidas, SolicitudExcluidaDefinitiva{Antecedente: e.Antecedente, Motivos: motivos, Resolucion: r.Resultado, Via: r.Via})
		case ResolucionNoPresentada:
			if r.Via != "" || len(r.MotivosPersistentes) != 0 {
				return cero, ErrListaDefinitiva
			}
			out.Excluidas = append(out.Excluidas, SolicitudExcluidaDefinitiva{Antecedente: e.Antecedente,
				Motivos: append([]MotivoAplicado{}, e.Motivos...), Resolucion: r.Resultado})
			out.Resumen.ExcluidasSinEscrito++
		default:
			return cero, ErrListaDefinitiva
		}
	}
	ordenarAdmitidasDefinitivas(out.Admitidas)
	out.Resumen.Solicitudes = len(out.Admitidas) + len(out.Excluidas)
	out.Resumen.Admitidas = len(out.Admitidas)
	out.Resumen.Excluidas = len(out.Excluidas)
	return out, nil
}

// motivosPersistentes exige un subconjunto no vacío y sin repetidos de los
// motivos de la provisional: al desestimar no aparecen motivos nuevos.
func motivosPersistentes(anteriores []MotivoAplicado, codigos []string) ([]MotivoAplicado, error) {
	if len(codigos) == 0 {
		return nil, ErrListaDefinitiva
	}
	previos := make(map[string]MotivoAplicado, len(anteriores))
	for _, m := range anteriores {
		previos[m.Codigo] = m
	}
	vistos := map[string]bool{}
	out := make([]MotivoAplicado, 0, len(codigos))
	for _, c := range codigos {
		m, ok := previos[c]
		if !ok || vistos[c] {
			return nil, ErrListaDefinitiva
		}
		vistos[c] = true
		out = append(out, m)
	}
	return out, nil
}

func ordenarAdmitidasDefinitivas(a []SolicitudAdmitidaDefinitiva) {
	sort.Slice(a, func(i, j int) bool { return a[i].Antecedente.PreparacionRef < a[j].Antecedente.PreparacionRef })
}
