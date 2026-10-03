// Package competenciacentral compone la identidad acreditada del certificado
// con la asignación nominal central. No concede permisos ni crea sesiones.
package competenciacentral

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Estos puertos pertenecen a la autoridad central de ContextoActor.
type VinculoCertificadoFirmante = vecports.VinculoCertificadoFirmante
type FuenteVinculoCertificado = vecports.FuenteVinculoCertificadoFirmante

// FuenteSolicitudCentral resuelve las autoridades del consultante y del
// recurso desde el servidor. La solicitud no procede de atributos HTTP.
// Cargo corresponde a Personal; el perfil lógico no se convierte en un cargo.
type FuenteSolicitudCentral interface {
	// AutorizarLecturaCertificado exige V3 positiva, exacta y vigente del
	// consultante para esta huella/recurso/finalidad, y audita éxito o denegación
	// mediante la autoridad común. Un LOGIN técnico no basta. Se ejecuta antes
	// de leer identidad. El proveedor no reutiliza concesiones de otra petición.
	AutorizarLecturaCertificado(context.Context, ctports.SolicitudCompetenciaFirmante) error
	ResolverSolicitudCentral(context.Context, ctports.SolicitudCompetenciaFirmante, VinculoCertificadoFirmante) (vecdomain.SolicitudAsignacionCompetencialV1, error)
}

type Reloj interface{ Ahora() time.Time }

// Acreditacion retiene las fuentes completas para la transacción final.
// Sus campos privados impiden usar una proyección HTTP como evidencia central.
type Acreditacion struct {
	proteccionAcreditacion
	solicitud  vecdomain.SolicitudAsignacionCompetencialV1
	evidencia  vecdomain.EvidenciaAsignacionCompetencialV1
	vinculo    VinculoCertificadoFirmante
	proyeccion ctports.EvidenciaCompetenciaFirmante
}

func (a Acreditacion) Proyeccion() ctports.EvidenciaCompetenciaFirmante { return a.proyeccion }

// DescriptorPaso conserva el mapeo aprobado del perfil lógico del circuito
// al RolID nominal. La aplicación obtiene este descriptor de la misma versión
// publicada del catálogo; no se recibe del navegador.
type DescriptorPaso struct {
	Documento, PasoRef                        string
	AccionCompetencial, FinalidadCompetencial string
	Orden                                     int
	Perfiles                                  map[string]string
}

type DescriptorCircuito struct {
	CatalogoRef, CatalogoHuella                  string
	AccionLectura, FinalidadLectura, TipoRecurso string
	CatalogoVersion                              uint64
	Pasos                                        []DescriptorPaso
}

type Fuente struct {
	identidad    FuenteVinculoCertificado
	asignaciones vecports.LectorAsignacionesCompetencialesV1
	solicitudes  FuenteSolicitudCentral
	reloj        Reloj
	descriptor   DescriptorCircuito
}

func NuevaFuente(identidad FuenteVinculoCertificado, asignaciones vecports.LectorAsignacionesCompetencialesV1, solicitudes FuenteSolicitudCentral, reloj Reloj, descriptor DescriptorCircuito) (*Fuente, error) {
	if interfazNula(identidad) || interfazNula(asignaciones) || interfazNula(solicitudes) || interfazNula(reloj) || !ctdomain.ReferenciaOpacaValida(descriptor.CatalogoRef) ||
		!ctdomain.HuellaSHA256FirmaValida(descriptor.CatalogoHuella) || descriptor.CatalogoVersion == 0 || len(descriptor.Pasos) == 0 ||
		!ctdomain.ReferenciaOpacaValida(descriptor.AccionLectura) || !ctdomain.ReferenciaOpacaValida(descriptor.FinalidadLectura) || !ctdomain.ReferenciaOpacaValida(descriptor.TipoRecurso) {
		return nil, ctports.ErrCompetenciaFirmanteNoDisponible
	}
	copia := DescriptorCircuito{CatalogoRef: descriptor.CatalogoRef, CatalogoHuella: descriptor.CatalogoHuella, CatalogoVersion: descriptor.CatalogoVersion, AccionLectura: descriptor.AccionLectura, FinalidadLectura: descriptor.FinalidadLectura, TipoRecurso: descriptor.TipoRecurso}
	vistos := map[string]bool{}
	for _, p := range descriptor.Pasos {
		if !ctdomain.ClaveDocumentoFirmaValida(p.Documento) || p.PasoRef == "" || p.Orden < 1 || p.Orden > ctdomain.MaximoPasosCircuitoFirma ||
			len(p.Perfiles) == 0 || vistos[p.PasoRef] || !ctdomain.ReferenciaOpacaValida(p.AccionCompetencial) || !ctdomain.ReferenciaOpacaValida(p.FinalidadCompetencial) {
			return nil, ctports.ErrCompetenciaFirmanteNoDisponible
		}
		vistos[p.PasoRef] = true
		cp := DescriptorPaso{Documento: p.Documento, PasoRef: p.PasoRef, Orden: p.Orden, Perfiles: map[string]string{}, AccionCompetencial: p.AccionCompetencial, FinalidadCompetencial: p.FinalidadCompetencial}
		for perfil, rol := range p.Perfiles {
			if !ctdomain.ReferenciaOpacaValida(perfil) || !rolDescriptorValido(rol) {
				return nil, ctports.ErrCompetenciaFirmanteNoDisponible
			}
			cp.Perfiles[perfil] = rol
		}
		copia.Pasos = append(copia.Pasos, cp)
	}
	return &Fuente{identidad: identidad, asignaciones: asignaciones, solicitudes: solicitudes, reloj: reloj, descriptor: copia}, nil
}

