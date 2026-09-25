package application

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type politicaIntentosFalsa struct {
	politica    dominiobolsa.PoliticaIntentosTelefonicos
	configurada bool
	err         error
}

func (p politicaIntentosFalsa) PoliticaIntentosTelefonicos(context.Context) (dominiobolsa.PoliticaIntentosTelefonicos, []puertosbolsa.ReglaIntentosContacto, bool, error) {
	return p.politica, []puertosbolsa.ReglaIntentosContacto{{Clave: "b02.intentos_contacto", Referencia: "vec.bolsa.reglas:1:b02.intentos_contacto"}}, p.configurada, p.err
}

type calendarioFalso struct {
	habil bool
	err   error
	usos  int
}

func (c *calendarioFalso) EsDiaHabil(context.Context, time.Time) (bool, error) {
	c.usos++
	return c.habil, c.err
}

var diezMadrid = time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)

func politicaServicioPrueba(t *testing.T, franja string) dominiobolsa.PoliticaIntentosTelefonicos {
	t.Helper()
	zona, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	return dominiobolsa.PoliticaIntentosTelefonicos{
		IntentosPorProceso: 2, Procesos: 2, SeparacionMinima: 2 * time.Hour, ControlSeparacion: dominiobolsa.ControlReglaImpedir,
		ResultadosSinContacto: []string{dominiobolsa.ResultadoContactoNoContesta, dominiobolsa.ResultadoContactoNumeroErroneo},
		Franja:                dominiobolsa.FranjaLlamadas{DesdeMinuto: 540, HastaMinuto: 840, Zona: zona, SoloDiasHabiles: true, Control: franja, Texto: "09:00-14:00"},
	}
}

func servicioIntentosPrueba(t *testing.T, p politicaIntentosFalsa, c *calendarioFalso, ahora time.Time) *ServicioContactoParticipacion {
	t.Helper()
	s := &ServicioContactoParticipacion{}
	if err := s.EstablecerControlIntentos(p, c, func() time.Time { return ahora }); err != nil {
		t.Fatal(err)
	}
	return s
}

func intentoTelefonico(resultado string, instante time.Time) dominiobolsa.ContactoParticipacion {
	return dominiobolsa.ContactoParticipacion{Canal: dominiobolsa.CanalContactoTelefono, LlamamientoRef: "llamamiento:1", Resultado: resultado, Instante: instante}
}

func TestPrepararIntentoSoloControlaTelefonoLigadoALlamamiento(t *testing.T) {
	calendario := &calendarioFalso{habil: true}
	s := servicioIntentosPrueba(t, politicaIntentosFalsa{politica: politicaServicioPrueba(t, dominiobolsa.ControlReglaAdvertir), configurada: true}, calendario, diezMadrid)
	correo := intentoTelefonico(dominiobolsa.ResultadoContactoNoEntregado, diezMadrid)
	correo.Canal = dominiobolsa.CanalContactoCorreo
	sinLlamamiento := intentoTelefonico(dominiobolsa.ResultadoContactoNoContesta, diezMadrid)
	sinLlamamiento.LlamamientoRef = ""
	for _, c := range []dominiobolsa.ContactoParticipacion{correo, sinLlamamiento} {
		if p, err := s.prepararIntento(t.Context(), c); p != nil || err != nil {
			t.Fatalf("no debía controlarse %+v: %v %v", c, p, err)
		}
	}
	p, err := s.prepararIntento(t.Context(), intentoTelefonico(dominiobolsa.ResultadoContactoNoContesta, diezMadrid))
	if err != nil || p == nil || !p.diaHabil || calendario.usos != 1 {
		t.Fatalf("intento telefónico: %+v %v usos=%d", p, err, calendario.usos)
	}
	if p, err := (&ServicioContactoParticipacion{}).prepararIntento(t.Context(), intentoTelefonico("no_contesta", diezMadrid)); p != nil || err != nil {
		t.Fatalf("sin control compuesto se conserva la conducta de siempre: %v %v", p, err)
	}
}

