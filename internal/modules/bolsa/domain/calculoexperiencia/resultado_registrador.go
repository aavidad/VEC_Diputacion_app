package calculoexperiencia

import (
	"sort"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

// registradorResultadoExperienciaV1 es el unico punto de integracion del
// futuro motor. Recibe las decisiones ya tomadas en cada fase; no selecciona
// reglas ni recalcula intervalos, jornada, restos, redondeos o topes.
type registradorResultadoExperienciaV1 struct {
	vinculos     VinculosResultadoExperienciaV1
	seleccion    SeleccionResultadoExperienciaV1
	intervalos   []IntervaloAplicacionResultadoExperienciaV1
	aplicaciones []AplicacionCalculadaResultadoExperienciaV1
	reglas       []ResultadoReglaExperienciaV1
	secciones    []SubtotalSeccionResultadoExperienciaV1
	bloqueos     []BloqueoResultadoExperienciaV1
}

func nuevoRegistradorResultadoExperienciaV1(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
	seleccion seleccionExperiencia,
) (*registradorResultadoExperienciaV1, error) {
	vinculos, err := nuevosVinculosResultadoExperienciaV1(plan, entrada)
	if err != nil {
		return nil, err
	}
	return nuevoRegistradorResultadoConVinculosV1(vinculos, seleccion)
}

func nuevoRegistradorResultadoConVinculosV1(
	vinculos VinculosResultadoExperienciaV1,
	seleccion seleccionExperiencia,
) (*registradorResultadoExperienciaV1, error) {
	if err := validarVinculosResultadoV1(vinculos); err != nil {
		return nil, err
	}
	materializada, bloqueos, err := materializarSeleccionResultadoV1(seleccion)
	if err != nil {
		return nil, err
	}
	return &registradorResultadoExperienciaV1{
		vinculos:  vinculos,
		seleccion: materializada,
		bloqueos:  bloqueos,
	}, nil
}

func nuevosVinculosResultadoExperienciaV1(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
) (VinculosResultadoExperienciaV1, error) {
	if err := plan.Validar(); err != nil {
		return VinculosResultadoExperienciaV1{}, err
	}
	if err := entrada.Validar(); err != nil {
		return VinculosResultadoExperienciaV1{}, err
	}
	huellaEntrada, err := entrada.HuellaSHA256()
	if err != nil {
		return VinculosResultadoExperienciaV1{}, err
	}
	motor, err := vinculoMotorResultadoExperienciaV1()
	if err != nil {
		return VinculosResultadoExperienciaV1{}, err
	}
	huellaPlan, err := huellaPlanResultadoExperienciaV1(motor, plan.Conjunto())
	if err != nil {
		return VinculosResultadoExperienciaV1{}, err
	}
	resultado := VinculosResultadoExperienciaV1{
		motor: motor,
		plan: VinculoPlanResultadoExperienciaV1{
			esquema: esquemaPlanResultadoV1, huellaSHA256: huellaPlan,
		},
		conjunto: plan.Conjunto(),
		entrada: VinculoEntradaResultadoExperienciaV1{
			instantanea: entrada.Instantanea(), huellaContenidoSHA256: huellaEntrada,
		},
		fechaCorte: plan.FechaCorte(),
	}
	if err := validarVinculosResultadoV1(resultado); err != nil {
		return VinculosResultadoExperienciaV1{}, err
	}
	return resultado, nil
}

func materializarSeleccionResultadoV1(
	origen seleccionExperiencia,
) (SeleccionResultadoExperienciaV1, []BloqueoResultadoExperienciaV1, error) {
	if len(origen.aplicaciones) > maximoAplicacionesResultadoV1 ||
		len(origen.descartes) > maximoDescartesResultadoV1 ||
		len(origen.noCoincidencias) > maximoSinCoincidenciaResultadoV1 ||
		len(origen.bloqueos) > maximoBloqueosResultadoV1 {
		return SeleccionResultadoExperienciaV1{}, nil,
			nuevoError("resultado.seleccion", CodigoFueraDeLimites)
	}
	resultado := SeleccionResultadoExperienciaV1{
		aplicaciones:    make([]AplicacionSeleccionResultadoExperienciaV1, len(origen.aplicaciones)),
		descartes:       make([]DescarteSeleccionResultadoExperienciaV1, len(origen.descartes)),
		sinCoincidencia: make([]SinCoincidenciaResultadoExperienciaV1, len(origen.noCoincidencias)),
		evaluaciones:    uint64(origen.evaluaciones),
	}
	for indice, aplicacion := range origen.aplicaciones {
		resultado.aplicaciones[indice] = AplicacionSeleccionResultadoExperienciaV1{
			tramo: aplicacion.tramo, reglaClave: aplicacion.reglaClave,
			grupoClave: aplicacion.grupoClave, seccionClave: aplicacion.seccionClave,
			prioridad: aplicacion.prioridad,
			razon:     CodigoRazonResultadoExperienciaV1(aplicacion.razon),
		}
	}
	for indice, descarte := range origen.descartes {
		resultado.descartes[indice] = DescarteSeleccionResultadoExperienciaV1{
			tramo: descarte.tramo, reglaClave: descarte.reglaClave,
			grupoClave: descarte.grupoClave, reglaSeleccionada: descarte.reglaSeleccionada,
			razon: CodigoRazonResultadoExperienciaV1(descarte.razon),
		}
	}
	for indice, ausencia := range origen.noCoincidencias {
		resultado.sinCoincidencia[indice] = SinCoincidenciaResultadoExperienciaV1{
			tramo: ausencia.tramo, razon: CodigoRazonResultadoExperienciaV1(ausencia.razon),
		}
	}
	bloqueos := make([]BloqueoResultadoExperienciaV1, len(origen.bloqueos))
	for indice, bloqueo := range origen.bloqueos {
		bloqueos[indice] = BloqueoResultadoExperienciaV1{
			codigo:         CodigoBloqueoResultadoExperienciaV1(bloqueo.codigo),
			tramos:         []reglasbaremo.ReferenciaVersionada{bloqueo.tramo},
			reglas:         append([]string(nil), bloqueo.reglas...),
			claveGobernada: bloqueo.claveGobernada,
		}
	}
	if err := validarSeleccionResultadoV1(resultado); err != nil {
		return SeleccionResultadoExperienciaV1{}, nil, err
	}
	for _, bloqueo := range bloqueos {
		if err := validarBloqueoResultadoV1(bloqueo); err != nil {
			return SeleccionResultadoExperienciaV1{}, nil, err
		}
	}
	return resultado, bloqueos, nil
}

func (r *registradorResultadoExperienciaV1) registrarIntervalo(
	valor IntervaloAplicacionResultadoExperienciaV1,
) error {
	if r == nil || len(r.bloqueos) != 0 || len(r.intervalos) >= maximoAplicacionesResultadoV1 ||
		validarIntervaloResultadoV1(valor) != nil {
		return nuevoError("resultado.registrar_intervalo", CodigoValorInvalido)
	}
	r.intervalos = append(r.intervalos, valor)
	return nil
}

func (r *registradorResultadoExperienciaV1) registrarAplicacion(
	valor AplicacionCalculadaResultadoExperienciaV1,
) error {
	if r == nil || len(r.bloqueos) != 0 || len(r.aplicaciones) >= maximoAplicacionesResultadoV1 ||
		validarAplicacionCalculadaResultadoV1(valor) != nil {
		return nuevoError("resultado.registrar_aplicacion", CodigoValorInvalido)
	}
	r.aplicaciones = append(r.aplicaciones, valor)
	return nil
}

func (r *registradorResultadoExperienciaV1) registrarRegla(
	valor ResultadoReglaExperienciaV1,
) error {
	if r == nil || len(r.bloqueos) != 0 || len(r.reglas) >= maximoReglasResultadoV1 ||
		validarReglaResultadoV1(valor) != nil {
		return nuevoError("resultado.registrar_regla", CodigoValorInvalido)
	}
	r.reglas = append(r.reglas, valor)
	return nil
}

func (r *registradorResultadoExperienciaV1) registrarSeccion(
	valor SubtotalSeccionResultadoExperienciaV1,
) error {
	if r == nil || len(r.bloqueos) != 0 || len(r.secciones) >= maximoSeccionesResultadoV1 ||
		validarSeccionResultadoV1(valor) != nil {
		return nuevoError("resultado.registrar_seccion", CodigoValorInvalido)
	}
	r.secciones = append(r.secciones, valor)
	return nil
}

func (r *registradorResultadoExperienciaV1) registrarBloqueo(
	valor BloqueoResultadoExperienciaV1,
) error {
	if r == nil || len(r.bloqueos) >= maximoBloqueosResultadoV1 ||
		validarBloqueoResultadoV1(valor) != nil {
		return nuevoError("resultado.registrar_bloqueo", CodigoValorInvalido)
	}
	r.bloqueos = append(r.bloqueos, clonarBloqueosResultadoV1(
		[]BloqueoResultadoExperienciaV1{valor},
	)[0])
	return nil
}

func (r *registradorResultadoExperienciaV1) sellarBloqueado(
	fase FaseResultadoExperienciaV1,
) (ResultadoExperienciaV1, error) {
	if r == nil {
		return ResultadoExperienciaV1{}, nuevoError("resultado.registrador", CodigoValorInvalido)
	}
	resultado := r.materializar(ResultadoExperienciaBloqueado, fase, baremacion.Puntos{}, false)
	return sellarResultadoExperienciaV1(resultado)
}

func (r *registradorResultadoExperienciaV1) sellarCompletado(
	total baremacion.Puntos,
) (ResultadoExperienciaV1, error) {
	if r == nil || len(r.bloqueos) != 0 || !total.EsValido() {
		return ResultadoExperienciaV1{}, nuevoError("resultado.registrador", CodigoValorInvalido)
	}
	resultado := r.materializar(ResultadoExperienciaCompletado, FaseResultadoCompletado, total, true)
	return sellarResultadoExperienciaV1(resultado)
}

func (r *registradorResultadoExperienciaV1) materializar(
	estado EstadoResultadoExperienciaV1,
	fase FaseResultadoExperienciaV1,
	total baremacion.Puntos,
	tieneTotal bool,
) ResultadoExperienciaV1 {
	resultado := ResultadoExperienciaV1{
		estado: estado, fase: fase, vinculos: r.vinculos,
		seleccion:    clonarSeleccionResultadoV1(r.seleccion),
		intervalos:   append([]IntervaloAplicacionResultadoExperienciaV1(nil), r.intervalos...),
		aplicaciones: append([]AplicacionCalculadaResultadoExperienciaV1(nil), r.aplicaciones...),
		reglas:       append([]ResultadoReglaExperienciaV1(nil), r.reglas...),
		secciones:    append([]SubtotalSeccionResultadoExperienciaV1(nil), r.secciones...),
		total:        total, tieneTotal: tieneTotal, bloqueos: clonarBloqueosResultadoV1(r.bloqueos),
	}
	ordenarResultadoExperienciaV1(&resultado)
	return resultado
}

func ordenarResultadoExperienciaV1(resultado *ResultadoExperienciaV1) {
	if resultado == nil {
		return
	}
	sort.Slice(resultado.reglas, func(i, j int) bool {
		izquierda, derecha := resultado.reglas[i], resultado.reglas[j]
		return izquierda.seccionClave < derecha.seccionClave ||
			(izquierda.seccionClave == derecha.seccionClave && izquierda.reglaClave < derecha.reglaClave)
	})
	sort.Slice(resultado.secciones, func(i, j int) bool {
		return resultado.secciones[i].seccionClave < resultado.secciones[j].seccionClave
	})
	sort.Slice(resultado.bloqueos, func(i, j int) bool {
		return compararBloqueosResultadoV1(resultado.bloqueos[i], resultado.bloqueos[j]) < 0
	})
}

func sellarResultadoExperienciaV1(
	resultado ResultadoExperienciaV1,
) (ResultadoExperienciaV1, error) {
	if err := resultado.Validar(); err != nil {
		return ResultadoExperienciaV1{}, err
	}
	if _, err := resultado.RepresentacionCanonica(); err != nil {
		return ResultadoExperienciaV1{}, err
	}
	return resultado.clonar(), nil
}
