package application

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// SolicitudConfirmarGINPIX registra el número de alta que devolvió GINPIX
// para la ficha de la incorporación acreditada del expediente.
type SolicitudConfirmarGINPIX struct {
	Canal             ContextoCanalSeguimiento
	ExpedienteRef     string
	VersionEsperada   uint64
	ClaveIdempotencia string
	GINPIXNumero      string
	GINPIXConfirmada  time.Time
	Observaciones     string
}

// ConfirmarGINPIX registra la confirmación de GINPIX con la política de la
// regla del cierre (c10). La incorporación, su fecha y la unicidad las
// comprueba SQL.
func (s *ServicioOperacionesSeguimiento) ConfirmarGINPIX(ctx context.Context, sol SolicitudConfirmarGINPIX) (ports.ReciboOperacionSeguimiento, error) {
	if s == nil {
		return ports.ReciboOperacionSeguimiento{}, ErrServicioSeguimientoInvalido
	}
	if s.ginpix == nil {
		return ports.ReciboOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	ctx, cancelar, err := contextoOperacionSeguimiento(ctx)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	defer cancelar()
	actor, perfil, err := s.identidad(ctx, sol.Canal)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	ahora := instanteCanonico(s.reloj.Ahora())
	politica, err := s.ginpix.PoliticaConfirmacionGINPIX(ctx, ahora)
	if err != nil || !politica.ValidaEn(ahora) {
		return ports.ReciboOperacionSeguimiento{}, ports.ErrOperacionSeguimientoNoDisponible
	}
	datos := domain.DatosConfirmacionGINPIX{Numero: sol.GINPIXNumero, ConfirmadaEn: sol.GINPIXConfirmada, Observaciones: sol.Observaciones}
	material := ports.MaterialConfirmacionGINPIX{OrganizacionRef: sol.Canal.OrganizacionRef, ExpedienteRef: sol.ExpedienteRef, ActorRef: actor,
		PerfilRef: perfil, VersionEsperada: sol.VersionEsperada, ClaveIdempotencia: sol.ClaveIdempotencia, Datos: datos}
	if !material.Valido() {
		return ports.ReciboOperacionSeguimiento{}, ErrSolicitudSeguimientoInvalida
	}
	fecha := datos.ConfirmadaEn.Format(time.DateOnly)
	huella, _ := json.Marshal(struct {
		Operacion, Organizacion, Expediente, Actor, Perfil, Numero, Fecha, Observaciones string
		Version                                                                          uint64
	}{ports.OperacionConfirmarGINPIX, material.OrganizacionRef, material.ExpedienteRef, actor, perfil, datos.Numero, fecha,
		datos.Observaciones, sol.VersionEsperada})
	prep, err := s.preparar(ctx, ports.OperacionConfirmarGINPIX, material.OrganizacionRef, actor, perfil, sol.ClaveIdempotencia, material, huella)
	if err != nil {
		return ports.ReciboOperacionSeguimiento{}, err
	}
	instante := instanteCanonico(s.reloj.Ahora())
	orden := ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionConfirmarGINPIX, Material: material, Preparacion: prep,
		Politica: politica, InstanteEfecto: instante, Accion: domain.AccionConfirmarGINPIX, Finalidad: ports.FinalidadConfirmarGINPIX,
		Audiencia: ports.AudienciaConsumoConfirmacionGINPIXV1}
	orden.Contexto = ports.ContextoAutorizadoSeguimiento{Ambitos: ambitosSeguimiento(material.OrganizacionRef, material.ExpedienteRef),
		Atributos: atributosPoliticaSeguimiento(map[string]string{"ginpix_numero": datos.Numero, "ginpix_confirmada_en": fecha,
			"observaciones_huella_sha256": huellaTexto(datos.Observaciones), "incorporacion_ref": prep.IncorporacionRef}, politica, prep, sol.VersionEsperada)}
	if !prep.Confirmada {
		if !domain.ReferenciaOpacaValida(prep.IncorporacionRef) || prep.Expediente.Version != sol.VersionEsperada ||
			prep.Expediente.Referencia != sol.ExpedienteRef || prep.Expediente.OrganizacionRef != material.OrganizacionRef ||
			prep.Expediente.Asignacion == nil {
			return ports.ReciboOperacionSeguimiento{}, ports.ErrResultadoSeguimientoNoConfiable
		}
		orden.Siguiente, err = prep.Expediente.ConfirmarGINPIX(sol.VersionEsperada, datos, domain.DatosActuacion{AccionClave: domain.AccionConfirmarGINPIX,
			ActorRef: actor, UnidadRef: prep.Expediente.Asignacion.UnidadRef, ReciboRef: prep.Referencias.ReciboRef, RealizadaEn: instante,
			FaseDestino: domain.FaseNombramiento, EstadoDestino: domain.EstadoEnCurso, Observaciones: datos.Observaciones,
			DocumentosRef: []string{datos.DocumentoGINPIX()}})
		if err != nil {
			return ports.ReciboOperacionSeguimiento{}, err
		}
	}
	return s.confirmar(ctx, orden, material.OrganizacionRef, material.ExpedienteRef, sol.VersionEsperada)
}

// ginpixDelCierre fija el número y la fecha de GINPIX del cierre. Con la
// incorporación acreditada compuesta salen de la confirmación registrada: si
// la solicitud trae otros, se rechaza; si la regla exige GINPIX y no hay
// confirmación, el cierre no procede. Sin ella se conserva la conducta
// anterior (los aporta la solicitud).
func (s *ServicioOperacionesSeguimiento) ginpixDelCierre(ctx context.Context, sol SolicitudCerrarExpediente, regla ports.ReglaCierreExpediente) (string, *time.Time, error) {
	if s.acreditada == nil {
		return sol.GINPIXNumero, sol.GINPIXConfirmadaEn, nil
	}
	if !slices.Contains(regla.Condiciones, domain.CondicionGINPIXConfirmado) {
		if sol.GINPIXNumero != "" || sol.GINPIXConfirmadaEn != nil {
			return "", nil, ErrSolicitudSeguimientoInvalida
		}
		return "", nil, nil
	}
	if !sol.Canal.Valido() || !domain.ReferenciaOpacaValida(sol.ExpedienteRef) {
		return "", nil, ErrSolicitudSeguimientoInvalida
	}
	acreditada, err := s.acreditada.ConsultarIncorporacionAcreditada(ctx, sol.Canal.OrganizacionRef, sol.ExpedienteRef)
	if err != nil {
		return "", nil, err
	}
	if !acreditada.Valido() {
		return "", nil, ports.ErrResultadoSeguimientoNoConfiable
	}
	if acreditada.GINPIX == nil {
		return "", nil, ports.ErrGINPIXNoConfirmado
	}
	fecha, err := time.Parse(time.DateOnly, acreditada.GINPIX.ConfirmadaEn)
	if err != nil {
		return "", nil, ports.ErrResultadoSeguimientoNoConfiable
	}
	fecha = fecha.UTC()
	if (sol.GINPIXNumero != "" && sol.GINPIXNumero != acreditada.GINPIX.Numero) ||
		(sol.GINPIXConfirmadaEn != nil && !sol.GINPIXConfirmadaEn.Equal(fecha)) {
		return "", nil, ports.ErrGINPIXDistinto
	}
	return acreditada.GINPIX.Numero, &fecha, nil
}
