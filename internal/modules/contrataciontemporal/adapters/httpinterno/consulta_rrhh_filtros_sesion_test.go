package httpinterno

import (
	"net/http/httptest"
	"testing"
)

func TestParserCuadroRRHHSesionV2FijaFiltrosYResumen(t *testing.T) {
	const cuerpo = `{"filtros":{"texto":"2026/CT","centro_ref":"centro:desarrollo:001","categoria_ref":"categoria:desarrollo:c2","estados_clave":["en_curso","completado"],"fases_clave":["solicitud","fiscalizacion"]},"paginacion":{"limite":25,"cursor":""},"resumen":true}`
	solicitud, resumen, err := solicitudCuadroRRHHSesionV2DesdePeticion(
		httptest.NewRecorder(), nuevaPeticionConsultaRRHHPrueba(RutaConsultaCuadroRRHH, cuerpo))
	if err != nil || !resumen || solicitud.CentroRef() != "centro:desarrollo:001" ||
		solicitud.CategoriaRef() != "categoria:desarrollo:c2" || solicitud.Limite() != 25 ||
		len(solicitud.EstadosClave()) != 2 || len(solicitud.FasesClave()) != 2 {
		t.Fatalf("contrato de filtros v2 perdido: %v", err)
	}
}

func TestParserCuadroRRHHSesionV2ValidaPlazoGobernado(t *testing.T) {
	const valido = `{"filtros":{"texto":"","plazo_estado":"vencido"},"paginacion":{"limite":25,"cursor":""}}`
	s, _, err := solicitudCuadroRRHHSesionV2DesdePeticion(
		httptest.NewRecorder(), nuevaPeticionConsultaRRHHPrueba(RutaConsultaCuadroRRHH, valido))
	if err != nil || s.PlazoEstado() != "vencido" {
		t.Fatalf("filtro de plazo perdido: %v", err)
	}
	const cuerpo = `{"filtros":{"texto":"","plazo_estado":"otro"},"paginacion":{"limite":25,"cursor":""}}`
	if _, _, err := solicitudCuadroRRHHSesionV2DesdePeticion(
		httptest.NewRecorder(), nuevaPeticionConsultaRRHHPrueba(RutaConsultaCuadroRRHH, cuerpo)); err == nil {
		t.Fatal("un filtro de plazo desconocido pasó el contrato HTTP")
	}
}
