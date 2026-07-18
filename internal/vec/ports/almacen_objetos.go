package ports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"time"

	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
)

var (
	ErrSolicitudAlmacenInvalida              = almacencanonico.ErrSolicitudAlmacenInvalida
	ErrObjetoAlmacenNoEncontrado             = errors.New("vec: objeto de almacen no encontrado")
	ErrIntegridadObjetoAlmacen               = errors.New("vec: integridad del objeto de almacen no valida")
	ErrLimiteObjetoAlmacenExcedido           = errors.New("vec: limite del objeto de almacen excedido")
	ErrIdempotenciaAlmacenReutilizada        = errors.New("vec: idempotencia de almacen reutilizada para otra operacion")
	ErrCapacidadAlmacenNoDisponible          = errors.New("vec: capacidad de almacen no disponible")
	ErrTransicionZonaAlmacenNoPermitida      = errors.New("vec: transicion de zona de almacen no permitida")
	ErrRetencionObjetoAlmacenVigente         = errors.New("vec: retencion del objeto de almacen vigente")
	ErrObjetoAlmacenInmovilizado             = errors.New("vec: objeto de almacen inmovilizado")
	ErrObjetoAlmacenEliminado                = errors.New("vec: objeto de almacen eliminado")
	ErrSesionCargaDirectaNoValida            = errors.New("vec: sesion de carga directa no valida")
	ErrConfirmacionCargaDirectaNoDisponible  = errors.New("vec: confirmacion de carga directa no disponible")
	ErrInstruccionesCargaDirectaNoValidas    = almacencanonico.ErrInstruccionesCargaDirectaNoValidas
	ErrSerializacionCargaDirectaProhibida    = errors.New("vec: serializacion accidental de carga directa prohibida")
	ErrSelladoIdempotenciaCargaNoDisponible  = errors.New("vec: sellado de idempotencia de carga no disponible")
	ErrReciboCargaDirectaNoValido            = almacencanonico.ErrReciboCargaDirectaNoValido
	ErrReciboCargaDirectaNoDisponible        = errors.New("vec: verificacion del recibo de carga directa no disponible")
	ErrAtestacionReciboCargaDirectaNoValida  = errors.New("vec: atestacion de consumo de carga directa no valida")
	ErrRegistroReciboCargaDirectaConflicto   = errors.New("vec: indice de recibo de carga directa ya registrado")
	ErrConsumoReciboCargaDirectaDenegado     = errors.New("vec: consumo de recibo de carga directa denegado")
	ErrSerializacionReciboCargaProhibida     = almacencanonico.ErrSerializacionReciboCargaProhibida
	ErrSerializacionSeudonimizacionProhibida = almacencanonico.ErrSerializacionSeudonimizacionProhibida
	ErrSeudonimizacionAlmacenNoDisponible    = almacencanonico.ErrSeudonimizacionAlmacenNoDisponible
)

// Las acciones del puerto forman una lista positiva cerrada. Son operaciones
// tecnicas, no permisos de negocio, y nunca se infieren de la ruta, el rol o
// la finalidad. Una autorizacion para una accion no habilita ninguna otra.
const (
	AccionAlmacenEscribir               = almacencanonico.AccionEscribir
	AccionAlmacenLeer                   = almacencanonico.AccionLeer
	AccionAlmacenPrepararCargaDirecta   = almacencanonico.AccionPrepararCargaDirecta
	AccionAlmacenConfirmarCargaDirecta  = almacencanonico.AccionConfirmarCargaDirecta
	AccionAlmacenAbandonarCargaDirecta  = almacencanonico.AccionAbandonarCargaDirecta
	AccionAlmacenPromover               = almacencanonico.AccionPromover
	AccionAlmacenAplicarRetencion       = almacencanonico.AccionAplicarRetencion
	AccionAlmacenInmovilizar            = almacencanonico.AccionInmovilizar
	AccionAlmacenLevantarInmovilizacion = almacencanonico.AccionLevantarInmovilizacion
	AccionAlmacenEliminar               = almacencanonico.AccionEliminar
	AccionAlmacenAnalizarContenido      = almacencanonico.AccionAnalizarContenido
)

// ZonaAlmacen separa tecnicamente objetos que aun no son confiables de los
// que ya pueden incorporarse a un expediente. No representa un estado de
// negocio configurable.
type ZonaAlmacen = almacencanonico.Zona

const (
	ZonaAlmacenCuarentena = almacencanonico.ZonaCuarentena
	ZonaAlmacenAdmitida   = almacencanonico.ZonaAdmitida
)

// SolicitudSeudonimizarSujetoAlmacen mantiene el identificador interno fuera
// del contexto, evidencias y conectores de objetos. Solo el sellador local
// confiable lo revela durante la operacion HMAC con clave exclusiva.
type SolicitudSeudonimizarSujetoAlmacen = almacencanonico.SolicitudSeudonimizarSujetoAlmacen

func NuevaSolicitudSeudonimizarSujetoAlmacen(
	sujetoRef, ambitoRef string,
) (SolicitudSeudonimizarSujetoAlmacen, error) {
	return almacencanonico.NuevaSolicitudSeudonimizarSujetoAlmacen(sujetoRef, ambitoRef)
}

// SeudonimizadorSujetoAlmacen debe usar una clave y version propias; no se
// reutiliza la clave de sesiones, idempotencia ni huellas de solicitudes.
type SeudonimizadorSujetoAlmacen interface {
	SeudonimizarSujetoAlmacen(
		context.Context,
		SolicitudSeudonimizarSujetoAlmacen,
	) (string, error)
}

// CapacidadesAlmacenObjetos permite validar el perfil de despliegue al
// arrancar. Declarar una capacidad no basta: cada conector productivo debe
// superar sus pruebas de contrato y de recuperacion.
type CapacidadesAlmacenObjetos = almacencanonico.Capacidades

type RequisitosAlmacenObjetos = almacencanonico.Requisitos

func VerificarCapacidadesAlmacen(capacidades CapacidadesAlmacenObjetos, requisitos RequisitosAlmacenObjetos) error {
	if !almacencanonico.CapacidadesSatisfacen(capacidades, requisitos) {
		return ErrCapacidadAlmacenNoDisponible
	}
	return nil
}

type SolicitudEscribirObjeto struct {
	Contexto          ContextoOperacionAlmacen
	ClaveIdempotencia string
	Zona              ZonaAlmacen
	MIME              string
	Tamano            int64
	HuellaSHA256      string
	Contenido         io.Reader
}

func (s SolicitudEscribirObjeto) Validar() error {
	if err := s.Contexto.validarParaPaso(AccionAlmacenEscribir); err != nil {
		return err
	}
	if !referenciaOpacaAlmacenValida(s.ClaveIdempotencia, 512) || !s.Zona.Valida() ||
		!textoSeguroAlmacen(s.MIME, 255) ||
		s.Tamano < 1 || !esSHA256Hexadecimal(s.HuellaSHA256) || lectorContenidoNulo(s.Contenido) {
		return ErrSolicitudAlmacenInvalida
	}
	if err := s.Contexto.validarEscrituraContraManifiesto(
		s.ClaveIdempotencia, s.Zona, s.MIME, s.Tamano, s.HuellaSHA256,
	); err != nil {
		return err
	}
	return nil
}

type ReferenciaObjetoAlmacen = almacencanonico.ReferenciaObjetoAlmacen

type ObjetoAlmacenado = almacencanonico.ObjetoAlmacenado

