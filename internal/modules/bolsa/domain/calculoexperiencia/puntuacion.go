package calculoexperiencia

import "vec-diputacion-granada/internal/shared/baremacion"

type numeroOperacionesPuntuacionV1 uint64

const (
	baseOperacionesPuntuacionV1   numeroOperacionesPuntuacionV1 = 32
	porAplicacionPuntuacionV1     numeroOperacionesPuntuacionV1 = 20
	porReglaPuntuacionV1          numeroOperacionesPuntuacionV1 = 24
	porSeccionPuntuacionV1        numeroOperacionesPuntuacionV1 = 8
	maximoOperacionesPuntuacionV1 numeroOperacionesPuntuacionV1 = maximoOperacionesExactas
)

// resultadoPuntuacionExperienciaV1 es la salida privada de la fase aritmetica.
// Solo contiene material minimizado apto para el resultado sellado. Los hechos
// de entrada usados para calcularlo no sobreviven en esta estructura.
type resultadoPuntuacionExperienciaV1 struct {
	intervalos   []IntervaloAplicacionResultadoExperienciaV1
	aplicaciones []AplicacionCalculadaResultadoExperienciaV1
	reglas       []ResultadoReglaExperienciaV1
	secciones    []SubtotalSeccionResultadoExperienciaV1
	bloqueos     []BloqueoResultadoExperienciaV1
	total        baremacion.Puntos
	operaciones  numeroOperacionesPuntuacionV1
}

func (r resultadoPuntuacionExperienciaV1) bloqueada() bool {
	return len(r.bloqueos) != 0
}

// puntuarExperienciaV1 valida las cuatro instantaneas de la cadena y ejecuta
// una sola evaluacion exacta y acotada. Un redondeo exacto imposible es una
// decision de negocio reproducible; cualquier otra anomalia devuelve error y
// descarta todo el material parcial.
func puntuarExperienciaV1(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
	seleccion seleccionExperiencia,
	temporales resultadoAplicacionesTemporales,
) (resultadoPuntuacionExperienciaV1, error) {
	contexto, err := prepararContextoPuntuacionV1(plan, entrada, seleccion, temporales)
	if err != nil {
		return resultadoPuntuacionExperienciaV1{}, err
	}
	if err := comprobarPresupuestoSalidaPuntuacionV1(
		len(contexto.orden), len(contexto.reglas), len(contexto.secciones),
	); err != nil {
		return resultadoPuntuacionExperienciaV1{}, err
	}
	previstas, err := comprobarPresupuestoPuntuacionV1(
		numeroOperacionesPuntuacionV1(len(contexto.orden)),
		numeroOperacionesPuntuacionV1(len(contexto.reglas)),
		numeroOperacionesPuntuacionV1(len(contexto.secciones)),
	)
	if err != nil {
		return resultadoPuntuacionExperienciaV1{}, err
	}
	contador, err := nuevoContadorOperacionesConLimite(uint64(previstas))
	if err != nil {
		return resultadoPuntuacionExperienciaV1{}, err
	}
	resultado, err := ejecutarPuntuacionV1(contexto, contador)
	if err != nil {
		return resultadoPuntuacionExperienciaV1{}, err
	}
	resultado.operaciones = numeroOperacionesPuntuacionV1(contador.realizadas)
	return resultado, nil
}

func comprobarPresupuestoPuntuacionV1(
	aplicaciones numeroOperacionesPuntuacionV1,
	reglas numeroOperacionesPuntuacionV1,
	secciones numeroOperacionesPuntuacionV1,
) (numeroOperacionesPuntuacionV1, error) {
	if aplicaciones > maximoAplicacionesResultadoV1 ||
		reglas > maximoReglasResultadoV1 || secciones > maximoSeccionesResultadoV1 {
		return 0, nuevoError("puntuacion.presupuesto", CodigoFueraDeLimites)
	}
	total := baseOperacionesPuntuacionV1 +
		porAplicacionPuntuacionV1*aplicaciones +
		porReglaPuntuacionV1*reglas +
		porSeccionPuntuacionV1*secciones
	if total > maximoOperacionesPuntuacionV1 {
		return 0, nuevoError("puntuacion.presupuesto", CodigoLimiteOperaciones)
	}
	return total, nil
}

// registrarPuntuacionExperienciaV1 es el unico puente con el constructor del
// resultado. En caso de bloqueo conserva intervalos y calculos por aplicacion,
// pero nunca publica reglas, secciones ni un total parcial.
func registrarPuntuacionExperienciaV1(
	registrador *registradorResultadoExperienciaV1,
	resultado resultadoPuntuacionExperienciaV1,
) error {
	if registrador == nil || len(registrador.intervalos) != 0 ||
		len(registrador.aplicaciones) != 0 || len(registrador.reglas) != 0 ||
		len(registrador.secciones) != 0 || len(registrador.bloqueos) != 0 {
		return nuevoError("puntuacion.registrador", CodigoContextoIncompatible)
	}
	for _, intervalo := range resultado.intervalos {
		if err := registrador.registrarIntervalo(intervalo); err != nil {
			return err
		}
	}
	for _, aplicacion := range resultado.aplicaciones {
		if err := registrador.registrarAplicacion(aplicacion); err != nil {
			return err
		}
	}
	if resultado.bloqueada() {
		for _, bloqueo := range resultado.bloqueos {
			if err := registrador.registrarBloqueo(bloqueo); err != nil {
				return err
			}
		}
		return nil
	}
	for _, regla := range resultado.reglas {
		if err := registrador.registrarRegla(regla); err != nil {
			return err
		}
	}
	for _, seccion := range resultado.secciones {
		if err := registrador.registrarSeccion(seccion); err != nil {
			return err
		}
	}
	return nil
}
