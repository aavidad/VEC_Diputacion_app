package internactproveedores

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	gocose "github.com/veraison/go-cose"
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

var ErrPersonalB2V3NoDisponible = errors.New("composicion interna: autorizacion Personal B2 no disponible")

// versionMaterialPersonalB2 4: el inventario ya no trae raíz, audiencia de
// atestación ni configuración propias. B2 usa la raíz única que carga CT y la
// configuración publicada vigente; el formato 3 (raíz B2) se rechaza.
const versionMaterialPersonalB2 = 4

// materialPersonalB2V3 contiene solo las ocho capacidades HMAC de consumo B2,
// en el orden cerrado de leer_configuracion_interna_v2('personal_b2').
type materialPersonalB2V3 struct {
	Capacidades struct {
		Ficha             capacidadMaterial `json:"ficha"`
		Vacantes          capacidadMaterial `json:"vacantes"`
		Alta              capacidadMaterial `json:"alta"`
		Hecho             capacidadMaterial `json:"hecho"`
		CatalogoConsultar capacidadMaterial `json:"catalogo_consultar"`
		CatalogoPublicar  capacidadMaterial `json:"catalogo_publicar"`
		CatalogoRetirar   capacidadMaterial `json:"catalogo_retirar"`
		Empleados         capacidadMaterial `json:"empleados"`
	} `json:"capacidades"`
}

// MaterialPersonalB2 sólo contiene nombres y huellas; todos sus archivos
// están en un directorio privado distinto del inventario de CT.
type MaterialPersonalB2 struct {
	Version         int    `json:"version"`
	CatalogoMotivos string `json:"catalogo_motivos"`
	Motivos         struct {
		Ficha             core.ReferenciaEntradaCatalogo `json:"ficha"`
		Vacantes          core.ReferenciaEntradaCatalogo `json:"vacantes"`
		Alta              core.ReferenciaEntradaCatalogo `json:"alta"`
		Hecho             core.ReferenciaEntradaCatalogo `json:"hecho"`
		CatalogoConsultar core.ReferenciaEntradaCatalogo `json:"catalogo_consultar"`
		CatalogoPublicar  core.ReferenciaEntradaCatalogo `json:"catalogo_publicar"`
		CatalogoRetirar   core.ReferenciaEntradaCatalogo `json:"catalogo_retirar"`
		Empleados         core.ReferenciaEntradaCatalogo `json:"empleados"`
	} `json:"motivos"`
	V3   materialPersonalB2V3 `json:"v3"`
	raiz *os.Root
}

func (MaterialPersonalB2) String() string     { return "[material Personal B2 privado]" }
func (m MaterialPersonalB2) GoString() string { return m.String() }
func (MaterialPersonalB2) MarshalJSON() ([]byte, error) {
	return []byte(`"[material Personal B2 privado]"`), nil
}
func (m *MaterialPersonalB2) Cerrar() error {
	if m == nil || m.raiz == nil {
		return nil
	}
	err := m.raiz.Close()
	m.raiz = nil
	return err
}

