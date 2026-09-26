package bootstrap

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Informe jurídico nuevo tras subsanar un reparo (duda 5 de RRHH), gobernado
// por los atributos «informe_nuevo_tras_subsanacion» (si/no) y
// «documento_firma_informe_nuevo» (clave del circuito de firma) de la regla
// c19.fiscalizacion_resultados. Sin catálogo, sin la regla o sin el atributo
// rige la conducta de siempre: se fiscaliza de nuevo con el mismo informe.
// Si la regla lo exige, CT123 debe estar instalada o no se arranca. Con
// CT123 instalada la política se publica también en la base, que la aplica
// a la nueva fiscalización aunque la aplicación no lo comprobara.

const (
	atributoInformeNuevoTrasSubsanacion = "informe_nuevo_tras_subsanacion"
	atributoDocumentoFirmaInformeNuevo  = "documento_firma_informe_nuevo"
)

var (
	errInformeTrasSubsanacionNoValido = errors.New(
		"contratacion temporal: atributos de informe nuevo tras subsanar no validos en la regla c19.fiscalizacion_resultados",
	)
	errInformeTrasSubsanacionSinCT123 = errors.New(
		"contratacion temporal: la regla c19 exige informe nuevo tras subsanar y falta la migracion CT123 (000123_informe_nuevo_tras_subsanacion)",
	)
	errInformeTrasSubsanacionNoPublicada = errors.New(
		"contratacion temporal: politica de informe nuevo tras subsanar no publicada en PostgreSQL (CT123)",
	)
)

// fuentePoliticaInformeNuevoPredeterminada identifica la conducta de siempre
// (no exigir) cuando no hay catálogo o regla c19 que la fije.
const fuentePoliticaInformeNuevoPredeterminada = "configuracion:ct:informe-tras-subsanacion:predeterminada"

// consultaPublicacionPoliticaCT123Instalada detecta, con el rol de gobierno,
// la función que publica la política.
const consultaPublicacionPoliticaCT123Instalada = `SELECT pg_catalog.to_regprocedure(
  'vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1(boolean,text)') IS NOT NULL`

// consultaCT123InstaladaDesarrollo detecta CT123 por sus funciones públicas y
// el permiso del rol de ejecución.
const consultaCT123InstaladaDesarrollo = `SELECT coalesce(bool_and(pg_catalog.to_regprocedure(f) IS NOT NULL
  AND pg_catalog.has_function_privilege(current_user, pg_catalog.to_regprocedure(f), 'EXECUTE')), false)
  FROM pg_catalog.unnest(ARRAY[
    'vec_contratacion_temporal.preparar_informe_juridico_tras_subsanacion_v1(jsonb)',
    'vec_contratacion_temporal.confirmar_informe_juridico_tras_subsanacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
    'vec_contratacion_temporal.inicio_ronda_informe_nuevo_v1(text,text)']) AS f`

// fuenteInformeTrasSubsanacionReglas lee la regla vigente en cada petición.
type fuenteInformeTrasSubsanacionReglas struct{ resolutor *reglas.Resolutor }

var _ ports.FuenteInformeTrasSubsanacion = fuenteInformeTrasSubsanacionReglas{}

func (f fuenteInformeTrasSubsanacionReglas) InformeTrasSubsanacion(ctx context.Context) (ctdomain.PoliticaInformeTrasSubsanacion, error) {
	regla, err := f.resolutor.Regla(ctx, reglas.CTFiscalizacionResultados)
	if errors.Is(err, reglas.ErrReglaNoEncontrada) {
		return ctdomain.PoliticaInformeTrasSubsanacion{}, nil
	}
	if err != nil {
		return ctdomain.PoliticaInformeTrasSubsanacion{}, err
	}
	return politicaInformeTrasSubsanacionDesdeRegla(regla)
}

// politicaInformeTrasSubsanacionDesdeRegla traduce los atributos. Un valor
// distinto de «si» o «no» impide arrancar: la decisión no se adivina.
func politicaInformeTrasSubsanacionDesdeRegla(regla reglas.Regla) (ctdomain.PoliticaInformeTrasSubsanacion, error) {
	var p ctdomain.PoliticaInformeTrasSubsanacion
	switch valor, declarado := regla.Atributos[atributoInformeNuevoTrasSubsanacion]; {
	case !declarado || valor == "no":
	case valor == "si":
		p.ExigeInformeNuevo = true
	default:
		return ctdomain.PoliticaInformeTrasSubsanacion{}, errInformeTrasSubsanacionNoValido
	}
	p.DocumentoFirma = regla.Atributos[atributoDocumentoFirmaInformeNuevo]
	if p.Validar() != nil {
		return ctdomain.PoliticaInformeTrasSubsanacion{}, errors.Join(errInformeTrasSubsanacionNoValido, p.Validar())
	}
	return p, nil
}

