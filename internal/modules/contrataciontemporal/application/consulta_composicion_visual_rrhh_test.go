package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autoridadComposicionVisualPrueba struct {
	contexto ports.ContextoConsultaRRHH
	err      error
	llamadas atomic.Int32
}

func (a *autoridadComposicionVisualPrueba) ResolverContextoConsultaRRHH(
	context.Context,
) (ports.ContextoConsultaRRHH, error) {
	a.llamadas.Add(1)
	return a.contexto, a.err
}

type autorizadorComposicionVisualPrueba struct {
	err      error
	mutar    func(*ports.CapacidadComposicionVisualRRHH)
	llamadas atomic.Int32
}

func (a *autorizadorComposicionVisualPrueba) AutorizarComposicionVisualRRHH(
	_ context.Context,
	contexto ports.ContextoConsultaRRHH,
	vocabulario ports.VocabularioComposicionVisualRRHH,
	solicitud ports.SolicitudComposicionVisualRRHH,
	instante time.Time,
) (ports.CapacidadComposicionVisualRRHH, error) {
	a.llamadas.Add(1)
	if a.err != nil {
		return ports.CapacidadComposicionVisualRRHH{}, a.err
	}
	capacidad, err := ports.NuevaCapacidadComposicionVisualRRHH(
		"decision:visual:app", "correlacion:visual:app", "motivo:visual:app",
		contexto, ports.AmbitoOrganizacionRRHH, contexto.OrganizacionRef(),
		vocabulario, solicitud, instante, instante.Add(time.Minute),
	)
	if err != nil {
		return ports.CapacidadComposicionVisualRRHH{}, err
	}
	if a.mutar != nil {
		a.mutar(&capacidad)
	}
	return capacidad, nil
}

type sesionComposicionVisualPrueba struct {
	base      ports.ComposicionVisualRRHH
	err       error
	mutar     func(*ports.ComposicionVisualRRHH)
	llamadas  atomic.Int32
	secuencia atomic.Int32
}

func (s *sesionComposicionVisualPrueba) ConsultarComposicionVisualYRegistrar(
	_ context.Context,
	orden ports.OrdenConsultaComposicionVisualRRHH,
) (ports.ComposicionVisualRRHH, error) {
	s.llamadas.Add(1)
	if s.err != nil {
		return ports.ComposicionVisualRRHH{}, s.err
	}
	resultado, err := s.base.Clonar()
	if err != nil {
		return ports.ComposicionVisualRRHH{}, err
	}
	if s.mutar != nil {
		s.mutar(&resultado)
	}
	huella, err := ports.CalcularHuellaComposicionVisualRRHH(resultado)
	if err != nil {
		return resultado, nil
	}
	numero := s.secuencia.Add(1)
	resultado.Lectura, err = ports.NuevoReciboComposicionVisualRRHH(
		"lectura:visual:app:"+strings.Repeat("x", int(numero%3+1)),
		"auditoria:visual:app:"+strings.Repeat("y", int(numero%3+1)),
		orden, huella, resultado.GeneradaEn.Add(time.Second),
	)
	return resultado, err
}

type autoridadPublicacionesVisualesPrueba struct {
	esperadas     map[string]string
	vinculoValido func(ports.SolicitudAtestacionPublicacionesVisualesRRHH) bool
	err           error
	llamadas      atomic.Int32
}

func (a *autoridadPublicacionesVisualesPrueba) AtestarPublicacionesVisualesYRegistrar(
	_ context.Context,
	solicitud ports.SolicitudAtestacionPublicacionesVisualesRRHH,
) error {
	a.llamadas.Add(1)
	if a.err != nil {
		return a.err
	}
	if solicitud.Validar() != nil {
		return ports.ErrPublicacionesVisualesRRHHNoAtestadas
	}
	if a.vinculoValido == nil || !a.vinculoValido(solicitud) {
		return ports.ErrPublicacionesVisualesRRHHNoAtestadas
	}
	publicaciones := solicitud.Publicaciones()
	if len(publicaciones) != len(a.esperadas) {
		return ports.ErrPublicacionesVisualesRRHHNoAtestadas
	}
	for _, publicacion := range publicaciones {
		identidad := string(publicacion.Clase()) + "\x00" +
			publicacion.Referencia() + "\x00" +
			fmt.Sprint(publicacion.Version())
		if a.esperadas[identidad] != publicacion.Huella() {
			return ports.ErrPublicacionesVisualesRRHHNoAtestadas
		}
	}
	return nil
}

type relojComposicionVisualPrueba struct{ ahora time.Time }

