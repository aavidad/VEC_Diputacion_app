package bolsa

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Dobles de transporte: estos tests no acreditan firma ni consumo SQL real.
type txPrueba struct {
	pgx.Tx
	commits, rollbacks int
	commitErr          error
	estado             string
}

func (x *txPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (x *txPrueba) Commit(context.Context) error   { x.commits++; return x.commitErr }
func (x *txPrueba) Rollback(context.Context) error { x.rollbacks++; return nil }
func (x *txPrueba) QueryRow(_ context.Context, q string, p ...any) pgx.Row {
	return filaPrueba{estado: x.estado, q: q, parametros: p}
}

type filaPrueba struct {
	estado, q  string
	parametros []any
}

func (f filaPrueba) Scan(d ...any) error {
	if f.q != consultaVersionV3 || len(f.parametros) != 11 || len(d) != 9 {
		return errors.New("contrato SQL alterado")
	}
	*d[0].(*string) = f.estado
	*d[1].(*[]byte) = nil
	*d[2].(**string) = nil
	*d[3].(*string) = "decision:prueba"
	*d[4].(*string) = strings.Repeat("a", 64)
	*d[5].(*string) = "auditoria:prueba"
	*d[6].(*string) = "lectura_convocatoria_" + strings.Repeat("a", 64)
	*d[7].(*string) = "correlacion_" + strings.Repeat("a", 32)
	*d[8].(*time.Time) = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	return nil
}

type poolPrueba struct {
	tx       *txPrueba
	opciones pgx.TxOptions
	llamadas int
}

func (p *poolPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.llamadas++
	p.opciones = o
	return p.tx, nil
}

type correlacionPrueba struct{}

func (correlacionPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_" + strings.Repeat("a", 32), nil
}

func ordenEstructuralPrueba(t *testing.T) ports.OrdenConsultaConvocatoria {
	t.Helper()
	z := strings.Repeat("a", 24)
	ahora := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	cuenta := vecdomain.CuentaAutenticadaContextoActor{CuentaRef: "cta_" + z, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}
	instantanea := vecdomain.InstantaneaContextoActor{VinculoRef: "vca_" + z, VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_" + z, PersonaVersion: 1, PerfilActivoRef: "prf_" + z, PerfilVersion: 1, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), Vinculos: []vecdomain.VinculoReferenciaContextoActor{{VinculoRef: "vin_" + z, Version: 1, Tipo: vecdomain.TipoReferenciaContextoActorEmpleado, Referencia: "emp_" + z, Estado: vecdomain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}}}
	actor, err := vecdomain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), correlacionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	s := ports.SolicitudConsultaConvocatoria{Selector: bolsaports.SelectorVersionConvocatoriaExacta{ID: "convocatoria:prueba", Secuencia: 2}, Actor: actor, Correlacion: correlacion}
	p, err := application.PrepararConsultaConvocatoria(s)
	if err != nil {
		t.Fatal(err)
	}
	h, err := p.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	aCanon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	x, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "contexto:prueba", strings.Repeat("c", 64), bolsaports.AccionConsultarVersionConvocatoria, p.Recurso.Referencia, h, application.AudienciaConsultaConvocatoriaV3, ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), x, []byte("decision"), []byte("motivo"), aCanon, 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenConsultaConvocatoria{Solicitud: s, Autorizacion: a}
}

func TestAusenciaConAuditoriaConfirmaAntesDeDevolver(t *testing.T) {
	o := ordenEstructuralPrueba(t)
	tx := &txPrueba{estado: "no_encontrada"}
	p := &poolPrueba{tx: tx}
	r, err := nuevoRepositorioVersiones(p)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := r.ObtenerVersionExactaV3(context.Background(), o)
	if err != nil || resultado.Estado != "no_encontrada" || tx.commits != 1 || p.opciones.IsoLevel != pgx.Serializable || p.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("lectura no confirmada: %v", err)
	}
}

func TestLecturaInvalidaOCommitFallidoNoDevuelveResultado(t *testing.T) {
	for nombre, tx := range map[string]*txPrueba{"estado ajeno": {estado: "encontrada"}, "commit falla": {estado: "no_encontrada", commitErr: errors.New("detalle privado")}} {
		t.Run(nombre, func(t *testing.T) {
			r, _ := nuevoRepositorioVersiones(&poolPrueba{tx: tx})
			resultado, err := r.ObtenerVersionExactaV3(context.Background(), ordenEstructuralPrueba(t))
			if err != ports.ErrConvocatoriaNoDisponible || resultado.Estado != "" || tx.rollbacks != 1 {
				t.Fatal("se anunció un resultado parcial")
			}
			if nombre == "estado ajeno" && tx.commits != 0 {
				t.Fatal("se confirmó una respuesta insegura")
			}
		})
	}
}
