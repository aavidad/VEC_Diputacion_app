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
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func ordenHistoriaRelacionesPGPrueba(t *testing.T) ports.OrdenHistoriaRelacionesPropia {
	t.Helper()
	base := ordenFichaPropiaPrueba(t)
	m, err := domain.NuevoMaterialHistoriaRelacionesPropia(domain.SolicitudHistoriaRelacionesPropia{Actor: base.Material.Actor(), Corte: domain.CorteHistoriaRelacionesPropia{Desde: "2020-01-01", Hasta: "2027-01-01", ConocidoEn: base.Material.Corte().ConocidoEn}})
	if err != nil {
		t.Fatal(err)
	}
	b := base.Autorizacion
	x := b.ResumenCapacidad()
	h, _ := m.HuellaSHA256()
	res, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(x.DecisionRef(), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionHistoriaRelacionesPropia, m.EmpleadoRef(), h, domain.AudienciaHistoriaRelacionesPropia, x.EmitidaEn(), x.ExpiraEn())
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(b.CapacidadCanonica(), res, b.DecisionCanonica(), b.MotivoCanonico(), b.ContextoActorCanonico(), b.PersonaVersion(), b.PerfilVersion(), b.PayloadVECAD3(), b.SobreCOSESign1(), b.EvidenciaVerificacion(), b.RaizPublicaSPKI())
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenHistoriaRelacionesPropia{Material: m, Autorizacion: a}
}

func respuestaHistoriaRelacionesPGPrueba(t *testing.T, o ports.OrdenHistoriaRelacionesPropia) []byte {
	t.Helper()
	x := o.Autorizacion.ResumenCapacidad()
	ref := "aud_v3_" + strings.Repeat("d", 32)
	r := ports.ResultadoHistoriaRelacionesPropia{Historia: domain.HistoriaRelacionesPropia{EmpleadoRef: o.Material.EmpleadoRef(), Corte: o.Material.Corte(), Cobertura: "parcial", Revisiones: []domain.RevisionRelacionPropia{{RelacionRef: "rel_" + strings.Repeat("A", 24), Estado: "vigente", Regimen: "Funcionarial", Modalidad: "Temporal", Unidad: "Unidad sintética", Puesto: "Técnico/a", Situacion: "Servicio activo", Traza: domain.TrazaEmpleadoB2{Desde: "2020-01-01", RegistradaEn: o.Material.Corte().ConocidoEn.Add(-time.Hour), Version: 2, ActoRef: "acto:relacion", FuenteRef: "fuente:personal", FuenteVersion: 3}}}}, Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: ref, DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: ref, ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)}}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestHistoriaRelacionesPGConfirmaSoloFachadaNominal(t *testing.T) {
	o := ordenHistoriaRelacionesPGPrueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaHistoriaRelacionesPGPrueba(t, o)}}}
	pool := &poolP{tx: tx}
	r, _ := nuevoRepositorioHistoriaRelacionesPropia(pool)
	out, err := r.ConsultarHistoriaRelacionesPropia(context.Background(), o)
	if err != nil || len(out.Historia.Revisiones) != 1 || tx.commits != 1 || tx.rollbacks != 0 || pool.o.IsoLevel != pgx.Serializable || tx.q[1] != consultaHistoriaRelacionesPropiaSQL || len(tx.a[0]) != 11 || !bytes.Equal([]byte(tx.a[0][0].(string)), o.Material.Canonico()) {
		t.Fatal("lectura no nominal", err)
	}
}

func TestHistoriaRelacionesPGRechazaRespuestaAntesDeCommit(t *testing.T) {
	o := ordenHistoriaRelacionesPGPrueba(t)
	base := respuestaHistoriaRelacionesPGPrueba(t, o)
	var exceso ports.ResultadoHistoriaRelacionesPropia
	if json.Unmarshal(base, &exceso) != nil {
		t.Fatal("fixture inválido")
	}
	fila := exceso.Historia.Revisiones[0]
	exceso.Historia.Revisiones = make([]domain.RevisionRelacionPropia, domain.LimiteHistoriaRelacionesPropia+1)
	for i := range exceso.Historia.Revisiones {
		exceso.Historia.Revisiones[i] = fila
	}
	masDeDoscientas, _ := json.Marshal(exceso)
	for nombre, b := range map[string][]byte{
		"empleado_ajeno": bytes.Replace(base, []byte(`"empleado_ref":"`+o.Material.EmpleadoRef()+`"`), []byte(`"empleado_ref":"emp_`+strings.Repeat("Z", 24)+`"`), 1),
		"corte_ajeno":    bytes.Replace(base, []byte(`"efectos_hasta":"2027-01-01"`), []byte(`"efectos_hasta":"2028-01-01"`), 1),
		"nulo":           bytes.Replace(base, []byte(`"estado":"vigente"`), []byte(`"estado":null`), 1),
		"campo_extra":    bytes.Replace(base, []byte(`"traza":{`), []byte(`"traza":{"persona_ref":"per_ajena",`), 1),
		"exceso":         masDeDoscientas,
	} {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{b}}}
			r, _ := nuevoRepositorioHistoriaRelacionesPropia(&poolP{tx: tx})
			out, err := r.ConsultarHistoriaRelacionesPropia(context.Background(), o)
			esperado := domain.ErrHistoriaRelacionesPropiaNoDisponible
			if nombre == "exceso" {
				esperado = domain.ErrHistoriaRelacionesPropiaExcedeLimite
			}
			if !errors.Is(err, esperado) || tx.commits != 0 || tx.rollbacks != 1 || out.Historia.Revisiones != nil {
				t.Fatal("confirmó respuesta inválida", err)
			}
		})
	}
}

func TestHistoriaRelacionesPGRechazaPermisoDeFichaYOpacaSQL(t *testing.T) {
	o := ordenHistoriaRelacionesPGPrueba(t)
	o.Autorizacion = ordenFichaPropiaPrueba(t).Autorizacion
	pool := &poolP{tx: &txP{}}
	r, _ := nuevoRepositorioHistoriaRelacionesPropia(pool)
	if _, err := r.ConsultarHistoriaRelacionesPropia(context.Background(), o); !errors.Is(err, domain.ErrHistoriaRelacionesPropiaInvalida) || pool.n != 0 {
		t.Fatal("permiso de ficha reutilizado", err)
	}
	o = ordenHistoriaRelacionesPGPrueba(t)
	tx := &txP{errQ: &pgconn.PgError{Code: "42501", Message: "empleado privado"}}
	r, _ = nuevoRepositorioHistoriaRelacionesPropia(&poolP{tx: tx})
	out, err := r.ConsultarHistoriaRelacionesPropia(context.Background(), o)
	if !errors.Is(err, domain.ErrHistoriaRelacionesPropiaDenegada) || tx.commits != 0 || tx.rollbacks != 1 || out.Historia.Revisiones != nil || strings.Contains(err.Error(), "privado") {
		t.Fatal("error SQL filtrado", err)
	}
}
