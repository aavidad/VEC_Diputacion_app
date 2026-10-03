package application

import (
	"context"
	"time"

	meritos "vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
)

// ContextoHechosPreparacion conserva las referencias y el hito aportados por
// el ensayo. Uso separa requisito y mérito; no decide aplicabilidad ni puntos.
type ContextoHechosPreparacion struct {
	ConvocatoriaRef string `json:"convocatoria_ref"`
	BasesRef        string `json:"bases_ref"`
	BasesVersion    int    `json:"bases_version"`
	SolicitudRef    string `json:"solicitud_ref"`
	HitoRef         string `json:"hito_ref"`
	Uso             string `json:"uso"`
}

type SolicitudHechosPreparacion struct {
	Alcance  string                          `json:"alcance"`
	Contexto ContextoHechosPreparacion       `json:"contexto"`
	Selector ports.SelectorHechosPreparacion `json:"selector"`
}

type PreparacionHechosProceso struct {
	Alcance           string                    `json:"alcance"`
	Contexto          ContextoHechosPreparacion `json:"contexto"`
	FechaCorte        string                    `json:"fecha_corte"`
	VersionPaquete    string                    `json:"version_paquete"`
	Hechos            []ports.HechoPreparado    `json:"hechos"`
	LecturaAutorizada bool                      `json:"lectura_autorizada"`
	DecisionReal      bool                      `json:"decision_real"`
	Pendientes        []string                  `json:"pendientes"`
}

// ConsultarHechosPreparacion usa exclusivamente el puerto sintético. false en
// LecturaAutorizada indica que este recorrido no obtiene autorización real;
// no afirma haber observado una denegación de la autoridad central.
func ConsultarHechosPreparacion(ctx context.Context, s SolicitudHechosPreparacion, lector ports.LectorHechosPreparacion) (PreparacionHechosProceso, error) {
	vacio := PreparacionHechosProceso{}
	c := s.Contexto
	if ctx == nil || lector == nil || s.Alcance != meritos.AlcanceSintetico ||
		!meritos.ReferenciaValida(c.ConvocatoriaRef) || !meritos.ReferenciaValida(c.BasesRef) || c.BasesVersion < 1 ||
		!meritos.ReferenciaValida(c.SolicitudRef) || !meritos.ReferenciaValida(c.HitoRef) || (c.Uso != "requisito" && c.Uso != "merito") ||
		!meritos.ReferenciaValida(s.Selector.PersonaRef) || len(s.Selector.Hechos) == 0 || len(s.Selector.Hechos) > 1000 {
		return vacio, ports.ErrHechosPreparacion
	}
	if _, err := time.Parse("2006-01-02", s.Selector.FechaCorte); err != nil {
		return vacio, ports.ErrHechosPreparacion
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	vistos := make(map[string]bool)
	for _, h := range s.Selector.Hechos {
		if !meritos.ReferenciaValida(h.Referencia) || h.VersionEsperada < 1 || vistos[h.Referencia] {
			return vacio, ports.ErrHechosPreparacion
		}
		vistos[h.Referencia] = true
	}
	hechos, err := lector.LeerHechosSinteticos(ctx, s.Selector)
	if err != nil {
		return vacio, err
	}
	if hechos.Alcance != meritos.AlcanceSintetico || !meritos.ReferenciaValida(hechos.VersionPaquete) || len(hechos.Hechos) != len(s.Selector.Hechos) {
		return vacio, ports.ErrHechosPreparacion
	}
	for i, h := range hechos.Hechos {
		if h.Referencia != s.Selector.Hechos[i].Referencia || h.Version != s.Selector.Hechos[i].VersionEsperada || len(h.Pendientes) == 0 {
			return vacio, ports.ErrHechosPreparacion
		}
	}
	return PreparacionHechosProceso{Alcance: meritos.AlcanceSintetico, Contexto: c, FechaCorte: s.Selector.FechaCorte,
		VersionPaquete: hechos.VersionPaquete, Hechos: hechos.Hechos,
		Pendientes: []string{"meritos.pendiente.lectura_autorizada_proceso", "meritos.pendiente.correspondencia_bases", "meritos.pendiente.valoracion_proceso"}}, nil
}
