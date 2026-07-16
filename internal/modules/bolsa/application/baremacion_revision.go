package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ActorBaremacion contiene exclusivamente la motivacion administrativa. La
// identidad, el perfil y la garantia se resuelven siempre desde la sesion
// autoritativa; repetirlos en una orden ampliaria innecesariamente la
// superficie de datos personales y nunca podria conceder acceso.
type ActorBaremacion struct {
	Motivo string
}

// OrdenIniciarRevisionBaremacion identifica el merito y sujeto cuya version
// se fija como base inmutable para la revision; no solicita una reserva.
type OrdenIniciarRevisionBaremacion struct {
	Actor               ActorBaremacion
	BaremacionMeritoRef string
	SujetoRef           string
	Finalidad           string
	CorrelacionRef      string
}

// RevisionBaremacionIniciada conserva la version exacta y la autorizacion de
// consulta que la obtuvo. Su contenido no es un DTO transportable.
type RevisionBaremacionIniciada struct {
	version             puertosbolsa.VersionBaremacion
	principalReservaRef string
	perfilActorClave    string
	sujetoRef           string
	finalidadClave      string
	correlacionRef      string
	autorizacionesRefs  []string
}

func (RevisionBaremacionIniciada) MarshalJSON() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (*RevisionBaremacionIniciada) UnmarshalJSON([]byte) error { return ErrOrdenBaremacionInvalida }
func (RevisionBaremacionIniciada) MarshalText() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (RevisionBaremacionIniciada) MarshalBinary() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (RevisionBaremacionIniciada) String() string     { return "[REVISION-BAREMACION-INICIADA-OPACA]" }
func (r RevisionBaremacionIniciada) GoString() string { return r.String() }
func (r RevisionBaremacionIniciada) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r RevisionBaremacionIniciada) LogValue() slog.Value { return slog.StringValue(r.String()) }