func TestPrepararIntentoFallaCerradoYAplicaFranjaQueImpide(t *testing.T) {
	impide := politicaServicioPrueba(t, dominiobolsa.ControlReglaImpedir)
	s := servicioIntentosPrueba(t, politicaIntentosFalsa{politica: impide, configurada: true}, &calendarioFalso{habil: false}, diezMadrid)
	if _, err := s.prepararIntento(t.Context(), intentoTelefonico("no_contesta", diezMadrid)); !errors.Is(err, dominiobolsa.ErrIntentoFueraDeFranja) {
		t.Fatalf("día no hábil con franja que impide: %v", err)
	}
	s = servicioIntentosPrueba(t, politicaIntentosFalsa{politica: impide, configurada: true}, &calendarioFalso{err: errors.New("caído")}, diezMadrid)
	if _, err := s.prepararIntento(t.Context(), intentoTelefonico("no_contesta", diezMadrid)); !errors.Is(err, puertosbolsa.ErrContactoParticipacionNoDisponible) {
		t.Fatalf("calendario caído no es día hábil: %v", err)
	}
	s = servicioIntentosPrueba(t, politicaIntentosFalsa{err: errors.New("catálogo ilegible")}, &calendarioFalso{habil: true}, diezMadrid)
	if _, err := s.prepararIntento(t.Context(), intentoTelefonico("no_contesta", diezMadrid)); !errors.Is(err, puertosbolsa.ErrContactoParticipacionNoDisponible) {
		t.Fatalf("catálogo ilegible: %v", err)
	}
}

func TestCompletarIntentoProponeBajaTrasDosProcesos(t *testing.T) {
	p := &intentoPreparado{politica: politicaServicioPrueba(t, dominiobolsa.ControlReglaAdvertir), diaHabil: true}
	ultimo := diezMadrid.Add(-24 * time.Hour)
	registro := puertosbolsa.RegistroContactoParticipacion{ResumenPrevio: &dominiobolsa.ResumenIntentosTelefonicos{SinContacto: 3, UltimoIntento: ultimo}}
	tarde := diezMadrid.Add(5 * time.Hour)
	if err := completarIntento(p, intentoTelefonico(dominiobolsa.ResultadoContactoNumeroErroneo, tarde), &registro); err != nil {
		t.Fatal(err)
	}
	if e := registro.Intentos; e == nil || !e.BajaPropuesta || e.SinContacto != 4 || !slices.Equal(e.Avisos, []string{dominiobolsa.AvisoIntentoFueraDeFranja}) {
		t.Fatalf("estado tras el cuarto intento: %+v", registro.Intentos)
	}
	if err := completarIntento(p, intentoTelefonico("no_contesta", tarde), &puertosbolsa.RegistroContactoParticipacion{}); err == nil {
		t.Fatal("sin resumen del repositorio no se inventa el estado")
	}
}

func TestEstadoIntentosDesdeElHistorico(t *testing.T) {
	ahora := diezMadrid.Add(time.Hour)
	s := servicioIntentosPrueba(t, politicaIntentosFalsa{politica: politicaServicioPrueba(t, dominiobolsa.ControlReglaAdvertir), configurada: true}, &calendarioFalso{habil: true}, ahora)
	contactos := []dominiobolsa.ContactoParticipacion{intentoTelefonico(dominiobolsa.ResultadoContactoNoContesta, diezMadrid)}
	e, err := s.EstadoIntentosTelefonicos(t.Context(), "llamamiento:1", contactos, true)
	if err != nil || !e.Configurada || !e.Completo || e.Estado.Proceso != 1 || e.Estado.Intento != 2 || len(e.Reglas) != 1 ||
		!slices.Equal(e.Estado.Avisos, []string{dominiobolsa.AvisoIntentoAntesDeSeparacion}) {
		t.Fatalf("estado: %+v %v", e, err)
	}
	sin := servicioIntentosPrueba(t, politicaIntentosFalsa{}, &calendarioFalso{}, ahora)
	if e, err := sin.EstadoIntentosTelefonicos(t.Context(), "llamamiento:1", contactos, true); err != nil || e.Configurada {
		t.Fatalf("sin catálogo: %+v %v", e, err)
	}
}
