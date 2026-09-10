package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// SolicitudRecuperacionIncorporacion identifica el registro, no su autoridad
// ni su historia. Los datos del efecto y el permiso los resuelve el servidor.
type SolicitudRecuperacionIncorporacion struct {
	ExpedienteRef        string
	SolicitudPersonalRef string
}

func (s SolicitudRecuperacionIncorporacion) Validar() error {
	if !domain.ReferenciaOpacaValida(s.ExpedienteRef) || !domain.ReferenciaOpacaValida(s.SolicitudPersonalRef) {
		return ErrRecuperacionIncorporacionInvalida
	}
	return nil
}

func (s SolicitudRecuperacionIncorporacion) ValidarPara(orden OrdenConfirmarIncorporacion, ahora time.Time) error {
	datos, err := orden.Datos()
	if s.Validar() != nil || err != nil || orden.ValidarDentroDeTransaccion(ahora) != nil ||
		datos.EvaluadaEn.After(ahora) || datos.Confirmacion.SolicitudPersonal.ExpedienteRef != s.ExpedienteRef ||
		datos.Confirmacion.SolicitudPersonal.SolicitudRef != s.SolicitudPersonalRef {
		return ErrRecuperacionIncorporacionInvalida
	}
	return nil
}

// ResolutorRecuperacionIncorporacion obtiene del contexto confiable una orden
// nominal fresca para el efecto identificado. Reutiliza la autorización de
// incorporación, sin ejecutarla ni fabricar datos/historia desde el canal.
// Debe resolver los datos comprometidos desde fuentes servidor autorizadas;
// no consultar el registro de recuperación antes de conceder el permiso.
type ResolutorRecuperacionIncorporacion interface {
	ResolverRecuperacionIncorporacion(context.Context, SolicitudRecuperacionIncorporacion) (OrdenConfirmarIncorporacion, error)
}

// LectorRegistroIncorporacion es exclusivamente de lectura: devuelve recibo y
// evidencia original del MISMO registro durable verificado, o error sin datos.
// Recibe una orden ya autorizada, debe respetar cancelación y autorización en
// su frontera y no llamar ConfirmarIncorporacion, SolicitarAlta ni reservar un
// efecto. Puede auditar el acceso. La aplicación revalida al concluir la lectura.
// Un doble de este puerto no acredita persistencia ni procedencia del historial.
type LectorRegistroIncorporacion interface {
	LeerRegistroIncorporacion(context.Context, OrdenConfirmarIncorporacion) (ReciboConfirmacionIncorporacion, EvidenciaRegistroIncorporacion, error)
}