// consultaPoliticaInformeNuevoCT publica la política con el pool de gobierno.
type consultaPoliticaInformeNuevoCT interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// publicarPoliticaInformeTrasSubsanacionCT publica en la base (CT123) la
// política vigente para que la nueva fiscalización la compruebe también en
// SQL. Sin CT123 no hay nada que publicar; la función SQL solo añade versión
// si difiere de la vigente (o, sin ninguna, si exige): arrancar dos veces no
// añade historia.
func publicarPoliticaInformeTrasSubsanacionCT(
	ctx context.Context, gobierno consultaPoliticaInformeNuevoCT, exige bool, fuenteRef string,
) error {
	if dependenciaEsNulaContratacionTemporalDesarrollo(gobierno) {
		if exige {
			return errInformeTrasSubsanacionSinCT123
		}
		return nil
	}
	var instalada bool
	if err := gobierno.QueryRow(ctx, consultaPublicacionPoliticaCT123Instalada).Scan(&instalada); err != nil {
		return errors.Join(errInformeTrasSubsanacionNoPublicada, err)
	}
	if !instalada {
		if exige {
			return errInformeTrasSubsanacionSinCT123
		}
		return nil
	}
	var resultado string
	var version int64
	if err := gobierno.QueryRow(ctx, `
		SELECT resultado, version
		  FROM vec_contratacion_temporal.publicar_politica_informe_tras_subsanacion_v1($1, $2)`,
		exige, fuenteRef,
	).Scan(&resultado, &version); err != nil {
		return errors.Join(errInformeTrasSubsanacionNoPublicada, err)
	}
	if (resultado != "publicada" && resultado != "vigente") || version < 0 {
		return errInformeTrasSubsanacionNoPublicada
	}
	return nil
}

// gobernarInformeTrasSubsanacionDesarrollo valida la regla al arrancar, la
// publica en la base si CT123 está instalada y, si exige informe nuevo,
// comprueba CT123 y entrega la fuente a la fiscalización y al informe
// jurídico. Devuelve la fuente (nil si no se exige) para que el registro de
// firmas abra la segunda ronda. Sin catálogo o sin la regla se publica «no
// exigir», que solo escribe si antes se exigía.
func gobernarInformeTrasSubsanacionDesarrollo(
	resolutor *reglas.Resolutor, pool *pgxpool.Pool, gobierno consultaPoliticaInformeNuevoCT,
	fiscalizacion *ctapplication.ServicioFiscalizaciones,
	informes *ctapplication.ServicioInformesJuridicos,
) (ports.FuenteInformeTrasSubsanacion, error) {
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	politica, fuenteRef := ctdomain.PoliticaInformeTrasSubsanacion{}, fuentePoliticaInformeNuevoPredeterminada
	if resolutor != nil {
		vigentes, err := resolutor.Reglas(ctx)
		if err != nil {
			return nil, errors.Join(errInformeTrasSubsanacionNoValido, err)
		}
		for _, regla := range vigentes {
			if regla.Clave != reglas.CTFiscalizacionResultados {
				continue
			}
			if politica, err = politicaInformeTrasSubsanacionDesdeRegla(regla); err != nil {
				return nil, err
			}
			fuenteRef = regla.Referencia
			break
		}
	}
	if politica.ExigeInformeNuevo {
		instalada := false
		if pool == nil || pool.QueryRow(ctx, consultaCT123InstaladaDesarrollo).Scan(&instalada) != nil || !instalada {
			log.Print("contratacion temporal: informe nuevo tras subsanar exigido por el catalogo; falta CT123")
			return nil, errInformeTrasSubsanacionSinCT123
		}
	}
	if err := publicarPoliticaInformeTrasSubsanacionCT(ctx, gobierno, politica.ExigeInformeNuevo, fuenteRef); err != nil {
		log.Print("contratacion temporal: politica de informe nuevo tras subsanar no publicada")
		return nil, err
	}
	if !politica.ExigeInformeNuevo {
		return nil, nil
	}
	fuente := fuenteInformeTrasSubsanacionReglas{resolutor: resolutor}
	if fiscalizacion == nil || informes == nil ||
		fiscalizacion.GobernarInformeTrasSubsanacion(fuente) != nil ||
		informes.GobernarInformeTrasSubsanacion(fuente) != nil {
		return nil, errInformeTrasSubsanacionNoValido
	}
	return fuente, nil
}
