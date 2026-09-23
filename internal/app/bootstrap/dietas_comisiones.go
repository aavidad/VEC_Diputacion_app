package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	dietas "vec-diputacion-granada/internal/modules/dietas"
	dietashttp "vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
	dietaspg "vec-diputacion-granada/internal/modules/dietas/adapters/postgres"
	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El archivo privado sólo selecciona cuentas, conexiones y motivos. Las claves
// V3 no viven aquí: Dietas firma con el gobierno único de desarrollo que
// publica Contratación (ver dietas_material_ct_desarrollo.go). Su ausencia con
// el selector activado impide montar la ruta; no genera claves.
type configuracionComisionesDietasDesarrollo struct {
	redaccionMaterialRutasDietas
	Version                  int                            `json:"version"`
	Autoridad                string                         `json:"autoridad"`
	Cuentas                  []cuentaRutasDietasDesarrollo  `json:"cuentas"`
	DSNRegistroIdentidad     string                         `json:"dsn_registro_identidad"`
	DSNRevalidacionIdentidad string                         `json:"dsn_revalidacion_identidad"`
	DSNContexto              string                         `json:"dsn_contexto"`
	DSNFuenteAutorizacion    string                         `json:"dsn_fuente_autorizacion"`
	DSNRegistroAutorizacion  string                         `json:"dsn_registro_autorizacion"`
	DSNMotivos               string                         `json:"dsn_motivos"`
	MotivoPersonal           core.ReferenciaEntradaCatalogo `json:"motivo_personal"`
	MotivoCrear              core.ReferenciaEntradaCatalogo `json:"motivo_crear"`
	MotivoConsultar          core.ReferenciaEntradaCatalogo `json:"motivo_consultar"`
}

type claveContextoComisionesDietas struct{}
type contextoComisionesDietas struct {
	autoridad    *autoridadComisionesDietasDesarrollo
	ruta, metodo string
	seguridad    contextoSeguridadComunDesarrollo
}

type seguridadComisionesDietasDesarrollo struct {
	autoridad *autoridadComisionesDietasDesarrollo
}

func (s seguridadComisionesDietasDesarrollo) ResolverContexto(ctx context.Context) (contextoSeguridadComunDesarrollo, error) {
	if ctx == nil || s.autoridad == nil {
		return contextoSeguridadComunDesarrollo{}, ErrIdentidadPersonalDietasNoDisponible
	}
	c, ok := ctx.Value(claveContextoComisionesDietas{}).(contextoComisionesDietas)
	if !ok || c.autoridad != s.autoridad || c.seguridad.Resultado.Validar() != nil || c.seguridad.Vinculo.ValidarPara(c.seguridad.Resultado) != nil || !c.seguridad.Vinculo.VigenteEn(s.autoridad.reloj.Ahora(), c.seguridad.Resultado) {
		return contextoSeguridadComunDesarrollo{}, ErrIdentidadPersonalDietasNoDisponible
	}
	clon, err := c.seguridad.Resultado.Clonar()
	if err != nil {
		return contextoSeguridadComunDesarrollo{}, ErrIdentidadPersonalDietasNoDisponible
	}
	return contextoSeguridadComunDesarrollo{Vinculo: c.seguridad.Vinculo, Resultado: clon}, nil
}

// emisorMaterialDietasDesarrollo es lo único que Dietas necesita de un emisor
// V3: el renovable de Contratación sigue la configuración vigente cada día.
type emisorMaterialDietasDesarrollo interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type emisorComisionesDietasDesarrollo struct {
	personal, crear, consultar emisorMaterialDietasDesarrollo
}

func (e *emisorComisionesDietasDesarrollo) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, solicitud core.SolicitudAutorizacionLigadaV3, resultado core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	var elegido emisorMaterialDietasDesarrollo
	datos, err := solicitud.Datos()
	if err == nil && e != nil {
		switch datos.Accion {
		case "personal.relacion.propia.consultar_dietas":
			elegido = e.personal
		case "dietas.borrador.propio.crear":
			elegido = e.crear
		case "dietas.borrador.propio.consultar":
			elegido = e.consultar
		}
	}
	if elegido == nil {
		return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	return elegido.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
}

type autoridadComisionesDietasDesarrollo struct {
	base        *autoridadRutasDietasDesarrollo
	reloj       vecports.Reloj
	cuentas     map[string]cuentaRutasDietasDesarrollo
	rutas       []vechttp.RutaExacta
	colecciones []vechttp.RutaColeccion
	cerrar      func()
}

func esRutaComisionesDietas(ruta string) bool {
	if ruta == dietashttp.RutaBorradores {
		return true
	}
	resto, ok := strings.CutPrefix(ruta, dietashttp.RutaBorradores+"/")
	return ok && resto != "" && !strings.Contains(resto, "/")
}

func metodoComisionesDietasValido(ruta, metodo string) bool {
	if ruta == dietashttp.RutaBorradores {
		return metodo == http.MethodGet || metodo == http.MethodPost
	}
	return esRutaComisionesDietas(ruta) && metodo == http.MethodGet
}

