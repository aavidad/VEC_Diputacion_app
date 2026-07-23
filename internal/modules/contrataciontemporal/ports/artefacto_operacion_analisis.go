package ports

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrSolicitudArtefactoAnalisisInvalida = errors.New(
		"contratacion temporal: solicitud de artefacto de analisis invalida",
	)
	ErrArtefactoAnalisisNoDisponible = errors.New(
		"contratacion temporal: artefacto de analisis no disponible",
	)
	ErrArtefactoAnalisisNoConfiable = errors.New(
		"contratacion temporal: artefacto de analisis no confiable",
	)
)

type DatosFuncionalesOperacionAnalisis struct {
	ModalidadClave    domain.ClaveCatalogo
	CategoriaRef      string
	GrupoSubgrupo     string
	CausaClave        domain.ClaveCatalogo
	Periodo           domain.PeriodoPrevisto
	PorcentajeJornada domain.JornadaDiezmilesimas
	EntradaRC         domain.VinculoEntradaRC
}

func (d DatosFuncionalesOperacionAnalisis) Validar() error {
	if !d.ModalidadClave.Valida() ||
		!domain.ReferenciaOpacaValida(d.CategoriaRef) ||
		!domain.GrupoSubgrupoValido(d.GrupoSubgrupo) ||
		!d.CausaClave.Valida() || d.Periodo.Validar() != nil ||
		d.PorcentajeJornada.Validar() != nil ||
		d.EntradaRC.Validar() != nil {
		return ErrSolicitudArtefactoAnalisisInvalida
	}
	return nil
}

type SolicitudPrepararArtefactoAnalisis struct {
	ArtefactoRef      string
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	DatosFuncionales  DatosFuncionalesOperacionAnalisis
	SolicitadaEn      time.Time
}

func (s SolicitudPrepararArtefactoAnalisis) Validar() error {
	if !domain.ReferenciaOpacaValida(s.ArtefactoRef) ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(s.VersionExpediente) ||
		s.DatosFuncionales.Validar() != nil ||
		!instanteSeguroOperacionAnalisis(s.SolicitadaEn) {
		return ErrSolicitudArtefactoAnalisisInvalida
	}
	return nil
}

type MotivoRCGobernado struct {
	ReferenciaCatalogo dominiovec.ReferenciaEntradaCatalogo
	ClaveMensajeI18N   domain.ClaveCatalogo
}

func (m MotivoRCGobernado) validarPara(
	resultado domain.ResultadoValidacionRC,
) error {
	if resultado == domain.RCValidada {
		if m != (MotivoRCGobernado{}) {
			return ErrArtefactoAnalisisNoConfiable
		}
		return nil
	}
	if m.ReferenciaCatalogo.Validar() != nil ||
		uint64(m.ReferenciaCatalogo.CatalogoVersion) >
			MaximoEnteroSeguroOperacionAnalisis ||
		!m.ClaveMensajeI18N.Valida() ||
		!strings.HasPrefix(
			string(m.ClaveMensajeI18N),
			"contratacion_temporal.rc.",
		) {
		return ErrArtefactoAnalisisNoConfiable
	}
	return nil
}

