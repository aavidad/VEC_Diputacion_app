package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func fichaServiciosCER() FichaEmpleadoB2 {
	return FichaEmpleadoB2{EmpleadoRef: "emp_" + strings.Repeat("a", 24), OrganismoRef: "organismo:uno", Version: 3,
		Corte: CorteEmpleadoB2{VigenteEn: "2026-10-01", ConocidoEn: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)}, Servicios: []ServicioReconocidoB2{}}
}

func servicioCER(ref, estado, desde, hasta string) ServicioReconocidoB2 {
	return ServicioReconocidoB2{ServicioRef: ref, RelacionRef: "rel_" + strings.Repeat("r", 24), Estado: estado, ClaseRef: "clase:uno",
		PeriodoDesde: FechaCivil(desde), PeriodoHasta: FechaCivil(hasta), DiasReconocidos: 77,
		Traza:            TrazaEmpleadoB2{Desde: "2025-01-01", RegistradaEn: time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC), Version: 2, ActoRef: "acto:uno", FuenteRef: "fuente:uno", FuenteVersion: 4},
		CatalogoSnapshot: SnapshotCatalogoEmpleadoB2{ClaseServicio: &SnapshotEntradaCatalogoEmpleadoB2{OrganismoRef: "organismo:uno", Tipo: "clase_servicio", Ref: "clase:uno", Version: 1, Revision: 2, Denominacion: "Servicio previo", HuellaSHA256: strings.Repeat("a", 64), VigenteDesde: "2020-01-01", Estado: "publicada"}}}
}

func TestPreparacionServiciosCERConservaEstadosProcedenciaYLimites(t *testing.T) {
	f := fichaServiciosCER()
	f.Servicios = []ServicioReconocidoB2{servicioCER("servicio:uno", "declarado", "2020-01-01", "2020-02-01"), servicioCER("servicio:dos", "comprobado", "2020-01-01", "2020-02-01"), servicioCER("servicio:tres", "reconocido", "2020-01-01", "2020-02-01")}
	p, err := PrepararServiciosParaCertificados(f)
	if err != nil || p.EmpleadoRef != f.EmpleadoRef || p.Cobertura != "no_acreditada" || p.Estado != "preparacion_sintetica" || p.FirmaOficial || p.EficaciaAdministrativa || p.Corte != f.Corte || p.Version != 3 {
		t.Fatalf("límites: %+v, %v", p, err)
	}
	for i, fila := range p.Servicios {
		if fila.Estado != f.Servicios[i].Estado || fila.Traza != f.Servicios[i].Traza || fila.Solapado || fila.SeleccionTemporal != "incluido" {
			t.Fatalf("promoción o procedencia perdida: %+v", fila)
		}
	}
	b, _ := json.Marshal(p)
	if strings.Contains(string(b), "dias") || strings.Contains(string(b), "antiguedad") {
		t.Fatal("la preparación suma o publica un cómputo")
	}
}

func TestPreparacionServiciosCERMarcaSolapesSinFusionarPeriodos(t *testing.T) {
	f := fichaServiciosCER()
	f.Servicios = []ServicioReconocidoB2{servicioCER("servicio:uno", "reconocido", "2020-01-01", "2020-02-01"), servicioCER("servicio:dos", "reconocido", "2020-01-20", "2020-03-01"), servicioCER("servicio:tres", "reconocido", "2021-01-01", "2021-02-01")}
	f.Servicios[1].RelacionRef = "rel_" + strings.Repeat("s", 24)
	p, err := PrepararServiciosParaCertificados(f)
	if err != nil || len(p.Servicios) != 3 || !p.Servicios[0].Solapado || !p.Servicios[1].Solapado || p.Servicios[2].Solapado || p.Servicios[1].PeriodoDesde != "2020-01-20" {
		t.Fatalf("solapes y periodos: %+v, %v", p, err)
	}
}

func TestPreparacionServiciosCERSeleccionaLosDosEjesSinRecortar(t *testing.T) {
	for nombre, cambiar := range map[string]func(*ServicioReconocidoB2){
		"reconocimiento_futuro":  func(s *ServicioReconocidoB2) { s.Traza.Desde = "2026-10-02" },
		"revision_ya_finalizada": func(s *ServicioReconocidoB2) { s.Traza.Hasta = "2026-10-01" },
		"conocimiento_futuro":    func(s *ServicioReconocidoB2) { s.Traza.RegistradaEn = time.Date(2026, 10, 1, 12, 0, 1, 0, time.UTC) },
		"periodo_futuro":         func(s *ServicioReconocidoB2) { s.PeriodoHasta = "2026-10-02" },
	} {
		t.Run(nombre, func(t *testing.T) {
			f := fichaServiciosCER()
			s := servicioCER("servicio:uno", "reconocido", "2020-01-01", "2020-02-01")
			cambiar(&s)
			f.Servicios = []ServicioReconocidoB2{s}
			p, err := PrepararServiciosParaCertificados(f)
			if err != nil || p.Servicios[0].SeleccionTemporal != "fuera_corte" || p.Servicios[0].PeriodoHasta != s.PeriodoHasta || p.Servicios[0].Traza != s.Traza {
				t.Fatalf("selección: %+v, %v", p, err)
			}
		})
	}
}