func (a *autoridadComisionesDietasDesarrollo) proteger(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			responderDenegacionComisionesDietas(w, http.StatusServiceUnavailable)
			return
		}
		if !esRutaComisionesDietas(r.URL.Path) {
			siguiente.ServeHTTP(w, r)
			return
		}
		if a == nil || a.base == nil || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.VerifiedChains[0]) == 0 || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			responderDenegacionComisionesDietas(w, http.StatusUnauthorized)
			return
		}
		principal, err := a.base.resolvedor.ResolveDemoIdentity(r.Context(), r)
		ahora := a.reloj.Ahora()
		cert := r.TLS.VerifiedChains[0][0]
		if err != nil || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh || ahora.Before(cert.NotBefore) || !ahora.Before(cert.NotAfter) {
			responderDenegacionComisionesDietas(w, http.StatusUnauthorized)
			return
		}
		cuenta, ok := a.cuentas[principal.Attributes["certificate_sha256"]]
		if !ok || cuenta.Sujeto != principal.ID {
			responderDenegacionComisionesDietas(w, http.StatusForbidden)
			return
		}
		vinculo, resultado, err := a.base.resolverSesion(r.Context(), r, &capsulaRutasDietasDesarrollo{autoridad: a.base, peticion: r, cuenta: cuenta, instante: ahora})
		if err != nil {
			responderDenegacionComisionesDietas(w, http.StatusServiceUnavailable)
			return
		}
		ctx := context.WithValue(r.Context(), claveContextoComisionesDietas{}, contextoComisionesDietas{autoridad: a, ruta: r.URL.Path, metodo: r.Method, seguridad: contextoSeguridadComunDesarrollo{Vinculo: vinculo, Resultado: resultado}})
		siguiente.ServeHTTP(w, r.WithContext(ctx))
	})
}

func responderDenegacionComisionesDietas(w http.ResponseWriter, estado int) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
}

type autoridadExactasConDietas struct {
	delegada vechttp.AutoridadRutasExactas
	dietas   *autoridadComisionesDietasDesarrollo
}

