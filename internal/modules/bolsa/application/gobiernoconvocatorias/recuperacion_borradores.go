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
	Identidades  []ProyeccionIdentidadOperacion
	SolicitadaEn time.Time
}

func nuevaSolicitudConsultaIdentidadesBorrador(
	conjunto ConjuntoIdentidadesOperacion,
	solicitadaEn time.Time,
) (SolicitudConsultaIdentidadesBorrador, error) {
	identidades, err := conjunto.proyecciones()
	if err != nil || !identidadesConsultaValidas(identidades) ||
		!instanteOperacionCanonico(solicitadaEn) {
		return SolicitudConsultaIdentidadesBorrador{}, ErrRotacionIdempotenciaInvalida
	}
	return SolicitudConsultaIdentidadesBorrador{
		Identidades: identidades, SolicitadaEn: solicitadaEn,
	}, nil
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

// ResolucionIdentidadBorrador distingue las identidades efectivamente
// consultadas de la identidad primaria canonica que posee la operacion. Es una
// proyeccion del DiarioOperacionesBorrador, que forma parte de la base de
// confianza de persistencia; no se presenta como prueba criptografica. El
// adaptador debe resolver la ventana completa en una unica instantanea.
type ResolucionIdentidadBorrador struct {
	bloqueoSerializacionDiario
	IdentidadesConsultadas []ProyeccionIdentidadOperacion
	IdentidadPrimaria      ProyeccionIdentidadOperacion
}

func (r ResolucionIdentidadBorrador) validarPara(
	candidatas []ProyeccionIdentidadOperacion,
) bool {
	if !r.IdentidadPrimaria.valida() || len(r.IdentidadesConsultadas) == 0 ||
		len(r.IdentidadesConsultadas) > len(candidatas) {
		return false
	}
	posicionCandidata := -1
	for indice, consultada := range r.IdentidadesConsultadas {
		posicion := posicionIdentidadExacta(consultada, candidatas)
		if posicion < 0 || posicion <= posicionCandidata {
			return false
		}
		posicionCandidata = posicion
		for anterior := 0; anterior < indice; anterior++ {
			if identidadesProyectadasCoinciden(consultada, r.IdentidadesConsultadas[anterior]) {
				return false
			}
		}
	}
	if identidadIncluidaExactamente(r.IdentidadPrimaria, candidatas) &&
		!identidadIncluidaExactamente(r.IdentidadPrimaria, r.IdentidadesConsultadas) {
		return false
	}
	return true
}

func posicionIdentidadExacta(
	identidad ProyeccionIdentidadOperacion,
	candidatas []ProyeccionIdentidadOperacion,
) int {
	posicion := -1
	for indice, candidata := range candidatas {
		if identidadesProyectadasCoinciden(identidad, candidata) {
			if posicion >= 0 {
				return -1
			}
			posicion = indice
		}
	}
	return posicion
}

type CoincidenciaIdentidadBorrador struct {
	bloqueoSerializacionDiario
	Resolucion ResolucionIdentidadBorrador
	Resultado  ResultadoOperacionDiario
}

type ResultadoConsultaIdentidadesBorrador struct {
	bloqueoSerializacionDiario
	Coincidencias []CoincidenciaIdentidadBorrador
}

type ResultadoReservaDecisionBorrador struct {
	bloqueoSerializacionDiario
	Resolucion ResolucionIdentidadBorrador
	Resultado  ResultadoOperacionDiario
}

func (r ResultadoReservaDecisionBorrador) ValidarPara(
	s SolicitudReservaDecisionBorrador,
) error {
	if s.Validar() != nil || !r.Resolucion.validarPara(s.IdentidadesConsulta) ||
		!resultadoDiarioValido(r.Resultado) || r.Resultado.Estado == ResultadoDiarioAusente {
		return ErrReservaBorradorInvalida
	}
	// Una respuesta exacta distinta de conflicto debe acreditar la ventana
	// completa que se entregó a la reserva. Así un adaptador no puede ocultar
	// una generación solapada al proyectar el resultado del ganador.
	if r.Resultado.Estado != ResultadoDiarioConflicto &&
		!identidadesExactasEnOrden(r.Resolucion.IdentidadesConsultadas, s.IdentidadesConsulta) {
		return ErrReservaBorradorInvalida
	}
	if r.Resultado.Estado == ResultadoDiarioReservado {
		// "reservado" significa que esta solicitud creó la reserva. Una
		// reserva ya existente se proyecta como "en_curso" aunque pertenezca
		// a la misma operación lógica; nunca se adopta su decisión ni su lease.
		if !identidadesProyectadasCoinciden(
			r.Resolucion.IdentidadPrimaria, s.Proyeccion.IdentidadPrimaria,
		) ||
			r.Resultado.Revision == 0 || r.Resultado.Cercado == 0 ||
			!r.Resultado.ArrendamientoIniciaEn.Equal(s.Proyeccion.ArrendamientoIniciaEn) ||
			!r.Resultado.ArrendamientoVenceEn.Equal(s.Proyeccion.ArrendamientoVenceEn) {
			return ErrReservaBorradorInvalida
		}
	}
	return nil
}

func identidadesExactasEnOrden(a, b []ProyeccionIdentidadOperacion) bool {
	if len(a) != len(b) {
		return false
	}
	for indice := range a {
		if !identidadesProyectadasCoinciden(a[indice], b[indice]) {
			return false
		}
	}
	return true
}

func (r ResultadoConsultaIdentidadesBorrador) ValidarPara(
	s SolicitudConsultaIdentidadesBorrador,
) error {
	if !identidadesConsultaValidas(s.Identidades) || !instanteOperacionCanonico(s.SolicitadaEn) ||
		len(r.Coincidencias) > len(s.Identidades) {
		return ErrReservaBorradorInvalida
	}
	for indice, coincidencia := range r.Coincidencias {
		if !coincidencia.Resolucion.validarPara(s.Identidades) ||
			!resultadoDiarioValido(coincidencia.Resultado) ||
			coincidencia.Resultado.Estado == ResultadoDiarioAusente ||
			(coincidencia.Resultado.Recibo != nil && !identidadesProyectadasCoinciden(
				coincidencia.Resultado.Recibo.IdentidadPrimaria,
				coincidencia.Resolucion.IdentidadPrimaria,
			)) {
			return ErrReservaBorradorInvalida
		}
		for anterior := 0; anterior < indice; anterior++ {
			if identidadesProyectadasCoinciden(
				coincidencia.Resolucion.IdentidadPrimaria,
				r.Coincidencias[anterior].Resolucion.IdentidadPrimaria,
			) || resolucionesCompartenAlias(coincidencia.Resolucion, r.Coincidencias[anterior].Resolucion) {
				return ErrConsultaIdempotenciaAmbigua
			}
		}
	}
	if len(r.Coincidencias) > 1 {
		return ErrConsultaIdempotenciaAmbigua
	}
	return nil
}

func resolucionesCompartenAlias(a, b ResolucionIdentidadBorrador) bool {
	for _, izquierda := range a.IdentidadesConsultadas {
		for _, derecha := range b.IdentidadesConsultadas {
			if identidadesProyectadasCoinciden(izquierda, derecha) {
				return true
			}
		}
	}
	return false
}

type SolicitudReconciliacionBorrador struct {
	bloqueoSerializacionDiario
	IdentidadPrimaria ProyeccionIdentidadOperacion
	Control           ResultadoOperacionDiario
	SolicitadaEn      time.Time
}

func (s SolicitudReconciliacionBorrador) Validar() error {
	estadoReconciliable := s.Control.Estado == ResultadoDiarioReservado ||
		s.Control.Estado == ResultadoDiarioEnCurso ||
		s.Control.Estado == ResultadoDiarioIndeterminado ||
		s.Control.Estado == ResultadoDiarioNoAplicado
	if !s.IdentidadPrimaria.valida() || !resultadoDiarioValido(s.Control) || !estadoReconciliable ||
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
	ResolucionAnterior ResolucionIdentidadBorrador
	Reconciliacion     ResultadoReconciliacionBorrador
	Nueva              SolicitudReservaDecisionBorrador
	SolicitadaEn       time.Time
}

func (s SolicitudReclamacionDecisionBorrador) Validar() error {
	anterior := s.Reconciliacion.Resultado
	if !s.ResolucionAnterior.validarPara(s.Nueva.IdentidadesConsulta) ||
		anterior.Estado != ResultadoDiarioNoAplicado ||
		!resultadoDiarioValido(anterior) || !referenciaProyeccionValida(s.Reconciliacion.PruebaDesenlaceRef) ||
		!huellaHexValida(s.Reconciliacion.HuellaPruebaSHA256) ||
		!instanteOperacionCanonico(s.Reconciliacion.ComprobadaEn) ||
		s.Reconciliacion.ComprobadaEn.Before(anterior.ArrendamientoVenceEn) ||
		!instanteOperacionCanonico(s.SolicitadaEn) || s.SolicitadaEn.Before(s.Reconciliacion.ComprobadaEn) ||
		s.Nueva.validar(false) != nil || !s.Nueva.SolicitadaEn.Equal(s.SolicitadaEn) ||
		!identidadesProyectadasCoinciden(
			s.ResolucionAnterior.IdentidadPrimaria, s.Nueva.Proyeccion.IdentidadPrimaria,
		) {
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
