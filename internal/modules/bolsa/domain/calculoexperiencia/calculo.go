package calculoexperiencia

// CalcularExperienciaV1 ejecuta de extremo a extremo el contrato puro del
// motor. No decide si el plan esta administrativamente activo: esa garantia
// corresponde al caso de uso oficial, mientras que una simulacion puede usar
// un borrador compilable. Ante un fallo tecnico nunca devuelve material
// parcial; los impedimentos de negocio forman un resultado bloqueado,
// canonico y explicable.
func CalcularExperienciaV1(
	plan PlanExperiencia,
	entrada EntradaExperiencia,
) (ResultadoExperienciaV1, error) {
	seleccion, err := seleccionarAplicaciones(plan, entrada)
	if err != nil {
		return ResultadoExperienciaV1{}, err
	}
	if err := comprobarPresupuestoSalidaResultadoV1(plan, seleccion); err != nil {
		return ResultadoExperienciaV1{}, err
	}
	registrador, err := nuevoRegistradorResultadoExperienciaV1(plan, entrada, seleccion)
	if err != nil {
		return ResultadoExperienciaV1{}, err
	}
	if seleccion.bloqueada() {
		return registrador.sellarBloqueado(FaseResultadoSeleccion)
	}

	temporales, err := resolverAplicacionesTemporales(plan, entrada, seleccion)
	if err != nil {
		return ResultadoExperienciaV1{}, err
	}
	if temporales.bloqueada() {
		if err := registrarBloqueoTemporalResultadoV1(
			registrador, plan, entrada, seleccion, temporales,
		); err != nil {
			return ResultadoExperienciaV1{}, err
		}
		return registrador.sellarBloqueado(FaseResultadoIntervalos)
	}

	puntuacion, err := puntuarExperienciaV1(plan, entrada, seleccion, temporales)
	if err != nil {
		return ResultadoExperienciaV1{}, err
	}
	if err := registrarPuntuacionExperienciaV1(registrador, puntuacion); err != nil {
		return ResultadoExperienciaV1{}, err
	}
	if puntuacion.bloqueada() {
		return registrador.sellarBloqueado(FaseResultadoPuntuacion)
	}
	return registrador.sellarCompletado(puntuacion.total)
}
