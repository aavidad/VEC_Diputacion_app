package bootstrap

import (
	"context"
	"errors"
	"regexp"
	"strconv"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/vec/reglas"
)

// Numeración visible de los expedientes (regla «c16.numeracion», duda 15):
// «AAAA/<prefijo><número>», con el número rellenado con ceros hasta los
// dígitos indicados. El año delante lo exigen las restricciones de las tablas
// de expedientes. Sin la regla rige «AAAA/CT-NNNNNN». Se publica al arrancar
// en la tabla de parámetros de solo adición (CT-000126), que es la que lee la
// función que reserva cada número; los números ya asignados no cambian.
const (
	reglaNumeracionCT         = reglas.CTNumeracion
	atributoPrefijoNumeracion = "prefijo"
	atributoDigitosNumeracion = "digitos"
	prefijoNumeracionDefecto  = "CT-"
	digitosNumeracionDefecto  = 6
	maximoDigitosNumeracion   = 9
	fuenteNumeracionDefecto   = "configuracion:ct:numeracion:predeterminada"
)

var (
	// El prefijo no termina en cifra: «CT-1» con «5» y «CT-» con «15» darían
	// el mismo número visible. La tabla de CT-000126 exige lo mismo.
	patronPrefijoNumeracionCT = regexp.MustCompile(`^([A-Za-z0-9._-]{0,19}[A-Za-z._-])?$`)

	errNumeracionNoValida = errors.New(
		"bootstrap: numeración de expedientes del catálogo de reglas no válida",
	)
	errMigracionNumeracionNoInstalada = errors.New(
		"bootstrap: falta la migración CT-000126 (parámetros de numeración de expedientes) en PostgreSQL de Contratación temporal",
	)
	errNumeracionNoPublicada = errors.New(
		"bootstrap: parámetros de numeración de expedientes no publicados",
	)
)

// numeracionExpedientesCT son los parámetros de la numeración visible.
// FuenteRef identifica la regla que los fija; no forma parte del contenido
// comparado, de modo que otra revisión del catálogo con los mismos valores no
// publica otra versión.
type numeracionExpedientesCT struct {
	Prefijo   string
	Digitos   int
	FuenteRef string
}

func numeracionExpedientesPredeterminadaCT() numeracionExpedientesCT {
	return numeracionExpedientesCT{
		Prefijo: prefijoNumeracionDefecto, Digitos: digitosNumeracionDefecto,
		FuenteRef: fuenteNumeracionDefecto,
	}
}

func (n numeracionExpedientesCT) valida() bool {
	return patronPrefijoNumeracionCT.MatchString(n.Prefijo) &&
		n.Digitos >= 1 && n.Digitos <= maximoDigitosNumeracion && n.FuenteRef != ""
}

// numeracionDesdeReglasCT lee la regla c16; sin ella devuelve nil.
func numeracionDesdeReglasCT(porClave map[string]reglas.Regla) (*numeracionExpedientesCT, error) {
	regla, existe := porClave[reglaNumeracionCT]
	if !existe {
		return nil, nil
	}
	texto := regla.Atributos[atributoDigitosNumeracion]
	digitos, err := strconv.Atoi(texto)
	prefijo, conPrefijo := regla.Atributos[atributoPrefijoNumeracion]
	numeracion := numeracionExpedientesCT{Prefijo: prefijo, Digitos: digitos, FuenteRef: regla.Referencia}
	if err != nil || strconv.Itoa(digitos) != texto || !conPrefijo ||
		regla.Unidad != reglas.UnidadNinguna || !numeracion.valida() {
		return nil, errNumeracionNoValida
	}
	return &numeracion, nil
}

// numeracionVigente es la del catálogo o, sin ella, la de siempre.
func (o *opcionesAnalisisCTDesarrollo) numeracionVigente() numeracionExpedientesCT {
	if o == nil || o.numeracion == nil {
		return numeracionExpedientesPredeterminadaCT()
	}
	return *o.numeracion
}

// publicarNumeracionExpedientesCT comprueba que CT-000126 está instalada y
// publica los parámetros. La función SQL solo añade una versión si difieren de
// los vigentes (o, sin ninguna, de los predeterminados): arrancar dos veces con
// el mismo catálogo no añade historia.
func publicarNumeracionExpedientesCT(
	ctx context.Context,
	consulta interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	numeracion numeracionExpedientesCT,
) error {
	if ctx == nil || dependenciaEsNulaContratacionTemporalDesarrollo(consulta) || !numeracion.valida() {
		return errNumeracionNoPublicada
	}
	var instalada bool
	if err := consulta.QueryRow(ctx, `
		SELECT pg_catalog.to_regprocedure(
		           'vec_contratacion_temporal.publicar_numeracion_parametros_v1(text,integer,text)'
		       ) IS NOT NULL`,
	).Scan(&instalada); err != nil {
		return errors.Join(errMigracionNumeracionNoInstalada, err)
	}
	if !instalada {
		return errMigracionNumeracionNoInstalada
	}
	var resultado string
	var version int64
	if err := consulta.QueryRow(ctx, `
		SELECT resultado, version
		  FROM vec_contratacion_temporal.publicar_numeracion_parametros_v1($1, $2, $3)`,
		numeracion.Prefijo, numeracion.Digitos, numeracion.FuenteRef,
	).Scan(&resultado, &version); err != nil {
		return errors.Join(errNumeracionNoPublicada, err)
	}
	if (resultado != "publicada" && resultado != "vigente") || version < 0 {
		return errNumeracionNoPublicada
	}
	return nil
}

func codigoFalloNumeracionExpedientesCT(err error) string {
	if errors.Is(err, errMigracionNumeracionNoInstalada) {
		return "migracion_ct126_no_instalada"
	}
	return "numeracion_no_publicada"
}
