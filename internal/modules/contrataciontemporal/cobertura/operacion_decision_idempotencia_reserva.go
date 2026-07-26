package cobertura

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type EstadoPreparacionOperacionDecisionCobertura string

const (
	PreparacionOperacionDecisionCoberturaPropietaria EstadoPreparacionOperacionDecisionCobertura = "propietaria"
	PreparacionOperacionDecisionCoberturaOcupada     EstadoPreparacionOperacionDecisionCobertura = "ocupada"
	PreparacionOperacionDecisionCoberturaConfirmada  EstadoPreparacionOperacionDecisionCobertura = "confirmada"
)

func (e EstadoPreparacionOperacionDecisionCobertura) valida() bool {
	return e == PreparacionOperacionDecisionCoberturaPropietaria ||
		e == PreparacionOperacionDecisionCoberturaOcupada ||
		e == PreparacionOperacionDecisionCoberturaConfirmada
}

// DatosReservaPropietariaOperacionDecisionCobertura contiene el material que
// debe nacer junto en la transacción de reserva. PropiedadHasta procede del
// reloj de base de datos y solo sirve para reapropiar; nunca autoriza efectos.
type DatosReservaPropietariaOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	ReservaRef              string
	ReciboRef               string
	ActuacionRef            string
	AuditoriaRef            string
	EventoRef               string
	CorrelacionVECRef       string
	DecisionVECRef          string
	AnalisisRef             string
	AnalisisHuellaSHA256    string
	TokenPropietarioSHA256  string
	AmbitoIdempotenciaHMAC  string
	HuellaSemanticaHMAC     string
	AgregadoAnterior        *domain.Expediente
	RevisionCercadoAnterior uint64
	RevisionCercado         uint64
	ObservadaEnDB           time.Time
	PropiedadHasta          time.Time
}

