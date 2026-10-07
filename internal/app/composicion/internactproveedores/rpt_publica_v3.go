package internactproveedores

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrRPTPublicaV3NoDisponible = errors.New("composicion interna: autorizacion RPT publicada no disponible")

type PerfilRPTPublicaV3 struct {
	PerfilActivoRef string `json:"perfil_activo_ref"`
	PerfilVersion   uint64 `json:"perfil_version"`
}

// El inventario privado restringe cuentas, perfil y versión. No concede la
// acción RPT: el PDP central decide cada lectura con su recurso exacto.
type MaterialRPTPublicaV3 struct {
	Version         int                            `json:"version"`
	CatalogoMotivos string                         `json:"catalogo_motivos"`
	MotivoConsulta  core.ReferenciaEntradaCatalogo `json:"motivo_consulta"`
	Capacidad       capacidadMaterial              `json:"capacidad"`
	Perfiles        map[string]PerfilRPTPublicaV3  `json:"perfiles"`
	raiz            *os.Root
}

func (MaterialRPTPublicaV3) String() string     { return "[material RPT publicada privado]" }
func (m MaterialRPTPublicaV3) GoString() string { return m.String() }
func (MaterialRPTPublicaV3) MarshalJSON() ([]byte, error) {
	return []byte(`"[material RPT publicada privado]"`), nil
}
func (m *MaterialRPTPublicaV3) Cerrar() error {
	if m == nil || m.raiz == nil {
		return nil
	}
	err := m.raiz.Close()
	m.raiz = nil
	return err
}

func CargarMaterialRPTPublicaV3(directorio string) (MaterialRPTPublicaV3, error) {
	var m MaterialRPTPublicaV3
	if !filepath.IsAbs(directorio) || filepath.Clean(directorio) != directorio || strings.TrimSpace(directorio) != directorio || dentroRepositorioGit(directorio) {
		return m, ErrRPTPublicaV3NoDisponible
	}
	resuelta, err := filepath.EvalSymlinks(directorio)
	if err != nil || resuelta != directorio {
		return m, ErrRPTPublicaV3NoDisponible
	}
	info, err := os.Lstat(directorio)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return m, ErrRPTPublicaV3NoDisponible
	}
	raiz, err := os.OpenRoot(directorio)
	if err != nil {
		return m, ErrRPTPublicaV3NoDisponible
	}
	defer func() {
		if m.raiz == nil {
			_ = raiz.Close()
		}
	}()
	actual, err := raiz.Stat(".")
	if err != nil || !os.SameFile(info, actual) {
		return m, ErrRPTPublicaV3NoDisponible
	}
	bruto, err := leerArchivoPrivado(raiz, "rpt_publica_v3.json", maximoInventarioCT)
	if err != nil {
		return m, ErrRPTPublicaV3NoDisponible
	}
	defer clear(bruto)
	if clavesJSONDuplicadas(bruto) {
		return m, ErrRPTPublicaV3NoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(bruto))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil || d.Decode(new(any)) != io.EOF || !materialRPTPublicaV3Valido(m) {
		return MaterialRPTPublicaV3{}, ErrRPTPublicaV3NoDisponible
	}
	m.raiz = raiz
	return m, nil
}

func materialRPTPublicaV3Valido(m MaterialRPTPublicaV3) bool {
	if m.Version != 1 || m.CatalogoMotivos == "" || !core.ReferenciaMotivoAutorizacionV2Valida(m.MotivoConsulta) ||
		m.MotivoConsulta.CatalogoID != m.CatalogoMotivos || !filepath.IsLocal(m.Capacidad.Archivo) || m.Capacidad.Archivo == "." ||
		len(m.Perfiles) == 0 || len(m.Perfiles) > 128 {
		return false
	}
	for cuenta, perfil := range m.Perfiles {
		s := core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: cuenta, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceSubstantial}, PerfilActivoRef: perfil.PerfilActivoRef}
		if s.Validar() != nil || perfil.PerfilVersion == 0 {
			return false
		}
	}
	return true
}

// V sella la identidad F1 específica de esta operación una sola vez tras
// mTLS. El proveedor vuelve a leer ese mismo sello; jamás resuelve otro actor.
type fuenteContextoRPTPublicaV3 interface {
	ContextoVinculadoRPTPublicaV2(context.Context) (ct.ContextoAutorizacionAltaV3, error)
}

