package ports

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const esquemaRespuestaCalculoCoste = "VEC-CT-FUENTE-ANALISIS-RESPUESTA-COSTE-V1"

type ResultadoCalculoCoste struct {
	datos     *DatosResultadoCalculoCoste
	preimagen []byte
}

type DatosResultadoCalculoCoste struct {
	PeticionRef           string
	OrganizacionRef       string
	ExpedienteRef         string
	VersionExpediente     uint64
	FuenteRef             string
	ReciboRef             string
	Importe               domain.Importe
	CalculadoEn           time.Time
	Atestacion            AtestacionRespuestaFuenteAnalisis
	HuellaRespuestaSHA256 string
}

func (ResultadoCalculoCoste) String() string {
	return "[RESULTADO-CALCULO-COSTE-ATESTADO-REDACTADO]"
}

func (r ResultadoCalculoCoste) GoString() string { return r.String() }
func (r ResultadoCalculoCoste) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, r.String())
}
func (r ResultadoCalculoCoste) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func NuevaPreimagenRespuestaCalculoCoste(
	solicitud SolicitudCalcularCoste,
	fuenteRef string,
	reciboRef string,
	importe domain.Importe,
	calculadoEn time.Time,
	metadatos MetadatosAtestacionRespuestaFuenteAnalisis,
) (PreimagenRespuestaFuenteAnalisis, error) {
	if validarSalidaCosteParaRespuesta(
		solicitud,
		fuenteRef,
		reciboRef,
		importe,
		calculadoEn,
		metadatos,
	) != nil {
		return PreimagenRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	contenido, err := canonRespuestaCalculoCoste(
		solicitud,
		fuenteRef,
		reciboRef,
		importe,
		calculadoEn,
		metadatos,
	)
	if err != nil {
		return PreimagenRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return PreimagenRespuestaFuenteAnalisis{contenido: contenido}, nil
}

func NuevoResultadoCalculoCoste(
	solicitud SolicitudCalcularCoste,
	fuenteRef string,
	reciboRef string,
	importe domain.Importe,
	calculadoEn time.Time,
	atestacion AtestacionRespuestaFuenteAnalisis,
) (ResultadoCalculoCoste, error) {
	preimagen, err := NuevaPreimagenRespuestaCalculoCoste(
		solicitud,
		fuenteRef,
		reciboRef,
		importe,
		calculadoEn,
		atestacion.Metadatos,
	)
	s, errSolicitud := solicitud.Datos()
	huella, errHuella := preimagen.huellaSHA256()
	if err != nil || errSolicitud != nil || errHuella != nil ||
		atestacion.Validar() != nil {
		return ResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	contenido, _ := preimagen.Bytes()
	return ResultadoCalculoCoste{
		datos: &DatosResultadoCalculoCoste{
			PeticionRef: s.PeticionRef, OrganizacionRef: s.OrganizacionRef,
			ExpedienteRef: s.ExpedienteRef, VersionExpediente: s.VersionExpediente,
			FuenteRef: fuenteRef, ReciboRef: reciboRef, Importe: importe,
			CalculadoEn: calculadoEn, Atestacion: atestacion,
			HuellaRespuestaSHA256: huella,
		},
		preimagen: contenido,
	}, nil
}

func (r ResultadoCalculoCoste) Datos() (DatosResultadoCalculoCoste, error) {
	if r.datos == nil {
		return DatosResultadoCalculoCoste{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	return *r.datos, nil
}

func (r ResultadoCalculoCoste) ValidarPara(
	solicitud SolicitudCalcularCoste,
	comprobadaEn time.Time,
) error {
	s, errSolicitud := solicitud.Datos()
	datos, errResultado := r.Datos()
	if errSolicitud != nil || errResultado != nil ||
		!instanteFuenteAnalisisCanonico(comprobadaEn) ||
		datos.PeticionRef != s.PeticionRef ||
		datos.OrganizacionRef != s.OrganizacionRef ||
		datos.ExpedienteRef != s.ExpedienteRef ||
		datos.VersionExpediente != s.VersionExpediente ||
		datos.Atestacion.Validar() != nil ||
		comprobadaEn.Before(datos.Atestacion.Metadatos.EmitidaEn) ||
		!comprobadaEn.Before(datos.Atestacion.Metadatos.ValidaHasta) ||
		validarSalidaCosteParaRespuesta(
			solicitud,
			datos.FuenteRef,
			datos.ReciboRef,
			datos.Importe,
			datos.CalculadoEn,
			datos.Atestacion.Metadatos,
		) != nil {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	canonica, err := canonRespuestaCalculoCoste(
		solicitud,
		datos.FuenteRef,
		datos.ReciboRef,
		datos.Importe,
		datos.CalculadoEn,
		datos.Atestacion.Metadatos,
	)
	huella := PreimagenRespuestaFuenteAnalisis{contenido: canonica}
	huellaCanonica, errHuella := huella.huellaSHA256()
	if err != nil || errHuella != nil ||
		!bytes.Equal(canonica, r.preimagen) ||
		datos.HuellaRespuestaSHA256 != huellaCanonica {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func (r ResultadoCalculoCoste) solicitudVerificacion() SolicitudVerificarRespuestaFuenteAnalisis {
	if r.datos == nil {
		return SolicitudVerificarRespuestaFuenteAnalisis{}
	}
	solicitud, _ := nuevaSolicitudVerificarRespuestaFuenteAnalisis(
		PreimagenRespuestaFuenteAnalisis{
			contenido: append([]byte(nil), r.preimagen...),
		},
		r.datos.Atestacion,
	)
	return solicitud
}

func (r ResultadoCalculoCoste) clonar() ResultadoCalculoCoste {
	if r.datos == nil {
		return ResultadoCalculoCoste{}
	}
	datos := *r.datos
	return ResultadoCalculoCoste{
		datos:     &datos,
		preimagen: append([]byte(nil), r.preimagen...),
	}
}

func validarSalidaCosteParaRespuesta(
	solicitud SolicitudCalcularCoste,
	fuenteRef string,
	reciboRef string,
	importe domain.Importe,
	calculadoEn time.Time,
	metadatos MetadatosAtestacionRespuestaFuenteAnalisis,
) error {
	s, err := solicitud.Datos()
	if err != nil || metadatos.Validar() != nil ||
		!domain.ReferenciaOpacaValida(fuenteRef) ||
		fuenteRef != metadatos.AutoridadRef ||
		!domain.ReferenciaOpacaValida(reciboRef) ||
		reciboRef != metadatos.ReciboRef ||
		!importeFuenteAnalisisValido(importe) ||
		!instanteFuenteAnalisisCanonico(calculadoEn) ||
		calculadoEn.Before(s.SolicitadaEn) ||
		calculadoEn.After(metadatos.EmitidaEn) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func canonRespuestaCalculoCoste(
	solicitud SolicitudCalcularCoste,
	fuenteRef string,
	reciboRef string,
	importe domain.Importe,
	calculadoEn time.Time,
	metadatos MetadatosAtestacionRespuestaFuenteAnalisis,
) ([]byte, error) {
	s, err := solicitud.Datos()
	if err != nil {
		return nil, ErrResultadoFuenteAnalisisNoConfiable
	}
	canon := nuevoEscritorCanonFuenteAnalisis()
	canon.texto(esquemaRespuestaCalculoCoste)
	escribirMetadatosRespuesta(canon, metadatos)
	canon.texto(s.PeticionRef)
	canon.texto(s.HuellaPeticionHMAC)
	canon.texto(s.OrganizacionRef)
	canon.texto(s.ExpedienteRef)
	canon.entero64(s.VersionExpediente)
	canon.texto(s.CategoriaRef)
	canon.texto(s.GrupoSubgrupo)
	canon.texto(string(s.ModalidadClave))
	canon.texto(string(s.CausaClave))
	canon.instante(s.Periodo.Inicio)
	canon.instante(s.Periodo.Fin)
	canon.entero16(uint16(s.Jornada))
	canon.instante(s.SolicitadaEn)
	canon.texto(fuenteRef)
	canon.texto(reciboRef)
	canon.entero64(uint64(importe.Centimos))
	canon.texto(importe.Moneda)
	canon.instante(calculadoEn)
	return canon.resultado()
}
