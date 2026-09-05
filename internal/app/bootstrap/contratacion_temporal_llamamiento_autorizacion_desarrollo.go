package bootstrap

import (
	"context"
	"maps"
	"sort"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type clavePreparacionLlamamientoDesarrollo struct{}

// La instala el decorador confiable con la reserva que va a recuperar.
type claveReanudacionSeleccionDesarrollo struct{}

const accionConsultarLlamamientoDesarrollo = "contratacion_temporal.llamamiento.consultar"
const tipoRecursoConsultaLlamamientoDesarrollo = "seleccion_llamamiento_contratacion_temporal"

// Solo la raíz construye esta ligadura después de leer el expediente propio
// fiscalizado. Ningún dato de identidad, permiso u orden se toma del formulario.
type preparacionLlamamientoDesarrollo struct {
	expediente         ports.ExpedienteParaSeleccion
	clave              string
	necesidad          string
	categoria          string
	unidad             string
	operacionOrden     string
	operacionPropuesta string
}

func rutaLlamamientoContratacionTemporalDesarrollo(ruta string) bool {
	return ruta == httpinterno.RutaSeleccionLlamamiento ||
		ruta == httpinterno.RutaRegistroComunicacionLlamamiento ||
		ruta == httpinterno.RutaResolucionComunicacionLlamamiento
}

func rutaMutacionDurableContratacionTemporalDesarrollo(ruta string) bool {
	return rutaAnalisisContratacionTemporalDesarrollo(ruta) ||
		rutaAsignacionContratacionTemporalDesarrollo(ruta) ||
		rutaInformeJuridicoContratacionTemporalDesarrollo(ruta) ||
		rutaLlamamientoContratacionTemporalDesarrollo(ruta)
}

func ambitosLlamamientoDesarrollo(recurso dominiovec.RecursoAutorizable) []dominiovec.AmbitoPerfil {
	claves := make([]string, 0, len(recurso.Ambitos))
	for clave := range recurso.Ambitos {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	ambitos := make([]dominiovec.AmbitoPerfil, 0, len(claves))
	for _, clave := range claves {
		ambitos = append(ambitos, dominiovec.AmbitoPerfil{Clave: clave, Valores: []string{recurso.Ambitos[clave]}})
	}
	return ambitos
}

func solicitudAutorizacionLlamamientoDesarrolloValida(ctx context.Context, ruta string, datos dominiovec.DatosSolicitudAutorizacionLigadaV3) bool {
	if ctx == nil || datos.Finalidad != "gestionar_contratacion_temporal" {
		return false
	}
	p, ok := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	if !ok || p.expediente.Fiscalizado.Validar() != nil ||
		p.expediente.Fiscalizado.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo ||
		p.expediente.Fiscalizado.Fiscalizacion == nil ||
		p.expediente.Fiscalizado.Fiscalizacion.Resultado == domain.FiscalizacionDesfavorable {
		return false
	}
	r := datos.Recurso
	if ruta == httpinterno.RutaSeleccionLlamamiento {
		if datos.Accion == ports.AccionReanudacionSeleccionLlamamiento {
			reserva, existe := ctx.Value(claveReanudacionSeleccionDesarrollo{}).(ports.SolicitudReservaEjecucionSeleccionLlamamiento)
			if !existe || datos.ReferenciaMotivo != motivoLlamamientoDesarrollo(false) ||
				!reservaReanudacionLigadaAPreparacionDesarrollo(p, reserva) {
				return false
			}
			esperado, err := ports.NuevoRecursoReanudacionSeleccionLlamamiento(reserva)
			return err == nil && r.Referencia == esperado.Referencia &&
				r.ModuloID == esperado.ModuloID && r.Tipo == esperado.Tipo &&
				maps.Equal(r.Ambitos, esperado.Ambitos) && maps.Equal(r.Atributos, esperado.Atributos)
		}
		if datos.Accion == accionConsultarLlamamientoDesarrollo {
			return datos.ReferenciaMotivo == motivoLlamamientoDesarrollo(false) &&
				r.ModuloID == "contratacion_temporal" && r.Tipo == tipoRecursoConsultaLlamamientoDesarrollo &&
				r.Referencia == p.expediente.Fiscalizado.Referencia &&
				len(r.Ambitos) == 1 && r.Ambitos["organizacion_ref"] == p.expediente.Fiscalizado.OrganizacionRef &&
				len(r.Atributos) == 1 && r.Atributos["version_expediente"] == numeroDecimal64(p.expediente.Fiscalizado.Version)
		}
		return datos.ReferenciaMotivo == motivoLlamamientoDesarrollo(false) &&
			r.ModuloID == "bolsa" && r.Tipo == "integracion_llamamientos_bolsa" &&
			len(r.Ambitos) == 2 && r.Ambitos["categoria_ref"] == p.categoria &&
			r.Ambitos["unidad_ref"] == p.unidad && len(r.Atributos) == 2 &&
			r.Atributos["necesidad_ref"] == p.necesidad &&
			huellaSHA256ValidaContratacionTemporalDesarrollo(r.Atributos["contenido_sha256"]) &&
			((datos.Accion == puertosbolsa.AccionPrepararOrdenDesarrollo && r.Referencia == p.operacionOrden) ||
				(datos.Accion == puertosbolsa.AccionAbrirLlamamientoDesarrollo && r.Referencia == p.operacionPropuesta))
	}
	return rutaLlamamientoContratacionTemporalDesarrollo(ruta) &&
		datos.ReferenciaMotivo == motivoLlamamientoDesarrollo(true) &&
		datos.Accion == postgresct.AccionRegistroComunicacionLlamamiento &&
		r.ModuloID == "contratacion_temporal" && r.Tipo == postgresct.TipoRecursoRegistroComunicacionLlamamiento &&
		r.Referencia == p.expediente.Fiscalizado.Referencia &&
		len(r.Ambitos) == 1 && r.Ambitos["organizacion_ref"] == p.expediente.Fiscalizado.OrganizacionRef &&
		len(r.Atributos) == 1 && huellaSHA256ValidaContratacionTemporalDesarrollo(r.Atributos["material_sha256"])
}

func huellaSHA256ValidaContratacionTemporalDesarrollo(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	noNula := false
	for _, c := range valor {
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
		noNula = noNula || c != '0'
	}
	return noNula
}

func motivoLlamamientoDesarrollo(comunicacion bool) dominiovec.ReferenciaEntradaCatalogo {
	accion := "seleccion"
	if comunicacion {
		accion = "comunicacion"
	}
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_llamamiento",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("llamamiento-desarrollo-seleccion-comunicacion-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", "llamamiento-"+accion),
	}
}

func configurarAutoridadLlamamientoDesarrollo(alta *dependenciasAltaContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo) error {
	if alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil {
		return ports.ErrIntegracionBolsaNoDisponible
	}
	s := alta.soporte
	vinculo, err := s.contexto.Vinculo.Datos()
	if err != nil {
		return err
	}
	concesion := func(accion, modulo, tipo string) dominiovec.ConcesionRol {
		return dominiovec.ConcesionRol{Accion: accion, ModuloID: modulo, TipoRecurso: tipo,
			Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: dominiovec.AuthAssuranceHigh}
	}
	seleccion, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		vinculo.PrincipalID, vinculo.PerfilActivoRef, reloj.Ahora(),
		"llamamiento_desarrollo", "Llamamiento de desarrollo", "llamamiento-desarrollo",
		[]dominiovec.ConcesionRol{
			concesion(accionConsultarLlamamientoDesarrollo, "contratacion_temporal", tipoRecursoConsultaLlamamientoDesarrollo),
			concesion(puertosbolsa.AccionPrepararOrdenDesarrollo, "bolsa", "integracion_llamamientos_bolsa"),
			concesion(puertosbolsa.AccionAbrirLlamamientoDesarrollo, "bolsa", "integracion_llamamientos_bolsa"),
		}, []dominiovec.AmbitoPerfil{
			{Clave: "categoria_ref", Valores: []string{"categoria:desarrollo:c2"}},
			{Clave: "unidad_ref", Valores: []string{unidadCoberturaContratacionTemporalDesarrollo}},
		})
	if err != nil {
		return err
	}
	comunicacion, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		vinculo.PrincipalID, vinculo.PerfilActivoRef, reloj.Ahora(),
		"comunicacion_llamamiento_desarrollo", "Comunicación de desarrollo", "comunicacion-llamamiento-desarrollo",
		[]dominiovec.ConcesionRol{concesion(postgresct.AccionRegistroComunicacionLlamamiento, "contratacion_temporal", postgresct.TipoRecursoRegistroComunicacionLlamamiento)},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	// No alterar una versión de rol ya publicada para los efectos de Bolsa.
	// Esta instantánea concede únicamente la reanudación de la orden en CT.
	reanudacion, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		vinculo.PrincipalID, vinculo.PerfilActivoRef, reloj.Ahora(),
		"reanudacion_llamamiento_desarrollo", "Reanudación de orden de desarrollo", "reanudacion-llamamiento-desarrollo",
		[]dominiovec.ConcesionRol{concesion(ports.AccionReanudacionSeleccionLlamamiento, "contratacion_temporal", ports.TipoRecursoReanudacionSeleccionLlamamiento)},
		[]dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		return err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	desde, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(reloj.Ahora())
	if !vigente {
		return ports.ErrIntegracionBolsaNoDisponible
	}
	if err := publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno,
		[]dominiovec.ReferenciaEntradaCatalogo{motivoLlamamientoDesarrollo(false), motivoLlamamientoDesarrollo(true)}, desde); err != nil {
		return err
	}
	s.mu.Lock()
	s.instantaneaLlamamiento, s.instantaneaComunicacion = seleccion, comunicacion
	s.instantaneaReanudacionLlamamiento = reanudacion
	s.motivoLlamamiento, s.motivoComunicacion = motivoLlamamientoDesarrollo(false), motivoLlamamientoDesarrollo(true)
	s.mu.Unlock()
	return nil
}

