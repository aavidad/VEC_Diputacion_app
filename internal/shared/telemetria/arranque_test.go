package telemetria

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRegistrarArranqueDistingueComposicionDeEscucha(t *testing.T) {
	var destino bytes.Buffer
	RegistrarArranque(&destino, EventoArranque{
		Servicio: "vec-server", Superficie: "interno", Entorno: "desarrollo",
		Fase: "composicion", Resultado: "preparada", Duracion: 45 * time.Millisecond,
	})
	var got map[string]any
	if err := json.Unmarshal(destino.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["msg"] != "vec.process.startup" || got["vec.arranque.fase"] != "composicion" ||
		got["vec.arranque.resultado"] != "preparada" || got["vec.arranque.duracion"] != 0.045 ||
		got["error.type"] != nil {
		t.Fatalf("hito de composición inesperado: %v", got)
	}
}

func TestRegistrarArranqueNoExponeDatosDeErrorNiConfiguracion(t *testing.T) {
	secreto := "/ruta/privada?password=irrepetible"
	var destino bytes.Buffer
	RegistrarArranque(&destino, EventoArranque{
		Servicio: secreto, Superficie: secreto, Entorno: secreto,
		Fase: secreto, Resultado: secreto, Causa: secreto,
	})
	if strings.Contains(destino.String(), secreto) || strings.Contains(destino.String(), "irrepetible") {
		t.Fatalf("dato privado en registro técnico: %s", destino.String())
	}
	var got map[string]any
	if err := json.Unmarshal(destino.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["error.type"] != "otro" || got["vec.arranque.resultado"] != "fallida" {
		t.Fatalf("causa no saneada: %v", got)
	}
	if ClaseErrorArranque(errors.New(secreto)) != "otro" {
		t.Fatal("texto de error aceptado como causa")
	}
}
