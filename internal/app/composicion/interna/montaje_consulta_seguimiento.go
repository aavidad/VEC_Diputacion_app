package interna

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	ctcomposicion "vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal"
	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El adaptador de certificado mTLS directo emite una aserción firmada por
// petición desde el handshake verificado. No acepta Bearer, cabeceras de
// identidad del navegador ni certificados reenviados.
type extractorAsercionInstitucional interface {
	ExtraerAsercionProtegida(*http.Request) ([]byte, error)
}

// El cargador transfiere los once pools y recursos auxiliares a la aplicación.
// ConfiguracionV2 incluye F1 registrado, V3 y roles PostgreSQL acreditados.
type proveedoresConsultaSeguimiento struct {
	identidad          *httpseguridad.ServicioIdentidad
	extractor          extractorAsercionInstitucional
	autoridadRutas     httpapi.AutoridadRutasExactas
	auditoriaRutas     vecports.RegistradorAuditoriaFronteraRutaExacta
	vincularPersonalB2 func(context.Context) (context.Context, error)
	fichaPersonalB2    http.Handler
	vacantesPersonalB2 http.Handler
	altaPersonalB2     http.Handler
	hechoPersonalB2    http.Handler
	configuracionV2    inc.ConfiguracionServidorV2PostgreSQL
	recursos           []recursoCerrableAplicacionInterna
}

// La carga exige que todas las autoridades estén presentes antes de abrir red.
func obtenerProveedoresConsultaSeguimiento(ctx context.Context, cfg Configuracion) (proveedoresConsultaSeguimiento, error) {
	return cargarProveedoresGobernados(ctx, cfg)
}

type puenteConsultaSeguimiento struct {
	extractor    extractorAsercionInstitucional
	api          http.Handler
	auditoria    vecports.RegistradorAuditoriaFronteraRutaExacta
	vincularB2   func(context.Context) (context.Context, error)
	fachada      atomic.Pointer[FachadaIdentidadOffline]
	limiteCuerpo int64
	personalB2   bool
}

const plazoAuditoriaDenegacionSeguimiento = 250 * time.Millisecond

func (p *puenteConsultaSeguimiento) sellar(f *FachadaIdentidadOffline) bool {
	return p != nil && f != nil && p.fachada.CompareAndSwap(nil, f)
}

func (p *puenteConsultaSeguimiento) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p == nil || r == nil || r.URL == nil || interfazNulaIdentidadOffline(p.extractor) ||
		manejadorNulo(p.api) || p.fachada.Load() == nil {
		responderPuenteSeguimiento(w, http.StatusServiceUnavailable)
		return
	}
	if r.URL.Path != httpct.RutaConsultaSeguimientoV2 &&
		(!p.personalB2 || !internagobierno.RutaInternaGobernada(r.URL.Path)) {
		responderPuenteSeguimiento(w, http.StatusNotFound)
		return
	}
	if r.URL.RawPath != "" || r.URL.Opaque != "" || r.URL.EscapedPath() != r.URL.Path ||
		r.URL.Fragment != "" || r.URL.ForceQuery || r.URL.Scheme != "" || r.URL.Host != "" || r.URL.User != nil {
		responderPuenteSeguimiento(w, http.StatusNotFound)
		return
	}
	preparada, err := httpseguridad.PrepararPeticionAsercionPasarela(r, p.limiteCuerpo)
	if err != nil {
		p.auditarAutenticacionRequerida(r.Context(), r.URL.Path)
		responderPuenteSeguimiento(w, http.StatusUnauthorized)
		return
	}
	defer preparada.Body.Close()
	asercion, err := p.extractor.ExtraerAsercionProtegida(preparada)
	if err != nil || len(asercion) == 0 {
		clear(asercion)
		p.auditarAutenticacionRequerida(preparada.Context(), preparada.URL.Path)
		responderPuenteSeguimiento(w, http.StatusUnauthorized)
		return
	}
	defer clear(asercion)
	ctx, err := p.fachada.Load().AutenticarYVincular(preparada.Context(), asercion)
	if err != nil || ctx == nil {
		p.auditarAutenticacionRequerida(preparada.Context(), preparada.URL.Path)
		responderPuenteSeguimiento(w, http.StatusUnauthorized)
		return
	}
	if preparada.URL.Path != httpct.RutaConsultaSeguimientoV2 {
		if p.vincularB2 == nil {
			responderPuenteSeguimiento(w, http.StatusServiceUnavailable)
			return
		}
		ctx, err = p.vincularB2(ctx)
		if err != nil || ctx == nil {
			if p.auditarDenegacionPersonalB2(preparada.Context(), preparada.URL.Path) {
				responderPuenteSeguimiento(w, http.StatusForbidden)
			} else {
				responderPuenteSeguimiento(w, http.StatusServiceUnavailable)
			}
			return
		}
	}
	p.api.ServeHTTP(w, preparada.WithContext(ctx))
}

