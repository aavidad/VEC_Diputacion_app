package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/contactopropio"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/i18n"
	seguridaddoc "vec-diputacion-granada/internal/vec/adapters/documentos/seguridad"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var errContactoPropioDesarrolloNoDisponible = errors.New("bootstrap: contacto propio no disponible")

const clavePoliticaContactoPropioDesarrollo = "politica-contacto-propio-v3"

func rutaContactoPropioDesarrollo(ruta string) bool {
	return ruta == contactopropio.RutaContactoPropio || ruta == "/api/vec/usuarios/contacto-propio/recibo"
}

func descriptoresFronterasContactoPropioDesarrollo(perfil string) []descriptorFronteraComunDesarrollo {
	return []descriptorFronteraComunDesarrollo{
		{Clave: "contacto-propio-guardar", Superficie: superficieExternaPersonalSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: contactopropio.RutaContactoPropio, PerfilesActivosRef: []string{perfil}, ClavePolitica: clavePoliticaContactoPropioDesarrollo, ClaveCapacidad: "vec.contacto_usuario.guardar"},
		{Clave: "contacto-propio-version", Superficie: superficieExternaPersonalSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: contactopropio.RutaContactoPropio, PerfilesActivosRef: []string{perfil}, ClavePolitica: clavePoliticaContactoPropioDesarrollo, ClaveCapacidad: aplicacionvec.AccionVersionContactoPropia},
		{Clave: "contacto-propio-recibo", Superficie: superficieExternaPersonalSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: "/api/vec/usuarios/contacto-propio/recibo", PerfilesActivosRef: []string{perfil}, ClavePolitica: clavePoliticaContactoPropioDesarrollo, ClaveCapacidad: aplicacionvec.AccionConsultarContactoUsuario},
	}
}

func motivoContactoPropioDesarrollo(entrada string) dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_contacto_propio_desarrollo", CatalogoVersion: 1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-contacto-propio-v1"),
		EntradaClave:         referenciaAltaContratacionTemporalDesarrollo("motivo_", entrada),
	}
}

func descriptoresMaterialContactoPropioDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: "vec.contacto_usuario.registro.v1", Dominio: "vec.contacto.usuario.registro.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:contacto-registro:", ProveedorNominal: "proveedor-material-contacto-usuario"},
		{Audiencia: aplicacionvec.AudienciaConsultaReciboContactoUsuario, Dominio: "vec.contacto.usuario.recibo.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:contacto-recibo:", ProveedorNominal: "proveedor-material-contacto-recibo"},
		{Audiencia: aplicacionvec.AudienciaVersionContactoPropia, Dominio: "vec.contacto.usuario.version-propia.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:contacto-version-propia:", ProveedorNominal: "proveedor-material-contacto-version-propia"},
		{Audiencia: aplicacionvec.AudienciaVersionContactoLlamamiento, Dominio: "vec.contacto.usuario.version-llamamiento.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:contacto-version-llamamiento:", ProveedorNominal: "proveedor-material-contacto-version-llamamiento"},
	}
}

type sesionContactoPropioDesarrollo struct {
	delegado              *proveedorSesionConsultaRRHHDesarrollo
	personaRef, perfilRef string
}

func (s sesionContactoPropioDesarrollo) ResolverContactoPropio(ctx context.Context) (dominiovec.VinculoAutenticacionActorV2, dominiovec.ResultadoContextoActorRegistradoV2, error) {
	if s.delegado == nil || ctx == nil || ctx.Err() != nil {
		return dominiovec.VinculoAutenticacionActorV2{}, dominiovec.ResultadoContextoActorRegistradoV2{}, errContactoPropioDesarrolloNoDisponible
	}
	frontera, ok := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || !rutaContactoPropioDesarrollo(frontera.ruta) || frontera.superficie != superficieExternaPersonalSeguridadComunDesarrollo || !frontera.descriptor.admitePerfil(s.perfilRef) {
		return dominiovec.VinculoAutenticacionActorV2{}, dominiovec.ResultadoContextoActorRegistradoV2{}, errContactoPropioDesarrolloNoDisponible
	}
	c, err := s.delegado.ResolverContexto(ctx)
	if err != nil || c.Resultado.Contexto.PersonaRef != s.personaRef || c.Resultado.Contexto.PerfilActivoRef != s.perfilRef || c.Vinculo.ValidarPara(c.Resultado) != nil {
		return dominiovec.VinculoAutenticacionActorV2{}, dominiovec.ResultadoContextoActorRegistradoV2{}, errContactoPropioDesarrolloNoDisponible
	}
	return c.Vinculo, c.Resultado, nil
}

