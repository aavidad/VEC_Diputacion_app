package ejecucioncopias

import (
	"context"

	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	ej "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
	cs07 "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

// AbandonarCaptura no transforma un fallo en evidencia de ensayo. CS07 vuelve
// a observar autoridad, efecto y reserva antes de permitir su estado terminal.
func (r *RegistroCS07) AbandonarCaptura(ctx context.Context, ref, fallo string) error {
	if r == nil || r.diario == nil || r.registro == nil {
		return errRegistroConfiguracion
	}
	if fallo != "captura_fallida" && fallo != "captura_no_comprobable" {
		return errRegistroEntrada
	}
	e, err := r.diario.leer(ctx, ref)
	if err != nil {
		return err
	}
	if !capturaAbandonable(e, fallo) {
		return errRegistroTransicion
	}
	proveedor, ok := r.registro.(cs07.AbandonadorCaptura)
	if !ok {
		return errRegistroConfiguracion
	}
	actual, err := r.registro.Consultar(ctx, e.declaracion(), ref)
	if err != nil || actual.Solicitud.Operacion != ref || actual.Solicitud.SHA256 != e.SolicitudSHA256 ||
		actual.Solicitud.Conjunto != e.ConjuntoRef || actual.Solicitud.Destino != e.DestinoRef || actual.Solicitud.Politica != e.PoliticaRef {
		return errRegistroVinculo
	}
	falloSHA := huellaEvidencia(struct{ Operacion, Fallo string }{ref, fallo})
	s := cs07.SolicitudAbandono{Operacion: ref, Clave: "cs11:abandono:" + falloSHA[:48], VersionEsperada: actual.Recibo.Version,
		SolicitudSHA256: e.SolicitudSHA256, Destino: e.DestinoRef, FalloReferencia: fallo, FalloSHA256: falloSHA}
	for _, evento := range actual.Historia {
		if evento.Comando.Clave == s.Clave {
			s.VersionEsperada = evento.VersionPrevia
			break
		}
	}
	confirmada, err := proveedor.AbandonarCaptura(ctx, e.declaracion(), s)
	if err != nil {
		return err
	}
	if confirmada.Solicitud != actual.Solicitud || confirmada.Recibo.Estado != operacionescopias.AbandonadaDeclarada {
		return errRegistroVinculo
	}
	return r.diario.actualizar(ctx, ref, "abandono_captura", func(actual estadoExterior) (estadoExterior, error) {
		if actual.SolicitudSHA256 != e.SolicitudSHA256 || actual.Actor != e.Actor || actual.DestinoRef != e.DestinoRef || !capturaAbandonable(actual, fallo) {
			return actual, errRegistroVinculo
		}
		actual.Estado, actual.FalloCapturaRef = string(operacionescopias.AbandonadaDeclarada), fallo
		return actual, nil
	})
}

func capturaAbandonable(e estadoExterior, fallo string) bool {
	return e.PreimagenSHA256 == "" && (e.Estado == "capturando" || e.Estado == "captura_pendiente_conciliacion" || e.Estado == string(operacionescopias.AbandonadaDeclarada)) &&
		(e.FalloCapturaRef == "" || e.FalloCapturaRef == fallo)
}

var _ ej.RegistroAbandono = (*RegistroCS07)(nil)
