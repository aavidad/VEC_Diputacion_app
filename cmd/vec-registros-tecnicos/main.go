package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"syscall"

	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	"vec-diputacion-granada/internal/vec/domain"
)

type textosRecolector struct {
	Esquema   string `json:"esquema"`
	Ayuda     string `json:"ayuda"`
	Terminado string `json:"terminado"`
	Error     string `json:"error"`
}

// La configuración y el catálogo son archivos locales elegidos por Sistemas.
// Nunca se escriben su ruta, contenido ni error de lectura en el diagnóstico.
func leerArchivoRecolector(ruta string, destino any) error {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0) // #nosec G304 -- explicit local configuration/catalogue path; no leaf symlink, regular file, bounded read.
	if err != nil {
		return os.ErrInvalid
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return os.ErrInvalid
	}
	b, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil || len(b) > 1<<20 {
		return os.ErrInvalid
	}
	defer clear(b)
	permitidas := []string{"esquema", "ayuda", "terminado", "error"}
	if _, configuracion := destino.(*observabilidad.ConfiguracionRecolector); configuracion {
		permitidas = []string{"directorio", "max_linea_bytes", "max_archivo_bytes", "max_archivos", "retencion_segundos", "ventana_alertas_segundos", "umbrales_alerta", "umbrales_resultado", "catalogo_incidencias"}
	}
	if clavesUnicasCLI(b, permitidas) != nil {
		return os.ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return os.ErrInvalid
	}
	return nil
}

func clavesUnicasCLI(datos []byte, permitidas []string) error {
	d := json.NewDecoder(bytes.NewReader(datos))
	inicio, err := d.Token()
	if err != nil || inicio != json.Delim('{') {
		return os.ErrInvalid
	}
	vistas := make(map[string]bool)
	for d.More() {
		token, err := d.Token()
		clave, ok := token.(string)
		admitida := false
		for _, p := range permitidas {
			admitida = admitida || clave == p
		}
		if err != nil || !ok || vistas[clave] || !admitida {
			return os.ErrInvalid
		}
		vistas[clave] = true
		var valor json.RawMessage
		if d.Decode(&valor) != nil {
			return os.ErrInvalid
		}
		if clave == "catalogo_incidencias" {
			var ruta string
			if json.Unmarshal(valor, &ruta) != nil || ruta == "" {
				return os.ErrInvalid
			}
		}
		if clave == "umbrales_alerta" || clave == "umbrales_resultado" {
			codigos := []string{}
			if clave == "umbrales_alerta" {
				for _, c := range domain.CodigosIncidenciaTecnica() {
					codigos = append(codigos, string(c))
				}
			} else {
				codigos = []string{string(domain.ResultadoTecnicoDenegado), string(domain.ResultadoTecnicoNoDisponible)}
			}
			if clavesUnicasCLI(valor, codigos) != nil {
				return os.ErrInvalid
			}
		}
	}
	fin, err := d.Token()
	if err != nil || fin != json.Delim('}') || d.Decode(new(any)) != io.EOF {
		return os.ErrInvalid
	}
	return nil
}

func ejecutar(args []string, entrada io.Reader, salida, diagnostico io.Writer) int {
	f := flag.NewFlagSet("vec-registros-tecnicos", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	configuracion := f.String("config", "", "")
	catalogo := f.String("textos", "", "")
	ayuda := f.Bool("ayuda", false, "")
	if f.Parse(args) != nil || f.NArg() != 0 || entrada == nil || salida == nil || diagnostico == nil {
		return 2
	}
	var textos textosRecolector
	if leerArchivoRecolector(*catalogo, &textos) != nil || textos.Esquema != "1" || textos.Ayuda == "" || textos.Terminado == "" || textos.Error == "" {
		return 2
	}
	if *ayuda {
		if json.NewEncoder(salida).Encode(struct {
			Ayuda string `json:"ayuda"`
		}{textos.Ayuda}) != nil {
			return 2
		}
		return 0
	}
	var cfg observabilidad.ConfiguracionRecolector
	if leerArchivoRecolector(*configuracion, &cfg) != nil {
		return errorCLI(diagnostico, textos.Error)
	}
	metricas, err := observabilidad.RecolectarIncidencias(entrada, diagnostico, cfg)
	if err != nil {
		return errorCLI(diagnostico, textos.Error)
	}
	if json.NewEncoder(salida).Encode(struct {
		Estado   string                            `json:"estado"`
		Mensaje  string                            `json:"mensaje"`
		Metricas observabilidad.MetricasRecolector `json:"metricas"`
	}{"terminado", textos.Terminado, metricas}) != nil {
		return 2
	}
	return 0
}

func errorCLI(destino io.Writer, mensaje string) int {
	_ = json.NewEncoder(destino).Encode(struct {
		Estado  string `json:"estado"`
		Mensaje string `json:"mensaje"`
	}{"recoleccion_no_completada", mensaje})
	return 2
}

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
