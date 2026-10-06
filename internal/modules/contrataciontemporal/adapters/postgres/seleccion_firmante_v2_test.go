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

type filaSeleccionPrueba struct {
	bruto string
	err   error
}

func (f filaSeleccionPrueba) Scan(d ...any) error {
	if f.err != nil {
		return f.err
	}
	*(d[0].(*[]byte)) = []byte(f.bruto)
	return nil
}

type txSeleccionPrueba struct {
	pgx.Tx
	fila      filaSeleccionPrueba
	args      []any
	rollbacks int
	commits   int
}

func (t *txSeleccionPrueba) QueryRow(_ context.Context, _ string, a ...any) pgx.Row {
	t.args = a
	return t.fila
}
func (t *txSeleccionPrueba) Rollback(context.Context) error { t.rollbacks++; return nil }
func (t *txSeleccionPrueba) Commit(context.Context) error   { t.commits++; return nil }

type poolSeleccionPrueba struct {
	tx   *txSeleccionPrueba
	opts pgx.TxOptions
}

func (p *poolSeleccionPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.opts = o
	return p.tx, nil
}

func respuestaSeleccionPrueba(cambiar func(string) string) string {
	h := strings.Repeat("9", 64)
	r := `{"esquema":"vec.autorizacion.seleccion-firmante-plan.ct.v1","persona_ref":"per_x","perfil_activo_ref":"prf_x","rol_id":"rol_x",` +
		`"cuenta_ref":"cta_x","vinculo_certificado":{"referencia":"vcc_x","version":1,"huella_sha256":"` + h + `"},` +
		`"cargo_ref":"car_x","enlace_ejercicio_ref":"enc_x",` +
		`"asignacion":{"referencia":"asignacion:x:v1","version":1,"huella_sha256":"` + h + `","vigente_desde":"2026-01-01T00:00:00Z","vigente_hasta":"2027-01-01T00:00:00Z"},` +
		`"rol":{"referencia":"rol:x:v1","huella_sha256":"` + h + `"},"control_rol":{"referencia":"rol:x:v1","revision":1,"huella_sha256":"` + h + `"}}`
	if cambiar != nil {
		return cambiar(r)
	}
	return r
}

var solicitudSeleccionPrueba = ports.SolicitudSeleccionFirmanteV2{CertificadoHuella: strings.Repeat("7", 64), CargoRef: "car_x", RolID: "rol_x",
	Accion: "a.b", TipoRecurso: "firma_vec_documento_contratacion_temporal", Finalidad: "f", OrganizacionRef: "org_x", UnidadRef: "uni_x"}

// Lee en SERIALIZABLE de escritura (lo exige AUT56), deshace siempre y
// devuelve la selección exacta; 42501 es «no acreditada» y una respuesta
// incoherente, «no disponible».
func TestSeleccionFirmantePostgreSQL(t *testing.T) {
	pool := &poolSeleccionPrueba{tx: &txSeleccionPrueba{fila: filaSeleccionPrueba{bruto: respuestaSeleccionPrueba(nil)}}}
	s, err := NuevaSeleccionFirmantePostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.SeleccionarFirmanteV2(t.Context(), solicitudSeleccionPrueba)
	if err != nil || r.EnlaceEjercicioRef != "enc_x" || r.VinculoCertificado.Version != 1 || r.ControlRol.Version != 1 || r.CuentaRef != "cta_x" ||
		pool.opts.IsoLevel != pgx.Serializable || pool.opts.AccessMode != pgx.ReadWrite || pool.tx.rollbacks != 1 || pool.tx.commits != 0 ||
		len(pool.tx.args) != 8 || pool.tx.args[4] != solicitudSeleccionPrueba.TipoRecurso {
		t.Fatalf("selección distinta: %+v %v", r, err)
	}
	for caso, x := range map[string]struct {
		fila filaSeleccionPrueba
		err  error
	}{
		"denegada":     {filaSeleccionPrueba{err: &pgconn.PgError{Code: "42501"}}, ports.ErrCompetenciaFirmanteNoAcreditada},
		"caida":        {filaSeleccionPrueba{err: errors.New("x")}, ports.ErrCompetenciaFirmanteNoDisponible},
		"otro_rol":     {filaSeleccionPrueba{bruto: respuestaSeleccionPrueba(func(s string) string { return strings.Replace(s, `"rol_id":"rol_x"`, `"rol_id":"rol_y"`, 1) })}, ports.ErrCompetenciaFirmanteNoDisponible},
		"campo_de_mas": {filaSeleccionPrueba{bruto: respuestaSeleccionPrueba(func(s string) string { return strings.Replace(s, `{"esquema"`, `{"extra":1,"esquema"`, 1) })}, ports.ErrCompetenciaFirmanteNoDisponible},
		"otro_cargo":   {filaSeleccionPrueba{bruto: respuestaSeleccionPrueba(func(s string) string { return strings.Replace(s, `"car_x"`, `"car_y"`, 1) })}, ports.ErrCompetenciaFirmanteNoDisponible},
		"version_cero": {filaSeleccionPrueba{bruto: respuestaSeleccionPrueba(func(s string) string {
			return strings.Replace(s, `"asignacion:x:v1","version":1`, `"asignacion:x:v1","version":0`, 1)
		})}, ports.ErrCompetenciaFirmanteNoDisponible},
		"control_de_otro_rol": {filaSeleccionPrueba{bruto: respuestaSeleccionPrueba(func(s string) string {
			return strings.Replace(s, `"control_rol":{"referencia":"rol:x:v1"`, `"control_rol":{"referencia":"rol:y:v1"`, 1)
		})}, ports.ErrCompetenciaFirmanteNoDisponible},
		"huella_mala": {filaSeleccionPrueba{bruto: respuestaSeleccionPrueba(func(s string) string {
			return strings.Replace(s, `"rol":{"referencia":"rol:x:v1","huella_sha256":"9`, `"rol":{"referencia":"rol:x:v1","huella_sha256":"X`, 1)
		})}, ports.ErrCompetenciaFirmanteNoDisponible},
	} {
		pool.tx.fila = x.fila
		if _, err := s.SeleccionarFirmanteV2(t.Context(), solicitudSeleccionPrueba); !errors.Is(err, x.err) {
			t.Fatalf("%s: %v", caso, err)
		}
	}
	mala := solicitudSeleccionPrueba
	mala.CertificadoHuella = "x"
	if _, err := s.SeleccionarFirmanteV2(t.Context(), mala); !errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) {
		t.Fatalf("certificado mal formado: %v", err)
	}
	larga := solicitudSeleccionPrueba
	larga.CargoRef = strings.Repeat("c", maximoCampoSeleccionFirmante+1)
	pool.tx.args = nil
	if _, err := s.SeleccionarFirmanteV2(t.Context(), larga); !errors.Is(err, ports.ErrCompetenciaFirmanteNoAcreditada) || pool.tx.args != nil {
		t.Fatalf("campo demasiado largo enviado a SQL: %v", err)
	}
	ctx, cancelar := context.WithCancel(t.Context())
	cancelar()
	if _, err := s.SeleccionarFirmanteV2(ctx, solicitudSeleccionPrueba); !errors.Is(err, context.Canceled) || pool.tx.args != nil {
		t.Fatalf("contexto cancelado: %v", err)
	}
}
