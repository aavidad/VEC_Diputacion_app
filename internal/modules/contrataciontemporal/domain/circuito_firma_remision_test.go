package domain

import "testing"

func circuitoRemisionPrueba() CircuitoFirma {
	return CircuitoFirma{Documentos: []CircuitoFirmaDocumento{
		{Documento: "informe_definitivo", Etiqueta: "Informe definitivo", Pasos: []PasoCircuitoFirma{
			{Orden: 1, Cargo: "Técnico/a de RRHH", Habilita: "siguiente_paso"},
			{Orden: 2, Cargo: "Jefatura del Servicio de RRHH", Habilita: HabilitaRemisionIntervencion},
		}},
		{Documento: "resolucion", Etiqueta: "Resolución", Pasos: []PasoCircuitoFirma{
			{Orden: 1, Cargo: "Diputado/a delegado/a", Habilita: "cierre_circuito"},
		}},
	}}
}

func TestPasosRemisionSinFirmar(t *testing.T) {
	c := circuitoRemisionPrueba()
	if p := PasosRemisionSinFirmar(c, nil); len(p) != 1 || p[0].Documento != "informe_definitivo" || p[0].Orden != 2 {
		t.Fatalf("sin historia el paso habilitante está pendiente: %+v", p)
	}
	soloPrimero := []EstadoCircuitoDocumento{{Documento: "informe_definitivo", Pasos: []EstadoPasoCalculado{
		{Orden: 1, Estado: EstadoPasoFirmado}, {Orden: 2, Estado: EstadoPasoPendienteFirma}}}}
	if p := PasosRemisionSinFirmar(c, soloPrimero); len(p) != 1 {
		t.Fatalf("la firma del paso 1 no habilita la remisión: %+v", p)
	}
	devuelto := []EstadoCircuitoDocumento{{Documento: "informe_definitivo", Pasos: []EstadoPasoCalculado{
		{Orden: 1, Estado: EstadoPasoEnEspera}, {Orden: 2, Estado: EstadoPasoDevuelto}}}}
	if p := PasosRemisionSinFirmar(c, devuelto); len(p) != 1 {
		t.Fatalf("un paso devuelto no habilita la remisión: %+v", p)
	}
	completo := []EstadoCircuitoDocumento{{Documento: "informe_definitivo", Pasos: []EstadoPasoCalculado{
		{Orden: 1, Estado: EstadoPasoFirmado}, {Orden: 2, Estado: EstadoPasoFirmado}}}}
	if p := PasosRemisionSinFirmar(c, completo); len(p) != 0 {
		t.Fatalf("con el paso habilitante firmado no falta nada: %+v", p)
	}
	sinMarca := CircuitoFirma{Documentos: c.Documentos[1:]}
	if p := PasosRemisionSinFirmar(sinMarca, nil); len(p) != 0 {
		t.Fatalf("sin paso habilitante no se exige firma: %+v", p)
	}
}
