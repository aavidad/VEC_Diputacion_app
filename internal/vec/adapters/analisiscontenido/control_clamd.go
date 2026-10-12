package analisiscontenido

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ConfiguracionClamdControlado solo acepta rutas preparadas por operación en
// directorios privados. FreshClam actualiza DBDir fuera de este proceso; el
// parser clamd se ejecuta sin red y lee esas firmas en modo solo lectura.
type ConfiguracionClamdControlado struct {
	Bwrap, ClamdBin, Prlimit            string
	RutaConfiguracion, DBDir, SocketDir string
	MaxEdadFirmas                       time.Duration
}

// ClamdControlado acredita que este servicio lanzó y mantiene vivo un daemon
// con la copia privada de la política comprobada. Un socket ajeno no crea esta
// capacidad y nunca puede producir `limpio`.
type ClamdControlado struct {
	socket, huellaPolitica string
	terminado              chan error
	cancelar               context.CancelFunc
	proceso                *exec.Cmd
	directorioPolitica     string
}

func (c *ClamdControlado) vivo() bool {
	if c == nil || c.terminado == nil || c.huellaPolitica == "" {
		return false
	}
	select {
	case <-c.terminado:
		return false
	default:
		return true
	}
}

func (c *ClamdControlado) HuellaPoliticaSHA256() string {
	if c == nil {
		return ""
	}
	return c.huellaPolitica
}

func (c *ClamdControlado) Socket() string {
	if c == nil {
		return ""
	}
	return c.socket
}

func (c *ClamdControlado) Cerrar() error {
	if c == nil {
		return ErrConfiguracion
	}
	if c.cancelar != nil {
		c.cancelar()
	}
	var err error
	if c.proceso != nil {
		select {
		case <-c.terminado:
		case <-time.After(5 * time.Second):
			if c.proceso.Process != nil {
				_ = c.proceso.Process.Kill()
			}
			select {
			case <-c.terminado:
			case <-time.After(2 * time.Second):
				err = ErrRemoto
			}
		}
	}
	if c.directorioPolitica != "" {
		err = errors.Join(err, os.RemoveAll(c.directorioPolitica))
	}
	if c.socket != "" {
		if e := os.Remove(c.socket); e != nil && !errors.Is(e, os.ErrNotExist) {
			err = errors.Join(err, e)
		}
	}
	return err
}

func IniciarClamdControlado(ctx context.Context, cfg ConfiguracionClamdControlado) (*ClamdControlado, error) {
	if ctx == nil || cfg.Bwrap == "" || cfg.ClamdBin == "" || cfg.Prlimit == "" ||
		!directorioClamdPrivado(cfg.DBDir) || !directorioClamdPrivado(cfg.SocketDir) ||
		cfg.DBDir == cfg.SocketDir || strings.HasPrefix(cfg.DBDir, cfg.SocketDir+string(os.PathSeparator)) || strings.HasPrefix(cfg.SocketDir, cfg.DBDir+string(os.PathSeparator)) ||
		cfg.RutaConfiguracion == "" || cfg.MaxEdadFirmas <= 0 || cfg.MaxEdadFirmas > 72*time.Hour {
		return nil, ErrConfiguracion
	}
	for _, ruta := range []string{cfg.Bwrap, cfg.ClamdBin, cfg.Prlimit} {
		info, err := os.Lstat(ruta)
		if err != nil || !filepath.IsAbs(ruta) || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
			return nil, ErrConfiguracion
		}
	}
	for _, dir := range []string{cfg.DBDir, cfg.SocketDir} {
		info, err := os.Lstat(dir)
		if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
			return nil, ErrConfiguracion
		}
	}
	contenido, err := leerArchivoTLS(cfg.RutaConfiguracion, true)
	if err != nil {
		return nil, ErrConfiguracion
	}
	socket := filepath.Join(cfg.SocketDir, "clamd.sock")
	if validarPoliticaClamd(contenido, cfg.DBDir, socket) != nil {
		return nil, ErrConfiguracion
	}
	if _, err = os.Lstat(socket); !errors.Is(err, os.ErrNotExist) {
		return nil, ErrConfiguracion
	}
	dir, err := os.MkdirTemp("/tmp", "vec-analizador-clamd-politica-")
	if err != nil {
		return nil, ErrConfiguracion
	}
	rutaCopia := filepath.Join(dir, "clamd.conf")
	if os.WriteFile(rutaCopia, contenido, 0400) != nil {
		_ = os.RemoveAll(dir)
		return nil, ErrConfiguracion
	}
	ctxProceso, cancelar := context.WithCancel(ctx)
	args := []string{
		"--unshare-net", "--unshare-pid", "--die-with-parent", "--clearenv", "--setenv", "PATH", "/usr/bin", "--setenv", "HOME", "/tmp",
		"--ro-bind", "/usr", "/usr", "--ro-bind", "/lib", "/lib", "--ro-bind", "/lib64", "/lib64",
		"--tmpfs", "/tmp", "--ro-bind", cfg.DBDir, cfg.DBDir, "--ro-bind", dir, dir, "--bind", cfg.SocketDir, cfg.SocketDir,
		"--dev", "/dev", "--proc", "/proc", "--chdir", "/tmp",
		"--", cfg.Prlimit, "--as=3221225472", "--fsize=67108864", "--nproc=64", "--", cfg.ClamdBin, "--foreground", "--config-file=" + rutaCopia,
	}
	// #nosec G204 -- ejecutables y configuración local validados; clamd vive en bwrap sin red, con DB de solo lectura.
	cmd := exec.CommandContext(ctxProceso, cfg.Bwrap, args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=/tmp"}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err = cmd.Start(); err != nil {
		cancelar()
		_ = os.RemoveAll(dir)
		return nil, ErrConfiguracion
	}
	control := &ClamdControlado{socket: socket, huellaPolitica: huella(contenido), terminado: make(chan error, 1), cancelar: cancelar, proceso: cmd, directorioPolitica: dir}
	go func() { control.terminado <- cmd.Wait(); close(control.terminado) }()
	limite := time.NewTimer(90 * time.Second)
	defer limite.Stop()
	paso := time.NewTicker(100 * time.Millisecond)
	defer paso.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = control.Cerrar()
			return nil, ErrRemoto
		case <-limite.C:
			_ = control.Cerrar()
			return nil, ErrRemoto
		case <-control.terminado:
			_ = control.Cerrar()
			return nil, ErrRemoto
		case <-paso.C:
			if !control.vivo() {
				_ = control.Cerrar()
				return nil, ErrRemoto
			}
			if _, err := versionClamd(ctx, socket, time.Now().UTC(), cfg.MaxEdadFirmas); err == nil {
				return control, nil
			}
		}
	}
}

