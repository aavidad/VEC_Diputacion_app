package contactopropio

import (
	"context"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

// ConfirmarOperacion consume sólo una intención preparada por Contacto3. La
// referencia se compromete en el recurso V3; el adaptador PostgreSQL compara
// sujeto, versión y HMAC antes de insertar la versión y el recibo original.
func (s *Servicio) ConfirmarOperacion(ctx context.Context, operacionRef, correo string, versionEsperada uint64) (ports.OperacionContactoUsuario, error) {
	vacio := ports.OperacionContactoUsuario{}
	if s == nil || !application.ReferenciaOperacionContactoValida(operacionRef) || versionEsperada >= 1<<53-1 || dependenciaContactoPropioNula(s.registroOperaciones) {
		return vacio, ErrContactoPropioNoDisponible
	}
	recibo, err := s.guardar(ctx, correo, versionEsperada, "", operacionRef, s.registroOperaciones)
	if err != nil {
		return vacio, err
	}
	op := ports.OperacionContactoUsuario{OperacionRef: operacionRef, Estado: ports.OperacionContactoConfirmada, VersionEsperada: versionEsperada, Version: recibo.Version, ReciboRef: recibo.EvidenciaCentral.Referencia, ReplayConfirmado: recibo.ReplayConfirmado}
	if application.ValidarOperacionContacto(op) != nil {
		return vacio, ErrContactoPropioNoDisponible
	}
	return op, nil
}
