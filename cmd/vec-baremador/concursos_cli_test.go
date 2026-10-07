package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"vec-diputacion-granada/internal/modules/bolsa/application/simulacionbaremo"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
)

func TestConcursosCLIArchivoYHuella(t *testing.T) {
	ejemplos, err := simulacion.Ejemplos()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	args := []string{"--modo", "concursos"}
	for _, x := range []struct {
		nombre string
		valor  any
	}{{"reglas", ejemplos[0].Configuracion}, {"entrada", ejemplos[0].Entrada}} {
		datos, err := json.Marshal(x.valor)
		if err != nil {
			t.Fatal(err)
		}
		ruta := filepath.Join(dir, x.nombre+".json")
		if err := os.WriteFile(ruta, datos, 0600); err != nil {
			t.Fatal(err)
		}
		suma := sha256.Sum256(datos)
		args = append(args, "--"+x.nombre, ruta, "--"+x.nombre+"-sha256", hex.EncodeToString(suma[:]))
	}
	var salida, diagnostico bytes.Buffer
	if rc := ejecutar(args, &salida, &diagnostico, simulacionbaremo.Servicio{}); rc != 0 {
		t.Fatalf("rc=%d diagnóstico=%s", rc, diagnostico.String())
	}
	var sobre simulacion.SobreResultado
	if err := json.Unmarshal(salida.Bytes(), &sobre); err != nil {
		t.Fatal(err)
	}
	if sobre.Resultado.Total == nil || sobre.Resultado.Total.Micropuntos() != 28_386_027 || sobre.Alcance != "simulacion" {
		t.Fatalf("sobre incorrecto: %+v", sobre)
	}
	salida.Reset()
	diagnostico.Reset()
	args[len(args)-1] = "0000000000000000000000000000000000000000000000000000000000000000"
	if rc := ejecutar(args, &salida, &diagnostico, simulacionbaremo.Servicio{}); rc != 2 || salida.Len() != 0 {
		t.Fatalf("SHA distinta se aceptó: rc=%d salida=%s", rc, salida.String())
	}
}
func TestConcursosCLIEjemploYReproduccion(t *testing.T) {
	var uno, dos, errout bytes.Buffer
	args := []string{"--modo", "concursos", "--ejemplo", "ejemplo:concursos:v1"}
	if ejecutar(args, &uno, &errout, simulacionbaremo.Servicio{}) != 0 || ejecutar(args, &dos, &errout, simulacionbaremo.Servicio{}) != 0 {
		t.Fatal(errout.String())
	}
	if !bytes.Equal(uno.Bytes(), dos.Bytes()) {
		t.Fatal("CLI no reprodujo los bytes")
	}
}
