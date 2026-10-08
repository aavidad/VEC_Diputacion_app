package internactproveedores

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrOrganizacionHistoricaV3NoDisponible = errors.New("composicion interna: autorizacion organizacion historica no disponible")

// MaterialOrganizacionHistorica selecciona perfil y ámbito privados, nunca
// concede permisos. La capacidad HMAC es propia de OH; raíz y confianza son
// las de CT. Los archivos de secretos permanecen fuera de Git.
type MaterialOrganizacionHistorica struct {
	Version         int                                                    `json:"version"`
	CatalogoMotivos string                                                 `json:"catalogo_motivos"`
	MotivoConsulta  core.ReferenciaEntradaCatalogo                         `json:"motivo_consulta"`
	Capacidad       capacidadMaterial                                      `json:"capacidad"`
	Contextos       map[string]internagobierno.AmbitoOrganizacionHistorica `json:"contextos"`
	raiz            *os.Root
}

func (MaterialOrganizacionHistorica) String() string {
	return "[material organizacion historica privado]"
}
func (m MaterialOrganizacionHistorica) GoString() string { return m.String() }
func (m MaterialOrganizacionHistorica) Format(f fmt.State, _ rune) {
	_, _ = f.Write([]byte(m.String()))
}
func (m MaterialOrganizacionHistorica) LogValue() slog.Value { return slog.StringValue(m.String()) }
func (MaterialOrganizacionHistorica) MarshalJSON() ([]byte, error) {
	return []byte(`"[material organizacion historica privado]"`), nil
}

func (m *MaterialOrganizacionHistorica) Cerrar() error {
	if m == nil || m.raiz == nil {
		return nil
	}
	err := m.raiz.Close()
	m.raiz = nil
	return err
}

func CargarMaterialOrganizacionHistorica(directorio string) (MaterialOrganizacionHistorica, error) {
	var m MaterialOrganizacionHistorica
	if !filepath.IsAbs(directorio) || filepath.Clean(directorio) != directorio || strings.TrimSpace(directorio) != directorio || dentroRepositorioGit(directorio) {
		return m, ErrOrganizacionHistoricaV3NoDisponible
	}
	resuelta, err := filepath.EvalSymlinks(directorio)
	if err != nil || resuelta != directorio {
		return m, ErrOrganizacionHistoricaV3NoDisponible
	}
	info, err := os.Lstat(directorio)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return m, ErrOrganizacionHistoricaV3NoDisponible
	}
	raiz, err := os.OpenRoot(directorio)
	if err != nil {
		return m, ErrOrganizacionHistoricaV3NoDisponible
	}
	defer func() {
		if m.raiz == nil {
			_ = raiz.Close()
		}
	}()
	actual, err := raiz.Stat(".")
	if err != nil || !os.SameFile(info, actual) {
		return m, ErrOrganizacionHistoricaV3NoDisponible
	}
	b, err := leerArchivoPrivado(raiz, "organizacion_historica_v3.json", maximoInventarioCT)
	if err != nil {
		return m, ErrOrganizacionHistoricaV3NoDisponible
	}
	defer clear(b)
	if clavesJSONDuplicadas(b) {
		return m, ErrOrganizacionHistoricaV3NoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil || d.Decode(new(any)) != io.EOF || !materialOrganizacionHistoricaValido(m) {
		return MaterialOrganizacionHistorica{}, ErrOrganizacionHistoricaV3NoDisponible
	}
	m.raiz = raiz
	return m, nil
}

func materialOrganizacionHistoricaValido(m MaterialOrganizacionHistorica) bool {
	if m.Version != 1 || m.CatalogoMotivos == "" || !core.ReferenciaMotivoAutorizacionV2Valida(m.MotivoConsulta) || m.MotivoConsulta.CatalogoID != m.CatalogoMotivos ||
		!filepath.IsLocal(m.Capacidad.Archivo) || m.Capacidad.Archivo == "." || len(m.Contextos) == 0 || len(m.Contextos) > 128 {
		return false
	}
	for cuenta, a := range m.Contextos {
		// Sólo sintaxis: F1, el PDP y el consumo transaccional comprueban la
		// asignación y sus versiones. La selección sólo restringe el PDP.
		if (core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: cuenta, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceSubstantial}, PerfilActivoRef: a.PerfilActivoRef}).Validar() != nil || a.PerfilVersion == 0 ||
			a.Validar() != nil {
			return false
		}
	}
	return true
}

