package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ServicioContactoParticipacion struct {
	contexto    puertosbolsa.ResolutorContextoContactoParticipacion
	autorizador puertosbolsa.AutorizadorSituacionParticipacionV3
	repositorio puertosbolsa.RepositorioContactoParticipacion
	intentos    *controlIntentosContacto
}

func NuevoServicioContactoParticipacion(c puertosbolsa.ResolutorContextoContactoParticipacion, a puertosbolsa.AutorizadorSituacionParticipacionV3, r puertosbolsa.RepositorioContactoParticipacion) (*ServicioContactoParticipacion, error) {
	if c == nil || a == nil || r == nil {
		return nil, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	return &ServicioContactoParticipacion{contexto: c, autorizador: a, repositorio: r}, nil
}

func (s *ServicioContactoParticipacion) RegistrarContactoParticipacion(ctx context.Context, solicitud puertosbolsa.SolicitudRegistrarContactoParticipacion) (puertosbolsa.RegistroContactoParticipacion, error) {
	if ctx == nil || s == nil || solicitud.Validar() != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	actor := solicitud.ResultadoContexto.Contexto
	resuelto, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, actor, solicitud.BolsaRef, solicitud.ParticipacionRef)
	if err != nil || resuelto.Validar() != nil || actor.PersonaRef == "" {
		if errors.Is(err, dominiovec.ErrAutorizacionDenegada) || errors.Is(err, dominiovec.ErrPermissionDenied) {
			return puertosbolsa.RegistroContactoParticipacion{}, err
		}
		return puertosbolsa.RegistroContactoParticipacion{}, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	pertenece, err := s.repositorio.ParticipacionPerteneceABolsa(ctx, solicitud.BolsaRef, solicitud.ParticipacionRef)
	if err != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, err
	}
	if !pertenece {
		return puertosbolsa.RegistroContactoParticipacion{}, dominiovec.ErrAutorizacionDenegada
	}
	h := sha256.Sum256([]byte(solicitud.ParticipacionRef + "\x1f" + solicitud.ClaveIdempotencia))
	sufijo := hex.EncodeToString(h[:])
	instante := time.Time{}
	if !solicitud.InstanteServidor {
		instante = solicitud.Instante.UTC().Truncate(time.Microsecond)
	}
	contacto := dominiobolsa.ContactoParticipacion{ContactoRef: "contacto:" + sufijo, BolsaRef: solicitud.BolsaRef, ParticipacionRef: solicitud.ParticipacionRef, LlamamientoRef: solicitud.LlamamientoRef, OfertaRef: solicitud.OfertaRef, EvidenciaRef: solicitud.EvidenciaRef, EvidenciaHuellaSHA256: solicitud.EvidenciaHuellaSHA256, Canal: solicitud.Canal, Instante: instante, InstanteServidor: solicitud.InstanteServidor, Actor: actor.PersonaRef, Resultado: solicitud.Resultado, Anotacion: solicitud.Anotacion}
	if err := contacto.Validar(); err != nil {
		if errors.Is(err, dominiobolsa.ErrAnotacionContactoDatoPersonal) {
			return puertosbolsa.RegistroContactoParticipacion{}, err
		}
		return puertosbolsa.RegistroContactoParticipacion{}, dominiobolsa.ErrContactoParticipacionInvalido
	}
	intento, err := s.prepararIntento(ctx, contacto)
	if err != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, err
	}
	// «Comunica» solo se admite si las reglas vigentes lo cuentan como intento
	// sin contacto: con otras reglas contaría como persona localizada.
	if !resultadoAdmitidoPorReglas(intento, contacto.Resultado) {
		return puertosbolsa.RegistroContactoParticipacion{}, dominiobolsa.ErrContactoParticipacionInvalido
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: solicitud.ParticipacionRef, ModuloID: puertosbolsa.ModuloSituacionParticipacion, Tipo: puertosbolsa.TipoRecursoSituacionParticipacion, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: solicitud.Vinculo, ReferenciaMotivo: solicitud.MotivoAutorizacion, Accion: puertosbolsa.AccionRegistrarContactoParticipacion, Recurso: recurso, Finalidad: puertosbolsa.FinalidadRegistrarContactoParticipacion, Correlacion: solicitud.Correlacion})
	if err != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, solicitud.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, errorDependenciaSituacion(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, solicitud.ResultadoContexto, solicitud.MotivoAutorizacion, material, puertosbolsa.AudienciaRegistrarContactoParticipacion) {
		return puertosbolsa.RegistroContactoParticipacion{}, errorDependenciaSituacion(err)
	}
	comando := puertosbolsa.ComandoRegistrarContactoParticipacion{Contacto: contacto, ClaveIdempotencia: solicitud.ClaveIdempotencia, ReciboRef: "recibo:contacto:" + sufijo, SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material, InstanteServidor: solicitud.InstanteServidor}
	if intento != nil {
		politica := intento.politica
		comando.ControlIntentos = &politica
		comando.FechaDiaHabil = intento.fechaDiaHabil
		comando.DiaHabil = intento.diaHabil
	}
	registro, err := s.repositorio.RegistrarContacto(ctx, comando)
	if err != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, err
	}
	// La llamada ya está confirmada: en un registro nuevo SQL comprobó que su
	// fecha es la del calendario consultado antes. Solo en una repetición de un
	// día anterior se vuelve a mirar el calendario, y si falla se conserva el
	// valor previo: un aviso de franja nunca convierte en error lo ya guardado.
	if solicitud.InstanteServidor && registro.Reutilizado && intento != nil && intento.politica.Franja.Zona != nil && intento.politica.Franja.SoloDiasHabiles {
		if habil, errCalendario := s.intentos.calendario.EsDiaHabil(ctx, registro.Contacto.Instante.In(intento.politica.Franja.Zona)); errCalendario == nil {
			intento.diaHabil = habil
		}
	}
	if err = completarIntento(intento, registro.Contacto, &registro); err != nil {
		return puertosbolsa.RegistroContactoParticipacion{}, err
	}
	return registro, nil
}
func (s *ServicioContactoParticipacion) ListarContactosParticipacion(ctx context.Context, q puertosbolsa.ConsultaContactosParticipacion) (puertosbolsa.PaginaContactosParticipacion, error) {
	if ctx == nil || s == nil || q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil || q.BolsaRef == "" || q.ParticipacionRef == "" || q.Limite < 1 || q.Limite > 100 {
		return puertosbolsa.PaginaContactosParticipacion{}, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	if _, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, q.ResultadoContexto.Contexto, q.BolsaRef, q.ParticipacionRef); err != nil {
		return puertosbolsa.PaginaContactosParticipacion{}, err
	}
	resuelto, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, q.ResultadoContexto.Contexto, q.BolsaRef, q.ParticipacionRef)
	if err != nil {
		return puertosbolsa.PaginaContactosParticipacion{}, err
	}
	q, err = s.autorizarConsulta(ctx, q, resuelto)
	if err != nil {
		return puertosbolsa.PaginaContactosParticipacion{}, err
	}
	return s.repositorio.ListarContactosParticipacion(ctx, q)
}
func (s *ServicioContactoParticipacion) ListarContactosBolsa(ctx context.Context, q puertosbolsa.ConsultaContactosBolsa) (puertosbolsa.PaginaContactosParticipacion, error) {
	if ctx == nil || s == nil || q.ResultadoContexto.Validar() != nil || q.Vinculo.ValidarPara(q.ResultadoContexto) != nil || q.BolsaRef == "" || q.Limite < 1 || q.Limite > 100 {
		return puertosbolsa.PaginaContactosParticipacion{}, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	resuelto, err := s.contexto.ResolverContextoContactosBolsa(ctx, q.ResultadoContexto.Contexto, q.BolsaRef)
	if err != nil {
		return puertosbolsa.PaginaContactosParticipacion{}, err
	}
	base := puertosbolsa.ConsultaContactosParticipacion{Vinculo: q.Vinculo, ResultadoContexto: q.ResultadoContexto, BolsaRef: q.BolsaRef, OfertaRef: q.OfertaRef, Cursor: q.Cursor, Limite: q.Limite, Correlacion: q.Correlacion, MotivoAutorizacion: q.MotivoAutorizacion}
	base, err = s.autorizarConsulta(ctx, base, resuelto)
	if err != nil {
		return puertosbolsa.PaginaContactosParticipacion{}, err
	}
	q.SolicitudAutorizacion, q.Decision, q.Confirmacion, q.Material = base.SolicitudAutorizacion, base.Decision, base.Confirmacion, base.Material
	return s.repositorio.ListarContactosBolsa(ctx, q)
}

func (s *ServicioContactoParticipacion) autorizarConsulta(ctx context.Context, q puertosbolsa.ConsultaContactosParticipacion, resuelto puertosbolsa.ContextoSituacionParticipacionResuelto) (puertosbolsa.ConsultaContactosParticipacion, error) {
	referencia := q.ParticipacionRef
	if referencia == "" {
		referencia = q.BolsaRef
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: referencia, ModuloID: puertosbolsa.ModuloSituacionParticipacion, Tipo: puertosbolsa.TipoRecursoSituacionParticipacion, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	auth, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion, Accion: puertosbolsa.AccionConsultarContactoParticipacion, Recurso: recurso, Finalidad: puertosbolsa.FinalidadConsultarContactoParticipacion, Correlacion: q.Correlacion})
	if err != nil {
		return q, dominiovec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, q.ResultadoContexto)
	if err != nil || exportador == nil || decision.ValidarPara(auth) != nil {
		// Solo la denegación nominal ya registrada por el emisor es una
		// denegación; cualquier otro fallo sigue siendo indisponibilidad.
		if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return q, dominiovec.ErrAutorizacionDenegada
		}
		return q, errorDependenciaSituacion(err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(auth, decision, confirmacion, q.ResultadoContexto, q.MotivoAutorizacion, material, puertosbolsa.AudienciaConsultarContactoParticipacion) {
		return q, errorDependenciaSituacion(err)
	}
	q.SolicitudAutorizacion, q.Decision, q.Confirmacion, q.Material = auth, decision, confirmacion, material
	return q, nil
}

func resultadoAdmitidoPorReglas(intento *intentoPreparado, resultado string) bool {
	return intento == nil || resultado != dominiobolsa.ResultadoContactoComunica || intento.politica.SinContacto(resultado)
}
