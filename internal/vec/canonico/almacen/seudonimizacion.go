package almacen

import (
	"fmt"
	"io"
	"log/slog"
)

// SolicitudSeudonimizarSujetoAlmacen mantiene las referencias internas fuera
// de contextos y registros hasta su revelacion deliberada al sellador local.
type SolicitudSeudonimizarSujetoAlmacen struct {
	sujetoRef string
	ambitoRef string
}

func NuevaSolicitudSeudonimizarSujetoAlmacen(
	sujetoRef, ambitoRef string,
) (SolicitudSeudonimizarSujetoAlmacen, error) {
	solicitud := SolicitudSeudonimizarSujetoAlmacen{sujetoRef: sujetoRef, ambitoRef: ambitoRef}
	if !ReferenciaOpacaValida(solicitud.sujetoRef, 512) || !ReferenciaOpacaValida(solicitud.ambitoRef, 256) {
		return SolicitudSeudonimizarSujetoAlmacen{}, ErrSeudonimizacionAlmacenNoDisponible
	}
	return solicitud, nil
}

func (s SolicitudSeudonimizarSujetoAlmacen) RevelarParaSellado() (
	sujetoRef, ambitoRef string,
	err error,
) {
	if !ReferenciaOpacaValida(s.sujetoRef, 512) || !ReferenciaOpacaValida(s.ambitoRef, 256) {
		return "", "", ErrSeudonimizacionAlmacenNoDisponible
	}
	return s.sujetoRef, s.ambitoRef, nil
}

func (SolicitudSeudonimizarSujetoAlmacen) String() string {
	return "[SOLICITUD-SEUDONIMIZACION-ALMACEN-CONFIDENCIAL]"
}

func (SolicitudSeudonimizarSujetoAlmacen) GoString() string {
	return "[SOLICITUD-SEUDONIMIZACION-ALMACEN-CONFIDENCIAL]"
}

func (s SolicitudSeudonimizarSujetoAlmacen) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}

func (SolicitudSeudonimizarSujetoAlmacen) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSeudonimizacionProhibida
}

func (SolicitudSeudonimizarSujetoAlmacen) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSeudonimizacionProhibida
}

func (s SolicitudSeudonimizarSujetoAlmacen) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
