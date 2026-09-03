package bootstrap

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestMaterialAtestacionContratacionTemporalDesarrolloEsEstableYSeparado(
	t *testing.T,
) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 9, 4, 0, 45, 0, 0, time.UTC)
	primero, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(
		composicion.derivadorIdempotencia, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer primero.borrarCopiasEfimeras()
	segundo, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(
		composicion.derivadorIdempotencia, ahora.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer segundo.borrarCopiasEfimeras()
	if primero.claveID != segundo.claveID ||
		primero.claveHMACID != segundo.claveHMACID ||
		primero.spkiHuella != segundo.spkiHuella ||
		primero.claveHMACSecreto != segundo.claveHMACSecreto ||
		primero.configuracionHuella != segundo.configuracionHuella {
		t.Fatal("el material cambio al reconstruir la composicion")
	}
	if bytes.Equal(primero.claveHMAC, primero.privada[:32]) {
		t.Fatal("firma y capacidad reutilizan el mismo material")
	}
	if primero.configuracionRef != "confianza:atestacion:ct:desarrollo:2026-09-04" ||
		primero.configuracionOrden != 20260904 {
		t.Fatalf("gobierno diario inesperado: %q/%d",
			primero.configuracionRef, primero.configuracionOrden)
	}
}
