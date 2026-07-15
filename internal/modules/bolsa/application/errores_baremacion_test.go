package application

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestErroresBaremacionNoExponenReferenciasNiRecibos(t *testing.T) {
	causa := errors.New("causa-secreta-ref")
	errores := []error{
		&ErrorDocumentoFirmadoHuerfano{
			DecisionRef: "decision-secreta-ref",
			Documento: puertosbolsa.DocumentoFirmadoCustodiado{
				DocumentoFirmadoRef: "documento-secreto-ref",
			},
			Causa: causa,
		},
		&ErrorCustodiaBaremacionIncompleta{
			DecisionRef:  "decision-secreta-ref",
			DocumentoRef: "documento-secreto-ref",
			Escritura: puertosvec.ResultadoOperacionObjeto{
				Objeto: puertosvec.ObjetoAlmacenado{
					Objeto: puertosvec.ReferenciaObjetoAlmacen{
						Referencia: "objeto-secreto-ref",
					},
				},
			},
			Retencion: &puertosvec.ResultadoOperacionObjeto{
				Evidencia: puertosvec.EvidenciaOperacionAlmacen{
					Referencia: "retencion-secreta-ref",
				},
			},
			Causa: causa,
		},
	}
	secretos := []string{
		"decision-secreta-ref", "documento-secreto-ref", "objeto-secreto-ref",
		"retencion-secreta-ref", "causa-secreta-ref",
	}

	for _, err := range errores {
		t.Run(fmt.Sprintf("%T", err), func(t *testing.T) {
			salidas := []string{
				fmt.Sprintf("%s", err), fmt.Sprintf("%v", err), fmt.Sprintf("%+v", err),
				fmt.Sprintf("%#v", err), fmt.Sprint(err), fmt.Sprintf("%q", err),
			}
			texto, errTexto := err.(encoding.TextMarshaler).MarshalText()
			if errTexto != nil {
				t.Fatalf("MarshalText: %v", errTexto)
			}
			binario, errBinario := err.(encoding.BinaryMarshaler).MarshalBinary()
			if errBinario != nil {
				t.Fatalf("MarshalBinary: %v", errBinario)
			}
			serializado, errJSON := json.Marshal(err)
			if errJSON != nil {
				t.Fatalf("MarshalJSON: %v", errJSON)
			}
			salidas = append(salidas, string(texto), string(binario), string(serializado))

			var registro bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&registro, nil))
			logger.Error("operacion fallida", "error", err)
			salidas = append(salidas, registro.String())

			for _, salida := range salidas {
				for _, secreto := range secretos {
					if strings.Contains(salida, secreto) {
						t.Fatalf("la representacion %q expone %q", salida, secreto)
					}
				}
			}
			if !errors.Is(err, causa) {
				t.Fatal("se perdio la causa programatica del error")
			}
		})
	}
}

func TestErroresBaremacionNulosMantienenRepresentacionSegura(t *testing.T) {
	var huerfano *ErrorDocumentoFirmadoHuerfano
	var custodia *ErrorCustodiaBaremacionIncompleta

	if fmt.Sprint(huerfano) != mensajeDocumentoFirmadoHuerfano {
		t.Fatalf("huerfano nulo = %q", fmt.Sprint(huerfano))
	}
	if fmt.Sprint(custodia) != mensajeCustodiaBaremacionIncompleta {
		t.Fatalf("custodia nula = %q", fmt.Sprint(custodia))
	}
}
