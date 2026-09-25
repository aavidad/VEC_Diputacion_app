package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type catalogoSancionesPrueba struct {
	err       error
	llamadas  int
	ultimoDia time.Time
	// efectos activa «al final» y el fin automático; revocatorios, los
	// estados que revierten la sanción.
	efectos      bool
	revocatorios []string
}

func (c *catalogoSancionesPrueba) ReversionRecurso(context.Context) (ports.ReversionCatalogoSancion, error) {
	if c.err != nil {
		return ports.ReversionCatalogoSancion{}, c.err
	}
	if len(c.revocatorios) == 0 {
		return ports.ReversionCatalogoSancion{}, nil
	}
	return ports.ReversionCatalogoSancion{Estados: c.revocatorios, Motivo: "Readmisión", ReglaRef: "vec.bolsa.reglas:1:b24.recurso_revierte", Huella: strings.Repeat("d", 64)}, nil
}

func (c *catalogoSancionesPrueba) Consecuencias(context.Context) ([]ports.ConsecuenciaSancion, error) {
	if c.err != nil {
		return nil, c.err
	}
	return []ports.ConsecuenciaSancion{
		{Clave: "b24.sancion.baja", Etiqueta: "Baja", Efecto: domain.OperacionExcluir, ReglaRef: "vec.bolsa.reglas:1:b24.sancion.baja", Huella: strings.Repeat("b", 64)},
		{Clave: "b24.sancion.pasar_al_final", Etiqueta: "Final", Efecto: domain.EfectoSancionNinguno, OrdenFinal: c.efectos, ReglaRef: "vec.bolsa.reglas:1:b24.sancion.pasar_al_final", Huella: strings.Repeat("b", 64)},
		{Clave: "b24.sancion.suspension", Etiqueta: "Suspensión", Efecto: domain.OperacionPausar, ConPlazo: true, FinAutomatico: c.efectos, ReglaRef: "vec.bolsa.reglas:1:b24.sancion.suspension", Huella: strings.Repeat("b", 64)},
	}, nil
}

func (c *catalogoSancionesPrueba) EstadosRecurso(context.Context) ([]string, error) {
	if c.err != nil {
		return nil, c.err
	}
	return []string{"interpuesto", "desestimado"}, nil
}

func (c *catalogoSancionesPrueba) ResolverSancion(ctx context.Context, clave string, inicio time.Time) (ports.ResolucionCatalogoSancion, error) {
	c.llamadas++
	c.ultimoDia = inicio
	todas, err := c.Consecuencias(ctx)
	if err != nil {
		return ports.ResolucionCatalogoSancion{}, err
	}
	for _, consecuencia := range todas {
		if consecuencia.Clave == clave {
			resultado := ports.ResolucionCatalogoSancion{Consecuencia: consecuencia, Recurso: ports.PlazoSancion{UltimoDia: "2026-10-20", ReglaRef: "vec.bolsa.reglas:1:b24.consecuencias", Huella: strings.Repeat("b", 64)}}
			if consecuencia.ConPlazo {
				resultado.SuspensionHasta = "2027-03-20"
			}
			return resultado, nil
		}
	}
	return ports.ResolucionCatalogoSancion{}, domain.ErrSancionParticipacionInvalida
}

type repositorioSancionesPrueba struct {
	ultimo     ports.ComandoRegistrarSancion
	recurso    ports.ComandoRegistrarRecursoSancion
	escrituras int
	lista      []domain.SancionParticipacion
}

func (r *repositorioSancionesPrueba) RegistrarSancion(_ context.Context, c ports.ComandoRegistrarSancion) (ports.RegistroSancion, error) {
	r.escrituras++
	r.ultimo = c
	return ports.RegistroSancion{SancionRef: c.Sancion.SancionRef, ReciboRef: c.ReciboRef}, nil
}

func (r *repositorioSancionesPrueba) RegistrarRecursoSancion(_ context.Context, c ports.ComandoRegistrarRecursoSancion) (ports.RegistroRecursoSancion, error) {
	r.escrituras++
	r.recurso = c
	return ports.RegistroRecursoSancion{SancionRef: c.SancionRef, Estado: c.Evento.Estado, RegistradaEn: c.Evento.RegistradaEn}, nil
}

