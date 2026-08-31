package application

import (
	"context"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"strconv"
	"strings"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrOrdenTerminalLlamamientoInvalida               = errors.New("bolsa: orden terminal de llamamiento invalida")
	ErrSerializacionOrdenTerminalLlamamientoProhibida = errors.New("bolsa: serializacion de orden terminal de llamamiento prohibida")
)

const (
	moduloOrdenTerminalLlamamiento      = "bolsa"
	tipoRecursoOrdenTerminalLlamamiento = "llamamiento_abierto"
	accionAceptarLlamamiento            = "bolsa.llamamiento.aceptar"
	finalidadAceptarLlamamiento         = "gestion_aceptacion_llamamiento"
	accionRenunciarLlamamiento          = "bolsa.llamamiento.renunciar"
	finalidadRenunciarLlamamiento       = "gestion_renuncia_llamamiento"
	accionExpirarLlamamiento            = "bolsa.llamamiento.expirar"
	finalidadExpirarLlamamiento         = "gestion_expiracion_gobernada_llamamiento"
	etiquetaOrdenTerminalLlamamiento    = "[ORDEN-TERMINAL-LLAMAMIENTO-AUTORIZADA-V2-OPACA]"
)

type especificacionTerminalLlamamiento struct {
	accion    string
	finalidad string
	perfil    aplicacionvec.PerfilProteccionUsoAutorizacion
}

// EmisorOrdenTerminalLlamamientoAutorizadaV2 emite la capacidad efimera V2 sin aplicar B2.
type EmisorOrdenTerminalLlamamientoAutorizadaV2 struct {
	exigidor  aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	politicas [3]aplicacionvec.PoliticaUsoDecisionAutorizacion
}

func NuevoEmisorOrdenTerminalLlamamientoAutorizadaV2(
	exigidor aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
) (*EmisorOrdenTerminalLlamamientoAutorizadaV2, error) {
	if dependenciaOrdenTerminalLlamamientoNula(exigidor) {
		return nil, ErrOrdenTerminalLlamamientoInvalida
	}
	estados := [...]dominiobolsa.EstadoLlamamiento{
		dominiobolsa.EstadoLlamamientoAceptado, dominiobolsa.EstadoLlamamientoRenunciado,
		dominiobolsa.EstadoLlamamientoExpirado,
	}
	emisor := &EmisorOrdenTerminalLlamamientoAutorizadaV2{exigidor: exigidor}
	for indice, estado := range estados {
		politica, err := nuevaPoliticaTerminalLlamamiento(estado)
		if err != nil {
			return nil, ErrOrdenTerminalLlamamientoInvalida
		}
		emisor.politicas[indice] = politica
	}
	return emisor, nil
}

// Emitir coteja la evidencia V2 exacta usando el instante acreditado por la fachada.
func (e *EmisorOrdenTerminalLlamamientoAutorizadaV2) Emitir(
	ctx context.Context,
	actor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	llamamiento dominiobolsa.LlamamientoAbierto,
	versionEsperada uint64,
	terminal dominiobolsa.TerminalLlamamiento,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
) (OrdenTerminalLlamamientoAutorizadaV2, error) {
	vacia := OrdenTerminalLlamamientoAutorizadaV2{}
	if ctx == nil || e == nil || dependenciaOrdenTerminalLlamamientoNula(e.exigidor) {
		return vacia, ErrOrdenTerminalLlamamientoInvalida
	}
	if ctx.Err() != nil {
		return vacia, ErrOrdenTerminalLlamamientoInvalida
	}
	actorCanonico, recurso, especificacion, politica, huellaSolicitud, err :=
		e.prepararEmision(actor, vinculo, llamamiento, versionEsperada, terminal, correlacion, motivo)
	if err != nil {
		return vacia, ErrOrdenTerminalLlamamientoInvalida
	}
	evidencia, err := e.exigidor.ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
		ctx, actorCanonico, vinculo, recurso, correlacion, motivo, politica,
	)
	if err != nil || ctx.Err() != nil {
		return vacia, ErrOrdenTerminalLlamamientoInvalida
	}
	if !evidenciaExactaOrdenTerminal(
		evidencia, actorCanonico, vinculo, recurso, correlacion, motivo,
		especificacion, huellaSolicitud,
	) {
		return vacia, ErrOrdenTerminalLlamamientoInvalida
	}
	orden := OrdenTerminalLlamamientoAutorizadaV2{datos: &datosOrdenTerminalLlamamientoAutorizadaV2{
		llamamiento: llamamiento, versionEsperada: versionEsperada, terminal: terminal,
		correlacion: correlacion, motivo: motivo, huellaSolicitud: huellaSolicitud,
		evidencia: evidencia,
	}}
	if ctx.Err() != nil || orden.validar() != nil {
		return vacia, ErrOrdenTerminalLlamamientoInvalida
	}
	return orden, nil
}