// IniciarRevision consulta y fija una instantanea defensiva de la version
// vigente. No adquiere ninguna reserva durante la revision humana.
func (s *ServicioBaremacion) IniciarRevision(
	ctx context.Context,
	orden OrdenIniciarRevisionBaremacion,
) (RevisionBaremacionIniciada, error) {
	if err := validarActorBaremacion(orden.Actor); err != nil ||
		!referenciaAplicacionBaremacionValida(orden.BaremacionMeritoRef) ||
		!referenciaAplicacionBaremacionValida(orden.SujetoRef) ||
		!claveAplicacionBaremacionValida(orden.Finalidad) ||
		!referenciaAplicacionBaremacionValida(orden.CorrelacionRef) {
		return RevisionBaremacionIniciada{}, ErrOrdenBaremacionInvalida
	}
	consulta, err := s.autorizar(ctx, orden.Actor, puertosbolsa.AccionConsultarBaremacionVigente,
		puertosbolsa.ClaseRecursoBaremacion, orden.BaremacionMeritoRef, orden.SujetoRef,
		orden.Finalidad, orden.CorrelacionRef)
	if err != nil {
		return RevisionBaremacionIniciada{}, err
	}
	version, err := s.repositorio.ObtenerVersionVigente(ctx, puertosbolsa.SolicitudObtenerBaremacionVigente{
		Contexto: consulta, BaremacionMeritoRef: orden.BaremacionMeritoRef,
	})
	if err != nil || version.Validar() != nil || version.Agregado.SujetoRef != orden.SujetoRef {
		return RevisionBaremacionIniciada{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	autorizaciones, err := incorporarAutorizacionesBaremacion(nil, consulta)
	if err != nil {
		return RevisionBaremacionIniciada{}, err
	}
	versionClonada, err := version.Clonar()
	if err != nil {
		return RevisionBaremacionIniciada{}, ErrResultadoBaremacionNoConfiable
	}
	resultado := RevisionBaremacionIniciada{
		version: versionClonada, principalReservaRef: consulta.Proyeccion().PrincipalRef,
		perfilActorClave: consulta.Proyeccion().PerfilActorClave, sujetoRef: versionClonada.Agregado.SujetoRef,
		finalidadClave: orden.Finalidad, correlacionRef: orden.CorrelacionRef,
		autorizacionesRefs: autorizaciones,
	}
	if err := validarRevisionIniciada(resultado); err != nil {
		return RevisionBaremacionIniciada{}, err
	}
	return resultado, nil
}

// OrdenAdoptarDecisionBaremacion expresa el juicio tecnico; actor, numero,
// calculo oficial y referencias administrativas se obtienen o verifican en el
// servidor.
type OrdenAdoptarDecisionBaremacion struct {
	Actor                  ActorBaremacion
	Revision               RevisionBaremacionIniciada
	Clase                  dominiobolsa.ClaseDecisionTecnica
	CalculoRef             string
	HuellaResultadoCalculo string
	PuntosReconocidos      dominiobolsa.Puntos
	Resultado              dominiobolsa.ResultadoDecisionTecnica
	ValoracionesEvidencia  []dominiobolsa.ValoracionEvidencia
	MotivoClave            string
	MotivoDecision         string
	FuentesNormativasRefs  []string
}

// RevisionBaremacionAdoptada contiene el documento administrativo aun no
// firmado y la capacidad opaca exacta que autorizo su adopcion.
type RevisionBaremacionAdoptada struct {
	revision           RevisionBaremacionIniciada
	contenido          dominiobolsa.ContenidoDecisionTecnica
	contextoAdopcion   puertosbolsa.ContextoOperacionBaremacion
	calculo            puertosbolsa.ResultadoCalculoOficial
	autorizacionesRefs []string
}

func (RevisionBaremacionAdoptada) MarshalJSON() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (*RevisionBaremacionAdoptada) UnmarshalJSON([]byte) error { return ErrOrdenBaremacionInvalida }
func (RevisionBaremacionAdoptada) MarshalText() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (RevisionBaremacionAdoptada) MarshalBinary() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (RevisionBaremacionAdoptada) String() string     { return "[REVISION-BAREMACION-ADOPTADA-OPACA]" }
func (r RevisionBaremacionAdoptada) GoString() string { return r.String() }
func (r RevisionBaremacionAdoptada) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r RevisionBaremacionAdoptada) LogValue() slog.Value { return slog.StringValue(r.String()) }

// AdoptarDecision recupera el calculo oficial, revalida todas sus fuentes y
// construye el contenido mediante las invariantes del agregado.
func (s *ServicioBaremacion) AdoptarDecision(
	ctx context.Context,
	orden OrdenAdoptarDecisionBaremacion,
) (RevisionBaremacionAdoptada, error) {
	if err := s.validarRevisionVigente(orden.Revision); err != nil {
		return RevisionBaremacionAdoptada{}, err
	}
	if validarActorRevision(orden.Actor, orden.Revision) != nil || !orden.Clase.Valida() ||
		!referenciaAplicacionBaremacionValida(orden.CalculoRef) ||
		!huellaAplicacionBaremacionValida(orden.HuellaResultadoCalculo) ||
		!orden.PuntosReconocidos.Validos() || !orden.Resultado.Valido() ||
		!claveAplicacionBaremacionValida(orden.MotivoClave) ||
		!textoAplicacionBaremacionValido(orden.MotivoDecision, 8000, true) ||
		len(orden.ValoracionesEvidencia) == 0 || len(orden.ValoracionesEvidencia) > 256 ||
		len(orden.FuentesNormativasRefs) == 0 || len(orden.FuentesNormativasRefs) > 256 {
		return RevisionBaremacionAdoptada{}, ErrOrdenBaremacionInvalida
	}
	agregado := orden.Revision.version.Agregado
	autorizacionCalculo, err := s.autorizarRevision(ctx, orden.Actor, orden.Revision, puertosbolsa.AccionRecuperarCalculoBaremacion,
		puertosbolsa.ClaseRecursoCalculo, orden.CalculoRef, agregado.SujetoRef,
		agregadoFinalidad(orden.Revision), orden.Revision.correlacionRef)
	if err != nil {
		return RevisionBaremacionAdoptada{}, err
	}
	autorizaciones, err := incorporarAutorizacionesBaremacion(orden.Revision.autorizacionesRefs, autorizacionCalculo)
	if err != nil {
		return RevisionBaremacionAdoptada{}, err
	}
	solicitudCalculo := puertosbolsa.SolicitudRecuperarCalculoOficial{
		Contexto: autorizacionCalculo, CalculoRef: orden.CalculoRef, HuellaResultado: orden.HuellaResultadoCalculo,
	}
	calculo, err := s.calculador.RecuperarCalculoOficial(ctx, solicitudCalculo)
	if err != nil || calculo.Validar() != nil || calculo.Calculo.CalculoRef != orden.CalculoRef ||
		calculo.Calculo.HuellaResultadoSHA256 != orden.HuellaResultadoCalculo ||
		validarCalculoParaAgregado(calculo.Calculo, agregado) != nil {
		return RevisionBaremacionAdoptada{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	if orden.Clase == dominiobolsa.ClaseDecisionInicial && !calculo.Calculo.CoincideCon(agregado.CalculoInicial) {
		return RevisionBaremacionAdoptada{}, ErrResultadoBaremacionNoConfiable
	}
	autorizacionesFuentes, err := s.verificarFuentesDecision(ctx, orden.Actor, orden.Revision, calculo.Calculo)
	if err != nil {
		return RevisionBaremacionAdoptada{}, err
	}
	autorizaciones, err = incorporarReferenciasAutorizacionBaremacion(autorizaciones, autorizacionesFuentes...)
	if err != nil {
		return RevisionBaremacionAdoptada{}, err
	}
	accionAdopcion, conocida := puertosbolsa.AccionAdopcionParaClase(orden.Clase)
	if !conocida {
		return RevisionBaremacionAdoptada{}, ErrOrdenBaremacionInvalida
	}
	autorizacionAdopcion, err := s.autorizarRevision(ctx, orden.Actor, orden.Revision, accionAdopcion,
		puertosbolsa.ClaseRecursoBaremacion, agregado.ID, agregado.SujetoRef,
		agregadoFinalidad(orden.Revision), orden.Revision.correlacionRef)
	if err != nil {
		return RevisionBaremacionAdoptada{}, err
	}
	autorizaciones, err = incorporarAutorizacionesBaremacion(autorizaciones, autorizacionAdopcion)
	if err != nil {
		return RevisionBaremacionAdoptada{}, err
	}
	id, err := s.generador.NuevoIDDecisionTecnica()
	if err != nil || !referenciaAplicacionBaremacionValida(id) {
		return RevisionBaremacionAdoptada{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	ahora, err := s.ahora()
	if err != nil {
		return RevisionBaremacionAdoptada{}, err
	}
	proyeccion := autorizacionAdopcion.Proyeccion()
	propuesta := dominiobolsa.PropuestaDecisionTecnica{
		ID: id, CalculoOficial: calculo.Calculo, PuntosReconocidos: orden.PuntosReconocidos,
		Resultado: orden.Resultado, DecisorRef: proyeccion.PrincipalRef,
		PerfilDecisorClave:    proyeccion.PerfilActorClave,
		ValoracionesEvidencia: append([]dominiobolsa.ValoracionEvidencia(nil), orden.ValoracionesEvidencia...),
		MotivoClave:           orden.MotivoClave, Motivo: orden.MotivoDecision,
		FuentesNormativasRefs: append([]string(nil), orden.FuentesNormativasRefs...),
		AutorizacionRef:       proyeccion.AutorizacionRef, FinalidadClave: proyeccion.FinalidadClave,
		CorrelacionRef: proyeccion.CorrelacionRef, DecididaEn: ahora,
	}
	contenido, err := prepararContenidoDecision(agregado, orden.Clase, propuesta)
	if err != nil {
		return RevisionBaremacionAdoptada{}, err
	}
	contenidoClonado, err := contenido.ClonarCanonico()
	if err != nil {
		return RevisionBaremacionAdoptada{}, ErrResultadoBaremacionNoConfiable
	}
	calculoClonado, err := calculo.Clonar()
	if err != nil {
		return RevisionBaremacionAdoptada{}, ErrResultadoBaremacionNoConfiable
	}
	resultado := RevisionBaremacionAdoptada{
		revision: orden.Revision, contenido: contenidoClonado, contextoAdopcion: autorizacionAdopcion, calculo: calculoClonado,
		autorizacionesRefs: autorizaciones,
	}
	if err := validarRevisionAdoptada(resultado); err != nil {
		return RevisionBaremacionAdoptada{}, err
	}
	return resultado, nil
}

// OrdenCodificarDecisionBaremacion selecciona una politica publicada por
// referencia, version y huella exactas.
type OrdenCodificarDecisionBaremacion struct {
	Actor                ActorBaremacion
	Revision             RevisionBaremacionAdoptada
	PoliticaFirmaRef     string
	PoliticaFirmaVersion int
	HuellaPoliticaSHA256 string
}

// DecisionBaremacionCodificada enlaza contenido, politica y bytes canonicos
// sin exponer estos ultimos en formatos de registro.
type DecisionBaremacionCodificada struct {
	revision           RevisionBaremacionAdoptada
	politica           puertosbolsa.PoliticaFirmaBaremacion
	codificacion       puertosbolsa.CodificacionCanonicaDecision
	autorizacionesRefs []string
}

func (DecisionBaremacionCodificada) MarshalJSON() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (*DecisionBaremacionCodificada) UnmarshalJSON([]byte) error { return ErrOrdenBaremacionInvalida }
func (DecisionBaremacionCodificada) MarshalText() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (DecisionBaremacionCodificada) MarshalBinary() ([]byte, error) {
	return nil, ErrOrdenBaremacionInvalida
}
func (DecisionBaremacionCodificada) String() string     { return "[DECISION-BAREMACION-CODIFICADA-OPACA]" }
func (d DecisionBaremacionCodificada) GoString() string { return d.String() }
func (d DecisionBaremacionCodificada) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DecisionBaremacionCodificada) LogValue() slog.Value { return slog.StringValue(d.String()) }

// CodificarDecision obtiene una politica vigente y delega la representacion
// canonica en el conector especializado.
func (s *ServicioBaremacion) CodificarDecision(
	ctx context.Context,
	orden OrdenCodificarDecisionBaremacion,
) (DecisionBaremacionCodificada, error) {
	if validarRevisionAdoptada(orden.Revision) != nil {
		return DecisionBaremacionCodificada{}, ErrOrdenBaremacionInvalida
	}
	if err := s.validarRevisionVigente(orden.Revision.revision); err != nil {
		return DecisionBaremacionCodificada{}, err
	}
	if validarActorRevision(orden.Actor, orden.Revision.revision) != nil ||
		!referenciaAplicacionBaremacionValida(orden.PoliticaFirmaRef) || orden.PoliticaFirmaVersion < 1 ||
		!huellaAplicacionBaremacionValida(orden.HuellaPoliticaSHA256) {
		return DecisionBaremacionCodificada{}, ErrOrdenBaremacionInvalida
	}
	contenido := orden.Revision.contenido
	autorizacionPolitica, err := s.autorizarRevision(ctx, orden.Actor, orden.Revision.revision, puertosbolsa.AccionConsultarPoliticaFirmaBaremacion,
		puertosbolsa.ClaseRecursoPoliticaFirma, orden.PoliticaFirmaRef, contenido.SujetoRef,
		contenido.FinalidadClave, contenido.CorrelacionRef)
	if err != nil {
		return DecisionBaremacionCodificada{}, err
	}
	ahora, err := s.ahora()
	if err != nil {
		return DecisionBaremacionCodificada{}, err
	}
	solicitudPolitica := puertosbolsa.SolicitudObtenerPoliticaFirma{
		Contexto: autorizacionPolitica, Referencia: orden.PoliticaFirmaRef,
		Version: orden.PoliticaFirmaVersion, HuellaEsperadaSHA256: orden.HuellaPoliticaSHA256, VigenteEn: ahora,
	}
	politica, err := s.catalogoFirma.ObtenerPoliticaFirma(ctx, solicitudPolitica)
	if err != nil || politica.ValidarPara(solicitudPolitica) != nil {
		return DecisionBaremacionCodificada{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	autorizacionCodificacion, err := s.autorizarRevision(ctx, orden.Actor, orden.Revision.revision, puertosbolsa.AccionCodificarDecisionBaremacion,
		puertosbolsa.ClaseRecursoDecision, contenido.ID, contenido.SujetoRef,
		contenido.FinalidadClave, contenido.CorrelacionRef)
	if err != nil {
		return DecisionBaremacionCodificada{}, err
	}
	autorizaciones, err := incorporarAutorizacionesBaremacion(
		orden.Revision.autorizacionesRefs, autorizacionPolitica, autorizacionCodificacion,
	)
	if err != nil || autorizacionesRepetidas(orden.Revision.contextoAdopcion, autorizacionPolitica, autorizacionCodificacion) {
		return DecisionBaremacionCodificada{}, ErrResultadoBaremacionNoConfiable
	}
	solicitud := puertosbolsa.SolicitudCodificarDecisionCanonica{
		Contexto: autorizacionCodificacion, AutorizacionDecision: orden.Revision.contextoAdopcion,
		Contenido: contenido, Politica: politica,
	}
	codificacion, err := s.codificador.CodificarDecision(ctx, solicitud)
	if err != nil || codificacion.ValidarPara(solicitud) != nil {
		return DecisionBaremacionCodificada{}, errors.Join(ErrResultadoBaremacionNoConfiable, err)
	}
	politica.ComprobacionesObligatorias = append([]string(nil), politica.ComprobacionesObligatorias...)
	return DecisionBaremacionCodificada{
		revision: orden.Revision, politica: politica, codificacion: codificacion,
		autorizacionesRefs: autorizaciones,
	}, nil
}
