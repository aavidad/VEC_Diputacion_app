package seguridad

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"vec-diputacion-granada/internal/vec/ports"
)

var ErrHMACSubidaLocalNoDisponible = errors.New("seguridad: HMAC de subida local no disponible")

func RaicesHMACSubidaLocalDistintas(correlacion, seudonimo, solicitud []byte) bool {
	if len(correlacion) != 32 || len(seudonimo) != 32 || len(solicitud) != 32 {
		return false
	}
	cero := make([]byte, 32)
	return subtle.ConstantTimeCompare(correlacion, cero) == 0 &&
		subtle.ConstantTimeCompare(seudonimo, cero) == 0 &&
		subtle.ConstantTimeCompare(solicitud, cero) == 0 &&
		subtle.ConstantTimeCompare(correlacion, seudonimo) == 0 &&
		subtle.ConstantTimeCompare(correlacion, solicitud) == 0 &&
		subtle.ConstantTimeCompare(seudonimo, solicitud) == 0
}

type SeudonimizadorSujetoSubidaLocal struct{ clave [32]byte }
type SelladorSolicitudSubidaLocal struct{ clave [32]byte }

func NuevoSeudonimizadorSujetoSubidaLocal(raiz []byte) (*SeudonimizadorSujetoSubidaLocal, error) {
	if len(raiz) != 32 || subtle.ConstantTimeCompare(raiz, make([]byte, 32)) == 1 {
		return nil, ErrHMACSubidaLocalNoDisponible
	}
	s := &SeudonimizadorSujetoSubidaLocal{}
	copy(s.clave[:], raiz)
	return s, nil
}

func NuevoSelladorSolicitudSubidaLocal(raiz []byte) (*SelladorSolicitudSubidaLocal, error) {
	if len(raiz) != 32 || subtle.ConstantTimeCompare(raiz, make([]byte, 32)) == 1 {
		return nil, ErrHMACSubidaLocalNoDisponible
	}
	s := &SelladorSolicitudSubidaLocal{}
	copy(s.clave[:], raiz)
	return s, nil
}

func (s *SeudonimizadorSujetoSubidaLocal) SeudonimizarSujetoAlmacen(ctx context.Context,
	q ports.SolicitudSeudonimizarSujetoAlmacen) (string, error) {
	if s == nil || ctx == nil || ctx.Err() != nil {
		return "", ErrHMACSubidaLocalNoDisponible
	}
	sujeto, ambito, err := q.RevelarParaSellado()
	if err != nil || len(sujeto) == 0 || len(ambito) == 0 || len(sujeto) > 512 || len(ambito) > 512 {
		return "", ErrHMACSubidaLocalNoDisponible
	}
	mac := hmac.New(sha256.New, s.clave[:])
	_, _ = mac.Write([]byte("vec.documentos.subida.local.seudonimo.v1\x00"))
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len(sujeto)))
	_, _ = mac.Write(longitud[:])
	_, _ = mac.Write([]byte(sujeto))
	binary.BigEndian.PutUint64(longitud[:], uint64(len(ambito)))
	_, _ = mac.Write(longitud[:])
	_, _ = mac.Write([]byte(ambito))
	return "hmac-sha256:local_seudonimo_v1:" + hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *SelladorSolicitudSubidaLocal) SellarSolicitudCargaDocumental(ctx context.Context, contenido []byte) (string, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || len(contenido) == 0 || len(contenido) > 131072 {
		return "", ErrHMACSubidaLocalNoDisponible
	}
	mac := hmac.New(sha256.New, s.clave[:])
	_, _ = mac.Write([]byte("vec.documentos.subida.local.solicitud.v1\x00"))
	_, _ = mac.Write(contenido)
	return "hmac-sha256:local_solicitud_v1:" + hex.EncodeToString(mac.Sum(nil)), nil
}

func (SeudonimizadorSujetoSubidaLocal) String() string {
	return "seudonimizador_subida_local_redactado"
}
func (SeudonimizadorSujetoSubidaLocal) GoString() string {
	return "seudonimizador_subida_local_redactado"
}
func (s SeudonimizadorSujetoSubidaLocal) Format(f fmt.State, _ rune) {
	_, _ = f.Write([]byte(s.String()))
}
func (s SeudonimizadorSujetoSubidaLocal) LogValue() slog.Value { return slog.StringValue(s.String()) }
func (SeudonimizadorSujetoSubidaLocal) MarshalJSON() ([]byte, error) {
	return json.Marshal("seudonimizador_subida_local_redactado")
}

func (SelladorSolicitudSubidaLocal) String() string {
	return "sellador_solicitud_subida_local_redactado"
}
func (SelladorSolicitudSubidaLocal) GoString() string {
	return "sellador_solicitud_subida_local_redactado"
}
func (s SelladorSolicitudSubidaLocal) Format(f fmt.State, _ rune) {
	_, _ = f.Write([]byte(s.String()))
}
func (s SelladorSolicitudSubidaLocal) LogValue() slog.Value { return slog.StringValue(s.String()) }
func (SelladorSolicitudSubidaLocal) MarshalJSON() ([]byte, error) {
	return json.Marshal("sellador_solicitud_subida_local_redactado")
}

var _ ports.SeudonimizadorSujetoAlmacen = (*SeudonimizadorSujetoSubidaLocal)(nil)
var _ ports.SelladorSolicitudCargaDocumental = (*SelladorSolicitudSubidaLocal)(nil)
