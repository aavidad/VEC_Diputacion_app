package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type autorizadorConfirmacionOfertaPrueba struct {
	base  *autorizadorBorradorPrueba
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3
}

func (a *autorizadorConfirmacionOfertaPrueba) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, solicitud dominiovec.SolicitudAutorizacionLigadaV3, resultado dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	a.datos, _ = solicitud.Datos()
	decision, confirmacion, exportador, err := a.base.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
	if err != nil {
		return decision, confirmacion, nil, err
	}
	m, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return decision, confirmacion, nil, err
	}
	r := m.ResumenCapacidad()
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3(r.DecisionRef(), r.DecisionHuellaSHA256(), r.MotivoHuellaSHA256(), r.ContextoRef(), r.ContextoHuellaSHA256(), r.Operacion(), r.EfectoRef(), r.EfectoHuellaSHA256(), puertosbolsa.AudienciaConfirmarAdjudicacionOferta, r.EmitidaEn(), r.ExpiraEn())
	if err != nil {
		return decision, confirmacion, nil, err
	}
	nuevo, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(m.CapacidadCanonica(), resumen, m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		return decision, confirmacion, nil, err
	}
	return decision, confirmacion, exportadorBorradorPrueba{material: nuevo}, nil
}

type plazoOfertaPrueba struct {
	err     error
	llamado int
}

func (p *plazoOfertaPrueba) PlazoDisposicion(_ context.Context, desde time.Time) (puertosbolsa.PlazoOferta, time.Time, error) {
	p.llamado++
	if p.err != nil {
		return puertosbolsa.PlazoOferta{}, time.Time{}, p.err
	}
	return puertosbolsa.PlazoOferta{ReglaRef: "vec.bolsa.reglas:1:b10.plazo_publicacion", Unidad: "dias_habiles", Cantidad: 2, UltimoDia: "2026-09-29"}, desde.Add(72 * time.Hour), nil
}

type repositorioOfertasPrueba struct {
	publicado  *puertosbolsa.ComandoPublicarOferta
	resuelto   *puertosbolsa.ComandoResolverOferta
	confirmado *puertosbolsa.ComandoConfirmarAdjudicacionOferta
	devolver   *puertosbolsa.OfertaPublicada
	err        error
	listadas   int
}

func (r *repositorioOfertasPrueba) Publicar(_ context.Context, c puertosbolsa.ComandoPublicarOferta) (puertosbolsa.OfertaPublicada, error) {
	r.publicado = &c
	if r.err != nil {
		return puertosbolsa.OfertaPublicada{}, r.err
	}
	if r.devolver != nil {
		return *r.devolver, nil
	}
	return puertosbolsa.OfertaPublicada{OfertaRef: c.OfertaRef, BolsaRef: c.BolsaRef, Datos: c.Datos, Plazo: c.Plazo, PublicadaEn: c.PublicadaEn, VenceAntesDe: c.VenceAntesDe, Estado: dominiobolsa.EstadoOfertaAbierta}, nil
}

func (r *repositorioOfertasPrueba) Resolver(_ context.Context, c puertosbolsa.ComandoResolverOferta) (puertosbolsa.OfertaPublicada, error) {
	r.resuelto = &c
	if r.err != nil {
		return puertosbolsa.OfertaPublicada{}, r.err
	}
	return puertosbolsa.OfertaPublicada{OfertaRef: c.OfertaRef, BolsaRef: c.BolsaRef, Estado: dominiobolsa.EstadoOfertaAdjudicada}, nil
}

func (r *repositorioOfertasPrueba) ConfirmarAdjudicacion(_ context.Context, c puertosbolsa.ComandoConfirmarAdjudicacionOferta) (puertosbolsa.OfertaPublicada, error) {
	r.confirmado = &c
	if r.err != nil {
		return puertosbolsa.OfertaPublicada{}, r.err
	}
	return puertosbolsa.OfertaPublicada{OfertaRef: c.OfertaRef, BolsaRef: c.BolsaRef, Estado: dominiobolsa.EstadoOfertaAdjudicada,
		Adjudicaciones: []puertosbolsa.AdjudicacionOferta{{NumeroDePlaza: c.NumeroDePlaza, ReciboRef: "recibo:adjudicacion:prueba"}}}, nil
}

func (r *repositorioOfertasPrueba) Listar(context.Context, string, time.Time, int) ([]puertosbolsa.OfertaPublicada, error) {
	r.listadas++
	return []puertosbolsa.OfertaPublicada{}, r.err
}

