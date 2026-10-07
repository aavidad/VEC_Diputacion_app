// Package capturafisica captura bytes locales bajo una ventana externa observada.
// No cifra, autentica ni acredita una restauración.
package capturafisica

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	control "vec-diputacion-granada/internal/modules/administracion/ports/capturafisica"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

var (
	ErrConfiguracion = errors.New("captura_fisica_configuracion")
	ErrOrigen        = errors.New("captura_fisica_origen")
	ErrCambio        = errors.New("captura_fisica_cambio")
	ErrLimite        = errors.New("captura_fisica_limite")
	ErrTablespaces   = errors.New("captura_fisica_tablespaces_no_soportados")
	ErrControl       = errors.New("captura_fisica_control")
)

type Fuente struct {
	ID   string `json:"id"`
	Tipo string `json:"tipo"`
	Ruta string `json:"ruta"`
}
type Configuracion struct {
	PGDATA                     string   `json:"pgdata"`
	Destino                    string   `json:"destino"`
	Fuentes                    []Fuente `json:"fuentes"`
	MaxBytes                   int64    `json:"max_bytes"`
	MaxEntradas                int      `json:"max_entradas"`
	TiempoRecuperacionSegundos int      `json:"tiempo_recuperacion_segundos"`
}

type Capturador struct {
	Config  Configuracion
	Control control.Control
	// Directorio contiene solo la captura propia terminada; queda vacío en error.
	Directorio string
}

// Capturar satisface el puerto opcional CS04. No libera su exclusión. Reanuda PG
// antes de retornar para permitir la observación final del inventario en CS04.
func (c *Capturador) Capturar(ctx context.Context, inv copias.Inventario) (artefactos []copias.Artefacto, err error) {
	c.Directorio = ""
	if c.Control == nil || c.validar(inv) != nil {
		return nil, ErrConfiguracion
	}
	if c.Control.ComprobarExclusion(ctx) != nil {
		return nil, ErrControl
	}
	// El intento de parada puede haber surtido efecto aunque falle su respuesta.
	recuperar := true
	defer func() {
		if !recuperar {
			return
		}
		recuperacion, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(time.Duration(c.Config.TiempoRecuperacionSegundos)*time.Second))
		defer cancelar()
		if c.Control.ReanudarPostgreSQL(recuperacion) != nil {
			err = errors.Join(err, ErrControl)
			artefactos = nil
			c.Directorio = ""
		}
	}()
	if c.Control.DetenerPostgreSQL(ctx) != nil || c.Control.ComprobarFrio(ctx) != nil {
		return nil, ErrControl
	}
	if c.Control.ComprobarExclusion(ctx) != nil {
		return nil, ErrControl
	}
	pg, e := os.OpenRoot(c.Config.PGDATA)
	if e != nil {
		return nil, ErrOrigen
	}
	defer pg.Close()
	// V1 no omite almacenamiento externo: cualquier tablespace bloquea hasta que
	// el capturador/restaurador acuerden su mapeo. No sigue enlaces pg_wal externos.
	tsp, e := pg.Open("pg_tblspc")
	if e != nil {
		return nil, ErrOrigen
	}
	entradas, e := tsp.ReadDir(1)
	cierre := tsp.Close()
	if len(entradas) != 0 || (e != nil && !errors.Is(e, io.EOF)) || cierre != nil {
		return nil, ErrTablespaces
	}
	destino, e := os.OpenRoot(c.Config.Destino)
	if e != nil {
		return nil, ErrOrigen
	}
	defer destino.Close()
	var nonce [16]byte
	if _, e = rand.Read(nonce[:]); e != nil {
		return nil, ErrOrigen
	}
	nombre := "captura-" + hex.EncodeToString(nonce[:])
	if e = destino.Mkdir(nombre, 0700); e != nil {
		return nil, ErrOrigen
	}
	completo := false
	defer func() {
		if !completo {
			_ = destino.RemoveAll(nombre)
		}
	}()
	salida, e := destino.OpenRoot(nombre)
	if e != nil {
		return nil, ErrOrigen
	}
	defer salida.Close()
	fuentes := append([]Fuente{{ID: "pgdata", Tipo: "base_fisica", Ruta: c.Config.PGDATA}}, c.Config.Fuentes...)
	indice := Indice{FormatoVersion: 1, Estado: "pendiente_cifrado_y_verificacion", InventarioSHA256: copias.HuellaInventario(inv)}
	limite := presupuesto{bytes: c.Config.MaxBytes, entradas: c.Config.MaxEntradas}
	for i, fuente := range fuentes {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		archivo := fmt.Sprintf("componente-%04d.tar", i)
		a, e := capturarFuente(ctx, fuente, salida, archivo, &limite, esperado(inv, fuente.ID))
		if e != nil {
			return nil, e
		}
		artefactos = append(artefactos, a)
		indice.Entradas = append(indice.Entradas, EntradaIndice{Archivo: archivo, Artefacto: a})
		if raw := esperado(inv, fuente.ID); raw != nil {
			nombreRaw := fmt.Sprintf("componente-%04d.bin", i)
			a, e = copiarRaw(ctx, salida, archivo, nombreRaw, *raw, &limite)
			if e != nil {
				return nil, e
			}
			artefactos = append(artefactos, a)
			indice.Entradas = append(indice.Entradas, EntradaIndice{Archivo: nombreRaw, Artefacto: a})
		}
	}
	if c.Control.ComprobarFrio(ctx) != nil || c.Control.ComprobarExclusion(ctx) != nil {
		return nil, ErrControl
	}
	idx, e := salida.OpenFile("indice.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return nil, ErrOrigen
	}
	e = json.NewEncoder(idx).Encode(indice)
	if x := idx.Sync(); e == nil {
		e = x
	}
	if x := idx.Close(); e == nil {
		e = x
	}
	if e != nil {
		return nil, ErrOrigen
	}
	// Reanudar antes de marcar completa evita dejar archivos aparentemente
	// terminados si la recuperación de la plataforma falla.
	if c.Control.ReanudarPostgreSQL(ctx) != nil {
		return nil, ErrControl
	}
	recuperar = false
	completo = true
	c.Directorio = filepath.Join(c.Config.Destino, nombre)
	return artefactos, nil
}

