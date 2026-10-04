package registrocopias

import (
	"context"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	port "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

// AbrirConObservadorAbandono wires an existing trusted authority/observer. It
// never derives permission from declared roles, JSON or the existence of a key.
func AbrirConObservadorAbandono(c Config, o port.ObservadorAbandono) (*Fichero, error) {
	f, err := Abrir(c)
	if err != nil {
		return nil, err
	}
	f.observadorAbandono = o
	return f, nil
}

func (f *Fichero) AbandonarCaptura(ctx context.Context, d port.Declaracion, s port.SolicitudAbandono) (port.Resultado, error) {
	a := operacionescopias.ObservacionAbandono{Operacion: s.Operacion, Destino: s.Destino, FalloReferencia: s.FalloReferencia, FalloSHA256: s.FalloSHA256}
	c := operacionescopias.Comando{Clave: s.Clave, VersionEsperada: s.VersionEsperada, SolicitudSHA256: s.SolicitudSHA256, Accion: "abandonar_captura", Abandono: &a}
	return f.ejecutar(ctx, peticion{Accion: "abandonar_captura", Declaracion: d, Operacion: s.Operacion, Comando: &c})
}

func (f *Fichero) confirmarAbandono(ctx context.Context, p peticion) (peticion, error) {
	if f.observadorAbandono == nil {
		return p, port.ErrAbandonoNoAutorizado
	}
	c, a := p.Comando, p.Comando.Abandono
	s := port.SolicitudAbandono{Operacion: p.Operacion, Clave: c.Clave, VersionEsperada: c.VersionEsperada, SolicitudSHA256: c.SolicitudSHA256, Destino: a.Destino, FalloReferencia: a.FalloReferencia, FalloSHA256: a.FalloSHA256}
	observacion, err := f.observadorAbandono.ConfirmarAbandono(ctx, p.Declaracion, s)
	if err != nil || !observacionSegura(observacion) || observacion.Operacion != s.Operacion || observacion.Destino != s.Destino || observacion.FalloReferencia != s.FalloReferencia || observacion.FalloSHA256 != s.FalloSHA256 {
		return p, port.ErrAbandonoNoAutorizado
	}
	c.Abandono = &observacion
	return p, nil
}

func observacionSegura(a operacionescopias.ObservacionAbandono) bool {
	return referencia.MatchString(a.Operacion) && referencia.MatchString(a.Destino) && referencia.MatchString(a.FalloReferencia) && huella.MatchString(a.FalloSHA256) && (a.Lease == "" || referencia.MatchString(a.Lease)) && estadoSeguro(a.EstadoEfecto, "", "inactivo", "activo", "incierto", "pendiente") && estadoSeguro(a.EstadoLease, "", "cancelada", "vigente", "incierta") && estadoSeguro(a.EstadoPlataforma, "", "sin_efectos_pendientes", "escribiendo", "mantenimiento", "restaurando", "incierta") && estadoSeguro(a.EstadoVerificador, "", "detenido", "activo", "incierto") && estadoSeguro(a.EstadoVentana, "", "inactiva", "activa", "incierta")
}

func estadoSeguro(s string, admitidos ...string) bool {
	for _, a := range admitidos {
		if s == a {
			return true
		}
	}
	return false
}
