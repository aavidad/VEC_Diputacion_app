package ports

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSolicitudCorreoLlamamientoExigeSucesorYLigaduras(t *testing.T) {
	s := SolicitudDespacharCorreoLlamamiento{OrganizacionRef: "org-001", ExpedienteRef: "exp-001", LlamamientoRef: "llamamiento-001", ComunicacionRef: "comunicacion-ct54", IntencionEnvioRef: "intencion-ct54"}
	if s.Validar() != nil {
		t.Fatal("solicitud valida rechazada")
	}
	s.LlamamientoRef = ""
	if s.Validar() == nil {
		t.Fatal("falta llamamiento")
	}
}

func TestCorreoEfimeroSeRedactaEnFmtYJSON(t *testing.T) {
	m, err := NuevoMensajeCorreoLlamamiento("persona@ejemplo.invalid", "secreto", "secreto")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%v", m); got != "MensajeCorreoLlamamiento{redactado}" {
		t.Fatal(got)
	}
	b, e := json.Marshal(m)
	if e != nil || string(b) != "{\"redactado\":true}" {
		t.Fatalf("%s %v", b, e)
	}
}

func TestDestinoCorreoSeRedactaEnFmtYJSON(t *testing.T) {
	d, err := NuevoDestinoCorreoLlamamiento("persona@ejemplo.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%v %#v", d, d); got != "DestinoCorreoLlamamiento{redactado} DestinoCorreoLlamamiento{redactado}" {
		t.Fatal(got)
	}
	b, err := json.Marshal(&d)
	if err != nil || string(b) != "{\"redactado\":true}" {
		t.Fatalf("%s %v", b, err)
	}
}
