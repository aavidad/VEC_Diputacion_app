package httpinterno

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

func cuerpoNecesidadAltaPrueba(t *testing.T) []byte {
	t.Helper()
	var sobre map[string]any
	if err := json.Unmarshal(cuerpoValidoPrueba(), &sobre); err != nil {
		t.Fatal(err)
	}
	sobre["esquema"] = application.EsquemaAltaNecesidadV1
	sobre["solicitud"].(map[string]any)["motivo_clave"] = "vacante"
	sobre["necesidad"] = map[string]any{
		"esquema": "vec.ct.necesidad_alta.v1", "catalogo_ref": "catalogo:ct:necesidades_alta:ejemplo",
		"catalogo_version": 1, "catalogo_huella_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"causa_clave": "vacante", "jornada_minutos": 2250,
		"campos": map[string]string{"plaza_codigo": "1201", "puesto_codigo": "3388"},
	}
	contenido, err := json.Marshal(sobre)
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func TestAltaNecesidadVersionadaNoAceptaInstantaneaDelCliente(t *testing.T) {
	var legado map[string]any
	if err := json.Unmarshal(cuerpoValidoPrueba(), &legado); err != nil {
		t.Fatal(err)
	}
	legado["esquema"] = ""
	legadoExplicito, err := json.Marshal(legado)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := solicitudAltaDesdePeticion(httptest.NewRecorder(), nuevaPeticionPrueba(t, legadoExplicito)); err == nil {
		t.Fatal("v1 aceptó campo de esquema añadido")
	}
	entrada, err := solicitudAltaDesdePeticion(httptest.NewRecorder(), nuevaPeticionPrueba(t, cuerpoNecesidadAltaPrueba(t)))
	if err != nil || entrada.EsquemaAlta != application.EsquemaAltaNecesidadV1 || entrada.Necesidad == nil ||
		entrada.Solicitud.Necesidad != nil || len(entrada.Necesidad.CatalogoInstantanea) != 0 ||
		entrada.Necesidad.CausaClave != "vacante" || entrada.Necesidad.Periodo != entrada.Solicitud.Periodo {
		t.Fatalf("DTO nuevo no quedó separado para sellado servidor: %#v %v", entrada, err)
	}
	comando, ok := comandoDesdeContextoCanal(contextoCanalValidoPrueba(), entrada)
	if !ok || comando.EsquemaAlta != application.EsquemaAltaNecesidadV1 ||
		comando.NecesidadEntrada == nil || comando.NecesidadEntrada.CausaClave != "vacante" {
		t.Fatal("la necesidad no llegó al mismo servicio de alta")
	}
	var sobre map[string]any
	if err := json.Unmarshal(cuerpoNecesidadAltaPrueba(t), &sobre); err != nil {
		t.Fatal(err)
	}
	sobre["necesidad"].(map[string]any)["catalogo_instantanea"] = "YWJj"
	adulterado, err := json.Marshal(sobre)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := solicitudAltaDesdePeticion(httptest.NewRecorder(), nuevaPeticionPrueba(t, adulterado)); err == nil {
		t.Fatal("el cliente aportó una instantánea que sólo debe fijar el servidor")
	}
	delete(sobre["necesidad"].(map[string]any), "catalogo_instantanea")
	delete(sobre, "numero_expediente_moad")
	sinNumero, err := json.Marshal(sobre)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := solicitudAltaDesdePeticion(httptest.NewRecorder(), nuevaPeticionPrueba(t, sinNumero)); err == nil {
		t.Fatal("v3 nuevo sin número MOAD admitido")
	}
	if _, err := solicitudAltaDesdePeticion(httptest.NewRecorder(), nuevaPeticionPrueba(t,
		bytes.Replace(cuerpoNecesidadAltaPrueba(t), []byte(`"vec.ct.alta_necesidad.v1"`), []byte(`"vec.ct.alta_necesidad.v9"`), 1))); err == nil {
		t.Fatal("esquema desconocido admitido")
	}
}
