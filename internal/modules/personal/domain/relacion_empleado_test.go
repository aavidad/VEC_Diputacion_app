package domain

import (
	"bytes"
	"strings"
	"testing"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func TestReferenciasRelacionEmpleadoAlineadasConContextoActor(t *testing.T) {
	for _, prueba := range []struct {
		nombre, valor string
		valida        func(string) bool
		quiere        bool
	}{
		{"persona canónica", "per_AbCdef0123456789_-QRST", ReferenciaPersonaValida, true},
		{"empleado propio", "emp_AbCdef0123456789_-QRST", ReferenciaEmpleadoValida, true},
		{"relación propia", "rel_AbCdef0123456789_-QRST", ReferenciaRelacionValida, true},
		{"sufijo corto", "rel_0123456789abcdefghijk", ReferenciaRelacionValida, false},
		{"prefijo distinto", "otra_AbCdef0123456789_-QRST", ReferenciaRelacionValida, false},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			if obtenida := prueba.valida(prueba.valor); obtenida != prueba.quiere {
				t.Fatalf("valida(%q) = %v, quiere %v", prueba.valor, obtenida, prueba.quiere)
			}
		})
	}
}

func TestRelacionEmpleadoExigeSelloCompleto(t *testing.T) {
	desde, err := NuevaFechaCivil("2026-01-01")
	if err != nil {
		t.Fatal(err)
	}
	r := RelacionEmpleado{PersonaRef: "per_0123456789abcdefghijkl", EmpleadoRef: "emp_0123456789abcdefghijkl", RelacionRef: "rel_0123456789abcdefghijkl", UnidadRef: "unidad_sintetica", Estado: "activa", Desde: desde, Version: 3, ProcedenciaActoRef: "acto_sintetico", FuenteRef: "fuente_sintetica", FuenteVersion: 2}
	if err := r.Validar(); err != nil {
		t.Fatal(err)
	}
	for _, mutar := range []func(*RelacionEmpleado){
		func(x *RelacionEmpleado) { x.UnidadRef = "" },
		func(x *RelacionEmpleado) { x.Version = 0 },
		func(x *RelacionEmpleado) { x.ProcedenciaActoRef = "" },
		func(x *RelacionEmpleado) { x.FuenteRef = "" },
		func(x *RelacionEmpleado) { x.FuenteVersion = 0 },
	} {
		x := r
		mutar(&x)
		if x.Validar() == nil {
			t.Fatal("aceptó sello incompleto")
		}
	}
}

func TestMaterialConsultaPropiaEsCanonicoYNoAlias(t *testing.T) {
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{
		VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1,
		PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1,
		Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
		Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}},
	}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	fecha, _ := NuevaFechaCivil("2026-09-20")
	material, err := NuevoMaterialConsultaRelacionPropia(SolicitudConsultaRelacionPropia{FechaReferencia: fecha, Operacion: OperacionListaRelacionPropia, Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"esquema":"vec.personal.relacion-propia-dietas.v1","fecha_referencia":"2026-09-20","identidad":{"actor_ref":"per_` + z + `","contexto_actor_ref":"vca_` + z + `","contexto_version":1,"cuenta_ref":"cta_` + z + `","cuenta_version":1,"empleado_ref":"emp_` + z + `","perfil_ref":"prf_` + z + `","perfil_version":1,"persona_ref":"per_` + z + `","persona_version":1},"operacion":"lista","relacion_ref":null}`
	if !bytes.Equal(material.Canonico(), []byte(esperado)) || material.Recurso().Atributos["relacion_ref"] != "sin_seleccion" {
		t.Fatalf("canonico distinto: %s", material.Canonico())
	}
	bytesMutables := material.Canonico()
	bytesMutables[0] = '!'
	if bytes.Equal(bytesMutables, material.Canonico()) {
		t.Fatal("alias de bytes")
	}
	recurso := material.Recurso()
	recurso.Atributos["operacion"] = "otra"
	if material.Recurso().Atributos["operacion"] != "lista" {
		t.Fatal("alias de recurso")
	}
	instantanea.Vinculos[0].Referencia = "emp_" + strings.Repeat("b", 24)
	if material.Solicitud().Actor.Instantanea.Vinculos[0].Referencia != "emp_"+z {
		t.Fatal("alias de actor")
	}
}
