package bootstrap

import (
	"context"
	"crypto/rand"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/config"
	aspirantescatalogo "vec-diputacion-granada/internal/modules/aspirantes/adapters/catalogo"
	aspirantescertificado "vec-diputacion-granada/internal/modules/aspirantes/adapters/certificado"
	aspiranteshttp "vec-diputacion-granada/internal/modules/aspirantes/adapters/httpapi"
	aspirantespg "vec-diputacion-granada/internal/modules/aspirantes/adapters/postgres"
	aspirantesseguridad "vec-diputacion-granada/internal/modules/aspirantes/adapters/seguridad"
	aspirantesapp "vec-diputacion-granada/internal/modules/aspirantes/application"
	aspirantescanonico "vec-diputacion-granada/internal/modules/aspirantes/canonico"
	aspirantesports "vec-diputacion-granada/internal/modules/aspirantes/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/datospersonales"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// dependenciasAspirantesDesarrollo agrupa lo que no depende de la petición:
// material V3 publicado, configuración privada, claves, catálogo y lector.
type dependenciasAspirantesDesarrollo struct {
	materiales proveedoresMaterialAspirantes
	config     configuracionAspirantesDesarrollo
	cripto     *aspirantesseguridad.Adaptador
	catalogo   *aspirantescatalogo.Adaptador
	lector     *aspirantescertificado.Lector
}

// fuenteClavesAspirantesDesarrollo deriva del KMS de desarrollo cuatro
// claves independientes del portal externo. En producción la fuente será el
// gestor de claves del portal externo (F1.1), con rotación y retención.
type fuenteClavesAspirantesDesarrollo struct {
	claves aspirantesseguridad.ClavesAspirantes
}

func (f fuenteClavesAspirantesDesarrollo) CargarClavesAspirantes(context.Context) (aspirantesseguridad.ClavesAspirantes, error) {
	return f.claves, nil
}

func nuevaFuenteClavesAspirantesDesarrollo(kms *emisorKMSDesarrollo) (fuenteClavesAspirantesDesarrollo, error) {
	if kms == nil || claveContactoKMSCero(kms.claveEnvoltura) {
		return fuenteClavesAspirantesDesarrollo{}, errComposicionAspirantes
	}
	clave := func(ambito string) aspirantesseguridad.Clave {
		return aspirantesseguridad.Clave{Ref: "clave:kms:desarrollo:portal-externo:aspirantes-" + ambito + ":v1",
			Material: derivarClaveDesarrollo(kms.claveEnvoltura, "vec.kms.desarrollo.portal-externo.aspirantes."+ambito+".v1")}
	}
	return fuenteClavesAspirantesDesarrollo{claves: aspirantesseguridad.ClavesAspirantes{
		CifradoActivo: clave("contacto"), DocumentoActivo: clave("documento"),
		Indice: clave("indice"), SemanticaActiva: clave("huella"),
	}}, nil
}

// nuevasDependenciasAspirantesDesarrollo devuelve nil si Aspirantes no se ha
// pedido. Pedirlo sin preferencias, material, configuración o catálogo es
// un error: nunca arranca a medias.
func nuevasDependenciasAspirantesDesarrollo(cfg config.Config, kms *emisorKMSDesarrollo, materiales proveedoresMaterialAspirantes) (*dependenciasAspirantesDesarrollo, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envAspirantesDesarrollo)
	if err != nil || !activo {
		return nil, err
	}
	preferencias, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil || !preferencias {
		return nil, errComposicionAspirantes
	}
	for _, p := range materiales {
		if p == nil {
			return nil, errComposicionAspirantes
		}
	}
	c, err := leerConfiguracionAspirantesDesarrollo(cfg)
	if err != nil {
		return nil, err
	}
	perfiles, err := perfilesCertificadoAspirantes(cfg, c)
	if err != nil {
		return nil, err
	}
	lector, err := aspirantescertificado.NuevoLector(perfiles)
	if err != nil {
		return nil, errComposicionAspirantes
	}
	fuente, err := nuevaFuenteClavesAspirantesDesarrollo(kms)
	if err != nil {
		return nil, err
	}
	cripto, err := aspirantesseguridad.NuevoAdaptador(fuente)
	if err != nil {
		return nil, errComposicionAspirantes
	}
	consulta, err := fichero.NuevaConsultaCatalogos(rutaCatalogoDatosPersonalesAspirantes())
	if err != nil {
		return nil, errComposicionAspirantes
	}
	resolutor, err := datospersonales.NuevoResolutor(consulta, consulta, relojRutasDietas{})
	if err != nil {
		return nil, errComposicionAspirantes
	}
	catalogo, err := aspirantescatalogo.Nuevo(resolutor, c.TiposConvocatoria)
	if err != nil {
		return nil, errComposicionAspirantes
	}
	return &dependenciasAspirantesDesarrollo{materiales: materiales, config: c, cripto: cripto, catalogo: catalogo, lector: lector}, nil
}

