package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	application "vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type poolConsultaFirmasPrueba struct {
	tx       *txConsultaFirmasPrueba
	inicios  int
	opciones pgx.TxOptions
	err      error
}

func (p *poolConsultaFirmasPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.inicios++
	p.opciones = o
	return p.tx, p.err
}

type txConsultaFirmasPrueba struct {
	pgx.Tx
	t                                              *testing.T
	contenido                                      []byte
	falloConsulta, falloCommit                     error
	consultas, configuraciones, commits, rollbacks int
	material                                       string
	parametros                                     []any
}

func (tx *txConsultaFirmasPrueba) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	tx.configuraciones++
	if q != ajustesRegistroIncorporacionTXV2 || len(args) != 0 {
		tx.t.Fatal("ajustes de TX alterados")
	}
	return pgconn.CommandTag{}, nil
}
func (tx *txConsultaFirmasPrueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	tx.consultas++
	if q != consultarFirmasAtestadasSQL || len(args) != 11 || args[0] != tx.material || !reflect.DeepEqual(args[1:], tx.parametros) {
		tx.t.Fatal("consulta o parámetros fuera del contrato CT")
	}
	return filaConsultaFirmasPrueba{tx}
}
func (tx *txConsultaFirmasPrueba) Commit(context.Context) error   { tx.commits++; return tx.falloCommit }
func (tx *txConsultaFirmasPrueba) Rollback(context.Context) error { tx.rollbacks++; return nil }

type filaConsultaFirmasPrueba struct{ tx *txConsultaFirmasPrueba }

func (f filaConsultaFirmasPrueba) Scan(dest ...any) error {
	if f.tx.falloConsulta != nil {
		return f.tx.falloConsulta
	}
	*dest[0].(*[]byte) = append([]byte(nil), f.tx.contenido...)
	return nil
}

func capacidadConsultaFirmasPrueba(t *testing.T, m ports.MaterialConsultaFirmasDocumento) ports.CapacidadConsultaFirmasDocumento {
	t.Helper()
	r, err := application.RecursoConsultaFirmasDocumento(m)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	ahora := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:ct152", h, h, "contexto:ct152", h,
		ports.AccionConsultarFirmasDocumento, m.ExpedienteRef, huella, ports.AudienciaConsultaFirmasDocumentoV3, ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)).Public())
	if err != nil {
		t.Fatal(err)
	}
	a, err := vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen,
		[]byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("unidad"), []byte("unidad"), []byte("unidad"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return ports.TransportarMaterialConsultaFirmasDocumento(a)
}
func prepararConsultaFirmasPrueba(t *testing.T, contenido string) (*LectorFirmasDocumentoAutorizadasPostgreSQL, *poolConsultaFirmasPrueba, ports.MaterialConsultaFirmasDocumento, ports.CapacidadConsultaFirmasDocumento) {
	t.Helper()
	m := ports.MaterialConsultaFirmasDocumento{OrganizacionRef: "organizacion:ct152", ExpedienteRef: "expediente:ct152"}
	c := capacidadConsultaFirmasPrueba(t, m)
	canonico, err := application.CanonicoConsultaFirmasDocumento(m)
	if err != nil {
		t.Fatal(err)
	}
	parametros, err := exportacionParametrosRegistroV2(c.ExportarMaterialParaConsumidor(), true)
	if err != nil {
		t.Fatal(err)
	}
	tx := &txConsultaFirmasPrueba{t: t, contenido: []byte(contenido), material: string(canonico), parametros: parametros}
	p := &poolConsultaFirmasPrueba{tx: tx}
	return &LectorFirmasDocumentoAutorizadasPostgreSQL{pool: p}, p, m, c
}

func TestConsultaFirmasAtestadasConfirmaVacioYAusencia(t *testing.T) {
	for _, tc := range []struct {
		nombre     string
		encontrado bool
		esperado   error
	}{
		{"existente_sin_firmas", true, nil},
		{"ausente", false, ports.ErrExpedienteConsultaFirmasNoEncontrado},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			b, _ := json.Marshal(struct {
				Encontrado    bool
				ExpedienteRef string
				Firmas        []any
			}{tc.encontrado, "expediente:ct152", []any{}})
			l, p, m, c := prepararConsultaFirmasPrueba(t, string(b))
			firmas, err := l.ConsultarFirmasAutorizadas(context.Background(), m, c)
			if !errors.Is(err, tc.esperado) || (tc.esperado == nil && err != nil) ||
				(tc.encontrado && firmas == nil) || (!tc.encontrado && firmas != nil) ||
				p.opciones.IsoLevel != pgx.Serializable || p.opciones.AccessMode != pgx.ReadWrite ||
				p.tx.configuraciones != 1 || p.tx.consultas != 1 || p.tx.commits != 1 || p.tx.rollbacks != 0 {
				t.Fatalf("resultado=%v error=%v commits=%d rollback=%d", firmas, err, p.tx.commits, p.tx.rollbacks)
			}
		})
	}
}