func datosOfertaPrueba() dominiobolsa.DatosOferta {
	return dominiobolsa.DatosOferta{Categoria: "Auxiliar administrativo", Centro: "Residencia Sierra", FechaInicio: "2026-10-01", FechaFin: "2026-12-31", Descripcion: "Sustitución por baja"}
}

func TestResolucionOfertaSeparaRecibosPorPlaza(t *testing.T) {
	uno := huellaResolucionOferta("oferta:prueba", 1, "clave-repetida")
	dos := huellaResolucionOferta("oferta:prueba", 2, "clave-repetida")
	if uno == dos || len(uno) != 64 || uno != huellaResolucionOferta("oferta:prueba", 1, "clave-repetida") {
		t.Fatal("una misma clave cruzó plazas o perdió estabilidad")
	}
}

func TestPublicarOfertaLigaNumeroDePlazasAlMaterialAutorizado(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	p := puertosbolsa.PlazoOferta{ReglaRef: "politica-ofertas:bolsa:of:1", HuellaCatalogo: strings.Repeat("a", 64),
		Unidad: "horas_naturales", Cantidad: 48, Computo: "continuo_utc", MunicipioSede: "18087",
		UltimoDia: "2026-09-27", PoliticaVersion: 1, Calendarios: []string{"calendario:utc-continuo:v1"},
		AperturaEn: ahora.Format(formatoInstanteMaterialOferta), VenceEn: ahora.Add(48 * time.Hour).Format(formatoInstanteMaterialOferta)}
	uno := huellaMaterialPlazoOferta("bolsa:of", ahora, ahora.Add(48*time.Hour), p, 1)
	cien := huellaMaterialPlazoOferta("bolsa:of", ahora, ahora.Add(48*time.Hour), p, 100)
	if uno == cien || uno == huellaMaterialPlazoOferta("bolsa:of", ahora, ahora.Add(48*time.Hour), p) {
		t.Fatal("numero de plazas no cambió el material V3")
	}
}

func TestConfirmacionOfertaRechazaPreparacionAusenteAntesDeAutorizar(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	base := solicitudPublicarOfertaPrueba(t, ahora)
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, autorizador, &repositorioOfertasPrueba{}, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	_, err := servicio.ConfirmarAdjudicacionOferta(context.Background(), puertosbolsa.SolicitudConfirmarAdjudicacionOferta{
		Vinculo: base.Vinculo, ResultadoContexto: base.ResultadoContexto, BolsaRef: base.BolsaRef,
		OfertaRef: "oferta:" + strings.Repeat("a", 64), NumeroDePlaza: 1, ClaveIdempotencia: "clave-confirmar-1",
		Correlacion: base.Correlacion, MotivoAutorizacion: base.MotivoAutorizacion})
	if !errors.Is(err, puertosbolsa.ErrOfertaInvalida) || autorizador.llamadas != 0 {
		t.Fatalf("confirmación sin preparación alcanzó autoridad: %v llamadas=%d", err, autorizador.llamadas)
	}
}

func TestConfirmacionOfertaExigeCertificadoAntesDeAutorizar(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodSSO, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	base := solicitudPublicarOfertaPrueba(t, ahora)
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, autorizador, &repositorioOfertasPrueba{}, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	_, err = servicio.ConfirmarAdjudicacionOferta(context.Background(), puertosbolsa.SolicitudConfirmarAdjudicacionOferta{
		Vinculo: vinculo, ResultadoContexto: resultado, BolsaRef: base.BolsaRef,
		OfertaRef: "oferta:" + strings.Repeat("a", 64), NumeroDePlaza: 1,
		PreparacionRef: "recibo:preparacion-oferta:" + strings.Repeat("b", 64), ClaveIdempotencia: "clave-confirmar-1",
		Correlacion: base.Correlacion, MotivoAutorizacion: base.MotivoAutorizacion})
	if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || autorizador.llamadas != 0 {
		t.Fatalf("autenticación no certificada llegó al PDP: %v llamadas=%d", err, autorizador.llamadas)
	}
}

