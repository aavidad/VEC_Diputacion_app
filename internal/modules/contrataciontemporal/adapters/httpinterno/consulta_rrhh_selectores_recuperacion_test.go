package httpinterno

import (
	"encoding/json"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDetalleRRHHExponeSoloSelectorHistoricoCerrado(t *testing.T) {
	base := detalleRRHHPrueba()
	prueba := ports.ReciboSelectoresRecuperacionRRHH{AccesoRef: "lectura:rrhh:001",
		SelloDetalleSHA256: strings.Repeat("a", 64), SelloSelectoresSHA256: strings.Repeat("b", 64)}
	legacy, err := json.Marshal(proyectarDetalleRRHH(base))
	if err != nil || strings.Contains(string(legacy), "selectores_recuperacion_firmas") {
		t.Fatalf("la ficha legacy cambió: %s, %v", legacy, err)
	}
	vacio, err := base.ConSelectoresRecuperacionFirmas(nil, prueba)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(proyectarDetalleRRHH(vacio))
	if err != nil || !strings.Contains(string(b), `"selectores_recuperacion_firmas":[]`) {
		t.Fatalf("falta la lista vacía del contrato extendido: %s, %v", b, err)
	}
	fila := ports.FirmaSelectorRecuperacionRRHH{FirmaRef: "firma:historica:001", Selector: ports.SelectorRecuperacionFirmaRRHH{
		ExpedienteRef: base.Resumen.ExpedienteRef, VersionExpediente: base.Resumen.Version,
		Documento: "resolucion", PasoOrden: 3, ClaveIdempotencia: "clave-firma-historica-000001",
		CatalogoHuella: strings.Repeat("a", 64), Via: ports.ViaFirmaCertificadoVEC,
	}}
	ampliado, err := base.ConSelectoresRecuperacionFirmas([]ports.FirmaSelectorRecuperacionRRHH{fila}, prueba)
	if err != nil {
		t.Fatal(err)
	}
	b, err = json.Marshal(proyectarDetalleRRHH(ampliado))
	if err != nil {
		t.Fatal(err)
	}
	var salida map[string]any
	if err := json.Unmarshal(b, &salida); err != nil {
		t.Fatal(err)
	}
	filas, ok := salida["selectores_recuperacion_firmas"].([]any)
	if !ok || len(filas) != 1 {
		t.Fatalf("selector ausente: %s", b)
	}
	wrapper := filas[0].(map[string]any)
	selector := wrapper["selector"].(map[string]any)
	if len(wrapper) != 2 || len(selector) != 7 || wrapper["firma_ref"] != fila.FirmaRef ||
		selector["expediente_ref"] != fila.Selector.ExpedienteRef || selector["via"] != ports.ViaFirmaCertificadoVEC ||
		strings.Contains(string(b), "unidad_ref\":\"unidad:historica") || strings.Contains(string(b), "certificado_huella") {
		t.Fatalf("selector público divergente: %s", b)
	}
	filasPrivadas, _ := ampliado.SelectoresRecuperacionFirmas()
	filasPrivadas[0].Selector.Documento = "alterado"
	comprobacion, _ := ampliado.SelectoresRecuperacionFirmas()
	if comprobacion[0].Selector.Documento != "resolucion" {
		t.Fatal("el selector comparte memoria con el lector")
	}
}
