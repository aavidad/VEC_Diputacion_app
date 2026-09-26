package bootstrap

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Informe jurídico nuevo tras subsanar un reparo (duda 5 de RRHH), gobernado
// por los atributos «informe_nuevo_tras_subsanacion» (si/no) y
// «documento_firma_informe_nuevo» (clave del circuito de firma) de la regla
// c12.fiscalizacion_resultados. Sin catálogo, sin la regla o sin el atributo
// rige la conducta de siempre: se fiscaliza de nuevo con el mismo informe.
// Si la regla lo exige, CT123 debe estar instalada o no se arranca.

const (
	atributoInformeNuevoTrasSubsanacion = "informe_nuevo_tras_subsanacion"
	atributoDocumentoFirmaInformeNuevo  = "documento_firma_informe_nuevo"
)

var (
	errInformeTrasSubsanacionNoValido = errors.New(
		"contratacion temporal: atributos de informe nuevo tras subsanar no validos en la regla c12.fiscalizacion_resultados",
	)
	errInformeTrasSubsanacionSinCT123 = errors.New(
		"contratacion temporal: la regla c12 exige informe nuevo tras subsanar y falta la migracion CT123 (000123_informe_nuevo_tras_subsanacion)",
	)
)

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

// gobernarInformeTrasSubsanacionDesarrollo valida la regla al arrancar y, si
// exige informe nuevo, comprueba CT123 y entrega la fuente a la fiscalización
// y al informe jurídico. Devuelve la fuente (nil si no se exige) para que el
// registro de firmas abra la segunda ronda.
func gobernarInformeTrasSubsanacionDesarrollo(
	resolutor *reglas.Resolutor, pool *pgxpool.Pool,
	fiscalizacion *ctapplication.ServicioFiscalizaciones,
	informes *ctapplication.ServicioInformesJuridicos,
) (ports.FuenteInformeTrasSubsanacion, error) {
	if resolutor == nil {
		return nil, nil
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	vigentes, err := resolutor.Reglas(ctx)
	if err != nil {
		return nil, errors.Join(errInformeTrasSubsanacionNoValido, err)
	}
	for _, regla := range vigentes {
		if regla.Clave != reglas.CTFiscalizacionResultados {
			continue
		}
		politica, err := politicaInformeTrasSubsanacionDesdeRegla(regla)
		if err != nil {
			return nil, err
		}
		if !politica.ExigeInformeNuevo {
			return nil, nil
		}
		instalada := false
		if pool == nil || pool.QueryRow(ctx, consultaCT123InstaladaDesarrollo).Scan(&instalada) != nil || !instalada {
			log.Print("contratacion temporal: informe nuevo tras subsanar exigido por el catalogo; falta CT123")
			return nil, errInformeTrasSubsanacionSinCT123
		}
		fuente := fuenteInformeTrasSubsanacionReglas{resolutor: resolutor}
		if fiscalizacion == nil || informes == nil ||
			fiscalizacion.GobernarInformeTrasSubsanacion(fuente) != nil ||
			informes.GobernarInformeTrasSubsanacion(fuente) != nil {
			return nil, errInformeTrasSubsanacionNoValido
		}
		return fuente, nil
	}
	return nil, nil
}