func TestConfirmacionOfertaAutorizaPreparacionNominal(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	base := solicitudPublicarOfertaPrueba(t, ahora)
	a := &autorizadorConfirmacionOfertaPrueba{base: &autorizadorBorradorPrueba{t: t, instante: ahora}}
	repo := &repositorioOfertasPrueba{}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, a, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	preparacion := "recibo:preparacion-oferta:" + strings.Repeat("b", 64)
	o, err := servicio.ConfirmarAdjudicacionOferta(context.Background(), puertosbolsa.SolicitudConfirmarAdjudicacionOferta{
		Vinculo: base.Vinculo, ResultadoContexto: base.ResultadoContexto, BolsaRef: base.BolsaRef,
		OfertaRef: "oferta:" + strings.Repeat("a", 64), NumeroDePlaza: 1, PreparacionRef: preparacion,
		ClaveIdempotencia: "clave-confirmar-1", Correlacion: base.Correlacion, MotivoAutorizacion: base.MotivoAutorizacion})
	if err != nil || o.Adjudicaciones[0].NumeroDePlaza != 1 || repo.confirmado == nil ||
		a.datos.Accion != puertosbolsa.AccionConfirmarAdjudicacionOferta || a.datos.Finalidad != puertosbolsa.FinalidadConfirmarAdjudicacionOferta ||
		a.datos.Recurso.Referencia != preparacion || a.datos.Recurso.Tipo != "preparacion_adjudicacion_oferta" ||
		a.datos.Recurso.Ambitos["unidad_ref"] != "unidad:rrhh" || a.datos.Recurso.Ambitos["ambito_ref"] != "ambito:bolsa" ||
		a.datos.Recurso.Atributos["numero_de_plaza"] != "1" ||
		repo.confirmado.Material.ResumenCapacidad().AudienciaConsumo() != puertosbolsa.AudienciaConfirmarAdjudicacionOferta {
		t.Fatalf("confirmación no quedó ligada al permiso nominal: %v %+v", err, a.datos)
	}
}

func solicitudPublicarOfertaPrueba(t *testing.T, ahora time.Time) puertosbolsa.SolicitudPublicarOferta {
	t.Helper()
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return puertosbolsa.SolicitudPublicarOferta{Vinculo: vinculo, ResultadoContexto: resultado, BolsaRef: "bolsa:of", Datos: datosOfertaPrueba(), ClaveIdempotencia: "oferta-clave-0001", Correlacion: correlacionBorradorPrueba(t), MotivoAutorizacion: motivoBorradorPrueba()}
}

func TestPublicarOfertaFijaPlazoDeLaReglaYConsumeLaAutorizacionDeEmision(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	repo, plazos := &repositorioOfertasPrueba{}, &plazoOfertaPrueba{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, err := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, autorizador, repo, plazos, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	oferta, err := servicio.PublicarOferta(context.Background(), solicitudPublicarOfertaPrueba(t, ahora))
	if err != nil || oferta.Estado != dominiobolsa.EstadoOfertaAbierta || repo.publicado == nil || autorizador.llamadas != 1 {
		t.Fatalf("oferta=%+v err=%v", oferta, err)
	}
	c := repo.publicado
	if !strings.HasPrefix(c.OfertaRef, "oferta:") || strings.TrimPrefix(c.OfertaRef, "oferta:") != strings.TrimPrefix(c.ReciboRef, "recibo:oferta:") ||
		!c.VenceAntesDe.Equal(ahora.Add(72*time.Hour)) || c.Plazo.ReglaRef == "" || c.Material.ValidarEstructura() != nil || c.ActorRef != "per_0123456789abcdefghijkl" {
		t.Fatalf("comando incorrecto: %+v", c)
	}
}

func TestPublicarOfertaSinReglaNoInventaPlazo(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	repo := &repositorioOfertasPrueba{}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, &plazoOfertaPrueba{err: puertosbolsa.ErrPlazoOfertaNoConfigurado}, func() time.Time { return ahora })
	if _, err := servicio.PublicarOferta(context.Background(), solicitudPublicarOfertaPrueba(t, ahora)); !errors.Is(err, puertosbolsa.ErrPlazoOfertaNoConfigurado) || repo.publicado != nil {
		t.Fatalf("err=%v publicado=%v", err, repo.publicado)
	}
}

func TestPublicarOfertaRechazaDatosInvalidosAntesDeAutorizar(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	plazos := &plazoOfertaPrueba{}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, autorizador, &repositorioOfertasPrueba{}, plazos, func() time.Time { return ahora })
	for nombre, mutar := range map[string]func(*puertosbolsa.SolicitudPublicarOferta){
		"fin anterior":  func(s *puertosbolsa.SolicitudPublicarOferta) { s.Datos.FechaFin = "2026-09-01" },
		"centro vacío":  func(s *puertosbolsa.SolicitudPublicarOferta) { s.Datos.Centro = "" },
		"clave corta":   func(s *puertosbolsa.SolicitudPublicarOferta) { s.ClaveIdempotencia = "corta" },
		"bolsa espacio": func(s *puertosbolsa.SolicitudPublicarOferta) { s.BolsaRef = " bolsa" },
	} {
		s := solicitudPublicarOfertaPrueba(t, ahora)
		mutar(&s)
		if _, err := servicio.PublicarOferta(context.Background(), s); !errors.Is(err, puertosbolsa.ErrOfertaInvalida) {
			t.Fatalf("%s: err=%v", nombre, err)
		}
	}
	if autorizador.llamadas != 0 || plazos.llamado != 0 {
		t.Fatalf("autorizó o calculó plazo con entrada inválida: %d %d", autorizador.llamadas, plazos.llamado)
	}
}

