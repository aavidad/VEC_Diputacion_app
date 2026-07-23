package ports

import (
	"context"
	"time"
)

const (
	esquemaArtefactoProbatorioBolsa = "vec.contratacion-temporal.artefacto-bolsa"
	versionArtefactoProbatorioBolsa = uint64(1)
	tipoArtefactoOrdenBolsa         = "recibo_orden"
	tipoArtefactoLlamamientoBolsa   = "recibo_llamamiento"
	tipoArtefactoEventoBolsa        = "evento_llamamiento"
)

// DatosDurablesComandoOrdenBolsa es la representación probatoria del comando,
// no una capacidad para volver a ejecutar la operación.
type DatosDurablesComandoOrdenBolsa struct {
	Contexto         RegistroContextoPeticionIntegracionBolsa `json:"contexto"`
	Necesidad        ReferenciaVersionadaIntegracionBolsa     `json:"necesidad"`
	Bolsa            ReferenciaVersionadaIntegracionBolsa     `json:"bolsa"`
	Politica         ReferenciaVersionadaIntegracionBolsa     `json:"politica"`
	MaximoPosiciones uint32                                   `json:"maximo_posiciones"`
}

// ArtefactoProbatorioOrdenBolsa conserva los bytes lógicos necesarios para
// volver a autenticar una orden tras reinicio. Su HMAC está en Evidencia; la
// huella exterior detecta corrupción del sobre antes de usar el verificador.
type ArtefactoProbatorioOrdenBolsa struct {
	Esquema               string                           `json:"esquema"`
	Version               uint64                           `json:"version"`
	Tipo                  string                           `json:"tipo"`
	Comando               DatosDurablesComandoOrdenBolsa   `json:"comando"`
	Recibo                ReciboOrdenBolsa                 `json:"recibo"`
	Evidencia             EvidenciaDurableIntegracionBolsa `json:"evidencia"`
	ClaveVerificacionRef  string                           `json:"clave_verificacion_ref"`
	SelloHMAC             string                           `json:"sello_hmac"`
	HuellaArtefactoSHA256 string                           `json:"huella_artefacto_sha256"`
}

// OrdenProbatoriaRehidratadaBolsa acredita una orden histórica sin exponer el
// comando original para volver a preparar la orden.
type OrdenProbatoriaRehidratadaBolsa struct {
	comando     ComandoPrepararOrdenBolsa
	recibo      ReciboOrdenBolsa
	comprobante ComprobanteEvidenciaIntegracionBolsa
}

func (OrdenProbatoriaRehidratadaBolsa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCapacidadBolsa
}

func (*OrdenProbatoriaRehidratadaBolsa) UnmarshalJSON([]byte) error {
	return ErrSerializacionCapacidadBolsa
}

type PreparacionLlamamientoDesdeOrdenProbatoriaBolsa struct {
	Contexto                ContextoPeticionIntegracionBolsa
	OrdenProbatoria         OrdenProbatoriaRehidratadaBolsa
	MaximaPosicionEvaluable uint32
}

func NuevoComandoLlamamientoDesdeOrdenProbatoriaBolsa(
	preparacion PreparacionLlamamientoDesdeOrdenProbatoriaBolsa,
	instante time.Time,
) (ComandoSolicitarLlamamientoBolsa, error) {
	orden := preparacion.OrdenProbatoria
	return NuevoComandoSolicitarLlamamientoBolsa(
		PreparacionComandoSolicitarLlamamientoBolsa{
			Contexto:     preparacion.Contexto,
			ComandoOrden: orden.comando, ReciboOrden: orden.recibo,
			ComprobanteOrden:        orden.comprobante,
			MaximaPosicionEvaluable: preparacion.MaximaPosicionEvaluable,
		},
		instante,
	)
}

