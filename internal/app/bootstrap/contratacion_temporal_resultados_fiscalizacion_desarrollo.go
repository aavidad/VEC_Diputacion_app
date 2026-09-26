package bootstrap

import (
	"context"
	"errors"
	"time"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Resultados de fiscalización gobernados por la regla c12 del catálogo de
// ejemplo (duda 5 de RRHH). Sin catálogo, o sin la regla, rige la conducta de
// siempre. Una regla declarada pero incoherente impide arrancar.

var errResultadosFiscalizacionDesarrolloNoValidos = errors.New(
	"contratacion temporal: regla c12.fiscalizacion_resultados no valida",
)

// fuenteResultadosFiscalizacionReglas lee la regla vigente en cada petición.
type fuenteResultadosFiscalizacionReglas struct{ resolutor *reglas.Resolutor }

var _ ports.FuenteResultadosFiscalizacion = fuenteResultadosFiscalizacionReglas{}

func (f fuenteResultadosFiscalizacionReglas) ResultadosFiscalizacion(ctx context.Context) (ctdomain.PoliticaResultadosFiscalizacion, error) {
	regla, err := f.resolutor.Regla(ctx, reglas.CTFiscalizacionResultados)
	if err != nil {
		return ctdomain.PoliticaResultadosFiscalizacion{}, err
	}
	return politicaResultadosFiscalizacionDesdeRegla(regla)
}

// politicaResultadosFiscalizacionDesdeRegla traduce la lista y los atributos
// «efecto_<resultado>» al vocabulario del dominio.
func politicaResultadosFiscalizacionDesdeRegla(regla reglas.Regla) (ctdomain.PoliticaResultadosFiscalizacion, error) {
	elementos := regla.Elementos()
	resultados := make([]ctdomain.ResultadoFiscalizacion, 0, len(elementos))
	efectos := make(map[ctdomain.ResultadoFiscalizacion]ctdomain.EfectoResultadoFiscalizacion, len(elementos))
	for _, e := range elementos {
		r := ctdomain.ResultadoFiscalizacion(e)
		resultados = append(resultados, r)
		if efecto, ok := regla.Atributos["efecto_"+e]; ok {
			efectos[r] = ctdomain.EfectoResultadoFiscalizacion(efecto)
		}
	}
	p, err := ctdomain.NuevaPoliticaResultadosFiscalizacion(resultados, efectos)
	if err != nil {
		return ctdomain.PoliticaResultadosFiscalizacion{}, errors.Join(errResultadosFiscalizacionDesarrolloNoValidos, err)
	}
	return p, nil
}

// gobernarResultadosFiscalizacionDesarrollo valida la regla al arrancar y la
// entrega al servicio. Sin resolutor o sin la regla no hace nada.
func gobernarResultadosFiscalizacionDesarrollo(servicio *ctapplication.ServicioFiscalizaciones, resolutor *reglas.Resolutor) error {
	if resolutor == nil {
		return nil
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	vigentes, err := resolutor.Reglas(ctx)
	if err != nil {
		return errors.Join(errResultadosFiscalizacionDesarrolloNoValidos, err)
	}
	for _, regla := range vigentes {
		if regla.Clave != reglas.CTFiscalizacionResultados {
			continue
		}
		if _, err := politicaResultadosFiscalizacionDesdeRegla(regla); err != nil {
			return err
		}
		if servicio == nil {
			return errResultadosFiscalizacionDesarrolloNoValidos
		}
		return servicio.GobernarResultados(fuenteResultadosFiscalizacionReglas{resolutor: resolutor})
	}
	return nil
}
