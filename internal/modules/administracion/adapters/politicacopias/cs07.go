package politicacopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/politicacopias"
	register "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

// ReservadorCS07 delegates all operation uniqueness, receipts, destination
// exclusion and reconciliation to CS07. It adds no operation journal. CS07's
// offline adapter remains declared-progress infrastructure, not production audit.
// The caller must resolve/revalidate current authority before invoking this port.
type ReservadorCS07 struct{ Registro register.Registro }

func (r ReservadorCS07) Reservar(ctx context.Context, v p.Reserva, a p.Atribucion) (p.ReciboReserva, error) {
	if r.Registro == nil {
		return p.ReciboReserva{}, d.ErrDependencia
	}
	if !d.Referencia(v.Clave) || !d.Referencia(v.Politica) || !d.Referencia(v.Destino) || v.Version == 0 || v.Evento.Fecha.IsZero() || !v.Evento.FinVentana.After(v.Evento.Fecha) || !d.Referencia(a.Actor) || !d.Referencia(a.Correlacion) {
		return p.ReciboReserva{}, d.ErrEntrada
	}
	if b, e := hex.DecodeString(v.PoliticaSHA256); e != nil || len(b) != 32 || strings.ToLower(v.PoliticaSHA256) != v.PoliticaSHA256 {
		return p.ReciboReserva{}, d.ErrEntrada
	}
	encoded, e := json.Marshal(v)
	if e != nil {
		return p.ReciboReserva{}, d.ErrEntrada
	}
	seal := sha256.Sum256(encoded)
	id := sha256.Sum256([]byte(v.Clave))
	suffix := hex.EncodeToString(id[:])
	s := operacionescopias.Solicitud{Operacion: "operacion:" + suffix, Clave: v.Clave, SHA256: hex.EncodeToString(seal[:]), Conjunto: "conjunto:" + suffix, Destino: v.Destino, Politica: "politica:" + v.PoliticaSHA256}
	result, e := r.Registro.Reservar(ctx, register.Declaracion{Actor: a.Actor, Correlacion: a.Correlacion}, s)
	if e != nil {
		return p.ReciboReserva{}, e
	}
	// A provider returning another reservation cannot silently authorize execution.
	if result.Solicitud != s {
		return p.ReciboReserva{}, d.ErrEntrada
	}
	return p.ReciboReserva{Operacion: result.Solicitud.Operacion, Replay: result.Replay}, nil
}
