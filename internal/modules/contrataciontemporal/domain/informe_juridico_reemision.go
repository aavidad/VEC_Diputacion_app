package domain

// Informe jurídico emitido de nuevo tras subsanar un reparo (duda 5 de RRHH:
// «desfavorable: vuelve a la unidad gestora; tras subsanar, informe nuevo y
// nueva firma»). Que haga falta o no lo decide el catálogo; el dominio solo
// fija cómo se registra: una sola vez por reparo, después de su subsanación,
// sin salir de la subsanación y sustituyendo al informe que se fiscalizó.

// SustitucionInformeJuridico liga el informe nuevo con el que fiscalizó
// Intervención y con el retorno cuya subsanación lo motivó.
type SustitucionInformeJuridico struct {
	InformeRef   string `json:"informe_ref"`
	DocumentoRef string `json:"documento_ref"`
	RetornoRef   string `json:"retorno_ref"`
}

func (s SustitucionInformeJuridico) validar() error {
	if !referenciaValida(s.InformeRef) || !referenciaValida(s.DocumentoRef) ||
		!referenciaValida(s.RetornoRef) {
		return ErrInformeJuridicoEmitidoInvalido
	}
	return nil
}

func (i InformeJuridicoEmitido) retornoSustituido() string {
	if i.Sustituye == nil {
		return ""
	}
	return i.Sustituye.RetornoRef
}

// InformeReemitidoTrasSubsanacion indica que el informe vigente se emitió de
// nuevo para el reparo vigente, tras su subsanación.
func (e Expediente) InformeReemitidoTrasSubsanacion() bool {
	return e.InformeJuridico != nil && e.InformeJuridico.Sustituye != nil &&
		e.Fiscalizacion != nil && e.Fiscalizacion.Retorno != nil &&
		e.InformeJuridico.Sustituye.RetornoRef == e.Fiscalizacion.Retorno.RetornoRef
}

// PuedeReemitirInformeTrasSubsanacion: el reparo vigente ya está subsanado,
// el expediente sigue en la subsanación y todavía no tiene informe nuevo.
func (e Expediente) PuedeReemitirInformeTrasSubsanacion() bool {
	return e.esAntecedenteRefiscalizable() && e.InformeJuridico != nil &&
		!e.InformeReemitidoTrasSubsanacion()
}

// RefiscalizacionEsperaInformeNuevo: el expediente está listo para volver a
// fiscalizarse salvo por el informe nuevo. Solo se aplica cuando el catálogo
// exige informe nuevo tras subsanar.
func (e Expediente) RefiscalizacionEsperaInformeNuevo() bool {
	return e.PuedeReemitirInformeTrasSubsanacion()
}

// ReemitirInformeJuridicoTrasSubsanacion registra el informe nuevo sin salir
// de la subsanación. La actuación cita el retorno subsanado y el informe
// sustituido queda identificado en el informe nuevo; la fiscalización
// desfavorable se conserva intacta.
func (e Expediente) ReemitirInformeJuridicoTrasSubsanacion(
	versionEsperada uint64,
	informe InformeJuridicoEmitido,
	actuacion DatosActuacion,
) (Expediente, error) {
	borrador, err := informe.validarEntrada()
	if e.Validar() != nil || err != nil || actuacion.validar() != nil ||
		!e.PuedeReemitirInformeTrasSubsanacion() ||
		borrador.Estado().ExpedienteRef != e.Referencia ||
		borrador.Estado().VersionEsperadaExpediente != e.Version ||
		!informe.EmitidoEn.Equal(actuacion.RealizadaEn) ||
		actuacion.AccionClave != AccionEmitirInformeJuridico ||
		actuacion.UnidadRef != e.Asignacion.UnidadRef ||
		actuacion.FaseDestino != FaseSubsanacionUnidad ||
		actuacion.EstadoDestino != EstadoIncidencia ||
		actuacion.RetornoRef != e.Fiscalizacion.Retorno.RetornoRef ||
		len(actuacion.DocumentosRef) != 1 ||
		actuacion.DocumentosRef[0] != informe.DocumentoRef ||
		informe.InformeRef == e.InformeJuridico.InformeRef ||
		informe.DocumentoRef == e.InformeJuridico.DocumentoRef ||
		informe.Sustituye != nil || informe.ActuacionRegistro != nil {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	clon := informe.clonar()
	clon.Sustituye = &SustitucionInformeJuridico{
		InformeRef:   e.InformeJuridico.InformeRef,
		DocumentoRef: e.InformeJuridico.DocumentoRef,
		RetornoRef:   e.Fiscalizacion.Retorno.RetornoRef,
	}
	vinculo := nuevoVinculoActuacionInformeJuridico(
		e.Version+1, uint64(len(e.Actuaciones)+1), actuacion, clon,
	)
	clon.ActuacionRegistro = &vinculo
	siguiente.InformeJuridico = &clon
	return siguiente.confirmarTransicion(actuacion)
}

// fiscalizacionApuntaAInformeValido: la fiscalización cita el informe vigente
// o, mientras se espera la nueva fiscalización, el informe que el vigente
// sustituyó tras subsanar ese mismo reparo.
func fiscalizacionApuntaAInformeValido(f *FiscalizacionRegistrada, i *InformeJuridicoEmitido) bool {
	if f.InformeJuridicoRef == i.InformeRef && f.DocumentoInformeRef == i.DocumentoRef {
		return true
	}
	return i.Sustituye != nil && f.Resultado == FiscalizacionDesfavorable && f.Retorno != nil &&
		f.InformeJuridicoRef == i.Sustituye.InformeRef &&
		f.DocumentoInformeRef == i.Sustituye.DocumentoRef &&
		f.Retorno.RetornoRef == i.Sustituye.RetornoRef &&
		f.ActuacionRegistro != nil && i.ActuacionRegistro != nil &&
		i.ActuacionRegistro.Secuencia > f.ActuacionRegistro.Secuencia
}

// EmitirInformeJuridico registra el informe inicial o, si el expediente ya
// tiene informe, el informe nuevo tras subsanar. Completa el destino de la
// actuación según el caso: la emisión inicial pasa al informe jurídico y la
// nueva sigue en la subsanación citando el retorno subsanado.
func (e Expediente) EmitirInformeJuridico(
	versionEsperada uint64,
	informe InformeJuridicoEmitido,
	actuacion DatosActuacion,
) (Expediente, error) {
	if e.InformeJuridico == nil {
		actuacion.FaseDestino, actuacion.EstadoDestino = FaseInformeJuridico, EstadoEnCurso
		return e.RegistrarInformeJuridico(versionEsperada, informe, actuacion)
	}
	if e.Fiscalizacion == nil || e.Fiscalizacion.Retorno == nil {
		return Expediente{}, ErrTransicionInvalida
	}
	actuacion.FaseDestino, actuacion.EstadoDestino = FaseSubsanacionUnidad, EstadoIncidencia
	actuacion.RetornoRef = e.Fiscalizacion.Retorno.RetornoRef
	return e.ReemitirInformeJuridicoTrasSubsanacion(versionEsperada, informe, actuacion)
}
