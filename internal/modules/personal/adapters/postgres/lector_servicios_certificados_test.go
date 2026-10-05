package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func ordenServiciosCertificadosPGPrueba(t *testing.T) ports.OrdenLectorServiciosCertificados {
	t.Helper()
	base := ordenFichaPropiaPrueba(t)
	m, e := domain.NuevoMaterialLectorServiciosCertificados(domain.SolicitudLectorServiciosCertificados{Actor: base.Material.Actor(), EmpleadoRef: base.Material.EmpleadoRef(), OrganismoRef: "dipgra", Corte: base.Material.Corte()})
	if e != nil {
		t.Fatal(e)
	}
	b := base.Autorizacion
	x := b.ResumenCapacidad()
	h, _ := m.HuellaSHA256()
	res, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(x.DecisionRef(), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), ports.AccionServiciosParaCertificadosV1, m.EmpleadoRef(), h, ports.AudienciaServiciosParaCertificadosV1, x.EmitidaEn(), x.ExpiraEn())
	if e != nil {
		t.Fatal(e)
	}
	a, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(b.CapacidadCanonica(), res, b.DecisionCanonica(), b.MotivoCanonico(), b.ContextoActorCanonico(), b.PersonaVersion(), b.PerfilVersion(), b.PayloadVECAD3(), b.SobreCOSESign1(), b.EvidenciaVerificacion(), b.RaizPublicaSPKI())
	if e != nil {
		t.Fatal(e)
	}
	return ports.OrdenLectorServiciosCertificados{Material: m, Autorizacion: a}
}

// respuestaServiciosCertificadosPGPrueba reproduce la forma exacta que
// devuelve Personal36, con n servicios ordenados por inicio y referencia.
func respuestaServiciosCertificadosPGPrueba(o ports.OrdenLectorServiciosCertificados, n int) []byte {
	x := o.Autorizacion.ResumenCapacidad()
	c := o.Material.Corte()
	filas := make([]string, 0, n)
	for i := range n {
		filas = append(filas, fmt.Sprintf(`{"servicio_ref":"srv_%024d","relacion_ref":"rel_%s","version":1,"periodo":{"desde":"2019-01-01","hasta":"2019-12-31"},"dias_reconocidos":364,"estado":"reconocido","clase_ref":"interinidad","clase_version":2,"procedencia":{"acto_ref":"acto:servicios","fuente_ref":"fuente:personal","fuente_version":"3","certeza":"no_acreditado"}}`, i, strings.Repeat("B", 24)))
	}
	huella := strings.Repeat("d", 64)
	return []byte(fmt.Sprintf(`{"servicios":{"empleado_ref":%q,"organismo_ref":"dipgra","version":%d,"corte":{"vigente_en":%q,"conocido_en":%q},"cobertura":"no_acreditada","servicios":[%s]},"evidencia":{"recibo_ref":"aud_v3_%s","decision_ref":%q,"efecto_ref":%q,"consumo_huella_sha256":%q,"auditoria_ref":"aud_v3_%s","consultada_en":%q}}`,
		o.Material.EmpleadoRef(), n+1, c.VigenteEn, c.ConocidoEn.UTC().Format("2006-01-02T15:04:05.000000Z"), strings.Join(filas, ","),
		huella[:32], x.DecisionRef(), x.EfectoRef(), huella, huella[:32], x.EmitidaEn().Add(time.Microsecond).UTC().Format("2006-01-02T15:04:05.000000Z")))
}

func TestServiciosCertificadosPGConfirmaSoloFachadaYRespuestaValidada(t *testing.T) {
	o := ordenServiciosCertificadosPGPrueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaServiciosCertificadosPGPrueba(o, 2)}}}
	pool := &poolP{tx: tx}
	r, _ := nuevoRepositorioLectorServiciosCertificados(pool)
	out, e := r.ConsultarServiciosParaCertificados(context.Background(), o)
	if e != nil || len(out.Servicios) != 2 || out.Servicios[0].DiasReconocidos != 364 || out.Servicios[1].ClaseVersion != 2 || tx.commits != 1 || tx.rollbacks != 0 ||
		pool.o.IsoLevel != pgx.Serializable || tx.q[1] != consultaLectorServiciosCertificadosSQL || !bytes.Equal([]byte(tx.a[0][0].(string)), o.Material.Canonico()) {
		t.Fatal("lectura no nominal", e)
	}
}