func (r *relojComposicionVisualPrueba) Ahora() time.Time { return r.ahora }

func TestServicioComposicionVisualRecorreAutoridadPDPYRegistroDurable(
	t *testing.T,
) {
	t.Parallel()
	entorno := nuevoEntornoAplicacionComposicionVisual(t)
	resultado, err := entorno.servicio.Consultar(
		context.Background(), entorno.solicitud,
	)
	if err != nil {
		t.Fatalf("consultar: %v", err)
	}
	if resultado.Flujo.Referencia != entorno.solicitud.FlujoRef() ||
		resultado.Flujo.Version != entorno.solicitud.FlujoVersion() ||
		entorno.autoridad.llamadas.Load() != 1 ||
		entorno.autorizador.llamadas.Load() != 1 ||
		entorno.sesion.llamadas.Load() != 1 ||
		entorno.publicaciones.llamadas.Load() != 1 {
		t.Fatalf(
			"recorrido incompleto: autoridad=%d autorizador=%d sesión=%d publicaciones=%d resultado=%#v",
			entorno.autoridad.llamadas.Load(),
			entorno.autorizador.llamadas.Load(),
			entorno.sesion.llamadas.Load(),
			entorno.publicaciones.llamadas.Load(), resultado,
		)
	}
	resultado.Flujo.Fases[0].ClaveI18n = "ui.alterada"
	segundo, err := entorno.servicio.Consultar(
		context.Background(), entorno.solicitud,
	)
	if err != nil || segundo.Flujo.Fases[0].ClaveI18n == "ui.alterada" {
		t.Fatalf("resultado comparte memoria con fuente: %#v, %v", segundo, err)
	}
}

func TestServicioComposicionVisualFallaCerradoAntesDeLaFuente(
	t *testing.T,
) {
	t.Parallel()
	t.Run("identidad_denegada", func(t *testing.T) {
		entorno := nuevoEntornoAplicacionComposicionVisual(t)
		entorno.autoridad.err = ports.ErrContextoConsultaRRHHInvalido
		_, err := entorno.servicio.Consultar(context.Background(), entorno.solicitud)
		if !errors.Is(err, ErrResultadoComposicionVisualRRHHNoConfiable) ||
			entorno.autorizador.llamadas.Load() != 0 ||
			entorno.sesion.llamadas.Load() != 0 {
			t.Fatalf("denegación alcanzó fuente: %v, %d/%d", err,
				entorno.autorizador.llamadas.Load(), entorno.sesion.llamadas.Load())
		}
	})
	t.Run("pdp_denegado", func(t *testing.T) {
		entorno := nuevoEntornoAplicacionComposicionVisual(t)
		entorno.autorizador.err = ports.ErrComposicionVisualRRHHNoObservable
		_, err := entorno.servicio.Consultar(context.Background(), entorno.solicitud)
		if !errors.Is(err, ErrComposicionVisualRRHHNoObservable) ||
			entorno.sesion.llamadas.Load() != 0 {
			t.Fatalf("denegación PDP alcanzó fuente: %v, %d",
				err, entorno.sesion.llamadas.Load())
		}
	})
}

func TestServicioComposicionVisualRechazaFuenteAdulteradaOSinRecibo(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre string
		mutar  func(*ports.ComposicionVisualRRHH)
	}{
		{"i18n_libre", func(c *ports.ComposicionVisualRRHH) {
			c.Flujo.ClaveI18n = "Flujo general"
		}},
		{"capacidad_no_soportada", func(c *ports.ComposicionVisualRRHH) {
			c.Capacidades[0].OperacionClave = "efecto.no_publicado"
		}},
		{"catalogo_obsoleto", func(c *ports.ComposicionVisualRRHH) {
			c.Flujo.VigenteHasta = c.GeneradaEn
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoAplicacionComposicionVisual(t)
			entorno.sesion.mutar = caso.mutar
			_, err := entorno.servicio.Consultar(
				context.Background(), entorno.solicitud,
			)
			if !errors.Is(
				err, ErrResultadoComposicionVisualRRHHNoConfiable,
			) {
				t.Fatalf("fuente adulterada aceptada: %v", err)
			}
		})
	}
	t.Run("recibo_ausente", func(t *testing.T) {
		entorno := nuevoEntornoAplicacionComposicionVisual(t)
		entorno.sesion.mutar = func(c *ports.ComposicionVisualRRHH) {
			c.Lectura = ports.ReciboComposicionVisualRRHH{}
		}
		// La sesión de prueba vuelve a sellar el resultado. Se fuerza una
		// sesión mínima que omite el registro durable.
		sinRecibo := &sesionSinReciboComposicionVisualPrueba{
			base: entorno.sesion.base,
		}
		servicio, err := NuevoServicioConsultaComposicionVisualRRHH(
			entorno.autoridad, entorno.autorizador, sinRecibo,
			entorno.publicaciones, entorno.reloj, entorno.vocabulario,
		)
		if err != nil {
			t.Fatal(err)
		}
		_, err = servicio.Consultar(context.Background(), entorno.solicitud)
		if !errors.Is(err, ErrResultadoComposicionVisualRRHHNoConfiable) {
			t.Fatalf("resultado sin recibo aceptado: %v", err)
		}
	})
}

