package application

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// ResolutorActoSubidaOriginalC8 es inmutable por petición. El adaptador del
// puerto común de Documentos podrá convertir su salida a ActoSubidaLocal una
// vez integrado D21; este contrato no simula la subida ni su admisión.
type ResolutorActoSubidaOriginalC8 struct {
	selector   ports.SelectorSubidaOriginalC8
	capturador ports.CapturadorPreparacionSubidaOriginalC8
	fuente     ports.FuenteActoSubidaOriginalC8
	reloj      ports.Reloj
}

var contextoPreparacionC8 = regexp.MustCompile(`^rca_[-A-Za-z0-9_]{22,128}$`)

func NuevoResolutorActoSubidaOriginalC8(selector ports.SelectorSubidaOriginalC8,
	capturador ports.CapturadorPreparacionSubidaOriginalC8,
	fuente ports.FuenteActoSubidaOriginalC8, reloj ports.Reloj) (*ResolutorActoSubidaOriginalC8, error) {
	if !domain.SolicitudPermisoRefValida(selector.SolicitudRef) || selector.VersionEsperada < 1 ||
		selector.VersionEsperada > 1000 || dependenciaRemotaNula(capturador) ||
		dependenciaRemotaNula(fuente) || dependenciaRemotaNula(reloj) {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &ResolutorActoSubidaOriginalC8{selector: selector, capturador: capturador,
		fuente: fuente, reloj: reloj}, nil
}

func (r *ResolutorActoSubidaOriginalC8) ResolverActoSubidaOriginal(ctx context.Context,
	actor vecdomain.ResultadoContextoActorRegistradoV2, vinculo vecdomain.VinculoAutenticacionActorV2,
	claveIdempotencia string) (ports.ActoSubidaOriginalC8, error) {
	var vacio ports.ActoSubidaOriginalC8
	ahora := time.Time{}
	if r != nil && !dependenciaRemotaNula(r.reloj) {
		ahora = r.reloj.AhoraUTC().UTC().Truncate(time.Microsecond)
	}
	if r == nil || ctx == nil || ctx.Err() != nil || dependenciaRemotaNula(r.capturador) || dependenciaRemotaNula(r.fuente) ||
		dependenciaRemotaNula(r.reloj) || !domain.SolicitudPermisoRefValida(r.selector.SolicitudRef) ||
		r.selector.VersionEsperada < 1 || r.selector.VersionEsperada > 1000 ||
		claveIdempotencia == "" || len(claveIdempotencia) > 256 || actor.Validar() != nil ||
		vinculo.ValidarPara(actor) != nil || !vinculo.VigenteEn(ahora, actor) {
		return vacio, ports.ErrJustificacionNoDisponible
	}
	empleados, err := actor.Contexto.Referencias(vecdomain.TipoReferenciaContextoActorEmpleado)
	if err != nil || len(empleados) != 1 {
		return vacio, ports.ErrJustificacionNoDisponible
	}
	captura, err := r.capturador.CapturarPreparacionSubidaOriginal(ctx, actor, vinculo, r.selector)
	vinculoCanonico, errorCanonico := json.Marshal(vinculo)
	if err != nil || errorCanonico != nil || !bytes.Equal(captura.ContextoCanonico, actor.RepresentacionCanonica) ||
		!bytes.Equal(captura.VinculoCanonico, vinculoCanonico) {
		return vacio, ports.ErrJustificacionNoDisponible
	}
	var orden map[string]json.RawMessage
	var accion, recursoRef, contexto string
	var version int64
	if json.Unmarshal(captura.Orden, &orden) != nil ||
		json.Unmarshal(orden["accion"], &accion) != nil ||
		json.Unmarshal(orden["recurso_ref"], &recursoRef) != nil ||
		json.Unmarshal(orden["registro_contexto_ref"], &contexto) != nil ||
		json.Unmarshal(orden["version_esperada"], &version) != nil ||
		accion != "cronos.justificacion.subida.preparar" || recursoRef != r.selector.SolicitudRef ||
		version != r.selector.VersionEsperada || contexto != actor.RegistroContextoRef ||
		!contextoPreparacionC8.MatchString(contexto) {
		return vacio, ports.ErrJustificacionNoDisponible
	}
	estado, err := r.fuente.ConsultarActoSubidaOriginal(ctx, r.selector, captura)
	if err != nil || estado.SolicitudRef != r.selector.SolicitudRef ||
		estado.Version != r.selector.VersionEsperada || estado.PersonaRef != actor.Contexto.PersonaRef ||
		estado.EmpleadoRef != empleados[0] || estado.Auditoria.Referencia == "" ||
		estado.Auditoria.HuellaSHA256 == "" {
		return vacio, ports.ErrJustificacionNoDisponible
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: estado.SolicitudRef,
		ModuloID: "cronos", Tipo: "justificacion",
		Ambitos: map[string]string{"persona_ref": estado.PersonaRef, "empleado_ref": estado.EmpleadoRef}}
	if recurso.Validar() != nil {
		return vacio, ports.ErrJustificacionNoDisponible
	}
	return ports.ActoSubidaOriginalC8{Recurso: recurso, PersonaRef: estado.PersonaRef,
		Finalidad: "custodiar_justificacion_cronos", Clasificacion: "confidencial",
		VersionDueno: uint64(estado.Version)}, nil
}
