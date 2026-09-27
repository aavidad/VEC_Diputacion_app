package application

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

var ErrSolicitudReincorporacionTitularInvalida = errors.New("contratacion temporal: solicitud de reincorporacion titular invalida")

type SolicitudRegistrarReincorporacionTitular struct {
	Canal                         ContextoCanalSeguimiento
	ExpedienteRef, RelacionRef    string
	FechaEfectiva                 time.Time
	DocumentoRef, DocumentoSHA256 string
	VersionEsperada               uint64
	ClaveIdempotencia             string
}

type DependenciasReincorporacionTitular struct {
	Contextos   ports.ResolutorContextoAutorizacionAltaV3
	Sellos      ports.SelladorReincorporacionTitular
	Repositorio ports.RepositorioReincorporacionTitular
	Reglas      ports.FuenteReglasSeguimiento
	Autorizador ports.AutorizadorOperacionSeguimiento
	Referencias ports.GeneradorReferenciasSeguimiento
	Reloj       ports.Reloj
}

type ServicioReincorporacionTitular struct {
	d DependenciasReincorporacionTitular
}

func NuevoServicioReincorporacionTitular(d DependenciasReincorporacionTitular) (*ServicioReincorporacionTitular, error) {
	if dependenciaNula(d.Contextos) || dependenciaNula(d.Sellos) || dependenciaNula(d.Repositorio) ||
		dependenciaNula(d.Reglas) || dependenciaNula(d.Autorizador) || dependenciaNula(d.Referencias) || dependenciaNula(d.Reloj) {
		return nil, ports.ErrOperacionSeguimientoNoDisponible
	}
	return &ServicioReincorporacionTitular{d: d}, nil
}