// DatosArtefactoAnalisis contiene la proyección mínima necesaria para derivar
// el análisis. Está bloqueado para codecs: solo cruza llamadas tipadas.
type DatosArtefactoAnalisis struct {
	bloqueoSerializacionOperacionAnalisis
	ArtefactoRef                string
	ArtefactoHuellaSHA256       string
	OrganizacionRef             string
	ExpedienteRef               string
	VersionExpediente           uint64
	DatosFuncionales            DatosFuncionalesOperacionAnalisis
	ResultadoRC                 domain.ResultadoValidacionRC
	FuenteRCRef                 string
	ReciboRCRef                 string
	ValidadaEn                  time.Time
	FechaRC                     *time.Time
	NumeroRC                    string
	ImporteRC                   *domain.Importe
	DocumentoRCRef              string
	MotivoRC                    MotivoRCGobernado
	PeticionRCRef               string
	HuellaPeticionRCHMAC        string
	HuellaRespuestaRC           string
	SelloRespuestaRCHMAC        string
	GeneracionRespuestaRC       uint32
	ConfirmadaRCEn              time.Time
	RespuestaRCValidaHasta      time.Time
	ConsumoRCRef                string
	ConsumidaRCEn               time.Time
	AutoridadFuenteRC           VinculoAutoridadFuenteAnalisisO3
	AutoridadVerificadorRC      VinculoAutoridadFuenteAnalisisO3
	AutoridadPublicadorRC       VinculoAutoridadFuenteAnalisisO3
	PublicacionMotivoRef        string
	ReciboVerificacionMotivoRef string
	CostePrevisto               *domain.Importe
	FuenteCosteRef              string
	ReciboCosteRef              string
	CalculadoEn                 time.Time
	PeticionCosteRef            string
	HuellaPeticionCosteHMAC     string
	HuellaRespuestaCoste        string
	SelloRespuestaCosteHMAC     string
	GeneracionRespuestaCoste    uint32
	ConfirmadaCosteEn           time.Time
	RespuestaCosteValidaHasta   time.Time
	ConsumoCosteRef             string
	ConsumidaCosteEn            time.Time
	AutoridadFuenteCoste        VinculoAutoridadFuenteAnalisisO3
	AutoridadVerificadorCoste   VinculoAutoridadFuenteAnalisisO3
	PreparadoEn                 time.Time
}

type ArtefactoAnalisisPreparado struct {
	bloqueoSerializacionOperacionAnalisis
	datos   *DatosArtefactoAnalisis
	pruebas *pruebasArtefactoAnalisisO3
}

func (a ArtefactoAnalisisPreparado) DatosPara(
	solicitud SolicitudPrepararArtefactoAnalisis,
) (DatosArtefactoAnalisis, error) {
	if validarArtefactoAnalisisPreparado(solicitud, a) != nil {
		return DatosArtefactoAnalisis{}, ErrArtefactoAnalisisNoConfiable
	}
	return clonarDatosArtefactoAnalisis(*a.datos), nil
}

type PreparadorArtefactoAnalisisO3 interface {
	PrepararArtefactoAnalisis(
		context.Context,
		SolicitudPrepararArtefactoAnalisis,
	) (ArtefactoAnalisisPreparado, error)
	ConsumirArtefactoAnalisisO3(
		context.Context,
		SolicitudPrepararArtefactoAnalisis,
		ArtefactoAnalisisPreparado,
	) (ArtefactoAnalisisPreparado, error)
}

func validarDatosArtefactoAnalisis(
	solicitud SolicitudPrepararArtefactoAnalisis,
	datos DatosArtefactoAnalisis,
) error {
	if solicitud.Validar() != nil ||
		datos.ArtefactoRef != solicitud.ArtefactoRef ||
		!huellaSHA256OperacionAnalisisValida(datos.ArtefactoHuellaSHA256) ||
		datos.OrganizacionRef != solicitud.OrganizacionRef ||
		datos.ExpedienteRef != solicitud.ExpedienteRef ||
		datos.VersionExpediente != solicitud.VersionExpediente ||
		!datosFuncionalesOperacionAnalisisIguales(
			datos.DatosFuncionales,
			solicitud.DatosFuncionales,
		) ||
		!instanteSeguroOperacionAnalisis(datos.PreparadoEn) ||
		datos.PreparadoEn.Before(solicitud.SolicitadaEn) ||
		!instanteSeguroOperacionAnalisis(datos.ValidadaEn) ||
		datos.ValidadaEn.After(datos.PreparadoEn) ||
		!domain.ReferenciaOpacaValida(datos.FuenteRCRef) ||
		!domain.ReferenciaOpacaValida(datos.ReciboRCRef) ||
		datos.MotivoRC.validarPara(datos.ResultadoRC) != nil ||
		validarPruebaRCDatosArtefacto(datos) != nil {
		return ErrArtefactoAnalisisNoConfiable
	}
	validacion, err := derivarValidacionRC(datos)
	if err != nil || validacion.Validar() != nil ||
		validacion.EntradaRef != solicitud.DatosFuncionales.EntradaRC.Referencia ||
		subtle.ConstantTimeCompare(
			[]byte(validacion.HuellaEntradaSHA256),
			[]byte(solicitud.DatosFuncionales.EntradaRC.HuellaSHA256),
		) != 1 ||
		validarCosteArtefactoAnalisis(datos, validacion) != nil {
		return ErrArtefactoAnalisisNoConfiable
	}
	return nil
}

