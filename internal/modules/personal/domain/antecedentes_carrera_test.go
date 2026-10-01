package domain

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func fichaAntecedentesCarrera() FichaEmpleadoB2 {
	c := CorteEmpleadoB2{VigenteEn: "2026-10-01", ConocidoEn: time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)}
	t := TrazaEmpleadoB2{Desde: "2020-01-01", RegistradaEn: c.ConocidoEn, Version: 2, ActoRef: "acto:relacion", FuenteRef: "fuente:personal", FuenteVersion: 3}
	ref := "rel_" + strings.Repeat("r", 24)
	return FichaEmpleadoB2{
		EmpleadoRef: "emp_" + strings.Repeat("e", 24), PersonaRef: "per_" + strings.Repeat("p", 24), OrganismoRef: "organismo:uno", Corte: c, Version: 4,
		Relaciones:  []RelacionRegistroEmpleadoB2{{RelacionRef: ref, OrganismoRef: "organismo:uno", RegimenRef: "regimen:funcionario", Estado: "vigente", Traza: t}},
		Servicios:   []ServicioReconocidoB2{{ServicioRef: "servicio:uno", RelacionRef: ref, Estado: "reconocido", PeriodoDesde: "2020-01-01", PeriodoHasta: "2020-12-31", DiasReconocidos: 365, Traza: t}},
		Situaciones: []SituacionEmpleadoB2{{SituacionRef: "situacion:uno", RelacionRef: ref, CodigoRef: "situacion:activo", Estado: "vigente", Traza: t}},
	}
}

func TestAntecedentesCarreraConservaFuenteSinInferir(t *testing.T) {
	f := fichaAntecedentesCarrera()
	f.Ocupaciones = []OcupacionEmpleadoB2{{PuestoRef: "puesto:no_es_autoridad_m"}}
	p, err := PrepararAntecedentesCarrera(f)
	if err != nil || p.Esquema != EsquemaAntecedentesCarrera || p.Alcance != "preparacion" || p.Version != f.Version || !corteRegistroB2Igual(p.Corte, f.Corte) {
		t.Fatalf("proyección: %+v, %v", p, err)
	}
	r := p.Relaciones[0]
	if r.Traza != f.Relaciones[0].Traza || r.Servicios[0].DiasReconocidos != 365 || r.Servicios[0].Traza != f.Servicios[0].Traza || r.Situaciones[0].Traza != f.Situaciones[0].Traza {
		t.Fatal("perdió procedencia/versiones o cambió los días del acto")
	}
	bruto, _ := json.Marshal(p)
	for _, prohibido := range []string{"no_es_autoridad_m", "persona_ref", "grado_personal\":", "antiguedad\":", "nivel_puesto\":"} {
		if strings.Contains(string(bruto), prohibido) {
			t.Fatalf("dato inferido o duplicado: %s", prohibido)
		}
	}
	if !slices.Contains(p.Pendientes, "fuente_institucional") || !slices.Contains(r.Pendientes, "puesto_nivel_m") || !slices.Contains(r.Pendientes, "grado_personal_h") {
		t.Fatal("ocultó dependencias")
	}
	f.Servicios[0].DiasReconocidos = 999
	if p.Relaciones[0].Servicios[0].DiasReconocidos != 365 {
		t.Fatal("alias con ficha")
	}
}

func TestAntecedentesCarreraConservaEstadosYSolapesSinSumar(t *testing.T) {
	f := fichaAntecedentesCarrera()
	for i, estado := range []string{"declarado", "comprobado", "reconocido"} {
		s := f.Servicios[0]
		s.ServicioRef = "servicio:" + string(rune('a'+i))
		s.Estado = estado
		f.Servicios = append(f.Servicios, s)
	}
	p, err := PrepararAntecedentesCarrera(f)
	if err != nil {
		t.Fatal(err)
	}
	r := p.Relaciones[0]
	if len(r.Servicios) != 4 || r.Servicios[1].Estado != "declarado" || r.Servicios[2].Estado != "comprobado" || !slices.Contains(r.Pendientes, "servicios_no_reconocidos") || !slices.Contains(r.Pendientes, "periodos_solapados") {
		t.Fatalf("estados/solapes: %+v", r)
	}
	for _, s := range r.Servicios {
		if s.DiasReconocidos != 365 {
			t.Fatal("sumó o repartió días")
		}
	}
}

