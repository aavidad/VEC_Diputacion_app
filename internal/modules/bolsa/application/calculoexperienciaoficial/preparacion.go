package calculoexperienciaoficial

import (
	"errors"

	calculo "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func prepararCalculo(
	datos DatosOrdenConfiable,
	fuente puertosbolsa.FuenteExactaCalculoReglasBaremo,
) (calculoPreparado, error) {
	conjunto, err := fuente.Version.Conjunto()
	if err != nil {
		return calculoPreparado{}, errors.Join(ErrFuenteNoConfiable, err)
	}
	plan, err := calculo.Compilar(conjunto)
	if err != nil {
		return calculoPreparado{}, err
	}
	resultado, err := calculo.CalcularExperienciaV1(plan, fuente.Entrada)
	if err != nil {
		return calculoPreparado{}, err
	}
	canonico, err := resultado.RepresentacionCanonica()
	if err != nil {
		return calculoPreparado{}, errors.Join(ErrResultadoNoConfiable, err)
	}
	huella, err := resultado.HuellaSHA256()
	if err != nil {
		return calculoPreparado{}, errors.Join(ErrResultadoNoConfiable, err)
	}
	restaurado, err := calculo.RestaurarResultadoExperienciaV1ConHuellaSHA256(canonico, huella)
	errVinculos := validarVinculosResultado(resultado, datos, fuente)
	if err != nil || restaurado.Validar() != nil || errVinculos != nil {
		return calculoPreparado{}, errors.Join(ErrResultadoNoConfiable, err, errVinculos)
	}
	clave, err := nuevaClaveEfecto(datos, fuente, resultado)
	if err != nil {
		return calculoPreparado{}, errors.Join(ErrResultadoNoConfiable, err)
	}
	estado, fase, err := estadoYFaseOficial(resultado)
	if err != nil {
		return calculoPreparado{}, err
	}
	intencion, err := oficial.NuevaIntencionResultadoV1(clave, huella, estado, fase)
	if err != nil {
		return calculoPreparado{}, errors.Join(ErrResultadoNoConfiable, err)
	}
	return calculoPreparado{
		resultado: resultado, canonico: canonico, huella: huella,
		clave: clave, intencion: intencion,
	}, nil
}

func validarVinculosResultado(
	resultado calculo.ResultadoExperienciaV1,
	datos DatosOrdenConfiable,
	fuente puertosbolsa.FuenteExactaCalculoReglasBaremo,
) error {
	if resultado.Validar() != nil {
		return ErrResultadoNoConfiable
	}
	vinculos := resultado.Vinculos()
	motor := vinculos.Motor()
	entrada := vinculos.Entrada()
	if motor.Contrato() != datos.MotorEsperado.Contrato ||
		motor.Version() != datos.MotorEsperado.Version ||
		motor.HuellaContratoSHA256() != datos.MotorEsperado.HuellaContratoSHA256 {
		return ErrMotorNoCoincide
	}
	if !referenciasIguales(vinculos.Conjunto(), datos.Selector.EstadoReglas.Contenido()) ||
		!referenciasIguales(entrada.Instantanea(), datos.Selector.InstantaneaEntrada) ||
		entrada.HuellaContenidoSHA256() != fuente.Prueba.HuellaEntradaSHA256 ||
		!huellaSHA256Valida(vinculos.Plan().HuellaSHA256()) {
		return ErrResultadoNoConfiable
	}
	return nil
}

func nuevaClaveEfecto(
	datos DatosOrdenConfiable,
	fuente puertosbolsa.FuenteExactaCalculoReglasBaremo,
	resultado calculo.ResultadoExperienciaV1,
) (oficial.ClaveEfectoV1, error) {
	vinculos := resultado.Vinculos()
	estado := datos.Selector.EstadoReglas
	entrada := vinculos.Entrada()
	motor := vinculos.Motor()
	predecesor := datos.Predecesor
	if predecesor != nil {
		clon := *predecesor
		predecesor = &clon
	}
	return oficial.NuevaClaveEfectoV1(oficial.DatosClaveEfectoV1{
		SujetoPseudonimizado: referenciaOficial(datos.Selector.SujetoPseudonimo),
		Convocatoria:         referenciaOficial(datos.Selector.Convocatoria),
		Reglas: oficial.VinculoReglasV1{
			Contenido: referenciaOficial(estado.Contenido()), Revision: estado.Revision(),
			HuellaEstadoSHA256: estado.HuellaEstadoSHA256(),
		},
		Entrada: oficial.VinculoEntradaV1{
			Instantanea:           referenciaOficial(entrada.Instantanea()),
			HuellaContenidoSHA256: fuente.Prueba.HuellaEntradaSHA256,
		},
		Motor: oficial.VinculoMotorV1{
			Contrato: motor.Contrato(), Version: motor.Version(),
			HuellaContratoSHA256: motor.HuellaContratoSHA256(),
		},
		HuellaPlanSHA256: vinculos.Plan().HuellaSHA256(), Causa: datos.Causa,
		Tipo: datos.TipoEfecto, Predecesor: predecesor,
	})
}

func referenciaOficial(referencia reglas.ReferenciaVersionada) oficial.ReferenciaExactaV1 {
	return oficial.ReferenciaExactaV1{
		Referencia: referencia.Referencia(), Version: referencia.Version(),
		HuellaSHA256: referencia.HuellaSHA256(),
	}
}

func estadoYFaseOficial(
	resultado calculo.ResultadoExperienciaV1,
) (oficial.EstadoResultadoV1, oficial.FaseResultadoV1, error) {
	if resultado.Estado() == calculo.ResultadoExperienciaCompletado &&
		resultado.Fase() == calculo.FaseResultadoCompletado {
		return oficial.ResultadoCompletado, oficial.FaseCompletado, nil
	}
	if resultado.Estado() != calculo.ResultadoExperienciaBloqueado {
		return "", "", ErrResultadoNoConfiable
	}
	switch resultado.Fase() {
	case calculo.FaseResultadoSeleccion:
		return oficial.ResultadoBloqueado, oficial.FaseSeleccion, nil
	case calculo.FaseResultadoIntervalos:
		return oficial.ResultadoBloqueado, oficial.FaseIntervalos, nil
	case calculo.FaseResultadoPuntuacion:
		return oficial.ResultadoBloqueado, oficial.FasePuntuacion, nil
	default:
		return "", "", ErrResultadoNoConfiable
	}
}