func derivarValidacionRC(
	datos DatosArtefactoAnalisis,
) (domain.ValidacionRC, error) {
	validacion := domain.ValidacionRC{
		Resultado:           datos.ResultadoRC,
		EntradaRef:          datos.DatosFuncionales.EntradaRC.Referencia,
		HuellaEntradaSHA256: datos.DatosFuncionales.EntradaRC.HuellaSHA256,
		FuenteRef:           datos.FuenteRCRef,
		ReciboRef:           datos.ReciboRCRef,
		ValidadaEn:          datos.ValidadaEn,
		FechaRC:             clonarTiempo(datos.FechaRC),
		Numero:              datos.NumeroRC,
		Importe:             clonarImporte(datos.ImporteRC),
		DocumentoRef:        datos.DocumentoRCRef,
	}
	if datos.ResultadoRC != domain.RCValidada {
		validacion.Motivo = string(datos.MotivoRC.ClaveMensajeI18N)
	}
	if validacion.Validar() != nil {
		return domain.ValidacionRC{}, ErrArtefactoAnalisisNoConfiable
	}
	return validacion, nil
}

func DerivarAnalisisDesdeArtefacto(
	solicitud SolicitudPrepararArtefactoAnalisis,
	artefacto ArtefactoAnalisisPreparado,
) (domain.AnalisisRRHH, error) {
	datos, err := artefacto.DatosPara(solicitud)
	if err != nil {
		return domain.AnalisisRRHH{}, ErrArtefactoAnalisisNoConfiable
	}
	validacion, err := derivarValidacionRC(datos)
	if err != nil {
		return domain.AnalisisRRHH{}, ErrArtefactoAnalisisNoConfiable
	}
	analisis := domain.AnalisisRRHH{
		ModalidadClave:    datos.DatosFuncionales.ModalidadClave,
		CategoriaRef:      datos.DatosFuncionales.CategoriaRef,
		GrupoSubgrupo:     datos.DatosFuncionales.GrupoSubgrupo,
		CausaClave:        datos.DatosFuncionales.CausaClave,
		Periodo:           datos.DatosFuncionales.Periodo,
		PorcentajeJornada: datos.DatosFuncionales.PorcentajeJornada,
		EntradaRCEsperada: datos.DatosFuncionales.EntradaRC,
		ValidacionRC:      validacion,
		CostePrevisto:     clonarImporte(datos.CostePrevisto),
		FuenteCosteRef:    datos.FuenteCosteRef,
	}
	if analisis.Validar() != nil || analisis.ActuacionRegistro != nil ||
		analisis.Observaciones != "" {
		return domain.AnalisisRRHH{}, ErrArtefactoAnalisisNoConfiable
	}
	return analisis, nil
}

