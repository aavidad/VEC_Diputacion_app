package httpinterno

import (
	"net/http/httptest"
	"testing"
)

func TestContratoCuadroFiltradoUsaVersionTipadaYRechazaPlazoNoSoportado(t *testing.T) {
	const base = `{"filtros":{"texto":"2026/CT","centro_ref":"centro:desarrollo:001","categoria_ref":"categoria:desarrollo:c2","estados_clave":["completado","en_curso"],"fases_clave":["solicitud","fiscalizacion"]},"paginacion":{"limite":25,"cursor":""}}`
	cuadro, err := solicitudCuadroRRHHDesdePeticion(httptest.NewRecorder(),
		nuevaPeticionConsultaRRHHPrueba(RutaConsultaCuadroRRHH, base))
	if err != nil || cuadro.Version() != 2 || cuadro.CentroRef() != "centro:desarrollo:001" ||
		cuadro.CategoriaRef() != "categoria:desarrollo:c2" || len(cuadro.EstadosClave()) != 2 ||
		len(cuadro.FasesClave()) != 2 {
		t.Fatalf("contrato v2 perdido: %+v %v", cuadro, err)
	}
	const conPlazo = `{"filtros":{"texto":"","plazo_estado":"vencido"},"paginacion":{"limite":25,"cursor":""}}`
	if _, err := solicitudCuadroRRHHDesdePeticion(httptest.NewRecorder(),
		nuevaPeticionConsultaRRHHPrueba(RutaConsultaCuadroRRHH, conPlazo)); err == nil {
		t.Fatal("el filtro de plazo aún no tiene predicado SQL autorizado")
	}
	const mezclado = `{"filtros":{"texto":"","estado_clave":"en_curso","estados_clave":["completado"]},"paginacion":{"limite":25,"cursor":""}}`
	if _, err := solicitudCuadroRRHHDesdePeticion(httptest.NewRecorder(),
		nuevaPeticionConsultaRRHHPrueba(RutaConsultaCuadroRRHH, mezclado)); err == nil {
		t.Fatal("dos contratos de estado ambiguos")
	}
}
