package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/ports"
)

type filaCertificadoPrueba struct {
	certificado, principal, cuenta, credencial, revision, huella *string
	vigente                                                      *bool
}

func textoCertificadoPrueba(s string) *string { return &s }
func booleanoCertificadoPrueba(b bool) *bool  { return &b }

func filaCertificadoValida() filaCertificadoPrueba {
	return filaCertificadoPrueba{
		certificado: textoCertificadoPrueba(strings.Repeat("a", 64)),
		principal:   textoCertificadoPrueba("per_" + strings.Repeat("p", 22)),
		cuenta:      textoCertificadoPrueba("cta_" + strings.Repeat("c", 22)),
		credencial:  textoCertificadoPrueba("vcc_" + strings.Repeat("v", 22)),
		revision:    textoCertificadoPrueba("3"),
		huella:      textoCertificadoPrueba(strings.Repeat("b", 64)),
		vigente:     booleanoCertificadoPrueba(true),
	}
}

type filasCertificadoPrueba struct {
	pgx.Rows
	filas    []filaCertificadoPrueba
	actual   int
	err      error
	cerradas bool
}

func (f *filasCertificadoPrueba) Next() bool {
	if f.actual >= len(f.filas) {
		return false
	}
	f.actual++
	return true
}
func (f *filasCertificadoPrueba) Scan(dest ...any) error {
	if f.actual < 1 || len(dest) != 7 {
		return errors.New("scan invalido")
	}
	fila := f.filas[f.actual-1]
	*dest[0].(**string), *dest[1].(**string), *dest[2].(**string) = fila.certificado, fila.principal, fila.cuenta
	*dest[3].(**string), *dest[4].(**string), *dest[5].(**string) = fila.credencial, fila.revision, fila.huella
	*dest[6].(**bool) = fila.vigente
	return nil
}
func (f *filasCertificadoPrueba) Err() error { return f.err }
func (f *filasCertificadoPrueba) Close()     { f.cerradas = true }

type txCertificadoPrueba struct {
	pgx.Tx
	filas       *filasCertificadoPrueba
	errExec     error
	errQuery    error
	errCommit   error
	consulta    string
	argumentos  []any
	preparacion string
	consultas   int
	commits     int
	rollbacks   int
}

func (t *txCertificadoPrueba) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	t.preparacion = sql
	return pgconn.CommandTag{}, t.errExec
}
func (t *txCertificadoPrueba) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	t.consultas++
	t.consulta = sql
	t.argumentos = append([]any(nil), args...)
	return t.filas, t.errQuery
}
func (t *txCertificadoPrueba) Commit(context.Context) error {
	t.commits++
	return t.errCommit
}
func (t *txCertificadoPrueba) Rollback(context.Context) error {
	t.rollbacks++
	return nil
}

type poolCertificadoPrueba struct {
	tx       *txCertificadoPrueba
	errBegin error
	opciones pgx.TxOptions
	begins   int
}

func (p *poolCertificadoPrueba) BeginTx(_ context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	p.begins++
	p.opciones = opts
	return p.tx, p.errBegin
}
func (*poolCertificadoPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaContextoActorDoble{err: errors.New("constructor no usado")}
}

func fuenteCertificadoPrueba(filas ...filaCertificadoPrueba) (*FuenteVinculoCertificadoFirmantePostgreSQL, *poolCertificadoPrueba) {
	p := &poolCertificadoPrueba{tx: &txCertificadoPrueba{filas: &filasCertificadoPrueba{filas: filas}}}
	return &FuenteVinculoCertificadoFirmantePostgreSQL{pool: p}, p
}

func TestFuenteCertificadoFirmanteLeeUnaFilaExactaEnSerializable(t *testing.T) {
	f, p := fuenteCertificadoPrueba(filaCertificadoValida())
	resultado, err := f.ResolverFirmantePorCertificado(context.Background(), strings.Repeat("a", 64))
	if err != nil || !resultado.Vigente || resultado.Revision != 3 ||
		resultado.PrincipalRef != "per_"+strings.Repeat("p", 22) {
		t.Fatalf("vinculo valido rechazado: %#v, %v", resultado, err)
	}
	if p.begins != 1 || p.opciones.IsoLevel != pgx.Serializable ||
		p.opciones.AccessMode != pgx.ReadWrite || p.tx.consultas != 1 ||
		!strings.Contains(p.tx.consulta, "resolver_vinculo_certificado_firmante_ct_v1($1)") ||
		len(p.tx.argumentos) != 1 || p.tx.argumentos[0] != strings.Repeat("a", 64) ||
		!strings.Contains(p.tx.preparacion, "set_config('timezone', 'UTC', true)") ||
		p.tx.commits != 1 || !p.tx.filas.cerradas {
		t.Fatal("lectura fuera de la unica transaccion serializable UTC")
	}
}

