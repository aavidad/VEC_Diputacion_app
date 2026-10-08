// Command vec-ratificar-catalogo-admin consume un plan aprobado por una autoridad externa.
// La configuración DBA y el acto de aprobación no se crean desde esta CLI.
package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/shared/i18n"
)

const limiteDocumento = 32768
const limiteConexion = 8192

//go:embed textos/*.json
var textos embed.FS

type documentoPlan struct {
	Esquema         string       `json:"esquema"`
	Version         string       `json:"version"`
	OperacionRef    string       `json:"operacion_ref"`
	PreparadoEn     string       `json:"preparado_en"`
	CaducaEn        string       `json:"caduca_en"`
	RolRef          string       `json:"rol_ref"`
	RolSHA          string       `json:"rol_sha256"`
	ControlRevision json.Number  `json:"control_revision"`
	ControlSHA      string       `json:"control_sha256"`
	CatalogoSHA     string       `json:"catalogo_sha256"`
	PreimagenSHA    string       `json:"preimagen_sha256"`
	Descriptores    []descriptor `json:"descriptores"`
}

type descriptor struct {
	AccionRef         string    `json:"accion_ref"`
	Concesion         concesion `json:"concesion"`
	DimensionesAmbito []string  `json:"dimensiones_ambito"`
}

type concesion struct {
	Accion         string   `json:"accion"`
	ModuloID       string   `json:"modulo_id"`
	TipoRecurso    string   `json:"tipo_recurso"`
	Finalidades    []string `json:"finalidades"`
	GarantiaMinima string   `json:"garantia_minima"`
}

type aprobacionPrivada struct {
	HuellaPlanSHA256 string `json:"huella_plan_sha256"`
	AprobacionRef    string `json:"aprobacion_ref"`
	AprobacionSHA256 string `json:"aprobacion_sha256"`
}

type diagnostico struct {
	Codigo     string `json:"codigo"`
	Mensaje    string `json:"mensaje"`
	Estado     string `json:"estado"`
	Confirmado bool   `json:"confirmado"`
	Replay     bool   `json:"replay"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr, nuevaTransaccionPG)) }

func ejecutar(args []string, out, errout io.Writer, abrir abrirTransaccion) int {
	f := flag.NewFlagSet("vec-ratificar-catalogo-admin", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var planRuta, conexionRuta, aprobacionRuta, acuseRuta, idioma string
	f.StringVar(&planRuta, "plan", "", "")
	f.StringVar(&conexionRuta, "conexion", "", "")
	f.StringVar(&aprobacionRuta, "aprobacion", "", "")
	f.StringVar(&acuseRuta, "acuse", "", "")
	f.StringVar(&idioma, "idioma", "es", "")
	parseErr := f.Parse(args)
	catalogo, err := cargarTextos(idioma)
	emitir := func(w io.Writer, d diagnostico, salida int) int {
		if catalogo != nil {
			d.Mensaje = catalogo.T(idioma, d.Codigo)
		}
		_ = json.NewEncoder(w).Encode(d)
		return salida
	}
	fallo := func(c string) int { return emitir(errout, diagnostico{Codigo: c, Estado: "sin_confirmar"}, 1) }
	if err != nil {
		return fallo("catalogo_no_disponible")
	}
	if parseErr != nil || f.NArg() != 0 || abrir == nil || !rutasDistintas(planRuta, conexionRuta, aprobacionRuta, acuseRuta) {
		return fallo("uso_invalido")
	}
	plan, err := leerPrivado(planRuta)
	if err != nil {
		return fallo("entrada_insegura")
	}
	defer clear(plan)
	conexion, err := leerPrivadoLimite(conexionRuta, limiteConexion)
	if err != nil {
		return fallo("entrada_insegura")
	}
	defer clear(conexion)
	aprobacion, err := leerPrivado(aprobacionRuta)
	if err != nil {
		return fallo("entrada_insegura")
	}
	defer clear(aprobacion)
	var p documentoPlan
	var c conexionPrivada
	var a aprobacionPrivada
	if !utf8.Valid(plan) || decodificarEstricto(plan, &p) != nil || decodificarEstricto(conexion, &c) != nil ||
		decodificarEstricto(aprobacion, &a) != nil || !validarPlan(p) || c.DSN == "" ||
		len(c.DSN) > 4096 || !hashValido(a.HuellaPlanSHA256) ||
		!referenciaAprobacion(a.AprobacionRef) || !hashValido(a.AprobacionSHA256) {
		return fallo("entrada_invalida")
	}
	h := sha256.Sum256(plan)
	sha := hex.EncodeToString(h[:])
	if sha != a.HuellaPlanSHA256 {
		return fallo("aprobacion_divergente")
	}
	raiz, err := abrirRaizPrivada(acuseRuta)
	if err != nil {
		return fallo("acuse_inseguro")
	}
	defer raiz.Close()
	// Reservar el nombre antes del efecto evita sobrescribir un recibo existente.
	archivo, err := raiz.OpenFile(filepath.Base(acuseRuta), os.O_WRONLY|os.O_CREATE|os.O_EXCL|noSeguirEnlaces, 0600)
	if err != nil {
		return fallo("acuse_inseguro")
	}
	guardado := false
	defer func() {
		_ = archivo.Close()
		if !guardado {
			_ = raiz.Remove(filepath.Base(acuseRuta))
		}
	}()
	if archivo.Chmod(0600) != nil {
		return fallo("acuse_inseguro")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resultado, err := ejecutarOperacion(ctx, c, plan, sha, p, a, abrir)
	if err == errCommit {
		return emitir(errout, diagnostico{Codigo: "commit_indeterminado", Estado: "indeterminado"}, 2)
	}
	if err != nil {
		return fallo("ratificacion_no_confirmada")
	}
	if resultado.Estado == "denegado" {
		return emitir(errout, diagnostico{Codigo: "ratificacion_rechazada", Estado: "denegado"}, 1)
	}
	if resultado.Estado == "error" {
		return emitir(errout, diagnostico{Codigo: "ratificacion_no_disponible", Estado: "error"}, 1)
	}
	if _, err = archivo.Write(append(resultado.Acuse, '\n')); err != nil {
		return emitir(errout, diagnostico{Codigo: "acuse_no_guardado", Estado: "confirmado", Confirmado: true, Replay: resultado.Replay}, 2)
	}
	if archivo.Sync() != nil || archivo.Close() != nil {
		return emitir(errout, diagnostico{Codigo: "acuse_no_guardado", Estado: "confirmado", Confirmado: true, Replay: resultado.Replay}, 2)
	}
	guardado = true
	return emitir(out, diagnostico{Codigo: "ratificacion_confirmada", Estado: "confirmado", Confirmado: true, Replay: resultado.Replay}, 0)
}

func rutasDistintas(rutas ...string) bool {
	m := map[string]bool{}
	for _, p := range rutas {
		if !filepath.IsAbs(p) || filepath.Clean(p) != p || m[p] {
			return false
		}
		m[p] = true
	}
	return true
}

func cargarTextos(idioma string) (*i18n.Catalog, error) {
	if idioma != "es" && idioma != "en" {
		return nil, os.ErrInvalid
	}
	b, err := textos.ReadFile("textos/" + idioma + ".json")
	if err != nil {
		return nil, err
	}
	var v struct {
		Idioma   string            `json:"idioma"`
		Mensajes map[string]string `json:"mensajes"`
	}
	if decodificarEstricto(b, &v) != nil || v.Idioma != idioma || len(v.Mensajes) == 0 {
		return nil, os.ErrInvalid
	}
	return i18n.New(idioma, map[string]map[string]string{idioma: v.Mensajes})
}
