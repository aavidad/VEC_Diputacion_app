package ports

import (
	"context"
	"crypto/subtle"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	dominioAmbitoOperacionAnalisis    = "vec.contratacion-temporal.analisis.ambito-idempotencia"
	dominioSemanticaOperacionAnalisis = "vec.contratacion-temporal.analisis.huella-semantica"
)

var (
	ErrPreparacionOperacionAnalisisInvalida = errors.New(
		"contratacion temporal: preparacion de operacion de analisis invalida",
	)
	ErrClaveIdempotenciaOperacionAnalisisUsada = errors.New(
		"contratacion temporal: clave de operacion de analisis usada con otros datos",
	)
)

type PreimagenesOperacionAnalisis struct {
	bloqueoSerializacionOperacionAnalisis
	ambito    []byte
	semantica []byte
}

func (p PreimagenesOperacionAnalisis) BytesAmbito() ([]byte, error) {
	if !preimagenOperacionAnalisisValida(p.ambito) {
		return nil, ErrOperacionAnalisisInvalida
	}
	return append([]byte(nil), p.ambito...), nil
}

func (p PreimagenesOperacionAnalisis) BytesSemantica() ([]byte, error) {
	if !preimagenOperacionAnalisisValida(p.semantica) {
		return nil, ErrOperacionAnalisisInvalida
	}
	return append([]byte(nil), p.semantica...), nil
}

type SellosOperacionAnalisis struct {
	AmbitosIdempotenciaHMAC ColeccionSellosHMAC
	HuellasSemanticasHMAC   ColeccionSellosHMAC
}

func (s SellosOperacionAnalisis) Validar() error {
	_, _, valido := coleccionesOperacionAnalisisAlineadas(
		s.AmbitosIdempotenciaHMAC,
		s.HuellasSemanticasHMAC,
	)
	if !valido {
		return ErrPreparacionOperacionAnalisisInvalida
	}
	return nil
}

func (s SellosOperacionAnalisis) ParActivo() (string, string, error) {
	ambitos, huellas, valido := coleccionesOperacionAnalisisAlineadas(
		s.AmbitosIdempotenciaHMAC,
		s.HuellasSemanticasHMAC,
	)
	if !valido {
		return "", "", ErrPreparacionOperacionAnalisisInvalida
	}
	return ambitos.Activo.Valor, huellas.Activo.Valor, nil
}

func (s SellosOperacionAnalisis) ContienePar(
	ambito string,
	semantica string,
) bool {
	ambitos, huellas, valido := coleccionesOperacionAnalisisAlineadas(
		s.AmbitosIdempotenciaHMAC,
		s.HuellasSemanticasHMAC,
	)
	if !valido {
		return false
	}
	coincide := func(
		candidatoAmbito SelloGeneracionalHMAC,
		candidataSemantica SelloGeneracionalHMAC,
	) bool {
		return candidatoAmbito.Generacion == candidataSemantica.Generacion &&
			subtle.ConstantTimeCompare(
				[]byte(candidatoAmbito.Valor),
				[]byte(ambito),
			) == 1 &&
			subtle.ConstantTimeCompare(
				[]byte(candidataSemantica.Valor),
				[]byte(semantica),
			) == 1
	}
	encontrado := coincide(ambitos.Activo, huellas.Activo)
	for indice := range ambitos.Retenidos {
		encontrado = coincide(
			ambitos.Retenidos[indice],
			huellas.Retenidos[indice],
		) || encontrado
	}
	return encontrado
}

type SelladorOperacionAnalisis interface {
	SellarOperacionAnalisis(
		context.Context,
		PreimagenesOperacionAnalisis,
	) (SellosOperacionAnalisis, error)
}

type SolicitudPrepararOperacionAnalisis struct {
	Operacion             TipoOperacionAnalisis
	OrganizacionRef       string
	ExpedienteRef         string
	VersionExpediente     uint64
	ActorRef              string
	PerfilRef             string
	ArtefactoRef          string
	ArtefactoHuellaSHA256 string
	Sellos                SellosOperacionAnalisis
}

func (s SolicitudPrepararOperacionAnalisis) Validar() error {
	if !s.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(s.VersionExpediente) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) ||
		!domain.ReferenciaOpacaValida(s.ArtefactoRef) ||
		!huellaSHA256OperacionAnalisisValida(s.ArtefactoHuellaSHA256) ||
		s.Sellos.Validar() != nil {
		return ErrPreparacionOperacionAnalisisInvalida
	}
	return nil
}

