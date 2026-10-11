package application

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"time"

	baseapp "vec-diputacion-granada/internal/vec/application"
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
	Politicas   vecports.ResolutorPoliticaConservacionDocumental
	Reloj       interface{ Ahora() time.Time }
}

func (s *ServicioIngestaMayan) disponible() bool {
	return s != nil && !dependenciaNula(s.Repositorio) && !dependenciaNula(s.Cargas) && !dependenciaNula(s.Politicas) && !dependenciaNula(s.Autoridad) &&
		!dependenciaNula(s.Custodio) && !dependenciaNula(s.Reloj)
}

func (s *ServicioIngestaMayan) reservaDisponible() bool {
	return s != nil && !dependenciaNula(s.Repositorio) && !dependenciaNula(s.Autoridad) && !dependenciaNula(s.Reloj)
}

// ReservarRecurso se ejecuta antes de preparar la carga S3. La referencia
// candidata procede del emisor de B en el servidor; solo la fila SQL confirma
// cuál quedó reservada por la clave. La identidad viene del vínculo V2.
func (s *ServicioIngestaMayan) ReservarRecurso(ctx context.Context, candidatoRef, clave, huellaSolicitud string,
	resultado vecdomain.ResultadoContextoActorRegistradoV2, vinculo vecdomain.VinculoAutenticacionActorV2,
) (ports.ReservaRecursoDocumental, error) {
	if !s.reservaDisponible() || ctx == nil || resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil {
		return ports.ReservaRecursoDocumental{}, ports.ErrSolicitudInvalida
	}
	r := ports.ReservaRecursoDocumental{RecursoRef: candidatoRef, PersonaRef: resultado.Contexto.PersonaRef,
		RegistroContextoRef: resultado.RegistroContextoRef, HuellaContextoSHA256: resultado.HuellaSHA256,
		ClaveIdempotencia: clave, HuellaSolicitudSHA256: huellaSolicitud}
	if r.ValidarCandidata() != nil {
		return ports.ReservaRecursoDocumental{}, ports.ErrSolicitudInvalida
	}
	a, err := s.autorizar(ctx, ports.AccionReservarRecursoMayan, r, clave, clave)
	if err != nil || a.PrincipalID != r.PersonaRef {
		return ports.ReservaRecursoDocumental{}, ports.ErrSolicitudInvalida
	}
	return s.Repositorio.ReservarRecurso(ctx, ports.OrdenReservarRecursoMayan{Reserva: r, Autorizacion: a})
}

func (s *ServicioIngestaMayan) ConsultarReserva(ctx context.Context, clave string) (ports.ReservaRecursoDocumental, error) {
	if !s.reservaDisponible() || ctx == nil {
		return ports.ReservaRecursoDocumental{}, ports.ErrSolicitudInvalida
	}
	dato := struct {
		ClaveIdempotencia string `json:"clave_idempotencia"`
	}{clave}
	a, err := s.autorizar(ctx, ports.AccionConsultarReservaMayan, dato, clave, clave)
	if err != nil {
		return ports.ReservaRecursoDocumental{}, err
	}
	return s.Repositorio.ConsultarReserva(ctx, ports.OrdenConsultaReservaMayan{ClaveIdempotencia: clave, Autorizacion: a})
}

// AbandonarRecurso solo cierra una reserva que nunca fue reclamada por S3.
// La fila y su referencia permanecen para auditoría e idempotencia.
func (s *ServicioIngestaMayan) AbandonarRecurso(ctx context.Context, clave string) (ports.ReservaRecursoDocumental, error) {
	if !s.reservaDisponible() || ctx == nil {
		return ports.ReservaRecursoDocumental{}, ports.ErrSolicitudInvalida
	}
	dato := struct {
		ClaveIdempotencia string `json:"clave_idempotencia"`
	}{clave}
	a, err := s.autorizar(ctx, ports.AccionAbandonarRecursoMayan, dato, clave, clave)
	if err != nil {
		return ports.ReservaRecursoDocumental{}, err
	}
	return s.Repositorio.AbandonarRecurso(ctx, ports.OrdenAbandonarRecursoMayan{ClaveIdempotencia: clave, Autorizacion: a})
}

