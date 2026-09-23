// Package composicion conecta dependencias ya autorizadas del borrador de Dietas.
package composicion

import (
	"context"
	"errors"
	"reflect"

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrIdentidadEfectivaBorradorNoDisponible = errors.New("dietas: identidad efectiva de borrador no disponible")

// IdentidadRegistradaBorrador procede exclusivamente de la frontera de sesión
// corporativa. FechaReferencia también es autoridad de canal: nunca se toma del
// navegador para una lectura de Personal.
type IdentidadRegistradaBorrador struct {
	Contexto        vecdomain.ResultadoContextoActorRegistradoV2
	Vinculo         vecdomain.VinculoAutenticacionActorV2
	FechaReferencia personaldomain.FechaCivil
}

type ResolutorIdentidadRegistradaBorrador interface {
	ResolverIdentidadRegistradaBorrador(context.Context) (IdentidadRegistradaBorrador, error)
}

// ProveedorAutorizacionBorrador recibe el efecto canónico, no referencias
// libres. La implementación productiva emite el material V3 que SQL volverá a
// verificar y consumir junto al efecto.
type ProveedorAutorizacionBorrador interface {
	AutorizarBorradorPropio(context.Context, dietasports.EfectoAutorizacionBorrador) (dietasports.AutorizacionBorradorDurable, error)
}

type ResolutorIdentidadEfectivaBorrador struct {
	identidad ResolutorIdentidadRegistradaBorrador
	personal  interface {
		ConsultarPropiasParaDietas(context.Context, personaldomain.SolicitudConsultaRelacionPropia) (personalports.ResultadoConsultaRelacionPropia, error)
	}
	autorizacion ProveedorAutorizacionBorrador
}

func NuevoResolutorIdentidadEfectivaBorrador(i ResolutorIdentidadRegistradaBorrador, p *personalapp.ServicioConsultaRelacionEmpleado, a ProveedorAutorizacionBorrador) (*ResolutorIdentidadEfectivaBorrador, error) {
	if nulo(i) || p == nil || nulo(a) {
		return nil, ErrIdentidadEfectivaBorradorNoDisponible
	}
	return &ResolutorIdentidadEfectivaBorrador{identidad: i, personal: p, autorizacion: a}, nil
}

func (r *ResolutorIdentidadEfectivaBorrador) ResolverIdentidadEfectivaBorrador(ctx context.Context, solicitud dietasports.SolicitudOperacionBorrador) (dietasports.IdentidadEfectivaBorrador, error) {
	var cero dietasports.IdentidadEfectivaBorrador
	if r == nil || ctx == nil || nulo(r.identidad) || nulo(r.personal) || nulo(r.autorizacion) {
		return cero, ErrIdentidadEfectivaBorradorNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	base, err := r.identidad.ResolverIdentidadRegistradaBorrador(ctx)
	if err != nil {
		return cero, opaco(ctx, err)
	}
	if base.Contexto.Validar() != nil || base.Vinculo.ValidarPara(base.Contexto) != nil || base.FechaReferencia.Validar() != nil {
		return cero, dietasports.ErrAccesoBorradorDenegado
	}
	fecha, seleccion, err := seleccionSolicitud(solicitud, base.FechaReferencia)
	if err != nil {
		return cero, dietasports.ErrRelacionNoValida
	}
	operacion := personaldomain.OperacionListaRelacionPropia
	if seleccion != "" {
		operacion = personaldomain.OperacionDetalleRelacionPropia
	}
	resultado, err := r.personal.ConsultarPropiasParaDietas(ctx, personaldomain.SolicitudConsultaRelacionPropia{FechaReferencia: fecha, Operacion: operacion, RelacionRef: seleccion, Actor: base.Contexto.Contexto})
	if err != nil {
		return cero, opaco(ctx, err)
	}
	relacion, err := relacionUnica(resultado, seleccion)
	if err != nil {
		return cero, err
	}
	acreditada := traducirRelacion(relacion)
	sello := selloRelacion(relacion, fecha)
	efecto, err := dietasapp.ConstruirEfectoAutorizacionBorrador(base.Contexto, acreditada, sello, solicitud)
	if err != nil {
		return cero, dietasports.ErrAccesoBorradorDenegado
	}
	autorizacion, err := r.autorizacion.AutorizarBorradorPropio(ctx, efecto)
	if err != nil {
		return cero, opaco(ctx, err)
	}
	accion, recurso, finalidad := contratoSolicitud(solicitud)
	if autorizacion.Material.ValidarEstructura() != nil || autorizacion.Accion != accion || autorizacion.RecursoRef != recurso || autorizacion.Finalidad != finalidad || !sellosIguales(autorizacion.Revalidacion, sello) || autorizacion.Material.PersonaVersion() != base.Contexto.Contexto.Instantanea.PersonaVersion || autorizacion.Material.PerfilVersion() != base.Contexto.Contexto.Instantanea.PerfilVersion {
		return cero, dietasports.ErrAccesoBorradorDenegado
	}
	return dietasports.IdentidadEfectivaBorrador{Vinculo: base.Vinculo, ContextoRegistrado: base.Contexto, Relacion: acreditada, Autorizacion: autorizacion}, nil
}

func seleccionSolicitud(s dietasports.SolicitudOperacionBorrador, fecha personaldomain.FechaCivil) (personaldomain.FechaCivil, string, error) {
	if s.Operacion == dietasports.OperacionCrearBorrador {
		f, err := personaldomain.NuevaFechaCivil(s.Crear.FechaInicio)
		if err != nil || s.RelacionRef != s.Crear.RelacionRef {
			return "", "", dietasports.ErrRelacionNoValida
		}
		// La fecha del comando se autoriza como parte del efecto. No procede de
		// identidad ni sustituye la fecha de canal usada en lecturas sin fecha.
		// Sin selector explícito, Personal debe acreditar una sola relación
		// vigente; nunca elegimos una entre varias por orden o por nombre.
		return f, s.RelacionRef, nil
	}
	if s.Operacion != dietasports.OperacionConsultarBorrador {
		return "", "", dietasports.ErrRelacionNoValida
	}
	// Una lista sin relación nunca escoge una de varias: Personal devuelve la
	// lista y sólo una relación acreditada permite continuar con Dietas.
	return fecha, s.RelacionRef, nil
}

func relacionUnica(resultado personalports.ResultadoConsultaRelacionPropia, pedida string) (personaldomain.RelacionEmpleado, error) {
	if resultado.Evidencia.ReciboRef == "" || resultado.Evidencia.DecisionRef == "" || resultado.Evidencia.EfectoRef == "" || resultado.Evidencia.ConsumoHuellaSHA256 == "" || resultado.Evidencia.AuditoriaRef == "" || resultado.Evidencia.ConsultadaEn.IsZero() {
		return personaldomain.RelacionEmpleado{}, dietasports.ErrAccesoBorradorDenegado
	}
	if len(resultado.Relaciones) == 0 {
		return personaldomain.RelacionEmpleado{}, dietasports.ErrRelacionNoDisponible
	}
	if len(resultado.Relaciones) != 1 {
		return personaldomain.RelacionEmpleado{}, dietasports.ErrRelacionAmbigua
	}
	r := resultado.Relaciones[0]
	if r.Validar() != nil || (pedida != "" && r.RelacionRef != pedida) {
		return personaldomain.RelacionEmpleado{}, dietasports.ErrRelacionNoValida
	}
	return r, nil
}

func traducirRelacion(r personaldomain.RelacionEmpleado) dietasports.RelacionServicioAcreditada {
	return dietasports.RelacionServicioAcreditada{RelacionRef: r.RelacionRef, PersonaRef: r.PersonaRef, EmpleadoRef: r.EmpleadoRef, UnidadRef: r.UnidadRef, VigenteDesde: r.Desde.Texto(), VigenteHasta: r.Hasta.Texto(), Version: r.Version, ProcedenciaActoRef: r.ProcedenciaActoRef, FuenteRef: r.FuenteRef, FuenteVersion: r.FuenteVersion}
}
func selloRelacion(r personaldomain.RelacionEmpleado, f personaldomain.FechaCivil) dietasports.RevalidacionRelacionPersonal {
	return dietasports.RevalidacionRelacionPersonal{RelacionRef: r.RelacionRef, PersonaRef: r.PersonaRef, EmpleadoRef: r.EmpleadoRef, UnidadRef: r.UnidadRef, VigenteDesde: r.Desde.Texto(), VigenteHasta: r.Hasta.Texto(), FechaReferencia: f.Texto(), Version: r.Version, ProcedenciaActoRef: r.ProcedenciaActoRef, FuenteRef: r.FuenteRef, FuenteVersion: r.FuenteVersion}
}
func sellosIguales(a, b dietasports.RevalidacionRelacionPersonal) bool { return a == b }
func contratoSolicitud(s dietasports.SolicitudOperacionBorrador) (string, string, string) {
	if s.Operacion == dietasports.OperacionCrearBorrador {
		return dietasapp.AccionCrearBorradorPropio, dietasapp.RecursoMisBorradores, dietasapp.FinalidadCrearBorradorPropio
	}
	if s.Referencia != "" {
		return dietasapp.AccionConsultarBorradorPropio, s.Referencia, dietasapp.FinalidadConsultarBorradorPropio
	}
	return dietasapp.AccionConsultarBorradorPropio, dietasapp.RecursoMisBorradores, dietasapp.FinalidadConsultarBorradorPropio
}
func opaco(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return dietasports.ErrAccesoBorradorDenegado
}
func nulo(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Ptr || x.Kind() == reflect.Interface || x.Kind() == reflect.Func || x.Kind() == reflect.Map || x.Kind() == reflect.Slice) && x.IsNil()
}

var _ dietasports.ResolutorIdentidadEfectivaBorrador = (*ResolutorIdentidadEfectivaBorrador)(nil)
