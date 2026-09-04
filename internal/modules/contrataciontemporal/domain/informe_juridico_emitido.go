package domain

import (
	"errors"
	"time"
)

const (
	AccionEmitirInformeJuridico ClaveCatalogo = "contratacion_temporal.informe_juridico.generar"
	FaseInformeJuridico         ClaveFase     = "informe_juridico"
)

var ErrInformeJuridicoEmitidoInvalido = errors.New(
	"contratacion temporal: informe juridico emitido invalido",
)

// InformeJuridicoEmitido conserva la proyeccion juridica y la identidad del
// documento. El contenido pertenece al adaptador documental.
type InformeJuridicoEmitido struct {
	Borrador              EstadoBorradorInformeJuridico    `json:"borrador"`
	InformeRef            string                           `json:"informe_ref"`
	DocumentoRef          string                           `json:"documento_ref"`
	VersionDocumento      uint64                           `json:"version_documento"`
	HuellaDocumentoSHA256 string                           `json:"huella_documento_sha256"`
	EmitidoEn             time.Time                        `json:"emitido_en"`
	ActuacionRegistro     *VinculoActuacionInformeJuridico `json:"actuacion_registro"`
}

type VinculoActuacionInformeJuridico struct {
	Secuencia             uint64        `json:"secuencia"`
	VersionExpediente     uint64        `json:"version_expediente"`
	AccionClave           ClaveCatalogo `json:"accion_clave"`
	FaseDestino           ClaveFase     `json:"fase_destino"`
	ReciboRef             string        `json:"recibo_ref"`
	InformeRef            string        `json:"informe_ref"`
	DocumentoRef          string        `json:"documento_ref"`
	VersionDocumento      uint64        `json:"version_documento"`
	HuellaDocumentoSHA256 string        `json:"huella_documento_sha256"`
	HuellaBorradorSHA256  string        `json:"huella_borrador_sha256"`
}

func (i InformeJuridicoEmitido) Validar() error {
	borrador, err := i.validarEntrada()
	if err != nil || i.ActuacionRegistro == nil ||
		i.ActuacionRegistro.validar() != nil ||
		i.ActuacionRegistro.InformeRef != i.InformeRef ||
		i.ActuacionRegistro.DocumentoRef != i.DocumentoRef ||
		i.ActuacionRegistro.VersionDocumento != i.VersionDocumento ||
		i.ActuacionRegistro.HuellaDocumentoSHA256 != i.HuellaDocumentoSHA256 ||
		i.ActuacionRegistro.HuellaBorradorSHA256 != borrador.HuellaSHA256() {
		return ErrInformeJuridicoEmitidoInvalido
	}
	return nil
}

func (i InformeJuridicoEmitido) validarEntrada() (BorradorInformeJuridico, error) {
	borrador, err := RestaurarBorradorInformeJuridico(i.Borrador)
	if err != nil || !referenciaValida(i.InformeRef) ||
		!referenciaValida(i.DocumentoRef) ||
		!versionInformeJuridicoValida(i.VersionDocumento) ||
		!huellaValida(i.HuellaDocumentoSHA256) ||
		!instanteCanonico(i.EmitidoEn) {
		return BorradorInformeJuridico{}, ErrInformeJuridicoEmitidoInvalido
	}
	return borrador, nil
}

func (v VinculoActuacionInformeJuridico) validar() error {
	if v.Secuencia == 0 || v.VersionExpediente == 0 ||
		v.Secuencia != v.VersionExpediente ||
		v.AccionClave != AccionEmitirInformeJuridico ||
		v.FaseDestino != FaseInformeJuridico ||
		!referenciaValida(v.ReciboRef) || !referenciaValida(v.InformeRef) ||
		!referenciaValida(v.DocumentoRef) ||
		!versionInformeJuridicoValida(v.VersionDocumento) ||
		!huellaValida(v.HuellaDocumentoSHA256) ||
		!huellaValida(v.HuellaBorradorSHA256) {
		return ErrInformeJuridicoEmitidoInvalido
	}
	return nil
}