func TestFuenteCertificadoFirmanteFallaCerradaSinFilaODuplicada(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		filas    []filaCertificadoPrueba
		esperado error
	}{
		{"ausente", nil, ports.ErrVinculoCertificadoFirmanteNoAcreditado},
		{"duplicada", []filaCertificadoPrueba{filaCertificadoValida(), filaCertificadoValida()}, ports.ErrVinculoCertificadoFirmanteNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			f, p := fuenteCertificadoPrueba(caso.filas...)
			resultado, err := f.ResolverFirmantePorCertificado(context.Background(), strings.Repeat("a", 64))
			if !errors.Is(err, caso.esperado) || resultado != (ports.VinculoCertificadoFirmante{}) ||
				p.tx.commits != 0 || p.tx.rollbacks != 1 {
				t.Fatalf("lectura no fallo cerrada: %#v, %v", resultado, err)
			}
		})
	}
}

func TestFuenteCertificadoFirmanteRechazaRespuestaInvalida(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*filaCertificadoPrueba)
	}{
		{"sin huella", func(f *filaCertificadoPrueba) { f.certificado = nil }},
		{"eco distinto", func(f *filaCertificadoPrueba) { f.certificado = textoCertificadoPrueba(strings.Repeat("c", 64)) }},
		{"huella mayuscula", func(f *filaCertificadoPrueba) { f.huella = textoCertificadoPrueba(strings.Repeat("A", 64)) }},
		{"revision nula", func(f *filaCertificadoPrueba) { f.revision = nil }},
		{"revision cero", func(f *filaCertificadoPrueba) { f.revision = textoCertificadoPrueba("0") }},
		{"revision desbordada", func(f *filaCertificadoPrueba) { f.revision = textoCertificadoPrueba("18446744073709551616") }},
		{"principal incorrecto", func(f *filaCertificadoPrueba) { f.principal = textoCertificadoPrueba("per_otra") }},
		{"cuenta distinta", func(f *filaCertificadoPrueba) { f.cuenta = textoCertificadoPrueba("per_" + strings.Repeat("p", 22)) }},
		{"credencial con espacio", func(f *filaCertificadoPrueba) {
			f.credencial = textoCertificadoPrueba("vcc_" + strings.Repeat("v", 21) + " ")
		}},
		{"retirada", func(f *filaCertificadoPrueba) { f.vigente = booleanoCertificadoPrueba(false) }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			fila := filaCertificadoValida()
			caso.mutar(&fila)
			f, p := fuenteCertificadoPrueba(fila)
			resultado, err := f.ResolverFirmantePorCertificado(context.Background(), strings.Repeat("a", 64))
			if !errors.Is(err, ports.ErrVinculoCertificadoFirmanteNoAcreditado) ||
				resultado != (ports.VinculoCertificadoFirmante{}) || p.tx.commits != 0 {
				t.Fatalf("respuesta invalida aceptada: %#v, %v", resultado, err)
			}
		})
	}
}

func TestFuenteCertificadoFirmanteNoExponeErroresProveedorNiReintentaCommit(t *testing.T) {
	for _, fallo := range []string{"begin", "exec", "query", "rows", "commit"} {
		t.Run(fallo, func(t *testing.T) {
			f, p := fuenteCertificadoPrueba(filaCertificadoValida())
			secreto := errors.New("dsn privado de prueba")
			switch fallo {
			case "begin":
				p.errBegin = secreto
			case "exec":
				p.tx.errExec = secreto
			case "query":
				p.tx.errQuery = secreto
			case "rows":
				p.tx.filas.err = secreto
			case "commit":
				p.tx.errCommit = secreto
			}
			resultado, err := f.ResolverFirmantePorCertificado(context.Background(), strings.Repeat("a", 64))
			if !errors.Is(err, ports.ErrVinculoCertificadoFirmanteNoDisponible) ||
				strings.Contains(err.Error(), "dsn privado") || resultado != (ports.VinculoCertificadoFirmante{}) ||
				p.begins != 1 || p.tx.commits > 1 {
				t.Fatalf("fallo %s expuso datos o reintento: %#v, %v", fallo, resultado, err)
			}
		})
	}
}

