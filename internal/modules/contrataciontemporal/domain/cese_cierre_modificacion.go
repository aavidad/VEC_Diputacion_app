package domain

import (
	"errors"
	"regexp"
	"time"
)

// Acciones de la fase de seguimiento del expediente (petición RRHH p. 1,
// punto 3; dudas 11 y 12). El cese termina la relación; el cierre la da por
// concluida tras el cese; la modificación cambia fechas o jornada después del
// nombramiento y devuelve el expediente a la fase que fija el catálogo.
const (
	AccionCesarNombramiento         ClaveCatalogo = "contratacion_temporal.seguimiento.cesar"
	AccionCerrarExpediente          ClaveCatalogo = "contratacion_temporal.expediente.cerrar"
	AccionModificarTrasNombramiento ClaveCatalogo = "contratacion_temporal.expediente.modificar_tras_nombramiento"

	FaseNombramiento ClaveFase = "nombramiento"

	// CondicionCeseRegistrado y CondicionGINPIXConfirmado son las únicas
	// condiciones de cierre que el sistema sabe comprobar; la regla del
	// catálogo elige cuáles se exigen. El cese es siempre obligatorio.
	CondicionCeseRegistrado   = "cese_registrado"
	CondicionGINPIXConfirmado = "ginpix_confirmado"

	prefijoDocumentoGINPIX = "ginpix:"
)

