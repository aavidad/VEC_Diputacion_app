package domain

import (
	"errors"
	"sort"

	calendarios "vec-diputacion-granada/internal/modules/calendarios/domain"
	meritos "vec-diputacion-granada/internal/modules/meritos/domain"
)

var ErrListaAdmision = errors.New("seleccion.lista_admision.entrada_invalida")

const (
	EsquemaListaAdmisionProvisional = "vec.seleccion.lista-admision-provisional.v1"
	DecisionAdmitida                = "admitida"
	DecisionExcluida                = "excluida"
	maximoMotivosCatalogo           = 64
	maximoMotivosDecision           = 16
	// MaximoSolicitudesLista acota memoria y trabajo antes de componer.
	MaximoSolicitudesLista = 2000
)

// PlazoSubsanacion es la regla configurada en el catálogo. Su vencimiento se
// calcula con Calendarios al publicar la lista, porque empieza el día
// siguiente a la publicación (Ley 39/2015, arts. 30 y 68).
type PlazoSubsanacion struct {
	Unidad   calendarios.UnidadPlazo `json:"unidad"`
	Cantidad int                     `json:"cantidad"`
}

// MotivoExclusion es una entrada del catálogo. El texto visible vive en los
// catálogos de idioma; aquí solo hay código y si admite subsanación.
type MotivoExclusion struct {
	Codigo     string `json:"codigo"`
	Subsanable bool   `json:"subsanable"`
}

// CatalogoAdmision versiona los motivos de exclusión y el plazo de
// subsanación de una convocatoria. PaqueteEjemplo marca el catálogo inventado
// y retirable mientras RRHH no confirme el suyo (DudaRef).
type CatalogoAdmision struct {
	Referencia       string            `json:"referencia"`
	Version          string            `json:"version"`
	PaqueteEjemplo   bool              `json:"paquete_ejemplo"`
	DudaRef          string            `json:"duda_ref,omitempty"`
	PlazoSubsanacion PlazoSubsanacion  `json:"plazo_subsanacion"`
	Motivos          []MotivoExclusion `json:"motivos"`
}

func (c CatalogoAdmision) Validar() error {
	if !meritos.ReferenciaValida(c.Referencia) || !meritos.ReferenciaValida(c.Version) ||
		(c.DudaRef != "" && !meritos.ReferenciaValida(c.DudaRef)) || (c.PaqueteEjemplo && c.DudaRef == "") ||
		len(c.Motivos) == 0 || len(c.Motivos) > maximoMotivosCatalogo {
		return ErrListaAdmision
	}
	// La regla debe ser calculable por Calendarios; la fecha solo permite
	// reutilizar sus límites por unidad, no fija ningún inicio.
	referencia, err := calendarios.NuevaFechaCivil(2026, 1, 1)
	plazo := calendarios.SolicitudPlazo{Inicio: referencia, Unidad: c.PlazoSubsanacion.Unidad, Cantidad: c.PlazoSubsanacion.Cantidad}
	if err != nil || plazo.Validar() != nil {
		return ErrListaAdmision
	}
	vistos := map[string]bool{}
	for _, m := range c.Motivos {
		if !meritos.ReferenciaValida(m.Codigo) || len(m.Codigo) > 64 || vistos[m.Codigo] {
			return ErrListaAdmision
		}
		vistos[m.Codigo] = true
	}
	return nil
}

// DecisionAdmision es la propuesta de RRHH para una revisión S4 exacta.
type DecisionAdmision struct {
	Antecedente AntecedenteAdmision `json:"antecedente"`
	Decision    string              `json:"decision"`
	Motivos     []string            `json:"motivos,omitempty"`
}

type MotivoAplicado struct {
	Codigo     string `json:"codigo"`
	Subsanable bool   `json:"subsanable"`
}

type SolicitudAdmitida struct {
	Antecedente AntecedenteAdmision `json:"antecedente"`
}

type SolicitudExcluida struct {
	Antecedente AntecedenteAdmision `json:"antecedente"`
	Motivos     []MotivoAplicado    `json:"motivos"`
	// Subsanable es cierto solo si todos sus motivos admiten subsanación.
	Subsanable bool `json:"subsanable"`
}

type ResumenListaAdmision struct {
	Solicitudes          int `json:"solicitudes"`
	Admitidas            int `json:"admitidas"`
	Excluidas            int `json:"excluidas"`
	ExcluidasSubsanables int `json:"excluidas_subsanables"`
}

type ReferenciaCatalogoAdmision struct {
	Referencia     string `json:"referencia"`
	Version        string `json:"version"`
	PaqueteEjemplo bool   `json:"paquete_ejemplo"`
	DudaRef        string `json:"duda_ref,omitempty"`
}

