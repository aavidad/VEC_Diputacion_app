package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type colaAvisoPrueba struct {
	estado          string
	intento, envios int
	falloFinal      bool
}

func (c *colaAvisoPrueba) ReclamarAvisoOferta(context.Context, ports.OfertaPublicada) (ports.IntentoAvisoOferta, bool, error) {
	if c.estado != "pendiente" {
		return ports.IntentoAvisoOferta{}, false, nil
	}
	c.estado = "enviado_sin_confirmacion"
	c.intento++
	return ports.IntentoAvisoOferta{ParticipacionRef: "participacion:uno", MessageID: "<oferta-uno@aviso.invalid>", Intento: c.intento}, true, nil
}
func (c *colaAvisoPrueba) FinalizarAvisoOferta(_ context.Context, o ports.OfertaPublicada, _ ports.IntentoAvisoOferta, e string) (ports.OfertaPublicada, error) {
	if c.falloFinal {
		return ports.OfertaPublicada{}, errors.New("fallo")
	}
	switch e {
	case ports.AvisoOfertaAceptado:
		c.estado = "enviado"
	case ports.AvisoOfertaTransitorio:
		c.estado = "pendiente"
	case ports.AvisoOfertaPermanente:
		c.estado = "no_enviado"
	default:
		c.estado = "enviado_sin_confirmacion"
	}
	a := ports.AvisosOferta{Total: 1}
	switch c.estado {
	case "enviado":
		a.Enviados = 1
	case "pendiente":
		a.Pendientes = 1
	case "no_enviado":
		a.Fallidos = 1
	default:
		a.SinConfirmacion = 1
	}
	o.Aviso = &a
	return o, nil
}
func (c *colaAvisoPrueba) CorreoParticipacion(context.Context, string) (string, error) {
	return "destino@aviso.invalid", nil
}

type envioAvisoPrueba struct {
	cola     *colaAvisoPrueba
	estado   string
	mensajes []ports.MensajeAvisoOferta
}

func (e *envioAvisoPrueba) EnviarAvisoOferta(_ context.Context, m ports.MensajeAvisoOferta) string {
	if e.cola.estado != "enviado_sin_confirmacion" {
		panic("envío antes de confirmar intención")
	}
	e.cola.envios++
	e.mensajes = append(e.mensajes, m)
	return e.estado
}
func ofertaAvisoPrueba() ports.OfertaPublicada {
	return ports.OfertaPublicada{OfertaRef: "oferta:uno", PublicadaEn: time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC), VenceAntesDe: time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC), Aviso: &ports.AvisosOferta{Total: 1, Pendientes: 1}}
}

func TestAvisoOfertaResultadoDurableYRecuperacion(t *testing.T) {
	for _, caso := range []struct {
		resultado, estado string
		reintenta         bool
	}{
		{ports.AvisoOfertaAceptado, "enviado", false}, {ports.AvisoOfertaTransitorio, "pendiente", true},
		{ports.AvisoOfertaPermanente, "no_enviado", false}, {ports.AvisoOfertaIndeterminado, "enviado_sin_confirmacion", false},
	} {
		t.Run(caso.resultado, func(t *testing.T) {
			c := &colaAvisoPrueba{estado: "pendiente"}
			e := &envioAvisoPrueba{cola: c, estado: caso.resultado}
			d, err := NuevoDespachadorAvisosOferta(c, c, e, "Oferta", "{descripcion}\n{enlace}", "https://aviso.invalid/area-personal/")
			if err != nil {
				t.Fatal(err)
			}
			s := &ServicioOfertasPublicadas{avisos: d}
			original := ofertaAvisoPrueba()
			actual := s.despacharAvisos(t.Context(), original)
			if c.estado != caso.estado || c.envios != 1 || !actual.PublicadaEn.Equal(original.PublicadaEn) || !actual.VenceAntesDe.Equal(original.VenceAntesDe) {
				t.Fatalf("resultado no conservado: %+v", c)
			}
			if original.Aviso.Pendientes != 1 || original.Aviso.SinConfirmacion != 0 {
				t.Fatal("mutó la proyección original")
			}
			// Nuevo caso de uso sobre la misma cola: reproduce un reinicio del proceso.
			e.estado = ports.AvisoOfertaAceptado
			s = &ServicioOfertasPublicadas{avisos: d}
			s.despacharAvisos(t.Context(), actual)
			esperados := 1
			if caso.reintenta {
				esperados = 2
			}
			if c.envios != esperados {
				t.Fatalf("envíos=%d, esperados=%d", c.envios, esperados)
			}
			if len(e.mensajes) == 2 && (e.mensajes[0].MessageID != e.mensajes[1].MessageID || !e.mensajes[0].Fecha.Equal(e.mensajes[1].Fecha)) {
				t.Fatal("reintento cambió identificador o fecha")
			}
		})
	}
}
func TestAvisoOfertaFalloTrasRedConservaIncertidumbre(t *testing.T) {
	c := &colaAvisoPrueba{estado: "pendiente", falloFinal: true}
	e := &envioAvisoPrueba{cola: c, estado: ports.AvisoOfertaAceptado}
	d, _ := NuevoDespachadorAvisosOferta(c, c, e, "Oferta", "{enlace}", "https://aviso.invalid/area-personal/")
	s := &ServicioOfertasPublicadas{avisos: d}
	actual := s.despacharAvisos(t.Context(), ofertaAvisoPrueba())
	if actual.Aviso.SinConfirmacion != 1 || actual.Aviso.Pendientes != 0 {
		t.Fatal("presentó como pendiente un resultado incierto")
	}
	s.despacharAvisos(t.Context(), actual)
	if c.envios != 1 {
		t.Fatal("repitió automáticamente un correo incierto")
	}
}
func TestAvisoOfertaSinConfiguracionNoReclama(t *testing.T) {
	s := &ServicioOfertasPublicadas{}
	o := s.despacharAvisos(t.Context(), ofertaAvisoPrueba())
	if !o.AvisoConfiguracionPendiente || o.Aviso.Pendientes != 1 {
		t.Fatal("configuración ausente perdió intención")
	}
}

func TestAvisoOfertaNoCopiaDescripcionLibre(t *testing.T) {
	c := &colaAvisoPrueba{estado: "pendiente"}
	e := &envioAvisoPrueba{cola: c, estado: ports.AvisoOfertaAceptado}
	d, _ := NuevoDespachadorAvisosOferta(c, c, e, "{categoria}", "{centro} {fecha_inicio} {enlace}", "https://aviso.invalid/area-personal/")
	s := &ServicioOfertasPublicadas{avisos: d}
	o := ofertaAvisoPrueba()
	o.Datos.Descripcion = "Referencia privada y nombre de la persona sustituida"
	s.despacharAvisos(t.Context(), o)
	if len(e.mensajes) != 1 || strings.Contains(e.mensajes[0].Cuerpo, o.Datos.Descripcion) {
		t.Fatal("copió texto libre al correo")
	}
}
