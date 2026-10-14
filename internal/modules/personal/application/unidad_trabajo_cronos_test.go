package application

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func materialUnidadCronosPrueba(t *testing.T) (domain.MaterialUnidadTrabajoCronos, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
	t.Helper()
	s := solicitudFichaPropiaPrueba(t, "pep_")
	m, err := domain.NuevoMaterialUnidadTrabajoCronos(s.Actor, s.Corte.VigenteEn, s.Corte.ConocidoEn)
	if err != nil {
		t.Fatal(err)
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	actor := m.Actor()
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_unidad", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_unidad", strings.Repeat("c", 64), ports.AccionUnidadTrabajoPropiaCronos, m.EmpleadoRef(), h, ports.AudienciaUnidadTrabajoPropiaCronos, actor.ResueltoEn, actor.ResueltoEn.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return m, a
}

func TestUnidadTrabajoCronosValidaFuenteUnicaYCobertura(t *testing.T) {
	m, a := materialUnidadCronosPrueba(t)
	x := a.ResumenCapacidad()
	huella := strings.Repeat("d", 64)
	base := ports.ResultadoUnidadTrabajoPropiaCronos{
		Cobertura: "completa",
		Unidad:    &ports.UnidadTrabajoCronos{EmpleadoRef: m.EmpleadoRef(), RelacionRef: "rel_" + strings.Repeat("r", 24), OrganismoRef: "org_prueba", UnidadClave: "unidad_prueba", Desde: "2026-01-01", FuenteRef: "fuente:personal", FuenteVersionRef: "2", Version: 3},
		Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "unidadcronos:01234567-89ab-cdef-0123-456789abcdef", DecisionRef: x.DecisionRef(), EfectoRef: m.EmpleadoRef(), ConsumoHuellaSHA256: huella, AuditoriaRef: "aud_v3_" + huella[:32], ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)},
	}
	if !ResultadoUnidadTrabajoCronosValido(m, a, base) {
		t.Fatal("unidad propia unica rechazada")
	}
	for nombre, cambiar := range map[string]func(*ports.ResultadoUnidadTrabajoPropiaCronos){
		"organismo_ausente":     func(r *ports.ResultadoUnidadTrabajoPropiaCronos) { r.Unidad.OrganismoRef = "" },
		"fecha_fuera":           func(r *ports.ResultadoUnidadTrabajoPropiaCronos) { r.Unidad.Desde = "2026-09-26" },
		"fuente_invalida":       func(r *ports.ResultadoUnidadTrabajoPropiaCronos) { r.Unidad.FuenteVersionRef = "0" },
		"cobertura_incoherente": func(r *ports.ResultadoUnidadTrabajoPropiaCronos) { r.Cobertura = "no_acreditada" },
		"auditoria_ajena": func(r *ports.ResultadoUnidadTrabajoPropiaCronos) {
			r.Evidencia.AuditoriaRef = "aud_v3_" + strings.Repeat("a", 32)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			r := base
			u := *base.Unidad
			r.Unidad = &u
			cambiar(&r)
			if ResultadoUnidadTrabajoCronosValido(m, a, r) {
				t.Fatal("fuente divergente admitida")
			}
		})
	}
	base.Cobertura = "no_acreditada"
	base.Unidad = nil
	if !ResultadoUnidadTrabajoCronosValido(m, a, base) {
		t.Fatal("ambigüedad nominal sin unidad rechazada")
	}
}
