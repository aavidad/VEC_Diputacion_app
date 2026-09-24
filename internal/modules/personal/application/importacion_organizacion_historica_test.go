package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorImportacionPrueba struct {
	n        int
	alterada bool
}

func (a *autorizadorImportacionPrueba) AutorizarImportacionOrganizacion(_ context.Context, m domain.MaterialImportacionOrganizacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.n++
	r := m.Recurso()
	h, _ := r.HuellaContextoAutorizacionSHA256()
	op := m.Solicitud().Fase.Accion()
	if a.alterada {
		op = "personal.organizacion_historica.otra"
	}
	n, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), op, r.Referencia, h, domain.AudienciaImportacionOrganizacion, time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC), time.Date(2026, 9, 20, 10, 0, 3, 0, time.UTC))
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	actor, _ := m.Solicitud().Actor.RepresentacionCanonicaVinculadaV2()
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), n, []byte("d"), []byte("m"), actor, 1, 1, []byte("p"), []byte("s"), []byte("e"), raiz)
}

type catalogosImportacionPrueba struct {
	n   int
	err error
}

func (c *catalogosImportacionPrueba) ValidarReferencias(_ context.Context, _, _ domain.ReferenciaCatalogoImportacion) error {
	c.n++
	return c.err
}
func (c *catalogosImportacionPrueba) ValidarHechos(_ context.Context, _, _ domain.ReferenciaCatalogoImportacion, _ []domain.HechoImportacionOrganizacion) error {
	c.n++
	return c.err
}

type politicaImportacionPrueba struct {
	n   int
	err error
	a   ports.AcreditacionFuenteOrganizacionHistorica
}

func (p *politicaImportacionPrueba) AcreditarPublicacion(_ context.Context, _ domain.ManifiestoImportacionOrganizacion) (ports.AcreditacionFuenteOrganizacionHistorica, error) {
	p.n++
	return p.a, p.err
}

type repositorioImportacionPrueba struct {
	n int
	r ports.ReciboImportacionOrganizacion
}

func (r *repositorioImportacionPrueba) Ejecutar(_ context.Context, o ports.OrdenImportacionOrganizacion) (ports.ReciboImportacionOrganizacion, error) {
	r.n++
	if r.r.ReciboRef != "" {
		return r.r, nil
	}
	s := o.Material.Solicitud()
	x := o.Autorizacion.ResumenCapacidad()
	estado := "preparacion_no_autoritativa"
	if s.Fase == domain.FaseConciliarOrganizacion {
		estado = "conciliada"
	} else if s.Fase == domain.FasePublicarOrganizacion {
		estado = "publicada"
	}
	lote := s.LoteRef
	if lote == "" {
		lote = "lote:uno"
	}
	return ports.ReciboImportacionOrganizacion{ReciboRef: "recibo:uno", LoteRef: lote, Fase: s.Fase, Estado: estado, RevisionAnterior: s.RevisionEsperada, RevisionNueva: s.RevisionEsperada + 1,
		ClaveIdempotencia: s.ClaveIdempotencia, MaterialHuellaSHA256: o.Material.HuellaSHA256(), FuenteHuellaSHA256: s.Manifiesto.FuenteHuellaSHA256,
		ActorRef: s.Actor.Principal.ID, DecisionRef: x.DecisionRef(), AuditoriaRef: "auditoria:uno", RegistradoEn: x.EmitidaEn().Add(time.Microsecond)}, nil
}

func solicitudImportacionPrueba(t *testing.T) domain.SolicitudImportacionOrganizacion {
	t.Helper()
	a := solicitudP(t).Actor
	h := strings.Repeat("a", 64)
	return domain.SolicitudImportacionOrganizacion{Fase: domain.FasePrepararOrganizacion, ClaveIdempotencia: "018f47a2-6b31-4c80-8a95-4d2e707c5a11", CorrelacionRef: "corr_una",
		Manifiesto: domain.ManifiestoImportacionOrganizacion{OrganismoRef: "organismo:dipgra", Tipo: "rpt", VersionRef: "018f47a2-6b31-4c80-8a95-4d2e707c5a11", VersionRevision: 1, FuenteRef: "fuente:pdf", FuenteVersion: "v1", FuenteHuellaSHA256: h,
			CatalogoUnidades:        domain.ReferenciaCatalogoImportacion{ID: "estructura-organizativa-dipgra", Version: 1, Revision: 1, HuellaSHA256: h},
			CatalogoClasificaciones: domain.ReferenciaCatalogoImportacion{ID: "clasificaciones-dipgra", Version: 1, Revision: 1, HuellaSHA256: h}},
		Hechos: []domain.HechoImportacionOrganizacion{{Clase: "puesto_tipo", HechoRef: "018f47a2-6b31-4c80-8a95-4d2e707c5a12", Revision: 1, FilaFuenteRef: "fila:uno", PaginaFuente: 1, OrganismoRef: "organismo:dipgra", UnidadRef: "centro:uno", VigenteDesde: "2026-01-03", CodigoFuente: "001", Denominacion: "Puesto tipo", ClasificacionRef: "categoria:uno"}}, Actor: a}
}

