package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
)

// La operación de datos es explícita. Nunca concede perfiles ni presenta
// esta publicación técnica como gobierno o firma de un documento.
type operacionPlantillas struct {
	Version         int    `json:"version_destino"`
	Huella          string `json:"huella_destino_sha256"`
	VersionAnterior int    `json:"version_anterior"`
	HuellaAnterior  string `json:"huella_anterior_sha256"`
	Clave           string `json:"clave_operacion"`
	Aprobacion      string `json:"aprobacion_ref"`
}

func operacionV3() operacionPlantillas {
	return operacionPlantillas{Version: 3, Huella: huellaV3, VersionAnterior: 1, HuellaAnterior: huellaBase, Clave: claveOperacion, Aprobacion: aprobacionOperacion}
}

func argumentosOperacion(args []string) (string, bool, operacionPlantillas, error) {
	op := operacionV3()
	if len(args) >= 5 && args[0] == "--catalogo" && args[2] == "--operacion" {
		f, err := os.Open(args[3])
		if err != nil {
			return "", false, op, errors.New("operacion_archivo_no_valido")
		}
		b, lecturaErr := io.ReadAll(io.LimitReader(f, 4097))
		cierreErr := f.Close()
		if lecturaErr != nil || cierreErr != nil || len(b) > 4096 {
			return "", false, op, errors.New("operacion_archivo_no_valido")
		}
		op = operacionPlantillas{}
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if dec.Decode(&op) != nil || dec.Decode(new(any)) != io.EOF || op.Version != 4 || op.VersionAnterior != 3 || op.HuellaAnterior != huellaV3 ||
			!regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(op.Huella) || !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(op.Clave) ||
			op.Aprobacion != "operacion:datos-ficticios:plantillas-informe-v4:20261013" {
			return "", false, op, errors.New("operacion_contrato_no_valido")
		}
		args = append(append([]string(nil), args[:2]...), args[4:]...)
		if len(args) == 4 && args[2] == "--aplicar" {
			if args[3] != "PUBLICA-INFORME-PREVIO-V4" {
				return "", false, op, errors.New("operacion_confirmacion_no_valida")
			}
			args[3] = "PUBLICA-PLANTILLAS-V3"
		}
	}
	ruta, comprobar, err := argumentos(args)
	return ruta, comprobar, op, err
}
