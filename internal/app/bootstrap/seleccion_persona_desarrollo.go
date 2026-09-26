package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	"vec-diputacion-granada/internal/modules/seleccion/adapters/httpcomun"
	seleccionpersonal "vec-diputacion-granada/internal/modules/seleccion/adapters/httppersonal"
	"vec-diputacion-granada/internal/modules/seleccion/adapters/nodisponible"
	seleccionpg "vec-diputacion-granada/internal/modules/seleccion/adapters/postgres"
	"vec-diputacion-granada/internal/modules/seleccion/adapters/referencias"
	seleccionapp "vec-diputacion-granada/internal/modules/seleccion/application"
	seleccionports "vec-diputacion-granada/internal/modules/seleccion/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// dependenciasSeleccionDesarrollo reúne lo que Selección necesita de la
// composición de desarrollo: repositorio (LOGIN de Bolsa), un proveedor de
// material por audiencia, el KMS para los datos personales y el reloj.
type dependenciasSeleccionDesarrollo struct {
	repositorio *seleccionpg.Repositorio
	proveedores map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo
	kms         *emisorKMSDesarrollo
	reloj       relojContratacionTemporalDesarrollo
}

func (d *dependenciasSeleccionDesarrollo) valida() bool {
	if d == nil || d.repositorio == nil || d.kms == nil || len(d.proveedores) != len(descriptoresMaterialSeleccionDesarrollo()) {
		return false
	}
	for _, p := range d.proveedores {
		if p == nil {
			return false
		}
	}
	return true
}

// rutaSeleccionPersonalDesarrollo: rutas de la persona, protegidas por la
// frontera mTLS de la superficie personal (como «Mi bolsa»).
func rutaSeleccionPersonalDesarrollo(ruta string) bool {
	return seleccionpersonal.EsRuta(ruta)
}

// operacionSeleccionPersonalDesarrollo dice si el método y la ruta forman una
// operación declarada de la superficie personal de Selección.
func operacionSeleccionPersonalDesarrollo(metodo, ruta string) bool {
	for _, o := range seleccionpersonal.Operaciones() {
		if o.Metodo == metodo && o.Ruta == ruta {
			return true
		}
	}
	return false
}

// accionesSeleccionPersonalEnRuta devuelve las acciones V3 admitidas en la
// ruta (cualquiera de sus métodos).
func accionesSeleccionPersonalEnRuta(ruta string) []string {
	var acciones []string
	for _, o := range seleccionpersonal.Operaciones() {
		if o.Ruta == ruta && o.Accion != "" {
			acciones = append(acciones, o.Accion)
		}
	}
	return acciones
}

// descriptoresFronterasSeleccionPersonalDesarrollo declara cada operación de
// la persona en la frontera común, con el perfil de la identidad de la persona
// y la política de su perfil («Mi bolsa»). Lo no declarado queda denegado.
func descriptoresFronterasSeleccionPersonalDesarrollo(perfilRef string) []descriptorFronteraComunDesarrollo {
	operaciones := seleccionpersonal.Operaciones()
	descriptores := make([]descriptorFronteraComunDesarrollo, 0, len(operaciones))
	nombres := map[string]string{
		seleccionpersonal.RutaMisSolicitudes: "lista", seleccionpersonal.RutaConvocatorias: "convocatorias",
		seleccionpersonal.RutaConvocatoria: "convocatoria", seleccionpersonal.RutaBorrador: "borrador", seleccionpersonal.RutaPresentacion: "presentacion",
	}
	for _, o := range operaciones {
		clave := "seleccion-propia-" + nombres[o.Ruta] + "-" + map[string]string{http.MethodGet: "leer", http.MethodPut: "guardar", http.MethodPost: "presentar"}[o.Metodo]
		descriptores = append(descriptores, descriptorFronteraComunDesarrollo{
			Clave: clave, Superficie: superficieExternaPersonalSeguridadComunDesarrollo, Metodo: o.Metodo, Ruta: o.Ruta,
			PerfilesActivosRef: []string{perfilRef}, ClavePolitica: "politica-bolsa-mi-bolsa", ClaveCapacidad: "capacidad-" + clave,
		})
	}
	return descriptores
}

// preparadorSeleccionPersonaDesarrollo es el adaptador de identidad de la
// persona: certificado mTLS de desarrollo ya verificado por la frontera y
// contexto registrado y revalidado en cada petición. Otro adaptador (Cl@ve,
// DNIe) produciría la misma Orden sin tocar el módulo. No exige que la
// persona tenga participación en ninguna bolsa.
type preparadorSeleccionPersonaDesarrollo struct {
	sello     *selloConsultasContratacionTemporalDesarrollo
	identidad *identidadCandidatoBolsaDesarrollo
	sesion    *proveedorSesionConsultaRRHHDesarrollo
	reloj     relojContratacionTemporalDesarrollo
}

