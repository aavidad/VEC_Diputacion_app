// vec-copias-comprobar compara declaraciones locales sin red ni efectos.
package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

type salida struct {
	Alcance                  string           `json:"alcance"`
	HabilitaRestauracion     bool             `json:"habilita_restauracion"`
	Autenticidad             string           `json:"autenticidad"`
	VerificacionRestauracion string           `json:"verificacion_restauracion"`
	Resultado                copias.Resultado `json:"resultado"`
	EstadoClave              string           `json:"estado_clave"`
	Mensaje                  string           `json:"mensaje,omitempty"`
	Mensajes                 []string         `json:"mensajes,omitempty"`
}

func diagnosticar(w io.Writer, key string) error {
	return json.NewEncoder(w).Encode(struct {
		ErrorClave string `json:"error_clave"`
	}{key})
}

func fallar(w io.Writer, key string) int {
	if diagnosticar(w, key) != nil {
		return 4
	}
	return 3
}

func run(args []string, in io.Reader, out, diag io.Writer) int {
	var catalogo map[string]string
	if len(args) > 0 {
		if len(args) != 2 || args[0] != "-catalogo" {
			return fallar(diag, "copias_seguridad_error_argumentos")
		}
		// G304/G703: lectura local solicitada explícitamente por el operador; no ejecuta
		// el contenido ni publica la ruta o errores del sistema de archivos.
		f, err := os.Open(args[1]) // #nosec G304 G703 -- catálogo local indicado por el operador, sin frontera HTTP.
		if err != nil {
			return fallar(diag, "copias_seguridad_error_catalogo")
		}
		info, statErr := f.Stat()
		if statErr != nil || !info.Mode().IsRegular() {
			if closeErr := f.Close(); closeErr != nil {
				return fallar(diag, "copias_seguridad_error_catalogo")
			}
			return fallar(diag, "copias_seguridad_error_catalogo")
		}
		err = errors.Join(leerEstricto(f, &catalogo), f.Close())
		if err != nil {
			return fallar(diag, "copias_seguridad_error_catalogo")
		}
	}
	var e entrada
	if leerEstricto(in, &e) != nil {
		return fallar(diag, "copias_seguridad_error_entrada")
	}
	var resultado copias.Resultado
	if e.FormatoVersion != copias.FormatoVersion {
		resultado = copias.Resultado{Estado: copias.NoComprobable, Razones: []copias.Razon{{Codigo: "dato_no_comprobable", Clave: "entrada.formato_version", Esperado: "1", Obtenido: "formato_no_admitido", Accion: "actualizar_formato"}}}
	} else {
		resultado = copias.CompararVersiones(e.Manifiesto, e.Destino, e.Politica, e.Modo)
	}
	s := salida{Alcance: "comparacion_offline_declarada", HabilitaRestauracion: false, Autenticidad: "no_comprobada", VerificacionRestauracion: "no_comprobada", Resultado: resultado, EstadoClave: "copias_seguridad_estado_" + string(resultado.Estado)}
	if catalogo != nil {
		var ok bool
		s.Mensaje, ok = catalogo[s.EstadoClave]
		if !ok {
			return fallar(diag, "copias_seguridad_error_catalogo")
		}
		for _, r := range resultado.Razones {
			texto, ok := catalogo["copias_seguridad_razon_"+r.Codigo]
			accion, okAccion := catalogo["copias_seguridad_accion_"+r.Accion]
			if !ok || !okAccion {
				return fallar(diag, "copias_seguridad_error_catalogo")
			}
			s.Mensajes = append(s.Mensajes, texto+" "+accion)
		}
	}
	if json.NewEncoder(out).Encode(s) != nil {
		return fallar(diag, "copias_seguridad_error_salida")
	}
	switch resultado.Estado {
	case copias.Compatible:
		return 0
	case copias.Incompatible:
		return 1
	default:
		return 2
	}
}

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }
