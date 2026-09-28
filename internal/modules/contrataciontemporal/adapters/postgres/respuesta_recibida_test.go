package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type proveedorRespuestaRecibidaPrueba func(context.Context, ports.SolicitudRegistrarRespuestaRecibida) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)

func (p proveedorRespuestaRecibidaPrueba) AutorizarRegistroRespuestaRecibida(ctx context.Context, s ports.SolicitudRegistrarRespuestaRecibida) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return p(ctx, s)
}

type poolRespuestaRecibidaPrueba struct {
	txs     []pgx.Tx
	inicios int
}

func (p *poolRespuestaRecibidaPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	if opciones.IsoLevel != pgx.Serializable || opciones.AccessMode != pgx.ReadWrite || p.inicios >= len(p.txs) {
		return nil, errors.New("transacción inesperada")
	}
	tx := p.txs[p.inicios]
	p.inicios++
	return tx, nil
}

func atestacionRespuestaRecibidaPrueba(t *testing.T, s ports.SolicitudRegistrarRespuestaRecibida, numero int) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	recurso, err := RecursoRegistroRespuestaRecibida(s)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	ahora := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:respuesta:"+strconv.Itoa(numero), h, h, "contexto:respuesta", h,
		AccionRegistroRespuestaRecibida, s.ExpedienteRef, huella,
		AudienciaRegistroComunicacionLlamamiento, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		t.Fatal(err)
	}
	a, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		[]byte(strings.Repeat("x", 512)), resumen, []byte("{}"), []byte("{}"), []byte("{}"),
		1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestRespuestaRecibidaPGReintentaSoloSerializacionConAutorizacionFresca(t *testing.T) {
	s := ports.SolicitudRegistrarRespuestaRecibida{OrganizacionRef: "org:sintetica", ExpedienteRef: "exp:sintetico",
		LlamamientoRef: "llamamiento:sintetico", ComunicacionRef: "comunicacion:sintetica",
		VersionComunicacionEsperada: 2, Respuesta: ports.RespuestaLlamamientoAceptada,
		CorreoRef: "correo:sintetico", CorreoSHA256: strings.Repeat("a", 64),
		RecibidaEn: time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)}
	original := s
	original.ClaveIdempotencia = "e53cb792-4c62-4daf-8c80-d5d18521748a"
	recibo := ports.RespuestaRecibidaRegistrada{Solicitud: original, JustificanteRef: "justificante:sintetico",
		ReciboRef: "recibo:sintetico", AuditoriaRef: "auditoria:sintetica",
		RegistradaEn: s.RecibidaEn.Add(time.Minute), Estado: ports.EstadoRespuestaRecibidaReplay}
	b, _ := json.Marshal(recibo)
	fallida := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{
		err: &pgconn.PgError{Code: "40001", Message: "sin datos"}}}
	exitosa := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{valores: []any{string(b)}}}
	pool := &poolRespuestaRecibidaPrueba{txs: []pgx.Tx{fallida, exitosa}}
	autorizaciones := 0
	proveedor := proveedorRespuestaRecibidaPrueba(func(_ context.Context, recibida ports.SolicitudRegistrarRespuestaRecibida) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		if recibida != s {
			t.Fatal("material de retry sustituido")
		}
		autorizaciones++
		return atestacionRespuestaRecibidaPrueba(t, s, autorizaciones), nil
	})
	registro := &RegistroRespuestasRecibidasPostgreSQL{pool: pool, proveedor: proveedor}
	resultado, err := registro.RegistrarRespuestaRecibida(context.Background(), s)
	if err != nil || resultado != recibo || autorizaciones != 2 || pool.inicios != 2 ||
		fallida.reversiones != 1 || exitosa.confirmaciones != 1 ||
		len(exitosa.consultas) != 1 || !strings.Contains(exitosa.consultas[0], "registrar_respuesta_recibida_rrhh_v2") {
		t.Fatalf("retry/frescura falló: %v autorizaciones=%d inicios=%d", err, autorizaciones, pool.inicios)
	}
}

