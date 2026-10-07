// Package releasebinario inspecciona bytes y metadatos públicos Go sin ejecutar
// el archivo. Los metadatos VCS son declaraciones del binario, no atestaciones.
package releasebinario

import (
	"bytes"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

// MaxSnapshotBytes es un techo técnico de memoria, no una política de release.
// El operador debe elegir además un límite explícito dentro de este rango.
const MaxSnapshotBytes int64 = 256 << 20

type Observado struct {
	SHA256      string `json:"sha256"`
	TamanoBytes int64  `json:"tamano_bytes"`
	GoVersion   string `json:"go_version"`
	GOOS        string `json:"goos"`
	GOARCH      string `json:"goarch"`
	VCS         string `json:"vcs"`
	RevisionVCS string `json:"revision_vcs"`
	FechaVCS    string `json:"fecha_vcs"`
	Modificado  *bool  `json:"modificado"`
}

type Informe struct {
	Estado               string         `json:"estado"`
	Autenticidad         string         `json:"autenticidad"`
	CommitProcedencia    string         `json:"commit_procedencia,omitempty"`
	HabilitaCopia        bool           `json:"habilita_copia"`
	HabilitaRestauracion bool           `json:"habilita_restauracion"`
	Observado            Observado      `json:"observado"`
	Razones              []copias.Razon `json:"razones"`
}

// Inspeccionar necesita un límite de lectura elegido explícitamente por el
// operador. Hash y BuildInfo proceden del mismo buffer acotado de bytes.
func Inspeccionar(raiz, ruta string, maxBytes int64) Informe {
	informe := Informe{Estado: "datos_observados", Autenticidad: "no_comprobada", Razones: []copias.Razon{}}
	if maxBytes <= 0 || maxBytes > MaxSnapshotBytes {
		fallo(&informe, "limite_no_valido", "max_bytes", "1.."+strconv.FormatInt(MaxSnapshotBytes, 10), strconv.FormatInt(maxBytes, 10), "fijar_limite_operador")
		return informe
	}
	if ruta == "." || !fs.ValidPath(ruta) || strings.ContainsAny(ruta, "\\\x00") {
		fallo(&informe, "ruta_no_valida", "ruta", "relativa_a_raiz", "no_valida", "elegir_binario_autorizado")
		return informe
	}
	r, err := os.OpenRoot(raiz)
	if err != nil {
		return falloArchivo(&informe)
	}
	defer r.Close()
	f, err := r.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return falloArchivo(&informe)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return falloArchivo(&informe)
	}
	if info.Size() > maxBytes {
		fallo(&informe, "limite_superado", "tamano_bytes", strconv.FormatInt(maxBytes, 10), strconv.FormatInt(info.Size(), 10), "revisar_limite_operador")
		return informe
	}
	datos, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil || int64(len(datos)) > maxBytes || int64(len(datos)) != info.Size() {
		return fallo(&informe, "lectura_no_comprobable", "bytes", "lectura_completa_acotada", "no_completa", "capturar_binario_estable")
	}
	h := sha256.Sum256(datos)
	informe.Observado.SHA256 = hex.EncodeToString(h[:])
	informe.Observado.TamanoBytes = int64(len(datos))
	bi, err := buildinfo.Read(bytes.NewReader(datos))
	if err != nil {
		return fallo(&informe, "metadatos_go_ausentes", "build_info", "metadatos_go", "ausentes_o_no_validos", "obtener_descriptor_autenticado")
	}
	extraer(&informe, bi)
	return informe
}

var (
	versionRE    = regexp.MustCompile(`^go[0-9]+(\.[0-9]+){1,2}$`)
	plataformaRE = regexp.MustCompile(`^[a-z0-9_]{1,24}$`)
	revisionRE   = regexp.MustCompile(`^([0-9a-f]{40}|[0-9a-f]{64})$`)
)

func extraer(informe *Informe, bi *debug.BuildInfo) {
	if versionRE.MatchString(bi.GoVersion) {
		informe.Observado.GoVersion = bi.GoVersion
	} else {
		falloMetadato(informe, "go_version")
	}
	valores := map[string]string{}
	for _, dato := range bi.Settings {
		switch dato.Key {
		case "GOOS", "GOARCH", "vcs", "vcs.revision", "vcs.time", "vcs.modified":
			if _, existe := valores[dato.Key]; existe {
				fallo(informe, "metadato_ambiguo", dato.Key, "una_declaracion", "repetida", "obtener_descriptor_autenticado")
				return
			}
			valores[dato.Key] = dato.Value
		}
	}
	if valores["vcs"] == "git" {
		informe.Observado.VCS = "git"
	} else {
		fallo(informe, "vcs_no_admitido", "vcs", "git", "ausente_o_no_git", "obtener_descriptor_autenticado")
	}
	for _, dato := range []struct {
		campo   string
		destino *string
		patron  *regexp.Regexp
	}{
		{"GOOS", &informe.Observado.GOOS, plataformaRE}, {"GOARCH", &informe.Observado.GOARCH, plataformaRE},
	} {
		if dato.patron.MatchString(valores[dato.campo]) {
			*dato.destino = valores[dato.campo]
		} else {
			falloMetadato(informe, dato.campo)
		}
	}
	if informe.Observado.VCS == "git" && revisionRE.MatchString(valores["vcs.revision"]) {
		informe.Observado.RevisionVCS = valores["vcs.revision"]
		informe.CommitProcedencia = "commit_declarado_en_binario"
	} else {
		falloMetadato(informe, "vcs.revision")
	}
	fecha, err := time.Parse(time.RFC3339Nano, valores["vcs.time"])
	if err == nil {
		informe.Observado.FechaVCS = fecha.UTC().Format(time.RFC3339Nano)
	} else {
		falloMetadato(informe, "vcs.time")
	}
	modificado := valores["vcs.modified"]
	if modificado == "true" || modificado == "false" {
		valor := modificado == "true"
		informe.Observado.Modificado = &valor
		if valor {
			fallo(informe, "fuentes_modificadas", "vcs.modified", "false", "true", "obtener_release_aprobada")
		}
	} else {
		falloMetadato(informe, "vcs.modified")
	}
}

func falloMetadato(informe *Informe, campo string) {
	fallo(informe, "metadato_no_comprobable", campo, "declaracion_valida", "ausente_o_invalida", "obtener_descriptor_autenticado")
}

func falloArchivo(informe *Informe) Informe {
	return fallo(informe, "binario_no_legible", "binario", "regular_legible_en_raiz", "no_legible", "elegir_binario_autorizado")
}

func fallo(informe *Informe, codigo, campo, esperado, obtenido, accion string) Informe {
	informe.Estado = "no_comprobable"
	informe.Razones = append(informe.Razones, copias.Razon{Codigo: codigo, Clave: campo, Esperado: esperado, Obtenido: obtenido, Accion: accion})
	return *informe
}
