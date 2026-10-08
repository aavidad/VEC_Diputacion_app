package seguridad

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/catalogoalta"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func materialHuellaNecesidadPrueba(t *testing.T) domain.DatosNecesidadAlta {
	t.Helper()
	catalogo, err := catalogoalta.CargarNecesidades("")
	if err != nil {
		t.Fatal(err)
	}
	material := materialHuellaPrueba()
	dato, err := catalogo.SellarDatos(domain.DatosNecesidadAlta{
		Esquema: "vec.ct.necesidad_alta.v1", CatalogoRef: catalogo.Referencia,
		CatalogoVersion: catalogo.Version, CatalogoHuellaSHA256: catalogo.HuellaSHA256,
		CausaClave: "acumulacion_tareas", Periodo: material.Solicitud.Periodo,
		JornadaMinutos: 1125,
		Campos: map[string]string{
			"justificacion_temporal": "Refuerzo sintético para el servicio.",
			"organica_codigo":        "100", "funcional_codigo": "200",
			"proyecto_gasto_codigo": "300", "porcentaje_financiacion": "100",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return dato
}

func TestHuellaAltaV2LigaNecesidadYConservaV1(t *testing.T) {
	material := materialHuellaPrueba()
	legado, err := materialCanonicoHuellaAlta(material)
	if err != nil || string(legado) != materialHuellaAltaV1Dorado {
		t.Fatalf("preimagen v1 alterada: %v", err)
	}
	necesidad := materialHuellaNecesidadPrueba(t)
	material.Solicitud.MotivoClave = necesidad.CausaClave
	material.Solicitud.Necesidad = &necesidad
	preimagen, err := materialCanonicoHuellaAlta(material)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(preimagen, []byte(`"esquema":"`+esquemaHuellaAltaV2+`"`)) ||
		!bytes.Contains(preimagen, []byte(`"catalogo_instantanea":"`)) ||
		!bytes.Contains(preimagen, []byte(`"jornada_minutos":1125`)) ||
		bytes.Contains(preimagen, necesidad.CatalogoInstantanea) {
		t.Fatal("la preimagen v2 no contiene la necesidad y su instantánea base64")
	}
	sellador := &selladorPrueba{clave: []byte("clave-sintetica-para-v2-000000000000"), referencia: claveHuellaPrueba}
	derivador, err := NuevoDerivadorHuellaAltaHMAC(claveHuellaPrueba, sellador)
	if err != nil {
		t.Fatal(err)
	}
	base, err := derivador.DerivarHuellaAlta(context.Background(), material)
	if err != nil {
		t.Fatal(err)
	}
	baseDatos, _ := base.Datos()
	if string(sellador.material) != string(preimagen) {
		t.Fatal("el sellador recibió otros bytes")
	}
	// Una rotación de clave cambia el sello, pero conserva la preimagen semántica.
	selladorRotado := &selladorPrueba{clave: []byte("otra-clave-sintetica-v2-0000000000"), referencia: dominioClaveHuella + "/v2"}
	conf, err := NuevaConfiguracionSelladorHMAC(claveHuellaPrueba, sellador)
	if err != nil {
		t.Fatal(err)
	}
	confRotada, err := NuevaConfiguracionSelladorHMAC(dominioClaveHuella+"/v2", selladorRotado)
	if err != nil {
		t.Fatal(err)
	}
	rotado, err := NuevoDerivadorHuellaAltaHMACRotable(confRotada, []ConfiguracionSelladorHMAC{conf})
	if err != nil {
		t.Fatal(err)
	}
	coleccion, err := rotado.DerivarHuellaAlta(context.Background(), material)
	if err != nil {
		t.Fatal(err)
	}
	datosRotados, _ := coleccion.Datos()
	if !bytes.Equal(selladorRotado.material, preimagen) || !bytes.Equal(sellador.material, preimagen) ||
		datosRotados.Retenidos[0].Valor != baseDatos.Activo.Valor ||
		datosRotados.Activo.Valor == baseDatos.Activo.Valor {
		t.Fatal("rotación de clave alteró el canon semántico")
	}
	mutaciones := []struct {
		nombre string
		valida bool
		mutar  func(*domain.DatosNecesidadAlta)
	}{
		{"esquema", false, func(d *domain.DatosNecesidadAlta) { d.Esquema += ".otro" }},
		{"catalogo_ref", false, func(d *domain.DatosNecesidadAlta) { d.CatalogoRef += ":otro" }},
		{"catalogo_version", false, func(d *domain.DatosNecesidadAlta) { d.CatalogoVersion++ }},
		{"catalogo_huella", false, func(d *domain.DatosNecesidadAlta) { d.CatalogoHuellaSHA256 = strings.Repeat("b", 64) }},
		{"causa", false, func(d *domain.DatosNecesidadAlta) { d.CausaClave = "sustitucion" }},
		{"periodo", false, func(d *domain.DatosNecesidadAlta) { d.Periodo.Fin = d.Periodo.Fin.AddDate(0, 0, -1) }},
		{"jornada", true, func(d *domain.DatosNecesidadAlta) { d.JornadaMinutos++ }},
		{"campo", true, func(d *domain.DatosNecesidadAlta) { d.Campos["justificacion_temporal"] += " Más horas." }},
		{"instantanea", false, func(d *domain.DatosNecesidadAlta) { d.CatalogoInstantanea = []byte("otra") }},
	}
	for _, tc := range mutaciones {
		t.Run(tc.nombre, func(t *testing.T) {
			copia := material
			n := necesidad
			n.Campos = make(map[string]string, len(necesidad.Campos))
			for k, v := range necesidad.Campos {
				n.Campos[k] = v
			}
			tc.mutar(&n)
			copia.Solicitud.Necesidad = &n
			otro, err := derivador.DerivarHuellaAlta(context.Background(), copia)
			if (err == nil) != tc.valida {
				t.Fatalf("validación inesperada: %v", err)
			}
			// Las mutaciones admisibles cambian el sello; las demás
			// invalidan el vínculo o la instantánea y cierran la operación.
			if err == nil && otro.Contiene(baseDatos.Activo.Valor) {
				t.Fatal("mutación sin efecto en HMAC")
			}
		})
	}
	// La construcción de mapas en otro orden mantiene los mismos bytes.
	inverso := material
	n := necesidad
	n.Campos = make(map[string]string, len(necesidad.Campos))
	for _, k := range []string{"porcentaje_financiacion", "proyecto_gasto_codigo", "funcional_codigo", "organica_codigo", "justificacion_temporal"} {
		n.Campos[k] = necesidad.Campos[k]
	}
	inverso.Solicitud.Necesidad = &n
	igual, err := materialCanonicoHuellaAlta(inverso)
	if err != nil || !bytes.Equal(igual, preimagen) {
		t.Fatalf("orden de mapa cambió la preimagen: %v", err)
	}
	var js map[string]any
	if err := json.Unmarshal(preimagen, &js); err != nil || !strings.Contains(string(preimagen), `"necesidad":{`) {
		t.Fatalf("preimagen inválida: %v", err)
	}
}