func TestFuenteCertificadoFirmanteCancelacionYEntrada(t *testing.T) {
	f, p := fuenteCertificadoPrueba(filaCertificadoValida())
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := f.ResolverFirmantePorCertificado(ctx, strings.Repeat("a", 64)); err != context.Canceled || p.begins != 0 {
		t.Fatalf("cancelacion ignorada: %v", err)
	}
	for _, entrada := range []string{"", strings.Repeat("0", 64), strings.Repeat("A", 64), strings.Repeat("a", 63)} {
		if _, err := f.ResolverFirmantePorCertificado(context.Background(), entrada); !errors.Is(err, ports.ErrVinculoCertificadoFirmanteNoAcreditado) || p.begins != 0 {
			t.Fatalf("huella invalida consultada: %q, %v", entrada, err)
		}
	}
	if _, err := NuevoFuenteVinculoCertificadoFirmantePostgreSQL(context.Background(), nil); !errors.Is(err, ports.ErrVinculoCertificadoFirmanteNoDisponible) {
		t.Fatalf("constructor acepto pool ausente: %v", err)
	}
}

type filaRevalidacionCertificado struct {
	vigente *bool
	err     error
}

func (f filaRevalidacionCertificado) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*dest[0].(**bool) = f.vigente
	return nil
}

type txRevalidacionCertificado struct {
	pgx.Tx
	fila       filaRevalidacionCertificado
	consulta   string
	argumentos []any
	llamadas   int
}

func (t *txRevalidacionCertificado) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	t.llamadas++
	t.consulta = q
	t.argumentos = append([]any(nil), args...)
	return t.fila
}

func TestRevalidadorCertificadoComparteTxYBindingExacto(t *testing.T) {
	fila := filaCertificadoValida()
	v := ports.VinculoCertificadoFirmante{CertificadoHuella: *fila.certificado, PrincipalRef: *fila.principal, CuentaRef: *fila.cuenta, VinculoCredencialRef: *fila.credencial, Revision: 3, Huella: *fila.huella, Vigente: true}
	tx := &txRevalidacionCertificado{fila: filaRevalidacionCertificado{vigente: booleanoCertificadoPrueba(true)}}
	r, err := NuevoRevalidadorVinculoCertificadoFirmanteTransaccion(tx)
	if err != nil {
		t.Fatal(err)
	}
	if err = r.RevalidarVinculoCertificadoFirmante(context.Background(), v); err != nil {
		t.Fatal(err)
	}
	if tx.llamadas != 1 || !strings.Contains(tx.consulta, "revalidar_vinculo_certificado_firmante_ct_v1($1,$2::numeric,$3,$4,$5,$6)") || len(tx.argumentos) != 6 || tx.argumentos[0] != v.VinculoCredencialRef || tx.argumentos[1] != "3" || tx.argumentos[2] != v.Huella || tx.argumentos[3] != v.CertificadoHuella || tx.argumentos[4] != v.PrincipalRef || tx.argumentos[5] != v.CuentaRef {
		t.Fatal("binding exacto sustituido")
	}
	for _, valor := range []*bool{nil, booleanoCertificadoPrueba(false)} {
		tx.fila.vigente = valor
		if err = r.RevalidarVinculoCertificadoFirmante(context.Background(), v); err != ports.ErrVinculoCertificadoFirmanteNoAcreditado {
			t.Fatal("revocación o NULL aceptados")
		}
	}
	tx.fila.err = errors.New("valor privado")
	if err = r.RevalidarVinculoCertificadoFirmante(context.Background(), v); err != ports.ErrVinculoCertificadoFirmanteNoDisponible {
		t.Fatal("error privado expuesto")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = r.RevalidarVinculoCertificadoFirmante(ctx, v); err != context.Canceled {
		t.Fatal("cancelación ignorada")
	}
	if _, err = NuevoRevalidadorVinculoCertificadoFirmanteTransaccion(nil); err != ports.ErrVinculoCertificadoFirmanteNoDisponible {
		t.Fatal("tx ausente aceptada")
	}
}