func TestConsultaFirmasAtestadasDevuelveHistoriaCustodiada(t *testing.T) {
	original := strings.Repeat("a", 64)
	firmada := strings.Repeat("b", 64)
	sello := "valido"
	custodia := "documento:ct152"
	version := uint64(2)
	f := firmaSQL118{
		FirmaRef: "firma:ct152", ReciboRef: "recibo:ct152", Documento: "resolucion", Secuencia: 1,
		ExpedienteVersion: 7, CatalogoRef: "catalogo:ct152", CatalogoHuella: strings.Repeat("c", 64),
		PasoRef: "circuito:1:p1", PasoOrden: 1, Resultado: "firmado", OriginalHuella: &original,
		FirmadoHuella: &firmada, SelloTiempoEstado: &sello, RegistradaEn: time.Date(2026, 9, 30, 10, 1, 0, 0, time.UTC),
		ClaveIdempotencia: "clave-firma-ct152-0001", DocumentoCustodia: &custodia, VersionCustodia: &version,
	}
	b, err := json.Marshal(struct {
		Encontrado    bool
		ExpedienteRef string
		Firmas        []firmaSQL118
	}{true, "expediente:ct152", []firmaSQL118{f}})
	if err != nil {
		t.Fatal(err)
	}
	l, p, m, c := prepararConsultaFirmasPrueba(t, string(b))
	filas, err := l.ConsultarFirmasAutorizadas(context.Background(), m, c)
	if err != nil || len(filas) != 1 || filas[0].DocumentoCustodiaRef != custodia ||
		filas[0].DocumentoCustodiaVersion != version || filas[0].ClaveIdempotencia != f.ClaveIdempotencia ||
		filas[0].FirmadoHuella != firmada || p.tx.commits != 1 || p.tx.rollbacks != 0 {
		t.Fatalf("historia o confirmación: error=%v filas=%d commits=%d", err, len(filas), p.tx.commits)
	}
}

func TestConsultaFirmasAtestadasDeniegaAntesDeTransaccion(t *testing.T) {
	l, p, m, _ := prepararConsultaFirmasPrueba(t, "")
	if _, err := l.ConsultarFirmasAutorizadas(context.Background(), m, ports.CapacidadConsultaFirmasDocumento{}); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || p.inicios != 0 {
		t.Fatalf("capacidad vacía: %v, inicios=%d", err, p.inicios)
	}
	m.ExpedienteRef = "expediente:ajeno"
	if _, err := l.ConsultarFirmasAutorizadas(context.Background(), m, capacidadConsultaFirmasPrueba(t, ports.MaterialConsultaFirmasDocumento{OrganizacionRef: m.OrganizacionRef, ExpedienteRef: "expediente:ct152"})); !errors.Is(err, ports.ErrFirmaDocumentoDenegada) || p.inicios != 0 {
		t.Fatalf("capacidad de otro expediente: %v, inicios=%d", err, p.inicios)
	}
}

func TestConsultaFirmasAtestadasRechazaRespuestaIncoherenteAntesDeCommit(t *testing.T) {
	casos := []string{
		`{"Encontrado":true,"ExpedienteRef":"expediente:ajeno","Firmas":[]}`,
		`{"Encontrado":false,"ExpedienteRef":"expediente:ct152","Firmas":[{}]}`,
		`{"Encontrado":true,"ExpedienteRef":"expediente:ct152","Firmas":null}`,
		`{"Encontrado":true,"ExpedienteRef":"expediente:ct152","Firmas":[{"FirmaRef":"firma:ct152"}]}`,
		`{"Encontrado":true,"ExpedienteRef":"expediente:ct152","Firmas":[],"PersonaRef":"persona:ajena"}`,
	}
	for i, salida := range casos {
		l, p, m, c := prepararConsultaFirmasPrueba(t, salida)
		firmas, err := l.ConsultarFirmasAutorizadas(context.Background(), m, c)
		if !errors.Is(err, ports.ErrResultadoFirmaDocumentoInvalido) || firmas != nil || p.tx.commits != 0 || p.tx.rollbacks != 1 {
			t.Fatalf("caso %d: firmas=%v error=%v commits=%d rollback=%d", i, firmas, err, p.tx.commits, p.tx.rollbacks)
		}
	}
}

func TestConsultaFirmasAtestadasNoExponeSalidaSiFallaConsultaOCommit(t *testing.T) {
	for _, tc := range []struct {
		nombre   string
		consulta bool
	}{{"consulta", true}, {"commit", false}} {
		t.Run(tc.nombre, func(t *testing.T) {
			l, p, m, c := prepararConsultaFirmasPrueba(t, `{"Encontrado":true,"ExpedienteRef":"expediente:ct152","Firmas":[]}`)
			if tc.consulta {
				p.tx.falloConsulta = errors.New("fallo privado")
			} else {
				p.tx.falloCommit = errors.New("fallo privado")
			}
			firmas, err := l.ConsultarFirmasAutorizadas(context.Background(), m, c)
			if !errors.Is(err, ports.ErrRegistroFirmaDocumentoNoDisponible) || firmas != nil || p.tx.rollbacks != 1 {
				t.Fatalf("filtración del resultado: %v, %v", firmas, err)
			}
		})
	}
}

func TestConsultaFirmasAtestadasTraduceRechazoSQL(t *testing.T) {
	for _, tc := range []struct {
		codigo   string
		esperado error
	}{
		{"42501", ports.ErrFirmaDocumentoDenegada},
		{"22023", ports.ErrSolicitudFirmaDocumentoInvalida},
		{"P1525", ports.ErrRegistroFirmaDocumentoNoDisponible},
	} {
		l, p, m, c := prepararConsultaFirmasPrueba(t, "")
		p.tx.falloConsulta = &pgconn.PgError{Code: tc.codigo}
		if firmas, err := l.ConsultarFirmasAutorizadas(context.Background(), m, c); !errors.Is(err, tc.esperado) || firmas != nil || p.tx.commits != 0 || p.tx.rollbacks != 1 {
			t.Fatalf("código %s: error=%v commits=%d rollback=%d", tc.codigo, err, p.tx.commits, p.tx.rollbacks)
		}
	}
}
