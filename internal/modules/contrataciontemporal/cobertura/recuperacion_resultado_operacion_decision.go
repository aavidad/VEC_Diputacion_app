package cobertura

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// PreimagenAmbitoRecuperacionOperacionDecisionCobertura conserva únicamente
// el ámbito estable C3. Su canon es compartido byte a byte con la operación de
// efecto; no reconstruye vía, motivo, catálogo ni identidad semántica.
type PreimagenAmbitoRecuperacionOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *datosAmbitoRecuperacionOperacionDecisionCobertura
	bytes []byte
}

type datosAmbitoRecuperacionOperacionDecisionCobertura struct {
	organizacionRef string
	expedienteRef   string
}

func NuevaPreimagenAmbitoRecuperacionOperacionDecisionCobertura(
	claveIdempotencia string,
	expedienteRef string,
	contextoRecuperacion ports.ContextoRecuperacionResultadoCobertura,
	autenticadaEn time.Time,
) (PreimagenAmbitoRecuperacionOperacionDecisionCobertura, error) {
	solicitudContexto, contexto, organizacionRef, err :=
		contextoRecuperacion.DatosEn(autenticadaEn)
	if !ports.ClaveIdempotenciaValida(claveIdempotencia) ||
		!domain.ReferenciaOpacaValida(organizacionRef) ||
		!domain.ReferenciaOpacaValida(expedienteRef) ||
		err != nil ||
		contexto.ValidarPara(solicitudContexto, autenticadaEn) != nil {
		return PreimagenAmbitoRecuperacionOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		return PreimagenAmbitoRecuperacionOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	canon, err := canonAmbitoOperacionDecisionCobertura(
		claveIdempotencia,
		organizacionRef,
		expedienteRef,
		vinculo.PrincipalID,
		vinculo.PerfilActivoRef,
	)
	if err != nil {
		return PreimagenAmbitoRecuperacionOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return PreimagenAmbitoRecuperacionOperacionDecisionCobertura{
		datos: &datosAmbitoRecuperacionOperacionDecisionCobertura{
			organizacionRef: organizacionRef,
			expedienteRef:   expedienteRef,
		},
		bytes: canon,
	}, nil
}

func (p PreimagenAmbitoRecuperacionOperacionDecisionCobertura) Bytes() (
	[]byte,
	error,
) {
	if p.datos == nil ||
		!domain.ReferenciaOpacaValida(p.datos.organizacionRef) ||
		!domain.ReferenciaOpacaValida(p.datos.expedienteRef) ||
		!preimagenOperacionDecisionCoberturaValida(p.bytes) {
		return nil, ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return append([]byte(nil), p.bytes...), nil
}

// SelladorAmbitoOperacionDecisionCobertura usa el mismo dominio y llavero
// rotatorio que SelladorOperacionDecisionCobertura, pero no acepta semántica.
type SelladorAmbitoOperacionDecisionCobertura interface {
	SellarAmbitoOperacionDecisionCobertura(
		context.Context,
		PreimagenAmbitoRecuperacionOperacionDecisionCobertura,
	) (ports.ColeccionSellosHMAC, error)
}

// DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura es la vista
// mínima del lector durable. No incluye actor, perfil, clave ni versión.
type DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	OrganizacionRef string
	ExpedienteRef   string
	AmbitosHMAC     ports.ColeccionSellosHMAC
}

// SolicitudRecuperacionResultadoOperacionDecisionCobertura es una capacidad
// nominal de lectura. Solo nace de la preimagen confiable y sus sellos C3.
type SolicitudRecuperacionResultadoOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura
}

func NuevaSolicitudRecuperacionResultadoOperacionDecisionCobertura(
	preimagen PreimagenAmbitoRecuperacionOperacionDecisionCobertura,
	ambitos ports.ColeccionSellosHMAC,
) (SolicitudRecuperacionResultadoOperacionDecisionCobertura, error) {
	if _, err := preimagen.Bytes(); err != nil ||
		ambitos.ValidarDominio(dominioAmbitoOperacionDecisionCobertura) != nil {
		return SolicitudRecuperacionResultadoOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	datos := &DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura{
		OrganizacionRef: preimagen.datos.organizacionRef,
		ExpedienteRef:   preimagen.datos.expedienteRef,
		AmbitosHMAC:     ambitos,
	}
	solicitud := SolicitudRecuperacionResultadoOperacionDecisionCobertura{
		datos: datos,
	}
	if _, err := solicitud.DatosLectura(); err != nil {
		return SolicitudRecuperacionResultadoOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return solicitud, nil
}

func (s SolicitudRecuperacionResultadoOperacionDecisionCobertura) DatosLectura() (
	DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura,
	error,
) {
	if s.datos == nil ||
		!domain.ReferenciaOpacaValida(s.datos.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.datos.ExpedienteRef) ||
		s.datos.AmbitosHMAC.ValidarDominio(
			dominioAmbitoOperacionDecisionCobertura,
		) != nil {
		return DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	ambitos, err := clonarColeccionSellosHMAC(s.datos.AmbitosHMAC)
	if err != nil {
		return DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura{},
			ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura{
		OrganizacionRef: s.datos.OrganizacionRef,
		ExpedienteRef:   s.datos.ExpedienteRef,
		AmbitosHMAC:     ambitos,
	}, nil
}

func (s SolicitudRecuperacionResultadoOperacionDecisionCobertura) contieneAmbito(
	ambito string,
) bool {
	return s.datos != nil &&
		s.datos.AmbitosHMAC.ValidarDominio(
			dominioAmbitoOperacionDecisionCobertura,
		) == nil &&
		s.datos.AmbitosHMAC.Contiene(ambito)
}

func clonarColeccionSellosHMAC(
	origen ports.ColeccionSellosHMAC,
) (ports.ColeccionSellosHMAC, error) {
	datos, err := origen.Datos()
	if err != nil {
		return ports.ColeccionSellosHMAC{}, err
	}
	retenidos := make([]string, 0, len(datos.Retenidos))
	for _, retenido := range datos.Retenidos {
		retenidos = append(retenidos, retenido.Valor)
	}
	return ports.NuevaColeccionSellosHMAC(datos.Activo.Valor, retenidos)
}

func clonarSolicitudRecuperacionResultadoOperacionDecisionCobertura(
	origen SolicitudRecuperacionResultadoOperacionDecisionCobertura,
) (SolicitudRecuperacionResultadoOperacionDecisionCobertura, error) {
	datos, err := origen.DatosLectura()
	if err != nil {
		return SolicitudRecuperacionResultadoOperacionDecisionCobertura{}, err
	}
	return SolicitudRecuperacionResultadoOperacionDecisionCobertura{
		datos: &datos,
	}, nil
}

func solicitudesRecuperacionResultadoOperacionDecisionCoberturaIguales(
	primera SolicitudRecuperacionResultadoOperacionDecisionCobertura,
	segunda SolicitudRecuperacionResultadoOperacionDecisionCobertura,
) bool {
	datosPrimera, errPrimera := primera.DatosLectura()
	datosSegunda, errSegunda := segunda.DatosLectura()
	if errPrimera != nil || errSegunda != nil ||
		datosPrimera.OrganizacionRef != datosSegunda.OrganizacionRef ||
		datosPrimera.ExpedienteRef != datosSegunda.ExpedienteRef {
		return false
	}
	coleccionPrimera, _ := datosPrimera.AmbitosHMAC.Datos()
	coleccionSegunda, _ := datosSegunda.AmbitosHMAC.Datos()
	if coleccionPrimera.Activo != coleccionSegunda.Activo ||
		len(coleccionPrimera.Retenidos) != len(coleccionSegunda.Retenidos) {
		return false
	}
	for indice := range coleccionPrimera.Retenidos {
		if coleccionPrimera.Retenidos[indice] !=
			coleccionSegunda.Retenidos[indice] {
			return false
		}
	}
	return true
}
