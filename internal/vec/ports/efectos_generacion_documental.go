package ports

import (
	"context"
	"errors"
	"time"
)

var (
	ErrReservaEfectoGeneracionDocumentalInvalida = errors.New("vec: reserva de efecto documental invalida")
	ErrPasoGeneracionDocumentalIndeterminado     = errors.New("vec: resultado de paso documental indeterminado")
)

type EstadoPasoEfectoGeneracionDocumental string

const (
	EstadoPasoEfectoDocumentalReservado     EstadoPasoEfectoGeneracionDocumental = "reservado"
	EstadoPasoEfectoDocumentalConfirmado    EstadoPasoEfectoGeneracionDocumental = "confirmado"
	EstadoPasoEfectoDocumentalIndeterminado EstadoPasoEfectoGeneracionDocumental = "indeterminado"
)

// SolicitudReservarEfectoGeneracionDocumental no permite proponer la tupla
// durable como campos libres. El repositorio la extrae de la capacidad y del
// manifiesto opacos y consume DecisionRef de forma unica en la misma
// transaccion que reserva EfectoRef y todos sus pasos.
type SolicitudReservarEfectoGeneracionDocumental struct {
	Contexto   ContextoOperacionAlmacen
	Manifiesto ManifiestoGeneracionDocumental
}

func (s SolicitudReservarEfectoGeneracionDocumental) ValidarEn(instante time.Time) error {
	if s.Contexto.ValidarParaEn(AccionAlmacenEscribir, instante) != nil ||
		!s.Contexto.coincideManifiestoGeneracionDocumental(s.Manifiesto) ||
		!s.Contexto.esPrimerPasoManifiesto() {
		return ErrReservaEfectoGeneracionDocumentalInvalida
	}
	return nil
}

// EstadoPasoDuraderoGeneracionDocumental es una proyeccion no autoritativa de
// la reserva. Un paso confirmado conserva tambien el conector exacto para que
// un replay pueda reconstruir la misma auditoria sin inventar metadatos. Un
// paso indeterminado no contiene un objeto asumido: requiere reconciliacion
// expresa antes de confirmar o volver a ejecutar.
type EstadoPasoDuraderoGeneracionDocumental struct {
	PasoRef               PasoOperacionAlmacen
	HuellaPasoSHA256      string
	Estado                EstadoPasoEfectoGeneracionDocumental
	Objeto                ReferenciaObjetoAlmacen
	ConectorID            string
	EvidenciaOperacionRef string
	IncidenteRef          string
}

func (e EstadoPasoDuraderoGeneracionDocumental) validar() error {
	if e.PasoRef == "" || contieneComodinContextoAlmacen(string(e.PasoRef)) ||
		!esSHA256Hexadecimal(e.HuellaPasoSHA256) {
		return ErrReservaEfectoGeneracionDocumentalInvalida
	}
	switch e.Estado {
	case EstadoPasoEfectoDocumentalReservado:
		if e.Objeto != (ReferenciaObjetoAlmacen{}) || e.ConectorID != "" ||
			e.EvidenciaOperacionRef != "" || e.IncidenteRef != "" {
			return ErrReservaEfectoGeneracionDocumentalInvalida
		}
	case EstadoPasoEfectoDocumentalConfirmado:
		if e.Objeto.Validar() != nil || !referenciaOpacaAlmacenValida(e.ConectorID, 128) ||
			!referenciaOpacaAlmacenValida(e.EvidenciaOperacionRef, 512) ||
			e.IncidenteRef != "" {
			return ErrReservaEfectoGeneracionDocumentalInvalida
		}
	case EstadoPasoEfectoDocumentalIndeterminado:
		if e.Objeto != (ReferenciaObjetoAlmacen{}) || e.ConectorID != "" || e.EvidenciaOperacionRef != "" ||
			!referenciaOpacaAlmacenValida(e.IncidenteRef, 512) {
			return ErrReservaEfectoGeneracionDocumentalInvalida
		}
	default:
		return ErrReservaEfectoGeneracionDocumentalInvalida
	}
	return nil
}

type ResultadoReservaEfectoGeneracionDocumental struct {
	ReservaRef             string
	EfectoRef              string
	HuellaDecisionSHA256   string
	HuellaPlanEfectoSHA256 string
	HuellaManifiestoSHA256 string
	Repetida               bool
	Pasos                  []EstadoPasoDuraderoGeneracionDocumental
}

