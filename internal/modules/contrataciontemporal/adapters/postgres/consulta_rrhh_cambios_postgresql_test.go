package postgres

import (
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDecodificarCambiosExpedienteRRHH(t *testing.T) {
	cambios, err := decodificarCambiosExpedienteRRHH([]byte(`[{"version_expediente":2,"registrada_en":"2026-09-25T09:00:00.000000Z","origen_version":"analisis_o3","operacion_ref":"op:2","ruta":"solicitud.grupo_subgrupo","valor_anterior":"C2","valor_nuevo":"C1"},{"version_expediente":3,"registrada_en":"2026-09-25T10:00:00.123456Z","origen_version":"cobertura_o4","operacion_ref":"op:3","ruta":"analisis.observaciones","valor_anterior":null,"valor_nuevo":"sha256:` + strings.Repeat("a", 64) + `"}]`))
	if err != nil || len(cambios) != 2 || *cambios[0].ValorAnterior != "C2" || cambios[1].ValorAnterior != nil || cambios[1].RegistradaEn.Nanosecond() != 123456000 {
		t.Fatalf("cambios=%+v err=%v", cambios, err)
	}
	vacio, err := decodificarCambiosExpedienteRRHH([]byte(`[]`))
	if err != nil || len(vacio) != 0 {
		t.Fatalf("vacío: %v %v", vacio, err)
	}
	for nombre, documento := range map[string]string{
		"campo desconocido": `[{"version_expediente":2,"registrada_en":"2026-09-25T09:00:00.000000Z","origen_version":"a","operacion_ref":"b","ruta":"x","valor_anterior":null,"valor_nuevo":"1","actor":"per"}]`,
		"fecha no canónica": `[{"version_expediente":2,"registrada_en":"25/09/2026","origen_version":"a","operacion_ref":"b","ruta":"x","valor_anterior":null,"valor_nuevo":"1"}]`,
		"nulo":              `null`,
		"basura final":      `[] []`,
		"vacío":             ``,
		"demasiado grande":  `[` + strings.Repeat(" ", maximoCambiosJSONRRHH) + `]`,
	} {
		if _, err := decodificarCambiosExpedienteRRHH([]byte(documento)); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
			t.Errorf("%s aceptado: %v", nombre, err)
		}
	}
}