type fuenteOrganizacionHistorica interface {
	ContextoVinculadoOrganizacionHistorica(context.Context) (ct.ContextoAutorizacionAltaV3, string, string, error)
}

type ProveedorAutorizacionOrganizacionHistorica struct {
	fuente    fuenteOrganizacionHistorica
	reloj     ct.Reloj
	motivo    core.ReferenciaEntradaCatalogo
	contextos map[string]internagobierno.AmbitoOrganizacionHistorica
	emisor    emisorMaterialV3
	firmante  *firmanteOrganizacionHistoricaV3
}

func (p *ProveedorAutorizacionOrganizacionHistorica) Cerrar() {
	if p == nil {
		return
	}
	// El firmante y el lector son compartidos: OH sólo suelta sus referencias.
	p.firmante = nil
	p.emisor = nil
	p.fuente = nil
	p.contextos = nil
}

func ConstruirOrganizacionHistorica(ctx context.Context, m MaterialOrganizacionHistorica, base *Proveedores, fuente *internagobierno.FuenteF1, reloj ct.Reloj) (*ProveedorAutorizacionOrganizacionHistorica, error) {
	if base == nil {
		return nil, ErrOrganizacionHistoricaV3NoDisponible
	}
	return construirOrganizacionHistorica(ctx, m, base.pdp, base.catalogoMotivos, base.gobierno, base.firmante, fuente, reloj)
}

func construirOrganizacionHistorica(ctx context.Context, m MaterialOrganizacionHistorica, pdp vecports.AutorizadorSolicitudLigadaV3, catalogo string, g *gobiernoV3Compartido, firmante *firmanteV3, fuente fuenteOrganizacionHistorica, reloj ct.Reloj) (*ProveedorAutorizacionOrganizacionHistorica, error) {
	if ctx == nil || ctx.Err() != nil || m.raiz == nil || !materialOrganizacionHistoricaValido(m) || interfazNula(pdp) || interfazNula(fuente) || interfazNula(reloj) ||
		catalogo != m.CatalogoMotivos || g == nil || g.lector == nil || g.coord.AudienciaDespliegue != audienciaAtestacionCTInterna ||
		firmante == nil || len(firmante.privada) != ed25519.PrivateKeySize || firmante.claveID != g.coord.ClaveID || firmante.audiencia != g.coord.AudienciaDespliegue {
		return nil, ErrOrganizacionHistoricaV3NoDisponible
	}
	// Este lector acredita la configuración y raíz comunes. AD3-69 no tiene
	// consumidor OH: no se envía su clave a las sondas CT o Personal B2.
	// AD3-51/P10 revalidan clave OH, revocación, ventana, perfil y F1 al
	// consumir dentro de la transacción final. No hay renovación HMAC aquí.
	if _, err := g.lector.Instantanea(ctx); err != nil {
		return nil, ErrOrganizacionHistoricaV3NoDisponible
	}
	capacidad, err := crearCapacidad(m.raiz, m.Capacidad, personal.AudienciaConsultaOrganizacionHistorica, reloj)
	if err != nil {
		return nil, ErrOrganizacionHistoricaV3NoDisponible
	}
	f := &firmanteOrganizacionHistoricaV3{base: firmante}
	atestador, err := app.NuevoServicioAtestacionesAutorizacionV3(core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: g.coord.ClaveID, Audiencia: g.coord.AudienciaDespliegue}, f)
	if err != nil {
		return nil, ErrOrganizacionHistoricaV3NoDisponible
	}
	contextos := make(map[string]internagobierno.AmbitoOrganizacionHistorica, len(m.Contextos))
	for cuenta, ambito := range m.Contextos {
		contextos[cuenta] = ambito
	}
	return &ProveedorAutorizacionOrganizacionHistorica{fuente: fuente, reloj: reloj, motivo: m.MotivoConsulta, contextos: contextos, firmante: f,
		emisor: &emisorMaterialRenovable{lector: g.lector, pdp: pdp, atestador: atestador, capacidad: capacidad}}, nil
}

