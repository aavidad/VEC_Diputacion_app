package ejecucioncopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

var ErrMaterial = errors.New("copias_ejecucion_material_no_comprobable")

type UbicacionMaterial struct {
	Raiz, Archivo string
	Artefacto     copias.Artefacto
}

// MaterialArchivos sólo entrega los ficheros privados de una captura terminada.
// Las ubicaciones proceden de CS04/CS05; nunca del manifiesto ni del navegador.
type MaterialArchivos struct {
	RaizLogica       string
	DirectorioFisico func() string
	Bases            []string
	LimiteBytes      int64
	mu               sync.Mutex
	conjuntos        map[string]map[string]UbicacionMaterial
}

type indiceFisico struct {
	FormatoVersion   int    `json:"formato_version"`
	Estado           string `json:"estado"`
	InventarioSHA256 string `json:"inventario_sha256"`
	Entradas         []struct {
		Archivo   string           `json:"archivo"`
		Artefacto copias.Artefacto `json:"artefacto"`
	} `json:"entradas"`
}

func (m *MaterialArchivos) Registrar(ctx context.Context, ref string, originales []copias.Artefacto, i copias.Inventario) ([]copias.Artefacto, error) {
	if m == nil || ctx == nil || ctx.Err() != nil || m.DirectorioFisico == nil || m.RaizLogica == "" || m.LimiteBytes <= 0 || len(originales) == 0 {
		return nil, ErrMaterial
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conjuntos == nil {
		m.conjuntos = make(map[string]map[string]UbicacionMaterial)
	}
	if m.conjuntos[ref] != nil {
		return nil, ErrMaterial
	}
	fisico := m.DirectorioFisico()
	raw, err := leerArchivoCapturado(fisico, "indice.json", 1<<20)
	if err != nil {
		return nil, ErrMaterial
	}
	var indice indiceFisico
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if dec.Decode(&indice) != nil || dec.Decode(new(any)) != io.EOF || indice.FormatoVersion != 1 || indice.Estado != "pendiente_cifrado_y_verificacion" || indice.InventarioSHA256 != copias.HuellaInventario(i) {
		return nil, ErrMaterial
	}
	ubicaciones := make(map[string]UbicacionMaterial, len(originales))
	for _, e := range indice.Entradas {
		if e.Artefacto.ID == "" || ubicaciones[e.Artefacto.ID].Archivo != "" {
			return nil, ErrMaterial
		}
		ubicaciones[e.Artefacto.ID] = UbicacionMaterial{fisico, e.Archivo, e.Artefacto}
	}
	salida := make([]copias.Artefacto, 0, len(originales))
	seen := map[string]bool{}
	var total int64
	for _, a := range originales {
		if seen[a.ID] || a.TamanoBytes < 0 || a.TamanoBytes > m.LimiteBytes-total {
			return nil, ErrMaterial
		}
		seen[a.ID] = true
		total += a.TamanoBytes
		u, ok := ubicaciones[a.ID]
		if !ok {
			switch a.Tipo {
			case "postgresql_logico":
				encontrado := -1
				for n, id := range m.Bases {
					if id == a.ID {
						encontrado = n
					}
				}
				if encontrado < 0 {
					return nil, ErrMaterial
				}
				u = UbicacionMaterial{m.RaizLogica, "base-" + strconv.Itoa(encontrado) + ".dump", a}
				a.Tipo = "base_logica"
			case "postgresql_globals":
				u = UbicacionMaterial{m.RaizLogica, "globals.sql", a}
				a.Tipo = "globals"
			default:
				return nil, ErrMaterial
			}
		} else if u.Artefacto != a {
			return nil, ErrMaterial
		}
		b, err := leerArchivoCapturado(u.Raiz, u.Archivo, min(m.LimiteBytes, a.TamanoBytes))
		if err != nil || !bytesArtefacto(b, a) {
			clear(b)
			return nil, ErrMaterial
		}
		clear(b)
		u.Artefacto = a
		ubicaciones[a.ID] = u
		salida = append(salida, a)
	}
	if len(ubicaciones) != len(salida) {
		return nil, ErrMaterial
	}
	m.conjuntos[ref] = ubicaciones
	return salida, nil
}
func bytesArtefacto(b []byte, a copias.Artefacto) bool {
	h := sha256.Sum256(b)
	return int64(len(b)) == a.TamanoBytes && hex.EncodeToString(h[:]) == a.SHA256
}
func (m *MaterialArchivos) Leer(ctx context.Context, ref string, a copias.Artefacto) ([]byte, error) {
	if m == nil || ctx == nil || ctx.Err() != nil {
		return nil, ErrMaterial
	}
	m.mu.Lock()
	u, ok := m.conjuntos[ref][a.ID]
	m.mu.Unlock()
	if !ok || u.Artefacto != a {
		return nil, ErrMaterial
	}
	b, err := leerArchivoCapturado(u.Raiz, u.Archivo, min(m.LimiteBytes, a.TamanoBytes))
	if err != nil || !bytesArtefacto(b, a) {
		clear(b)
		return nil, ErrMaterial
	}
	return b, nil
}
func leerArchivoCapturado(raiz, archivo string, limite int64) ([]byte, error) {
	if raiz == "" || archivo == "" || strings.ContainsAny(archivo, "/\\") || archivo == "." || archivo == ".." || limite < 0 {
		return nil, ErrMaterial
	}
	r, err := os.OpenRoot(raiz)
	if err != nil {
		return nil, ErrMaterial
	}
	defer r.Close()
	leaf, err := r.Lstat(archivo)
	if err != nil || !leaf.Mode().IsRegular() {
		return nil, ErrMaterial
	}
	f, err := r.OpenFile(archivo, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, ErrMaterial
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || !stat.Mode().IsRegular() || !os.SameFile(leaf, stat) || stat.Size() > limite {
		return nil, ErrMaterial
	}
	before, ok := stat.Sys().(*syscall.Stat_t)
	if !ok || before.Nlink != 1 {
		return nil, ErrMaterial
	}
	b, err := io.ReadAll(io.LimitReader(f, limite+1))
	if err != nil || int64(len(b)) > limite {
		clear(b)
		return nil, ErrMaterial
	}
	after, err := f.Stat()
	if err != nil || after.Size() != stat.Size() || !os.SameFile(stat, after) {
		clear(b)
		return nil, ErrMaterial
	}
	return b, nil
}
