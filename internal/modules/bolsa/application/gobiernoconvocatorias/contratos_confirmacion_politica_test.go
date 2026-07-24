package gobiernoconvocatorias

import (
	"context"
	"testing"
)

func TestConfirmacionConservaYValidaPoliticaGobernadaDeCifrado(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	if _, err := e.servicio.Crear(context.Background(), e.orden); err != nil {
		t.Fatal(err)
	}

	original := *e.confirmador.ultima
	if original.PoliticaCifrado == (PoliticaGobernadaCifradoBorrador{}) {
		t.Fatal("la confirmacion descarto la politica gobernada de cifrado")
	}
	if err := original.Validar(); err != nil {
		t.Fatalf("la politica gobernada valida no llego integra a la confirmacion: %v", err)
	}

	t.Run("ausente", func(t *testing.T) {
		alterada := original
		alterada.PoliticaCifrado = PoliticaGobernadaCifradoBorrador{}
		if alterada.Validar() == nil {
			t.Fatal("se acepto una confirmacion sin politica gobernada de cifrado")
		}
	})

	t.Run("alterada", func(t *testing.T) {
		alterada := original
		alterada.PoliticaCifrado.AutoridadRef += ":suplantada"
		if alterada.Validar() == nil {
			t.Fatal("se acepto una politica gobernada de cifrado alterada")
		}
	})
}