func CargarMaterialPersonalB2(directorio string) (MaterialPersonalB2, error) {
	var m MaterialPersonalB2
	if !filepath.IsAbs(directorio) || filepath.Clean(directorio) != directorio || strings.TrimSpace(directorio) != directorio || dentroRepositorioGit(directorio) {
		return m, ErrPersonalB2V3NoDisponible
	}
	resuelta, err := filepath.EvalSymlinks(directorio)
	if err != nil || resuelta != directorio {
		return m, ErrPersonalB2V3NoDisponible
	}
	info, err := os.Lstat(directorio)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return m, ErrPersonalB2V3NoDisponible
	}
	raiz, err := os.OpenRoot(directorio)
	if err != nil {
		return m, ErrPersonalB2V3NoDisponible
	}
	defer func() {
		if m.raiz == nil {
			_ = raiz.Close()
		}
	}()
	actual, err := raiz.Stat(".")
	if err != nil || !os.SameFile(info, actual) {
		return m, ErrPersonalB2V3NoDisponible
	}
	b, err := leerArchivoPrivado(raiz, "personal_b2_v3.json", maximoInventarioCT)
	if err != nil {
		return m, ErrPersonalB2V3NoDisponible
	}
	defer clear(b)
	if clavesJSONDuplicadas(b) {
		return m, ErrPersonalB2V3NoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil || d.Decode(new(any)) != io.EOF || m.Version != versionMaterialPersonalB2 || m.CatalogoMotivos == "" {
		return MaterialPersonalB2{}, ErrPersonalB2V3NoDisponible
	}
	for _, motivo := range []core.ReferenciaEntradaCatalogo{m.Motivos.Ficha, m.Motivos.Vacantes, m.Motivos.Alta, m.Motivos.Hecho, m.Motivos.CatalogoConsultar, m.Motivos.CatalogoPublicar, m.Motivos.CatalogoRetirar, m.Motivos.Empleados} {
		if !core.ReferenciaMotivoAutorizacionV2Valida(motivo) || motivo.CatalogoID != m.CatalogoMotivos {
			return MaterialPersonalB2{}, ErrPersonalB2V3NoDisponible
		}
	}
	for _, nombre := range []string{m.V3.Capacidades.Ficha.Archivo, m.V3.Capacidades.Vacantes.Archivo, m.V3.Capacidades.Alta.Archivo, m.V3.Capacidades.Hecho.Archivo, m.V3.Capacidades.CatalogoConsultar.Archivo, m.V3.Capacidades.CatalogoPublicar.Archivo, m.V3.Capacidades.CatalogoRetirar.Archivo, m.V3.Capacidades.Empleados.Archivo} {
		if !filepath.IsLocal(nombre) || nombre == "." {
			return MaterialPersonalB2{}, ErrPersonalB2V3NoDisponible
		}
	}
	m.raiz = raiz
	return m, nil
}

type ConfiguracionPersonalB2 struct {
	Material MaterialPersonalB2
	Base     *Proveedores
	Fuente   *internagobierno.FuenteF1
	Reloj    ct.Reloj
}

// emisorMaterialV3 es la emisión atestada que consume cada capacidad B2.
type emisorMaterialV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type ProveedorAutorizacionPersonalB2 struct {
	fuente   *internagobierno.FuenteF1
	reloj    ct.Reloj
	motivos  [8]core.ReferenciaEntradaCatalogo
	emisores [8]emisorMaterialV3
	firmante *firmantePersonalB2V3
}

func (p *ProveedorAutorizacionPersonalB2) Cerrar() {
	if p == nil {
		return
	}
	if p.firmante != nil {
		clear(p.firmante.privada)
		p.firmante = nil
	}
	p.emisores = [8]emisorMaterialV3{}
	p.fuente = nil
}

// dependenciasPersonalB2 reúne lo que B2 toma de la composición CT: la raíz
// compartida ya validada, su firmante, el PDP de su catálogo y las dos
// funciones v2 del preflight. Separarlo permite probar la renovación sin SQL.
type dependenciasPersonalB2 struct {
	pdp       vecports.AutorizadorSolicitudLigadaV3
	catalogo  string
	gobierno  *gobiernoV3Compartido
	firmante  *firmanteV3
	comprobar comprobacionGobiernoV3
	leer      consultaGobiernoV3
	fuente    *internagobierno.FuenteF1
	reloj     ct.Reloj
}

// ConstruirPersonalB2 no carga otra raíz ni otra configuración: reutiliza la
// raíz y la audiencia de atestación compartidas que ya acreditó CT y lee la
// configuración vigente con leer_configuracion_interna_v2('personal_b2').
// Cualquier fallo deja B2 fuera sin tocar los proveedores CT.
func ConstruirPersonalB2(ctx context.Context, c ConfiguracionPersonalB2) (*ProveedorAutorizacionPersonalB2, error) {
	if c.Base == nil || c.Base.pdp == nil || len(c.Base.pools) < 4 || c.Base.pools[3] == nil {
		return nil, ErrPersonalB2V3NoDisponible
	}
	return construirPersonalB2(ctx, c.Material, dependenciasPersonalB2{
		pdp: c.Base.pdp, catalogo: c.Base.catalogoMotivos, gobierno: c.Base.gobierno, firmante: c.Base.firmante,
		comprobar: comprobacionMaterialGobiernoV3(c.Base.pools[3], sqlComprobarMaterialB2),
		leer:      consultaLecturaGobiernoV3(c.Base.pools[3], sqlLeerConfiguracionB2),
		fuente:    c.Fuente, reloj: c.Reloj,
	})
}