func (p *preparadorSeleccionPersonaDesarrollo) PrepararOrdenPersona(r *http.Request) (seleccionapp.Orden, error) {
	if p == nil || r == nil || p.sello == nil || p.identidad == nil || p.sesion == nil {
		return seleccionapp.Orden{}, seleccionports.ErrNoDisponible
	}
	capacidad, ok := r.Context().Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !ok || capacidad.sello != p.sello || r.URL == nil || capacidad.ruta != r.URL.Path || !rutaSeleccionPersonalDesarrollo(capacidad.ruta) ||
		capacidad.principal.ID != p.identidad.identidad.principal.ID ||
		capacidad.principal.Attributes["certificate_sha256"] != p.identidad.identidad.principal.Attributes["certificate_sha256"] ||
		len(capacidad.principal.Roles) != 1 || capacidad.principal.Roles[0] != "candidato_bolsa" ||
		capacidad.certificadoVerificadoEn.IsZero() || capacidad.certificadoVerificadoEn.After(ahora) ||
		!ahora.Before(capacidad.certificadoValidoHasta) || !ahora.Before(p.identidad.validoHasta) {
		return seleccionapp.Orden{}, httpcomun.ErrAutenticacionAusente
	}
	registrado, err := p.sesion.ResolverContexto(r.Context())
	if err != nil || registrado.Vinculo.ValidarPara(registrado.Resultado) != nil ||
		registrado.Resultado.Contexto.PersonaRef != p.identidad.personaRef ||
		registrado.Resultado.Contexto.PerfilActivoRef != p.identidad.perfilRef {
		return seleccionapp.Orden{}, errors.Join(seleccionports.ErrNoDisponible, err)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(r.Context(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return seleccionapp.Orden{}, errors.Join(seleccionports.ErrNoDisponible, err)
	}
	// El perfil de la persona es el de «Mi bolsa»: su asignación acota con
	// candidato_ref, y el recurso propio lleva ese mismo ámbito.
	return seleccionapp.Orden{ResultadoContexto: registrado.Resultado, Vinculo: registrado.Vinculo, Motivo: motivoSeleccionPropiaDesarrollo(),
		Correlacion: correlacion, Ambitos: map[string]string{"candidato_ref": p.identidad.candidatoRef}}, nil
}

// autorizadorSeleccionPropiaDesarrollo liga cada decisión a la capacidad mTLS
// de la ruta: acción admitida en la ruta y recurso propio de la persona
// exacto. Después decide la política del perfil de la persona.
type autorizadorSeleccionPropiaDesarrollo struct {
	delegado  *aplicacionvec.ServicioAutorizacionSolicitudLigadaV3
	identidad *identidadCandidatoBolsaDesarrollo
	sello     *selloConsultasContratacionTemporalDesarrollo
}

func (a *autorizadorSeleccionPropiaDesarrollo) ExigirSolicitudLigadaV3(ctx context.Context, s dominiovec.SolicitudAutorizacionLigadaV3, r dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	var decision dominiovec.DecisionAutorizacionLigadaV3
	var confirmacion puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	if a == nil || a.delegado == nil || a.identidad == nil || ctx == nil {
		return decision, confirmacion, dominiovec.ErrAutorizacionDenegada
	}
	capacidad, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	datos, err := s.Datos()
	recurso := datos.Recurso
	if !ok || err != nil || a.sello == nil || capacidad.sello != a.sello || !rutaSeleccionPersonalDesarrollo(capacidad.ruta) ||
		capacidad.principal.ID != a.identidad.identidad.principal.ID ||
		capacidad.principal.Attributes["certificate_sha256"] != a.identidad.identidad.principal.Attributes["certificate_sha256"] ||
		len(capacidad.principal.Roles) != 1 || capacidad.principal.Roles[0] != "candidato_bolsa" ||
		!slices.Contains(accionesSeleccionPersonalEnRuta(capacidad.ruta), datos.Accion) ||
		recurso.Referencia != seleccionports.PrefijoRecursoPropio+a.identidad.personaRef || recurso.ModuloID != seleccionports.ModuloSeleccion ||
		recurso.Tipo != seleccionports.TipoRecursoSolicitudesPropias || len(recurso.Ambitos) != 1 || recurso.Ambitos["candidato_ref"] != a.identidad.candidatoRef ||
		r.Contexto.PersonaRef != a.identidad.personaRef {
		return decision, confirmacion, dominiovec.ErrAutorizacionDenegada
	}
	return a.delegado.ExigirSolicitudLigadaV3(ctx, s, r)
}

// emisorSeleccionDesarrollo emite el material de cada acción con el proveedor
// de su audiencia; una acción sin proveedor se deniega.
type emisorSeleccionDesarrollo struct {
	porAccion map[string]*emisorMaterialRenovableCTDesarrollo
}

var _ seleccionports.EmisorMaterialV3 = (*emisorSeleccionDesarrollo)(nil)

func (e *emisorSeleccionDesarrollo) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s dominiovec.SolicitudAutorizacionLigadaV3, r dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := s.Datos()
	if err != nil || e == nil || e.porAccion[datos.Accion] == nil {
		return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil,
			errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	return e.porAccion[datos.Accion].EmitirMaterialAutorizacionAtestadaV3(ctx, s, r)
}

// nuevoEmisorSeleccionDesarrollo crea un emisor por acción con el proveedor
// de su audiencia y la autoridad dada.
func nuevoEmisorSeleccionDesarrollo(autoridad puertosvec.AutorizadorSolicitudLigadaV3, proveedores map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo, acciones [][2]string) (*emisorSeleccionDesarrollo, error) {
	e := &emisorSeleccionDesarrollo{porAccion: make(map[string]*emisorMaterialRenovableCTDesarrollo, len(acciones))}
	for _, par := range acciones {
		emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(autoridad, proveedores[par[1]])
		if err != nil {
			return nil, err
		}
		e.porAccion[par[0]] = emisor
	}
	return e, nil
}

// serviciosExternosSeleccionDesarrollo: sin proveedor real de firma, registro
// en sede, tasas ni notificación; todos responden «no disponible».
func serviciosExternosSeleccionDesarrollo() seleccionapp.ServiciosExternos {
	return seleccionapp.ServiciosExternos{Firma: nodisponible.Servicios{}, Registro: nodisponible.Servicios{}, Tasas: nodisponible.Servicios{}, Notificacion: nodisponible.Servicios{}}
}

// rutasSeleccionPersonaDesarrollo compone las rutas de la persona sobre la
// autoridad de su perfil (la política de «Mi bolsa», que ya concede las
// acciones de AD3-89 con el selector encendido).
func rutasSeleccionPersonaDesarrollo(d *dependenciasSeleccionDesarrollo, delegado *aplicacionvec.ServicioAutorizacionSolicitudLigadaV3,
	identidad *identidadCandidatoBolsaDesarrollo, sello *selloConsultasContratacionTemporalDesarrollo,
	sesion *proveedorSesionConsultaRRHHDesarrollo, reloj relojContratacionTemporalDesarrollo) ([]vechttp.RutaExacta, error) {
	if !d.valida() || delegado == nil || identidad == nil || sello == nil || sesion == nil {
		return nil, errSeleccionNoDisponible
	}
	autoridad := &autorizadorSeleccionPropiaDesarrollo{delegado: delegado, identidad: identidad, sello: sello}
	emisor, err := nuevoEmisorSeleccionDesarrollo(autoridad, d.proveedores, seleccionports.AccionesPropias())
	if err != nil {
		return nil, errors.Join(errSeleccionNoDisponible, err)
	}
	servicio, err := seleccionapp.NuevoServicioSolicitudesPropias(d.repositorio, d.repositorio, emisor, d.kms, referencias.Aleatorio{}, serviciosExternosSeleccionDesarrollo(), reloj)
	if err != nil {
		return nil, errors.Join(errSeleccionNoDisponible, err)
	}
	handler, err := seleccionpersonal.Nuevo(&preparadorSeleccionPersonaDesarrollo{sello: sello, identidad: identidad, sesion: sesion, reloj: reloj}, servicio)
	if err != nil {
		return nil, errors.Join(errSeleccionNoDisponible, err)
	}
	rutas := make([]vechttp.RutaExacta, 0, len(seleccionpersonal.Rutas()))
	for _, ruta := range seleccionpersonal.Rutas() {
		rutas = append(rutas, vechttp.RutaExacta{Ruta: ruta, Manejador: handler})
	}
	return rutas, nil
}

// conConcesionesSeleccionPropiaDesarrollo añade al rol de la persona las
// acciones propias de Selección (sin campos ni obligaciones).
func conConcesionesSeleccionPropiaDesarrollo(i dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error) {
	copia := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(i)
	copia.VersionRol.Concesiones = append(copia.VersionRol.Concesiones, concesionesSeleccionPropiaDesarrollo()...)
	if err := copia.Validar(); err != nil {
		return dominiovec.InstantaneaAutorizacion{}, err
	}
	return copia, nil
}

// rutaPersonalCandidatoDesarrollo: rutas de la superficie personal que
// atiende la sesión de la persona (certificado de «Mi bolsa»).
func rutaPersonalCandidatoDesarrollo(ruta string) bool {
	return bolsapersonal.EsRutaPortal(ruta) || rutaSeleccionPersonalDesarrollo(ruta)
}
