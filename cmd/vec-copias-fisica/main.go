// vec-copias-fisica ejecuta una captura local. Su configuración es una autoridad
// de plataforma privada; no debe exponerse este proceso como endpoint web.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	fisica "vec-diputacion-granada/internal/modules/administracion/adapters/capturafisica"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

type configuracion struct {
	Captura        fisica.Configuracion `json:"captura"`
	Control        fisica.ConfigControl `json:"control"`
	Inventario     copias.Inventario    `json:"inventario"`
	Bloqueo        string               `json:"bloqueo"`
	TiempoSegundos int                  `json:"tiempo_segundos"`
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
func run(args []string, out, diagnostico io.Writer) int {
	flags := flag.NewFlagSet("vec-copias-fisica", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	ruta := flags.String("config", "", "")
	if flags.Parse(args) != nil || *ruta == "" || flags.NArg() != 0 {
		fmt.Fprintln(diagnostico, "captura_fisica_uso")
		return 2
	}
	cfg, e := leer(*ruta)
	if e != nil || cfg.TiempoSegundos < 1 || cfg.TiempoSegundos > 86400 || cfg.Control.PGDATA != cfg.Captura.PGDATA {
		fmt.Fprintln(diagnostico, "captura_fisica_configuracion")
		return 2
	}
	control := fisica.ControlComandos{Config: cfg.Control}
	capturador := fisica.Capturador{Config: cfg.Captura, Control: control}
	if capturador.ComprobarConfiguracion(cfg.Inventario, cfg.Bloqueo) != nil {
		fmt.Fprintln(diagnostico, "captura_fisica_configuracion")
		return 2
	}
	lock, e := os.OpenFile(cfg.Bloqueo, os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if e != nil {
		fmt.Fprintln(diagnostico, "captura_fisica_bloqueo")
		return 1
	}
	defer lock.Close()
	fd := lock.Fd()
	if fd > uintptr(^uint(0)>>1) {
		fmt.Fprintln(diagnostico, "captura_fisica_bloqueo")
		return 1
	}
	info, e := lock.Stat()
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || syscall.Flock(int(fd), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		fmt.Fprintln(diagnostico, "captura_fisica_bloqueo")
		return 1
	}
	defer syscall.Flock(int(fd), syscall.LOCK_UN)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	ctx, cancelTiempo := context.WithTimeout(ctx, time.Duration(cfg.TiempoSegundos)*time.Second)
	defer cancelTiempo()
	var artefactos []copias.Artefacto
	e = control.CerrarEscritores(ctx)
	if e == nil {
		artefactos, e = capturador.Capturar(ctx, cfg.Inventario)
	}
	recuperacion, cancelRecuperacion := context.WithTimeout(context.WithoutCancel(ctx), time.Duration(cfg.Captura.TiempoRecuperacionSegundos)*time.Second)
	defer cancelRecuperacion()
	if control.ReabrirEscritores(recuperacion) != nil {
		e = fisica.ErrControl
	}
	if e != nil {
		informarError(diagnostico, e)
		return 1
	}
	if json.NewEncoder(out).Encode(struct {
		Estado     string             `json:"estado"`
		Directorio string             `json:"directorio_privado"`
		Artefactos []copias.Artefacto `json:"artefactos"`
	}{"pendiente_cifrado_y_verificacion", capturador.Directorio, artefactos}) != nil {
		fmt.Fprintln(diagnostico, "captura_fisica_salida")
		return 1
	}
	return 0
}
func leer(ruta string) (configuracion, error) {
	var cfg configuracion
	// Configuración contiene rutas privadas; nunca se imprimen errores de OS/JSON.
	f, e := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0) // #nosec G304 -- local operator chooses the private configuration file; no network input
	if e != nil {
		return cfg, fisica.ErrConfiguracion
	}
	defer f.Close()
	s, e := f.Stat()
	if e != nil || !s.Mode().IsRegular() || s.Size() > 2<<20 || s.Mode().Perm()&0077 != 0 {
		return cfg, fisica.ErrConfiguracion
	}
	d := json.NewDecoder(io.LimitReader(f, (2<<20)+1))
	d.DisallowUnknownFields()
	if e = d.Decode(&cfg); e != nil {
		return cfg, fisica.ErrConfiguracion
	}
	var extra any
	if e = d.Decode(&extra); e != io.EOF {
		return cfg, fisica.ErrConfiguracion
	}
	return cfg, nil
}

// La frontera CLI emite solo códigos conocidos, incluso si un adaptador devuelve
// un PathError de Sync/Close o un error unido con detalles privados.
func informarError(destino io.Writer, err error) {
	fmt.Fprintln(destino, codigoError(err))
}

func codigoError(err error) string {
	for _, caso := range []struct {
		err    error
		codigo string
	}{
		{fisica.ErrControl, "captura_fisica_control"},
		{fisica.ErrConfiguracion, "captura_fisica_configuracion"},
		{fisica.ErrTablespaces, "captura_fisica_tablespaces_no_soportados"},
		{fisica.ErrCambio, "captura_fisica_cambio"},
		{fisica.ErrLimite, "captura_fisica_limite"},
		{context.Canceled, "captura_fisica_cancelada"},
		{context.DeadlineExceeded, "captura_fisica_tiempo_agotado"},
	} {
		if errors.Is(err, caso.err) {
			return caso.codigo
		}
	}
	return "captura_fisica_origen"
}
