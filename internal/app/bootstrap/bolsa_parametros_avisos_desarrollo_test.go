package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	bolsadominio "vec-diputacion-granada/internal/modules/bolsa/domain"
)

func candidatosConMarcasPrueba(t *testing.T, datos datasetBolsasRRHHDesarrollo) map[string]map[string]any {
	t.Helper()
	manejador := nuevoManejadorBolsasRRHHDesarrollo(func(context.Context) (datasetBolsasRRHHDesarrollo, error) { return datos, nil })
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:constituida:administrativo/candidatos", nil))
	if respuesta.Code != http.StatusOK {
		t.Fatalf("candidatos=%d %s", respuesta.Code, respuesta.Body.String())
	}
	var pagina struct {
		Data struct {
			Candidatos []map[string]any `json:"candidatos"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &pagina); err != nil {
		t.Fatal(err)
	}
	salida := map[string]map[string]any{}
	for _, c := range pagina.Data.Candidatos {
		marcas, _ := c["marcas"].(map[string]any)
		salida[c["participacion_ref"].(string)] = marcas
	}
	return salida
}

func TestCandidatosSinMarcasCompuestasConservanElContrato(t *testing.T) {
	for referencia, marcas := range candidatosConMarcasPrueba(t, datosBolsasRRHHPrueba()) {
		if marcas != nil {
			t.Fatalf("%s: sin 000041 no hay marcas: %v", referencia, marcas)
		}
	}
}

func TestCandidatosRotulanServiciosRevisionYEncadenamiento(t *testing.T) {
	datos := datosBolsasRRHHPrueba()
	datos.Marcas = map[string]bolsadominio.MarcasParticipacion{
		"participacion:001": {ParticipacionRef: "participacion:001", PrestaServicios: bolsadominio.ModoPrestaServiciosExcluir, EnRevision: bolsadominio.RevisionRenunciaPendiente, EncadenamientoDias: 580, EncadenamientoUmbralMeses: 18, EncadenamientoVentanaMeses: 24},
	}
	marcas := candidatosConMarcasPrueba(t, datos)
	uno, dos := marcas["participacion:001"], marcas["participacion:002"]
	encadenamiento, _ := uno["encadenamiento"].(map[string]any)
	if uno["presta_servicios"] != "excluir" || uno["en_revision"] != "renuncia_pendiente" || encadenamiento["dias_acumulados"] != float64(580) || encadenamiento["umbral_meses"] != float64(18) {
		t.Fatalf("marcas de la primera: %v", uno)
	}
	if dos == nil || dos["presta_servicios"] != nil || dos["en_revision"] != nil || dos["encadenamiento"] != nil {
		t.Fatalf("la segunda lleva marcas vacías explícitas: %v", dos)
	}
}

func TestCandidatosRotulanLaBajaPropuestaPorIntentosAgotados(t *testing.T) {
	datos := datosBolsasRRHHPrueba()
	datos.Marcas = map[string]bolsadominio.MarcasParticipacion{}
	politica := bolsadominio.PoliticaIntentosTelefonicos{IntentosPorProceso: 2, Procesos: 1, SeparacionMinima: time.Hour, ResultadosSinContacto: []string{bolsadominio.ResultadoContactoNoContesta}}
	datos.PoliticaIntentos = &politica
	base := time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)
	for i := range 2 {
		datos.Contactos = append(datos.Contactos, bolsadominio.ContactoParticipacion{ContactoRef: "contacto:" + string(rune('a'+i)), BolsaRef: "bolsa:constituida:administrativo", ParticipacionRef: "participacion:001", LlamamientoRef: "llamamiento:1", Canal: bolsadominio.CanalContactoTelefono, Instante: base.Add(time.Duration(i) * 2 * time.Hour), Actor: "per_actor", Resultado: bolsadominio.ResultadoContactoNoContesta})
	}
	marcas := candidatosConMarcasPrueba(t, datos)
	if marcas["participacion:001"]["en_revision"] != "baja_propuesta" || marcas["participacion:002"]["en_revision"] != nil {
		t.Fatalf("baja propuesta: %v", marcas)
	}
	datos.Candidaturas[0].Estado = bolsadominio.SituacionExcluido
	if marcas := candidatosConMarcasPrueba(t, datos); marcas["participacion:001"]["en_revision"] != nil {
		t.Fatalf("una exclusión ya confirmada no está en revisión: %v", marcas)
	}
}

func TestParametrosAvisosSinFuenteOSinCatalogoNoCambianNada(t *testing.T) {
	if err := componerParametrosAvisosBolsaDesarrollo(context.Background(), nil, nil); err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteConstituidaRRHHDesarrollo{}
	if err := componerParametrosAvisosBolsaDesarrollo(context.Background(), nil, fuente); err != nil || fuente.marcas != nil {
		t.Fatalf("sin repositorio no se compone: %v", err)
	}
	datos := datasetBolsasRRHHDesarrollo{}
	if err := fuente.cargarMarcasBase(context.Background(), &datos); err != nil || datos.Marcas != nil {
		t.Fatalf("sin marcas el conjunto no cambia: %v", err)
	}
}
