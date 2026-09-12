package postgres

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type poolCorreoLlamamientoPrueba struct {
	tx       pgx.Tx
	opciones pgx.TxOptions
	inicios  int
}

func (p *poolCorreoLlamamientoPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.inicios++
	p.opciones = o
	return p.tx, nil
}

type txCorreoLlamamientoPrueba struct {
	pgx.Tx
	fila      pgx.Row
	orden     []string
	errCommit error
	args      []any
}

func (t *txCorreoLlamamientoPrueba) Exec(_ context.Context, q string, a ...any) (pgconn.CommandTag, error) {
	t.orden = append(t.orden, "exec")
	if len(a) == 5 {
		t.args = append([]any(nil), a...)
	}
	return pgconn.CommandTag{}, nil
}
func (t *txCorreoLlamamientoPrueba) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	t.orden = append(t.orden, "query")
	return t.fila
}
func (t *txCorreoLlamamientoPrueba) Commit(context.Context) error {
	t.orden = append(t.orden, "commit")
	return t.errCommit
}
func (t *txCorreoLlamamientoPrueba) Rollback(context.Context) error {
	t.orden = append(t.orden, "rollback")
	return nil
}

type filaCorreoLlamamientoPrueba struct{ texto string }

func (f filaCorreoLlamamientoPrueba) Scan(dest ...any) error {
	*dest[0].(*string) = f.texto
	return nil
}

func solicitudCorreoLlamamientoPGPrueba() ports.SolicitudDespacharCorreoLlamamiento {
	return ports.SolicitudDespacharCorreoLlamamiento{OrganizacionRef: "org-001", ExpedienteRef: "exp-001", LlamamientoRef: "llamamiento-001", ComunicacionRef: "comunicacion-001", IntencionEnvioRef: "intencion-001"}
}
func reservaCorreoLlamamientoPGPrueba(t *testing.T) (ports.ReservaIntentoCorreoLlamamiento, ports.CapacidadFinalizacionIntentoCorreoLlamamiento) {
	t.Helper()
	s := solicitudCorreoLlamamientoPGPrueba()
	b, _ := json.Marshal(s)
	r := ports.ReservaIntentoCorreoLlamamiento{IntentoRef: "intento-001", MessageID: "<intento@vec.local>", FechaOrigen: time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC), SolicitudHuella: hex.EncodeToString(sha256Sum(b)), Estado: ports.CorreoLlamamientoIniciado}
	c, e := ports.NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(r, make([]byte, 32))
	if e == nil {
		t.Fatal("secreto nulo aceptado")
	}
	secreto := make([]byte, 32)
	secreto[0] = 1
	c, e = ports.NuevaCapacidadFinalizacionIntentoCorreoLlamamiento(r, secreto)
	if e != nil {
		t.Fatal(e)
	}
	return r, c
}
func sha256Sum(b []byte) []byte { h := sha256.Sum256(b); return h[:] }

