package ensayofisicopg

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

var errRuntime = errors.New("ensayo_runtime_no_comprobable")
var argumentoEnv = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,95}$`)

// RuntimeObservacion sólo recibe nombres generados por el ensayador y una raíz
// privada. Se exporta para reutilizar el mismo hook en el restaurador lógico.
type RuntimeObservacion struct {
	fisico                      *anclajeFisicoRuntime
	Nombre                      string
	Raiz                        string
	ImagenSHA256                string
	UsuarioBootstrap            string
	Componentes                 []puertos.Componente
	ConfiguracionOrigenExcluida bool
}

func (r RuntimeObservacion) EjecutarPostgreSQL(ctx context.Context, herramienta string, args []string, entrada []byte, limite int) ([]byte, error) {
	if ctx == nil || limite < 1 || limite > 64<<20 || len(entrada) > 8<<20 || len(args) > 128 {
		return nil, errRuntime
	}
	switch herramienta {
	case "psql", "pg_dump", "pg_dumpall":
	default:
		return nil, errRuntime
	}
	for _, a := range args {
		if len(a) > 8<<20 || strings.ContainsRune(a, 0) {
			return nil, errRuntime
		}
	}
	if _, err := r.ComprobarExclusion(ctx); err != nil {
		return nil, err
	}
	// El único servidor alcanzable usa el socket del ensayo. Ni entorno del host
	// ni perfiles psql ni contraseñas heredadas forman parte de este canal.
	cmd := []string{"exec", "--interactive", r.Nombre, "env", "-i", "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "PGHOST=/var/run/postgresql", "PGOPTIONS=-c default_transaction_read_only=on", herramienta}
	return docker(ctx, bytes.NewReader(entrada), limite, append(cmd, args...)...)
}
func (r RuntimeObservacion) EjecutarArchivado(ctx context.Context, id string, args []string, env map[string]string, limite int) ([]byte, error) {
	desbloquear := r.faseArchivada()
	defer desbloquear()
	if ctx == nil || limite < 1 || limite > 1<<20 || len(args) > 64 || len(env) > 128 {
		return nil, errRuntime
	}
	var binario string
	for _, c := range r.Componentes {
		if c.ID == id && c.Tipo == "binario" {
			binario = c.RutaInterna
			break
		}
	}
	if binario == "" {
		return nil, errRuntime
	}
	if _, err := r.ComprobarExclusion(ctx); err != nil {
		return nil, err
	}
	cmd := []string{"exec", "--interactive", r.Nombre, "env", "-i", "PATH=/usr/bin:/bin", "HOME=/tmp", "TMPDIR=/tmp", "LANG=C", "LC_ALL=C", "PGHOST=/var/run/postgresql"}
	for k, v := range env {
		if !argumentoEnv.MatchString(k) || (!strings.HasPrefix(k, "VEC_") && k != "PGUSER" && k != "PGDATABASE" && k != "PGPASSWORD") || len(v) > 16384 || strings.ContainsRune(v, 0) {
			return nil, errRuntime
		}
		cmd = append(cmd, k+"="+v)
	}
	for _, a := range args {
		if len(a) > 16384 || strings.ContainsRune(a, 0) {
			return nil, errRuntime
		}
	}
	cmd = append(cmd, binario)
	cmd = append(cmd, args...)
	// El observador conoce la configuración de esa release y verifica por separado
	// despacho bloqueado/salud/consulta autorizada. No hay valor mágico universal.
	return docker(ctx, nil, limite, cmd...)
}

type inspeccion struct {
	ID         string                 `json:"Id"`
	Config     struct{ Image string } `json:"Config"`
	HostConfig struct {
		NetworkMode    string
		ReadonlyRootfs bool
		Privileged     bool
		CapDrop        []string
		SecurityOpt    []string
	} `json:"HostConfig"`
	Mounts []struct {
		Type        string
		Source      string
		Destination string
		RW          bool
	} `json:"Mounts"`
	State struct{ Running bool } `json:"State"`
}

func (r RuntimeObservacion) ComprobarExclusion(ctx context.Context) (string, error) {
	b, err := docker(ctx, nil, 32768, "inspect", r.Nombre)
	if err != nil {
		return "", errRuntime
	}
	var l []inspeccion
	if json.Unmarshal(b, &l) != nil || len(l) != 1 {
		return "", errRuntime
	}
	i := l[0]
	if !i.State.Running || i.ID == "" || i.Config.Image != "sha256:"+r.ImagenSHA256 || i.HostConfig.NetworkMode != "none" || !i.HostConfig.ReadonlyRootfs || i.HostConfig.Privileged {
		return "", errRuntime
	}
	if !contiene(i.HostConfig.CapDrop, "ALL") || !sinNuevosPrivilegios(i.HostConfig.SecurityOpt) {
		return "", errRuntime
	}
	raiz := filepath.Clean(r.Raiz)
	if !raizTemporalAdmitida(raiz) {
		return "", errRuntime
	}
	for _, m := range i.Mounts {
		switch m.Type {
		case "bind":
			if !strings.HasPrefix(filepath.Clean(m.Source), raiz+"/") {
				return "", errRuntime
			}
		case "tmpfs":
			switch m.Destination {
			case "/tmp", "/var/run/postgresql", "/var/lib/postgresql":
			default:
				return "", errRuntime
			}
		default:
			return "", errRuntime
		}
	}
	if !usuarioValido.MatchString(r.UsuarioBootstrap) {
		return "", errRuntime
	}
	sonda, err := docker(ctx, nil, 4096, "exec", r.Nombre, "psql", "-X", "-At", "-U", r.UsuarioBootstrap, "-d", "postgres", "-c", "SELECT current_setting('listen_addresses'),current_setting('archive_mode'),current_setting('default_transaction_read_only')")
	if err != nil || strings.TrimSpace(string(sonda)) != "|off|on" {
		return "", errRuntime
	}
	// El sello liga el contenedor real, imagen y montaje propios. Releerlo detecta
	// sustitución o desaparición antes/después de cada observación.
	sort.Strings(i.HostConfig.CapDrop)
	sort.Strings(i.HostConfig.SecurityOpt)
	sort.Slice(i.Mounts, func(a, b int) bool { return i.Mounts[a].Destination < i.Mounts[b].Destination })
	estable, err := json.Marshal(i)
	if err != nil {
		return "", errRuntime
	}
	h := sha256.Sum256(estable)
	return hex.EncodeToString(h[:]), nil
}
func contiene(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}
func (r RuntimeObservacion) Entorno(ctx context.Context) (puertos.Entorno, error) {
	if _, err := r.ComprobarExclusion(ctx); err != nil {
		return puertos.Entorno{}, err
	}
	return puertos.Entorno{PostgreSQL: r, Archivado: r, Componentes: r.Componentes, Aislamiento: puertos.EvidenciaAislamiento{ImagenSHA256: r.ImagenSHA256, RedSinSalida: true, VolumenesPropios: true, ConfiguracionOrigenExcluida: r.ConfiguracionOrigenExcluida, RuntimeInspeccionado: true}}, nil
}

// LeerArchivado lee sólo el componente por ID resuelto en el inventario interno.
// La extracción rechazó enlaces y cada componente está montado de sólo lectura.
func (r RuntimeObservacion) LeerArchivado(ctx context.Context, id string, limite int) ([]byte, error) {
	if ctx == nil || limite < 1 || limite > 64<<20 {
		return nil, errRuntime
	}
	var ruta string
	for _, c := range r.Componentes {
		if c.ID == id {
			ruta = c.RutaInterna
			break
		}
	}
	if ruta == "" {
		return nil, errRuntime
	}
	if _, err := r.ComprobarExclusion(ctx); err != nil {
		return nil, err
	}
	return docker(ctx, nil, limite, "exec", r.Nombre, "cat", "--", ruta)
}

func sinNuevosPrivilegios(opciones []string) bool {
	admitida := false
	for _, s := range opciones {
		if s == "no-new-privileges:true" || s == "no-new-privileges" {
			admitida = true
		} else if strings.HasPrefix(s, "no-new-privileges") {
			return false
		}
	}
	return admitida
}

// escritorPlantillas excluye el comienzo de procesos archivados mientras se
// modifica el clon técnico. Tras cualquier arranque no se presume que el
// binario coopere: el ensayo sólo puede limpiar su contenedor completo.
func (r RuntimeObservacion) escritorPlantillas() (func(), error) {
	if r.fisico == nil {
		return nil, falloPlantilla("fase_fisica_ausente")
	}
	r.fisico.fase.Lock()
	if r.fisico.archivadosIniciados {
		r.fisico.fase.Unlock()
		return nil, falloPlantilla("fase_archivada_iniciada")
	}
	return r.fisico.fase.Unlock, nil
}
func (r RuntimeObservacion) faseArchivada() func() {
	if r.fisico == nil {
		return func() {}
	}
	r.fisico.fase.Lock()
	r.fisico.archivadosIniciados = true
	return r.fisico.fase.Unlock
}
