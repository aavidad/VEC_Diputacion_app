package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaIdentidadDocumentoPGPrueba struct {
	datos []byte
	err   error
}

func (f filaIdentidadDocumentoPGPrueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*dest[0].(*[]byte) = append([]byte(nil), f.datos...)
	return nil
}

type txIdentidadDocumentoPGPrueba struct {
	pgx.Tx
	fila               pgx.Row
	consulta, ajustes  string
	args               []any
	commits, rollbacks int
	rollbackErr        error
}

func (t *txIdentidadDocumentoPGPrueba) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	t.ajustes = q
	return pgconn.NewCommandTag("SELECT 1"), nil
}
func (t *txIdentidadDocumentoPGPrueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	t.consulta = q
	t.args = append([]any(nil), args...)
	return t.fila
}
func (t *txIdentidadDocumentoPGPrueba) Commit(context.Context) error { t.commits++; return nil }
func (t *txIdentidadDocumentoPGPrueba) Rollback(context.Context) error {
	t.rollbacks++
	return t.rollbackErr
}

type poolIdentidadDocumentoPGPrueba struct {
	tx       *txIdentidadDocumentoPGPrueba
	opciones pgx.TxOptions
	inicios  int
}

func (p *poolIdentidadDocumentoPGPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	p.opciones = opciones
	p.inicios++
	return p.tx, nil
}

func ordenIdentidadDocumentoPGPrueba(t *testing.T) (ports.OrdenIdentidadDocumentoAceptacionCT, time.Time) {
	t.Helper()
	anterior, ahora := ordenPersonaAceptacionCTPrueba(t)
	x := anterior.Solicitud.Selector
	s := ports.SolicitudIdentidadDocumentoAceptacionCT{Selector: ports.SelectorAceptacionNativaCT{
		OrganizacionRef: "organizacion:demo", ExpedienteRef: "expediente:demo", ClaveCT: "clave:demo",
		UnidadRef: x.UnidadRef, CategoriaRef: x.CategoriaRef, NecesidadRef: x.NecesidadRef,
		BolsaRef: "bolsa:demo", LlamamientoRef: x.LlamamientoRef,
		ReciboEmisionRef: "recibo:emision:demo", ReciboRespuestaRef: "recibo:respuesta:demo",
	}, ActorConfiable: anterior.Solicitud.ActorConfiable}
	p, err := application.PrepararIdentidadDocumentoAceptacionCT(s)
	if err != nil {
		t.Fatal(err)
	}
	h, _ := p.Recurso.HuellaContextoAutorizacionSHA256()
	c := s.ActorConfiable.Resultado
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", strings.Repeat("c", 64), strings.Repeat("d", 64),
		c.RegistroContextoRef, c.HuellaSHA256, ports.AccionIdentidadDocumentoAceptacionCT,
		p.Recurso.Referencia, h, ports.AudienciaIdentidadDocumentoAceptacionCT, ahora.Add(-time.Microsecond), ahora.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	m := anterior.Material
	m, err = vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(m.CapacidadCanonica(), resumen,
		m.DecisionCanonica(), m.MotivoCanonico(), c.RepresentacionCanonica, m.PersonaVersion(), m.PerfilVersion(),
		m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenIdentidadDocumentoAceptacionCT{Solicitud: s, Material: m}, ahora
}

func lecturaIdentidadDocumentoPGPrueba(o ports.OrdenIdentidadDocumentoAceptacionCT, ahora time.Time) ports.LecturaFuenteIdentidadDocumentoAceptacionCT {
	s := o.Solicitud.Selector
	return ports.LecturaFuenteIdentidadDocumentoAceptacionCT{Estado: "acreditado", Fuente: &ports.FuenteIdentidadDocumentoAceptacionCT{
		ActaRef: "acta:importacion-convoca:" + strings.Repeat("a", 64), FilaNumero: 2,
		InstantaneaRef: "instantanea:demo", VersionInstantanea: 1,
		HuellaInstantaneaSHA256: strings.Repeat("b", 64), BolsaRef: s.BolsaRef,
		HuellaListadoSHA256: strings.Repeat("c", 64), ReciboEmisionRef: s.ReciboEmisionRef,
		ReciboRespuestaRef: s.ReciboRespuestaRef},
		Evidencia: ports.EvidenciaConsultaPersonaAceptacionCT{DecisionRef: o.Material.ResumenCapacidad().DecisionRef(),
			ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "auditoria:demo", ConsultadaEn: ahora}}
}

func TestIdentidadDocumentoAceptacionCTPostgreSQLTransaccionYSalidaCerrada(t *testing.T) {
	for _, caso := range []string{"acreditado", "pendiente", "expediente_cruzado", "campo_inesperado", "denegado"} {
		t.Run(caso, func(t *testing.T) {
			o, ahora := ordenIdentidadDocumentoPGPrueba(t)
			v := lecturaIdentidadDocumentoPGPrueba(o, ahora)
			if caso == "pendiente" {
				v.Estado, v.Fuente = "pendiente", nil
			}
			if caso == "expediente_cruzado" {
				v.Fuente.ReciboRespuestaRef = "recibo:respuesta:ajeno"
			}
			b, _ := json.Marshal(v)
			if caso == "campo_inesperado" {
				b = append(b[:len(b)-1], []byte(`,"documento":"dato privado"}`)...)
			}
			tx := &txIdentidadDocumentoPGPrueba{fila: filaIdentidadDocumentoPGPrueba{datos: b}}
			if caso == "denegado" {
				tx.fila = filaIdentidadDocumentoPGPrueba{err: &pgconn.PgError{Code: "42501"}}
			}
			pool := &poolIdentidadDocumentoPGPrueba{tx: tx}
			r := &RepositorioIdentidadDocumentoAceptacionCTPostgreSQL{pool: pool, ahora: func() time.Time { return ahora }}
			salida, err := r.ConsultarFuenteIdentidadDocumentoAceptacionCT(context.Background(), o)
			if caso == "acreditado" || caso == "pendiente" {
				if err != nil || salida.Estado != caso || tx.commits != 1 || tx.rollbacks != 0 ||
					pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
					t.Fatalf("lectura: %v", err)
				}
				p, _ := application.PrepararIdentidadDocumentoAceptacionCT(o.Solicitud)
				if tx.consulta != funcionIdentidadDocumentoAceptacionCT || tx.ajustes != ajustesIdentidadDocumentoAceptacionCT ||
					len(tx.args) != 11 || tx.args[0] != string(p.MaterialCanonico) ||
					!bytes.Equal(tx.args[4].([]byte), o.Material.ContextoActorCanonico()) {
					t.Fatal("consulta sin contrato V3 exacto")
				}
			} else if err == nil || salida.Estado != "" || tx.commits != 0 || tx.rollbacks != 1 ||
				(caso == "denegado" && !errors.Is(err, ports.ErrIdentidadDocumentoAceptacionCTDenegada)) {
				t.Fatalf("resultado privado aceptado: %v", err)
			}
		})
	}
}

