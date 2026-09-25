package domain

import (
	"errors"
	"slices"
	"strings"
)

var ErrPoliticaTransicionesInvalida = errors.New("bolsa: politica de transiciones de situacion invalida")

// separadorTransicion une origen y destino en la forma «origen>destino» que
// también guarda la base de datos (migración 000032 de bolsa_llamamientos).
const separadorTransicion = ">"

// maximoTransiciones acota una lista recibida antes de reservar memoria:
// siete situaciones, sin salir de «excluido» ni repetir la misma.
const maximoTransiciones = 42

// PoliticaTransicionesSituacion es la tabla de transiciones de situación que
// rige un cambio. Procede del catálogo configurable (b28.transiciones.<origen>)
// publicado en la base de datos; sin publicación rige la compilada, que es el
// literal de la migración 000012. Toda política respeta tres invariantes
// fijas, las mismas que exige la base de datos: nunca se sale de «excluido»
// (B2 no tiene readmisión), no hay transiciones a la misma situación y desde
// cualquier otra situación se puede dar de baja definitiva (art. 11).
// El valor cero equivale a la compilada.
type PoliticaTransicionesSituacion struct {
	destinos map[string][]string
}

// PoliticaTransicionesSituacionCompilada es la conducta sin política
// publicada: la tabla de transicionesSituacionParticipacion.
func PoliticaTransicionesSituacionCompilada() PoliticaTransicionesSituacion {
	destinos := make(map[string][]string, len(transicionesSituacionParticipacion))
	for _, origen := range SituacionesParticipacion() {
		destinos[origen] = DestinosSituacionParticipacion(origen)
	}
	return PoliticaTransicionesSituacion{destinos: destinos}
}

// NuevaPoliticaTransicionesSituacion valida una tabla origen→destinos. Un
// origen ausente no tiene destinos. Rechaza situaciones desconocidas,
// repetidas y cualquier tabla que incumpla una invariante fija.
func NuevaPoliticaTransicionesSituacion(tabla map[string][]string) (PoliticaTransicionesSituacion, error) {
	total := 0
	for origen, destinos := range tabla {
		if !situacionParticipacionValida(origen) {
			return PoliticaTransicionesSituacion{}, ErrPoliticaTransicionesInvalida
		}
		total += len(destinos)
	}
	if total > maximoTransiciones {
		return PoliticaTransicionesSituacion{}, ErrPoliticaTransicionesInvalida
	}
	canonica := make(map[string][]string, len(tabla))
	for _, origen := range SituacionesParticipacion() {
		destinos := tabla[origen]
		for i, destino := range destinos {
			if !situacionParticipacionValida(destino) || destino == origen || origen == SituacionExcluido || slices.Contains(destinos[:i], destino) {
				return PoliticaTransicionesSituacion{}, ErrPoliticaTransicionesInvalida
			}
		}
		if origen != SituacionExcluido && !slices.Contains(destinos, SituacionExcluido) {
			return PoliticaTransicionesSituacion{}, ErrPoliticaTransicionesInvalida
		}
		ordenados := make([]string, 0, len(destinos))
		for _, situacion := range SituacionesParticipacion() {
			if slices.Contains(destinos, situacion) {
				ordenados = append(ordenados, situacion)
			}
		}
		canonica[origen] = ordenados
	}
	return PoliticaTransicionesSituacion{destinos: canonica}, nil
}

// PoliticaTransicionesDesdePares lee la forma «origen>destino» de la base de
// datos.
func PoliticaTransicionesDesdePares(pares []string) (PoliticaTransicionesSituacion, error) {
	if len(pares) == 0 || len(pares) > maximoTransiciones {
		return PoliticaTransicionesSituacion{}, ErrPoliticaTransicionesInvalida
	}
	tabla := make(map[string][]string, len(SituacionesParticipacion()))
	for _, par := range pares {
		origen, destino, ok := strings.Cut(par, separadorTransicion)
		if !ok || strings.Contains(destino, separadorTransicion) {
			return PoliticaTransicionesSituacion{}, ErrPoliticaTransicionesInvalida
		}
		tabla[origen] = append(tabla[origen], destino)
	}
	return NuevaPoliticaTransicionesSituacion(tabla)
}

func (p PoliticaTransicionesSituacion) tabla() map[string][]string {
	if p.destinos == nil {
		return PoliticaTransicionesSituacionCompilada().destinos
	}
	return p.destinos
}

// Destinos devuelve una copia de los destinos admitidos desde origen, en el
// orden estable de las situaciones.
func (p PoliticaTransicionesSituacion) Destinos(origen string) []string {
	return slices.Clone(p.tabla()[origen])
}

// Admite indica si la política permite pasar de origen a destino.
func (p PoliticaTransicionesSituacion) Admite(origen, destino string) bool {
	return slices.Contains(p.tabla()[origen], destino)
}

// Pares devuelve la forma canónica «origen>destino» que publica la base.
func (p PoliticaTransicionesSituacion) Pares() []string {
	tabla := p.tabla()
	pares := make([]string, 0, maximoTransiciones)
	for _, origen := range SituacionesParticipacion() {
		for _, destino := range tabla[origen] {
			pares = append(pares, origen+separadorTransicion+destino)
		}
	}
	return pares
}

// Restringir devuelve la política sin los destinos de origen que no figuran
// en permitidos. Sirve para aplicar el catálogo sobre la política vigente: el
// catálogo puede cerrar una transición, nunca abrir otra.
func (p PoliticaTransicionesSituacion) Restringir(origen string, permitidos []string) PoliticaTransicionesSituacion {
	tabla := p.tabla()
	copia := make(map[string][]string, len(tabla))
	for o, destinos := range tabla {
		copia[o] = slices.Clone(destinos)
	}
	copia[origen] = slices.DeleteFunc(copia[origen], func(d string) bool { return !slices.Contains(permitidos, d) })
	return PoliticaTransicionesSituacion{destinos: copia}
}