func (r *repositorioSancionesPrueba) ListarSanciones(_ context.Context, _, _ string, m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]domain.SancionParticipacion, error) {
	if m.ValidarEstructura() != nil {
		return nil, errors.New("material no válido")
	}
	return r.lista, nil
}

func servicioSancionesPrueba(t *testing.T, ahora time.Time, pertenece bool, catalogo ports.CatalogoSancionesParticipacion) (*ServicioSancionesParticipacion, *repositorioSituacionPrueba, *repositorioSancionesPrueba) {
	t.Helper()
	situacionRepo := &repositorioSituacionPrueba{pertenece: pertenece, vigente: ports.SituacionParticipacion{ParticipacionRef: "participacion:b2", Situacion: domain.SituacionDisponible, Desde: ahora.Add(-time.Hour)}}
	situacion, err := NuevoServicioSituacionParticipacion(contextoSituacionPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, situacionRepo, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioSancionesPrueba{}
	servicio, err := NuevoServicioSancionesParticipacion(situacion, catalogo, repo)
	if err != nil {
		t.Fatal(err)
	}
	return servicio, situacionRepo, repo
}

func solicitudSancionPrueba(t *testing.T, ahora time.Time, consecuencia string) ports.SolicitudRegistrarSancion {
	base := solicitudSituacionPrueba(t, ahora)
	base.Motivo = "No se presentó"
	return ports.SolicitudRegistrarSancion{SolicitudCambiarSituacionParticipacion: base, Datos: domain.DatosSancion{
		Consecuencia: consecuencia, Causa: "No se presentó", FechaNotificacion: "2026-09-20",
		Resolucion:  domain.DocumentoSancion{Referencia: "registro:2026/000123", SHA256: strings.Repeat("a", 64)},
		ResueltaPor: "persona:jefatura",
	}}
}

func TestSancionConEfectoAplicaLaOperacionB8(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	catalogo := &catalogoSancionesPrueba{}
	s, _, repo := servicioSancionesPrueba(t, ahora, true, catalogo)
	res, err := s.Registrar(context.Background(), solicitudSancionPrueba(t, ahora, "b24.sancion.suspension"))
	if err != nil || repo.escrituras != 1 || res.SancionRef == "" {
		t.Fatalf("registro=%+v err=%v", res, err)
	}
	op := repo.ultimo.Operacion
	if op == nil || op.Operacion != domain.OperacionPausar || op.Cambio.Destino != domain.SituacionNoDisponible || op.Cambio.Origen != domain.SituacionDisponible ||
		op.Justificante.Tipo != domain.JustificanteResolucion || op.Validador != "persona:jefatura" || op.Cambio.Motivo != "No se presentó" {
		t.Fatalf("operación B8 inesperada: %+v", op)
	}
	if !strings.HasPrefix(repo.ultimo.ReciboRef, "recibo:situacion:") || repo.ultimo.Sancion.SuspensionHasta != "2027-03-20" || repo.ultimo.Sancion.RecursoVence != "2026-10-20" {
		t.Fatalf("sanción inesperada: %+v", repo.ultimo)
	}
	if catalogo.ultimoDia.Format(time.DateOnly) != "2026-09-20" {
		t.Fatalf("inicio del cómputo inesperado: %v", catalogo.ultimoDia)
	}
}

func TestSancionSinEfectoNoCambiaSituacion(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	s, situacionRepo, repo := servicioSancionesPrueba(t, ahora, true, &catalogoSancionesPrueba{})
	if _, err := s.Registrar(context.Background(), solicitudSancionPrueba(t, ahora, "b24.sancion.pasar_al_final")); err != nil {
		t.Fatal(err)
	}
	if repo.ultimo.Operacion != nil || situacionRepo.lecturas != 0 || !strings.HasPrefix(repo.ultimo.ReciboRef, "recibo:sancion:") {
		t.Fatalf("pasar al final no debe tocar la situación: %+v", repo.ultimo)
	}
}

func TestSancionBajaExigeOtraPersona(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	s, _, repo := servicioSancionesPrueba(t, ahora, true, &catalogoSancionesPrueba{})
	q := solicitudSancionPrueba(t, ahora, "b24.sancion.baja")
	q.Datos.ResueltaPor = q.ResultadoContexto.Contexto.PersonaRef
	if _, err := s.Registrar(context.Background(), q); !errors.Is(err, domain.ErrSancionParticipacionInvalida) || repo.escrituras != 0 {
		t.Fatalf("autoresolución de baja: %v escrituras=%d", err, repo.escrituras)
	}
}

func TestSancionSinCatalogoNoRegistraPeroConsulta(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	s, _, repo := servicioSancionesPrueba(t, ahora, true, nil)
	if _, err := s.Registrar(context.Background(), solicitudSancionPrueba(t, ahora, "b24.sancion.baja")); !errors.Is(err, ports.ErrSancionesNoConfiguradas) || repo.escrituras != 0 {
		t.Fatalf("sin catálogo: %v", err)
	}
	vista, err := s.Consultar(context.Background(), solicitudSituacionPrueba(t, ahora))
	if err != nil || vista.CatalogoDisponible {
		t.Fatalf("consulta sin catálogo: %+v %v", vista, err)
	}
}

func TestSancionFueraDeAmbitoSeDeniega(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	catalogo := &catalogoSancionesPrueba{}
	s, _, repo := servicioSancionesPrueba(t, ahora, false, catalogo)
	if _, err := s.Registrar(context.Background(), solicitudSancionPrueba(t, ahora, "b24.sancion.baja")); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || repo.escrituras != 0 {
		t.Fatalf("fuera de ámbito: %v", err)
	}
	if _, err := s.Consultar(context.Background(), solicitudSituacionPrueba(t, ahora)); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("consulta fuera de ámbito: %v", err)
	}
}

