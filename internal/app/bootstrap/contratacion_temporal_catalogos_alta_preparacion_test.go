package bootstrap

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// servirCatalogosAltaPrueba responde la ruta de catálogos del alta sobre un
// catálogo ya compuesto, sin red ni identidad: solo el contenido de la respuesta.
func servirCatalogosAltaPrueba(t *testing.T, catalogo *catalogosAltaContratacionTemporalDesarrollo) map[string]json.RawMessage {
	t.Helper()
	ruta, err := nuevaRutaCatalogosAltaContratacionTemporalDesarrollo(nuevoOrigenConsultasConCatalogoDesarrollo(catalogo))
	if err != nil {
		t.Fatal(err)
	}
	grabador := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(grabador, httptest.NewRequest(http.MethodGet, rutaCatalogosAltaContratacionTemporalDesarrollo, nil))
	if grabador.Code != http.StatusOK {
		t.Fatalf("catalogos=%d %s", grabador.Code, grabador.Body.Bytes())
	}
	var sobre struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(grabador.Body.Bytes(), &sobre); err != nil {
		t.Fatal(err)
	}
	return sobre.Data
}

// La relación de documentos y datos de cada vía llega con los catálogos del
// alta, antes de crear la petición, con la misma identidad que la publicación
// que el arranque deja vigente y que usará la propuesta de cobertura.
func TestCatalogosAltaPublicanPreparacionDeLasViasVigentes(t *testing.T) {
	opciones, err := nuevasOpcionesAnalisisCT(t.Context(), resolutorReglasCTPrueba(t, rutaReglasCTEjemploPrueba))
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoDesarrollo("", "")
	if err != nil {
		t.Fatal(err)
	}
	catalogo.componerOpcionesAnalisis(opciones)
	datos := servirCatalogosAltaPrueba(t, catalogo)
	if len(datos) != 7 || datos["preparacion_vias"] == nil || datos["numero_expediente_moad"] == nil {
		t.Fatalf("falta la relación por vía: %v", datos)
	}
	var preparacion struct {
		Referencia   string `json:"referencia"`
		Version      uint64 `json:"version"`
		HuellaSHA256 string `json:"huella_sha256"`
		EsEjemplo    bool   `json:"es_ejemplo"`
		Vias         []struct {
			Clave      string `json:"clave"`
			Orden      uint16 `json:"orden"`
			Documentos []struct {
				Clave     string `json:"clave"`
				Orden     uint16 `json:"orden"`
				ClaveI18n string `json:"clave_i18n"`
			} `json:"documentos"`
			Datos []json.RawMessage `json:"datos"`
		} `json:"vias"`
	}
	decodificador := json.NewDecoder(bytes.NewReader(datos["preparacion_vias"]))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&preparacion); err != nil {
		t.Fatalf("relación con campos no previstos: %v %s", err, datos["preparacion_vias"])
	}
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	deseado, err := gobiernoCoberturaDeseadoParaCatalogoCT(soporte, opciones.viasCoberturaVigentes())
	if err != nil {
		t.Fatal(err)
	}
	if preparacion.Referencia != deseado.catalogo.Referencia || preparacion.Version != deseado.catalogo.Version ||
		preparacion.HuellaSHA256 != deseado.catalogo.HuellaSHA256 || !preparacion.EsEjemplo {
		t.Fatalf("identidad distinta de la publicación vigente: %+v frente a %s v%d", preparacion,
			deseado.catalogo.Referencia, deseado.catalogo.Version)
	}
	if len(preparacion.Vias) != 3 || preparacion.Vias[0].Clave != "bolsa_vigente" ||
		preparacion.Vias[1].Clave != "oferta_sae" || preparacion.Vias[1].Orden != 2 ||
		len(preparacion.Vias[1].Documentos) != 2 || len(preparacion.Vias[1].Datos) != 4 ||
		preparacion.Vias[1].Documentos[1].ClaveI18n != "contratacion_temporal.cobertura.doc.nota_informativa_sae" {
		t.Fatalf("vías o elementos inesperados: %+v", preparacion.Vias)
	}
}

// Sin documentos ni datos declarados en el catálogo de reglas (vías de
// siempre) no hay relación: la respuesta conserva los cinco campos del alta.
func TestCatalogosAltaSinPreparacionConservanSuForma(t *testing.T) {
	catalogo, err := nuevoCatalogoDesarrollo("", "")
	if err != nil {
		t.Fatal(err)
	}
	if datos := servirCatalogosAltaPrueba(t, catalogo); len(datos) != 6 || datos["preparacion_vias"] != nil || datos["numero_expediente_moad"] == nil {
		t.Fatalf("relación inventada sin catálogo: %v", datos)
	}
	if preparacionViasCatalogosAlta(nil) != nil {
		t.Fatal("un catálogo nulo no publica relación")
	}
}