func esperado(inv copias.Inventario, id string) *copias.Artefacto {
	for _, a := range append(append([]copias.Artefacto{}, inv.Release.Binarios...), inv.Release.Componentes...) {
		if a.ID == id {
			return &a
		}
	}
	return nil
}

func (c *Capturador) validar(inv copias.Inventario) error {
	cfg := c.Config
	if cfg.MaxBytes <= 0 || cfg.MaxEntradas <= 0 || cfg.TiempoRecuperacionSegundos < 1 || cfg.TiempoRecuperacionSegundos > 3600 || len(copias.ValidarInventario(inv)) != 0 {
		return ErrConfiguracion
	}
	ids := map[string]bool{"pgdata": true}
	requeridos := map[string]bool{}
	for _, a := range inv.Release.Binarios {
		requeridos[a.ID] = true
	}
	for _, a := range inv.Release.Componentes {
		requeridos[a.ID] = true
	}
	for _, a := range inv.PostgreSQL.Almacenes {
		requeridos[a.ID] = true
	}
	if len(cfg.Fuentes) != len(requeridos) {
		return ErrConfiguracion
	}
	producidos := map[string]bool{"fisica:pgdata": true}
	rutas := []string{cfg.PGDATA}
	for _, f := range cfg.Fuentes {
		if f.ID == "" || f.Tipo == "" || ids[f.ID] || !requeridos[f.ID] {
			return ErrConfiguracion
		}
		idTar := "fisica:" + f.ID
		if len(idTar) > 128 || producidos[idTar] {
			return ErrConfiguracion
		}
		producidos[idTar] = true
		if raw := esperado(inv, f.ID); raw != nil {
			if raw.Tipo != f.Tipo || producidos[raw.ID] {
				return ErrConfiguracion
			}
			producidos[raw.ID] = true
		}
		for _, a := range inv.PostgreSQL.Almacenes {
			if a.ID == f.ID && a.Tipo != f.Tipo {
				return ErrConfiguracion
			}
		}
		ids[f.ID] = true
		rutas = append(rutas, f.Ruta)
	}
	rutas = append(rutas, cfg.Destino)
	for i, r := range rutas {
		if !filepath.IsAbs(r) || filepath.Clean(r) != r || sinEnlaces(r) != nil {
			return ErrConfiguracion
		}
		for j := 0; j < i; j++ {
			if solapan(r, rutas[j]) {
				return ErrConfiguracion
			}
		}
	}
	d, e := os.Stat(cfg.Destino)
	if e != nil || !d.IsDir() || d.Mode().Perm()&0077 != 0 {
		return ErrConfiguracion
	}
	return nil
}
func solapan(a, b string) bool {
	contiene := func(raiz, ruta string) bool {
		relativa, err := filepath.Rel(raiz, ruta)
		return err == nil && relativa != ".." && !strings.HasPrefix(relativa, ".."+string(os.PathSeparator))
	}
	return contiene(a, b) || contiene(b, a)
}
func sinEnlaces(r string) error {
	for p := r; ; p = filepath.Dir(p) {
		s, e := os.Lstat(p)
		if e != nil || s.Mode()&os.ModeSymlink != 0 {
			return ErrOrigen
		}
		if p == filepath.Dir(p) {
			break
		}
	}
	return nil
}

// Indice es metadato privado no autenticado hasta que CS03 proteja el conjunto.
type Indice struct {
	FormatoVersion   int             `json:"formato_version"`
	Estado           string          `json:"estado"`
	InventarioSHA256 string          `json:"inventario_sha256"`
	Entradas         []EntradaIndice `json:"entradas"`
}
type EntradaIndice struct {
	Archivo   string           `json:"archivo"`
	Artefacto copias.Artefacto `json:"artefacto"`
}

// ComprobarConfiguracion permite validar la CLI antes de detener escritores.
func (c *Capturador) ComprobarConfiguracion(inv copias.Inventario, bloqueo string) error {
	if c.validar(inv) != nil || !filepath.IsAbs(bloqueo) || filepath.Clean(bloqueo) != bloqueo || sinEnlaces(filepath.Dir(bloqueo)) != nil {
		return ErrConfiguracion
	}
	for _, r := range append([]Fuente{{Ruta: c.Config.PGDATA}, {Ruta: c.Config.Destino}}, c.Config.Fuentes...) {
		if solapan(bloqueo, r.Ruta) {
			return ErrConfiguracion
		}
	}
	return nil
}
