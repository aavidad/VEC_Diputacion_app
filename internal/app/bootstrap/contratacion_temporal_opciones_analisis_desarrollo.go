package bootstrap

import (
	"context"
	"crypto/hmac"
	"errors"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Prefijos del catálogo de reglas de Contratación temporal con las opciones
// del análisis. Cada entrada con el prefijo es una opción: añadir una modalidad,
// una causa o una entrada de retención de crédito es publicar otra entrada, sin
// cambiar código. Sin ninguna entrada de un prefijo rige la opción de siempre.
const (
	prefijoModalidadAnalisisCT = reglas.CTPrefijoModalidad
	prefijoCausaAnalisisCT     = reglas.CTPrefijoCausa
	prefijoEntradaRCAnalisisCT = reglas.CTPrefijoEntradaRC
	// reglaUrgenciaAnalisisCT habilita la marca de urgencia del análisis
	// (duda 63). Sin ella el formulario no la ofrece, como hasta ahora.
	reglaUrgenciaAnalisisCT = reglas.CTUrgencia

	atributoDuracionReglaCT    = "duracion_regla"
	atributoAlSuperarCT        = "al_superar"
	alSuperarBloquearCT        = "bloquear"
	alSuperarAvisarCT          = "avisar"
	maximoOpcionesAnalisisCT   = 100
	maximoCentimosRCAnalisis   = int64(1) << 50
	formatoFechaRCAnalisisCT   = time.DateOnly
	maximoCaracteresEtiquetaCT = 160
)

var errOpcionesAnalisisNoValidas = errors.New("bootstrap: opciones del análisis del catálogo de reglas no válidas")

// duracionMaximaAnalisisCT es el máximo de una modalidad según su regla c08.
// El periodo lo supera cuando su fin llega al mismo día del mes (o año, o día)
// de vencimiento contado desde el inicio, como en el cómputo civil de fecha a
// fecha: nueve meses desde el 1 de enero terminan como tarde el 30 de
// septiembre.
type duracionMaximaAnalisisCT struct {
	Unidad   reglas.Unidad
	Cantidad int
	Bloquear bool
	ReglaRef string
}

type modalidadAnalisisCT struct {
	Clave    domain.ClaveCatalogo
	Etiqueta string
	Duracion *duracionMaximaAnalisisCT
}

type entradaRCAnalisisCT struct {
	Referencia  string
	Huella      string
	Etiqueta    string
	Declaracion domain.DeclaracionRC
}

// opcionesAnalisisCTDesarrollo son las opciones cerradas del análisis RRHH.
// Es inmutable tras construirse; se comparte entre la configuración servida a
// la web y la validación de cada solicitud, para que ambas digan lo mismo.
type opcionesAnalisisCTDesarrollo struct {
	modalidades        []modalidadAnalisisCT
	causas             []opcionClaveCatalogosAltaContratacionTemporalDesarrollo
	entradasRC         []entradaRCAnalisisCT
	urgenciaDisponible bool
}

// opcionesAnalisisPredeterminadas son los valores anteriores al catálogo:
// cinco modalidades sin máximo, una causa y la retención de crédito sintética.
func opcionesAnalisisPredeterminadas() *opcionesAnalisisCTDesarrollo {
	return &opcionesAnalisisCTDesarrollo{
		modalidades: []modalidadAnalisisCT{
			{Clave: "sustitucion", Etiqueta: "Sustitución"},
			{Clave: "vacante", Etiqueta: "Vacante"},
			{Clave: "acumulacion_tareas", Etiqueta: "Acumulación de tareas"},
			{Clave: "programa", Etiqueta: "Programa"},
			{Clave: "relevo", Etiqueta: "Relevo"},
		},
		causas: []opcionClaveCatalogosAltaContratacionTemporalDesarrollo{{
			Clave: string(causaAnalisisContratacionTemporalDesarrollo), Etiqueta: "Necesidad temporal",
		}},
		entradasRC: []entradaRCAnalisisCT{{
			Referencia:  entradaRCAnalisisContratacionTemporalDesarrollo,
			Huella:      huellaEntradaRCAnalisisContratacionTemporalDesarrollo,
			Etiqueta:    "Retención de crédito sintética 001",
			Declaracion: declaracionRCAnalisisContratacionTemporalDesarrollo(),
		}},
	}
}

// nuevasOpcionesAnalisisCT resuelve las opciones con el catálogo de reglas. Sin
// catálogo devuelve las predeterminadas; un catálogo declarado con opciones
// mal formadas impide arrancar en lugar de mostrarse a medias.
func nuevasOpcionesAnalisisCT(ctx context.Context, resolutor *reglas.Resolutor) (*opcionesAnalisisCTDesarrollo, error) {
	vigentes, err := resolutor.Reglas(ctx)
	if errors.Is(err, reglas.ErrReglasNoConfiguradas) {
		return opcionesAnalisisPredeterminadas(), nil
	}
	if err != nil {
		return nil, errors.Join(errOpcionesAnalisisNoValidas, err)
	}
	porClave := make(map[string]reglas.Regla, len(vigentes))
	for _, regla := range vigentes {
		porClave[regla.Clave] = regla
	}
	opciones := opcionesAnalisisPredeterminadas()
	var modalidades []modalidadAnalisisCT
	var causas []opcionClaveCatalogosAltaContratacionTemporalDesarrollo
	var entradas []entradaRCAnalisisCT
	for _, regla := range vigentes {
		switch {
		case strings.HasPrefix(regla.Clave, prefijoModalidadAnalisisCT):
			modalidad, err := modalidadDesdeReglaCT(regla, porClave)
			if err != nil {
				return nil, err
			}
			modalidades = append(modalidades, modalidad)
		case strings.HasPrefix(regla.Clave, prefijoCausaAnalisisCT):
			clave, err := claveOpcionDesdeReglaCT(regla, prefijoCausaAnalisisCT)
			if err != nil {
				return nil, err
			}
			causas = append(causas, opcionClaveCatalogosAltaContratacionTemporalDesarrollo{
				Clave: string(clave), Etiqueta: regla.Etiqueta,
			})
		case strings.HasPrefix(regla.Clave, prefijoEntradaRCAnalisisCT):
			entrada, err := entradaRCDesdeReglaCT(regla)
			if err != nil {
				return nil, err
			}
			entradas = append(entradas, entrada)
		case regla.Clave == reglaUrgenciaAnalisisCT:
			opciones.urgenciaDisponible = true
		}
	}
	if len(modalidades) > 0 {
		opciones.modalidades = modalidades
	}
	if len(causas) > 0 {
		opciones.causas = causas
	}
	if len(entradas) > 0 {
		opciones.entradasRC = entradas
	}
	if !opciones.unicas() {
		return nil, errOpcionesAnalisisNoValidas
	}
	return opciones, nil
}

func claveOpcionDesdeReglaCT(regla reglas.Regla, prefijo string) (domain.ClaveCatalogo, error) {
	clave := domain.ClaveCatalogo(strings.TrimPrefix(regla.Clave, prefijo))
	if !clave.Valida() || !etiquetaOpcionAnalisisValida(regla.Etiqueta) {
		return "", errOpcionesAnalisisNoValidas
	}
	return clave, nil
}

func etiquetaOpcionAnalisisValida(etiqueta string) bool {
	return etiqueta != "" && etiqueta == strings.TrimSpace(etiqueta) &&
		len([]rune(etiqueta)) <= maximoCaracteresEtiquetaCT
}

func modalidadDesdeReglaCT(regla reglas.Regla, porClave map[string]reglas.Regla) (modalidadAnalisisCT, error) {
	clave, err := claveOpcionDesdeReglaCT(regla, prefijoModalidadAnalisisCT)
	if err != nil {
		return modalidadAnalisisCT{}, err
	}
	modalidad := modalidadAnalisisCT{Clave: clave, Etiqueta: regla.Etiqueta}
	referencia, conDuracion := regla.Atributos[atributoDuracionReglaCT]
	if !conDuracion {
		return modalidad, nil
	}
	duracion, existe := porClave[referencia]
	if !existe {
		// Una modalidad que cita una regla de duración inexistente es un
		// catálogo roto, no una modalidad sin máximo.
		return modalidadAnalisisCT{}, errOpcionesAnalisisNoValidas
	}
	switch duracion.Unidad {
	case reglas.UnidadMeses, reglas.UnidadAnios, reglas.UnidadDiasNaturales:
	case reglas.UnidadNinguna:
		// La regla existe pero no fija un máximo (sustitución: hasta la
		// reincorporación).
		return modalidad, nil
	default:
		return modalidadAnalisisCT{}, errOpcionesAnalisisNoValidas
	}
	alSuperar := duracion.Atributos[atributoAlSuperarCT]
	if alSuperar != "" && alSuperar != alSuperarAvisarCT && alSuperar != alSuperarBloquearCT {
		return modalidadAnalisisCT{}, errOpcionesAnalisisNoValidas
	}
	modalidad.Duracion = &duracionMaximaAnalisisCT{
		Unidad: duracion.Unidad, Cantidad: duracion.Cantidad,
		Bloquear: alSuperar == alSuperarBloquearCT, ReglaRef: duracion.Referencia,
	}
	return modalidad, nil
}

func entradaRCDesdeReglaCT(regla reglas.Regla) (entradaRCAnalisisCT, error) {
	a := regla.Atributos
	fecha, errFecha := time.Parse(formatoFechaRCAnalisisCT, a["fecha"])
	centimos, errImporte := strconv.ParseInt(a["importe_centimos"], 10, 64)
	entrada := entradaRCAnalisisCT{
		Referencia: a["referencia"], Huella: a["huella_sha256"], Etiqueta: regla.Etiqueta,
		Declaracion: domain.DeclaracionRC{
			Existe: true, Numero: a["numero"], Fecha: fecha.UTC(),
			Importe:      domain.Importe{Centimos: centimos, Moneda: "EUR"},
			DocumentoRef: a["documento_ref"],
		},
	}
	vinculo := domain.VinculoEntradaRC{Referencia: entrada.Referencia, HuellaSHA256: entrada.Huella}
	if errFecha != nil || errImporte != nil || centimos < 1 || centimos > maximoCentimosRCAnalisis ||
		strconv.FormatInt(centimos, 10) != a["importe_centimos"] ||
		vinculo.Validar() != nil || entrada.Declaracion.Validar() != nil ||
		!etiquetaOpcionAnalisisValida(entrada.Etiqueta) {
		return entradaRCAnalisisCT{}, errOpcionesAnalisisNoValidas
	}
	return entrada, nil
}

func (o *opcionesAnalisisCTDesarrollo) unicas() bool {
	if len(o.modalidades) > maximoOpcionesAnalisisCT || len(o.causas) > maximoOpcionesAnalisisCT ||
		len(o.entradasRC) > maximoOpcionesAnalisisCT {
		return false
	}
	vistas := make(map[string]bool)
	for _, modalidad := range o.modalidades {
		if vistas["m:"+string(modalidad.Clave)] {
			return false
		}
		vistas["m:"+string(modalidad.Clave)] = true
	}
	for _, causa := range o.causas {
		if vistas["c:"+causa.Clave] {
			return false
		}
		vistas["c:"+causa.Clave] = true
	}
	for _, entrada := range o.entradasRC {
		if vistas["r:"+entrada.Referencia] {
			return false
		}
		vistas["r:"+entrada.Referencia] = true
	}
	return true
}

func (o *opcionesAnalisisCTDesarrollo) modalidad(clave domain.ClaveCatalogo) (modalidadAnalisisCT, bool) {
	for _, modalidad := range o.modalidades {
		if modalidad.Clave == clave {
			return modalidad, true
		}
	}
	return modalidadAnalisisCT{}, false
}

func (o *opcionesAnalisisCTDesarrollo) causaValida(clave domain.ClaveCatalogo) bool {
	for _, causa := range o.causas {
		if causa.Clave == string(clave) {
			return true
		}
	}
	return false
}

// entradaRC busca la entrada por referencia y exige su huella exacta.
func (o *opcionesAnalisisCTDesarrollo) entradaRC(vinculo domain.VinculoEntradaRC) (entradaRCAnalisisCT, bool) {
	for _, entrada := range o.entradasRC {
		if entrada.Referencia == vinculo.Referencia {
			return entrada, hmac.Equal([]byte(entrada.Huella), []byte(vinculo.HuellaSHA256))
		}
	}
	return entradaRCAnalisisCT{}, false
}

// primeraEntradaRC es la entrada con la que se comprueba la forma de una
// solicitud de coste, que no lleva RC propia.
func (o *opcionesAnalisisCTDesarrollo) primeraEntradaRC() entradaRCAnalisisCT {
	return o.entradasRC[0]
}

// superaDuracion indica si el periodo alcanza el vencimiento del máximo de la
// modalidad. El periodo son días civiles (medianoche UTC) y se cuenta de fecha
// a fecha como el artículo 30.4 de la Ley 39/2015: si el mes de vencimiento no
// tiene ese día, vale su último día. Sin máximo, no lo supera.
func (d *duracionMaximaAnalisisCT) superaDuracion(periodo domain.PeriodoPrevisto) bool {
	if d == nil || d.Cantidad < 1 {
		return false
	}
	inicio := periodo.Inicio.UTC()
	var vencimiento time.Time
	switch d.Unidad {
	case reglas.UnidadMeses:
		vencimiento = sumarMesesMismoDiaCT(inicio, d.Cantidad)
	case reglas.UnidadAnios:
		vencimiento = sumarMesesMismoDiaCT(inicio, 12*d.Cantidad)
	case reglas.UnidadDiasNaturales:
		vencimiento = time.Date(inicio.Year(), inicio.Month(), inicio.Day()+d.Cantidad, 0, 0, 0, 0, time.UTC)
	default:
		return false
	}
	return !periodo.Fin.UTC().Before(vencimiento)
}

func sumarMesesMismoDiaCT(inicio time.Time, meses int) time.Time {
	total := inicio.Year()*12 + int(inicio.Month()) - 1 + meses
	anio, mes := total/12, time.Month(total%12+1)
	ultimoDia := time.Date(anio, mes+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return time.Date(anio, mes, min(inicio.Day(), ultimoDia), 0, 0, 0, 0, time.UTC)
}

// opcionesAnalisis devuelve las opciones compuestas o, sin ellas, las de
// siempre. Un catálogo nulo también tiene las de siempre.
func (c *catalogosAltaContratacionTemporalDesarrollo) opcionesAnalisis() *opcionesAnalisisCTDesarrollo {
	if c == nil || c.analisis == nil {
		return opcionesAnalisisPredeterminadas()
	}
	return c.analisis
}

// componerOpcionesAnalisis fija las opciones resueltas del catálogo de reglas.
// Se llama una sola vez al componer, antes de servir.
func (c *catalogosAltaContratacionTemporalDesarrollo) componerOpcionesAnalisis(opciones *opcionesAnalisisCTDesarrollo) {
	if c != nil && opciones != nil {
		c.analisis = opciones
	}
}