func (f *Fuente) AcreditarCompetenciaFirmante(ctx context.Context, q ctports.SolicitudCompetenciaFirmante) (ctports.EvidenciaCompetenciaFirmante, error) {
	a, err := f.AcreditarCompetenciaCentral(ctx, q)
	return a.Proyeccion(), err
}

func (f *Fuente) AcreditarCompetenciaCentral(ctx context.Context, q ctports.SolicitudCompetenciaFirmante) (Acreditacion, error) {
	var cero Acreditacion
	if f == nil || ctx == nil || interfazNula(f.identidad) || interfazNula(f.asignaciones) || interfazNula(f.solicitudes) || interfazNula(f.reloj) {
		return cero, ctports.ErrCompetenciaFirmanteNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if q.CatalogoRef != f.descriptor.CatalogoRef || q.CatalogoHuella != f.descriptor.CatalogoHuella ||
		q.CatalogoVersion != f.descriptor.CatalogoVersion || !ctdomain.HuellaSHA256FirmaValida(q.CertificadoHuella) ||
		q.FirmanteRef != "ref:"+q.CertificadoHuella || !ctdomain.ReferenciaOpacaValida(q.OrganizacionRef) || !ctdomain.ReferenciaOpacaValida(q.ExpedienteRef) {
		return cero, ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	rol := ""
	accion, finalidad := "", ""
	for _, p := range f.descriptor.Pasos {
		if p.Documento == q.Documento && p.PasoRef == q.PasoRef && p.Orden == q.PasoOrden {
			rol = p.Perfiles[q.PerfilFirmanteRef]
			accion, finalidad = p.AccionCompetencial, p.FinalidadCompetencial
		}
	}
	if rol == "" || q.CargoFirmante != "" && q.CargoFirmante != rol {
		return cero, ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	if err := f.solicitudes.AutorizarLecturaCertificado(ctx, q); err != nil {
		return cero, errorFuente(ctx, err)
	}
	v, err := f.identidad.ResolverFirmantePorCertificado(ctx, q.CertificadoHuella)
	if err != nil {
		return cero, errorFuente(ctx, err)
	}
	if !v.Vigente || v.CertificadoHuella != q.CertificadoHuella || !strings.HasPrefix(v.PrincipalRef, "per_") ||
		!ctdomain.ReferenciaOpacaValida(v.PrincipalRef) || !ctdomain.ReferenciaOpacaValida(v.CuentaRef) || !strings.HasPrefix(v.CuentaRef, "cta_") ||
		!ctdomain.ReferenciaOpacaValida(v.VinculoCredencialRef) || !strings.HasPrefix(v.VinculoCredencialRef, "vcc_") || v.Revision == 0 || !ctdomain.HuellaSHA256FirmaValida(v.Huella) {
		return cero, ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	s, err := f.solicitudes.ResolverSolicitudCentral(ctx, q, v)
	if err != nil {
		return cero, errorFuente(ctx, err)
	}
	if s.PersonaRef != v.PrincipalRef || s.CertificadoHuellaSHA256 != q.CertificadoHuella || s.PerfilFirmanteRef != q.PerfilFirmanteRef ||
		s.AccionLectura != f.descriptor.AccionLectura || s.FinalidadLectura != f.descriptor.FinalidadLectura || s.AccionCompetencial != accion || s.FinalidadCompetencial != finalidad || s.Recurso.Tipo != f.descriptor.TipoRecurso || s.Recurso.ModuloID != "contratacion_temporal" || s.Recurso.Ambitos["organizacion_ref"] != q.OrganizacionRef ||
		s.Recurso.Ambitos["expediente_ref"] != q.ExpedienteRef || s.ValidarEn(f.reloj.Ahora()) != nil {
		return cero, ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	s, err = clonarSolicitud(s)
	if err != nil {
		return cero, ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	enviada, err := clonarSolicitud(s)
	if err != nil {
		return cero, ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	e, err := f.asignaciones.LeerAsignacionCompetencialV1(ctx, enviada)
	if err != nil {
		return cero, errorFuente(ctx, err)
	}
	e = clonarEvidencia(e)
	if e.ValidarParaEn(s, f.reloj.Ahora()) != nil || e.VersionRol.RolID != rol {
		return cero, ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	p := ctports.EvidenciaCompetenciaFirmante{
		Solicitud: q, FirmantePrincipalRef: e.PersonaRef, PerfilFirmanteRef: q.PerfilFirmanteRef, PerfilActivoFirmanteRef: e.PerfilActivoFirmanteRef,
		CargoFirmante: rol, RolIDFirmante: rol, UnidadFirmanteRef: e.UnidadRef, PuestoFirmanteRef: e.PuestoRef, AmbitoFirmanteRef: e.AmbitoRef,
		CuentaFirmanteRef: v.CuentaRef, VinculoCredencialFirmanteRef: v.VinculoCredencialRef, VinculoCredencialFirmanteRevision: v.Revision, VinculoCredencialFirmanteHuella: v.Huella,
		AsignacionFirmanteRef: e.Asignacion.Referencia(), AsignacionFirmanteVersion: uint64(e.Asignacion.Version), AsignacionFirmanteHuella: e.AsignacionHuellaSHA256,
		VersionRolFirmanteRef: e.VersionRol.Referencia(), VersionRolFirmanteHuella: e.VersionRolHuellaSHA256,
		ControlVigenciaFirmanteRef: e.ControlVigencia.VersionRolRef, ControlVigenciaFirmanteRevision: e.ControlVigencia.Revision, ControlVigenciaFirmanteHuella: e.ControlVigenciaHuellaSHA256,
		AsignacionVigenteDesde: e.Asignacion.VigenteDesde.Format(time.RFC3339Nano), AsignacionVigenteHasta: e.Asignacion.VigenteHasta.Format(time.RFC3339Nano),
		CompetenciaComprobadaEn: e.ComprobadaEn.Format(time.RFC3339Nano), ActoCompetenciaRef: e.ActoCompetenciaRef, Vigente: true,
	}
	if e.Delegacion != nil {
		p.DelegacionRef = e.Delegacion.Referencia
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return Acreditacion{solicitud: s, evidencia: e, vinculo: v, proyeccion: p}, nil
}

func errorFuente(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, ctports.ErrCompetenciaFirmanteNoAcreditada) || errors.Is(err, vecports.ErrVinculoCertificadoFirmanteNoAcreditado) {
		return ctports.ErrCompetenciaFirmanteNoAcreditada
	}
	return ctports.ErrCompetenciaFirmanteNoDisponible
}

// La lista positiva pertenece al descriptor publicado, no a este adaptador.
// Se conserva la gramática opaca existente y el máximo de RolID de
// vec/domain.VersionRol.Validar; la evidencia completa se valida allí y su
// RolID debe coincidir exactamente con el seleccionado en el descriptor.
func rolDescriptorValido(rol string) bool {
	return len(rol) <= 128 && ctdomain.ReferenciaOpacaValida(rol)
}

func interfazNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Func, reflect.Slice, reflect.Chan:
		return r.IsNil()
	default:
		return false
	}
}

func clonarSolicitud(s vecdomain.SolicitudAsignacionCompetencialV1) (vecdomain.SolicitudAsignacionCompetencialV1, error) {
	var err error
	s.Actor, err = s.Actor.Clonar()
	if err != nil {
		return vecdomain.SolicitudAsignacionCompetencialV1{}, err
	}
	s.ResultadoContexto, err = s.ResultadoContexto.Clonar()
	if err != nil {
		return vecdomain.SolicitudAsignacionCompetencialV1{}, err
	}
	s.Recurso.Ambitos = maps.Clone(s.Recurso.Ambitos)
	s.Recurso.Atributos = maps.Clone(s.Recurso.Atributos)
	return s, nil
}
func clonarEvidencia(e vecdomain.EvidenciaAsignacionCompetencialV1) vecdomain.EvidenciaAsignacionCompetencialV1 {
	e.Asignacion.Ambitos = slices.Clone(e.Asignacion.Ambitos)
	for i := range e.Asignacion.Ambitos {
		e.Asignacion.Ambitos[i].Valores = slices.Clone(e.Asignacion.Ambitos[i].Valores)
	}
	e.VersionRol.Concesiones = slices.Clone(e.VersionRol.Concesiones)
	for i := range e.VersionRol.Concesiones {
		c := &e.VersionRol.Concesiones[i]
		c.Finalidades = slices.Clone(c.Finalidades)
		c.CamposPermitidos = slices.Clone(c.CamposPermitidos)
		c.Obligaciones = slices.Clone(c.Obligaciones)
	}
	if e.Delegacion != nil {
		c := *e.Delegacion
		e.Delegacion = &c
	}
	return e
}
