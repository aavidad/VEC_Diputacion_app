// vec-certificados-borrador is the local rehearsal consumer of the preparation
// ports. It does not connect to Personal, HTTP, signature providers or a database.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/certificados/adapters/fichero"
	"vec-diputacion-granada/internal/modules/certificados/adapters/pdf"
	"vec-diputacion-granada/internal/modules/certificados/adapters/personalv1"
	"vec-diputacion-granada/internal/modules/certificados/application"
	"vec-diputacion-granada/internal/modules/certificados/domain"
	"vec-diputacion-granada/internal/modules/certificados/ports"
)

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr)) }

// These two bootstrap options locate the catalogue before flag help or errors
// are printed. If it cannot load, the only diagnostic is a technical key.
func opcionInicial(args []string, nombre, defecto string) string {
	valor := defecto
	for i := 0; i < len(args); i++ {
		a := strings.TrimPrefix(args[i], "-")
		a = strings.TrimPrefix(a, "-")
		if a == nombre && i+1 < len(args) {
			i++
			valor = args[i]
		} else if strings.HasPrefix(a, nombre+"=") {
			valor = strings.TrimPrefix(a, nombre+"=")
		}
	}
	return valor
}
func ejecutar(args []string, salida, diagnostico io.Writer) int {
	rutaIndice := opcionInicial(args, "indice-idiomas", filepath.Join("web", "static", "textos", "idiomas.json"))
	var indice struct {
		PorDefecto      string `json:"por_defecto"`
		SeguirNavegador bool   `json:"seguir_navegador"`
		Idiomas         []struct {
			Codigo       string `json:"codigo"`
			Nombre       string `json:"nombre"`
			Localizacion string `json:"localizacion"`
		} `json:"idiomas"`
	}
	if fichero.LeerJSON(rutaIndice, &indice) != nil || !domain.IdiomaValido(indice.PorDefecto) {
		fmt.Fprintln(diagnostico, "certificados.textos_no_disponibles")
		return 2
	}
	idioma := opcionInicial(args, "idioma", indice.PorDefecto)
	admitidos := map[string]bool{}
	for _, i := range indice.Idiomas {
		if !domain.IdiomaValido(i.Codigo) || admitidos[i.Codigo] {
			fmt.Fprintln(diagnostico, "certificados.textos_no_disponibles")
			return 2
		}
		admitidos[i.Codigo] = true
	}
	if !admitidos[idioma] || !admitidos[indice.PorDefecto] {
		fmt.Fprintln(diagnostico, "certificados.textos_no_disponibles")
		return 2
	}
	rutaTextos := opcionInicial(args, "textos", filepath.Join("web", "static", "textos", idioma, "certificados.json"))
	var textos domain.Textos
	if fichero.LeerJSON(rutaTextos, &textos) != nil || textos.Validar() != nil || textos.Idioma != idioma {
		fmt.Fprintln(diagnostico, "certificados.textos_no_disponibles")
		return 2
	}
	fs := flag.NewFlagSet("vec-certificados-borrador", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		fmt.Fprintln(salida, textos.Mensaje("cli_uso"))
		fs.VisitAll(func(f *flag.Flag) { fmt.Fprintln(salida, "-"+f.Name+": "+f.Usage) })
	}
	var fuente, plantilla, destino string
	var version int
	var ensayo bool
	fs.StringVar(&fuente, "fuente", "", textos.Mensaje("cli_fuente"))
	fs.StringVar(&plantilla, "plantilla", "data/certificados/plantillas/servicios.v1.json", textos.Mensaje("cli_plantilla"))
	fs.IntVar(&version, "version", 0, textos.Mensaje("cli_version"))
	fs.StringVar(&idioma, "idioma", idioma, textos.Mensaje("cli_idioma"))
	fs.StringVar(&rutaTextos, "textos", rutaTextos, textos.Mensaje("cli_textos"))
	fs.StringVar(&rutaIndice, "indice-idiomas", rutaIndice, textos.Mensaje("cli_indice_idiomas"))
	fs.StringVar(&destino, "salida", "", textos.Mensaje("cli_salida"))
	fs.BoolVar(&ensayo, "ensayo-sintetico", false, textos.Mensaje("cli_ensayo"))
	if e := fs.Parse(args); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintln(diagnostico, textos.Mensaje("error_argumentos"))
		return 2
	}
	if !ensayo || !domain.TextoValido(fuente, 1024) || !domain.TextoValido(destino, 1024) || fs.NArg() != 0 || idioma != textos.Idioma {
		fmt.Fprintln(diagnostico, textos.Mensaje("error_argumentos"))
		return 2
	}
	var definicion domain.Plantilla
	if fichero.LeerJSON(plantilla, &definicion) != nil {
		fmt.Fprintln(diagnostico, textos.Mensaje("error_catalogo"))
		return 1
	}
	if version == 0 {
		version = definicion.Version
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	preparador := application.Preparador{Fuente: elegirFuente(fuente),
		Catalogo: fichero.CatalogoPlantillas{RutaPlantilla: plantilla, RutaTextos: rutaTextos}, Renderizador: pdf.Renderizador{}}
	r, e := preparador.PrepararEnsayo(ctx, application.Orden{PlantillaID: definicion.ID, Version: version, Idioma: idioma})
	if e != nil {
		mensajeID := "error_no_disponible"
		if errors.Is(e, domain.ErrEntrada) {
			mensajeID = "error_entrada"
		}
		if errors.Is(e, domain.ErrCatalogo) {
			mensajeID = "error_catalogo"
		}
		fmt.Fprintln(diagnostico, textos.Mensaje(mensajeID))
		return 1
	}
	if guardar(destino, r) != nil {
		fmt.Fprintln(diagnostico, textos.Mensaje("error_salida"))
		return 1
	}
	fmt.Fprintln(salida, textos.Mensaje("cli_ok", "salida", destino))
	return 0
}

