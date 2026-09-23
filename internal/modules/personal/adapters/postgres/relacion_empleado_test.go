package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
	"time"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaP struct {
	vals []any
	err  error
}

func (f filaP) Scan(d ...any) error {
	if f.err != nil {
		return f.err
	}
	for i, x := range d {
		switch p := x.(type) {
		case *string:
			*p = f.vals[i].(string)
		case *int:
			*p = f.vals[i].(int)
		case *time.Time:
			*p = f.vals[i].(time.Time)
		case *[]byte:
			*p = append([]byte(nil), f.vals[i].([]byte)...)
		}
	}
	return nil
}

type txP struct {
	pgx.Tx
	q                   []string
	a                   [][]any
	commits, rollbacks  int
	errQ, errExec, errC error
	cancel              func()
	cancelQuery         func()
	fila                pgx.Row
}

func (t *txP) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	t.q = append(t.q, q)
	return pgconn.CommandTag{}, t.errExec
}
func (t *txP) QueryRow(_ context.Context, q string, a ...any) pgx.Row {
	t.q = append(t.q, q)
	copia := make([]any, len(a))
	for i, v := range a {
		if b, ok := v.([]byte); ok {
			copia[i] = append([]byte(nil), b...)
		} else {
			copia[i] = v
		}
	}
	t.a = append(t.a, copia)
	if t.cancelQuery != nil {
		t.cancelQuery()
	}
	if t.errQ != nil {
		return filaP{err: t.errQ}
	}
	return t.fila
}
func (t *txP) Commit(context.Context) error {
	t.commits++
	if t.cancel != nil {
		t.cancel()
	}
	return t.errC
}
func (t *txP) Rollback(context.Context) error { t.rollbacks++; return nil }

type poolP struct {
	tx  pgx.Tx
	n   int
	o   pgx.TxOptions
	err error
}

func (p *poolP) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.n++
	p.o = o
	return p.tx, p.err
}
func ordenP(t *testing.T) personalports.OrdenConsultaRelacionPropia {
	t.Helper()
	z := strings.Repeat("a", 24)
	now := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	cu := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	in := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cu.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: now.Add(-time.Hour), VigenteHasta: now.Add(time.Hour), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: now.Add(-time.Hour), VigenteHasta: now.Add(time.Hour)}}}
	actor, e := vecdomain.NuevoContextoActor(cu, in, now)
	if e != nil {
		t.Fatal(e)
	}
	f, _ := personaldomain.NuevaFechaCivil("2026-09-20")
	m, e := personaldomain.NuevoMaterialConsultaRelacionPropia(personaldomain.SolicitudConsultaRelacionPropia{FechaReferencia: f, Operacion: personaldomain.OperacionListaRelacionPropia, Actor: actor})
	if e != nil {
		t.Fatal(e)
	}
	h, _ := m.HuellaSHA256()
	r := m.Recurso()
	s, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), "personal.relacion.propia.consultar_dietas", r.Referencia, h, "vec_personal.relacion_propia.consultar_dietas.v1", now, now.Add(3*time.Second))
	if e != nil {
		t.Fatal(e)
	}
	root, e := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if e != nil {
		t.Fatal(e)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	a, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), s, []byte("d"), []byte("m"), canon, 1, 1, []byte("p"), []byte("s"), []byte("e"), root)
	if e != nil {
		t.Fatal(e)
	}
	return personalports.OrdenConsultaRelacionPropia{Material: m, Autorizacion: a}
}
func filaOk(t *testing.T, o personalports.OrdenConsultaRelacionPropia) pgx.Row {
	t.Helper()
	c := o.Material.Solicitud()
	emp, _ := c.Actor.Referencias("empleado")
	j := []byte(`[{"desde":"2026-01-01","empleado_ref":"` + emp[0] + `","estado":"activa","fuente_ref":"fuente:x","fuente_version":1,"hasta":null,"persona_ref":"` + c.Actor.PersonaRef + `","procedencia_acto_ref":"acto:x","relacion_ref":"rel_` + strings.Repeat("a", 24) + `","unidad_ref":"unidad:x","version":1}]`)
	s := o.Autorizacion.ResumenCapacidad()
	return filaP{vals: []any{"rpd_" + strings.Repeat("a", 32), s.DecisionRef(), s.EfectoRef(), strings.Repeat("a", 64), "aud", s.EmitidaEn().Add(time.Microsecond), 1, j}}
}

func filaCon(o personalports.OrdenConsultaRelacionPropia, recibo, huella string, cardinalidad int, relaciones string) pgx.Row {
	s := o.Autorizacion.ResumenCapacidad()
	return filaP{vals: []any{recibo, s.DecisionRef(), s.EfectoRef(), huella, "aud", s.EmitidaEn().Add(time.Microsecond), cardinalidad, []byte(relaciones)}}
}