func TestPreparacionServiciosCERNoAcreditaFuenteIncompleta(t *testing.T) {
	f := fichaServiciosCER()
	s := servicioCER("servicio:uno", "reconocido", "2020-01-01", "2020-02-01")
	s.Traza.FuenteRef = ""
	s.Traza.ActoRef = ""
	s.Traza.FuenteVersion = 0
	s.CatalogoSnapshot.ClaseServicio.OrganismoRef = "organismo:otro"
	f.Servicios = []ServicioReconocidoB2{s}
	p, err := PrepararServiciosParaCertificados(f)
	if err != nil || p.Servicios[0].SeleccionTemporal != "pendiente" || len(p.Servicios[0].Faltantes) != 4 || p.Servicios[0].Clase != "" || p.Cobertura != "no_acreditada" {
		t.Fatalf("fuente incompleta: %+v, %v", p, err)
	}
}

func TestPreparacionServiciosCERDeniegaAfirmacionesOficialesYEstadosDesconocidos(t *testing.T) {
	for nombre, cambiar := range map[string]func(*FichaEmpleadoB2){
		"firma": func(f *FichaEmpleadoB2) { f.FirmaOficial = true }, "eficacia": func(f *FichaEmpleadoB2) { f.EficaciaAdministrativa = true },
		"corte_ausente": func(f *FichaEmpleadoB2) { f.Corte = CorteEmpleadoB2{} },
		"estado_desconocido": func(f *FichaEmpleadoB2) {
			f.Servicios = []ServicioReconocidoB2{servicioCER("servicio:uno", "oficial", "2020-01-01", "2020-02-01")}
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			f := fichaServiciosCER()
			cambiar(&f)
			p, err := PrepararServiciosParaCertificados(f)
			if !errors.Is(err, ErrPreparacionServiciosInvalida) || p.Servicios != nil {
				t.Fatalf("aceptó afirmación inválida: %+v, %v", p, err)
			}
		})
	}
}

func TestPreparacionServiciosCERVacioSigueSinCoberturaYNoMutaFuente(t *testing.T) {
	f := fichaServiciosCER()
	p, err := PrepararServiciosParaCertificados(f)
	if err != nil || p.Servicios == nil || len(p.Servicios) != 0 || p.Cobertura != "no_acreditada" {
		t.Fatal("vacío no acredita cobertura", err)
	}
	f.Servicios = []ServicioReconocidoB2{servicioCER("servicio:uno", "reconocido", "2020-01-01", "2020-02-01")}
	p, _ = PrepararServiciosParaCertificados(f)
	f.Servicios[0].CatalogoSnapshot.ClaseServicio.Denominacion = "Alterada"
	f.Servicios[0].Traza.FuenteRef = "fuente:alterada"
	if p.Servicios[0].Clase != "Servicio previo" || p.Servicios[0].Traza.FuenteRef != "fuente:uno" {
		t.Fatal("alias de fuente mutable")
	}
}

func TestPreparacionServiciosCERUltimaRevisionAntesDeVigencia(t *testing.T) {
	for _, invertido := range []bool{false, true} {
		f := fichaServiciosCER()
		antigua := servicioCER("servicio:uno", "reconocido", "2020-01-01", "2020-02-01")
		ultima := antigua
		ultima.Estado = "comprobado"
		ultima.Traza.Version = 3
		ultima.Traza.RegistradaEn = time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
		ultima.Traza.Hasta = f.Corte.VigenteEn
		otro := servicioCER("servicio:dos", "reconocido", "2020-01-15", "2020-02-10")
		f.Servicios = []ServicioReconocidoB2{antigua, ultima, otro}
		if invertido {
			f.Servicios = []ServicioReconocidoB2{ultima, antigua, otro}
		}
		p, err := PrepararServiciosParaCertificados(f)
		if err != nil || len(p.Servicios) != 3 {
			t.Fatal("historia perdida", err)
		}
		for _, s := range p.Servicios {
			if s.Solapado {
				t.Fatal("revisión antigua produjo un solape")
			}
			if s.ServicioRef == "servicio:uno" && s.Traza.Version == 2 && s.SeleccionTemporal != "sustituido" {
				t.Fatal("resucitó el reconocimiento anterior")
			}
			if s.ServicioRef == "servicio:uno" && s.Traza.Version == 3 && s.SeleccionTemporal != "fuera_corte" {
				t.Fatal("la revisión cerrada parece vigente")
			}
		}
	}
}

func TestPreparacionServiciosCERRevisionFuturaNoCambiaHistoriaConocida(t *testing.T) {
	f := fichaServiciosCER()
	antigua := servicioCER("servicio:uno", "reconocido", "2020-01-01", "2020-02-01")
	futura := antigua
	futura.Estado = "declarado"
	futura.Traza.Version = 3
	futura.Traza.RegistradaEn = f.Corte.ConocidoEn.Add(time.Second)
	f.Servicios = []ServicioReconocidoB2{futura, antigua}
	p, err := PrepararServiciosParaCertificados(f)
	if err != nil || p.Servicios[0].SeleccionTemporal != "fuera_corte" || p.Servicios[1].SeleccionTemporal != "incluido" {
		t.Fatalf("corte de conocimiento alterado: %+v %v", p, err)
	}
	// Una revisión con el mismo instante gana por versión, antes de vigencia.
	f.Servicios[0].Traza.RegistradaEn = antigua.Traza.RegistradaEn
	f.Servicios[0].Traza.Hasta = f.Corte.VigenteEn
	p, err = PrepararServiciosParaCertificados(f)
	if err != nil || p.Servicios[1].SeleccionTemporal != "sustituido" || p.Servicios[0].SeleccionTemporal != "fuera_corte" {
		t.Fatal("desempate por versión o fin no aplicado", err)
	}
}