// The new folder is reserved exclusively. Both files use private permissions;
// failure removes only files created by this command, never an existing folder.
func guardar(destino string, r application.Resultado) error {
	b, e := json.MarshalIndent(r.Borrador, "", "  ")
	if e != nil {
		return e
	}
	if e = os.Mkdir(destino, 0700); e != nil {
		return e
	}
	archivos := []struct {
		nombre    string
		contenido []byte
	}{{"borrador.pdf", r.PDF}, {"borrador.json", append(b, '\n')}}
	creados := []string{}
	completo := false
	defer func() {
		if !completo {
			for _, ruta := range creados {
				_ = os.Remove(ruta)
			}
			_ = os.Remove(destino)
		}
	}()
	for _, a := range archivos {
		ruta := filepath.Join(destino, a.nombre)
		f, e := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600) // #nosec G304 -- fixed names in a newly reserved local folder.
		if e != nil {
			return e
		}
		creados = append(creados, ruta)
		_, escribir := f.Write(a.contenido)
		cerrar := f.Close()
		if escribir != nil {
			return escribir
		}
		if cerrar != nil {
			return cerrar
		}
	}
	completo = true
	return nil
}

// elegirFuente compone el adaptador según el esquema de la muestra: el formato
// de ensayo propio o la forma de la respuesta V1 de Personal. Una muestra
// ilegible sigue al adaptador de ensayo, que la rechaza con su error.
func elegirFuente(ruta string) ports.FuenteServicios {
	var cabecera map[string]json.RawMessage
	var esquema string
	if fichero.LeerJSON(ruta, &cabecera) == nil && json.Unmarshal(cabecera["esquema"], &esquema) == nil && esquema == personalv1.EsquemaMuestra {
		return personalv1.FicheroMuestra{Ruta: ruta}
	}
	return fichero.FuenteServicios{Ruta: ruta}
}
