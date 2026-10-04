package ejecucioncopias

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func dentroCS11(raiz, ruta string) bool {
	rel, err := filepath.Rel(raiz, ruta)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
func rutaScratchCS11(path string) bool {
	return strings.HasPrefix(path, "/dev/shm/vec-cs11-") || strings.HasPrefix(path, "/var/tmp/vec-cs11-")
}
func directorioPrivadoCS11(ruta string) bool {
	if !rutaSinEnlacesCS11(ruta) {
		return false
	}
	i, err := os.Stat(ruta)
	if err != nil || !i.IsDir() || i.Mode().Perm()&0077 != 0 {
		return false
	}
	s, ok := i.Sys().(*syscall.Stat_t)
	return ok && s.Uid == uint32(os.Geteuid()) // #nosec G115 -- UID Linux corresponde a uid_t de 32 bits.
}
func rutaSinEnlacesCS11(ruta string) bool {
	if !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return false
	}
	for p := ruta; ; p = filepath.Dir(p) {
		i, err := os.Lstat(p)
		if err != nil || i.Mode()&os.ModeSymlink != 0 {
			return false
		}
		if p == filepath.Dir(p) {
			return true
		}
	}
}
func rutaHerramientaCS11(p string) bool {
	return strings.HasPrefix(p, "/usr/") && filepath.Clean(p) == p && !strings.ContainsAny(p, "\x00\r\n")
}

func dockerCS11(ctx context.Context, in io.Reader, max int, args ...string) ([]byte, error) {
	if ctx == nil || max < 1 || max > 64<<20 || len(args) == 0 {
		return nil, ErrOrigenLocal
	}
	limitado, cancelar := context.WithTimeout(ctx, 2*time.Minute)
	defer cancelar()
	cmd := exec.CommandContext(limitado, "/usr/bin/docker", append([]string{"--host", "unix:///var/run/docker.sock"}, args...)...) // #nosec G204 G702 -- binario y socket fijos; argumentos cerrados del adaptador.
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=/tmp", "LANG=C", "LC_ALL=C", "DOCKER_CONFIG=/dev/null"}
	cmd.Stdin, cmd.Stderr = in, io.Discard
	var out limiteDockerCS11
	out.max = max
	cmd.Stdout = &out
	if cmd.Run() != nil || out.exceso {
		return nil, ErrOrigenLocal
	}
	return out.Bytes(), nil
}

type limiteDockerCS11 struct {
	bytes.Buffer
	max    int
	exceso bool
}

func (l *limiteDockerCS11) Write(b []byte) (int, error) {
	if len(b) > l.max-l.Len() {
		l.exceso = true
		return len(b), nil
	}
	return l.Buffer.Write(b)
}

type estadoDockerCS11 struct {
	ID         string
	Config     struct{ Image string }
	HostConfig struct {
		NetworkMode    string
		Privileged     bool
		ReadonlyRootfs bool
		CapDrop        []string
		SecurityOpt    []string
	}
	Mounts []struct {
		Type, Source, Destination string
		RW                        bool
	}
	State struct{ Running bool }
}

func (o *OrigenLocal) observarContenedor(ctx context.Context, enMarcha bool) (string, error) {
	if o == nil {
		return "", ErrOrigenLocal
	}
	b, err := dockerCS11(ctx, nil, 1<<16, "inspect", o.c.Contenedor)
	if err != nil {
		return "", err
	}
	var estados []estadoDockerCS11
	if json.Unmarshal(b, &estados) != nil || len(estados) != 1 {
		return "", ErrOrigenLocal
	}
	e := estados[0]
	if e.ID == "" || e.Config.Image != "sha256:"+o.c.ImagenSHA256 || e.HostConfig.NetworkMode != "none" || e.HostConfig.Privileged || !e.HostConfig.ReadonlyRootfs || e.State.Running != enMarcha || !contieneCS11(e.HostConfig.CapDrop, "ALL") || !(contieneCS11(e.HostConfig.SecurityOpt, "no-new-privileges") || contieneCS11(e.HostConfig.SecurityOpt, "no-new-privileges:true")) {
		return "", ErrOrigenLocal
	}
	pgMontado := false
	for _, m := range e.Mounts {
		switch m.Type {
		case "bind":
			if !dentroCS11(o.c.RaizPropia, m.Source) {
				return "", ErrOrigenLocal
			}
			if m.Source == filepath.Dir(o.c.PGDATA) && m.Destination == "/var/lib/postgresql" && m.RW {
				pgMontado = true
			}
		case "tmpfs":
			if m.Destination != "/tmp" && m.Destination != "/var/run/postgresql" {
				return "", ErrOrigenLocal
			}
		default:
			return "", ErrOrigenLocal
		}
	}
	if !pgMontado {
		return "", ErrOrigenLocal
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func contieneCS11(v []string, x string) bool {
	for _, s := range v {
		if s == x {
			return true
		}
	}
	return false
}
