package adapters

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrTiposOriginalFirmableRRHHNoDisponibles = errors.New("contratacion temporal: tipos de original firmable no disponibles")

// CatalogoTiposOriginalFirmableRRHH es la vista necesaria del catálogo de
// conservación de Documentos v2. La reserva separa los originales CT del
// borrador genérico histórico y del tipo reservado a documentos firmados.
type CatalogoTiposOriginalFirmableRRHH interface {
	TipoDocumentalRef(string) (string, error)
	CustodiaOriginalCTReservada(string) bool
}

// TiposOriginalFirmableRRHH congela las seis referencias publicadas al
// componer la fuente. Si falta una o no está reservada, no hay productor CT.
type TiposOriginalFirmableRRHH struct {
	referencias map[ports.TipoBorradorRRHH]string
}

var _ ports.TiposOriginalFirmableRRHH = (*TiposOriginalFirmableRRHH)(nil)
var _ vecports.ResolutorTipoOriginalCT = (*TiposOriginalFirmableRRHH)(nil)

var clavesOriginalFirmableRRHH = map[ports.TipoBorradorRRHH]string{
	ports.BorradorInformeDefinitivo:  "contratacion_temporal.borrador.informe_definitivo.v1",
	ports.BorradorResolucion:         "contratacion_temporal.borrador.resolucion.v1",
	ports.BorradorDiligencia:         "contratacion_temporal.borrador.diligencia.v1",
	ports.BorradorTomaPosesion:       "contratacion_temporal.borrador.toma_posesion.v1",
	ports.BorradorNotificacion:       "contratacion_temporal.borrador.notificacion.v1",
	ports.BorradorComunicacionCentro: "contratacion_temporal.borrador.comunicacion_centro.v1",
}

func NuevosTiposOriginalFirmableRRHH(catalogo CatalogoTiposOriginalFirmableRRHH) (*TiposOriginalFirmableRRHH, error) {
	if catalogo == nil || (reflect.ValueOf(catalogo).Kind() == reflect.Pointer && reflect.ValueOf(catalogo).IsNil()) {
		return nil, ErrTiposOriginalFirmableRRHHNoDisponibles
	}
	referencias := make(map[ports.TipoBorradorRRHH]string, len(clavesOriginalFirmableRRHH))
	vistas := make(map[string]struct{}, len(clavesOriginalFirmableRRHH))
	for tipo, clave := range clavesOriginalFirmableRRHH {
		referencia, err := catalogo.TipoDocumentalRef(clave)
		if err != nil || !almacencanonico.ReferenciaDocumentoValida(referencia) ||
			!catalogo.CustodiaOriginalCTReservada(referencia) {
			return nil, ErrTiposOriginalFirmableRRHHNoDisponibles
		}
		if _, duplicada := vistas[referencia]; duplicada {
			return nil, ErrTiposOriginalFirmableRRHHNoDisponibles
		}
		referencias[tipo] = referencia
		vistas[referencia] = struct{}{}
	}
	return &TiposOriginalFirmableRRHH{referencias: referencias}, nil
}

func (t *TiposOriginalFirmableRRHH) ResolverTipoOriginalRRHH(ctx context.Context, tipo ports.TipoBorradorRRHH) (string, error) {
	if ctx == nil || t == nil {
		return "", ErrTiposOriginalFirmableRRHHNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	referencia := t.referencias[tipo]
	if !almacencanonico.ReferenciaDocumentoValida(referencia) {
		return "", ErrTiposOriginalFirmableRRHHNoDisponibles
	}
	return referencia, nil
}

// ResolverTipoOriginalCT reutiliza la misma instantánea gobernada de seis
// tipos que consume la fuente RRHH; no interpreta cargos ni datos del cliente.
func (t *TiposOriginalFirmableRRHH) ResolverTipoOriginalCT(ctx context.Context, documento string) (string, error) {
	return t.ResolverTipoOriginalRRHH(ctx, ports.TipoBorradorRRHH(documento))
}
