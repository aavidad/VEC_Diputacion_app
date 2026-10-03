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
	Comprobado       bool                                   `json:"comprobado"`
	Publicado        bool                                   `json:"publicado"`
	Codigo           string                                 `json:"codigo"`
	Mensaje          string                                 `json:"mensaje"`
	Limite           string                                 `json:"limite"`
	Dictamen         *domain.DictamenPerfilAdministracionV1 `json:"dictamen,omitempty"`
	Plan             *domain.PlanGobiernoPerfil             `json:"plan,omitempty"`
	PlanHuellaSHA256 string                                 `json:"plan_huella_sha256,omitempty"`
}

// Argumentos posicionales: textos.json catalogo.json propuesta.json instanteUTC.
// El catálogo lingüístico proporciona su idioma, sin política por idioma en Go.
func ejecutar(args []string, salida io.Writer) int {
	prepararPlan := len(args) == 5 && args[4] == "--preparar-plan"
	if (len(args) != 4 && !prepararPlan) || salida == nil {
		return 2
	}
	var textos textosComprobador
	if leerJSON(args[0], &textos, 65536) != nil {
		return 2
	}
	traductor, err := i18n.New(textos.Idioma, map[string]map[string]string{textos.Idioma: textos.Mensajes})
	if err != nil {
		return codigoErrorInicioCatalogo(err)
	}
	claves := []string{"admin_comprobar_correcto", "admin_comprobar_limite", errEntrada.Error(),
		domain.ErrCatalogoAccionesAdministracionInvalido.Error(), domain.ErrPropuestaPerfilAdministracionInvalida.Error(),
		domain.ErrPerfilAdministracionFijo.Error(), domain.ErrCatalogoAccionesAdministracionNoVigente.Error(),
		domain.ErrOrigenPerfilAdministracionNoCoincide.Error(), domain.ErrPermisoPerfilAdministracionNoCoincide.Error()}
	if prepararPlan {
		claves = append(claves, "admin_gobierno_plan_preparado", "admin_gobierno_plan_limite", domain.ErrPlanGobiernoPerfilInvalido.Error())
	}
	for _, clave := range claves {
		if mensaje, existe := traductor.Message(textos.Idioma, clave); !existe || mensaje == "" {
			return 2
		}
	}
	var catalogo domain.CatalogoAccionesAdministracionV1
	instante, errFecha := time.Parse(time.RFC3339Nano, args[3])
	if leerJSON(args[1], &catalogo, 4<<20) != nil || errFecha != nil {
		return emitir(salida, traductor, textos.Idioma, nil, errEntrada)
	}
	if prepararPlan {
		return ejecutarPlanGobiernoPerfil(args[2], catalogo, instante, salida, traductor, textos.Idioma)
	}
	var propuesta domain.PropuestaPerfilAdministracionV1
	if leerJSON(args[2], &propuesta, 4<<20) != nil {
		return emitir(salida, traductor, textos.Idioma, nil, errEntrada)
	}
	dictamen, err := domain.ComprobarPropuestaPerfilAdministracionV1(catalogo, propuesta, instante)
	if err != nil {
		return emitir(salida, traductor, textos.Idioma, nil, err)
	}
	return emitir(salida, traductor, textos.Idioma, &dictamen, nil)
}

// El error de inicialización se propaga como código de proceso. Sin un catálogo
// válido no se emite un mensaje traducido ni se vuelca su contenido en la salida.
func codigoErrorInicioCatalogo(fallo error) int {
	if fallo == nil {
		return 0
	}
	return 2
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
