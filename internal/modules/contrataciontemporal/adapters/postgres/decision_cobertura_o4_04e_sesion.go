package postgres

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func (s *sesionDecisionCoberturaO404E) Abrir(
	fragmento cobertura.CabeceraSesionTCBOperacionDecisionCobertura,
) error {
	if !s.entrar() {
		return errSesionDecisionCoberturaO404EInvalida
	}
	defer s.salir()
	if s.estado != estadoSesionDecisionCoberturaNueva {
		return s.invalidar()
	}
	datos, err := fragmento.Datos()
	if err != nil {
		return s.invalidar()
	}
	cabecera, err := nuevaCabeceraDecisionCoberturaO404E(datos)
	if err != nil {
		return s.invalidar()
	}
	s.carga = cargaConfirmarDecisionCoberturaO404E{
		Esquema:  esquemaCargaDecisionCoberturaO404E,
		Rama:     datos.Rama,
		Cabecera: cabecera,
		ConsumosC1: make(
			[]consumoC1DecisionCoberturaO404E,
			0,
			datos.NumeroConsumosC1,
		),
	}
	s.rama = datos.Rama
	s.totalC1 = datos.NumeroConsumosC1
	s.siguienteC1 = 1
	s.peticionesC1 = make(map[string]struct{}, datos.NumeroConsumosC1)
	s.respuestasC1 = make(
		map[claveRespuestaC1DecisionCoberturaO404E]struct{},
		datos.NumeroConsumosC1,
	)
	s.estado = estadoSesionDecisionCoberturaAbierta
	return nil
}

func (s *sesionDecisionCoberturaO404E) Gobierno(
	fragmento cobertura.GobiernoSesionTCBOperacionDecisionCobertura,
) error {
	if !s.entrar() {
		return errSesionDecisionCoberturaO404EInvalida
	}
	defer s.salir()
	if s.estado != estadoSesionDecisionCoberturaAbierta ||
		s.rama != cobertura.RamaSesionTCBOperacionDecisionCoberturaConcedida {
		return s.invalidar()
	}
	datos, err := fragmento.Datos()
	if err != nil {
		return s.invalidar()
	}
	gobierno, err := nuevoGobiernoDecisionCoberturaO404E(
		s.carga.Cabecera,
		datos,
	)
	if err != nil {
		return s.invalidar()
	}
	s.carga.Gobierno = &gobierno
	s.estado = estadoSesionDecisionCoberturaGobernada
	return nil
}

func (s *sesionDecisionCoberturaO404E) DecisionVEC(
	fragmento cobertura.DecisionVECSesionTCBOperacionDecisionCobertura,
) error {
	if !s.entrar() {
		return errSesionDecisionCoberturaO404EInvalida
	}
	defer s.salir()
	estadoEsperado := estadoSesionDecisionCoberturaAbierta
	if s.rama == cobertura.RamaSesionTCBOperacionDecisionCoberturaConcedida {
		estadoEsperado = estadoSesionDecisionCoberturaGobernada
	}
	if s.estado != estadoEsperado {
		return s.invalidar()
	}
	datos, err := fragmento.Datos()
	if err != nil ||
		datos.Concedida !=
			(s.rama ==
				cobertura.RamaSesionTCBOperacionDecisionCoberturaConcedida) {
		return s.invalidar()
	}
	decision, err := nuevaDecisionVECDecisionCoberturaO404E(
		s.carga.Cabecera,
		datos,
	)
	if err != nil {
		return s.invalidar()
	}
	bytesDecision := len(decision.DecisionCanonica) +
		len(decision.MotivoCanonico)
	if bytesDecision == 0 ||
		bytesDecision > maximoBytesMaterialCanonicoDecisionCoberturaO404E {
		limpiarDecisionVECDecisionCoberturaO404E(&decision)
		return s.invalidar()
	}
	s.carga.DecisionVEC = decision
	s.bytesCanonicos = bytesDecision
	s.estado = estadoSesionDecisionCoberturaVEC
	return nil
}

func (s *sesionDecisionCoberturaO404E) ConsumoC1(
	fragmento cobertura.ConsumoC1SesionTCBOperacionDecisionCobertura,
) error {
	if !s.entrar() {
		return errSesionDecisionCoberturaO404EInvalida
	}
	defer s.salir()
	if (s.estado != estadoSesionDecisionCoberturaVEC &&
		s.estado != estadoSesionDecisionCoberturaC1) ||
		s.rama != cobertura.RamaSesionTCBOperacionDecisionCoberturaConcedida {
		return s.invalidar()
	}
	datos, err := fragmento.Datos()
	if err != nil || !posicionConsumoC1DecisionCoberturaO404EValida(
		datos.Posicion,
		datos.Total,
		s.siguienteC1,
		s.totalC1,
	) {
		return s.invalidar()
	}
	consumo, err := nuevoConsumoC1DecisionCoberturaO404E(
		s.carga.Cabecera,
		datos,
	)
	if err != nil {
		return s.invalidar()
	}
	bytesPruebas, err := tamanioPruebasC1DecisionCoberturaO404E(
		consumo.Pruebas,
	)
	if err != nil ||
		s.bytesCanonicos >
			maximoBytesMaterialCanonicoDecisionCoberturaO404E-bytesPruebas {
		limpiarPruebasC1DecisionCoberturaO404E(&consumo.Pruebas)
		return s.invalidar()
	}
	if !s.registrarIdentidadConsumoC1(consumo) {
		limpiarPruebasC1DecisionCoberturaO404E(&consumo.Pruebas)
		return s.invalidar()
	}
	s.carga.ConsumosC1 = append(s.carga.ConsumosC1, consumo)
	s.bytesCanonicos += bytesPruebas
	s.siguienteC1++
	s.estado = estadoSesionDecisionCoberturaC1
	return nil
}

