package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type poolConsultaR5Prueba struct {
	tx      *txConsultaR5Prueba
	inicios int
}

func (p *poolConsultaR5Prueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.inicios++
	if o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		p.tx.t.Fatal("la consulta R5 requiere SERIALIZABLE y consumo V3")
	}
	return p.tx, nil
}

type txConsultaR5Prueba struct {
	pgx.Tx
	t                     *testing.T
	canonico              string
	contenido             []byte
	commits, rollbacks    int
	configurada, consulto bool
}

func (tx *txConsultaR5Prueba) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	if q != ajustesRegistroIncorporacionTXV2 || len(args) != 0 {
		tx.t.Fatal("ajustes de consulta R5 alterados")
	}
	tx.configurada = true
	return pgconn.CommandTag{}, nil
}
func (tx *txConsultaR5Prueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	if !tx.configurada || q != consultarFirmasSQL170 || len(args) != 11 || args[0] != tx.canonico {
		tx.t.Fatal("lector R5 no usa fachada atestada y material exacto")
	}
	tx.consulto = true
	return filaConsultaR5Prueba{contenido: tx.contenido}
}
func (tx *txConsultaR5Prueba) Commit(context.Context) error   { tx.commits++; return nil }
func (tx *txConsultaR5Prueba) Rollback(context.Context) error { tx.rollbacks++; return nil }

type filaConsultaR5Prueba struct{ contenido []byte }

func (f filaConsultaR5Prueba) Scan(dst ...any) error {
	*dst[0].(*[]byte) = append([]byte(nil), f.contenido...)
	return nil
}

func capacidadConsultaR5Prueba(t *testing.T, m ports.MaterialConsultaFirmasR5) ports.CapacidadConsultaFirmasR5 {
	t.Helper()
	r, err := ctapp.RecursoConsultaFirmasR5(m)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	ahora := time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC)
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:ct170:lectura", h, h,
		"contexto:ct170:lectura", h, ports.AccionConsultarFirmasR5, m.ExpedienteRef, huella,
		ports.AudienciaConsultaFirmasR5V3, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		t.Fatal(err)
	}
	x, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen,
		[]byte("{}"), []byte("{}"), []byte("{}"), 1, 1,
		[]byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return ports.TransportarMaterialConsultaFirmasR5(x)
}

