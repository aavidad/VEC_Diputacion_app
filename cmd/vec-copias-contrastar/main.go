package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	a "vec-diputacion-granada/internal/modules/administracion/adapters/contrastecopias"
	s "vec-diputacion-granada/internal/modules/administracion/application/contrastecopias"
	d "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

type configuracion struct {
	DSN                       string                   `json:"dsn"`
	VersionPostgreSQL         string                   `json:"version_postgresql"`
	TiempoMaximoSegundos      int64                    `json:"tiempo_maximo_segundos"`
	MaxFilas                  int64                    `json:"max_filas"`
	MaxBytes                  int64                    `json:"max_bytes"`
	MaxObjetos                int                      `json:"max_objetos"`
	ObjetosGrandesSemanticos  bool                     `json:"objetos_grandes_semanticos"`
	ReferenciasObjetosGrandes []referenciaObjetoGrande `json:"referencias_objetos_grandes"`
	BasesInventariadas        []string                 `json:"bases_inventariadas"`
}

type referenciaObjetoGrande struct {
	Base    string `json:"base"`
	Esquema string `json:"esquema"`
	Tabla   string `json:"tabla"`
	Columna string `json:"columna"`
}

type diagnostico struct {
	Clave   string `json:"clave"`
	Mensaje string `json:"mensaje,omitempty"`
}

var errEntrada = errors.New("copias_contraste_error_entrada")

// JSON duplicado no tiene una interpretación única entre lectores.
func unico(dec *json.Decoder, nivel int, tipo reflect.Type) error {
	if nivel > 32 {
		return errEntrada
	}
	for tipo != nil && tipo.Kind() == reflect.Pointer {
		tipo = tipo.Elem()
	}
	t, err := dec.Token()
	if err != nil {
		return errEntrada
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		keys := map[string]bool{}
		var campos map[string]reflect.Type
		if tipo != nil && tipo.Kind() == reflect.Struct {
			campos = map[string]reflect.Type{}
			for i := 0; i < tipo.NumField(); i++ {
				f := tipo.Field(i)
				if !f.IsExported() {
					continue
				}
				nombre := strings.Split(f.Tag.Get("json"), ",")[0]
				if nombre == "-" {
					continue
				}
				if nombre == "" {
					nombre = f.Name
				}
				campos[nombre] = f.Type
			}
		}
		for dec.More() {
			k, e := dec.Token()
			if e != nil {
				return errEntrada
			}
			key, ok := k.(string)
			if !ok || keys[key] {
				return errEntrada
			}
			keys[key] = true
			var valor reflect.Type
			if campos != nil {
				var existe bool
				valor, existe = campos[key]
				if !existe {
					return errEntrada
				}
			} else if tipo != nil && tipo.Kind() == reflect.Map {
				valor = tipo.Elem()
			}
			if unico(dec, nivel+1, valor) != nil {
				return errEntrada
			}
		}
	case '[':
		var elemento reflect.Type
		if tipo != nil && (tipo.Kind() == reflect.Slice || tipo.Kind() == reflect.Array) {
			elemento = tipo.Elem()
		}
		for dec.More() {
			if unico(dec, nivel+1, elemento) != nil {
				return errEntrada
			}
		}
	default:
		return errEntrada
	}
	_, err = dec.Token()
	return err
}