func (e *EmisorOrdenTerminalLlamamientoAutorizadaV2) prepararEmision(
	actor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	llamamiento dominiobolsa.LlamamientoAbierto,
	versionEsperada uint64,
	terminal dominiobolsa.TerminalLlamamiento,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
) (dominiovec.ContextoActor, dominiovec.RecursoAutorizable,
	especificacionTerminalLlamamiento, aplicacionvec.PoliticaUsoDecisionAutorizacion, string, error,
) {
	actorCanonico, err := actor.Clonar()
	if err != nil || vinculo.ValidarPara(actorCanonico) != nil || terminal.Validar() != nil ||
		correlacion.Validar() != nil || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return dominiovec.ContextoActor{}, dominiovec.RecursoAutorizable{},
			especificacionTerminalLlamamiento{}, aplicacionvec.PoliticaUsoDecisionAutorizacion{}, "", err
	}
	recurso, err := recursoOrdenTerminalLlamamiento(llamamiento, versionEsperada, terminal)
	if err != nil {
		return dominiovec.ContextoActor{}, dominiovec.RecursoAutorizable{},
			especificacionTerminalLlamamiento{}, aplicacionvec.PoliticaUsoDecisionAutorizacion{}, "", err
	}
	especificacion, politica, ok := e.politicaPara(terminal.Estado)
	if !ok || !credencialesCompatiblesConPerfil(actorCanonico, vinculo, especificacion.perfil) ||
		!referenciasOperacionYCorrelacionDistintas(terminal, correlacion) {
		return dominiovec.ContextoActor{}, dominiovec.RecursoAutorizable{},
			especificacionTerminalLlamamiento{}, aplicacionvec.PoliticaUsoDecisionAutorizacion{}, "",
			ErrOrdenTerminalLlamamientoInvalida
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV2(
		dominiovec.DatosSolicitudAutorizacionLigadaV2{
			ContextoActor: actorCanonico, VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo: motivo, Accion: especificacion.accion, Recurso: recurso,
			Finalidad: especificacion.finalidad, Correlacion: correlacion,
		},
	)
	if err != nil {
		return dominiovec.ContextoActor{}, dominiovec.RecursoAutorizable{},
			especificacionTerminalLlamamiento{}, aplicacionvec.PoliticaUsoDecisionAutorizacion{}, "", err
	}
	huellaSolicitud, err := dominiovec.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	return actorCanonico, recurso, especificacion, politica, huellaSolicitud, err
}

func (e *EmisorOrdenTerminalLlamamientoAutorizadaV2) politicaPara(
	estado dominiobolsa.EstadoLlamamiento,
) (especificacionTerminalLlamamiento, aplicacionvec.PoliticaUsoDecisionAutorizacion, bool) {
	especificacion, ok := especificacionParaTerminalLlamamiento(estado)
	if !ok || e == nil {
		return especificacionTerminalLlamamiento{}, aplicacionvec.PoliticaUsoDecisionAutorizacion{}, false
	}
	switch estado {
	case dominiobolsa.EstadoLlamamientoAceptado:
		return especificacion, e.politicas[0], true
	case dominiobolsa.EstadoLlamamientoRenunciado:
		return especificacion, e.politicas[1], true
	case dominiobolsa.EstadoLlamamientoExpirado:
		return especificacion, e.politicas[2], true
	default:
		return especificacionTerminalLlamamiento{}, aplicacionvec.PoliticaUsoDecisionAutorizacion{}, false
	}
}