func jsonRelacionP(o personalports.OrdenConsultaRelacionPropia, relacion string) string {
	c := o.Material.Solicitud()
	empleados, _ := c.Actor.Referencias("empleado")
	return `{"desde":"2026-01-01","empleado_ref":"` + empleados[0] + `","estado":"activa","fuente_ref":"fuente:x","fuente_version":1,"hasta":null,"persona_ref":"` + c.Actor.PersonaRef + `","procedencia_acto_ref":"acto:x","relacion_ref":"` + relacion + `","unidad_ref":"unidad:x","version":1}`
}

func ordenDetalleP(t *testing.T, lista personalports.OrdenConsultaRelacionPropia, relacion string) personalports.OrdenConsultaRelacionPropia {
	t.Helper()
	c := lista.Material.Solicitud()
	c.Operacion, c.RelacionRef = personaldomain.OperacionDetalleRelacionPropia, relacion
	m, err := personaldomain.NuevoMaterialConsultaRelacionPropia(c)
	if err != nil {
		t.Fatal(err)
	}
	lista.Material = m
	return lista
}
func TestRepositorioPropioTransaccionYEnvelope(t *testing.T) {
	o := ordenP(t)
	tx := &txP{fila: filaOk(t, o)}
	p := &poolP{tx: tx}
	r, e := nuevoRepositorioRelacionEmpleadoPostgreSQL(p)
	if e != nil {
		t.Fatal(e)
	}
	x, e := r.ConsultarRelacionesPropiasDietas(context.Background(), o)
	if e != nil || len(x.Relaciones) != 1 || p.n != 1 || p.o.IsoLevel != pgx.Serializable || p.o.AccessMode != pgx.ReadWrite || len(tx.a) != 1 || len(tx.a[0]) != 11 || tx.commits != 1 || tx.rollbacks != 0 {
		t.Fatalf("e=%v n=%d args=%d commit=%d rollback=%d", e, p.n, len(tx.a), tx.commits, tx.rollbacks)
	}
	if tx.q[1] != consultaRelacionesPropiasDietas {
		t.Fatal("sql distinto")
	}
}
func TestRepositorioPropioRevierteYCierraErrores(t *testing.T) {
	o := ordenP(t)
	for _, c := range []struct {
		n      string
		row    pgx.Row
		commit error
		want   error
	}{{"p7202", filaP{err: &pgconn.PgError{Code: "P7202"}}, nil, personalports.ErrRelacionEmpleadoAmbigua}, {"commit", filaOk(t, o), errors.New("incierto"), personalports.ErrRelacionEmpleadoNoDisponible}, {"json", filaP{vals: []any{"r", o.Autorizacion.ResumenCapacidad().DecisionRef(), o.Autorizacion.ResumenCapacidad().EfectoRef(), strings.Repeat("a", 64), "aud", o.Autorizacion.ResumenCapacidad().EmitidaEn().Add(time.Microsecond), 1, []byte(`[{"extra":1}]`)}}, nil, personalports.ErrRelacionEmpleadoNoDisponible}} {
		t.Run(c.n, func(t *testing.T) {
			tx := &txP{fila: c.row, errC: c.commit}
			r, _ := nuevoRepositorioRelacionEmpleadoPostgreSQL(&poolP{tx: tx})
			_, e := r.ConsultarRelacionesPropiasDietas(context.Background(), o)
			if !errors.Is(e, c.want) || tx.rollbacks != 1 {
				t.Fatalf("e=%v rb=%d", e, tx.rollbacks)
			}
		})
	}
}
func TestRepositorioRelacionPropiaNoExponeResolutorNiSelect(t *testing.T) {
	if !strings.Contains(consultaRelacionesPropiasDietas, "consultar_relaciones_propias_dietas_v1") || strings.Contains(consultaRelacionesPropiasDietas, "resolver_relacion_dietas") {
		t.Fatal("fachada")
	}
}

