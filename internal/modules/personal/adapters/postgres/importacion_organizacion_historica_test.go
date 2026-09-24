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

func ordenImportacionOrganizacionPrueba(t *testing.T) ports.OrdenImportacionOrganizacion {
	t.Helper()
	actor := ordenOrganizacionHistoricaPrueba(t).Material.Solicitud().Actor
	h := strings.Repeat("a", 64)
	uuidVersion := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	uuidHecho := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	m := domain.ManifiestoImportacionOrganizacion{OrganismoRef: "organismo:dipgra", Tipo: "rpt", VersionRef: uuidVersion, VersionRevision: 1, FuenteRef: "fuente:prueba", FuenteVersion: "v1", FuenteHuellaSHA256: h,
		CatalogoUnidades: domain.ReferenciaCatalogoImportacion{ID: "estructura-organizativa-dipgra", Version: 1, Revision: 1, HuellaSHA256: h}, CatalogoClasificaciones: domain.ReferenciaCatalogoImportacion{ID: "clasificaciones-dipgra", Version: 1, Revision: 1, HuellaSHA256: h}}
	hecho := domain.HechoImportacionOrganizacion{Clase: "nodo", HechoRef: uuidHecho, Revision: 1, FilaFuenteRef: "fila:uno", OrganismoRef: m.OrganismoRef, UnidadRef: "unidad:uno", VigenteDesde: "2026-01-01", CatalogoEntradaClave: "unidad:uno", Denominacion: "Unidad sintética", TipoUnidad: "centro"}
	s := domain.SolicitudImportacionOrganizacion{Fase: domain.FasePrepararOrganizacion, ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11", CorrelacionRef: "corr:uno", Manifiesto: m, Hechos: []domain.HechoImportacionOrganizacion{hecho}, Actor: actor}
	material, err := domain.NuevoMaterialImportacionOrganizacion(s)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := material.Recurso().HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_importacion", strings.Repeat("b", 64), strings.Repeat("c", 64), "ctx_importacion", strings.Repeat("d", 64), s.Fase.Accion(), material.Recurso().Referencia, huella, domain.AudienciaImportacionOrganizacion, ahora, ahora.Add(3*time.Second))
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
	return ports.OrdenImportacionOrganizacion{Material: material, Autorizacion: a}
}

func reciboImportacionPrueba(t *testing.T, o ports.OrdenImportacionOrganizacion) []byte {
	t.Helper()
	s := o.Material.Solicitud()
	r := ports.ReciboImportacionOrganizacion{ReciboRef: "recibo:importacion", LoteRef: "lote:uno", Fase: s.Fase, Estado: "preparacion_no_autoritativa", RevisionAnterior: 0, RevisionNueva: 1, ClaveIdempotencia: s.ClaveIdempotencia, MaterialHuellaSHA256: o.Material.HuellaSHA256(), FuenteHuellaSHA256: s.Manifiesto.FuenteHuellaSHA256, ActorRef: s.Actor.Principal.ID, DecisionRef: o.Autorizacion.ResumenCapacidad().DecisionRef(), AuditoriaRef: "auditoria:uno", RegistradoEn: o.Autorizacion.ResumenCapacidad().EmitidaEn().Add(time.Microsecond)}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRepositorioImportacionOrganizacionConsumeNominalYConfirma(t *testing.T) {
	o := ordenImportacionOrganizacionPrueba(t)
	tx := &txP{fila: filaP{vals: []any{reciboImportacionPrueba(t, o)}}}
	p := &poolP{tx: tx}
	r, err := nuevoRepositorioImportacionOrganizacionPostgreSQL(p)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := r.Ejecutar(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	if recibo.Replay || recibo.RevisionNueva != 1 || tx.commits != 1 || tx.rollbacks != 0 || p.o.IsoLevel != pgx.Serializable || p.o.AccessMode != pgx.ReadWrite {
		t.Fatal("transacción o recibo incorrectos")
	}
	if len(tx.q) != 2 || tx.q[1] != ejecutarImportacionOrganizacion || len(tx.a) != 1 || len(tx.a[0]) != 12 || tx.a[0][1] != nil || tx.a[0][0] != string(o.Material.Canonico()) || strings.Contains(strings.ToLower(tx.q[1]), " from ") {
		t.Fatal("firma SQL o parámetros incorrectos")
	}
}

func TestRepositorioImportacionOrganizacionDeniegaSinConcesionAntesDeSQL(t *testing.T) {
	o := ordenImportacionOrganizacionPrueba(t)
	o.Autorizacion = vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	p := &poolP{tx: &txP{}}
	r, _ := nuevoRepositorioImportacionOrganizacionPostgreSQL(p)
	_, err := r.Ejecutar(context.Background(), o)
	if !errors.Is(err, domain.ErrImportacionOrganizacionInvalida) || p.n != 0 {
		t.Fatal("autorización ausente llegó a SQL", err)
	}
}

func TestRepositorioImportacionOrganizacionRevierteReciboAlterado(t *testing.T) {
	o := ordenImportacionOrganizacionPrueba(t)
	base := reciboImportacionPrueba(t, o)
	casos := map[string][]byte{
		"desconocido": bytes.Replace(base, []byte(`"recibo_ref":`), []byte(`"intruso":true,"recibo_ref":`), 1),
		"duplicado":   bytes.Replace(base, []byte(`"recibo_ref":`), []byte(`"recibo_ref":"x","recibo_ref":`), 1),
		"huella":      bytes.Replace(base, []byte(o.Material.HuellaSHA256()), []byte(strings.Repeat("f", 64)), 1),
		"nulo":        []byte(`null`),
	}
	for nombre, bruto := range casos {
		t.Run(nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{bruto}}}
			p := &poolP{tx: tx}
			r, _ := nuevoRepositorioImportacionOrganizacionPostgreSQL(p)
			_, err := r.Ejecutar(context.Background(), o)
			if !errors.Is(err, domain.ErrImportacionOrganizacionNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("recibo alterado confirmado", err)
			}
		})
	}
}

