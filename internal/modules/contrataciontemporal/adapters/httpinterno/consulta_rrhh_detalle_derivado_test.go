package httpinterno

import (
	"encoding/json"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaRRHHDetalleDerivadoProyectaInstruccionSinConfundirHitos(t *testing.T) {
	instante := time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)
	detalle := ports.DetalleExpedienteRRHH{
		Hitos: []ports.HitoExpedienteRRHH{{Secuencia: 6, VersionExpediente: 6,
			AccionClave: domain.AccionDerivarNombramiento, RealizadaEn: instante,
			FaseOrigen: domain.FaseFiscalizacion, FaseDestino: domain.FaseFiscalizacion,
			EstadoOrigen: domain.EstadoEnCurso, EstadoDestino: domain.EstadoEnCurso}},
		InstruccionCompartida: &ports.InstruccionCompartidaRRHH{
			DerivacionRef: "derivacion:01", PadreRef: "expediente:padre:01",
			NumeroVisiblePadre: "2026/CT-01", OfertaRef: "oferta:01", NumeroPlaza: 1,
			ReciboDerivacionRef: "recibo:derivacion:01", HuellaPadreSHA256: "abc",
			CreadoEnPadre: instante.Add(-time.Hour), ActualizadoEnPadre: instante.Add(-time.Minute),
			HitosPadre: []ports.HitoExpedienteRRHH{{Secuencia: 1, VersionExpediente: 1,
				AccionClave: "solicitud.registrar", RealizadaEn: instante.Add(-time.Hour),
				FaseDestino: "solicitud", EstadoOrigen: domain.EstadoPendiente,
				EstadoDestino: domain.EstadoEnCurso}},
		},
	}
	contenido, err := json.Marshal(proyectarDetalleRRHH(detalle))
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &wire); err != nil {
		t.Fatal(err)
	}
	var compartida map[string]json.RawMessage
	if err := json.Unmarshal(wire["instruccion_compartida"], &compartida); err != nil || len(compartida) != 10 {
		t.Fatalf("contrato de 10 campos: %v %s", err, wire["instruccion_compartida"])
	}
	var propios, originales []map[string]any
	if json.Unmarshal(wire["hitos"], &propios) != nil ||
		json.Unmarshal(compartida["hitos_padre"], &originales) != nil ||
		len(propios) != 1 || len(originales) != 1 ||
		propios[0]["secuencia"] != float64(6) || originales[0]["secuencia"] != float64(1) {
		t.Fatal("se mezclaron los hitos")
	}
	detalle.InstruccionCompartida = nil
	legado, _ := json.Marshal(proyectarDetalleRRHH(detalle))
	var wireLegado map[string]json.RawMessage
	_ = json.Unmarshal(legado, &wireLegado)
	if _, aparece := wireLegado["instruccion_compartida"]; aparece {
		t.Fatal("el legado añadió el bloque derivado")
	}
}

func TestConsultaRRHHDetalleDerivadoProyectaPropuestaMinima(t *testing.T) {
	instante := time.Date(2026, 10, 18, 8, 4, 0, 0, time.UTC)
	detalle := ports.DetalleExpedienteRRHH{PropuestaFormalizacion: &ports.PropuestaFormalizacionRRHH{
		PropuestaRef: "propuesta:ct:01", ReciboRef: "recibo:propuesta:01",
		Version: 7, ConfirmadaEn: instante,
	}}
	contenido, err := json.Marshal(proyectarDetalleRRHH(detalle))
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if json.Unmarshal(contenido, &wire) != nil {
		t.Fatal("detalle ilegible")
	}
	var propuesta map[string]json.RawMessage
	if json.Unmarshal(wire["propuesta_formalizacion"], &propuesta) != nil || len(propuesta) != 4 {
		t.Fatalf("la propuesta debe contener cuatro campos: %s", wire["propuesta_formalizacion"])
	}
	if string(propuesta["version"]) != "7" || string(propuesta["confirmada_en"]) != `"2026-10-18T08:04:00Z"` {
		t.Fatal("versión o fecha de confirmación distintas del recibo")
	}
	detalle.PropuestaFormalizacion = nil
	legado, _ := json.Marshal(proyectarDetalleRRHH(detalle))
	wire = make(map[string]json.RawMessage)
	_ = json.Unmarshal(legado, &wire)
	if _, aparece := wire["propuesta_formalizacion"]; aparece {
		t.Fatal("la ausencia de propuesta añadió un bloque")
	}
}