type ProveedorAutorizacionRPTPublicaV3 struct {
	fuente   fuenteContextoRPTPublicaV3
	reloj    ct.Reloj
	motivo   core.ReferenciaEntradaCatalogo
	perfiles map[string]PerfilRPTPublicaV3
	emisor   emisorMaterialV3
	firmante *firmanteRPTPublicaV3
}

func (p *ProveedorAutorizacionRPTPublicaV3) Cerrar() {
	if p == nil {
		return
	}
	p.fuente, p.reloj, p.emisor, p.firmante, p.perfiles = nil, nil, nil, nil, nil
}

func ConstruirRPTPublicaV3(ctx context.Context, m MaterialRPTPublicaV3, base *Proveedores, fuente fuenteContextoRPTPublicaV3, reloj ct.Reloj) (*ProveedorAutorizacionRPTPublicaV3, error) {
	if ctx == nil || ctx.Err() != nil || base == nil || m.raiz == nil || !materialRPTPublicaV3Valido(m) || interfazNula(fuente) || interfazNula(reloj) ||
		interfazNula(base.pdp) || base.catalogoMotivos != m.CatalogoMotivos || base.gobierno == nil || base.gobierno.lector == nil ||
		base.gobierno.coord.AudienciaDespliegue != audienciaAtestacionCTInterna || base.firmante == nil ||
		len(base.firmante.privada) != ed25519.PrivateKeySize || base.firmante.claveID != base.gobierno.coord.ClaveID ||
		base.firmante.audiencia != base.gobierno.coord.AudienciaDespliegue {
		return nil, ErrRPTPublicaV3NoDisponible
	}
	if _, err := base.gobierno.lector.Instantanea(ctx); err != nil {
		return nil, ErrRPTPublicaV3NoDisponible
	}
	capacidad, err := crearCapacidad(m.raiz, m.Capacidad, personal.AudienciaConsultaRPTPublicaV2, reloj)
	if err != nil {
		return nil, ErrRPTPublicaV3NoDisponible
	}
	firmante := &firmanteRPTPublicaV3{base: base.firmante}
	atestador, err := app.NuevoServicioAtestacionesAutorizacionV3(core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: base.gobierno.coord.ClaveID, Audiencia: base.gobierno.coord.AudienciaDespliegue}, firmante)
	if err != nil {
		return nil, ErrRPTPublicaV3NoDisponible
	}
	perfiles := make(map[string]PerfilRPTPublicaV3, len(m.Perfiles))
	for cuenta, perfil := range m.Perfiles {
		perfiles[cuenta] = perfil
	}
	return &ProveedorAutorizacionRPTPublicaV3{fuente: fuente, reloj: reloj, motivo: m.MotivoConsulta, perfiles: perfiles,
		emisor: &emisorMaterialRenovable{lector: base.gobierno.lector, pdp: base.pdp, atestador: atestador, capacidad: capacidad}, firmante: firmante}, nil
}