func TestRepositorioImportacionOrganizacionReplayConservaDecisionOriginal(t *testing.T) {
	o := ordenImportacionOrganizacionPrueba(t)
	base := reciboImportacionPrueba(t, o)
	base = bytes.Replace(base, []byte(`"decision_ref":"dec_importacion"`), []byte(`"decision_ref":"dec_original"`), 1)
	base = bytes.Replace(base, []byte(`"replay":false`), []byte(`"replay":true`), 1)
	tx := &txP{fila: filaP{vals: []any{base}}}
	p := &poolP{tx: tx}
	r, _ := nuevoRepositorioImportacionOrganizacionPostgreSQL(p)
	recibo, err := r.Ejecutar(context.Background(), o)
	if err != nil || !recibo.Replay || recibo.DecisionRef != "dec_original" || tx.commits != 1 {
		t.Fatal("replay no conservó recibo", err)
	}
}

func TestRepositorioImportacionOrganizacionPropagaCancelacionYConflicto(t *testing.T) {
	o := ordenImportacionOrganizacionPrueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	tx := &txP{fila: filaP{vals: []any{reciboImportacionPrueba(t, o)}}, cancelQuery: cancelar}
	p := &poolP{tx: tx}
	r, _ := nuevoRepositorioImportacionOrganizacionPostgreSQL(p)
	_, err := r.Ejecutar(ctx, o)
	if !errors.Is(err, context.Canceled) || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("cancelación no revirtió", err)
	}
	tx = &txP{errQ: &pgconn.PgError{Code: "P0112", Message: "detalle privado"}}
	p = &poolP{tx: tx}
	r, _ = nuevoRepositorioImportacionOrganizacionPostgreSQL(p)
	_, err = r.Ejecutar(context.Background(), o)
	if !errors.Is(err, domain.ErrImportacionOrganizacionConflicto) || strings.Contains(err.Error(), "privado") || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("conflicto nominal mal traducido", err)
	}
	tx = &txP{errQ: &pgconn.PgError{Code: "42501", Message: "identidad privada"}}
	p = &poolP{tx: tx}
	r, _ = nuevoRepositorioImportacionOrganizacionPostgreSQL(p)
	_, err = r.Ejecutar(context.Background(), o)
	if !errors.Is(err, domain.ErrImportacionOrganizacionDenegada) || strings.Contains(err.Error(), "privada") || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("denegación nominal mal traducida", err)
	}
}

