// vec-revisar-organizacion comprueba un paquete local sin efectos duraderos.
package main

import (
	"encoding/json"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/web"
)

type salida struct {
	Informe  application.InformeRevisionPreparacionOrganizacion `json:"informe"`
	Idioma   string                                             `json:"idioma"`
	Mensajes map[string]string                                  `json:"mensajes"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdin, os.Stdout)) }

func ejecutar(args []string, input io.Reader, output io.Writer) int {
	catalogo, mensajes, err := web.CatalogoRevisionOrganizacion()
	if err != nil {
		return codigoErrorSalida(err)
	}
	idioma := catalogo.DefaultLocale()
	codigoMensaje := ""
	if len(args) > 0 {
		if len(args) != 2 || args[0] != "--idioma" || len(mensajes[args[1]]) == 0 {
			codigoMensaje = "argumentos_invalidos"
		} else {
			idioma = args[1]
		}
	}
	var informe application.InformeRevisionPreparacionOrganizacion
	if codigoMensaje == "" {
		paquete, err := leerPaquete(input)
		if err != nil {
			codigoMensaje = "entrada_invalida"
		} else {
			informe = application.RevisarPreparacionOrganizacion(paquete)
		}
	}
	if codigoMensaje != "" {
		informe = application.InformeRevisionPreparacionOrganizacion{Estado: "preparacion_no_autoritativa", ClaveError: codigoMensaje,
			Seccion: "entrada", RecuentosClase: map[string]int{}, RecuentosDecision: map[string]int{}, PendientesPublicacion: []string{"revision_paquete"}}
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(salida{informe, idioma, mensajes[idioma]}); err != nil {
		return codigoErrorSalida(err)
	}
	if !informe.Valido {
		return 1
	}
	return 0
}

// El estado del proceso propaga al invocante el fallo de carga o escritura.
// El error se consume sin exponer rutas ni datos en un mensaje de usuario.
func codigoErrorSalida(err error) int {
	switch err {
	case nil:
		return 0
	default:
		return 2
	}
}
