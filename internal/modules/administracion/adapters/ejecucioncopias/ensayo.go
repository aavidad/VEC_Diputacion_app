package ejecucioncopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"

	lector "vec-diputacion-granada/internal/modules/administracion/adapters/contrastecopias"
	fisica "vec-diputacion-granada/internal/modules/administracion/adapters/ensayofisicopg"
	logica "vec-diputacion-granada/internal/modules/administracion/adapters/ensayologicopg"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
)

type LectorComponentes interface {
	LeerComponente(context.Context, string, string) ([]byte, error)
}

// EnsayadorCS06 recupera únicamente payloads autenticados CS03 y usa los dos
// ejecutores ya existentes. Los paths son privados generados por este puente.
type EnsayadorCS06 struct {
	Fuente           LectorComponentes
	Fisico           fisica.Configuracion
	Logico           logica.Configuracion
	Lector           *lector.Lector
	Base             string
	Arrancador       ArrancadorConjunto
	Ficheros         InspectorFicheros
	RaizTemporal     string
	LimiteTotalBytes int64
}

func (e EnsayadorCS06) Ensayar(ctx context.Context, c p.Conjunto, modo p.ModoEnsayo) (resultado p.Ensayo, err error) {
	if ctx == nil || ctx.Err() != nil || e.Fuente == nil || e.Lector == nil || e.Arrancador == nil || e.Ficheros == nil || e.Base == "" || e.LimiteTotalBytes <= 0 || len(c.Manifiesto.Inventario.PostgreSQL.Bases) != 1 || (modo != p.Fisico && modo != p.Logico) {
		return resultado, ErrEnsayoEnsemble
	}
	if e.Fisico.ImagenSHA256 != c.Manifiesto.Inventario.PostgreSQL.RuntimeSHA256 || e.Logico.ImagenSHA256 != e.Fisico.ImagenSHA256 || e.Fisico.VersionPostgreSQL != c.Manifiesto.Inventario.PostgreSQL.Version || e.Logico.VersionPostgreSQL != e.Fisico.VersionPostgreSQL {
		return resultado, ErrEnsayoEnsemble
	}
	if !filepath.IsAbs(e.RaizTemporal) {
		return resultado, ErrEnsayoEnsemble
	}
	raiz, err := os.MkdirTemp(e.RaizTemporal, "ensayo-ensemble-")
	if err != nil {
		return resultado, ErrEnsayoEnsemble
	}
	defer func() {
		if er := os.RemoveAll(raiz); er != nil {
			err = ErrEnsayoEnsemble
		}
	}() // #nosec G703 -- generated private scratch root, never from backup paths.
	archivos := map[string]string{}
	var total int64
	for n, a := range c.Manifiesto.Componentes {
		if ctx.Err() != nil || a.TamanoBytes < 0 || a.TamanoBytes > e.LimiteTotalBytes-total {
			return resultado, ErrEnsayoEnsemble
		}
		total += a.TamanoBytes
		b, er := e.Fuente.LeerComponente(ctx, c.Ref, a.ID)
		if er != nil || !bytesArtefacto(b, a) {
			clear(b)
			return resultado, ErrEnsayoEnsemble
		}
		h := sha256.Sum256([]byte(a.ID))
		nombre := hex.EncodeToString(h[:]) + ".payload"
		path := filepath.Join(raiz, nombre)
		if er = os.WriteFile(path, b, 0600); er != nil {
			clear(b)
			return resultado, ErrEnsayoEnsemble
		}
		clear(b)
		if archivos[a.ID] != "" || n > 4096 {
			return resultado, ErrEnsayoEnsemble
		}
		archivos[a.ID] = path
	}
	var tar []fisica.Componente
	var dump, globals logica.Archivo
	for _, a := range c.Manifiesto.Componentes {
		switch {
		case strings.HasPrefix(a.ID, "fisica:"):
			tar = append(tar, fisica.Componente{ID: a.ID, Tipo: a.Tipo, Tar: fisica.Archivo{Ruta: archivos[a.ID], SHA256: a.SHA256}})
		case a.Tipo == "base_logica":
			if dump.Ruta != "" {
				return resultado, ErrEnsayoEnsemble
			}
			dump = logica.Archivo{Ruta: archivos[a.ID], SHA256: a.SHA256}
		case a.Tipo == "globals":
			if globals.Ruta != "" {
				return resultado, ErrEnsayoEnsemble
			}
			globals = logica.Archivo{Ruta: archivos[a.ID], SHA256: a.SHA256}
		}
	}
	sort.Slice(tar, func(i, j int) bool {
		if tar[i].ID == "fisica:pgdata" {
			return true
		}
		if tar[j].ID == "fisica:pgdata" {
			return false
		}
		return tar[i].ID < tar[j].ID
	})
	if len(tar) == 0 || tar[0].ID != "fisica:pgdata" || dump.Ruta == "" || globals.Ruta == "" {
		return resultado, ErrEnsayoEnsemble
	}
	observador := &ObservadorEnsemble{Lector: e.Lector, Base: e.Base, Conjunto: c, Modo: modo, Arrancador: e.Arrancador, Ficheros: e.Ficheros}
	if modo == p.Fisico {
		r := (fisica.Ensayador{Configuracion: e.Fisico, Observador: observador}).Ensayar(ctx, fisica.Solicitud{Sintetica: true, Componentes: tar})
		if r.Estado != "restauracion_fisica_completada" || !r.LimpiezaCompletada {
			return resultado, ErrEnsayoEnsemble
		}
	} else {
		r := (logica.Ensayador{Configuracion: e.Logico, Observador: observador, ComponentesArchivados: tar}).Ensayar(ctx, logica.Solicitud{Sintetica: true, Dump: dump, Globals: globals})
		if r.Estado != "restauracion_logica_completada" || !r.LimpiezaCompletada {
			return resultado, ErrEnsayoEnsemble
		}
	}
	return observador.resultado()
}

func mismoDatos(a, b copias.Evidencia) bool {
	return a.RecuentosSHA256 == b.RecuentosSHA256 && a.ContenidoSHA256 == b.ContenidoSHA256 && a.EsquemaSHA256 == b.EsquemaSHA256 && a.RolesSHA256 == b.RolesSHA256 && a.ACLSHA256 == b.ACLSHA256 && a.SecuenciasSHA256 == b.SecuenciasSHA256 && a.ObjetosGrandesSHA256 == b.ObjetosGrandesSHA256 && a.FicherosSHA256 == b.FicherosSHA256
}
