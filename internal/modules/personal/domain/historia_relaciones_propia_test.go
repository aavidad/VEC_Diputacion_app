package domain

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func materialHistoriaRelacionesPrueba(t *testing.T) MaterialHistoriaRelacionesPropia {
	t.Helper()
	actor := actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba("pep_", "emp_"+strings.Repeat("b", 24)))
	m, e := NuevoMaterialHistoriaRelacionesPropia(SolicitudHistoriaRelacionesPropia{Actor: actor, Corte: CorteHistoriaRelacionesPropia{Desde: "2020-01-01", Hasta: "2027-01-01", ConocidoEn: corteFichaPropiaPrueba().ConocidoEn}})
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func historiaRelacionesPrueba(m MaterialHistoriaRelacionesPropia) HistoriaRelacionesPropia {
	traza := TrazaEmpleadoB2{Desde: "2020-01-01", RegistradaEn: time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC), Version: 2, ActoRef: "acto:relaciones:2", FuenteRef: "fuente:personal", FuenteVersion: 7}
	fila := RevisionRelacionPropia{RelacionRef: "rel_" + strings.Repeat("C", 24), Estado: "vigente", Regimen: "Funcionarial", Modalidad: "Temporal", Unidad: "Unidad sintética", Traza: traza}
	anterior := fila
	anterior.Traza.Version = 1
	anterior.Traza.FuenteVersion = 6
	anterior.Traza.ActoRef = "acto:relaciones:1"
	anterior.Traza.RegistradaEn = traza.RegistradaEn.Add(-time.Hour)
	anterior.Estado = "suspendida"
	return HistoriaRelacionesPropia{EmpleadoRef: m.EmpleadoRef(), Corte: m.Corte(), Cobertura: "parcial", Revisiones: []RevisionRelacionPropia{fila, anterior}}
}
func TestHistoriaRelacionesPropiaMaterialPropioLigaActorCorteYNoComparteMapas(t *testing.T) {
	m := materialHistoriaRelacionesPrueba(t)
	s := SolicitudHistoriaRelacionesPropia{Actor: m.Actor(), Corte: m.Corte()}
	igual, e := NuevoMaterialHistoriaRelacionesPropia(s)
	if e != nil || !bytes.Equal(m.Canonico(), igual.Canonico()) {
		t.Fatal("canon inestable", e)
	}
	s.Corte.ConocidoEn = s.Corte.ConocidoEn.Add(time.Microsecond)
	otro, _ := NuevoMaterialHistoriaRelacionesPropia(s)
	h, _ := m.HuellaSHA256()
	ho, _ := otro.HuellaSHA256()
	if h == ho || m.Recurso().Tipo != TipoRecursoHistoriaRelacionesPropia || m.Recurso().Atributos["efectos_hasta"] != "2027-01-01" {
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
	if _, e := NuevoMaterialHistoriaRelacionesPropia(s); !errors.Is(e, ErrHistoriaRelacionesPropiaDenegada) {
		t.Fatal("vínculo ausente aceptado", e)
	}
	s.Actor = m.Actor()
	s.Corte.Hasta = s.Corte.Desde
	if _, e := NuevoMaterialHistoriaRelacionesPropia(s); !errors.Is(e, ErrHistoriaRelacionesPropiaInvalida) {
		t.Fatal("periodo vacío aceptado")
	}
}
func TestHistoriaRelacionesPropiaConservaRevisionesYSeparacionTemporal(t *testing.T) {
	m := materialHistoriaRelacionesPrueba(t)
	h := historiaRelacionesPrueba(m)
	if e := h.ValidarPara(m); e != nil {
		t.Fatal("dos revisiones propias rechazadas", e)
	}
	for nombre, mutar := range map[string]func(*HistoriaRelacionesPropia){
		"empleado_ajeno": func(h *HistoriaRelacionesPropia) { h.EmpleadoRef = "emp_" + strings.Repeat("x", 24) },
		"corte_ajeno":    func(h *HistoriaRelacionesPropia) { h.Corte.Hasta = "2028-01-01" },
		"revision_futura": func(h *HistoriaRelacionesPropia) {
			h.Revisiones[0].Traza.RegistradaEn = h.Corte.ConocidoEn.Add(time.Microsecond)
		},
		"efectos_fuera":        func(h *HistoriaRelacionesPropia) { h.Revisiones[0].Traza.Desde = h.Corte.Hasta },
		"fuente_ausente":       func(h *HistoriaRelacionesPropia) { h.Revisiones[0].Traza.FuenteRef = "" },
		"estado_inventado":     func(h *HistoriaRelacionesPropia) { h.Revisiones[0].Estado = "reconocido" },
		"denominacion_control": func(h *HistoriaRelacionesPropia) { h.Revisiones[0].Puesto = "\x00" },
		"revision_repetida":    func(h *HistoriaRelacionesPropia) { h.Revisiones[1] = h.Revisiones[0] },
		"orden_invertido":      func(h *HistoriaRelacionesPropia) { h.Revisiones[0], h.Revisiones[1] = h.Revisiones[1], h.Revisiones[0] },
		"nulo":                 func(h *HistoriaRelacionesPropia) { h.Revisiones = nil },
		"cobertura_inventada":  func(h *HistoriaRelacionesPropia) { h.Cobertura = "confirmada" },
	} {
		t.Run(nombre, func(t *testing.T) {
			h := historiaRelacionesPrueba(m)
			mutar(&h)
			if !errors.Is(h.ValidarPara(m), ErrHistoriaRelacionesPropiaInvalida) {
				t.Fatal("fuente alterada aceptada")
			}
		})
	}
	h = historiaRelacionesPrueba(m)
	h.Revisiones = make([]RevisionRelacionPropia, LimiteHistoriaRelacionesPropia+1)
	if !errors.Is(h.ValidarPara(m), ErrHistoriaRelacionesPropiaExcedeLimite) {
		t.Fatal("exceso no explícito")
	}
	h = historiaRelacionesPrueba(m)
	h.Revisiones = []RevisionRelacionPropia{}
	h.Cobertura = "no_acreditada"
	if h.ValidarPara(m) != nil {
		t.Fatal("vacío convertido en cobertura completa")
	}
}

func TestHistoriaRelacionesPropiaReferenciasFielesAPersonal17(t *testing.T) {
	m := materialHistoriaRelacionesPrueba(t)
	for _, longitud := range []int{21, 22, 128, 129} {
		h := historiaRelacionesPrueba(m)
		h.Revisiones = h.Revisiones[:1]
		h.Revisiones[0].RelacionRef = "rel_" + strings.Repeat("Z", longitud)
		err := h.ValidarPara(m)
		valida := longitud == 22 || longitud == 128
		if (err == nil) != valida {
			t.Fatalf("límites SQL divergentes longitud=%d error=%v", longitud, err)
		}
	}
	h := historiaRelacionesPrueba(m)
	h.Revisiones[0].RelacionRef = "relacion:1"
	if h.ValidarPara(m) == nil {
		t.Fatal("referencia de fixture sin contrato admitida")
	}
}

func TestHistoriaRelacionesPropiaReciboEsLaAuditoriaAD8DelConsumo(t *testing.T) {
	h := strings.Repeat("d", 64)
	ref := "aud_v3_" + h[:32]
	if !ReciboHistoriaRelacionesPropiaLigado(ref, ref, h) {
		t.Fatal("PK AD8 rechazada")
	}
	for _, caso := range [][3]string{{ref, "aud_v3_" + strings.Repeat("e", 32), h}, {ref, ref, "corta"}, {ref, ref, strings.Repeat("e", 64)}, {"historia:relaciones:uuid", "historia:relaciones:uuid", h}} {
		if ReciboHistoriaRelacionesPropiaLigado(caso[0], caso[1], caso[2]) {
			t.Fatal("recibo sin consumo durable admitido")
		}
	}
	m := materialHistoriaRelacionesPrueba(t)
	historia := historiaRelacionesPrueba(m)
	historia.Revisiones[0].Traza.Hasta = historia.Revisiones[0].Traza.Desde
	if historia.ValidarPara(m) == nil {
		t.Fatal("periodo de relación vacío admitido")
	}
}