func nuevaPoliticaTerminalLlamamiento(
	estado dominiobolsa.EstadoLlamamiento,
) (aplicacionvec.PoliticaUsoDecisionAutorizacion, error) {
	especificacion, ok := especificacionParaTerminalLlamamiento(estado)
	if !ok {
		return aplicacionvec.PoliticaUsoDecisionAutorizacion{}, ErrOrdenTerminalLlamamientoInvalida
	}
	return aplicacionvec.NuevaPoliticaUsoDecisionAutorizacion(
		especificacion.accion, moduloOrdenTerminalLlamamiento,
		tipoRecursoOrdenTerminalLlamamiento, especificacion.finalidad, nil,
		especificacion.perfil,
	)
}

func especificacionParaTerminalLlamamiento(
	estado dominiobolsa.EstadoLlamamiento,
) (especificacionTerminalLlamamiento, bool) {
	switch estado {
	case dominiobolsa.EstadoLlamamientoAceptado:
		return especificacionTerminalLlamamiento{
			accion: accionAceptarLlamamiento, finalidad: finalidadAceptarLlamamiento,
			perfil: aplicacionvec.PerfilProteccionUsoAutorizacionOrdinario,
		}, true
	case dominiobolsa.EstadoLlamamientoRenunciado:
		return especificacionTerminalLlamamiento{
			accion: accionRenunciarLlamamiento, finalidad: finalidadRenunciarLlamamiento,
			perfil: aplicacionvec.PerfilProteccionUsoAutorizacionOrdinario,
		}, true
	case dominiobolsa.EstadoLlamamientoExpirado:
		return especificacionTerminalLlamamiento{
			accion: accionExpirarLlamamiento, finalidad: finalidadExpirarLlamamiento,
			perfil: aplicacionvec.PerfilProteccionUsoAutorizacionInternoAlto,
		}, true
	default:
		return especificacionTerminalLlamamiento{}, false
	}
}

func recursoOrdenTerminalLlamamiento(
	llamamiento dominiobolsa.LlamamientoAbierto,
	versionEsperada uint64,
	terminal dominiobolsa.TerminalLlamamiento,
) (dominiovec.RecursoAutorizable, error) {
	datos := llamamiento.Datos()
	if llamamiento.Validar() != nil || llamamiento.EsTerminal() ||
		llamamiento.Estado() != dominiobolsa.EstadoLlamamientoAbierto ||
		versionEsperada == 0 || versionEsperada != datos.Version || terminal.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, ErrOrdenTerminalLlamamientoInvalida
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: datos.LlamamientoRef, ModuloID: moduloOrdenTerminalLlamamiento,
		Tipo: tipoRecursoOrdenTerminalLlamamiento,
		Ambitos: map[string]string{
			"bolsa_ref": datos.BolsaRef, "necesidad_ref": datos.NecesidadRef,
			"propuesta_ref": datos.PropuestaRef,
		},
		Atributos: map[string]string{
			"version_esperada": strconv.FormatUint(versionEsperada, 10),
			"estado_terminal":  string(terminal.Estado), "operacion_ref": terminal.OperacionRef,
		},
	}
	if recurso.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, ErrOrdenTerminalLlamamientoInvalida
	}
	return recurso, nil
}

func evidenciaExactaOrdenTerminal(
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	actor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	recurso dominiovec.RecursoAutorizable,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	especificacion especificacionTerminalLlamamiento,
	huellaSolicitud string,
) bool {
	primeraProyeccion, err := evidencia.Datos()
	if err != nil || primeraProyeccion.VerificadaEn.IsZero() ||
		evidencia.ValidarEn(primeraProyeccion.VerificadaEn) != nil ||
		evidencia.ValidarMotivo(motivo) != nil {
		return false
	}
	proyeccion, err := evidencia.Datos()
	if err != nil || !vinculo.VigenteEn(proyeccion.VerificadaEn, actor) ||
		!vinculo.CoincideExactamenteCon(proyeccion.Decision.VinculoAutenticacionActor) {
		return false
	}
	return decisionExactaOrdenTerminal(
		proyeccion.Decision, actor.Principal.ID, actor.PerfilActivoRef, recurso,
		correlacion, motivo, especificacion, huellaSolicitud,
	)
}

