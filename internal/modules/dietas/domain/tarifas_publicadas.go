package domain

import (
	"regexp"
	"strings"
	"time"
)

var versionTarifaPublicada = regexp.MustCompile(`^tarifa:[A-Za-z0-9:_-]{3,120}$`)
var referenciaReglaPublicada = regexp.MustCompile(`^regla:[A-Za-z0-9:_-]{3,120}$`)

func VersionTarifaPublicadaValida(version string) bool {
	return versionTarifaPublicada.MatchString(version) && !strings.Contains(strings.ToLower(version), "provisional")
}

func VersionTarifaComisionValida(version string) bool {
	return VersionTarifaProvisionalValida(version) || VersionTarifaPublicadaValida(version)
}

func metadatosTarifaComisionValidos(version, rotulo, reglaRef, reglaHuella string) bool {
	if VersionTarifaProvisionalValida(version) {
		return rotulo == RotuloTarifaProvisional &&
			(reglaRef == "" && reglaHuella == "" || referenciaReglaProvisional.MatchString(reglaRef) && huellaReglaProvisional.MatchString(reglaHuella))
	}
	return VersionTarifaPublicadaValida(version) && rotulo == "dietas_tarifa_publicada" &&
		referenciaReglaPublicada.MatchString(reglaRef) && huellaJustificante.MatchString(reglaHuella)
}

// ConfiguracionDevengoPublicada es dato del catálogo publicado. El esquema
// nacional ordinario comparte la aritmética civil con los expedientes antiguos;
// sus umbrales y porcentajes proceden de esta publicación versionada.
type ConfiguracionDevengoPublicada struct {
	Esquema                      string `json:"esquema"`
	Zona                         string `json:"zona"`
	DuracionMinimaMismoDiaHoras  int    `json:"duracion_minima_mismo_dia_horas"`
	HoraSalida100AntesDe         int    `json:"hora_salida_100_antes_de"`
	HoraSalida50AntesDe          int    `json:"hora_salida_50_antes_de"`
	HoraRegreso50DespuesDe       int    `json:"hora_regreso_50_despues_de"`
	HoraRegresoMismoDiaDespuesDe int    `json:"hora_regreso_mismo_dia_despues_de"`
	DiasMaximos                  int    `json:"dias_maximos"`
	PorcentajeMismoDia           int    `json:"porcentaje_mismo_dia"`
	PorcentajeSalidaTemprana     int    `json:"porcentaje_salida_temprana"`
	PorcentajeSalidaMedia        int    `json:"porcentaje_salida_media"`
	PorcentajeRegreso            int    `json:"porcentaje_regreso"`
	PorcentajeIntermedio         int    `json:"porcentaje_intermedio"`
	Alojamiento                  string `json:"alojamiento"`
}

type ReglaDevengoPublicada struct {
	Referencia    string                        `json:"referencia"`
	NormaRef      string                        `json:"norma_ref"`
	HuellaSHA256  string                        `json:"huella_sha256"`
	Configuracion ConfiguracionDevengoPublicada `json:"configuracion"`
}

func (r ReglaDevengoPublicada) HuellaCalculada() (string, error) {
	r.HuellaSHA256 = ""
	return HuellaDatosLiquidacion(r)
}

