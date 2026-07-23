package ports

import (
	"crypto/hmac"
	"errors"
	"regexp"
	"strconv"
	"strings"
)

const MaximoGeneracionesHMACAlta = 4

var (
	ErrColeccionSellosHMACInvalida = errors.New(
		"contratacion temporal: coleccion de sellos HMAC invalida",
	)
	patronReferenciaGeneracionHMAC = regexp.MustCompile(
		`^([a-z][a-z0-9._/-]{1,86})/v([1-9][0-9]{0,8})$`,
	)
	patronDominioHMAC = regexp.MustCompile(`^[a-z][a-z0-9._/-]{1,86}$`)
)

// SelloGeneracionalHMAC identifica una generación mediante una referencia
// pública y su sello. No contiene material criptográfico ni la clave.
type SelloGeneracionalHMAC struct {
	Generacion uint32
	Valor      string
}

// DatosColeccionSellosHMAC expone una copia defensiva del sello activo y de
// los retenidos. Los retenidos están ordenados de generación mayor a menor.
type DatosColeccionSellosHMAC struct {
	Activo    SelloGeneracionalHMAC
	Retenidos []SelloGeneracionalHMAC
}

// ColeccionSellosHMAC es el resultado inmutable de un llavero rotatorio.
type ColeccionSellosHMAC struct {
	datos *DatosColeccionSellosHMAC
}

func NuevaColeccionSellosHMAC(
	activo string,
	retenidos []string,
) (ColeccionSellosHMAC, error) {
	if len(retenidos)+1 > MaximoGeneracionesHMACAlta {
		return ColeccionSellosHMAC{}, ErrColeccionSellosHMACInvalida
	}
	dominioActivo, generacionActiva, valida := descomponerSelloHMAC(activo)
	if !valida {
		return ColeccionSellosHMAC{}, ErrColeccionSellosHMACInvalida
	}
	datos := DatosColeccionSellosHMAC{
		Activo: SelloGeneracionalHMAC{
			Generacion: generacionActiva,
			Valor:      activo,
		},
		Retenidos: make([]SelloGeneracionalHMAC, 0, len(retenidos)),
	}
	anterior := generacionActiva
	vistos := map[string]struct{}{activo: {}}
	for _, sello := range retenidos {
		dominio, generacion, valida := descomponerSelloHMAC(sello)
		if !valida || dominio != dominioActivo || generacion >= anterior {
			return ColeccionSellosHMAC{}, ErrColeccionSellosHMACInvalida
		}
		if _, repetido := vistos[sello]; repetido {
			return ColeccionSellosHMAC{}, ErrColeccionSellosHMACInvalida
		}
		vistos[sello] = struct{}{}
		datos.Retenidos = append(datos.Retenidos, SelloGeneracionalHMAC{
			Generacion: generacion,
			Valor:      sello,
		})
		anterior = generacion
	}
	return ColeccionSellosHMAC{datos: &datos}, nil
}

func (c ColeccionSellosHMAC) Datos() (DatosColeccionSellosHMAC, error) {
	if c.datos == nil {
		return DatosColeccionSellosHMAC{}, ErrColeccionSellosHMACInvalida
	}
	datos := DatosColeccionSellosHMAC{
		Activo: c.datos.Activo,
		Retenidos: append([]SelloGeneracionalHMAC(nil),
			c.datos.Retenidos...),
	}
	sellos := make([]string, 0, len(datos.Retenidos))
	for _, retenido := range datos.Retenidos {
		sellos = append(sellos, retenido.Valor)
	}
	reconstruida, err := NuevaColeccionSellosHMAC(datos.Activo.Valor, sellos)
	if err != nil || reconstruida.datos.Activo.Generacion != datos.Activo.Generacion {
		return DatosColeccionSellosHMAC{}, ErrColeccionSellosHMACInvalida
	}
	for indice := range datos.Retenidos {
		if reconstruida.datos.Retenidos[indice].Generacion !=
			datos.Retenidos[indice].Generacion {
			return DatosColeccionSellosHMAC{}, ErrColeccionSellosHMACInvalida
		}
	}
	return datos, nil
}

func (c ColeccionSellosHMAC) Contiene(sello string) bool {
	datos, err := c.Datos()
	if err != nil || !SelloHMACSHA256Valido(sello) {
		return false
	}
	if hmac.Equal([]byte(datos.Activo.Valor), []byte(sello)) {
		return true
	}
	for _, retenido := range datos.Retenidos {
		if hmac.Equal([]byte(retenido.Valor), []byte(sello)) {
			return true
		}
	}
	return false
}

func (c ColeccionSellosHMAC) ValidarDominio(dominio string) error {
	datos, err := c.Datos()
	if err != nil || !dominioHMACValido(dominio) {
		return ErrColeccionSellosHMACInvalida
	}
	dominioActivo, _, _ := descomponerSelloHMAC(datos.Activo.Valor)
	if dominioActivo != dominio {
		return ErrColeccionSellosHMACInvalida
	}
	return nil
}

func (c ColeccionSellosHMAC) Generaciones() ([]uint32, error) {
	datos, err := c.Datos()
	if err != nil {
		return nil, err
	}
	generaciones := make([]uint32, 0, len(datos.Retenidos)+1)
	generaciones = append(generaciones, datos.Activo.Generacion)
	for _, retenido := range datos.Retenidos {
		generaciones = append(generaciones, retenido.Generacion)
	}
	return generaciones, nil
}

func descomponerSelloHMAC(sello string) (string, uint32, bool) {
	if !SelloHMACSHA256Valido(sello) {
		return "", 0, false
	}
	partes := strings.Split(sello, ":")
	coincidencias := patronReferenciaGeneracionHMAC.FindStringSubmatch(partes[1])
	if len(coincidencias) != 3 {
		return "", 0, false
	}
	generacion, err := strconv.ParseUint(coincidencias[2], 10, 32)
	if err != nil || generacion == 0 {
		return "", 0, false
	}
	return coincidencias[1], uint32(generacion), true
}

func dominioHMACValido(dominio string) bool {
	return patronDominioHMAC.MatchString(dominio)
}
