package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/documentos/ports"
)

// La capacidad V3 debe estar ligada a la operación y a la huella de la
// preimagen exacta que se presenta; una decisión fresca de otra preimagen o de
// otra operación se rechaza antes de abrir transacción.
func TestValidarAutorizacionLigaOperacionYHuellaDeLaPreimagen(t *testing.T) {
	expediente := "ref:" + strings.Repeat("b2", 32)
	consulta := ports.ConsultaExpediente{ExpedienteRef: expediente, Limite: 10}
	preimagen, err := consulta.PreimagenListar()
	if err != nil {
		t.Fatal(err)
	}
	otra := ports.ConsultaExpediente{ExpedienteRef: expediente, Limite: 11}
	otraPreimagen, err := otra.PreimagenListar()
	if err != nil {
		t.Fatal(err)
	}
	campos := []string{"items", "siguiente_cursor"}
	ligada := autorizacion(materialSintetico(t, ports.AccionListar, expediente, finalidadListarPrueba, "expediente_documental", campos,
		preimagen, "decision:00000000-0000-4000-8000-0000000000b1"), ports.AccionListar, finalidadListarPrueba, expediente, expediente)
	if _, err := validarAutorizacion(ligada, ports.AccionListar, preimagen); err != nil {
		t.Fatalf("autorización ligada rechazada: %v", err)
	}
	ajena := autorizacion(materialSintetico(t, ports.AccionListar, expediente, finalidadListarPrueba, "expediente_documental", campos,
		otraPreimagen, "decision:00000000-0000-4000-8000-0000000000b2"), ports.AccionListar, finalidadListarPrueba, expediente, expediente)
	if _, err := validarAutorizacion(ajena, ports.AccionListar, preimagen); !errors.Is(err, ports.ErrSolicitudInvalida) {
		t.Fatalf("decisión de otra preimagen aceptada: %v", err)
	}
	otraOperacion := autorizacion(materialSintetico(t, ports.AccionDescargar, expediente, finalidadListarPrueba, "expediente_documental", campos,
		preimagen, "decision:00000000-0000-4000-8000-0000000000b3"), ports.AccionListar, finalidadListarPrueba, expediente, expediente)
	if _, err := validarAutorizacion(otraOperacion, ports.AccionListar, preimagen); !errors.Is(err, ports.ErrSolicitudInvalida) {
		t.Fatalf("capacidad de otra operación aceptada: %v", err)
	}
	// El repositorio rechaza sin tocar PostgreSQL: sin pool, la denegación
	// no se confunde con indisponibilidad.
	consulta.Autorizacion = ajena
	if _, err := (&Repositorio{}).ListarExpediente(context.Background(), consulta); !errors.Is(err, ports.ErrSolicitudInvalida) {
		t.Fatalf("lista con decisión ajena: %v", err)
	}
}

const finalidadListarPrueba = "listar_documentos_expediente"
