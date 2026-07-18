package gobiernoconvocatorias

import (
	"errors"
	"time"
)

var (
	ErrReconciliacionBorradorInvalida = errors.New("gobierno convocatorias: reconciliacion de borrador invalida")
	ErrReclamacionBorradorInvalida    = errors.New("gobierno convocatorias: reclamacion de borrador invalida")
)

// SolicitudConsultaIdentidadesBorrador obliga al adaptador a consultar en una
// unica instantanea todas las generaciones; no permite elegir la primera fila.
type SolicitudConsultaIdentidadesBorrador struct {
	bloqueoSerializacionDiario
	Identidades []ProyeccionIdentidadOperacion
}

func nuevaSolicitudConsultaIdentidadesBorrador(
	conjunto ConjuntoIdentidadesOperacion,
) (SolicitudConsultaIdentidadesBorrador, error) {
	identidades, err := conjunto.proyecciones()
	if err != nil || !identidadesConsultaValidas(identidades) {
		return SolicitudConsultaIdentidadesBorrador{}, ErrRotacionIdempotenciaInvalida
	}
	return SolicitudConsultaIdentidadesBorrador{Identidades: identidades}, nil
}

func identidadesConsultaValidas(identidades []ProyeccionIdentidadOperacion) bool {
	if len(identidades) == 0 || len(identidades) > maximoIdentidadesRotacionBorrador {
		return false
	}
	for indice, identidad := range identidades {
		if !identidad.valida() ||
			identidad.Localizador.VersionEsquema != identidad.HuellaSolicitud.VersionEsquema ||
			identidad.Localizador.GeneracionClave != identidad.HuellaSolicitud.GeneracionClave {
			return false
		}
		if indice > 0 && identidades[indice-1].Localizador.GeneracionClave <=
			identidad.Localizador.GeneracionClave {
			return false
		}
		for anterior := 0; anterior < indice; anterior++ {
			if identidadesProyectadasCoinciden(identidad, identidades[anterior]) {
				return false
			}
		}
	}
	return true
}

type CoincidenciaIdentidadBorrador struct {
	bloqueoSerializacionDiario
	Identidad ProyeccionIdentidadOperacion
	Resultado ResultadoOperacionDiario
}

type ResultadoConsultaIdentidadesBorrador struct {
	bloqueoSerializacionDiario
	Coincidencias []CoincidenciaIdentidadBorrador
}

type ResultadoReservaDecisionBorrador struct {
	bloqueoSerializacionDiario
	Identidad ProyeccionIdentidadOperacion
	Resultado ResultadoOperacionDiario
}

func (r ResultadoReservaDecisionBorrador) ValidarPara(
	s SolicitudReservaDecisionBorrador,
) error {
	if s.Validar() != nil || !identidadIncluidaExactamente(r.Identidad, s.IdentidadesConsulta) ||
		!resultadoDiarioValido(r.Resultado) || r.Resultado.Estado == ResultadoDiarioAusente {
		return ErrReservaBorradorInvalida
	}
	if r.Resultado.Estado == ResultadoDiarioReservado &&
		(!identidadesProyectadasCoinciden(r.Identidad, s.Proyeccion.Identidad) ||
			r.Resultado.Revision == 0 || r.Resultado.Cercado == 0 ||
			!r.Resultado.ArrendamientoIniciaEn.Equal(s.Proyeccion.ArrendamientoIniciaEn) ||
			!r.Resultado.ArrendamientoVenceEn.Equal(s.Proyeccion.ArrendamientoVenceEn)) {
		return ErrReservaBorradorInvalida
	}
	return nil
}

func (r ResultadoConsultaIdentidadesBorrador) ValidarPara(
	s SolicitudConsultaIdentidadesBorrador,
) error {
	if !identidadesConsultaValidas(s.Identidades) || len(r.Coincidencias) > len(s.Identidades) {
		return ErrReservaBorradorInvalida
	}
	for indice, coincidencia := range r.Coincidencias {
		if !identidadIncluidaExactamente(coincidencia.Identidad, s.Identidades) ||
			!resultadoDiarioValido(coincidencia.Resultado) ||
			coincidencia.Resultado.Estado == ResultadoDiarioAusente {
			return ErrReservaBorradorInvalida
		}
		for anterior := 0; anterior < indice; anterior++ {
			if identidadesProyectadasCoinciden(
				coincidencia.Identidad, r.Coincidencias[anterior].Identidad,
			) {
				return ErrConsultaIdempotenciaAmbigua
			}
		}
	}
	if len(r.Coincidencias) > 1 {
		return ErrConsultaIdempotenciaAmbigua
	}
	return nil
}

type SolicitudReconciliacionBorrador struct {
	bloqueoSerializacionDiario
	Identidad    ProyeccionIdentidadOperacion
	Control      ResultadoOperacionDiario
	SolicitadaEn time.Time
}

func (s SolicitudReconciliacionBorrador) Validar() error {
	estadoReconciliable := s.Control.Estado == ResultadoDiarioReservado ||
		s.Control.Estado == ResultadoDiarioEnCurso ||
		s.Control.Estado == ResultadoDiarioIndeterminado ||
		s.Control.Estado == ResultadoDiarioNoAplicado
	if !s.Identidad.valida() || !resultadoDiarioValido(s.Control) || !estadoReconciliable ||
		!instanteOperacionCanonico(s.SolicitadaEn) ||
		s.SolicitadaEn.Before(s.Control.ArrendamientoIniciaEn) {
		return ErrReconciliacionBorradorInvalida
	}
	return nil
}