func TestImportacionPreparacionConservaNoAutoritativa(t *testing.T) {
	a, c, p, r := &autorizadorImportacionPrueba{}, &catalogosImportacionPrueba{}, &politicaImportacionPrueba{}, &repositorioImportacionPrueba{}
	srv, err := NuevoServicioImportacionOrganizacion(a, c, p, r)
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := srv.Preparar(context.Background(), solicitudImportacionPrueba(t))
	if err != nil || recibo.Estado != "preparacion_no_autoritativa" || a.n != 1 || c.n != 1 || p.n != 0 || r.n != 1 {
		t.Fatalf("recibo=%+v err=%v llamadas=%d/%d/%d/%d", recibo, err, a.n, c.n, p.n, r.n)
	}
}

func TestImportacionDeniegaCapacidadAjenaAntesDeEscribir(t *testing.T) {
	a, c, p, r := &autorizadorImportacionPrueba{alterada: true}, &catalogosImportacionPrueba{}, &politicaImportacionPrueba{}, &repositorioImportacionPrueba{}
	srv, _ := NuevoServicioImportacionOrganizacion(a, c, p, r)
	_, err := srv.Preparar(context.Background(), solicitudImportacionPrueba(t))
	if !errors.Is(err, domain.ErrImportacionOrganizacionDenegada) || r.n != 0 {
		t.Fatalf("err=%v escritura=%d", err, r.n)
	}
}

func TestImportacionPublicacionExigePoliticaAdmitida(t *testing.T) {
	s := solicitudImportacionPrueba(t)
	s.Fase = domain.FasePublicarOrganizacion
	s.LoteRef = "lote:uno"
	s.RevisionEsperada = 2
	s.RevisorActorRef = s.Actor.Principal.ID
	s.Hechos = nil
	s.Manifiesto.DocumentoRef = "documento:uno"
	s.Manifiesto.CustodiaRef = "custodia:uno"
	s.Manifiesto.DiccionarioRef = "diccionario:uno"
	s.Manifiesto.ActoRef = "acto:uno"
	s.Manifiesto.AprobadaEn = "2026-01-01"
	s.Manifiesto.PublicadaEn = "2026-01-02"
	s.Manifiesto.EfectosDesde = "2026-01-03"
	a, c, p, r := &autorizadorImportacionPrueba{}, &catalogosImportacionPrueba{}, &politicaImportacionPrueba{err: errors.New("fuente no admitida")}, &repositorioImportacionPrueba{}
	srv, _ := NuevoServicioImportacionOrganizacion(a, c, p, r)
	_, err := srv.Publicar(context.Background(), s)
	if !errors.Is(err, domain.ErrImportacionOrganizacionNoDisponible) || a.n != 1 || c.n != 1 || p.n != 1 || r.n != 0 {
		t.Fatalf("err=%v llamadas=%d/%d/%d/%d", err, a.n, c.n, p.n, r.n)
	}
}

func TestImportacionReciboReplayConservaDecisionOriginal(t *testing.T) {
	s := solicitudImportacionPrueba(t)
	m, err := domain.NuevoMaterialImportacionOrganizacion(s)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := (&autorizadorImportacionPrueba{}).AutorizarImportacionOrganizacion(context.Background(), m)
	x := a.ResumenCapacidad()
	r := ports.ReciboImportacionOrganizacion{ReciboRef: "recibo:original", LoteRef: "lote:uno", Fase: s.Fase, Estado: "preparacion_no_autoritativa", RevisionAnterior: 0, RevisionNueva: 1,
		ClaveIdempotencia: s.ClaveIdempotencia, MaterialHuellaSHA256: m.HuellaSHA256(), FuenteHuellaSHA256: s.Manifiesto.FuenteHuellaSHA256,
		ActorRef: s.Actor.Principal.ID, DecisionRef: "dec_original", AuditoriaRef: "auditoria:original", RegistradoEn: x.EmitidaEn().Add(-time.Hour), Replay: true}
	if !reciboImportacionValido(m, a, r) {
		t.Fatal("rechazo replay original")
	}
	r.Replay = false
	if reciboImportacionValido(m, a, r) {
		t.Fatal("acepto recibo anterior como efecto nuevo")
	}
}

