package ports

import (
	"context"
	"errors"
)

// ErrVerificacionMotivadaIncoherente indica que un adaptador ha producido un
// estado y un motivo que no se corresponden. Nunca se interpreta como firma.
var ErrVerificacionMotivadaIncoherente = errors.New("documentos: verificacion de firma motivada incoherente")

// MotivoVerificacionFirma es el catalogo cerrado que explica un resultado de
// verificacion. No transporta texto del proveedor: la evidencia detallada se
// conserva, si procede, bajo el regimen propio del servicio verificador.
type MotivoVerificacionFirma string

const (
	// MotivoFirmaVerificada solo acompana a EstadoVerificacionValida.
	MotivoFirmaVerificada MotivoVerificacionFirma = "verificada"

	// Motivos de EstadoVerificacionNoValida: el verificador concluye un defecto.
	MotivoIntegridadNoValida  MotivoVerificacionFirma = "integridad_no_valida"
	MotivoCertificadoNoValido MotivoVerificacionFirma = "certificado_no_valido"
	MotivoConfianzaNoValida   MotivoVerificacionFirma = "confianza_no_valida"

	// Motivos de EstadoVerificacionIndeterminada: falta una comprobacion
	// necesaria o no pudo concluirse. Ninguno habilita la transicion a firmado.
	MotivoVinculoOriginalNoAcreditado MotivoVerificacionFirma = "vinculo_original_no_acreditado"
	MotivoFirmanteNoIdentificado      MotivoVerificacionFirma = "firmante_no_identificado"
	MotivoCertificadoNoAcreditado     MotivoVerificacionFirma = "certificado_no_acreditado"
	MotivoConfianzaNoAcreditada       MotivoVerificacionFirma = "confianza_no_acreditada"
	MotivoRevocacionNoAcreditada      MotivoVerificacionFirma = "revocacion_no_acreditada"
	MotivoSelloTiempoNoAcreditado     MotivoVerificacionFirma = "sello_tiempo_no_acreditado"
	MotivoRechazadaPorValidador       MotivoVerificacionFirma = "rechazada_por_validador"
	MotivoCredencialRechazada         MotivoVerificacionFirma = "credencial_rechazada"
	MotivoValidadorNoDisponible       MotivoVerificacionFirma = "validador_no_disponible"
	MotivoRespuestaNoInterpretable    MotivoVerificacionFirma = "respuesta_no_interpretable"
)

// EstadoAsociado devuelve el unico estado compatible con el motivo, o una
// cadena vacia si el motivo no pertenece al catalogo.
func (m MotivoVerificacionFirma) EstadoAsociado() EstadoVerificacionFirma {
	switch m {
	case MotivoFirmaVerificada:
		return EstadoVerificacionValida
	case MotivoIntegridadNoValida, MotivoCertificadoNoValido, MotivoConfianzaNoValida:
		return EstadoVerificacionNoValida
	case MotivoVinculoOriginalNoAcreditado, MotivoFirmanteNoIdentificado,
		MotivoCertificadoNoAcreditado, MotivoConfianzaNoAcreditada,
		MotivoRevocacionNoAcreditada, MotivoSelloTiempoNoAcreditado,
		MotivoRechazadaPorValidador, MotivoCredencialRechazada,
		MotivoValidadorNoDisponible, MotivoRespuestaNoInterpretable:
		return EstadoVerificacionIndeterminada
	default:
		return ""
	}
}

// VerificacionFirmaMotivada acompana el resultado del puerto con su motivo.
type VerificacionFirmaMotivada struct {
	Resultado ResultadoVerificacionFirma
	Motivo    MotivoVerificacionFirma
}

// ValidarContra exige que estado y motivo casen y, si el estado es valido,
// que el resultado supere la comprobacion estricta del puerto.
func (v VerificacionFirmaMotivada) ValidarContra(s SolicitudVerificacionFirma) error {
	estado := v.Motivo.EstadoAsociado()
	if estado == "" || estado != v.Resultado.Estado {
		return ErrVerificacionMotivadaIncoherente
	}
	if estado == EstadoVerificacionValida && v.Resultado.ValidarContra(s) != nil {
		return ErrVerificacionMotivadaIncoherente
	}
	return nil
}

// VerificadorFirmaMotivado es el puerto neutral que devuelve siempre un
// resultado tipado. La indisponibilidad del proveedor se expresa como
// EstadoVerificacionIndeterminada con MotivoValidadorNoDisponible; solo una
// solicitud invalida produce error. Ningun resultado distinto de
// EstadoVerificacionValida con MotivoFirmaVerificada acredita firma.
type VerificadorFirmaMotivado interface {
	VerificarMotivado(context.Context, SolicitudVerificacionFirma) (VerificacionFirmaMotivada, error)
}