// ResultadoReconciliacionBorrador procede de una transaccion de lectura/CAS.
// NoAplicado sólo es aceptable con prueba durable de rollback o ausencia total
// de agregado, auditoria, outbox y consumo del sellado.
type ResultadoReconciliacionBorrador struct {
	bloqueoSerializacionDiario
	Resultado          ResultadoOperacionDiario
	ComprobadaEn       time.Time
	PruebaDesenlaceRef string
	HuellaPruebaSHA256 string
}

func (r ResultadoReconciliacionBorrador) ValidarPara(
	s SolicitudReconciliacionBorrador,
) error {
	if s.Validar() != nil || !resultadoDiarioValido(r.Resultado) ||
		!instanteOperacionCanonico(r.ComprobadaEn) || r.ComprobadaEn.Before(s.SolicitadaEn) ||
		!r.Resultado.ArrendamientoIniciaEn.Equal(s.Control.ArrendamientoIniciaEn) ||
		!r.Resultado.ArrendamientoVenceEn.Equal(s.Control.ArrendamientoVenceEn) {
		return ErrReconciliacionBorradorInvalida
	}
	entradaTerminal := s.Control.Estado == ResultadoDiarioNoAplicado
	switch {
	case entradaTerminal:
		if r.Resultado.Estado != ResultadoDiarioNoAplicado ||
			r.Resultado.Revision != s.Control.Revision || r.Resultado.Cercado != s.Control.Cercado {
			return ErrReconciliacionBorradorInvalida
		}
	case r.Resultado.Estado == ResultadoDiarioConfirmado:
		if r.Resultado.Revision <= s.Control.Revision || r.Resultado.Cercado != s.Control.Cercado {
			return ErrReconciliacionBorradorInvalida
		}
	case r.Resultado.Estado == ResultadoDiarioNoAplicado:
		if r.Resultado.Revision <= s.Control.Revision || r.Resultado.Cercado <= s.Control.Cercado {
			return ErrReconciliacionBorradorInvalida
		}
	default:
		if r.Resultado.Estado != s.Control.Estado || r.Resultado.Revision != s.Control.Revision ||
			r.Resultado.Cercado != s.Control.Cercado {
			return ErrReconciliacionBorradorInvalida
		}
	}
	switch r.Resultado.Estado {
	case ResultadoDiarioConfirmado:
		if r.Resultado.Recibo == nil || r.PruebaDesenlaceRef != "" || r.HuellaPruebaSHA256 != "" {
			return ErrReconciliacionBorradorInvalida
		}
	case ResultadoDiarioNoAplicado:
		if !referenciaProyeccionValida(r.PruebaDesenlaceRef) || !huellaHexValida(r.HuellaPruebaSHA256) {
			return ErrReconciliacionBorradorInvalida
		}
	case ResultadoDiarioReservado, ResultadoDiarioEnCurso, ResultadoDiarioIndeterminado:
		if r.PruebaDesenlaceRef != "" || r.HuellaPruebaSHA256 != "" {
			return ErrReconciliacionBorradorInvalida
		}
	default:
		return ErrReconciliacionBorradorInvalida
	}
	return nil
}

type SolicitudReclamacionDecisionBorrador struct {
	bloqueoSerializacionDiario
	IdentidadAnterior ProyeccionIdentidadOperacion
	Reconciliacion    ResultadoReconciliacionBorrador
	Nueva             SolicitudReservaDecisionBorrador
	SolicitadaEn      time.Time
}

func (s SolicitudReclamacionDecisionBorrador) Validar() error {
	anterior := s.Reconciliacion.Resultado
	if !s.IdentidadAnterior.valida() || anterior.Estado != ResultadoDiarioNoAplicado ||
		!resultadoDiarioValido(anterior) || !referenciaProyeccionValida(s.Reconciliacion.PruebaDesenlaceRef) ||
		!huellaHexValida(s.Reconciliacion.HuellaPruebaSHA256) ||
		!instanteOperacionCanonico(s.Reconciliacion.ComprobadaEn) ||
		s.Reconciliacion.ComprobadaEn.Before(anterior.ArrendamientoVenceEn) ||
		!instanteOperacionCanonico(s.SolicitadaEn) || s.SolicitadaEn.Before(s.Reconciliacion.ComprobadaEn) ||
		s.Nueva.validar(false) != nil || !s.Nueva.SolicitadaEn.Equal(s.SolicitadaEn) ||
		!identidadesProyectadasCoinciden(s.IdentidadAnterior, s.Nueva.Proyeccion.Identidad) {
		return ErrReclamacionBorradorInvalida
	}
	return nil
}

// comprobarReclamacionCreciente se aplica a la respuesta del adaptador: una
// reclamacion nunca puede conservar revision ni fence anteriores.
func comprobarReclamacionCreciente(
	anterior ResultadoOperacionDiario,
	nuevo ResultadoOperacionDiario,
	proyeccion ProyeccionReservaDecision,
) error {
	if !resultadoDiarioValido(nuevo) || nuevo.Estado != ResultadoDiarioReservado ||
		nuevo.Revision <= anterior.Revision || nuevo.Cercado <= anterior.Cercado ||
		!nuevo.ArrendamientoIniciaEn.Equal(proyeccion.ArrendamientoIniciaEn) ||
		!nuevo.ArrendamientoVenceEn.Equal(proyeccion.ArrendamientoVenceEn) {
		return ErrReclamacionBorradorInvalida
	}
	return nil
}
