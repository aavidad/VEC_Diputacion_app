package application

import (
	"context"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Límites de la consulta a «Mis correos» durante una emisión. La consulta
// nunca bloquea el llamamiento: agotado el presupuesto, o si una consulta
// tarda o falla, el aviso sale al correo del alta y se anota por qué.
const (
	presupuestoCorreoAvisosEmision = 10 * time.Second
	limiteConsultaCorreoAvisos     = 2 * time.Second
)

// AvisadorLlamamiento decide a qué correo va el aviso de cada participación
// y lo envía una sola vez. Sin B59 usa siempre el correo del alta en la
// bolsa, como antes; con B59 usa el correo activo de «Mis correos» cuando la
// persona lo tiene y deja constancia de la fuente.
type AvisadorLlamamiento struct {
	correos    puertosbolsa.FuenteCorreoParticipacion
	emisor     puertosbolsa.EmisorCorreoBolsa
	candidatos puertosbolsa.LectorCandidatoParticipacion
	fuente     puertosbolsa.FuenteCorreoAvisoPersona
}

func NuevoAvisadorLlamamiento(correos puertosbolsa.FuenteCorreoParticipacion, emisor puertosbolsa.EmisorCorreoBolsa) (*AvisadorLlamamiento, error) {
	if nulo(correos) || nulo(emisor) {
		return nil, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	return &AvisadorLlamamiento{correos: correos, emisor: emisor}, nil
}

// EstablecerCorreoAvisosPersona activa B59. Debe llamarse durante la
// composición, antes de atender peticiones.
func (a *AvisadorLlamamiento) EstablecerCorreoAvisosPersona(l puertosbolsa.LectorCandidatoParticipacion, f puertosbolsa.FuenteCorreoAvisoPersona) error {
	if a == nil || nulo(l) || nulo(f) {
		return puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	a.candidatos, a.fuente = l, f
	return nil
}

func (a *AvisadorLlamamiento) conMisCorreos() bool {
	return a != nil && a.candidatos != nil && a.fuente != nil
}

// AvisoLlamamiento reúne lo que hace falta para avisar a una participación.
// La identidad de quien emite sirve para autorizar la lectura del correo.
type AvisoLlamamiento struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	BolsaRef           string
	UnidadRef          string
	AmbitoRef          string
	LlamamientoRef     string
	ParticipacionRef   string
	Asunto             string
	Cuerpo             string
	MessageID          string
	Instante           time.Time
}

// Presupuesto devuelve el contexto común de consultas de una emisión.
func (a *AvisadorLlamamiento) Presupuesto(ctx context.Context) (context.Context, context.CancelFunc) {
	if !a.conMisCorreos() {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, presupuestoCorreoAvisosEmision)
}

// Avisar devuelve el resultado del relay («enviado» o «no_enviado») y la
// constancia de la fuente, que es nil cuando B59 no está activo.
func (a *AvisadorLlamamiento) Avisar(ctx, presupuesto context.Context, aviso AvisoLlamamiento) (string, *puertosbolsa.FuenteCorreoContacto) {
	enviar := func(destino string) bool {
		return destino != "" && a.emisor.EnviarCorreo(ctx, destino, aviso.Asunto, aviso.Cuerpo, aviso.MessageID, aviso.Instante)
	}
	resultado := func(enviado bool) string {
		if enviado {
			return "enviado"
		}
		return "no_enviado"
	}
	var fuente *puertosbolsa.FuenteCorreoContacto
	if a.conMisCorreos() {
		motivo, correoRef, enviado := a.consultarMisCorreos(ctx, presupuesto, aviso, enviar)
		if motivo == puertosbolsa.MotivoFuenteCorreoActivo {
			return resultado(enviado), &puertosbolsa.FuenteCorreoContacto{Fuente: puertosbolsa.FuenteCorreoMisCorreos, Motivo: motivo, CorreoRef: correoRef}
		}
		fuente = &puertosbolsa.FuenteCorreoContacto{Fuente: puertosbolsa.FuenteCorreoAltaBolsa, Motivo: motivo}
	}
	correo, err := a.correos.CorreoParticipacion(ctx, aviso.ParticipacionRef)
	return resultado(err == nil && enviar(correo)), fuente
}

// consultarMisCorreos pide a Usuarios el correo activo. Si lo hay, el envío
// ocurre dentro de la llamada y la dirección no sale de ella. Cualquier otro
// resultado deja el aviso para el correo del alta con su motivo.
func (a *AvisadorLlamamiento) consultarMisCorreos(ctx, presupuesto context.Context, aviso AvisoLlamamiento, enviar func(string) bool) (string, string, bool) {
	if presupuesto.Err() != nil || ctx.Err() != nil {
		return puertosbolsa.MotivoFuenteMisCorreosNoDisponible, "", false
	}
	consulta, cancelar := context.WithTimeout(presupuesto, limiteConsultaCorreoAvisos)
	defer cancelar()
	candidato, err := a.candidatos.CandidatoParticipacionAvisos(consulta, aviso.BolsaRef, aviso.LlamamientoRef, aviso.ParticipacionRef)
	if err != nil {
		return puertosbolsa.MotivoFuenteMisCorreosNoDisponible, "", false
	}
	if candidato == "" {
		return puertosbolsa.MotivoFuenteSinPersonaVinculada, "", false
	}
	enviado, llamado := false, false
	encontrado, correoRef, err := a.fuente.ConCorreoAvisoPersona(consulta, puertosbolsa.SolicitudCorreoAvisoPersona{
		Vinculo: aviso.Vinculo, ResultadoContexto: aviso.ResultadoContexto, MotivoAutorizacion: aviso.MotivoAutorizacion,
		BolsaRef: aviso.BolsaRef, UnidadRef: aviso.UnidadRef, AmbitoRef: aviso.AmbitoRef,
		LlamamientoRef: aviso.LlamamientoRef, CandidatoRef: candidato,
	}, func(destino string) {
		if llamado {
			return
		}
		llamado = true
		enviado = enviar(destino)
	})
	switch {
	case llamado:
		// Usuarios entregó la dirección y se intentó el envío: no se repite
		// al correo del alta aunque el relay no lo aceptara.
		return puertosbolsa.MotivoFuenteCorreoActivo, correoRef, enviado
	case err == nil && !encontrado:
		return puertosbolsa.MotivoFuenteSinCorreoActivo, "", false
	default:
		return puertosbolsa.MotivoFuenteMisCorreosNoDisponible, "", false
	}
}

// EstablecerCorreoAvisosPersona activa B59 en la emisión: los avisos irán al
// correo activo de «Mis correos» cuando la persona lo tenga, y cada contacto
// guardará la fuente usada. Sin esta llamada la emisión se comporta como antes.
func (s *ServicioEmisionLlamamiento) EstablecerCorreoAvisosPersona(l puertosbolsa.LectorCandidatoParticipacion, f puertosbolsa.FuenteCorreoAvisoPersona) error {
	if s == nil || s.avisador == nil {
		return puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	return s.avisador.EstablecerCorreoAvisosPersona(l, f)
}
