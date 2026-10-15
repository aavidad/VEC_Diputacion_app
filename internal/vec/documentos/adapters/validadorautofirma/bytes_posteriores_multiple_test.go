package validadorautofirma

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/vec/documentos/ports"
)

func TestV2CompletaBytesPosterioresParaOriginalFirmable(t *testing.T) {
	raw, solicitud := corpusV2(t, "01_una_firma")
	verificacion, err := clienteCorpusV2(t, envolturaV2(raw)).VerificarFirmas(context.Background(), solicitud)
	if err != nil || verificacion.Estado != ports.EstadoVerificacionValida ||
		verificacion.CambiosPosteriores.BytesNoFirmados == nil || *verificacion.CambiosPosteriores.BytesNoFirmados != 0 {
		t.Fatalf("revision completa sin longitud comprobable: estado=%s error=%v", verificacion.Estado, err)
	}
	if firma, err := ctapp.ValidarFirmaMultipleParaPaso(solicitud, verificacion, "informe_definitivo", 1, nil); err != nil || firma.Orden != 1 {
		t.Fatalf("el original no admite la revision completa: firma=%d error=%v", firma.Orden, err)
	}
}

func TestV2ColaSinFirmarNoSuperaPoliticaCT(t *testing.T) {
	raw, solicitud := corpusV2(t, "01_una_firma")
	cola := []byte("\ncola sin firmar del ensayo\n")
	solicitud.ContenidoFirmado = append(bytes.Clone(solicitud.ContenidoFirmado), cola...)
	var dictamen map[string]any
	if json.Unmarshal(raw, &dictamen) != nil {
		t.Fatal("fixture inválida")
	}
	// El eco global sigue los bytes de la solicitud. La revisión y sus dos
	// huellas continúan ligadas sólo al prefijo firmado del corpus.
	dictamen["huellaFirmadoSHA256"] = huellaBytesV2(solicitud.ContenidoFirmado)
	cuerpo, err := json.Marshal(dictamen)
	if err != nil {
		t.Fatal(err)
	}
	verificacion, err := clienteCorpusV2(t, envolturaV2(cuerpo)).VerificarFirmas(context.Background(), solicitud)
	if err != nil || verificacion.Estado != ports.EstadoVerificacionValida ||
		verificacion.CambiosPosteriores.BytesNoFirmados == nil || *verificacion.CambiosPosteriores.BytesNoFirmados != uint64(len(cola)) {
		t.Fatalf("cola posterior no detectada: estado=%s error=%v", verificacion.Estado, err)
	}
	if _, err := ctapp.ValidarFirmaMultipleParaPaso(solicitud, verificacion, "informe_definitivo", 1, nil); !errors.Is(err, ctapp.ErrFirmaNoVerificada) {
		t.Fatalf("la cola posterior pasó la política: %v", err)
	}
}

func TestV2ConservaCantidadExplicitaYNoCompletaIndeterminada(t *testing.T) {
	raw, solicitud := corpusV2(t, "04_pagina_posterior")
	verificacion, err := clienteCorpusV2(t, envolturaV2(raw)).VerificarFirmas(context.Background(), solicitud)
	if err != nil || verificacion.Estado != ports.EstadoVerificacionNoValida ||
		verificacion.CambiosPosteriores.BytesNoFirmados == nil || *verificacion.CambiosPosteriores.BytesNoFirmados != 203 {
		t.Fatalf("cantidad expresa alterada: estado=%s error=%v", verificacion.Estado, err)
	}
	for _, nombre := range []string{"06_original_ajeno", "13_doctimestamp"} {
		raw, solicitud := corpusV2(t, nombre)
		verificacion, err := clienteCorpusV2(t, envolturaV2(raw)).VerificarFirmas(context.Background(), solicitud)
		if err != nil || verificacion.Estado != ports.EstadoVerificacionIndeterminada || verificacion.CambiosPosteriores.BytesNoFirmados != nil {
			t.Fatalf("%s: evidencia incierta completada: estado=%s error=%v", nombre, verificacion.Estado, err)
		}
	}
}
