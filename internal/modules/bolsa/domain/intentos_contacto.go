package domain

import (
	"errors"
	"slices"
	"time"
)

// Control de los intentos telefónicos de un llamamiento (Reglamento de bolsas,
// art. 8.2.a): intentos por proceso, separación mínima entre intentos, franja
// de llamadas y número de procesos sin contacto tras el que se propone la
// baja. Todos los valores llegan del catálogo de reglas; aquí solo se evalúan.

// Control de una regla: impedir rechaza el intento; advertir lo registra y
// devuelve el aviso.
const (
	ControlReglaImpedir  = "impedir"
	ControlReglaAdvertir = "advertir"
)

// Avisos que acompañan a un intento o al estado de los intentos.
const (
	AvisoIntentoAntesDeSeparacion = "antes_de_separacion"
	AvisoIntentoFueraDeFranja     = "fuera_de_franja"
	AvisoIntentoDiaNoHabil        = "dia_no_habil"
)

const (
	maximoIntentosPorProceso = 20
	maximoProcesos           = 5
	maximaSeparacion         = 7 * 24 * time.Hour
	maximoResultadosSin      = 16
)

var (
	ErrPoliticaIntentosInvalida = errors.New("bolsa: politica de intentos de contacto invalida")
	ErrIntentoAntesDeSeparacion = errors.New("bolsa: intento de contacto antes de la separacion minima")
	ErrIntentoFueraDeFranja     = errors.New("bolsa: intento de contacto fuera de la franja de llamadas")
	ErrIntentosContactoAgotados = errors.New("bolsa: intentos de contacto del llamamiento agotados")
)

// FranjaLlamadas es la franja diaria en la que se llama, en minutos desde la
// medianoche de Zona. Zona nula significa que el catálogo no fija franja.
type FranjaLlamadas struct {
	DesdeMinuto, HastaMinuto int
	Zona                     *time.Location
	SoloDiasHabiles          bool
	Control                  string
	Texto                    string
}

// PoliticaIntentosTelefonicos reúne las reglas b02, b03 y b04 ya resueltas.
type PoliticaIntentosTelefonicos struct {
	IntentosPorProceso    int
	Procesos              int
	SeparacionMinima      time.Duration
	ControlSeparacion     string
	ResultadosSinContacto []string
	Franja                FranjaLlamadas
}

func controlValido(c string) bool { return c == ControlReglaImpedir || c == ControlReglaAdvertir }

func (p PoliticaIntentosTelefonicos) Validar() error {
	if p.IntentosPorProceso < 1 || p.IntentosPorProceso > maximoIntentosPorProceso || p.Procesos < 1 || p.Procesos > maximoProcesos ||
		p.SeparacionMinima < 0 || p.SeparacionMinima > maximaSeparacion || p.SeparacionMinima%time.Second != 0 || !controlValido(p.ControlSeparacion) ||
		len(p.ResultadosSinContacto) == 0 || len(p.ResultadosSinContacto) > maximoResultadosSin {
		return ErrPoliticaIntentosInvalida
	}
	for i, r := range p.ResultadosSinContacto {
		if _, ok := resultadosContacto[r]; !ok || slices.Contains(p.ResultadosSinContacto[:i], r) {
			return ErrPoliticaIntentosInvalida
		}
	}
	if p.Franja.Zona != nil && (p.Franja.DesdeMinuto < 0 || p.Franja.DesdeMinuto >= p.Franja.HastaMinuto || p.Franja.HastaMinuto > 24*60 || !controlValido(p.Franja.Control)) {
		return ErrPoliticaIntentosInvalida
	}
	return nil
}

// MaximoIntentos es el total de intentos sin contacto de todos los procesos.
func (p PoliticaIntentosTelefonicos) MaximoIntentos() int { return p.IntentosPorProceso * p.Procesos }

// SinContacto indica si un resultado telefónico cuenta como intento fallido.
func (p PoliticaIntentosTelefonicos) SinContacto(resultado string) bool {
	return slices.Contains(p.ResultadosSinContacto, resultado)
}

// ResumenIntentosTelefonicos es lo que consta de un llamamiento antes de un
// intento: fallidos, si ya hubo contacto y el último intento.
type ResumenIntentosTelefonicos struct {
	SinContacto   int
	Contactado    bool
	UltimoIntento time.Time
}

// ResumirIntentosTelefonicos resume el histórico de un llamamiento.
func ResumirIntentosTelefonicos(p PoliticaIntentosTelefonicos, llamamientoRef string, contactos []ContactoParticipacion) ResumenIntentosTelefonicos {
	var r ResumenIntentosTelefonicos
	for _, c := range contactos {
		if c.Canal != CanalContactoTelefono || llamamientoRef == "" || c.LlamamientoRef != llamamientoRef {
			continue
		}
		if p.SinContacto(c.Resultado) {
			r.SinContacto++
		} else {
			r.Contactado = true
		}
		if c.Instante.After(r.UltimoIntento) {
			r.UltimoIntento = c.Instante
		}
	}
	return r
}