func (r ReglaDevengoPublicada) Validar() error {
	c := r.Configuracion
	if !referenciaReglaPublicada.MatchString(r.Referencia) || strings.Contains(strings.ToLower(r.Referencia), "provisional") ||
		!referenciaJustificante.MatchString(r.NormaRef) || !huellaJustificante.MatchString(r.HuellaSHA256) ||
		c.Esquema != "nacional_ordinaria_v1" || c.Zona != "Europe/Madrid" || c.Alojamiento != "cerrado_pendiente_rrhh" ||
		c.DuracionMinimaMismoDiaHoras < 1 || c.DuracionMinimaMismoDiaHoras > 24 || c.DiasMaximos < 1 || c.DiasMaximos > 31 ||
		!horaCatalogo(c.HoraSalida100AntesDe) || !horaCatalogo(c.HoraSalida50AntesDe) ||
		!horaCatalogo(c.HoraRegreso50DespuesDe) || !horaCatalogo(c.HoraRegresoMismoDiaDespuesDe) ||
		c.HoraSalida100AntesDe >= c.HoraSalida50AntesDe || c.HoraRegreso50DespuesDe >= c.HoraRegresoMismoDiaDespuesDe ||
		!porcentajeCatalogo(c.PorcentajeMismoDia) || !porcentajeCatalogo(c.PorcentajeSalidaTemprana) ||
		!porcentajeCatalogo(c.PorcentajeSalidaMedia) || !porcentajeCatalogo(c.PorcentajeRegreso) || !porcentajeCatalogo(c.PorcentajeIntermedio) {
		return ErrLiquidacionPublicada
	}
	h, err := r.HuellaCalculada()
	if err != nil || h != r.HuellaSHA256 {
		return ErrLiquidacionPublicada
	}
	return nil
}

// CalcularTramosNacionalesPublicados no consulta otra autoridad. La cuantía
// completa del grupo y la regla de devengo salen de la misma fila publicada.
// Conserva el formato de cálculo del documento para lectura histórica.
func CalcularTramosNacionalesPublicados(inicio, fin time.Time, catalogo CatalogoLiquidacionPublicado) (CalculoDietasProvisional, error) {
	var vacio CalculoDietasProvisional
	if catalogo.ValidarForma() != nil || catalogo.PaisISO2 != "ES" || fin.IsZero() || inicio.IsZero() || !fin.After(inicio) {
		return vacio, ErrLiquidacionPublicada
	}
	zona, err := time.LoadLocation(catalogo.ReglaDevengo.Configuracion.Zona)
	if err != nil {
		return vacio, ErrLiquidacionPublicada
	}
	fechaInicio, fechaFin := inicio.In(zona).Format("2006-01-02"), fin.In(zona).Format("2006-01-02")
	if fechaInicio < catalogo.VigenteDesde || fechaFin > catalogo.VigenteHasta {
		return vacio, ErrLiquidacionPublicada
	}
	var base int64
	for _, regla := range catalogo.Reglas {
		if regla.Tipo == "dieta" && regla.Concepto == "manutencion" && regla.Grupo == catalogo.Grupo {
			if base != 0 {
				return vacio, ErrLiquidacionPublicada
			}
			base = regla.TopeCentimos
		}
	}
	if base < 1 || base > 1000000 {
		return vacio, ErrLiquidacionPublicada
	}
	c := catalogo.ReglaDevengo.Configuracion
	return calcularTramosNacionales(inicio, fin, zona, catalogo.VersionTarifaRef, catalogo.RotuloClave,
		catalogo.ReglaDevengo.Referencia, catalogo.ReglaDevengo.HuellaSHA256, base, 0,
		parametrosTramosNacionales{DuracionMinimaMismoDiaHoras: c.DuracionMinimaMismoDiaHoras,
			HoraSalida100AntesDe: c.HoraSalida100AntesDe, HoraSalida50AntesDe: c.HoraSalida50AntesDe,
			HoraRegreso50DespuesDe: c.HoraRegreso50DespuesDe, HoraRegresoMismoDiaDespuesDe: c.HoraRegresoMismoDiaDespuesDe,
			DiasMaximos: c.DiasMaximos, PorcentajeMismoDia: c.PorcentajeMismoDia,
			PorcentajeSalidaTemprana: c.PorcentajeSalidaTemprana, PorcentajeSalidaMedia: c.PorcentajeSalidaMedia,
			PorcentajeRegreso: c.PorcentajeRegreso, PorcentajeIntermedio: c.PorcentajeIntermedio})
}