func TestServicioComposicionVisualRechazaSustitucionReselladaPorLaFuente(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre string
		mutar  func(*ports.ComposicionVisualRRHH) error
	}{
		{"operacion", func(c *ports.ComposicionVisualRRHH) error {
			c.Flujo.Tareas[0].Operaciones[0].ClaveI18n =
				"ui.contratacion_temporal.operacion.sustituida"
			huella, err := ports.CalcularHuellaDefinicionFlujoVisualRRHH(c.Flujo)
			c.Flujo.Huella = huella
			return err
		}},
		{"opcion", func(c *ports.ComposicionVisualRRHH) error {
			c.Catalogos[0].Opciones[0].ClaveI18n =
				"ui.contratacion_temporal.opcion.sustituida"
			huella, err := ports.CalcularHuellaCatalogoVisualRRHH(c.Catalogos[0])
			c.Catalogos[0].Huella = huella
			return err
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoAplicacionComposicionVisual(t)
			entorno.sesion.mutar = func(c *ports.ComposicionVisualRRHH) {
				if err := caso.mutar(c); err != nil {
					t.Errorf("recalcular huella hostil: %v", err)
				}
			}
			_, err := entorno.servicio.Consultar(
				context.Background(), entorno.solicitud,
			)
			if !errors.Is(err, ErrResultadoComposicionVisualRRHHNoConfiable) ||
				entorno.publicaciones.llamadas.Load() != 1 {
				t.Fatalf(
					"fuente sustituyó %s bajo misma versión: %v, autoridad=%d",
					caso.nombre, err, entorno.publicaciones.llamadas.Load(),
				)
			}
		})
	}
}

type sesionSinReciboComposicionVisualPrueba struct {
	base ports.ComposicionVisualRRHH
}

func (s *sesionSinReciboComposicionVisualPrueba) ConsultarComposicionVisualYRegistrar(
	context.Context,
	ports.OrdenConsultaComposicionVisualRRHH,
) (ports.ComposicionVisualRRHH, error) {
	resultado, _ := s.base.Clonar()
	resultado.Lectura = ports.ReciboComposicionVisualRRHH{}
	return resultado, nil
}

func TestServicioComposicionVisualEsSeguroEnConsultasConcurrentes(t *testing.T) {
	entorno := nuevoEntornoAplicacionComposicionVisual(t)
	const total = 32
	var grupo sync.WaitGroup
	errores := make(chan error, total)
	for i := 0; i < total; i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			resultado, err := entorno.servicio.Consultar(
				context.Background(), entorno.solicitud,
			)
			if err == nil && len(resultado.Flujo.Fases) != 1 {
				err = errors.New("resultado concurrente incompleto")
			}
			errores <- err
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("consulta concurrente: %v", err)
		}
	}
	if entorno.sesion.llamadas.Load() != total {
		t.Fatalf("sesión recibió %d llamadas", entorno.sesion.llamadas.Load())
	}
}

