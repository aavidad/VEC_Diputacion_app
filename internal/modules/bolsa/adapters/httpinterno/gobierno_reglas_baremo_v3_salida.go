package httpinterno

import (
	"encoding/json"
	"time"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type reciboGobiernoHTTPV3 struct {
	ReciboRef             string                            `json:"recibo_ref"`
	ConfirmadaEn          string                            `json:"confirmada_en"`
	ClaveOperacion        string                            `json:"clave_operacion"`
	HuellaSolicitudSHA256 string                            `json:"huella_solicitud_sha256"`
	Selector              selectorGobiernoHTTPV3            `json:"selector"`
	Estado                reglas.EstadoGobiernoReglasBaremo `json:"estado"`
	Disponibilidad        string                            `json:"disponibilidad"`
}

func selectorSalidaGobiernoHTTPV3(v reglas.VersionGobernadaReglasBaremo) (selectorGobiernoHTTPV3, error) {
	var vacio selectorGobiernoHTTPV3
	conjunto, err := v.Conjunto()
	if err != nil {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	if v.Estado() != reglas.EstadoReglasBaremoBorrador || v.Revision() != 1 {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	estado, err := v.VinculoEstado()
	if err != nil {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	id := conjunto.Identidad()
	return selectorGobiernoHTTPV3{id.Referencia(), id.Version(), id.ConvocatoriaRef(), id.ExpedienteRef(), estado.Contenido().HuellaSHA256(), estado.Revision(), estado.HuellaEstadoSHA256()}, nil
}

func reciboSalidaGobiernoHTTPV3(r ports.ReciboAltaBorradorReglasV3) (reciboGobiernoHTTPV3, error) {
	var vacio reciboGobiernoHTTPV3
	v, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(r.VersionCanonica, r.Estado.HuellaEstadoSHA256())
	if err != nil || r.ReciboRef == "" || r.ConfirmadaEn.IsZero() {
		return vacio, ports.ErrConfirmacionReglasBaremoInvalida
	}
	s, err := selectorSalidaGobiernoHTTPV3(v)
	if err != nil {
		return vacio, err
	}
	return reciboGobiernoHTTPV3{r.ReciboRef, r.ConfirmadaEn.UTC().Format(time.RFC3339Nano), r.ClaveOperacion, r.HuellaSolicitudSHA256, s, v.Estado(), "disponible_para_preparacion"}, nil
}

func salidaAltaGobiernoReglasV3(r ports.ResultadoAltaBorradorReglasV3) (any, error) {
	recibo, err := reciboSalidaGobiernoHTTPV3(r.Recibo)
	if err != nil {
		return nil, err
	}
	return struct {
		Recibo reciboGobiernoHTTPV3 `json:"recibo"`
		Replay bool                 `json:"replay"`
	}{recibo, r.Replay}, nil
}

func salidaConsultaGobiernoReglasV3(r ports.ResultadoConsultaGobiernoReglasV3, esperado ports.SelectorGobiernoReglasV3) (any, error) {
	v, err := reglas.RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(r.VersionCanonica, esperado.Estado.HuellaEstadoSHA256())
	if err != nil {
		return nil, ports.ErrConfirmacionReglasBaremoInvalida
	}
	s, err := selectorSalidaGobiernoHTTPV3(v)
	if err != nil {
		return nil, err
	}
	conjunto, err := v.Conjunto()
	if err != nil {
		return nil, ports.ErrConfirmacionReglasBaremoInvalida
	}
	canon, err := conjunto.RepresentacionCanonica()
	if err != nil {
		return nil, ports.ErrConfirmacionReglasBaremoInvalida
	}
	return struct {
		Reglas         json.RawMessage                   `json:"reglas"`
		Selector       selectorGobiernoHTTPV3            `json:"selector"`
		Estado         reglas.EstadoGobiernoReglasBaremo `json:"estado"`
		Disponibilidad string                            `json:"disponibilidad"`
	}{canon, s, v.Estado(), "disponible_para_preparacion"}, nil
}

func salidaRecuperacionGobiernoReglasV3(r ports.ResultadoRecuperacionGobiernoReglasV3) (any, error) {
	var recibo *reciboGobiernoHTTPV3
	if r.Existe {
		valor, err := reciboSalidaGobiernoHTTPV3(r.Recibo)
		if err != nil {
			return nil, err
		}
		recibo = &valor
	}
	return struct {
		Existe bool                  `json:"existe"`
		Recibo *reciboGobiernoHTTPV3 `json:"recibo,omitempty"`
	}{r.Existe, recibo}, nil
}
