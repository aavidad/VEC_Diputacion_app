package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"slices"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var patronSancionRef = regexp.MustCompile(`^sancion:[a-f0-9]{64}$`)

// ServicioSancionesParticipacion registra y consulta el histórico de
// sanciones de una participación. Reutiliza la autorización de las
// operaciones de situación (misma acción, finalidad y audiencia) y aplica el
// efecto sobre la situación como una operación B8 dentro de la misma
// transacción que la sanción: no hay otra lógica de estados.
type ServicioSancionesParticipacion struct {
	situacion   *ServicioSituacionParticipacion
	catalogo    ports.CatalogoSancionesParticipacion
	repositorio ports.RepositorioSancionesParticipacion
}

// NuevoServicioSancionesParticipacion admite un catálogo nulo: sin catálogo
// el histórico sigue consultable y el registro responde que no está
// configurado, que es la conducta anterior a esta capacidad.
func NuevoServicioSancionesParticipacion(situacion *ServicioSituacionParticipacion, catalogo ports.CatalogoSancionesParticipacion, repositorio ports.RepositorioSancionesParticipacion) (*ServicioSancionesParticipacion, error) {
	if situacion == nil || repositorio == nil {
		return nil, ErrCambioSituacionParticipacionNoDisponible
	}
	return &ServicioSancionesParticipacion{situacion: situacion, catalogo: catalogo, repositorio: repositorio}, nil
}

func (s *ServicioSancionesParticipacion) ahora() time.Time {
	return s.situacion.reloj().UTC().Truncate(time.Microsecond)
}

