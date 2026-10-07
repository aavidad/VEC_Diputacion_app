package politicacopias

import (
	"testing"
	"time"
)

func policy() Politica {
	return Politica{Formato: 1, Referencia: "politica:ensayo", Destino: "destino:ensayo", ZonaHoraria: "Europe/Madrid", FechaInicial: "2026-01-01", CadaDias: 1, Ventana: Ventana{"02:30", "04:00"}, Retencion: Retencion{ConservarMinimo: 1, EdadMaximaDias: 30}}
}
func instant(s string) time.Time {
	t, e := time.Parse(time.RFC3339, s)
	if e != nil {
		panic(e)
	}
	return t
}
func TestAgendaDSTOneEventPerCivilDate(t *testing.T) {
	p := policy()
	for _, tc := range []struct{ from, want string }{{"2026-03-29T00:00:00Z", "2026-03-30T00:30:00Z"}, {"2026-10-25T00:00:00Z", "2026-10-25T00:30:00Z"}, {"2026-10-25T00:45:00Z", "2026-10-26T01:30:00Z"}} {
		got, e := p.Proximos(instant(tc.from), 2)
		if e != nil || got[0].Fecha != instant(tc.want) || got[0].FechaCivil == got[1].FechaCivil {
			t.Fatalf("%s: %v %v", tc.from, got, e)
		}
	}
	p.Ventana.Fin = "02:45"
	got, e := p.Proximos(instant("2026-10-25T00:00:00Z"), 1)
	if e != nil || !got[0].FinVentana.Equal(instant("2026-10-25T01:45:00Z")) {
		t.Fatalf("fold end: %v %v", got, e)
	}
}
func TestAgendaFrequencyAndValidation(t *testing.T) {
	p := policy()
	p.FechaInicial = "2026-12-30"
	p.CadaDias = 2
	p.DiasSemana = []int{5}
	events, e := p.Proximos(instant("2026-12-29T00:00:00Z"), 2)
	if e != nil || events[0].FechaCivil != "2027-01-01" || events[1].FechaCivil != "2027-01-15" {
		t.Fatalf("%v %v", events, e)
	}
	p.DiasSemana = []int{1, 1}
	if p.Validar() == nil {
		t.Fatal("duplicate day accepted")
	}
	p.DiasSemana = nil
	p.ZonaHoraria = "Local"
	if p.Validar() == nil {
		t.Fatal("machine dependent time zone accepted")
	}
}
func TestCanonicalPolicyPrivateAndDefaultDoubleControl(t *testing.T) {
	p := policy()
	p.Retencion.Protegidas = []string{"copia:b", "copia:a"}
	p.DiasSemana = []int{5, 1}
	q := p.Normalizar()
	if !q.Retencion.ExigeDobleControl() {
		t.Fatal("default must require two people")
	}
	q.DiasSemana[0] = 4
	q.Retencion.Protegidas[0] = "copia:c"
	*q.Retencion.DobleControl = false
	if p.DiasSemana[0] != 5 || p.Retencion.Protegidas[0] != "copia:b" || !p.Retencion.ExigeDobleControl() {
		t.Fatal("input mutated")
	}
	r := policy()
	r.Retencion.Protegidas = []string{"copia:a", "copia:b"}
	r.DiasSemana = []int{1, 5}
	if r.SHA256() != p.SHA256() {
		t.Fatal("list order changed policy seal")
	}
}
func TestRetentionPreservesSafetyAndSimultaneousMinimum(t *testing.T) {
	p := policy()
	p.Retencion.BorradoPermitido = true
	p.Retencion.ConservarMinimo = 2
	p.Retencion.Protegidas = []string{"copia:protected"}
	old := instant("2025-01-01T00:00:00Z")
	now := instant("2026-10-01T00:00:00Z")
	copies := []Copia{{Referencia: "copia:latest", Destino: p.Destino, Fecha: now, Verificada: true}, {Referencia: "copia:second", Destino: p.Destino, Fecha: old.Add(24 * time.Hour), Verificada: true}}
	for _, r := range []string{"copia:old", "copia:protected", "copia:previous", "copia:pending", "copia:dependency", "copia:unverified", "copia:foreign"} {
		copies = append(copies, Copia{Referencia: r, Destino: p.Destino, Fecha: old, Verificada: true})
	}
	copies[4].PreviaActiva = true
	copies[5].PendienteConciliacion = true
	copies[6].DependenciaNecesaria = true
	copies[7].Verificada = false
	copies[8].Destino = "destino:otro"
	plan, e := p.PlanificarRetencion(copies, now)
	if e != nil {
		t.Fatal(e)
	}
	n := 0
	for _, c := range plan.Decisiones {
		if c.Candidata {
			n++
			if c.Referencia != "copia:old" {
				t.Fatalf("unsafe candidate %v", c)
			}
		}
	}
	if n != 1 || !plan.DobleControl || !plan.RequiereAutorizacionActual {
		t.Fatalf("%+v", plan)
	}
	single, e := p.PlanificarRetencion(copies[2:3], now)
	if e != nil || single.Decisiones[0].Candidata {
		t.Fatal("only verified copy removable")
	}
	p.Retencion.BorradoPermitido = false
	plan, e = p.PlanificarRetencion(copies, now)
	if e != nil {
		t.Fatal(e)
	}
	for _, x := range plan.Decisiones {
		if x.Candidata {
			t.Fatal("deletion disabled")
		}
	}
	copies[1].Referencia = copies[0].Referencia
	if _, e = p.PlanificarRetencion(copies, now); e == nil {
		t.Fatal("ambiguous catalogue accepted")
	}
}
