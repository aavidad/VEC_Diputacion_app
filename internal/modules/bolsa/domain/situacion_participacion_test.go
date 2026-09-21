package domain

import (
	"testing"
	"time"
)

func TestCambioSituacionParticipacionRespetaCatalogoYTransicionesProvisionales(t *testing.T) {
	ahora := time.Date(2026, 9, 21, 14, 0, 0, 0, time.UTC)
	valido := CambioSituacionParticipacion{ParticipacionRef: "participacion:caso-b2", Origen: SituacionDisponible, Destino: SituacionNoDisponible, Desde: ahora, Motivo: "Indisponibilidad comunicada", RegistradaEn: ahora}
	if err := valido.Validar(); err != nil {
		t.Fatalf("cambio valido: %v", err)
	}
	if err := (CambioSituacionParticipacion{ParticipacionRef: valido.ParticipacionRef, Origen: SituacionExcluido, Destino: SituacionDisponible, Desde: ahora, Motivo: valido.Motivo, RegistradaEn: ahora}).Validar(); err == nil {
		t.Fatal("la readmision no pertenece a B2")
	}
	if err := (CambioSituacionParticipacion{ParticipacionRef: valido.ParticipacionRef, Origen: SituacionTrabajando, Destino: SituacionDisponibleDesde, Desde: ahora, Motivo: valido.Motivo, RegistradaEn: ahora}).Validar(); err == nil {
		t.Fatal("disponible_desde exige fecha futura")
	}
	fecha := ahora.Add(24 * time.Hour)
	if err := (CambioSituacionParticipacion{ParticipacionRef: valido.ParticipacionRef, Origen: SituacionTrabajando, Destino: SituacionDisponibleDesde, Desde: ahora, Motivo: valido.Motivo, FechaDisponible: &fecha, RegistradaEn: ahora}).Validar(); err != nil {
		t.Fatalf("fecha futura valida: %v", err)
	}
	if err := (CambioSituacionParticipacion{ParticipacionRef: valido.ParticipacionRef, Origen: SituacionTrabajando, Destino: SituacionDisponible, Desde: ahora.Add(-time.Microsecond), Motivo: valido.Motivo, RegistradaEn: ahora}).Validar(); err == nil {
		t.Fatal("desde anterior a la situacion vigente debe rechazarse")
	}
}

func TestDestinosSituacionParticipacionEsCatalogoCerrado(t *testing.T) {
	destinos := DestinosSituacionParticipacion(SituacionPendienteIncorporacion)
	if len(destinos) != 4 || destinos[0] != SituacionDisponible || destinos[1] != SituacionTrabajando {
		t.Fatalf("destinos=%v", destinos)
	}
	if len(DestinosSituacionParticipacion(SituacionExcluido)) != 0 {
		t.Fatal("excluido no tiene salida en B2")
	}
}
