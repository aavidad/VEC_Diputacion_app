package domain

import (
	"strings"
	"testing"
	"time"
)

func fuenteUnidadFixture() FuenteUnidadInicialAdminV1 {
	return FuenteUnidadInicialAdminV1{Version: 1, Referencia: "fuente:unidad_admin_sintetica", Entorno: "desarrollo", AlcanceFuente: "sintetico_declarado", ActoTecnicoRef: "acto_tecnico:unidad_admin_sintetica", Unidad: UnidadInicialAdmin{NodoRef: "11111111-2222-4333-8444-555555555555", OrganizacionRef: "organizacion:sintetica:dipgra", UnidadRef: "unidad:sintetica:administracion", Clase: "centro", Denominacion: "Unidad sintética de administración", CatalogoRef: "estructura-organizativa-dipgra", CatalogoVersion: 1, CatalogoRevision: 1, CatalogoEntradaClave: "unidad_admin_sintetica", VigenteDesde: "2026-10-04", VigenteHasta: "2026-10-06"}}
}
func planUnidadFixture(t *testing.T) (PlanUnidadInicialAdminV1, FuenteUnidadInicialAdminV1) {
	t.Helper()
	f := fuenteUnidadFixture()
	_, sha, e := f.CanonicoYHuella()
	if e != nil {
		t.Fatal(e)
	}
	p := PlanUnidadInicialAdminV1{Version: 1, OperacionRef: "pui_" + strings.Repeat("a", 22), PreparadoEn: time.Date(2026, 10, 4, 8, 0, 0, 0, time.UTC), CaducaEn: time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC), Entorno: f.Entorno, AlcanceFuente: f.AlcanceFuente, Unidad: f.Unidad, ActoTecnicoRef: f.ActoTecnicoRef, Fuente: EvidenciaUnidadInicialAdmin{f.Referencia, 1, sha}}
	return p, f
}
func TestPlanUnidadCanonYFuenteLigada(t *testing.T) {
	p, f := planUnidadFixture(t)
	if p.ValidarConFuente(f) != nil {
		t.Fatal("fuente no ligada")
	}
	b, h, e := p.CanonicoYHuella()
	if e != nil {
		t.Fatal(e)
	}
	b2, h2, e := p.CanonicoYHuella()
	if e != nil || string(b) != string(b2) || h != h2 {
		t.Fatal("canon no estable")
	}
	f.Unidad.Denominacion = "Otra unidad sintética"
	if p.ValidarConFuente(f) == nil {
		t.Fatal("fuente sustituida aceptada")
	}
	p.Fuente.HuellaSHA256 = strings.Repeat("b", 64)
	if p.ValidarConFuente(f) == nil {
		t.Fatal("huella falsa aceptada")
	}
}
func TestPlanUnidadRechazos(t *testing.T) {
	casos := map[string]func(*PlanUnidadInicialAdminV1){
		"entorno":                    func(p *PlanUnidadInicialAdminV1) { p.Entorno = "produccion" },
		"alcance":                    func(p *PlanUnidadInicialAdminV1) { p.AlcanceFuente = "institucional" },
		"acto_fuera_namespace":       func(p *PlanUnidadInicialAdminV1) { p.ActoTecnicoRef = "acto:resolucion" },
		"op_ajena":                   func(p *PlanUnidadInicialAdminV1) { p.OperacionRef = "pfi_" + strings.Repeat("a", 22) },
		"op_larga":                   func(p *PlanUnidadInicialAdminV1) { p.OperacionRef = "pui_" + strings.Repeat("a", 125) },
		"version_plan":               func(p *PlanUnidadInicialAdminV1) { p.Version = 2 },
		"fuente_version":             func(p *PlanUnidadInicialAdminV1) { p.Fuente.Version = 2 },
		"fuente_sha":                 func(p *PlanUnidadInicialAdminV1) { p.Fuente.HuellaSHA256 = "no" },
		"nodo_mayusculas":            func(p *PlanUnidadInicialAdminV1) { p.Unidad.NodoRef = "AAAAAAAA-2222-4333-8444-555555555555" },
		"ref_unicode":                func(p *PlanUnidadInicialAdminV1) { p.Unidad.OrganizacionRef = "organización:sintetica" },
		"catalogo_ajeno":             func(p *PlanUnidadInicialAdminV1) { p.Unidad.CatalogoRef = "otro_catalogo" },
		"version_catalogo":           func(p *PlanUnidadInicialAdminV1) { p.Unidad.CatalogoVersion = 2 },
		"revision_catalogo":          func(p *PlanUnidadInicialAdminV1) { p.Unidad.CatalogoRevision = 1 << 63 },
		"cas_distinto":               func(p *PlanUnidadInicialAdminV1) { p.Unidad.RevisionEsperada = 1 },
		"clase":                      func(p *PlanUnidadInicialAdminV1) { p.Unidad.Clase = "administrador" },
		"nombre_control":             func(p *PlanUnidadInicialAdminV1) { p.Unidad.Denominacion = "Unidad\nAdmin" },
		"nombre_largo":               func(p *PlanUnidadInicialAdminV1) { p.Unidad.Denominacion = strings.Repeat("é", 301) },
		"nombre_espacio":             func(p *PlanUnidadInicialAdminV1) { p.Unidad.Denominacion = " Unidad" },
		"fecha_civil_invalida":       func(p *PlanUnidadInicialAdminV1) { p.Unidad.VigenteDesde = "2026-02-30" },
		"fecha_civil_year0":          func(p *PlanUnidadInicialAdminV1) { p.Unidad.VigenteDesde = "0000-01-01" },
		"fecha_civil_formato":        func(p *PlanUnidadInicialAdminV1) { p.Unidad.VigenteDesde = "2026-1-04" },
		"fecha_civil_inversa":        func(p *PlanUnidadInicialAdminV1) { p.Unidad.VigenteHasta = p.Unidad.VigenteDesde },
		"caducidad_sin_cobertura":    func(p *PlanUnidadInicialAdminV1) { p.Unidad.VigenteHasta = "2026-10-05" },
		"inicio_despues_preparacion": func(p *PlanUnidadInicialAdminV1) { p.Unidad.VigenteDesde = "2026-10-05" },
		"fecha_fraccion":             func(p *PlanUnidadInicialAdminV1) { p.PreparadoEn = p.PreparadoEn.Add(time.Nanosecond) },
		"fecha_zona":                 func(p *PlanUnidadInicialAdminV1) { p.PreparadoEn = p.PreparadoEn.In(time.FixedZone("offset", 3600)) },
		"fecha_overflow":             func(p *PlanUnidadInicialAdminV1) { p.CaducaEn = time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC) },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			p, _ := planUnidadFixture(t)
			cambiar(&p)
			if p.Validar() == nil {
				t.Fatal("acepta invalido")
			}
			if b, h, e := p.CanonicoYHuella(); e == nil || b != nil || h != "" {
				t.Fatal("canon de invalido")
			}
		})
	}
}
func TestPlanUnidadVigenciaYCadenaFuente(t *testing.T) {
	p, f := planUnidadFixture(t)
	if p.ValidarEn(p.PreparadoEn) != nil || p.ValidarEn(p.CaducaEn.Add(-time.Second)) != nil {
		t.Fatal("ventana válida denegada")
	}
	for _, now := range []time.Time{{}, p.PreparadoEn.Add(-time.Second), p.CaducaEn} {
		if p.ValidarEn(now) == nil {
			t.Fatal("ventana inválida aceptada")
		}
	}
	f.ActoTecnicoRef = "acto_tecnico:otro"
	if p.ValidarConFuente(f) == nil {
		t.Fatal("acto cruzado")
	}
	f = fuenteUnidadFixture()
	f.Version = 2
	if f.Validar() == nil {
		t.Fatal("versión fuente abierta")
	}
	p.Unidad.Denominacion = strings.Repeat("é", 300)
	if p.Validar() != nil {
		t.Fatal("cuenta bytes en vez de runas")
	}
}