type EstadoPreparacionOperacionAnalisis string

const (
	PreparacionOperacionAnalisisReservada  EstadoPreparacionOperacionAnalisis = "reservada"
	PreparacionOperacionAnalisisConfirmada EstadoPreparacionOperacionAnalisis = "confirmada"
)

type DatosPreparacionOperacionAnalisis struct {
	bloqueoSerializacionOperacionAnalisis
	ReservaRef             string
	ReciboRef              string
	Operacion              TipoOperacionAnalisis
	OrganizacionRef        string
	ExpedienteRef          string
	VersionExpediente      uint64
	ActorRef               string
	PerfilRef              string
	ArtefactoRef           string
	ArtefactoHuellaSHA256  string
	AmbitoIdempotenciaHMAC string
	HuellaSemanticaHMAC    string
	Estado                 EstadoPreparacionOperacionAnalisis
	ExpedienteAnterior     *domain.Expediente
	ReciboConfirmado       *ReciboOperacionAnalisis
}

type PreparacionOperacionAnalisis struct {
	bloqueoSerializacionOperacionAnalisis
	datos *DatosPreparacionOperacionAnalisis
}

func NuevaPreparacionOperacionAnalisis(
	solicitud SolicitudPrepararOperacionAnalisis,
	datos DatosPreparacionOperacionAnalisis,
) (PreparacionOperacionAnalisis, error) {
	if validarDatosPreparacionOperacionAnalisis(solicitud, datos) != nil {
		return PreparacionOperacionAnalisis{},
			ErrPreparacionOperacionAnalisisInvalida
	}
	clon := clonarDatosPreparacionOperacionAnalisis(datos)
	return PreparacionOperacionAnalisis{datos: &clon}, nil
}

func (p PreparacionOperacionAnalisis) DatosPara(
	solicitud SolicitudPrepararOperacionAnalisis,
) (DatosPreparacionOperacionAnalisis, error) {
	if p.datos == nil ||
		validarDatosPreparacionOperacionAnalisis(solicitud, *p.datos) != nil {
		return DatosPreparacionOperacionAnalisis{},
			ErrPreparacionOperacionAnalisisInvalida
	}
	return clonarDatosPreparacionOperacionAnalisis(*p.datos), nil
}

type PreparadorOperacionAnalisisIdempotente interface {
	// ConsultarOperacionAnalisisConfirmada debe comparar de forma durable la
	// identidad semántica completa, no solo la clave. Un confirmado exacto se
	// devuelve antes de consultar fuentes; la misma clave con otros datos
	// devuelve ErrClaveIdempotenciaOperacionAnalisisUsada.
	ConsultarOperacionAnalisisConfirmada(
		context.Context,
		SolicitudConsultarOperacionAnalisisConfirmada,
	) (ReciboOperacionAnalisis, bool, error)
	PrepararOperacionAnalisis(
		context.Context,
		SolicitudPrepararOperacionAnalisis,
	) (PreparacionOperacionAnalisis, error)
}

// SolicitudConsultarOperacionAnalisisConfirmada contiene únicamente la
// semántica aportada por el cliente más actor y perfil ya resueltos por el
// contexto confiable. No incorpora artefactos, raíces ni resultados O3-03.
type SolicitudConsultarOperacionAnalisisConfirmada struct {
	Operacion           TipoOperacionAnalisis
	OrganizacionRef     string
	ExpedienteRef       string
	VersionExpediente   uint64
	ActorRef            string
	PerfilRef           string
	ClaveIdempotencia   string
	ArtefactoRef        string
	DatosFuncionales    DatosFuncionalesOperacionAnalisis
	MotivoRectificacion domain.ClaveCatalogo
}

func (s SolicitudConsultarOperacionAnalisisConfirmada) Validar() error {
	if !s.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(
			s.VersionExpediente,
		) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) ||
		!ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		!domain.ReferenciaOpacaValida(s.ArtefactoRef) ||
		s.DatosFuncionales.Validar() != nil {
		return ErrPreparacionOperacionAnalisisInvalida
	}
	if s.Operacion == OperacionRegistrarAnalisis {
		if s.MotivoRectificacion != "" {
			return ErrPreparacionOperacionAnalisisInvalida
		}
		return nil
	}
	if !s.MotivoRectificacion.Valida() {
		return ErrPreparacionOperacionAnalisisInvalida
	}
	return nil
}