func TestAntecedentesCarreraSeparaRelaciones(t *testing.T) {
	f := fichaAntecedentesCarrera()
	r := f.Relaciones[0]
	r.RelacionRef = "rel_" + strings.Repeat("s", 24)
	r.Estado = "finalizada"
	f.Relaciones = append(f.Relaciones, r)
	s := f.Servicios[0]
	s.ServicioRef = "servicio:dos"
	s.RelacionRef = r.RelacionRef
	f.Servicios = append(f.Servicios, s)
	p, err := PrepararAntecedentesCarrera(f)
	if err != nil || len(p.Relaciones) != 2 || len(p.Relaciones[0].Servicios) != 1 || p.Relaciones[1].Servicios[0].ServicioRef != "servicio:dos" || len(p.Relaciones[1].Situaciones) != 0 {
		t.Fatalf("mezcló relaciones: %+v %v", p, err)
	}
}

func TestAntecedentesCarreraAusenciaYFueraDeCorteSonPendientes(t *testing.T) {
	f := fichaAntecedentesCarrera()
	f.Relaciones = nil
	f.Servicios = nil
	f.Situaciones = nil
	p, err := PrepararAntecedentesCarrera(f)
	if err != nil || !slices.Contains(p.Pendientes, "sin_relacion") || p.Relaciones == nil {
		t.Fatal("vacío como completo")
	}
	f = fichaAntecedentesCarrera()
	f.Servicios[0].PeriodoHasta = "2026-10-02"
	p, err = PrepararAntecedentesCarrera(f)
	if err != nil || !slices.Contains(p.Relaciones[0].Pendientes, "servicios_fuera_corte") {
		t.Fatal("periodo posterior computable")
	}
}

func TestAntecedentesCarreraConservaHistoriaConocida(t *testing.T) {
	f := fichaAntecedentesCarrera()
	r := f.Relaciones[0]
	r.Estado = "finalizada"
	r.Traza.Version = 3
	r.Traza.Desde = "2026-10-02"
	f.Relaciones = append(f.Relaciones, r)
	s := f.Servicios[0]
	s.Estado = "comprobado"
	s.Traza.Version = 3
	f.Servicios = append(f.Servicios, s)
	p, err := PrepararAntecedentesCarrera(f)
	if err != nil || len(p.Relaciones) != 1 || len(p.Relaciones[0].Historia) != 2 || p.Relaciones[0].Traza.Version != 3 || len(p.Relaciones[0].Servicios) != 2 || !slices.Contains(p.Relaciones[0].Pendientes, "hechos_fuera_corte") || slices.Contains(p.Relaciones[0].Pendientes, "periodos_solapados") {
		t.Fatalf("historia/corte: %+v %v", p, err)
	}
}

func TestAntecedentesCarreraReconocimientoPosteriorNoQuedaPendiente(t *testing.T) {
	f := fichaAntecedentesCarrera()
	f.Servicios[0].Estado = "declarado"
	s := f.Servicios[0]
	s.Estado = "reconocido"
	s.Traza.Version = 3
	f.Servicios = append(f.Servicios, s)
	p, err := PrepararAntecedentesCarrera(f)
	if err != nil || len(p.Relaciones[0].Servicios) != 2 || p.Relaciones[0].Servicios[0].UltimaRevisionConocida || !p.Relaciones[0].Servicios[1].UltimaRevisionConocida || slices.Contains(p.Relaciones[0].Pendientes, "servicios_no_reconocidos") {
		t.Fatalf("confundió estado histórico con actual: %+v %v", p, err)
	}
}

