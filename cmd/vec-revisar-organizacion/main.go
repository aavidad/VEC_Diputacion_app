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
	Informe            application.InformeRevisionPreparacionOrganizacion `json:"informe"`
	Idioma             string                                             `json:"idioma"`
	Mensajes           map[string]string                                  `json:"mensajes"`
	Paquete            *application.PaquetePreparacionOrganizacion        `json:"paquete,omitempty"`
	Comprobacion       *application.ComprobacionCompletitudOrganizacion   `json:"comprobacion_completitud,omitempty"`
	BytesPaquete       int                                                `json:"bytes_paquete,omitempty"`
	LimiteBytesPaquete int                                                `json:"limite_bytes_paquete,omitempty"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdin, os.Stdout)) }

func ejecutar(args []string, input io.Reader, output io.Writer) int {
	catalogo, mensajes, err := web.CatalogoRevisionOrganizacion()
	if err != nil {
		return codigoErrorSalida(err)
	}
	idioma := catalogo.DefaultLocale()
	codigoMensaje := ""
	preparar, comprobarCompleto, idiomaSeleccionado := false, false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--preparar":
			if preparar {
				codigoMensaje = "argumentos_invalidos"
			}
			preparar = true
		case "--comprobar-completo":
			if comprobarCompleto {
				codigoMensaje = "argumentos_invalidos"
			}
			comprobarCompleto = true
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
	var bytesPaquete, limiteBytesPaquete int
	var exportado *application.PaquetePreparacionOrganizacion
	var comprobacion *application.ComprobacionCompletitudOrganizacion
	var informe application.InformeRevisionPreparacionOrganizacion
	if codigoMensaje == "" {
		paquete, err := leerPaquete(input)
		if err != nil {
			codigoMensaje = "entrada_invalida"
		} else {
			if preparar || comprobarCompleto {
				var normalizado application.PaquetePreparacionOrganizacion
				if comprobarCompleto {
					var resultado application.ComprobacionCompletitudOrganizacion
					normalizado, informe, resultado = application.ComprobarCompletitudPreparacionOrganizacion(paquete)
					comprobacion = &resultado
				} else {
					normalizado, informe = application.PrepararPaqueteOrganizacion(paquete)
				}
				if preparar && informe.Valido && (!comprobarCompleto || comprobacion.Completa) {
					material, err := json.Marshal(normalizado)
					if err != nil {
						codigoMensaje = "paquete_invalido"
					} else if len(material) > limiteEntrada {
						bytesPaquete, limiteBytesPaquete = len(material), limiteEntrada
						informe.Valido = false
						informe.ClaveError, informe.Seccion = "paquete_excede_limite", "paquete"
						informe.ManifiestoHuellaSHA256, informe.PaqueteHuellaSHA256 = "", ""
						informe.CoberturaConciliacion = nil
						if comprobacion != nil {
							comprobacion.Completa = false
							comprobacion.Faltantes = append(comprobacion.Faltantes, application.FaltanteCompletitudOrganizacion{
								Clave: "paquete_excede_limite", Esperado: "dentro_limite", Actual: "excede_limite",
							})
						}
					} else {
						exportado = &normalizado
					}
				}
			} else {
				informe = application.RevisarPreparacionOrganizacion(paquete)
			}
		}
	}
	if codigoMensaje != "" {
		informe = application.InformeRevisionPreparacionOrganizacion{Estado: "preparacion_no_autoritativa", ClaveError: codigoMensaje,
			Seccion: "entrada", RecuentosClase: map[string]int{}, RecuentosDecision: map[string]int{}, PendientesPublicacion: []string{"revision_paquete"}}
		if comprobarCompleto {
			resultado := application.ComprobacionCompletitudOrganizacion{Faltantes: []application.FaltanteCompletitudOrganizacion{{
				Clave: "revision_paquete", Esperado: "valido", Actual: "invalido",
			}}}
			comprobacion = &resultado
		}
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(salida{Informe: informe, Idioma: idioma, Mensajes: mensajes[idioma], Paquete: exportado, Comprobacion: comprobacion, BytesPaquete: bytesPaquete, LimiteBytesPaquete: limiteBytesPaquete}); err != nil {
		return codigoErrorSalida(err)
	}
	if !informe.Valido || comprobarCompleto && (comprobacion == nil || !comprobacion.Completa) {
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
