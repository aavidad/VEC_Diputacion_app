package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestUnidadTrabajoCronosLigaIdentidadYDosCortes(t *testing.T) {
	empleado := "emp_" + strings.Repeat("b", 24)
	actor := actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba("pep_", empleado))
	conocido := time.Date(2026, 9, 25, 9, 59, 59, 123456000, time.UTC)
	m, err := NuevoMaterialUnidadTrabajoCronos(actor, "2026-09-25", conocido)
	if err != nil {
		t.Fatal(err)
	}
	contexto, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"esquema":"vec.personal.unidad-trabajo-propia-cronos.consulta.v1","empleado_ref":"` + empleado + `","fecha":"2026-09-25","conocido_en":"2026-09-25T09:59:59.123456Z","contexto_actor":` + string(contexto) + `}`
	if string(m.Canonico()) != esperado {
		t.Fatalf("material divergente del canon SQL: %s", m.Canonico())
	}
	suma := sha256.Sum256([]byte(esperado))
	canonRecurso := `{"ambitos":{"empleado_ref":"` + empleado + `"},"atributos":{"conocido_en":"2026-09-25T09:59:59.123456Z","fecha":"2026-09-25","material_sha256":"` + hex.EncodeToString(suma[:]) + `","operacion":"unidad_trabajo_propia_cronos"}}`
	huella := sha256.Sum256([]byte(canonRecurso))
	if got, err := m.HuellaSHA256(); err != nil || got != hex.EncodeToString(huella[:]) {
		t.Fatalf("huella divergente del SQL: %s, %v", got, err)
	}
	otro, err := NuevoMaterialUnidadTrabajoCronos(actor, "2026-09-24", conocido)
	if err != nil {
		t.Fatal(err)
	}
	hOtro, _ := otro.HuellaSHA256()
	if hOtro == hex.EncodeToString(huella[:]) {
		t.Fatal("fecha de efectos no ligada")
	}
	otro, err = NuevoMaterialUnidadTrabajoCronos(actor, "2026-09-25", conocido.Add(time.Microsecond))
	if err != nil {
		t.Fatal(err)
	}
	hOtro, _ = otro.HuellaSHA256()
	if hOtro == hex.EncodeToString(huella[:]) {
		t.Fatal("corte de conocimiento no ligado")
	}
	r := m.Recurso()
	r.Ambitos["empleado_ref"] = "ajeno"
	b := m.Canonico()
	b[0] = 'x'
	if m.Recurso().Ambitos["empleado_ref"] != empleado || m.Canonico()[0] != '{' {
		t.Fatal("material mutable")
	}
}

func TestUnidadTrabajoCronosExigeEmpleadoPropioUnico(t *testing.T) {
	fecha := FechaCivil("2026-09-25")
	conocido := time.Date(2026, 9, 25, 9, 59, 59, 0, time.UTC)
	for _, caso := range []struct {
		nombre        string
		actorVinculos int
		prefijo       string
		esperado      error
	}{
		{"sin_empleado", 0, "pep_", ErrUnidadTrabajoCronosDenegada},
		{"puntero_heredado", 1, "vin_", ErrUnidadTrabajoCronosDenegada},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			v := vinculoEmpleadoPrueba(caso.prefijo, "emp_"+strings.Repeat("b", 24))
			actor := actorFichaPropiaPrueba(t)
			if caso.actorVinculos == 1 {
				actor = actorFichaPropiaPrueba(t, v)
			}
			_, err := NuevoMaterialUnidadTrabajoCronos(actor, fecha, conocido)
			if !errors.Is(err, caso.esperado) {
				t.Fatalf("error %v", err)
			}
		})
	}
}