func directorioClamdPrivado(ruta string) bool {
	if !filepath.IsAbs(ruta) || ruta != filepath.Clean(ruta) || !strings.HasPrefix(ruta, "/tmp/vec-analizador-clamd-") {
		return false
	}
	real, err := filepath.EvalSymlinks(ruta)
	return err == nil && real == ruta
}

func validarPoliticaClamd(datos []byte, dbDir, socket string) error {
	if len(datos) == 0 || len(datos) > 1<<20 {
		return ErrConfiguracion
	}
	valores := make(map[string]string)
	for _, linea := range strings.Split(string(datos), "\n") {
		linea = strings.TrimSpace(linea)
		if linea == "" || strings.HasPrefix(linea, "#") {
			continue
		}
		partes := strings.Fields(linea)
		if len(partes) != 2 || strings.Contains(partes[1], "#") {
			return ErrConfiguracion
		}
		clave := partes[0]
		if _, repetida := valores[clave]; repetida {
			return ErrConfiguracion
		}
		if strings.HasPrefix(clave, "Include") || strings.HasPrefix(clave, "TCP") || clave == "User" || clave == "LogFile" || clave == "PidFile" {
			return ErrConfiguracion
		}
		valores[clave] = partes[1]
	}
	if valores["AlertExceedsMax"] != "yes" || valores["LocalSocket"] != socket || valores["DatabaseDirectory"] != dbDir ||
		valores["LocalSocketMode"] != "600" || valores["ScanPDF"] != "yes" {
		return ErrConfiguracion
	}
	for _, clave := range []string{"StreamMaxLength", "MaxFileSize", "MaxScanSize"} {
		n, err := bytesClamd(valores[clave])
		if err != nil || n < tamanoMaximoPDF || n > 1<<30 {
			return ErrConfiguracion
		}
	}
	return nil
}

func bytesClamd(valor string) (int64, error) {
	if valor == "" {
		return 0, ErrConfiguracion
	}
	factor := int64(1)
	switch valor[len(valor)-1] {
	case 'K':
		factor = 1 << 10
		valor = valor[:len(valor)-1]
	case 'M':
		factor = 1 << 20
		valor = valor[:len(valor)-1]
	case 'G':
		factor = 1 << 30
		valor = valor[:len(valor)-1]
	}
	n, err := strconv.ParseInt(valor, 10, 64)
	if err != nil || n < 1 || n > (1<<30)/factor {
		return 0, ErrConfiguracion
	}
	return n * factor, nil
}
