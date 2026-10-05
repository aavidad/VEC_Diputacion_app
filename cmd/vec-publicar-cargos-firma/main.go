// Command vec-publicar-cargos-firma prepara y aplica la publicación gobernada
// de las versiones de rol de los cargos del plan nominal de firma (AUT53).
//
//	preparar: fichero de cargos → plan canónico con las huellas de cada rol.
//	aplicar:  plan aprobado → publicar_cargos_firma_admin_v1 con el LOGIN técnico.
//
// No concede permisos, no crea la aprobación ni el LOGIN y no asigna cargos
// a personas: eso lo hace Administración con el lote ordinario.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/shared/i18n"
)

const limiteDocumento = 65536

type aprobacionPrivada struct {
	HuellaPlanSHA256 string `json:"huella_plan_sha256"`
}
type diagnostico struct {
	Codigo     string `json:"codigo"`
	Mensaje    string `json:"mensaje,omitempty"`
	Estado     string `json:"estado"`
	Confirmado bool   `json:"confirmado"`
	Replay     bool   `json:"replay"`
	PlanSHA256 string `json:"plan_sha256,omitempty"`
}

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr, nuevaTransaccionPG, time.Now))
}

func ejecutar(args []string, out, errout io.Writer, abrir abrirTransaccion, ahora func() time.Time) int {
	orden := ""
	if len(args) > 0 {
		orden, args = args[0], args[1:]
	}
	f := flag.NewFlagSet("vec-publicar-cargos-firma", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var cargosRuta, planRuta, conexionRuta, aprobacionRuta, acuseRuta, textosRuta, idioma, timeout, caduca string
	f.StringVar(&textosRuta, "textos", "", "")
	f.StringVar(&idioma, "idioma", "", "")
	f.StringVar(&planRuta, "plan", "", "")
	switch orden {
	case "preparar":
		f.StringVar(&cargosRuta, "cargos", "", "")
		f.StringVar(&caduca, "caduca", "", "")
	case "aplicar":
		f.StringVar(&conexionRuta, "conexion", "", "")
		f.StringVar(&aprobacionRuta, "aprobacion", "", "")
		f.StringVar(&acuseRuta, "acuse", "", "")
		f.StringVar(&timeout, "timeout", "", "")
	}
	parseErr := f.Parse(args)
	catalogo, e := cargarTextos(textosRuta, idioma)
	emitir := func(w io.Writer, d diagnostico, exit int) int {
		if catalogo != nil {
			d.Mensaje = catalogo.T(idioma, d.Codigo)
		}
		b, e := json.Marshal(d)
		if e == nil {
			_, _ = w.Write(append(b, '\n'))
		}
		return exit
	}
	fallo := func(c string) int { return emitir(errout, diagnostico{Codigo: c, Estado: "sin_confirmar"}, 1) }
	if e != nil {
		return fallo("catalogo_no_disponible")
	}
	if parseErr != nil || f.NArg() != 0 || ahora == nil {
		return fallo("uso_invalido")
	}
	switch orden {
	case "preparar":
		return preparar(cargosRuta, planRuta, textosRuta, caduca, ahora(), out, emitir, fallo)
	case "aplicar":
		return aplicar(planRuta, conexionRuta, aprobacionRuta, acuseRuta, textosRuta, timeout, abrir, out, errout, emitir, fallo)
	}
	return fallo("uso_invalido")
}

type emisor func(io.Writer, diagnostico, int) int

func preparar(cargosRuta, planRuta, textosRuta, caduca string, ahora time.Time, out io.Writer, emitir emisor, fallo func(string) int) int {
	duracion, e := time.ParseDuration(caduca)
	if e != nil || !rutasDistintas(cargosRuta, planRuta, textosRuta) {
		return fallo("uso_invalido")
	}
	cb, e := leerPrivado(cargosRuta)
	if e != nil {
		return fallo("entrada_insegura")
	}
	defer clear(cb)
	var fc ficheroCargos
	if decodificarEstricto(cb, &fc) != nil {
		return fallo("entrada_invalida")
	}
	op, e := nuevaOperacionRef()
	if e != nil {
		return fallo("operacion_no_confirmada")
	}
	p, e := prepararPlan(fc, op, ahora.UTC().Truncate(time.Second), duracion)
	if e != nil {
		return fallo("cargos_invalidos")
	}
	b := textoPlan(p)
	if crearOComparar(planRuta, b) != nil {
		return fallo("plan_no_guardado")
	}
	h := sha256.Sum256(b)
	return emitir(out, diagnostico{Codigo: "plan_preparado", Estado: "preparado", PlanSHA256: hex.EncodeToString(h[:])}, 0)
}

// planCoherente vuelve a calcular en Go lo que la base comprobará: texto
// canónico y huella de cada versión de rol. Un plan editado a mano se para
// aquí, antes de conectar.
func planCoherente(pb []byte, p documentoPlan) bool {
	if p.Esquema != esquemaPlan || !reOperacion.MatchString(p.OperacionRef) || len(p.Cargos) < 1 || len(p.Cargos) > maximoCargos ||
		!bytes.Equal(textoPlan(p), pb) {
		return false
	}
	for _, c := range p.Cargos {
		v, err := versionRol(c.entrada(), p.OperacionRef, p.PreparadoEn)
		if err != nil || validarCargo(c.entrada()) != nil {
			return false
		}
		h, err := v.HuellaSHA256()
		if err != nil || h != c.VersionRolSHA256 {
			return false
		}
	}
	return true
}

func aplicar(planRuta, conexionRuta, aprobacionRuta, acuseRuta, textosRuta, timeout string, abrir abrirTransaccion, out, errout io.Writer, emitir emisor, fallo func(string) int) int {
	duracion, e := time.ParseDuration(timeout)
	if e != nil || duracion <= 0 || abrir == nil || !rutasDistintas(planRuta, conexionRuta, aprobacionRuta, acuseRuta, textosRuta) {
		return fallo("uso_invalido")
	}
	pb, e := leerPrivado(planRuta)
	if e != nil {
		return fallo("entrada_insegura")
	}
	defer clear(pb)
	cb, e := leerPrivado(conexionRuta)
	if e != nil {
		return fallo("entrada_insegura")
	}
	defer clear(cb)
	ab, e := leerPrivado(aprobacionRuta)
	if e != nil {
		return fallo("entrada_insegura")
	}
	defer clear(ab)
	var p documentoPlan
	var c conexionPrivada
	var a aprobacionPrivada
	if decodificarEstricto(pb, &p) != nil || decodificarEstricto(cb, &c) != nil || decodificarEstricto(ab, &a) != nil || !hashValido(a.HuellaPlanSHA256) || c.DSN == "" || !planCoherente(pb, p) {
		return fallo("entrada_invalida")
	}
	h := sha256.Sum256(pb)
	sha := hex.EncodeToString(h[:])
	if sha != a.HuellaPlanSHA256 {
		return fallo("aprobacion_divergente")
	}
	root, e := abrirRaizPrivada(acuseRuta)
	if e != nil {
		return fallo("acuse_inseguro")
	}
	defer root.Close()
	archivo, e := root.OpenFile(filepath.Base(acuseRuta), os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if e != nil {
		return fallo("acuse_inseguro")
	}
	guardado := false
	defer func() {
		_ = archivo.Close()
		if !guardado {
			_ = root.Remove(filepath.Base(acuseRuta))
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), duracion)
	defer cancel()
	b, res, e := ejecutarOperacion(ctx, c, duracion, pb, a.HuellaPlanSHA256, p, abrir)
	if e == errCommit {
		return emitir(errout, diagnostico{Codigo: "commit_no_confirmado", Estado: "indeterminado"}, 2)
	}
	if e != nil {
		return fallo("operacion_no_confirmada")
	}
	defer clear(b)
	if _, e = archivo.Write(append(b, '\n')); e != nil || archivo.Sync() != nil || archivo.Close() != nil {
		return emitir(errout, diagnostico{Codigo: "acuse_no_guardado", Estado: res.Estado, Confirmado: true, Replay: res.Replay}, 2)
	}
	guardado = true
	codigo, exit := "cargos_confirmados", 0
	if res.Estado == "denegado" {
		codigo, exit = "cargos_rechazados", 1
	}
	if res.Estado == "error" {
		codigo, exit = "cargos_no_disponibles", 1
	}
	return emitir(out, diagnostico{Codigo: codigo, Estado: res.Estado, Confirmado: true, Replay: res.Replay}, exit)
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

func cargarTextos(ruta, idioma string) (*i18n.Catalog, error) {
	if !filepath.IsAbs(ruta) || idioma == "" {
		return nil, io.ErrUnexpectedEOF
	}
	root, e := os.OpenRoot(filepath.Dir(ruta))
	if e != nil {
		return nil, e
	}
	defer root.Close()
	f, e := root.OpenFile(filepath.Base(ruta), os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_CLOEXEC|syscall.O_NONBLOCK, 0)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil || !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > limiteDocumento {
		return nil, io.ErrUnexpectedEOF
	}
	b, e := io.ReadAll(io.LimitReader(f, limiteDocumento+1))
	if e != nil || len(b) > limiteDocumento {
		return nil, io.ErrUnexpectedEOF
	}
	var d struct {
		Idioma   string            `json:"idioma"`
		Mensajes map[string]string `json:"mensajes"`
	}
	if decodificarEstricto(b, &d) != nil || d.Idioma != idioma {
		return nil, io.ErrUnexpectedEOF
	}
	return i18n.New(idioma, map[string]map[string]string{idioma: d.Mensajes})
}
