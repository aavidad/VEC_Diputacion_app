package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

const EsquemaPresentacionFlujoRRHH = "vec.contratacion_temporal.presentacion_flujo_rrhh.v1"

const maximoVersionPresentacionFlujoRRHHJSON = 9_007_199_254_740_991

var ErrPresentacionFlujoRRHHInvalida = ErrDatoInvalido

type FasePresentacionFlujoRRHH struct {
	Clave     ClaveFase `json:"clave"`
	Orden     uint16    `json:"orden"`
	ClaveI18n string    `json:"clave_i18n"`
}

type PresentacionFlujoRRHH struct {
	Esquema       string                      `json:"esquema"`
	Referencia    string                      `json:"referencia"`
	Version       uint64                      `json:"version"`
	Huella        string                      `json:"huella_sha256"`
	VinculoHuella string                      `json:"vinculo_huella_sha256"`
	ClaveI18n     string                      `json:"clave_i18n"`
	Fases         []FasePresentacionFlujoRRHH `json:"fases"`
	FaseActual    ClaveFase                   `json:"fase_actual,omitempty"`
}

func (p PresentacionFlujoRRHH) Validar() error {
	if p.Esquema != EsquemaPresentacionFlujoRRHH || !referenciaValida(p.Referencia) ||
		p.Version < 1 || p.Version > maximoVersionPresentacionFlujoRRHHJSON ||
		!huellaValida(p.Huella) || !huellaValida(p.VinculoHuella) ||
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
	huella, err := CalcularHuellaPresentacionFlujoRRHH(p)
	if err != nil || huella != p.Huella {
		return ErrPresentacionFlujoRRHHInvalida
	}
	return nil
}

func CalcularHuellaPresentacionFlujoRRHH(p PresentacionFlujoRRHH) (string, error) {
	if p.Esquema != EsquemaPresentacionFlujoRRHH || !referenciaValida(p.Referencia) ||
		p.Version < 1 || p.Version > maximoVersionPresentacionFlujoRRHHJSON ||
		!huellaValida(p.VinculoHuella) || !claveI18nPresentacionValida(p.ClaveI18n) ||
		len(p.Fases) < 1 || len(p.Fases) > 32 {
		return "", ErrPresentacionFlujoRRHHInvalida
	}
	copia := p
	copia.Huella, copia.FaseActual = "", ""
	copia.Fases = append([]FasePresentacionFlujoRRHH(nil), p.Fases...)
	sort.Slice(copia.Fases, func(i, j int) bool { return copia.Fases[i].Orden < copia.Fases[j].Orden })
	b, err := json.Marshal(copia)
	if err != nil {
		return "", ErrPresentacionFlujoRRHHInvalida
	}
	s := sha256.Sum256(append([]byte("VEC-CT-PRESENTACION-FLUJO-RRHH-V1\x00"), b...))
	return hex.EncodeToString(s[:]), nil
}

func claveI18nPresentacionValida(v string) bool {
	return v == strings.TrimSpace(v) && patronClave.MatchString(v) &&
		strings.ContainsRune(v, '.')
}
