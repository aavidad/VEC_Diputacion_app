package domain

import (
	"bytes"
	"strings"
	"testing"
)

func fotoEquipoPrueba() FotoEquipoAutoritativa {
	return FotoEquipoAutoritativa{
		EquipoRef: "equipo:unidad:demo", UnidadRef: "unidad:demo", OrganismoRef: "organismo:demo",
		VersionRef: "equipo:version:1", ReciboFuenteRef: "recibo:personal:1",
		FuenteRef: "fuente:personal:1", Desde: "2026-01-01",
	}
}

func TestMaterialConsultaEquipoLigaTipoAmbitoYPeriodo(t *testing.T) {
	m := MaterialConsultaEquipo{
		ActorRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl",
		EmpleadoRef: "emp_0123456789abcdefghijkl", Foto: fotoEquipoPrueba(),
		Tipo: ConsultaPresenciaEquipo, Desde: "2026-10-01", Hasta: "2026-10-31", ZonaHoraria: ZonaSaldoPeninsula,
	}
	canon, err := m.Canonico()
	r, e := m.Recurso()
	if err != nil || e != nil || r.Tipo != "presencia_equipo" || r.Ambitos["equipo_ref"] != m.Foto.EquipoRef ||
		r.Atributos["material_sha256"] == "" || !bytes.Contains(canon, []byte("equipo_version_ref")) {
		t.Fatalf("material=%s recurso=%+v errores=%v/%v", canon, r, err, e)
	}
	m.Tipo = ConsultaIncidenciasEquipo
	otro, err := m.Recurso()
	if err != nil || otro.Tipo != "incidencias_equipo" || otro.Atributos["material_sha256"] == r.Atributos["material_sha256"] {
		t.Fatalf("recurso de incidencias no separado: %+v %v", otro, err)
	}
	m.Foto.ReciboFuenteRef = ""
	if _, err := m.Canonico(); err == nil {
		t.Fatal("acepta equipo sin recibo Personal")
	}
}

func TestMaterialExportacionSeparaPropioEquipoYPolitica(t *testing.T) {
	m := MaterialExportacionCronos{
		ActorRef: "per_0123456789abcdefghijkl", PerfilRef: "prf_0123456789abcdefghijkl",
		EmpleadoRef: "emp_0123456789abcdefghijkl", Tipo: InformeSaldo, Formato: InformePDF,
		Desde: "2026-10-01", Hasta: "2026-10-31", PoliticaRef: "politica:informes:1",
		PoliticaVersion: 1, PoliticaSHA256: strings.Repeat("a", 64), ClaveOperacion: "exportar_0001",
	}
	propio, err := m.Recurso()
	if err != nil || propio.Tipo != "informe_propio" || propio.Ambitos["empleado_ref"] != m.EmpleadoRef {
		t.Fatalf("informe propio=%+v %v", propio, err)
	}
	foto := fotoEquipoPrueba()
	m.Foto = &foto
	equipo, err := m.Recurso()
	if err != nil || equipo.Tipo != "informe_equipo" || equipo.Ambitos["equipo_ref"] != foto.EquipoRef ||
		equipo.Atributos["material_sha256"] == propio.Atributos["material_sha256"] {
		t.Fatalf("informe equipo=%+v %v", equipo, err)
	}
	m.PoliticaSHA256 = strings.Repeat("0", 63)
	if _, err := m.Canonico(); err == nil {
		t.Fatal("acepta política sin huella")
	}
}
