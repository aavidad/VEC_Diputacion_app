package bootstrap

import (
	"context"
	"errors"
	"maps"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

// Atributos del catálogo de reglas que gobiernan el llamamiento. El inicio
// admitido es el contacto efectivo: el aviso por correo nunca abre plazo.
const (
	inicioPlazoContactoEfectivo    = "contacto_efectivo"
	atributoTratamientoPlazo       = "tratamiento"
	atributoConfirmacionPlazo      = "confirmacion"
	reglaPlazoRespuestaLlamamiento = reglas.BolsaPlazoRespuesta
)

// reglasPlazoLlamamientoDesarrollo traduce el catálogo de reglas de Bolsa
// (Reglamento de bolsas: plazo de respuesta, respuesta fuera de plazo, falta
// de respuesta y siguiente candidato) al puerto neutral de Contratación. Sin
// catálogo compuesto responde ErrReglasPlazoNoDisponibles: nunca supone.
type reglasPlazoLlamamientoDesarrollo struct {
	resolutor *reglas.Resolutor
}

var _ ports.ReglasPlazoRespuestaLlamamiento = reglasPlazoLlamamientoDesarrollo{}

func (r reglasPlazoLlamamientoDesarrollo) PlazoRespuesta(ctx context.Context, contacto time.Time) (ports.PlazoRespuestaGobernado, error) {
	vacio := ports.PlazoRespuestaGobernado{}
	if contextoInterfazNulo(ctx) || !r.resolutor.Disponible() || !domain.InstanteUTCCanonico(contacto) {
		return vacio, ports.ErrReglasPlazoNoDisponibles
	}
	plazo, vencimiento, err := r.resolutor.Vencimiento(ctx, reglaPlazoRespuestaLlamamiento, contacto, "")
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil || plazo.Inicio != inicioPlazoContactoEfectivo {
		return vacio, ports.ErrReglasPlazoNoDisponibles
	}
	respuesta, expiracion, ejemplo, err := r.criterios(ctx)
	if err != nil {
		return vacio, err
	}
	resultado := ports.PlazoRespuestaGobernado{
		RespuestaHasta: vencimiento.VenceAntesDe.UTC().Truncate(time.Microsecond), UltimoDia: vencimiento.UltimoDia,
		Politica:                referenciaGobernadaRegla(plazo),
		TratamientoFueraDePlazo: respuesta.tratamiento, ConfirmacionExpiracion: expiracion.confirmacion,
		CriterioRespuesta: respuesta.referencia, CriterioExpiracion: expiracion.referencia,
		ReglaEjemplo: ejemplo || plazo.EsEjemplo() || plazo.PaqueteEjemplo,
	}
	if resultado.ValidarDesde(contacto) != nil {
		return vacio, ports.ErrReglasPlazoNoDisponibles
	}
	return resultado, nil
}

type criterioRespuestaDesarrollo struct {
	referencia  ports.ReferenciaGobernadaComunicacionLlamamiento
	tratamiento domain.TratamientoRespuestaFueraDePlazo
}

type criterioExpiracionDesarrollo struct {
	referencia   ports.ReferenciaGobernadaComunicacionLlamamiento
	confirmacion domain.ConfirmacionExpiracionLlamamiento
}

// criterios lee las reglas vigentes que RRHH aplica al resolver. La
// confirmación de la no aceptación (b08) y del siguiente candidato (b09) debe
// coincidir: una propuesta confirmada por RRHH cubre ambas decisiones.
func (r reglasPlazoLlamamientoDesarrollo) criterios(ctx context.Context) (criterioRespuestaDesarrollo, criterioExpiracionDesarrollo, bool, error) {
	var respuesta criterioRespuestaDesarrollo
	var expiracion criterioExpiracionDesarrollo
	if contextoInterfazNulo(ctx) || !r.resolutor.Disponible() {
		return respuesta, expiracion, false, ports.ErrReglasPlazoNoDisponibles
	}
	fuera, errFuera := r.resolutor.Regla(ctx, reglas.BolsaFueraDePlazo)
	sin, errSin := r.resolutor.Regla(ctx, reglas.BolsaSinRespuestaBaja)
	siguiente, errSiguiente := r.resolutor.Regla(ctx, reglas.BolsaSiguienteCandidato)
	if ctx.Err() != nil {
		return respuesta, expiracion, false, ctx.Err()
	}
	respuesta.tratamiento = domain.TratamientoRespuestaFueraDePlazo(fuera.Atributos[atributoTratamientoPlazo])
	expiracion.confirmacion = domain.ConfirmacionExpiracionLlamamiento(sin.Atributos[atributoConfirmacionPlazo])
	if errFuera != nil || errSin != nil || errSiguiente != nil || !respuesta.tratamiento.Valido() ||
		!expiracion.confirmacion.Valida() ||
		siguiente.Atributos[atributoConfirmacionPlazo] != string(expiracion.confirmacion) {
		return respuesta, expiracion, false, ports.ErrReglasPlazoNoDisponibles
	}
	respuesta.referencia, expiracion.referencia = referenciaGobernadaRegla(fuera), referenciaGobernadaRegla(sin)
	ejemplo := fuera.EsEjemplo() || sin.EsEjemplo() || siguiente.EsEjemplo() || fuera.PaqueteEjemplo
	return respuesta, expiracion, ejemplo, nil
}

// referenciaGobernadaRegla identifica la regla como catalogo:version:entrada
// con la huella del catálogo; una versión no positiva queda sin validar.
func referenciaGobernadaRegla(r reglas.Regla) ports.ReferenciaGobernadaComunicacionLlamamiento {
	version := uint64(0)
	if r.ReferenciaEntrada.CatalogoVersion > 0 {
		version = uint64(r.ReferenciaEntrada.CatalogoVersion)
	}
	return ports.ReferenciaGobernadaComunicacionLlamamiento{
		Referencia: r.Referencia, Version: version, HuellaSHA256: r.HuellaCatalogo,
	}
}

// politicaResolucionDesarrollo devuelve la política con la que se resuelve:
// la histórica sintética (solo aceptación o renuncia, sin plazo abierto) o
// la regla vigente del catálogo que el formulario recibió con el contacto.
// PostgreSQL vuelve a comprobarla contra su lista versionada de admitidas.
func (s *soporteAltaContratacionTemporalDesarrollo) politicaResolucionDesarrollo(
	ctx context.Context, solicitud ports.SolicitudResolverLlamamiento,
) (ports.ReferenciaGobernadaComunicacionLlamamiento, error) {
	vacia := ports.ReferenciaGobernadaComunicacionLlamamiento{}
	if s == nil || contextoInterfazNulo(ctx) {
		return vacia, errPoliticaResolucionNoAdmitida
	}
	if solicitud.CriterioValidacionRef == criterioRevisionManualDesarrollo {
		if solicitud.Respuesta == ports.RespuestaLlamamientoExpirada {
			return vacia, errPoliticaResolucionNoAdmitida
		}
		return politicaManualDesarrollo(), nil
	}
	reglasPlazo, ok := s.reglasPlazo.(reglasPlazoLlamamientoDesarrollo)
	if !ok {
		return vacia, ports.ErrReglasPlazoNoDisponibles
	}
	respuesta, expiracion, _, err := reglasPlazo.criterios(ctx)
	if err != nil {
		return vacia, err
	}
	politica := respuesta.referencia
	if solicitud.Respuesta == ports.RespuestaLlamamientoExpirada {
		politica = expiracion.referencia
	}
	if politica.Validar() != nil || politica.Referencia != solicitud.CriterioValidacionRef {
		return vacia, errPoliticaResolucionNoAdmitida
	}
	return politica, nil
}

var errPoliticaResolucionNoAdmitida = errors.New("bootstrap: criterio de resolución no admitido")

// politicaResolucionAdmitidaDesarrollo reconoce, en antecedentes ya
// persistidos, la política histórica o una regla del catálogo de reglas.
func politicaResolucionAdmitidaDesarrollo(referencia string, version uint64, huella string) bool {
	historica := politicaManualDesarrollo()
	if referencia == historica.Referencia {
		return version == historica.Version && huella == historica.HuellaSHA256
	}
	prefijo := reglas.CatalogoBolsa + ":"
	return len(referencia) > len(prefijo) && referencia[:len(prefijo)] == prefijo && version > 0 &&
		huellaSHA256ValidaContratacionTemporalDesarrollo(huella)
}

type claveEventoPlazoDesarrollo struct{}

// ejecutorEventoPlazoDesarrollo registra contacto efectivo o causa
// justificada. Reutiliza la precondición del expediente fiscalizado; la
// relación exacta con el aviso y su versión la comprueba CT111 tras AD3.
type ejecutorEventoPlazoDesarrollo struct {
	soporte  *soporteAltaContratacionTemporalDesarrollo
	lector   ports.LectorExpedienteLlamamiento
	servicio httpinterno.EjecutorEventoPlazoLlamamiento
}

type proveedorEventoPlazoDesarrollo struct {
	soporte     *soporteAltaContratacionTemporalDesarrollo
	autorizador autorizacionComunicacionLlamamientoDesarrollo
	reloj       ports.Reloj
}

func nuevoManejadorEventoPlazoDesarrollo(alta *dependenciasAltaContratacionTemporalDesarrollo, reloj ports.Reloj) (http.Handler, error) {
	if alta == nil || alta.soporte == nil || alta.postgresql.ejecucion == nil || alta.postgresql.proveedorMaterial == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(reloj) || dependenciaEsNulaContratacionTemporalDesarrollo(alta.soporte.reglasPlazo) {
		return nil, application.ErrServicioEventosPlazoInvalido
	}
	lector, err := postgresct.NuevoLectorExpedienteSeleccionLlamamientoPostgreSQL(alta.postgresql.ejecucion)
	if err != nil {
		return nil, err
	}
	proveedor := &proveedorEventoPlazoDesarrollo{soporte: alta.soporte, reloj: reloj,
		autorizador: &autorizadorLlamamientoDesarrollo{alta: alta, material: alta.postgresql.proveedorMaterial, resolucionManual: true}}
	registro, err := postgresct.NuevoRegistroEventosPlazoLlamamientoPostgreSQL(alta.postgresql.ejecucion, proveedor)
	if err != nil {
		return nil, err
	}
	servicio, err := application.NuevoServicioEventosPlazoLlamamiento(alta.soporte.reglasPlazo, registro)
	if err != nil {
		return nil, err
	}
	return httpinterno.NuevoManejadorEventoPlazoLlamamiento(&ejecutorEventoPlazoDesarrollo{soporte: alta.soporte, lector: lector, servicio: servicio})
}

func (e *ejecutorEventoPlazoDesarrollo) Registrar(ctx context.Context, s ports.SolicitudRegistrarEventoPlazoLlamamiento) (ports.EventoPlazoLlamamientoRegistrado, error) {
	vacio := ports.EventoPlazoLlamamientoRegistrado{}
	if e == nil || contextoInterfazNulo(ctx) || e.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(e.lector) || dependenciaEsNulaContratacionTemporalDesarrollo(e.servicio) {
		return vacio, application.ErrServicioEventosPlazoInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if s.Validar() != nil {
		return vacio, application.ErrSolicitudEventoPlazoInvalida
	}
	c, valida := e.soporte.capacidadValida(ctx)
	if !valida || c.ruta != httpinterno.RutaEventoPlazoLlamamiento || s.OrganizacionRef != organizacionAltaContratacionTemporalDesarrollo {
		return vacio, application.ErrEventoPlazoDenegado
	}
	expediente, err := e.lector.LeerExpedienteParaAvisoConfirmado(ctx, s.OrganizacionRef, s.ExpedienteRef, s.LlamamientoRef)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil || !expedienteEventoPlazoDesarrolloValido(expediente, s) {
		return vacio, application.ErrEventoPlazoNoDisponible
	}
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacionLlamamientoDesarrollo{expediente: expediente})
	ctx = context.WithValue(ctx, claveEventoPlazoDesarrollo{}, s)
	return e.servicio.Registrar(ctx, s)
}

func expedienteEventoPlazoDesarrolloValido(e ports.ExpedienteParaSeleccion, s ports.SolicitudRegistrarEventoPlazoLlamamiento) bool {
	return s.Validar() == nil && e.Fiscalizado.Validar() == nil &&
		expedienteComunicacionLlamamientoDesarrolloValido(e, ports.SolicitudRegistrarComunicacionLlamamiento{
			OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		})
}

// AutorizarEventoPlazo pide, también en un replay, una decisión nueva del
// permiso nominal de validación manual de respuesta y plazo.
func (p *proveedorEventoPlazoDesarrollo) AutorizarEventoPlazo(ctx context.Context, m postgresct.MaterialEventoPlazoLlamamiento) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || contextoInterfazNulo(ctx) || p.soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(p.autorizador) || dependenciaEsNulaContratacionTemporalDesarrollo(p.reloj) {
		return vacio, ports.ErrOperacionEventoPlazoDenegada
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	c, valida := p.soporte.capacidadValida(ctx)
	ligada, existe := ctx.Value(claveEventoPlazoDesarrollo{}).(ports.SolicitudRegistrarEventoPlazoLlamamiento)
	preparacion, preparada := ctx.Value(clavePreparacionLlamamientoDesarrollo{}).(preparacionLlamamientoDesarrollo)
	if !valida || c.ruta != httpinterno.RutaEventoPlazoLlamamiento || !existe || ligada != m.Solicitud ||
		!preparada || !expedienteEventoPlazoDesarrolloValido(preparacion.expediente, m.Solicitud) || m.Validar() != nil {
		return vacio, ports.ErrOperacionEventoPlazoDenegada
	}
	if _, _, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(p.reloj.Ahora()); !vigente {
		return vacio, ports.ErrOperacionEventoPlazoDenegada
	}
	recurso, err := postgresct.RecursoEventoPlazoLlamamiento(m)
	if err != nil {
		return vacio, ports.ErrOperacionEventoPlazoDenegada
	}
	ctx = context.WithValue(ctx, claveMaterialEventoPlazoDesarrollo{}, m)
	a, err := p.autorizador.AutorizarOperacion(ctx, postgresct.AccionResolucionManualLlamamiento, recurso)
	if ctx.Err() != nil {
		return vacio, ctx.Err()
	}
	if err != nil || a.ValidarEstructura() != nil {
		return vacio, ports.ErrOperacionEventoPlazoDenegada
	}
	r, ahora := a.ResumenCapacidad(), p.reloj.Ahora()
	if !domain.InstanteUTCCanonico(ahora) || ahora.Before(r.EmitidaEn()) || !ahora.Before(r.ExpiraEn()) ||
		r.ExpiraEn().Sub(r.EmitidaEn()) > 5*time.Minute {
		return vacio, ports.ErrOperacionEventoPlazoDenegada
	}
	return a, nil
}

type claveMaterialEventoPlazoDesarrollo struct{}

// solicitudAutorizacionEventoPlazoDesarrolloValida liga la decisión al
// material exacto que se va a consumir: mismo permiso que la resolución.
func solicitudAutorizacionEventoPlazoDesarrolloValida(ctx context.Context, datos dominiovec.DatosSolicitudAutorizacionLigadaV3, p preparacionLlamamientoDesarrollo) bool {
	m, ok := ctx.Value(claveMaterialEventoPlazoDesarrollo{}).(postgresct.MaterialEventoPlazoLlamamiento)
	if !ok || m.Validar() != nil || !expedienteEventoPlazoDesarrolloValido(p.expediente, m.Solicitud) ||
		datos.Accion != postgresct.AccionResolucionManualLlamamiento ||
		datos.ReferenciaMotivo != motivoResolucionManualDesarrollo(false) {
		return false
	}
	esperado, err := postgresct.RecursoEventoPlazoLlamamiento(m)
	r := datos.Recurso
	return err == nil && r.Referencia == esperado.Referencia && r.ModuloID == esperado.ModuloID && r.Tipo == esperado.Tipo &&
		maps.Equal(r.Ambitos, esperado.Ambitos) && maps.Equal(r.Atributos, esperado.Atributos)
}

// confirmarExpiracion registra la confirmación de RRHH a la propuesta de VEC
// (no aceptación y siguiente candidato) tras el vencimiento sin respuesta.
// CT111 exige el contacto efectivo, el vencimiento con su reloj y la ausencia
// de respuesta; deja pendiente la intención de siguiente candidato en la misma
// fila. No invoca Bolsa ni ejecuta el siguiente llamamiento.
func (e *ejecutorComunicacionLlamamientoDesarrollo) confirmarExpiracion(
	ctx context.Context, s ports.SolicitudResolverLlamamiento, expediente ports.ExpedienteParaSeleccion,
) (ports.ResultadoResolucionLlamamiento, error) {
	vacio := ports.ResultadoResolucionLlamamiento{}
	if s.Respuesta != ports.RespuestaLlamamientoExpirada || dependenciaEsNulaContratacionTemporalDesarrollo(e.servicio) {
		return vacio, application.ErrComunicacionLlamamientoDenegada
	}
	if !s.RevisionManualConfirmada() {
		return vacio, application.ErrValidacionRespuestaLlamamientoPendiente
	}
	politica, err := e.soporte.politicaResolucionDesarrollo(ctx, s)
	if err != nil {
		return vacio, application.ErrComunicacionLlamamientoDenegada
	}
	ligada := aceptacionRevisadaDesarrollo{solicitud: s, politica: politica}
	if !resolucionLigadaAlExpedienteDesarrollo(expediente, ligada) {
		return vacio, application.ErrComunicacionLlamamientoNoDisponible
	}
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, preparacionLlamamientoDesarrollo{expediente: expediente})
	ctx = context.WithValue(ctx, claveResolucionManualDesarrollo{}, ligada)
	local, err := e.servicio.Resolver(ctx, s)
	if err != nil {
		return vacio, err
	}
	if local.ValidarPara(s) != nil || local.Politica != politica {
		return vacio, application.ErrResultadoComunicacionLlamamientoNoConfiable
	}
	return local, nil
}