func decisionExactaOrdenTerminal(
	decision dominiovec.DecisionAutorizacion,
	principalID string,
	perfilActivoRef string,
	recurso dominiovec.RecursoAutorizable,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	motivo dominiovec.ReferenciaEntradaCatalogo,
	especificacion especificacionTerminalLlamamiento,
	huellaSolicitud string,
) bool {
	correlacionRef, errCorrelacion := correlacion.ValorCanonico()
	huellaContexto, errContexto := recurso.HuellaContextoAutorizacionSHA256()
	huellaMotivo, errMotivo := dominiovec.HuellaSHA256MotivoAutorizacionV2(motivo)
	return errCorrelacion == nil && errContexto == nil && errMotivo == nil &&
		decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() == nil && decision.Concedida &&
		decision.Codigo == "concedida" && decision.PrincipalID == principalID &&
		decision.PerfilActivoRef == perfilActivoRef && decision.Accion == especificacion.accion &&
		decision.RecursoRef == recurso.Referencia && decision.ModuloID == recurso.ModuloID &&
		decision.TipoRecurso == recurso.Tipo && decision.ContextoRecursoHuellaSHA256 == huellaContexto &&
		decision.Finalidad == especificacion.finalidad && decision.CorrelacionRef == correlacionRef &&
		decision.EsquemaHuellaSolicitud == dominiovec.EsquemaHuellaSolicitudAutorizacionV2 &&
		decision.SolicitudHuellaSHA256 == huellaSolicitud &&
		decision.EsquemaHuellaMotivo == dominiovec.EsquemaHuellaMotivoAutorizacionV2 &&
		decision.MotivoHuellaSHA256 == huellaMotivo && len(decision.CamposPermitidos) == 0 &&
		len(decision.Obligaciones) == 0 &&
		perfilDecisionOrdenTerminalValido(especificacion.perfil, decision)
}

func credencialesCompatiblesConPerfil(
	actor dominiovec.ContextoActor,
	vinculo dominiovec.VinculoAutenticacionActorV1,
	perfil aplicacionvec.PerfilProteccionUsoAutorizacion,
) bool {
	datos, err := vinculo.Datos()
	if err != nil || actor.Principal.AuthMethod == dominiovec.AuthMethodDemo ||
		datos.MetodoObservado == dominiovec.AuthMethodDemo ||
		actor.Principal.AuthAssurance != datos.GarantiaObservada {
		return false
	}
	return superficieCompatibleConPerfil(perfil, datos)
}

func perfilDecisionOrdenTerminalValido(
	perfil aplicacionvec.PerfilProteccionUsoAutorizacion,
	decision dominiovec.DecisionAutorizacion,
) bool {
	datos, err := decision.VinculoAutenticacionActor.Datos()
	return err == nil && decision.PrincipalID == datos.PrincipalID &&
		decision.PerfilActivoRef == datos.PerfilActivoRef &&
		dominiovec.CumpleGarantiaAutenticacion(datos.GarantiaObservada, decision.GarantiaMinima) &&
		superficieCompatibleConPerfil(perfil, datos)
}

func superficieCompatibleConPerfil(
	perfil aplicacionvec.PerfilProteccionUsoAutorizacion,
	datos dominiovec.DatosVinculoAutenticacionActorV1,
) bool {
	if perfil == aplicacionvec.PerfilProteccionUsoAutorizacionOrdinario {
		return datos.Superficie == dominiovec.SuperficieAutenticacionExternaPersonalV1 &&
			!datos.CuentaPrivilegiada
	}
	if perfil != aplicacionvec.PerfilProteccionUsoAutorizacionInternoAlto ||
		datos.GarantiaObservada != dominiovec.AuthAssuranceHigh {
		return false
	}
	return (datos.Superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 &&
		!datos.CuentaPrivilegiada) ||
		(datos.Superficie == dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1 &&
			datos.CuentaPrivilegiada)
}

