package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func fichaPreparacionRPT() FichaEmpleadoB2 {
	return FichaEmpleadoB2{EmpleadoRef: "emp_" + strings.Repeat("e", 24), PersonaRef: "per_" + strings.Repeat("p", 24),
		OrganismoRef: "organismo:sintetico", Version: 7,
		Corte: CorteEmpleadoB2{VigenteEn: "2026-10-01", ConocidoEn: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)}}
}
func relacionPreparacionRPT(letra string, version int64, registrada time.Time, estado string) RelacionRegistroEmpleadoB2 {
	return RelacionRegistroEmpleadoB2{RelacionRef: "rel_" + strings.Repeat(letra, 24), OrganismoRef: "organismo:sintetico", Estado: estado,
		Traza: TrazaEmpleadoB2{Desde: "2024-01-01", RegistradaEn: registrada, Version: version, ActoRef: "acto:sintetico", FuenteRef: "fuente:sintetica", FuenteVersion: 2}}
}
func TestPreparacionRPTUltimaRevisionNoResucitaRelacion(t *testing.T) {
	f := fichaPreparacionRPT()
	anterior := relacionPreparacionRPT("a", 1, f.Corte.ConocidoEn.Add(-time.Hour), "vigente")
	ultima := relacionPreparacionRPT("a", 2, f.Corte.ConocidoEn, "finalizada")
	ultima.Traza.Hasta = f.Corte.VigenteEn // límite superior excluido.
	otra := relacionPreparacionRPT("b", 1, f.Corte.ConocidoEn, "suspendida")
	f.Relaciones = []RelacionRegistroEmpleadoB2{ultima, otra, anterior}
	p, err := PrepararRelacionParaRPT(f)
	if err != nil || len(p.Relaciones) != 2 || p.Relaciones[0].Estado != "finalizada" || p.Relaciones[0].EnIntervalo || !p.Relaciones[1].EnIntervalo {
		t.Fatalf("selección temporal incorrecta: %+v, %v", p, err)
	}
	if p.Uso != "preparacion" || p.Cobertura != "no_acreditada" || p.EstadoRPT != "pendiente_fuente_rpt" || p.VersionFicha != f.Version || p.EmpleadoRef != f.EmpleadoRef {
		t.Fatal("preparación sin límites de fuente o vínculo de ficha")
	}
	// Igual instante: vence la revisión mayor, incluso fuera de intervalo.
	anterior.Traza.RegistradaEn = ultima.Traza.RegistradaEn
	ultima.Traza.Desde, ultima.Traza.Hasta = "2026-10-02", ""
	f.Relaciones = []RelacionRegistroEmpleadoB2{anterior, ultima}
	p, err = PrepararRelacionParaRPT(f)
	if err != nil || p.Relaciones[0].Traza.Version != 2 || p.Relaciones[0].EnIntervalo {
		t.Fatal("se recuperó una revisión obsoleta por su vigencia")
	}
}
func TestPreparacionRPTDeniegaFuenteIncoherente(t *testing.T) {
	casos := map[string]func(*FichaEmpleadoB2){
		"firma":              func(f *FichaEmpleadoB2) { f.FirmaOficial = true },
		"eficacia":           func(f *FichaEmpleadoB2) { f.EficaciaAdministrativa = true },
		"futuro":             func(f *FichaEmpleadoB2) { f.Relaciones[0].Traza.RegistradaEn = f.Corte.ConocidoEn.Add(time.Second) },
		"otro_organismo":     func(f *FichaEmpleadoB2) { f.Relaciones[0].OrganismoRef = "organismo:otro" },
		"estado":             func(f *FichaEmpleadoB2) { f.Relaciones[0].Estado = "revocada" },
		"sin_fuente":         func(f *FichaEmpleadoB2) { f.Relaciones[0].Traza.FuenteRef = "" },
		"sin_version":        func(f *FichaEmpleadoB2) { f.Relaciones[0].Traza.Version = 0 },
		"revision_duplicada": func(f *FichaEmpleadoB2) { f.Relaciones = append(f.Relaciones, f.Relaciones[0]) },
		"fecha":              func(f *FichaEmpleadoB2) { f.Corte.VigenteEn = "2026-02-30" },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			f := fichaPreparacionRPT()
			f.Relaciones = []RelacionRegistroEmpleadoB2{relacionPreparacionRPT("a", 1, f.Corte.ConocidoEn, "vigente")}
			mutar(&f)
			p, err := PrepararRelacionParaRPT(f)
			if err == nil || p.EmpleadoRef != "" || p.Relaciones != nil {
				t.Fatal("se expuso preparación parcial inválida")
			}
		})
	}
}
func TestPreparacionRPTMinimizaYConservaHistoriaOriginal(t *testing.T) {
	f := fichaPreparacionRPT()
	r := relacionPreparacionRPT("a", 1, f.Corte.ConocidoEn, "vigente")
	r.UnidadDenominacion = "dato nominal excluido"
	f.Relaciones = []RelacionRegistroEmpleadoB2{r}
	f.Ocupaciones = []OcupacionEmpleadoB2{{PuestoRef: "puesto:excluido", PlazaRef: "plaza:excluida"}}
	p, err := PrepararRelacionParaRPT(f)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(p)
	for _, excluido := range []string{"persona_ref", "organismo_ref", "puesto_ref", "plaza_ref", "unidad_denominacion", "dato nominal excluido"} {
		if strings.Contains(string(b), excluido) {
			t.Fatalf("dato no necesario: %s", excluido)
		}
	}
	p.Relaciones[0].Traza.FuenteRef = "fuente:otra"
	if f.Relaciones[0].Traza.FuenteRef != r.Traza.FuenteRef {
		t.Fatal("se alteró historia de B2")
	}
	f.Relaciones = nil
	p, err = PrepararRelacionParaRPT(f)
	if err != nil || p.Relaciones == nil || p.Cobertura != "no_acreditada" {
		t.Fatal("vacío presentado como cobertura completa")
	}
}