// EvidenciaOperacionAlmacen es el recibo tecnico que el caso de uso incorpora
// a la auditoria probatoria. No contiene rutas, URL, nombres ni datos de la
// persona interesada.
type EvidenciaOperacionAlmacen struct {
	Referencia             string
	ConectorID             string
	EsquemaContexto        string
	AccionNegocio          string
	Accion                 string
	EfectoRef              string
	HuellaPlanEfectoSHA256 string
	HuellaManifiestoSHA256 string
	HuellaPasoSHA256       string
	PasoRef                PasoOperacionAlmacen
	HuellaDecisionSHA256   string
	Objeto                 ReferenciaObjetoAlmacen
	OperacionRef           string
	CorrelacionRef         string
	AutorizacionRef        string
	Finalidad              string
	Clasificacion          string
	RealizadaEn            time.Time
	CargaRef               string
	SujetoSeudonimoHMAC    string
	RecursoRef             string
	ModuloID               string
	HuellaSolicitudHMAC    string
	FundamentoRef          string
	// ReintentoIdempotente distingue una respuesta repetida de la evidencia
	// que creo realmente el objeto. La accion no cambia ni se relaja.
	ReintentoIdempotente bool
}

func (e EvidenciaOperacionAlmacen) Validar() error {
	return almacencanonico.ValidarEvidenciaOperacion(datosCanonicosEvidenciaAlmacen(e))
}

type ResultadoOperacionObjeto struct {
	Objeto    ObjetoAlmacenado
	Evidencia EvidenciaOperacionAlmacen
}

func (r ResultadoOperacionObjeto) Validar() error {
	return almacencanonico.ValidarResultadoOperacion(datosCanonicosResultadoAlmacen(r))
}