func posicionConsumoC1DecisionCoberturaO404EValida(
	posicion uint64,
	total uint64,
	siguiente uint64,
	totalEsperado uint64,
) bool {
	return total == totalEsperado && posicion == siguiente &&
		posicion > 0 && posicion <= total
}

func (s *sesionDecisionCoberturaO404E) registrarIdentidadConsumoC1(
	consumo consumoC1DecisionCoberturaO404E,
) bool {
	if s == nil || s.peticionesC1 == nil || s.respuestasC1 == nil {
		return false
	}
	clavePeticion := consumo.OrganizacionRef + "\x00" + consumo.PeticionRef
	claveRespuesta := claveRespuestaC1DecisionCoberturaO404E{
		autoridadRef: consumo.AutoridadRef, generacion: consumo.Generacion,
		reciboRespuestaRef: consumo.ReciboRespuestaRef,
	}
	if _, repetida := s.peticionesC1[clavePeticion]; repetida {
		return false
	}
	if _, repetida := s.respuestasC1[claveRespuesta]; repetida {
		return false
	}
	s.peticionesC1[clavePeticion] = struct{}{}
	s.respuestasC1[claveRespuesta] = struct{}{}
	return true
}

func (s *sesionDecisionCoberturaO404E) Concesion(
	fragmento cobertura.EfectoConcedidoSesionTCBOperacionDecisionCobertura,
) error {
	if !s.entrar() {
		return errSesionDecisionCoberturaO404EInvalida
	}
	defer s.salir()
	if s.estado != estadoSesionDecisionCoberturaC1 ||
		s.rama != cobertura.RamaSesionTCBOperacionDecisionCoberturaConcedida ||
		s.totalC1 == 0 || s.siguienteC1 != s.totalC1+1 {
		return s.invalidar()
	}
	datos, err := fragmento.Datos()
	if err != nil {
		return s.invalidar()
	}
	concesion, err := nuevaConcesionDecisionCoberturaO404E(
		s.carga.Cabecera,
		datos,
	)
	if err != nil {
		return s.invalidar()
	}
	s.carga.Concesion = &concesion
	s.estado = estadoSesionDecisionCoberturaLista
	return nil
}

func (s *sesionDecisionCoberturaO404E) Denegacion(
	fragmento cobertura.TerminalDenegadoSesionTCBOperacionDecisionCobertura,
) error {
	if !s.entrar() {
		return errSesionDecisionCoberturaO404EInvalida
	}
	defer s.salir()
	if s.estado != estadoSesionDecisionCoberturaVEC ||
		s.rama != cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada ||
		s.totalC1 != 0 {
		return s.invalidar()
	}
	datos, err := fragmento.Datos()
	if err != nil {
		return s.invalidar()
	}
	denegacion, err := nuevaDenegacionDecisionCoberturaO404E(
		s.carga.Cabecera,
		datos,
	)
	if err != nil {
		return s.invalidar()
	}
	s.carga.Denegacion = &denegacion
	s.estado = estadoSesionDecisionCoberturaLista
	return nil
}

func (s *sesionDecisionCoberturaO404E) Confirmar(
	ctx context.Context,
) (
	cobertura.DatosReciboSesionTCBOperacionDecisionCobertura,
	error,
) {
	if !s.entrar() {
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{},
			errSesionDecisionCoberturaO404EInvalida
	}
	defer s.salir()
	if ctx == nil || s.estado != estadoSesionDecisionCoberturaLista ||
		dependenciaNula(s.tx) || s.ctx == nil ||
		!s.guardia.activa.Load() {
		s.invalidar()
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{},
			errSesionDecisionCoberturaO404EInvalida
	}
	if err := ctx.Err(); err != nil {
		s.invalidar()
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{}, err
	}
	ctxOperacion, cancelarOperacion := contextoLigadoDecisionCoberturaO404E(
		s.ctx,
		ctx,
	)
	defer cancelarOperacion()
	s.estado = estadoSesionDecisionCoberturaConsumida
	contenido, err := codificarCargaConfirmarDecisionCoberturaO404E(s.carga)
	if err != nil {
		s.limpiar()
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{}, err
	}
	defer borrarBytes(contenido)

	var reciboJSON []byte
	err = s.tx.QueryRow(ctxOperacion, `
		SELECT recibo_json::text
		  FROM `+funcionConfirmarDecisionCoberturaO404E+`($1::jsonb)`,
		contenido,
	).Scan(&reciboJSON)
	if err != nil {
		s.limpiar()
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{}, err
	}
	defer borrarBytes(reciboJSON)
	recibo, err := decodificarReciboDecisionCoberturaO404E(reciboJSON)
	if err != nil {
		s.limpiar()
		return cobertura.DatosReciboSesionTCBOperacionDecisionCobertura{}, err
	}
	s.confirmada = true
	s.limpiar()
	return recibo, nil
}
