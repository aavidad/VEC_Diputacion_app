package ensayologicopg

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestErroresLecturaDeValidacionSonCerrados(t *testing.T) {
	ausente := filepath.Join(t.TempDir(), "ausente")
	incompleto := archivoFixture(t, "cabecera.dump", []byte("PG"))
	invalido := archivoFixture(t, "cabecera.dump", []byte("OTROSdatos"))
	for _, ruta := range []string{ausente, incompleto.Ruta, invalido.Ruta} {
		if err := validarDump(ruta); !errors.Is(err, errEntrada) {
			t.Fatalf("error cerrado esperado, obtenido %v", err)
		}
	}
	if err := validarGlobals(context.Background(), ausente, 1024); !errors.Is(err, errEntrada) {
		t.Fatalf("error cerrado esperado, obtenido %v", err)
	}
}

func TestNumeroVersionNoConfundeErroresNiDesborda(t *testing.T) {
	for _, caso := range []struct{ entrada, esperado string }{
		{"18.4", "180004"},
		{"18.0", "180000"},
		{"4294967295.4294967295", "42953967917295"},
		{"18", MotivoVersionNoComprobable},
		{"18.4.extra", MotivoVersionNoComprobable},
		{"texto.4", MotivoVersionNoComprobable},
		{"18.texto", MotivoVersionNoComprobable},
		{"4294967296.4", MotivoVersionNoComprobable},
		{"18.4294967296", MotivoVersionNoComprobable},
	} {
		if obtenido := numeroVersion(caso.entrada); obtenido != caso.esperado {
			t.Errorf("entrada=%q esperado=%q obtenido=%q", caso.entrada, caso.esperado, obtenido)
		}
	}
}
