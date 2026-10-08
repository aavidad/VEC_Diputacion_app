package domain

// AccionVincularOfertaBolsa representa la decisión expresa de RRHH de
// asociar una oferta ya publicada por Bolsa con este expediente. La oferta
// conserva su propia autoridad y esta acción no adjudica una persona.
const AccionVincularOfertaBolsa ClaveCatalogo = "contratacion_temporal.oferta_bolsa.vincular"

// VincularOfertaBolsa añade la actuación y el hito del circuito en una nueva
// versión. La persistencia debe confirmar esta postimagen, la asociación
// Bolsa, la auditoría y el outbox en una sola transacción.
func (e Expediente) VincularOfertaBolsa(
	versionEsperada uint64,
	definicion DefinicionCircuitoRRHH,
	ofertaRef, perfilRef string,
	actuacion DatosActuacion,
) (Expediente, error) {
	if e.Validar() != nil || e.Circuito == nil || e.Analisis == nil ||
		e.ViaCobertura == nil || e.ViaCobertura.BolsaRef == "" ||
		e.Asignacion == nil || e.InformeJuridico != nil ||
		!referenciaValida(ofertaRef) || !referenciaValida(perfilRef) ||
		actuacion.validar() != nil || actuacion.AccionClave != AccionVincularOfertaBolsa ||
		actuacion.UnidadRef != e.Asignacion.UnidadRef ||
		actuacion.FaseDestino != e.FaseActual || actuacion.EstadoDestino != e.EstadoActual ||
		actuacion.Observaciones != "" || len(actuacion.DocumentosRef) != 0 || actuacion.RetornoRef != "" ||
		len(e.Circuito.Hitos) == 0 {
		return Expediente{}, ErrTransicionInvalida
	}
	credito := e.Circuito.Hitos[len(e.Circuito.Hitos)-1]
	if credito.Tipo != HitoCreditoComprobado || credito.Destino != e.Circuito.EstadoActual ||
		e.Analisis.ValidacionRC.Resultado != RCValidada ||
		credito.CreditoRef != e.Analisis.ValidacionRC.ReciboRef ||
		credito.DocumentoRef != e.Analisis.ValidacionRC.DocumentoRef ||
		actuacion.RealizadaEn.Before(credito.RegistradoEn) {
		return Expediente{}, ErrTransicionInvalida
	}
	preparado, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	siguiente, err := preparado.confirmarTransicion(actuacion)
	if err != nil {
		return Expediente{}, err
	}
	var clave ClaveCatalogo
	var perfilClave ClaveCatalogo
	for _, transicion := range definicion.Transiciones {
		if transicion.Tipo == HitoOfertaEmitida && transicion.Origen == e.Circuito.EstadoActual {
			clave, perfilClave = transicion.Clave, transicion.PerfilClave
			break
		}
	}
	if !clave.Valida() || !perfilClave.Valida() {
		return Expediente{}, ErrTransicionInvalida
	}
	hito := HitoCircuitoRRHH{
		Clave: clave, ActuacionClave: actuacion.AccionClave,
		ActorRef: actuacion.ActorRef, PerfilClave: perfilClave,
		PerfilRef: perfilRef, UnidadRef: actuacion.UnidadRef,
		OfertaRef: ofertaRef, ReciboRef: actuacion.ReciboRef,
		RegistradoEn: actuacion.RealizadaEn,
	}
	return siguiente.AdjuntarHitosCircuito(definicion, siguiente.Version, []HitoCircuitoRRHH{hito})
}
