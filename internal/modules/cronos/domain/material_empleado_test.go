package domain

import (
	"strings"
	"testing"
	"time"
)

func TestMaterialConsultaSaldoPropioCerrado(t *testing.T) {
	base := MaterialConsultaSaldoPropio{ActorRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl", EmpleadoRef: "emp_0123456789abcdefghijkl", Desde: "2026-01-01", Hasta: "2026-12-31", ZonaHoraria: ZonaSaldoPeninsula}
	b, err := base.Canonico()
	if err != nil || !strings.HasPrefix(string(b), `{"actor_ref":`) || !strings.Contains(string(b), `"zona_horaria":"Europe/Madrid"}`) {
		t.Fatal(string(b), err)
	}
	for _, cambiar := range []func(*MaterialConsultaSaldoPropio){
		func(m *MaterialConsultaSaldoPropio) { m.Hasta = "2027-01-03" },
		func(m *MaterialConsultaSaldoPropio) { m.Hasta = "2025-12-31" },
		func(m *MaterialConsultaSaldoPropio) { m.Desde = "2026-1-01" },
		func(m *MaterialConsultaSaldoPropio) { m.ZonaHoraria = "UTC" },
		func(m *MaterialConsultaSaldoPropio) { m.EmpleadoRef = "per_0123456789abcdefghijkl" },
	} {
		m := base
		cambiar(&m)
		if m.Validar() == nil {
			t.Fatalf("acepta %+v", m)
		}
	}
}

func TestMaterialDisponibilidadRemotaExigeOrigenRemoto(t *testing.T) {
	remoto, _ := NuevaAcreditacionCanalMarcaje(DatosAcreditacionCanalMarcaje{PoliticaVersionRef: "pol", CanalRef: "web", OrigenRef: OrigenMarcajeRemoto, CalidadRef: "mtls"})
	terminal, _ := NuevaAcreditacionCanalMarcaje(DatosAcreditacionCanalMarcaje{PoliticaVersionRef: "pol", CanalRef: "web", OrigenRef: "terminal", CalidadRef: "mtls"})
	m := MaterialDisponibilidadMarcajeRemoto{ActorRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl", EmpleadoRef: "emp_0123456789abcdefghijkl", InstanteUTC: time.Now().UTC().Truncate(time.Microsecond), Canal: remoto}
	if b, err := m.Canonico(); err != nil || !strings.Contains(string(b), `"clave_operacion":""`) {
		t.Fatal("disponibilidad sin clave rechazada", err)
	}
	m.ClaveOperacion = "no"
	if m.Validar() == nil {
		t.Fatal("clave corta aceptada")
	}
	m.ClaveOperacion, m.Canal = "", terminal
	if m.Validar() == nil {
		t.Fatal("canal no remoto aceptado")
	}
}