type autorizadorLlamamientoDesarrollo struct {
	alta         *dependenciasAltaContratacionTemporalDesarrollo
	material     *proveedorMaterialAltaContratacionTemporalDesarrollo
	comunicacion bool
}

func reservaReanudacionLigadaAPreparacionDesarrollo(p preparacionLlamamientoDesarrollo, s ports.SolicitudReservaEjecucionSeleccionLlamamiento) bool {
	canonica, err := prepararReferenciasLlamamientoDesarrollo(p.expediente, p.clave)
	return err == nil && p.expediente.VersionActual == 6 && s.Validar() == nil &&
		s.ClaveIdempotencia == p.clave && s.VersionExpediente == 6 &&
		s.OrganizacionRef == p.expediente.Fiscalizado.OrganizacionRef &&
		s.ExpedienteRef == p.expediente.Fiscalizado.Referencia &&
		s.Necesidad.Referencia == canonica.necesidad &&
		p.necesidad == canonica.necesidad && p.categoria == canonica.categoria &&
		p.unidad == canonica.unidad && p.operacionOrden == canonica.operacionOrden &&
		p.operacionPropuesta == canonica.operacionPropuesta
}

func (a *autorizadorLlamamientoDesarrollo) AutorizarOperacion(ctx context.Context, accion string, recurso dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if a == nil || a.material == nil {
		return vacio, ports.ErrAutorizacionDenegada
	}
	solicitud, decision, confirmacion, err := a.exigirOperacion(ctx, accion, recurso)
	if err != nil {
		return vacio, err
	}
	return a.material.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion,
		motivoLlamamientoDesarrollo(a.comunicacion), a.alta.soporte.contexto.Resultado)
}

