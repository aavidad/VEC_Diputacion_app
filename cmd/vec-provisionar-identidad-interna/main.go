// Command vec-provisionar-identidad-interna invoca la autoridad técnica AUT57 sin
// publicar configuración, LOGIN, GRANT ni permisos desde la consola.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"time"

	"vec-diputacion-granada/internal/shared/i18n"
	"vec-diputacion-granada/internal/vec/domain"
)

const limiteDocumento = 64 << 10

type documento struct {
	Plan             domain.PlanIdentidadInternaSinteticaV1 `json:"plan"`
	HuellaPlanSHA256 string                                 `json:"huella_plan_sha256"`
}
type aprobacionPrivada struct {
	HuellaPlanSHA256    string `json:"huella_plan_sha256"`
	PreimagenSHA256     string `json:"preimagen_sha256"`
	ConfiguracionSHA256 string `json:"configuracion_sha256"`
	AprobacionRef       string `json:"aprobacion_ref"`
}
type datosTextos struct {
	Idioma   string            `json:"idioma"`
	Mensajes map[string]string `json:"mensajes"`
}
type diagnostico struct {
	Codigo        string `json:"codigo"`
	Mensaje       string `json:"mensaje"`
	Limite        string `json:"limite"`
	Estado        string `json:"estado"`
	Confirmado    bool   `json:"confirmado"`
	AcuseGuardado bool   `json:"acuse_guardado"`
	Replay        bool   `json:"replay"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr, nuevaTransaccionPG)) }
func ejecutar(args []string, salida, errores io.Writer, abrir abrirTransaccion) int {
	if len(args) == 0 {
		return informarFalloCatalogo(errores, os.ErrInvalid)
	}
	modo := args[0]
	args = args[1:]
	f := flag.NewFlagSet("vec-provisionar-identidad-interna", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var rutaFuente string
	f.StringVar(&rutaFuente, "fuente", "", "")
	var rutaPlan, rutaConexion, rutaAprobacion, rutaAcuse, rutaTextos, timeout string
	f.StringVar(&rutaPlan, "plan", "", "")
	f.StringVar(&rutaConexion, "conexion", "", "")
	f.StringVar(&rutaAprobacion, "aprobacion", "", "")
	f.StringVar(&rutaAcuse, "acuse", "", "")
	f.StringVar(&rutaTextos, "textos", "", "")
	f.StringVar(&timeout, "timeout", "", "")
	parseErr := f.Parse(args)
	textos, idioma, err := cargarTextos(rutaTextos)
	if err != nil {
		return informarFalloCatalogo(errores, err)
	}
	emitir := func(w io.Writer, d diagnostico, exit int) int {
		d.Mensaje = textos.T(idioma, d.Codigo)
		d.Limite = textos.T(idioma, "limite")
		if json.NewEncoder(w).Encode(d) != nil {
			return 2
		}
		return exit
	}
	fallo := func(codigo string) int {
		return emitir(errores, diagnostico{Codigo: codigo, Estado: "sin_confirmar"}, 1)
	}
	if modo == "plan" {
		return prepararPlan(f, parseErr, rutaFuente, rutaPlan, rutaConexion, rutaAprobacion, rutaAcuse, fallo, emitir, salida)
	}
	duracion, err := time.ParseDuration(timeout)
	if parseErr != nil || (modo != "apply" && modo != "reconcile") || rutaFuente != "" || err != nil || duracion <= 0 || duracion > time.Minute || f.NArg() != 0 || abrir == nil || !rutasDistintas(rutaPlan, rutaConexion, rutaAprobacion, rutaAcuse, rutaTextos) {
		return fallo("uso_invalido")
	}
	pb, err := leerPrivado(rutaPlan)
	if err != nil {
		return fallo("entrada_insegura")
	}
	defer clear(pb)
	cb, err := leerPrivado(rutaConexion)
	if err != nil {
		return fallo("entrada_insegura")
	}
	defer clear(cb)
	ab, err := leerPrivado(rutaAprobacion)
	if err != nil {
		return fallo("entrada_insegura")
	}
	defer clear(ab)
	var plan documento
	var cfg conexionPrivada
	var aprobacion aprobacionPrivada
	if decodificarEstricto(pb, &plan) != nil || decodificarEstricto(cb, &cfg) != nil || decodificarEstricto(ab, &aprobacion) != nil || cfg.DSN == "" || cfg.PermitirSocketDesarrollo || !hashValido(aprobacion.HuellaPlanSHA256) || !hashValido(aprobacion.PreimagenSHA256) || !hashValido(aprobacion.ConfiguracionSHA256) || aprobacion.AprobacionRef == "" {
		return fallo("entrada_invalida")
	}
	canon, sha, err := plan.Plan.CanonicoYHuella()
	if err != nil || sha != plan.HuellaPlanSHA256 || sha != aprobacion.HuellaPlanSHA256 || sha != cfg.HuellaPlanSHA256 || cfg.PreimagenSHA256 != aprobacion.PreimagenSHA256 || cfg.ConfiguracionSHA256 != aprobacion.ConfiguracionSHA256 || cfg.AprobacionRef != aprobacion.AprobacionRef {
		return fallo("entrada_invalida")
	}
	defer clear(canon)
	// Se reserva el archivo de acuse antes de enviar. Cada invocación usa otro
	// destino: replay recupera el recibo original y añade su propio intento.
	raiz, err := abrirRaizPrivada(rutaAcuse)
	if err != nil {
		return fallo("acuse_inseguro")
	}
	defer raiz.Close()
	archivo, err := raiz.OpenFile(filepath.Base(rutaAcuse), os.O_WRONLY|os.O_CREATE|os.O_EXCL|noSeguirEnlaces, 0600)
	if err != nil {
		return fallo("acuse_inseguro")
	}
	guardado := false
	defer func() {
		_ = archivo.Close()
		if !guardado {
			_ = raiz.Remove(filepath.Base(rutaAcuse))
		}
	}()
	if archivo.Chmod(0600) != nil {
		return fallo("acuse_inseguro")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), duracion)
	defer cancelar()
	b, e, err := ejecutarOperacion(ctx, cfg, duracion, canon, aprobacion.HuellaPlanSHA256, plan, modo, abrir)
	if err == errCommit {
		return emitir(errores, diagnostico{Codigo: "commit_no_confirmado", Estado: "indeterminado"}, 2)
	}
	if err != nil {
		return fallo("operacion_no_confirmada")
	}
	defer clear(b)
	b = append(b, '\n')
	if _, err = archivo.Write(b); err != nil {
		return emitir(errores, diagnostico{Codigo: "acuse_no_guardado", Estado: e.Estado, Confirmado: true, Replay: e.Replay}, 2)
	}
	if archivo.Sync() != nil || archivo.Close() != nil {
		return emitir(errores, diagnostico{Codigo: "acuse_no_guardado", Estado: e.Estado, Confirmado: true, Replay: e.Replay}, 2)
	}
	guardado = true
	codigo, exit := "fuentes_confirmadas", 0
	if e.Estado == "denegado" {
		codigo, exit = "fuentes_rechazadas", 1
	}
	if e.Estado == "error" {
		codigo, exit = "fuentes_no_disponibles", 1
	}
	return emitir(salida, diagnostico{Codigo: codigo, Estado: e.Estado, Confirmado: true, AcuseGuardado: true, Replay: e.Replay}, exit)
}
func rutasDistintas(rutas ...string) bool {
	vistas := map[string]bool{}
	for _, ruta := range rutas {
		if ruta == "" || vistas[ruta] {
			return false
		}
		vistas[ruta] = true
	}
	return true
}
func cargarTextos(ruta string) (*i18n.Catalog, string, error) {
	raiz, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return nil, "", err
	}
	defer raiz.Close()
	f, err := raiz.OpenFile(filepath.Base(ruta), os.O_RDONLY|noSeguirEnlaces, 0)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limiteDocumento {
		return nil, "", os.ErrInvalid
	}
	b, err := io.ReadAll(io.LimitReader(f, limiteDocumento+1))
	if err != nil || len(b) > limiteDocumento {
		return nil, "", os.ErrInvalid
	}
	var datos datosTextos
	if decodificarEstricto(b, &datos) != nil {
		return nil, "", os.ErrInvalid
	}
	claves := []string{"limite", "uso_invalido", "entrada_insegura", "entrada_invalida", "acuse_inseguro", "operacion_no_confirmada", "commit_no_confirmado", "acuse_no_guardado", "fuentes_confirmadas", "fuentes_rechazadas", "fuentes_no_disponibles", "plan_preparado"}
	if len(datos.Mensajes) != len(claves) {
		return nil, "", os.ErrInvalid
	}
	for _, k := range claves {
		if datos.Mensajes[k] == "" {
			return nil, "", os.ErrInvalid
		}
	}
	c, err := i18n.New(datos.Idioma, map[string]map[string]string{datos.Idioma: datos.Mensajes})
	return c, datos.Idioma, err
}