// ValidarEscritura coteja una escritura en flujo, incluida una repeticion
// idempotente inequívocamente marcada, con la solicitud exacta.
func (r ResultadoOperacionObjeto) ValidarEscritura(
	solicitud SolicitudEscribirObjeto,
	capacidades CapacidadesAlmacenObjetos,
) error {
	if solicitud.Validar() != nil || r.Validar() != nil || !capacidades.EscrituraEnFlujo ||
		!capacidades.ReferenciasOpacas || !capacidades.IntegridadSHA256 ||
		capacidades.TamanoMaximoObjeto < solicitud.Tamano ||
		r.Evidencia.Accion != AccionAlmacenEscribir || !evidenciaAlmacenLigada(r.Evidencia, solicitud.Contexto) ||
		r.Evidencia.FundamentoRef != "" ||
		r.Objeto.ConectorID != capacidades.ConectorID || r.Objeto.Zona != solicitud.Zona ||
		r.Objeto.MIME != solicitud.MIME || r.Objeto.Tamano != solicitud.Tamano ||
		r.Objeto.HuellaSHA256 != solicitud.HuellaSHA256 {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

// ValidarCargaDirecta comprueba la respuesta completa del conector contra la
// preparacion y la autorizacion de confirmacion. La huella declarada por el
// navegador solo se acepta si coincide con la calculada por el almacen sobre
// el objeto efectivamente recibido.
func (r ResultadoOperacionObjeto) ValidarCargaDirecta(
	preparacion SolicitudPrepararCargaDirecta,
	confirmacion SolicitudConfirmarCargaDirecta,
	capacidades CapacidadesAlmacenObjetos,
) error {
	if preparacion.Validar() != nil || confirmacion.Validar() != nil || r.Validar() != nil ||
		!capacidades.CargaDirectaTemporal || !capacidades.ReferenciasOpacas ||
		!capacidades.IntegridadSHA256 || capacidades.TamanoMaximoObjeto < preparacion.Tamano ||
		r.Objeto.ConectorID != capacidades.ConectorID ||
		r.Evidencia.Accion != AccionAlmacenConfirmarCargaDirecta ||
		r.Evidencia.FundamentoRef != confirmacion.comprobante.intencionRef ||
		r.Evidencia.RealizadaEn.Before(confirmacion.comprobante.consumidoEn) ||
		!r.Evidencia.RealizadaEn.Before(confirmacion.comprobante.expiraEn) ||
		!r.Evidencia.RealizadaEn.Before(confirmacion.comprobante.validaHasta) ||
		r.Objeto.Zona != ZonaAlmacenCuarentena || r.Objeto.MIME != preparacion.MIME ||
		r.Objeto.Tamano != preparacion.Tamano || r.Objeto.HuellaSHA256 != preparacion.HuellaSHA256 ||
		!contextosMismaOperacion(preparacion.Contexto, confirmacion.contexto) ||
		!evidenciaAlmacenLigada(r.Evidencia, confirmacion.contexto) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

// ValidarPromocion impide que una respuesta sustituya el contenido analizado
// por otro. En perfiles que preservan el original, la zona admitida debe usar
// una referencia distinta y conservar exactamente huella, tamano y MIME.
func (r ResultadoOperacionObjeto) ValidarPromocion(
	solicitud SolicitudPromoverObjeto,
	origen ObjetoAlmacenado,
	capacidades CapacidadesAlmacenObjetos,
) error {
	if solicitud.Validar() != nil || origen.Validar() != nil || r.Validar() != nil ||
		!capacidades.PromocionAtomica || !capacidades.ReferenciasOpacas ||
		!capacidades.IntegridadSHA256 || !capacidades.PreservaObjetoOriginal ||
		origen.Objeto != solicitud.Origen || origen.Zona != ZonaAlmacenCuarentena ||
		origen.Inmovilizado || origen.Eliminado || r.Objeto.Inmovilizado || r.Objeto.Eliminado ||
		(capacidades.RetencionAtomicaEnPromocion && r.Objeto.RetenidoHasta.IsZero()) ||
		(!capacidades.RetencionAtomicaEnPromocion && !r.Objeto.RetenidoHasta.IsZero()) ||
		r.Evidencia.Accion != AccionAlmacenPromover ||
		r.Evidencia.FundamentoRef != solicitud.EvidenciaAnalisisRef ||
		r.Objeto.ConectorID != origen.ConectorID || r.Objeto.ConectorID != capacidades.ConectorID ||
		r.Objeto.Zona != ZonaAlmacenAdmitida || r.Objeto.MIME != origen.MIME ||
		r.Objeto.Tamano != origen.Tamano || r.Objeto.HuellaSHA256 != origen.HuellaSHA256 ||
		!evidenciaAlmacenLigada(r.Evidencia, solicitud.Contexto) ||
		r.Objeto.AlmacenadoEn.Before(origen.AlmacenadoEn) ||
		r.Objeto.Objeto == origen.Objeto {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

type SolicitudAbrirObjeto struct {
	Contexto ContextoOperacionAlmacen
	Objeto   ReferenciaObjetoAlmacen
	Zona     ZonaAlmacen
	Limite   int64
}

func (s SolicitudAbrirObjeto) Validar() error {
	if err := s.Contexto.validarParaPaso(AccionAlmacenLeer); err != nil {
		return err
	}
	if err := s.Objeto.Validar(); err != nil {
		return ErrSolicitudAlmacenInvalida
	}
	if !s.Contexto.coincideObjeto(s.Objeto) || !s.Zona.Valida() || s.Limite < 1 {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

type LecturaObjetoAlmacen struct {
	Objeto    ObjetoAlmacenado
	Evidencia EvidenciaOperacionAlmacen
	Contenido io.ReadCloser
}

func (l LecturaObjetoAlmacen) ValidarContra(solicitud SolicitudAbrirObjeto) error {
	if err := solicitud.Validar(); err != nil || l.Objeto.Validar() != nil || l.Evidencia.Validar() != nil ||
		lectorContenidoNulo(l.Contenido) || l.Objeto.Objeto != solicitud.Objeto || l.Objeto.Zona != solicitud.Zona ||
		l.Objeto.Tamano > solicitud.Limite || l.Evidencia.Objeto != solicitud.Objeto ||
		l.Objeto.Eliminado || l.Evidencia.ConectorID != l.Objeto.ConectorID ||
		l.Evidencia.Accion != AccionAlmacenLeer || l.Evidencia.RealizadaEn.Before(l.Objeto.AlmacenadoEn) ||
		l.Evidencia.FundamentoRef != "" ||
		l.Evidencia.Referencia == l.Objeto.EvidenciaCreacionRef ||
		!evidenciaAlmacenLigada(l.Evidencia, solicitud.Contexto) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

type SolicitudPromoverObjeto struct {
	Contexto             ContextoOperacionAlmacen
	ClaveIdempotencia    string
	Origen               ReferenciaObjetoAlmacen
	EvidenciaAnalisisRef string
}

func (s SolicitudPromoverObjeto) Validar() error {
	if err := s.Contexto.validarParaPaso(AccionAlmacenPromover); err != nil {
		return err
	}
	if err := s.Origen.Validar(); err != nil {
		return ErrSolicitudAlmacenInvalida
	}
	if !s.Contexto.coincideObjeto(s.Origen) ||
		!referenciaOpacaAlmacenValida(s.ClaveIdempotencia, 512) ||
		!referenciaOpacaAlmacenValida(s.EvidenciaAnalisisRef, 512) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

type SolicitudRetenerObjeto struct {
	Contexto    ContextoOperacionAlmacen
	Objeto      ReferenciaObjetoAlmacen
	PoliticaRef string
	Hasta       time.Time
}

func (s SolicitudRetenerObjeto) Validar() error {
	if s.Contexto.validarParaPaso(AccionAlmacenAplicarRetencion) != nil || s.Objeto.Validar() != nil ||
		!s.Contexto.coincideObjeto(s.Objeto) ||
		!referenciaOpacaAlmacenValida(s.PoliticaRef, 512) || s.Hasta.IsZero() ||
		s.Hasta.Location() != time.UTC {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

func (s SolicitudRetenerObjeto) ValidarEn(instante time.Time) error {
	if s.Validar() != nil || instante.IsZero() || !s.Hasta.After(instante.UTC()) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

type SolicitudInmovilizarObjeto struct {
	Contexto      ContextoOperacionAlmacen
	Objeto        ReferenciaObjetoAlmacen
	AprobacionRef string
	Motivo        string
}

func (s SolicitudInmovilizarObjeto) Validar() error {
	if s.Contexto.validarParaPaso(AccionAlmacenInmovilizar) != nil || s.Objeto.Validar() != nil ||
		!referenciaOpacaAlmacenValida(s.AprobacionRef, 512) || !textoSeguroAlmacen(s.Motivo, 1024) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

type SolicitudLevantarInmovilizacionObjeto struct {
	Contexto      ContextoOperacionAlmacen
	Objeto        ReferenciaObjetoAlmacen
	AprobacionRef string
	Motivo        string
}

func (s SolicitudLevantarInmovilizacionObjeto) Validar() error {
	if s.Contexto.validarParaPaso(AccionAlmacenLevantarInmovilizacion) != nil || s.Objeto.Validar() != nil ||
		!referenciaOpacaAlmacenValida(s.AprobacionRef, 512) || !textoSeguroAlmacen(s.Motivo, 1024) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

type SolicitudEliminarObjeto struct {
	Contexto      ContextoOperacionAlmacen
	Objeto        ReferenciaObjetoAlmacen
	AprobacionRef string
	Motivo        string
}

func (s SolicitudEliminarObjeto) Validar() error {
	if s.Contexto.validarParaPaso(AccionAlmacenEliminar) != nil || s.Objeto.Validar() != nil ||
		!referenciaOpacaAlmacenValida(s.AprobacionRef, 512) || !textoSeguroAlmacen(s.Motivo, 1024) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

// ValidarRetencion exige que el conector solo modifique la fecha de
// retencion, nunca acorte una vigente ni altere los bytes o su custodia.
func (r ResultadoOperacionObjeto) ValidarRetencion(
	solicitud SolicitudRetenerObjeto,
	anterior ObjetoAlmacenado,
) error {
	if solicitud.ValidarEn(r.Evidencia.RealizadaEn) != nil ||
		validarResultadoMutacionAlmacen(r, anterior, solicitud.Contexto,
			AccionAlmacenAplicarRetencion, solicitud.PoliticaRef) != nil ||
		!r.Objeto.RetenidoHasta.Equal(solicitud.Hasta) || r.Objeto.Inmovilizado != anterior.Inmovilizado ||
		(!anterior.RetenidoHasta.IsZero() && r.Objeto.RetenidoHasta.Before(anterior.RetenidoHasta)) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

func (r ResultadoOperacionObjeto) ValidarInmovilizacion(
	solicitud SolicitudInmovilizarObjeto,
	anterior ObjetoAlmacenado,
) error {
	if solicitud.Validar() != nil || anterior.Inmovilizado ||
		validarResultadoMutacionAlmacen(r, anterior, solicitud.Contexto,
			AccionAlmacenInmovilizar, solicitud.AprobacionRef) != nil ||
		!r.Objeto.Inmovilizado || r.Objeto.RetenidoHasta != anterior.RetenidoHasta {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

func (r ResultadoOperacionObjeto) ValidarLevantamientoInmovilizacion(
	solicitud SolicitudLevantarInmovilizacionObjeto,
	anterior ObjetoAlmacenado,
) error {
	if solicitud.Validar() != nil || !anterior.Inmovilizado ||
		validarResultadoMutacionAlmacen(r, anterior, solicitud.Contexto,
			AccionAlmacenLevantarInmovilizacion, solicitud.AprobacionRef) != nil ||
		r.Objeto.Inmovilizado || r.Objeto.RetenidoHasta != anterior.RetenidoHasta {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

// ValidarEliminacion mantiene la eliminacion como operacion privilegiada
// separada: exige aprobacion exacta y rechaza bloqueo o retencion vigentes en
// el instante acreditado por el conector.
func (e EvidenciaOperacionAlmacen) ValidarEliminacion(
	solicitud SolicitudEliminarObjeto,
	anterior ObjetoAlmacenado,
) error {
	if solicitud.Validar() != nil || anterior.Validar() != nil || anterior.Eliminado || anterior.Inmovilizado ||
		e.Validar() != nil || e.Accion != AccionAlmacenEliminar || e.ReintentoIdempotente ||
		e.Objeto != solicitud.Objeto || e.Objeto != anterior.Objeto || e.ConectorID != anterior.ConectorID ||
		e.FundamentoRef != solicitud.AprobacionRef || e.RealizadaEn.Before(anterior.AlmacenadoEn) ||
		e.Referencia == anterior.EvidenciaCreacionRef ||
		(!anterior.RetenidoHasta.IsZero() && anterior.RetenidoHasta.After(e.RealizadaEn)) ||
		!evidenciaAlmacenLigada(e, solicitud.Contexto) {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

func validarResultadoMutacionAlmacen(
	resultado ResultadoOperacionObjeto,
	anterior ObjetoAlmacenado,
	contexto ContextoOperacionAlmacen,
	accion, fundamentoRef string,
) error {
	return almacencanonico.ValidarResultadoMutacion(
		datosCanonicosResultadoAlmacen(resultado), anterior, accion, fundamentoRef,
		evidenciaAlmacenLigada(resultado.Evidencia, contexto),
	)
}

// AlmacenObjetos es el puerto estable del nucleo. Filesystem, S3, una cabina,
// un gestor documental o una nube privada se conectan detras de este contrato.
// No expone rutas, buckets ni operaciones de listado.
type AlmacenObjetos interface {
	Capacidades(context.Context) (CapacidadesAlmacenObjetos, error)
	Escribir(context.Context, SolicitudEscribirObjeto) (ResultadoOperacionObjeto, error)
	Abrir(context.Context, SolicitudAbrirObjeto) (LecturaObjetoAlmacen, error)
	Promover(context.Context, SolicitudPromoverObjeto) (ResultadoOperacionObjeto, error)
	AplicarRetencion(context.Context, SolicitudRetenerObjeto) (ResultadoOperacionObjeto, error)
	Inmovilizar(context.Context, SolicitudInmovilizarObjeto) (ResultadoOperacionObjeto, error)
	LevantarInmovilizacion(context.Context, SolicitudLevantarInmovilizacionObjeto) (ResultadoOperacionObjeto, error)
	Eliminar(context.Context, SolicitudEliminarObjeto) (EvidenciaOperacionAlmacen, error)
}

type SolicitudPrepararCargaDirecta struct {
	Contexto          ContextoOperacionAlmacen
	ClaveIdempotencia string
	MIME              string
	Tamano            int64
	HuellaSHA256      string
	ExpiraEn          time.Time
}

func (s SolicitudPrepararCargaDirecta) Validar() error {
	if err := s.Contexto.validarParaPaso(AccionAlmacenPrepararCargaDirecta); err != nil {
		return err
	}
	if !referenciaOpacaAlmacenValida(s.ClaveIdempotencia, 512) || !textoSeguroAlmacen(s.MIME, 255) ||
		s.Tamano < 1 || !esSHA256Hexadecimal(s.HuellaSHA256) || s.ExpiraEn.IsZero() ||
		s.ExpiraEn.Location() != time.UTC {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

// InstruccionesCargaDirecta contiene un secreto de corta duracion. No debe
// persistirse, auditarse ni incluirse en trazas, metricas o mensajes.
type InstruccionesCargaDirecta struct {
	canonicas almacencanonico.InstruccionesCargaDirecta
}

type MetodoCargaDirecta = almacencanonico.MetodoCargaDirecta

const (
	MetodoCargaDirectaPUT  = almacencanonico.MetodoCargaDirectaPUT
	MetodoCargaDirectaPOST = almacencanonico.MetodoCargaDirectaPOST
)

// CabeceraCargaDirecta es una condicion puntual de la carga, no una
// credencial general. Se mantiene dentro de InstruccionesCargaDirecta hasta
// la revelacion deliberada para impedir que aparezca en registros.
type CabeceraCargaDirecta = almacencanonico.CabeceraCargaDirecta

// NuevasInstruccionesCargaDirecta solo acepta una concesion corta, limitada a
// un destino HTTPS y a un tamano. El conector no puede devolver mapas mutables
// ni una credencial utilizable para listar, leer o elegir otro objeto.
func NuevasInstruccionesCargaDirecta(
	conectorID, sesionRef string,
	metodo MetodoCargaDirecta,
	destino string,
	cabeceras []CabeceraCargaDirecta,
	emitidaEn, expiraEn time.Time,
	tamanoMaximo int64,
) (InstruccionesCargaDirecta, error) {
	canonicas, err := almacencanonico.NuevasInstruccionesCargaDirecta(
		almacencanonico.DatosInstruccionesCargaDirecta{
			ConectorID: conectorID, SesionRef: sesionRef, Metodo: metodo, Destino: destino,
			Cabeceras: cabeceras, EmitidaEn: emitidaEn, ExpiraEn: expiraEn, TamanoMaximo: tamanoMaximo,
		},
	)
	if err != nil {
		return InstruccionesCargaDirecta{}, err
	}
	return InstruccionesCargaDirecta{canonicas: canonicas}, nil
}

// NuevasInstruccionesCargaDirectaParaSolicitud es el constructor que deben
// usar los conectores productivos. Ademas de acotar la concesion, deja dentro
// del valor opaco una huella de todos los datos de la solicitud para que el
// nucleo pueda detectar respuestas cruzadas, repetidas o fabricadas por un
// conector defectuoso.
func NuevasInstruccionesCargaDirectaParaSolicitud(
	solicitud SolicitudPrepararCargaDirecta,
	conectorID, sesionRef string,
	metodo MetodoCargaDirecta,
	destino string,
	cabeceras []CabeceraCargaDirecta,
	emitidaEn time.Time,
) (InstruccionesCargaDirecta, error) {
	if err := solicitud.Validar(); err != nil {
		return InstruccionesCargaDirecta{}, err
	}
	instrucciones, err := NuevasInstruccionesCargaDirecta(
		conectorID,
		sesionRef,
		metodo,
		destino,
		cabeceras,
		emitidaEn,
		solicitud.ExpiraEn,
		solicitud.Tamano,
	)
	if err != nil {
		return InstruccionesCargaDirecta{}, err
	}
	canonicas, err := instrucciones.canonicas.VincularSolicitud(huellaSolicitudCargaDirecta(solicitud))
	if err != nil {
		return InstruccionesCargaDirecta{}, err
	}
	instrucciones.canonicas = canonicas
	return instrucciones, nil
}

func (i InstruccionesCargaDirecta) Validar() error {
	return i.canonicas.Validar()
}

func (i InstruccionesCargaDirecta) VigenteEn(instante time.Time) bool {
	return i.canonicas.VigenteEn(instante)
}

// ValidarContra impide aceptar un destino que no figure exactamente en el
// perfil de despliegue publicado para el mismo conector.
func (i InstruccionesCargaDirecta) ValidarContra(capacidades CapacidadesAlmacenObjetos) error {
	return i.canonicas.ValidarContra(capacidades)
}

func (i InstruccionesCargaDirecta) ValidarPara(
	solicitud SolicitudPrepararCargaDirecta,
	capacidades CapacidadesAlmacenObjetos,
) error {
	if err := solicitud.Validar(); err != nil || i.canonicas.ValidarPara(
		solicitud.Tamano, solicitud.ExpiraEn, huellaSolicitudCargaDirecta(solicitud), capacidades,
	) != nil {
		return ErrInstruccionesCargaDirectaNoValidas
	}
	return nil
}

// SolicitudEmitirReciboCargaDirecta encapsula la sesion temporal para que no
// circule como un campo registrable. El emisor firma tambien el vinculo
// inmutable de carga, sujeto seudonimizado, recurso, modulo y solicitud.
type SolicitudEmitirReciboCargaDirecta struct {
	contexto               ContextoOperacionAlmacen
	sesionRef              string
	expiraEn               time.Time
	vinculoSolicitudSHA256 string
}

func (s SolicitudEmitirReciboCargaDirecta) Validar() error {
	if s.contexto.validarParaPaso(AccionAlmacenPrepararCargaDirecta) != nil ||
		!referenciaOpacaAlmacenValida(s.sesionRef, 512) || s.expiraEn.IsZero() ||
		s.expiraEn.Location() != time.UTC ||
		!esSHA256Hexadecimal(s.vinculoSolicitudSHA256) {
		return ErrReciboCargaDirectaNoValido
	}
	return nil
}

// RevelarParaEmision es el unico punto donde el adaptador MAC puede obtener
// la sesion. No debe copiarla a errores, trazas o persistencia.
func (s SolicitudEmitirReciboCargaDirecta) RevelarParaEmision() (
	contexto ContextoOperacionAlmacen,
	sesionRef string,
	expiraEn time.Time,
	vinculoSolicitudSHA256 string,
	err error,
) {
	if err = s.Validar(); err != nil {
		return ContextoOperacionAlmacen{}, "", time.Time{}, "", err
	}
	return s.contexto, s.sesionRef, s.expiraEn, s.vinculoSolicitudSHA256, nil
}

func (SolicitudEmitirReciboCargaDirecta) String() string {
	return "[SOLICITUD-EMITIR-RECIBO-CARGA-DIRECTA-CONFIDENCIAL]"
}
func (SolicitudEmitirReciboCargaDirecta) GoString() string {
	return "[SOLICITUD-EMITIR-RECIBO-CARGA-DIRECTA-CONFIDENCIAL]"
}

func (s SolicitudEmitirReciboCargaDirecta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}

func (SolicitudEmitirReciboCargaDirecta) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

func (SolicitudEmitirReciboCargaDirecta) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

func (s SolicitudEmitirReciboCargaDirecta) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

type EmisorReciboCargaDirecta interface {
	EmitirReciboCargaDirecta(
		context.Context,
		SolicitudEmitirReciboCargaDirecta,
	) (ReciboCargaDirecta, error)
}

func (i InstruccionesCargaDirecta) EmitirReciboConfirmacion(
	ctx context.Context,
	solicitud SolicitudPrepararCargaDirecta,
	capacidades CapacidadesAlmacenObjetos,
	emisor EmisorReciboCargaDirecta,
) (ReciboCargaDirecta, error) {
	if dependenciaPuertoAlmacenNula(emisor) || i.ValidarPara(solicitud, capacidades) != nil {
		return ReciboCargaDirecta{}, ErrReciboCargaDirectaNoDisponible
	}
	datos, vinculoSolicitudSHA256, err := i.canonicas.DatosVerificados()
	if err != nil {
		return ReciboCargaDirecta{}, ErrReciboCargaDirectaNoDisponible
	}
	peticion := SolicitudEmitirReciboCargaDirecta{
		contexto: solicitud.Contexto, sesionRef: datos.SesionRef, expiraEn: datos.ExpiraEn,
		vinculoSolicitudSHA256: vinculoSolicitudSHA256,
	}
	if peticion.Validar() != nil {
		return ReciboCargaDirecta{}, ErrReciboCargaDirectaNoDisponible
	}
	recibo, err := emisor.EmitirReciboCargaDirecta(ctx, peticion)
	if err != nil || !recibo.Valido() {
		return ReciboCargaDirecta{}, ErrReciboCargaDirectaNoDisponible
	}
	return recibo, nil
}

// RevelarParaEntrega es el unico punto que expone la concesion. El adaptador
// HTTP debe copiarla inmediatamente a una respuesta no almacenable y borrar
// sus referencias; nunca debe pasar el valor InstruccionesCargaDirecta a un
// serializador o registrador generico.
func (i InstruccionesCargaDirecta) RevelarParaEntrega() (
	sesionRef string,
	metodo MetodoCargaDirecta,
	destino string,
	cabeceras []CabeceraCargaDirecta,
	expiraEn time.Time,
	tamanoMaximo int64,
	err error,
) {
	datos, _, err := i.canonicas.DatosVerificados()
	if err != nil {
		return "", "", "", nil, time.Time{}, 0, err
	}
	return datos.SesionRef, datos.Metodo, datos.Destino, datos.Cabeceras,
		datos.ExpiraEn, datos.TamanoMaximo, nil
}

// SellarVinculoSesion permite persistir solo un HMAC de la referencia de
// sesion sin revelarla al caso de uso. El sellador usa una clave exclusiva y
// devuelve un identificador versionado, nunca la referencia original.
func (i InstruccionesCargaDirecta) SellarVinculoSesion(
	ctx context.Context,
	sellador SelladorVinculoSesionCarga,
) (string, error) {
	if err := i.Validar(); err != nil || dependenciaPuertoAlmacenNula(sellador) {
		return "", ErrSelladoIdempotenciaCargaNoDisponible
	}
	datos, _, err := i.canonicas.DatosVerificados()
	if err != nil {
		return "", ErrSelladoIdempotenciaCargaNoDisponible
	}
	vinculo, err := sellador.SellarVinculoSesionCarga(ctx, datos.SesionRef)
	if err != nil || !hmacSHA256PuertoValido(vinculo) {
		return "", ErrSelladoIdempotenciaCargaNoDisponible
	}
	return vinculo, nil
}

// Abandonar revoca la concesion sin revelar la referencia de sesion al caso
// de uso. Se emplea como compensacion si no puede confirmarse la reserva
// transaccional del nucleo.
func (i InstruccionesCargaDirecta) Abandonar(
	ctx context.Context,
	gestor GestorCargaDirecta,
	contexto ContextoOperacionAlmacen,
) error {
	if err := i.Validar(); err != nil || dependenciaPuertoAlmacenNula(gestor) ||
		contexto.validarParaPaso(AccionAlmacenAbandonarCargaDirecta) != nil {
		return ErrSesionCargaDirectaNoValida
	}
	datos, _, err := i.canonicas.DatosVerificados()
	if err != nil {
		return ErrSesionCargaDirectaNoValida
	}
	return gestor.AbandonarCargaDirecta(ctx, contexto, datos.SesionRef)
}

func (InstruccionesCargaDirecta) String() string {
	return "[INSTRUCCIONES-CARGA-DIRECTA-CONFIDENCIALES]"
}
func (InstruccionesCargaDirecta) GoString() string {
	return "[INSTRUCCIONES-CARGA-DIRECTA-CONFIDENCIALES]"
}

func (i InstruccionesCargaDirecta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, i.String())
}

func (InstruccionesCargaDirecta) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCargaDirectaProhibida
}

func (InstruccionesCargaDirecta) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionCargaDirectaProhibida
}

// ReciboCargaDirecta es el valor opaco y de un uso entregado junto a la
// concesion temporal. Es un secreto efimero: solo se revela al transporte y
// nunca se persiste en claro, serializa por reflexion ni registra.
type ReciboCargaDirecta = almacencanonico.ReciboCargaDirecta

func NuevoReciboCargaDirecta(valor string) (ReciboCargaDirecta, error) {
	return almacencanonico.NuevoReciboCargaDirecta(valor)
}

// SolicitudConsumirReciboCargaDirecta se ejecuta antes de confirmar en el
// almacen. El adaptador debe verificar la MAC, caducidad y consumo atomico de
// un uso. Una consulta repetida siempre falla cerrada.
type SolicitudConsumirReciboCargaDirecta struct {
	Contexto  ContextoOperacionAlmacen
	SesionRef string
	Recibo    ReciboCargaDirecta
	// ValidaHasta es el menor limite entre la autorizacion vigente y la
	// sesion de carga. El repositorio debe cotejarlo con su hora durable en
	// la misma escritura condicional que consume el recibo.
	ValidaHasta time.Time
}

func (s SolicitudConsumirReciboCargaDirecta) Validar() error {
	if s.Contexto.validarParaPaso(AccionAlmacenConfirmarCargaDirecta) != nil ||
		!referenciaOpacaAlmacenValida(s.SesionRef, 512) || !s.Recibo.Valido() ||
		s.ValidaHasta.IsZero() || s.ValidaHasta.Location() != time.UTC {
		return ErrReciboCargaDirectaNoValido
	}
	return nil
}

func (SolicitudConsumirReciboCargaDirecta) String() string {
	return "[SOLICITUD-CONSUMIR-RECIBO-CARGA-DIRECTA-CONFIDENCIAL]"
}

func (SolicitudConsumirReciboCargaDirecta) GoString() string {
	return "[SOLICITUD-CONSUMIR-RECIBO-CARGA-DIRECTA-CONFIDENCIAL]"
}

func (s SolicitudConsumirReciboCargaDirecta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}

func (SolicitudConsumirReciboCargaDirecta) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

func (SolicitudConsumirReciboCargaDirecta) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

func (s SolicitudConsumirReciboCargaDirecta) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// ComprobanteConsumoReciboCargaDirecta no contiene el recibo ni la sesion.
// Su atestacion es un HMAC opaco de segunda fase. La mera forma valida no
// acredita nada: la fabrica de confirmacion exige siempre un verificador
// criptografico independiente antes de crear una solicitud para el conector.
type ComprobanteConsumoReciboCargaDirecta struct {
	indiceReciboHMAC    string
	grupoReciboHMAC     string
	vinculoReciboHMAC   string
	evidenciaConsumoRef string
	intencionRef        string
	huellaIntencionHMAC string
	registradoEn        time.Time
	consumidoEn         time.Time
	expiraEn            time.Time
	validaHasta         time.Time
	atestacionHMAC      string
}

// NuevoComprobanteConsumoReciboCargaDirecta acepta exclusivamente el HMAC
// opaco emitido despues del consumo durable. No lo verifica por si mismo: el
// verificador real es obligatorio en NuevaSolicitudConfirmarCargaDirecta.
func NuevoComprobanteConsumoReciboCargaDirecta(
	solicitud SolicitudConsumirReciboCargaDirecta,
	resultado ResultadoConsumoReciboCargaDirecta,
	atestacionHMAC string,
) (ComprobanteConsumoReciboCargaDirecta, error) {
	comprobante := ComprobanteConsumoReciboCargaDirecta{
		indiceReciboHMAC: resultado.IndiceHMAC, grupoReciboHMAC: resultado.GrupoHMAC,
		vinculoReciboHMAC: resultado.VinculoHMAC, evidenciaConsumoRef: resultado.EvidenciaConsumoRef,
		intencionRef: resultado.IntencionConfirmacionRef, huellaIntencionHMAC: resultado.HuellaIntencionHMAC,
		registradoEn: resultado.RegistradoEn, consumidoEn: resultado.ConsumidoEn, expiraEn: resultado.ExpiraEn,
		validaHasta: solicitud.ValidaHasta, atestacionHMAC: atestacionHMAC,
	}
	if solicitud.Validar() != nil || resultado.Validar() != nil || !almacencanonico.HMACSHA256Valido(atestacionHMAC) {
		return ComprobanteConsumoReciboCargaDirecta{}, ErrReciboCargaDirectaNoValido
	}
	return comprobante, nil
}

func (c ComprobanteConsumoReciboCargaDirecta) datosCanonicos() almacencanonico.DatosComprobanteConsumoReciboCargaDirecta {
	return almacencanonico.DatosComprobanteConsumoReciboCargaDirecta{
		IndiceReciboHMAC: c.indiceReciboHMAC, GrupoReciboHMAC: c.grupoReciboHMAC,
		VinculoReciboHMAC: c.vinculoReciboHMAC, EvidenciaConsumoRef: c.evidenciaConsumoRef,
		IntencionRef: c.intencionRef, HuellaIntencionHMAC: c.huellaIntencionHMAC,
		RegistradoEn: c.registradoEn, ConsumidoEn: c.consumidoEn, ExpiraEn: c.expiraEn,
		ValidaHasta: c.validaHasta, AtestacionHMAC: c.atestacionHMAC,
	}
}

func (c ComprobanteConsumoReciboCargaDirecta) validarEstructura() error {
	return almacencanonico.ValidarDatosComprobanteConsumoReciboCargaDirecta(c.datosCanonicos())
}

func (c ComprobanteConsumoReciboCargaDirecta) RevelarParaVerificacion() (
	indiceHMAC, grupoHMAC, vinculoHMAC, evidenciaConsumoRef, intencionRef, huellaIntencionHMAC string,
	registradoEn, consumidoEn, expiraEn, validaHasta time.Time,
	atestacionHMAC string,
	err error,
) {
	if err = c.validarEstructura(); err != nil {
		return "", "", "", "", "", "", time.Time{}, time.Time{}, time.Time{}, time.Time{}, "", err
	}
	return c.indiceReciboHMAC, c.grupoReciboHMAC, c.vinculoReciboHMAC,
		c.evidenciaConsumoRef, c.intencionRef, c.huellaIntencionHMAC,
		c.registradoEn, c.consumidoEn, c.expiraEn, c.validaHasta, c.atestacionHMAC, nil
}

func (ComprobanteConsumoReciboCargaDirecta) String() string {
	return "[COMPROBANTE-CONSUMO-RECIBO-CARGA-DIRECTA-ATESTADO]"
}

func (ComprobanteConsumoReciboCargaDirecta) GoString() string {
	return "[COMPROBANTE-CONSUMO-RECIBO-CARGA-DIRECTA-ATESTADO]"
}

func (c ComprobanteConsumoReciboCargaDirecta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}

func (ComprobanteConsumoReciboCargaDirecta) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

func (ComprobanteConsumoReciboCargaDirecta) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

func (c ComprobanteConsumoReciboCargaDirecta) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

// ConsumidorReciboCargaDirecta verifica MAC/caducidad y marca el recibo como
// usado en una unica operacion atomica duradera.
type ConsumidorReciboCargaDirecta interface {
	ConsumirReciboCargaDirecta(
		context.Context,
		SolicitudConsumirReciboCargaDirecta,
	) (ComprobanteConsumoReciboCargaDirecta, error)
}

// VerificadorAtestacionConsumoReciboCargaDirecta es una dependencia de
// seguridad obligatoria del nucleo. Una implementacion que solo compruebe la
// forma o devuelva nil no es apta para produccion: debe verificar con la clave
// HMAC exclusiva de atestacion el contexto completo, autorizacion, accion,
// sesion, evidencia y fecha durable.
type VerificadorAtestacionConsumoReciboCargaDirecta interface {
	VerificarAtestacionConsumoReciboCargaDirecta(
		context.Context,
		ContextoOperacionAlmacen,
		string,
		ComprobanteConsumoReciboCargaDirecta,
	) error
}

// RegistroReciboCargaDirecta es la unica informacion propuesta al alta
// durable. No contiene una fecha de emision del proceso: el repositorio elige
// RegistradoEn dentro de la transaccion y la devuelve en el resultado.
// IndiceHMAC deriva del material secreto estable anterior a esa fecha;
// VinculoHMAC autentica sus invariantes y la sesion. Ninguno puede sustituirse
// por el recibo o por la referencia de sesion.
type RegistroReciboCargaDirecta = almacencanonico.RegistroReciboCargaDirecta

// PredecesorReciboCargaDirecta conserva el enlace tipado que el repositorio
// crea al sustituir el recibo activo de un grupo. No es por si mismo un evento
// de auditoria: el adaptador durable debera incorporarlo a su registro
// transaccional y al outbox cuando estos existan.
type PredecesorReciboCargaDirecta = almacencanonico.PredecesorReciboCargaDirecta

// ResultadoRegistroReciboCargaDirecta acredita la fecha del alta durable y,
// cuando procede, la relacion con el predecesor del mismo grupo sustituido en
// la misma transaccion.
type ResultadoRegistroReciboCargaDirecta = almacencanonico.ResultadoRegistroReciboCargaDirecta

// OrdenConsumoReciboCargaDirecta no contiene secreto ni una hora de consumo
// propuesta por el proceso. RegistradoEn procede del recibo atestado tras el
// alta y debe coincidir con el registro; el repositorio decide ConsumidoEn con
// su reloj transaccional autoritativo y lo devuelve despues de persistirlo.
type OrdenConsumoReciboCargaDirecta = almacencanonico.OrdenConsumoReciboCargaDirecta

// ResultadoConsumoReciboCargaDirecta acredita la escritura condicional del
// repositorio. RegistradoEn, ConsumidoEn y ExpiraEn proceden del registro durable,
// no de datos propuestos por el proceso. Todos los identificadores deben
// coincidir exactamente con la orden.
type ResultadoConsumoReciboCargaDirecta = almacencanonico.ResultadoConsumoReciboCargaDirecta

// RepositorioRecibosCargaDirecta es un puerto saliente durable. Registrar
// conserva la unicidad permanente de IndiceHMAC, fija RegistradoEn con su reloj
// durable y, en la misma transaccion,
// sustituye el recibo activo anterior del GrupoHMAC. Debe conservar el enlace
// tipado al indice y autorizacion del predecesor; el anterior permanece
// inactivo y no se reactiva. Consumir exige indice, grupo, vinculo, intencion
// y huella exactos, que el recibo siga siendo el activo del grupo, que no se
// haya consumido y que la hora transaccional durable cumpla
// RegistradoEn <= ahora < min(ExpiraEn, ValidaHasta). En esa misma escritura
// desactiva el grupo, crea la intencion pendiente y persiste evidencia y fecha.
// El resultado devuelve RegistradoEn y ExpiraEn del registro, nunca fechas
// propuestas por el proceso.
// El puerto no ofrece lectura, listado, reapertura ni borrado de consumos.
//
// Una colision de alta devuelve ErrRegistroReciboCargaDirectaConflicto. Toda
// denegacion de consumo usa ErrConsumoReciboCargaDirectaDenegado. Los demas
// errores representan indisponibilidad y el adaptador falla cerrado.
type RepositorioRecibosCargaDirecta interface {
	RegistrarReciboCargaDirecta(context.Context, RegistroReciboCargaDirecta) (ResultadoRegistroReciboCargaDirecta, error)
	ConsumirReciboCargaDirecta(context.Context, OrdenConsumoReciboCargaDirecta) (ResultadoConsumoReciboCargaDirecta, error)
}

// SolicitudConfirmarCargaDirecta solo puede construirse mediante la fabrica
// verificada. Sus campos privados evitan que un conector reciba un comprobante
// estructuralmente valido pero no atestado.
type SolicitudConfirmarCargaDirecta struct {
	contexto             ContextoOperacionAlmacen
	sesionRef            string
	comprobante          ComprobanteConsumoReciboCargaDirecta
	atestacionVerificada bool
}

func (s SolicitudConfirmarCargaDirecta) Validar() error {
	if !s.atestacionVerificada || s.contexto.validarParaPaso(AccionAlmacenConfirmarCargaDirecta) != nil ||
		!referenciaOpacaAlmacenValida(s.sesionRef, 512) || s.comprobante.validarEstructura() != nil {
		return ErrSolicitudAlmacenInvalida
	}
	return nil
}

// RevelarParaConector entrega exclusivamente lo necesario al gestor de carga
// directa despues de que la fabrica haya verificado la atestacion.
func (s SolicitudConfirmarCargaDirecta) RevelarParaConector() (
	contexto ContextoOperacionAlmacen,
	sesionRef, intencionRef, huellaIntencionHMAC, evidenciaConsumoRef string,
	registradoEn, consumidoEn, expiraEn, validaHasta time.Time,
	err error,
) {
	if err = s.Validar(); err != nil {
		return ContextoOperacionAlmacen{}, "", "", "", "", time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	return s.contexto, s.sesionRef, s.comprobante.intencionRef, s.comprobante.huellaIntencionHMAC,
		s.comprobante.evidenciaConsumoRef, s.comprobante.registradoEn, s.comprobante.consumidoEn,
		s.comprobante.expiraEn, s.comprobante.validaHasta, nil
}

func (SolicitudConfirmarCargaDirecta) String() string {
	return "[SOLICITUD-CONFIRMAR-CARGA-DIRECTA-ATESTADA]"
}

func (SolicitudConfirmarCargaDirecta) GoString() string {
	return "[SOLICITUD-CONFIRMAR-CARGA-DIRECTA-ATESTADA]"
}

func (s SolicitudConfirmarCargaDirecta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}

func (SolicitudConfirmarCargaDirecta) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

func (SolicitudConfirmarCargaDirecta) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

func (s SolicitudConfirmarCargaDirecta) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

func NuevaSolicitudConfirmarCargaDirecta(
	ctx context.Context,
	contexto ContextoOperacionAlmacen,
	sesionRef string,
	comprobante ComprobanteConsumoReciboCargaDirecta,
	verificador VerificadorAtestacionConsumoReciboCargaDirecta,
) (SolicitudConfirmarCargaDirecta, error) {
	if ctx == nil || ctx.Err() != nil || dependenciaPuertoAlmacenNula(verificador) ||
		contexto.validarParaPaso(AccionAlmacenConfirmarCargaDirecta) != nil ||
		!referenciaOpacaAlmacenValida(sesionRef, 512) || comprobante.validarEstructura() != nil {
		return SolicitudConfirmarCargaDirecta{}, ErrSolicitudAlmacenInvalida
	}
	if err := verificador.VerificarAtestacionConsumoReciboCargaDirecta(
		ctx, contexto, sesionRef, comprobante,
	); err != nil || ctx.Err() != nil {
		return SolicitudConfirmarCargaDirecta{}, ErrAtestacionReciboCargaDirectaNoValida
	}
	solicitud := SolicitudConfirmarCargaDirecta{
		contexto: contexto, sesionRef: sesionRef, comprobante: comprobante, atestacionVerificada: true,
	}
	if err := solicitud.Validar(); err != nil {
		return SolicitudConfirmarCargaDirecta{}, err
	}
	return solicitud, nil
}

func dependenciaPuertoAlmacenNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

// GestorCargaDirecta es una capacidad opcional. ConfirmarCargaDirecta debe
// tratar IntencionConfirmacionRef como clave idempotente y
// HuellaIntencionHMAC como identidad exacta de la peticion: una repeticion con
// la misma pareja devuelve el mismo efecto; la misma referencia con otra
// huella falla cerrada. Tambien debe comprobar el limite temporal en el punto
// del efecto. Este contrato no aporta por si solo atomicidad distribuida.
type GestorCargaDirecta interface {
	PrepararCargaDirecta(context.Context, SolicitudPrepararCargaDirecta) (InstruccionesCargaDirecta, error)
	ConfirmarCargaDirecta(context.Context, SolicitudConfirmarCargaDirecta) (ResultadoOperacionObjeto, error)
	AbandonarCargaDirecta(context.Context, ContextoOperacionAlmacen, string) error
}

type SolicitudSellarIdempotenciaCarga = almacencanonico.SolicitudSellarIdempotenciaCarga

// SelladorIdempotenciaCarga liga una clave de reintento a la operacion y a
// todos sus datos exactos mediante una clave exclusiva del servidor. El
// navegador nunca aporta directamente la clave aceptada por el almacen.
type SelladorIdempotenciaCarga interface {
	SellarIdempotenciaCarga(context.Context, SolicitudSellarIdempotenciaCarga) (string, error)
}

func origenesCargaDirectaValidos(origenes []string) bool {
	return almacencanonico.OrigenesCargaDirectaValidos(origenes)
}

func textoSeguroAlmacen(valor string, maximo int) bool {
	return almacencanonico.TextoSeguro(valor, maximo)
}

func referenciaOpacaAlmacenValida(valor string, maximo int) bool {
	return almacencanonico.ReferenciaOpacaValida(valor, maximo)
}

func esSHA256Hexadecimal(valor string) bool {
	return almacencanonico.SHA256HexadecimalValido(valor)
}

func datosCanonicosEvidenciaAlmacen(e EvidenciaOperacionAlmacen) almacencanonico.DatosEvidenciaOperacionAlmacen {
	return almacencanonico.DatosEvidenciaOperacionAlmacen{
		Referencia: e.Referencia, ConectorID: e.ConectorID, EsquemaContexto: e.EsquemaContexto,
		EsquemaEsperado: EsquemaContextoOperacionAlmacenV1, AccionNegocio: e.AccionNegocio,
		Accion: e.Accion, EfectoRef: e.EfectoRef, HuellaPlanEfectoSHA256: e.HuellaPlanEfectoSHA256,
		HuellaManifiestoSHA256: e.HuellaManifiestoSHA256, HuellaPasoSHA256: e.HuellaPasoSHA256,
		PasoRef: string(e.PasoRef), HuellaDecisionSHA256: e.HuellaDecisionSHA256, Objeto: e.Objeto,
		OperacionRef: e.OperacionRef, CorrelacionRef: e.CorrelacionRef, AutorizacionRef: e.AutorizacionRef,
		Finalidad: e.Finalidad, Clasificacion: e.Clasificacion, RealizadaEn: e.RealizadaEn,
		CargaRef: e.CargaRef, SujetoSeudonimoHMAC: e.SujetoSeudonimoHMAC, RecursoRef: e.RecursoRef,
		ModuloID: e.ModuloID, HuellaSolicitudHMAC: e.HuellaSolicitudHMAC, FundamentoRef: e.FundamentoRef,
		ReintentoIdempotente: e.ReintentoIdempotente,
	}
}

func datosCanonicosResultadoAlmacen(r ResultadoOperacionObjeto) almacencanonico.DatosResultadoOperacionObjeto {
	return almacencanonico.DatosResultadoOperacionObjeto{
		Objeto: r.Objeto, Evidencia: datosCanonicosEvidenciaAlmacen(r.Evidencia),
	}
}

func evidenciaAlmacenLigada(evidencia EvidenciaOperacionAlmacen, contexto ContextoOperacionAlmacen) bool {
	proyeccion, err := contexto.Proyeccion()
	return err == nil && almacencanonico.LigaduraExacta(
		[]string{
			evidencia.EsquemaContexto, evidencia.OperacionRef, evidencia.CorrelacionRef,
			evidencia.AutorizacionRef, evidencia.Finalidad, evidencia.Clasificacion,
			evidencia.AccionNegocio, evidencia.Accion, evidencia.EfectoRef,
			evidencia.HuellaPlanEfectoSHA256, evidencia.HuellaManifiestoSHA256,
			evidencia.HuellaPasoSHA256, string(evidencia.PasoRef), evidencia.HuellaDecisionSHA256,
			evidencia.CargaRef, evidencia.SujetoSeudonimoHMAC, evidencia.RecursoRef,
			evidencia.ModuloID, evidencia.HuellaSolicitudHMAC,
		},
		[]string{
			proyeccion.Esquema, proyeccion.OperacionRef, proyeccion.CorrelacionRef,
			proyeccion.AutorizacionRef, proyeccion.Finalidad, proyeccion.Clasificacion,
			proyeccion.AccionNegocio, proyeccion.AccionTecnica, proyeccion.EfectoRef,
			proyeccion.HuellaPlanEfectoSHA256, proyeccion.HuellaManifiestoSHA256,
			proyeccion.HuellaPasoSHA256, string(proyeccion.PasoRef), proyeccion.HuellaDecisionSHA256,
			proyeccion.CargaRef, proyeccion.SujetoSeudonimoHMAC, proyeccion.RecursoRef,
			proyeccion.ModuloID, proyeccion.HuellaSolicitudHMAC,
		},
	)
}

func contextosMismaOperacion(primero, segundo ContextoOperacionAlmacen) bool {
	p, errPrimero := primero.Proyeccion()
	s, errSegundo := segundo.Proyeccion()
	return errPrimero == nil && errSegundo == nil && almacencanonico.LigaduraExacta(
		[]string{
			p.OperacionRef, p.CorrelacionRef, p.Finalidad, p.Clasificacion, p.CargaRef,
			p.SujetoSeudonimoHMAC, p.RecursoRef, p.ModuloID, p.HuellaSolicitudHMAC,
		},
		[]string{
			s.OperacionRef, s.CorrelacionRef, s.Finalidad, s.Clasificacion, s.CargaRef,
			s.SujetoSeudonimoHMAC, s.RecursoRef, s.ModuloID, s.HuellaSolicitudHMAC,
		},
	)
}

func huellaSolicitudCargaDirecta(solicitud SolicitudPrepararCargaDirecta) string {
	contexto, err := solicitud.Contexto.Proyeccion()
	if err != nil {
		return ""
	}
	return almacencanonico.HuellaPreparacionCargaDirecta(almacencanonico.DatosHuellaPreparacionCargaDirecta{
		Esquema: contexto.Esquema, OperacionRef: contexto.OperacionRef,
		CorrelacionRef: contexto.CorrelacionRef, AutorizacionRef: contexto.AutorizacionRef,
		Finalidad: contexto.Finalidad, Clasificacion: contexto.Clasificacion,
		AccionNegocio: contexto.AccionNegocio, AccionTecnica: contexto.AccionTecnica,
		CargaRef: contexto.CargaRef, SujetoSeudonimoHMAC: contexto.SujetoSeudonimoHMAC,
		RecursoRef: contexto.RecursoRef, ModuloID: contexto.ModuloID,
		HuellaSolicitudHMAC: contexto.HuellaSolicitudHMAC, EfectoRef: contexto.EfectoRef,
		HuellaPlanEfectoSHA256: contexto.HuellaPlanEfectoSHA256, PasoRef: string(contexto.PasoRef),
		HuellaDecisionSHA256: contexto.HuellaDecisionSHA256,
		ClaveIdempotencia:    solicitud.ClaveIdempotencia, MIME: solicitud.MIME,
		Tamano: solicitud.Tamano, HuellaSHA256: solicitud.HuellaSHA256, ExpiraEn: solicitud.ExpiraEn,
	})
}

func accionOperacionAlmacenValida(accion string) bool {
	return almacencanonico.AccionOperacionValida(accion)
}
