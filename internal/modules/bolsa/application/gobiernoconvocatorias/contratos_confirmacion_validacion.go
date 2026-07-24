package gobiernoconvocatorias

import puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"

func (s SolicitudConfirmacionBorrador) valida() bool {
	estado, err := puertosbolsa.EstadoVersionConvocatoria(s.Version)
	datosDecision, errDecision := s.Concesion.Evidencia.Datos()
	solicitudPolitica := SolicitudSeleccionPoliticaCifradoBorrador{
		Reserva: s.Reserva, Control: s.Control, Material: s.Material,
		SelladoMotivo: s.SelladoMotivo, SolicitadaEn: s.PoliticaCifrado.SolicitadaEn,
	}
	solicitudPerfil := SolicitudResolucionPerfilCifradoBorrador{
		Reserva: s.Reserva, Control: s.Control, Material: s.Material,
		SelladoMotivo: s.SelladoMotivo, PoliticaEsperada: s.PoliticaCifrado,
		SolicitadaEn: s.ResolucionPerfilCifrado.Evidencia.SolicitudResolucionEn,
	}
	decisionEsperada, errProyeccion := nuevaProyeccionDecisionDiario(
		s.Concesion.Evidencia, s.Material, s.Version, s.Actor, s.CorrelacionRef,
		s.SolicitadaEn, s.Concesion.Atestacion,
	)
	solicitudCifrado, errCifrado := nuevaSolicitudCifradoBorrador(
		s.Version, s.Reserva, s.Control, s.Material, s.SelladoMotivo,
		s.ResolucionPerfilCifrado, s.Procedencia, s.CorrelacionRef, s.Cifrado.SolicitadaEn,
	)
	return err == nil && errDecision == nil && s.Version.Validar() == nil &&
		errProyeccion == nil && errCifrado == nil && s.Cifrado.validaPara(solicitudCifrado) &&
		s.PoliticaCifrado.validaPara(solicitudPolitica) &&
		s.ResolucionPerfilCifrado.ValidarPara(solicitudPerfil) == nil &&
		perfilesCifradoCoinciden(s.PerfilCifrado, s.ResolucionPerfilCifrado.Perfil) &&
		s.Reserva.Decision == decisionEsperada &&
		s.Material.Validar() == nil && estado == s.Material.EstadoPrincipalNuevo &&
		s.Reserva.valida() && s.Reserva.Accion == s.Material.Accion &&
		s.Control.Estado == ResultadoDiarioReservado && s.Control.Revision > 0 &&
		s.Control.Cercado > 0 && s.Control.ArrendamientoIniciaEn.Equal(s.Reserva.ArrendamientoIniciaEn) &&
		s.Control.ArrendamientoVenceEn.Equal(s.Reserva.ArrendamientoVenceEn) &&
		s.Reserva.Decision.DecisionRef == datosDecision.Decision.DecisionRef &&
		s.Reserva.Decision.HuellaDecisionSHA256 == datosDecision.HuellaDecisionSHA256 &&
		s.SelladoMotivo.validaPara(s.Material, s.SolicitadaEn) &&
		!s.SolicitadaEn.Before(s.ResolucionPerfilCifrado.Evidencia.VerificadaEn) &&
		s.SolicitadaEn.Before(s.ResolucionPerfilCifrado.Evidencia.ValidaHasta) &&
		!s.SolicitadaEn.Before(s.Cifrado.CifradoEn) &&
		s.SolicitadaEn.Before(s.Cifrado.AtestacionKMS.ValidaHasta) &&
		instanteOperacionCanonico(s.SolicitadaEn) &&
		!s.SolicitadaEn.Before(s.Reserva.ArrendamientoIniciaEn) &&
		s.SolicitadaEn.Before(s.Reserva.ArrendamientoVenceEn)
}

func (s SolicitudConfirmacionBorrador) Validar() error {
	if !s.valida() {
		return ErrResultadoBorradorInseguro
	}
	return nil
}