func (r ResultadoReservaEfectoGeneracionDocumental) ValidarContra(
	solicitud SolicitudReservarEfectoGeneracionDocumental,
) error {
	contexto, errContexto := solicitud.Contexto.Proyeccion()
	manifiesto, errManifiesto := solicitud.Manifiesto.Proyeccion()
	if errContexto != nil || errManifiesto != nil ||
		!solicitud.Contexto.coincideManifiestoGeneracionDocumental(solicitud.Manifiesto) ||
		!referenciaOpacaAlmacenValida(r.ReservaRef, 512) ||
		r.EfectoRef != contexto.EfectoRef || r.HuellaDecisionSHA256 != contexto.HuellaDecisionSHA256 ||
		r.HuellaPlanEfectoSHA256 != contexto.HuellaPlanEfectoSHA256 ||
		r.HuellaManifiestoSHA256 != manifiesto.HuellaManifiestoSHA256 ||
		len(r.Pasos) != len(manifiesto.Pasos) {
		return ErrReservaEfectoGeneracionDocumentalInvalida
	}
	for indice, estado := range r.Pasos {
		declarado := manifiesto.Pasos[indice]
		if estado.validar() != nil || estado.PasoRef != declarado.PasoRef ||
			estado.HuellaPasoSHA256 != declarado.HuellaPasoSHA256 ||
			(!r.Repetida && estado.Estado != EstadoPasoEfectoDocumentalReservado) {
			return ErrReservaEfectoGeneracionDocumentalInvalida
		}
	}
	return nil
}

// SolicitudConfirmarPasoGeneracionDocumental confirma el resultado tecnico
// exacto de un paso ya reservado. No vuelve a consumir la decision: el
// repositorio debe comprobar ReservaRef y la tupla de plan que consumio antes
// del efecto remoto, y hacer la transicion de estado de manera condicional.
type SolicitudConfirmarPasoGeneracionDocumental struct {
	ReservaRef string
	Contexto   ContextoOperacionAlmacen
	Guardado   ContenidoDocumentoGuardado
}

func (s SolicitudConfirmarPasoGeneracionDocumental) Validar() error {
	if !referenciaOpacaAlmacenValida(s.ReservaRef, 512) ||
		s.Contexto.validarResultadoPasoGeneracionDocumental(s.Guardado) != nil {
		return ErrReservaEfectoGeneracionDocumentalInvalida
	}
	return nil
}

// SolicitudMarcarPasoGeneracionDocumentalIndeterminado se usa cuando no puede
// saberse si el almacen aplico el efecto. IncidenteRef es una referencia opaca
// a la traza; nunca incorpora el error, una URL ni datos del documento.
type SolicitudMarcarPasoGeneracionDocumentalIndeterminado struct {
	ReservaRef   string
	Contexto     ContextoOperacionAlmacen
	IncidenteRef string
}

func (s SolicitudMarcarPasoGeneracionDocumentalIndeterminado) Validar() error {
	proyeccion, err := s.Contexto.Proyeccion()
	if err != nil || proyeccion.HuellaManifiestoSHA256 == "" ||
		!esSHA256Hexadecimal(proyeccion.HuellaPasoSHA256) ||
		proyeccion.AccionTecnica != AccionAlmacenEscribir ||
		!referenciaOpacaAlmacenValida(s.ReservaRef, 512) ||
		!referenciaOpacaAlmacenValida(s.IncidenteRef, 512) ||
		contieneComodinContextoAlmacen(s.ReservaRef, s.IncidenteRef) {
		return ErrReservaEfectoGeneracionDocumentalInvalida
	}
	return nil
}

// RegistroEfectosGeneracionDocumental es deliberadamente un puerto distinto
// del almacen de objetos. Su adaptador productivo debe ser transaccional y
// duradero:
//   - Reservar consume una DecisionRef una sola vez y fija
//     (EfectoRef, HuellaDecision, HuellaPlan, HuellaManifiesto);
//   - Confirmar fija una sola respuesta tecnica para (EfectoRef, PasoRef);
//   - MarcarIndeterminado impide reintentos ciegos y fuerza reconciliacion.
//
// No se aporta implementacion en memoria como sustituto productivo: sin este
// puerto configurado la generacion con efectos remotos permanece cerrada.
type RegistroEfectosGeneracionDocumental interface {
	ReservarEfectoGeneracionDocumental(
		context.Context,
		SolicitudReservarEfectoGeneracionDocumental,
	) (ResultadoReservaEfectoGeneracionDocumental, error)
	ConfirmarPasoGeneracionDocumental(
		context.Context,
		SolicitudConfirmarPasoGeneracionDocumental,
	) error
	MarcarPasoGeneracionDocumentalIndeterminado(
		context.Context,
		SolicitudMarcarPasoGeneracionDocumentalIndeterminado,
	) error
}