func construirPersonalB2(ctx context.Context, m MaterialPersonalB2, d dependenciasPersonalB2) (*ProveedorAutorizacionPersonalB2, error) {
	g := d.gobierno
	if ctx == nil || ctx.Err() != nil || m.raiz == nil || interfazNula(d.pdp) || d.catalogo == "" || d.catalogo != m.CatalogoMotivos ||
		g == nil || g.lector == nil || g.coord.AudienciaDespliegue != audienciaAtestacionCTInterna ||
		d.firmante == nil || len(d.firmante.privada) != ed25519.PrivateKeySize || d.firmante.claveID != g.coord.ClaveID ||
		d.firmante.audiencia != g.coord.AudienciaDespliegue || d.comprobar == nil || d.leer == nil || d.fuente == nil || d.reloj == nil {
		return nil, ErrPersonalB2V3NoDisponible
	}
	capacidades := capacidadesPersonalB2(m)
	claves := make([]claveGobiernoV3, 0, len(capacidades))
	for _, c := range capacidades {
		claves = append(claves, claveGobierno(c.material, c.audiencia))
	}
	// Punto de partida: la publicación vigente que CT acaba de validar con la
	// misma raíz. El inventario B2 no fija configuración, así que sigue siendo
	// válido tras cualquier renovación diaria.
	_, vigente, err := g.lector.Leer(ctx)
	if err != nil {
		return nil, ErrPersonalB2V3NoDisponible
	}
	sonda, err := materialGobiernoV3{Claves: claves, Raiz: g.coord}.codificar(vigente)
	if err != nil {
		return nil, ErrPersonalB2V3NoDisponible
	}
	vale, err := d.comprobar(ctx, sonda)
	clear(sonda)
	if err != nil || !vale {
		return nil, ErrPersonalB2V3NoDisponible
	}
	lector, err := nuevoLectorGobiernoV3(ctx, vigente, g.raiz, d.reloj, fuenteGobiernoV3(d.leer, claves, g.coord))
	if err != nil {
		return nil, ErrPersonalB2V3NoDisponible
	}
	// Copia propia de la clave de la raíz compartida: B2 conserva su etiqueta de
	// evidencia y su ciclo de vida, pero firma con la misma raíz y audiencia.
	firmante := &firmantePersonalB2V3{claveID: g.coord.ClaveID, audiencia: g.coord.AudienciaDespliegue, privada: append(ed25519.PrivateKey(nil), d.firmante.privada...), reloj: d.reloj}
	atestador, err := app.NuevoServicioAtestacionesAutorizacionV3(core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: g.coord.ClaveID, Audiencia: g.coord.AudienciaDespliegue}, firmante)
	if err != nil {
		clear(firmante.privada)
		return nil, ErrPersonalB2V3NoDisponible
	}
	p := &ProveedorAutorizacionPersonalB2{fuente: d.fuente, reloj: d.reloj, firmante: firmante, motivos: [8]core.ReferenciaEntradaCatalogo{m.Motivos.Ficha, m.Motivos.Vacantes, m.Motivos.Alta, m.Motivos.Hecho, m.Motivos.CatalogoConsultar, m.Motivos.CatalogoPublicar, m.Motivos.CatalogoRetirar, m.Motivos.Empleados}}
	for i, capacidad := range capacidades {
		emisorCapacidad, e := crearCapacidad(m.raiz, capacidad.material, capacidad.audiencia, d.reloj)
		if e != nil {
			p.Cerrar()
			return nil, ErrPersonalB2V3NoDisponible
		}
		// Instantánea del gobierno antes de cada emisión, como CT interno.
		p.emisores[i] = &emisorMaterialRenovable{lector: lector, pdp: d.pdp, atestador: atestador, capacidad: emisorCapacidad}
	}
	return p, nil
}

type capacidadPersonalB2 struct {
	material  capacidadMaterial
	audiencia string
}

