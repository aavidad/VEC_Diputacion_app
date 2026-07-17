package calculoexperiencia

// El presupuesto de salida impide calcular o duplicar en memoria una
// explicacion que no podria superar el limite canonico de 64 MiB. Las cotas
// incluyen nombres JSON y la repeticion deliberada de referencias entre
// seleccion, intervalos y calculos. Se apoyan en los maximos V1: referencias
// de 512 caracteres, claves de 128 y racionales exactos de 2.500.
const (
	reservaBaseSalidaV1             uint64 = 64 * 1024
	costeSeleccionAplicacionV1      uint64 = 2 * 1024
	costeSeleccionDescarteV1        uint64 = 3 * 1024
	costeSeleccionSinCoincidenciaV1 uint64 = 2 * 1024
	costeSeleccionBloqueoV1         uint64 = 160 * 1024
	costeAplicacionPuntuadaV1       uint64 = 24 * 1024
	costeBloqueoPuntuacionV1        uint64 = 8 * 1024
	costeReglaPuntuadaV1            uint64 = 40 * 1024
	costeSeccionPuntuadaV1          uint64 = 16 * 1024
)

type presupuestoSalidaCalculoV1 struct {
	restantes uint64
}

func nuevoPresupuestoSalidaCalculoV1() presupuestoSalidaCalculoV1 {
	return presupuestoSalidaCalculoV1{restantes: uint64(maximoBytesResultadoV1)}
}

func (p *presupuestoSalidaCalculoV1) consumir(
	elementos int,
	coste uint64,
) error {
	if p == nil || elementos < 0 || coste == 0 {
		return errorPresupuestoSalidaV1()
	}
	cantidad := uint64(elementos)
	if cantidad > p.restantes/coste {
		return errorPresupuestoSalidaV1()
	}
	p.restantes -= cantidad * coste
	return nil
}

func comprobarPresupuestoSalidaPuntuacionV1(
	aplicaciones int,
	reglas int,
	secciones int,
) error {
	presupuesto := nuevoPresupuestoSalidaCalculoV1()
	if err := presupuesto.consumir(1, reservaBaseSalidaV1); err != nil {
		return err
	}
	if err := presupuesto.consumir(aplicaciones, costeAplicacionPuntuadaV1); err != nil {
		return err
	}
	if err := presupuesto.consumir(aplicaciones, costeBloqueoPuntuacionV1); err != nil {
		return err
	}
	if err := presupuesto.consumir(reglas, costeReglaPuntuadaV1); err != nil {
		return err
	}
	return presupuesto.consumir(secciones, costeSeccionPuntuadaV1)
}

func comprobarPresupuestoSalidaResultadoV1(
	plan PlanExperiencia,
	seleccion seleccionExperiencia,
) error {
	presupuesto := nuevoPresupuestoSalidaCalculoV1()
	if err := presupuesto.consumir(1, reservaBaseSalidaV1); err != nil {
		return err
	}
	if err := presupuesto.consumir(
		len(seleccion.aplicaciones), costeSeleccionAplicacionV1,
	); err != nil {
		return err
	}
	if err := presupuesto.consumir(
		len(seleccion.descartes), costeSeleccionDescarteV1,
	); err != nil {
		return err
	}
	if err := presupuesto.consumir(
		len(seleccion.noCoincidencias), costeSeleccionSinCoincidenciaV1,
	); err != nil {
		return err
	}
	if seleccion.bloqueada() {
		return presupuesto.consumir(len(seleccion.bloqueos), costeSeleccionBloqueoV1)
	}
	if err := presupuesto.consumir(
		len(seleccion.aplicaciones), costeAplicacionPuntuadaV1,
	); err != nil {
		return err
	}
	if err := presupuesto.consumir(
		len(seleccion.aplicaciones), costeBloqueoPuntuacionV1,
	); err != nil {
		return err
	}
	if err := presupuesto.consumir(len(plan.reglas), costeReglaPuntuadaV1); err != nil {
		return err
	}
	return presupuesto.consumir(len(plan.secciones), costeSeccionPuntuadaV1)
}

func errorPresupuestoSalidaV1() error {
	return nuevoError("calculo.presupuesto_salida", CodigoFueraDeLimites)
}
