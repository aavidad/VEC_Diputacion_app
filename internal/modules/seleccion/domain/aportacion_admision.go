package domain

import (
	"errors"

	meritos "vec-diputacion-granada/internal/modules/meritos/domain"
	vec "vec-diputacion-granada/internal/vec/domain"
)

var ErrAportacionAdmision = errors.New("seleccion.aportacion_admision.entrada_invalida")

const EsquemaMaterialAdmisionLocal = "seleccion.admision.material-local.v1"

// AntecedenteAdmision enlaza la revisión local con su material DTO serializado.
// La huella identifica ese material; no verifica bases, documentos o firma.
type AntecedenteAdmision struct {
	EsquemaMaterial      string `json:"esquema_material"`
	PreparacionRef       string `json:"preparacion_ref"`
	Revision             int    `json:"revision"`
	HuellaMaterialSHA256 string `json:"huella_material_sha256"`
}

// SoporteAportacionAdmision solo propone referencias exactas de documentos.
// Hecho enlaza una referencia ya esperada por el requisito S4;
// no declara un hecho nuevo ni modifica la versión o estado de RUM.
type SoporteAportacionAdmision struct {
	Hecho      ReferenciaHechoAdmision   `json:"hecho"`
	Documentos []vec.ReferenciaDocumento `json:"documentos"`
}

type PropuestaAportacionAdmision struct {
	Alcance            string                      `json:"alcance"`
	AportacionRef      string                      `json:"aportacion_ref"`
	Revision           int                         `json:"revision"`
	Antecedente        AntecedenteAdmision         `json:"antecedente"`
	RequisitoRef       string                      `json:"requisito_ref"`
	RequisitoVersion   string                      `json:"requisito_version"`
	SoportesPropuestos []SoporteAportacionAdmision `json:"soportes_propuestos"`
}

type AportacionAdmisionPreparada struct {
	Esquema              string                      `json:"esquema"`
	Propuesta            PropuestaAportacionAdmision `json:"propuesta"`
	Estado               string                      `json:"estado"`
	Bases                BasesAdmision               `json:"bases"`
	SolicitudContexto    *ContextoSolicitudLocal     `json:"solicitud_contexto,omitempty"`
	RequisitoAnterior    RevisionRequisitoAdmision   `json:"requisito_anterior"`
	Pendientes           []string                    `json:"pendientes"`
	Persistido           bool                        `json:"persistido"`
	Presentado           bool                        `json:"presentado"`
	RequerimientoEmitido bool                        `json:"requerimiento_emitido"`
	Resuelto             bool                        `json:"resuelto"`
}

// ValidarPropuestaAportacionAdmision comprueba únicamente el enlace local y
// la forma de las referencias. Aportar un archivo no resuelve un requisito.
func ValidarPropuestaAportacionAdmision(p PropuestaAportacionAdmision, a AntecedenteAdmision, r RequisitoAdmision) error {
	if p.Alcance != AlcanceAdmisionPreparacion || !meritos.ReferenciaValida(p.AportacionRef) ||
		p.Revision < 1 || p.Revision > 1_000_000 || p.Antecedente != a ||
		a.EsquemaMaterial != EsquemaMaterialAdmisionLocal || !meritos.ReferenciaValida(a.PreparacionRef) || a.Revision < 1 || !shaAdmision(a.HuellaMaterialSHA256) ||
		p.RequisitoRef != r.Referencia || p.RequisitoVersion != r.Version || len(p.SoportesPropuestos) > 32 {
		return ErrAportacionAdmision
	}
	hechosEsperados := map[string]int{}
	for _, h := range r.HechosEsperados {
		hechosEsperados[h.Referencia] = h.Version
	}
	hechos := map[string]bool{}
	documentos := map[string]bool{}
	totalDocumentos := 0
	for _, s := range p.SoportesPropuestos {
		if len(s.Documentos) > 16 {
			return ErrAportacionAdmision
		}
		h := s.Hecho
		if !meritos.ReferenciaValida(h.Referencia) || h.Version < 1 || hechosEsperados[h.Referencia] != h.Version || hechos[h.Referencia] {
			return ErrAportacionAdmision
		}
		hechos[h.Referencia] = true
		for _, d := range s.Documentos {
			if d.Validar() != nil || !meritos.ReferenciaValida(d.ID) || documentos[d.ID] {
				return ErrAportacionAdmision
			}
			documentos[d.ID] = true
			totalDocumentos++
		}
	}
	if totalDocumentos > 64 {
		return ErrAportacionAdmision
	}
	return nil
}

// PrepararAportacionAdmision conserva el resultado anterior, incluida cada
// causa pendiente. El objeto propuesto no tiene efectos sobre RUM ni S4.
func PrepararAportacionAdmision(p PropuestaAportacionAdmision, a AntecedenteAdmision, r RevisionRequisitoAdmision, bases BasesAdmision, solicitud *ContextoSolicitudLocal) (AportacionAdmisionPreparada, error) {
	if r.Estado != "pendiente" || ValidarPropuestaAportacionAdmision(p, a, r.Requisito) != nil {
		return AportacionAdmisionPreparada{}, ErrAportacionAdmision
	}
	p.SoportesPropuestos = append([]SoporteAportacionAdmision{}, p.SoportesPropuestos...)
	for i := range p.SoportesPropuestos {
		s := &p.SoportesPropuestos[i]
		s.Documentos = append([]vec.ReferenciaDocumento{}, s.Documentos...)
	}
	r.Requisito.HechosEsperados = append([]ReferenciaHechoAdmision{}, r.Requisito.HechosEsperados...)
	r.Causas = append([]string{}, r.Causas...)
	r.Soportes = append([]SoporteAdmision{}, r.Soportes...)
	if solicitud != nil {
		copia := *solicitud
		solicitud = &copia
	}
	pendientes := []string{"seleccion.aportacion_admision.pendiente.correspondencia", "seleccion.aportacion_admision.pendiente.documentos",
		"seleccion.aportacion_admision.pendiente.bases_subsanacion", "seleccion.aportacion_admision.pendiente.presentacion",
		"seleccion.aportacion_admision.pendiente.revision_competente"}
	if len(p.SoportesPropuestos) == 0 {
		pendientes = append(pendientes, "seleccion.aportacion_admision.pendiente.sin_soportes")
	}
	return AportacionAdmisionPreparada{Esquema: "vec.seleccion.aportacion-admision-preparacion.v1", Propuesta: p,
		Estado: "pendiente_revision_competente", Bases: bases, SolicitudContexto: solicitud, RequisitoAnterior: r, Pendientes: pendientes}, nil
}
