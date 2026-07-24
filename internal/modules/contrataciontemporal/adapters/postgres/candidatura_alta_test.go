package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func solicitudCandidaturaPostgreSQLPrueba(
	t *testing.T,
) ports.SolicitudResolverCandidaturaAlta {
	t.Helper()
	ambitoV2 := selloHMACPrueba(claveAmbitoAltaPruebaV2, "c")
	ambitoV1 := selloHMACPrueba(claveAmbitoAltaPrueba, "d")
	huellaV2 := selloHMACPrueba(clavePeticionAltaPruebaV2, "c")
	huellaV1 := selloHMACPrueba(clavePeticionAltaPrueba, "b")
	ambitos, err := ports.NuevaColeccionSellosHMAC(
		ambitoV2,
		[]string{ambitoV1},
	)
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := ports.NuevaColeccionSellosHMAC(
		huellaV2,
		[]string{huellaV1},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudResolverCandidaturaAlta{
		AmbitosIdempotenciaHMAC: ambitos,
		HuellasPeticionHMAC:     huellas,
		OrganizacionRef:         "organizacion:diputacion-granada",
		ActorRef:                "actor:tecnica-rrhh-001",
		PerfilRef:               "perfil:tecnica-rrhh",
		Propuesta: ports.CandidaturaAlta{
			ReservaRef: "reserva:alta-candidata-001",
			Referencias: ports.ReferenciasAlta{
				ExpedienteRef: "expediente:ct-2026-0001",
				NumeroVisible: "2026/CT-0001",
				ReciboRef:     "recibo:alta-001",
			},
			AmbitoIdempotenciaHMAC: ambitoV2,
			HuellaPeticionHMAC:     huellaV2,
			OrganizacionRef:        "organizacion:diputacion-granada",
			ActorRef:               "actor:tecnica-rrhh-001",
			PerfilRef:              "perfil:tecnica-rrhh",
		},
	}
}

func filaCandidaturaPostgreSQLPrueba(
	resultado string,
	candidatura ports.CandidaturaAlta,
) pgx.Row {
	return filaPreparacionPrueba{valores: []any{
		resultado,
		candidatura.AmbitoIdempotenciaHMAC,
		candidatura.HuellaPeticionHMAC,
		candidatura.ReservaRef,
		candidatura.Referencias.ExpedienteRef,
		candidatura.Referencias.NumeroVisible,
		candidatura.Referencias.ReciboRef,
		candidatura.OrganizacionRef,
		candidatura.ActorRef,
		candidatura.PerfilRef,
	}}
}

func TestResolutorCandidaturaPostgreSQLEstabilizaYConfirma(t *testing.T) {
	solicitud := solicitudCandidaturaPostgreSQLPrueba(t)
	tx := &transaccionPreparacionPrueba{
		fila: filaCandidaturaPostgreSQLPrueba(
			"estabilizada",
			solicitud.Propuesta,
		),
	}
	resolutor, err := nuevoResolutorCandidaturaAltaPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
	)
	if err != nil {
		t.Fatal(err)
	}
	obtenida, err := resolutor.ResolverCandidaturaAlta(
		context.Background(),
		solicitud,
	)
	if err != nil || obtenida != solicitud.Propuesta ||
		tx.confirmaciones != 1 || tx.reversiones != 1 ||
		!tx.configurada ||
		!strings.Contains(tx.consulta, funcionResolverCandidaturaAltaV1) {
		t.Fatalf("candidatura no estabilizada: %#v / %v / %#v",
			obtenida, err, tx)
	}
}

func TestResolutorCandidaturaPostgreSQLRecuperaParRetenido(t *testing.T) {
	solicitud := solicitudCandidaturaPostgreSQLPrueba(t)
	ambitos, _ := solicitud.AmbitosIdempotenciaHMAC.Datos()
	huellas, _ := solicitud.HuellasPeticionHMAC.Datos()
	recuperada := solicitud.Propuesta
	recuperada.AmbitoIdempotenciaHMAC = ambitos.Retenidos[0].Valor
	recuperada.HuellaPeticionHMAC = huellas.Retenidos[0].Valor
	recuperada.ReservaRef = "reserva:alta-recuperada-001"
	recuperada.Referencias = ports.ReferenciasAlta{
		ExpedienteRef: "expediente:ct-2025-0042",
		NumeroVisible: "2025/CT-0042",
		ReciboRef:     "recibo:alta-recuperada-001",
	}
	tx := &transaccionPreparacionPrueba{
		fila: filaCandidaturaPostgreSQLPrueba("recuperada", recuperada),
	}
	resolutor, _ := nuevoResolutorCandidaturaAltaPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
	)
	obtenida, err := resolutor.ResolverCandidaturaAlta(
		context.Background(),
		solicitud,
	)
	if err != nil || obtenida != recuperada || tx.confirmaciones != 1 {
		t.Fatalf("par retenido no recuperado: %#v / %v", obtenida, err)
	}
}

func TestResolutorCandidaturaPostgreSQLRechazaReutilizacion(t *testing.T) {
	solicitud := solicitudCandidaturaPostgreSQLPrueba(t)
	tx := &transaccionPreparacionPrueba{
		fila: filaCandidaturaPostgreSQLPrueba(
			"idempotencia_reutilizada",
			solicitud.Propuesta,
		),
	}
	resolutor, _ := nuevoResolutorCandidaturaAltaPostgreSQL(
		&iniciadorPreparacionPrueba{tx: tx},
	)
	_, err := resolutor.ResolverCandidaturaAlta(
		context.Background(),
		solicitud,
	)
	if !errors.Is(err, ports.ErrClaveIdempotenciaUsada) ||
		tx.confirmaciones != 0 {
		t.Fatalf("reutilización aceptada: %v / %#v", err, tx)
	}
}

func TestResolutorCandidaturaPostgreSQLReintentaSerializacion(t *testing.T) {
	solicitud := solicitudCandidaturaPostgreSQLPrueba(t)
	tx := &transaccionPreparacionPrueba{
		fila: filaCandidaturaPostgreSQLPrueba(
			"estabilizada",
			solicitud.Propuesta,
		),
	}
	pool := &iniciadorPreparacionPrueba{
		errores: []error{
			&pgconn.PgError{Code: "40001"},
			&pgconn.PgError{Code: "40P01"},
			nil,
		},
		transacciones: []pgx.Tx{nil, nil, tx},
	}
	resolutor, _ := nuevoResolutorCandidaturaAltaPostgreSQL(pool)
	obtenida, err := resolutor.ResolverCandidaturaAlta(
		context.Background(),
		solicitud,
	)
	if err != nil || obtenida != solicitud.Propuesta || pool.inicios != 3 {
		t.Fatalf("reintentos incorrectos: %#v / %v / %d",
			obtenida, err, pool.inicios)
	}
}
