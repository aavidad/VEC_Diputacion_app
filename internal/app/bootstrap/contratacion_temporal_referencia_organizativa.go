package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jackc/pgx/v5"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
)

const esquemaReferenciaOrganizativaCT = "vec.contratacion-temporal.referencia-organizativa.v1"

var (
	errReferenciaOrganizativaCT = errors.New("contratacion temporal: referencia organizativa no disponible")
	patronUnidadReferenciaCT    = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,127}$`)
	patronFuenteReferenciaCT    = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,159}$`)
	patronHuellaReferenciaCT    = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Las coordenadas son privadas e inmutables tras la lectura. Su forma no
// acredita publicación ni vigencia: Personal31 debe cotejarlas en SQL.
type referenciaCT struct {
	organizacionRef string
	unidadRef       string
	fuenteUnidad    fuenteUnidadReferenciaCT
}

type fuenteUnidadReferenciaCT struct {
	referencia   string
	version      int
	huellaSHA256 string
}

type archivoReferenciaOrganizativaCT struct {
	Esquema         string `json:"esquema"`
	OrganizacionRef string `json:"organizacion_ref"`
	UnidadRef       string `json:"unidad_ref"`
	FuenteUnidad    struct {
		Referencia   string `json:"referencia"`
		Version      int    `json:"version"`
		HuellaSHA256 string `json:"huella_sha256"`
	} `json:"fuente_unidad"`
}

func leerReferenciaOrganizativaCT(ruta string) (referenciaCT, error) {
	var vacia referenciaCT
	if !filepath.IsAbs(ruta) || dentroDeRepositorioGit(ruta) || validarArbolMaterialDesarrollo(filepath.Dir(ruta)) != nil {
		return vacia, errReferenciaOrganizativaCT
	}
	raiz, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return vacia, errReferenciaOrganizativaCT
	}
	contenido, err := leerArchivoIncorporacionV2(raiz, filepath.Base(ruta), 8<<10)
	cierre := raiz.Close()
	if err != nil || cierre != nil {
		borrarBytes(contenido)
		return vacia, errReferenciaOrganizativaCT
	}
	defer borrarBytes(contenido)

	var archivo archivoReferenciaOrganizativaCT
	lector := json.NewDecoder(bytes.NewReader(contenido))
	lector.DisallowUnknownFields()
	if validarClavesJSONUnicas(contenido) != nil || lector.Decode(&archivo) != nil || lector.Decode(new(any)) != io.EOF ||
		archivo.Esquema != esquemaReferenciaOrganizativaCT ||
		!patronUnidadReferenciaCT.MatchString(archivo.UnidadRef) || !ct.UnidadSeguimientoValida(archivo.UnidadRef) ||
		!patronFuenteReferenciaCT.MatchString(archivo.FuenteUnidad.Referencia) ||
		archivo.FuenteUnidad.Version < 1 || archivo.FuenteUnidad.Version > 2147483647 ||
		!patronHuellaReferenciaCT.MatchString(archivo.FuenteUnidad.HuellaSHA256) ||
		archivo.FuenteUnidad.HuellaSHA256 == "0000000000000000000000000000000000000000000000000000000000000000" {
		return vacia, errReferenciaOrganizativaCT
	}
	if _, err := personal.NuevaReferenciaOrganizacion(archivo.OrganizacionRef); err != nil {
		return vacia, errReferenciaOrganizativaCT
	}
	return referenciaCT{
		organizacionRef: archivo.OrganizacionRef,
		unidadRef:       archivo.UnidadRef,
		fuenteUnidad: fuenteUnidadReferenciaCT{
			referencia:   archivo.FuenteUnidad.Referencia,
			version:      archivo.FuenteUnidad.Version,
			huellaSHA256: archivo.FuenteUnidad.HuellaSHA256,
		},
	}, nil
}

// ambitoUnidad entrega exactamente el selector que coteja Personal31.
func (r referenciaCT) ambitoUnidad() (json.RawMessage, error) {
	selector := struct {
		Dimension string   `json:"dimension"`
		Valores   []string `json:"valores"`
		Fuente    struct {
			Referencia   string `json:"referencia"`
			Version      int    `json:"version"`
			HuellaSHA256 string `json:"huella_sha256"`
		} `json:"fuente"`
	}{Dimension: "unidad_ref", Valores: []string{r.unidadRef}}
	selector.Fuente.Referencia = r.fuenteUnidad.referencia
	selector.Fuente.Version = r.fuenteUnidad.version
	selector.Fuente.HuellaSHA256 = r.fuenteUnidad.huellaSHA256
	return json.Marshal(selector)
}

// El cotejo precede al gobierno de arranque y usa el ejecutor técnico que CT
// ya tiene. La conformidad no sustituye las decisiones de cada actuación.
func cotejarReferenciaOrganizativaCT(ctx context.Context, dsn string, referencia referenciaCT) error {
	if ctx == nil || ctx.Err() != nil {
		return errReferenciaOrganizativaCT
	}
	ambito, err := referencia.ambitoUnidad()
	if err != nil {
		return errReferenciaOrganizativaCT
	}
	pool, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsn,
		"vec-ct-referencia-organizativa", rolEjecucionPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		return errReferenciaOrganizativaCT
	}
	defer pool.Close()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return errReferenciaOrganizativaCT
	}
	defer func() {
		if err := tx.Rollback(context.WithoutCancel(ctx)); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Warn("contratación temporal: cotejo de fuente no disponible", "causa", "rollback_no_confirmado")
		}
	}()
	var conforme bool
	if err := tx.QueryRow(ctx,
		`SELECT vec_contratacion_temporal.cotejar_referencia_organizativa_v1($1,$2::jsonb)`,
		referencia.organizacionRef, string(ambito)).Scan(&conforme); err != nil || !conforme {
		return errReferenciaOrganizativaCT
	}
	if err := tx.Commit(ctx); err != nil {
		return errReferenciaOrganizativaCT
	}
	return nil
}