func leer(ruta string, destino any, limite int64) error {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK, 0) // #nosec G304 G703 -- entrada local explícita, regular y acotada; nunca se reproduce su ruta en errores.
	if err != nil {
		return errEntrada
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > limite {
		return errEntrada
	}
	b, err := io.ReadAll(io.LimitReader(f, limite+1))
	if err != nil || int64(len(b)) > limite || !utf8.Valid(b) {
		return errEntrada
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	if unico(dec, 0, reflect.TypeOf(destino)) != nil {
		return errEntrada
	}
	if _, err = dec.Token(); err != io.EOF {
		return errEntrada
	}
	dec = json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(destino) != nil {
		return errEntrada
	}
	return nil
}

var claves = []string{"alcance", "igual", "diferente", "no_comprobable", "captura_completada", "copias_contraste_error_argumentos", "copias_contraste_error_catalogo", "copias_contraste_error_entrada", "copias_contraste_error_configuracion", "copias_contraste_error_captura", "copias_contraste_error_salida", "ayuda"}

func emitirDiagnostico(w io.Writer, mensaje diagnostico, codigo int) int {
	if json.NewEncoder(w).Encode(mensaje) != nil {
		return 2
	}
	return codigo
}

func run(ctx context.Context, args []string, out, diag io.Writer) int {
	flags := flag.NewFlagSet("vec-copias-contrastar", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var catalogo, config, esperado, observado, modo string
	var ayuda bool
	flags.StringVar(&modo, "modo", "", "")
	flags.StringVar(&catalogo, "catalogo", "", "")
	flags.StringVar(&config, "configuracion", "", "")
	flags.StringVar(&esperado, "esperado", "", "")
	flags.StringVar(&observado, "observado", "", "")
	flags.BoolVar(&ayuda, "ayuda", false, "")
	textos := map[string]string{}
	emitir := func(clave string, codigo int) int {
		return emitirDiagnostico(diag, diagnostico{clave, textos[clave]}, codigo)
	}
	if flags.Parse(args) != nil || flags.NArg() != 0 || catalogo == "" {
		return emitir("copias_contraste_error_argumentos", 2)
	}
	if leer(catalogo, &textos, 1<<20) != nil {
		textos = nil
		return emitir("copias_contraste_error_catalogo", 2)
	}
	for _, k := range claves {
		if textos[k] == "" {
			return emitir("copias_contraste_error_catalogo", 2)
		}
	}
	if ayuda {
		if modo != "" || config != "" || esperado != "" || observado != "" {
			return emitir("copias_contraste_error_argumentos", 2)
		}
		if _, err := io.WriteString(out, textos["ayuda"]+"\n"); err != nil {
			return emitir("copias_contraste_error_salida", 2)
		}
		return 0
	}
	servicio := s.Servicio{}
	switch modo {
	case "capturar":
		if config == "" || esperado != "" || observado != "" {
			return emitir("copias_contraste_error_argumentos", 2)
		}
		var c configuracion
		if leer(config, &c, 1<<20) != nil || c.DSN == "" || c.TiempoMaximoSegundos < 1 || c.TiempoMaximoSegundos > 600 {
			return emitir("copias_contraste_error_configuracion", 2)
		}
		refs := make([]a.ReferenciaObjetoGrande, 0, len(c.ReferenciasObjetosGrandes))
		for _, r := range c.ReferenciasObjetosGrandes {
			refs = append(refs, a.ReferenciaObjetoGrande{Base: r.Base, Esquema: r.Esquema, Tabla: r.Tabla, Columna: r.Columna})
		}
		lector, err := a.Nuevo(a.Configuracion{DSN: c.DSN, VersionPostgreSQL: c.VersionPostgreSQL, TiempoMaximo: time.Duration(c.TiempoMaximoSegundos) * time.Second, MaxFilas: c.MaxFilas, MaxBytes: c.MaxBytes, MaxObjetos: c.MaxObjetos, ObjetosGrandesSemanticos: c.ObjetosGrandesSemanticos, ReferenciasObjetosGrandes: refs, BasesInventariadas: c.BasesInventariadas})
		if err != nil {
			return emitir("copias_contraste_error_configuracion", 2)
		}
		servicio.Lector = lector
		snapshot, err := servicio.Capturar(ctx)
		if err != nil {
			return emitir("copias_contraste_error_captura", 2)
		}
		if json.NewEncoder(out).Encode(snapshot) != nil {
			return emitir("copias_contraste_error_salida", 2)
		}
		if len(d.Validar(snapshot)) != 0 {
			return emitir("no_comprobable", 1)
		}
		return emitir("captura_completada", 0)
	case "comparar":
		if config != "" || esperado == "" || observado == "" {
			return emitir("copias_contraste_error_argumentos", 2)
		}
		var a, b d.Snapshot
		if leer(esperado, &a, 32<<20) != nil || leer(observado, &b, 32<<20) != nil {
			return emitir("copias_contraste_error_entrada", 2)
		}
		r := servicio.Comparar(a, b)
		respuesta := struct {
			Alcance   string      `json:"alcance"`
			Mensaje   string      `json:"mensaje"`
			Resultado d.Resultado `json:"resultado"`
		}{textos["alcance"], textos[r.Estado], r}
		if json.NewEncoder(out).Encode(respuesta) != nil {
			return emitir("copias_contraste_error_salida", 2)
		}
		if r.Estado == d.Igual {
			return 0
		}
		return 1
	default:
		return emitir("copias_contraste_error_argumentos", 2)
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	codigo := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	cancel()
	os.Exit(codigo)
}
