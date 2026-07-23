package ports

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const dominioResultadoDurableCobertura = "VEC-CT-EFECTO-COBERTURA-V1"

// DatosOrdenConsumoCobertura liga el efecto durable a la respuesta completa,
// a su verificación independiente y al catálogo gobernado usado para decidir.
type DatosOrdenConsumoCobertura struct {
	PeticionRef             string
	OrganizacionRef         string
	ExpedienteRef           string
	VersionExpediente       uint64
	HuellaPeticionSHA256    string
	HuellaResultadoSHA256   string
	AutoridadRef            string
	Generacion              uint32
	ReciboRespuestaRef      string
	HuellaRespuestaSHA256   string
	Atestacion              AtestacionRespuestaCobertura
	ConfirmacionRespuesta   ConfirmacionRespuestaCobertura
	ConfirmacionCatalogo    ConfirmacionPublicacionCobertura
	ClaveVerificadorEd25519 []byte
}

type OrdenConsumoCobertura struct {
	datos     *DatosOrdenConsumoCobertura
	solicitud SolicitudConsultarCobertura
	resultado ResultadoConsultaCobertura
}

func NuevaOrdenConsumoCobertura(
	solicitud SolicitudConsultarCobertura,
	resultado ResultadoConsultaCobertura,
	confirmacion ConfirmacionRespuestaCobertura,
	confirmacionCatalogo ConfirmacionPublicacionCobertura,
	claveVerificadorEd25519 ed25519.PublicKey,
) (OrdenConsumoCobertura, error) {
	if resultado.ValidarPara(solicitud) != nil {
		return OrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	huellaRespuesta, errHuellaRespuesta :=
		resultado.preimagen.huellaSHA256()
	huellaPeticion, errHuellaPeticion :=
		huellaPeticionCobertura(solicitud)
	huellaResultado, errHuellaResultado :=
		huellaResultadoDurableCobertura(solicitud, resultado)
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	datos := DatosOrdenConsumoCobertura{
		PeticionRef:           solicitud.PeticionRef,
		OrganizacionRef:       solicitud.OrganizacionRef,
		ExpedienteRef:         solicitud.ExpedienteRef,
		VersionExpediente:     solicitud.VersionExpediente,
		HuellaPeticionSHA256:  huellaPeticion,
		HuellaResultadoSHA256: huellaResultado,
		AutoridadRef:          resultado.atestacion.Metadatos.AutoridadRef,
		Generacion:            resultado.atestacion.Metadatos.Generacion,
		ReciboRespuestaRef:    resultado.atestacion.Metadatos.ReciboRef,
		HuellaRespuestaSHA256: huellaRespuesta,
		Atestacion:            resultado.atestacion,
		ConfirmacionRespuesta: confirmacion,
		ConfirmacionCatalogo:  confirmacionCatalogo,
		ClaveVerificadorEd25519: append(
			[]byte(nil),
			claveVerificadorEd25519...,
		),
	}
	if errHuellaRespuesta != nil || errHuellaPeticion != nil ||
		errHuellaResultado != nil || errConfirmacion != nil ||
		datosConfirmacion.HuellaMaterialSHA256 != huellaRespuesta ||
		validarOrdenConsumoCobertura(
			datos,
			solicitud,
			resultado,
			datosConfirmacion.VerificadaEn,
		) != nil {
		return OrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return OrdenConsumoCobertura{
		datos:     &datos,
		solicitud: solicitud,
		resultado: resultado,
	}, nil
}

func huellaResultadoDurableCobertura(
	solicitud SolicitudConsultarCobertura,
	resultado ResultadoConsultaCobertura,
) (string, error) {
	datos, err := resultado.Datos()
	huellaPeticion, errHuella := huellaPeticionCobertura(solicitud)
	if err != nil || errHuella != nil ||
		resultado.ValidarPara(solicitud) != nil {
		return "", ErrResultadoFuenteCoberturaNoConfiable
	}
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escritor.texto(dominioResultadoDurableCobertura)
	escritor.texto(huellaPeticion)
	escritor.texto(string(datos.Comprobacion.Clave))
	escritor.texto(string(datos.Comprobacion.Resultado))
	escritor.texto(datos.DefinicionFuenteRef)
	contenido, err := escritor.resultado()
	if err != nil {
		return "", ErrResultadoFuenteCoberturaNoConfiable
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

func validarOrdenConsumoCobertura(
	datos DatosOrdenConsumoCobertura,
	solicitud SolicitudConsultarCobertura,
	resultado ResultadoConsultaCobertura,
	comprobadaEn time.Time,
) error {
	huellaPeticion, errHuellaPeticion :=
		huellaPeticionCobertura(solicitud)
	huellaResultado, errHuellaResultado :=
		huellaResultadoDurableCobertura(solicitud, resultado)
	if solicitud.Validar() != nil || resultado.ValidarPara(solicitud) != nil ||
		errHuellaPeticion != nil || errHuellaResultado != nil ||
		!domain.ReferenciaOpacaValida(datos.PeticionRef) ||
		datos.PeticionRef != solicitud.PeticionRef ||
		datos.OrganizacionRef != solicitud.OrganizacionRef ||
		datos.ExpedienteRef != solicitud.ExpedienteRef ||
		datos.VersionExpediente != solicitud.VersionExpediente ||
		datos.VersionExpediente > maximoEnteroSeguroFuenteAnalisis ||
		datos.HuellaPeticionSHA256 != huellaPeticion ||
		datos.HuellaResultadoSHA256 != huellaResultado ||
		!huellaSHA256FuenteAnalisisValida(datos.HuellaPeticionSHA256) ||
		!huellaSHA256FuenteAnalisisValida(datos.HuellaResultadoSHA256) ||
		datos.AutoridadRef != resultado.atestacion.Metadatos.AutoridadRef ||
		datos.Generacion != resultado.atestacion.Metadatos.Generacion ||
		datos.ReciboRespuestaRef !=
			resultado.atestacion.Metadatos.ReciboRef ||
		!huellaSHA256FuenteAnalisisValida(datos.HuellaRespuestaSHA256) ||
		len(datos.ClaveVerificadorEd25519) != ed25519.PublicKeySize ||
		datos.Atestacion.Validar() != nil {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	datosResultado, errDatosResultado := resultado.Datos()
	solicitudVerificacion, err := nuevaSolicitudVerificarRespuestaCobertura(
		datosResultado.HuellaPeticionSHA256,
		resultado.preimagen,
		datos.Atestacion,
	)
	if errDatosResultado != nil || err != nil ||
		datos.ConfirmacionRespuesta.ValidarPara(
			solicitudVerificacion,
			comprobadaEn,
			ed25519.PublicKey(datos.ClaveVerificadorEd25519),
		) != nil ||
		datos.ConfirmacionCatalogo.ValidarPara(
			solicitud,
			comprobadaEn,
		) != nil {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (o OrdenConsumoCobertura) Datos() (
	DatosOrdenConsumoCobertura,
	error,
) {
	if o.datos == nil {
		return DatosOrdenConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	datos := *o.datos
	datos.ClaveVerificadorEd25519 = append(
		[]byte(nil),
		o.datos.ClaveVerificadorEd25519...,
	)
	return datos, nil
}

func (OrdenConsumoCobertura) String() string {
	return "[ORDEN-CONSUMO-COBERTURA-REDACTADA]"
}

func (o OrdenConsumoCobertura) GoString() string { return o.String() }
func (o OrdenConsumoCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, o.String())
}
func (o OrdenConsumoCobertura) LogValue() slog.Value {
	return slog.StringValue(o.String())
}

type ReciboConsumoCobertura struct {
	ConsumoRef            string
	PeticionRef           string
	OrganizacionRef       string
	HuellaPeticionSHA256  string
	HuellaResultadoSHA256 string
	AutoridadRef          string
	Generacion            uint32
	ReciboRespuestaRef    string
	HuellaRespuestaSHA256 string
	ConsumidaEn           time.Time
	SolicitudOriginal     SolicitudConsultarCobertura
	ResultadoOriginal     ResultadoConsultaCobertura
	ConfirmacionOriginal  ConfirmacionRespuestaCobertura
}

func NuevoReciboConsumoCobertura(
	orden OrdenConsumoCobertura,
	consumoRef string,
	consumidaEn time.Time,
) (ReciboConsumoCobertura, error) {
	datos, err := orden.Datos()
	recibo := ReciboConsumoCobertura{
		ConsumoRef:            consumoRef,
		PeticionRef:           datos.PeticionRef,
		OrganizacionRef:       datos.OrganizacionRef,
		HuellaPeticionSHA256:  datos.HuellaPeticionSHA256,
		HuellaResultadoSHA256: datos.HuellaResultadoSHA256,
		AutoridadRef:          datos.AutoridadRef,
		Generacion:            datos.Generacion,
		ReciboRespuestaRef:    datos.ReciboRespuestaRef,
		HuellaRespuestaSHA256: datos.HuellaRespuestaSHA256,
		ConsumidaEn:           consumidaEn,
		SolicitudOriginal:     orden.solicitud,
		ResultadoOriginal:     orden.resultado,
		ConfirmacionOriginal:  datos.ConfirmacionRespuesta,
	}
	if err != nil || recibo.ValidarPara(orden) != nil {
		return ReciboConsumoCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return recibo, nil
}

func (r ReciboConsumoCobertura) ValidarPara(
	orden OrdenConsumoCobertura,
) error {
	datos, err := orden.Datos()
	confirmacionActual, errActual := datos.ConfirmacionRespuesta.Datos()
	confirmacionOriginal, errOriginal := r.ConfirmacionOriginal.Datos()
	datosResultadoOriginal, errResultadoOriginal :=
		r.ResultadoOriginal.Datos()
	atestacionOriginal, errAtestacionOriginal :=
		r.ResultadoOriginal.Atestacion()
	solicitudVerificacionOriginal, errSolicitudOriginal :=
		r.ResultadoOriginal.SolicitudVerificacion()
	huellaPeticionOriginal, errHuellaPeticionOriginal :=
		huellaPeticionCobertura(r.SolicitudOriginal)
	huellaResultadoOriginal, errHuellaResultadoOriginal :=
		huellaResultadoDurableCobertura(
			r.SolicitudOriginal,
			r.ResultadoOriginal,
		)
	huellaRespuestaOriginal, errHuellaRespuestaOriginal :=
		r.ResultadoOriginal.preimagen.huellaSHA256()
	if err != nil ||
		errActual != nil || errOriginal != nil ||
		errResultadoOriginal != nil || errAtestacionOriginal != nil ||
		errSolicitudOriginal != nil || errHuellaPeticionOriginal != nil ||
		errHuellaResultadoOriginal != nil ||
		errHuellaRespuestaOriginal != nil ||
		!domain.ReferenciaOpacaValida(r.ConsumoRef) ||
		r.PeticionRef != datos.PeticionRef ||
		r.OrganizacionRef != datos.OrganizacionRef ||
		r.HuellaPeticionSHA256 != datos.HuellaPeticionSHA256 ||
		r.HuellaResultadoSHA256 != datos.HuellaResultadoSHA256 ||
		r.HuellaPeticionSHA256 != huellaPeticionOriginal ||
		r.HuellaResultadoSHA256 != huellaResultadoOriginal ||
		r.AutoridadRef != atestacionOriginal.Metadatos.AutoridadRef ||
		r.Generacion != atestacionOriginal.Metadatos.Generacion ||
		r.ReciboRespuestaRef != atestacionOriginal.Metadatos.ReciboRef ||
		r.HuellaRespuestaSHA256 != huellaRespuestaOriginal ||
		!instanteFuenteAnalisisCanonico(r.ConsumidaEn) ||
		r.ConsumidaEn.Before(atestacionOriginal.Metadatos.EmitidaEn) ||
		!r.ConsumidaEn.Before(atestacionOriginal.Metadatos.ValidaHasta) ||
		confirmacionOriginal.VerificadorRef !=
			confirmacionActual.VerificadorRef ||
		confirmacionOriginal.HuellaMaterialSHA256 !=
			r.HuellaRespuestaSHA256 ||
		confirmacionOriginal.HuellaPeticionSHA256 !=
			r.HuellaPeticionSHA256 ||
		datosResultadoOriginal.HuellaPeticionSHA256 !=
			r.HuellaPeticionSHA256 ||
		r.ConfirmacionOriginal.ValidarPara(
			solicitudVerificacionOriginal,
			r.ConsumidaEn,
			ed25519.PublicKey(datos.ClaveVerificadorEd25519),
		) != nil {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (ReciboConsumoCobertura) String() string {
	return "[RECIBO-CONSUMO-COBERTURA-REDACTADO]"
}

func (r ReciboConsumoCobertura) GoString() string { return r.String() }
func (r ReciboConsumoCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, r.String())
}
func (r ReciboConsumoCobertura) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

// ConsumidorCobertura debe imponer unicidad durable por
// (organización, petición). La misma petición y resultado semántico devuelve
// el recibo original aunque se renueven recibos o confirmaciones probatorias.
// Otra semántica o resultado para esa petición devuelve
// ErrRespuestaCoberturaYaConsumida. Además, una evidencia
// (autoridad, generación, recibo) no puede ligarse a otra respuesta o petición.
// El puerto no afirma que exista un adaptador productivo.
type ConsumidorCobertura interface {
	ConsumirCobertura(
		context.Context,
		OrdenConsumoCobertura,
	) (ReciboConsumoCobertura, error)
}
