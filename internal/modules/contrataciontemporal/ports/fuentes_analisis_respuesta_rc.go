package ports

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const esquemaRespuestaValidacionRC = "VEC-CT-FUENTE-ANALISIS-RESPUESTA-RC-V1"

type ResultadoValidacionRC struct {
	datos     *DatosResultadoValidacionRC
	preimagen []byte
}

type DatosResultadoValidacionRC struct {
	PeticionRef           string
	OrganizacionRef       string
	ExpedienteRef         string
	VersionExpediente     uint64
	Validacion            domain.ValidacionRC
	Motivo                MotivoFuenteAnalisis
	Atestacion            AtestacionRespuestaFuenteAnalisis
	HuellaRespuestaSHA256 string
}

func (ResultadoValidacionRC) String() string {
	return "[RESULTADO-VALIDACION-RC-ATESTADO-REDACTADO]"
}

func (r ResultadoValidacionRC) GoString() string { return r.String() }
func (r ResultadoValidacionRC) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, r.String())
}
func (r ResultadoValidacionRC) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func NuevaPreimagenRespuestaValidacionRC(
	solicitud SolicitudValidarRC,
	validacion domain.ValidacionRC,
	motivo MotivoFuenteAnalisis,
	metadatos MetadatosAtestacionRespuestaFuenteAnalisis,
) (PreimagenRespuestaFuenteAnalisis, error) {
	if validarSalidaRCParaRespuesta(
		solicitud,
		validacion,
		motivo,
		metadatos,
	) != nil {
		return PreimagenRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	contenido, err := canonRespuestaValidacionRC(
		solicitud,
		validacion,
		motivo,
		metadatos,
	)
	if err != nil {
		return PreimagenRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return PreimagenRespuestaFuenteAnalisis{contenido: contenido}, nil
}

func NuevoResultadoValidacionRC(
	solicitud SolicitudValidarRC,
	validacion domain.ValidacionRC,
	motivo MotivoFuenteAnalisis,
	atestacion AtestacionRespuestaFuenteAnalisis,
) (ResultadoValidacionRC, error) {
	preimagen, err := NuevaPreimagenRespuestaValidacionRC(
		solicitud,
		validacion,
		motivo,
		atestacion.Metadatos,
	)
	s, errSolicitud := solicitud.Datos()
	huella, errHuella := preimagen.huellaSHA256()
	if err != nil || errSolicitud != nil || errHuella != nil ||
		atestacion.Validar() != nil {
		return ResultadoValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	contenido, _ := preimagen.Bytes()
	return ResultadoValidacionRC{
		datos: &DatosResultadoValidacionRC{
			PeticionRef: s.PeticionRef, OrganizacionRef: s.OrganizacionRef,
			ExpedienteRef: s.ExpedienteRef, VersionExpediente: s.VersionExpediente,
			Validacion: clonarValidacionRC(validacion), Motivo: motivo.clonar(),
			Atestacion: atestacion, HuellaRespuestaSHA256: huella,
		},
		preimagen: contenido,
	}, nil
}

func (r ResultadoValidacionRC) Datos() (DatosResultadoValidacionRC, error) {
	if r.datos == nil {
		return DatosResultadoValidacionRC{}, ErrResultadoFuenteAnalisisNoConfiable
	}
	copia := *r.datos
	copia.Validacion = clonarValidacionRC(copia.Validacion)
	copia.Motivo = copia.Motivo.clonar()
	return copia, nil
}

func (r ResultadoValidacionRC) ValidarPara(
	solicitud SolicitudValidarRC,
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
		validarSalidaRCParaRespuesta(
			solicitud,
			datos.Validacion,
			datos.Motivo,
			datos.Atestacion.Metadatos,
		) != nil {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	canonica, err := canonRespuestaValidacionRC(
		solicitud,
		datos.Validacion,
		datos.Motivo,
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

func (r ResultadoValidacionRC) solicitudVerificacion() SolicitudVerificarRespuestaFuenteAnalisis {
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

func validarSalidaRCParaRespuesta(
	solicitud SolicitudValidarRC,
	validacion domain.ValidacionRC,
	motivo MotivoFuenteAnalisis,
	metadatos MetadatosAtestacionRespuestaFuenteAnalisis,
) error {
	s, err := solicitud.Datos()
	materializada, errMotivo := materializarMotivoValidacionRC(validacion, motivo)
	if err != nil || errMotivo != nil || metadatos.Validar() != nil ||
		validacion.Motivo != "" || materializada.Validar() != nil ||
		!importeFuenteAnalisisValidoOpcional(materializada.Importe) ||
		validacion.EntradaRef != s.Entrada.Referencia ||
		validacion.HuellaEntradaSHA256 != s.Entrada.HuellaSHA256 ||
		validacion.FuenteRef != metadatos.AutoridadRef ||
		validacion.ReciboRef != metadatos.ReciboRef ||
		validacion.ValidadaEn.Before(s.SolicitadaEn) ||
		validacion.ValidadaEn.After(metadatos.EmitidaEn) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func canonRespuestaValidacionRC(
	solicitud SolicitudValidarRC,
	validacion domain.ValidacionRC,
	motivo MotivoFuenteAnalisis,
	metadatos MetadatosAtestacionRespuestaFuenteAnalisis,
) ([]byte, error) {
	s, err := solicitud.Datos()
	if err != nil {
		return nil, ErrResultadoFuenteAnalisisNoConfiable
	}
	canon := nuevoEscritorCanonFuenteAnalisis()
	canon.texto(esquemaRespuestaValidacionRC)
	escribirMetadatosRespuesta(canon, metadatos)
	escribirSolicitudRCRespuesta(canon, s)
	canon.texto(string(validacion.Resultado))
	canon.texto(validacion.EntradaRef)
	canon.texto(validacion.HuellaEntradaSHA256)
	canon.texto(validacion.FuenteRef)
	canon.texto(validacion.ReciboRef)
	canon.instante(validacion.ValidadaEn)
	canon.booleano(validacion.FechaRC != nil)
	if validacion.FechaRC != nil {
		canon.instante(*validacion.FechaRC)
	}
	canon.texto(validacion.Numero)
	canon.booleano(validacion.Importe != nil)
	if validacion.Importe != nil {
		canon.entero64(uint64(validacion.Importe.Centimos))
		canon.texto(validacion.Importe.Moneda)
	}
	canon.texto(validacion.DocumentoRef)
	canon.texto(validacion.Motivo)
	escribirMotivoRespuesta(canon, motivo)
	return canon.resultado()
}

func escribirSolicitudRCRespuesta(
	canon *escritorCanonFuenteAnalisis,
	s DatosSolicitudValidarRC,
) {
	canon.texto(s.PeticionRef)
	canon.texto(s.HuellaPeticionHMAC)
	canon.texto(s.OrganizacionRef)
	canon.texto(s.ExpedienteRef)
	canon.entero64(s.VersionExpediente)
	canon.texto(s.Entrada.Referencia)
	canon.texto(s.Entrada.HuellaSHA256)
	canon.booleano(s.Declaracion.Existe)
	canon.texto(s.Declaracion.Numero)
	canon.instante(s.Declaracion.Fecha)
	canon.entero64(uint64(s.Declaracion.Importe.Centimos))
	canon.texto(s.Declaracion.Importe.Moneda)
	canon.texto(s.Declaracion.DocumentoRef)
	canon.instante(s.SolicitadaEn)
}

func escribirMotivoRespuesta(
	canon *escritorCanonFuenteAnalisis,
	motivo MotivoFuenteAnalisis,
) {
	datos, err := motivo.Datos()
	canon.booleano(err == nil)
	if err != nil {
		return
	}
	canon.texto(datos.CatalogoRef)
	canon.entero64(datos.CatalogoVersion)
	canon.texto(datos.CatalogoHuella)
	canon.texto(string(datos.EntradaClave))
	canon.texto(string(datos.ClaveMensajeI18N))
	canon.entero16(uint16(len(datos.Parametros)))
	for _, parametro := range datos.Parametros {
		canon.texto(string(parametro.Clave))
		canon.texto(string(parametro.Valor))
	}
}

func escribirMetadatosRespuesta(
	canon *escritorCanonFuenteAnalisis,
	metadatos MetadatosAtestacionRespuestaFuenteAnalisis,
) {
	canon.texto(metadatos.AutoridadRef)
	canon.entero64(uint64(metadatos.Generacion))
	canon.texto(metadatos.ReciboRef)
	canon.instante(metadatos.EmitidaEn)
	canon.instante(metadatos.ValidaHasta)
}
