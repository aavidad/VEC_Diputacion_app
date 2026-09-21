package bootstrap

import (
	"context"
	"errors"
	"testing"
)

func TestOperacionContextoBorradorBolsaEsDistintaYCTConservaReferencia(t *testing.T) {
	directorio, soporte, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	escribirManifiestoIdentidadBorradorBolsa(t, directorio, principal, ahora, nil)
	bolsa, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, soporte, ahora)
	if err != nil {
		t.Fatal(err)
	}
	base := soporte.principalID + "\x00" + soporte.certificadoSHA256
	ctEsperada := referenciaAltaContratacionTemporalDesarrollo(
		"oca_", base+"\x00registro-contexto",
	)
	if actual := operacionContextoContratacionTemporalDesarrollo(soporte); actual != ctEsperada {
		t.Fatalf("referencia CT cambió: %q", actual)
	}
	if bolsa.soporteCanal == nil || operacionContextoBorradorBolsaDesarrollo() == ctEsperada {
		t.Fatal("Bolsa no tiene una operación de contexto separada")
	}
}

func TestPublicarContextoBorradorBolsaFallaCerradoConResultadoInvalido(t *testing.T) {
	soporte := &soporteSesionBorradorBolsaDesarrollo{
		soporteCanal: &soporteAltaContratacionTemporalDesarrollo{},
	}
	err := publicarContextoPostgreSQLBorradorBolsaDesarrollo(context.Background(), nil, soporte)
	if !errors.Is(err, errPostgreSQLContratacionTemporalDesarrolloNoDisponible) {
		t.Fatalf("resultado inválido aceptado: %v", err)
	}
}