func TestPublicarOfertaDetectaReplayConOtroContenido(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	otra := datosOfertaPrueba()
	otra.Centro = "Otro centro"
	repo := &repositorioOfertasPrueba{devolver: &puertosbolsa.OfertaPublicada{OfertaRef: "oferta:x", BolsaRef: "bolsa:of", Datos: otra, Estado: "abierta", Reutilizada: true}}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	if _, err := servicio.PublicarOferta(context.Background(), solicitudPublicarOfertaPrueba(t, ahora)); !errors.Is(err, puertosbolsa.ErrOfertaConflicto) {
		t.Fatalf("err=%v", err)
	}
}

func TestPublicarOfertaConservaDenegacionDeAmbito(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	repo := &repositorioOfertasPrueba{}
	autorizador := &autorizadorBorradorPrueba{t: t, instante: ahora}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{err: dominiovec.ErrAutorizacionDenegada}, autorizador, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	if _, err := servicio.PublicarOferta(context.Background(), solicitudPublicarOfertaPrueba(t, ahora)); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || repo.publicado != nil || autorizador.llamadas != 0 {
		t.Fatalf("err=%v", err)
	}
}

func TestResolverOfertaPasaLaPropuestaYElReciboDeterminista(t *testing.T) {
	ahora := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	repo := &repositorioOfertasPrueba{}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	base := solicitudPublicarOfertaPrueba(t, ahora)
	q := puertosbolsa.SolicitudResolverOferta{Vinculo: base.Vinculo, ResultadoContexto: base.ResultadoContexto, BolsaRef: "bolsa:of", OfertaRef: "oferta:" + strings.Repeat("a", 64), ParticipacionRef: "participacion:3", ClaveIdempotencia: "resolucion-0001", Correlacion: base.Correlacion, MotivoAutorizacion: base.MotivoAutorizacion}
	if _, err := servicio.ResolverOferta(context.Background(), q); err != nil || repo.resuelto == nil {
		t.Fatalf("err=%v", err)
	}
	if repo.resuelto.ParticipacionRef != "participacion:3" || !strings.HasPrefix(repo.resuelto.ReciboRef, "recibo:resolucion-oferta:") || repo.resuelto.Material.ValidarEstructura() != nil || repo.resuelto.UnidadRef != "unidad:rrhh" || repo.resuelto.AmbitoRef != "ambito:bolsa" {
		t.Fatalf("comando=%+v", repo.resuelto)
	}
	q.OfertaRef = "llamamiento:x"
	if _, err := servicio.ResolverOferta(context.Background(), q); !errors.Is(err, puertosbolsa.ErrOfertaInvalida) {
		t.Fatalf("referencia ajena aceptada: %v", err)
	}
}

func TestConsultarOfertasExigeBolsaAdmitidaYLimite(t *testing.T) {
	ahora := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	repo := &repositorioOfertasPrueba{}
	servicio, _ := NuevoServicioOfertasPublicadas(contextoContactoPrueba{err: dominiovec.ErrAutorizacionDenegada}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, &plazoOfertaPrueba{}, func() time.Time { return ahora })
	actor := dominiovec.ContextoActor{PersonaRef: "per_0123456789abcdefghijkl"}
	if _, err := servicio.ConsultarOfertas(context.Background(), puertosbolsa.SolicitudConsultarOfertas{ContextoActor: actor, BolsaRef: "bolsa:of", Limite: 10}); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || repo.listadas != 0 {
		t.Fatalf("err=%v listadas=%d", err, repo.listadas)
	}
	if _, err := servicio.ConsultarOfertas(context.Background(), puertosbolsa.SolicitudConsultarOfertas{ContextoActor: actor, BolsaRef: "bolsa:of", Limite: 101}); !errors.Is(err, puertosbolsa.ErrOfertaInvalida) {
		t.Fatalf("límite excesivo aceptado: %v", err)
	}
}