func referenciasOperacionYCorrelacionDistintas(
	terminal dominiobolsa.TerminalLlamamiento,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
) bool {
	referencia, err := correlacion.ValorCanonico()
	return err == nil && terminal.OperacionRef != referencia
}

type datosOrdenTerminalLlamamientoAutorizadaV2 struct {
	llamamiento     dominiobolsa.LlamamientoAbierto
	versionEsperada uint64
	terminal        dominiobolsa.TerminalLlamamiento
	correlacion     dominiovec.ReferenciaCorrelacionAutorizacionV2
	motivo          dominiovec.ReferenciaEntradaCatalogo
	huellaSolicitud string
	evidencia       puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
}

// OrdenTerminalLlamamientoAutorizadaV2 es efimera, opaca y no reconstruible.
type OrdenTerminalLlamamientoAutorizadaV2 struct {
	bloqueoOrdenTerminal
	datos *datosOrdenTerminalLlamamientoAutorizadaV2
}

func (o OrdenTerminalLlamamientoAutorizadaV2) Validar() error { return o.validar() }

func (o OrdenTerminalLlamamientoAutorizadaV2) validar() error {
	if o.datos == nil {
		return ErrOrdenTerminalLlamamientoInvalida
	}
	recurso, err := recursoOrdenTerminalLlamamiento(
		o.datos.llamamiento, o.datos.versionEsperada, o.datos.terminal,
	)
	especificacion, ok := especificacionParaTerminalLlamamiento(o.datos.terminal.Estado)
	if err != nil || !ok || !referenciasOperacionYCorrelacionDistintas(
		o.datos.terminal, o.datos.correlacion,
	) || !dominiovec.ReferenciaMotivoAutorizacionV2Valida(o.datos.motivo) {
		return ErrOrdenTerminalLlamamientoInvalida
	}
	proyeccion, err := o.datos.evidencia.Datos()
	if err != nil || proyeccion.VerificadaEn.IsZero() ||
		o.datos.evidencia.ValidarEn(proyeccion.VerificadaEn) != nil ||
		o.datos.evidencia.ValidarMotivo(o.datos.motivo) != nil {
		return ErrOrdenTerminalLlamamientoInvalida
	}
	proyeccion, err = o.datos.evidencia.Datos()
	if err != nil || !decisionExactaOrdenTerminal(
		proyeccion.Decision, proyeccion.Decision.PrincipalID,
		proyeccion.Decision.PerfilActivoRef, recurso, o.datos.correlacion,
		o.datos.motivo, especificacion, o.datos.huellaSolicitud,
	) {
		return ErrOrdenTerminalLlamamientoInvalida
	}
	return nil
}

// ReacreditarYProyectar entrega a B3 solo agregado, version y terminal ligados.
func (o OrdenTerminalLlamamientoAutorizadaV2) ReacreditarYProyectar() (
	ProyeccionOrdenTerminalLlamamientoAutorizadaV2,
	error,
) {
	if o.validar() != nil {
		return ProyeccionOrdenTerminalLlamamientoAutorizadaV2{},
			ErrOrdenTerminalLlamamientoInvalida
	}
	proyeccion := ProyeccionOrdenTerminalLlamamientoAutorizadaV2{
		llamamiento: o.datos.llamamiento, versionEsperada: o.datos.versionEsperada,
		terminal: o.datos.terminal,
	}
	proyeccion.sello = selloProyeccionOrdenTerminal(
		proyeccion.llamamiento, proyeccion.versionEsperada, proyeccion.terminal,
	)
	return proyeccion, nil
}

// ProyeccionOrdenTerminalLlamamientoAutorizadaV2 es la vista defensiva minima de B3.
type ProyeccionOrdenTerminalLlamamientoAutorizadaV2 struct {
	bloqueoOrdenTerminal
	llamamiento     dominiobolsa.LlamamientoAbierto
	versionEsperada uint64
	terminal        dominiobolsa.TerminalLlamamiento
	sello           [sha256.Size]byte
}

