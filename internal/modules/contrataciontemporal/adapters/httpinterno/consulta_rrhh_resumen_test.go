package httpinterno

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func cuerpoCuadroResumenRRHHPrueba() string {
	return `{"filtros":{"texto":"2026/CT","estado_clave":"en_curso",` +
		`"fase_clave":"analisis"},"paginacion":{"limite":50,"cursor":""},"resumen":true}`
}

// La portada pide el resumen en la misma consulta del cuadro; la respuesta
// lleva solo recuentos, sin referencias. Sin agregados que cuadren, la
// consulta no se publica.
func TestManejadorConsultaCuadroRRHHPublicaElResumenPedido(t *testing.T) {
	pagina := paginaRRHHPrueba()
	pagina.Totales = &ports.TotalesCuadroRRHH{Total: 1, EnTramitacion: 1}
	pagina.Agregados = &ports.AgregadosCuadroRRHH{
		Recuentos: []ports.RecuentoCuadroRRHH{{EstadoClave: domain.EstadoEnCurso, FaseClave: "analisis", Numero: 1}},
		GruposPlazo: []ports.GrupoPlazoCuadroRRHH{{FaseClave: "analisis",
			Desde: pagina.GeneradaEn.Add(-72 * time.Hour), Numero: 1}},
	}
	pagina.Resumen = &ports.ResumenCuadroRRHH{EnTramite: 1, Vencidos: 1, VencenSemana: 0,
		PorFase: map[domain.ClaveFase]uint64{"analisis": 1}}
	consultor := &consultorCuadroRRHHPrueba{pagina: pagina}
	manejador, err := NuevoManejadorConsultaCuadroRRHH(consultor)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionConsultaRRHHPrueba(RutaConsultaCuadroRRHH, cuerpoCuadroResumenRRHHPrueba()))
	if respuesta.Code != http.StatusOK || !consultor.solicitud.Resumen() {
		t.Fatalf("estado=%d resumen pedido=%v cuerpo=%s", respuesta.Code, consultor.solicitud.Resumen(), respuesta.Body)
	}
	var salida struct {
		Data struct {
			Resumen map[string]json.RawMessage `json:"resumen"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	if string(salida.Data.Resumen["en_tramite"]) != "1" || string(salida.Data.Resumen["vencidos"]) != "1" ||
		string(salida.Data.Resumen["por_fase"]) != `{"analisis":1}` || len(salida.Data.Resumen) != 7 {
		t.Fatalf("resumen inesperado: %s", respuesta.Body)
	}
	if strings.Contains(respuesta.Body.String(), "desde") || strings.Contains(respuesta.Body.String(), "grupos") {
		t.Fatalf("se publicaron los agregados internos: %s", respuesta.Body)
	}

	// Pedido sin agregados (o con agregados que no cuadran): no se publica.
	pagina.Agregados = nil
	pagina.Resumen = nil
	consultor = &consultorCuadroRRHHPrueba{pagina: pagina}
	manejador, _ = NuevoManejadorConsultaCuadroRRHH(consultor)
	respuesta = httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionConsultaRRHHPrueba(RutaConsultaCuadroRRHH, cuerpoCuadroResumenRRHHPrueba()))
	if respuesta.Code == http.StatusOK {
		t.Fatalf("resumen pedido sin agregados publicado: %s", respuesta.Body)
	}

	// La lista no lo pide y la respuesta no lo lleva.
	consultor = &consultorCuadroRRHHPrueba{pagina: paginaRRHHPrueba()}
	manejador, _ = NuevoManejadorConsultaCuadroRRHH(consultor)
	respuesta = httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionConsultaRRHHPrueba(RutaConsultaCuadroRRHH, cuerpoCuadroRRHHPrueba()))
	if respuesta.Code != http.StatusOK || consultor.solicitud.Resumen() || strings.Contains(respuesta.Body.String(), `"resumen"`) {
		t.Fatalf("la lista pidió o recibió el resumen: %d %s", respuesta.Code, respuesta.Body)
	}
}
