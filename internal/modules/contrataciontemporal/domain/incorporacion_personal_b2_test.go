package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func planPersonalB2Prueba() PlanIncorporacionPersonalB2 {
	h := strings.Repeat("a", 64)
	fuente := FuentePlanPersonalB2{"fuente:original", 1, h}
	return PlanIncorporacionPersonalB2{OrganizacionRef: "org:uno", UnidadCTRef: "unidad:rrhh", ExpedienteRef: "expediente:uno", VersionExpediente: 8, AnalisisVersion: 2, AnalisisReciboRef: "recibo:analisis", AnalisisSHA256: h, PropuestaReciboRef: "recibo:propuesta", AceptacionRef: "aceptacion:uno", AceptacionReciboRef: "recibo:aceptacion", Bolsa: SelectorBolsaPlanB2{UnidadRef: "unidad:uno", CategoriaRef: "categoria:uno", NecesidadRef: "necesidad:uno", AceptacionOperacionRef: "operacion:aceptacion", AceptacionRegistroSHA256: h, AperturaOperacionRef: "operacion:apertura", AperturaRegistroSHA256: h, LlamamientoRef: "llamamiento:uno", PropuestaRef: "propuesta:uno"}, PersonaRef: "per_aaaaaaaaaaaaaaaaaaaaaaaa", PersonaVersion: 1, PersonaFuente: fuente, PersonaReciboBolsaRef: "recibo:bolsa", OrganismoRef: "organismo:uno", UnidadRef: "unidad:uno", FuenteOrganizacion: FuenteSinVersionPlanB2{fuente.Ref, fuente.SHA256}, FuentePlantilla: InstrumentoPlanPersonalB2{"11111111-1111-4111-8111-111111111111", 1, fuente.Ref, fuente.SHA256}, FuenteRPT: InstrumentoPlanPersonalB2{"22222222-2222-4222-8222-222222222222", 1, fuente.Ref, fuente.SHA256}, VersionPlazaRef: "plantilla:uno", VersionPuestoRef: "rpt:uno", RevisionPlaza: 1, RevisionPuesto: 1, PuestoRef: "puesto:uno", PlazaRef: "plaza:uno", CatalogoRPTID: "catalogo:rpt", CatalogoRPTModulo: "personal", CatalogoRPTVersion: 1, CatalogoRPTSHA256: h, CategoriaRef: "categoria:uno", VinculoRevision: 1, VinculoReciboRef: "recibo:vinculo", Regimen: EntradaPlanPersonalB2{"regimen:uno", 1}, Modalidad: EntradaPlanPersonalB2{"modalidad:uno", 1}, ClaseOcupacion: "temporal", Desde: "2026-09-30", Hasta: "", MotivoClave: "incorporacion", DocumentoRef: "documento:uno", DocumentoSHA256: h, EjercicioSintetico: true}
}
func TestPlanPersonalB2NoConvierteFechasNiFuentes(t *testing.T) {
	p := planPersonalB2Prueba()
	if e := p.Validar(); e != nil {
		t.Fatal(e)
	}
	casos := []struct {
		nombre  string
		cambiar func(*PlanIncorporacionPersonalB2)
	}{
		{"fin exclusivo anterior", func(p *PlanIncorporacionPersonalB2) { p.Hasta = p.Desde }},
		{"fecha civil imposible", func(p *PlanIncorporacionPersonalB2) { p.Desde = "2026-02-30" }},
		{"instante no fecha civil", func(p *PlanIncorporacionPersonalB2) { p.Desde = "2026-09-30T00:00:00Z" }},
		{"categoría de otra aceptación", func(p *PlanIncorporacionPersonalB2) { p.CategoriaRef = "categoria:otra" }},
		{"análisis futuro", func(p *PlanIncorporacionPersonalB2) { p.AnalisisVersion = 9 }},
		{"apertura sin huella", func(p *PlanIncorporacionPersonalB2) { p.Bolsa.AperturaRegistroSHA256 = "" }},
		{"persona sin procedencia", func(p *PlanIncorporacionPersonalB2) { p.PersonaFuente.SHA256 = strings.Repeat("0", 64) }},
		{"clase no acreditada", func(p *PlanIncorporacionPersonalB2) { p.ClaseOcupacion = "" }},
		{"año cero", func(p *PlanIncorporacionPersonalB2) { p.Desde = "0000-09-30" }},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := planPersonalB2Prueba()
			c.cambiar(&p)
			if p.Validar() == nil {
				t.Fatal("plan contradictorio admitido")
			}
		})
	}
}
func TestPlanPersonalB2ClaseConGramaticaPersonal(t *testing.T) {
	p := planPersonalB2Prueba()
	p.ClaseOcupacion = "temporal_especial_2"
	if e := p.Validar(); e != nil {
		t.Fatalf("clave canónica rechazada: %v", e)
	}
	for _, clase := range []string{"", "Reserva", "temporal-especial", "a" + strings.Repeat("b", 64)} {
		p.ClaseOcupacion = clase
		if p.Validar() == nil {
			t.Fatalf("clave fuera de gramática admitida: %q", clase)
		}
	}
}
func TestCanonicoPlanPersonalB2PreservaEnteroYOrdenRecursivo(t *testing.T) {
	m := map[string]any{"z": uint64(9007199254740991), "a": map[string]any{"z": "<&>", "a": "fecha"}}
	b, e := CanonicoPlanPersonalB2(m)
	if e != nil {
		t.Fatal(e)
	}
	if string(b) != `{"a":{"a":"fecha","z":"\u003c\u0026\u003e"},"z":9007199254740991}` {
		t.Fatalf("canon alterado: %s", b)
	}
	// Semántica idéntica no depende del orden de entrada del mapa.
	var raw map[string]any
	dec := json.NewDecoder(strings.NewReader(`{"z":9007199254740991,"a":{"z":"<&>","a":"fecha"}}`))
	dec.UseNumber()
	if e := dec.Decode(&raw); e != nil {
		t.Fatal(e)
	}
	h1, _ := SHA256PlanPersonalB2(m)
	h2, _ := SHA256PlanPersonalB2(raw)
	if h1 != h2 {
		t.Fatal("orden convierte misma intención en otra operación")
	}
	p := planPersonalB2Prueba()
	h1, _ = SHA256PlanPersonalB2(p)
	p.Bolsa.AperturaOperacionRef = "operacion:otra"
	h2, _ = SHA256PlanPersonalB2(p)
	if h1 == h2 {
		t.Fatal("sustitución de apertura no cambia el compromiso")
	}
}
