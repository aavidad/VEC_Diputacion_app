package bootstrap

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	bolsapg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgrespreparacionbases"
	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	selauth "vec-diputacion-granada/internal/modules/seleccion/adapters/autorizacion"
	selhttp "vec-diputacion-granada/internal/modules/seleccion/adapters/http"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El montaje se prepara antes del catálogo común y se compone después con
// esa misma instancia. La raíz conserva el registro y la protección de rutas.
type MontajePreparacionBasesV3 struct {
	configuracion configuracionPreparacionBasesV3
	directorio    string
	perfiles      [2]*perfilPreparacionBasesV3
	reloj         relojContratacionTemporalDesarrollo
}

func NuevoMontajePreparacionBasesV3(cfg config.Config, base *soporteAltaContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo) (*MontajePreparacionBasesV3, bool, error) {
	c, existe, err := leerConfiguracionPreparacionBasesV3(cfg)
	if err != nil || !existe {
		return nil, existe, err
	}
	var ps [2]*perfilPreparacionBasesV3
	for i := range ps {
		p, err := nuevoPerfilPreparacionBasesV3(base, c, i == 0, reloj.Ahora())
		if err != nil {
			return nil, true, err
		}
		ps[i] = p
	}
	return &MontajePreparacionBasesV3{c, cfg.DevelopmentMaterialDir, ps, reloj}, true, nil
}

func (m *MontajePreparacionBasesV3) Fronteras() ([]descriptorFronteraComunDesarrollo, error) {
	if m == nil {
		return nil, errMontajePreparacionBasesV3
	}
	return fronterasPreparacionBasesHTTPV3(m.perfiles[0].perfilRef(), m.perfiles[1].perfilRef())
}

func (m *MontajePreparacionBasesV3) Autorizaciones(p politicaAutorizacionSolicitudLigadaV3Desarrollo) ([]descriptorAutorizacionComunDesarrollo, error) {
	if m == nil {
		return nil, errMontajePreparacionBasesV3
	}
	return autorizacionesPreparacionBasesHTTPV3(p)
}

// Todos los puertos son autoridades reales ya compuestas por el servidor.
// RegistrarRechazoFrontera usa la autoridad común de auditoría de la raíz.
// El material central tiene clave nominal propia por audiencia y raíz común.
type DependenciasMontajePreparacionBasesV3 struct {
	SesionBase                         *proveedorSesionConsultaRRHHDesarrollo
	Fronteras                          catalogoFronterasComunDesarrollo
	PDP                                vecports.AutorizadorSolicitudLigadaV3
	Gobierno                           *pgxpool.Pool
	MaterialGuardar, MaterialConsultar *proveedorMaterialAltaContratacionTemporalDesarrollo
	RegistrarRechazoFrontera           func(*http.Request) error
	RegistradorIntentos                vecports.RegistradorIntentosAuditoria
	ProcesoIntentos                    string
	LoginsReservados                   []string
}

