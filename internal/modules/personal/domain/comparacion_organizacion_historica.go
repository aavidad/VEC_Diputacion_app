package domain

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var ErrComparacionOrganizacionHistoricaInvalida = errors.New("personal: comparacion de organizacion historica invalida")

const LimiteHechosComparacionOrganizacion = 10000

// Una instantánea reúne todas las páginas del mismo corte. El cursor debe estar
// vacío; completa describe la cobertura de la fuente, no una página aislada.
type InstantaneaComparacionOrganizacion struct {
	Selector            SelectorOrganizacionHistorica           `json:"selector"`
	Cobertura           CoberturaComparacionOrganizacion        `json:"cobertura"`
	Unidades            []UnidadOrganizacionHistorica           `json:"unidades"`
	PuestosTipo         []PuestoTipoOrganizacionHistorica       `json:"puestos_tipo"`
	Dotaciones          []DotacionOrganizacionHistorica         `json:"dotaciones"`
	Plazas              []PlazaOrganizacionHistorica            `json:"plazas"`
	PuestosIndividuales []PuestoIndividualOrganizacionHistorica `json:"puestos_individuales"`
	Vinculos            []VinculoPlazaPuestoHistorico           `json:"vinculos"`
}

type CoberturaComparacionOrganizacion struct {
	Unidades            string `json:"unidades"`
	PuestosTipo         string `json:"puestos_tipo"`
	Dotaciones          string `json:"dotaciones"`
	Plazas              string `json:"plazas"`
	PuestosIndividuales string `json:"puestos_individuales"`
	Vinculos            string `json:"vinculos"`
}

type ComparacionOrganizacionHistorica struct {
	SelectorAntes       SelectorOrganizacionHistorica    `json:"selector_antes"`
	SelectorDespues     SelectorOrganizacionHistorica    `json:"selector_despues"`
	CoberturaAntes      CoberturaComparacionOrganizacion `json:"cobertura_antes"`
	CoberturaDespues    CoberturaComparacionOrganizacion `json:"cobertura_despues"`
	Unidades            ColeccionComparacionOrganizacion `json:"unidades"`
	PuestosTipo         ColeccionComparacionOrganizacion `json:"puestos_tipo"`
	Dotaciones          ColeccionComparacionOrganizacion `json:"dotaciones"`
	Plazas              ColeccionComparacionOrganizacion `json:"plazas"`
	PuestosIndividuales ColeccionComparacionOrganizacion `json:"puestos_individuales"`
	Vinculos            ColeccionComparacionOrganizacion `json:"vinculos"`
}

type ColeccionComparacionOrganizacion struct {
	Estado         string                          `json:"estado"`
	Antes          int                             `json:"antes"`
	Despues        int                             `json:"despues"`
	Cambios        []CambioComparacionOrganizacion `json:"cambios"`
	NoVerificables []string                        `json:"no_verificables"`
}

// Entrada y salida describen presencia en los cortes, sin declarar creación,
// amortización ni supresión administrativa. Procedencia separa cambios de
// traza o versión documental de cambios en los campos estructurales.
type CambioComparacionOrganizacion struct {
	Tipo         string                      `json:"tipo"`
	ID           string                      `json:"id"`
	Campos       []string                    `json:"campos"`
	TrazaAntes   *TrazaOrganizacionHistorica `json:"traza_antes,omitempty"`
	TrazaDespues *TrazaOrganizacionHistorica `json:"traza_despues,omitempty"`
}

func CompararOrganizacionHistorica(antes, despues InstantaneaComparacionOrganizacion) (ComparacionOrganizacionHistorica, error) {
	if !instantaneaComparacionValida(antes) || !instantaneaComparacionValida(despues) ||
		antes.Selector.OrganismoRef != despues.Selector.OrganismoRef || antes.Selector.UnidadClave != despues.Selector.UnidadClave {
		return ComparacionOrganizacionHistorica{}, ErrComparacionOrganizacionHistoricaInvalida
	}
	return ComparacionOrganizacionHistorica{
		SelectorAntes: antes.Selector, SelectorDespues: despues.Selector,
		CoberturaAntes: antes.Cobertura, CoberturaDespues: despues.Cobertura,
		Unidades:            compararColeccionOrganizacion(antes.Unidades, despues.Unidades, antes.Cobertura.Unidades, despues.Cobertura.Unidades),
		PuestosTipo:         compararColeccionOrganizacion(antes.PuestosTipo, despues.PuestosTipo, antes.Cobertura.PuestosTipo, despues.Cobertura.PuestosTipo),
		Dotaciones:          compararColeccionOrganizacion(antes.Dotaciones, despues.Dotaciones, antes.Cobertura.Dotaciones, despues.Cobertura.Dotaciones),
		Plazas:              compararColeccionOrganizacion(antes.Plazas, despues.Plazas, antes.Cobertura.Plazas, despues.Cobertura.Plazas),
		PuestosIndividuales: compararColeccionOrganizacion(antes.PuestosIndividuales, despues.PuestosIndividuales, antes.Cobertura.PuestosIndividuales, despues.Cobertura.PuestosIndividuales),
		Vinculos:            compararColeccionOrganizacion(antes.Vinculos, despues.Vinculos, antes.Cobertura.Vinculos, despues.Cobertura.Vinculos),
	}, nil
}

