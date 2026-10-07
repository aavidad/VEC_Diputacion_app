package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// ConsultaPoliticaFinAnalisisConfirmada sólo contiene el ámbito sellado y
// coordenadas resueltas desde la identidad autenticada. No transporta datos
// funcionales ni una política propuesta por el cliente.
type ConsultaPoliticaFinAnalisisConfirmada struct {
	AmbitosHMAC       ColeccionSellosHMAC
	OrganizacionRef   string
	ExpedienteRef     string
	ActorRef          string
	PerfilRef         string
	Operacion         TipoOperacionAnalisis
	VersionExpediente uint64
}

// AmbitoPoliticaFinAnalisis genera la misma preimagen de ámbito publicada
// por el análisis. La segunda preimagen sólo satisface el sellador existente;
// su resultado semántico se descarta y nunca se usa para consultar recibos.
func AmbitoPoliticaFinAnalisis(
	clave, organizacion, expediente, actor, perfil string,
) (PreimagenesOperacionAnalisis, error) {
	if !ClaveIdempotenciaValida(clave) ||
		!domain.ReferenciaOpacaValida(organizacion) ||
		!domain.ReferenciaOpacaValida(expediente) ||
		!domain.ReferenciaOpacaValida(actor) ||
		!domain.ReferenciaOpacaValida(perfil) {
		return PreimagenesOperacionAnalisis{}, ErrOperacionAnalisisInvalida
	}
	canon := nuevoCanonOperacionAnalisis()
	escribirAmbitoOperacionAnalisis(canon, clave, organizacion, expediente, actor, perfil)
	ambito, err := canon.resultado()
	if err != nil {
		return PreimagenesOperacionAnalisis{}, err
	}
	return PreimagenesOperacionAnalisis{ambito: ambito, semantica: append([]byte(nil), ambito...)}, nil
}

func (c ConsultaPoliticaFinAnalisisConfirmada) Validar() error {
	if c.AmbitosHMAC.ValidarDominio(dominioAmbitoOperacionAnalisis) != nil ||
		!domain.ReferenciaOpacaValida(c.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(c.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(c.ActorRef) ||
		!domain.ReferenciaOpacaValida(c.PerfilRef) ||
		!c.Operacion.Valida() ||
		!VersionOperacionAnalisisConIncrementoValida(c.VersionExpediente) {
		return ErrPreparacionOperacionAnalisisInvalida
	}
	return nil
}

type ConsultaPoliticaFinAltaConfirmada struct {
	AmbitosHMAC     ColeccionSellosHMAC
	OrganizacionRef string
	ActorRef        string
	PerfilRef       string
}

func (c ConsultaPoliticaFinAltaConfirmada) Validar() error {
	if c.AmbitosHMAC.ValidarDominio("vec.contratacion-temporal.ambito-idempotencia") != nil ||
		!domain.ReferenciaOpacaValida(c.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(c.ActorRef) ||
		!domain.ReferenciaOpacaValida(c.PerfilRef) {
		return ErrPreparacionAltaInvalida
	}
	return nil
}

// El booleano distingue una confirmación antigua sin instantánea (política
// vacía) de una operación que aún no se confirmó. El consumidor siempre
// recalcula la huella completa antes de recuperar el recibo.
type RecuperadorPoliticaFinConfirmada interface {
	ConsultarPoliticaFinAnalisisConfirmada(context.Context, ConsultaPoliticaFinAnalisisConfirmada) (domain.PoliticaFin, bool, error)
	ConsultarPoliticaFinAltaConfirmada(context.Context, ConsultaPoliticaFinAltaConfirmada) (domain.PoliticaFin, bool, error)
}