// Componer se invoca una vez en arranque. Abre únicamente dos pools de Bolsa,
// consume identidad/contexto del núcleo y no registra rutas en otro listener.
func (m *MontajePreparacionBasesV3) Componer(ctx context.Context, d DependenciasMontajePreparacionBasesV3) ([]vechttp.RutaExacta, func(), error) {
	if ctx == nil || ctx.Err() != nil || m == nil || d.SesionBase == nil || d.SesionBase.soporte == nil || d.Gobierno == nil ||
		d.Fronteras.identidad == nil || dependenciaEsNulaContratacionTemporalDesarrollo(d.PDP) || d.RegistrarRechazoFrontera == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(d.RegistradorIntentos) || !procesoAuditoriaIntentosConfigurado(d.ProcesoIntentos) {
		return nil, nil, errMontajePreparacionBasesV3
	}
	for _, material := range []*proveedorMaterialAltaContratacionTemporalDesarrollo{d.MaterialGuardar, d.MaterialConsultar} {
		if material == nil || material.atestador == nil || material.confianza == nil || material.emisor == nil {
			return nil, nil, errMontajePreparacionBasesV3
		}
	}
	for _, p := range m.perfiles {
		if p == nil || p.soporte == nil || p.soporte.sello != d.SesionBase.soporte.sello ||
			p.soporte.principalID != d.SesionBase.soporte.principalID || p.soporte.certificadoSHA256 != d.SesionBase.soporte.certificadoSHA256 {
			return nil, nil, errMontajePreparacionBasesV3
		}
		f, ok := d.Fronteras.resolver(http.MethodPost, p.ruta)
		if !ok || len(f.PerfilesActivosRef) != 1 || f.PerfilesActivosRef[0] != p.perfilRef() ||
			f.ClavePolitica != clavePoliticaPreparacionBasesHTTPV3 || f.ClaveCapacidad != p.accion {
			return nil, nil, errMontajePreparacionBasesV3
		}
	}
	ctx, cancelar := context.WithTimeout(ctx, plazoarranque.Ampliar(30*time.Second))
	defer cancelar()
	pools, cerrar, err := m.abrirPools(ctx, d)
	if err != nil {
		return nil, nil, err
	}
	completo := false
	defer func() {
		if !completo {
			cerrar()
		}
	}()
	var sesiones [2]*proveedorSesionConsultaRRHHDesarrollo
	for i, p := range m.perfiles {
		if err := asegurarPerfilPreparacionBasesV3(ctx, d.Gobierno, p, m.reloj.Ahora()); err != nil {
			return nil, nil, err
		}
		operacion := referenciaAltaContratacionTemporalDesarrollo("oca_", p.soporte.principalID+"\x00"+p.soporte.certificadoSHA256+"\x00"+p.accion)
		if publicarResultadoContextoPostgreSQLDesarrollo(ctx, d.Gobierno, p.soporte.contexto.Resultado, operacion) != nil {
			return nil, nil, errMontajePreparacionBasesV3
		}
		esperado, err := contextoEsperadoRegistradoDesarrollo(ctx, d.SesionBase.resolutor, p.soporte)
		if err != nil {
			return nil, nil, errMontajePreparacionBasesV3
		}
		p.soporte.mu.Lock()
		p.soporte.contextoEsperadoRegistrado = esperado
		p.soporte.mu.Unlock()
		sesion, err := nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo(p.soporte, d.SesionBase.registro,
			d.SesionBase.revalidador, m.reloj, d.SesionBase.resolutor, d.Fronteras)
		if err != nil {
			return nil, nil, errMontajePreparacionBasesV3
		}
		sesiones[i] = sesion
	}
	broker, err := nuevoProveedorPreparacionBasesV3(m.perfiles, sesiones,
		[2]*proveedorMaterialAltaContratacionTemporalDesarrollo{d.MaterialGuardar, d.MaterialConsultar}, d.PDP, m.reloj, d.RegistrarRechazoFrontera)
	if err != nil {
		return nil, nil, err
	}
	var proveedores [2]*selauth.ProveedorPreparacionBases
	for i, p := range m.perfiles {
		a, err := selauth.NuevoProveedorPreparacionBases(broker, broker, p.motivo)
		if err != nil {
			return nil, nil, err
		}
		proveedores[i] = a
	}
	repo, err := bolsapg.NuevoRepositorio(pools[0], pools[1])
	if err != nil {
		return nil, nil, err
	}
	servicio, err := bolsaapp.NuevoServicioPreparacionBasesV3(autorizadorPreparacionBasesCompuestoV3{proveedores}, repo)
	if err != nil {
		return nil, nil, err
	}
	preparador, err := nuevoPreparadorBasesAuditadoV3(servicio, broker, d.RegistradorIntentos, d.ProcesoIntentos,
		m.configuracion.MotivoIntentoDenegado, m.configuracion.MotivoIntentoError)
	if err != nil {
		return nil, nil, err
	}
	broker.registrarEntrada = func(ctx context.Context, z contextoSeguridadComunDesarrollo, i int, c core.ReferenciaCorrelacionAutorizacionV2, err error) error {
		return preparador.registrar(ctx, z, c, "", broker.perfiles[i].accion, err)
	}
	frontera, err := nuevaFronteraPreparacionBasesHTTPV3(broker, d.RegistrarRechazoFrontera)
	if err != nil {
		return nil, nil, err
	}
	h, err := selhttp.NuevaPreparacionBasesHandler(selhttp.ConfigPreparacionBases{Preparador: preparador,
		ResolverContexto: broker.ResolverContextoHTTP, ValidarFrontera: frontera})
	if err != nil {
		return nil, nil, err
	}
	completo = true
	return []vechttp.RutaExacta{{Ruta: selhttp.RutaGuardarPreparacionBases, Manejador: h}, {Ruta: selhttp.RutaConsultarPreparacionBases, Manejador: h}}, cerrar, nil
}

