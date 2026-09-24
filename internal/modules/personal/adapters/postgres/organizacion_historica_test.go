package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
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

func ordenOrganizacionHistoricaPrueba(t *testing.T) ports.OrdenConsultaOrganizacionHistorica {
	t.Helper()
	actor := ordenP(t).Material.Solicitud().Actor
	fecha, err := domain.NuevaFechaCivil("2026-09-20")
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	material, err := domain.NuevoMaterialConsultaOrganizacionHistorica(domain.SolicitudConsultaOrganizacionHistorica{Actor: actor, Selector: domain.SelectorOrganizacionHistorica{OrganismoRef: "org:dipgra", UnidadClave: "uni:uno", VigenteEn: fecha, ConocidoEn: ahora, Limite: 10}})
	if err != nil {
		t.Fatal(err)
	}
	huella, err := material.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), domain.AccionConsultaOrganizacionHistorica, material.Recurso().Referencia, huella, domain.AudienciaConsultaOrganizacionHistorica, ahora, ahora.Add(3*time.Second))
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
	autorizacion, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return ports.OrdenConsultaOrganizacionHistorica{Material: material, Autorizacion: autorizacion}
}

func respuestaOrganizacionVacia(t *testing.T, o ports.OrdenConsultaOrganizacionHistorica) []byte {
	t.Helper()
	c := ports.CoberturaFuentesOrganizacionHistorica{Unidades: "sin_datos", PuestosTipo: "sin_datos", Dotaciones: "sin_datos", Plazas: "sin_datos", PuestosIndividuales: "sin_datos", Vinculos: "sin_datos"}
	r := ports.ResultadoConsultaOrganizacionHistorica{Pagina: ports.PaginaOrganizacionHistorica{Selector: o.Material.Solicitud().Selector, Cobertura: c, Unidades: []domain.UnidadOrganizacionHistorica{}, PuestosTipo: []domain.PuestoTipoOrganizacionHistorica{}, Dotaciones: []domain.DotacionOrganizacionHistorica{}, Plazas: []domain.PlazaOrganizacionHistorica{}, PuestosIndividuales: []domain.PuestoIndividualOrganizacionHistorica{}, Vinculos: []domain.VinculoPlazaPuestoHistorico{}}, Evidencia: ports.EvidenciaConsultaOrganizacionHistorica{ReciboRef: "recibo:prueba", DecisionRef: o.Autorizacion.ResumenCapacidad().DecisionRef(), EfectoRef: o.Autorizacion.ResumenCapacidad().EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria:prueba", ConsultadaEn: o.Autorizacion.ResumenCapacidad().EmitidaEn().Add(time.Microsecond)}}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRepositorioOrganizacionHistoricaConsumeNominalYConfirma(t *testing.T) {
	o := ordenOrganizacionHistoricaPrueba(t)
	tx := &txP{fila: filaP{vals: []any{respuestaOrganizacionVacia(t, o)}}}
	p := &poolP{tx: tx}
	r, err := nuevoRepositorioOrganizacionHistoricaPostgreSQL(p)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := r.ConsultarOrganizacionHistorica(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Pagina.Selector != o.Material.Solicitud().Selector || tx.commits != 1 || tx.rollbacks != 0 || p.o.IsoLevel != pgx.Serializable || p.o.AccessMode != pgx.ReadWrite {
		t.Fatal("transacción o selector incorrectos")
	}
	if len(tx.q) != 2 || tx.q[1] != consultaOrganizacionHistorica || len(tx.a) != 1 || len(tx.a[0]) != 11 || !strings.Contains(tx.q[1], "vec_personal.consultar_organizacion_historica_v1(") || strings.Contains(strings.ToLower(tx.q[1]), " from ") {
		t.Fatal("no se consumió la función nominal")
	}
	if tx.a[0][0] != string(o.Material.Canonico()) || tx.a[0][5] != int64(o.Autorizacion.PersonaVersion()) || tx.a[0][6] != int64(o.Autorizacion.PerfilVersion()) {
		t.Fatal("material V3 desalineado")
	}
}

func TestRepositorioOrganizacionHistoricaDeniegaAntesDeSQLSinConcesion(t *testing.T) {
	o := ordenOrganizacionHistoricaPrueba(t)
	o.Autorizacion = vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	p := &poolP{tx: &txP{}}
	r, _ := nuevoRepositorioOrganizacionHistoricaPostgreSQL(p)
	_, err := r.ConsultarOrganizacionHistorica(context.Background(), o)
	if !errors.Is(err, domain.ErrConsultaOrganizacionHistoricaInvalida) || p.n != 0 {
		t.Fatal("autorización ausente llegó a SQL")
	}
}

func TestRepositorioOrganizacionHistoricaRevierteRespuestaAlterada(t *testing.T) {
	o := ordenOrganizacionHistoricaPrueba(t)
	casos := map[string][]byte{
		"desconocido":     bytes.Replace(respuestaOrganizacionVacia(t, o), []byte(`"pagina":{`), []byte(`"pagina":{"intruso":true,`), 1),
		"duplicado":       bytes.Replace(respuestaOrganizacionVacia(t, o), []byte(`"pagina":{`), []byte(`"pagina":{"version_rpt_ref":"x","version_rpt_ref":"",`), 1),
		"coleccion_nula":  bytes.Replace(respuestaOrganizacionVacia(t, o), []byte(`"unidades":[]`), []byte(`"unidades":null`), 1),
		"evidencia_ajena": bytes.Replace(respuestaOrganizacionVacia(t, o), []byte(`"decision_ref":"dec_prueba"`), []byte(`"decision_ref":"dec_ajena"`), 1),
		"nulo":            []byte(`null`),
		"excesivo":        bytes.Repeat([]byte("x"), maxRespuestaOrganizacionHistorica+1),
	}
	for nombre, bruto := range casos {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{bruto}}}
			p := &poolP{tx: tx}
			r, _ := nuevoRepositorioOrganizacionHistoricaPostgreSQL(p)
			_, err := r.ConsultarOrganizacionHistorica(context.Background(), o)
			if !errors.Is(err, domain.ErrOrganizacionHistoricaNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("se confirmó respuesta inválida", err)
			}
		})
	}
}

