package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type relojFormalizacionPrueba struct{ instante time.Time }

func (r relojFormalizacionPrueba) Ahora() time.Time { return r.instante }

// reglasFormalizacionPrueba simula el catálogo: tres días hábiles para la
// documentación y un día hábil de margen, con vencimientos fijos.
type reglasFormalizacionPrueba struct {
	lista       ports.ReglaFormalizacion
	errRegla    error
	errPlazo    error
	vencimiento map[string]ports.VencimientoFormalizacion
	inicios     []time.Time
}

func (r *reglasFormalizacionPrueba) ReglaFormalizacion(_ context.Context, clave string) (ports.ReglaFormalizacion, error) {
	if r.errRegla != nil {
		return ports.ReglaFormalizacion{}, r.errRegla
	}
	if clave != "b22.documentos_incorporacion" {
		return ports.ReglaFormalizacion{}, ports.ErrReglasFormalizacionNoDisponibles
	}
	return r.lista, nil
}

func (r *reglasFormalizacionPrueba) VencimientoFormalizacion(_ context.Context, clave string, inicio time.Time) (ports.ReglaFormalizacion, ports.VencimientoFormalizacion, error) {
	r.inicios = append(r.inicios, inicio)
	v, ok := r.vencimiento[clave]
	if r.errPlazo != nil || !ok {
		return ports.ReglaFormalizacion{}, ports.VencimientoFormalizacion{}, errors.Join(ports.ErrReglasFormalizacionNoDisponibles, r.errPlazo)
	}
	return ports.ReglaFormalizacion{Clave: clave, Unidad: "dias_habiles", Cantidad: 3, Origen: "ejemplo", Ejemplo: true}, v, nil
}

type tiposFormalizacionPrueba map[string]bool

func (t tiposFormalizacionPrueba) TipoDocumentalCatalogado(tipo string) bool { return t[tipo] }