func validarCosteArtefactoAnalisis(
	datos DatosArtefactoAnalisis,
	validacion domain.ValidacionRC,
) error {
	if datos.CostePrevisto == nil {
		if datos.FuenteCosteRef != "" || datos.ReciboCosteRef != "" ||
			!datos.CalculadoEn.IsZero() ||
			datos.PeticionCosteRef != "" ||
			datos.HuellaPeticionCosteHMAC != "" ||
			datos.HuellaRespuestaCoste != "" ||
			datos.SelloRespuestaCosteHMAC != "" ||
			datos.GeneracionRespuestaCoste != 0 ||
			!datos.ConfirmadaCosteEn.IsZero() ||
			!datos.RespuestaCosteValidaHasta.IsZero() ||
			datos.ConsumoCosteRef != "" ||
			!datos.ConsumidaCosteEn.IsZero() ||
			datos.AutoridadFuenteCoste !=
				(VinculoAutoridadFuenteAnalisisO3{}) ||
			datos.AutoridadVerificadorCoste !=
				(VinculoAutoridadFuenteAnalisisO3{}) {
			return ErrArtefactoAnalisisNoConfiable
		}
		return nil
	}
	if datos.CostePrevisto.Validar(false) != nil ||
		!enteroCanonicoOperacionAnalisisValido(datos.CostePrevisto.Centimos) ||
		!domain.ReferenciaOpacaValida(datos.FuenteCosteRef) ||
		!domain.ReferenciaOpacaValida(datos.ReciboCosteRef) ||
		!instanteSeguroOperacionAnalisis(datos.CalculadoEn) ||
		datos.CalculadoEn.After(datos.PreparadoEn) ||
		!referenciaPeticionFuenteAnalisisValida(
			datos.PeticionCosteRef,
		) ||
		!selloPeticionFuenteAnalisisValido(
			datos.HuellaPeticionCosteHMAC,
		) ||
		!huellaSHA256FuenteAnalisisValida(
			datos.HuellaRespuestaCoste,
		) ||
		!selloRespuestaFuenteAnalisisValido(
			datos.SelloRespuestaCosteHMAC,
			datos.GeneracionRespuestaCoste,
		) ||
		!instanteSeguroOperacionAnalisis(datos.ConfirmadaCosteEn) ||
		!instanteSeguroOperacionAnalisis(
			datos.RespuestaCosteValidaHasta,
		) ||
		!datos.ConfirmadaCosteEn.Before(
			datos.RespuestaCosteValidaHasta,
		) ||
		!consumoRespuestaArtefactoAnalisisValido(
			datos.ConsumoCosteRef,
			datos.ConsumidaCosteEn,
			datos.ConfirmadaCosteEn,
			datos.RespuestaCosteValidaHasta,
		) ||
		!vinculoAutoridadAnalisisValido(
			datos.AutoridadFuenteCoste,
			RolCalculadorCoste,
		) ||
		!vinculoAutoridadAnalisisValido(
			datos.AutoridadVerificadorCoste,
			RolVerificadorRespuesta,
		) ||
		datos.AutoridadFuenteCoste.AutoridadRef !=
			datos.FuenteCosteRef ||
		(validacion.Resultado == domain.RCValidada &&
			datos.CostePrevisto.Centimos > validacion.Importe.Centimos) {
		return ErrArtefactoAnalisisNoConfiable
	}
	return nil
}

func validarPruebaRCDatosArtefacto(
	datos DatosArtefactoAnalisis,
) error {
	if !referenciaPeticionFuenteAnalisisValida(datos.PeticionRCRef) ||
		!selloPeticionFuenteAnalisisValido(
			datos.HuellaPeticionRCHMAC,
		) ||
		!huellaSHA256FuenteAnalisisValida(datos.HuellaRespuestaRC) ||
		!selloRespuestaFuenteAnalisisValido(
			datos.SelloRespuestaRCHMAC,
			datos.GeneracionRespuestaRC,
		) ||
		!instanteSeguroOperacionAnalisis(datos.ConfirmadaRCEn) ||
		!instanteSeguroOperacionAnalisis(
			datos.RespuestaRCValidaHasta,
		) ||
		!datos.ConfirmadaRCEn.Before(datos.RespuestaRCValidaHasta) ||
		!consumoRespuestaArtefactoAnalisisValido(
			datos.ConsumoRCRef,
			datos.ConsumidaRCEn,
			datos.ConfirmadaRCEn,
			datos.RespuestaRCValidaHasta,
		) ||
		!vinculoAutoridadAnalisisValido(
			datos.AutoridadFuenteRC,
			RolFuentePresupuestaria,
		) ||
		!vinculoAutoridadAnalisisValido(
			datos.AutoridadVerificadorRC,
			RolVerificadorRespuesta,
		) ||
		!vinculoAutoridadAnalisisValido(
			datos.AutoridadPublicadorRC,
			RolPublicadorCatalogo,
		) ||
		datos.AutoridadFuenteRC.AutoridadRef != datos.FuenteRCRef {
		return ErrArtefactoAnalisisNoConfiable
	}
	if datos.ResultadoRC == domain.RCValidada {
		if datos.PublicacionMotivoRef != "" ||
			datos.ReciboVerificacionMotivoRef != "" {
			return ErrArtefactoAnalisisNoConfiable
		}
		return nil
	}
	if !domain.ReferenciaOpacaValida(datos.PublicacionMotivoRef) ||
		!domain.ReferenciaOpacaValida(
			datos.ReciboVerificacionMotivoRef,
		) {
		return ErrArtefactoAnalisisNoConfiable
	}
	return nil
}

