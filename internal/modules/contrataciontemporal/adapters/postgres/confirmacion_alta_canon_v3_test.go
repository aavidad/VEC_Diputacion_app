package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/catalogoalta"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func evidenciaAltaConNecesidad(t *testing.T) domain.DatosNecesidadAlta {
	t.Helper()
	catalogo, err := catalogoalta.CargarNecesidades("")
	if err != nil {
		t.Fatal(err)
	}
	evidencia, _ := evidenciaConfirmacionPostgreSQLPrueba(t)
	n, err := catalogo.SellarDatos(domain.DatosNecesidadAlta{
		Esquema: "vec.ct.necesidad_alta.v1", CatalogoRef: catalogo.Referencia,
		CatalogoVersion: catalogo.Version, CatalogoHuellaSHA256: catalogo.HuellaSHA256,
		CausaClave: evidencia.Expediente.Solicitud.MotivoClave,
		Periodo:    evidencia.Expediente.Solicitud.Periodo, JornadaMinutos: 1125,
		Campos: map[string]string{
			"numero_personas":        "2",
			"justificacion_temporal": "Refuerzo sintético para el servicio.",
			"organica_codigo":        "100", "funcional_codigo": "200",
			"proyecto_gasto_codigo": "300", "porcentaje_financiacion": "100",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCanonEfectoAltaV3ConservaOrdenV2YLigaNecesidad(t *testing.T) {
	evidencia, datos := evidenciaConfirmacionPostgreSQLPrueba(t)
	legado, _, err := canonEfectoAlta(evidencia.Expediente, evidencia.Candidatura)
	if err != nil {
		t.Fatal(err)
	}
	n := evidenciaAltaConNecesidad(t)
	evidencia.Expediente.Solicitud.Necesidad = &n
	if evidencia.Expediente.Validar() != nil {
		t.Fatal("expediente v3 inválido")
	}
	actual, huella, err := canonEfectoAlta(evidencia.Expediente, evidencia.Candidatura)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(actual, []byte(`{"esquema":"`+esquemaEfectoAltaV3+`"`)) ||
		!bytes.Contains(actual, []byte(`"catalogo_instantanea":"`+base64.StdEncoding.EncodeToString(n.CatalogoInstantanea)+`"`)) {
		t.Fatal("efecto v3 no liga la instantánea completa en base64")
	}
	// Sustituir únicamente el esquema y la cola de solicitud produce el v3
	// exacto; todos los campos previos conservan bytes y posición.
	esperado := strings.Replace(string(legado), esquemaEfectoAltaV2, esquemaEfectoAltaV3, 1)
	cola := `"observaciones":""},"creado_en"`
	fragmento := `"observaciones":"","necesidad":{` +
		`"esquema":"vec.ct.necesidad_alta.v1",` +
		`"catalogo_ref":"` + n.CatalogoRef + `",` +
		`"catalogo_version":2,` +
		`"catalogo_huella_sha256":"` + n.CatalogoHuellaSHA256 + `",` +
		`"causa_clave":"acumulacion_tareas",` +
		`"periodo":{"inicio":"2026-09-01","fin":"2026-09-30"},` +
		`"jornada_minutos":1125,` +
		`"campos":{"funcional_codigo":"200","justificacion_temporal":"Refuerzo sintético para el servicio.","numero_personas":"2","organica_codigo":"100","porcentaje_financiacion":"100","proyecto_gasto_codigo":"300"},` +
		`"catalogo_instantanea":"` + base64.StdEncoding.EncodeToString(n.CatalogoInstantanea) + `"}},"creado_en"`
	esperado = strings.Replace(esperado, cola, fragmento, 1)
	if string(actual) != esperado {
		t.Fatalf("efecto v3 no preserva orden y bytes v2\nobtenido=%s\nesperado=%s", actual, esperado)
	}
	suma := sha256.Sum256(actual)
	if huella != hex.EncodeToString(suma[:]) {
		t.Fatal("huella no deriva de bytes v3")
	}
	// El constructor de v3 debe incluir cada campo. Aquí se inspecciona su
	// serialización directamente; validación de dominio gobierna valores legales.
	base := construirEfectoAltaCanonicoV3(evidencia.Expediente, datos)
	baseJSON, _ := json.Marshal(base)
	mutaciones := []struct {
		nombre string
		mutar  func(*domain.DatosNecesidadAlta)
	}{
		{"esquema", func(d *domain.DatosNecesidadAlta) { d.Esquema += ".otro" }},
		{"catalogo_ref", func(d *domain.DatosNecesidadAlta) { d.CatalogoRef += ":otro" }},
		{"catalogo_version", func(d *domain.DatosNecesidadAlta) { d.CatalogoVersion++ }},
		{"catalogo_huella", func(d *domain.DatosNecesidadAlta) { d.CatalogoHuellaSHA256 = strings.Repeat("b", 64) }},
		{"causa", func(d *domain.DatosNecesidadAlta) { d.CausaClave = "sustitucion" }},
		{"periodo", func(d *domain.DatosNecesidadAlta) { d.Periodo.Fin = d.Periodo.Fin.AddDate(0, 0, -1) }},
		{"jornada", func(d *domain.DatosNecesidadAlta) { d.JornadaMinutos++ }},
		{"campos", func(d *domain.DatosNecesidadAlta) { d.Campos = map[string]string{"otro": "valor"} }},
		{"instantanea", func(d *domain.DatosNecesidadAlta) { d.CatalogoInstantanea = []byte("otra") }},
	}
	for _, tc := range mutaciones {
		t.Run(tc.nombre, func(t *testing.T) {
			copia := evidencia.Expediente
			modificada := n
			tc.mutar(&modificada)
			copia.Solicitud.Necesidad = &modificada
			otro, err := json.Marshal(construirEfectoAltaCanonicoV3(copia, datos))
			if err != nil || bytes.Equal(baseJSON, otro) {
				t.Fatalf("campo no ligado al efecto: %v", err)
			}
		})
	}
	// El orden de inserción en map no altera el canon.
	inverso := n
	inverso.Campos = map[string]string{}
	for _, k := range []string{"porcentaje_financiacion", "proyecto_gasto_codigo", "numero_personas", "organica_codigo", "justificacion_temporal", "funcional_codigo"} {
		inverso.Campos[k] = n.Campos[k]
	}
	evidencia.Expediente.Solicitud.Necesidad = &inverso
	ordenDistinto, _, err := canonEfectoAlta(evidencia.Expediente, evidencia.Candidatura)
	if err != nil || !bytes.Equal(ordenDistinto, actual) {
		t.Fatalf("map alteró el canon: %v", err)
	}
}