var (
	ErrCeseInvalido         = errors.New("contratacion temporal: cese invalido")
	ErrCierreInvalido       = errors.New("contratacion temporal: cierre de expediente invalido")
	ErrModificacionInvalida = errors.New("contratacion temporal: modificacion tras nombramiento invalida")

	patronNumeroGINPIX = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,63}$`)
)

// DatosCese recoge el cese tal como lo registra RRHH. El justificante se
// identifica por referencia y huella; su contenido no entra en el expediente.
type DatosCese struct {
	CausaClave         ClaveCatalogo
	FechaEfecto        time.Time
	JustificanteTipo   ClaveCatalogo
	JustificanteRef    string
	JustificanteSHA256 string
	Observaciones      string
}

func (d DatosCese) Validar() error {
	if !d.CausaClave.Valida() || !fechaCivilCanonica(d.FechaEfecto) ||
		!d.JustificanteTipo.Valida() || !referenciaValida(d.JustificanteRef) ||
		!huellaEntradaValida(d.JustificanteSHA256) || !textoValido(d.Observaciones, 2000, true) {
		return ErrCeseInvalido
	}
	return nil
}

// DatosCierreExpediente fija las condiciones exigidas por la regla vigente y,
// si se exige, la confirmación de GINPIX con su número.
type DatosCierreExpediente struct {
	Condiciones        []string
	GINPIXNumero       string
	GINPIXConfirmadaEn *time.Time
	Observaciones      string
}

// CondicionesCierreValidas exige la lista ordenada, sin repetir, con el cese
// y solo condiciones conocidas.
func CondicionesCierreValidas(condiciones []string) bool {
	if len(condiciones) == 0 || len(condiciones) > 2 {
		return false
	}
	cese := false
	for i, c := range condiciones {
		if c != CondicionCeseRegistrado && c != CondicionGINPIXConfirmado {
			return false
		}
		if i > 0 && condiciones[i-1] >= c {
			return false
		}
		cese = cese || c == CondicionCeseRegistrado
	}
	return cese
}

func (d DatosCierreExpediente) exigeGINPIX() bool {
	for _, c := range d.Condiciones {
		if c == CondicionGINPIXConfirmado {
			return true
		}
	}
	return false
}

func (d DatosCierreExpediente) Validar() error {
	if !CondicionesCierreValidas(d.Condiciones) || !textoValido(d.Observaciones, 2000, true) {
		return ErrCierreInvalido
	}
	conGINPIX := d.GINPIXNumero != ""
	if conGINPIX != (d.GINPIXConfirmadaEn != nil) || d.exigeGINPIX() != conGINPIX ||
		(conGINPIX && (!patronNumeroGINPIX.MatchString(d.GINPIXNumero) || !fechaCivilCanonica(*d.GINPIXConfirmadaEn))) {
		return ErrCierreInvalido
	}
	return nil
}

// DocumentoGINPIX es la referencia documental de la ficha confirmada.
func (d DatosCierreExpediente) DocumentoGINPIX() string {
	if d.GINPIXNumero == "" {
		return ""
	}
	return prefijoDocumentoGINPIX + d.GINPIXNumero
}

// DatosModificacionTrasNombramiento contiene el análisis ya recalculado y la
// fase de vuelta que fija el catálogo (fiscalización o informe jurídico).
type DatosModificacionTrasNombramiento struct {
	MotivoClave   ClaveCatalogo
	Periodo       PeriodoPrevisto
	Jornada       JornadaDiezmilesimas
	Coste         Importe
	FuenteCoste   string
	FaseRetorno   ClaveFase
	Observaciones string
}

func (d DatosModificacionTrasNombramiento) Validar() error {
	if !d.MotivoClave.Valida() || !periodoAnalisisValido(d.Periodo) || d.Jornada.Validar() != nil ||
		!importeCalculable(d.Coste) || !referenciaValida(d.FuenteCoste) ||
		(d.FaseRetorno != FaseFiscalizacion && d.FaseRetorno != FaseInformeJuridico) ||
		!textoValido(d.Observaciones, 2000, false) {
		return ErrModificacionInvalida
	}
	return nil
}

func (e Expediente) enNombramientoVigente() bool {
	return e.FaseActual == FaseNombramiento && e.EstadoActual == EstadoEnCurso &&
		e.Asignacion != nil && e.Analisis != nil
}

// TieneAccion indica si alguna actuación del expediente registró la acción.
func (e Expediente) TieneAccion(accion ClaveCatalogo) bool {
	for _, actuacion := range e.Actuaciones {
		if actuacion.AccionClave == accion {
			return true
		}
	}
	return false
}

// RegistrarCese añade la actuación de cese. Conserva fase y estado: el
// expediente se cierra después, cuando se cumplen las condiciones de cierre.
func (e Expediente) RegistrarCese(versionEsperada uint64, datos DatosCese, actuacion DatosActuacion) (Expediente, error) {
	if e.Validar() != nil || datos.Validar() != nil || actuacion.validar() != nil ||
		!e.enNombramientoVigente() || e.TieneAccion(AccionCesarNombramiento) ||
		actuacion.AccionClave != AccionCesarNombramiento ||
		actuacion.FaseDestino != FaseNombramiento || actuacion.EstadoDestino != EstadoEnCurso ||
		actuacion.UnidadRef != e.Asignacion.UnidadRef || actuacion.Observaciones != datos.Observaciones ||
		len(actuacion.DocumentosRef) != 1 || actuacion.DocumentosRef[0] != datos.JustificanteRef ||
		actuacion.RetornoRef != "" {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	return siguiente.confirmarTransicion(actuacion)
}

// CerrarTrasCese deja el expediente completado. Exige el cese registrado; la
// confirmación de GINPIX la exige la regla del catálogo.
func (e Expediente) CerrarTrasCese(versionEsperada uint64, datos DatosCierreExpediente, actuacion DatosActuacion) (Expediente, error) {
	documentos := []string(nil)
	if ref := datos.DocumentoGINPIX(); ref != "" {
		documentos = []string{ref}
	}
	if e.Validar() != nil || datos.Validar() != nil || actuacion.validar() != nil ||
		!e.enNombramientoVigente() || !e.TieneAccion(AccionCesarNombramiento) ||
		actuacion.AccionClave != AccionCerrarExpediente ||
		actuacion.FaseDestino != FaseNombramiento || actuacion.EstadoDestino != EstadoCompletado ||
		actuacion.UnidadRef != e.Asignacion.UnidadRef || actuacion.Observaciones != datos.Observaciones ||
		len(actuacion.DocumentosRef) != len(documentos) || (len(documentos) == 1 && actuacion.DocumentosRef[0] != documentos[0]) ||
		actuacion.RetornoRef != "" {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	return siguiente.confirmarTransicion(actuacion)
}

// ModificarTrasNombramiento crea la versión nueva del análisis con el periodo,
// la jornada y el coste recalculados y devuelve el expediente a la fase del
// catálogo. Retira de la proyección vigente la fiscalización (y el informe si
// se vuelve a informe jurídico) del análisis anterior; su historia permanece
// en las actuaciones y en la versión anterior del expediente.
func (e Expediente) ModificarTrasNombramiento(versionEsperada uint64, datos DatosModificacionTrasNombramiento, actuacion DatosActuacion) (Expediente, error) {
	if e.Validar() != nil || datos.Validar() != nil || actuacion.validar() != nil ||
		!e.enNombramientoVigente() || e.InformeJuridico == nil || e.TieneAccion(AccionCesarNombramiento) ||
		actuacion.AccionClave != AccionModificarTrasNombramiento ||
		actuacion.FaseDestino != datos.FaseRetorno || actuacion.EstadoDestino != EstadoEnCurso ||
		actuacion.UnidadRef != e.Asignacion.UnidadRef || actuacion.Observaciones != datos.Observaciones ||
		len(actuacion.DocumentosRef) != 0 || actuacion.RetornoRef != "" ||
		(e.Analisis.Periodo == datos.Periodo && e.Analisis.PorcentajeJornada == datos.Jornada) {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	analisis := e.Analisis.clonar()
	analisis.Periodo = datos.Periodo
	analisis.PorcentajeJornada = datos.Jornada
	coste := datos.Coste
	analisis.CostePrevisto = &coste
	analisis.FuenteCosteRef = datos.FuenteCoste
	vinculo := nuevoVinculoActuacionAnalisis(e.Version+1, uint64(len(e.Actuaciones)+1), actuacion)
	analisis.ActuacionRegistro = &vinculo
	if analisis.Validar() != nil {
		return Expediente{}, ErrModificacionInvalida
	}
	siguiente.Analisis = &analisis
	siguiente.Fiscalizacion = nil
	if datos.FaseRetorno == FaseInformeJuridico {
		siguiente.InformeJuridico = nil
	}
	return siguiente.confirmarTransicion(actuacion)
}