func TestAntecedentesCarreraRechazaEntradaNoTrazable(t *testing.T) {
	casos := map[string]func(*FichaEmpleadoB2){
		"firma":               func(f *FichaEmpleadoB2) { f.FirmaOficial = true },
		"eficacia":            func(f *FichaEmpleadoB2) { f.EficaciaAdministrativa = true },
		"empleado":            func(f *FichaEmpleadoB2) { f.EmpleadoRef = "cuenta:no_es_empleo" },
		"persona":             func(f *FichaEmpleadoB2) { f.PersonaRef = "" },
		"version":             func(f *FichaEmpleadoB2) { f.Version = 0 },
		"corte":               func(f *FichaEmpleadoB2) { f.Corte.ConocidoEn = time.Time{} },
		"limite":              func(f *FichaEmpleadoB2) { f.Servicios = make([]ServicioReconocidoB2, LimiteFilasFichaPropia+1) },
		"otra_relacion":       func(f *FichaEmpleadoB2) { f.Servicios[0].RelacionRef = "rel_" + strings.Repeat("z", 24) },
		"otro_organismo":      func(f *FichaEmpleadoB2) { f.Relaciones[0].OrganismoRef = "organismo:otro" },
		"duplicada":           func(f *FichaEmpleadoB2) { f.Relaciones = append(f.Relaciones, f.Relaciones[0]) },
		"servicio_duplicado":  func(f *FichaEmpleadoB2) { f.Servicios = append(f.Servicios, f.Servicios[0]) },
		"estado":              func(f *FichaEmpleadoB2) { f.Servicios[0].Estado = "acreditado" },
		"periodo":             func(f *FichaEmpleadoB2) { f.Servicios[0].PeriodoHasta = "2019-12-31" },
		"dias":                func(f *FichaEmpleadoB2) { f.Servicios[0].DiasReconocidos = -1 },
		"acto":                func(f *FichaEmpleadoB2) { f.Servicios[0].Traza.ActoRef = "" },
		"fuente":              func(f *FichaEmpleadoB2) { f.Situaciones[0].Traza.FuenteVersion = 0 },
		"conocimiento_futuro": func(f *FichaEmpleadoB2) { f.Relaciones[0].Traza.RegistradaEn = f.Corte.ConocidoEn.Add(time.Hour) },
		"catalogo_ajeno": func(f *FichaEmpleadoB2) {
			f.Relaciones[0].CatalogoSnapshot.Regimen = &SnapshotEntradaCatalogoEmpleadoB2{Ref: "regimen:otro"}
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			f := fichaAntecedentesCarrera()
			cambiar(&f)
			p, err := PrepararAntecedentesCarrera(f)
			if !errors.Is(err, ErrAntecedentesCarreraInvalidos) || p.Esquema != "" || len(p.Relaciones) != 0 {
				t.Fatalf("devolvió datos inválidos: %+v %v", p, err)
			}
		})
	}
}

func TestAntecedentesCarreraNoResucitaReconocimientoAnterior(t *testing.T) {
	f := fichaAntecedentesCarrera()
	s := f.Servicios[0]
	s.Estado = "comprobado"
	s.Traza.Version = 3
	s.Traza.Desde = "2026-10-02"
	f.Servicios = append(f.Servicios, s)
	p, err := PrepararAntecedentesCarrera(f)
	if err != nil || !slices.Contains(p.Relaciones[0].Pendientes, "servicios_no_reconocidos") || !slices.Contains(p.Relaciones[0].Pendientes, "hechos_fuera_corte") || p.Relaciones[0].Servicios[0].UltimaRevisionConocida || !p.Relaciones[0].Servicios[1].UltimaRevisionConocida {
		t.Fatalf("resucitó versión anterior al filtrar por eficacia: %+v %v", p, err)
	}
}

func TestAntecedentesCarreraPriorizaConocimientoAntesDeVersion(t *testing.T) {
	for _, invertir := range []bool{false, true} {
		f := fichaAntecedentesCarrera()
		anterior := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
		posterior := anterior.Add(24 * time.Hour)
		f.Relaciones[0].Traza.Version = 3
		f.Relaciones[0].Traza.RegistradaEn = anterior
		r := f.Relaciones[0]
		r.Estado = "finalizada"
		r.Traza.Version = 2
		r.Traza.RegistradaEn = posterior
		f.Relaciones = append(f.Relaciones, r)
		f.Servicios[0].Traza.Version = 3
		f.Servicios[0].Traza.RegistradaEn = anterior
		s := f.Servicios[0]
		s.Estado = "comprobado"
		s.Traza.Version = 2
		s.Traza.RegistradaEn = posterior
		f.Servicios = append(f.Servicios, s)
		if invertir {
			slices.Reverse(f.Relaciones)
			slices.Reverse(f.Servicios)
		}
		p, err := PrepararAntecedentesCarrera(f)
		if err != nil {
			t.Fatal(err)
		}
		actual := p.Relaciones[0]
		if actual.Estado != "finalizada" || actual.Traza.Version != 2 || !actual.Traza.RegistradaEn.Equal(posterior) || !slices.Contains(actual.Pendientes, "servicios_no_reconocidos") {
			t.Fatalf("priorizó número de versión sobre conocimiento: %+v", actual)
		}
		for _, servicio := range actual.Servicios {
			if servicio.UltimaRevisionConocida != (servicio.Traza.Version == 2) {
				t.Fatal("último servicio no coincide con la fecha de conocimiento")
			}
		}
	}
}
