package application

import (
	"context"
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
	contexto    puertosbolsa.ResolutorContextoSituacionParticipacion
	autorizador puertosbolsa.AutorizadorSituacionParticipacionV3
	repositorio puertosbolsa.RepositorioEmisionLlamamiento
	correos     puertosbolsa.FuenteCorreoParticipacion
	emisor      puertosbolsa.EmisorCorreoBolsa
	contactos   puertosbolsa.RegistradorContactoEmision
	reloj       func() time.Time
}

func NuevoServicioEmisionLlamamiento(c puertosbolsa.ResolutorContextoSituacionParticipacion, a puertosbolsa.AutorizadorSituacionParticipacionV3, r puertosbolsa.RepositorioEmisionLlamamiento, f puertosbolsa.FuenteCorreoParticipacion, e puertosbolsa.EmisorCorreoBolsa, contactos puertosbolsa.RegistradorContactoEmision, reloj func() time.Time) (*ServicioEmisionLlamamiento, error) {
	if c == nil || a == nil || r == nil || f == nil || e == nil || contactos == nil || reloj == nil {
		return nil, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	return &ServicioEmisionLlamamiento{c, a, r, f, e, contactos, reloj}, nil
}

func (s *ServicioEmisionLlamamiento) EmitirLlamamiento(ctx context.Context, q puertosbolsa.SolicitudEmitirLlamamiento) (puertosbolsa.EmisionLlamamiento, error) {
	if ctx == nil || s == nil || validarSolicitudEmision(q) != nil {
		return puertosbolsa.EmisionLlamamiento{}, puertosbolsa.ErrEmisionLlamamientoInvalida
	}
	actor := q.ResultadoContexto.Contexto
	resuelto, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, actor, q.BolsaRef, q.Participaciones[0])
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
	emision, err := s.repositorio.Emitir(ctx, puertosbolsa.ComandoEmitirLlamamiento{LlamamientoRef: "llamamiento:" + sufijo, ReciboRef: "recibo:llamamiento:" + sufijo, BolsaRef: q.BolsaRef, ActorRef: actor.PersonaRef, ClaveIdempotencia: q.ClaveIdempotencia, Participaciones: append([]string(nil), q.Participaciones...), Configuracion: q.Configuracion, EmitidoEn: ahora, SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material})
	if err != nil || emision.Reutilizada {
		return emision, err
	}
	for i, participacion := range emision.Participaciones {
		resultado := "no_enviado"
		correo, correoErr := s.correos.CorreoParticipacion(ctx, participacion)
		messageID := fmt.Sprintf("<%s-%d@vec.dipgra.local>", sufijo, i+1)
		if correoErr == nil && s.emisor.EnviarCorreo(ctx, correo, q.Configuracion.Asunto, q.Configuracion.Cuerpo, messageID, emision.EmitidoEn) {
			resultado = "enviado"
		}
		contacto, contactoErr := s.contactos.RegistrarContactoParticipacion(ctx, puertosbolsa.SolicitudRegistrarContactoParticipacion{Vinculo: q.Vinculo, ResultadoContexto: q.ResultadoContexto, BolsaRef: q.BolsaRef, ParticipacionRef: participacion, LlamamientoRef: emision.LlamamientoRef, Canal: "correo", Instante: emision.EmitidoEn, Resultado: resultado, Anotacion: "Emisión de llamamiento por plantilla " + q.Configuracion.PlantillaVersion, ClaveIdempotencia: q.ClaveIdempotencia + fmt.Sprintf(":correo:%d", i+1), Correlacion: q.Correlacion, MotivoAutorizacion: q.MotivoAutorizacion})
		if contactoErr != nil {
			return puertosbolsa.EmisionLlamamiento{}, contactoErr
		}
		emision.Contactos = append(emision.Contactos, puertosbolsa.ResultadoContactoEmision{ParticipacionRef: participacion, Resultado: resultado, ReciboRef: contacto.ReciboRef})
	}
	return emision, nil
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
	return nil
}

func esConflictoEmision(err error) bool {
	return errors.Is(err, puertosbolsa.ErrEmisionLlamamientoConflicto)
}