func escenarioFormalizacion(t *testing.T, ahora time.Time) (*ServicioDocumentacionFormalizacion, *reglasFormalizacionPrueba) {
	t.Helper()
	madrid, _ := time.LoadLocation("Europe/Madrid")
	reglas := &reglasFormalizacionPrueba{
		lista: ports.ReglaFormalizacion{Clave: "b22.documentos_incorporacion", Unidad: "lista",
			Elementos:             []string{"documento_identidad", "titulacion"},
			ElementosPorModalidad: map[string][]string{"sustitucion": {"documento_identidad"}}, Origen: "ejemplo", Ejemplo: true},
		vencimiento: map[string]ports.VencimientoFormalizacion{
			"b21.plazo_documentacion": {UltimoDia: "2026-09-30", VenceAntesDe: time.Date(2026, 10, 1, 0, 0, 0, 0, madrid)},
			"b23.plazo_incorporacion": {UltimoDia: "2026-09-28", VenceAntesDe: time.Date(2026, 9, 29, 0, 0, 0, 0, madrid)},
		},
	}
	servicio, err := NuevoServicioDocumentacionFormalizacion(ConfiguracionDocumentacionFormalizacion{
		Reglas: reglas, Tipos: tiposFormalizacionPrueba{"contratacion_temporal.formalizacion.titulacion.v1": true},
		Reloj: relojFormalizacionPrueba{ahora}, ClavePlazoDocumentacion: "b21.plazo_documentacion",
		ClaveDocumentos: "b22.documentos_incorporacion", ClavePlazoIncorporacion: "b23.plazo_incorporacion",
		PrefijoTipoDocumental: "contratacion_temporal.formalizacion.", SufijoTipoDocumental: ".v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return servicio, reglas
}

var aceptadaFormalizacionPrueba = time.Date(2026, 9, 25, 9, 5, 0, 0, time.UTC)

const expedienteFormalizacionPrueba = "expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7"

func TestDocumentacionFormalizacionCalculaDesdeLaAceptacionConElCatalogo(t *testing.T) {
	servicio, reglas := escenarioFormalizacion(t, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	r, err := servicio.Consultar(context.Background(), SolicitudDocumentacionFormalizacion{ExpedienteRef: expedienteFormalizacionPrueba, AceptadaEn: aceptadaFormalizacionPrueba})
	if err != nil {
		t.Fatal(err)
	}
	agrupacion, _ := ReferenciaExpedienteDocumentalFormalizacion(expedienteFormalizacionPrueba)
	if r.ExpedienteDocumentalRef != agrupacion || !strings.HasPrefix(agrupacion, "ref:") || len(agrupacion) != 68 ||
		strings.Contains(agrupacion, "fe4934a1") {
		t.Fatalf("agrupación documental: %q", r.ExpedienteDocumentalRef)
	}
	if len(r.Documentos) != 2 || r.Documentos[0].Clave != "documento_identidad" || r.Documentos[0].Registrable ||
		r.Documentos[1].TipoDocumental != "contratacion_temporal.formalizacion.titulacion.v1" || !r.Documentos[1].Registrable || r.PorModalidad {
		t.Fatalf("documentos: %+v", r.Documentos)
	}
	if r.PlazoDocumentacion.Vencimiento.UltimoDia != "2026-09-30" || r.PlazoDocumentacion.Estado != PlazoFormalizacionEnCurso ||
		r.PlazoIncorporacion == nil || r.PlazoIncorporacion.Vencimiento.UltimoDia != "2026-09-28" || !r.ReglaDocumentos.Ejemplo {
		t.Fatalf("plazos: %+v", r)
	}
	for _, inicio := range reglas.inicios {
		if !inicio.Equal(aceptadaFormalizacionPrueba) {
			t.Fatalf("el plazo no parte de la aceptación: %v", inicio)
		}
	}
}

func TestDocumentacionFormalizacionUsaLaListaDeLaModalidadSoloSiElCatalogoLaDistingue(t *testing.T) {
	servicio, _ := escenarioFormalizacion(t, time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC))
	r, err := servicio.Consultar(context.Background(), SolicitudDocumentacionFormalizacion{ExpedienteRef: expedienteFormalizacionPrueba, AceptadaEn: aceptadaFormalizacionPrueba, Modalidad: "sustitucion"})
	if err != nil || !r.PorModalidad || len(r.Documentos) != 1 {
		t.Fatalf("modalidad distinguida: %v %+v", err, r)
	}
	r, err = servicio.Consultar(context.Background(), SolicitudDocumentacionFormalizacion{ExpedienteRef: expedienteFormalizacionPrueba, AceptadaEn: aceptadaFormalizacionPrueba, Modalidad: "vacante"})
	if err != nil || r.PorModalidad || len(r.Documentos) != 2 {
		t.Fatalf("modalidad sin lista propia: %v %+v", err, r)
	}
}

func TestDocumentacionFormalizacionMarcaUltimoDiaYVencidoEnHoraPeninsular(t *testing.T) {
	casos := map[time.Time]EstadoPlazoFormalizacion{
		time.Date(2026, 9, 29, 21, 59, 0, 0, time.UTC): PlazoFormalizacionEnCurso,   // 23:59 del 29 en Madrid
		time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC):  PlazoFormalizacionUltimoDia, // 00:00 del 30
		time.Date(2026, 9, 30, 21, 59, 0, 0, time.UTC): PlazoFormalizacionUltimoDia,
		time.Date(2026, 9, 30, 22, 0, 0, 0, time.UTC):  PlazoFormalizacionVencido, // 00:00 del 1 de octubre
	}
	for ahora, esperado := range casos {
		servicio, _ := escenarioFormalizacion(t, ahora)
		r, err := servicio.Consultar(context.Background(), SolicitudDocumentacionFormalizacion{ExpedienteRef: expedienteFormalizacionPrueba, AceptadaEn: aceptadaFormalizacionPrueba})
		if err != nil || r.PlazoDocumentacion.Estado != esperado {
			t.Errorf("%v: %v %s", ahora, err, r.PlazoDocumentacion.Estado)
		}
	}
}

func TestDocumentacionFormalizacionNoSuponeNadaSinCatalogoOConEntradaInvalida(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	casos := map[string]struct {
		alterar  func(*reglasFormalizacionPrueba, *SolicitudDocumentacionFormalizacion)
		esperado error
	}{
		"sin catalogo": {func(r *reglasFormalizacionPrueba, _ *SolicitudDocumentacionFormalizacion) {
			r.errRegla = ports.ErrReglasFormalizacionNoConfiguradas
		}, ports.ErrReglasFormalizacionNoConfiguradas},
		"calculo caido": {func(r *reglasFormalizacionPrueba, _ *SolicitudDocumentacionFormalizacion) {
			r.errPlazo = errors.New("calendarios caidos")
		}, ports.ErrReglasFormalizacionNoDisponibles},
		"lista vacia": {func(r *reglasFormalizacionPrueba, _ *SolicitudDocumentacionFormalizacion) {
			r.lista.Elementos = nil
		}, ports.ErrReglasFormalizacionNoDisponibles},
		"clave repetida": {func(r *reglasFormalizacionPrueba, _ *SolicitudDocumentacionFormalizacion) {
			r.lista.Elementos = []string{"titulacion", "titulacion"}
		}, ports.ErrReglasFormalizacionNoDisponibles},
		"clave con ruta": {func(r *reglasFormalizacionPrueba, _ *SolicitudDocumentacionFormalizacion) {
			r.lista.Elementos = []string{"../titulacion"}
		}, ports.ErrReglasFormalizacionNoDisponibles},
		"vencimiento anterior": {func(r *reglasFormalizacionPrueba, _ *SolicitudDocumentacionFormalizacion) {
			r.vencimiento["b21.plazo_documentacion"] = ports.VencimientoFormalizacion{UltimoDia: "2026-09-24", VenceAntesDe: aceptadaFormalizacionPrueba}
		}, ports.ErrReglasFormalizacionNoDisponibles},
		"aceptacion futura": {func(_ *reglasFormalizacionPrueba, s *SolicitudDocumentacionFormalizacion) {
			s.AceptadaEn = ahora.Add(time.Minute)
		}, ports.ErrSolicitudDocumentacionFormalizacion},
		"aceptacion ausente": {func(_ *reglasFormalizacionPrueba, s *SolicitudDocumentacionFormalizacion) {
			s.AceptadaEn = time.Time{}
		}, ports.ErrSolicitudDocumentacionFormalizacion},
		"expediente no opaco": {func(_ *reglasFormalizacionPrueba, s *SolicitudDocumentacionFormalizacion) {
			s.ExpedienteRef = "expediente ct"
		}, ports.ErrSolicitudDocumentacionFormalizacion},
		"modalidad no tecnica": {func(_ *reglasFormalizacionPrueba, s *SolicitudDocumentacionFormalizacion) {
			s.Modalidad = "Sustitución"
		}, ports.ErrSolicitudDocumentacionFormalizacion},
	}
	for nombre, caso := range casos {
		servicio, reglas := escenarioFormalizacion(t, ahora)
		solicitud := SolicitudDocumentacionFormalizacion{ExpedienteRef: expedienteFormalizacionPrueba, AceptadaEn: aceptadaFormalizacionPrueba}
		caso.alterar(reglas, &solicitud)
		if _, err := servicio.Consultar(context.Background(), solicitud); !errors.Is(err, caso.esperado) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
}

func TestDocumentacionFormalizacionExigeConfiguracionCompleta(t *testing.T) {
	base := ConfiguracionDocumentacionFormalizacion{
		Reglas: &reglasFormalizacionPrueba{}, Tipos: tiposFormalizacionPrueba{}, Reloj: relojFormalizacionPrueba{},
		ClavePlazoDocumentacion: "b21", ClaveDocumentos: "b22", PrefijoTipoDocumental: "ct.", SufijoTipoDocumental: ".v1",
	}
	for nombre, alterar := range map[string]func(*ConfiguracionDocumentacionFormalizacion){
		"sin reglas":      func(c *ConfiguracionDocumentacionFormalizacion) { c.Reglas = nil },
		"sin tipos":       func(c *ConfiguracionDocumentacionFormalizacion) { c.Tipos = nil },
		"sin plazo":       func(c *ConfiguracionDocumentacionFormalizacion) { c.ClavePlazoDocumentacion = "" },
		"sin lista":       func(c *ConfiguracionDocumentacionFormalizacion) { c.ClaveDocumentos = " " },
		"prefijo erroneo": func(c *ConfiguracionDocumentacionFormalizacion) { c.PrefijoTipoDocumental = "CT/" },
	} {
		c := base
		alterar(&c)
		if _, err := NuevoServicioDocumentacionFormalizacion(c); !errors.Is(err, ErrConfiguracionDocumentacionFormalizacion) {
			t.Errorf("%s aceptada", nombre)
		}
	}
	if _, err := NuevoServicioDocumentacionFormalizacion(base); err != nil {
		t.Fatal(err)
	}
}
