package diagnostico

type EtapaConsultaRRHH string

const (
	EtapaSerializacion       EtapaConsultaRRHH = "serializacion"
	EtapaSQL                 EtapaConsultaRRHH = "sql"
	EtapaResultadoSQL        EtapaConsultaRRHH = "resultado_sql"
	EtapaCursorDecod         EtapaConsultaRRHH = "cursor_decodificacion"
	EtapaCursorHuella        EtapaConsultaRRHH = "cursor_huella"
	EtapaPaginaInterna       EtapaConsultaRRHH = "pagina_interna"
	EtapaCapacidad           EtapaConsultaRRHH = "capacidad"
	EtapaOrden               EtapaConsultaRRHH = "orden"
	EtapaReloj               EtapaConsultaRRHH = "reloj"
	EtapaPagina              EtapaConsultaRRHH = "pagina"
	EtapaPublicable          EtapaConsultaRRHH = "publicable"
	EtapaAplicacion          EtapaConsultaRRHH = "aplicacion"
	EtapaContinuidadCursor   EtapaConsultaRRHH = "continuidad_cursor"
	EtapaCanalContinuidad    EtapaConsultaRRHH = "canal_continuidad"
	EtapaSesionRevalidada    EtapaConsultaRRHH = "sesion_revalidada"
	EtapaSesionPrecondicion  EtapaConsultaRRHH = "sesion_precondicion"
	EtapaSesionVinculoPrevio EtapaConsultaRRHH = "sesion_vinculo_previo"
	EtapaSesionRevalidador   EtapaConsultaRRHH = "sesion_revalidador"
	EtapaSesionContextoActor EtapaConsultaRRHH = "sesion_contexto_actor"
	// Salidas finas de la revalidación de sesión del cursor.
	EtapaSesionVinculoCreacion    EtapaConsultaRRHH = "sesion_vinculo_creacion"
	EtapaSesionIdentidadDistinta  EtapaConsultaRRHH = "sesion_identidad_distinta"
	EtapaSesionVinculoIncoherente EtapaConsultaRRHH = "sesion_vinculo_incoherente"
	EtapaSesionContextoInvalido   EtapaConsultaRRHH = "sesion_contexto_invalido"
	EtapaActorContexto            EtapaConsultaRRHH = "actor_contexto"
	EtapaAutoridadContexto        EtapaConsultaRRHH = "autoridad_contexto"
)

// La causa permanece accesible a errors.Is/As, pero nunca forma parte del texto.
type FalloConsultaRRHH struct {
	Etapa     EtapaConsultaRRHH
	Sentinela error
	Causa     error
	CodigoSQL string
}

func (e *FalloConsultaRRHH) Error() string { return "consulta RRHH: " + e.EtapaSegura() }
func (e *FalloConsultaRRHH) Unwrap() []error {
	causas := make([]error, 0, 2)
	if e.Sentinela != nil {
		causas = append(causas, e.Sentinela)
	}
	if e.Causa != nil {
		causas = append(causas, e.Causa)
	}
	return causas
}
func (e *FalloConsultaRRHH) EtapaSegura() string {
	switch e.Etapa {
	case EtapaSerializacion, EtapaSQL, EtapaResultadoSQL, EtapaCursorDecod, EtapaCursorHuella,
		EtapaPaginaInterna, EtapaCapacidad, EtapaOrden, EtapaReloj, EtapaPagina,
		EtapaPublicable, EtapaAplicacion, EtapaContinuidadCursor,
		EtapaCanalContinuidad, EtapaSesionRevalidada, EtapaActorContexto,
		EtapaSesionPrecondicion, EtapaSesionVinculoPrevio,
		EtapaSesionRevalidador, EtapaSesionContextoActor, EtapaSesionVinculoCreacion,
		EtapaSesionIdentidadDistinta, EtapaSesionVinculoIncoherente, EtapaSesionContextoInvalido,
		EtapaAutoridadContexto:
		return string(e.Etapa)
	default:
		return "desconocida"
	}
}