func consumoRespuestaArtefactoAnalisisValido(
	consumoRef string,
	consumidaEn time.Time,
	confirmadaEn time.Time,
	validaHasta time.Time,
) bool {
	if consumoRef == "" && consumidaEn.IsZero() {
		return true
	}
	return domain.ReferenciaOpacaValida(consumoRef) &&
		instanteSeguroOperacionAnalisis(consumidaEn) &&
		!consumidaEn.Before(confirmadaEn) &&
		consumidaEn.Before(validaHasta)
}

func vinculoAutoridadAnalisisValido(
	vinculo VinculoAutoridadFuenteAnalisisO3,
	rol RolAutoridadFuenteAnalisis,
) bool {
	return vinculo.Rol == rol &&
		domain.ReferenciaOpacaValida(vinculo.RaizClaveID) &&
		domain.ReferenciaOpacaValida(vinculo.AutoridadRef) &&
		domain.ReferenciaOpacaValida(vinculo.BackendRef) &&
		vinculo.Serie > 0 &&
		vinculo.Serie <= MaximoEnteroSeguroOperacionAnalisis &&
		vinculo.Generacion > 0 &&
		huellaSHA256OperacionAnalisisValida(vinculo.HuellaClaveSHA256) &&
		instanteSeguroOperacionAnalisis(vinculo.CredencialEmitidaEn) &&
		instanteSeguroOperacionAnalisis(
			vinculo.CredencialValidaHasta,
		) &&
		vinculo.CredencialValidaHasta.After(vinculo.CredencialEmitidaEn)
}

func clonarDatosArtefactoAnalisis(
	datos DatosArtefactoAnalisis,
) DatosArtefactoAnalisis {
	datos.FechaRC = clonarTiempo(datos.FechaRC)
	datos.ImporteRC = clonarImporte(datos.ImporteRC)
	datos.CostePrevisto = clonarImporte(datos.CostePrevisto)
	return datos
}

func clonarTiempo(valor *time.Time) *time.Time {
	if valor == nil {
		return nil
	}
	clon := *valor
	return &clon
}

func clonarImporte(valor *domain.Importe) *domain.Importe {
	if valor == nil {
		return nil
	}
	clon := *valor
	return &clon
}

func datosFuncionalesOperacionAnalisisIguales(
	primero DatosFuncionalesOperacionAnalisis,
	segundo DatosFuncionalesOperacionAnalisis,
) bool {
	return primero.ModalidadClave == segundo.ModalidadClave &&
		primero.CategoriaRef == segundo.CategoriaRef &&
		primero.GrupoSubgrupo == segundo.GrupoSubgrupo &&
		primero.CausaClave == segundo.CausaClave &&
		primero.Periodo.Inicio.Equal(segundo.Periodo.Inicio) &&
		primero.Periodo.Fin.Equal(segundo.Periodo.Fin) &&
		primero.PorcentajeJornada == segundo.PorcentajeJornada &&
		primero.EntradaRC.Referencia == segundo.EntradaRC.Referencia &&
		subtle.ConstantTimeCompare(
			[]byte(primero.EntradaRC.HuellaSHA256),
			[]byte(segundo.EntradaRC.HuellaSHA256),
		) == 1
}

func huellaSHA256OperacionAnalisisValida(valor string) bool {
	if len(valor) != 64 || valor == strings.Repeat("0", 64) {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') &&
			(caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

func instanteSeguroOperacionAnalisis(valor time.Time) bool {
	return domain.InstanteUTCCanonico(valor) &&
		enteroCanonicoOperacionAnalisisValido(valor.UnixMicro())
}