func (d DatosReservaPropietariaOperacionDecisionCobertura) validarPara(
	solicitud SolicitudReservarOperacionDecisionCobertura,
) error {
	datosSolicitud, err := solicitud.Datos()
	solicitudAnalisis, errAnalisis := NuevaSolicitudInstantaneaAnalisisDurableO3(
		datosSolicitud.OrganizacionRef,
		datosSolicitud.ExpedienteRef,
		datosSolicitud.VersionExpediente,
	)
	analisisRef, analisisHuella, errIdentidadAnalisis :=
		identidadAnalisisDurableO3(
			expedienteReservaOperacionDecisionCobertura(d),
			solicitudAnalisis,
		)
	if err != nil || errAnalisis != nil || errIdentidadAnalisis != nil ||
		!domain.ReferenciaOpacaValida(d.ReservaRef) ||
		!domain.ReferenciaOpacaValida(d.ReciboRef) ||
		!domain.ReferenciaOpacaValida(d.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(d.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(d.EventoRef) ||
		!domain.ReferenciaOpacaValida(d.CorrelacionVECRef) ||
		!domain.ReferenciaOpacaValida(d.DecisionVECRef) ||
		!referenciasOperacionDecisionCoberturaIguales(
			d.AnalisisRef,
			analisisRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			d.AnalisisHuellaSHA256,
			analisisHuella,
		) ||
		!solicitud.tokenCoincide(d.TokenPropietarioSHA256) ||
		!solicitud.consulta.contienePar(
			d.AmbitoIdempotenciaHMAC,
			d.HuellaSemanticaHMAC,
		) ||
		d.AgregadoAnterior == nil ||
		d.AgregadoAnterior.Validar() != nil ||
		d.AgregadoAnterior.OrganizacionRef !=
			datosSolicitud.OrganizacionRef ||
		d.AgregadoAnterior.Referencia !=
			datosSolicitud.ExpedienteRef ||
		d.AgregadoAnterior.Version !=
			datosSolicitud.VersionExpediente ||
		d.RevisionCercadoAnterior >= MaximoEnteroSeguroOperacionDecisionCobertura ||
		d.RevisionCercado != d.RevisionCercadoAnterior+1 ||
		d.RevisionCercado > MaximoEnteroSeguroOperacionDecisionCobertura ||
		!instanteOperacionDecisionCoberturaValido(d.ObservadaEnDB) ||
		!instanteOperacionDecisionCoberturaValido(d.PropiedadHasta) ||
		!d.PropiedadHasta.After(d.ObservadaEnDB) ||
		d.PropiedadHasta.Sub(d.ObservadaEnDB) >
			MaximoLeaseOperacionDecisionCobertura {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return nil
}

func expedienteReservaOperacionDecisionCobertura(
	datos DatosReservaPropietariaOperacionDecisionCobertura,
) domain.Expediente {
	if datos.AgregadoAnterior == nil {
		return domain.Expediente{}
	}
	return *datos.AgregadoAnterior
}

func clonarReservaPropietariaOperacionDecisionCobertura(
	datos DatosReservaPropietariaOperacionDecisionCobertura,
) DatosReservaPropietariaOperacionDecisionCobertura {
	if datos.AgregadoAnterior != nil {
		clon := datos.AgregadoAnterior.Clonar()
		datos.AgregadoAnterior = &clon
	}
	return datos
}

// DatosReservaTerminalOperacionDecisionCobertura es la proyección mínima que
// el adaptador durable rehidrata tras un reinicio. No contiene clave de
// idempotencia, token propietario, actor, perfil, motivo, predecesora, lease
// ni agregado. Las coordenadas y sellos impiden asociarla a otra consulta.
type DatosReservaTerminalOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	OrganizacionRef        string
	ExpedienteRef          string
	VersionExpediente      uint64
	ReservaRef             string
	ReciboRef              string
	ActuacionRef           string
	AuditoriaRef           string
	EventoRef              string
	CorrelacionVECRef      string
	DecisionVECRef         string
	AmbitoIdempotenciaHMAC string
	HuellaSemanticaHMAC    string
	RevisionCercado        uint64
	ObservadaEnDB          time.Time
}

// ReservaTerminalOperacionDecisionCobertura es la capacidad nominal que
// acredita una fila terminal durable ya ligada a la consulta. El replay no
// depende del token efímero que poseyó la reserva antes del reinicio.
type ReservaTerminalOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *DatosReservaTerminalOperacionDecisionCobertura
}

func RehidratarReservaTerminalOperacionDecisionCobertura(
	solicitud SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	datos DatosReservaTerminalOperacionDecisionCobertura,
) (ReservaTerminalOperacionDecisionCobertura, error) {
	reserva := ReservaTerminalOperacionDecisionCobertura{datos: &datos}
	if reserva.validarPara(solicitud) != nil {
		return ReservaTerminalOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return reserva, nil
}

func (r ReservaTerminalOperacionDecisionCobertura) validarPara(
	solicitud SolicitudConsultarOperacionDecisionCoberturaConfirmada,
) error {
	organizacionRef, expedienteRef, versionExpediente, err :=
		solicitud.coordenadas()
	if err != nil || r.datos == nil ||
		!domain.ReferenciaOpacaValida(r.datos.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(r.datos.ExpedienteRef) ||
		r.datos.OrganizacionRef != organizacionRef ||
		r.datos.ExpedienteRef != expedienteRef ||
		r.datos.VersionExpediente != versionExpediente ||
		!domain.ReferenciaOpacaValida(r.datos.ReservaRef) ||
		!domain.ReferenciaOpacaValida(r.datos.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.datos.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(r.datos.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.datos.EventoRef) ||
		!domain.ReferenciaOpacaValida(r.datos.CorrelacionVECRef) ||
		!domain.ReferenciaOpacaValida(r.datos.DecisionVECRef) ||
		!solicitud.contienePar(
			r.datos.AmbitoIdempotenciaHMAC,
			r.datos.HuellaSemanticaHMAC,
		) ||
		r.datos.RevisionCercado == 0 ||
		r.datos.RevisionCercado >
			MaximoEnteroSeguroOperacionDecisionCobertura ||
		!instanteOperacionDecisionCoberturaValido(r.datos.ObservadaEnDB) {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return nil
}

func clonarReservaTerminalOperacionDecisionCobertura(
	reserva ReservaTerminalOperacionDecisionCobertura,
) ReservaTerminalOperacionDecisionCobertura {
	if reserva.datos == nil {
		return ReservaTerminalOperacionDecisionCobertura{}
	}
	datos := *reserva.datos
	return ReservaTerminalOperacionDecisionCobertura{datos: &datos}
}

func reservaTerminalDesdePropietariaOperacionDecisionCobertura(
	solicitud SolicitudReservarOperacionDecisionCobertura,
	reserva DatosReservaPropietariaOperacionDecisionCobertura,
) (ReservaTerminalOperacionDecisionCobertura, error) {
	consulta, err := solicitud.consultaConfirmada()
	if err != nil || reserva.validarPara(solicitud) != nil {
		return ReservaTerminalOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	datosSolicitud, err := solicitud.Datos()
	if err != nil {
		return ReservaTerminalOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return RehidratarReservaTerminalOperacionDecisionCobertura(
		consulta,
		DatosReservaTerminalOperacionDecisionCobertura{
			OrganizacionRef: datosSolicitud.OrganizacionRef,
			ExpedienteRef:   datosSolicitud.ExpedienteRef,
			VersionExpediente: datosSolicitud.
				VersionExpediente,
			ReservaRef:             reserva.ReservaRef,
			ReciboRef:              reserva.ReciboRef,
			ActuacionRef:           reserva.ActuacionRef,
			AuditoriaRef:           reserva.AuditoriaRef,
			EventoRef:              reserva.EventoRef,
			CorrelacionVECRef:      reserva.CorrelacionVECRef,
			DecisionVECRef:         reserva.DecisionVECRef,
			AmbitoIdempotenciaHMAC: reserva.AmbitoIdempotenciaHMAC,
			HuellaSemanticaHMAC:    reserva.HuellaSemanticaHMAC,
			RevisionCercado:        reserva.RevisionCercado,
			ObservadaEnDB:          reserva.ObservadaEnDB,
		},
	)
}

// ResultadoAplicadoOperacionDecisionCobertura solo existe si C2 produjo
// efecto. La referencia de decisión deriva de la misma huella.
type ResultadoAplicadoOperacionDecisionCobertura struct {
	DecisionCoberturaRef    string
	DecisionCoberturaHuella string
	VersionResultante       uint64
	EventoRef               string
	ActuacionRef            string
}

// ResultadoDenegadoVECOperacionDecisionCobertura acredita un terminal sin
// efecto C2. La prueba VEC común del recibo explica la denegación.
type ResultadoDenegadoVECOperacionDecisionCobertura struct{}

// ReciboOperacionDecisionCobertura es el replay mínimo. Contiene exactamente
// una rama terminal: aplicada o denegada por VEC. Los sellos ligan el resto de
// la semántica sin repetir identidad, motivo ni predecesora.
type ReciboOperacionDecisionCobertura struct {
	ReciboRef               string
	ReservaRef              string
	AuditoriaRef            string
	CorrelacionVECRef       string
	DecisionVECRef          string
	DecisionVECHuellaSHA256 string
	CodigoProbatorioVEC     string
	ConcedidaVEC            bool
	RevisionCercado         uint64
	AmbitoIdempotenciaHMAC  string
	HuellaSemanticaHMAC     string
	ConfirmadaEn            time.Time
	Aplicada                *ResultadoAplicadoOperacionDecisionCobertura
	DenegadaVEC             *ResultadoDenegadoVECOperacionDecisionCobertura
}

func (r ReciboOperacionDecisionCobertura) ValidarPara(
	solicitud SolicitudConsultarOperacionDecisionCoberturaConfirmada,
) error {
	_, _, versionExpediente, err := solicitud.coordenadas()
	if err != nil ||
		!domain.ReferenciaOpacaValida(r.ReciboRef) ||
		!domain.ReferenciaOpacaValida(r.ReservaRef) ||
		!domain.ReferenciaOpacaValida(r.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.CorrelacionVECRef) ||
		!domain.ReferenciaOpacaValida(r.DecisionVECRef) ||
		!huellaSHA256OperacionDecisionCoberturaValida(
			r.DecisionVECHuellaSHA256,
		) ||
		!dominiovec.CodigoResultadoEvaluacionAutorizacionV3Valido(
			r.CodigoProbatorioVEC,
			r.ConcedidaVEC,
		) ||
		r.RevisionCercado == 0 ||
		r.RevisionCercado > MaximoEnteroSeguroOperacionDecisionCobertura ||
		!solicitud.contienePar(
			r.AmbitoIdempotenciaHMAC,
			r.HuellaSemanticaHMAC,
		) ||
		!instanteOperacionDecisionCoberturaValido(r.ConfirmadaEn) ||
		(r.Aplicada == nil) == (r.DenegadaVEC == nil) {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	if r.Aplicada != nil {
		if !r.ConcedidaVEC ||
			!referenciaDecisionCoberturaLigadaAHuella(
				r.Aplicada.DecisionCoberturaRef,
				r.Aplicada.DecisionCoberturaHuella,
			) ||
			!domain.ReferenciaOpacaValida(r.Aplicada.EventoRef) ||
			!domain.ReferenciaOpacaValida(r.Aplicada.ActuacionRef) ||
			r.Aplicada.VersionResultante != versionExpediente+1 ||
			r.Aplicada.VersionResultante >
				MaximoEnteroSeguroOperacionDecisionCobertura {
			return ErrOperacionDecisionCoberturaIdempotenteInvalida
		}
		return nil
	}
	if r.ConcedidaVEC {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return nil
}

// ValidarParaReservaCongelada coteja el recibo terminal con las referencias
// preasignadas. ValidarPara solo acredita coherencia local/HMAC; no acredita
// que el recibo se leyó de la misma fila durable. El adaptador O4 debe
// restaurarlo de esa fila y repetir este cotejo dentro de su transacción.
func (r ReciboOperacionDecisionCobertura) ValidarParaReservaCongelada(
	solicitud SolicitudReservarOperacionDecisionCobertura,
	reserva DatosReservaPropietariaOperacionDecisionCobertura,
) error {
	consulta, err := solicitud.consultaConfirmada()
	terminal, errTerminal :=
		reservaTerminalDesdePropietariaOperacionDecisionCobertura(
			solicitud,
			reserva,
		)
	if err != nil || errTerminal != nil ||
		r.validarParaReservaTerminal(consulta, terminal) != nil {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return nil
}

func (r ReciboOperacionDecisionCobertura) validarParaReservaTerminal(
	solicitud SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	reserva ReservaTerminalOperacionDecisionCobertura,
) error {
	if reserva.validarPara(solicitud) != nil ||
		r.ValidarPara(solicitud) != nil ||
		!referenciasOperacionDecisionCoberturaIguales(
			r.ReservaRef, reserva.datos.ReservaRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			r.ReciboRef, reserva.datos.ReciboRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			r.AuditoriaRef, reserva.datos.AuditoriaRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			r.CorrelacionVECRef, reserva.datos.CorrelacionVECRef,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			r.DecisionVECRef, reserva.datos.DecisionVECRef,
		) ||
		r.RevisionCercado != reserva.datos.RevisionCercado ||
		r.ConfirmadaEn.Before(reserva.datos.ObservadaEnDB) {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	if r.Aplicada != nil &&
		(!referenciasOperacionDecisionCoberturaIguales(
			r.Aplicada.EventoRef, reserva.datos.EventoRef,
		) ||
			!referenciasOperacionDecisionCoberturaIguales(
				r.Aplicada.ActuacionRef, reserva.datos.ActuacionRef,
			)) {
		return ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return nil
}

func (r ReciboOperacionDecisionCobertura) ResultadoAplicado() (
	ResultadoAplicadoOperacionDecisionCobertura,
	bool,
) {
	if r.Aplicada == nil || r.DenegadaVEC != nil {
		return ResultadoAplicadoOperacionDecisionCobertura{}, false
	}
	return *r.Aplicada, true
}

func (r ReciboOperacionDecisionCobertura) ResultadoDenegadoVEC() (
	ResultadoDenegadoVECOperacionDecisionCobertura,
	bool,
) {
	if r.DenegadaVEC == nil || r.Aplicada != nil {
		return ResultadoDenegadoVECOperacionDecisionCobertura{}, false
	}
	return *r.DenegadaVEC, true
}

func clonarReciboOperacionDecisionCobertura(
	recibo ReciboOperacionDecisionCobertura,
) ReciboOperacionDecisionCobertura {
	if recibo.Aplicada != nil {
		aplicada := *recibo.Aplicada
		recibo.Aplicada = &aplicada
	}
	if recibo.DenegadaVEC != nil {
		denegada := *recibo.DenegadaVEC
		recibo.DenegadaVEC = &denegada
	}
	return recibo
}

type datosPreparacionOperacionDecisionCobertura struct {
	estado    EstadoPreparacionOperacionDecisionCobertura
	ambito    string
	semantica string
	propiedad *DatosReservaPropietariaOperacionDecisionCobertura
	terminal  *ReservaTerminalOperacionDecisionCobertura
	recibo    *ReciboOperacionDecisionCobertura
}

type PreparacionOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosPreparacionOperacionDecisionCobertura
}

func NuevaPreparacionOperacionDecisionCoberturaPropietaria(
	solicitud SolicitudReservarOperacionDecisionCobertura,
	datos DatosReservaPropietariaOperacionDecisionCobertura,
) (PreparacionOperacionDecisionCobertura, error) {
	if datos.validarPara(solicitud) != nil {
		return PreparacionOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	copia := clonarReservaPropietariaOperacionDecisionCobertura(datos)
	return PreparacionOperacionDecisionCobertura{
		datos: &datosPreparacionOperacionDecisionCobertura{
			estado:    PreparacionOperacionDecisionCoberturaPropietaria,
			ambito:    datos.AmbitoIdempotenciaHMAC,
			semantica: datos.HuellaSemanticaHMAC,
			propiedad: &copia,
		},
	}, nil
}

func NuevaPreparacionOperacionDecisionCoberturaOcupada(
	solicitud SolicitudReservarOperacionDecisionCobertura,
	ambitoHMAC string,
	semanticaHMAC string,
) (PreparacionOperacionDecisionCobertura, error) {
	consulta, err := solicitud.consultaConfirmada()
	if err != nil || !consulta.contienePar(ambitoHMAC, semanticaHMAC) {
		return PreparacionOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return PreparacionOperacionDecisionCobertura{
		datos: &datosPreparacionOperacionDecisionCobertura{
			estado: PreparacionOperacionDecisionCoberturaOcupada,
			ambito: ambitoHMAC, semantica: semanticaHMAC,
		},
	}, nil
}

func NuevaPreparacionOperacionDecisionCoberturaConfirmada(
	solicitud SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	reserva ReservaTerminalOperacionDecisionCobertura,
	recibo ReciboOperacionDecisionCobertura,
) (PreparacionOperacionDecisionCobertura, error) {
	if recibo.validarParaReservaTerminal(solicitud, reserva) != nil {
		return PreparacionOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	copia := clonarReciboOperacionDecisionCobertura(recibo)
	copiaReserva := clonarReservaTerminalOperacionDecisionCobertura(reserva)
	return PreparacionOperacionDecisionCobertura{
		datos: &datosPreparacionOperacionDecisionCobertura{
			estado:    PreparacionOperacionDecisionCoberturaConfirmada,
			ambito:    recibo.AmbitoIdempotenciaHMAC,
			semantica: recibo.HuellaSemanticaHMAC,
			terminal:  &copiaReserva,
			recibo:    &copia,
		},
	}, nil
}

func (p PreparacionOperacionDecisionCobertura) EstadoPara(
	solicitud SolicitudReservarOperacionDecisionCobertura,
) (EstadoPreparacionOperacionDecisionCobertura, error) {
	if p.datos == nil || !p.datos.estado.valida() ||
		!solicitud.consulta.contienePar(p.datos.ambito, p.datos.semantica) {
		return "", ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	switch p.datos.estado {
	case PreparacionOperacionDecisionCoberturaPropietaria:
		if p.datos.propiedad == nil ||
			p.datos.propiedad.validarPara(solicitud) != nil ||
			p.datos.terminal != nil || p.datos.recibo != nil {
			return "", ErrOperacionDecisionCoberturaIdempotenteInvalida
		}
	case PreparacionOperacionDecisionCoberturaOcupada:
		if p.datos.propiedad != nil || p.datos.terminal != nil ||
			p.datos.recibo != nil {
			return "", ErrOperacionDecisionCoberturaIdempotenteInvalida
		}
	case PreparacionOperacionDecisionCoberturaConfirmada:
		consulta, err := solicitud.consultaConfirmada()
		if err != nil || p.datos.propiedad != nil ||
			p.datos.terminal == nil || p.datos.recibo == nil ||
			p.datos.recibo.validarParaReservaTerminal(
				consulta,
				*p.datos.terminal,
			) != nil {
			return "", ErrOperacionDecisionCoberturaIdempotenteInvalida
		}
	}
	return p.datos.estado, nil
}

func (p PreparacionOperacionDecisionCobertura) DatosPropietariaPara(
	solicitud SolicitudReservarOperacionDecisionCobertura,
) (DatosReservaPropietariaOperacionDecisionCobertura, error) {
	estado, err := p.EstadoPara(solicitud)
	if err != nil || estado != PreparacionOperacionDecisionCoberturaPropietaria {
		return DatosReservaPropietariaOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return clonarReservaPropietariaOperacionDecisionCobertura(*p.datos.propiedad), nil
}

func (p PreparacionOperacionDecisionCobertura) ReciboConfirmadoPara(
	solicitud SolicitudConsultarOperacionDecisionCoberturaConfirmada,
) (ReciboOperacionDecisionCobertura, error) {
	if p.datos == nil ||
		p.datos.estado != PreparacionOperacionDecisionCoberturaConfirmada ||
		p.datos.propiedad != nil || p.datos.terminal == nil ||
		p.datos.recibo == nil ||
		p.datos.recibo.validarParaReservaTerminal(
			solicitud,
			*p.datos.terminal,
		) != nil {
		return ReciboOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return clonarReciboOperacionDecisionCobertura(*p.datos.recibo), nil
}

var ErrOperacionDecisionCoberturaOcupada = errors.New(
	"contratacion temporal: operacion de decision de cobertura ocupada",
)

// PreparadorOperacionDecisionCoberturaIdempotente reserva o reapropia con CAS
// de token+RevisionCercado en base de datos. PropiedadHasta solo habilita el
// intento de reapropiación; nunca sustituye ese CAS ni autoriza la operación.
type PreparadorOperacionDecisionCoberturaIdempotente interface {
	ConsultarOperacionDecisionCoberturaConfirmada(
		context.Context,
		SolicitudConsultarOperacionDecisionCoberturaConfirmada,
	) (PreparacionOperacionDecisionCobertura, bool, error)
	ReservarOReapropiarOperacionDecisionCobertura(
		context.Context,
		SolicitudReservarOperacionDecisionCobertura,
	) (PreparacionOperacionDecisionCobertura, error)
}
