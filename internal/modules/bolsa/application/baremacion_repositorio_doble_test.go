package application

import (
	"context"
	"sync"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type repositorioBaremacionPrueba struct {
	mu             sync.Mutex
	version        puertosbolsa.VersionBaremacion
	token          puertosbolsa.TokenReservaBaremacion
	reserva        *puertosbolsa.SolicitudReservarCambioBaremacion
	consultas      int
	reservas       int
	confirmaciones int
	abandonos      int
}

func (r *repositorioBaremacionPrueba) ObtenerVersionVigente(
	_ context.Context,
	s puertosbolsa.SolicitudObtenerBaremacionVigente,
) (puertosbolsa.VersionBaremacion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.consultas++
	if s.Validar() != nil || s.BaremacionMeritoRef != r.version.Referencia.BaremacionMeritoRef ||
		s.Contexto.Proyeccion().SujetoRef != r.version.Agregado.SujetoRef {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrBaremacionNoEncontrada
	}
	return r.version.Clonar()
}

func (r *repositorioBaremacionPrueba) ObtenerVersion(
	_ context.Context,
	s puertosbolsa.SolicitudObtenerVersionBaremacion,
) (puertosbolsa.VersionBaremacion, error) {
	if s.Validar() != nil || s.Numero != r.version.Referencia.Numero {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrVersionBaremacionNoEncontrada
	}
	return r.version.Clonar()
}

func (r *repositorioBaremacionPrueba) ReservarCambio(
	_ context.Context,
	s puertosbolsa.SolicitudReservarCambioBaremacion,
) (puertosbolsa.ReservaCambioBaremacion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reservas++
	if s.Validar() != nil || s.VersionEsperada == nil || *s.VersionEsperada != r.version.Referencia {
		return puertosbolsa.ReservaCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	clon := s.Clonar()
	r.reserva = &clon
	respuesta := puertosbolsa.ReservaCambioBaremacion{
		Token: r.token, BaremacionMeritoRef: s.BaremacionMeritoRef, Clase: s.Clase,
		HuellaSolicitudHMAC: s.HuellaSolicitudHMAC, ExpiraEn: s.ExpiraEn.UTC(),
	}
	if s.VersionEsperada != nil {
		esperada := *s.VersionEsperada
		respuesta.VersionEsperada = &esperada
	}
	return respuesta, nil
}

func (r *repositorioBaremacionPrueba) ConfirmarCambio(
	_ context.Context,
	s puertosbolsa.SolicitudConfirmarCambioBaremacion,
) (puertosbolsa.ResultadoConfirmarCambioBaremacion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.confirmaciones++
	if s.Validar() != nil || r.reserva == nil || s.Token.Revelar() != r.token.Revelar() ||
		s.VersionEsperada == nil || *s.VersionEsperada != r.version.Referencia {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrReservaBaremacionNoValida
	}
	huella, err := s.Agregado.HuellaEstadoSHA256()
	if err != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, err
	}
	version := puertosbolsa.VersionBaremacion{
		Referencia: puertosbolsa.ReferenciaVersionBaremacion{
			BaremacionMeritoRef: s.Agregado.ID, Numero: r.version.Referencia.Numero + 1, HuellaEstadoSHA256: huella,
		},
		Agregado: s.Agregado, ConfirmadaEn: s.ConfirmadaEn,
	}
	resultado := puertosbolsa.ResultadoConfirmarCambioBaremacion{
		Version: version,
		Evidencia: puertosbolsa.EvidenciaTransaccionBaremacion{
			AuditoriaRef: "auditoria:baremacion:2", HuellaAuditoriaSHA256: huellaBaremacionPrueba("5"),
			EventoOutboxRef: "evento:baremacion:2", HuellaEventoOutboxSHA256: huellaBaremacionPrueba("6"),
			ConfirmadaEn: s.ConfirmadaEn,
		},
	}
	if resultado.Validar() != nil {
		return puertosbolsa.ResultadoConfirmarCambioBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	r.version = version
	return resultado, nil
}

func (r *repositorioBaremacionPrueba) AbandonarReserva(
	_ context.Context,
	s puertosbolsa.SolicitudAbandonarReservaBaremacion,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s.Validar() != nil || r.reserva == nil || s.Token.Revelar() != r.token.Revelar() {
		return puertosbolsa.ErrReservaBaremacionNoValida
	}
	r.abandonos++
	return nil
}

func (r *repositorioBaremacionPrueba) ObtenerEvidenciaTransaccion(
	context.Context,
	puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion,
) (puertosbolsa.EvidenciaTransaccionBaremacionRecuperada, error) {
	return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoEncontrada
}