func TestConsultaR5ConsumeAntesDeDevolverProyeccion(t *testing.T) {
	m := ports.MaterialConsultaFirmasR5{OrganizacionRef: "organizacion:ct170:prueba",
		ExpedienteRef: "expediente:ct170:prueba", VersionExpediente: 7, Documento: "informe_definitivo",
		FirmantePrincipalCandidatoRef: "per_ct170_firmante_prueba", ClaveIdempotencia: "clave-ct170-prueba-000006",
		PasoOrden: 1, CatalogoHuella: strings.Repeat("a", 64)}
	canonico, err := m.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	via := ports.ViaFirmaExternaPortafirmas
	original := "documento:ct170:original"
	declarada := "referencia declarada"
	fecha := "2026-10-02T17:00:00Z"
	version := uint64(1)
	observada := uint64(0)
	observadaHuella := strings.Repeat("b", 64)
	cabezaRevision := uint64(1)
	cabezaHuella := strings.Repeat("c", 64)
	firma := firmaExternaSQL170{firmaSQL118: firmaSQL118{
		FirmaRef:  "firma-ct:00000000-0000-4000-8000-000000000001",
		ReciboRef: "recibo-firma-ct:00000000-0000-4000-8000-000000000001",
		Documento: m.Documento, Secuencia: 1, ExpedienteVersion: 7,
		ClaveIdempotencia: "clave-ct170-prueba-000001",
		Resultado:         "firmado", RegistradaEn: time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC),
	}, Via: &via, FirmantePrincipalAcreditado: true, CoincideFirmanteCandidato: true,
		HistoriaRevision: &observada, HistoriaHuella: &observadaHuella, OriginalRef: &original,
		OriginalVersion: &version, ReferenciaPortafirmasDeclarada: &declarada,
		FechaPortafirmasDeclarada: &fecha}
	encontrado := true
	no := false
	si := true
	contenido, err := json.Marshal(respuestaFirmasR5SQL170{Encontrado: &encontrado,
		ExpedienteRef: m.ExpedienteRef, Firmas: []firmaExternaSQL170{firma},
		HistoriaRevision: &cabezaRevision, HistoriaHuella: cabezaHuella,
		CoincideFirmanteEnOtroPaso: &no, HistoriaSeparacionAcreditada: &si})
	if err != nil {
		t.Fatal(err)
	}
	tx := &txConsultaR5Prueba{t: t, canonico: string(canonico), contenido: contenido}
	p := &poolConsultaR5Prueba{tx: tx}
	r := &RegistroFirmasExternasPostgreSQL{pool: p}
	c := capacidadConsultaR5Prueba(t, m)
	lectura, err := r.ConsultarFirmasAutorizadas(context.Background(), m, c)
	if err != nil || len(lectura.Firmas) != 1 || lectura.Firmas[0].Via != via ||
		!lectura.Firmas[0].FirmantePrincipalAcreditado || !lectura.Firmas[0].CoincideFirmanteCandidato ||
		lectura.Firmas[0].OriginalRef != original || lectura.Firmas[0].HistoriaRevision != observada ||
		lectura.Firmas[0].HistoriaHuella != observadaHuella ||
		lectura.HistoriaRevision != cabezaRevision || lectura.HistoriaHuella != cabezaHuella ||
		lectura.CoincideFirmanteEnOtroPaso || !lectura.HistoriaSeparacionAcreditada ||
		lectura.Firmas[0].ReferenciaPortafirmasDeclarada != declarada || tx.commits != 1 || tx.rollbacks != 0 {
		t.Fatalf("proyección o consumo: err=%v filas=%d commit=%d rollback=%d", err, len(lectura.Firmas), tx.commits, tx.rollbacks)
	}
	mReplay := m
	mReplay.ClaveIdempotencia = firma.ClaveIdempotencia
	canonReplay, err := mReplay.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	contenidoReplay, err := json.Marshal(respuestaFirmasR5SQL170{Encontrado: &encontrado,
		ExpedienteRef: m.ExpedienteRef, Firmas: []firmaExternaSQL170{firma},
		HistoriaRevision: &observada, HistoriaHuella: observadaHuella,
		CoincideFirmanteEnOtroPaso: &no, HistoriaSeparacionAcreditada: &no})
	if err != nil {
		t.Fatal(err)
	}
	txReplay := &txConsultaR5Prueba{t: t, canonico: string(canonReplay), contenido: contenidoReplay}
	rReplay := &RegistroFirmasExternasPostgreSQL{pool: &poolConsultaR5Prueba{tx: txReplay}}
	lecturaReplay, err := rReplay.ConsultarFirmasAutorizadas(context.Background(), mReplay,
		capacidadConsultaR5Prueba(t, mReplay))
	if err != nil || len(lecturaReplay.Firmas) != 1 || lecturaReplay.HistoriaRevision != observada ||
		lecturaReplay.HistoriaHuella != observadaHuella || lecturaReplay.CoincideFirmanteEnOtroPaso ||
		lecturaReplay.HistoriaSeparacionAcreditada || txReplay.commits != 1 {
		t.Fatalf("replay con cabeza original: err=%v filas=%d commit=%d", err, len(lecturaReplay.Firmas), txReplay.commits)
	}
	tx2 := &txConsultaR5Prueba{t: t, canonico: string(canonico), contenido: contenido}
	r2 := &RegistroFirmasExternasPostgreSQL{pool: &poolConsultaR5Prueba{tx: tx2}}
	if _, err := r2.ConsultarFirmasAutorizadas(context.Background(), m, ports.CapacidadConsultaFirmasR5{}); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || tx2.commits != 0 {
		t.Fatalf("capacidad ausente: err=%v commits=%d", err, tx2.commits)
	}
}