// Un fallo del consumidor opcional retira sus rutas y cierra su pool sin
// cancelar el servidor CT. La configuración parcial se valida antes en raíz.
func intentarRutasContactoPropioDesarrollo(componer func() ([]vechttp.RutaExacta, func(), error)) ([]vechttp.RutaExacta, func()) {
	cerrarVacio := func() {}
	if componer == nil {
		return nil, cerrarVacio
	}
	rutas, cerrar, err := componer()
	if err != nil || cerrar == nil {
		if cerrar != nil {
			cerrar()
		}
		slog.Warn("contacto propio opcional no compuesto", "causa", "dependencia_no_disponible")
		return nil, cerrarVacio
	}
	return rutas, cerrar
}

func nuevasRutasContactoPropioDesarrollo(ctx context.Context, cfg config.Config, identidad *identidadCandidatoBolsaDesarrollo, dependencias *DependenciasCT, alta *dependenciasAltaContratacionTemporalDesarrollo, identidadCT *proveedorSesionConsultaRRHHDesarrollo, fronteras catalogoFronterasComunDesarrollo) ([]vechttp.RutaExacta, func(), error) {
	fallo := func() ([]vechttp.RutaExacta, func(), error) { return nil, nil, errContactoPropioDesarrolloNoDisponible }
	if ctx == nil || identidad == nil || dependencias == nil || dependencias.kms == nil || alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil || alta.postgresql.registroAutorizacion == nil || alta.postgresql.proveedorMaterialContactoUsuario == nil || alta.postgresql.proveedorMaterialReciboContacto == nil || alta.postgresql.proveedorMaterialVersionContactoPropia == nil || identidadCT == nil || identidadCT.resolutor == nil || fronteras.identidad == nil {
		return fallo()
	}
	dsnWriter, dsnReader, dsnFuente, dsnMotivos, err := cfg.ContactoUsuarioPostgreSQL.DSNSeparados(cfg)
	if err != nil {
		return fallo()
	}
	entradas := []struct{ dsn, app, rol string }{
		{dsnWriter, "vec-contacto-writer", "vec_contacto_usuario_writer"},
		{dsnReader, "vec-contacto-reader", "vec_contacto_usuario_reader"},
		{dsnFuente, "vec-contacto-fuente", "vec_autorizacion_fuente"},
		{dsnMotivos, "vec-contacto-motivos", "vec_autorizacion_motivos_evaluador"},
	}
	pools := make([]func(), 0, len(entradas))
	var writer, fuente, motivosPool *pgxpool.Pool
	cerrar := func() {
		for i := len(pools) - 1; i >= 0; i-- {
			pools[i]()
		}
	}
	completo := false
	defer func() {
		if !completo {
			cerrar()
		}
	}()
	for i, e := range entradas {
		p, _, errorPool := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, e.dsn, e.app, e.rol)
		if errorPool != nil {
			return fallo()
		}
		pools = append(pools, p.Close)
		switch i {
		case 0:
			writer = p
		case 1:
			// Rol reservado para la lectura nominal B7; el pool se cierra con F2.
		case 2:
			fuente = p
		case 3:
			motivosPool = p
		}
	}
	fuenteAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(fuente)
	if err != nil {
		return fallo()
	}
	registroAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if err != nil {
		return fallo()
	}
	motivoRegistro := motivoContactoPropioDesarrollo("contacto-propio-registro")
	motivoRecibo := motivoContactoPropioDesarrollo("contacto-propio-recibo")
	motivoVersion := motivoContactoPropioDesarrollo("contacto-propio-version")
	if publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []dominiovec.ReferenciaEntradaCatalogo{motivoRegistro, motivoRecibo, motivoVersion}, dependencias.reloj.Ahora()) != nil {
		return fallo()
	}
	validador, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivosPool, motivoRegistro.CatalogoID)
	if err != nil {
		return fallo()
	}
	autorizador, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(fuenteAutorizacion, registroAutorizacion, registroAutorizacion, validador, dependencias.reloj, seguridadvec.GeneradorReferenciasCriptograficas{}, aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		return fallo()
	}
	emisorRegistro, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, alta.postgresql.proveedorMaterialContactoUsuario)
	if err != nil {
		return fallo()
	}
	emisorRecibo, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, alta.postgresql.proveedorMaterialReciboContacto)
	if err != nil {
		return fallo()
	}
	emisorVersion, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, alta.postgresql.proveedorMaterialVersionContactoPropia)
	if err != nil {
		return fallo()
	}
	vinculo, resultado, err := vincularCertificadoMiBolsaDesarrollo(ctx, identidadCT.resolutor, identidad, dependencias.reloj, dependencias.reloj.Ahora().UTC().Truncate(time.Microsecond), identidad.verificadoEn, identidad.validoHasta)
	if err != nil || resultado.Contexto.PersonaRef != identidad.personaRef {
		return fallo()
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{sello: dependencias.sello, principalID: identidad.identidad.principal.ID, certificadoSHA256: identidad.identidad.principal.Attributes["certificate_sha256"], candidatoBolsa: true, contexto: ctports.ContextoAutorizacionAltaV3{Vinculo: vinculo, Resultado: resultado}, contextoEsperadoRegistrado: resultado, reloj: dependencias.reloj}
	sesion, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(soporte, identidadCT.registro, identidadCT.revalidador, dependencias.reloj, identidadCT.resolutor, fronteras)
	if err != nil {
		return fallo()
	}
	clave := derivarClaveDesarrollo(dependencias.kms.claveEnvoltura, "vec.contacto.usuario.auditoria.desarrollo.v1")
	seudonimizador, err := seguridaddoc.NuevoSelladorHMAC("contacto_desarrollo_v1", clave[:])
	borrarBytes(clave[:])
	if err != nil {
		return fallo()
	}
	servicio, err := contactopropio.NuevoServicio(contactopropio.Dependencias{Sesion: sesionContactoPropioDesarrollo{delegado: sesion, personaRef: identidad.personaRef, perfilRef: identidad.perfilRef}, FuenteAutorizacion: fuenteAutorizacion, Emisor: emisorRegistro, Seudonimizador: seudonimizador, Protector: dependencias.kms, Huellas: huellasContactoDesarrollo{derivador: dependencias.derivador}, PoolEscritor: writer, Reloj: dependencias.reloj, GeneradorCorrelacion: seguridadvec.GeneradorReferenciasCriptograficas{}, PerfilPropioRef: identidad.perfilRef, Motivo: motivoRegistro, AmbitosRecurso: map[string]string{"persona_ref": identidad.personaRef}})
	if err != nil {
		return fallo()
	}
	conRecibos, err := contactopropio.NuevoServicioConRecibos(servicio, contactopropio.DependenciasConsultaRecibo{PoolConsulta: writer, Emisor: emisorRecibo, Motivo: motivoRecibo, PoolVersion: writer, EmisorVersion: emisorVersion, MotivoVersion: motivoVersion})
	if err != nil {
		return fallo()
	}
	catalogo, err := i18n.Load()
	if err != nil {
		return fallo()
	}
	rutas, err := contactopropio.NuevasRutasConRecibos(conRecibos, catalogo)
	if err != nil {
		return fallo()
	}
	completo = true
	var unaVez sync.Once
	return rutas, func() { unaVez.Do(cerrar) }, nil
}
