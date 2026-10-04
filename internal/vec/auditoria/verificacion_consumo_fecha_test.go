package auditoria

import (
	"encoding/json"
	"strings"
	"testing"
)

// Vector calculado fuera de Go con hashlib.sha256 de Python, encuadrando los
// 15 componentes como len(valor.encode("utf8")) + ":" + valor + "\n".
// La decisión multibyte distingue longitud UTF-8 de número de caracteres.
func vectorConsumoFechaAD173() RegistroConsumoFechaV3 {
	return RegistroConsumoFechaV3{
		RegistroConsumoOrigenV2: RegistroConsumoOrigenV2{
			RegistroCadenaV3: RegistroCadenaV3{
				AuditoriaRef:        "aud_v3_" + strings.Repeat("b", 32),
				Secuencia:           7,
				DecisionRef:         "decision:ad173:pruéba",
				EfectoRef:           "efecto:ad173:prueba",
				HuellaEfectoSHA256:  strings.Repeat("a", 64),
				AnteriorSHA256:      strings.Repeat("0", 64),
				ConsumoHuellaSHA256: strings.Repeat("b", 64),
				HuellaSHA256:        "0a8319971a070a9684e3deee8659d2a149b274e0c80f5973155c2a3d4f74050d",
			},
			TipoRegistro:   TipoConsumoFechaV3,
			VersionConsumo: 3,
			Proceso:        "rrhh_prueba",
			Canal:          "interna_corporativa",
		},
		RegistradaEn:    "2026-10-03T12:34:56.123456Z",
		ConsumidaEn:     "2026-10-03T12:34:56.123456Z",
		ActorRef:        "per_aaaaaaaaaaaaaaaaaaaaaa",
		PerfilActivoRef: "prf_bbbbbbbbbbbbbbbbbbbbbb",
		FinalidadRef:    "gestion_personal",
	}
}

func TestCotejarConsumoFechaAD173VectorYMutaciones(t *testing.T) {
	if fallo := CotejarConsumoFechaV3(vectorConsumoFechaAD173()); fallo != nil {
		t.Fatalf("vector AD173 rechazado: %+v", fallo)
	}
	for _, tc := range []struct {
		nombre  string
		cambiar func(*RegistroConsumoFechaV3)
		codigo  string
		clave   string
	}{
		{"actor", func(r *RegistroConsumoFechaV3) { r.ActorRef = "per_cccccccccccccccccccccc" }, "huella_distinta", "huella_sha256"},
		{"perfil", func(r *RegistroConsumoFechaV3) { r.PerfilActivoRef = "prf_cccccccccccccccccccccc" }, "huella_distinta", "huella_sha256"},
		{"finalidad", func(r *RegistroConsumoFechaV3) { r.FinalidadRef = "otra_finalidad" }, "huella_distinta", "huella_sha256"},
		{"fechas", func(r *RegistroConsumoFechaV3) {
			r.RegistradaEn, r.ConsumidaEn = "2026-10-03T12:34:56.123457Z", "2026-10-03T12:34:56.123457Z"
		}, "huella_distinta", "huella_sha256"},
		{"registrada", func(r *RegistroConsumoFechaV3) { r.RegistradaEn = "2026-10-03T12:34:56.123457Z" }, "instante_distinto", "registrada_consumida_en"},
		{"consumida", func(r *RegistroConsumoFechaV3) { r.ConsumidaEn = "2026-10-03T12:34:56.123457Z" }, "instante_distinto", "registrada_consumida_en"},
		{"proceso", func(r *RegistroConsumoFechaV3) { r.Proceso = "otro_proceso" }, "huella_distinta", "huella_sha256"},
		{"canal", func(r *RegistroConsumoFechaV3) { r.Canal = "administracion_privilegiada" }, "huella_distinta", "huella_sha256"},
		{"secuencia", func(r *RegistroConsumoFechaV3) { r.Secuencia++ }, "huella_distinta", "huella_sha256"},
		{"anterior", func(r *RegistroConsumoFechaV3) { r.AnteriorSHA256 = strings.Repeat("c", 64) }, "huella_distinta", "huella_sha256"},
		{"decision", func(r *RegistroConsumoFechaV3) { r.DecisionRef = "decision:ad173:otra" }, "huella_distinta", "huella_sha256"},
		{"efecto", func(r *RegistroConsumoFechaV3) { r.EfectoRef = "efecto:ad173:otro" }, "huella_distinta", "huella_sha256"},
		{"huella_efecto", func(r *RegistroConsumoFechaV3) { r.HuellaEfectoSHA256 = strings.Repeat("c", 64) }, "huella_distinta", "huella_sha256"},
		{"consumo", func(r *RegistroConsumoFechaV3) {
			r.ConsumoHuellaSHA256, r.AuditoriaRef = strings.Repeat("c", 64), "aud_v3_"+strings.Repeat("c", 32)
		}, "huella_distinta", "huella_sha256"},
		{"version", func(r *RegistroConsumoFechaV3) { r.VersionConsumo = 2 }, "tipo_invalido", "tipo_version_consumo"},
		{"tipo", func(r *RegistroConsumoFechaV3) { r.TipoRegistro = TipoConsumoOrigenV2 }, "tipo_invalido", "tipo_version_consumo"},
		{"canal_libre", func(r *RegistroConsumoFechaV3) { r.Canal = "http_libre" }, "registro_invalido", "canal"},
		{"referencia", func(r *RegistroConsumoFechaV3) { r.AuditoriaRef = "aud_v3_" + strings.Repeat("c", 32) }, "referencia_distinta", "auditoria_ref"},
		{"cero", func(r *RegistroConsumoFechaV3) { r.Secuencia = 0 }, "registro_invalido", "coordenadas_origen"},
		{"secuencia_excesiva", func(r *RegistroConsumoFechaV3) { r.Secuencia = maxSecuenciaVerificacion + 1 }, "registro_invalido", "coordenadas_origen"},
		{"huella_corta", func(r *RegistroConsumoFechaV3) { r.ConsumoHuellaSHA256 = "b" }, "registro_invalido", "coordenadas_origen"},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			r := vectorConsumoFechaAD173()
			tc.cambiar(&r)
			fallo := CotejarConsumoFechaV3(r)
			if fallo == nil || fallo.Codigo != tc.codigo || fallo.Clave != tc.clave || fallo.Secuencia != r.Secuencia {
				t.Fatalf("alteración no detectada con diagnóstico esperado: %+v", fallo)
			}
		})
	}
}

