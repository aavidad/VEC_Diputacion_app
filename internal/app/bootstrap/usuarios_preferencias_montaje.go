package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	usuariospg "vec-diputacion-granada/internal/modules/usuarios/adapters/postgres"
	usuariosapp "vec-diputacion-granada/internal/modules/usuarios/application"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var errComposicionUsuariosPreferencias = errors.New("bootstrap: preferencias de Usuarios no disponibles")

type configuracionUsuariosPreferenciasDesarrollo struct {
	Version                  int                                    `json:"version"`
	Autoridad                string                                 `json:"autoridad"`
	Superficie               core.SuperficieAutenticacionActorV1    `json:"superficie"`
	Cuentas                  []cuentaUsuariosPreferenciasDesarrollo `json:"cuentas"`
	DSNRegistroIdentidad     string                                 `json:"dsn_registro_identidad"`
	DSNRevalidacionIdentidad string                                 `json:"dsn_revalidacion_identidad"`
	DSNContexto              string                                 `json:"dsn_contexto"`
	DSNFuenteAutorizacion    string                                 `json:"dsn_fuente_autorizacion"`
	DSNRegistroAutorizacion  string                                 `json:"dsn_registro_autorizacion"`
	DSNMotivos               string                                 `json:"dsn_motivos"`
	DSNUsuarios              string                                 `json:"dsn_usuarios"`
	DSNUsuariosFrontera      string                                 `json:"dsn_usuarios_frontera"`
	MotivoConsulta           core.ReferenciaEntradaCatalogo         `json:"motivo_consulta"`
	MotivoActualizacion      core.ReferenciaEntradaCatalogo         `json:"motivo_actualizacion"`
}

func (configuracionUsuariosPreferenciasDesarrollo) String() string {
	return "[CONFIGURACION PRIVADA USUARIOS]"
}
func (configuracionUsuariosPreferenciasDesarrollo) GoString() string {
	return "[CONFIGURACION PRIVADA USUARIOS]"
}

type cuentaUsuariosPreferenciasDesarrollo struct {
	cuentaRutasDietasDesarrollo
}

type claveContextoPreferenciasUsuarios struct{}
type contextoPreferenciasUsuarios struct {
	autoridad *autoridadPreferenciasUsuariosDesarrollo
	vinculo   core.VinculoAutenticacionActorV2
	resultado core.ResultadoContextoActorRegistradoV2
}

type autoridadPreferenciasUsuariosDesarrollo struct {
	base        *autoridadRutasDietasDesarrollo
	cuentas     map[string]cuentaUsuariosPreferenciasDesarrollo
	reloj       relojRutasDietas
	manejador   http.Handler
	proveedor   *proveedorPreferenciasUsuarios
	registrador registradorDenegacionPreferenciasUsuarios
	cerrar      func()
	ruta        string
	superficie  core.SuperficieAutenticacionActorV1
	logins      map[string]bool
}

type composicionPreferenciasUsuarios struct {
	interna *autoridadPreferenciasUsuariosDesarrollo
	externa *autoridadPreferenciasUsuariosDesarrollo
}

func (c *composicionPreferenciasUsuarios) cerrar() {
	if c == nil {
		return
	}
	if c.interna != nil && c.interna.cerrar != nil {
		c.interna.cerrar()
	}
	if c.externa != nil && c.externa.cerrar != nil {
		c.externa.cerrar()
	}
}

func (c *composicionPreferenciasUsuarios) proteger(siguiente http.Handler) http.Handler {
	if c == nil {
		return siguiente
	}
	return c.interna.proteger(c.externa.proteger(siguiente))
}

func principalParaSuperficieUsuariosPreferenciasValido(identidad *resolvedorIdentidadDesarrollo, principal core.Principal, superficie string) bool {
	if identidad == nil || !principalSinteticoContratacionTemporalDesarrolloValido(principal) {
		return false
	}
	return superficie == "externa_personal" || superficie == "interna_corporativa"
}

// La autoridad exacta reconoce únicamente el contexto que esta frontera fijó
// para la misma petición y ruta. El PDP V3 decide la acción nominal después.
type autoridadExactasConUsuariosPreferencias struct {
	delegada vechttp.AutoridadRutasExactas
	usuarios *composicionPreferenciasUsuarios
}

