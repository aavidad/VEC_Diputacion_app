package postgres

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func ordenExportacionPrueba(t *testing.T) ports.OrdenExportacionServiciosPropios {
	t.Helper()
	base := ordenFichaPropiaPrueba(t)
	f, e := domain.NuevoFormatoExportacionServiciosPropios(domain.DatosFormatoExportacionServiciosPropios{Referencia: "personal:servicios_propios:csv", Version: 1, Idioma: "xx", CatalogoSHA256: strings.Repeat("a", 64), NombreArchivo: "servicios.csv", Cabeceras: []string{"c1", "c2", "c3", "c4", "c5"}, Estados: map[string]string{"declarado": "e1", "comprobado": "e2", "reconocido": "e3"}})
	if e != nil {
		t.Fatal(e)
	}
	m, e := domain.NuevoMaterialExportacionServiciosPropios(domain.SolicitudExportacionServiciosPropios{Actor: base.Material.Actor(), ReciboRef: "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100", Corte: base.Material.Corte(), Idioma: "xx"}, f)
	if e != nil {
		t.Fatal(e)
	}
	h, _ := m.HuellaSHA256()
	x := base.Autorizacion.ResumenCapacidad()
	a, e := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(x.DecisionRef(), strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionExportacionServiciosPropios, m.EmpleadoRef(), h, domain.AudienciaExportacionServiciosPropios, x.EmitidaEn(), x.ExpiraEn())
	if e != nil {
		t.Fatal(e)
	}
	b := base.Autorizacion
	exp, e := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(b.CapacidadCanonica(), a, b.DecisionCanonica(), b.MotivoCanonico(), b.ContextoActorCanonico(), b.PersonaVersion(), b.PerfilVersion(), b.PayloadVECAD3(), b.SobreCOSESign1(), b.EvidenciaVerificacion(), b.RaizPublicaSPKI())
	if e != nil {
		t.Fatal(e)
	}
	return ports.OrdenExportacionServiciosPropios{Material: m, Autorizacion: exp}
}
func respuestaExportacionPrueba(t *testing.T, o ports.OrdenExportacionServiciosPropios) []byte {
	t.Helper()
	r := struct {
		Corte     domain.CorteEmpleadoB2            `json:"corte"`
		Servicios []domain.ServicioFichaPropia      `json:"servicios"`
		Evidencia ports.EvidenciaRegistroEmpleadoB2 `json:"evidencia"`
	}{o.Material.Corte(), []domain.ServicioFichaPropia{{Inicio: "2019-01-01", Fin: "2019-12-31", Clase: " =suma(1)", Dias: 365, Estado: "reconocido"}}, ports.EvidenciaRegistroEmpleadoB2{ReciboRef: o.Material.Solicitud().ReciboRef, DecisionRef: o.Autorizacion.ResumenCapacidad().DecisionRef(), EfectoRef: o.Material.EmpleadoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "auditoria:export", ConsultadaEn: o.Autorizacion.ResumenCapacidad().EmitidaEn().Add(time.Microsecond)}}
	b, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestExportacionServiciosCSVConfirmaSoloTrasSerializarFuenteExacta(t *testing.T) {
	o := ordenExportacionPrueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaExportacionPrueba(t, o)}}}
	pool := &poolP{tx: tx}
	repo, _ := nuevoRepositorioExportacionServiciosPropios(pool)
	r, e := repo.ExportarServiciosPropios(context.Background(), o)
	if e != nil || tx.commits != 1 || tx.rollbacks != 0 || pool.o.IsoLevel != pgx.Serializable || tx.q[1] != exportacionServiciosPropiosSQL || !application.ResultadoExportacionServiciosPropiosValido(o.Material, o.Autorizacion, r) {
		t.Fatal("no confirmó CSV exacto", e)
	}
	filas, e := csv.NewReader(bytes.NewReader(r.ContenidoCSV)).ReadAll()
	if e != nil || len(filas) != 2 || filas[1][2] != "' =suma(1)" || filas[0][0] != "c1" || filas[1][4] != "e3" {
		t.Fatal("CSV no neutro", e)
	}
}
func TestExportacionServiciosRespuestaAlteradaRevierteSinCSV(t *testing.T) {
	o := ordenExportacionPrueba(t)
	base := respuestaExportacionPrueba(t, o)
	for nombre, b := range map[string][]byte{"null": bytes.Replace(base, []byte(`"dias":365`), []byte(`"dias":null`), 1), "otra_decision": bytes.Replace(base, []byte(`"dec_prueba"`), []byte(`"dec_ajena"`), 1), "otra_fecha": bytes.Replace(base, []byte(`"vigente_en":"2026-09-25"`), []byte(`"vigente_en":"2026-09-24"`), 1), "campoajeno": bytes.Replace(base, []byte(`"servicios":[{`), []byte(`"servicios":[{"persona_ref":"per_x",`), 1)} {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{b}}}
			repo, _ := nuevoRepositorioExportacionServiciosPropios(&poolP{tx: tx})
			r, e := repo.ExportarServiciosPropios(context.Background(), o)
			if !errors.Is(e, domain.ErrExportacionServiciosPropiosNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 || len(r.ContenidoCSV) != 0 {
				t.Fatal("fuente alterada confirmada", e)
			}
		})
	}
}
func TestExportacionServiciosConsultaNoAutorizaNiSQLFiltraDatos(t *testing.T) {
	o := ordenExportacionPrueba(t)
	o.Autorizacion = ordenFichaPropiaPrueba(t).Autorizacion
	pool := &poolP{tx: &txP{}}
	repo, _ := nuevoRepositorioExportacionServiciosPropios(pool)
	if _, e := repo.ExportarServiciosPropios(context.Background(), o); !errors.Is(e, domain.ErrExportacionServiciosPropiosInvalida) || pool.n != 0 {
		t.Fatal("consulta usada para exportar", e)
	}
	o = ordenExportacionPrueba(t)
	tx := &txP{errQ: &pgconn.PgError{Code: "42501", Message: "recibo ajeno"}}
	repo, _ = nuevoRepositorioExportacionServiciosPropios(&poolP{tx: tx})
	if r, e := repo.ExportarServiciosPropios(context.Background(), o); !errors.Is(e, domain.ErrExportacionServiciosPropiosDenegada) || len(r.ContenidoCSV) != 0 || tx.rollbacks != 1 || strings.Contains(e.Error(), "ajeno") {
		t.Fatal("no denegó opaco", e)
	}
}
func TestExportacionServiciosNeutralizaFormulasInclusoTrasEspacios(t *testing.T) {
	for _, s := range []string{"=1", "+1", "-1", "@a", " \t\r\n=1", "\u200b@a", "\ufeff+1"} {
		if got := celdaCSVServiciosPropios(s); got != "'"+s {
			t.Fatalf("formula sin protección %q", s)
		}
	}
	if celdaCSVServiciosPropios("texto, \"dato\"") != "texto, \"dato\"" {
		t.Fatal("dato alterado")
	}
}
