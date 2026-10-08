package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func ordenHistoriaPGPrueba(t *testing.T) ports.OrdenHistoriaServiciosPropia {
	t.Helper()
	base := ordenFichaPropiaPrueba(t)
	m, e := domain.NuevoMaterialHistoriaServiciosPropia(domain.SolicitudHistoriaServiciosPropia{Actor: base.Material.Actor(), Corte: domain.CorteHistoriaServiciosPropia{Desde: "2020-01-01", Hasta: "2027-01-01", ConocidoEn: base.Material.Corte().ConocidoEn}})
	if e != nil {
		t.Fatal(e)
	}
	b := base.Autorizacion
	x := b.ResumenCapacidad()
	h, _ := m.HuellaSHA256()
	res, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(x.DecisionRef(), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionHistoriaServiciosPropia, m.EmpleadoRef(), h, domain.AudienciaHistoriaServiciosPropia, x.EmitidaEn(), x.ExpiraEn())
	if e != nil {
		t.Fatal(e)
	}
	a, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(b.CapacidadCanonica(), res, b.DecisionCanonica(), b.MotivoCanonico(), b.ContextoActorCanonico(), b.PersonaVersion(), b.PerfilVersion(), b.PayloadVECAD3(), b.SobreCOSESign1(), b.EvidenciaVerificacion(), b.RaizPublicaSPKI())
	if e != nil {
		t.Fatal(e)
	}
	return ports.OrdenHistoriaServiciosPropia{Material: m, Autorizacion: a}
}
func respuestaHistoriaPGPrueba(t *testing.T, o ports.OrdenHistoriaServiciosPropia) []byte {
	t.Helper()
	x := o.Autorizacion.ResumenCapacidad()
	r := ports.ResultadoHistoriaServiciosPropia{Historia: domain.HistoriaServiciosPropia{EmpleadoRef: o.Material.EmpleadoRef(), Corte: o.Material.Corte(), Cobertura: "parcial", Revisiones: []domain.RevisionServicioPropio{{ServicioRef: "srv_" + strings.Repeat("A", 24), RelacionRef: "rel_" + strings.Repeat("B", 24), PeriodoDesde: "2019-01-01", PeriodoHasta: "2019-12-31", DiasReconocidos: 365, Estado: "reconocido", Clase: "", Traza: domain.TrazaEmpleadoB2{Desde: "2020-01-01", RegistradaEn: o.Material.Corte().ConocidoEn.Add(-time.Hour), Version: 2, ActoRef: "acto:historia", FuenteRef: "fuente:personal", FuenteVersion: 3}}}}, Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "aud_v3_" + strings.Repeat("d", 32), DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "aud_v3_" + strings.Repeat("d", 32), ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}}
	b, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestHistoriaServiciosPGConfirmaSoloFachadaYRespuestaValidada(t *testing.T) {
	o := ordenHistoriaPGPrueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaHistoriaPGPrueba(t, o)}}}
	pool := &poolP{tx: tx}
	r, _ := nuevoRepositorioHistoriaServiciosPropia(pool)
	out, e := r.ConsultarHistoriaServiciosPropia(context.Background(), o)
	if e != nil || len(out.Historia.Revisiones) != 1 || tx.commits != 1 || tx.rollbacks != 0 || pool.o.IsoLevel != pgx.Serializable || tx.q[1] != consultaHistoriaServiciosPropiaSQL || !bytes.Equal([]byte(tx.a[0][0].(string)), o.Material.Canonico()) {
		t.Fatal("lectura no nominal", e)
	}
}
func TestHistoriaServiciosPGRespuestaAjenaOBytesInvalidosRevierte(t *testing.T) {
	o := ordenHistoriaPGPrueba(t)
	base := respuestaHistoriaPGPrueba(t, o)
	for nombre, b := range map[string][]byte{
		"decision":  bytes.Replace(base, []byte(`"decision_ref":"dec_prueba"`), []byte(`"decision_ref":"dec_ajena"`), 1),
		"corte":     bytes.Replace(base, []byte(`"efectos_hasta":"2027-01-01"`), []byte(`"efectos_hasta":"2028-01-01"`), 1),
		"null":      bytes.Replace(base, []byte(`"dias_reconocidos":365`), []byte(`"dias_reconocidos":null`), 1),
		"campo":     bytes.Replace(base, []byte(`"traza":{`), []byte(`"traza":{"documento":"supuesto",`), 1),
		"duplicada": bytes.Replace(base, []byte(`"cobertura":"parcial"`), []byte(`"cobertura":"parcial","cobertura":"completa"`), 1),
		"bytes":     bytes.Repeat([]byte("x"), maxRespuestaFichaEmpleadoB2+1),
	} {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{b}}}
			r, _ := nuevoRepositorioHistoriaServiciosPropia(&poolP{tx: tx})
			out, e := r.ConsultarHistoriaServiciosPropia(context.Background(), o)
			if !errors.Is(e, domain.ErrHistoriaServiciosPropiaNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 || out.Historia.Revisiones != nil {
				t.Fatal("confirmó bytes ajenos", e)
			}
		})
	}
}
func TestHistoriaServiciosPGPermisoRRHHNoIniciaFuenteYErroresSinDatos(t *testing.T) {
	o := ordenHistoriaPGPrueba(t)
	o.Autorizacion = ordenFichaB2Prueba(t).Autorizacion
	pool := &poolP{tx: &txP{}}
	r, _ := nuevoRepositorioHistoriaServiciosPropia(pool)
	if _, e := r.ConsultarHistoriaServiciosPropia(context.Background(), o); !errors.Is(e, domain.ErrHistoriaServiciosPropiaInvalida) || pool.n != 0 {
		t.Fatal("permiso RRHH prestado", e)
	}
	for _, caso := range []struct {
		codigo string
		err    error
	}{{"42501", domain.ErrHistoriaServiciosPropiaDenegada}, {"54000", domain.ErrHistoriaServiciosPropiaExcedeLimite}, {"55000", domain.ErrHistoriaServiciosPropiaNoDisponible}} {
		o := ordenHistoriaPGPrueba(t)
		tx := &txP{errQ: &pgconn.PgError{Code: caso.codigo, Message: "privado"}}
		r, _ := nuevoRepositorioHistoriaServiciosPropia(&poolP{tx: tx})
		out, e := r.ConsultarHistoriaServiciosPropia(context.Background(), o)
		if !errors.Is(e, caso.err) || tx.commits != 0 || tx.rollbacks != 1 || out.Historia.Revisiones != nil || strings.Contains(e.Error(), "privado") {
			t.Fatal(e)
		}
	}
}

