package pagos

import (
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const (
	maximoCaracteresAuditoriaCobro = 512
	audienciaPuertoCobro           = "vec.cobros"
)

// ErrMutacionOrdenCobroInvalida es el error contractual compartido por los
// registros canonicos y su reexportacion desde los puertos.
var ErrMutacionOrdenCobroInvalida = errors.New("vec: mutacion de orden de cobro invalida")

// RegistroAuditoriaCobro es la proyeccion inmutable del ultimo hecho de cobro.
type RegistroAuditoriaCobro struct {
	ID                          string
	ActorRef                    string
	PerfilActivoRef             string
	DecisionAutorizacionRef     string
	HuellaDecisionSHA256        string
	AutorizacionEmitidaEn       time.Time
	AutorizacionValidaHasta     time.Time
	AutorizacionEvaluadaEn      time.Time
	AtestacionAutenticacionRef  string
	AtestacionEmitidaEn         time.Time
	AtestacionValidaHasta       time.Time
	AutenticacionVerificadaEn   time.Time
	SesionRef                   string
	HuellaSesionHMAC            string
	MetodoAutenticacion         domain.AuthMethod
	GarantiaAutenticacion       domain.AuthAssurance
	Accion                      domain.AccionCobro
	Hecho                       domain.TipoHechoCobro
	OrdenRef                    string
	ExpedienteRef               string
	VersionAnterior             int
	VersionPosterior            int
	HuellaAnteriorSHA256        string
	HuellaPosteriorSHA256       string
	EvidenciaRef                string
	HuellaEvidenciaSHA256       string
	VerificacionEvidenciaRef    string
	HuellaVerificacionSHA256    string
	MetodoVerificacionEvidencia domain.MetodoAutenticacionEvidenciaCobro
	AudienciaEvidencia          string
	EvidenciaEmitidaEn          time.Time
	EvidenciaRecibidaEn         time.Time
	EvidenciaVerificadaEn       time.Time
	Resultado                   string
	Motivo                      string
	CorrelacionRef              string
	OcurridoEn                  time.Time
	Metadatos                   MetadatosAuditoriaCobro
}

// CanalAuditoriaCobro es informativo y nunca concede acceso.
type CanalAuditoriaCobro string

const (
	CanalAuditoriaCobroInterno           CanalAuditoriaCobro = "interno"
	CanalAuditoriaCobroPasarela          CanalAuditoriaCobro = "pasarela"
	CanalAuditoriaCobroProcesoAutomatico CanalAuditoriaCobro = "proceso_automatico"
)

// MetadatosAuditoriaCobro mantiene cerrado el unico metadato informativo.
type MetadatosAuditoriaCobro struct {
	Canal CanalAuditoriaCobro
}

func (m MetadatosAuditoriaCobro) validar() bool {
	switch m.Canal {
	case CanalAuditoriaCobroInterno, CanalAuditoriaCobroPasarela, CanalAuditoriaCobroProcesoAutomatico:
		return true
	default:
		return false
	}
}

// Validar comprueba la integridad completa y el identificador derivado.
func (r RegistroAuditoriaCobro) Validar() error {
	canalEsperado, existeCanal := CanalAuditoriaParaHecho(r.Hecho, r.Accion)
	if !ReferenciaOpacaValida(r.ActorRef, "per_") || !ReferenciaOpacaValida(r.PerfilActivoRef, "prf_") ||
		!TextoValido(r.DecisionAutorizacionRef, 512) || !HuellaSHA256Valida(r.HuellaDecisionSHA256) ||
		r.AutorizacionEmitidaEn.IsZero() || !r.AutorizacionValidaHasta.After(r.AutorizacionEmitidaEn) ||
		r.AutorizacionEvaluadaEn.Before(r.AutorizacionEmitidaEn) ||
		!r.AutorizacionEvaluadaEn.Before(r.AutorizacionValidaHasta) ||
		!ReferenciaOpacaValida(r.AtestacionAutenticacionRef, "aut_") ||
		r.AtestacionEmitidaEn.IsZero() || !r.AtestacionValidaHasta.After(r.AtestacionEmitidaEn) ||
		r.AutenticacionVerificadaEn.Before(r.AtestacionEmitidaEn) ||
		!r.AutenticacionVerificadaEn.Before(r.AtestacionValidaHasta) ||
		!ReferenciaOpacaValida(r.SesionRef, "ses_") || !HuellaHMACDeDominioValida(r.HuellaSesionHMAC, "sesion-v1") ||
		!MetodoAutenticacionPermitido(r.MetodoAutenticacion) ||
		!GarantiaAutenticacionPermitida(r.GarantiaAutenticacion) ||
		!AccionAuditoriaPermitida(r.Accion) || !r.Hecho.Valido() ||
		!domain.TuplaHechoCobroValida(r.Hecho, domain.EstadoCobro(r.Resultado), r.Accion) ||
		!ReferenciaOpacaValida(r.OrdenRef, "cob_") || !TextoValido(r.ExpedienteRef, 512) ||
		r.VersionAnterior < 0 || r.VersionPosterior != r.VersionAnterior+1 ||
		!HuellaSHA256Valida(r.HuellaAnteriorSHA256) || !HuellaSHA256Valida(r.HuellaPosteriorSHA256) ||
		!TextoValido(r.EvidenciaRef, 512) || !HuellaSHA256Valida(r.HuellaEvidenciaSHA256) ||
		!TextoValido(r.Resultado, 128) || !TextoValido(r.Motivo, maximoCaracteresAuditoriaCobro) ||
		!TextoValido(r.CorrelacionRef, 512) || r.OcurridoEn.IsZero() ||
		!r.Metadatos.validar() || !existeCanal || r.Metadatos.Canal != canalEsperado ||
		!MatrizEvidenciaAuditoriaValida(r) ||
		r.ID != IDAuditoria(r.OrdenRef, r.VersionPosterior, r.HuellaPosteriorSHA256, r.Hecho, r.Accion) {
		return ErrMutacionOrdenCobroInvalida
	}
	return nil
}

// CanalAuditoriaParaHecho deriva el canal sin aceptar metadatos del llamador.
func CanalAuditoriaParaHecho(hecho domain.TipoHechoCobro, accion domain.AccionCobro) (CanalAuditoriaCobro, bool) {
	switch hecho {
	case domain.HechoCobroOrdenCreada, domain.HechoCobroDevolucionSolicitada, domain.HechoCobroCancelado:
		return CanalAuditoriaCobroInterno, true
	case domain.HechoCobroCaducado:
		return CanalAuditoriaCobroProcesoAutomatico, true
	case domain.HechoCobroOperacionEnviada, domain.HechoCobroResultadoPendiente,
		domain.HechoCobroResultadoDesconocido, domain.HechoCobroConfirmado, domain.HechoCobroRechazado,
		domain.HechoCobroConciliado, domain.HechoCobroDevolucionResultadoPendiente,
		domain.HechoCobroDevolucionResultadoDesconocido, domain.HechoCobroDevolucionRechazada,
		domain.HechoCobroDevuelto, domain.HechoCobroDevolucionConciliada:
		return CanalAuditoriaCobroPasarela, true
	case domain.HechoCobroIncidenciaDetectada, domain.HechoCobroEvidenciaAdicional:
		switch accion {
		case domain.AccionCobroIniciarOperacion, domain.AccionCobroProcesarResultado,
			domain.AccionCobroProcesarDevolucion, domain.AccionCobroConciliar:
			return CanalAuditoriaCobroPasarela, true
		case domain.AccionCobroSolicitarDevolucion, domain.AccionCobroCancelar:
			return CanalAuditoriaCobroInterno, true
		case domain.AccionCobroCaducar:
			return CanalAuditoriaCobroProcesoAutomatico, true
		}
	}
	return "", false
}

// MatrizEvidenciaAuditoriaValida liga procedencia, hecho y prueba remota.
func MatrizEvidenciaAuditoriaValida(r RegistroAuditoriaCobro) bool {
	remota := r.VerificacionEvidenciaRef != "" || r.HuellaVerificacionSHA256 != "" ||
		r.MetodoVerificacionEvidencia != "" || r.AudienciaEvidencia != "" ||
		!r.EvidenciaEmitidaEn.IsZero() || !r.EvidenciaRecibidaEn.IsZero() || !r.EvidenciaVerificadaEn.IsZero()
	esperadaRemota := false
	switch r.Hecho {
	case domain.HechoCobroOperacionEnviada, domain.HechoCobroResultadoPendiente,
		domain.HechoCobroResultadoDesconocido, domain.HechoCobroConfirmado, domain.HechoCobroRechazado,
		domain.HechoCobroConciliado, domain.HechoCobroDevolucionResultadoPendiente,
		domain.HechoCobroDevolucionResultadoDesconocido, domain.HechoCobroDevolucionRechazada,
		domain.HechoCobroDevuelto, domain.HechoCobroDevolucionConciliada,
		domain.HechoCobroEvidenciaAdicional:
		esperadaRemota = true
	case domain.HechoCobroIncidenciaDetectada:
		switch r.Accion {
		case domain.AccionCobroIniciarOperacion, domain.AccionCobroProcesarResultado,
			domain.AccionCobroProcesarDevolucion, domain.AccionCobroConciliar:
			esperadaRemota = true
		case domain.AccionCobroSolicitarDevolucion, domain.AccionCobroCancelar, domain.AccionCobroCaducar:
			esperadaRemota = false
		default:
			return false
		}
	}
	if remota != esperadaRemota {
		return false
	}
	if !remota {
		return true
	}
	return TextoValido(r.VerificacionEvidenciaRef, 512) &&
		HuellaSHA256Valida(r.HuellaVerificacionSHA256) &&
		r.MetodoVerificacionEvidencia.Valido() && r.AudienciaEvidencia == audienciaPuertoCobro &&
		!r.EvidenciaEmitidaEn.IsZero() && !r.EvidenciaRecibidaEn.Before(r.EvidenciaEmitidaEn) &&
		!r.EvidenciaVerificadaEn.Before(r.EvidenciaRecibidaEn) &&
		r.EvidenciaVerificadaEn.Sub(r.EvidenciaRecibidaEn) <= 2*time.Minute
}

// TipoEventoSalidaCobro es la lista cerrada de tipos del outbox.
type TipoEventoSalidaCobro string

const (
	EventoCobroOrdenCreada                    TipoEventoSalidaCobro = "cobro.orden.creada"
	EventoCobroOperacionEnviada               TipoEventoSalidaCobro = "cobro.operacion.enviada"
	EventoCobroResultadoPendiente             TipoEventoSalidaCobro = "cobro.resultado.pendiente"
	EventoCobroResultadoDesconocido           TipoEventoSalidaCobro = "cobro.resultado.desconocido"
	EventoCobroConfirmado                     TipoEventoSalidaCobro = "cobro.confirmado"
	EventoCobroRechazado                      TipoEventoSalidaCobro = "cobro.rechazado"
	EventoCobroCancelado                      TipoEventoSalidaCobro = "cobro.cancelado"
	EventoCobroCaducado                       TipoEventoSalidaCobro = "cobro.caducado"
	EventoCobroConciliado                     TipoEventoSalidaCobro = "cobro.conciliado"
	EventoCobroDevolucionSolicitada           TipoEventoSalidaCobro = "cobro.devolucion.solicitada"
	EventoCobroDevolucionResultadoPendiente   TipoEventoSalidaCobro = "cobro.devolucion.resultado_pendiente"
	EventoCobroDevolucionResultadoDesconocido TipoEventoSalidaCobro = "cobro.devolucion.resultado_desconocido"
	EventoCobroDevolucionRechazada            TipoEventoSalidaCobro = "cobro.devolucion.rechazada"
	EventoCobroDevuelto                       TipoEventoSalidaCobro = "cobro.devuelto"
	EventoCobroDevolucionConciliada           TipoEventoSalidaCobro = "cobro.devolucion.conciliada"
	EventoCobroIncidenciaDetectada            TipoEventoSalidaCobro = "cobro.incidencia.detectada"
	EventoCobroEvidenciaAdicional             TipoEventoSalidaCobro = "cobro.evidencia.adicional"
)

// Valido comprueba la lista cerrada del outbox.
func (t TipoEventoSalidaCobro) Valido() bool {
	switch t {
	case EventoCobroOrdenCreada, EventoCobroOperacionEnviada, EventoCobroResultadoPendiente,
		EventoCobroResultadoDesconocido, EventoCobroConfirmado, EventoCobroRechazado,
		EventoCobroCancelado, EventoCobroCaducado, EventoCobroConciliado,
		EventoCobroDevolucionSolicitada, EventoCobroDevolucionResultadoPendiente,
		EventoCobroDevolucionResultadoDesconocido, EventoCobroDevolucionRechazada,
		EventoCobroDevuelto, EventoCobroDevolucionConciliada, EventoCobroIncidenciaDetectada,
		EventoCobroEvidenciaAdicional:
		return true
	default:
		return false
	}
}

// AtributosEventoSalidaCobro sustituye el mapa abierto del outbox.
type AtributosEventoSalidaCobro struct {
	Hecho  domain.TipoHechoCobro
	Estado domain.EstadoCobro
	Accion domain.AccionCobro
}

func (a AtributosEventoSalidaCobro) valido() bool {
	return a.Hecho.Valido() && a.Estado.Valido() && a.Accion.Valida()
}

// EventoSalidaCobro es el evento derivado del ultimo hecho confirmado.
type EventoSalidaCobro struct {
	ID                string
	Tipo              TipoEventoSalidaCobro
	OrdenRef          string
	VersionOrden      int
	SecuenciaHecho    int64
	HuellaHechoSHA256 string
	HuellaOrdenSHA256 string
	CorrelacionRef    string
	OcurridoEn        time.Time
	Atributos         AtributosEventoSalidaCobro
}

// Validar comprueba la tupla cerrada y el identificador derivado.
func (e EventoSalidaCobro) Validar() error {
	tipoEsperado, existe := TipoEventoParaHecho(e.Atributos.Hecho)
	if !e.Tipo.Valido() || !existe || e.Tipo != tipoEsperado || !ReferenciaOpacaValida(e.OrdenRef, "cob_") ||
		e.VersionOrden < 1 || e.SecuenciaHecho < 1 || int64(e.VersionOrden) != e.SecuenciaHecho ||
		!HuellaSHA256Valida(e.HuellaHechoSHA256) || e.HuellaHechoSHA256 != e.HuellaOrdenSHA256 ||
		!TextoValido(e.CorrelacionRef, 512) || e.OcurridoEn.IsZero() || !e.Atributos.valido() ||
		!domain.TuplaHechoCobroValida(e.Atributos.Hecho, e.Atributos.Estado, e.Atributos.Accion) ||
		e.ID != IDEvento(e.OrdenRef, e.VersionOrden, e.SecuenciaHecho, e.HuellaHechoSHA256,
			e.Atributos.Hecho, e.Atributos.Estado, e.Atributos.Accion) {
		return ErrMutacionOrdenCobroInvalida
	}
	return nil
}

// NuevoEventoSalidaCobro deriva el evento completo sin campos semanticos libres.
func NuevoEventoSalidaCobro(orden domain.OrdenCobro) (EventoSalidaCobro, error) {
	if orden.Validar() != nil || len(orden.Historial) == 0 {
		return EventoSalidaCobro{}, ErrMutacionOrdenCobroInvalida
	}
	version, huella, err := orden.ControlConcurrencia()
	if err != nil {
		return EventoSalidaCobro{}, ErrMutacionOrdenCobroInvalida
	}
	hecho := orden.Historial[len(orden.Historial)-1]
	tipo, existe := TipoEventoParaHecho(hecho.Tipo)
	if !existe {
		return EventoSalidaCobro{}, ErrMutacionOrdenCobroInvalida
	}
	evento := EventoSalidaCobro{
		Tipo: tipo, OrdenRef: orden.ID, VersionOrden: version, SecuenciaHecho: hecho.Secuencia,
		HuellaHechoSHA256: hecho.HuellaEstadoPosteriorSHA256,
		HuellaOrdenSHA256: huella, CorrelacionRef: orden.CorrelacionRef, OcurridoEn: hecho.OcurridoEn,
		Atributos: AtributosEventoSalidaCobro{
			Hecho: hecho.Tipo, Estado: hecho.EstadoPosterior, Accion: hecho.AccionAutorizada,
		},
	}
	evento.ID = IDEvento(evento.OrdenRef, evento.VersionOrden, evento.SecuenciaHecho,
		evento.HuellaHechoSHA256, evento.Atributos.Hecho, evento.Atributos.Estado, evento.Atributos.Accion)
	if evento.Validar() != nil {
		return EventoSalidaCobro{}, ErrMutacionOrdenCobroInvalida
	}
	return evento, nil
}

// TipoEventoParaHecho mantiene el mapeo uno a uno entre hecho y outbox.
func TipoEventoParaHecho(hecho domain.TipoHechoCobro) (TipoEventoSalidaCobro, bool) {
	switch hecho {
	case domain.HechoCobroOrdenCreada:
		return EventoCobroOrdenCreada, true
	case domain.HechoCobroOperacionEnviada:
		return EventoCobroOperacionEnviada, true
	case domain.HechoCobroResultadoPendiente:
		return EventoCobroResultadoPendiente, true
	case domain.HechoCobroResultadoDesconocido:
		return EventoCobroResultadoDesconocido, true
	case domain.HechoCobroConfirmado:
		return EventoCobroConfirmado, true
	case domain.HechoCobroRechazado:
		return EventoCobroRechazado, true
	case domain.HechoCobroCancelado:
		return EventoCobroCancelado, true
	case domain.HechoCobroCaducado:
		return EventoCobroCaducado, true
	case domain.HechoCobroConciliado:
		return EventoCobroConciliado, true
	case domain.HechoCobroDevolucionSolicitada:
		return EventoCobroDevolucionSolicitada, true
	case domain.HechoCobroDevolucionResultadoPendiente:
		return EventoCobroDevolucionResultadoPendiente, true
	case domain.HechoCobroDevolucionResultadoDesconocido:
		return EventoCobroDevolucionResultadoDesconocido, true
	case domain.HechoCobroDevolucionRechazada:
		return EventoCobroDevolucionRechazada, true
	case domain.HechoCobroDevuelto:
		return EventoCobroDevuelto, true
	case domain.HechoCobroDevolucionConciliada:
		return EventoCobroDevolucionConciliada, true
	case domain.HechoCobroIncidenciaDetectada:
		return EventoCobroIncidenciaDetectada, true
	case domain.HechoCobroEvidenciaAdicional:
		return EventoCobroEvidenciaAdicional, true
	default:
		return "", false
	}
}