func TestRepositorioImportacionOrganizacionPublicarExigeAcreditacionLigada(t *testing.T) {
	base := ordenImportacionOrganizacionPrueba(t)
	s := base.Material.Solicitud()
	s.Fase = domain.FasePublicarOrganizacion
	s.LoteRef = "lote:uno"
	s.RevisionEsperada = 2
	s.Hechos = nil
	s.RevisorActorRef = s.Actor.Principal.ID
	s.Manifiesto.DocumentoRef = "documento:uno"
	s.Manifiesto.CustodiaRef = "custodia:uno"
	s.Manifiesto.DiccionarioRef = "diccionario:uno"
	s.Manifiesto.ActoRef = "acto:uno"
	s.Manifiesto.AprobadaEn = "2026-01-01"
	s.Manifiesto.PublicadaEn = "2026-01-02"
	s.Manifiesto.EfectosDesde = "2026-01-03"
	m, err := domain.NuevoMaterialImportacionOrganizacion(s)
	if err != nil {
		t.Fatal(err)
	}
	h, err := m.Recurso().HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	x := base.Autorizacion.ResumenCapacidad()
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(x.DecisionRef(), x.DecisionHuellaSHA256(), x.MotivoHuellaSHA256(), x.ContextoRef(), x.ContextoHuellaSHA256(), s.Fase.Accion(), m.Recurso().Referencia, h, domain.AudienciaImportacionOrganizacion, x.EmitidaEn(), x.ExpiraEn())
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(base.Autorizacion.CapacidadCanonica(), resumen, base.Autorizacion.DecisionCanonica(), base.Autorizacion.MotivoCanonico(), base.Autorizacion.ContextoActorCanonico(), base.Autorizacion.PersonaVersion(), base.Autorizacion.PerfilVersion(), base.Autorizacion.PayloadVECAD3(), base.Autorizacion.SobreCOSESign1(), base.Autorizacion.EvidenciaVerificacion(), base.Autorizacion.RaizPublicaSPKI())
	if err != nil {
		t.Fatal(err)
	}
	huellaManifiesto, err := s.Manifiesto.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := ports.AcreditacionFuenteOrganizacionHistorica{ManifiestoHuellaSHA256: huellaManifiesto, OrganismoRef: s.Manifiesto.OrganismoRef, Tipo: s.Manifiesto.Tipo, FuenteRef: s.Manifiesto.FuenteRef, FuenteVersion: s.Manifiesto.FuenteVersion, FuenteHuellaSHA256: s.Manifiesto.FuenteHuellaSHA256, DiccionarioRef: s.Manifiesto.DiccionarioRef, ActoRef: s.Manifiesto.ActoRef, CustodiaRef: s.Manifiesto.CustodiaRef, AcreditacionRef: "acreditacion:uno", AcreditacionHuellaSHA256: strings.Repeat("e", 64), AcreditadaEn: x.EmitidaEn()}
	o := ports.OrdenImportacionOrganizacion{Material: m, Autorizacion: a, Acreditacion: acreditacion}
	recibo := ports.ReciboImportacionOrganizacion{ReciboRef: "recibo:publicar", LoteRef: s.LoteRef, Fase: s.Fase, Estado: "publicada", RevisionAnterior: s.RevisionEsperada, RevisionNueva: s.RevisionEsperada + 1, ClaveIdempotencia: s.ClaveIdempotencia, MaterialHuellaSHA256: m.HuellaSHA256(), FuenteHuellaSHA256: s.Manifiesto.FuenteHuellaSHA256, ActorRef: s.Actor.Principal.ID, DecisionRef: a.ResumenCapacidad().DecisionRef(), AuditoriaRef: "auditoria:publicar", RegistradoEn: x.EmitidaEn().Add(time.Microsecond)}
	b, err := json.Marshal(recibo)
	if err != nil {
		t.Fatal(err)
	}
	tx := &txP{fila: filaP{vals: []any{b}}}
	p := &poolP{tx: tx}
	r, _ := nuevoRepositorioImportacionOrganizacionPostgreSQL(p)
	if _, err := r.Ejecutar(context.Background(), o); err != nil || tx.commits != 1 {
		t.Fatal("publicación acreditada no pasó contrato", err)
	}
	if _, ok := tx.a[0][1].(string); !ok {
		t.Fatal("acreditación no llegó como JSON")
	}
	o.Acreditacion.ManifiestoHuellaSHA256 = strings.Repeat("f", 64)
	p = &poolP{tx: &txP{}}
	r, _ = nuevoRepositorioImportacionOrganizacionPostgreSQL(p)
	if _, err := r.Ejecutar(context.Background(), o); !errors.Is(err, domain.ErrImportacionOrganizacionInvalida) || p.n != 0 {
		t.Fatal("acreditación ajena alcanzó SQL", err)
	}
}
