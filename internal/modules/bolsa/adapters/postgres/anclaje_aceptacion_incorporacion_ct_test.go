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

func ordenAnclajeCTPrueba(t *testing.T) (ports.OrdenConsultaAnclajeAceptacionCT, time.Time) {
	t.Helper()
	anterior, ahora := ordenPersonaAceptacionCTPrueba(t)
	x := anterior.Solicitud.Selector
	q := ports.SolicitudConsultaAnclajeAceptacionCT{Selector: ports.SelectorAnclajeAceptacionCT{UnidadRef: x.UnidadRef, CategoriaRef: x.CategoriaRef, NecesidadRef: x.NecesidadRef, AceptacionOperacionRef: x.AceptacionOperacionRef, AceptacionRegistroSHA256: x.AceptacionRegistroSHA256, AperturaOperacionRef: x.AperturaOperacionRef, LlamamientoRef: x.LlamamientoRef, PropuestaRef: x.PropuestaRef}, ActorConfiable: anterior.Solicitud.ActorConfiable}
	p, err := application.PrepararConsultaAnclajeAceptacionCT(q)
	if err != nil {
		t.Fatal(err)
	}
	h, _ := p.Recurso.HuellaContextoAutorizacionSHA256()
	c := q.ActorConfiable.Resultado
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", strings.Repeat("c", 64), strings.Repeat("d", 64), c.RegistroContextoRef, c.HuellaSHA256, ports.AccionConsultaAnclajeAceptacionCT, p.Recurso.Referencia, h, ports.AudienciaConsultaAnclajeAceptacionCT, ahora.Add(-time.Microsecond), ahora.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	m := anterior.Material
	m, err = vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(m.CapacidadCanonica(), resumen, m.DecisionCanonica(), m.MotivoCanonico(), c.RepresentacionCanonica, m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenConsultaAnclajeAceptacionCT{Solicitud: q, Material: m}, ahora
}
func resultadoAnclajeCTPrueba(o ports.OrdenConsultaAnclajeAceptacionCT, ahora time.Time) ports.ResultadoConsultaAnclajeAceptacionCT {
	s := o.Solicitud.Selector
	return ports.ResultadoConsultaAnclajeAceptacionCT{Estado: "acreditado", Anclaje: &ports.AnclajeAceptacionIncorporacionCT{SelectorPersonaAceptacionCT: ports.SelectorPersonaAceptacionCT{UnidadRef: s.UnidadRef, CategoriaRef: s.CategoriaRef, NecesidadRef: s.NecesidadRef, AceptacionOperacionRef: s.AceptacionOperacionRef, AceptacionRegistroSHA256: s.AceptacionRegistroSHA256, AperturaOperacionRef: s.AperturaOperacionRef, AperturaRegistroSHA256: strings.Repeat("b", 64), LlamamientoRef: s.LlamamientoRef, PropuestaRef: s.PropuestaRef}, AceptacionReciboRef: "recibo:original"}, Evidencia: ports.EvidenciaConsultaPersonaAceptacionCT{DecisionRef: o.Material.ResumenCapacidad().DecisionRef(), ConsumoHuellaSHA256: strings.Repeat("e", 64), AuditoriaRef: "auditoria:lectura", ConsultadaEn: ahora}}
}
func TestAnclajeAceptacionCTPostgreSQLSoloFachadaNominalYCommitEstados(t *testing.T) {
	for _, estado := range []string{"acreditado", "pendiente", "no_encontrada"} {
		o, ahora := ordenAnclajeCTPrueba(t)
		v := resultadoAnclajeCTPrueba(o, ahora)
		v.Estado = estado
		if estado != "acreditado" {
			v.Anclaje = nil
		}
		b, _ := json.Marshal(v)
		tx := &txPersonaAceptacionCTPrueba{fila: filaPersonaAceptacionCTPrueba{datos: b}}
		pool := &poolPersonaAceptacionCTPrueba{transacciones: []*txPersonaAceptacionCTPrueba{tx}}
		r := &RepositorioConsultaAnclajeAceptacionCTPostgreSQL{pool: pool, ahora: func() time.Time { return ahora }}
		salida, err := r.ConsultarAnclajeAceptacionCT(context.Background(), o)
		if err != nil || salida.Estado != estado || tx.commits != 1 || tx.rollbacks != 1 || pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite {
			t.Fatalf("estado=%s error=%v", estado, err)
		}
		p, _ := application.PrepararConsultaAnclajeAceptacionCT(o.Solicitud)
		if tx.consulta != funcionConsultaAnclajeAceptacionCT || tx.ajustes != ajustesConsultaAnclajeAceptacionCT || len(tx.args) != 11 || tx.args[0] != string(p.MaterialCanonico) || !bytes.Equal(tx.args[4].([]byte), o.Material.ContextoActorCanonico()) {
			t.Fatal("protocolo no nominal")
		}
	}
}
func TestAnclajeAceptacionCTPostgreSQLRollbackSalidasPrivadasCrucesYFallos(t *testing.T) {
	for _, caso := range []string{"persona", "desligada", "sin_sha", "parcial", "exceso", "dos_json", "ajustes", "commit", "denegado"} {
		o, ahora := ordenAnclajeCTPrueba(t)
		v := resultadoAnclajeCTPrueba(o, ahora)
		if caso == "desligada" {
			v.Anclaje.AperturaOperacionRef = "apertura:ajena"
		}
		if caso == "sin_sha" {
			v.Anclaje.AperturaRegistroSHA256 = ""
		}
		if caso == "parcial" {
			v.Anclaje = nil
		}
		b, _ := json.Marshal(v)
		if caso == "persona" {
			b = append(b[:len(b)-1], []byte(`,"persona":{"ref":"per_inesperada"}}`)...)
		}
		if caso == "exceso" {
			b = bytes.Repeat([]byte("x"), 17<<10)
		}
		if caso == "dos_json" {
			b = append(b, []byte(`{}`)...)
		}
		tx := &txPersonaAceptacionCTPrueba{fila: filaPersonaAceptacionCTPrueba{datos: b}}
		if caso == "ajustes" {
			tx.execErr = errors.New("privado")
		}
		if caso == "commit" {
			tx.commitErr = errors.New("privado")
		}
		esperado := ports.ErrConsultaAnclajeAceptacionCTNoDisponible
		if caso == "denegado" {
			tx.fila = filaPersonaAceptacionCTPrueba{err: &pgconn.PgError{Code: "42501", Message: "privado"}}
			esperado = ports.ErrConsultaAnclajeAceptacionCTDenegada
		}
		pool := &poolPersonaAceptacionCTPrueba{transacciones: []*txPersonaAceptacionCTPrueba{tx}}
		r := &RepositorioConsultaAnclajeAceptacionCTPostgreSQL{pool: pool, ahora: func() time.Time { return ahora }}
		v, err := r.ConsultarAnclajeAceptacionCT(context.Background(), o)
		if err != esperado || v.Estado != "" || tx.rollbacks != 1 || (caso != "commit" && tx.commits != 0) {
			t.Fatalf("caso=%s error=%v", caso, err)
		}
	}
}
func TestAnclajeAceptacionCTPostgreSQLReintentaSoloSerializacionReal(t *testing.T) {
	for _, rutina := range []string{"ExecUpdate", "exec_stmt_raise"} {
		o, ahora := ordenAnclajeCTPrueba(t)
		b, _ := json.Marshal(resultadoAnclajeCTPrueba(o, ahora))
		tx1 := &txPersonaAceptacionCTPrueba{fila: filaPersonaAceptacionCTPrueba{err: &pgconn.PgError{Code: "40001", Routine: rutina}}}
		tx2 := &txPersonaAceptacionCTPrueba{fila: filaPersonaAceptacionCTPrueba{datos: b}}
		pool := &poolPersonaAceptacionCTPrueba{transacciones: []*txPersonaAceptacionCTPrueba{tx1, tx2}}
		r := &RepositorioConsultaAnclajeAceptacionCTPostgreSQL{pool: pool, ahora: func() time.Time { return ahora }}
		v, err := r.ConsultarAnclajeAceptacionCT(context.Background(), o)
		if rutina == "ExecUpdate" {
			if err != nil || v.Estado != "acreditado" || pool.inicios != 2 {
				t.Fatal("sin reintento real")
			}
		} else if err != ports.ErrConsultaAnclajeAceptacionCTNoDisponible || v.Estado != "" || pool.inicios != 1 {
			t.Fatal("reintentó SQL declarado")
		}
	}
}
func TestAnclajeAceptacionCTPostgreSQLCaducidadYCancelacionAntesTX(t *testing.T) {
	o, ahora := ordenAnclajeCTPrueba(t)
	pool := &poolPersonaAceptacionCTPrueba{}
	r := &RepositorioConsultaAnclajeAceptacionCTPostgreSQL{pool: pool, ahora: func() time.Time { return ahora.Add(time.Hour) }}
	if _, err := r.ConsultarAnclajeAceptacionCT(context.Background(), o); err != ports.ErrConsultaAnclajeAceptacionCTDenegada || pool.inicios != 0 {
		t.Fatal("caducado llegóSQL")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := r.ConsultarAnclajeAceptacionCT(ctx, o); err != context.Canceled || pool.inicios != 0 {
		t.Fatal("cancelado llegóSQL")
	}
}
