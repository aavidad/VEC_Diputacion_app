package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
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
	correo        CorreoPersonalizadoLlamamiento
	// origenes es opcional: sin él no se avisa del contacto no confirmado.
	origenes puertosbolsa.FuenteOrigenContactoParticipacion
	// avisador decide el correo de cada aviso (B59 opcional) y lo envía.
	avisador         *AvisadorLlamamiento
	derivadorTokenCT puertosbolsa.DerivadorTokenFinalizacionCT
	// motivosSalto es opcional: sin catálogo ningún salto de orden se admite.
	motivosSalto puertosbolsa.CatalogoMotivosSaltoOrden
}

// EstablecerCatalogoMotivosSalto conecta los motivos «b01.salto.*» del
// catálogo de reglas con los que RRHH puede justificar saltar el orden.
func (s *ServicioEmisionLlamamiento) EstablecerCatalogoMotivosSalto(c puertosbolsa.CatalogoMotivosSaltoOrden) error {
	if s == nil || c == nil {
		return puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	s.motivosSalto = c
	return nil
}

// EstablecerDerivadorTokenFinalizacionCT inyecta el secreto persistente por su
// autoridad privada antes de atender solicitudes CT. No afecta a B7 ordinaria.
func (s *ServicioEmisionLlamamiento) EstablecerDerivadorTokenFinalizacionCT(d puertosbolsa.DerivadorTokenFinalizacionCT) error {
	if s == nil || d == nil || s.derivadorTokenCT != nil {
		return puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	s.derivadorTokenCT = d
	return nil
}

// CorreoPersonalizadoLlamamiento agrupa el catálogo del correo, la fuente de
// datos de cada persona y el registro de las huellas de lo enviado.
type CorreoPersonalizadoLlamamiento struct {
	Catalogo        dominiobolsa.CatalogoCorreoLlamamiento
	Personalizacion puertosbolsa.FuentePersonalizacionLlamamiento
	Huellas         puertosbolsa.RegistroHuellasCuerpoLlamamiento
}

func NuevoServicioEmisionLlamamiento(cb puertosbolsa.ResolutorContextoContactoParticipacion, a puertosbolsa.AutorizadorSituacionParticipacionV3, r puertosbolsa.RepositorioEmisionLlamamiento, f puertosbolsa.FuenteCorreoParticipacion, e puertosbolsa.EmisorCorreoBolsa, reloj func() time.Time, correo CorreoPersonalizadoLlamamiento) (*ServicioEmisionLlamamiento, error) {
	if cb == nil || a == nil || r == nil || f == nil || e == nil || reloj == nil || !correo.Catalogo.Valido() || correo.Personalizacion == nil || correo.Huellas == nil {
		return nil, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	avisador, err := NuevoAvisadorLlamamiento(f, e)
	if err != nil {
		return nil, err
	}
	return &ServicioEmisionLlamamiento{contextoBolsa: cb, autorizador: a, repositorio: r, correos: f, emisor: e, reloj: reloj, correo: correo, avisador: avisador}, nil
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
	if ctx == nil || s == nil || s.validarSolicitudEmision(q) != nil {
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
	// El motivo de salto se resuelve ya autorizado: sin permiso no se revela
	// si una clave está en el catálogo.
	salto, err := s.resolverMotivoSalto(ctx, q.MotivoSalto)
	if err != nil {
		return puertosbolsa.EmisionLlamamiento{}, err
	}
	h := sha256.Sum256([]byte(q.BolsaRef + "\x1f" + q.ClaveIdempotencia))
	sufijo := hex.EncodeToString(h[:])
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	// Resuelve también la versión privada de un replay CT antes de componer
	// correos: si esa dependencia cae, aún podrá devolver un recibo completo
	// ya confirmado sin cambiar el cuerpo de la intención.
	tokenFinalizacion, origenCT, err := s.tokenFinalizacion(ctx, q, actor.PersonaRef)
	if err != nil {
		return puertosbolsa.EmisionLlamamiento{}, err
	}
	defer clear(tokenFinalizacion)
	q.OrigenCT = origenCT
	// Los correos se componen antes de reservar: un marcador sin dato o un texto
	// que supera el límite se rechaza sin consumir la autorización.
	correos, err := s.componerCorreos(ctx, q.BolsaRef, q.Participaciones, q.Configuracion, ahora)
	if err != nil {
		if previa, recErr := s.repositorio.Recuperar(ctx, q.BolsaRef, q.ClaveIdempotencia); recErr == nil && mismaSolicitudEmision(previa, q) && len(previa.Contactos) == len(q.Participaciones) {
			previa.Reutilizada = true
			return previa, nil
		}
		if q.OrigenCT != nil {
			if lector, ok := s.repositorio.(puertosbolsa.LectorOrigenEmisionCT); ok {
				if origen, readErr := lector.RecuperarOrigenEmisionCT(ctx, q.OrigenCT.ExpedienteRef, q.OrigenCT.ClaveCT); readErr == nil {
					previa, recErr := s.repositorio.Recuperar(ctx, q.BolsaRef, q.ClaveIdempotencia)
					previa.OrigenCT = &origen
					if recErr == nil && mismaSolicitudEmision(previa, q) && len(previa.Contactos) == len(q.Participaciones) {
						previa.Reutilizada = true
						return previa, nil
					}
				}
			}
		}
		return puertosbolsa.EmisionLlamamiento{}, err
	}
	reservada, err := s.repositorio.Reservar(ctx, puertosbolsa.ComandoEmitirLlamamiento{LlamamientoRef: "llamamiento:" + sufijo, ReciboRef: "recibo:llamamiento:" + sufijo, BolsaRef: q.BolsaRef, ActorRef: actor.PersonaRef, ClaveIdempotencia: q.ClaveIdempotencia, Participaciones: append([]string(nil), q.Participaciones...), Configuracion: q.Configuracion, EmitidoEn: ahora, SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material, TokenFinalizacion: tokenFinalizacion, OrigenCT: q.OrigenCT, SaltoOrden: salto})
	if err != nil {
		return puertosbolsa.EmisionLlamamiento{}, s.conMotivosSalto(ctx, err)
	}
	if !mismaSolicitudEmision(reservada, q) {
		return puertosbolsa.EmisionLlamamiento{}, puertosbolsa.ErrEmisionLlamamientoConflicto
	}
	if reservada.Reutilizada {
		if len(reservada.Contactos) == len(q.Participaciones) {
			return s.conAvisosContacto(ctx, reservada), nil
		}
		// CT puede completar una reserva propia sin contactos con el mismo
		// token derivado de la versión privada guardada. B7 ordinaria mantiene
		// su denegación al perder el token aleatorio.
		if q.OrigenCT == nil || len(reservada.Contactos) != 0 {
			return puertosbolsa.EmisionLlamamiento{}, puertosbolsa.ErrEmisionLlamamientoNoDisponible
		}
	}
	if err = s.correo.Huellas.RegistrarHuellasCuerpo(ctx, q.BolsaRef, q.ClaveIdempotencia, actor.PersonaRef, tokenFinalizacion, huellasCorreos(q.Participaciones, correos)); err != nil {
		return puertosbolsa.EmisionLlamamiento{}, err
	}
	contactos := make([]puertosbolsa.ResultadoContactoEmision, 0, len(q.Participaciones))
	presupuesto, cancelarAvisos := s.avisador.Presupuesto(ctx)
	defer cancelarAvisos()
	for i, participacion := range q.Participaciones {
		resultado, fuente := s.avisador.Avisar(ctx, presupuesto, AvisoLlamamiento{
			Vinculo: q.Vinculo, ResultadoContexto: q.ResultadoContexto, MotivoAutorizacion: q.MotivoAutorizacion,
			BolsaRef: q.BolsaRef, UnidadRef: resuelto.UnidadRef, AmbitoRef: resuelto.AmbitoRef,
			LlamamientoRef: reservada.LlamamientoRef, ParticipacionRef: participacion,
			Asunto: correos[i].Asunto, Cuerpo: correos[i].Cuerpo, MessageID: fmt.Sprintf("<%s-%d@vec.dipgra.local>", sufijo, i+1), Instante: ahora,
		})
		reciboContacto := sha256.Sum256([]byte(q.BolsaRef + "\x1f" + q.ClaveIdempotencia + "\x1f" + participacion))
		contactos = append(contactos, puertosbolsa.ResultadoContactoEmision{ParticipacionRef: participacion, Resultado: resultado, ReciboRef: "recibo:contacto:" + hex.EncodeToString(reciboContacto[:]), FuenteCorreo: fuente})
	}
	emitido, err := s.repositorio.RegistrarContactos(ctx, q.BolsaRef, q.ClaveIdempotencia, actor.PersonaRef, tokenFinalizacion, contactos)
	if err != nil {
		return emitido, err
	}
	emitido.OrigenCT = reservada.OrigenCT
	return s.conAvisosContacto(ctx, emitido), nil
}

// resolverMotivoSalto devuelve nil sin motivo. Con motivo, exige que sea una
// entrada vigente del catálogo; si no hay catálogo, no se admite ninguno.
func (s *ServicioEmisionLlamamiento) resolverMotivoSalto(ctx context.Context, clave string) (*puertosbolsa.SaltoOrdenLlamamiento, error) {
	if clave == "" {
		return nil, nil
	}
	if s.motivosSalto == nil {
		return nil, fmt.Errorf("%w: %w", puertosbolsa.ErrEmisionLlamamientoInvalida, puertosbolsa.ErrMotivoSaltoNoAdmitido)
	}
	salto, err := s.motivosSalto.ResolverMotivoSaltoOrden(ctx, clave)
	switch {
	case errors.Is(err, puertosbolsa.ErrMotivoSaltoNoAdmitido):
		return nil, fmt.Errorf("%w: %w", puertosbolsa.ErrEmisionLlamamientoInvalida, puertosbolsa.ErrMotivoSaltoNoAdmitido)
	case err != nil:
		return nil, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	case salto.Clave != clave || salto.ReglaRef == "" || salto.HuellaCatalogo == "":
		return nil, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	return &salto, nil
}

// conMotivosSalto añade al rechazo por orden los motivos de salto vigentes,
// para que RRHH sepa si puede justificar el salto. Si el catálogo no se puede
// leer, el rechazo sigue igual y sin motivos.
func (s *ServicioEmisionLlamamiento) conMotivosSalto(ctx context.Context, err error) error {
	var fuera *puertosbolsa.LlamamientoFueraDeOrden
	if !errors.As(err, &fuera) || s.motivosSalto == nil {
		return err
	}
	copia := *fuera
	if motivos, errMotivos := s.motivosSalto.MotivosSaltoOrden(ctx); errMotivos == nil {
		copia.Motivos = motivos
	}
	return &copia
}

func (s *ServicioEmisionLlamamiento) tokenFinalizacion(ctx context.Context, q puertosbolsa.SolicitudEmitirLlamamiento, actor string) ([]byte, *puertosbolsa.OrigenEmisionCT, error) {
	if q.OrigenCT == nil {
		token := make([]byte, 32)
		if _, err := rand.Read(token); err != nil {
			return nil, nil, puertosbolsa.ErrEmisionLlamamientoNoDisponible
		}
		return token, nil, nil
	}
	lector, ok := s.repositorio.(puertosbolsa.LectorOrigenEmisionCT)
	if !ok || s.derivadorTokenCT == nil {
		return nil, nil, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	origen := *q.OrigenCT
	origen.VersionToken = ""
	canon, err := json.Marshal(origen)
	if err != nil {
		return nil, nil, puertosbolsa.ErrEmisionLlamamientoInvalida
	}
	huella := sha256.Sum256(canon)
	version := ""
	previo, err := lector.RecuperarOrigenEmisionCT(ctx, origen.ExpedienteRef, origen.ClaveCT)
	switch {
	case err == nil:
		if previo.BolsaRef != q.BolsaRef || previo.LlamamientoRef == "" || previo.VersionToken == "" {
			return nil, nil, puertosbolsa.ErrEmisionLlamamientoConflicto
		}
		version = previo.VersionToken
	case errors.Is(err, puertosbolsa.ErrOrigenLlamamientoCTNoEncontrado):
		// Primera reserva: el derivador elige la versión actual persistente.
	default:
		return nil, nil, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	token, versionUsada, err := s.derivadorTokenCT.DerivarTokenFinalizacionCT(ctx,
		puertosbolsa.DatosDerivacionTokenFinalizacionCT{BolsaRef: q.BolsaRef,
			ClaveB7: q.ClaveIdempotencia, ActorRef: actor,
			HuellaOrigenSHA256: hex.EncodeToString(huella[:]), VersionClave: version})
	if err != nil || len(token) != 32 || versionUsada == "" || (version != "" && versionUsada != version) {
		return nil, nil, puertosbolsa.ErrEmisionLlamamientoNoDisponible
	}
	origen.VersionToken = versionUsada
	return token, &origen, nil
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
	if q.OrigenCT != nil && (previa.OrigenCT == nil || !reflect.DeepEqual(previa.OrigenCT.OrigenEmisionCT, *q.OrigenCT) ||
		previa.OrigenCT.LlamamientoRef != previa.LlamamientoRef || previa.OrigenCT.ReciboRef != previa.ReciboRef) {
		return false
	}
	for i := range q.Participaciones {
		if previa.Participaciones[i] != q.Participaciones[i] {
			return false
		}
	}
	return true
}

func (s *ServicioEmisionLlamamiento) validarSolicitudEmision(q puertosbolsa.SolicitudEmitirLlamamiento) error {
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
	// El motivo de salto solo existe en la emisión general: la de
	// Contratación ya toma el primer turno de la lectura de Bolsa.
	if q.MotivoSalto != "" && (q.OrigenCT != nil || len(q.MotivoSalto) > 128 || strings.TrimSpace(q.MotivoSalto) != q.MotivoSalto) {
		return puertosbolsa.ErrEmisionLlamamientoInvalida
	}
	if q.OrigenCT != nil && (q.OrigenCT.BolsaRef != q.BolsaRef || len(q.Participaciones) != 1 ||
		q.Participaciones[0] != q.OrigenCT.ParticipacionRef || q.OrigenCT.CandidatoRef == "" ||
		q.OrigenCT.ExpedienteRef == "" || q.OrigenCT.ClaveCT == "" || q.OrigenCT.OrdenSeleccionado == 0 ||
		q.OrigenCT.Lectura.BolsaRef != q.BolsaRef || q.OrigenCT.Lectura.ActaRef != q.OrigenCT.ActaRef ||
		q.OrigenCT.Lectura.InstantaneaRef != q.OrigenCT.InstantaneaRef || q.OrigenCT.Lectura.Corte.IsZero()) {
		return puertosbolsa.ErrEmisionLlamamientoInvalida
	}
	return s.validarConfiguracion(q.Configuracion)
}

// validarConfiguracion conserva los límites de cada campo (PostgreSQL 000017)
// y exige una versión de plantilla del catálogo con marcadores conocidos.
func (s *ServicioEmisionLlamamiento) validarConfiguracion(c puertosbolsa.ConfiguracionLlamamiento) error {
	for _, v := range []string{c.Referencia, c.Descripcion, c.Categoria, c.Centro, c.Modalidad, c.FechaInicio, c.Plazo, c.PlantillaVersion, c.Asunto, c.Cuerpo} {
		if strings.TrimSpace(v) != v || len(v) < 2 || len(v) > 4000 {
			return puertosbolsa.ErrEmisionLlamamientoInvalida
		}
	}
	if s.correo.Catalogo.ValidarPlantilla(c.PlantillaVersion, c.Asunto) != nil || s.correo.Catalogo.ValidarPlantilla(c.PlantillaVersion, c.Cuerpo) != nil {
		return fmt.Errorf("%w: %w", puertosbolsa.ErrEmisionLlamamientoInvalida, dominiobolsa.ErrPlantillaCorreoLlamamiento)
	}
	return nil
}

func esConflictoEmision(err error) bool {
	return errors.Is(err, puertosbolsa.ErrEmisionLlamamientoConflicto)
}