func TestRepositorioPropioArgsAjustesYCancelacion(t *testing.T) {
	o := ordenP(t)
	ctx, cancel := context.WithCancel(context.Background())
	tx := &txP{fila: filaOk(t, o), cancel: cancel}
	r, _ := nuevoRepositorioRelacionEmpleadoPostgreSQL(&poolP{tx: tx})
	x, e := r.ConsultarRelacionesPropiasDietas(ctx, o)
	if e != nil || x.Evidencia.ReciboRef != "rpd_"+strings.Repeat("a", 32) || len(tx.q) != 2 || tx.q[0] != ajustesConsultaRelacionesPropiasDietas || len(tx.a) != 1 || len(tx.a[0]) != 11 || tx.a[0][0] != string(o.Material.Canonico()) || tx.a[0][5] != int64(1) || tx.a[0][6] != int64(1) {
		t.Fatalf("e=%v q=%#v args=%#v", e, tx.q, tx.a)
	}
	for i, b := range [][]byte{o.Autorizacion.CapacidadCanonica(), o.Autorizacion.DecisionCanonica(), o.Autorizacion.MotivoCanonico(), o.Autorizacion.ContextoActorCanonico(), o.Autorizacion.PayloadVECAD3(), o.Autorizacion.SobreCOSESign1(), o.Autorizacion.EvidenciaVerificacion(), o.Autorizacion.RaizPublicaSPKI()} {
		if !bytes.Equal(tx.a[0][[]int{1, 2, 3, 4, 7, 8, 9, 10}[i]].([]byte), b) {
			t.Fatal("blob")
		}
	}
}
func TestRepositorioPropioCancelacionPreviaYFallosTransaccion(t *testing.T) {
	o := ordenP(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := &poolP{}
	r, _ := nuevoRepositorioRelacionEmpleadoPostgreSQL(p)
	_, e := r.ConsultarRelacionesPropiasDietas(ctx, o)
	if !errors.Is(e, context.Canceled) || p.n != 0 {
		t.Fatalf("e=%v n=%d", e, p.n)
	}
	for _, tx := range []*txP{{fila: filaOk(t, o), errExec: errors.New("x")}, {fila: filaOk(t, o), errQ: errors.New("q")}} {
		r, _ = nuevoRepositorioRelacionEmpleadoPostgreSQL(&poolP{tx: tx})
		_, e = r.ConsultarRelacionesPropiasDietas(context.Background(), o)
		if !errors.Is(e, personalports.ErrRelacionEmpleadoNoDisponible) || tx.rollbacks != 1 {
			t.Fatalf("e=%v rb=%d", e, tx.rollbacks)
		}
	}
}

func TestRepositorioPropioCancelaEntreLecturaYCommit(t *testing.T) {
	o := ordenP(t)
	ctx, cancelar := context.WithCancel(context.Background())
	tx := &txP{fila: filaOk(t, o), cancelQuery: cancelar}
	r, _ := nuevoRepositorioRelacionEmpleadoPostgreSQL(&poolP{tx: tx})
	_, err := r.ConsultarRelacionesPropiasDietas(ctx, o)
	if !errors.Is(err, context.Canceled) || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("err=%v commit=%d rollback=%d", err, tx.commits, tx.rollbacks)
	}
}
func TestEnvelopePropioRechazaHuellaYTiempoNoCanonicos(t *testing.T) {
	o := ordenP(t)
	c := o.Material.Solicitud()
	emp, _ := c.Actor.Referencias("empleado")
	j := []byte(`[{"desde":"2026-01-01","empleado_ref":"` + emp[0] + `","estado":"activa","fuente_ref":"fuente:x","fuente_version":1,"hasta":null,"persona_ref":"` + c.Actor.PersonaRef + `","procedencia_acto_ref":"acto:x","relacion_ref":"rel_` + strings.Repeat("a", 24) + `","unidad_ref":"unidad:x","version":1}]`)
	s := o.Autorizacion.ResumenCapacidad()
	for _, x := range []struct {
		recibo, huella string
		instante       time.Time
	}{{"r", "z" + strings.Repeat("a", 63), s.EmitidaEn().Add(time.Microsecond)}, {"rpd_" + strings.Repeat("a", 32), strings.Repeat("a", 64), s.ExpiraEn()}, {"rpd_" + strings.Repeat("a", 32), strings.Repeat("a", 64), s.EmitidaEn().Add(1500 * time.Nanosecond)}} {
		_, e := decodificarRelacionesPropias(j, 1, c, o.Autorizacion, x.recibo, s.DecisionRef(), s.EfectoRef(), x.huella, "aud", x.instante)
		if e == nil {
			t.Fatal("envelope aceptado")
		}
	}
}