// componenteAspirantes es lo propio de la autoridad hermana de Aspirantes.
type componenteAspirantes struct {
	proveedor *proveedorAspirantes
	lector    *aspirantescertificado.Lector
}

// montarAspirantesSuperficie crea la autoridad hermana de preferencias
// externas para la ruta de la ficha propia: misma frontera de identidad y
// PDP; su propio pool, registrador de frontera, proveedor V3 y manejador.
func montarAspirantesSuperficie(ctx context.Context, externa *autoridadPreferenciasUsuariosDesarrollo,
	autorizador vecports.AutorizadorSolicitudLigadaV3, c configuracionUsuariosPreferenciasDesarrollo, d *dependenciasAspirantesDesarrollo,
) (*autoridadPreferenciasUsuariosDesarrollo, error) {
	if ctx == nil || externa == nil || externa.superficie != core.SuperficieAutenticacionExternaPersonalV1 || autorizador == nil || d == nil {
		return nil, errComposicionAspirantes
	}
	pool, err := abrirPoolAspirantes(ctx, d.config.DSNAspirantes)
	if err != nil {
		return nil, err
	}
	completa := false
	defer func() {
		if !completa {
			pool.Close()
		}
	}()
	registro, err := aspirantespg.NuevoRegistroFichasPostgreSQL(ctx, pool)
	if err != nil {
		return nil, errComposicionAspirantes
	}
	emisores := make(map[string]emisorPreferenciasUsuarios, len(d.materiales))
	for i, material := range d.materiales {
		emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, material)
		if err != nil {
			return nil, errComposicionAspirantes
		}
		emisores[accionesAspirantes[i].accion] = emisor
	}
	servicio, err := aspirantesapp.NuevoServicioFichaPropia(aspirantesapp.Dependencias{Registro: registro, Protector: d.cripto,
		Sellador: d.cripto, Catalogo: d.catalogo, Azar: rand.Reader, AhoraUTC: time.Now})
	if err != nil {
		return nil, errComposicionAspirantes
	}
	a := *externa
	a.ruta, a.cerrar, a.manejador, a.proveedor = aspiranteshttp.RutaMiFicha, pool.Close, nil, nil
	a.proveedorCorreos, a.correos, a.proveedorImagen, a.imagen, a.aspirantes = nil, nil, nil, nil, nil
	a.prefijoError, a.metodoEscritura = "api.aspirantes.ficha.error.", "POST"
	a.registrador, a.superficieAuditoria = registro, vecports.SuperficieAuditoriaFronteraRutaExactaAspirantes
	a.componenteAspirantes = &componenteAspirantes{lector: d.lector,
		proveedor: &proveedorAspirantes{autoridad: &a, emisores: emisores, motivoConsulta: c.MotivoConsulta, motivoActualizacion: c.MotivoActualizacion}}
	manejador, err := aspiranteshttp.NuevoManejador(servicio, &a, auditorAspirantes{autoridad: &a})
	if err != nil {
		return nil, errComposicionAspirantes
	}
	a.manejador = manejador
	completa = true
	return &a, nil
}