// La recuperación es una lectura autorizada nueva, aunque el resultado sea
// histórico. No fabrica una capacidad de consumo ni otro efecto para un replay.
func (a *autorizadorLlamamientoDesarrollo) exigirLecturaSeleccion(ctx context.Context, expediente ports.ExpedienteParaSeleccion) error {
	_, _, _, err := a.exigirOperacion(ctx, accionConsultarLlamamientoDesarrollo, dominiovec.RecursoAutorizable{
		Referencia: expediente.Fiscalizado.Referencia, ModuloID: "contratacion_temporal", Tipo: tipoRecursoConsultaLlamamientoDesarrollo,
		Ambitos:   map[string]string{"organizacion_ref": expediente.Fiscalizado.OrganizacionRef},
		Atributos: map[string]string{"version_expediente": numeroDecimal64(expediente.Fiscalizado.Version)},
	})
	return err
}

func (a *autorizadorLlamamientoDesarrollo) exigirOperacion(ctx context.Context, accion string, recurso dominiovec.RecursoAutorizable) (dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	fallo := func(err error) (dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
		return dominiovec.SolicitudAutorizacionLigadaV3{}, dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, err
	}
	if ctx == nil || a == nil || a.alta == nil || a.alta.soporte == nil || a.material == nil {
		return fallo(ports.ErrAutorizacionDenegada)
	}
	s := a.alta.soporte
	capacidad, valida := s.capacidadValida(ctx)
	if !valida || !rutaLlamamientoContratacionTemporalDesarrollo(capacidad.ruta) {
		return fallo(ports.ErrAutorizacionDenegada)
	}
	if accion == ports.AccionReanudacionSeleccionLlamamiento && a.comunicacion {
		return fallo(ports.ErrAutorizacionDenegada)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return fallo(err)
	}
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: s.contexto.Vinculo, ReferenciaMotivo: motivoLlamamientoDesarrollo(a.comunicacion),
		Accion: accion, Recurso: recurso, Finalidad: "gestionar_contratacion_temporal", Correlacion: correlacion,
	}
	if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, capacidad.ruta, datos) {
		return fallo(ports.ErrAutorizacionDenegada)
	}
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return fallo(err)
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := a.alta.autorizador.ExigirSolicitudLigadaV3(ctx, solicitud, s.contexto.Resultado)
	if err != nil {
		return fallo(err)
	}
	return solicitud, decision, confirmacion, nil
}
