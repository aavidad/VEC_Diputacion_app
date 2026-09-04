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

func TestAutoridadSinteticaContratacionTemporalDesarrolloEsEstableYNoColisiona(
	t *testing.T,
) {
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	primero, err := nuevoContextoAltaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := nuevoContextoAltaContratacionTemporalDesarrollo(
		principal, ahora.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	if primero.Resultado.RegistroContextoRef != segundo.Resultado.RegistroContextoRef ||
		!bytes.Equal(
			primero.Resultado.RepresentacionCanonica,
			segundo.Resultado.RepresentacionCanonica,
		) ||
		!bytes.Equal(
			primero.Resultado.ManifiestoProcedenciaCanonico,
			segundo.Resultado.ManifiestoProcedenciaCanonico,
		) {
		t.Fatal("el contexto sintetico cambio entre dos arranques")
	}
	altaPrimera, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"prf_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	altaRepetida, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"prf_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ahora.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	otraIdentidad, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		"per_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"prf_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	if altaPrimera.AsignacionPerfil.Referencia() !=
		altaRepetida.AsignacionPerfil.Referencia() {
		t.Fatal("la asignacion cambio entre dos arranques")
	}
	if altaPrimera.AsignacionPerfil.Referencia() ==
		otraIdentidad.AsignacionPerfil.Referencia() {
		t.Fatal("dos identidades comparten referencia de asignacion")
	}
	otroPerfil, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"prf_cccccccccccccccccccccccccccccccc",
		ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	if altaPrimera.AsignacionPerfil.Referencia() ==
		otroPerfil.AsignacionPerfil.Referencia() {
		t.Fatal("dos certificados de una identidad comparten asignacion")
	}
}
