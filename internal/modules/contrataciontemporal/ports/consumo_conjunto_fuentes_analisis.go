package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const esquemaConsumoConjuntoFuentesAnalisisO3 = "VEC-CT-CONSUMO-CONJUNTO-FUENTES-O3-V1"

var ErrConjuntoFuentesAnalisisYaConsumido = errors.New(
	"contratacion temporal: conjunto de fuentes ya consumido con otros datos",
)

// OrdenConsumoConjuntoFuentesAnalisisO3 es la unidad indivisible de consumo.
// Un adaptador no puede confirmar RC y coste mediante llamadas separadas.
type OrdenConsumoConjuntoFuentesAnalisisO3 struct {
	bloqueoSerializacionOperacionAnalisis
	datos *datosOrdenConsumoConjuntoFuentesAnalisisO3
}

type datosOrdenConsumoConjuntoFuentesAnalisisO3 struct {
	artefactoRef      string
	organizacionRef   string
	expedienteRef     string
	versionExpediente uint64
	ordenRC           OrdenConsumoRespuestaFuenteAnalisis
	ordenCoste        *OrdenConsumoRespuestaFuenteAnalisis
	huellaSHA256      string
}

type DatosOrdenConsumoConjuntoFuentesAnalisisO3 struct {
	bloqueoSerializacionOperacionAnalisis
	ArtefactoRef      string
	OrganizacionRef   string
	ExpedienteRef     string
	VersionExpediente uint64
	OrdenRC           OrdenConsumoRespuestaFuenteAnalisis
	OrdenCoste        *OrdenConsumoRespuestaFuenteAnalisis
	HuellaSHA256      string
}

func nuevaOrdenConsumoConjuntoFuentesAnalisisO3(
	solicitud SolicitudPrepararArtefactoAnalisis,
	rc EvidenciaValidacionRCVerificadaO3,
	coste EvidenciaCalculoCosteVerificadaO3,
) (OrdenConsumoConjuntoFuentesAnalisisO3, error) {
	if solicitud.Validar() != nil || rc.datos == nil {
		return OrdenConsumoConjuntoFuentesAnalisisO3{},
			ErrArtefactoAnalisisNoConfiable
	}
	datos := DatosOrdenConsumoConjuntoFuentesAnalisisO3{
		ArtefactoRef:      solicitud.ArtefactoRef,
		OrganizacionRef:   solicitud.OrganizacionRef,
		ExpedienteRef:     solicitud.ExpedienteRef,
		VersionExpediente: solicitud.VersionExpediente,
		OrdenRC:           rc.datos.orden,
	}
	if coste.datos != nil {
		orden := coste.datos.orden
		datos.OrdenCoste = &orden
	}
	huella, err := huellaOrdenConsumoConjuntoFuentesAnalisisO3(datos)
	if err != nil {
		return OrdenConsumoConjuntoFuentesAnalisisO3{},
			ErrArtefactoAnalisisNoConfiable
	}
	datos.HuellaSHA256 = huella
	if validarDatosOrdenConsumoConjuntoFuentesAnalisisO3(datos) != nil {
		return OrdenConsumoConjuntoFuentesAnalisisO3{},
			ErrArtefactoAnalisisNoConfiable
	}
	return ordenConsumoConjuntoDesdeDatosO3(datos), nil
}

func (o OrdenConsumoConjuntoFuentesAnalisisO3) Datos() (
	DatosOrdenConsumoConjuntoFuentesAnalisisO3,
	error,
) {
	if o.datos == nil {
		return DatosOrdenConsumoConjuntoFuentesAnalisisO3{},
			ErrArtefactoAnalisisNoConfiable
	}
	datos := datosPublicosOrdenConsumoConjuntoO3(*o.datos)
	if validarDatosOrdenConsumoConjuntoFuentesAnalisisO3(datos) != nil {
		return DatosOrdenConsumoConjuntoFuentesAnalisisO3{},
			ErrArtefactoAnalisisNoConfiable
	}
	return datos, nil
}

func ordenConsumoConjuntoDesdeDatosO3(
	datos DatosOrdenConsumoConjuntoFuentesAnalisisO3,
) OrdenConsumoConjuntoFuentesAnalisisO3 {
	privados := &datosOrdenConsumoConjuntoFuentesAnalisisO3{
		artefactoRef:      datos.ArtefactoRef,
		organizacionRef:   datos.OrganizacionRef,
		expedienteRef:     datos.ExpedienteRef,
		versionExpediente: datos.VersionExpediente,
		ordenRC:           datos.OrdenRC,
		huellaSHA256:      datos.HuellaSHA256,
	}
	if datos.OrdenCoste != nil {
		orden := *datos.OrdenCoste
		privados.ordenCoste = &orden
	}
	return OrdenConsumoConjuntoFuentesAnalisisO3{datos: privados}
}