func validarDatosPreparacionOperacionAnalisis(
	solicitud SolicitudPrepararOperacionAnalisis,
	datos DatosPreparacionOperacionAnalisis,
) error {
	if solicitud.Validar() != nil ||
		!domain.ReferenciaOpacaValida(datos.ReservaRef) ||
		!domain.ReferenciaOpacaValida(datos.ReciboRef) ||
		datos.Operacion != solicitud.Operacion ||
		datos.OrganizacionRef != solicitud.OrganizacionRef ||
		datos.ExpedienteRef != solicitud.ExpedienteRef ||
		datos.VersionExpediente != solicitud.VersionExpediente ||
		datos.ActorRef != solicitud.ActorRef ||
		datos.PerfilRef != solicitud.PerfilRef ||
		datos.ArtefactoRef != solicitud.ArtefactoRef ||
		datos.ArtefactoHuellaSHA256 !=
			solicitud.ArtefactoHuellaSHA256 ||
		!solicitud.Sellos.ContienePar(
			datos.AmbitoIdempotenciaHMAC,
			datos.HuellaSemanticaHMAC,
		) {
		return ErrPreparacionOperacionAnalisisInvalida
	}
	switch datos.Estado {
	case PreparacionOperacionAnalisisReservada:
		if datos.ExpedienteAnterior == nil ||
			datos.ReciboConfirmado != nil ||
			datos.ExpedienteAnterior.Validar() != nil ||
			datos.ExpedienteAnterior.Referencia != datos.ExpedienteRef ||
			datos.ExpedienteAnterior.OrganizacionRef !=
				datos.OrganizacionRef ||
			datos.ExpedienteAnterior.Version != datos.VersionExpediente ||
			!VersionOperacionAnalisisConIncrementoValida(
				datos.ExpedienteAnterior.Version,
			) {
			return ErrPreparacionOperacionAnalisisInvalida
		}
	case PreparacionOperacionAnalisisConfirmada:
		if datos.ExpedienteAnterior != nil ||
			datos.ReciboConfirmado == nil ||
			datos.ReciboConfirmado.ValidarParaPreparacion(datos) != nil {
			return ErrPreparacionOperacionAnalisisInvalida
		}
	default:
		return ErrPreparacionOperacionAnalisisInvalida
	}
	return nil
}

func clonarDatosPreparacionOperacionAnalisis(
	datos DatosPreparacionOperacionAnalisis,
) DatosPreparacionOperacionAnalisis {
	if datos.ExpedienteAnterior != nil {
		expediente := datos.ExpedienteAnterior.Clonar()
		datos.ExpedienteAnterior = &expediente
	}
	if datos.ReciboConfirmado != nil {
		recibo := *datos.ReciboConfirmado
		datos.ReciboConfirmado = &recibo
	}
	return datos
}

func coleccionesOperacionAnalisisAlineadas(
	ambitos ColeccionSellosHMAC,
	huellas ColeccionSellosHMAC,
) (
	DatosColeccionSellosHMAC,
	DatosColeccionSellosHMAC,
	bool,
) {
	if ambitos.ValidarDominio(dominioAmbitoOperacionAnalisis) != nil ||
		huellas.ValidarDominio(dominioSemanticaOperacionAnalisis) != nil {
		return DatosColeccionSellosHMAC{}, DatosColeccionSellosHMAC{}, false
	}
	datosAmbitos, errAmbitos := ambitos.Datos()
	datosHuellas, errHuellas := huellas.Datos()
	if errAmbitos != nil || errHuellas != nil ||
		datosAmbitos.Activo.Generacion != datosHuellas.Activo.Generacion ||
		len(datosAmbitos.Retenidos) != len(datosHuellas.Retenidos) {
		return DatosColeccionSellosHMAC{}, DatosColeccionSellosHMAC{}, false
	}
	for indice := range datosAmbitos.Retenidos {
		if datosAmbitos.Retenidos[indice].Generacion !=
			datosHuellas.Retenidos[indice].Generacion {
			return DatosColeccionSellosHMAC{}, DatosColeccionSellosHMAC{}, false
		}
	}
	return datosAmbitos, datosHuellas, true
}