func instantaneaComparacionValida(i InstantaneaComparacionOrganizacion) bool {
	s, c := i.Selector, i.Cobertura
	if s.Validar() != nil || s.Cursor != "" || len(i.Unidades)+len(i.PuestosTipo)+len(i.Dotaciones)+len(i.Plazas)+len(i.PuestosIndividuales)+len(i.Vinculos) > LimiteHechosComparacionOrganizacion {
		return false
	}
	for _, x := range []struct {
		cobertura string
		cantidad  int
	}{
		{c.Unidades, len(i.Unidades)}, {c.PuestosTipo, len(i.PuestosTipo)}, {c.Dotaciones, len(i.Dotaciones)},
		{c.Plazas, len(i.Plazas)}, {c.PuestosIndividuales, len(i.PuestosIndividuales)}, {c.Vinculos, len(i.Vinculos)},
	} {
		if x.cobertura != "completa" && x.cobertura != "parcial" && x.cobertura != "sin_datos" || x.cobertura == "sin_datos" && x.cantidad != 0 {
			return false
		}
	}
	if s.VersionRPTRef == "" && (c.PuestosTipo != "sin_datos" || c.Dotaciones != "sin_datos" || c.PuestosIndividuales != "sin_datos") ||
		s.VersionPlantillaRef == "" && c.Plazas != "sin_datos" {
		return false
	}
	if !trazasComparacionValidas(i.Unidades, s) || !trazasComparacionValidas(i.PuestosTipo, s) || !trazasComparacionValidas(i.Dotaciones, s) ||
		!trazasComparacionValidas(i.Plazas, s) || !trazasComparacionValidas(i.PuestosIndividuales, s) || !trazasComparacionValidas(i.Vinculos, s) {
		return false
	}
	for _, v := range i.Unidades {
		if v.CatalogoID != "estructura-organizativa-dipgra" || v.CatalogoVersion < 1 || v.CatalogoRevision < 1 || !referenciaComparacionValida(v.ClaveCatalogo) ||
			v.PadreID != "" && !referenciaComparacionValida(v.PadreID) || v.Tipo != "delegacion" && v.Tipo != "centro" && v.Tipo != "puesto_responsabilidad" || !textoComparacionValido(v.Etiqueta, 512) {
			return false
		}
	}
	for _, v := range i.PuestosTipo {
		if v.VersionRPTRef != s.VersionRPTRef || !textoComparacionValido(v.CodigoFuente, 128) || !referenciaComparacionValida(v.UnidadID) || !textoComparacionValido(v.Denominacion, 512) || !referenciaComparacionValida(v.ClasificacionRef) {
			return false
		}
	}
	for _, v := range i.Dotaciones {
		if v.VersionRPTRef != s.VersionRPTRef || !referenciaComparacionValida(v.PuestoTipoID) || v.Cantidad < 1 || v.Cantidad > 100000 {
			return false
		}
	}
	for _, v := range i.Plazas {
		if v.VersionPlantillaRef != s.VersionPlantillaRef || !textoComparacionValido(v.CodigoFuente, 128) || !referenciaComparacionValida(v.ClasificacionRef) || !referenciaComparacionValida(v.UnidadID) || v.EstadoEstructural != "vigente" && v.EstadoEstructural != "amortizada" {
			return false
		}
	}
	for _, v := range i.PuestosIndividuales {
		if v.VersionRPTRef != s.VersionRPTRef || !textoComparacionValido(v.CodigoFuente, 128) || !referenciaComparacionValida(v.PuestoTipoID) || !referenciaComparacionValida(v.UnidadID) || v.EstadoEstructural != "vigente" && v.EstadoEstructural != "suprimido" {
			return false
		}
	}
	for _, v := range i.Vinculos {
		if !referenciaComparacionValida(v.PlazaID) || !referenciaComparacionValida(v.PuestoID) {
			return false
		}
	}
	return true
}

