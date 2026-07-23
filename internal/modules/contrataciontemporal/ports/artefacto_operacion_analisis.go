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
		m.ReferenciaCatalogo.EntradaClave !=
			string(m.ClaveMensajeI18N) ||
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
	ArtefactoRef          string
	ArtefactoHuellaSHA256 string
	OrganizacionRef       string
	ExpedienteRef         string
	VersionExpediente     uint64
	DatosFuncionales      DatosFuncionalesOperacionAnalisis
	ResultadoRC           domain.ResultadoValidacionRC
	FuenteRCRef           string
	ReciboRCRef           string
	ValidadaEn            time.Time
	FechaRC               *time.Time
	NumeroRC              string
	ImporteRC             *domain.Importe
	DocumentoRCRef        string
	MotivoRC              MotivoRCGobernado
	CostePrevisto         *domain.Importe
	FuenteCosteRef        string
	ReciboCosteRef        string
	CalculadoEn           time.Time
	PreparadoEn           time.Time
}

type ArtefactoAnalisisPreparado struct {
	bloqueoSerializacionOperacionAnalisis
	datos *DatosArtefactoAnalisis
}

func NuevoArtefactoAnalisisPreparado(
	solicitud SolicitudPrepararArtefactoAnalisis,
	datos DatosArtefactoAnalisis,
) (ArtefactoAnalisisPreparado, error) {
	if validarDatosArtefactoAnalisis(solicitud, datos) != nil {
		return ArtefactoAnalisisPreparado{},
			ErrArtefactoAnalisisNoConfiable
	}
	clon := clonarDatosArtefactoAnalisis(datos)
	return ArtefactoAnalisisPreparado{datos: &clon}, nil
}

func (a ArtefactoAnalisisPreparado) DatosPara(
	solicitud SolicitudPrepararArtefactoAnalisis,
) (DatosArtefactoAnalisis, error) {
	if a.datos == nil ||
		validarDatosArtefactoAnalisis(solicitud, *a.datos) != nil {
		return DatosArtefactoAnalisis{}, ErrArtefactoAnalisisNoConfiable
	}
	return clonarDatosArtefactoAnalisis(*a.datos), nil
}

type PreparadorArtefactoAnalisisO3 interface {
	PrepararArtefactoAnalisis(
		context.Context,
		SolicitudPrepararArtefactoAnalisis,
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
		datos.MotivoRC.validarPara(datos.ResultadoRC) != nil {
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
			!datos.CalculadoEn.IsZero() {
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
		(validacion.Resultado == domain.RCValidada &&
			datos.CostePrevisto.Centimos > validacion.Importe.Centimos) {
		return ErrArtefactoAnalisisNoConfiable
	}
	return nil
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