// Registrar resuelve la consecuencia en el catálogo, autoriza, y registra la
// sanción. Si la consecuencia cambia la situación, el repositorio la aplica
// mediante la operación B8 correspondiente en la misma transacción.
func (s *ServicioSancionesParticipacion) Registrar(ctx context.Context, q ports.SolicitudRegistrarSancion) (ports.RegistroSancion, error) {
	if s == nil || ctx == nil || q.SolicitudCambiarSituacionParticipacion.Validar() != nil {
		return ports.RegistroSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
	}
	ahora := s.ahora()
	if q.Datos.Validar(ahora) != nil {
		return ports.RegistroSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
	}
	if s.catalogo == nil {
		return ports.RegistroSancion{}, ports.ErrSancionesNoConfiguradas
	}
	fecha, _ := dominiobolsa.FechaCivilSancion(q.Datos.FechaNotificacion)
	resolucion, err := s.catalogo.ResolverSancion(ctx, q.Datos.Consecuencia, inicioCivilSancion(fecha))
	if err != nil {
		return ports.RegistroSancion{}, err
	}
	consecuencia := resolucion.Consecuencia
	if consecuencia.Clave != q.Datos.Consecuencia || !dominiobolsa.EfectoSancionValido(consecuencia.Efecto) || resolucion.Recurso.UltimoDia == "" {
		return ports.RegistroSancion{}, ports.ErrSancionesNoConfiguradas
	}
	auth, decision, confirmacion, material, err := s.situacion.autorizarOperacion(ctx, q.SolicitudCambiarSituacionParticipacion)
	if err != nil {
		return ports.RegistroSancion{}, err
	}
	actor := q.ResultadoContexto.Contexto.PersonaRef
	// La misma política configurable de segunda persona que B8 (duda 6):
	// si pausar la exige, una suspensión también.
	politica, err := s.situacion.politicaSegregacion(ctx)
	if err != nil {
		return ports.RegistroSancion{}, err
	}
	if politica.ExigeSegundaPersona(consecuencia.Efecto) && q.Datos.ResueltaPor == actor {
		return ports.RegistroSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
	}
	huella := huellaClaveSancion(q.ParticipacionRef, q.ClaveIdempotencia)
	sancion := dominiobolsa.SancionParticipacion{
		SancionRef: "sancion:" + huella, ParticipacionRef: q.ParticipacionRef,
		Consecuencia: consecuencia.Clave, ConsecuenciaEtiqueta: consecuencia.Etiqueta, Efecto: consecuencia.Efecto,
		Datos: q.Datos, ReglaRef: consecuencia.ReglaRef, ReglaHuella: consecuencia.Huella,
		SuspensionHasta: resolucion.SuspensionHasta, RecursoVence: resolucion.Recurso.UltimoDia,
		RecursoReglaRef: resolucion.Recurso.ReglaRef, RecursoReglaHuella: resolucion.Recurso.Huella,
		Actor: actor, RegistradaEn: ahora,
	}
	// Una suspensión que termina sola necesita su fecha de fin; sin ella
	// el catálogo está mal configurado y no se supone una suspensión
	// indefinida.
	finAutomatico := consecuencia.FinAutomatico && consecuencia.Efecto == dominiobolsa.OperacionPausar
	if finAutomatico && resolucion.SuspensionHasta == "" {
		return ports.RegistroSancion{}, ports.ErrSancionesNoConfiguradas
	}
	comando := ports.ComandoRegistrarSancion{Sancion: sancion, BolsaRef: q.BolsaRef, ClaveIdempotencia: q.ClaveIdempotencia, ReciboRef: "recibo:sancion:" + huella, Material: material,
		OrdenFinal: consecuencia.OrdenFinal && consecuencia.Efecto != dominiobolsa.OperacionExcluir, FinAutomatico: finAutomatico}
	if consecuencia.Efecto != dominiobolsa.EfectoSancionNinguno {
		destino, _ := dominiobolsa.DestinoOperacionSituacion(consecuencia.Efecto)
		if finAutomatico {
			// La base calcula la vuelta al turno desde la fecha de fin, que
			// es la única fuente: aquí solo se espera la situación.
			destino = dominiobolsa.SituacionDisponibleDesde
		}
		vigente, err := s.situacion.repositorio.SituacionVigente(ctx, q.ParticipacionRef)
		if err != nil {
			return ports.RegistroSancion{}, err
		}
		// Como en B8, la transición la decide la base bajo el mismo consumo
		// de la autorización, también en el reintento idempotente.
		if ahora.Before(vigente.Desde) {
			return ports.RegistroSancion{}, dominiobolsa.ErrCambioSituacionParticipacionInvalido
		}
		comando.ReciboRef = "recibo:situacion:" + huella
		comando.Operacion = &ports.ComandoOperacionSituacion{
			ComandoCambiarSituacionParticipacion: ports.ComandoCambiarSituacionParticipacion{
				Cambio: dominiobolsa.CambioSituacionParticipacion{
					ParticipacionRef: q.ParticipacionRef, Origen: vigente.Situacion, Destino: destino,
					Desde: ahora, Motivo: q.Datos.Causa, RegistradaEn: ahora,
				},
				Actor: actor, BolsaRef: q.BolsaRef, ClaveIdempotencia: q.ClaveIdempotencia, ReciboRef: comando.ReciboRef,
				SolicitudAutorizacion: auth, Decision: decision, Confirmacion: confirmacion, Material: material,
			},
			Operacion: consecuencia.Efecto,
			Justificante: dominiobolsa.JustificanteOperacionSituacion{
				Tipo: dominiobolsa.JustificanteResolucion, Referencia: q.Datos.Resolucion.Referencia, SHA256: q.Datos.Resolucion.SHA256,
			},
			Validador: q.Datos.ResueltaPor, ValidadaEn: ahora,
		}
	}
	return s.repositorio.RegistrarSancion(ctx, comando)
}

