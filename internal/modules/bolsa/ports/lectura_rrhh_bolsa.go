package ports

// SituacionBolsaRRHH conserva el número de fila del acta junto al estado leído
// en B92. La composición comprueba ese vínculo antes de recuperar identidades.
type SituacionBolsaRRHH struct {
	SituacionResumenParticipacion
	FilaNumero int
}
