package seguridad

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const bytesAleatoriosReferenciaAlta = 32

var ErrGeneracionReferenciaAlta = errors.New(
	"contratacion temporal: generacion de referencia no disponible",
)

type GeneradorReferenciasAltaCriptografico struct {
	lector io.Reader
	ahora  func() time.Time
}

func NuevoGeneradorReferenciasAltaCriptografico() *GeneradorReferenciasAltaCriptografico {
	return &GeneradorReferenciasAltaCriptografico{
		lector: rand.Reader,
		ahora:  func() time.Time { return time.Now().UTC() },
	}
}

func (g *GeneradorReferenciasAltaCriptografico) GenerarReferenciasAlta(
	ctx context.Context,
) (ports.ReferenciasAlta, error) {
	if !generadorValido(g) || ctx == nil {
		return ports.ReferenciasAlta{}, ErrGeneracionReferenciaAlta
	}
	if err := ctx.Err(); err != nil {
		return ports.ReferenciasAlta{}, err
	}
	expediente, err := g.generar(ctx, "expediente:ct:")
	if err != nil {
		return ports.ReferenciasAlta{}, err
	}
	recibo, err := g.generar(ctx, "recibo:ct-alta:")
	if err != nil {
		return ports.ReferenciasAlta{}, err
	}
	visible, err := g.numeroVisible(ctx)
	if err != nil {
		return ports.ReferenciasAlta{}, err
	}
	referencias := ports.ReferenciasAlta{
		ExpedienteRef: expediente,
		NumeroVisible: visible,
		ReciboRef:     recibo,
	}
	if referencias.Validar() != nil {
		return ports.ReferenciasAlta{}, ErrGeneracionReferenciaAlta
	}
	return referencias, nil
}

func (g *GeneradorReferenciasAltaCriptografico) NuevaReferenciaReservaAlta(
	ctx context.Context,
) (string, error) {
	if !generadorValido(g) || ctx == nil {
		return "", ErrGeneracionReferenciaAlta
	}
	return g.generar(ctx, "reserva:ct-alta:")
}

func (g *GeneradorReferenciasAltaCriptografico) NuevaReferenciaComprobacionCobertura(
	ctx context.Context,
) (string, error) {
	if !generadorValido(g) || ctx == nil {
		return "", ErrGeneracionReferenciaAlta
	}
	return g.generar(ctx, "peticion:ct-cobertura:")
}

func (g *GeneradorReferenciasAltaCriptografico) GenerarReferenciasAsignacion(
	ctx context.Context,
) (ports.ReferenciasEfectoAsignacion, error) {
	if !generadorValido(g) || ctx == nil {
		return ports.ReferenciasEfectoAsignacion{}, ErrGeneracionReferenciaAlta
	}
	if err := ctx.Err(); err != nil {
		return ports.ReferenciasEfectoAsignacion{}, err
	}
	prefijos := [...]string{
		"reserva:ct-asignacion:",
		"recibo:ct-asignacion:",
		"notificacion:ct-asignacion:",
		"bandeja:ct-asignacion:",
		"auditoria:ct-asignacion:",
		"evento:ct-asignacion:",
	}
	valores := make([]string, len(prefijos))
	for indice, prefijo := range prefijos {
		referencia, err := g.generar(ctx, prefijo)
		if err != nil {
			return ports.ReferenciasEfectoAsignacion{}, err
		}
		valores[indice] = referencia
	}
	referencias := ports.ReferenciasEfectoAsignacion{
		ReservaRef:      valores[0],
		ReciboRef:       valores[1],
		NotificacionRef: valores[2],
		BandejaRef:      valores[3],
		AuditoriaRef:    valores[4],
		EventoRef:       valores[5],
	}
	if referencias.Validar() != nil {
		return ports.ReferenciasEfectoAsignacion{}, ErrGeneracionReferenciaAlta
	}
	return referencias, nil
}