func capacidadCorreoLlamamientoPGPrueba(t *testing.T, s ports.SolicitudDespacharCorreoLlamamiento) ports.CapacidadDespachoCorreoLlamamiento {
	t.Helper()
	recurso, err := ports.RecursoDespachoCorreoLlamamiento(s)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	vencimiento := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision-001", strings.Repeat("a", 64), strings.Repeat("a", 64), "contexto-001", strings.Repeat("a", 64), ports.AccionDespacharCorreoLlamamiento, s.IntencionEnvioRef, huella, ports.AudienciaDespachoCorreoLlamamientoV3, vencimiento.Add(-5*time.Second), vencimiento)
	if err != nil {
		t.Fatal(err)
	}
	clave := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{7}, ed25519.SeedSize))
	raiz, err := x509.MarshalPKIXPublicKey(clave.Public())
	if err != nil {
		t.Fatal(err)
	}
	material, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte{5}, vecports.TamanoMinimoCapacidadCanonicaV3), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("cose"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := ports.NuevaCapacidadDespachoCorreoLlamamiento(s, material, vencimiento.Add(-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	return capacidad
}

func TestRegistroCorreoLlamamientoFinalizaSoloTrasConfigurarYCommit(t *testing.T) {
	r, c := reservaCorreoLlamamientoPGPrueba(t)
	tx := &txCorreoLlamamientoPrueba{}
	pool := &poolCorreoLlamamientoPrueba{tx: tx}
	a := &RegistroIntentosCorreoLlamamientoPostgreSQL{pool: pool}
	if err := a.RegistrarResultadoIntentoCorreoLlamamiento(context.Background(), r, c, ports.CorreoLlamamientoAceptadoPorRelay, ports.PlantillaCorreoLlamamientoV1); err != nil {
		t.Fatal(err)
	}
	if pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite || !reflect.DeepEqual(tx.orden, []string{"exec", "exec", "commit", "rollback"}) || len(tx.args) != 5 {
		t.Fatalf("orden transaccional: %#v", tx.orden)
	}
}

func TestRegistroCorreoLlamamientoNoEmiteCapacidadSiCommitEsIncierto(t *testing.T) {
	s := solicitudCorreoLlamamientoPGPrueba()
	b, _ := json.Marshal(s)
	dto := reservaCorreoLlamamientoDTO{IntentoRef: "intento-001", MessageID: "<intento@vec.local>", FechaOrigen: time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC), SolicitudHuella: hex.EncodeToString(sha256Sum(b)), Estado: "iniciado", CapacidadFinalizacion: hex.EncodeToString(append([]byte{1}, make([]byte, 31)...))}
	contenido, _ := json.Marshal(dto)
	tx := &txCorreoLlamamientoPrueba{fila: filaCorreoLlamamientoPrueba{texto: string(contenido)}, errCommit: errors.New("commit incierto secreto-privado")}
	a := &RegistroIntentosCorreoLlamamientoPostgreSQL{pool: &poolCorreoLlamamientoPrueba{tx: tx}}
	reserva, finalizacion, err := a.ReservarIntentoCorreoLlamamiento(context.Background(), s, capacidadCorreoLlamamientoPGPrueba(t, s))
	if err == nil || strings.Contains(err.Error(), "secreto-privado") || reserva != (ports.ReservaIntentoCorreoLlamamiento{}) || !finalizacion.EsCero() || !reflect.DeepEqual(tx.orden, []string{"exec", "query", "commit", "rollback"}) {
		t.Fatalf("commit incierto emitió capacidad: reserva=%#v capacidad=%v orden=%#v err=%v", reserva, finalizacion, tx.orden, err)
	}
}

func dtoReservaCorreoLlamamientoPGPrueba(t *testing.T, s ports.SolicitudDespacharCorreoLlamamiento, yaReservado bool, secreto string) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(reservaCorreoLlamamientoDTO{
		IntentoRef:            "intento-001",
		MessageID:             "<intento@vec.local>",
		FechaOrigen:           time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC),
		SolicitudHuella:       hex.EncodeToString(sha256Sum(b)),
		Estado:                "iniciado",
		YaReservado:           yaReservado,
		CapacidadFinalizacion: secreto,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(contenido)
}

func TestRegistroCorreoLlamamientoReservaNuevaEntregaCapacidadTrasCommit(t *testing.T) {
	s := solicitudCorreoLlamamientoPGPrueba()
	secreto := hex.EncodeToString(append([]byte{1}, make([]byte, 31)...))
	tx := &txCorreoLlamamientoPrueba{fila: filaCorreoLlamamientoPrueba{texto: dtoReservaCorreoLlamamientoPGPrueba(t, s, false, secreto)}}
	a := &RegistroIntentosCorreoLlamamientoPostgreSQL{pool: &poolCorreoLlamamientoPrueba{tx: tx}}
	reserva, finalizacion, err := a.ReservarIntentoCorreoLlamamiento(context.Background(), s, capacidadCorreoLlamamientoPGPrueba(t, s))
	if err != nil || reserva.YaReservado || finalizacion.ValidarPara(reserva) != nil || !reflect.DeepEqual(tx.orden, []string{"exec", "query", "commit", "rollback"}) {
		t.Fatalf("reserva nueva no confirmada: reserva=%#v capacidad=%v orden=%#v err=%v", reserva, finalizacion, tx.orden, err)
	}
}

func TestRegistroCorreoLlamamientoReplayNoEntregaSecreto(t *testing.T) {
	s := solicitudCorreoLlamamientoPGPrueba()
	tx := &txCorreoLlamamientoPrueba{fila: filaCorreoLlamamientoPrueba{texto: dtoReservaCorreoLlamamientoPGPrueba(t, s, true, "")}}
	a := &RegistroIntentosCorreoLlamamientoPostgreSQL{pool: &poolCorreoLlamamientoPrueba{tx: tx}}
	reserva, finalizacion, err := a.ReservarIntentoCorreoLlamamiento(context.Background(), s, capacidadCorreoLlamamientoPGPrueba(t, s))
	if err != nil || !reserva.YaReservado || !finalizacion.EsCero() || !reflect.DeepEqual(tx.orden, []string{"exec", "query", "commit", "rollback"}) {
		t.Fatalf("replay filtró capacidad: reserva=%#v capacidad=%v orden=%#v err=%v", reserva, finalizacion, tx.orden, err)
	}
}

func TestRegistroCorreoLlamamientoRechazaReplayConSecretoYHuellaDivergente(t *testing.T) {
	s := solicitudCorreoLlamamientoPGPrueba()
	secreto := hex.EncodeToString(append([]byte{1}, make([]byte, 31)...))
	for nombre, contenido := range map[string]string{
		"secreto_en_replay": dtoReservaCorreoLlamamientoPGPrueba(t, s, true, secreto),
		"huella_divergente": `{"IntentoRef":"intento-001","MessageID":"<intento@vec.local>","FechaOrigen":"2026-09-12T10:00:00Z","SolicitudHuella":"` + strings.Repeat("b", 64) + `","Estado":"iniciado","YaReservado":false,"CapacidadFinalizacion":"` + secreto + `"}`,
	} {
		t.Run(nombre, func(t *testing.T) {
			tx := &txCorreoLlamamientoPrueba{fila: filaCorreoLlamamientoPrueba{texto: contenido}}
			a := &RegistroIntentosCorreoLlamamientoPostgreSQL{pool: &poolCorreoLlamamientoPrueba{tx: tx}}
			reserva, finalizacion, err := a.ReservarIntentoCorreoLlamamiento(context.Background(), s, capacidadCorreoLlamamientoPGPrueba(t, s))
			if !errors.Is(err, ports.ErrResultadoCorreoLlamamientoNoConfiable) || reserva != (ports.ReservaIntentoCorreoLlamamiento{}) || !finalizacion.EsCero() || !reflect.DeepEqual(tx.orden, []string{"exec", "query", "rollback"}) {
				t.Fatalf("respuesta SQL no fiable cruzó commit: reserva=%#v capacidad=%v orden=%#v err=%v", reserva, finalizacion, tx.orden, err)
			}
		})
	}
}
