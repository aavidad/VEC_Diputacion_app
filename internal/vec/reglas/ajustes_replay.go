package reglas

import (
	"slices"
	"strings"
	"time"
)

// PrepararRepeticionAjustes produce material exclusivo de repetición desde
// la cabeza íntegra leída con autorización. Solo admite una versión esperada
// anterior a esa cabeza; CT148 reconocerá una clave ya guardada antes del CAS
// o rechazará una clave nueva sin efecto. No valida contra la base actual,
// que podría haber cambiado desde la primera operación.
func PrepararRepeticionAjustes(
	instante time.Time, versionEsperada int, cabeza VersionAjustes,
	baseVersion int, baseHuellaSHA256 string, solicitadas []SolicitudCambioAjuste,
) (PreparacionAjustes, error) {
	var vacia PreparacionAjustes
	if instante.IsZero() || versionEsperada < 0 || versionEsperada > 9_999_998 ||
		cabeza.Version <= versionEsperada || baseVersion < 1 || baseVersion > 9_999_999 ||
		!huellaCanonicaAjustes(baseHuellaSHA256) || len(solicitadas) == 0 ||
		len(solicitadas) > maximoReglasAjustadas*4 {
		return vacia, ErrAjusteInvalido
	}
	id := CatalogoAjustesDe(CatalogoContratacionTemporal)
	if err := validarVersionAjustes(cabeza, id, instante.UTC()); err != nil {
		return vacia, err
	}
	ajustes := copiarConjuntoAjustes(cabeza.Ajustes)
	cambios := make([]CambioAjustePreparado, 0, len(solicitadas))
	vistos := make(map[string]bool, len(solicitadas))
	for _, solicitud := range solicitadas {
		par := solicitud.ReglaClave + "\x00" + solicitud.Campo
		if vistos[par] || !claveAjusteCanonica(solicitud.ReglaClave) ||
			!campoAjustable(solicitud.Campo) || !valorAjusteCanonico(solicitud.Nuevo) {
			return vacia, ErrAjusteInvalido
		}
		vistos[par] = true
		anterior := cabeza.Ajustes[solicitud.ReglaClave][solicitud.Campo]
		if anterior == "" {
			// CT148 no compara este campo en un replay. Si la clave no existe,
			// el CAS obsoleto rechaza la operación antes de insertar nada.
			anterior = solicitud.Nuevo
		}
		if ajustes[solicitud.ReglaClave] == nil {
			ajustes[solicitud.ReglaClave] = make(map[string]string)
		}
		ajustes[solicitud.ReglaClave][solicitud.Campo] = solicitud.Nuevo
		cambios = append(cambios, CambioAjustePreparado{
			ReglaClave: solicitud.ReglaClave, Campo: solicitud.Campo,
			Anterior: anterior, Nuevo: solicitud.Nuevo,
		})
	}
	canonico, err := CanonicoAjustes(ajustes)
	if err != nil {
		return vacia, err
	}
	huella, err := HuellaAjustes(ajustes)
	if err != nil {
		return vacia, err
	}
	slices.SortFunc(cambios, func(a, b CambioAjustePreparado) int {
		if orden := strings.Compare(a.ReglaClave, b.ReglaClave); orden != 0 {
			return orden
		}
		return strings.Compare(a.Campo, b.Campo)
	})
	return PreparacionAjustes{datos: DatosPreparacionAjustes{
		CatalogoAjustesID: id, VersionEsperada: versionEsperada,
		BaseVersion: baseVersion, BaseHuellaSHA256: baseHuellaSHA256,
		Ajustes: ajustes, Canonico: canonico, HuellaSHA256: huella, Cambios: cambios,
	}}, nil
}

func huellaCanonicaAjustes(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for _, c := range valor {
		if c < '0' || c > '9' {
			if c < 'a' || c > 'f' {
				return false
			}
		}
	}
	return true
}
