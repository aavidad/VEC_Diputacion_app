package postgres

import (
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestSelectorSubidaOriginalC8ProyeccionExacta(t *testing.T) {
	selector := ports.SelectorSubidaOriginalC8{SolicitudRef: "permiso:cronos:solicitud:perm-0001", VersionEsperada: 3}
	const empleado = "emp_0123456789abcdefghijkl"
	audit := `,"auditoria":{"auditoria_ref":"auditoria:cronos:uno","huella_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","correlacion_ref":"correlacion_0123456789abcdef","registrada_en":"2026-10-14T20:00:00Z"}`
	base := `{"solicitud_ref":"permiso:cronos:solicitud:perm-0001","version":3,"persona_ref":"per_0123456789abcdefghijkl","empleado_ref":"emp_0123456789abcdefghijkl"` + audit + `}`
	v, err := decodificarActoSubidaOriginalC8([]byte(base), selector)
	if err != nil || v.EmpleadoRef != empleado {
		t.Fatalf("proyección válida denegada: %#v, %v", v, err)
	}
	for _, raw := range []string{
		`{"solicitud_ref":"permiso:cronos:solicitud:otra-0001","version":3,"persona_ref":"per_0123456789abcdefghijkl","empleado_ref":"emp_0123456789abcdefghijkl"` + audit + `}`,
		`{"solicitud_ref":"permiso:cronos:solicitud:perm-0001","version":4,"persona_ref":"per_0123456789abcdefghijkl","empleado_ref":"emp_0123456789abcdefghijkl"` + audit + `}`,
		`{"solicitud_ref":"permiso:cronos:solicitud:perm-0001","version":3,"persona_ref":"per_0123456789abcdefghijkl","empleado_ref":"emp_0123456789abcdefghijkl","extra":true` + audit + `}`,
		`{"solicitud_ref":"permiso:cronos:solicitud:perm-0001","version":3,"persona_ref":"per_0123456789abcdefghijkl","empleado_ref":"emp_0123456789abcdefghijkl"}`,
	} {
		if _, err := decodificarActoSubidaOriginalC8([]byte(raw), selector); err == nil {
			t.Fatalf("proyección incompatible aceptada: %s", raw)
		}
	}
}