func (p *ProveedorAutorizacionOrganizacionHistorica) AutorizarConsultaOrganizacionHistorica(ctx context.Context, m personal.MaterialConsultaOrganizacionHistorica) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacio vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil || ctx.Err() != nil || p == nil || interfazNula(p.fuente) || interfazNula(p.reloj) || p.firmante == nil || interfazNula(p.emisor) {
		return vacio, ErrOrganizacionHistoricaV3NoDisponible
	}
	s := m.Solicitud()
	recurso := m.Recurso()
	if s.Actor.Validar() != nil || s.Selector.Validar() != nil || recurso.Validar() != nil || recurso.ModuloID != "personal" || recurso.Tipo != "organizacion_historica" {
		return vacio, ErrOrganizacionHistoricaV3NoDisponible
	}
	// Conservar exactamente el resultado sellado para el actor HTTP; una
	// segunda resolución F1 rompería la ligadura de ResueltoEn.
	var base ct.ContextoAutorizacionAltaV3
	var organismo, unidad string
	var err error
	if original, errOriginal := intentoOriginalOrganizacionHistorica(ctx); errOriginal == nil {
		base, organismo, unidad, err = original.identidad, original.organismo, original.unidad, nil
	} else {
		base, organismo, unidad, err = p.fuente.ContextoVinculadoOrganizacionHistorica(ctx)
	}
	if err != nil || base.Resultado.Validar() != nil || base.Vinculo.ValidarPara(base.Resultado) != nil || !reflect.DeepEqual(s.Actor, base.Resultado.Contexto) {
		return vacio, ErrOrganizacionHistoricaV3NoDisponible
	}
	datos, err := base.Vinculo.Datos()
	ambito, ok := p.contextos[s.Actor.Instantanea.CuentaRef]
	if err != nil || !ok || datos.CuentaRef != s.Actor.Instantanea.CuentaRef || ambito.PerfilActivoRef != s.Actor.PerfilActivoRef || ambito.PerfilVersion != s.Actor.Instantanea.PerfilVersion ||
		ambito.OrganismoRef != organismo || ambito.UnidadClave != unidad || s.Selector.OrganismoRef != organismo || (unidad != "" && s.Selector.UnidadClave != unidad) ||
		!base.Vinculo.VigenteEn(p.reloj.Ahora(), base.Resultado) {
		return vacio, ErrOrganizacionHistoricaV3NoDisponible
	}
	var correlacion core.ReferenciaCorrelacionAutorizacionV2
	if _, errOriginal := intentoOriginalOrganizacionHistorica(ctx); errOriginal == nil {
		correlacion, err = vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	} else {
		correlacion, err = core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seg.GeneradorReferenciasCriptograficas{})
	}
	if err != nil {
		return vacio, ErrOrganizacionHistoricaV3NoDisponible
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: base.Vinculo, ReferenciaMotivo: p.motivo, Accion: personal.AccionConsultaOrganizacionHistorica, Recurso: recurso, Finalidad: "consultar_organizacion_historica", Correlacion: correlacion})
	if err != nil {
		return vacio, ErrOrganizacionHistoricaV3NoDisponible
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, base.Resultado)
	if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) && ctx.Err() == nil {
		return vacio, personal.ErrConsultaOrganizacionHistoricaDenegada
	}
	if err != nil || decision.ValidarPara(solicitud) != nil || interfazNula(exportador) {
		return vacio, ErrOrganizacionHistoricaV3NoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, base.Resultado, p.motivo, material, personal.AudienciaConsultaOrganizacionHistorica) || ctx.Err() != nil {
		return vacio, ErrOrganizacionHistoricaV3NoDisponible
	}
	return material, nil
}

type firmanteOrganizacionHistoricaV3 struct{ base *firmanteV3 }

func (f *firmanteOrganizacionHistoricaV3) FirmarAtestacionAutorizacionV3(ctx context.Context, s vecports.SolicitudFirmaAtestacionAutorizacionV3) (vecports.ResultadoFirmaAtestacionAutorizacionV3, error) {
	if f == nil || f.base == nil {
		return vecports.ResultadoFirmaAtestacionAutorizacionV3{}, vecports.ErrFirmaAtestacionNoDisponible
	}
	return f.base.firmarConEvidencia(ctx, s, "evidencia:firma:personal:organizacion-historica:")
}

var _ personalports.ProveedorAutorizacionConsultaOrganizacionHistorica = (*ProveedorAutorizacionOrganizacionHistorica)(nil)