func TestNuevoServicioComposicionVisualRechazaNulosTipados(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoAplicacionComposicionVisual(t)
	var autoridad *autoridadComposicionVisualPrueba
	var autorizador *autorizadorComposicionVisualPrueba
	var sesion *sesionComposicionVisualPrueba
	var publicaciones *autoridadPublicacionesVisualesPrueba
	var reloj *relojComposicionVisualPrueba
	casos := []struct {
		nombre        string
		autoridad     ports.AutoridadContextoConsultaRRHH
		autorizador   ports.AutorizadorComposicionVisualRRHH
		sesion        ports.SesionComposicionVisualRRHH
		publicaciones ports.AutoridadPublicacionesVisualesRRHH
		reloj         ports.Reloj
	}{
		{"autoridad", autoridad, entorno.autorizador, entorno.sesion, entorno.publicaciones, entorno.reloj},
		{"autorizador", entorno.autoridad, autorizador, entorno.sesion, entorno.publicaciones, entorno.reloj},
		{"sesion", entorno.autoridad, entorno.autorizador, sesion, entorno.publicaciones, entorno.reloj},
		{"publicaciones", entorno.autoridad, entorno.autorizador, entorno.sesion, publicaciones, entorno.reloj},
		{"reloj", entorno.autoridad, entorno.autorizador, entorno.sesion, entorno.publicaciones, reloj},
	}
	for _, caso := range casos {
		if servicio, err := NuevoServicioConsultaComposicionVisualRRHH(
			caso.autoridad, caso.autorizador, caso.sesion,
			caso.publicaciones, caso.reloj,
			entorno.vocabulario,
		); servicio != nil ||
			!errors.Is(err, ErrServicioComposicionVisualRRHHInvalido) {
			t.Fatalf("%s nulo aceptado: %#v, %v", caso.nombre, servicio, err)
		}
	}
}

type entornoAplicacionComposicionVisual struct {
	servicio      *ServicioConsultaComposicionVisualRRHH
	autoridad     *autoridadComposicionVisualPrueba
	autorizador   *autorizadorComposicionVisualPrueba
	sesion        *sesionComposicionVisualPrueba
	publicaciones *autoridadPublicacionesVisualesPrueba
	reloj         *relojComposicionVisualPrueba
	vocabulario   ports.VocabularioComposicionVisualRRHH
	solicitud     ports.SolicitudComposicionVisualRRHH
}