func referenciaComparacionValida(s string) bool { return patronIDOrganizacionHistorica.MatchString(s) }
func textoComparacionValido(s string, max int) bool {
	if s == "" || !utf8.ValidString(s) || strings.TrimSpace(s) != s || len([]rune(s)) > max {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// Las colecciones contienen exclusivamente structs de dominio con campos
// escalares. La reflexión recorre sus campos declarados; no interpreta mapas
// libres ni decide identidades por el código fuente.
func trazaComparacion[T any](v T) TrazaOrganizacionHistorica {
	return reflect.ValueOf(v).FieldByName("Traza").Interface().(TrazaOrganizacionHistorica)
}
func trazasComparacionValidas[T any](filas []T, s SelectorOrganizacionHistorica) bool {
	ids := make(map[string]bool, len(filas))
	for _, fila := range filas {
		t := trazaComparacion(fila)
		if t.ValidarEn(s) != nil || ids[t.ID] {
			return false
		}
		ids[t.ID] = true
	}
	return true
}

func compararColeccionOrganizacion[T any](antes, despues []T, coberturaAntes, coberturaDespues string) ColeccionComparacionOrganizacion {
	r := ColeccionComparacionOrganizacion{Estado: "parcial", Antes: len(antes), Despues: len(despues), Cambios: []CambioComparacionOrganizacion{}, NoVerificables: []string{}}
	if coberturaAntes == "completa" && coberturaDespues == "completa" {
		r.Estado = "completa"
	}
	if coberturaAntes == "sin_datos" && coberturaDespues == "sin_datos" {
		r.Estado = "sin_datos"
	}
	a, d := make(map[string]T, len(antes)), make(map[string]T, len(despues))
	ids := make([]string, 0, len(antes)+len(despues))
	for _, fila := range antes {
		id := trazaComparacion(fila).ID
		a[id] = fila
		ids = append(ids, id)
	}
	for _, fila := range despues {
		id := trazaComparacion(fila).ID
		d[id] = fila
		if _, ok := a[id]; !ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		va, existeAntes := a[id]
		vd, existeDespues := d[id]
		c := CambioComparacionOrganizacion{ID: id, Campos: []string{}}
		if existeAntes {
			t := trazaComparacion(va)
			c.TrazaAntes = &t
		}
		if existeDespues {
			t := trazaComparacion(vd)
			c.TrazaDespues = &t
		}
		switch {
		case !existeAntes || !existeDespues:
			if r.Estado != "completa" {
				r.NoVerificables = append(r.NoVerificables, id)
				continue
			}
			c.Tipo = "entrada_en_corte"
			if !existeDespues {
				c.Tipo = "salida_del_corte"
			}
		default:
			c.Campos = camposComparacionCambiados(va, vd, false)
			if len(c.Campos) > 0 {
				c.Tipo = "cambio"
			} else {
				c.Campos = camposComparacionCambiados(va, vd, true)
				if len(c.Campos) == 0 {
					continue
				}
				c.Tipo = "procedencia"
			}
		}
		r.Cambios = append(r.Cambios, c)
	}
	return r
}

func camposComparacionCambiados[T any](antes, despues T, procedencia bool) []string {
	a, d := reflect.ValueOf(antes), reflect.ValueOf(despues)
	campos := []string{}
	for j := 0; j < a.NumField(); j++ {
		f := a.Type().Field(j)
		p := f.Name == "Traza" || f.Name == "VersionRPTRef" || f.Name == "VersionPlantillaRef"
		iguales := reflect.DeepEqual(a.Field(j).Interface(), d.Field(j).Interface())
		if f.Name == "Traza" {
			iguales = trazasComparacionIguales(trazaComparacion(antes), trazaComparacion(despues))
		}
		if p == procedencia && !iguales {
			campos = append(campos, strings.Split(f.Tag.Get("json"), ",")[0])
		}
	}
	sort.Strings(campos)
	return campos
}

func trazasComparacionIguales(a, d TrazaOrganizacionHistorica) bool {
	return a.ID == d.ID && a.Version == d.Version && a.FuenteRef == d.FuenteRef && a.ActoRef == d.ActoRef &&
		a.HuellaSHA256 == d.HuellaSHA256 && a.EfectosDesde == d.EfectosDesde && a.EfectosHasta == d.EfectosHasta &&
		a.ConocidoDesde.Equal(d.ConocidoDesde) && a.ConocidoHasta.Equal(d.ConocidoHasta)
}
