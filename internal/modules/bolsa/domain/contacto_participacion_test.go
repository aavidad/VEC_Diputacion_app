package domain

import (
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
