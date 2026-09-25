package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

var ErrDocumentoComisionNoDisponible = errors.New("dietas: documento de comision no disponible")
var ErrInstantaneaDocumentoComisionInvalida = errors.New("dietas: instantanea documental de comision invalida")

// InstantaneaDocumentoComision conserva el borrador durable exacto que se
// representara. No contiene una orden de servicio, aprobacion ni liquidacion.
// Repeticion del recibo se omite: es una propiedad de la consulta, no del hecho.
type InstantaneaDocumentoComision struct {
	Referencia   string
	Version      uint64
	ReciboRef    string
	RegistradoEn time.Time
	Borrador     domain.ComisionBorrador
}

func (i InstantaneaDocumentoComision) Validar() error {
	if i.Borrador.Validar() != nil || i.Referencia != i.Borrador.Referencia ||
		i.Version == 0 || i.ReciboRef == "" || i.RegistradoEn.IsZero() ||
		i.RegistradoEn.Location() != time.UTC {
		return ErrInstantaneaDocumentoComisionInvalida
	}
	return nil
}

func (i InstantaneaDocumentoComision) Clonar() InstantaneaDocumentoComision {
	i.Borrador.CodigosRuta = append([]string(nil), i.Borrador.CodigosRuta...)
	if i.Borrador.Calculo != nil {
		calculo := *i.Borrador.Calculo
		calculo.TramosRuta = append([]domain.TramoRutaComision(nil), calculo.TramosRuta...)
		calculo.OpcionesDieta = append([]domain.OpcionDietaComision(nil), calculo.OpcionesDieta...)
		for n := range calculo.OpcionesDieta {
			calculo.OpcionesDieta[n].Calculo.Tramos = append([]domain.TramoDietaProvisional(nil), calculo.OpcionesDieta[n].Calculo.Tramos...)
		}
		i.Borrador.Calculo = &calculo
	}
	return i
}

type RenderizadorDocumentoComision interface {
	Renderizar(context.Context, InstantaneaDocumentoComision) ([]byte, error)
}

// TextosDocumentoComision se conecta al catalogo i18n comun. La ausencia de
// una clave impide generar un PDF con textos mezclados o sin traducir.
type TextosDocumentoComision interface {
	Texto(context.Context, string) (string, error)
	FechaCivil(context.Context, string) (string, error)
	FechaHora(context.Context, time.Time) (string, error)
}
