package reglas

import (
	"context"
	"errors"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecreglas "vec-diputacion-granada/internal/vec/reglas"
)

// Atributos propios de las reglas de intentos. Sus valores viven en el
// catálogo; aquí solo se nombran.
const (
	atributoControl               = "control"
	atributoResultadosSinContacto = "resultados_sin_contacto"
	atributoDias                  = "dias"
	atributoZonaHoraria           = "zona_horaria"
	diasHabiles                   = "habiles"
	diasTodos                     = "todos"
	// zonaPeninsular es la hora oficial en la que el resolutor común ya
	// computa todos los plazos; el catálogo puede fijar otra.
	zonaPeninsular = "Europe/Madrid"
)

var errReglasIntentos = errors.New("bolsa: reglas de intentos de contacto no validas")

var _ puertosbolsa.PoliticaIntentosContacto = (*IntentosContacto)(nil)

// IntentosContacto traduce b02, b03 y b04 (y muestra b06). Es válido con
// resolutor nulo: responde «no configurado».
type IntentosContacto struct {
	resolutor *vecreglas.Resolutor
}

func NuevosIntentosContacto(resolutor *vecreglas.Resolutor) *IntentosContacto {
	return &IntentosContacto{resolutor: resolutor}
}

func (r *IntentosContacto) Configurada() bool { return r != nil && r.resolutor.Disponible() }

// PoliticaIntentosTelefonicos devuelve la política vigente. Sin catálogo o
// sin la regla b02.intentos_contacto responde no configurada; con ella, las
// demás reglas que necesita deben estar completas.
func (r *IntentosContacto) PoliticaIntentosTelefonicos(ctx context.Context) (dominiobolsa.PoliticaIntentosTelefonicos, []puertosbolsa.ReglaIntentosContacto, bool, error) {
	var vacia dominiobolsa.PoliticaIntentosTelefonicos
	if !r.Configurada() {
		return vacia, nil, false, nil
	}
	todas, err := r.resolutor.Reglas(ctx)
	if errors.Is(err, vecreglas.ErrReglasNoConfiguradas) {
		return vacia, nil, false, nil
	}
	if err != nil {
		return vacia, nil, false, err
	}
	porClave := make(map[string]vecreglas.Regla, len(todas))
	for _, regla := range todas {
		porClave[regla.Clave] = regla
	}
	intentos, ok := porClave[vecreglas.BolsaIntentosContacto]
	if !ok {
		return vacia, nil, false, nil
	}
	separacion, okSeparacion := porClave[vecreglas.BolsaSeparacionIntentos]
	procesos, okProcesos := porClave[vecreglas.BolsaProcesosSinContacto]
	if !okSeparacion || !okProcesos || intentos.Unidad != vecreglas.UnidadIntentos || separacion.Unidad != vecreglas.UnidadHoras ||
		procesos.Unidad != vecreglas.UnidadProcesos {
		return vacia, nil, false, errReglasIntentos
	}
	politica := dominiobolsa.PoliticaIntentosTelefonicos{
		IntentosPorProceso: intentos.Cantidad, Procesos: procesos.Cantidad,
		SeparacionMinima:      time.Duration(separacion.Cantidad) * time.Hour,
		ControlSeparacion:     separacion.Atributos[atributoControl],
		ResultadosSinContacto: listaAtributo(procesos.Atributos[atributoResultadosSinContacto]),
	}
	mostradas := []vecreglas.Regla{intentos, separacion, procesos}
	if franja, ok := porClave[vecreglas.BolsaFranjaLlamadas]; ok {
		if politica.Franja, err = franjaDesdeRegla(franja); err != nil {
			return vacia, nil, false, err
		}
		mostradas = append(mostradas, franja)
	}
	if correo, ok := porClave[vecreglas.BolsaCorreoNoAbrePlazo]; ok {
		mostradas = append(mostradas, correo)
	}
	if politica.Validar() != nil {
		return vacia, nil, false, errReglasIntentos
	}
	reglas := make([]puertosbolsa.ReglaIntentosContacto, 0, len(mostradas))
	for _, regla := range mostradas {
		reglas = append(reglas, puertosbolsa.ReglaIntentosContacto{
			Clave: regla.Clave, Etiqueta: regla.Etiqueta, Descripcion: regla.Descripcion,
			Referencia: regla.Referencia, Ejemplo: regla.EsEjemplo(),
		})
	}
	return politica, reglas, true, nil
}

func franjaDesdeRegla(regla vecreglas.Regla) (dominiobolsa.FranjaLlamadas, error) {
	var franja dominiobolsa.FranjaLlamadas
	if regla.Unidad != vecreglas.UnidadFranjaHoraria {
		return franja, errReglasIntentos
	}
	desde, hasta, ok := strings.Cut(regla.Valor, "-")
	inicio, errInicio := time.Parse("15:04", desde)
	fin, errFin := time.Parse("15:04", hasta)
	if !ok || errInicio != nil || errFin != nil {
		return franja, errReglasIntentos
	}
	zona := regla.Atributos[atributoZonaHoraria]
	if zona == "" {
		zona = zonaPeninsular
	}
	ubicacion, err := time.LoadLocation(zona)
	if err != nil {
		return franja, errReglasIntentos
	}
	switch regla.Atributos[atributoDias] {
	case diasHabiles:
		franja.SoloDiasHabiles = true
	case diasTodos, "":
	default:
		return franja, errReglasIntentos
	}
	franja.DesdeMinuto = inicio.Hour()*60 + inicio.Minute()
	franja.HastaMinuto = fin.Hour()*60 + fin.Minute()
	franja.Zona = ubicacion
	franja.Control = regla.Atributos[atributoControl]
	franja.Texto = regla.Valor
	return franja, nil
}

func listaAtributo(valor string) []string {
	if strings.TrimSpace(valor) == "" {
		return nil
	}
	partes := strings.Split(valor, ",")
	for i := range partes {
		partes[i] = strings.TrimSpace(partes[i])
	}
	return partes
}