// Sumar devuelve el resumen tras añadir un intento.
func (r ResumenIntentosTelefonicos) Sumar(p PoliticaIntentosTelefonicos, resultado string, instante time.Time) ResumenIntentosTelefonicos {
	if p.SinContacto(resultado) {
		r.SinContacto++
	} else {
		r.Contactado = true
	}
	if instante.After(r.UltimoIntento) {
		r.UltimoIntento = instante
	}
	return r
}

// EstadoIntentosTelefonicos es la situación del llamamiento para RRHH.
type EstadoIntentosTelefonicos struct {
	SinContacto, Maximo int
	// Proceso y Intento numeran el siguiente intento (desde 1); cero si ya
	// no caben más.
	Proceso, Intento        int
	Contactado              bool
	UltimoIntento           time.Time
	SiguientePermitidoDesde time.Time
	// BajaPropuesta: se agotaron los procesos sin contacto; RRHH decide.
	BajaPropuesta bool
	Avisos        []string
}

// EstadoIntentos calcula la situación de un llamamiento a partir del resumen.
func EstadoIntentos(p PoliticaIntentosTelefonicos, r ResumenIntentosTelefonicos) EstadoIntentosTelefonicos {
	e := EstadoIntentosTelefonicos{SinContacto: r.SinContacto, Maximo: p.MaximoIntentos(), Contactado: r.Contactado, UltimoIntento: r.UltimoIntento}
	if r.Contactado {
		return e
	}
	if r.SinContacto >= e.Maximo {
		e.BajaPropuesta = true
		return e
	}
	e.Proceso = r.SinContacto/p.IntentosPorProceso + 1
	e.Intento = r.SinContacto%p.IntentosPorProceso + 1
	if !r.UltimoIntento.IsZero() {
		e.SiguientePermitidoDesde = r.UltimoIntento.Add(p.SeparacionMinima)
	}
	return e
}

// EnFranja indica si el instante cae dentro de la franja horaria (sin mirar
// si el día es hábil). Sin franja siempre es cierto.
func (f FranjaLlamadas) EnFranja(instante time.Time) bool {
	if f.Zona == nil {
		return true
	}
	local := instante.In(f.Zona)
	minuto := local.Hour()*60 + local.Minute()
	return minuto >= f.DesdeMinuto && minuto < f.HastaMinuto
}

// EvaluarIntentoTelefonico decide si un intento nuevo puede registrarse.
// diaHabil solo se consulta si la franja lo exige. Devuelve los avisos de las
// reglas en modo «advertir» o el error de la primera regla que lo impide.
func EvaluarIntentoTelefonico(p PoliticaIntentosTelefonicos, r ResumenIntentosTelefonicos, instante time.Time, diaHabil bool) ([]string, error) {
	if p.Validar() != nil || instante.IsZero() || r.SinContacto < 0 {
		return nil, ErrPoliticaIntentosInvalida
	}
	if r.Contactado {
		return nil, nil
	}
	if r.SinContacto >= p.MaximoIntentos() {
		return nil, ErrIntentosContactoAgotados
	}
	var avisos []string
	if !r.UltimoIntento.IsZero() {
		distancia := instante.Sub(r.UltimoIntento)
		if distancia < 0 {
			distancia = -distancia
		}
		if distancia < p.SeparacionMinima {
			if p.ControlSeparacion == ControlReglaImpedir {
				return nil, ErrIntentoAntesDeSeparacion
			}
			avisos = append(avisos, AvisoIntentoAntesDeSeparacion)
		}
	}
	for _, aviso := range p.Franja.AvisosFranja(instante, diaHabil) {
		if p.Franja.Control == ControlReglaImpedir {
			return nil, ErrIntentoFueraDeFranja
		}
		avisos = append(avisos, aviso)
	}
	return avisos, nil
}

// AvisosFranja devuelve los avisos de franja y día hábil para un instante.
func (f FranjaLlamadas) AvisosFranja(instante time.Time, diaHabil bool) []string {
	if f.Zona == nil {
		return nil
	}
	var avisos []string
	if !f.EnFranja(instante) {
		avisos = append(avisos, AvisoIntentoFueraDeFranja)
	}
	if f.SoloDiasHabiles && !diaHabil {
		avisos = append(avisos, AvisoIntentoDiaNoHabil)
	}
	return avisos
}