func TestCotejarConsumoFechaAD173FechasCanonicas(t *testing.T) {
	for _, fecha := range []string{"", "no_fecha", "0000-01-01T00:00:00.000000Z", "2026-02-30T12:34:56.123456Z",
		"2026-10-03T12:34:56Z", "2026-10-03T12:34:56.12345Z", "2026-10-03T12:34:56.1234560Z",
		"2026-10-03T12:34:56.1234561Z", "2026-10-03T12:34:56,123456Z", "2026-10-03T12:34:56.123456+00:00",
		"2026-10-03T14:34:56.123456+02:00", "2026-10-03T12:34:56.123456z", strings.Repeat("9", 4096)} {
		for _, clave := range []string{"registrada_en", "consumida_en"} {
			t.Run(clave+"/"+fecha[:min(len(fecha), 40)], func(t *testing.T) {
				r := vectorConsumoFechaAD173()
				if clave == "registrada_en" {
					r.RegistradaEn = fecha
				} else {
					r.ConsumidaEn = fecha
				}
				r.HuellaSHA256 = huellaConsumoFechaV3(r)
				fallo := CotejarConsumoFechaV3(r)
				if fallo == nil || fallo.Codigo != "instante_invalido" || fallo.Clave != clave {
					t.Fatalf("fecha no canónica admitida: %+v", fallo)
				}
			})
		}
	}
	// Una huella recalculada no permite dos instantes distintos en AD173.
	r := vectorConsumoFechaAD173()
	r.ConsumidaEn = "2026-10-03T12:34:56.123457Z"
	r.HuellaSHA256 = huellaConsumoFechaV3(r)
	if fallo := CotejarConsumoFechaV3(r); fallo == nil || fallo.Codigo != "instante_distinto" {
		t.Fatalf("fechas distintas admitidas: %+v", fallo)
	}
}

func TestCotejarConsumoFechaAD173ReferenciasYMinimizacion(t *testing.T) {
	for _, clave := range []string{"actor_ref", "perfil_activo_ref", "finalidad_ref"} {
		for _, valor := range []string{"", "referencia con blancos", "referencia\nprivada", "referencia:*", "referencia:?", "referencia:\xff", strings.Repeat("a", 513)} {
			t.Run(clave+"/"+valor[:min(len(valor), 30)], func(t *testing.T) {
				r := vectorConsumoFechaAD173()
				switch clave {
				case "actor_ref":
					r.ActorRef = valor
				case "perfil_activo_ref":
					r.PerfilActivoRef = valor
				case "finalidad_ref":
					r.FinalidadRef = valor
				}
				r.HuellaSHA256 = huellaConsumoFechaV3(r)
				fallo := CotejarConsumoFechaV3(r)
				if fallo == nil || fallo.Codigo != "registro_invalido" || fallo.Clave != clave {
					t.Fatalf("referencia inválida admitida: %+v", fallo)
				}
				if fallo.Esperado != "asiento_ad173_valido" || fallo.Obtenido != "incompatible" {
					t.Fatalf("diagnóstico expone datos de entrada: %+v", fallo)
				}
			})
		}
	}
	// La proyección mantiene los nombres JSON que consumirá el parser mixto.
	material, err := json.Marshal(vectorConsumoFechaAD173())
	if err != nil {
		t.Fatal(err)
	}
	var recuperado RegistroConsumoFechaV3
	if err := json.Unmarshal(material, &recuperado); err != nil || recuperado != vectorConsumoFechaAD173() {
		t.Fatalf("proyección no recuperable: %v", err)
	}
}
