package bootstrap

import (
	"io"
	"path/filepath"
	"testing"
	"time"
)

func TestDerivarCoordenadasCTPreparacionCoincideConPublicador(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	ahora := time.Date(2026, 10, 9, 1, 0, 0, 0, time.UTC)
	idempotencia := filepath.Join(cfg.DevelopmentMaterialDir, "idempotencia")
	actual, err := DerivarCoordenadasCTPreparacion(idempotencia, ahora)
	if err != nil {
		t.Fatal(err)
	}
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	publicador, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(composicion.derivadorIdempotencia, ahora)
	if err != nil {
		t.Fatal(err)
	}
	defer publicador.borrarCopiasEfimeras()
	if actual.RaizID != publicador.claveID || actual.RaizVersion != publicador.claveVersion ||
		actual.HuellaSPKI != publicador.spkiHuella || actual.EmisorID != publicador.emisorID ||
		actual.Audiencia != audienciaAtestacionContratacionTemporalDesarrollo {
		t.Fatal("coordenadas CT distintas del publicador")
	}
	if _, err := DerivarCoordenadasCTPreparacion(filepath.Dir(idempotencia), ahora); err == nil {
		t.Fatal("ruta distinta de idempotencia aceptada")
	}
}
