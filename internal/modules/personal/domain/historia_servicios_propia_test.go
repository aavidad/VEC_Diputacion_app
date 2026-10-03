package domain

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func materialHistoriaPrueba(t *testing.T) MaterialHistoriaServiciosPropia {
	t.Helper()
	actor := actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba("pep_", "emp_"+strings.Repeat("b", 24)))
	m, e := NuevoMaterialHistoriaServiciosPropia(SolicitudHistoriaServiciosPropia{Actor: actor, Corte: CorteHistoriaServiciosPropia{Desde: "2020-01-01", Hasta: "2027-01-01", ConocidoEn: corteFichaPropiaPrueba().ConocidoEn}})
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func historiaPrueba(m MaterialHistoriaServiciosPropia) HistoriaServiciosPropia {
	traza := TrazaEmpleadoB2{Desde: "2020-01-01", RegistradaEn: time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC), Version: 2, ActoRef: "acto:servicios:2", FuenteRef: "fuente:personal", FuenteVersion: 7}
	fila := RevisionServicioPropio{ServicioRef: "servicio:1", RelacionRef: "rel_" + strings.Repeat("c", 24), PeriodoDesde: "2019-01-01", PeriodoHasta: "2019-12-31", DiasReconocidos: 365, Estado: "reconocido", Clase: "", Traza: traza}
	anterior := fila
	anterior.Traza.Version = 1
	anterior.Traza.FuenteVersion = 6
	anterior.Traza.ActoRef = "acto:servicios:1"
	anterior.Traza.RegistradaEn = traza.RegistradaEn.Add(-time.Hour)
	anterior.Estado = "declarado"
	return HistoriaServiciosPropia{EmpleadoRef: m.EmpleadoRef(), Corte: m.Corte(), Cobertura: "parcial", Revisiones: []RevisionServicioPropio{fila, anterior}}
}
func TestHistoriaServiciosPropiaMaterialPropioLigaActorCorteYNoComparteMapas(t *testing.T) {
	m := materialHistoriaPrueba(t)
	s := SolicitudHistoriaServiciosPropia{Actor: m.Actor(), Corte: m.Corte()}
	igual, e := NuevoMaterialHistoriaServiciosPropia(s)
	if e != nil || !bytes.Equal(m.Canonico(), igual.Canonico()) {
		t.Fatal("canon inestable", e)
	}
	s.Corte.ConocidoEn = s.Corte.ConocidoEn.Add(time.Microsecond)
	otro, _ := NuevoMaterialHistoriaServiciosPropia(s)
	h, _ := m.HuellaSHA256()
	ho, _ := otro.HuellaSHA256()
	if h == ho || m.Recurso().Tipo != TipoRecursoHistoriaServiciosPropia || m.Recurso().Atributos["efectos_hasta"] != "2027-01-01" {
		t.Fatal("corte o operación no ligados")
	}
	r := m.Recurso()
	r.Ambitos["empleado_ref"] = "otro"
	b := m.Canonico()
	b[0] = 'x'
	if m.Recurso().Ambitos["empleado_ref"] != m.EmpleadoRef() || m.Canonico()[0] != '{' {
		t.Fatal("material mutable")
	}
	s.Actor = actorFichaPropiaPrueba(t)
	if _, e := NuevoMaterialHistoriaServiciosPropia(s); !errors.Is(e, ErrHistoriaServiciosPropiaDenegada) {
		t.Fatal("vínculo ausente aceptado", e)
	}
	s.Actor = m.Actor()
	s.Corte.Hasta = s.Corte.Desde
	if _, e := NuevoMaterialHistoriaServiciosPropia(s); !errors.Is(e, ErrHistoriaServiciosPropiaInvalida) {
		t.Fatal("periodo vacío aceptado")
	}
}
func TestHistoriaServiciosPropiaConservaRevisionesYSeparacionTemporal(t *testing.T) {
	m := materialHistoriaPrueba(t)
	h := historiaPrueba(m)
	if e := h.ValidarPara(m); e != nil {
		t.Fatal("dos revisiones propias rechazadas", e)
	}
	for nombre, mutar := range map[string]func(*HistoriaServiciosPropia){
		"empleado_ajeno": func(h *HistoriaServiciosPropia) { h.EmpleadoRef = "emp_" + strings.Repeat("x", 24) },
		"corte_ajeno":    func(h *HistoriaServiciosPropia) { h.Corte.Hasta = "2028-01-01" },
		"revision_futura": func(h *HistoriaServiciosPropia) {
			h.Revisiones[0].Traza.RegistradaEn = h.Corte.ConocidoEn.Add(time.Microsecond)
		},
		"efectos_fuera":       func(h *HistoriaServiciosPropia) { h.Revisiones[0].Traza.Desde = h.Corte.Hasta },
		"fuente_ausente":      func(h *HistoriaServiciosPropia) { h.Revisiones[0].Traza.FuenteRef = "" },
		"revision_repetida":   func(h *HistoriaServiciosPropia) { h.Revisiones[1] = h.Revisiones[0] },
		"orden_invertido":     func(h *HistoriaServiciosPropia) { h.Revisiones[0], h.Revisiones[1] = h.Revisiones[1], h.Revisiones[0] },
		"nulo":                func(h *HistoriaServiciosPropia) { h.Revisiones = nil },
		"cobertura_inventada": func(h *HistoriaServiciosPropia) { h.Cobertura = "confirmada" },
	} {
		t.Run(nombre, func(t *testing.T) {
			h := historiaPrueba(m)
			mutar(&h)
			if !errors.Is(h.ValidarPara(m), ErrHistoriaServiciosPropiaInvalida) {
				t.Fatal("fuente alterada aceptada")
			}
		})
	}
	h = historiaPrueba(m)
	h.Revisiones = make([]RevisionServicioPropio, LimiteHistoriaServiciosPropia+1)
	if !errors.Is(h.ValidarPara(m), ErrHistoriaServiciosPropiaExcedeLimite) {
		t.Fatal("exceso no explícito")
	}
	h = historiaPrueba(m)
	h.Revisiones = []RevisionServicioPropio{}
	h.Cobertura = "no_acreditada"
	if h.ValidarPara(m) != nil {
		t.Fatal("vacío convertido en cobertura completa")
	}
}
