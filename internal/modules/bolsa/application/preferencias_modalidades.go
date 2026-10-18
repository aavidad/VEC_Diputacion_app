package application

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"slices"
	"strconv"

	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

const dominioHuellaPreferenciasModalidades = "vec.bolsa.preferencias-modalidades.v1"

var clavePreferenciaModalidad = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)

// HuellaSolicitudPreferenciasModalidades fija la selección y la instantánea
// gobernada que el servidor ligará al recurso de autorización V3. El orden de
// entrada no tiene significado; nil expresa todas y [] expresa ninguna.
func HuellaSolicitudPreferenciasModalidades(candidato string, version uint64,
	seleccion []string, disponibles []ports.ModalidadPreferencia,
) (string, error) {
	if !constitucion.ReferenciaCandidatoValida(candidato) || version >= 2147483647 || len(seleccion) > 256 {
		return "", ports.ErrOrdenPreferenciasInvalido
	}
	if len(disponibles) < 1 || len(disponibles) > 256 {
		return "", ports.ErrPreferenciasNoDisponibles
	}
	ordenDisponibles := slices.Clone(disponibles)
	slices.SortFunc(ordenDisponibles, func(a, b ports.ModalidadPreferencia) int {
		if a.Clave < b.Clave {
			return -1
		}
		if a.Clave > b.Clave {
			return 1
		}
		return 0
	})
	claves := make(map[string]struct{}, len(ordenDisponibles))
	for _, opcion := range ordenDisponibles {
		if !clavePreferenciaModalidad.MatchString(opcion.Clave) || opcion.Fuente.Validar() != nil ||
			opcion.Fuente.CatalogoID != reglas.CatalogoContratacionTemporal ||
			opcion.Fuente.EntradaClave != reglas.CTPrefijoModalidad+opcion.Clave {
			return "", ports.ErrPreferenciasNoDisponibles
		}
		if _, repetida := claves[opcion.Clave]; repetida {
			return "", ports.ErrPreferenciasNoDisponibles
		}
		claves[opcion.Clave] = struct{}{}
	}
	ordenSeleccion := slices.Clone(seleccion)
	slices.Sort(ordenSeleccion)
	for indice, clave := range ordenSeleccion {
		if !clavePreferenciaModalidad.MatchString(clave) ||
			(indice > 0 && ordenSeleccion[indice-1] == clave) {
			return "", ports.ErrOrdenPreferenciasInvalido
		}
		if _, disponible := claves[clave]; !disponible {
			return "", ports.ErrOrdenPreferenciasInvalido
		}
	}

	h := sha256.New()
	encuadrar := func(valor string) {
		_, _ = h.Write([]byte(strconv.Itoa(len(valor)) + ":" + valor))
	}
	encuadrar(dominioHuellaPreferenciasModalidades)
	encuadrar(candidato)
	encuadrar(strconv.FormatUint(version, 10))
	if seleccion == nil {
		encuadrar("todas")
	} else {
		encuadrar("seleccion")
	}
	encuadrar(strconv.Itoa(len(ordenSeleccion)))
	for _, clave := range ordenSeleccion {
		encuadrar(clave)
	}
	encuadrar(strconv.Itoa(len(ordenDisponibles)))
	for _, opcion := range ordenDisponibles {
		encuadrar(opcion.Clave)
		encuadrar(opcion.Fuente.CatalogoID)
		encuadrar(strconv.Itoa(opcion.Fuente.CatalogoVersion))
		encuadrar(opcion.Fuente.CatalogoHuellaSHA256)
		encuadrar(opcion.Fuente.EntradaClave)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