func (s *ServicioIngestaMayan) comprobarCarga(ctx context.Context, p ports.PlanIngestaMayan, resultado vecdomain.ResultadoContextoActorRegistradoV2, vinculo vecdomain.VinculoAutenticacionActorV2) error {
	if resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil ||
		resultado.Contexto.PersonaRef != p.PersonaRef || resultado.Contexto.Principal.ID != p.PersonaRef {
		return ports.ErrSolicitudInvalida
	}
	c, err := s.Cargas.Obtener(ctx, p.CargaID)
	if err != nil || c.Validar() != nil || c.Estado != vecdomain.EstadoCargaDocumentalAdmitida ||
		c.ModuloID != "meritos" || c.RecursoRef != p.ExpedienteRef || c.PrincipalID != p.PersonaRef ||
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

func (s *ServicioIngestaMayan) Preparar(ctx context.Context, p ports.PlanIngestaMayan, solicitud vecports.SolicitudPoliticaConservacionDocumental, resultado vecdomain.ResultadoContextoActorRegistradoV2, vinculo vecdomain.VinculoAutenticacionActorV2) (ports.PlanIngestaMayan, error) {
	if !s.disponible() || ctx == nil || solicitud.Validar() != nil || solicitud.TipoDocumentalRef() != p.TipoRef ||
		p.PlanRef != "" || p.ExpedienteRef != "" ||
		p.PersonaRef != "" || p.RegistroContextoRef != "" || p.HuellaContextoSHA256 != "" ||
		p.Version != 0 || p.Estado != "" || p.DocumentoID != "" || p.ArchivoID != "" || p.VersionID != "" ||
		!p.CreadoEn.IsZero() || !p.ActualizadoEn.IsZero() {
		return ports.PlanIngestaMayan{}, ports.ErrSolicitudInvalida
	}
	if resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil {
		return ports.PlanIngestaMayan{}, ports.ErrSolicitudInvalida
	}
	reserva, err := s.ConsultarReserva(ctx, p.ClaveReserva)
	if err != nil || reserva.PersonaRef != resultado.Contexto.PersonaRef ||
		reserva.Estado != ports.EstadoRecursoVinculado ||
		solicitud.ExpedienteRef() != reserva.RecursoRef {
		return ports.PlanIngestaMayan{}, ports.ErrSolicitudInvalida
	}
	p.ExpedienteRef = reserva.RecursoRef
	p.PersonaRef = resultado.Contexto.PersonaRef
	p.RegistroContextoRef = resultado.RegistroContextoRef
	p.HuellaContextoSHA256 = resultado.HuellaSHA256
	politica, err := baseapp.ResolverPoliticaConservacionDocumentalAdmitiendoProvisional(ctx, s.Politicas, s.Reloj, solicitud)
	if err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	instancia := politica.Politica()
	resuelta := instancia.Solicitud()
	p.PoliticaRef = resuelta.PoliticaRef()
	p.VersionPolitica = int64(resuelta.VersionPolitica())
	p.HuellaPoliticaSHA256 = hex.EncodeToString(resuelta.HuellaPoliticaSHA256())
	p.ConservacionHasta = instancia.ConservacionHasta().UTC()
	p.EstadoPolitica = ports.EstadoPolitica(instancia)
	p.Proteccion = string(instancia.Proteccion())
	if p.ValidarBase() != nil {
		return ports.PlanIngestaMayan{}, ports.ErrSolicitudInvalida
	}
	if err := s.comprobarCarga(ctx, p, resultado, vinculo); err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	a, err := s.autorizar(ctx, ports.AccionPrepararIngestaMayan, p, p.CargaID, p.ExpedienteRef)
	if err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	if a.PrincipalID != p.PersonaRef {
		return ports.PlanIngestaMayan{}, ports.ErrSolicitudInvalida
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
func (s *ServicioIngestaMayan) Ingerir(ctx context.Context, planRef, expedienteRef string, resultado vecdomain.ResultadoContextoActorRegistradoV2, vinculo vecdomain.VinculoAutenticacionActorV2, contenido io.Reader) (ports.PlanIngestaMayan, error) {
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
	if err := s.comprobarCarga(ctx, p, resultado, vinculo); err != nil {
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
func (s *ServicioIngestaMayan) Conciliar(ctx context.Context, planRef, expedienteRef string, resultado vecdomain.ResultadoContextoActorRegistradoV2, vinculo vecdomain.VinculoAutenticacionActorV2) (ports.PlanIngestaMayan, error) {
	p, err := s.Consultar(ctx, planRef, expedienteRef)
	if err != nil {
		return ports.PlanIngestaMayan{}, err
	}
	if err := s.comprobarCarga(ctx, p, resultado, vinculo); err != nil {
		return p, err
	}
	if p.Estado != ports.EstadoIngestaIndeterminada && p.Estado != ports.EstadoIngestaDespachada {
		return p, ports.ErrSolicitudInvalida
	}
	if p.DocumentoID == "" {
		return p, ports.ErrIngestaIndeterminada
	}
	d, err := s.Custodio.ReconciliarOriginal(ctx, ports.DocumentoMayan{
		DocumentoID: p.DocumentoID, ArchivoID: p.ArchivoID, VersionID: p.VersionID,
		HuellaSHA256: p.HuellaSHA256, Tamano: p.Tamano, MIME: p.MIME,
	})
	if err != nil || !evidenciaMayanConservaIDs(p, d) {
		return p, ports.ErrIngestaIndeterminada
	}
	o := ports.OrdenCambiarIngestaMayan{PlanRef: planRef, ExpedienteRef: expedienteRef, VersionEsperada: p.Version, Estado: ports.EstadoIngestaConfirmada, Documento: d}
	r, _, err := s.cambiar(ctx, ports.AccionConciliarIngestaMayan, o)
	return r, err
}

func evidenciaMayanConservaIDs(p ports.PlanIngestaMayan, d ports.DocumentoMayan) bool {
	return d.ValidarContra(p) == nil &&
		(p.DocumentoID == "" || d.DocumentoID == p.DocumentoID) &&
		(p.ArchivoID == "" || d.ArchivoID == p.ArchivoID) &&
		(p.VersionID == "" || d.VersionID == p.VersionID)
}
