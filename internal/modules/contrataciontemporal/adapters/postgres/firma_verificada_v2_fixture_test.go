package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	ctapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// El doble prueba transporte y cierre de TX; no acredita ACL ni firma real.
type txFirmaV2Prueba struct {
	pgx.Tx
	t                             *testing.T
	canonico                      string
	contenido                     []byte
	falloConsulta, falloCommit    error
	commits, rollbacks, consultas int
	configurada                   bool
	sql                           string
	argumentos                    int
	inspeccionar                  func([]any)
}

func (tx *txFirmaV2Prueba) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	if q != ajustesRegistroIncorporacionTXV2 || len(args) != 0 {
		tx.t.Fatal("ajustes fuera del contrato V2")
	}
	tx.configurada = true
	return pgconn.CommandTag{}, nil
}
func (tx *txFirmaV2Prueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	esperada, n := tx.sql, tx.argumentos
	if esperada == "" {
		esperada, n = consultarFirmasSQL172, 11
	}
	if !tx.configurada || q != esperada || len(args) != n || args[0] != tx.canonico {
		tx.t.Fatal("consulta fuera de la fachada nominal V2")
	}
	tx.consultas++
	if tx.inspeccionar != nil {
		tx.inspeccionar(args)
	}
	return filaFirmaV2Prueba{tx.contenido, tx.falloConsulta}
}
func (tx *txFirmaV2Prueba) Commit(context.Context) error   { tx.commits++; return tx.falloCommit }
func (tx *txFirmaV2Prueba) Rollback(context.Context) error { tx.rollbacks++; return nil }

type filaFirmaV2Prueba struct {
	contenido []byte
	fallo     error
}

func (f filaFirmaV2Prueba) Scan(dst ...any) error {
	if f.fallo != nil {
		return f.fallo
	}
	*dst[0].(*[]byte) = append([]byte(nil), f.contenido...)
	return nil
}

type poolFirmaV2Prueba struct {
	tx      *txFirmaV2Prueba
	inicios int
}

func (p *poolFirmaV2Prueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		p.tx.t.Fatal("TX V2 no serializable")
	}
	p.inicios++
	return p.tx, nil
}

func capacidadConsultaFirmaV2Prueba(t *testing.T, m ports.MaterialConsultaFirmasR5V2) ports.CapacidadConsultaFirmasR5V2 {
	t.Helper()
	r, err := ctapp.RecursoConsultaFirmasR5V2(m)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	ahora := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:ct172:lectura", h, h,
		"contexto:ct172:lectura", h, ports.AccionConsultarFirmasR5V2, m.ExpedienteRef, huella,
		ports.AudienciaConsultaFirmasR5V2, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		t.Fatal(err)
	}
	x, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen,
		[]byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return ports.TransportarMaterialConsultaFirmasR5V2(x)
}

func fixtureConsultaFirmaV2() (ports.MaterialConsultaFirmasR5V2, respuestaFirmasR5SQL172) {
	m := ports.MaterialConsultaFirmasR5V2{Via: ports.ViaFirmaCertificadoVEC, MaterialConsultaFirmasR5: ports.MaterialConsultaFirmasR5{
		OrganizacionRef: "organizacion:ct172:prueba", ExpedienteRef: "expediente:ct172:prueba", VersionExpediente: 7,
		Documento: "informe_definitivo", FirmantePrincipalCandidatoRef: "per_ct172_firmante_prueba",
		ClaveIdempotencia: "clave-ct172-prueba-000001", PasoOrden: 1, CatalogoHuella: strings.Repeat("a", 64)}}
	encontrado, no, si := true, false, true
	historia := uint64(1)
	observada := uint64(0)
	original, firmado, evidencia := strings.Repeat("b", 64), strings.Repeat("c", 64), `[{"orden":1}]`
	certificado := strings.Repeat("d", 64)
	refOriginal, version := "documento:ct172:original", uint64(1)
	base := firmaExternaSQL170{firmaSQL118: firmaSQL118{FirmaRef: "firma:ct172:prueba", ReciboRef: "recibo:ct172:prueba",
		Documento: m.Documento, Secuencia: 1, ExpedienteVersion: 7, CatalogoRef: "catalogo:ct172:prueba",
		CatalogoHuella: m.CatalogoHuella, PasoRef: "paso:ct172:primero", PasoOrden: 1, Resultado: "firmado",
		OriginalHuella: &original, FirmadoHuella: &firmado, RegistradaEn: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		ClaveIdempotencia: m.ClaveIdempotencia}, Via: &m.Via, FirmantePrincipalAcreditado: true,
		HistoriaRevision: &observada, HistoriaHuella: &original, OriginalRef: &refOriginal, OriginalVersion: &version}
	h := sha256.Sum256([]byte(evidencia))
	revision := firmaRevisionPDFSQL172{firmaExternaSQL170: base, FirmanteRef: "ref:" + certificado,
		CertificadoHuella: certificado, EntradaDocumentoRef: refOriginal, EntradaDocumentoVersion: version,
		EntradaDocumentoLongitud: 100, EntradaDocumentoHuella: original, OrdenFirmaPDF: 1,
		ByteRange: []uint64{0, 120, 180, 20}, RevisionHuellaSHA256: firmado,
		ContenidoFirmadoHuellaSHA256: original, RevisionLongitud: 200,
		EvidenciaFirmasCanonica: evidencia, EvidenciaFirmasHuellaSHA256: hex.EncodeToString(h[:])}
	return m, respuestaFirmasR5SQL172{respuestaFirmasR5SQL170: respuestaFirmasR5SQL170{Encontrado: &encontrado,
		ExpedienteRef: m.ExpedienteRef, Firmas: []firmaExternaSQL170{base}, HistoriaRevision: &historia,
		HistoriaHuella: original, CoincideFirmanteEnOtroPaso: &no, HistoriaSeparacionAcreditada: &si},
		RevisionesPDF: []firmaRevisionPDFSQL172{revision}}
}
