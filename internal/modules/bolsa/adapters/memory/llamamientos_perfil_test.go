package memory

// PerfilRegistroPropuestasSoloPruebas solo existe durante la compilacion de
// tests. Un binario productivo no puede obtener la capacidad que habilita el
// registro efimero de propuestas de llamamiento.
func PerfilRegistroPropuestasSoloPruebas() PerfilUsoRegistroPropuestasMemoria {
	return PerfilUsoRegistroPropuestasMemoria{soloPruebas: true}
}