func (p *puenteConsultaSeguimiento) auditarDenegacionPersonalB2(ctx context.Context, ruta string) bool {
	if p == nil || ctx == nil || interfazNulaIdentidadOffline(p.auditoria) {
		return false
	}
	if ruta != httpapi.RutaVacantesEmpleadoB2 && ruta != "/api/vec/personal/empleados" && ruta != "/api/vec/personal/hechos" {
		ruta = "/api/vec/personal/empleados/{emp_ref}"
	}
	orden := vecports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: correlacionDenegacionSeguimiento(),
		Motivo:         vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado,
		Superficie:     vecports.SuperficieAuditoriaFronteraRutaExactaPersonal,
		Ruta:           ruta,
	}
	if orden.Validar() != nil {
		return false
	}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoAuditoriaDenegacionSeguimiento)
	defer cancelar()
	if err := p.auditoria.RegistrarAuditoriaFronteraRutaExacta(ctxAuditoria, orden); err != nil {
		log.Printf("composicion interna: auditoria_frontera_no_registrada correlacion=%s", orden.CorrelacionRef)
		return false
	}
	return true
}

func (p *puenteConsultaSeguimiento) auditarAutenticacionRequerida(ctx context.Context, ruta string) {
	if p == nil || ctx == nil || interfazNulaIdentidadOffline(p.auditoria) {
		return
	}
	correlacion := correlacionDenegacionSeguimiento()
	superficie := vecports.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal
	if ruta != httpct.RutaConsultaSeguimientoV2 {
		superficie = vecports.SuperficieAuditoriaFronteraRutaExactaPersonal
		if ruta != httpapi.RutaVacantesEmpleadoB2 && ruta != "/api/vec/personal/empleados" && ruta != "/api/vec/personal/hechos" {
			ruta = "/api/vec/personal/empleados/{emp_ref}"
		}
	}
	orden := vecports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: correlacion,
		Motivo:         vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida,
		Superficie:     superficie,
		Ruta:           ruta,
	}
	if orden.Validar() != nil {
		return
	}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoAuditoriaDenegacionSeguimiento)
	defer cancelar()
	if err := p.auditoria.RegistrarAuditoriaFronteraRutaExacta(ctxAuditoria, orden); err != nil {
		log.Printf("composicion interna: auditoria_frontera_no_registrada correlacion=%s", correlacion)
	}
}

func correlacionDenegacionSeguimiento() string {
	aleatorio := make([]byte, 16)
	if _, err := rand.Read(aleatorio); err != nil {
		return "corr_no_disponible"
	}
	return "corr_" + hex.EncodeToString(aleatorio)
}

func responderPuenteSeguimiento(w http.ResponseWriter, estado int) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
}

func nuevaAPIConsultaSeguimiento(p proveedoresConsultaSeguimiento, s *inc.ServidorV2PostgreSQL) (http.Handler, error) {
	if interfazNulaIdentidadOffline(p.autoridadRutas) ||
		interfazNulaIdentidadOffline(p.auditoriaRutas) || s == nil {
		return nil, ErrAPIInternaNoDisponible
	}
	ruta, err := ctcomposicion.NuevaRutaConsultaSeguimientoV2(s)
	if err != nil || ruta.Ruta != httpct.RutaConsultaSeguimientoV2 || manejadorNulo(ruta.Manejador) {
		return nil, ErrAPIInternaNoDisponible
	}
	ctAPI, err := httpapi.NewHandlerSoloRutasExactas([]httpapi.RutaExacta{ruta}, p.autoridadRutas, p.auditoriaRutas)
	if err != nil {
		return nil, ErrAPIInternaNoDisponible
	}
	if manejadorNulo(p.fichaPersonalB2) && manejadorNulo(p.vacantesPersonalB2) &&
		manejadorNulo(p.altaPersonalB2) && manejadorNulo(p.hechoPersonalB2) {
		return ctAPI, nil
	}
	return nuevoEnrutadorPersonalB2(ctAPI, p.fichaPersonalB2, p.vacantesPersonalB2,
		p.altaPersonalB2, p.hechoPersonalB2, p.autoridadRutas, p.auditoriaRutas)
}