func (m *MontajePreparacionBasesV3) abrirPools(ctx context.Context, d DependenciasMontajePreparacionBasesV3) ([2]*pgxpool.Pool, func(), error) {
	var pools [2]*pgxpool.Pool
	var una sync.Once
	cerrar := func() {
		una.Do(func() {
			for _, p := range pools {
				if p != nil {
					p.Close()
				}
			}
		})
	}
	raiz, err := os.OpenRoot(m.directorio)
	if err != nil {
		return pools, cerrar, errMontajePreparacionBasesV3
	}
	defer raiz.Close()
	usuarios := make(map[string]bool)
	for _, u := range d.LoginsReservados {
		if u != "" {
			usuarios[u] = true
		}
	}
	usuarios[d.Gobierno.Config().ConnConfig.User] = true
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, d.Gobierno)
	if err != nil {
		return pools, cerrar, errMontajePreparacionBasesV3
	}
	for i, archivo := range []string{m.configuracion.EscrituraFile, m.configuracion.LecturaFile} {
		b, err := leerArchivoIncorporacionV2(raiz, archivo, 16<<10)
		if err != nil {
			cerrar()
			return pools, cerrar, errMontajePreparacionBasesV3
		}
		p, usuario, err := abrirPoolPreparacionBasesV3(ctx, string(b), i == 0)
		borrarBytes(b)
		if err != nil {
			cerrar()
			return pools, cerrar, errMontajePreparacionBasesV3
		}
		pools[i] = p
		if usuarios[usuario] || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, p, topologia) != nil {
			cerrar()
			return pools, cerrar, errMontajePreparacionBasesV3
		}
		usuarios[usuario] = true
	}
	return pools, cerrar, nil
}

type autorizadorPreparacionBasesCompuestoV3 struct {
	proveedores [2]*selauth.ProveedorPreparacionBases
}

func (a autorizadorPreparacionBasesCompuestoV3) AutorizarGuardadoPreparacionBases(ctx context.Context, s bolsaports.SolicitudGuardarPreparacionBasesV3) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return a.proveedores[0].AutorizarGuardadoPreparacionBases(ctx, s)
}
func (a autorizadorPreparacionBasesCompuestoV3) AutorizarConsultaPreparacionBases(ctx context.Context, s bolsaports.SolicitudConsultarPreparacionBasesV3) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return a.proveedores[1].AutorizarConsultaPreparacionBases(ctx, s)
}

// Conserva las dos rutas al fallar una dependencia. La raíz debe mantenerlas
// bajo la misma frontera declarada; este manejador nunca autentica ni autoriza.
func RutasPreparacionBasesIndisponiblesV3() []vechttp.RutaExacta {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"servicio_no_disponible"}`))
	})
	return []vechttp.RutaExacta{{Ruta: selhttp.RutaGuardarPreparacionBases, Manejador: h}, {Ruta: selhttp.RutaConsultarPreparacionBases, Manejador: h}}
}

var _ bolsaports.ProveedorAutorizacionPreparacionBasesV3 = autorizadorPreparacionBasesCompuestoV3{}