func (p *ProveedorAutorizacionRPTPublicaV3) AutorizarConsultaRPTPublicaV2(ctx context.Context, m personal.MaterialConsultaRPTPublicaV2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacio vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if p == nil || ctx == nil || ctx.Err() != nil || interfazNula(p.fuente) || interfazNula(p.reloj) || interfazNula(p.emisor) || p.firmante == nil {
		return vacio, ErrRPTPublicaV3NoDisponible
	}
	s := m.Solicitud()
	reconstruido, err := personal.NuevoMaterialConsultaRPTPublicaV2(s)
	recurso := m.Recurso()
	if err != nil || !bytes.Equal(reconstruido.Canonico(), m.Canonico()) || !reflect.DeepEqual(recurso, reconstruido.Recurso()) ||
		recurso.ModuloID != "personal" || recurso.Tipo != "rpt_publica_publicacion" {
		return vacio, ErrRPTPublicaV3NoDisponible
	}
	base, err := p.fuente.ContextoVinculadoRPTPublicaV2(ctx)
	if err != nil || base.Resultado.Validar() != nil || base.Vinculo.ValidarPara(base.Resultado) != nil ||
		!reflect.DeepEqual(s.Actor, base.Resultado.Contexto) || !base.Vinculo.VigenteEn(p.reloj.Ahora(), base.Resultado) {
		return vacio, ErrRPTPublicaV3NoDisponible
	}
	datos, err := base.Vinculo.Datos()
	perfil, existe := p.perfiles[s.Actor.Instantanea.CuentaRef]
	if err != nil || !existe || datos.CuentaRef != s.Actor.Instantanea.CuentaRef ||
		perfil.PerfilActivoRef != s.Actor.PerfilActivoRef || perfil.PerfilVersion != s.Actor.Instantanea.PerfilVersion {
		return vacio, personal.ErrRPTPublicaV2Denegada
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seg.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, ErrRPTPublicaV3NoDisponible
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: base.Vinculo,
		ReferenciaMotivo: p.motivo, Accion: personal.AccionConsultaRPTPublicaV2, Recurso: recurso,
		Finalidad: personal.FinalidadConsultaRPTPublicaV2, Correlacion: correlacion})
	if err != nil {
		return vacio, ErrRPTPublicaV3NoDisponible
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, base.Resultado)
	if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) && ctx.Err() == nil {
		return vacio, personal.ErrRPTPublicaV2Denegada
	}
	if err != nil || decision.ValidarPara(solicitud) != nil || interfazNula(exportador) {
		return vacio, ErrRPTPublicaV3NoDisponible
	}
	if decision.ExigirProyeccionPara(solicitud, personal.CamposRespuestaRPTPublicaV2(), nil) != nil {
		return vacio, personal.ErrRPTPublicaV2Denegada
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, base.Resultado, p.motivo, material, personal.AudienciaConsultaRPTPublicaV2) || ctx.Err() != nil {
		return vacio, ErrRPTPublicaV3NoDisponible
	}
	return material, nil
}

type firmanteRPTPublicaV3 struct{ base *firmanteV3 }

func (f *firmanteRPTPublicaV3) FirmarAtestacionAutorizacionV3(ctx context.Context, s vecports.SolicitudFirmaAtestacionAutorizacionV3) (vecports.ResultadoFirmaAtestacionAutorizacionV3, error) {
	if f == nil || f.base == nil {
		return vecports.ResultadoFirmaAtestacionAutorizacionV3{}, vecports.ErrFirmaAtestacionNoDisponible
	}
	return f.base.firmarConEvidencia(ctx, s, "evidencia:firma:personal:rpt-publica:")
}

type AutoridadContextoRPTPublicaV3 struct{ Fuente fuenteContextoRPTPublicaV3 }

func (a AutoridadContextoRPTPublicaV3) ResolverContextoRPTPublicaV2(ctx context.Context) (core.ContextoActor, error) {
	if interfazNula(a.Fuente) || ctx == nil {
		return core.ContextoActor{}, ErrRPTPublicaV3NoDisponible
	}
	r, err := a.Fuente.ContextoVinculadoRPTPublicaV2(ctx)
	if err != nil {
		return core.ContextoActor{}, httpapi.ErrAccesoRutaExactaDenegado
	}
	return r.Resultado.Contexto.Clonar()
}

type AuditorDenegacionRPTPublicaV3 struct {
	Registrador vecports.RegistradorAuditoriaFronteraRutaExacta
}

func (a AuditorDenegacionRPTPublicaV3) RegistrarDenegacionRPTPublicaV2(ctx context.Context, d httpapi.DenegacionRPTPublicaV2) error {
	if ctx == nil || ctx.Err() != nil || interfazNula(a.Registrador) {
		return ErrRPTPublicaV3NoDisponible
	}
	o := vecports.OrdenAuditoriaFronteraRutaExacta{CorrelacionRef: d.CorrelacionRef, Motivo: vecports.MotivoAuditoriaFronteraRutaExacta(d.Motivo),
		Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaPersonal, Ruta: d.Ruta, ActorRef: d.ActorRef}
	if o.Validar() != nil {
		return ErrRPTPublicaV3NoDisponible
	}
	return a.Registrador.RegistrarAuditoriaFronteraRutaExacta(ctx, o)
}

var _ personalports.ProveedorAutorizacionRPTPublicaV2 = (*ProveedorAutorizacionRPTPublicaV3)(nil)
var _ httpapi.AutoridadContextoRPTPublicaV2 = AutoridadContextoRPTPublicaV3{}
var _ httpapi.AuditorDenegacionRPTPublicaV2 = AuditorDenegacionRPTPublicaV3{}
