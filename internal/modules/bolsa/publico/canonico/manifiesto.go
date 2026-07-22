package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"time"
)

const EsquemaManifiestoPublicoV2 = "vec.bolsa.manifiesto-publico.canonico.v2"

const (
	maximoSnapshotsCategoriasManifiesto = 64
	maximoCategoriasTotalesManifiesto   = 4_096
)

var (
	ErrManifiestoPublicoInvalido  = errors.New("bolsa publica: manifiesto canonico invalido")
	patronReferenciaCatalogo      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$`)
	patronIdentificadorManifiesto = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,79}$`)
	patronRevisionManifiesto      = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,79}$`)
	patronHuellaManifiesto        = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type FuenteManifiestoPublicoV2 struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
}

type EntradaCatalogoManifiestoV2 struct {
	Clave       string `json:"clave"`
	Etiqueta    string `json:"etiqueta"`
	Descripcion string `json:"descripcion"`
	Semantica   string `json:"semantica"`
	Orden       int    `json:"orden"`
}

type CatalogoManifiestoV2 struct {
	Referencia string                        `json:"referencia"`
	Version    int                           `json:"version"`
	Entradas   []EntradaCatalogoManifiestoV2 `json:"entradas"`
}

// ReferenciaCatalogoCategoriasManifiestoV2 identifica sin ambiguedad el
// snapshot que gobierna el directorio vigente. Nunca significa "la ultima".
type ReferenciaCatalogoCategoriasManifiestoV2 struct {
	CatalogoID                     string `json:"catalogo_id"`
	CatalogoVersion                int    `json:"catalogo_version"`
	CatalogoHuellaSHA256           string `json:"catalogo_huella_sha256"`
	CatalogoHuellaProyeccionSHA256 string `json:"catalogo_huella_proyeccion_sha256"`
}

// SnapshotCategoriasManifiestoV2 conserva el material de resolucion de cada
// referencia usada por una convocatoria, aunque sus entradas hayan caducado.
type SnapshotCategoriasManifiestoV2 struct {
	HuellaGobernadaSHA256  string               `json:"huella_gobernada_sha256"`
	HuellaProyeccionSHA256 string               `json:"huella_proyeccion_sha256"`
	Catalogo               CatalogoCategoriasV1 `json:"catalogo"`
}

type CategoriasManifiestoPublicoV2 struct {
	Actual    ReferenciaCatalogoCategoriasManifiestoV2 `json:"actual"`
	Snapshots []SnapshotCategoriasManifiestoV2         `json:"snapshots"`
}

type ConvocatoriaManifiestoPublicoV2 struct {
	IdentificadorPublico string `json:"identificador_publico"`
	HuellaCompletaSHA256 string `json:"huella_completa_sha256"`
	HuellaResumenSHA256  string `json:"huella_resumen_sha256"`
}

// ManifiestoPublicoV2 fija toda la proyeccion visible, el catalogo actual y
// todos los snapshots historicos necesarios para resolver convocatorias.
type ManifiestoPublicoV2 struct {
	Esquema       string                            `json:"esquema"`
	Fuente        FuenteManifiestoPublicoV2         `json:"fuente"`
	Catalogos     []CatalogoManifiestoV2            `json:"catalogos"`
	Categorias    CategoriasManifiestoPublicoV2     `json:"categorias"`
	Convocatorias []ConvocatoriaManifiestoPublicoV2 `json:"convocatorias"`
}

func (m ManifiestoPublicoV2) HuellaSHA256() (string, error) {
	canonico, err := m.canonico()
	if err != nil {
		return "", err
	}
	contenido, err := json.Marshal(canonico)
	if err != nil {
		return "", ErrManifiestoPublicoInvalido
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func (m ManifiestoPublicoV2) Validar() error {
	_, err := m.canonico()
	return err
}

func (m ManifiestoPublicoV2) canonico() (ManifiestoPublicoV2, error) {
	if m.Esquema != EsquemaManifiestoPublicoV2 ||
		!patronRevisionManifiesto.MatchString(m.Fuente.Revision) ||
		!instanteCanonico(m.Fuente.ActualizadaEn) || len(m.Catalogos) == 0 ||
		len(m.Catalogos) > 1_024 || len(m.Convocatorias) > 12_000 ||
		!patronIDCatalogo.MatchString(m.Categorias.Actual.CatalogoID) ||
		m.Categorias.Actual.CatalogoVersion < 1 ||
		!patronHuellaManifiesto.MatchString(m.Categorias.Actual.CatalogoHuellaSHA256) ||
		!patronHuellaManifiesto.MatchString(m.Categorias.Actual.CatalogoHuellaProyeccionSHA256) ||
		len(m.Categorias.Snapshots) == 0 ||
		len(m.Categorias.Snapshots) > maximoSnapshotsCategoriasManifiesto {
		return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
	}

	resultado := m
	resultado.Fuente.ActualizadaEn = m.Fuente.ActualizadaEn.UTC().Truncate(time.Microsecond)
	resultado.Catalogos = clonarCatalogosManifiesto(m.Catalogos)
	resultado.Categorias.Snapshots = clonarSnapshotsCategoriasManifiesto(m.Categorias.Snapshots)
	resultado.Convocatorias = append([]ConvocatoriaManifiestoPublicoV2(nil), m.Convocatorias...)

	type referenciaSnapshot struct {
		catalogoID string
		version    int
	}
	referenciasSnapshots := make(map[referenciaSnapshot]struct{}, len(resultado.Categorias.Snapshots))
	actualEncontrado := false
	totalCategorias := 0
	for indice := range resultado.Categorias.Snapshots {
		snapshot := &resultado.Categorias.Snapshots[indice]
		if !patronHuellaManifiesto.MatchString(snapshot.HuellaGobernadaSHA256) ||
			!patronHuellaManifiesto.MatchString(snapshot.HuellaProyeccionSHA256) ||
			snapshot.Catalogo.Validar() != nil ||
			snapshot.Catalogo.CatalogoID != resultado.Categorias.Actual.CatalogoID {
			return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
		}
		huellaCategorias, err := snapshot.Catalogo.HuellaSHA256()
		if err != nil || huellaCategorias != snapshot.HuellaProyeccionSHA256 {
			return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
		}
		sort.Slice(snapshot.Catalogo.Categorias, func(i, j int) bool {
			if snapshot.Catalogo.Categorias[i].Orden != snapshot.Catalogo.Categorias[j].Orden {
				return snapshot.Catalogo.Categorias[i].Orden < snapshot.Catalogo.Categorias[j].Orden
			}
			return snapshot.Catalogo.Categorias[i].Clave < snapshot.Catalogo.Categorias[j].Clave
		})
		clave := referenciaSnapshot{
			catalogoID: snapshot.Catalogo.CatalogoID,
			version:    snapshot.Catalogo.Version,
		}
		if _, duplicada := referenciasSnapshots[clave]; duplicada {
			return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
		}
		referenciasSnapshots[clave] = struct{}{}
		totalCategorias += len(snapshot.Catalogo.Categorias)
		if snapshot.Catalogo.CatalogoID == resultado.Categorias.Actual.CatalogoID &&
			snapshot.Catalogo.Version == resultado.Categorias.Actual.CatalogoVersion &&
			snapshot.HuellaGobernadaSHA256 == resultado.Categorias.Actual.CatalogoHuellaSHA256 &&
			snapshot.HuellaProyeccionSHA256 == resultado.Categorias.Actual.CatalogoHuellaProyeccionSHA256 {
			actualEncontrado = true
		}
	}
	if !actualEncontrado || totalCategorias > maximoCategoriasTotalesManifiesto {
		return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
	}
	sort.Slice(resultado.Categorias.Snapshots, func(i, j int) bool {
		a, b := resultado.Categorias.Snapshots[i].Catalogo, resultado.Categorias.Snapshots[j].Catalogo
		if a.CatalogoID != b.CatalogoID {
			return a.CatalogoID < b.CatalogoID
		}
		return a.Version < b.Version
	})

	referencias := make(map[string]struct{}, len(resultado.Catalogos))
	totalEntradas := 0
	for indice := range resultado.Catalogos {
		catalogo := &resultado.Catalogos[indice]
		if !patronReferenciaCatalogo.MatchString(catalogo.Referencia) || catalogo.Version < 1 ||
			len(catalogo.Entradas) == 0 || len(catalogo.Entradas) > 256 {
			return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
		}
		if _, duplicada := referencias[catalogo.Referencia]; duplicada {
			return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
		}
		referencias[catalogo.Referencia] = struct{}{}
		claves, ordenes := make(map[string]struct{}), make(map[int]struct{})
		for _, entrada := range catalogo.Entradas {
			if !patronReferenciaCatalogo.MatchString(entrada.Clave) || entrada.Orden < 1 ||
				!textoCanonico(entrada.Etiqueta, 120, false) ||
				!textoCanonico(entrada.Descripcion, 600, true) ||
				!patronReferenciaCatalogo.MatchString(entrada.Semantica) {
				return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
			}
			if _, duplicada := claves[entrada.Clave]; duplicada {
				return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
			}
			if _, duplicado := ordenes[entrada.Orden]; duplicado {
				return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
			}
			claves[entrada.Clave], ordenes[entrada.Orden] = struct{}{}, struct{}{}
		}
		totalEntradas += len(catalogo.Entradas)
		sort.Slice(catalogo.Entradas, func(i, j int) bool {
			if catalogo.Entradas[i].Orden != catalogo.Entradas[j].Orden {
				return catalogo.Entradas[i].Orden < catalogo.Entradas[j].Orden
			}
			return catalogo.Entradas[i].Clave < catalogo.Entradas[j].Clave
		})
	}
	if totalEntradas > 1_024 {
		return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
	}
	sort.Slice(resultado.Catalogos, func(i, j int) bool {
		return resultado.Catalogos[i].Referencia < resultado.Catalogos[j].Referencia
	})

	identificadores := make(map[string]struct{}, len(resultado.Convocatorias))
	for _, convocatoria := range resultado.Convocatorias {
		if !patronIdentificadorManifiesto.MatchString(convocatoria.IdentificadorPublico) ||
			!patronHuellaManifiesto.MatchString(convocatoria.HuellaCompletaSHA256) ||
			!patronHuellaManifiesto.MatchString(convocatoria.HuellaResumenSHA256) {
			return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
		}
		if _, duplicada := identificadores[convocatoria.IdentificadorPublico]; duplicada {
			return ManifiestoPublicoV2{}, ErrManifiestoPublicoInvalido
		}
		identificadores[convocatoria.IdentificadorPublico] = struct{}{}
	}
	sort.Slice(resultado.Convocatorias, func(i, j int) bool {
		return resultado.Convocatorias[i].IdentificadorPublico < resultado.Convocatorias[j].IdentificadorPublico
	})
	return resultado, nil
}

func clonarCatalogosManifiesto(origen []CatalogoManifiestoV2) []CatalogoManifiestoV2 {
	resultado := make([]CatalogoManifiestoV2, len(origen))
	for indice, catalogo := range origen {
		resultado[indice] = catalogo
		resultado[indice].Entradas = append([]EntradaCatalogoManifiestoV2(nil), catalogo.Entradas...)
	}
	return resultado
}

func clonarSnapshotsCategoriasManifiesto(
	origen []SnapshotCategoriasManifiestoV2,
) []SnapshotCategoriasManifiestoV2 {
	resultado := make([]SnapshotCategoriasManifiestoV2, len(origen))
	for indice, snapshot := range origen {
		resultado[indice] = snapshot
		resultado[indice].Catalogo.Categorias = clonarCategorias(snapshot.Catalogo.Categorias)
	}
	return resultado
}