func TestImportacionNoPermiteSustituirLoteEnConciliacion(t *testing.T) {
	s := solicitudImportacionPrueba(t)
	s.Fase = domain.FaseConciliarOrganizacion
	s.LoteRef = "lote:uno"
	s.RevisionEsperada = 1
	s.Decisiones = []domain.DecisionConciliacionOrganizacion{{FilaFuenteRef: "fila:uno", Clase: "puesto_tipo", DestinoRef: "tipo:uno", Resultado: "vinculada", Motivo: "concilia codigo", EvidenciaRef: "evidencia:uno"}}
	a, c, p, r := &autorizadorImportacionPrueba{}, &catalogosImportacionPrueba{}, &politicaImportacionPrueba{}, &repositorioImportacionPrueba{}
	srv, _ := NuevoServicioImportacionOrganizacion(a, c, p, r)
	_, err := srv.Conciliar(context.Background(), s)
	if !errors.Is(err, domain.ErrImportacionOrganizacionInvalida) || a.n != 0 || r.n != 0 {
		t.Fatalf("acepto filas nuevas: %v %d/%d", err, a.n, r.n)
	}
	s.Hechos = nil
	recibo, err := srv.Conciliar(context.Background(), s)
	if err != nil || recibo.Estado != "conciliada" || c.n != 1 || r.n != 1 {
		t.Fatalf("recibo=%+v err=%v", recibo, err)
	}
}

func TestImportacionPublicacionExigeRevisorDelContexto(t *testing.T) {
	s := solicitudImportacionPrueba(t)
	s.Fase = domain.FasePublicarOrganizacion
	s.LoteRef = "lote:uno"
	s.RevisionEsperada = 2
	s.Hechos = nil
	s.Manifiesto.DocumentoRef = "documento:uno"
	s.Manifiesto.CustodiaRef = "custodia:uno"
	s.Manifiesto.DiccionarioRef = "diccionario:uno"
	s.Manifiesto.ActoRef = "acto:uno"
	s.Manifiesto.AprobadaEn = "2026-01-01"
	s.Manifiesto.PublicadaEn = "2026-01-02"
	s.Manifiesto.EfectosDesde = "2026-01-03"
	a, c, p, r := &autorizadorImportacionPrueba{}, &catalogosImportacionPrueba{}, &politicaImportacionPrueba{}, &repositorioImportacionPrueba{}
	srv, _ := NuevoServicioImportacionOrganizacion(a, c, p, r)
	s.RevisorActorRef = "persona:otra"
	_, err := srv.Publicar(context.Background(), s)
	if !errors.Is(err, domain.ErrImportacionOrganizacionInvalida) || a.n != 0 || p.n != 0 || r.n != 0 {
		t.Fatalf("revisor de cliente aceptado: %v", err)
	}
	s.RevisorActorRef = s.Actor.Principal.ID
	_, err = srv.Publicar(context.Background(), s)
	if !errors.Is(err, domain.ErrImportacionOrganizacionDenegada) || p.n != 1 || r.n != 0 {
		t.Fatalf("acreditacion vacia aceptada: %v", err)
	}
}

func TestMaterialImportacionSellaFilasYManifiesto(t *testing.T) {
	s := solicitudImportacionPrueba(t)
	m, err := domain.NuevoMaterialImportacionOrganizacion(s)
	if err != nil {
		t.Fatal(err)
	}
	canonico := m.Canonico()
	huella := m.HuellaSHA256()
	s.Hechos[0].CodigoFuente = "alterado"
	if m.Solicitud().Hechos[0].CodigoFuente != "001" || !bytes.Equal(canonico, m.Canonico()) || huella != m.HuellaSHA256() {
		t.Fatal("material cambio con slice del caller")
	}
	copia := m.Solicitud()
	copia.Hechos[0].CodigoFuente = "otra"
	if m.Solicitud().Hechos[0].CodigoFuente != "001" {
		t.Fatal("material expuso slice mutable")
	}
	manifiesto := m.Solicitud().Manifiesto
	h1, err := manifiesto.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto.ActoRef = "acto:nuevo"
	h2, err := manifiesto.HuellaSHA256()
	if err != nil || h1 == h2 {
		t.Fatal("acreditacion no distingue acto")
	}
}