func TestRepositorioOrganizacionHistoricaPropagaCancelacion(t *testing.T) {
	o := ordenOrganizacionHistoricaPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	tx := &txP{fila: filaP{vals: []any{respuestaOrganizacionVacia(t, o)}}, cancelQuery: cancelar}
	p := &poolP{tx: tx}
	r, _ := nuevoRepositorioOrganizacionHistoricaPostgreSQL(p)
	_, err := r.ConsultarOrganizacionHistorica(ctx, o)
	if !errors.Is(err, context.Canceled) || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("cancelación no revirtió", err)
	}
}

func TestDecodificarOrganizacionHistoricaAceptaOffsetCeroSQL(t *testing.T) {
	o := ordenOrganizacionHistoricaPrueba(t)
	bruto := bytes.ReplaceAll(respuestaOrganizacionVacia(t, o), []byte(`Z"`), []byte(`+00:00"`))
	if _, err := decodificarOrganizacionHistorica(bruto, o); err != nil {
		t.Fatal(err)
	}
	bruto = bytes.Replace(bruto, []byte(`+00:00"`), []byte(`+01:00"`), 1)
	if _, err := decodificarOrganizacionHistorica(bruto, o); err == nil {
		t.Fatal("offset ajeno aceptado")
	}
}

func TestRepositorioOrganizacionHistoricaDistingueDenegacionNominal(t *testing.T) {
	o := ordenOrganizacionHistoricaPrueba(t)
	tx := &txP{errQ: &pgconn.PgError{Code: "42501", Message: "detalle privado"}}
	p := &poolP{tx: tx}
	r, _ := nuevoRepositorioOrganizacionHistoricaPostgreSQL(p)
	_, err := r.ConsultarOrganizacionHistorica(context.Background(), o)
	if !errors.Is(err, domain.ErrConsultaOrganizacionHistoricaDenegada) || tx.commits != 0 || tx.rollbacks != 1 || strings.Contains(err.Error(), "privado") {
		t.Fatal("denegación nominal mal traducida", err)
	}
}