func TestServiciosCertificadosPGRespuestaAjenaOBytesInvalidosRevierte(t *testing.T) {
	o := ordenServiciosCertificadosPGPrueba(t)
	base := respuestaServiciosCertificadosPGPrueba(o, 1)
	for nombre, b := range map[string][]byte{
		"decision":  bytes.Replace(base, []byte(`"decision_ref":"dec_prueba"`), []byte(`"decision_ref":"dec_ajena"`), 1),
		"organismo": bytes.Replace(base, []byte(`"organismo_ref":"dipgra"`), []byte(`"organismo_ref":"otro"`), 1),
		"null":      bytes.Replace(base, []byte(`"dias_reconocidos":364`), []byte(`"dias_reconocidos":null`), 1),
		"campo":     bytes.Replace(base, []byte(`"procedencia":{`), []byte(`"procedencia":{"documento":"supuesto",`), 1),
		"duplicada": bytes.Replace(base, []byte(`"cobertura":"no_acreditada"`), []byte(`"cobertura":"no_acreditada","cobertura":"completa"`), 1),
		"certeza":   bytes.Replace(base, []byte(`"certeza":"no_acreditado"`), []byte(`"certeza":"acreditado"`), 1),
		"bytes":     bytes.Repeat([]byte("x"), maxRespuestaLectorServiciosCertificados+1),
	} {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{b}}}
			r, _ := nuevoRepositorioLectorServiciosCertificados(&poolP{tx: tx})
			out, e := r.ConsultarServiciosParaCertificados(context.Background(), o)
			if !errors.Is(e, domain.ErrLectorServiciosCertificadosNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 || out.Servicios != nil {
				t.Fatal("confirmó bytes ajenos", e)
			}
		})
	}
}

func TestServiciosCertificadosPGPermisoAjenoNoIniciaFuenteYErroresSinDatos(t *testing.T) {
	o := ordenServiciosCertificadosPGPrueba(t)
	o.Autorizacion = ordenFichaPropiaPrueba(t).Autorizacion
	pool := &poolP{tx: &txP{}}
	r, _ := nuevoRepositorioLectorServiciosCertificados(pool)
	if _, e := r.ConsultarServiciosParaCertificados(context.Background(), o); !errors.Is(e, domain.ErrLectorServiciosCertificadosInvalido) || pool.n != 0 {
		t.Fatal("permiso de ficha propia prestado", e)
	}
	for _, caso := range []struct {
		codigo string
		err    error
	}{{"42501", domain.ErrLectorServiciosCertificadosDenegado}, {"54000", domain.ErrLectorServiciosCertificadosExcedeLimite}, {"55000", domain.ErrLectorServiciosCertificadosNoDisponible}, {"40001", domain.ErrLectorServiciosCertificadosNoDisponible}} {
		o := ordenServiciosCertificadosPGPrueba(t)
		tx := &txP{errQ: &pgconn.PgError{Code: caso.codigo, Message: "privado"}}
		r, _ := nuevoRepositorioLectorServiciosCertificados(&poolP{tx: tx})
		out, e := r.ConsultarServiciosParaCertificados(context.Background(), o)
		if !errors.Is(e, caso.err) || tx.commits != 0 || tx.rollbacks != 1 || out.Servicios != nil || strings.Contains(e.Error(), "privado") {
			t.Fatal(caso.codigo, e)
		}
	}
}

func TestServiciosCertificadosPGExcesoRevierteSinTruncar(t *testing.T) {
	o := ordenServiciosCertificadosPGPrueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaServiciosCertificadosPGPrueba(o, domain.LimiteLectorServiciosCertificados+1)}}}
	r, _ := nuevoRepositorioLectorServiciosCertificados(&poolP{tx: tx})
	out, e := r.ConsultarServiciosParaCertificados(context.Background(), o)
	if !errors.Is(e, domain.ErrLectorServiciosCertificadosExcedeLimite) || tx.commits != 0 || tx.rollbacks != 1 || out.Servicios != nil {
		t.Fatal("truncó o confirmó exceso", e)
	}
}
