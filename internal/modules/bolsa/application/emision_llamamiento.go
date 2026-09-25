package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type ServicioEmisionLlamamiento struct {
	contextoBolsa puertosbolsa.ResolutorContextoContactoParticipacion
	autorizador   puertosbolsa.AutorizadorSituacionParticipacionV3
	repositorio   puertosbolsa.RepositorioEmisionLlamamiento
	correos       puertosbolsa.FuenteCorreoParticipacion
	emisor        puertosbolsa.EmisorCorreoBolsa
	reloj         func() time.Time
	// origenes es opcional: sin él no se avisa del contacto no confirmado.
	origenes puertosbolsa.FuenteOrigenContactoParticipacion
}

func NuevoServicioEmisionLlamamiento(cb puertosbolsa.ResolutorContextoContactoParticipacion, a puertosbolsa.AutorizadorSituacionParticipacionV3, r puertosbolsa.RepositorioEmisionLlamamiento, f puertosbolsa.FuenteCorreoParticipacion, e puertosbolsa.EmisorCorreoBolsa, reloj func() time.Time) (*ServicioEmisionLlamamiento, error) {
	if cb == nil || a == nil || r == nil || f == nil || e == nil || reloj == nil {
		return nil, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	return &ServicioEmisionLlamamiento{contextoBolsa: cb, autorizador: a, repositorio: r, correos: f, emisor: e, reloj: reloj}, nil
}

// EstablecerAvisoContactoNoConfirmado habilita el aviso a RRHH cuando el
// contacto de una persona llamada es de origen CONVOCA y ya ha vencido.
func (s *ServicioEmisionLlamamiento) EstablecerAvisoContactoNoConfirmado(f puertosbolsa.FuenteOrigenContactoParticipacion) error {
	if s == nil || f == nil {
		return puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	s.origenes = f
	return nil
}

func (s *ServicioEmisionLlamamiento) EmitirLlamamiento(ctx context.Context, q puertosbolsa.SolicitudEmitirLlamamiento) (puertosbolsa.EmisionLlamamiento, error) {
	if ctx == nil || s == nil || validarSolicitudEmision(q) != nil {
		return puertosbolsa.EmisionLlamamiento{}, puertosbolsa.ErrEmisionLlamamientoInvalida
	}
	actor := q.ResultadoContexto.Contexto
	resuelto, err := s.contextoBolsa.ResolverContextoContactosBolsa(ctx, actor, q.BolsaRef)
	if err != nil || resuelto.Validar() != nil {
		return puertosbolsa.EmisionLlamamiento{}, errorDependenciaSituacion(err)
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: q.BolsaRef, ModuloID: puertosbolsa.ModuloBorradorLlamamiento, Tipo: puertosbolsa.TipoRecursoEmision, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion, Accion: puertosbolsa.AccionEmitirLlamamiento, Recurso: recurso, Finalidad: puertosbolsa.FinalidadEmitirLlamamiento, Correlacion: q.Correlacion})
	if err != nil {
		return puertosbolsa.EmisionLlamamiento{}, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, q.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		return puertosbolsa.EmisionLlamamiento{}, errorDependenciaSituacion(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, q.ResultadoContexto, q.MotivoAutorizacion, material, puertosbolsa.AudienciaEmitirLlamamiento) {
		return puertosbolsa.EmisionLlamamiento{}, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	h := sha256.Sum256([]byte(q.BolsaRef + "\x1f" + q.ClaveIdempotencia))
	sufijo := hex.EncodeToString(h[:])
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	tokenFinalizacion := make([]byte, 32)
	if _, err = rand.Read(tokenFinalizacion); err != nil {
		return puertosbolsa.EmisionLlamamiento{}, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	reservada, err := s.repositorio.Reservar(ctx, puertosbolsa.ComandoEmitirLlamamiento{LlamamientoRef: "llamamiento:" + sufijo, ReciboRef: "recibo:llamamiento:" + sufijo, BolsaRef: q.BolsaRef, ActorRef: actor.PersonaRef, ClaveIdempotencia: q.ClaveIdempotencia, Participaciones: append([]string(nil), q.Participaciones...), Configuracion: q.Configuracion, EmitidoEn: ahora, SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material, TokenFinalizacion: tokenFinalizacion})
	if err != nil {
		return puertosbolsa.EmisionLlamamiento{}, err
	}
	if !mismaSolicitudEmision(reservada, q) {
		return puertosbolsa.EmisionLlamamiento{}, puertosbolsa.ErrEmisionLlamamientoConflicto
	}
	if reservada.Reutilizada {
		if len(reservada.Contactos) == len(q.Participaciones) {
			return s.conAvisosContacto(ctx, reservada), nil
		}
		return puertosbolsa.EmisionLlamamiento{}, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	contactos := make([]puertosbolsa.ResultadoContactoEmision, 0, len(q.Participaciones))
	for i, participacion := range q.Participaciones {
		resultado := "no_enviado"
		correo, correoErr := s.correos.CorreoParticipacion(ctx, participacion)
		messageID := fmt.Sprintf("<%s-%d@vec.dipgra.local>", sufijo, i+1)
		if correoErr == nil && s.emisor.EnviarCorreo(ctx, correo, q.Configuracion.Asunto, q.Configuracion.Cuerpo, messageID, ahora) {
			resultado = "enviado"
		}
		reciboContacto := sha256.Sum256([]byte(q.BolsaRef + "\x1f" + q.ClaveIdempotencia + "\x1f" + participacion))
		contactos = append(contactos, puertosbolsa.ResultadoContactoEmision{ParticipacionRef: participacion, Resultado: resultado, ReciboRef: "recibo:contacto:" + hex.EncodeToString(reciboContacto[:])})
	}
	emitido, err := s.repositorio.RegistrarContactos(ctx, q.BolsaRef, q.ClaveIdempotencia, actor.PersonaRef, tokenFinalizacion, contactos)
	if err != nil {
		return emitido, err
	}
	return s.conAvisosContacto(ctx, emitido), nil
}

func (s *ServicioEmisionLlamamiento) RecuperarLlamamiento(ctx context.Context, q puertosbolsa.SolicitudRecuperarLlamamiento) (puertosbolsa.EmisionLlamamiento, error) {
	if ctx == nil || s == nil || q.ContextoActor.PersonaRef == "" || q.BolsaRef == "" || q.ClaveIdempotencia == "" {
		return puertosbolsa.EmisionLlamamiento{}, puertosbolsa.ErrEmisionLlamamientoInvalida
	}
	resuelto, err := s.contextoBolsa.ResolverContextoContactosBolsa(ctx, q.ContextoActor, q.BolsaRef)
	if err != nil || resuelto.Validar() != nil {
		return puertosbolsa.EmisionLlamamiento{}, errorDependenciaSituacion(err)
	}
	recuperado, err := s.repositorio.Recuperar(ctx, q.BolsaRef, q.ClaveIdempotencia)
	if err != nil {
		return recuperado, err
	}
	return s.conAvisosContacto(ctx, recuperado), nil
}

// conAvisosContacto añade, a la hora de la respuesta, un aviso por cada
// persona cuyo contacto de origen CONVOCA ha vencido sin confirmarse. Un
// origen que no se puede leer se avisa como no comprobado: nunca se da por
// confirmado. El llamamiento no se bloquea.
func (s *ServicioEmisionLlamamiento) conAvisosContacto(ctx context.Context, emision puertosbolsa.EmisionLlamamiento) puertosbolsa.EmisionLlamamiento {
	emision.AvisosContacto = nil
	if s.origenes == nil {
		return emision
	}
	ahora := s.reloj().UTC()
	for _, participacion := range emision.Participaciones {
		marca, err := s.origenes.OrigenContactoParticipacion(ctx, participacion)
		switch {
		case err != nil:
			emision.AvisosContacto = append(emision.AvisosContacto, puertosbolsa.AvisoContactoEmision{ParticipacionRef: participacion, Aviso: puertosbolsa.AvisoEstadoContactoNoDisponible})
		case marca != nil && marca.Vencida(ahora):
			emision.AvisosContacto = append(emision.AvisosContacto, puertosbolsa.AvisoContactoEmision{ParticipacionRef: participacion, Aviso: puertosbolsa.AvisoContactoNoConfirmado, UltimoDia: marca.UltimoDia})
		}
	}
	return emision
}

func mismaSolicitudEmision(previa puertosbolsa.EmisionLlamamiento, q puertosbolsa.SolicitudEmitirLlamamiento) bool {
	if previa.BolsaRef != q.BolsaRef || previa.Configuracion != q.Configuracion || len(previa.Participaciones) != len(q.Participaciones) {
		return false
	}
	for i := range q.Participaciones {
		if previa.Participaciones[i] != q.Participaciones[i] {
			return false
		}
	}
	return true
}

func validarSolicitudEmision(q puertosbolsa.SolicitudEmitirLlamamiento) error {
	if q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil || q.BolsaRef == "" || len(q.Participaciones) < 1 || len(q.Participaciones) > 100 || q.ClaveIdempotencia == "" || q.Correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(q.MotivoAutorizacion) {
		return puertosbolsa.ErrEmisionLlamamientoInvalida
	}
	vistos := map[string]bool{}
	for _, p := range q.Participaciones {
		if strings.TrimSpace(p) != p || p == "" || vistos[p] {
			return puertosbolsa.ErrEmisionLlamamientoInvalida
		}
		vistos[p] = true
	}
	c := q.Configuracion
	for _, v := range []string{c.Referencia, c.Descripcion, c.Categoria, c.Centro, c.Modalidad, c.FechaInicio, c.Plazo, c.PlantillaVersion, c.Asunto, c.Cuerpo} {
		if strings.TrimSpace(v) != v || len(v) < 2 || len(v) > 4000 {
			return puertosbolsa.ErrEmisionLlamamientoInvalida
		}
	}
	if c.PlantillaVersion != puertosbolsa.PlantillaCorreoLlamamiento {
		return puertosbolsa.ErrEmisionLlamamientoInvalida
	}
	return nil
}

func esConflictoEmision(err error) bool {
	return errors.Is(err, puertosbolsa.ErrEmisionLlamamientoConflicto)
}
