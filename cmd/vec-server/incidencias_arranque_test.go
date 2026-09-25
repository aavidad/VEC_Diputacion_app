package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestRegistrarFalloArranqueEmiteIncidenciaCerrada(t *testing.T) {
	t.Setenv(envEntornoSupervision, "presentacion")
	var destino bytes.Buffer
	registrarFalloArranque(&destino, domain.ComponenteIncidenciaComposicion, domain.EtapaIncidenciaComposicion)
	lineas := strings.Split(strings.TrimSuffix(destino.String(), "\n"), "\n")
	if len(lineas) != 1 {
		t.Fatalf("se esperaba una linea: %q", destino.String())
	}
	var campos map[string]any
	if err := json.Unmarshal([]byte(lineas[0]), &campos); err != nil {
		t.Fatal(err)
	}
	if campos["codigo"] != "ARRANQUE_FALLIDO" || campos["componente"] != "composicion" || campos["etapa"] != "composicion" || campos["severidad"] != "critica" || campos["entorno"] != "presentacion" {
		t.Fatalf("incidencia inesperada: %v", campos)
	}
}

func TestRegistrarFalloArranqueNoRetieneLaSalidaConDestinoBloqueado(t *testing.T) {
	t.Setenv(envEntornoSupervision, "10.1.2.3")
	bloqueo := make(chan struct{})
	defer close(bloqueo)
	inicio := time.Now()
	registrarFalloArranque(escritorBloqueado(bloqueo), domain.ComponenteIncidenciaServidor, domain.EtapaIncidenciaEscucha)
	if transcurrido := time.Since(inicio); transcurrido > plazoRegistroFalloArranque+time.Second {
		t.Fatalf("el registro retuvo la salida %v", transcurrido)
	}
}

type escritorBloqueado chan struct{}

func (e escritorBloqueado) Write(p []byte) (int, error) {
	<-e
	return len(p), nil
}