func TestIdentidadDocumentoAceptacionCTPostgreSQLRollbackFallidoConservaAmbasCausas(t *testing.T) {
	o, ahora := ordenIdentidadDocumentoPGPrueba(t)
	falloConsulta := &pgconn.PgError{Code: "40001", Routine: "ExecUpdate"}
	falloRollback := errors.New("rollback sin respuesta")
	tx := &txIdentidadDocumentoPGPrueba{fila: filaIdentidadDocumentoPGPrueba{err: falloConsulta}, rollbackErr: falloRollback}
	pool := &poolIdentidadDocumentoPGPrueba{tx: tx}
	r := &RepositorioIdentidadDocumentoAceptacionCTPostgreSQL{pool: pool, ahora: func() time.Time { return ahora }}
	v, err := r.ConsultarFuenteIdentidadDocumentoAceptacionCT(context.Background(), o)
	if v.Estado != "" || !errors.Is(err, falloConsulta) || !errors.Is(err, falloRollback) ||
		!errors.Is(err, errRollbackIdentidadDocumentoAceptacionCT) || tx.rollbacks != 1 || tx.commits != 0 || pool.inicios != 1 {
		t.Fatalf("rollback fallido ocultado o repetido: %v", err)
	}

	tx = &txIdentidadDocumentoPGPrueba{fila: filaIdentidadDocumentoPGPrueba{err: falloConsulta}, rollbackErr: pgx.ErrTxClosed}
	r = &RepositorioIdentidadDocumentoAceptacionCTPostgreSQL{pool: &poolIdentidadDocumentoPGPrueba{tx: tx}, ahora: func() time.Time { return ahora }}
	_, err = r.consultar(context.Background(), o, []byte("material sintético"))
	if !errors.Is(err, falloConsulta) || errors.Is(err, errRollbackIdentidadDocumentoAceptacionCT) {
		t.Fatalf("ErrTxClosed cambió causa original: %v", err)
	}
}
