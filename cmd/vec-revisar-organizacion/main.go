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
	Paquete  *application.PaquetePreparacionOrganizacion        `json:"paquete,omitempty"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdin, os.Stdout)) }

func ejecutar(args []string, input io.Reader, output io.Writer) int {
	catalogo, mensajes, err := web.CatalogoRevisionOrganizacion()
	if err != nil {
		return codigoErrorSalida(err)
	}
	idioma := catalogo.DefaultLocale()
	codigoMensaje := ""
	preparar, idiomaSeleccionado := false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--preparar":
			if preparar {
				codigoMensaje = "argumentos_invalidos"
			}
			preparar = true
		case "--idioma":
			if i+1 >= len(args) || len(mensajes[args[i+1]]) == 0 || idiomaSeleccionado {
				codigoMensaje = "argumentos_invalidos"
			} else {
				i++
				idioma = args[i]
				idiomaSeleccionado = true
			}
		default:
			codigoMensaje = "argumentos_invalidos"
		}
	}
	var exportado *application.PaquetePreparacionOrganizacion
	var informe application.InformeRevisionPreparacionOrganizacion
	if codigoMensaje == "" {
		paquete, err := leerPaquete(input)
		if err != nil {
			codigoMensaje = "entrada_invalida"
		} else {
			if preparar {
				var normalizado application.PaquetePreparacionOrganizacion
				normalizado, informe = application.PrepararPaqueteOrganizacion(paquete)
				if informe.Valido {
					exportado = &normalizado
				}
			} else {
				informe = application.RevisarPreparacionOrganizacion(paquete)
			}
		}
	}
	if codigoMensaje != "" {
		informe = application.InformeRevisionPreparacionOrganizacion{Estado: "preparacion_no_autoritativa", ClaveError: codigoMensaje,
			Seccion: "entrada", RecuentosClase: map[string]int{}, RecuentosDecision: map[string]int{}, PendientesPublicacion: []string{"revision_paquete"}}
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(salida{Informe: informe, Idioma: idioma, Mensajes: mensajes[idioma], Paquete: exportado}); err != nil {
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
