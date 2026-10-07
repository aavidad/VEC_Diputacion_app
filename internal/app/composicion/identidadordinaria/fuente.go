// Package identidadordinaria resuelve la identidad interna ordinaria para
// consumidores con una audiencia y una autorización V3 propias.
package identidadordinaria

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrIdentidadOrdinariaNoDisponible = errors.New("identidad_ordinaria_no_disponible")

// errorFuente oculta los mensajes de las autoridades subordinadas al cruzar
// la frontera HTTP. La causa sigue disponible para clasificación interna con
// errors.Is/As, sin aparecer en Error().
type errorFuente struct{ causa error }

func (e *errorFuente) Error() string { return ErrIdentidadOrdinariaNoDisponible.Error() }
func (e *errorFuente) Unwrap() []error {
	return []error{ErrIdentidadOrdinariaNoDisponible, e.causa}
}

func falloFuente(causa error) error {
	if causa == nil {
		return ErrIdentidadOrdinariaNoDisponible
	}
	return &errorFuente{causa: causa}
}

// PerfilNominal fija la única selección permitida para una cuenta. La
// provisión procede de configuración privada, nunca de datos HTTP.
type PerfilNominal struct {
	PerfilActivoRef string
	VersionRolRef   string
}

type ConfiguracionFuente struct {
	Identidad    *httpseguridad.ServicioIdentidad
	Revalidador  core.RevalidadorAutenticacionActorV1
	Resolutor    core.ResolutorContextoActorRegistradoV2
	Autorizacion ports.FuenteAutorizacion
	Reloj        core.RelojVinculoAutenticacionActorV2
	PorCuenta    map[string]PerfilNominal
}

// Fuente conserva sólo la selección nominal. No conserva contextos,
// instantáneas ni decisiones entre peticiones.
type Fuente struct {
	identidad    *httpseguridad.ServicioIdentidad
	revalidador  core.RevalidadorAutenticacionActorV1
	resolutor    core.ResolutorContextoActorRegistradoV2
	autorizacion ports.FuenteAutorizacion
	reloj        core.RelojVinculoAutenticacionActorV2
	porCuenta    map[string]PerfilNominal
}

func NuevaFuente(c ConfiguracionFuente) (*Fuente, error) {
	if c.Identidad == nil || dependenciaNula(c.Revalidador) || dependenciaNula(c.Resolutor) ||
		dependenciaNula(c.Autorizacion) || dependenciaNula(c.Reloj) ||
		len(c.PorCuenta) == 0 || len(c.PorCuenta) > 512 {
		return nil, ErrIdentidadOrdinariaNoDisponible
	}
	porCuenta := make(map[string]PerfilNominal, len(c.PorCuenta))
	for cuenta, nominal := range c.PorCuenta {
		if (core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{
			CuentaRef: cuenta, Metodo: core.AuthMethodKerberos, Garantia: core.AuthAssuranceHigh,
		}, PerfilActivoRef: nominal.PerfilActivoRef}).Validar() != nil ||
			!versionRolRefValida(nominal.VersionRolRef) {
			return nil, ErrIdentidadOrdinariaNoDisponible
		}
		porCuenta[cuenta] = nominal
	}
	return &Fuente{c.Identidad, c.Revalidador, c.Resolutor, c.Autorizacion, c.Reloj, porCuenta}, nil
}

