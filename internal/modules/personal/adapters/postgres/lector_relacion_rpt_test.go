package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
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

func ordenLectorRPTPrueba(t *testing.T) ports.OrdenLectorRelacionRPT {
	t.Helper()
	base := ordenFichaPropiaPrueba(t)
	actor := base.Material.Actor()
	m, err := domain.NuevoMaterialLectorRelacionRPT(domain.SolicitudLectorRelacionRPT{Actor: actor, EmpleadoRef: "emp_" + strings.Repeat("b", 24), RelacionRef: "rel_" + strings.Repeat("c", 24), OrganismoRef: "organismo:dipgra", VersionEsperada: 2, Corte: base.Material.Corte()})
	if err != nil {
		t.Fatal(err)
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_crn11", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), ports.AccionRelacionParaRPTV1, m.Solicitud().RelacionRef, h, ports.AudienciaRelacionParaRPTV1, actor.ResueltoEn, actor.ResueltoEn.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenLectorRelacionRPT{Material: m, Autorizacion: a}
}
func respuestaLectorRPTPrueba(t *testing.T, o ports.OrdenLectorRelacionRPT) []byte {
	t.Helper()
	m := o.Material.Solicitud()
	x := o.Autorizacion.ResumenCapacidad()
	w := resultadoRelacionRPTWire{Corte: m.Corte, Cobertura: ports.CoberturaPersonalNoAcreditadaV1, Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:rpt:1", DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "auditoria:rpt:1", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}}
	w.Relacion.EmpleadoRef = m.EmpleadoRef
	w.Relacion.RelacionRef = m.RelacionRef
	w.Relacion.OrganismoRef = m.OrganismoRef
	w.Relacion.Version = m.VersionEsperada
	w.Relacion.Estado = "finalizada"
	w.Relacion.Periodo.Desde = "2020-01-01"
	w.Relacion.Periodo.Hasta = ""
	w.Relacion.Procedencia.ActoRef = "acto:prueba"
	w.Relacion.Procedencia.FuenteRef = "fuente:prueba"
	w.Relacion.Procedencia.FuenteVersion = "2"
	w.Relacion.Procedencia.Certeza = ports.CertezaPersonalNoAcreditadaV1
	b, err := json.Marshal(w)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestLectorRPTPostgresNominalValidaAntesCommit(t *testing.T) {
	o := ordenLectorRPTPrueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaLectorRPTPrueba(t, o)}}}
	pool := &poolP{tx: tx}
	r, err := nuevoRepositorioLectorRPT(pool)
	if err != nil {
		t.Fatal(err)
	}
	result, err := r.ConsultarRelacionParaRPT(context.Background(), o)
	if err != nil || result.Relacion.Estado != "finalizada" || result.Relacion.Periodo.Hasta != "" || pool.o.IsoLevel != pgx.Serializable || pool.o.AccessMode != pgx.ReadWrite || tx.commits != 1 || tx.rollbacks != 0 || tx.q[1] != consultaLectorRelacionRPTSQL || len(tx.a[0]) != 11 {
		t.Fatal("consulta nominal no confirmada", err)
	}
}
func TestLectorRPTPostgresRevierteWireDivergente(t *testing.T) {
	o := ordenLectorRPTPrueba(t)
	base := respuestaLectorRPTPrueba(t, o)
	for nombre, b := range map[string][]byte{
		"version":       bytes.Replace(base, []byte(`"version":2`), []byte(`"version":1`), 1),
		"otra_terna":    bytes.Replace(base, []byte(o.Material.Solicitud().EmpleadoRef), []byte("emp_"+strings.Repeat("x", 24)), 1),
		"certeza":       bytes.Replace(base, []byte(`"certeza":"no_acreditado"`), []byte(`"certeza":"acreditado"`), 1),
		"cobertura":     bytes.Replace(base, []byte(`"cobertura":"no_acreditada"`), []byte(`"cobertura":"completa"`), 1),
		"hasta_nula":    bytes.Replace(base, []byte(`"hasta":""`), []byte(`"hasta":null`), 1),
		"hasta_omitida": bytes.Replace(base, []byte(`,"hasta":""`), nil, 1),
		"extra":         bytes.Replace(base, []byte(`"relacion":{`), []byte(`"relacion":{"persona_ref":"per_privada",`), 1),
		"duplicado":     bytes.Replace(base, []byte(`"version":2`), []byte(`"version":2,"version":2`), 1),
		"sin_auditoria": bytes.Replace(base, []byte(`"auditoria_ref":"auditoria:rpt:1",`), nil, 1),
	} {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{b}}}
			r, _ := nuevoRepositorioLectorRPT(&poolP{tx: tx})
			result, err := r.ConsultarRelacionParaRPT(context.Background(), o)
			if !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) || result != (ports.ResultadoRelacionParaRPTV1{}) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("wire divergente confirmado", err)
			}
		})
	}
}
func TestLectorRPTPostgresPermisoPrestadoNoLlegaSQL(t *testing.T) {
	o := ordenLectorRPTPrueba(t)
	o.Autorizacion = ordenFichaPropiaPrueba(t).Autorizacion
	pool := &poolP{tx: &txP{}}
	r, _ := nuevoRepositorioLectorRPT(pool)
	if _, err := r.ConsultarRelacionParaRPT(context.Background(), o); !errors.Is(err, domain.ErrLectorRelacionRPTInvalido) || pool.n != 0 {
		t.Fatal("permiso ficha usado por RPT", err)
	}
}
func TestLectorRPTPostgresErrorRollbackNoDevuelveDatos(t *testing.T) {
	for _, fase := range []string{"consulta", "commit"} {
		t.Run(fase, func(t *testing.T) {
			o := ordenLectorRPTPrueba(t)
			tx := &txP{fila: filaP{vals: []any{respuestaLectorRPTPrueba(t, o)}}}
			e := &pgconn.PgError{Code: "40001", Message: "detalle privado"}
			if fase == "consulta" {
				tx.errQ = e
			} else {
				tx.errC = e
			}
			r, _ := nuevoRepositorioLectorRPT(&poolP{tx: tx})
			result, err := r.ConsultarRelacionParaRPT(context.Background(), o)
			if !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) || result != (ports.ResultadoRelacionParaRPTV1{}) || tx.rollbacks != 1 || strings.Contains(err.Error(), "privado") {
				t.Fatal("fallo no cerrado", err)
			}
		})
	}
}