func TestRespuestaRecibidaPGNoReintentaCommitIncierto(t *testing.T) {
	s := ports.SolicitudRegistrarRespuestaRecibida{OrganizacionRef: "org:sintetica", ExpedienteRef: "exp:sintetico",
		LlamamientoRef: "llamamiento:sintetico", ComunicacionRef: "comunicacion:sintetica",
		VersionComunicacionEsperada: 2, Respuesta: ports.RespuestaLlamamientoAceptada,
		CorreoRef: "correo:sintetico", CorreoSHA256: strings.Repeat("a", 64),
		RecibidaEn: time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)}
	recibo := ports.RespuestaRecibidaRegistrada{Solicitud: s, JustificanteRef: "justificante:sintetico",
		ReciboRef: "recibo:sintetico", AuditoriaRef: "auditoria:sintetica",
		RegistradaEn: s.RecibidaEn.Add(time.Minute), Estado: ports.EstadoRespuestaRecibidaRegistrada}
	recibo.Solicitud.ClaveIdempotencia = "e53cb792-4c62-4daf-8c80-d5d18521748a"
	b, _ := json.Marshal(recibo)
	tx := &transaccionEjecucionSeleccionO6Prueba{fila: filaEjecucionSeleccionO6Prueba{valores: []any{string(b)}},
		errCommit: errors.New("commit incierto sintético")}
	pool := &poolRespuestaRecibidaPrueba{txs: []pgx.Tx{tx}}
	llamadas := 0
	p := proveedorRespuestaRecibidaPrueba(func(_ context.Context, _ ports.SolicitudRegistrarRespuestaRecibida) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		llamadas++
		return atestacionRespuestaRecibidaPrueba(t, s, llamadas), nil
	})
	resultado, err := (&RegistroRespuestasRecibidasPostgreSQL{pool: pool, proveedor: p}).RegistrarRespuestaRecibida(context.Background(), s)
	if !errors.Is(err, ports.ErrRespuestaRecibidaNoDisponible) || resultado != (ports.RespuestaRecibidaRegistrada{}) || llamadas != 1 || pool.inicios != 1 {
		t.Fatalf("commit incierto reintentado o revelado: %v", err)
	}
}

func TestRespuestaRecibidaMaterialLigaDeclaracionCompleta(t *testing.T) {
	s := ports.SolicitudRegistrarRespuestaRecibida{ClaveIdempotencia: "11111111-1111-4111-8111-111111111111", OrganizacionRef: "org:sintetica", ExpedienteRef: "exp:sintetico", LlamamientoRef: "llamamiento:sintetico", ComunicacionRef: "comunicacion:sintetica", VersionComunicacionEsperada: 2, Respuesta: ports.RespuestaLlamamientoAceptada, CorreoRef: "correo:sintetico", CorreoSHA256: strings.Repeat("a", 64), RecibidaEn: time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)}
	r, err := RecursoRegistroRespuestaRecibida(s)
	if err != nil {
		t.Fatal(err)
	}
	h, _ := r.HuellaContextoAutorizacionSHA256()
	for nombre, mutar := range map[string]func(*ports.SolicitudRegistrarRespuestaRecibida){
		"clave": func(s *ports.SolicitudRegistrarRespuestaRecibida) {
			s.ClaveIdempotencia = "21111111-1111-4111-8111-111111111111"
		},
		"organizacion": func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.OrganizacionRef += "b" },
		"expediente":   func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.ExpedienteRef += "b" },
		"llamamiento":  func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.LlamamientoRef += "b" },
		"comunicacion": func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.ComunicacionRef += "b" },
		"respuesta":    func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.Respuesta = ports.RespuestaLlamamientoRenunciada },
		"correo":       func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.CorreoRef += "b" },
		"huella":       func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.CorreoSHA256 = strings.Repeat("b", 64) },
		"fecha":        func(s *ports.SolicitudRegistrarRespuestaRecibida) { s.RecibidaEn = s.RecibidaEn.Add(time.Microsecond) },
	} {
		t.Run(nombre, func(t *testing.T) {
			otra := s
			mutar(&otra)
			r, err := RecursoRegistroRespuestaRecibida(otra)
			if err != nil {
				t.Fatal(err)
			}
			nueva, _ := r.HuellaContextoAutorizacionSHA256()
			if nueva == h {
				t.Fatal("campo no ligado")
			}
		})
	}
	s.VersionComunicacionEsperada = 3
	if _, err := RecursoRegistroRespuestaRecibida(s); err == nil {
		t.Fatal("otra versión admitida")
	}
}

func TestRespuestaRecibidaErroresNoExponenDetallesSQL(t *testing.T) {
	for codigo, esperado := range map[string]error{"P0560": ports.ErrSolicitudRespuestaRecibidaInvalida, "P0561": ports.ErrClaveRespuestaRecibidaUsada, "P0562": ports.ErrVersionRespuestaRecibidaEnConflicto, "P0563": ports.ErrOperacionRespuestaRecibidaDenegada, "42501": ports.ErrOperacionRespuestaRecibidaDenegada, "40001": ports.ErrRespuestaRecibidaNoDisponible, "08006": ports.ErrRespuestaRecibidaNoDisponible} {
		if err := normalizarErrorRespuestaRecibida(context.Background(), &pgconn.PgError{Code: codigo, Message: "detalle privado"}); !errors.Is(err, esperado) || strings.Contains(err.Error(), "privado") {
			t.Fatal(codigo, err)
		}
	}
}