func TestHistoriaServiciosPGExcesoRespuestaRevierteSinTruncar(t *testing.T) {
	o := ordenHistoriaPGPrueba(t)
	var respuesta ports.ResultadoHistoriaServiciosPropia
	if json.Unmarshal(respuestaHistoriaPGPrueba(t, o), &respuesta) != nil {
		t.Fatal("fixture inválido")
	}
	fila := respuesta.Historia.Revisiones[0]
	respuesta.Historia.Revisiones = make([]domain.RevisionServicioPropio, domain.LimiteHistoriaServiciosPropia+1)
	for i := range respuesta.Historia.Revisiones {
		respuesta.Historia.Revisiones[i] = fila
	}
	b, e := json.Marshal(respuesta)
	if e != nil {
		t.Fatal(e)
	}
	tx := &txP{fila: filaP{vals: []any{b}}}
	r, _ := nuevoRepositorioHistoriaServiciosPropia(&poolP{tx: tx})
	out, e := r.ConsultarHistoriaServiciosPropia(context.Background(), o)
	if !errors.Is(e, domain.ErrHistoriaServiciosPropiaExcedeLimite) || tx.commits != 0 || tx.rollbacks != 1 || out.Historia.Revisiones != nil {
		t.Fatal("truncó o confirmó exceso", e)
	}
}
