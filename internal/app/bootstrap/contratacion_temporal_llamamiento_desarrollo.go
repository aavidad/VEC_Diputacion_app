package bootstrap

import (
	"context"
	"net/http"
	"path/filepath"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ejecutorSeleccionLlamamientoDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	lector      ports.LectorExpedienteSeleccionLlamamiento
	autorizador *autorizadorLlamamientoDesarrollo
	preparar    func(ports.ExpedienteParaSeleccion, string) (preparacionLlamamientoDesarrollo, error)
	servicio    httpinterno.EjecutorSeleccionLlamamiento
}

var _ httpinterno.EjecutorSeleccionLlamamiento = (*ejecutorSeleccionLlamamientoDesarrollo)(nil)

func nuevasDependenciasLlamamientoContratacionTemporalDesarrollo(cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo, reloj relojContratacionTemporalDesarrollo,
) (httpinterno.EjecutorSeleccionLlamamiento, http.Handler, error) {
	if alta == nil || alta.postgresql.bolsa == nil || derivador == nil || !derivador.valido() ||
		!cfg.DevelopmentEnabledByDoubleKey() || !filepath.IsAbs(cfg.DevelopmentMaterialDir) {
		return nil, nil, ports.ErrIntegracionBolsaNoDisponible
	}
	if err := configurarAutoridadLlamamientoDesarrollo(alta, reloj); err != nil {
		return nil, nil, err
	}
	lector, err := postgresct.NuevoLectorExpedienteSeleccionLlamamientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, nil, err
	}
	secretos, err := derivador.calcularHMAC([]byte("vec.ct.desarrollo.puente-bolsa.v1"), []byte("vec.ct.desarrollo.puente-bolsa.huella.v1"))
	if err != nil || len(secretos) == 0 {
		return nil, nil, ports.ErrIntegracionBolsaNoDisponible
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(secretos)
	puente, err := nuevoPuenteBolsaLlamamientoDesarrollo(alta.postgresql.bolsa, alta,
		alta.postgresql.proveedorMaterialBolsa, reloj, secretos[0].localizador[:])
	if err != nil {
		return nil, nil, err
	}
	ejecuciones, err := postgresct.NuevasEjecucionesSeleccionLlamamientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, nil, err
	}
	reanudables := &ejecucionesSeleccionReanudablesDesarrollo{
		EjecucionesSeleccionLlamamientoPostgreSQL: ejecuciones,
		autorizador: &autorizadorLlamamientoDesarrollo{alta: alta, material: alta.postgresql.proveedorMaterial},
	}
	servicio, err := application.NuevoServicioSeleccionLlamamiento(puente, reanudables, puente, puente, puente, puente.Verificador(), reloj)
	if err != nil {
		return nil, nil, err
	}
	seleccion := &ejecutorSeleccionLlamamientoDesarrollo{
		soporte: alta.soporte, lector: lector, preparar: prepararReferenciasLlamamientoDesarrollo, servicio: servicio,
		autorizador: &autorizadorLlamamientoDesarrollo{alta: alta, material: alta.postgresql.proveedorMaterialBolsa},
	}
	comunicacion, err := nuevoEjecutorComunicacionLlamamientoDesarrollo(alta.postgresql.ejecucion, alta,
		alta.postgresql.proveedorMaterial, reloj, lector, filepath.Join(cfg.DevelopmentMaterialDir, "comunicaciones"))
	if err != nil {
		return nil, nil, err
	}
	manejador, err := httpinterno.NuevoManejadorComunicacionLlamamiento(comunicacion)
	if err != nil {
		return nil, nil, err
	}
	return seleccion, manejador, nil
}

func (e *ejecutorSeleccionLlamamientoDesarrollo) SeleccionarYLlamarParaAdaptador(ctx context.Context, solicitud application.SolicitudSeleccionLlamamiento) (application.DatosReciboSeleccionLlamamientoParaAdaptador, error) {
	vacio := application.DatosReciboSeleccionLlamamientoParaAdaptador{}
	if ctx == nil || e == nil || e.soporte == nil || e.autorizador == nil || e.preparar == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.lector) || dependenciaEsNulaContratacionTemporalDesarrollo(e.servicio) {
		return vacio, application.ErrServicioSeleccionLlamamientoInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if !domain.ReferenciaOpacaValida(solicitud.ExpedienteRef) || solicitud.VersionEsperada != 6 ||
		!ports.ClaveIdempotenciaValida(solicitud.ClaveIdempotencia) {
		return vacio, application.ErrSolicitudSeleccionLlamamientoInvalida
	}
	capacidad, valida := e.soporte.capacidadValida(ctx)
	if !valida || capacidad.ruta != httpinterno.RutaSeleccionLlamamiento {
		return vacio, ports.ErrAutorizacionDenegada
	}
	// Solo se lee el agregado de la organización sintética ligada a la identidad.
	// La petición no puede elegir organización, perfil, orden ni autorización.
	expediente, err := e.lector.LeerExpedienteParaSeleccion(ctx,
		organizacionAltaContratacionTemporalDesarrollo, solicitud.ExpedienteRef, solicitud.VersionEsperada)
	if err != nil {
		return vacio, err
	}
	if expediente.Fiscalizado.Validar() != nil || expediente.Fiscalizado.Referencia != solicitud.ExpedienteRef ||
		expediente.Fiscalizado.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		expediente.Fiscalizado.Version != solicitud.VersionEsperada {
		return vacio, ports.ErrIntegracionBolsaNoDisponible
	}
	preparacion, err := e.preparar(expediente, solicitud.ClaveIdempotencia)
	if err != nil {
		return vacio, err
	}
	preparacion.expediente, preparacion.clave = expediente, solicitud.ClaveIdempotencia
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacion)
	if err := e.autorizador.exigirLecturaSeleccion(ctx, expediente); err != nil {
		return vacio, err
	}
	// El servicio conserva la recuperación terminal existente. El proveedor
	// rechaza efectos nuevos sobre versiones históricas; el replay sigue ligado
	// al mismo expediente, intención y actor autorizado en esta petición.
	return e.servicio.SeleccionarYLlamarParaAdaptador(ctx, solicitud)
}
