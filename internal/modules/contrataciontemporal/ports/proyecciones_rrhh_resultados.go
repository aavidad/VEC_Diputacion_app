package ports

import (
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const versionMaximaJSONSegura uint64 = 9_007_199_254_740_991

// ResumenExpedienteRRHH es la proyección mínima serializable del cuadro. No
// contiene contacto, actor, responsable personal, observaciones ni etiquetas.
type ResumenExpedienteRRHH struct {
	ExpedienteRef   string                 `json:"expediente_ref"`
	OrganizacionRef string                 `json:"organizacion_ref"`
	NumeroVisible   string                 `json:"numero_visible"`
	Version         uint64                 `json:"version"`
	FlujoRef        string                 `json:"flujo_ref"`
	FlujoVersion    uint64                 `json:"flujo_version"`
	FlujoHuella     string                 `json:"flujo_huella_sha256"`
	FaseClave       domain.ClaveFase       `json:"fase_clave"`
	EstadoClave     domain.EstadoOperativo `json:"estado_clave"`
	CentroRef       string                 `json:"centro_ref"`
	CategoriaRef    string                 `json:"categoria_ref"`
	ModalidadClave  domain.ClaveCatalogo   `json:"modalidad_clave,omitempty"`
	UnidadRef       string                 `json:"unidad_ref,omitempty"`
	CreadoEn        time.Time              `json:"creado_en"`
	ActualizadoEn   time.Time              `json:"actualizado_en"`
}

func (r ResumenExpedienteRRHH) Validar() error {
	if !domain.ReferenciaOpacaValida(r.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(r.OrganizacionRef) ||
		!domain.NumeroExpedienteValido(r.NumeroVisible) ||
		r.Version < 1 || r.Version > versionMaximaJSONSegura ||
		!domain.ReferenciaOpacaValida(r.FlujoRef) ||
		r.FlujoVersion < 1 || r.FlujoVersion > versionMaximaJSONSegura ||
		!patronHuellaRRHH.MatchString(r.FlujoHuella) ||
		r.FlujoHuella == strings.Repeat("0", 64) ||
		!r.FaseClave.Valida() || !r.EstadoClave.Valido() ||
		!domain.ReferenciaOpacaValida(r.CentroRef) ||
		!domain.ReferenciaOpacaValida(r.CategoriaRef) ||
		(r.ModalidadClave != "" && !r.ModalidadClave.Valida()) ||
		(r.UnidadRef != "" && !domain.ReferenciaOpacaValida(r.UnidadRef)) ||
		!domain.InstanteUTCCanonico(r.CreadoEn) ||
		!domain.InstanteUTCCanonico(r.ActualizadoEn) ||
		r.ActualizadoEn.Before(r.CreadoEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (r ResumenExpedienteRRHH) cumpleAmbito(
	capacidad CapacidadConsultaRRHH,
) bool {
	if r.OrganizacionRef != capacidad.organizacionRef {
		return false
	}
	switch capacidad.claseAmbito {
	case AmbitoOrganizacionRRHH:
		return capacidad.ambitoRef == r.OrganizacionRef
	case AmbitoCentroRRHH:
		return capacidad.ambitoRef == r.CentroRef
	case AmbitoUnidadGestionRRHH:
		return capacidad.ambitoRef == r.UnidadRef
	default:
		return false
	}
}

// ReciboLecturaRRHH es evidencia interna sellada. Sus identificadores de
// auditoría, decisión, sesión y ámbito nunca se serializan ni se representan.
type ReciboLecturaRRHH struct {
	bloqueoSerializacionConsultaRRHH
	versionRecibo           uint8
	lecturaRef              string
	auditoriaRef            string
	decisionRef             string
	decisionHuella          string
	capacidadHuella         string
	materialHuella          string
	consultaHuella          string
	correlacionRef          string
	sesionRef               string
	organizacionRef         string
	claseAmbito             ClaseAmbitoConsultaRRHH
	ambitoRef               string
	accion                  string
	finalidad               string
	expedienteRef           string
	version                 uint64
	totalPublicado          uint16
	registradaEn            time.Time
	autenticacionRefV2      string
	autenticacionHuellaV2   string
	controlSesionRefV2      string
	controlSesionRevisionV2 uint64
	controlSesionHuellaV2   string
	actorRefV2              string
	perfilRefV2             string
	perfilVersionV2         uint64
	registroV2              ResultadoRegistradorAccesoRRHHV2
	evidenciaV2             EvidenciaConsumoResultadoRRHHV2
	selloCanonV2            [32]byte
}

func NuevoReciboLecturaRRHH(
	lecturaRef, auditoriaRef string,
	contexto ContextoConsultaRRHH,
	capacidad CapacidadConsultaRRHH,
	expedienteRef string,
	version uint64,
	totalPublicado uint16,
	registradaEn time.Time,
) (ReciboLecturaRRHH, error) {
	recibo := ReciboLecturaRRHH{
		versionRecibo: 1,
		lecturaRef:    lecturaRef, auditoriaRef: auditoriaRef,
		decisionRef: capacidad.decisionRef, correlacionRef: capacidad.correlacionRef,
		decisionHuella: capacidad.decisionHuella, capacidadHuella: capacidad.capacidadHuella,
		materialHuella: capacidad.materialHuella,
		consultaHuella: capacidad.consultaHuella,
		sesionRef:      contexto.sesionRef, organizacionRef: contexto.organizacionRef,
		claseAmbito: capacidad.claseAmbito, ambitoRef: capacidad.ambitoRef,
		accion: capacidad.accion, finalidad: capacidad.finalidad,
		expedienteRef: expedienteRef, version: version,
		totalPublicado: totalPublicado, registradaEn: registradaEn,
	}
	if capacidad.validaPara(
		contexto, capacidad.consultaDominio, capacidad.consultaHuella,
		capacidad.accion, capacidad.finalidad, expedienteRef, registradaEn,
	) != nil || recibo.validar() != nil {
		return ReciboLecturaRRHH{}, ErrResultadoConsultaRRHHNoConfiable
	}
	return recibo, nil
}

func (r ReciboLecturaRRHH) validar() error {
	if r.versionRecibo == 2 {
		return r.validarV2()
	}
	if r.versionRecibo != 1 {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return r.validarCamposComunes()
}

func (r ReciboLecturaRRHH) validarCamposComunes() error {
	esCuadro := r.accion == AccionConsultarCuadroRRHH &&
		r.finalidad == FinalidadConsultarCuadroRRHH &&
		r.expedienteRef == "" && r.version == 0
	esDetalle := r.accion == AccionConsultarDetalleRRHH &&
		r.finalidad == FinalidadConsultarDetalleRRHH &&
		domain.ReferenciaOpacaValida(r.expedienteRef) &&
		r.version >= 1 && r.version <= versionMaximaJSONSegura &&
		r.totalPublicado == 1
	if !domain.ReferenciaOpacaValida(r.lecturaRef) ||
		!domain.ReferenciaOpacaValida(r.auditoriaRef) ||
		!domain.ReferenciaOpacaValida(r.decisionRef) ||
		!patronHuellaRRHH.MatchString(r.decisionHuella) ||
		!patronHuellaRRHH.MatchString(r.capacidadHuella) ||
		!patronHuellaRRHH.MatchString(r.materialHuella) ||
		!patronHuellaRRHH.MatchString(r.consultaHuella) ||
		!domain.ReferenciaOpacaValida(r.correlacionRef) ||
		!domain.ReferenciaOpacaValida(r.sesionRef) ||
		!domain.ReferenciaOpacaValida(r.organizacionRef) ||
		!r.claseAmbito.valida() ||
		!domain.ReferenciaOpacaValida(r.ambitoRef) ||
		(!esCuadro && !esDetalle) ||
		r.totalPublicado > LimiteMaximoCuadroRRHH ||
		!domain.InstanteUTCCanonico(r.registradaEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (r ReciboLecturaRRHH) coincideCon(
	contexto ContextoConsultaRRHH,
	capacidad CapacidadConsultaRRHH,
	expedienteRef string,
	version uint64,
) bool {
	return r.validar() == nil &&
		capacidad.validaPara(
			contexto, capacidad.consultaDominio, r.consultaHuella,
			r.accion, r.finalidad, expedienteRef, r.registradaEn,
		) == nil &&
		r.decisionRef == capacidad.decisionRef &&
		r.decisionHuella == capacidad.decisionHuella &&
		r.capacidadHuella == capacidad.capacidadHuella &&
		r.materialHuella == capacidad.materialHuella &&
		r.consultaHuella == capacidad.consultaHuella &&
		r.correlacionRef == capacidad.correlacionRef &&
		r.sesionRef == contexto.sesionRef &&
		r.organizacionRef == contexto.organizacionRef &&
		r.claseAmbito == capacidad.claseAmbito &&
		r.ambitoRef == capacidad.ambitoRef &&
		r.accion == capacidad.accion &&
		r.finalidad == capacidad.finalidad &&
		r.expedienteRef == expedienteRef &&
		r.version == version &&
		(r.versionRecibo != 2 || r.coincideConContextoV2(contexto))
}

func (r ReciboLecturaRRHH) ExpedienteRef() string { return r.expedienteRef }
func (r ReciboLecturaRRHH) Version() uint64       { return r.version }
func (r ReciboLecturaRRHH) DecisionHuellaSHA256() string {
	return r.decisionHuella
}
func (r ReciboLecturaRRHH) CapacidadHuellaSHA256() string {
	return r.capacidadHuella
}
func (r ReciboLecturaRRHH) MaterialHuellaSHA256() string {
	return r.materialHuella
}
func (r ReciboLecturaRRHH) ConsultaHuellaSHA256() string {
	return r.consultaHuella
}
func (r ReciboLecturaRRHH) TotalPublicado() uint16 {
	return r.totalPublicado
}
func (r ReciboLecturaRRHH) RegistradaEn() time.Time { return r.registradaEn }
func (ReciboLecturaRRHH) String() string            { return "[recibo-lectura-rrhh-redactado]" }
func (ReciboLecturaRRHH) GoString() string          { return "[recibo-lectura-rrhh-redactado]" }

type PaginaCuadroRRHH struct {
	GeneradaEn      time.Time               `json:"generada_en"`
	Expedientes     []ResumenExpedienteRRHH `json:"expedientes"`
	HayMas          bool                    `json:"hay_mas"`
	CursorSiguiente string                  `json:"cursor_siguiente,omitempty"`
	Lectura         ReciboLecturaRRHH       `json:"-"`
}

func (p PaginaCuadroRRHH) ValidarPara(orden OrdenConsultaCuadroRRHH) error {
	solicitud := orden.solicitud
	if solicitud.validar() != nil ||
		orden.capacidad.validaPara(
			orden.contexto, DominioHuellaConsultaCuadroRRHH,
			orden.consultaHuella, AccionConsultarCuadroRRHH,
			FinalidadConsultarCuadroRRHH, "", orden.instante,
		) != nil ||
		!domain.InstanteUTCCanonico(p.GeneradaEn) ||
		p.GeneradaEn.Before(orden.instante) ||
		len(p.Expedientes) > int(solicitud.limite) ||
		(p.HayMas && !cursorRRHHValido(p.CursorSiguiente)) ||
		(!p.HayMas && p.CursorSiguiente != "") ||
		!p.Lectura.coincideCon(orden.contexto, orden.capacidad, "", 0) ||
		p.Lectura.totalPublicado != uint16(len(p.Expedientes)) ||
		p.Lectura.registradaEn.Before(orden.instante) ||
		p.Lectura.registradaEn.Before(p.GeneradaEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	vistas := make(map[string]struct{}, len(p.Expedientes))
	for i, resumen := range p.Expedientes {
		if resumen.Validar() != nil || !resumen.cumpleAmbito(orden.capacidad) ||
			resumen.ActualizadoEn.After(p.GeneradaEn) ||
			(solicitud.texto != "" &&
				!strings.HasPrefix(resumen.NumeroVisible, solicitud.texto)) ||
			(solicitud.estadoClave != "" &&
				resumen.EstadoClave != solicitud.estadoClave) ||
			(solicitud.faseClave != "" &&
				resumen.FaseClave != solicitud.faseClave) {
			return ErrResultadoConsultaRRHHNoConfiable
		}
		if _, repetida := vistas[resumen.ExpedienteRef]; repetida {
			return ErrResultadoConsultaRRHHNoConfiable
		}
		vistas[resumen.ExpedienteRef] = struct{}{}
		if i == 0 {
			continue
		}
		anterior := p.Expedientes[i-1]
		if anterior.ActualizadoEn.Before(resumen.ActualizadoEn) ||
			(anterior.ActualizadoEn.Equal(resumen.ActualizadoEn) &&
				anterior.ExpedienteRef <= resumen.ExpedienteRef) {
			return ErrResultadoConsultaRRHHNoConfiable
		}
	}
	return nil
}

func (PaginaCuadroRRHH) String() string {
	return "[pagina-cuadro-rrhh-redactada]"
}

func (PaginaCuadroRRHH) GoString() string {
	return "[pagina-cuadro-rrhh-redactada]"
}