// Resolver usa exclusivamente la cápsula vinculada a la petición por la
// misma instancia de ServicioIdentidad. El consumidor debe emitir y consumir
// su decisión V3 para cada acción, recurso y finalidad concretos.
func (f *Fuente) Resolver(ctx context.Context) (
	core.VinculoAutenticacionActorV2,
	core.ResultadoContextoActorRegistradoV2,
	core.InstantaneaAutorizacion,
	error,
) {
	var vinculo core.VinculoAutenticacionActorV2
	var resultado core.ResultadoContextoActorRegistradoV2
	var snapshot core.InstantaneaAutorizacion
	if f == nil || f.identidad == nil || dependenciaNula(f.revalidador) ||
		dependenciaNula(f.resolutor) || dependenciaNula(f.autorizacion) ||
		dependenciaNula(f.reloj) || ctx == nil {
		return vinculo, resultado, snapshot, ErrIdentidadOrdinariaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vinculo, resultado, snapshot, falloFuente(err)
	}
	cuenta, auditoria, err := f.identidad.ExtraerCapsulaIdentidadPeticion(ctx)
	if err != nil {
		return vinculo, resultado, snapshot, falloFuente(err)
	}
	if err := cuenta.Validar(); err != nil {
		return vinculo, resultado, snapshot, falloFuente(err)
	}
	if cuenta.Garantia != core.AuthAssuranceHigh ||
		(cuenta.Metodo != core.AuthMethodKerberos && cuenta.Metodo != core.AuthMethodCertificate) ||
		auditoria.CuentaRef() != cuenta.CuentaRef ||
		auditoria.CuentaOrdinariaRef() != cuenta.CuentaRef || auditoria.CuentaPrivilegiada() ||
		auditoria.Superficie() != httpseguridad.SuperficieInternaCorporativa ||
		auditoria.ControlSesionEstado() != httpseguridad.EstadoControlSesionActiva ||
		auditoria.GarantiaObservada() != cuenta.Garantia ||
		auditoria.MetodoObservado() != cuenta.Metodo ||
		!factoresCorporativosValidos(auditoria) {
		return vinculo, resultado, snapshot, ErrIdentidadOrdinariaNoDisponible
	}
	nominal, ok := f.porCuenta[cuenta.CuentaRef]
	if !ok {
		return vinculo, resultado, snapshot, ErrIdentidadOrdinariaNoDisponible
	}
	vinculo, resultado, err = core.CrearVinculoAutenticacionActorV2ConResultado(
		ctx, f.revalidador,
		core.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: auditoria.AutenticacionRef(), SesionRef: auditoria.SesionRef(),
		},
		f.resolutor, core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: nominal.PerfilActivoRef}, f.reloj,
	)
	if err != nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, snapshot, falloFuente(err)
	}
	if err := ctx.Err(); err != nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, snapshot, falloFuente(err)
	}
	datos, err := vinculo.Datos()
	if err != nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, snapshot, falloFuente(err)
	}
	if datos.CuentaRef != cuenta.CuentaRef || datos.CuentaOrdinariaRef != cuenta.CuentaRef ||
		datos.CuentaPrivilegiada || datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		datos.GarantiaObservada != core.AuthAssuranceHigh || datos.MetodoObservado != cuenta.Metodo ||
		datos.AutenticacionRef != auditoria.AutenticacionRef() || datos.SesionRef != auditoria.SesionRef() ||
		datos.PerfilActivoRef != nominal.PerfilActivoRef ||
		datos.PoliticaGarantiaRef != auditoria.PoliticaGarantiaRef() ||
		datos.PoliticaGarantiaHuellaSHA256 != auditoria.PoliticaGarantiaHuellaSHA256() ||
		!vinculo.VigenteEn(f.reloj.Ahora(), resultado) {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, snapshot, ErrIdentidadOrdinariaNoDisponible
	}
	snapshot, err = f.autorizacion.ObtenerInstantaneaAutorizacion(ctx, datos.PrincipalID, nominal.PerfilActivoRef)
	if err != nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, core.InstantaneaAutorizacion{}, falloFuente(err)
	}
	if err := ctx.Err(); err != nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, core.InstantaneaAutorizacion{}, falloFuente(err)
	}
	if err := snapshot.Validar(); err != nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, core.InstantaneaAutorizacion{}, falloFuente(err)
	}
	ahora := f.reloj.Ahora()
	if snapshot.AsignacionPerfil.PrincipalID != datos.PrincipalID ||
		snapshot.AsignacionPerfil.PerfilActivoRef != nominal.PerfilActivoRef ||
		snapshot.AsignacionPerfil.VersionRolRef != nominal.VersionRolRef ||
		snapshot.VersionRol.Referencia() != nominal.VersionRolRef ||
		snapshot.VersionRol.Estado != core.EstadoVersionRolPublicada ||
		snapshot.ControlVigenciaVersionRol.Estado != core.EstadoControlVigenciaVersionRolHabilitada ||
		!snapshot.AsignacionPerfil.VigenteEn(ahora) ||
		ahora.Before(snapshot.VersionRol.PublicadaEn) ||
		ahora.Before(snapshot.ControlVigenciaVersionRol.ActualizadoEn) ||
		!vinculo.VigenteEn(ahora, resultado) {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, core.InstantaneaAutorizacion{}, ErrIdentidadOrdinariaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, core.InstantaneaAutorizacion{}, falloFuente(err)
	}
	return vinculo, resultado, snapshot, nil
}

func factoresCorporativosValidos(a httpseguridad.ContextoAuditoriaAutenticada) bool {
	factores := a.Factores()
	if len(factores) != 2 {
		return false
	}
	var kerberos, certificado bool
	for _, factor := range factores {
		if factor.EvidenciaRef == "" || factor.GrupoCriptograficoRef == "" ||
			factor.VerificadoEn.IsZero() {
			return false
		}
		switch factor.Metodo {
		case httpseguridad.MetodoKerberos:
			if kerberos {
				return false
			}
			kerberos = true
		case httpseguridad.MetodoCertificado:
			if certificado {
				return false
			}
			certificado = true
		default:
			return false
		}
	}
	return kerberos && certificado && factores[0].GrupoCriptograficoRef != factores[1].GrupoCriptograficoRef
}

func versionRolRefValida(ref string) bool {
	if len(ref) < len("rol:a:v1") || len(ref) > 512 || strings.TrimSpace(ref) != ref ||
		!strings.HasPrefix(ref, "rol:") || strings.ContainsAny(ref, "*\x00\r\n\t") {
		return false
	}
	parte := strings.LastIndex(ref, ":v")
	if parte <= len("rol:") || parte+2 >= len(ref) || ref[parte+2] == '0' {
		return false
	}
	for _, digito := range ref[parte+2:] {
		if digito < '0' || digito > '9' {
			return false
		}
	}
	return true
}

func dependenciaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}