// ListaAdmisionProvisional es un borrador. No está aprobada, publicada ni
// persistida, y no identifica a las personas: nombre y documento enmascarado
// se resuelven al publicar mediante su autoridad.
type ListaAdmisionProvisional struct {
	Esquema          string                     `json:"esquema"`
	ListaRef         string                     `json:"lista_ref"`
	Revision         int                        `json:"revision"`
	Alcance          string                     `json:"alcance"`
	Estado           string                     `json:"estado"`
	Bases            BasesAdmision              `json:"bases"`
	Catalogo         ReferenciaCatalogoAdmision `json:"catalogo"`
	PlazoSubsanacion PlazoSubsanacion           `json:"plazo_subsanacion"`
	Vencimiento      string                     `json:"vencimiento_subsanacion"`
	Admitidas        []SolicitudAdmitida        `json:"admitidas"`
	Excluidas        []SolicitudExcluida        `json:"excluidas"`
	Resumen          ResumenListaAdmision       `json:"resumen"`
	Pendientes       []string                   `json:"pendientes"`
	Aprobada         bool                       `json:"aprobada"`
	Publicada        bool                       `json:"publicada"`
	Persistida       bool                       `json:"persistida"`
}

// ComponerListaProvisional reparte las decisiones en admitidas y excluidas.
// Cada exclusión lleva al menos un motivo del catálogo; una admisión, ninguno.
// El llamador garantiza que los antecedentes corresponden a revisiones S4
// reales de esas bases; aquí solo se comprueba forma, unicidad y catálogo.
func ComponerListaProvisional(listaRef string, revision int, bases BasesAdmision, catalogo CatalogoAdmision, decisiones []DecisionAdmision) (ListaAdmisionProvisional, error) {
	cero := ListaAdmisionProvisional{}
	if !meritos.ReferenciaValida(listaRef) || revision < 1 || revision > 1_000_000 || ValidarBasesAdmision(bases, nil) != nil ||
		catalogo.Validar() != nil || len(decisiones) == 0 || len(decisiones) > MaximoSolicitudesLista {
		return cero, ErrListaAdmision
	}
	motivos := map[string]bool{}
	for _, m := range catalogo.Motivos {
		motivos[m.Codigo] = m.Subsanable
	}
	out := ListaAdmisionProvisional{
		Esquema: EsquemaListaAdmisionProvisional, ListaRef: listaRef, Revision: revision, Alcance: AlcanceAdmisionPreparacion,
		Estado: "borrador_pendiente_aprobacion", Bases: bases,
		Catalogo:         ReferenciaCatalogoAdmision{catalogo.Referencia, catalogo.Version, catalogo.PaqueteEjemplo, catalogo.DudaRef},
		PlazoSubsanacion: catalogo.PlazoSubsanacion, Vencimiento: "pendiente_publicacion",
		Admitidas: []SolicitudAdmitida{}, Excluidas: []SolicitudExcluida{},
		Pendientes: []string{"seleccion.lista_admision.pendiente.aprobacion_competente",
			"seleccion.lista_admision.pendiente.identidad_publicacion", "seleccion.lista_admision.pendiente.vencimiento_al_publicar",
			"seleccion.lista_admision.pendiente.publicacion_oficial"},
	}
	if catalogo.PaqueteEjemplo {
		out.Pendientes = append(out.Pendientes, "seleccion.lista_admision.pendiente.catalogo_ejemplo")
	}
	vistas := map[string]bool{}
	for _, d := range decisiones {
		a := d.Antecedente
		if a.EsquemaMaterial != EsquemaMaterialAdmisionLocal || !meritos.ReferenciaValida(a.PreparacionRef) || a.Revision < 1 ||
			!shaAdmision(a.HuellaMaterialSHA256) || vistas[a.PreparacionRef] || len(d.Motivos) > maximoMotivosDecision {
			return cero, ErrListaAdmision
		}
		vistas[a.PreparacionRef] = true
		switch d.Decision {
		case DecisionAdmitida:
			if len(d.Motivos) != 0 {
				return cero, ErrListaAdmision
			}
			out.Admitidas = append(out.Admitidas, SolicitudAdmitida{Antecedente: a})
		case DecisionExcluida:
			if len(d.Motivos) == 0 {
				return cero, ErrListaAdmision
			}
			excluida := SolicitudExcluida{Antecedente: a, Motivos: make([]MotivoAplicado, 0, len(d.Motivos)), Subsanable: true}
			aplicados := map[string]bool{}
			for _, codigo := range d.Motivos {
				subsanable, existe := motivos[codigo]
				if !existe || aplicados[codigo] {
					return cero, ErrListaAdmision
				}
				aplicados[codigo] = true
				excluida.Motivos = append(excluida.Motivos, MotivoAplicado{codigo, subsanable})
				excluida.Subsanable = excluida.Subsanable && subsanable
			}
			out.Excluidas = append(out.Excluidas, excluida)
			if excluida.Subsanable {
				out.Resumen.ExcluidasSubsanables++
			}
		default:
			return cero, ErrListaAdmision
		}
	}
	// Orden estable por referencia: el orden alfabético por apellidos exige
	// la identidad, que se resuelve al publicar.
	sort.Slice(out.Admitidas, func(i, j int) bool {
		return out.Admitidas[i].Antecedente.PreparacionRef < out.Admitidas[j].Antecedente.PreparacionRef
	})
	sort.Slice(out.Excluidas, func(i, j int) bool {
		return out.Excluidas[i].Antecedente.PreparacionRef < out.Excluidas[j].Antecedente.PreparacionRef
	})
	out.Resumen.Solicitudes = len(decisiones)
	out.Resumen.Admitidas = len(out.Admitidas)
	out.Resumen.Excluidas = len(out.Excluidas)
	return out, nil
}
