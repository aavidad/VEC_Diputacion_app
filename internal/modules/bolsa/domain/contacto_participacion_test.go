package domain

import (
	"strings"
	"testing"
	"time"
)

func TestContactoParticipacionCatalogoYPrivacidad(t *testing.T) {
	base := ContactoParticipacion{ContactoRef: "contacto:01234567", BolsaRef: "bolsa:01234567", ParticipacionRef: "participacion:01234567", Canal: "telefono", Actor: "per_0123456789abcdefghijkl", Resultado: "contactado", Anotacion: "Se explicó la oferta y queda pendiente de respuesta", Instante: time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)}
	if err := base.Validar(); err != nil {
		t.Fatalf("contacto válido: %v", err)
	}
	for _, mutar := range []func(*ContactoParticipacion){func(c *ContactoParticipacion) { c.Canal = "fax" }, func(c *ContactoParticipacion) { c.Resultado = "inventado" }, func(c *ContactoParticipacion) { c.Anotacion = "Correo tercero@example.test" }} {
		c := base
		mutar(&c)
		if c.Validar() == nil {
			t.Fatal("contacto inválido admitido")
		}
	}
}

func TestContactoDeOfertaDistingueEnvioYEntregaDeclarada(t *testing.T) {
	base := ContactoParticipacion{ContactoRef: "contacto:01234567", BolsaRef: "bolsa:01234567", ParticipacionRef: "participacion:01234567", OfertaRef: "oferta:" + strings.Repeat("a", 64), Canal: CanalContactoCorreo, Actor: "per_0123456789abcdefghijkl", Resultado: ResultadoContactoEnviado, Anotacion: "RRHH anotó el envío del extracto", Instante: time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC)}
	if err := base.Validar(); err != nil {
		t.Fatalf("envío válido: %v", err)
	}
	declarada := base
	declarada.Resultado = ResultadoContactoEntregaDeclarada
	declarada.EvidenciaRef = "acuse:oferta_sintetica_01"
	declarada.EvidenciaHuellaSHA256 = strings.Repeat("b", 64)
	if err := declarada.Validar(); err != nil {
		t.Fatalf("declaración válida: %v", err)
	}
	for nombre, cambiar := range map[string]func(*ContactoParticipacion){
		"sin evidencia":                func(c *ContactoParticipacion) { c.EvidenciaRef = "" },
		"huella inválida":              func(c *ContactoParticipacion) { c.EvidenciaHuellaSHA256 = strings.Repeat("Z", 64) },
		"correo personal en evidencia": func(c *ContactoParticipacion) { c.EvidenciaRef = "correo@ejemplo.test" },
		"llamamiento simultáneo":       func(c *ContactoParticipacion) { c.LlamamientoRef = "llamamiento:01234567" },
		"resultado telefónico":         func(c *ContactoParticipacion) { c.Resultado = ResultadoContactoContactado },
	} {
		c := declarada
		cambiar(&c)
		if c.Validar() == nil {
			t.Errorf("%s admitido", nombre)
		}
	}
}