func (g *GeneradorReferenciasAltaCriptografico) GenerarReferenciasInformeJuridico(
	ctx context.Context,
) (ports.ReferenciasEfectoInformeJuridico, error) {
	if !generadorValido(g) || ctx == nil {
		return ports.ReferenciasEfectoInformeJuridico{}, ErrGeneracionReferenciaAlta
	}
	if err := ctx.Err(); err != nil {
		return ports.ReferenciasEfectoInformeJuridico{}, err
	}
	prefijos := [...]string{
		"reserva:ct-informe-juridico:",
		"informe:ct:",
		"documento:ct-informe-juridico:",
		"recibo:ct-informe-juridico:",
		"auditoria:ct-informe-juridico:",
		"evento:ct-informe-juridico:",
	}
	valores := make([]string, len(prefijos))
	for indice, prefijo := range prefijos {
		referencia, err := g.generar(ctx, prefijo)
		if err != nil {
			return ports.ReferenciasEfectoInformeJuridico{}, err
		}
		valores[indice] = referencia
	}
	referencias := ports.ReferenciasEfectoInformeJuridico{
		ReservaRef:   valores[0],
		InformeRef:   valores[1],
		DocumentoRef: valores[2],
		ReciboRef:    valores[3],
		AuditoriaRef: valores[4],
		EventoRef:    valores[5],
	}
	if referencias.Validar() != nil {
		return ports.ReferenciasEfectoInformeJuridico{}, ErrGeneracionReferenciaAlta
	}
	return referencias, nil
}

func (g *GeneradorReferenciasAltaCriptografico) GenerarReferenciasFiscalizacion(
	ctx context.Context,
	resultado domain.ResultadoFiscalizacion,
) (ports.ReferenciasEfectoFiscalizacion, error) {
	if !generadorValido(g) || ctx == nil || !resultado.Valido() {
		return ports.ReferenciasEfectoFiscalizacion{}, ErrGeneracionReferenciaAlta
	}
	if err := ctx.Err(); err != nil {
		return ports.ReferenciasEfectoFiscalizacion{}, err
	}
	prefijos := [...]string{
		"reserva:ct-fiscalizacion:",
		"fiscalizacion:ct:",
		"recibo:ct-fiscalizacion:",
		"evento:ct-fiscalizacion:",
	}
	valores := make([]string, len(prefijos))
	for indice, prefijo := range prefijos {
		referencia, err := g.generar(ctx, prefijo)
		if err != nil {
			return ports.ReferenciasEfectoFiscalizacion{}, err
		}
		valores[indice] = referencia
	}
	referencias := ports.ReferenciasEfectoFiscalizacion{
		ReservaRef:       valores[0],
		FiscalizacionRef: valores[1],
		ReciboRef:        valores[2],
		EventoRef:        valores[3],
	}
	if resultado == domain.FiscalizacionDesfavorable {
		retorno, err := g.generar(ctx, "retorno:ct-fiscalizacion:")
		if err != nil {
			return ports.ReferenciasEfectoFiscalizacion{}, err
		}
		referencias.RetornoRef = retorno
	}
	if referencias.ValidarPara(resultado) != nil {
		return ports.ReferenciasEfectoFiscalizacion{}, ErrGeneracionReferenciaAlta
	}
	return referencias, nil
}

func (g *GeneradorReferenciasAltaCriptografico) numeroVisible(
	ctx context.Context,
) (string, error) {
	instante := g.ahora().UTC()
	if instante.Year() < 1 || instante.Year() > 9999 {
		return "", ErrGeneracionReferenciaAlta
	}
	aleatorio, err := g.entropia(ctx)
	if err != nil {
		return "", err
	}
	defer borrar(aleatorio)
	return fmt.Sprintf(
		"%04d/CT-%s",
		instante.Year(),
		hex.EncodeToString(aleatorio[:16]),
	), nil
}

func (g *GeneradorReferenciasAltaCriptografico) generar(
	ctx context.Context,
	prefijo string,
) (string, error) {
	aleatorio, err := g.entropia(ctx)
	if err != nil {
		return "", err
	}
	defer borrar(aleatorio)
	return prefijo + hex.EncodeToString(aleatorio), nil
}

func (g *GeneradorReferenciasAltaCriptografico) entropia(
	ctx context.Context,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	aleatorio := make([]byte, bytesAleatoriosReferenciaAlta)
	if _, err := io.ReadFull(g.lector, aleatorio); err != nil {
		borrar(aleatorio)
		return nil, ErrGeneracionReferenciaAlta
	}
	if err := ctx.Err(); err != nil {
		borrar(aleatorio)
		return nil, err
	}
	return aleatorio, nil
}

func generadorValido(g *GeneradorReferenciasAltaCriptografico) bool {
	return g != nil && g.lector != nil && g.ahora != nil
}

var _ ports.GeneradorReferenciasAlta = (*GeneradorReferenciasAltaCriptografico)(nil)
var _ ports.GeneradorReferenciasAsignacion = (*GeneradorReferenciasAltaCriptografico)(nil)
var _ ports.GeneradorReferenciasInformeJuridico = (*GeneradorReferenciasAltaCriptografico)(nil)
