// vec-copias-comprobar compara declaraciones locales sin red ni efectos.
package main

import (
	"encoding/json"
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

func diagnosticar(w io.Writer, key string) {
	_ = json.NewEncoder(w).Encode(struct {
		ErrorClave string `json:"error_clave"`
	}{key})
}

func run(args []string, in io.Reader, out, diag io.Writer) int {
	var catalogo map[string]string
	if len(args) > 0 {
		if len(args) != 2 || args[0] != "-catalogo" {
			diagnosticar(diag, "copias_seguridad.error.argumentos")
			return 3
		}
		// G304/G703: lectura local solicitada explícitamente por el operador; no ejecuta
		// el contenido ni publica la ruta o errores del sistema de archivos.
		f, err := os.Open(args[1]) // #nosec G304 G703 -- catálogo local indicado por el operador, sin frontera HTTP.
		if err != nil {
			diagnosticar(diag, "copias_seguridad.error.catalogo")
			return 3
		}
		info, statErr := f.Stat()
		if statErr != nil || !info.Mode().IsRegular() {
			_ = f.Close()
			diagnosticar(diag, "copias_seguridad.error.catalogo")
			return 3
		}
		err = leerEstricto(f, &catalogo)
		_ = f.Close()
		if err != nil {
			diagnosticar(diag, "copias_seguridad.error.catalogo")
			return 3
		}
	}
	var e entrada
	if leerEstricto(in, &e) != nil {
		diagnosticar(diag, "copias_seguridad.error.entrada")
		return 3
	}
	var resultado copias.Resultado
	if e.FormatoVersion != copias.FormatoVersion {
		resultado = copias.Resultado{Estado: copias.NoComprobable, Razones: []copias.Razon{{Codigo: "dato_no_comprobable", Clave: "entrada.formato_version", Esperado: "1", Obtenido: "formato_no_admitido", Accion: "actualizar_formato"}}}
	} else {
		resultado = copias.CompararVersiones(e.Manifiesto, e.Destino, e.Politica, e.Modo)
	}
	s := salida{Alcance: "comparacion_offline_declarada", HabilitaRestauracion: false, Autenticidad: "no_comprobada", VerificacionRestauracion: "no_comprobada", Resultado: resultado, EstadoClave: "copias_seguridad.estado." + string(resultado.Estado)}
	if catalogo != nil {
		var ok bool
		s.Mensaje, ok = catalogo[s.EstadoClave]
		if !ok {
			diagnosticar(diag, "copias_seguridad.error.catalogo")
			return 3
		}
		for _, r := range resultado.Razones {
			texto, ok := catalogo["copias_seguridad.razon."+r.Codigo]
			accion, okAccion := catalogo["copias_seguridad.accion."+r.Accion]
			if !ok || !okAccion {
				diagnosticar(diag, "copias_seguridad.error.catalogo")
				return 3
			}
			s.Mensajes = append(s.Mensajes, texto+" "+accion)
		}
	}
	if json.NewEncoder(out).Encode(s) != nil {
		diagnosticar(diag, "copias_seguridad.error.salida")
		return 3
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
