package cobertura

import (
	"strconv"
	"testing"
)

func nuevaMaquinaIdentidadesC1Prueba(
	total uint64,
) *sesionControladaOperacionDecisionCobertura {
	return &sesionControladaOperacionDecisionCobertura{
		cabecera: DatosCabeceraSesionTCBOperacionDecisionCobertura{
			NumeroConsumosC1: total,
		},
		peticiones: make(map[string]struct{}, total),
		respuestas: make(map[string]struct{}, total),
	}
}

func TestMaquinaSesionTCBAceptaExactamenteUnoHastaQuinientosDoceConsumos(
	t *testing.T,
) {
	unitaria := nuevaMaquinaIdentidadesC1Prueba(1)
	if !unitaria.registrarIdentidadConsumoC1(
		1,
		1,
		"peticion_unica",
		"respuesta_unica",
	) || unitaria.consumos != 1 {
		t.Fatal("se rechazó un lote C1 unitario")
	}

	maquina := nuevaMaquinaIdentidadesC1Prueba(
		MaximoConsumosC1SesionTCBOperacionDecisionCobertura,
	)
	for posicion := uint64(1); posicion <=
		MaximoConsumosC1SesionTCBOperacionDecisionCobertura; posicion++ {
		sufijo := strconv.FormatUint(posicion, 10)
		if !maquina.registrarIdentidadConsumoC1(
			posicion,
			MaximoConsumosC1SesionTCBOperacionDecisionCobertura,
			"peticion_"+sufijo,
			"respuesta_"+sufijo,
		) {
			t.Fatalf("se rechazó el consumo válido %d", posicion)
		}
	}
	if maquina.consumos !=
		MaximoConsumosC1SesionTCBOperacionDecisionCobertura {
		t.Fatalf("conteo final incorrecto: %d", maquina.consumos)
	}
	excesiva := nuevaMaquinaIdentidadesC1Prueba(
		MaximoConsumosC1SesionTCBOperacionDecisionCobertura + 1,
	)
	if excesiva.registrarIdentidadConsumoC1(
		1,
		MaximoConsumosC1SesionTCBOperacionDecisionCobertura+1,
		"peticion_513",
		"respuesta_513",
	) {
		t.Fatal("se aceptó desde su inicio un lote declarado de 513")
	}
}

func TestMaquinaSesionTCBRechazaCeroDesordenYDuplicados(t *testing.T) {
	cero := nuevaMaquinaIdentidadesC1Prueba(0)
	if cero.registrarIdentidadConsumoC1(0, 0, "peticion", "respuesta") ||
		cero.registrarIdentidadConsumoC1(1, 0, "peticion", "respuesta") {
		t.Fatal("se aceptó un lote C1 vacío")
	}

	desorden := nuevaMaquinaIdentidadesC1Prueba(2)
	if desorden.registrarIdentidadConsumoC1(
		2,
		2,
		"peticion_2",
		"respuesta_2",
	) {
		t.Fatal("se aceptó una posición C1 fuera de orden")
	}

	duplicadaPeticion := nuevaMaquinaIdentidadesC1Prueba(2)
	if !duplicadaPeticion.registrarIdentidadConsumoC1(
		1,
		2,
		"peticion_1",
		"respuesta_1",
	) || duplicadaPeticion.registrarIdentidadConsumoC1(
		2,
		2,
		"peticion_1",
		"respuesta_2",
	) {
		t.Fatal("no se rechazó una petición C1 duplicada")
	}

	duplicadaRespuesta := nuevaMaquinaIdentidadesC1Prueba(2)
	if !duplicadaRespuesta.registrarIdentidadConsumoC1(
		1,
		2,
		"peticion_1",
		"respuesta_1",
	) || duplicadaRespuesta.registrarIdentidadConsumoC1(
		2,
		2,
		"peticion_2",
		"respuesta_1",
	) {
		t.Fatal("no se rechazó una respuesta C1 duplicada")
	}
}