func TestRepositorioPropioEnvelopesYCardinalidadAislados(t *testing.T) {
	o := ordenP(t)
	valido := jsonRelacionP(o, "rel_"+strings.Repeat("a", 24))
	for _, prueba := range []struct {
		nombre                         string
		recibo, huella, relaciones     string
		cardinalidad, quiereRelaciones int
		quiereError                    bool
	}{
		{"recibo invalido", "recibo", strings.Repeat("a", 64), "[" + valido + "]", 1, 0, true},
		{"huella no hexadecimal", "rpd_" + strings.Repeat("a", 32), "z" + strings.Repeat("a", 63), "[" + valido + "]", 1, 0, true},
		{"null con cero", "rpd_" + strings.Repeat("a", 32), strings.Repeat("a", 64), "null", 0, 0, true},
		{"array vacio con cero", "rpd_" + strings.Repeat("a", 32), strings.Repeat("a", 64), "[]", 0, 0, false},
		{"dos relaciones distintas", "rpd_" + strings.Repeat("a", 32), strings.Repeat("a", 64), "[" + valido + "," + jsonRelacionP(o, "rel_"+strings.Repeat("b", 24)) + "]", 2, 2, false},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			tx := &txP{fila: filaCon(o, prueba.recibo, prueba.huella, prueba.cardinalidad, prueba.relaciones)}
			repo, _ := nuevoRepositorioRelacionEmpleadoPostgreSQL(&poolP{tx: tx})
			resultado, err := repo.ConsultarRelacionesPropiasDietas(context.Background(), o)
			if prueba.quiereError {
				if !errors.Is(err, personalports.ErrRelacionEmpleadoNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
					t.Fatalf("err=%v commit=%d rollback=%d", err, tx.commits, tx.rollbacks)
				}
				return
			}
			if err != nil || len(resultado.Relaciones) != prueba.quiereRelaciones || tx.commits != 1 || tx.rollbacks != 0 {
				t.Fatalf("err=%v relaciones=%d commit=%d rollback=%d", err, len(resultado.Relaciones), tx.commits, tx.rollbacks)
			}
		})
	}
}

func TestRepositorioPropioDetalleRechazaRelacionDistintaAntesDeCommit(t *testing.T) {
	lista := ordenP(t)
	detalle := ordenDetalleP(t, lista, "rel_"+strings.Repeat("a", 24))
	tx := &txP{fila: filaCon(detalle, "rpd_"+strings.Repeat("a", 32), strings.Repeat("a", 64), 1, "["+jsonRelacionP(detalle, "rel_"+strings.Repeat("b", 24))+"]")}
	repo, _ := nuevoRepositorioRelacionEmpleadoPostgreSQL(&poolP{tx: tx})
	_, err := repo.ConsultarRelacionesPropiasDietas(context.Background(), detalle)
	if !errors.Is(err, personalports.ErrRelacionEmpleadoNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("err=%v commit=%d rollback=%d", err, tx.commits, tx.rollbacks)
	}
}

func TestEnvelopePropioRechazaJSONParcialONoPropio(t *testing.T) {
	o := ordenP(t)
	c := o.Material.Solicitud()
	emp, _ := c.Actor.Referencias("empleado")
	base := `{"desde":"2026-01-01","empleado_ref":"` + emp[0] + `","estado":"activa","fuente_ref":"fuente:x","fuente_version":1,"hasta":null,"persona_ref":"` + c.Actor.PersonaRef + `","procedencia_acto_ref":"acto:x","relacion_ref":"rel_` + strings.Repeat("a", 24) + `","unidad_ref":"unidad:x","version":1}`
	s := o.Autorizacion.ResumenCapacidad()
	probar := func(nombre, documento string) {
		t.Helper()
		t.Run(nombre, func(t *testing.T) {
			_, err := decodificarRelacionesPropias([]byte(documento), 1, c, o.Autorizacion, "rpd_"+strings.Repeat("a", 32), s.DecisionRef(), s.EfectoRef(), strings.Repeat("a", 64), "aud", s.EmitidaEn().Add(time.Microsecond))
			if err == nil {
				t.Fatal("wire aceptado")
			}
		})
	}
	probar("lista nula", "null")
	probar("hasta omitida", `[{"desde":"2026-01-01","empleado_ref":"`+emp[0]+`","estado":"activa","fuente_ref":"fuente:x","fuente_version":1,"persona_ref":"`+c.Actor.PersonaRef+`","procedencia_acto_ref":"acto:x","relacion_ref":"rel_`+strings.Repeat("a", 24)+`","unidad_ref":"unidad:x","version":1}]`)
	probar("persona ajena", "["+strings.Replace(base, c.Actor.PersonaRef, "per_"+strings.Repeat("b", 24), 1)+"]")
	probar("empleado ajeno", "["+strings.Replace(base, emp[0], "emp_"+strings.Repeat("b", 24), 1)+"]")
	probar("fecha no vigente", "["+strings.Replace(base, "2026-01-01", "2026-09-21", 1)+"]")
	probar("campo adicional", "["+strings.TrimSuffix(base, "}")+`,"otro":true}]`)
}