func (a autoridadExactasConUsuariosPreferencias) AutorizarRutaExacta(ctx context.Context, ruta string) error {
	if ruta != usuarioshttp.RutaMisPreferencias && ruta != usuarioshttp.RutaMisPreferenciasAreaPersonal {
		if a.delegada == nil {
			return vechttp.ErrAutenticacionRutaExactaRequerida
		}
		return a.delegada.AutorizarRutaExacta(ctx, ruta)
	}
	if ctx == nil || a.usuarios == nil {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	seleccionada := a.usuarios.interna
	if ruta == usuarioshttp.RutaMisPreferenciasAreaPersonal {
		seleccionada = a.usuarios.externa
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	if !ok || seleccionada == nil || c.autoridad != seleccionada || c.resultado.Validar() != nil {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	if c.vinculo.ValidarPara(c.resultado) != nil || !c.vinculo.VigenteEn(seleccionada.reloj.Ahora(), c.resultado) {
		return vechttp.ErrAccesoRutaExactaDenegado
	}
	return nil
}

func (a *autoridadPreferenciasUsuariosDesarrollo) ResolverOrdenPreferencias(ctx context.Context) (usuariosports.OrdenPreferencias, error) {
	if a == nil || ctx == nil || a.proveedor == nil {
		return usuariosports.OrdenPreferencias{}, usuariosports.ErrNoDisponible
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	if !ok || c.autoridad != a || c.resultado.Validar() != nil || c.vinculo.ValidarPara(c.resultado) != nil || !c.vinculo.VigenteEn(a.reloj.Ahora(), c.resultado) {
		return usuariosports.OrdenPreferencias{}, usuariosports.ErrNoAutenticado
	}
	return usuariosports.NuevaOrdenPreferencias(c.resultado.Contexto, c.vinculo, a.superficie, a.proveedor)
}

func (a *autoridadPreferenciasUsuariosDesarrollo) proteger(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || r.URL.Path != a.ruta {
			siguiente.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "private, no-store, max-age=0")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		fallo := func(estado int, codigo string) {
			w.WriteHeader(estado)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.usuarios.preferencias.error." + codigo}})
		}
		if a == nil || a.base == nil || a.manejador == nil {
			fallo(503, "no_disponible")
			return
		}
		denegarTemprano := func() {
			if a.registrarDenegacion(r.Context(), http.StatusUnauthorized, "") != nil {
				fallo(503, "no_disponible")
				return
			}
			fallo(401, "no_autenticado")
		}
		if r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
			fallo(422, "peticion_invalida")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodPut {
			w.Header().Set("Allow", "GET, PUT")
			fallo(405, "metodo_no_permitido")
			return
		}
		if r.TLS == nil || len(r.TLS.VerifiedChains) != 1 || len(r.TLS.VerifiedChains[0]) == 0 || r.TLS.VerifiedChains[0][0] == nil {
			denegarTemprano()
			return
		}
		r = peticionIdentidadConsultasContratacionTemporalDesarrollo(r)
		if cabeceraLibreComisionesDietas(r.Header) {
			denegarTemprano()
			return
		}
		principal, err := a.base.resolvedor.ResolveDemoIdentity(r.Context(), r)
		cert := r.TLS.VerifiedChains[0][0]
		ahora := a.reloj.Ahora()
		if err != nil || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh || ahora.Before(cert.NotBefore) || !ahora.Before(cert.NotAfter) {
			denegarTemprano()
			return
		}
		cuenta, ok := a.cuentas[principal.Attributes["certificate_sha256"]]
		if !ok || cuenta.Sujeto != principal.ID {
			// Sin cuenta canónica todavía no existe una persona verificable para
			// la auditoría de una denegación 403. Mantener 401 con actor vacío.
			denegarTemprano()
			return
		}
		if !principalParaSuperficieUsuariosPreferenciasValido(a.base.resolvedor, principal, string(a.superficie)) {
			denegarTemprano()
			return
		}
		vinculo, resultado, err := a.resolverSesion(r, cuenta, ahora)
		if err != nil {
			fallo(503, "no_disponible")
			return
		}
		ctx := context.WithValue(r.Context(), claveContextoPreferenciasUsuarios{}, contextoPreferenciasUsuarios{autoridad: a, vinculo: vinculo, resultado: resultado})
		ctx, err = vechttp.ConActorVerificadoAuditoriaPreferenciasUsuarios(ctx, resultado.Contexto)
		if err != nil {
			fallo(503, "no_disponible")
			return
		}
		siguiente.ServeHTTP(w, r.WithContext(ctx))
	})
}

func nuevasRutasUsuariosPreferenciasDesarrollo(cfg config.Config, resolvedor vechttp.DemoIdentityResolver,
	derivador *derivadorIdentidadOperacionDesarrollo,
	consultaInterna, actualizacionInterna, consultaExterna, actualizacionExterna *proveedorMaterialAltaContratacionTemporalDesarrollo,
) (*composicionPreferenciasUsuarios, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil {
		return nil, err
	}
	if !activo {
		return nil, nil
	}
	interna, err := nuevaRutaUsuariosPreferenciasSuperficieDesarrollo(cfg, resolvedor, derivador, core.SuperficieAutenticacionInternaCorporativaV1, usuarioshttp.RutaMisPreferencias, consultaInterna, actualizacionInterna)
	if err != nil {
		return nil, err
	}
	externa, err := nuevaRutaUsuariosPreferenciasSuperficieDesarrollo(cfg, resolvedor, derivador, core.SuperficieAutenticacionExternaPersonalV1, usuarioshttp.RutaMisPreferenciasAreaPersonal, consultaExterna, actualizacionExterna)
	if err != nil {
		interna.cerrar()
		return nil, err
	}
	if !superficiesPreferenciasSeparadas(interna, externa) {
		interna.cerrar()
		externa.cerrar()
		return nil, errComposicionUsuariosPreferencias
	}
	return &composicionPreferenciasUsuarios{interna: interna, externa: externa}, nil
}

func superficiesPreferenciasSeparadas(interna, externa *autoridadPreferenciasUsuariosDesarrollo) bool {
	if interna == nil || externa == nil || interna.superficie != core.SuperficieAutenticacionInternaCorporativaV1 || externa.superficie != core.SuperficieAutenticacionExternaPersonalV1 || interna.ruta != usuarioshttp.RutaMisPreferencias || externa.ruta != usuarioshttp.RutaMisPreferenciasAreaPersonal {
		return false
	}
	for login := range interna.logins {
		if externa.logins[login] {
			return false
		}
	}
	cuentasInternas := map[string]bool{}
	perfilesInternos := map[string]bool{}
	for huella, cuenta := range interna.cuentas {
		cuentasInternas[cuenta.CuentaRef] = true
		perfilesInternos[cuenta.PerfilRef] = true
		if _, ok := externa.cuentas[huella]; ok {
			return false
		}
	}
	for _, cuenta := range externa.cuentas {
		if cuentasInternas[cuenta.CuentaRef] || perfilesInternos[cuenta.PerfilRef] {
			return false
		}
	}
	return true
}

func nombreConfiguracionPreferencias(superficie core.SuperficieAutenticacionActorV1) string {
	if superficie == core.SuperficieAutenticacionInternaCorporativaV1 {
		return "usuarios-preferencias-interna.json"
	}
	if superficie == core.SuperficieAutenticacionExternaPersonalV1 {
		return "usuarios-preferencias-externa.json"
	}
	return ""
}

func nuevaRutaUsuariosPreferenciasSuperficieDesarrollo(cfg config.Config, resolvedor vechttp.DemoIdentityResolver,
	derivador *derivadorIdentidadOperacionDesarrollo, superficie core.SuperficieAutenticacionActorV1, ruta string,
	consulta, actualizacion *proveedorMaterialAltaContratacionTemporalDesarrollo,
) (*autoridadPreferenciasUsuariosDesarrollo, error) {
	identidad, ok := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !ok || identidad == nil || derivador == nil || !derivador.valido() || consulta == nil || actualizacion == nil {
		return nil, errComposicionUsuariosPreferencias
	}
	c, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, superficie)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	cuentas := map[string]cuentaUsuariosPreferenciasDesarrollo{}
	for _, cuenta := range c.Cuentas {
		huella, e := hex.DecodeString(cuenta.CertificadoSHA256)
		if e != nil || len(huella) != sha256.Size || hex.EncodeToString(huella) != cuenta.CertificadoSHA256 || cuenta.Sujeto == "" || cuenta.CuentaRef == "" || cuenta.PerfilRef == "" || cuentas[cuenta.CertificadoSHA256].Sujeto != "" {
			return nil, errComposicionUsuariosPreferencias
		}
		var digest [32]byte
		copy(digest[:], huella)
		principal, existe := identidad.porHuella[digest]
		if !existe || principal.ID != cuenta.Sujeto || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh {
			return nil, errComposicionUsuariosPreferencias
		}
		if !principalParaSuperficieUsuariosPreferenciasValido(identidad, principal, string(superficie)) {
			return nil, errComposicionUsuariosPreferencias
		}
		cuentas[cuenta.CertificadoSHA256] = cuenta
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	entradas := []struct{ dsn, rol string }{
		{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"}, {c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"},
		{c.DSNContexto, "vec_contexto_actor_v1_runtime"}, {c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
		{c.DSNRegistroAutorizacion, "vec_autorizacion_registro"}, {c.DSNMotivos, "vec_autorizacion_motivos_evaluador"},
	}
	var pools []*pgxpool.Pool
	var unaVez sync.Once
	cerrar := func() {
		unaVez.Do(func() {
			for _, p := range pools {
				p.Close()
			}
		})
	}
	completa := false
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	logins := map[string]bool{}
	for _, entrada := range entradas {
		pool, login, e := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
		if e != nil || logins[login] {
			if pool != nil {
				pool.Close()
			}
			return nil, errComposicionUsuariosPreferencias
		}
		pools = append(pools, pool)
		logins[login] = true
	}
	ejecutor, login, err := abrirPoolUsuariosPreferencias(ctx, c.DSNUsuarios, rolEjecutorPreferencias(string(superficie)))
	if err != nil || logins[login] {
		if ejecutor != nil {
			ejecutor.Close()
		}
		return nil, errComposicionUsuariosPreferencias
	}
	pools = append(pools, ejecutor)
	logins[login] = true
	registradorPool, login, err := abrirPoolUsuariosPreferencias(ctx, c.DSNUsuariosFrontera, rolRegistradorPreferencias(string(superficie)))
	if err != nil || logins[login] {
		if registradorPool != nil {
			registradorPool.Close()
		}
		return nil, errComposicionUsuariosPreferencias
	}
	pools = append(pools, registradorPool)
	repositorio, err := usuariospg.NuevoRegistroPreferenciasPostgreSQL(ctx, ejecutor, superficie)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	registrador, err := usuariospg.NuevoRegistradorDenegacionPreferenciasPostgreSQL(ctx, registradorPool, superficie)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	registro, err := identidadpg.NuevoRegistroSesionesPostgreSQL(ctx, pools[0], pools[1], &seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	revalidador, err := identidadpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pools[1])
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pools[2])
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	reloj := relojRutasDietas{}
	servicioContexto, err := vecapp.NuevoServicioContextoActorProductivoV2(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	contextos, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	fuente, err := vecpg.NuevoAlmacenAutorizacion(pools[3])
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	registroAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(pools[4])
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools[5], c.MotivoConsulta.CatalogoID)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	autorizador, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registroAutorizacion, registroAutorizacion, motivos, reloj, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	emisorConsulta, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, consulta)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	emisorActualizacion, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, actualizacion)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	nonce, err := nonceRutasDietas()
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, cuentas: map[string]cuentaRutasDietasDesarrollo{}, registro: registro, revalidador: revalidador, contextos: contextos, reloj: reloj, instancia: nonce}
	a := &autoridadPreferenciasUsuariosDesarrollo{base: base, cuentas: cuentas, reloj: reloj, cerrar: cerrar, registrador: registrador, ruta: ruta, superficie: superficie, logins: logins}
	a.proveedor = &proveedorPreferenciasUsuarios{autoridad: a, consulta: emisorConsulta, actualizacion: emisorActualizacion, motivoConsulta: c.MotivoConsulta, motivoActualizacion: c.MotivoActualizacion}
	servicio, err := usuariosapp.NuevoServicioPreferencias(repositorio, time.Now)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	a.manejador, err = usuarioshttp.NuevoManejadorPreferenciasEnRuta(servicio, a, a, ruta)
	if err != nil {
		return nil, errComposicionUsuariosPreferencias
	}
	completa = true
	return a, nil
}

func rutaUsuariosPreferencias(a *composicionPreferenciasUsuarios) []vechttp.RutaExacta {
	if a == nil || a.interna == nil || a.externa == nil || a.interna.manejador == nil || a.externa.manejador == nil {
		return nil
	}
	return []vechttp.RutaExacta{{Ruta: usuarioshttp.RutaMisPreferencias, Manejador: a.interna.manejador}, {Ruta: usuarioshttp.RutaMisPreferenciasAreaPersonal, Manejador: a.externa.manejador}}
}

var _ vecports.Reloj = relojRutasDietas{}
