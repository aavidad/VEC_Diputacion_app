// vec-admin-comprobar-perfil comprueba archivos offline. No compone identidad,
// PDP, publicación de roles, servidor, base de datos ni proveedores de red.
package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"time"

	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/domain"
)

var errEntrada = errors.New("admin_comprobar_entrada_invalida")

type textosComprobador struct {
	Idioma   string            `json:"idioma"`
	Mensajes map[string]string `json:"mensajes"`
}

type salidaComprobador struct {
	Comprobado bool                                   `json:"comprobado"`
	Publicado  bool                                   `json:"publicado"`
	Codigo     string                                 `json:"codigo"`
	Mensaje    string                                 `json:"mensaje"`
	Limite     string                                 `json:"limite"`
	Dictamen   *domain.DictamenPerfilAdministracionV1 `json:"dictamen,omitempty"`
}

// Argumentos posicionales: textos.json catalogo.json propuesta.json instanteUTC.
// El catálogo lingüístico proporciona su idioma, sin política por idioma en Go.
func ejecutar(args []string, salida io.Writer) int {
	if len(args) != 4 || salida == nil {
		return 2
	}
	var textos textosComprobador
	if leerJSON(args[0], &textos, 65536) != nil {
		return 2
	}
	traductor, err := i18n.New(textos.Idioma, map[string]map[string]string{textos.Idioma: textos.Mensajes})
	if err != nil {
		return 2
	}
	claves := []string{"admin_comprobar_correcto", "admin_comprobar_limite", errEntrada.Error(),
		domain.ErrCatalogoAccionesAdministracionInvalido.Error(), domain.ErrPropuestaPerfilAdministracionInvalida.Error(),
		domain.ErrPerfilAdministracionFijo.Error(), domain.ErrCatalogoAccionesAdministracionNoVigente.Error(),
		domain.ErrOrigenPerfilAdministracionNoCoincide.Error(), domain.ErrPermisoPerfilAdministracionNoCoincide.Error()}
	for _, clave := range claves {
		if mensaje, existe := traductor.Message(textos.Idioma, clave); !existe || mensaje == "" {
			return 2
		}
	}
	var catalogo domain.CatalogoAccionesAdministracionV1
	var propuesta domain.PropuestaPerfilAdministracionV1
	instante, errFecha := time.Parse(time.RFC3339Nano, args[3])
	if leerJSON(args[1], &catalogo, 4<<20) != nil || leerJSON(args[2], &propuesta, 4<<20) != nil || errFecha != nil {
		return emitir(salida, traductor, textos.Idioma, nil, errEntrada)
	}
	dictamen, err := domain.ComprobarPropuestaPerfilAdministracionV1(catalogo, propuesta, instante)
	if err != nil {
		return emitir(salida, traductor, textos.Idioma, nil, err)
	}
	return emitir(salida, traductor, textos.Idioma, &dictamen, nil)
}

func emitir(salida io.Writer, traductor *i18n.Catalog, idioma string, dictamen *domain.DictamenPerfilAdministracionV1, fallo error) int {
	codigo := "admin_comprobar_correcto"
	exitCode := 0
	if fallo != nil {
		codigo, exitCode = fallo.Error(), 1
	}
	resultado := salidaComprobador{Comprobado: fallo == nil, Codigo: codigo, Mensaje: traductor.T(idioma, codigo),
		Limite: traductor.T(idioma, "admin_comprobar_limite"), Dictamen: dictamen}
	if json.NewEncoder(salida).Encode(resultado) != nil {
		return 2
	}
	return exitCode
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout)) }
