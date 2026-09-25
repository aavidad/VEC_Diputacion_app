package domain

import (
	"strings"
	"testing"
)

func TestMaterialesSolicitudesCanonicosYValidados(t *testing.T) {
	id := func(p string) string { return p + strings.Repeat("a", 22) }
	mov := MaterialConsultaMovimientosPropios{ActorRef: id("per_"), PerfilRef: id("prf_"), EmpleadoRef: id("emp_"), Desde: "2026-01-01", Hasta: "2026-12-31", ZonaHoraria: ZonaSaldoPeninsula}
	b, err := mov.Canonico()
	if err != nil || !strings.HasPrefix(string(b), `{"actor_ref":`) || !strings.Contains(string(b), `"zona_horaria":"Europe/Madrid"}`) {
		t.Fatal(string(b), err)
	}
	mov.Hasta = "2027-02-01"
	if mov.Validar() == nil {
		t.Fatal("periodo de más de un año")
	}
	per := MaterialConsultaPermisosPropios{ActorRef: id("per_"), PerfilRef: id("prf_"), EmpleadoRef: id("emp_"), Anio: 2026, ZonaHoraria: ZonaSaldoCanarias}
	if b, err := per.Canonico(); err != nil || !strings.Contains(string(b), `"anio":"2026"`) {
		t.Fatal(string(b), err)
	}
	sol := MaterialSolicitudPermisoPropio{ActorRef: id("per_"), PerfilRef: id("prf_"), EmpleadoRef: id("emp_"), ClaveOperacion: "perm-clave-0001",
		PermisoRef: "permiso:cronos:horas-medico", Desde: "2026-10-05", Hasta: "2026-10-05", HoraInicio: "09:00", HoraFin: "11:30", ZonaHoraria: ZonaSaldoPeninsula}
	if b, err := sol.Canonico(); err != nil || !strings.Contains(string(b), `"hora_fin":"11:30"`) {
		t.Fatal(string(b), err)
	}
	for _, mal := range []func(*MaterialSolicitudPermisoPropio){
		func(m *MaterialSolicitudPermisoPropio) { m.Hasta = "2026-10-06" },
		func(m *MaterialSolicitudPermisoPropio) { m.HoraFin = "" },
		func(m *MaterialSolicitudPermisoPropio) { m.HoraFin = "24:00" },
		func(m *MaterialSolicitudPermisoPropio) { m.PermisoRef = "permiso:otro:x" },
		func(m *MaterialSolicitudPermisoPropio) { m.HoraInicio, m.HoraFin, m.Hasta = "", "", "2027-01-02" },
		func(m *MaterialSolicitudPermisoPropio) { m.EmpleadoRef = "emp_x" },
	} {
		m := sol
		mal(&m)
		if m.Validar() == nil {
			t.Fatalf("material inválido aceptado: %+v", m)
		}
	}
	if SolicitudPermisoPropioRef("perm-clave-0001") != "permiso:cronos:solicitud:perm-clave-0001" {
		t.Fatal("referencia de solicitud distinta de la durable")
	}
}