// capacidadesPersonalB2 fija el orden de las ocho audiencias de consumo B2; es
// el mismo que imponen AD3-69 y los descriptores del publicador.
func capacidadesPersonalB2(m MaterialPersonalB2) [8]capacidadPersonalB2 {
	return [8]capacidadPersonalB2{
		{m.V3.Capacidades.Ficha, personal.AudienciaFichaEmpleadoB2},
		{m.V3.Capacidades.Vacantes, personal.AudienciaVacantesB2},
		{m.V3.Capacidades.Alta, personal.AudienciaAltaEmpleadoB2},
		{m.V3.Capacidades.Hecho, personal.AudienciaHechoEmpleadoB2},
		{m.V3.Capacidades.CatalogoConsultar, personal.AudienciaConsultarCatalogoEmpleadoB2},
		{m.V3.Capacidades.CatalogoPublicar, personal.AudienciaPublicarCatalogoEmpleadoB2},
		{m.V3.Capacidades.CatalogoRetirar, personal.AudienciaRetirarCatalogoEmpleadoB2},
		{m.V3.Capacidades.Empleados, personal.AudienciaEmpleadosB2},
	}
}

func interfazNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return r.Kind() == reflect.Pointer && r.IsNil()
}

func (p *ProveedorAutorizacionPersonalB2) AutorizarConsultaRegistroEmpleadoB2(ctx context.Context, m personal.MaterialConsultaRegistroEmpleadoB2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	switch m.Operacion() {
	case "ficha":
		return p.autorizar(ctx, 0, m.Actor(), m.Recurso(), personal.AccionFichaEmpleadoB2, personal.AudienciaFichaEmpleadoB2)
	case "vacantes":
		return p.autorizar(ctx, 1, m.Actor(), m.Recurso(), personal.AccionVacantesB2, personal.AudienciaVacantesB2)
	case "empleados":
		return p.autorizar(ctx, 7, m.Actor(), m.Recurso(), personal.AccionEmpleadosB2, personal.AudienciaEmpleadosB2)
	default:
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ErrPersonalB2V3NoDisponible
	}
}

func (p *ProveedorAutorizacionPersonalB2) AutorizarActoRegistroEmpleadoB2(ctx context.Context, m personal.MaterialActoRegistroEmpleadoB2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	switch m.Tipo() {
	case "alta":
		return p.autorizar(ctx, 2, m.Actor(), m.Recurso(), personal.AccionAltaEmpleadoB2, personal.AudienciaAltaEmpleadoB2)
	case "hecho":
		return p.autorizar(ctx, 3, m.Actor(), m.Recurso(), personal.AccionHechoEmpleadoB2, personal.AudienciaHechoEmpleadoB2)
	default:
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ErrPersonalB2V3NoDisponible
	}
}

// AutorizarCatalogoRegistroEmpleadoB2 mantiene tres concesiones nominales;
// publicar y retirar requieren consumo nuevo dentro de la transacción Personal.
func (p *ProveedorAutorizacionPersonalB2) AutorizarCatalogoRegistroEmpleadoB2(ctx context.Context, m personal.MaterialCatalogoEmpleadoB2) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	switch m.Operacion() {
	case "consultar":
		return p.autorizar(ctx, 4, m.Actor(), m.Recurso(), personal.AccionConsultarCatalogoEmpleadoB2, personal.AudienciaConsultarCatalogoEmpleadoB2)
	case "publicar":
		return p.autorizar(ctx, 5, m.Actor(), m.Recurso(), personal.AccionPublicarCatalogoEmpleadoB2, personal.AudienciaPublicarCatalogoEmpleadoB2)
	case "retirar":
		return p.autorizar(ctx, 6, m.Actor(), m.Recurso(), personal.AccionRetirarCatalogoEmpleadoB2, personal.AudienciaRetirarCatalogoEmpleadoB2)
	default:
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ErrPersonalB2V3NoDisponible
	}
}