// RegistrarReincorporacionTitular exige identidad del canal y decisión V3
// específica incluso al recuperar un recibo existente.
func (s *ServicioReincorporacionTitular) RegistrarReincorporacionTitular(ctx context.Context, sol SolicitudRegistrarReincorporacionTitular) (ports.ReciboReincorporacionTitular, error) {
	var vacio ports.ReciboReincorporacionTitular
	if s == nil {
		return vacio, ports.ErrOperacionSeguimientoNoDisponible
	}
	if ctx == nil {
		return vacio, ErrSolicitudReincorporacionTitularInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	ctx, cancelar := context.WithTimeout(ctx, tiempoMaximoOperacionSeguimiento)
	defer cancelar()
	if !sol.Canal.Valido() {
		return vacio, ErrSolicitudReincorporacionTitularInvalida
	}
	ahora := instanteCanonico(s.d.Reloj.Ahora())
	contexto, err := s.d.Contextos.ResolverContextoAutorizacionAltaV3(ctx, sol.Canal.solicitud())
	if err != nil || contexto.ValidarPara(sol.Canal.solicitud(), ahora) != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	v, err := contexto.Vinculo.Datos()
	if err != nil || !domain.ReferenciaOpacaValida(v.PrincipalID) || !domain.ReferenciaOpacaValida(v.PerfilActivoRef) {
		return vacio, ports.ErrAutorizacionDenegada
	}
	m := ports.MaterialReincorporacionTitular{OrganizacionRef: sol.Canal.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef,
		RelacionRef: sol.RelacionRef, ActorRef: v.PrincipalID, PerfilRef: v.PerfilActivoRef, FechaEfectiva: sol.FechaEfectiva,
		DocumentoRef: sol.DocumentoRef, DocumentoSHA256: sol.DocumentoSHA256, VersionEsperada: sol.VersionEsperada, ClaveIdempotencia: sol.ClaveIdempotencia}
	if !m.Valido() {
		return vacio, ErrSolicitudReincorporacionTitularInvalida
	}
	causas, politica, err := s.d.Reglas.CausasCese(ctx, ahora)
	if err != nil || !politica.ValidaEn(ahora) {
		return vacio, ports.ErrOperacionSeguimientoNoDisponible
	}
	encontrada := false
	for _, causa := range causas {
		if causa.Clave == "fin_sustitucion" {
			encontrada = true
			break
		}
	}
	if !encontrada {
		return vacio, ports.ErrOperacionSeguimientoNoDisponible
	}
	materialHuella, _ := json.Marshal(struct {
		Operacion, Organizacion, Expediente, Relacion, Actor, Perfil, Fecha, Documento, DocumentoSHA256 string
		Version                                                                                         uint64
	}{ports.OperacionRegistrarReincorporacionTitular, m.OrganizacionRef, m.ExpedienteRef, m.RelacionRef, m.ActorRef, m.PerfilRef,
		m.FechaEfectiva.Format(time.DateOnly), m.DocumentoRef, m.DocumentoSHA256, m.VersionEsperada})
	ambitos, err := s.d.Sellos.SellarAmbitoReincorporacionTitular(ctx, ports.SolicitudSellarAmbitoIdempotencia{ClaveIdempotencia: m.ClaveIdempotencia,
		OrganizacionRef: m.OrganizacionRef, ActorRef: m.ActorRef, PerfilRef: m.PerfilRef})
	if err != nil {
		return vacio, ports.ErrOperacionSeguimientoNoDisponible
	}
	huellas, err := s.d.Sellos.DerivarHuellaReincorporacionTitular(ctx, materialHuella)
	if err != nil {
		return vacio, ports.ErrOperacionSeguimientoNoDisponible
	}
	refs, err := s.d.Referencias.GenerarReferenciasSeguimiento(ctx)
	if err != nil || !refs.Validas() {
		return vacio, ports.ErrOperacionSeguimientoNoDisponible
	}
	prep, err := s.d.Repositorio.PrepararReincorporacionTitular(ctx, m, ports.SellosOperacionSeguimiento{Ambitos: ambitos, Huellas: huellas}, refs)
	if err != nil {
		return vacio, err
	}
	if !prep.Referencias.Validas() || !ports.ColeccionesHMACContienenPar(ambitos, ports.DominioAmbitoReincorporacionTitular,
		huellas, ports.DominioHuellaReincorporacionTitular, prep.AmbitoIdempotenciaHMAC, prep.HuellaPeticionHMAC) ||
		!domain.ReferenciaOpacaValida(prep.CeseEventoRef) || !domain.ReferenciaOpacaValida(prep.CeseReciboRef) {
		return vacio, ports.ErrResultadoSeguimientoNoConfiable
	}
	atributos := map[string]string{"version_expediente": strconv.FormatUint(m.VersionEsperada, 10), "relacion_ref": m.RelacionRef,
		"fecha_efectiva": m.FechaEfectiva.Format(time.DateOnly), "documento_ref": m.DocumentoRef, "documento_sha256": m.DocumentoSHA256,
		"cese_evento_ref": prep.CeseEventoRef, "cese_recibo_ref": prep.CeseReciboRef,
		"ambito_idempotencia_hmac": prep.AmbitoIdempotenciaHMAC, "huella_peticion_hmac": prep.HuellaPeticionHMAC,
		"politica_ref": politica.DefinicionRef, "politica_version": strconv.FormatUint(politica.DefinicionVersion, 10),
		"politica_huella_sha256": politica.DefinicionHuellaSHA256}
	ambitosRecurso := ambitosSeguimiento(m.OrganizacionRef, m.ExpedienteRef)
	recurso := vd.RecursoAutorizable{Referencia: m.ExpedienteRef, ModuloID: ports.ModuloContratacion,
		Tipo: ports.TipoRecursoReincorporacionTitular, Ambitos: ambitosRecurso, Atributos: atributos}
	autorizacion, err := s.d.Autorizador.AutorizarOperacionSeguimiento(ctx, ports.SolicitudAutorizarOperacionSeguimiento{
		Accion: domain.AccionRegistrarReincorporacionTitular, Finalidad: ports.FinalidadRegistrarReincorporacionTitular,
		Audiencia: ports.AudienciaConsumoReincorporacionTitularV1, Motivo: politica.MotivoAutorizacion, Recurso: recurso})
	if err != nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	if prep.Confirmada {
		if prep.Recibo == nil || !prep.Recibo.ValidoPara(m) || prep.Recibo.CeseEventoRef != prep.CeseEventoRef ||
			prep.Recibo.CeseReciboRef != prep.CeseReciboRef {
			return vacio, ports.ErrResultadoSeguimientoNoConfiable
		}
		return *prep.Recibo, nil
	}
	if prep.Expediente.Validar() != nil || prep.Expediente.Referencia != m.ExpedienteRef || prep.Expediente.OrganizacionRef != m.OrganizacionRef ||
		prep.Expediente.Version != m.VersionEsperada || prep.Expediente.Asignacion == nil {
		return vacio, ports.ErrResultadoSeguimientoNoConfiable
	}
	instante := instanteCanonico(s.d.Reloj.Ahora())
	if !politica.ValidaEn(instante) {
		return vacio, ports.ErrOperacionSeguimientoNoDisponible
	}
	siguiente, err := prep.Expediente.RegistrarReincorporacionTitular(m.VersionEsperada, domain.DatosReincorporacionTitular{
		RelacionRef: m.RelacionRef, FechaEfectiva: m.FechaEfectiva, DocumentoRef: m.DocumentoRef, DocumentoSHA256: m.DocumentoSHA256},
		domain.DatosActuacion{AccionClave: domain.AccionRegistrarReincorporacionTitular, ActorRef: m.ActorRef,
			UnidadRef: prep.Expediente.Asignacion.UnidadRef, ReciboRef: prep.Referencias.ReciboRef, RealizadaEn: instante,
			FaseDestino: domain.FaseNombramiento, EstadoDestino: domain.EstadoEnCurso, DocumentosRef: []string{m.DocumentoRef}})
	if err != nil {
		return vacio, err
	}
	orden := ports.OrdenConfirmarReincorporacionTitular{Material: m, Preparacion: prep, Siguiente: siguiente, Politica: politica,
		Contexto: ports.ContextoAutorizadoSeguimiento{Ambitos: ambitosRecurso, Atributos: atributos}, Autorizacion: autorizacion, InstanteEfecto: instante}
	recibo, err := s.d.Repositorio.ConfirmarReincorporacionTitular(ctx, orden)
	if err != nil {
		return vacio, err
	}
	if !recibo.ValidoPara(m) || recibo.CeseEventoRef != prep.CeseEventoRef || recibo.CeseReciboRef != prep.CeseReciboRef ||
		recibo.ReciboRef != prep.Referencias.ReciboRef || recibo.EventoRef != prep.Referencias.EventoRef ||
		!recibo.RegistradaEn.Equal(instante) {
		return vacio, ports.ErrResultadoSeguimientoNoConfiable
	}
	return recibo, nil
}