func componerConsultaSeguimiento(ctx context.Context, cfg Configuracion, p proveedoresConsultaSeguimiento) (*AplicacionInterna, error) {
	transferida := false
	defer func() {
		if !transferida {
			cerrarProveedoresConsultaSeguimiento(p)
		}
	}()
	if ctx == nil || ctx.Err() != nil || cfg.Validar() != nil || p.identidad == nil ||
		interfazNulaIdentidadOffline(p.extractor) ||
		interfazNulaIdentidadOffline(p.autoridadRutas) ||
		interfazNulaIdentidadOffline(p.auditoriaRutas) ||
		(!manejadorNulo(p.fichaPersonalB2) && p.vincularPersonalB2 == nil) ||
		interfazNulaIdentidadOffline(p.configuracionV2.FuenteAutoridad) {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	p.configuracionV2.FuenteAutoridad = fuenteAutoridadIdentidadVinculada{
		identidad: p.identidad, siguiente: p.configuracionV2.FuenteAutoridad,
		desarrolloCertificadoPersonal: !cfg.RetiradaPoliticaInternaEn.IsZero(),
	}
	servidorV2, err := inc.NuevoServidorV2PostgreSQL(p.configuracionV2)
	if err != nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	api, err := nuevaAPIConsultaSeguimiento(p, servidorV2)
	if err != nil {
		return nil, ErrAPIInternaNoDisponible
	}
	limiteCuerpo := min(cfg.normalizar().MaximoBytesPeticion, int64(1<<20))
	puente := &puenteConsultaSeguimiento{extractor: p.extractor, api: api, auditoria: p.auditoriaRutas, limiteCuerpo: limiteCuerpo,
		vincularB2: p.vincularPersonalB2,
		personalB2: !manejadorNulo(p.fichaPersonalB2) && !manejadorNulo(p.vacantesPersonalB2) &&
			!manejadorNulo(p.altaPersonalB2) && !manejadorNulo(p.hechoPersonalB2)}
	servidor, err := construirServidorInterno(cfg, puente)
	if err != nil {
		return nil, err
	}
	fachada, err := NuevaFachadaIdentidadOffline(p.identidad, servidor)
	if err != nil || !puente.sellar(fachada) {
		_ = servidor.Apagar(context.Background())
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	recursos := []recursoCerrableAplicacionInterna{
		&recursoPoolsSeguimiento{pools: poolsConsultaSeguimiento(p.configuracionV2)},
	}
	recursos = append(recursos, p.recursos...)
	a, err := nuevaAplicacionInterna(servidor, recursos...)
	if err != nil {
		return nil, ErrAplicacionInternaNoDisponible
	}
	transferida = true
	return a, nil
}

type recursoPoolsSeguimiento struct {
	pools     []*pgxpool.Pool
	propiedad atomic.Bool
	cerrarUna sync.Once
}

func (r *recursoPoolsSeguimiento) reclamarPropiedad() bool {
	return r != nil && len(r.pools) == 11 && r.propiedad.CompareAndSwap(false, true)
}

func (r *recursoPoolsSeguimiento) cerrar() error {
	if r == nil || !r.propiedad.Load() {
		return ErrAplicacionInternaNoDisponible
	}
	r.cerrarUna.Do(func() { cerrarPoolsSeguimiento(r.pools) })
	return nil
}

func poolsConsultaSeguimiento(c inc.ConfiguracionServidorV2PostgreSQL) []*pgxpool.Pool {
	p := c.Preparacion.Pools
	return []*pgxpool.Pool{c.AltaPersonal, c.RegistroCT, p.InicialCT, p.LocalizadorCT,
		p.LocalizadorPersonal, p.LecturaPersonal, p.Historia.RegistroCT,
		p.Historia.Autenticacion, p.Historia.Contexto, p.Historia.Evaluacion,
		p.Historia.Concesion}
}

func cerrarPoolsSeguimiento(pools []*pgxpool.Pool) {
	vistos := make(map[*pgxpool.Pool]bool, len(pools))
	for _, pool := range pools {
		if pool != nil && !vistos[pool] {
			vistos[pool] = true
			pool.Close()
		}
	}
}

func cerrarProveedoresConsultaSeguimiento(p proveedoresConsultaSeguimiento) {
	for i := len(p.recursos) - 1; i >= 0; i-- {
		recurso := p.recursos[i]
		if !interfazNulaIdentidadOffline(recurso) && recurso.reclamarPropiedad() {
			_ = cerrarRecursoAplicacionInterna(recurso)
		}
	}
	cerrarPoolsSeguimiento(poolsConsultaSeguimiento(p.configuracionV2))
}
