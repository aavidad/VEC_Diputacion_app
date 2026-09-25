package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// Operar aplica B8 a la situación B2 vigente. La operación se conserva junto
// al cambio en una sola transacción del repositorio.
func (s *ServicioSituacionParticipacion) Operar(ctx context.Context, q ports.SolicitudOperacionSituacion) (ports.RegistroSituacionParticipacion, error) {
	if s == nil || ctx == nil || q.SolicitudCambiarSituacionParticipacion.Validar() != nil || q.Justificante.Validar() != nil {
		return ports.RegistroSituacionParticipacion{}, dominiobolsa.ErrOperacionSituacionParticipacionInvalida
	}
	destino, ok := dominiobolsa.DestinoOperacionSituacion(q.Operacion)
	if !ok || q.Destino != destino || q.FechaDisponible != nil {
		return ports.RegistroSituacionParticipacion{}, dominiobolsa.ErrOperacionSituacionParticipacionInvalida
	}
	repo, ok := s.repositorio.(ports.RepositorioOperacionSituacion)
	if !ok {
		return ports.RegistroSituacionParticipacion{}, ErrCambioSituacionParticipacionNoDisponible
	}
	auth, decision, confirmacion, material, err := s.autorizarOperacion(ctx, q.SolicitudCambiarSituacionParticipacion)
	if err != nil {
		return ports.RegistroSituacionParticipacion{}, err
	}
	actor := q.ResultadoContexto.Contexto.PersonaRef
	ahora := s.reloj().UTC().Truncate(time.Microsecond)
	// El validador es una identidad declarada por RRHH, no un firmante. Qué
	// operaciones exigen otra persona lo fija la política configurable; la
	// base de datos vuelve a comprobarlo con la misma versión.
	if q.Validador == "" || strings.TrimSpace(q.Validador) != q.Validador || len(q.Validador) > 256 {
		return ports.RegistroSituacionParticipacion{}, dominiobolsa.ErrOperacionSituacionParticipacionInvalida
	}
	politica, err := s.politicaSegregacion(ctx)
	if err != nil {
		return ports.RegistroSituacionParticipacion{}, err
	}
	if politica.ExigeSegundaPersona(q.Operacion) && q.Validador == actor {
		return ports.RegistroSituacionParticipacion{}, dominiobolsa.ErrOperacionSituacionParticipacionInvalida
	}
	vigente, err := s.repositorio.SituacionVigente(ctx, q.ParticipacionRef)
	if err != nil {
		return ports.RegistroSituacionParticipacion{}, err
	}
	cambio := dominiobolsa.CambioSituacionParticipacion{ParticipacionRef: q.ParticipacionRef, Origen: vigente.Situacion, Destino: destino, Desde: ahora, Motivo: q.Motivo, RegistradaEn: ahora}
	// El cambio vigente puede ser ya el efecto de esta clave. B2 resuelve
	// replay antes de validar transiciones, bajo el mismo consumo V3; una
	// lectura previa del recibo abriría una vía SQL sin permiso consumido.
	if ahora.Before(vigente.Desde) {
		return ports.RegistroSituacionParticipacion{}, dominiobolsa.ErrCambioSituacionParticipacionInvalido
	}
	// Si la situación vigente ya es el destino puede tratarse del replay de
	// esta misma clave: lo resuelve la base de datos, como hasta ahora.
	if vigente.Situacion != destino {
		politica, err := s.politicaEfectiva(ctx)
		if err != nil {
			return ports.RegistroSituacionParticipacion{}, err
		}
		if !politica.Admite(vigente.Situacion, destino) {
			return ports.RegistroSituacionParticipacion{}, dominiobolsa.ErrCambioSituacionParticipacionInvalido
		}
	}
	h := sha256.Sum256([]byte(q.ParticipacionRef + "\x1f" + q.ClaveIdempotencia))
	recibo := "recibo:situacion:" + hex.EncodeToString(h[:])
	return repo.RegistrarOperacion(ctx, ports.ComandoOperacionSituacion{ComandoCambiarSituacionParticipacion: ports.ComandoCambiarSituacionParticipacion{Cambio: cambio, Actor: actor, BolsaRef: q.BolsaRef, ClaveIdempotencia: q.ClaveIdempotencia, ReciboRef: recibo, SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material}, Operacion: q.Operacion, Justificante: q.Justificante, Validador: q.Validador, ValidadaEn: ahora})
}

