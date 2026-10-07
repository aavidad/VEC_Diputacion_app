package domain

import (
	"errors"
	"regexp"
)

const EsquemaPlanCompetenciaFirmaV2 = "ct.plan-competencia-firma.v2"

var ErrPlanCompetenciaFirmaV2 = errors.New("ct: plan competencia firma v2 no disponible")
var clavePlanFirmaV2 = regexp.MustCompile(`^[a-z][a-zA-Z0-9_.:-]{2,511}$`)

// VersionPlanFirmaV2 identifica bytes publicados; no acredita su autoridad.
type VersionPlanFirmaV2 struct {
	Referencia   string
	Version      uint64
	HuellaSHA256 string
}

func (v VersionPlanFirmaV2) Valida() bool {
	return clavePlanFirmaV2.MatchString(v.Referencia) && v.Version > 0 && v.Version <= 9007199254740991 && HuellaSHA256FirmaValida(v.HuellaSHA256)
}

// CompetenciaPasoFirmaV2 es un mapeo explícito del perfil lógico del circuito.
// Cargo y rol conservan sus autoridades en Personal y AUT. El enlace del
// ocupante y la persona nunca forman parte de este catálogo.
type CompetenciaPasoFirmaV2 struct {
	EntradaClave                                    string
	Circuito                                        VersionPlanFirmaV2
	Documento, PasoRef                              string
	PasoOrden                                       uint64
	PerfilEsperadoRef, RolID, CargoRef              string
	OrganizacionRef, UnidadRef                      string
	Accion, Finalidad, TipoRecurso, EsquemaContexto string
	MapeoVersion                                    uint64
	MapeoFuenteRef                                  string
}

func (p CompetenciaPasoFirmaV2) Validar() error {
	if !p.Circuito.Valida() || p.PasoOrden < 1 || p.PasoOrden > 16 || p.MapeoVersion < 1 || p.MapeoVersion > 9007199254740991 {
		return ErrPlanCompetenciaFirmaV2
	}
	for _, v := range []string{p.EntradaClave, p.Documento, p.PasoRef, p.PerfilEsperadoRef, p.RolID, p.CargoRef, p.OrganizacionRef, p.UnidadRef, p.Accion, p.Finalidad, p.TipoRecurso, p.EsquemaContexto, p.MapeoFuenteRef} {
		if !clavePlanFirmaV2.MatchString(v) {
			return ErrPlanCompetenciaFirmaV2
		}
	}
	return nil
}

type PlanCompetenciaFirmaV2 struct {
	Version   VersionPlanFirmaV2
	FuenteRef string
	Pasos     []CompetenciaPasoFirmaV2
}

func (p PlanCompetenciaFirmaV2) Validar() error {
	if !p.Version.Valida() || !clavePlanFirmaV2.MatchString(p.FuenteRef) || len(p.Pasos) == 0 || len(p.Pasos) > 64 {
		return ErrPlanCompetenciaFirmaV2
	}
	entradas := map[string]bool{}
	selecciones := map[string]bool{}
	for _, paso := range p.Pasos {
		if paso.Validar() != nil || entradas[paso.EntradaClave] {
			return ErrPlanCompetenciaFirmaV2
		}
		clave := paso.Documento + "\x00" + paso.PasoRef + "\x00" + paso.PerfilEsperadoRef + "\x00" + paso.OrganizacionRef + "\x00" + paso.UnidadRef
		if selecciones[clave] {
			return ErrPlanCompetenciaFirmaV2
		}
		entradas[paso.EntradaClave], selecciones[clave] = true, true
	}
	return nil
}

// Seleccionar exige todos los selectores exactos, sin precedencia por etiqueta
// o acumulación de perfiles. Una ambigüedad deniega la selección entera.
func (p PlanCompetenciaFirmaV2) Seleccionar(circuito VersionPlanFirmaV2, documento, paso, perfil, organizacion, unidad string) (CompetenciaPasoFirmaV2, error) {
	if p.Validar() != nil {
		return CompetenciaPasoFirmaV2{}, ErrPlanCompetenciaFirmaV2
	}
	var elegido CompetenciaPasoFirmaV2
	encontrados := 0
	for _, candidato := range p.Pasos {
		if candidato.Circuito == circuito && candidato.Documento == documento && candidato.PasoRef == paso && candidato.PerfilEsperadoRef == perfil && candidato.OrganizacionRef == organizacion && candidato.UnidadRef == unidad {
			elegido = candidato
			encontrados++
		}
	}
	if encontrados != 1 {
		return CompetenciaPasoFirmaV2{}, ErrPlanCompetenciaFirmaV2
	}
	return elegido, nil
}