func TestRecursoSancionUsaEstadosDelCatalogo(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	s, _, repo := servicioSancionesPrueba(t, ahora, true, &catalogoSancionesPrueba{})
	q := ports.SolicitudRegistrarRecursoSancion{SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora), SancionRef: "sancion:" + strings.Repeat("c", 64), Evento: domain.EventoRecursoSancion{Estado: "interpuesto", Fecha: "2026-09-24"}}
	res, err := s.RegistrarRecurso(context.Background(), q)
	if err != nil || res.Estado != "interpuesto" || repo.recurso.Evento.Actor != q.ResultadoContexto.Contexto.PersonaRef || !repo.recurso.Evento.RegistradaEn.Equal(ahora) {
		t.Fatalf("recurso=%+v err=%v", res, err)
	}
	q.Evento.Estado = "estimado"
	if _, err := s.RegistrarRecurso(context.Background(), q); !errors.Is(err, domain.ErrSancionParticipacionInvalida) {
		t.Fatalf("estado fuera de catálogo: %v", err)
	}
	q.Evento.Estado = "interpuesto"
	q.SancionRef = "sancion:otra"
	if _, err := s.RegistrarRecurso(context.Background(), q); !errors.Is(err, domain.ErrSancionParticipacionInvalida) {
		t.Fatalf("referencia mal formada: %v", err)
	}
}

func TestConsultaSancionesConCatalogo(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	s, _, repo := servicioSancionesPrueba(t, ahora, true, &catalogoSancionesPrueba{})
	repo.lista = []domain.SancionParticipacion{{SancionRef: "sancion:" + strings.Repeat("c", 64)}}
	vista, err := s.Consultar(context.Background(), solicitudSituacionPrueba(t, ahora))
	if err != nil || !vista.CatalogoDisponible || len(vista.Sanciones) != 1 || len(vista.Consecuencias) != 3 || len(vista.EstadosRecurso) != 2 {
		t.Fatalf("vista=%+v err=%v", vista, err)
	}
	s2, _, _ := servicioSancionesPrueba(t, ahora, true, &catalogoSancionesPrueba{err: errors.New("caído")})
	vista, err = s2.Consultar(context.Background(), solicitudSituacionPrueba(t, ahora))
	if err != nil || vista.CatalogoDisponible {
		t.Fatalf("catálogo caído debe mostrar el histórico sin registro: %+v %v", vista, err)
	}
}

