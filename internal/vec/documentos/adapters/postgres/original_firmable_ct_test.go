package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	"vec-diputacion-granada/internal/vec/documentos/ports"
)

func TestRepositorioOriginalFirmableRechazaDecisionDeOtroEfectoAntesDeSQL(t *testing.T) {
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	expediente, tipo := ref("7"), ref("8")
	r := ports.ReservaOriginalFirmable{ID: ref("1"), ClaveIdempotencia: ref("4"), ModuloID: "contrataciontemporal",
		ExpedienteRef: expediente, TipoRef: tipo, Version: 1, MIME: "application/pdf",
		HuellaSHA256: strings.Repeat("a", 64), Tamano: 16, Politica: politicaEnsayo(t, expediente, tipo)}
	preimagen, err := docapp.PreimagenReservaOriginalFirmable(r)
	if err != nil {
		t.Fatal(err)
	}
	r.Autorizacion = autorizacion(materialSintetico(t, ports.AccionReservarOriginalFirmable, r.ID,
		ports.FinalidadOriginalFirmable, "documento_original_firmable", []string{"documento"}, preimagen,
		"decision:00000000-0000-4000-8000-0000000000b1"),
		ports.AccionReservarOriginalFirmable, ports.FinalidadOriginalFirmable, r.ID, expediente)
	if _, err := validarAutorizacionOriginalFirmable(r.Autorizacion, ports.AccionReservarOriginalFirmable, preimagen); err != nil {
		t.Fatalf("decision ligada rechazada: %v", err)
	}
	r.HuellaSHA256 = strings.Repeat("b", 64)
	if _, err := (&Repositorio{}).ReservarOriginalFirmable(context.Background(), r); !errors.Is(err, ports.ErrSolicitudInvalida) {
		t.Fatalf("decision de otros bytes llego a SQL: %v", err)
	}
	r.HuellaSHA256 = strings.Repeat("a", 64)
	r.Autorizacion.Accion = ports.AccionConfirmarOriginalFirmable
	if _, err := (&Repositorio{}).ReservarOriginalFirmable(context.Background(), r); !errors.Is(err, ports.ErrSolicitudInvalida) {
		t.Fatalf("decision de otra operacion llego a SQL: %v", err)
	}
}