func nuevoVinculoActuacionInformeJuridico(
	version uint64,
	secuencia uint64,
	actuacion DatosActuacion,
	informe InformeJuridicoEmitido,
) VinculoActuacionInformeJuridico {
	return VinculoActuacionInformeJuridico{
		Secuencia: secuencia, VersionExpediente: version,
		AccionClave: actuacion.AccionClave, FaseDestino: actuacion.FaseDestino,
		ReciboRef: actuacion.ReciboRef, InformeRef: informe.InformeRef,
		DocumentoRef: informe.DocumentoRef, VersionDocumento: informe.VersionDocumento,
		HuellaDocumentoSHA256: informe.HuellaDocumentoSHA256,
		HuellaBorradorSHA256:  informe.Borrador.HuellaSHA256,
	}
}

func (v VinculoActuacionInformeJuridico) correspondeA(
	actuacion Actuacion,
	informe InformeJuridicoEmitido,
) bool {
	return v.validar() == nil && v.Secuencia == actuacion.Secuencia &&
		v.VersionExpediente == actuacion.VersionExpediente &&
		v.AccionClave == actuacion.AccionClave &&
		v.FaseDestino == actuacion.FaseDestino && v.ReciboRef == actuacion.ReciboRef &&
		len(actuacion.DocumentosRef) == 1 && actuacion.DocumentosRef[0] == v.DocumentoRef &&
		v.InformeRef == informe.InformeRef && v.DocumentoRef == informe.DocumentoRef &&
		v.VersionDocumento == informe.VersionDocumento &&
		v.HuellaDocumentoSHA256 == informe.HuellaDocumentoSHA256 &&
		v.HuellaBorradorSHA256 == informe.Borrador.HuellaSHA256
}

func (i InformeJuridicoEmitido) clonar() InformeJuridicoEmitido {
	i.Borrador.ReferenciasNormativas = append(
		[]ReferenciaNormativaInformeJuridico(nil), i.Borrador.ReferenciasNormativas...,
	)
	i.Borrador.Anexos = append(
		[]AnexoDocumentalInformeJuridico(nil), i.Borrador.Anexos...,
	)
	if i.ActuacionRegistro != nil {
		vinculo := *i.ActuacionRegistro
		i.ActuacionRegistro = &vinculo
	}
	return i
}

func (e Expediente) RegistrarInformeJuridico(
	versionEsperada uint64,
	informe InformeJuridicoEmitido,
	actuacion DatosActuacion,
) (Expediente, error) {
	borrador, err := informe.validarEntrada()
	if e.Validar() != nil || err != nil ||
		actuacion.validar() != nil || e.Asignacion == nil || e.InformeJuridico != nil ||
		e.FaseActual != ClaveFase("asignacion_unidad") || e.EstadoActual != EstadoEnCurso ||
		borrador.Estado().ExpedienteRef != e.Referencia ||
		borrador.Estado().VersionEsperadaExpediente != e.Version ||
		!informe.EmitidoEn.Equal(actuacion.RealizadaEn) ||
		actuacion.AccionClave != AccionEmitirInformeJuridico ||
		actuacion.UnidadRef != e.Asignacion.UnidadRef ||
		actuacion.FaseDestino != FaseInformeJuridico ||
		actuacion.EstadoDestino != EstadoEnCurso || len(actuacion.DocumentosRef) != 1 ||
		actuacion.DocumentosRef[0] != informe.DocumentoRef ||
		informe.ActuacionRegistro != nil {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	clon := informe.clonar()
	vinculo := nuevoVinculoActuacionInformeJuridico(
		e.Version+1, uint64(len(e.Actuaciones)+1), actuacion, informe,
	)
	clon.ActuacionRegistro = &vinculo
	siguiente.InformeJuridico = &clon
	return siguiente.confirmarTransicion(actuacion)
}