// politicaSegregacion lee la política vigente del repositorio. Sin consulta
// rige el mínimo fijo; una consulta que falla nunca se interpreta como una
// política más laxa.
func (s *ServicioSituacionParticipacion) politicaSegregacion(ctx context.Context) (dominiobolsa.PoliticaSegregacion, error) {
	consulta, ok := s.repositorio.(ports.ConsultaPoliticaSegregacion)
	if !ok {
		return dominiobolsa.PoliticaSegregacionMinima(), nil
	}
	vigente, err := consulta.PoliticaSegregacion(ctx)
	if err != nil {
		return dominiobolsa.PoliticaSegregacion{}, ErrCambioSituacionParticipacionNoDisponible
	}
	return vigente.Politica, nil
}

func (s *ServicioSituacionParticipacion) ListarOperaciones(ctx context.Context, q ports.SolicitudCambiarSituacionParticipacion) ([]ports.RegistroOperacionSituacion, error) {
	if s == nil || ctx == nil || q.Validar() != nil {
		return nil, ErrCambioSituacionParticipacionNoDisponible
	}
	repo, ok := s.repositorio.(ports.RepositorioOperacionSituacion)
	if !ok {
		return nil, ErrCambioSituacionParticipacionNoDisponible
	}
	_, _, _, material, err := s.autorizarOperacion(ctx, q)
	if err != nil {
		return nil, err
	}
	return repo.ListarOperaciones(ctx, q.ParticipacionRef, q.ResultadoContexto.Contexto.PersonaRef, material)
}

func (s *ServicioSituacionParticipacion) autorizarOperacion(ctx context.Context, q ports.SolicitudCambiarSituacionParticipacion) (dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var a dominiovec.SolicitudAutorizacionLigadaV3
	var d dominiovec.DecisionAutorizacionLigadaV3
	var c puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	var m puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if s.contexto == nil || s.autorizador == nil || s.repositorio == nil || q.ResultadoContexto.Contexto.PersonaRef == "" {
		return a, d, c, m, ErrCambioSituacionParticipacionNoDisponible
	}
	resuelto, err := s.contexto.ResolverContextoSituacionParticipacion(ctx, q.ResultadoContexto.Contexto, q.BolsaRef, q.ParticipacionRef)
	if err != nil || resuelto.Validar() != nil {
		return a, d, c, m, errorDependenciaSituacion(err)
	}
	pertenece, err := s.repositorio.ParticipacionPerteneceABolsa(ctx, q.BolsaRef, q.ParticipacionRef)
	if err != nil {
		return a, d, c, m, err
	}
	if !pertenece {
		return a, d, c, m, dominiovec.ErrAutorizacionDenegada
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: q.ParticipacionRef, ModuloID: ports.ModuloSituacionParticipacion, Tipo: ports.TipoRecursoSituacionParticipacion, Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	a, err = dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: q.Vinculo, ReferenciaMotivo: q.MotivoAutorizacion, Accion: ports.AccionCambiarSituacionParticipacion, Recurso: recurso, Finalidad: ports.FinalidadCambiarSituacionParticipacion, Correlacion: q.Correlacion})
	if err != nil {
		return a, d, c, m, dominiovec.ErrAutorizacionDenegada
	}
	var exportador puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3
	d, c, exportador, err = s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, a, q.ResultadoContexto)
	if err != nil || exportador == nil || d.ValidarPara(a) != nil {
		return a, d, c, m, errorDependenciaSituacion(err)
	}
	m, err = exportador.ExportarMaterialParaConsumidor()
	if err != nil || !materialAutorizacionBorradorLlamamientoExacto(a, d, c, q.ResultadoContexto, q.MotivoAutorizacion, m, ports.AudienciaCambiarSituacionParticipacion) {
		return a, d, c, m, errorDependenciaSituacion(err)
	}
	return a, d, c, m, nil
}