func (a autoridadExactasConDietas) AutorizarRutaExacta(ctx context.Context, ruta string) error {
	if !esRutaComisionesDietas(ruta) {
		return a.delegada.AutorizarRutaExacta(ctx, ruta)
	}
	if ctx == nil {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	c, ok := ctx.Value(claveContextoComisionesDietas{}).(contextoComisionesDietas)
	if !ok {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	if a.dietas == nil || c.autoridad != a.dietas || c.ruta != ruta || !metodoComisionesDietasValido(ruta, c.metodo) || c.seguridad.Resultado.Validar() != nil || !c.seguridad.Vinculo.VigenteEn(a.dietas.reloj.Ahora(), c.seguridad.Resultado) {
		return vechttp.ErrAccesoRutaExactaDenegado
	}
	return nil
}

func nuevasComisionesDietasDesarrollo(cfg config.Config, resolvedor vechttp.DemoIdentityResolver, derivador *derivadorIdentidadOperacionDesarrollo, material materialDietasDesdeCTDesarrollo) (*autoridadComisionesDietasDesarrollo, error) {
	activo, err := cfg.DietasBorradoresDesarrolloActivos()
	if err != nil {
		return nil, err
	}
	if !activo {
		return nil, nil
	}
	identidad, ok := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !ok || identidad == nil || derivador == nil || !derivador.valido() || !material.completo() {
		return nil, errComposicionDietasEn()
	}
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "dietas-comisiones.json")
	contenido, err := leerFicheroMaterialSeguro(ruta, 256<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return nil, errComposicionDietasEn()
	}
	defer borrarBytes(contenido)
	var c configuracionComisionesDietasDesarrollo
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa || len(c.Cuentas) == 0 || len(c.Cuentas) > 64 || c.MotivoPersonal.Validar() != nil || c.MotivoCrear.Validar() != nil || c.MotivoConsultar != c.MotivoCrear || c.MotivoPersonal.CatalogoID != c.MotivoCrear.CatalogoID {
		return nil, errComposicionDietasEn()
	}
	cuentas := map[string]cuentaRutasDietasDesarrollo{}
	for _, cuenta := range c.Cuentas {
		b, e := hex.DecodeString(cuenta.CertificadoSHA256)
		if e != nil || len(b) != sha256.Size || hex.EncodeToString(b) != cuenta.CertificadoSHA256 || cuenta.Sujeto == "" || cuenta.CuentaRef == "" || cuenta.PerfilRef == "" {
			return nil, errComposicionDietasEn()
		}
		var digest [32]byte
		copy(digest[:], b)
		principal, existe := identidad.porHuella[digest]
		if !existe || principal.ID != cuenta.Sujeto || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh || cuentas[cuenta.CertificadoSHA256].Sujeto != "" {
			return nil, errComposicionDietasEn()
		}
		cuentas[cuenta.CertificadoSHA256] = cuenta
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	entradas := []struct{ dsn, rol string }{{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"}, {c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"}, {c.DSNContexto, "vec_contexto_actor_v1_runtime"}, {c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"}, {c.DSNRegistroAutorizacion, "vec_autorizacion_registro"}, {c.DSNMotivos, "vec_autorizacion_motivos_evaluador"}}
	var pools []*pgxpool.Pool
	var propios *poolsPostgreSQLDietasDesarrollo
	completa := false
	cerrar := func() {
		if propios != nil {
			propios.Cerrar()
		}
		for _, p := range pools {
			p.Close()
		}
	}
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	usuarios := map[string]bool{}
	for _, entrada := range entradas {
		pool, usuario, e := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
		if e != nil || usuarios[usuario] {
			if pool != nil {
				pool.Close()
			}
			return nil, errComposicionDietasEn()
		}
		pools = append(pools, pool)
		usuarios[usuario] = true
	}
	propios, err = nuevosPoolsPostgreSQLDietasDesarrollo(ctx, cfg)
	if err != nil {
		// Sus errores son centinelas fijos (conexión, identidad, configuración),
		// sin DSN ni identidades: se conservan para diagnosticar el arranque.
		return nil, fmt.Errorf("%w: %w", errComposicionDietasEn(), err)
	}
	for _, pool := range []*pgxpool.Pool{propios.Dietas(), propios.Personal()} {
		var usuario string
		if pool == nil || pool.QueryRow(ctx, `SELECT session_user::text`).Scan(&usuario) != nil || usuarios[usuario] {
			return nil, errComposicionDietasEn()
		}
		usuarios[usuario] = true
	}
	registro, err := identidadpg.NuevoRegistroSesionesPostgreSQL(ctx, pools[0], pools[1], &seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	revalidador, err := identidadpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pools[1])
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pools[2])
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	reloj := relojRutasDietas{}
	servicioContexto, err := vecapp.NuevoServicioContextoActorProductivoV2(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	contextos, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	fuente, err := vecpg.NuevoAlmacenAutorizacion(pools[3])
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	registroAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(pools[4])
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools[5], c.MotivoPersonal.CatalogoID)
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	autorizador, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registroAutorizacion, registroAutorizacion, motivos, reloj, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	var emisores [3]emisorMaterialDietasDesarrollo
	for i, p := range []*proveedorMaterialAltaContratacionTemporalDesarrollo{material.personal, material.crear, material.consultar} {
		emisor, e := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, p)
		if e != nil {
			return nil, errComposicionDietasEn()
		}
		emisores[i] = emisor
	}
	nonce, err := nonceRutasDietas()
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, cuentas: cuentas, registro: registro, revalidador: revalidador, contextos: contextos, reloj: reloj, instancia: nonce}
	a := &autoridadComisionesDietasDesarrollo{base: base, reloj: reloj, cuentas: cuentas, cerrar: cerrar}
	seguridadComisiones := seguridadComisionesDietasDesarrollo{autoridad: a}
	calculador, err := nuevoCasoUsoCalculoRutas(cfg)
	if err != nil || calculador == nil {
		return nil, errComposicionDietasEn()
	}
	tarifas, err := dietaspg.NuevoRepositorioTarifasProvisionales(propios.Dietas())
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	puntos := map[string]dietasports.CoordenadaRuta{}
	for _, punto := range dietas.ProvinceRoutePoints() {
		puntos[punto.Code] = dietasports.CoordenadaRuta{Latitud: punto.Latitude, Longitud: punto.Longitude, Nombre: punto.Name}
	}
	preparador, err := dietasapp.NuevoPreparadorComision(puntos, calculador, tarifas)
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	rutas, colecciones, err := componerBorradoresDietas(dependenciasBorradoresDietas{personal: propios.Personal(), dietas: propios.Dietas(), seguridad: seguridadComisiones, reloj: reloj, emisorPersonal: &emisorComisionesDietasDesarrollo{personal: emisores[0]}, emisorDietas: &emisorComisionesDietasDesarrollo{crear: emisores[1], consultar: emisores[2]}, motivoPersonal: c.MotivoPersonal, motivoDietas: c.MotivoCrear, preparador: preparador})
	if err != nil {
		return nil, err
	}
	a.rutas, a.colecciones = rutas, colecciones
	completa = true
	return a, nil
}

// errComposicionDietasEn conserva ErrComposicionBorradoresDietasNoDisponible y
// añade solo fichero y línea del rechazo: sin DSN, identidades ni material.
func errComposicionDietasEn() error {
	_, fichero, linea, ok := runtime.Caller(1)
	if !ok {
		return ErrComposicionBorradoresDietasNoDisponible
	}
	return fmt.Errorf("%w (%s:%d)", ErrComposicionBorradoresDietasNoDisponible, filepath.Base(fichero), linea)
}