func datosPublicosOrdenConsumoConjuntoO3(
	datos datosOrdenConsumoConjuntoFuentesAnalisisO3,
) DatosOrdenConsumoConjuntoFuentesAnalisisO3 {
	salida := DatosOrdenConsumoConjuntoFuentesAnalisisO3{
		ArtefactoRef:      datos.artefactoRef,
		OrganizacionRef:   datos.organizacionRef,
		ExpedienteRef:     datos.expedienteRef,
		VersionExpediente: datos.versionExpediente,
		OrdenRC:           datos.ordenRC,
		HuellaSHA256:      datos.huellaSHA256,
	}
	if datos.ordenCoste != nil {
		orden := *datos.ordenCoste
		salida.OrdenCoste = &orden
	}
	return salida
}

func validarDatosOrdenConsumoConjuntoFuentesAnalisisO3(
	datos DatosOrdenConsumoConjuntoFuentesAnalisisO3,
) error {
	rc, errRC := datos.OrdenRC.Datos()
	if !domain.ReferenciaOpacaValida(datos.ArtefactoRef) ||
		!domain.ReferenciaOpacaValida(datos.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(datos.ExpedienteRef) ||
		!VersionOperacionAnalisisConIncrementoValida(
			datos.VersionExpediente,
		) ||
		errRC != nil || rc.Tipo != RespuestaValidacionRC ||
		rc.OrganizacionRef != datos.OrganizacionRef ||
		rc.ExpedienteRef != datos.ExpedienteRef ||
		rc.VersionExpediente != datos.VersionExpediente ||
		!huellaSHA256OperacionAnalisisValida(datos.HuellaSHA256) {
		return ErrArtefactoAnalisisNoConfiable
	}
	if datos.OrdenCoste != nil {
		coste, errCoste := datos.OrdenCoste.Datos()
		if errCoste != nil || coste.Tipo != RespuestaCalculoCoste ||
			coste.OrganizacionRef != datos.OrganizacionRef ||
			coste.ExpedienteRef != datos.ExpedienteRef ||
			coste.VersionExpediente != datos.VersionExpediente {
			return ErrArtefactoAnalisisNoConfiable
		}
	}
	copia := datos
	copia.HuellaSHA256 = ""
	huella, err := huellaOrdenConsumoConjuntoFuentesAnalisisO3(copia)
	if err != nil || huella != datos.HuellaSHA256 {
		return ErrArtefactoAnalisisNoConfiable
	}
	return nil
}

func huellaOrdenConsumoConjuntoFuentesAnalisisO3(
	datos DatosOrdenConsumoConjuntoFuentesAnalisisO3,
) (string, error) {
	if datos.HuellaSHA256 != "" {
		return "", ErrArtefactoAnalisisNoConfiable
	}
	canon := nuevoCanonOperacionAnalisis()
	canon.texto(esquemaConsumoConjuntoFuentesAnalisisO3)
	canon.texto(datos.ArtefactoRef)
	canon.texto(datos.OrganizacionRef)
	canon.texto(datos.ExpedienteRef)
	canon.enteroSinSigno(datos.VersionExpediente)
	escribirOrdenRespuestaFuenteAnalisisO3(canon, datos.OrdenRC)
	canon.booleano(datos.OrdenCoste != nil)
	if datos.OrdenCoste != nil {
		escribirOrdenRespuestaFuenteAnalisisO3(canon, *datos.OrdenCoste)
	}
	contenido, err := canon.resultado()
	if err != nil {
		return "", ErrArtefactoAnalisisNoConfiable
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

func escribirOrdenRespuestaFuenteAnalisisO3(
	canon *canonOperacionAnalisis,
	orden OrdenConsumoRespuestaFuenteAnalisis,
) {
	datos, err := orden.Datos()
	confirmacion, errConfirmacion := datos.ConfirmacionRespuesta.Datos()
	if err != nil || errConfirmacion != nil {
		canon.err = ErrArtefactoAnalisisNoConfiable
		return
	}
	canon.texto(string(datos.Tipo))
	canon.texto(datos.PeticionRef)
	canon.texto(datos.OrganizacionRef)
	canon.texto(datos.ExpedienteRef)
	canon.enteroSinSigno(datos.VersionExpediente)
	canon.texto(datos.HuellaRespuestaSHA256)
	canon.texto(datos.Atestacion.Metadatos.AutoridadRef)
	canon.enteroSinSigno(uint64(datos.Atestacion.Metadatos.Generacion))
	canon.texto(datos.Atestacion.Metadatos.ReciboRef)
	canon.texto(datos.Atestacion.SelloHMAC)
	canon.instante(datos.Atestacion.Metadatos.EmitidaEn)
	canon.instante(datos.Atestacion.Metadatos.ValidaHasta)
	canon.texto(confirmacion.VerificadorRef)
	canon.texto(confirmacion.HuellaMaterialSHA256)
	canon.instante(confirmacion.VerificadaEn)
	canon.booleano(datos.ConfirmacionPublicacion != nil)
	if datos.ConfirmacionPublicacion != nil {
		publicacion, errPublicacion :=
			datos.ConfirmacionPublicacion.Datos()
		if errPublicacion != nil {
			canon.err = ErrArtefactoAnalisisNoConfiable
			return
		}
		canon.texto(publicacion.PublicadorRef)
		canon.texto(publicacion.PublicacionRef)
		canon.texto(publicacion.ReciboVerificacionRef)
		canon.texto(publicacion.HuellaSolicitudSHA256)
		canon.instante(publicacion.VerificadaEn)
	}
}

type ReciboConsumoConjuntoFuentesAnalisisO3 struct {
	bloqueoSerializacionOperacionAnalisis
	ConsumoConjuntoRef string
	ArtefactoRef       string
	HuellaConjunto     string
	ReciboRC           ReciboConsumoRespuestaFuenteAnalisis
	ReciboCoste        *ReciboConsumoRespuestaFuenteAnalisis
	ConsumidaEn        time.Time
}

func NuevoReciboConsumoConjuntoFuentesAnalisisO3(
	orden OrdenConsumoConjuntoFuentesAnalisisO3,
	consumoConjuntoRef string,
	reciboRC ReciboConsumoRespuestaFuenteAnalisis,
	reciboCoste *ReciboConsumoRespuestaFuenteAnalisis,
	consumidaEn time.Time,
) (ReciboConsumoConjuntoFuentesAnalisisO3, error) {
	datos, err := orden.Datos()
	recibo := ReciboConsumoConjuntoFuentesAnalisisO3{
		ConsumoConjuntoRef: consumoConjuntoRef,
		ArtefactoRef:       datos.ArtefactoRef,
		HuellaConjunto:     datos.HuellaSHA256,
		ReciboRC:           reciboRC,
		ConsumidaEn:        consumidaEn,
	}
	if reciboCoste != nil {
		copia := *reciboCoste
		recibo.ReciboCoste = &copia
	}
	if err != nil || recibo.ValidarPara(orden) != nil {
		return ReciboConsumoConjuntoFuentesAnalisisO3{},
			ErrArtefactoAnalisisNoConfiable
	}
	return recibo, nil
}

func (r ReciboConsumoConjuntoFuentesAnalisisO3) ValidarPara(
	orden OrdenConsumoConjuntoFuentesAnalisisO3,
) error {
	datos, err := orden.Datos()
	if err != nil ||
		!domain.ReferenciaOpacaValida(r.ConsumoConjuntoRef) ||
		r.ArtefactoRef != datos.ArtefactoRef ||
		r.HuellaConjunto != datos.HuellaSHA256 ||
		!instanteSeguroOperacionAnalisis(r.ConsumidaEn) ||
		r.ReciboRC.ValidarPara(datos.OrdenRC) != nil ||
		!r.ReciboRC.ConsumidaEn.Equal(r.ConsumidaEn) ||
		(datos.OrdenCoste == nil) != (r.ReciboCoste == nil) {
		return ErrArtefactoAnalisisNoConfiable
	}
	if datos.OrdenCoste != nil &&
		(r.ReciboCoste.ValidarPara(*datos.OrdenCoste) != nil ||
			!r.ReciboCoste.ConsumidaEn.Equal(r.ConsumidaEn)) {
		return ErrArtefactoAnalisisNoConfiable
	}
	return nil
}

func clonarReciboConsumoConjuntoFuentesAnalisisO3(
	recibo ReciboConsumoConjuntoFuentesAnalisisO3,
) ReciboConsumoConjuntoFuentesAnalisisO3 {
	if recibo.ReciboCoste != nil {
		coste := *recibo.ReciboCoste
		recibo.ReciboCoste = &coste
	}
	return recibo
}
