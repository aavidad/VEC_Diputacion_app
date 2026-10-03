package auditoria

import (
	"strings"
	"testing"
)

// Vector calculado con el encuadre SQL de AD172, independiente del verificador.
func vectorConsumoOrigenAD172() RegistroConsumoOrigenV2 {
	return RegistroConsumoOrigenV2{
		RegistroCadenaV3: RegistroCadenaV3{
			AuditoriaRef:        "aud_v3_" + strings.Repeat("b", 32),
			Secuencia:           7,
			DecisionRef:         "decision:ad172:prueba",
			EfectoRef:           "efecto:ad172:prueba",
			HuellaEfectoSHA256:  strings.Repeat("a", 64),
			AnteriorSHA256:      strings.Repeat("0", 64),
			ConsumoHuellaSHA256: strings.Repeat("b", 64),
			HuellaSHA256:        "82d759097556f17a8c4549006433ba250c388bcdc7d1cc9a2a76c43f454fcec9",
		},
		TipoRegistro:   TipoConsumoOrigenV2,
		VersionConsumo: 2,
		Proceso:        "rrhh_prueba",
		Canal:          "interna_corporativa",
	}
}

func TestCotejarConsumoOrigenAD172(t *testing.T) {
	if fallo := CotejarConsumoOrigenV2(vectorConsumoOrigenAD172()); fallo != nil {
		t.Fatalf("vector AD172 rechazado: %+v", fallo)
	}
	for _, tc := range []struct {
		nombre  string
		cambiar func(*RegistroConsumoOrigenV2)
		codigo  string
	}{
		{"proceso", func(r *RegistroConsumoOrigenV2) { r.Proceso = "otro_proceso" }, "huella_distinta"},
		{"canal", func(r *RegistroConsumoOrigenV2) { r.Canal = "administracion_privilegiada" }, "huella_distinta"},
		{"version", func(r *RegistroConsumoOrigenV2) { r.VersionConsumo = 1 }, "tipo_invalido"},
		{"tipo", func(r *RegistroConsumoOrigenV2) { r.TipoRegistro = "consumo_confirmado" }, "tipo_invalido"},
		{"canal_libre", func(r *RegistroConsumoOrigenV2) { r.Canal = "http_libre" }, "registro_invalido"},
		{"referencia", func(r *RegistroConsumoOrigenV2) { r.AuditoriaRef = "aud_v3_" + strings.Repeat("c", 32) }, "referencia_distinta"},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			r := vectorConsumoOrigenAD172()
			tc.cambiar(&r)
			if fallo := CotejarConsumoOrigenV2(r); fallo == nil || fallo.Codigo != tc.codigo {
				t.Fatalf("alteración no detectada: %+v", fallo)
			}
		})
	}
}
