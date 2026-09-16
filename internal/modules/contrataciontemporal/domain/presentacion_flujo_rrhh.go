package domain

import "strings"

const EsquemaPresentacionFlujoRRHH = "vec.contratacion_temporal.presentacion_flujo_rrhh.v1"

const maximoVersionPresentacionFlujoRRHHJSON = 9_007_199_254_740_991

var ErrPresentacionFlujoRRHHInvalida = ErrDatoInvalido

type FasePresentacionFlujoRRHH struct {
	Clave     ClaveFase `json:"clave"`
	Orden     uint16    `json:"orden"`
	ClaveI18n string    `json:"clave_i18n"`
}

type PresentacionFlujoRRHH struct {
	Esquema    string                      `json:"esquema"`
	Referencia string                      `json:"referencia"`
	Version    uint64                      `json:"version"`
	ClaveI18n  string                      `json:"clave_i18n"`
	Fases      []FasePresentacionFlujoRRHH `json:"fases"`
	FaseActual ClaveFase                   `json:"fase_actual,omitempty"`
}

func (p PresentacionFlujoRRHH) Validar() error {
	if p.Esquema != EsquemaPresentacionFlujoRRHH || !referenciaValida(p.Referencia) ||
		p.Version < 1 || p.Version > maximoVersionPresentacionFlujoRRHHJSON ||
		!claveI18nPresentacionValida(p.ClaveI18n) || len(p.Fases) < 1 || len(p.Fases) > 32 {
		return ErrPresentacionFlujoRRHHInvalida
	}
	claves, ordenes := map[ClaveFase]struct{}{}, map[uint16]struct{}{}
	for _, fase := range p.Fases {
		if !fase.Clave.Valida() || fase.Orden < 1 || !claveI18nPresentacionValida(fase.ClaveI18n) {
			return ErrPresentacionFlujoRRHHInvalida
		}
		if _, ok := claves[fase.Clave]; ok {
			return ErrPresentacionFlujoRRHHInvalida
		}
		if _, ok := ordenes[fase.Orden]; ok {
			return ErrPresentacionFlujoRRHHInvalida
		}
		claves[fase.Clave], ordenes[fase.Orden] = struct{}{}, struct{}{}
	}
	if p.FaseActual != "" {
		if _, ok := claves[p.FaseActual]; !ok {
			return ErrPresentacionFlujoRRHHInvalida
		}
	}
	return nil
}

func claveI18nPresentacionValida(v string) bool {
	return v == strings.TrimSpace(v) && patronClave.MatchString(v) &&
		strings.ContainsRune(v, '.')
}