// RegistrarRecurso añade un estado del recurso de reposición. Los estados
// admitidos los fija el catálogo; la sanción debe ser de esa participación.
func (s *ServicioSancionesParticipacion) RegistrarRecurso(ctx context.Context, q ports.SolicitudRegistrarRecursoSancion) (ports.RegistroRecursoSancion, error) {
	if s == nil || ctx == nil || q.SolicitudCambiarSituacionParticipacion.Validar() != nil || !patronSancionRef.MatchString(q.SancionRef) {
		return ports.RegistroRecursoSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
	}
	ahora := s.ahora()
	if q.Evento.ValidarDatos(ahora) != nil {
		return ports.RegistroRecursoSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
	}
	if s.catalogo == nil {
		return ports.RegistroRecursoSancion{}, ports.ErrSancionesNoConfiguradas
	}
	estados, err := s.catalogo.EstadosRecurso(ctx)
	if err != nil {
		return ports.RegistroRecursoSancion{}, err
	}
	if !slices.Contains(estados, q.Evento.Estado) {
		return ports.RegistroRecursoSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
	}
	reversion, err := s.catalogo.ReversionRecurso(ctx)
	if err != nil {
		return ports.RegistroRecursoSancion{}, err
	}
	revierte := reversion.Revierte(q.Evento.Estado)
	// Solo un estado revocatorio lleva quien lo resuelve, y la resolución
	// que estima el recurso debe constar.
	if revierte != (q.ResueltaPor != "") || (revierte && q.Evento.Documento == nil) {
		return ports.RegistroRecursoSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
	}
	_, _, _, material, err := s.situacion.autorizarOperacion(ctx, q.SolicitudCambiarSituacionParticipacion)
	if err != nil {
		return ports.RegistroRecursoSancion{}, err
	}
	actor := q.ResultadoContexto.Contexto.PersonaRef
	evento := q.Evento
	evento.Actor = actor
	evento.RegistradaEn = ahora
	if evento.Documento != nil {
		copia := *evento.Documento
		evento.Documento = &copia
	}
	comando := ports.ComandoRegistrarRecursoSancion{
		ParticipacionRef: q.ParticipacionRef, SancionRef: q.SancionRef, Evento: evento,
		ClaveIdempotencia: q.ClaveIdempotencia, Material: material,
	}
	if revierte {
		// Como la baja, la readmisión la resuelve otra persona.
		if !dominiobolsa.IdentidadResolucionValida(q.ResueltaPor) || q.ResueltaPor == actor {
			return ports.RegistroRecursoSancion{}, dominiobolsa.ErrSancionParticipacionInvalida
		}
		comando.Reversion = &ports.ComandoReversionSancion{
			ResueltaPor: q.ResueltaPor, ReglaRef: reversion.ReglaRef, Huella: reversion.Huella, Motivo: reversion.Motivo,
			ReciboRef: "recibo:readmision:" + huellaClaveSancion(q.SancionRef, q.ClaveIdempotencia),
		}
	}
	return s.repositorio.RegistrarRecursoSancion(ctx, comando)
}

// Consultar devuelve el histórico autorizado y el catálogo vigente. Un fallo
// del catálogo no oculta el histórico: solo impide registrar.
func (s *ServicioSancionesParticipacion) Consultar(ctx context.Context, q ports.SolicitudCambiarSituacionParticipacion) (ports.VistaSancionesParticipacion, error) {
	if s == nil || ctx == nil || q.Validar() != nil {
		return ports.VistaSancionesParticipacion{}, ErrCambioSituacionParticipacionNoDisponible
	}
	_, _, _, material, err := s.situacion.autorizarOperacion(ctx, q)
	if err != nil {
		return ports.VistaSancionesParticipacion{}, err
	}
	sanciones, err := s.repositorio.ListarSanciones(ctx, q.ParticipacionRef, q.ResultadoContexto.Contexto.PersonaRef, material)
	if err != nil {
		return ports.VistaSancionesParticipacion{}, err
	}
	vista := ports.VistaSancionesParticipacion{Sanciones: sanciones}
	if s.catalogo == nil {
		return vista, nil
	}
	consecuencias, errConsecuencias := s.catalogo.Consecuencias(ctx)
	estados, errEstados := s.catalogo.EstadosRecurso(ctx)
	reversion, errReversion := s.catalogo.ReversionRecurso(ctx)
	if errConsecuencias == nil && errEstados == nil && errReversion == nil && len(consecuencias) > 0 {
		vista.Consecuencias, vista.EstadosRecurso, vista.CatalogoDisponible = consecuencias, estados, true
		vista.EstadosRevocatorios = reversion.Estados
	} else if errors.Is(ctx.Err(), context.Canceled) {
		return ports.VistaSancionesParticipacion{}, ctx.Err()
	}
	return vista, nil
}

// inicioCivilSancion sitúa la fecha civil a mediodía UTC: en hora
// peninsular sigue siendo el mismo día, que es el que usa el cómputo.
func inicioCivilSancion(fecha time.Time) time.Time {
	return time.Date(fecha.Year(), fecha.Month(), fecha.Day(), 12, 0, 0, 0, time.UTC)
}

func huellaClaveSancion(participacion, clave string) string {
	h := sha256.Sum256([]byte(participacion + "\x1f" + clave))
	return hex.EncodeToString(h[:])
}