func NuevoArtefactoProbatorioOrdenBolsa(
	comando ComandoPrepararOrdenBolsa,
	recibo ReciboOrdenBolsa,
	evidencia EvidenciaDurableIntegracionBolsa,
	comprobante ComprobanteEvidenciaIntegracionBolsa,
) (ArtefactoProbatorioOrdenBolsa, error) {
	contexto, err := comando.Contexto.datosDurables()
	if err != nil || recibo.ValidarDurablePara(comando) != nil {
		return ArtefactoProbatorioOrdenBolsa{}, ErrRespuestaBolsaNoConfiable
	}
	esperada := nuevaEvidenciaDurableBolsa(
		tipoArtefactoOrdenBolsa,
		contexto.OperacionRef,
		materialComandoOrdenBolsa(comando),
		materialReciboOrdenBolsa(comando, recibo),
		recibo.Procedencia,
	)
	if !evidenciasDurablesBolsaIguales(evidencia, esperada) ||
		!comprobante.coincide(esperada) {
		return ArtefactoProbatorioOrdenBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	registro, err := comando.Contexto.Registro()
	if err != nil {
		return ArtefactoProbatorioOrdenBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	artefacto := ArtefactoProbatorioOrdenBolsa{
		Esquema: esquemaArtefactoProbatorioBolsa,
		Version: versionArtefactoProbatorioBolsa,
		Tipo:    tipoArtefactoOrdenBolsa,
		Comando: DatosDurablesComandoOrdenBolsa{
			Contexto: registro, Necesidad: comando.Necesidad,
			Bolsa: comando.Bolsa, Politica: comando.Politica,
			MaximoPosiciones: comando.MaximoPosiciones,
		},
		Recibo: recibo, Evidencia: evidencia,
		ClaveVerificacionRef: evidencia.ClaveVerificacionRef,
		SelloHMAC:            evidencia.SelloHMAC,
	}
	artefacto.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(artefacto)
	return artefacto, nil
}

func (a ArtefactoProbatorioOrdenBolsa) Validar() error {
	if !cabeceraArtefactoProbatorioBolsaValida(
		a.Esquema, a.Version, a.Tipo, tipoArtefactoOrdenBolsa,
		a.ClaveVerificacionRef, a.SelloHMAC, a.Evidencia,
		a.HuellaArtefactoSHA256, huellaArtefactoProbatorioBolsa(a),
	) || a.Comando.Contexto.validarSintaxis() != nil ||
		a.Comando.Necesidad.Validar() != nil ||
		a.Comando.Bolsa.Validar() != nil ||
		a.Comando.Politica.Validar() != nil ||
		a.Comando.MaximoPosiciones == 0 ||
		a.Comando.MaximoPosiciones > MaximoElementosIntegracionBolsa {
		return ErrEvidenciaBolsaNoAutenticada
	}
	return nil
}

func (a *ArtefactoProbatorioOrdenBolsa) UnmarshalJSON(contenido []byte) error {
	type alias ArtefactoProbatorioOrdenBolsa
	var valor alias
	if err := decodificarArtefactoCerradoBolsa(contenido, &valor); err != nil {
		return ErrEvidenciaBolsaNoAutenticada
	}
	*a = ArtefactoProbatorioOrdenBolsa(valor)
	return a.Validar()
}

func (a ArtefactoProbatorioOrdenBolsa) Rehidratar(
	ctx context.Context,
	autenticador *AutenticadorContextoPeticionIntegracionBolsa,
	verificador *VerificadorEvidenciaIntegracionBolsa,
	instante time.Time,
) (OrdenProbatoriaRehidratadaBolsa, error) {
	if a.Validar() != nil || autenticador == nil || verificador == nil {
		return OrdenProbatoriaRehidratadaBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	contexto, err := autenticador.reautenticarDurable(
		ctx, a.Comando.Contexto, instante,
	)
	if err != nil {
		return OrdenProbatoriaRehidratadaBolsa{}, err
	}
	comando := ComandoPrepararOrdenBolsa{
		Contexto: contexto, Necesidad: a.Comando.Necesidad,
		Bolsa: a.Comando.Bolsa, Politica: a.Comando.Politica,
		MaximoPosiciones: a.Comando.MaximoPosiciones,
	}
	comprobante, err := verificador.reautenticarReciboOrden(
		ctx, comando, a.Recibo, a.Evidencia, instante,
	)
	if err != nil {
		return OrdenProbatoriaRehidratadaBolsa{}, err
	}
	return OrdenProbatoriaRehidratadaBolsa{
		comando: comando, recibo: a.Recibo, comprobante: comprobante,
	}, nil
}

// DatosDurablesComandoLlamamientoBolsa conserva la petición exacta que fue
// atendida por Bolsa, sin convertirla en un comando ejecutable por transporte.
type DatosDurablesComandoLlamamientoBolsa struct {
	Contexto                RegistroContextoPeticionIntegracionBolsa `json:"contexto"`
	Necesidad               ReferenciaVersionadaIntegracionBolsa     `json:"necesidad"`
	Bolsa                   ReferenciaVersionadaIntegracionBolsa     `json:"bolsa"`
	Orden                   ReferenciaVersionadaIntegracionBolsa     `json:"orden"`
	Politica                ReferenciaVersionadaIntegracionBolsa     `json:"politica"`
	TotalPosicionesOrden    uint32                                   `json:"total_posiciones_orden"`
	MaximaPosicionEvaluable uint32                                   `json:"maxima_posicion_evaluable"`
	HuellaReciboOrden       string                                   `json:"huella_recibo_orden"`
}

type ArtefactoProbatorioLlamamientoBolsa struct {
	Esquema               string                               `json:"esquema"`
	Version               uint64                               `json:"version"`
	Tipo                  string                               `json:"tipo"`
	Comando               DatosDurablesComandoLlamamientoBolsa `json:"comando"`
	Recibo                ReciboSolicitudLlamamientoBolsa      `json:"recibo"`
	Evidencia             EvidenciaDurableIntegracionBolsa     `json:"evidencia"`
	ClaveVerificacionRef  string                               `json:"clave_verificacion_ref"`
	SelloHMAC             string                               `json:"sello_hmac"`
	HuellaArtefactoSHA256 string                               `json:"huella_artefacto_sha256"`
}

// LlamamientoProbatorioRehidratadoBolsa solo puede derivar el enlace para
// registrar un evento. No satisface GestorLlamamientosBolsa ni expone el
// comando original para repetir el llamamiento.
type LlamamientoProbatorioRehidratadoBolsa struct {
	comando     ComandoSolicitarLlamamientoBolsa
	recibo      ReciboSolicitudLlamamientoBolsa
	comprobante ComprobanteEvidenciaIntegracionBolsa
}

func (LlamamientoProbatorioRehidratadoBolsa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCapacidadBolsa
}

func (*LlamamientoProbatorioRehidratadoBolsa) UnmarshalJSON([]byte) error {
	return ErrSerializacionCapacidadBolsa
}

func enlaceDesdeLlamamientoProbatorioBolsa(
	prueba LlamamientoProbatorioRehidratadoBolsa,
) (EnlaceEventoLlamamientoBolsa, error) {
	return NuevoEnlaceEventoLlamamientoBolsa(
		PreparacionEnlaceEventoLlamamientoBolsa{
			Comando: prueba.comando, Recibo: prueba.recibo,
			Comprobante: prueba.comprobante,
		},
	)
}

func NuevoArtefactoProbatorioLlamamientoBolsa(
	comando ComandoSolicitarLlamamientoBolsa,
	recibo ReciboSolicitudLlamamientoBolsa,
	evidencia EvidenciaDurableIntegracionBolsa,
	comprobante ComprobanteEvidenciaIntegracionBolsa,
) (ArtefactoProbatorioLlamamientoBolsa, error) {
	datos, err := comando.datosCanonicos()
	if err != nil || recibo.ValidarDurablePara(comando) != nil {
		return ArtefactoProbatorioLlamamientoBolsa{}, ErrRespuestaBolsaNoConfiable
	}
	contexto, _ := datos.Contexto.datosDurables()
	esperada := nuevaEvidenciaDurableBolsa(
		tipoArtefactoLlamamientoBolsa,
		contexto.OperacionRef,
		materialComandoLlamamientoBolsa(comando),
		materialReciboLlamamientoBolsa(comando, recibo),
		recibo.Procedencia,
	)
	if !evidenciasDurablesBolsaIguales(evidencia, esperada) ||
		!comprobante.coincide(esperada) {
		return ArtefactoProbatorioLlamamientoBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	registro, err := datos.Contexto.Registro()
	if err != nil {
		return ArtefactoProbatorioLlamamientoBolsa{}, ErrPeticionIntegracionBolsaInvalida
	}
	artefacto := ArtefactoProbatorioLlamamientoBolsa{
		Esquema: esquemaArtefactoProbatorioBolsa,
		Version: versionArtefactoProbatorioBolsa,
		Tipo:    tipoArtefactoLlamamientoBolsa,
		Comando: DatosDurablesComandoLlamamientoBolsa{
			Contexto: registro, Necesidad: datos.Necesidad, Bolsa: datos.Bolsa,
			Orden: datos.Orden, Politica: datos.Politica,
			TotalPosicionesOrden:    datos.TotalPosicionesOrden,
			MaximaPosicionEvaluable: datos.MaximaPosicionEvaluable,
			HuellaReciboOrden:       datos.HuellaReciboOrden,
		},
		Recibo: recibo, Evidencia: evidencia,
		ClaveVerificacionRef: evidencia.ClaveVerificacionRef,
		SelloHMAC:            evidencia.SelloHMAC,
	}
	artefacto.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(artefacto)
	return artefacto, nil
}

func (a ArtefactoProbatorioLlamamientoBolsa) Validar() error {
	if !cabeceraArtefactoProbatorioBolsaValida(
		a.Esquema, a.Version, a.Tipo, tipoArtefactoLlamamientoBolsa,
		a.ClaveVerificacionRef, a.SelloHMAC, a.Evidencia,
		a.HuellaArtefactoSHA256, huellaArtefactoProbatorioBolsa(a),
	) || a.Comando.Contexto.validarSintaxis() != nil ||
		a.Comando.Necesidad.Validar() != nil ||
		a.Comando.Bolsa.Validar() != nil || a.Comando.Orden.Validar() != nil ||
		a.Comando.Politica.Validar() != nil ||
		a.Comando.TotalPosicionesOrden == 0 ||
		a.Comando.TotalPosicionesOrden > MaximoElementosIntegracionBolsa ||
		a.Comando.MaximaPosicionEvaluable == 0 ||
		a.Comando.MaximaPosicionEvaluable > a.Comando.TotalPosicionesOrden ||
		!huellaSHA256Valida(a.Comando.HuellaReciboOrden) {
		return ErrEvidenciaBolsaNoAutenticada
	}
	return nil
}

func (a *ArtefactoProbatorioLlamamientoBolsa) UnmarshalJSON(contenido []byte) error {
	type alias ArtefactoProbatorioLlamamientoBolsa
	var valor alias
	if err := decodificarArtefactoCerradoBolsa(contenido, &valor); err != nil {
		return ErrEvidenciaBolsaNoAutenticada
	}
	*a = ArtefactoProbatorioLlamamientoBolsa(valor)
	return a.Validar()
}

func (a ArtefactoProbatorioLlamamientoBolsa) Rehidratar(
	ctx context.Context,
	autenticador *AutenticadorContextoPeticionIntegracionBolsa,
	verificador *VerificadorEvidenciaIntegracionBolsa,
	instante time.Time,
) (LlamamientoProbatorioRehidratadoBolsa, error) {
	if a.Validar() != nil || autenticador == nil || verificador == nil {
		return LlamamientoProbatorioRehidratadoBolsa{},
			ErrEvidenciaBolsaNoAutenticada
	}
	contexto, err := autenticador.reautenticarDurable(
		ctx, a.Comando.Contexto, instante,
	)
	if err != nil {
		return LlamamientoProbatorioRehidratadoBolsa{}, err
	}
	comando := ComandoSolicitarLlamamientoBolsa{
		datos: &datosComandoSolicitarLlamamientoBolsa{
			contexto: contexto, necesidad: a.Comando.Necesidad,
			bolsa: a.Comando.Bolsa, orden: a.Comando.Orden,
			politica:                a.Comando.Politica,
			totalPosicionesOrden:    a.Comando.TotalPosicionesOrden,
			maximaPosicionEvaluable: a.Comando.MaximaPosicionEvaluable,
			huellaReciboOrden:       a.Comando.HuellaReciboOrden,
		},
	}
	if _, err := comando.datosCanonicos(); err != nil {
		return LlamamientoProbatorioRehidratadoBolsa{},
			ErrEvidenciaBolsaNoAutenticada
	}
	comprobante, err := verificador.reautenticarReciboLlamamiento(
		ctx, comando, a.Recibo, a.Evidencia, instante,
	)
	if err != nil {
		return LlamamientoProbatorioRehidratadoBolsa{}, err
	}
	return LlamamientoProbatorioRehidratadoBolsa{
		comando: comando, recibo: a.Recibo, comprobante: comprobante,
	}, nil
}

type ArtefactoProbatorioEventoBolsa struct {
	Esquema               string                           `json:"esquema"`
	Version               uint64                           `json:"version"`
	Tipo                  string                           `json:"tipo"`
	Evento                EventoLlamamientoBolsa           `json:"evento"`
	Evidencia             EvidenciaDurableIntegracionBolsa `json:"evidencia"`
	ClaveVerificacionRef  string                           `json:"clave_verificacion_ref"`
	SelloHMAC             string                           `json:"sello_hmac"`
	HuellaArtefactoSHA256 string                           `json:"huella_artefacto_sha256"`
}

// EventoProbatorioRehidratadoBolsa es una capacidad histórica de un solo
// propósito: construir el comando idempotente de registro.
type EventoProbatorioRehidratadoBolsa struct {
	evento      EventoLlamamientoBolsa
	enlace      EnlaceEventoLlamamientoBolsa
	comprobante ComprobanteEvidenciaIntegracionBolsa
	validadoEn  time.Time
}

func (EventoProbatorioRehidratadoBolsa) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionCapacidadBolsa
}

func (*EventoProbatorioRehidratadoBolsa) UnmarshalJSON([]byte) error {
	return ErrSerializacionCapacidadBolsa
}

func NuevoComandoRegistrarEventoRehidratadoBolsa(
	prueba EventoProbatorioRehidratadoBolsa,
) (ComandoRegistrarEventoBolsa, error) {
	return NuevoComandoRegistrarEventoBolsa(
		prueba.evento,
		prueba.enlace,
		prueba.comprobante,
		prueba.validadoEn,
	)
}

func NuevoArtefactoProbatorioEventoBolsa(
	evento EventoLlamamientoBolsa,
	enlace EnlaceEventoLlamamientoBolsa,
	evidencia EvidenciaDurableIntegracionBolsa,
	comprobante ComprobanteEvidenciaIntegracionBolsa,
	instante time.Time,
) (ArtefactoProbatorioEventoBolsa, error) {
	esperada := nuevaEvidenciaDurableBolsa(
		tipoArtefactoEventoBolsa,
		evento.PeticionRef,
		materialEnlaceEventoBolsa(enlace),
		materialEventoBolsa(evento),
		evento.Procedencia,
	)
	if !comprobante.instanteVerificacion().Equal(instante) ||
		evento.ValidarDurableParaEn(enlace, instante) != nil ||
		!evidenciasDurablesBolsaIguales(evidencia, esperada) ||
		!comprobante.coincide(esperada) {
		return ArtefactoProbatorioEventoBolsa{}, ErrEvidenciaBolsaNoAutenticada
	}
	artefacto := ArtefactoProbatorioEventoBolsa{
		Esquema: esquemaArtefactoProbatorioBolsa,
		Version: versionArtefactoProbatorioBolsa,
		Tipo:    tipoArtefactoEventoBolsa,
		Evento:  evento, Evidencia: evidencia,
		ClaveVerificacionRef: evidencia.ClaveVerificacionRef,
		SelloHMAC:            evidencia.SelloHMAC,
	}
	artefacto.HuellaArtefactoSHA256 = huellaArtefactoProbatorioBolsa(artefacto)
	return artefacto, nil
}

func (a ArtefactoProbatorioEventoBolsa) Validar() error {
	if !cabeceraArtefactoProbatorioBolsaValida(
		a.Esquema, a.Version, a.Tipo, tipoArtefactoEventoBolsa,
		a.ClaveVerificacionRef, a.SelloHMAC, a.Evidencia,
		a.HuellaArtefactoSHA256, huellaArtefactoProbatorioBolsa(a),
	) || a.Evento.validarEstructuraDurable() != nil {
		return ErrEvidenciaBolsaNoAutenticada
	}
	return nil
}

func (a *ArtefactoProbatorioEventoBolsa) UnmarshalJSON(contenido []byte) error {
	type alias ArtefactoProbatorioEventoBolsa
	var valor alias
	if err := decodificarArtefactoCerradoBolsa(contenido, &valor); err != nil {
		return ErrEvidenciaBolsaNoAutenticada
	}
	*a = ArtefactoProbatorioEventoBolsa(valor)
	return a.Validar()
}

func (a ArtefactoProbatorioEventoBolsa) Rehidratar(
	ctx context.Context,
	llamamiento LlamamientoProbatorioRehidratadoBolsa,
	verificador *VerificadorEvidenciaIntegracionBolsa,
	instante time.Time,
) (EventoProbatorioRehidratadoBolsa, error) {
	if a.Validar() != nil || verificador == nil {
		return EventoProbatorioRehidratadoBolsa{},
			ErrEvidenciaBolsaNoAutenticada
	}
	enlace, err := enlaceDesdeLlamamientoProbatorioBolsa(llamamiento)
	if err != nil {
		return EventoProbatorioRehidratadoBolsa{}, err
	}
	comprobante, err := verificador.reautenticarEvento(
		ctx, a.Evento, enlace, a.Evidencia, instante,
	)
	if err != nil {
		return EventoProbatorioRehidratadoBolsa{}, err
	}
	return EventoProbatorioRehidratadoBolsa{
		evento: a.Evento, enlace: enlace,
		comprobante: comprobante, validadoEn: instante,
	}, nil
}