func TestSancionConEfectosDelCatalogo(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	s, _, repo := servicioSancionesPrueba(t, ahora, true, &catalogoSancionesPrueba{efectos: true})
	if _, err := s.Registrar(context.Background(), solicitudSancionPrueba(t, ahora, "b24.sancion.suspension")); err != nil {
		t.Fatal(err)
	}
	if !repo.ultimo.FinAutomatico || repo.ultimo.OrdenFinal || repo.ultimo.Operacion == nil ||
		repo.ultimo.Operacion.Operacion != domain.OperacionPausar || repo.ultimo.Operacion.Cambio.Destino != domain.SituacionDisponibleDesde {
		t.Fatalf("suspensión con fin: %+v", repo.ultimo)
	}
	if _, err := s.Registrar(context.Background(), solicitudSancionPrueba(t, ahora, "b24.sancion.pasar_al_final")); err != nil {
		t.Fatal(err)
	}
	if !repo.ultimo.OrdenFinal || repo.ultimo.FinAutomatico || repo.ultimo.Operacion != nil {
		t.Fatalf("pasar al final: %+v", repo.ultimo)
	}
	// Sin los atributos del catálogo, la conducta anterior.
	s2, _, repo2 := servicioSancionesPrueba(t, ahora, true, &catalogoSancionesPrueba{})
	if _, err := s2.Registrar(context.Background(), solicitudSancionPrueba(t, ahora, "b24.sancion.pasar_al_final")); err != nil || repo2.ultimo.OrdenFinal {
		t.Fatalf("sin efecto de orden: %+v %v", repo2.ultimo, err)
	}
}

func TestRecursoRevocatorioReadmiteConOtraPersona(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	s, _, repo := servicioSancionesPrueba(t, ahora, true, &catalogoSancionesPrueba{revocatorios: []string{"desestimado"}})
	documento := &domain.DocumentoSancion{Referencia: "registro:2026/000300", SHA256: strings.Repeat("e", 64)}
	sancion := "sancion:" + strings.Repeat("c", 64)
	q := ports.SolicitudRegistrarRecursoSancion{SolicitudCambiarSituacionParticipacion: solicitudSituacionPrueba(t, ahora), SancionRef: sancion,
		Evento: domain.EventoRecursoSancion{Estado: "desestimado", Fecha: "2026-09-24", Documento: documento}, ResueltaPor: "persona:jefatura"}
	if _, err := s.RegistrarRecurso(context.Background(), q); err != nil {
		t.Fatal(err)
	}
	r := repo.recurso.Reversion
	if r == nil || r.ResueltaPor != "persona:jefatura" || r.Motivo != "Readmisión" || r.ReglaRef != "vec.bolsa.reglas:1:b24.recurso_revierte" ||
		r.ReciboRef != "recibo:readmision:"+huellaClaveSancion(sancion, q.ClaveIdempotencia) {
		t.Fatalf("reversión: %+v", r)
	}
	invalidas := map[string]func(*ports.SolicitudRegistrarRecursoSancion){
		"sin quien resuelve": func(x *ports.SolicitudRegistrarRecursoSancion) { x.ResueltaPor = "" },
		"autoresuelta": func(x *ports.SolicitudRegistrarRecursoSancion) {
			x.ResueltaPor = x.ResultadoContexto.Contexto.PersonaRef
		},
		"sin resolución":          func(x *ports.SolicitudRegistrarRecursoSancion) { x.Evento.Documento = nil },
		"no revocatorio resuelto": func(x *ports.SolicitudRegistrarRecursoSancion) { x.Evento.Estado = "interpuesto" },
	}
	for nombre, cambiar := range invalidas {
		repo.escrituras = 0
		x := q
		cambiar(&x)
		if _, err := s.RegistrarRecurso(context.Background(), x); !errors.Is(err, domain.ErrSancionParticipacionInvalida) || repo.escrituras != 0 {
			t.Fatalf("%s: %v escrituras=%d", nombre, err, repo.escrituras)
		}
	}
	// Un estado no revocatorio solo se anota.
	q.Evento.Estado, q.ResueltaPor = "interpuesto", ""
	if _, err := s.RegistrarRecurso(context.Background(), q); err != nil || repo.recurso.Reversion != nil {
		t.Fatalf("no revocatorio: %+v %v", repo.recurso, err)
	}
}
