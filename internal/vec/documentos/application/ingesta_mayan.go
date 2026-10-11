package application

import (
	"context"
	"errors"
	"io"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ServicioIngestaMayan no abre una segunda escritura remota después de un
// despacho confirmado o ambiguo. Su fuente de bytes debe ser la lectura
// autorizada de una carga admitida; este servicio no acepta contenido HTTP.
type ServicioIngestaMayan struct {
	Repositorio ports.RepositorioIngestaMayan
	Autoridad   ports.AutoridadIngestaMayan
	Custodio    ports.CustodioMayan
	Cargas      vecports.RepositorioCargasDocumentales
	Reloj       interface{ Ahora() time.Time }
}

func (s *ServicioIngestaMayan) disponible() bool {
	return s != nil && !dependenciaNula(s.Repositorio) && !dependenciaNula(s.Cargas) && !dependenciaNula(s.Autoridad) &&
		!dependenciaNula(s.Custodio) && !dependenciaNula(s.Reloj)
}

func (s *ServicioIngestaMayan) comprobarCarga(ctx context.Context, p ports.PlanIngestaMayan) error {
	c, err := s.Cargas.Obtener(ctx, p.CargaID)
	if err != nil || c.Validar() != nil || c.Estado != vecdomain.EstadoCargaDocumentalAdmitida ||
		int64(c.Version) != p.CargaVersion || c.ContenidoAdmitido == nil ||
		c.ContenidoAdmitido.EvidenciaRef != p.EvidenciaRef ||
		c.ContenidoAdmitido.HuellaSHA256 != p.HuellaSHA256 ||
		c.ContenidoAdmitido.Tamano != p.Tamano || c.ContenidoAdmitido.MIME != p.MIME {
		return ports.ErrSolicitudInvalida
	}
	return nil
}

func (s *ServicioIngestaMayan) autorizar(ctx context.Context, accion string, dato any, planRef, expedienteRef string) (ports.AutorizacionV3, error) {
	preimagen, err := ports.PreimagenIngestaMayan(accion, dato)
	if err != nil {
		return ports.AutorizacionV3{}, err
	}
	a, err := s.Autoridad.AutorizarIngestaMayan(ctx, accion, preimagen, planRef, expedienteRef)
	if err != nil || a.ValidarParaIngestaMayan(accion, planRef, expedienteRef, s.Reloj.Ahora().UTC()) != nil ||
		a.Material.ResumenCapacidad().EfectoHuellaSHA256() != ports.HuellaEfectoV3(preimagen) {
		return ports.AutorizacionV3{}, ports.ErrSolicitudInvalida
	}
	return a, nil
}

func (s *ServicioIngestaMayan) Preparar(ctx context.Context, p ports.PlanIngestaMayan) (ports.PlanIngestaMayan, error) {
	if !s.disponible() || ctx == nil || p.ValidarBase() != nil || p.Version != 0 || p.Estado != "" {
		return ports.PlanIngestaMayan{}, ports.ErrSolicitudInvalida
	}
	if err := s.comprobarCarga(ctx, p); err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	a, err := s.autorizar(ctx, ports.AccionPrepararIngestaMayan, p, p.PlanRef, p.ExpedienteRef)
	if err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	return s.Repositorio.PrepararIngesta(ctx, ports.OrdenPrepararIngestaMayan{Plan: p, Autorizacion: a})
}

func (s *ServicioIngestaMayan) Consultar(ctx context.Context, planRef, expedienteRef string) (ports.PlanIngestaMayan, error) {
	if !s.disponible() || ctx == nil {
		return ports.PlanIngestaMayan{}, ports.ErrSolicitudInvalida
	}
	dato := struct {
		PlanRef       string `json:"plan_ref"`
		ExpedienteRef string `json:"expediente_ref"`
	}{planRef, expedienteRef}
	a, err := s.autorizar(ctx, ports.AccionConsultarIngestaMayan, dato, planRef, expedienteRef)
	if err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	return s.Repositorio.ConsultarIngesta(ctx, ports.OrdenConsultarIngestaMayan{PlanRef: planRef, ExpedienteRef: expedienteRef, Autorizacion: a})
}

func (s *ServicioIngestaMayan) cambiar(ctx context.Context, accion string, o ports.OrdenCambiarIngestaMayan) (ports.PlanIngestaMayan, bool, error) {
	dato := struct {
		PlanRef         string               `json:"plan_ref"`
		ExpedienteRef   string               `json:"expediente_ref"`
		VersionEsperada int64                `json:"version_esperada"`
		Estado          string               `json:"estado"`
		Documento       ports.DocumentoMayan `json:"documento"`
	}{o.PlanRef, o.ExpedienteRef, o.VersionEsperada, o.Estado, o.Documento}
	a, err := s.autorizar(ctx, accion, dato, o.PlanRef, o.ExpedienteRef)
	if err != nil {
		return ports.PlanIngestaMayan{}, false, err
	}
	o.Autorizacion = a
	if accion == ports.AccionDespacharIngestaMayan {
		return s.Repositorio.DespacharIngesta(ctx, o)
	}
	p, err := s.Repositorio.ResolverIngesta(ctx, o)
	return p, false, err
}

// Ingerir marca el despacho y espera su COMMIT antes de invocar al proveedor.
// Un fallo del proveedor conserva el mismo plan como indeterminado. Si la
// persistencia de la respuesta falla, el despacho durable sigue impidiendo
// otro POST y la consulta permite continuar la conciliación.
func (s *ServicioIngestaMayan) Ingerir(ctx context.Context, planRef, expedienteRef string, contenido io.Reader) (ports.PlanIngestaMayan, error) {
	if !s.disponible() || ctx == nil || contenido == nil {
		return ports.PlanIngestaMayan{}, ports.ErrSolicitudInvalida
	}
	p, err := s.Consultar(ctx, planRef, expedienteRef)
	if err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	if p.Estado != ports.EstadoIngestaPreparada {
		return p, ports.ErrIngestaNoReenviable
	}
	if err := s.comprobarCarga(ctx, p); err != nil {
		return p, err
	}
	despacho := ports.OrdenCambiarIngestaMayan{PlanRef: planRef, ExpedienteRef: expedienteRef, VersionEsperada: p.Version, Estado: ports.EstadoIngestaDespachada}
	p, nuevo, err := s.cambiar(ctx, ports.AccionDespacharIngestaMayan, despacho)
	if err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	if !nuevo {
		return p, ports.ErrIngestaNoReenviable
	}
	d, errProveedor := s.Custodio.IngerirOriginal(ctx, p, contenido)
	estado := ports.EstadoIngestaConfirmada
	if errProveedor != nil || d.ValidarContra(p) != nil {
		estado = ports.EstadoIngestaIndeterminada
	}
	orden := ports.OrdenCambiarIngestaMayan{PlanRef: planRef, ExpedienteRef: expedienteRef, VersionEsperada: p.Version, Estado: estado, Documento: d}
	// El caller puede cancelar su espera, pero la respuesta remota ya pudo crear
	// un original. Persistimos con un plazo propio y nunca reintentamos el POST.
	ctxPersistencia, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancelar()
	resuelto, _, err := s.cambiar(ctxPersistencia, ports.AccionResolverIngestaMayan, orden)
	if err != nil {
		return p, errors.Join(ports.ErrIngestaIndeterminada, err)
	}
	if estado == ports.EstadoIngestaIndeterminada {
		return resuelto, ports.ErrIngestaIndeterminada
	}
	return resuelto, nil
}

// Conciliar solo consulta un ID recibido previamente. Una caída anterior al
// ID exige conciliación supervisada fuera de este método.
func (s *ServicioIngestaMayan) Conciliar(ctx context.Context, planRef, expedienteRef string) (ports.PlanIngestaMayan, error) {
	p, err := s.Consultar(ctx, planRef, expedienteRef)
	if err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	if p.Estado != ports.EstadoIngestaIndeterminada && p.Estado != ports.EstadoIngestaDespachada {
		return p, ports.ErrSolicitudInvalida
	}
	if p.DocumentoID == "" {
		return p, ports.ErrIngestaIndeterminada
	}
	d, err := s.Custodio.ReconciliarOriginal(ctx, ports.DocumentoMayan{DocumentoID: p.DocumentoID, HuellaSHA256: p.HuellaSHA256, Tamano: p.Tamano, MIME: p.MIME})
	if err != nil || d.ValidarContra(p) != nil {
		return p, ports.ErrIngestaIndeterminada
	}
	o := ports.OrdenCambiarIngestaMayan{PlanRef: planRef, ExpedienteRef: expedienteRef, VersionEsperada: p.Version, Estado: ports.EstadoIngestaConfirmada, Documento: d}
	r, _, err := s.cambiar(ctx, ports.AccionConciliarIngestaMayan, o)
	return r, err
}
