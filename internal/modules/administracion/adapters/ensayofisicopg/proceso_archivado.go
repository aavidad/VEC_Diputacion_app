package ensayofisicopg

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

func prepararAuxiliar(raiz string) error {
	if prepararNSS(raiz) != nil {
		return errRuntime
	}
	ejecutable, err := os.Executable()
	if err != nil {
		return errRuntime
	}
	info, err := os.Stat(ejecutable)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 64<<20 {
		return errRuntime
	}
	b, err := os.ReadFile(ejecutable) // #nosec G304 -- ruta obtenida de os.Executable, no seleccionable por entradas; auxiliar de la propia CLI.
	if err != nil {
		return errRuntime
	}
	root, err := os.OpenRoot(raiz)
	if err != nil {
		return errRuntime
	}
	defer root.Close()
	if root.WriteFile("verificador", b, 0500) != nil || root.Mkdir("control", 0700) != nil { // #nosec G306 -- executable privado sólo del UID propio, requiere bit de ejecución; sin escritura de grupo/otros.
		return errRuntime
	}
	return nil
}
func (r RuntimeObservacion) IniciarArchivado(ctx context.Context, id string, args []string, env map[string]string) (puertos.ProcesoArchivado, error) {
	desbloquear := r.faseArchivada()
	defer desbloquear()
	if ctx == nil || len(args) > 64 || len(env) > 128 {
		return nil, errRuntime
	}
	if _, err := r.ComprobarExclusion(ctx); err != nil {
		return nil, err
	}
	var binario string
	for _, c := range r.Componentes {
		if c.ID == id && c.Tipo == "binario" {
			binario = c.RutaInterna
		}
	}
	if binario == "" {
		return nil, errRuntime
	}
	entorno := []string{"PATH=/usr/bin:/bin", "HOME=/tmp", "TMPDIR=/tmp", "LANG=C", "LC_ALL=C", "PGHOST=/var/run/postgresql"}
	for k, v := range env {
		if !argumentoEnv.MatchString(k) || (!strings.HasPrefix(k, "VEC_") && k != "PGUSER" && k != "PGDATABASE" && k != "PGPASSWORD") || len(v) > 16384 || strings.ContainsRune(v, 0) {
			return nil, errRuntime
		}
		entorno = append(entorno, k+"="+v)
	}
	for _, a := range args {
		if len(a) > 16384 || strings.ContainsRune(a, 0) {
			return nil, errRuntime
		}
	}
	// Un proceso archivado por runtime: el archivo O_EXCL impide arrancar otro
	// para ocultar el fallo del anterior o sobrescribir el vínculo de PID.
	b, err := json.Marshal(inicioArchivado{binario, args, entorno})
	if err != nil {
		return nil, errRuntime
	}
	f, err := os.OpenFile(filepath.Join(r.Raiz, "control", "inicio.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, errRuntime
	}
	_, escribir := f.Write(b)
	cerrar := f.Close()
	if escribir != nil || cerrar != nil {
		return nil, errRuntime
	}
	if _, err = docker(ctx, nil, 4096, "exec", "--detach", r.Nombre, "env", "-i", "VEC_CS06_INTERNO=1", "/verificador", modoInicio); err != nil {
		return nil, errRuntime
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(filepath.Join(r.Raiz, "control", "proceso.pid")); err == nil {
			return procesoArchivado{r}, nil
		}
		select {
		case <-ctx.Done():
			return nil, errRuntime
		case <-ticker.C:
		}
	}
}

type procesoArchivado struct{ runtime RuntimeObservacion }

func (p procesoArchivado) SondarHTTP(ctx context.Context, s puertos.SondaHTTP) (puertos.RespuestaHTTP, error) {
	r := puertos.RespuestaHTTP{}
	if _, err := p.runtime.ComprobarExclusion(ctx); err != nil {
		return r, err
	}
	if _, err := os.Stat(filepath.Join(p.runtime.Raiz, "control", "proceso.terminado")); err == nil {
		return r, errRuntime
	}
	b, err := json.Marshal(s)
	if err != nil {
		return r, errRuntime
	}
	b, err = docker(ctx, bytes.NewReader(b), 32768, "exec", "--interactive", p.runtime.Nombre, "env", "-i", "VEC_CS06_INTERNO=1", "/verificador", modoSonda)
	if err != nil || json.Unmarshal(b, &r) != nil {
		return puertos.RespuestaHTTP{}, errRuntime
	}
	return r, nil
}
func (p procesoArchivado) Detener(ctx context.Context) error {
	if _, err := p.runtime.ComprobarExclusion(ctx); err != nil {
		return err
	}
	if _, err := docker(ctx, nil, 4096, "exec", p.runtime.Nombre, "env", "-i", "VEC_CS06_INTERNO=1", "/verificador", modoDetener); err != nil {
		return errRuntime
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(filepath.Join(p.runtime.Raiz, "control", "proceso.terminado")); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return errRuntime
		case <-ticker.C:
		}
	}
}
