package bootstrap

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrSolicitudTSADesarrolloInvalida = errors.New("bootstrap: solicitud TSA de desarrollo invalida")

type selladorTiempoDesarrollo struct {
	clave [sha256.Size]byte
}

func nuevoSelladorTiempoDesarrollo(clave [sha256.Size]byte) *selladorTiempoDesarrollo {
	return &selladorTiempoDesarrollo{clave: clave}
}

func (s *selladorTiempoDesarrollo) Timestamp(
	ctx context.Context,
	solicitud vecports.InteropRequest,
) (vecports.InteropResult, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || !solicitudTSAValida(solicitud) {
		return vecports.InteropResult{}, ErrSolicitudTSADesarrolloInvalida
	}
	preimagen, err := json.Marshal(struct {
		Dominio   string            `json:"dominio"`
		Autoridad string            `json:"autoridad"`
		Operacion string            `json:"operacion"`
		Sujeto    string            `json:"sujeto,omitempty"`
		Carga     map[string]string `json:"carga,omitempty"`
	}{
		Dominio: "vec.tsa.desarrollo.v1", Autoridad: AutoridadNoAutoritativa,
		Operacion: solicitud.Operation, Sujeto: solicitud.Subject, Carga: solicitud.Payload,
	})
	if err != nil {
		return vecports.InteropResult{}, ErrSolicitudTSADesarrolloInvalida
	}
	mac := hmac.New(sha256.New, s.clave[:])
	_, _ = mac.Write(preimagen)
	sello := hex.EncodeToString(mac.Sum(nil))
	return vecports.InteropResult{
		Operation: solicitud.Operation,
		Reference: "tsa-desarrollo:hmac-sha256:" + sello,
		Status:    AutoridadNoAutoritativa,
		Payload: map[string]string{
			"esquema": "vec.tsa.desarrollo.v1", "autoridad": AutoridadNoAutoritativa,
			"migrable_a_produccion": "false", "huella_preimagen_sha256": huellaSHA256Texto(preimagen),
		},
	}, nil
}

func solicitudTSAValida(solicitud vecports.InteropRequest) bool {
	if !textoTSAValido(solicitud.Operation, 1, 128) ||
		(solicitud.Subject != "" && !textoTSAValido(solicitud.Subject, 1, 512)) || len(solicitud.Payload) > 64 {
		return false
	}
	for clave, valor := range solicitud.Payload {
		if !textoTSAValido(clave, 1, 128) || !textoTSAValido(valor, 0, 4096) {
			return false
		}
	}
	return true
}

func textoTSAValido(valor string, minimo, maximo int) bool {
	if len(valor) < minimo || len(valor) > maximo || valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) {
		return false
	}
	return !strings.ContainsFunc(valor, func(caracter rune) bool {
		return unicode.IsControl(caracter) || unicode.Is(unicode.Cf, caracter)
	})
}

func huellaSHA256Texto(contenido []byte) string {
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:])
}

var _ vecports.TimestampPort = (*selladorTiempoDesarrollo)(nil)
