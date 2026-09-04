package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDTOConfirmacionAnalisisDerivaHuellaYMotivoSinDuplicarlos(t *testing.T) {
	t.Parallel()
	contenido, err := json.Marshal(operacionConfirmarAnalisisV1{
		Politica: politicaConfirmarAnalisisV1{
			MotivoRectificacionClave: ports.ValorMotivoRectificacionNoAplica,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var dto map[string]any
	if err := json.Unmarshal(contenido, &dto); err != nil {
		t.Fatal(err)
	}
	politica, esObjeto := dto["politica"].(map[string]any)
	_, duplicaHuella := dto[ports.AtributoAnalisisDerivadoHuella]
	if duplicaHuella || !esObjeto ||
		politica["motivo_rectificacion_clave"] !=
			ports.ValorMotivoRectificacionNoAplica {
		t.Fatalf("contrato PostgreSQL ambiguo: %#v", dto)
	}
}

func TestNormalizarInstantePostgreSQLProduceUTCCanonico(t *testing.T) {
	t.Parallel()
	zonaLocalUTC := time.FixedZone("zona-local-utc-prueba", 0)
	entrada := time.Date(
		2026, time.July, 25, 0, 30, 7, 123456789, zonaLocalUTC,
	)
	resultado := normalizarInstantePostgreSQL(entrada)
	if !domain.InstanteUTCCanonico(resultado) ||
		resultado.Location() != time.UTC ||
		resultado.Nanosecond() != 123456000 {
		t.Fatalf("instante PostgreSQL no canónico: %#v", resultado)
	}
}

func TestDecodificarReciboConfirmacionAnalisisNormalizaInstantePostgreSQL(
	t *testing.T,
) {
	t.Parallel()
	recibo, err := decodificarReciboConfirmacionAnalisis(
		`{"confirmada_en":"2026-09-04T13:50:52.123456789+00:00"}`,
	)
	if err != nil || !domain.InstanteUTCCanonico(recibo.ConfirmadaEn) ||
		recibo.ConfirmadaEn.Location() != time.UTC ||
		recibo.ConfirmadaEn.Nanosecond() != 123456000 {
		t.Fatalf("instante de recibo de análisis no normalizado: %#v, %v", recibo, err)
	}
}

func TestTransaccionAnalisisPostgreSQLUsaFronteraDominioV3(t *testing.T) {
	t.Parallel()
	const esperada = "vec_contratacion_temporal.confirmar_operacion_analisis_v3"
	if funcionConfirmarAnalisis != esperada {
		t.Fatalf(
			"frontera PostgreSQL sin invariantes V3: %q",
			funcionConfirmarAnalisis,
		)
	}
}

func TestTransaccionAnalisisPostgreSQLFallaCerradaSinPool(t *testing.T) {
	t.Parallel()
	adaptador, err := nuevaTransaccionOperacionesAnalisisPostgreSQL(nil)
	if adaptador != nil ||
		!errors.Is(
			err,
			ports.ErrPersistenciaOperacionAnalisisNoDisponible,
		) {
		t.Fatalf("constructor no falló cerrado: adaptador=%v err=%v", adaptador, err)
	}
}

func TestTransaccionAnalisisRechazaOrdenVaciaAntesDePostgreSQL(t *testing.T) {
	t.Parallel()
	iniciador := &iniciadorPreparacionPrueba{}
	adaptador, err := nuevaTransaccionOperacionesAnalisisPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := adaptador.ConfirmarOperacionAnalisis(
		context.Background(),
		ports.OrdenConfirmarOperacionAnalisis{},
	)
	if recibo != (ports.ReciboOperacionAnalisis{}) ||
		!errors.Is(err, ports.ErrOrdenOperacionAnalisisInvalida) ||
		iniciador.inicios != 0 {
		t.Fatalf(
			"orden vacía cruzó la frontera: recibo=%#v inicios=%d err=%v",
			recibo,
			iniciador.inicios,
			err,
		)
	}
}

func TestErrorConfirmacionAnalisisClasificaConflictoUnico(t *testing.T) {
	t.Parallel()
	err := errorConfirmacionAnalisis(&pgconn.PgError{Code: "23505"})
	if !errors.Is(err, ports.ErrConjuntoFuentesAnalisisYaConsumido) {
		t.Fatalf("23505 no se clasificó como conflicto: %v", err)
	}
	err = errorConfirmacionAnalisis(&pgconn.PgError{Code: "42501"})
	if !errors.Is(
		err,
		ports.ErrPersistenciaOperacionAnalisisNoDisponible,
	) {
		t.Fatalf("fallo de autoridad se expuso: %v", err)
	}
}
