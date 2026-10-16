package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/auditoria"
)

func TestCLIPreperfilBolsaRRHHVerificaYDetectaManipulacion(t *testing.T) {
	b, err := os.ReadFile("../../internal/vec/auditoria/testdata/preperfil_bolsa_ad287.json")
	if err != nil {
		t.Fatal(err)
	}
	var d auditoria.DocumentoVerificacionMixta
	if err = json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	cp, err := json.Marshal(d.Manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "checkpoint.json")
	if err = os.WriteFile(ruta, cp, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--checkpoint", ruta, "--max-bytes", "16384", "--max-registros", "2"}
	for _, esquema := range []string{auditoria.EsquemaVerificacionPreperfil, auditoria.EsquemaVerificacionFronteraAdminTecnicaV1} {
		copia := d
		copia.Esquema = esquema
		valido, _ := json.Marshal(copia)
		var salida bytes.Buffer
		if codigo := ejecutar(args, bytes.NewReader(valido), &salida); codigo != 0 {
			t.Fatalf("vector AD287 %s: código=%d salida=%s", esquema, codigo, &salida)
		}
	}
	alterada := d
	alterada.Registros = append([]auditoria.RegistroMixtoV2(nil), d.Registros...)
	evento := *d.Registros[0].PreperfilBolsaRRHH
	evento.RecursoRef = "cuenta:" + strings.Repeat("f", 64)
	alterada.Registros[0].PreperfilBolsaRRHH = &evento
	mutada, _ := json.Marshal(alterada)
	var salida bytes.Buffer
	if codigo := ejecutar(args, bytes.NewReader(mutada), &salida); codigo != 1 || strings.Contains(salida.String(), evento.RecursoRef) {
		t.Fatalf("manipulación aceptada o reflejada: código=%d salida=%s", codigo, &salida)
	}
	conExtra := bytes.Replace(b, []byte(`"preperfil_bolsa_rrhh": {`), []byte(`"preperfil_bolsa_rrhh": {"contacto_personal":"secreto",`), 1)
	salida.Reset()
	if bytes.Equal(conExtra, b) || ejecutar(args, bytes.NewReader(conExtra), &salida) != 2 || strings.Contains(salida.String(), "secreto") {
		t.Fatalf("campo extraño admitido o reflejado: %s", &salida)
	}
}
