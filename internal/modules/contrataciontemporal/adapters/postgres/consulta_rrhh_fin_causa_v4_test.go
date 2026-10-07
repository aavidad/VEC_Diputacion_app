package postgres

import (
	"bytes"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDetalleRRHHFinPorCausaCanonV4Roundtrip(t *testing.T) {
	ahora := instanteCanonRRHHPostgreSQLPrueba()
	resumen := resumenCanonRRHHPostgreSQLPrueba(1, ahora)
	resumen.CreadoEn = ahora
	solicitud := ports.SolicitudOperativaRRHH{
		GrupoSubgrupo: "C2", MotivoClave: "sustitucion",
		PeriodoInicio:   time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
		PeriodoCausaFin: "reincorporacion_titular",
	}
	hitos := []ports.HitoExpedienteRRHH{{
		Secuencia: 1, VersionExpediente: 1, AccionClave: "alta",
		RealizadaEn: ahora, FaseDestino: "solicitud",
		EstadoOrigen: domain.EstadoPendiente, EstadoDestino: domain.EstadoEnCurso,
	}}
	entrada, err := ports.NuevaEntradaDetalleExpedienteRRHHMinimizadaV3(
		resumen, solicitud, nil, ports.ReferenciaHitoAnalisisRRHH{},
		nil, ports.ReferenciaHitoCoberturaRRHH{}, nil, ports.ReferenciaHitoAsignacionRRHH{},
		nil, ports.ReferenciaHitoFiscalizacionRRHH{}, ports.ReferenciaHitoSubsanacionFiscalizacionRRHH{}, hitos,
	)
	if err != nil {
		t.Fatal(err)
	}
	exportacion, err := entrada.ExportarContenidoCanonicoParaSQL(ahora)
	if err != nil {
		t.Fatal(err)
	}
	canon := exportacion.BytesCanonicos()
	if !bytes.HasPrefix(canon, []byte(cabeceraContenidoDetalleRRHHPostgreSQLV4)) {
		t.Fatal("falta cabecera V4")
	}
	decodificado, err := decodificarContenidoDetalleRRHHPostgreSQL(canon)
	if err != nil {
		t.Fatal(err)
	}
	reexportacion, err := decodificado.entrada.ExportarContenidoCanonicoParaSQL(ahora)
	if err != nil || !bytes.Equal(canon, reexportacion.BytesCanonicos()) {
		t.Fatal("canon V4 no se recuperó byte a byte")
	}
}