// ResolverOrdenFicha solo acepta el contexto que esta misma autoridad fijó en
// la frontera. La identidad sale de la hoja verificada de la misma conexión
// TLS con la que se estableció la sesión, nunca del cuerpo ni de cabeceras.
func (a *autoridadPreferenciasUsuariosDesarrollo) ResolverOrdenFicha(ctx context.Context) (aspirantesports.OrdenFicha, error) {
	if a == nil || ctx == nil || a.componenteAspirantes == nil || a.componenteAspirantes.lector == nil {
		return aspirantesports.OrdenFicha{}, aspirantesports.ErrNoDisponible
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	if !ok || c.autoridad != a || c.certificado == nil || c.resultado.Validar() != nil || c.vinculo.ValidarPara(c.resultado) != nil || !c.vinculo.VigenteEn(a.reloj.Ahora(), c.resultado) {
		return aspirantesports.OrdenFicha{}, aspirantesports.ErrNoAutenticado
	}
	identidad, err := a.componenteAspirantes.lector.Identidad(c.certificado)
	if err != nil {
		// El certificado es válido para entrar, pero no acredita a una
		// persona física con DNI o NIE: no hay ficha que mostrar.
		return aspirantesports.OrdenFicha{}, aspirantesports.ErrProhibido
	}
	return aspirantesapp.NuevaOrdenFicha(c.resultado.Contexto, c.vinculo, a.superficie, identidad, a.componenteAspirantes.proveedor)
}

// auditorAspirantes adapta el registro de denegaciones de la frontera al
// contrato de la API de Aspirantes (sin persona).
type auditorAspirantes struct {
	autoridad *autoridadPreferenciasUsuariosDesarrollo
}

func (a auditorAspirantes) AuditarDenegacion(ctx context.Context, estado int) error {
	return a.autoridad.AuditarDenegacionPreferencias(ctx, estado)
}

// proveedorAspirantes emite la V3 nominal de cada acción de la ficha propia
// con el recurso canónico que PostgreSQL vuelve a calcular. Solo sirve a la
// persona de la sesión de esta misma frontera.
type proveedorAspirantes struct {
	autoridad           *autoridadPreferenciasUsuariosDesarrollo
	emisores            map[string]emisorPreferenciasUsuarios
	motivoConsulta      core.ReferenciaEntradaCatalogo
	motivoActualizacion core.ReferenciaEntradaCatalogo
}

func (p *proveedorAspirantes) ProveerMaterialFicha(ctx context.Context, vinculo core.VinculoAutenticacionActorV2, m aspirantesports.MaterialFicha) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || p.autoridad == nil || ctx == nil || ctx.Err() != nil {
		return vacia, aspirantesports.ErrNoDisponible
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	datosEntrada, errEntrada := vinculo.Datos()
	datosContexto, errContexto := c.vinculo.Datos()
	if !ok || c.autoridad != p.autoridad || c.resultado.Validar() != nil || c.vinculo.ValidarPara(c.resultado) != nil ||
		!c.vinculo.VigenteEn(p.autoridad.reloj.Ahora(), c.resultado) || c.resultado.Contexto.PersonaRef != m.PersonaRef ||
		c.resultado.Contexto.PerfilActivoRef != m.PerfilRef || m.FinalidadRef != aspirantesports.FinalidadFicha ||
		m.Superficie != p.autoridad.superficie || errEntrada != nil || errContexto != nil || !reflect.DeepEqual(datosEntrada, datosContexto) {
		return vacia, aspirantesports.ErrProhibido
	}
	emisor := p.emisores[m.Accion]
	motivo := p.motivoActualizacion
	if m.Accion == aspirantesports.AccionConsultar {
		motivo = p.motivoConsulta
	}
	audiencia, err := aspirantesports.Audiencia(m.Accion)
	if err != nil {
		return vacia, aspirantesports.ErrProhibido
	}
	if emisor == nil || !core.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return vacia, aspirantesports.ErrNoDisponible
	}
	recurso, err := aspirantescanonico.Recurso(m)
	if err != nil {
		return vacia, aspirantesports.ErrInvalida
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, aspirantesports.ErrNoDisponible
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: c.vinculo, ReferenciaMotivo: motivo, Accion: m.Accion,
		Recurso: recurso, Finalidad: m.FinalidadRef, Correlacion: correlacion,
	})
	if err != nil {
		return vacia, aspirantesports.ErrProhibido
	}
	decision, confirmacion, exportador, err := emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, c.resultado)
	if errors.Is(err, core.ErrAutorizacionDenegada) {
		return vacia, aspirantesports.ErrProhibido
	}
	if err != nil {
		return vacia, aspirantesports.ErrNoDisponible
	}
	if decision.ValidarPara(solicitud) != nil || exportador == nil {
		return vacia, aspirantesports.ErrProhibido
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, c.resultado, motivo, material, audiencia) {
		return vacia, aspirantesports.ErrProhibido
	}
	return material, nil
}
