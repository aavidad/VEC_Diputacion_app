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
	"vec-diputacion-granada/internal/shared/telemetria"
)

type falloReferenciaOrganizativaCT struct {
	codigo string
	origen error
}

func (*falloReferenciaOrganizativaCT) Error() string { return errReferenciaOrganizativaCT.Error() }
func (f *falloReferenciaOrganizativaCT) Unwrap() []error {
	if f.origen == nil {
		return []error{errReferenciaOrganizativaCT}
	}
	return []error{errReferenciaOrganizativaCT, f.origen}
}

func falloReferenciaCT(codigo string, origen error) error {
	return &falloReferenciaOrganizativaCT{codigo: codigo, origen: origen}
}

func registrarFalloReferenciaOrganizativaCT(err error) {
	causa, clase := "referencia_no_disponible", "otro"
	var fallo *falloReferenciaOrganizativaCT
	if errors.As(err, &fallo) {
		causa, clase = fallo.codigo, telemetria.ClaseErrorArranque(fallo.origen)
	}
	slog.Warn("ct_referencia_organizativa_no_disponible", "causa", causa, "tipo", clase)
}

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
	if ruta == "" {
		return vacia, falloReferenciaCT("archivo_no_configurado", nil)
	}
	if !filepath.IsAbs(ruta) || dentroDeRepositorioGit(ruta) || validarArbolMaterialDesarrollo(filepath.Dir(ruta)) != nil {
		return vacia, falloReferenciaCT("ruta_privada_invalida", nil)
	}
	raiz, err := os.OpenRoot(filepath.Dir(ruta))
	if err != nil {
		return vacia, falloReferenciaCT("archivo_no_accesible", err)
	}
	contenido, err := leerArchivoIncorporacionV2(raiz, filepath.Base(ruta), 8<<10)
	cierre := raiz.Close()
	if err != nil {
		borrarBytes(contenido)
		return vacia, falloReferenciaCT("archivo_privado_invalido", err)
	}
	if cierre != nil {
		borrarBytes(contenido)
		return vacia, falloReferenciaCT("cierre_archivo_no_confirmado", cierre)
	}
	defer borrarBytes(contenido)

	var archivo archivoReferenciaOrganizativaCT
	lector := json.NewDecoder(bytes.NewReader(contenido))
	lector.DisallowUnknownFields()
	if err := validarClavesJSONUnicas(contenido); err != nil {
		return vacia, falloReferenciaCT("claves_json_invalidas", err)
	}
	if err := lector.Decode(&archivo); err != nil {
		return vacia, falloReferenciaCT("contenido_json_invalido", err)
	}
	if err := lector.Decode(new(any)); err != io.EOF {
		return vacia, falloReferenciaCT("contenido_json_adicional", err)
	}
	if archivo.Esquema != esquemaReferenciaOrganizativaCT ||
		!patronUnidadReferenciaCT.MatchString(archivo.UnidadRef) || !ct.UnidadSeguimientoValida(archivo.UnidadRef) ||
		!patronFuenteReferenciaCT.MatchString(archivo.FuenteUnidad.Referencia) ||
		archivo.FuenteUnidad.Version < 1 || archivo.FuenteUnidad.Version > 2147483647 ||
		!patronHuellaReferenciaCT.MatchString(archivo.FuenteUnidad.HuellaSHA256) ||
		archivo.FuenteUnidad.HuellaSHA256 == "0000000000000000000000000000000000000000000000000000000000000000" {
		return vacia, falloReferenciaCT("coordenadas_invalidas", nil)
	}
	if _, err := personal.NuevaReferenciaOrganizacion(archivo.OrganizacionRef); err != nil {
		return vacia, falloReferenciaCT("organizacion_no_canonica", err)
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
		return falloReferenciaCT("cotejo_interrumpido", nil)
	}
	ambito, err := referencia.ambitoUnidad()
	if err != nil {
		return falloReferenciaCT("selector_unidad_invalido", err)
	}
	pool, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsn,
		"vec-ct-referencia-organizativa", rolEjecucionPostgreSQLContratacionTemporalDesarrollo)
	if err != nil {
		return falloReferenciaCT("conexion_cotejo_no_disponible", err)
	}
	defer pool.Close()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return falloReferenciaCT("transaccion_cotejo_no_disponible", err)
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
		return falloReferenciaCT("fuente_maestra_no_confirmada", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return falloReferenciaCT("confirmacion_cotejo_no_disponible", err)
	}
	return nil
}
