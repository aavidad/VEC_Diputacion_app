package main

import (
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

type documentoPlan struct {
	Version         uint64          `json:"version"`
	OperacionRef    string          `json:"operacion_ref"`
	PreparadoEn     time.Time       `json:"preparado_en"`
	CaducaEn        time.Time       `json:"caduca_en"`
	RolOrigenSHA    string          `json:"rol_origen_sha256"`
	ControlRevision uint64          `json:"control_revision_esperada"`
	ControlSHA      string          `json:"control_huella_sha256"`
	CatalogoSHA     string          `json:"catalogo_sha256"`
	RolDestino      json.RawMessage `json:"rol_destino_doc"`
	Asignaciones    []objetivo      `json:"asignaciones"`
}
type objetivo struct {
	PerfilRef      string          `json:"perfil_ref"`
	AsignacionRef  string          `json:"asignacion_origen_ref"`
	AsignacionSHA  string          `json:"asignacion_origen_sha256"`
	PersonaRef     string          `json:"persona_ref"`
	CuentaRef      string          `json:"cuenta_ref"`
	VinculoRef     string          `json:"vinculo_ref"`
	CuentaVersion  uint64          `json:"cuenta_version"`
	PersonaVersion uint64          `json:"persona_version"`
	PerfilVersion  uint64          `json:"perfil_version"`
	VinculoVersion uint64          `json:"vinculo_version"`
	AmbitosFuente  json.RawMessage `json:"ambitos_fuente"`
}
type aprobacionPrivada struct {
	HuellaPlanSHA256 string `json:"huella_plan_sha256"`
}
type diagnostico struct {
	Codigo     string `json:"codigo"`
	Mensaje    string `json:"mensaje,omitempty"`
	Estado     string `json:"estado"`
	Confirmado bool   `json:"confirmado"`
	Replay     bool   `json:"replay"`
}

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr, nuevaTransaccionPG)) }
func ejecutar(args []string, out, errout io.Writer, abrir abrirTransaccion) int {
	f := flag.NewFlagSet("vec-mantener-admin-fijo", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var planRuta, conexionRuta, aprobacionRuta, acuseRuta, textosRuta, idioma, timeout string
	f.StringVar(&planRuta, "plan", "", "")
	f.StringVar(&conexionRuta, "conexion", "", "")
	f.StringVar(&aprobacionRuta, "aprobacion", "", "")
	f.StringVar(&acuseRuta, "acuse", "", "")
	f.StringVar(&textosRuta, "textos", "", "")
	f.StringVar(&idioma, "idioma", "", "")
	f.StringVar(&timeout, "timeout", "", "")
	parseErr := f.Parse(args)
	catalog, e := cargarTextos(textosRuta, idioma)
	emitir := func(w io.Writer, d diagnostico, exit int) int {
		if catalog != nil {
			d.Mensaje = catalog.T(idioma, d.Codigo)
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
	duracion, e := time.ParseDuration(timeout)
	if parseErr != nil || e != nil || duracion <= 0 || f.NArg() != 0 || abrir == nil || !rutasDistintas(planRuta, conexionRuta, aprobacionRuta, acuseRuta, textosRuta) {
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
	if decodificarEstricto(pb, &p) != nil || decodificarEstricto(cb, &c) != nil || decodificarEstricto(ab, &a) != nil || !versionMantenimientoAdmitida(p.Version) || len(p.Asignaciones) != 2 || !hashValido(p.RolOrigenSHA) || !hashValido(p.CatalogoSHA) || !hashValido(a.HuellaPlanSHA256) || c.DSN == "" {
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
	if _, e = archivo.Write(append(b, '\n')); e != nil {
		return emitir(errout, diagnostico{Codigo: "acuse_no_guardado", Estado: res.Estado, Confirmado: true, Replay: res.Replay}, 2)
	}
	if archivo.Sync() != nil || archivo.Close() != nil {
		return emitir(errout, diagnostico{Codigo: "acuse_no_guardado", Estado: res.Estado, Confirmado: true, Replay: res.Replay}, 2)
	}
	guardado = true
	codigo, exit := "mantenimiento_confirmado", 0
	if res.Estado == "denegado" {
		codigo, exit = "mantenimiento_rechazado", 1
	}
	if res.Estado == "error" {
		codigo, exit = "mantenimiento_no_disponible", 1
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