func nuevoEntornoAplicacionComposicionVisual(
	t *testing.T,
) entornoAplicacionComposicionVisual {
	t.Helper()
	ahora := time.Date(2026, 7, 26, 11, 0, 0, 0, time.UTC)
	contexto, err := ports.NuevoContextoConsultaRRHH(
		"autenticacion:visual:app", "sesion:visual:app",
		"actor:visual:app", "perfil:visual:app",
		"organizacion:diputacion-granada", ahora, ahora.Add(10*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	vocabulario, err := ports.NuevoVocabularioComposicionVisualRRHH(
		"contratacion_temporal.composicion_visual.consultar",
		"rrhh.contratacion_temporal.tramitacion",
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudComposicionVisualRRHH(
		"flujo:visual:app", 4,
	)
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadComposicionVisualPrueba{contexto: contexto}
	autorizador := &autorizadorComposicionVisualPrueba{}
	base := composicionVisualAplicacionPrueba(t, ahora, solicitud)
	sesion := &sesionComposicionVisualPrueba{base: base}
	huellaComposicion, err := ports.CalcularHuellaComposicionVisualRRHH(base)
	if err != nil {
		t.Fatal(err)
	}
	publicaciones := &autoridadPublicacionesVisualesPrueba{
		esperadas: publicacionesEsperadasComposicionVisual(base),
		vinculoValido: func(
			s ports.SolicitudAtestacionPublicacionesVisualesRRHH,
		) bool {
			return s.AutenticacionRef() == contexto.AutenticacionRef() &&
				s.DecisionRef() == "decision:visual:app" &&
				s.CorrelacionRef() == "correlacion:visual:app" &&
				strings.HasPrefix(s.LecturaRef(), "lectura:visual:app:") &&
				strings.HasPrefix(s.AuditoriaRef(), "auditoria:visual:app:") &&
				s.SesionRef() == contexto.SesionRef() &&
				s.ActorRef() == contexto.ActorRef() &&
				s.PerfilRef() == contexto.PerfilRef() &&
				s.OrganizacionRef() == contexto.OrganizacionRef() &&
				s.ClaseAmbito() == ports.AmbitoOrganizacionRRHH &&
				s.AmbitoRef() == contexto.OrganizacionRef() &&
				s.Accion() == vocabulario.Accion() &&
				s.Finalidad() == vocabulario.Finalidad() &&
				s.FlujoRef() == solicitud.FlujoRef() &&
				s.FlujoVersion() == solicitud.FlujoVersion() &&
				s.HuellaComposicion() == huellaComposicion &&
				s.RegistradaEn().Equal(base.GeneradaEn.Add(time.Second)) &&
				s.CapacidadValidaHasta().Equal(ahora.Add(time.Minute))
		},
	}
	reloj := &relojComposicionVisualPrueba{ahora: ahora}
	servicio, err := NuevoServicioConsultaComposicionVisualRRHH(
		autoridad, autorizador, sesion, publicaciones, reloj, vocabulario,
	)
	if err != nil {
		t.Fatal(err)
	}
	return entornoAplicacionComposicionVisual{
		servicio: servicio, autoridad: autoridad, autorizador: autorizador,
		sesion: sesion, publicaciones: publicaciones,
		reloj: reloj, vocabulario: vocabulario,
		solicitud: solicitud,
	}
}

func composicionVisualAplicacionPrueba(
	t *testing.T,
	ahora time.Time,
	solicitud ports.SolicitudComposicionVisualRRHH,
) ports.ComposicionVisualRRHH {
	t.Helper()
	resultado := ports.ComposicionVisualRRHH{
		Esquema:    ports.EsquemaComposicionVisualRRHH,
		GeneradaEn: ahora.Add(time.Second),
		Flujo: ports.DefinicionFlujoVisualRRHH{
			Referencia: solicitud.FlujoRef(), Version: solicitud.FlujoVersion(),
			Huella:       strings.Repeat("c", 64),
			ClaveI18n:    "ui.contratacion_temporal.flujo.general",
			PublicadoEn:  ahora.Add(-2 * time.Hour),
			VigenteDesde: ahora.Add(-time.Hour),
			VigenteHasta: ahora.Add(time.Hour),
			Fases: []ports.FaseVisualRRHH{{
				Clave: "solicitud", Orden: 1,
				ClaveI18n: "ui.contratacion_temporal.fase.solicitud",
			}},
			Tareas: []ports.TareaVisualRRHH{{
				Referencia: "tarea:visual:app", FaseClave: "solicitud", Orden: 1,
				ClaveI18n: "ui.contratacion_temporal.tarea.solicitud",
				Paneles:   []string{"panel:visual:app"},
				Operaciones: []ports.OperacionVisualRRHH{{
					Clave:          "solicitud.crear",
					ClaveI18n:      "ui.contratacion_temporal.operacion.crear",
					CapacidadClave: "contratacion_temporal.solicitud.crear",
				}},
			}},
			Paneles: []ports.PanelVisualRRHH{{
				Referencia: "panel:visual:app", Orden: 1,
				Tipo:      ports.PanelVisualDatos,
				ClaveI18n: "ui.contratacion_temporal.panel.datos",
				Campos: []ports.CampoVisualRRHH{{
					Clave: "numero", Orden: 1,
					ClaveI18n:   "ui.contratacion_temporal.campo.numero",
					Control:     ports.ControlVisualSeleccion,
					CatalogoRef: "catalogo:visual:app", CatalogoVersion: 2,
				}},
			}},
		},
		Catalogos: []ports.CatalogoVisualRRHH{{
			Referencia: "catalogo:visual:app", Version: 2,
			Huella:       strings.Repeat("d", 64),
			ClaveI18n:    "ui.contratacion_temporal.catalogo.general",
			PublicadoEn:  ahora.Add(-2 * time.Hour),
			VigenteDesde: ahora.Add(-time.Hour),
			VigenteHasta: ahora.Add(time.Hour),
			Opciones: []ports.OpcionCatalogoVisualRRHH{{
				Clave: "ordinaria", ClaveI18n: "ui.contratacion_temporal.opcion.ordinaria",
			}},
		}},
		Capacidades: []ports.CapacidadVisualConcedidaRRHH{{
			OperacionClave: "solicitud.crear",
			CapacidadClave: "contratacion_temporal.solicitud.crear",
		}},
	}
	var err error
	resultado.Flujo.Huella, err =
		ports.CalcularHuellaDefinicionFlujoVisualRRHH(resultado.Flujo)
	if err != nil {
		t.Fatal(err)
	}
	for indice := range resultado.Catalogos {
		resultado.Catalogos[indice].Huella, err =
			ports.CalcularHuellaCatalogoVisualRRHH(resultado.Catalogos[indice])
		if err != nil {
			t.Fatal(err)
		}
	}
	return resultado
}

func publicacionesEsperadasComposicionVisual(
	composicion ports.ComposicionVisualRRHH,
) map[string]string {
	esperadas := map[string]string{
		string(ports.PublicacionFlujoVisualRRHH) + "\x00" +
			composicion.Flujo.Referencia + "\x00" +
			fmt.Sprint(composicion.Flujo.Version): composicion.Flujo.Huella,
	}
	for _, catalogo := range composicion.Catalogos {
		esperadas[string(ports.PublicacionCatalogoVisualRRHH)+"\x00"+
			catalogo.Referencia+"\x00"+fmt.Sprint(catalogo.Version)] =
			catalogo.Huella
	}
	return esperadas
}
