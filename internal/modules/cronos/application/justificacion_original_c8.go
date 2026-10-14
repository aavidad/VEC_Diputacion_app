package application

import (
	"context"
	"regexp"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const AudienciaJustificacionOriginalC8 = "vec_cronos_v1.justificacion.v1"

var reciboCronosOriginalC8 = regexp.MustCompile(`^recibo:cronos:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// RecursoJustificacionOriginalC8 liga la decisión a la solicitud y a los
// bytes canónicos que identifican original, versión, recibo y actor.
func RecursoJustificacionOriginalC8(m domain.MaterialJustificacionOriginalC8) (vecdomain.RecursoAutorizable, error) {
	h, err := m.Huella()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	r := vecdomain.RecursoAutorizable{Referencia: m.SolicitudRef, ModuloID: "cronos", Tipo: "justificacion",
		Ambitos:   map[string]string{"empleado_ref": m.EmpleadoObjetivoRef, "solicitud_ref": m.SolicitudRef},
		Atributos: map[string]string{"material_sha256": h}}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrJustificacionNoDisponible
	}
	return r, nil
}

type ServicioJustificacionOriginalC8 struct {
	solicitudes ports.FuenteSolicitudOriginalC8
	originales  ports.FuenteOriginalAdmitidoC8
	repositorio ports.RepositorioOriginalC8
	reloj       ports.Reloj
	zona        *time.Location
}

func NuevoServicioJustificacionOriginalC8(s ports.FuenteSolicitudOriginalC8, d ports.FuenteOriginalAdmitidoC8,
	r ports.RepositorioOriginalC8, reloj ports.Reloj, zona *time.Location) (*ServicioJustificacionOriginalC8, error) {
	if dependenciaRemotaNula(s) || dependenciaRemotaNula(d) || dependenciaRemotaNula(r) || dependenciaRemotaNula(reloj) || zona == nil ||
		(zona.String() != "Europe/Madrid" && zona.String() != "Atlantic/Canary") {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &ServicioJustificacionOriginalC8{s, d, r, reloj, zona}, nil
}

func (s *ServicioJustificacionOriginalC8) preparar(ctx context.Context, orden ports.OrdenOriginalC8,
	ref string, modo ports.ModoLecturaOriginalC8) (vecdomain.ContextoActor, string, ports.SolicitudOriginalC8, error) {
	if s == nil || ctx == nil || ctx.Err() != nil || dependenciaRemotaNula(s.solicitudes) || dependenciaRemotaNula(s.originales) ||
		dependenciaRemotaNula(s.repositorio) || dependenciaRemotaNula(s.reloj) || s.zona == nil ||
		!domain.SolicitudPermisoRefValida(ref) {
		return vecdomain.ContextoActor{}, "", ports.SolicitudOriginalC8{}, ports.ErrJustificacionNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return vecdomain.ContextoActor{}, "", ports.SolicitudOriginalC8{}, err
	}
	empleado, _, err := empleadoVigente(ctx, actor, s.reloj)
	if err != nil {
		return vecdomain.ContextoActor{}, "", ports.SolicitudOriginalC8{}, err
	}
	fuente, err := s.solicitudes.PrepararOriginal(ctx, orden, ref, modo)
	if err != nil || fuente.SolicitudRef != ref || fuente.VersionPermiso < 1 ||
		fuente.EstadoPermiso != domain.EstadoPermisoConcedido || !fuente.PendienteJustificar || !fuente.JustificanteExigido ||
		fuente.EmpleadoRef == "" || fuente.PersonaTitularRef == "" || fuente.ContextoRef == "" ||
		(fuente.Actual != nil && (fuente.Actual.SolicitudRef != ref || fuente.Actual.EmpleadoRef != fuente.EmpleadoRef ||
			fuente.Actual.Original.ValidarPara(ref, fuente.PersonaTitularRef) != nil)) {
		return vecdomain.ContextoActor{}, "", ports.SolicitudOriginalC8{}, ports.ErrJustificacionNoDisponible
	}
	return actor, empleado, fuente, nil
}

func (s *ServicioJustificacionOriginalC8) confirmar(ctx context.Context, orden ports.OrdenOriginalC8,
	m domain.MaterialJustificacionOriginalC8, siguiente domain.JustificacionOriginalC8V2) (ports.ReciboOriginalC8, error) {
	recurso, err := RecursoJustificacionOriginalC8(m)
	if err != nil {
		return ports.ReciboOriginalC8{}, err
	}
	material, err := orden.ProveedorMaterial().ProveerMaterialOriginal(ctx, m)
	if err != nil {
		return ports.ReciboOriginalC8{}, err
	}
	resumen := material.ResumenCapacidad()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	ahora := s.reloj.AhoraUTC()
	if err != nil || material.ValidarEstructura() != nil || resumen.Operacion() != m.Accion ||
		resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != huella ||
		resumen.AudienciaConsumo() != AudienciaJustificacionOriginalC8 ||
		ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return ports.ReciboOriginalC8{}, ports.ErrJustificacionNoDisponible
	}
	r, err := s.repositorio.ConfirmarOriginal(ctx, orden, m, material)
	if err != nil {
		return ports.ReciboOriginalC8{}, err
	}
	if !reciboOriginalC8Coherente(r, siguiente) {
		return ports.ReciboOriginalC8{}, ports.ErrJustificacionNoDisponible
	}
	return r, nil
}

func reciboOriginalC8Coherente(r ports.ReciboOriginalC8, siguiente domain.JustificacionOriginalC8V2) bool {
	return r.SolicitudRef == siguiente.SolicitudRef && r.DocumentoID == siguiente.Original.DocumentoID &&
		r.DocumentoVersion == siguiente.Original.DocumentoVersion && r.SHA256 == siguiente.Original.HuellaSHA256Bytes &&
		r.Estado == siguiente.Estado && r.Version == siguiente.Version && reciboCronosOriginalC8.MatchString(r.ReciboRef) &&
		!r.InstanteUTC.IsZero() && r.InstanteUTC.Location() == time.UTC && r.InstanteUTC.Nanosecond()%1000 == 0
}

func (s *ServicioJustificacionOriginalC8) Aportar(ctx context.Context, orden ports.OrdenOriginalC8,
	in ports.PeticionAnexoOriginalC8) (ports.ReciboOriginalC8, error) {
	actor, empleado, fuente, err := s.preparar(ctx, orden, in.SolicitudRef, ports.LecturaOriginalPropia)
	if err != nil {
		return ports.ReciboOriginalC8{}, err
	}
	if actor.PersonaRef != fuente.PersonaTitularRef || empleado != fuente.EmpleadoRef ||
		!domain.RefDocumentoJustificacionValida(in.ClaveOperacion) || !domain.RefDocumentoJustificacionValida(in.DocumentoID) ||
		in.DocumentoVersion == 0 {
		return ports.ReciboOriginalC8{}, domain.ErrJustificacionInvalida
	}
	original, err := s.originales.AcreditarOriginalPropio(ctx, orden, fuente, in.DocumentoID, in.DocumentoVersion)
	if err != nil || original.ValidarPara(fuente.SolicitudRef, fuente.PersonaTitularRef) != nil ||
		original.DocumentoID != in.DocumentoID || original.DocumentoVersion != in.DocumentoVersion {
		return ports.ReciboOriginalC8{}, ports.ErrJustificacionNoDisponible
	}
	siguiente, err := domain.PrepararAnexoOriginalC8V2(fuente.SolicitudRef, fuente.EmpleadoRef,
		fuente.PersonaTitularRef, original, fuente.Actual, in.VersionEsperada)
	if err != nil {
		return ports.ReciboOriginalC8{}, err
	}
	m := materialOriginalC8(actor, empleado, fuente, original, in.ClaveOperacion, in.VersionEsperada,
		domain.AccionAnexarOriginalC8, "", "", s.zona.String())
	return s.confirmar(ctx, orden, m, siguiente)
}

func (s *ServicioJustificacionOriginalC8) Revisar(ctx context.Context, orden ports.OrdenOriginalC8,
	in ports.PeticionRevisionOriginalC8V2) (ports.ReciboOriginalC8, error) {
	actor, empleado, fuente, err := s.preparar(ctx, orden, in.SolicitudRef, ports.LecturaOriginalRRHH)
	if err != nil {
		return ports.ReciboOriginalC8{}, err
	}
	if actor.PersonaRef == fuente.PersonaTitularRef || empleado == fuente.EmpleadoRef ||
		!domain.RefDocumentoJustificacionValida(in.ClaveOperacion) || fuente.Actual == nil {
		return ports.ReciboOriginalC8{}, ports.ErrJustificacionNoDisponible
	}
	original := fuente.Actual.Original
	acreditado, err := s.originales.AcreditarOriginalRRHH(ctx, orden, fuente, original.DocumentoID, original.DocumentoVersion)
	if err != nil || acreditado != original || acreditado.ValidarPara(fuente.SolicitudRef, fuente.PersonaTitularRef) != nil {
		return ports.ReciboOriginalC8{}, ports.ErrJustificacionNoDisponible
	}
	siguiente, err := domain.PrepararRevisionOriginalC8V2(*fuente.Actual, in.VersionEsperada, in.Decision, in.MotivoRef)
	if err != nil {
		return ports.ReciboOriginalC8{}, err
	}
	m := materialOriginalC8(actor, empleado, fuente, original, in.ClaveOperacion, in.VersionEsperada,
		domain.AccionRevisarOriginalC8, string(in.Decision), in.MotivoRef, s.zona.String())
	return s.confirmar(ctx, orden, m, siguiente)
}

func materialOriginalC8(actor vecdomain.ContextoActor, empleado string, fuente ports.SolicitudOriginalC8,
	o domain.OriginalJustificacionC8V2, clave string, version int64, accion, decision, motivo, zona string) domain.MaterialJustificacionOriginalC8 {
	return domain.MaterialJustificacionOriginalC8{
		ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado,
		EmpleadoObjetivoRef: fuente.EmpleadoRef, SolicitudRef: fuente.SolicitudRef,
		ClaveOperacion: clave, VersionEsperada: strconv.FormatInt(version, 10),
		DocumentoID: o.DocumentoID, DocumentoVersion: strconv.FormatUint(o.DocumentoVersion, 10),
		DocumentoSHA256: o.HuellaSHA256Bytes, ReciboDocumentoRef: o.ReciboDocumentoRef,
		CustodioID: o.CustodioID, PersonaTitularRef: fuente.PersonaTitularRef,
		PersonaAportanteRef: o.PersonaAportanteRef, ContextoRef: fuente.ContextoRef,
		Accion: accion, Decision: decision, MotivoRef: motivo, ZonaHoraria: zona,
	}
}
