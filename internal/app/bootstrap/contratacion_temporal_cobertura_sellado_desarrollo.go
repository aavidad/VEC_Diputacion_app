package bootstrap

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	dominioHMACAmbitoCoberturaDesarrollo = "vec.contratacion-temporal." +
		"cobertura-decision.ambito"
	dominioHMACSemanticaCoberturaDesarrollo = "vec.contratacion-temporal." +
		"cobertura-decision.semantica"
)

var (
	_ cobertura.SelladorOperacionDecisionCobertura       = (*selladorHMACCoberturaDesarrollo)(nil)
	_ cobertura.SelladorAmbitoOperacionDecisionCobertura = (*selladorHMACCoberturaDesarrollo)(nil)
)

// selladorHMACCoberturaDesarrollo reutiliza el único llavero estable del
// proceso. Los dominios separan cobertura del alta aunque compartan material.
type selladorHMACCoberturaDesarrollo struct {
	derivador *derivadorIdentidadOperacionDesarrollo
}

func nuevoSelladorHMACCoberturaDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
) (*selladorHMACCoberturaDesarrollo, error) {
	if derivador == nil || !derivador.valido() {
		return nil, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return &selladorHMACCoberturaDesarrollo{derivador: derivador}, nil
}

func (s *selladorHMACCoberturaDesarrollo) SellarOperacionDecisionCobertura(
	ctx context.Context,
	preimagenes cobertura.PreimagenesOperacionDecisionCobertura,
) (cobertura.SellosOperacionDecisionCobertura, error) {
	vacio := cobertura.SellosOperacionDecisionCobertura{}
	if err := s.validarContexto(ctx); err != nil {
		return vacio, err
	}
	ambito, err := preimagenes.BytesAmbito()
	if err != nil {
		return vacio, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	defer borrarBytes(ambito)
	semantica, err := preimagenes.BytesSemantica()
	if err != nil {
		return vacio, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	defer borrarBytes(semantica)

	resultados, err := s.derivador.calcularHMAC(ambito, semantica)
	if err != nil {
		return vacio, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	ambitos, err := nuevaColeccionHMACCoberturaDesarrollo(
		resultados,
		dominioHMACAmbitoCoberturaDesarrollo,
		true,
	)
	if err != nil {
		return vacio, err
	}
	semanticas, err := nuevaColeccionHMACCoberturaDesarrollo(
		resultados,
		dominioHMACSemanticaCoberturaDesarrollo,
		false,
	)
	if err != nil {
		return vacio, err
	}
	if err := s.validarContexto(ctx); err != nil {
		return vacio, err
	}
	sellos := cobertura.SellosOperacionDecisionCobertura{
		AmbitosIdempotenciaHMAC: ambitos,
		HuellasSemanticasHMAC:   semanticas,
	}
	if sellos.Validar() != nil {
		return vacio, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return sellos, nil
}

func (s *selladorHMACCoberturaDesarrollo) SellarAmbitoOperacionDecisionCobertura(
	ctx context.Context,
	preimagen cobertura.PreimagenAmbitoRecuperacionOperacionDecisionCobertura,
) (ports.ColeccionSellosHMAC, error) {
	vacio := ports.ColeccionSellosHMAC{}
	if err := s.validarContexto(ctx); err != nil {
		return vacio, err
	}
	ambito, err := preimagen.Bytes()
	if err != nil {
		return vacio, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	defer borrarBytes(ambito)
	resultados, err := s.derivador.calcularHMAC(ambito, ambito)
	if err != nil {
		return vacio, cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	coleccion, err := nuevaColeccionHMACCoberturaDesarrollo(
		resultados,
		dominioHMACAmbitoCoberturaDesarrollo,
		true,
	)
	if err != nil {
		return vacio, err
	}
	if err := s.validarContexto(ctx); err != nil {
		return vacio, err
	}
	return coleccion, nil
}

func (s *selladorHMACCoberturaDesarrollo) validarContexto(ctx context.Context) error {
	if s == nil || s.derivador == nil || !s.derivador.valido() ||
		contextoInterfazNulo(ctx) {
		return cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(
			cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida,
			err,
		)
	}
	return nil
}

func nuevaColeccionHMACCoberturaDesarrollo(
	resultados []resultadoHMACIdempotenciaDesarrollo,
	dominio string,
	usarAmbito bool,
) (ports.ColeccionSellosHMAC, error) {
	if len(resultados) < minimoGeneracionesIdempotenciaDesarrollo ||
		len(resultados) > maximoGeneracionesIdempotenciaDesarrollo {
		return ports.ColeccionSellosHMAC{},
			cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	sellos := make([]string, len(resultados))
	for indice := range resultados {
		resultado := &resultados[indice]
		valor := resultado.huellaSolicitud[:]
		if usarAmbito {
			valor = resultado.localizador[:]
		}
		sellos[indice] = fmt.Sprintf(
			"hmac-sha256:%s/v%d:%s",
			dominio,
			resultado.generacion,
			hex.EncodeToString(valor),
		)
	}
	coleccion, err := ports.NuevaColeccionSellosHMAC(sellos[0], sellos[1:])
	if err != nil {
		return ports.ColeccionSellosHMAC{},
			cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida
	}
	return coleccion, nil
}