func (p ProyeccionOrdenTerminalLlamamientoAutorizadaV2) Datos() (
	dominiobolsa.LlamamientoAbierto,
	uint64,
	dominiobolsa.TerminalLlamamiento,
	error,
) {
	if p.llamamiento.Validar() != nil || p.llamamiento.EsTerminal() ||
		p.versionEsperada == 0 || p.versionEsperada != p.llamamiento.Datos().Version ||
		p.terminal.Validar() != nil || p.sello != selloProyeccionOrdenTerminal(
		p.llamamiento, p.versionEsperada, p.terminal,
	) {
		return dominiobolsa.LlamamientoAbierto{}, 0, dominiobolsa.TerminalLlamamiento{},
			ErrOrdenTerminalLlamamientoInvalida
	}
	return p.llamamiento, p.versionEsperada, p.terminal, nil
}

func selloProyeccionOrdenTerminal(
	llamamiento dominiobolsa.LlamamientoAbierto,
	versionEsperada uint64,
	terminal dominiobolsa.TerminalLlamamiento,
) [sha256.Size]byte {
	datos := llamamiento.Datos()
	preimagen := strings.Join([]string{
		datos.LlamamientoRef, datos.BolsaRef, datos.NecesidadRef, datos.PropuestaRef,
		strconv.FormatUint(datos.Version, 10), strconv.FormatUint(versionEsperada, 10),
		string(llamamiento.Estado()), string(terminal.Estado), terminal.OperacionRef,
	}, "\x00")
	return sha256.Sum256([]byte(preimagen))
}

type bloqueoOrdenTerminal struct{}

func (bloqueoOrdenTerminal) String() string     { return etiquetaOrdenTerminalLlamamiento }
func (b bloqueoOrdenTerminal) GoString() string { return b.String() }
func (b bloqueoOrdenTerminal) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoOrdenTerminal) LogValue() slog.Value       { return slog.StringValue(b.String()) }
func (bloqueoOrdenTerminal) MarshalJSON() ([]byte, error) { return nil, errorCodecOrdenTerminal() }
func (*bloqueoOrdenTerminal) UnmarshalJSON([]byte) error  { return errorCodecOrdenTerminal() }
func (bloqueoOrdenTerminal) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return errorCodecOrdenTerminal()
}
func (*bloqueoOrdenTerminal) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return errorCodecOrdenTerminal()
}
func (bloqueoOrdenTerminal) MarshalText() ([]byte, error)         { return nil, errorCodecOrdenTerminal() }
func (*bloqueoOrdenTerminal) UnmarshalText([]byte) error          { return errorCodecOrdenTerminal() }
func (bloqueoOrdenTerminal) MarshalBinary() ([]byte, error)       { return nil, errorCodecOrdenTerminal() }
func (*bloqueoOrdenTerminal) UnmarshalBinary([]byte) error        { return errorCodecOrdenTerminal() }
func (bloqueoOrdenTerminal) GobEncode() ([]byte, error)           { return nil, errorCodecOrdenTerminal() }
func (*bloqueoOrdenTerminal) GobDecode([]byte) error              { return errorCodecOrdenTerminal() }
func (bloqueoOrdenTerminal) MarshalCBOR() ([]byte, error)         { return nil, errorCodecOrdenTerminal() }
func (*bloqueoOrdenTerminal) UnmarshalCBOR([]byte) error          { return errorCodecOrdenTerminal() }
func (bloqueoOrdenTerminal) MarshalYAML() (any, error)            { return nil, errorCodecOrdenTerminal() }
func (*bloqueoOrdenTerminal) UnmarshalYAML(func(any) error) error { return errorCodecOrdenTerminal() }
func errorCodecOrdenTerminal() error                              { return ErrSerializacionOrdenTerminalLlamamientoProhibida }

func dependenciaOrdenTerminalLlamamientoNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