func (p *ProveedorAutorizacionPersonalB2) autorizar(ctx context.Context, i int, actor core.ContextoActor, recurso core.RecursoAutorizable, accion, audiencia string) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacio vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil || ctx.Err() != nil || p == nil || p.fuente == nil || p.reloj == nil || p.firmante == nil || i < 0 || i >= len(p.emisores) || p.emisores[i] == nil || actor.Validar() != nil || recurso.Validar() != nil || recurso.ModuloID != "personal" {
		return vacio, ErrPersonalB2V3NoDisponible
	}
	// El puente selló este resultado F1 para la petición. Reconsultar F1 aquí
	// crearía otro ResueltoEn y rompería la ligadura exacta al actor del HTTP.
	base, err := p.fuente.ContextoVinculadoPersonalB2(ctx)
	if err != nil || base.Resultado.Validar() != nil || base.Vinculo.ValidarPara(base.Resultado) != nil || !reflect.DeepEqual(actor, base.Resultado.Contexto) {
		return vacio, ErrPersonalB2V3NoDisponible
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seg.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, ErrPersonalB2V3NoDisponible
	}
	finalidades := [8]string{"consultar_ficha_empleado", "consultar_vacantes", "registrar_empleado", "registrar_hecho_empleado", "consultar_catalogo_empleado", "gobernar_catalogo_empleado", "gobernar_catalogo_empleado", "consultar_empleados"}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: base.Vinculo, ReferenciaMotivo: p.motivos[i], Accion: accion, Recurso: recurso, Finalidad: finalidades[i], Correlacion: correlacion})
	if err != nil {
		return vacio, ErrPersonalB2V3NoDisponible
	}
	decision, confirmacion, exportador, err := p.emisores[i].EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, base.Resultado)
	if err != nil || decision.ValidarPara(solicitud) != nil || exportador == nil {
		return vacio, ErrPersonalB2V3NoDisponible
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, base.Resultado, p.motivos[i], material, audiencia) || ctx.Err() != nil {
		return vacio, ErrPersonalB2V3NoDisponible
	}
	return material, nil
}

var _ personalports.ProveedorAutorizacionRegistroEmpleadoB2 = (*ProveedorAutorizacionPersonalB2)(nil)
var _ personalports.ProveedorAutorizacionActosRegistroEmpleadoB2 = (*ProveedorAutorizacionPersonalB2)(nil)
var _ personalports.ProveedorAutorizacionCatalogosRegistroEmpleadoB2 = (*ProveedorAutorizacionPersonalB2)(nil)

type firmantePersonalB2V3 struct {
	claveID, audiencia string
	privada            ed25519.PrivateKey
	reloj              ct.Reloj
}

func (f *firmantePersonalB2V3) FirmarAtestacionAutorizacionV3(ctx context.Context, s vecports.SolicitudFirmaAtestacionAutorizacionV3) (vecports.ResultadoFirmaAtestacionAutorizacionV3, error) {
	var vacio vecports.ResultadoFirmaAtestacionAutorizacionV3
	if f == nil || ctx == nil || ctx.Err() != nil || len(f.privada) != ed25519.PrivateKeySize || f.reloj == nil {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	c, err := s.Cabecera()
	if err != nil || c.ClaveID != f.claveID || c.Audiencia != f.audiencia {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	mensaje, err := s.Mensaje()
	if err != nil {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	defer clear(mensaje)
	aad, err := confianza.AADExternoAtestacionAutorizacionV3(c.Audiencia)
	if err != nil {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	sobre := gocose.NewSign1Message()
	sobre.Headers.Protected.SetAlgorithm(gocose.AlgorithmEdDSA)
	sobre.Headers.Protected[gocose.HeaderLabelKeyID] = []byte(f.claveID)
	sobre.Payload = append([]byte(nil), mensaje...)
	firmante, err := gocose.NewSigner(gocose.AlgorithmEdDSA, f.privada)
	if err != nil || sobre.Sign(rand.Reader, aad, firmante) != nil {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	sobre.Payload = nil
	sobre.Headers.RawProtected = nil
	sobre.Headers.RawUnprotected = nil
	firma, err := sobre.MarshalCBOR()
	if err != nil {
		return vacio, vecports.ErrFirmaAtestacionNoDisponible
	}
	defer clear(firma)
	huella := sha256.Sum256(mensaje)
	return vecports.NuevoResultadoFirmaAtestacionAutorizacionV3(s, firma, "evidencia:firma:personal:b2:"+hex.EncodeToString(huella[:8]), f.reloj.Ahora())
}
