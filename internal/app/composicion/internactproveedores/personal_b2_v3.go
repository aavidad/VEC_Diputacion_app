package internactproveedores

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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

const audienciaAtestacionPersonalB2 = "vec:interno:personal:registro-empleado:atestacion:v3"

type materialPersonalB2V3 struct {
	ClaveID                string    `json:"clave_id"`
	ClaveVersion           uint64    `json:"clave_version"`
	ClaveArchivo           string    `json:"clave_archivo"`
	ClaveSHA256            string    `json:"clave_sha256"`
	Audiencia              string    `json:"audiencia"`
	RaizDesde              time.Time `json:"raiz_desde"`
	RaizHasta              time.Time `json:"raiz_hasta"`
	ConfiguracionRef       string    `json:"configuracion_ref"`
	ConfiguracionOrden     uint64    `json:"configuracion_orden"`
	ConfiguracionPublicada time.Time `json:"configuracion_publicada"`
	ConfiguracionExpira    time.Time `json:"configuracion_expira"`
	ConfiguracionSHA256    string    `json:"configuracion_sha256"`
	Capacidades            struct {
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
	if d.Decode(&m) != nil || d.Decode(new(any)) != io.EOF || m.Version != 3 || m.CatalogoMotivos == "" {
		return MaterialPersonalB2{}, ErrPersonalB2V3NoDisponible
	}
	for _, motivo := range []core.ReferenciaEntradaCatalogo{m.Motivos.Ficha, m.Motivos.Vacantes, m.Motivos.Alta, m.Motivos.Hecho, m.Motivos.CatalogoConsultar, m.Motivos.CatalogoPublicar, m.Motivos.CatalogoRetirar, m.Motivos.Empleados} {
		if !core.ReferenciaMotivoAutorizacionV2Valida(motivo) || motivo.CatalogoID != m.CatalogoMotivos {
			return MaterialPersonalB2{}, ErrPersonalB2V3NoDisponible
		}
	}
	for _, nombre := range []string{m.V3.ClaveArchivo, m.V3.Capacidades.Ficha.Archivo, m.V3.Capacidades.Vacantes.Archivo, m.V3.Capacidades.Alta.Archivo, m.V3.Capacidades.Hecho.Archivo, m.V3.Capacidades.CatalogoConsultar.Archivo, m.V3.Capacidades.CatalogoPublicar.Archivo, m.V3.Capacidades.CatalogoRetirar.Archivo, m.V3.Capacidades.Empleados.Archivo} {
		if !filepath.IsLocal(nombre) || nombre == "." {
			return MaterialPersonalB2{}, ErrPersonalB2V3NoDisponible
		}
	}
	if m.V3.Audiencia != audienciaAtestacionPersonalB2 {
		return MaterialPersonalB2{}, ErrPersonalB2V3NoDisponible
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

type ProveedorAutorizacionPersonalB2 struct {
	fuente   *internagobierno.FuenteF1
	reloj    ct.Reloj
	motivos  [8]core.ReferenciaEntradaCatalogo
	emisores [8]*confianza.EmisorMaterialAutorizacionAtestadaV3
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
	p.emisores = [8]*confianza.EmisorMaterialAutorizacionAtestadaV3{}
	p.fuente = nil
}

func ConstruirPersonalB2(ctx context.Context, c ConfiguracionPersonalB2) (*ProveedorAutorizacionPersonalB2, error) {
	m := c.Material
	if ctx == nil || ctx.Err() != nil || m.raiz == nil || c.Base == nil || c.Base.pdp == nil || len(c.Base.pools) < 4 || c.Base.pools[3] == nil || c.Base.catalogoMotivos != m.CatalogoMotivos || c.Fuente == nil || c.Reloj == nil || m.V3.Audiencia != audienciaAtestacionPersonalB2 {
		return nil, ErrPersonalB2V3NoDisponible
	}
	privada, err := leerConHuella(m.raiz, m.V3.ClaveArchivo, m.V3.ClaveSHA256, ed25519.PrivateKeySize)
	if err != nil || len(privada) != ed25519.PrivateKeySize {
		clear(privada)
		return nil, ErrPersonalB2V3NoDisponible
	}
	defer clear(privada)
	clave := ed25519.PrivateKey(privada)
	if subtle.ConstantTimeCompare(clave.Public().(ed25519.PublicKey), privada[32:]) != 1 {
		return nil, ErrPersonalB2V3NoDisponible
	}
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(m.V3.ClaveID, m.V3.ClaveVersion, clave.Public().(ed25519.PublicKey), m.V3.Audiencia, confianza.EstadoClaveAtestacionAutorizacionV3Activa, m.V3.RaizDesde, m.V3.RaizHasta, time.Time{})
	if err != nil {
		return nil, ErrPersonalB2V3NoDisponible
	}
	config, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(m.V3.ConfiguracionRef, m.V3.ConfiguracionOrden, m.V3.ConfiguracionPublicada, m.V3.ConfiguracionExpira, raiz)
	if err != nil || config.ValidarHuellaSHA256Esperada(m.V3.ConfiguracionSHA256) != nil || !c.Reloj.Ahora().Before(m.V3.ConfiguracionExpira) {
		return nil, ErrPersonalB2V3NoDisponible
	}
	verificador, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(config, c.Reloj)
	if err != nil {
		return nil, ErrPersonalB2V3NoDisponible
	}
	firmante := &firmantePersonalB2V3{claveID: m.V3.ClaveID, audiencia: m.V3.Audiencia, privada: append(ed25519.PrivateKey(nil), privada...), reloj: c.Reloj}
	atestador, err := app.NuevoServicioAtestacionesAutorizacionV3(core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: m.V3.ClaveID, Audiencia: m.V3.Audiencia}, firmante)
	if err != nil {
		clear(firmante.privada)
		return nil, ErrPersonalB2V3NoDisponible
	}
	if sondearGobiernoPersonalB2(ctx, c.Base.pools[3], m, clave.Public().(ed25519.PublicKey)) != nil {
		clear(firmante.privada)
		return nil, ErrPersonalB2V3NoDisponible
	}
	p := &ProveedorAutorizacionPersonalB2{fuente: c.Fuente, reloj: c.Reloj, firmante: firmante, motivos: [8]core.ReferenciaEntradaCatalogo{m.Motivos.Ficha, m.Motivos.Vacantes, m.Motivos.Alta, m.Motivos.Hecho, m.Motivos.CatalogoConsultar, m.Motivos.CatalogoPublicar, m.Motivos.CatalogoRetirar, m.Motivos.Empleados}}
	for i, capacidad := range []struct {
		material  capacidadMaterial
		audiencia string
	}{
		{m.V3.Capacidades.Ficha, personal.AudienciaFichaEmpleadoB2},
		{m.V3.Capacidades.Vacantes, personal.AudienciaVacantesB2},
		{m.V3.Capacidades.Alta, personal.AudienciaAltaEmpleadoB2},
		{m.V3.Capacidades.Hecho, personal.AudienciaHechoEmpleadoB2},
		{m.V3.Capacidades.CatalogoConsultar, personal.AudienciaConsultarCatalogoEmpleadoB2},
		{m.V3.Capacidades.CatalogoPublicar, personal.AudienciaPublicarCatalogoEmpleadoB2},
		{m.V3.Capacidades.CatalogoRetirar, personal.AudienciaRetirarCatalogoEmpleadoB2},
		{m.V3.Capacidades.Empleados, personal.AudienciaEmpleadosB2},
	} {
		emisorCapacidad, e := crearCapacidad(m.raiz, capacidad.material, capacidad.audiencia, c.Reloj)
		if e != nil {
			p.Cerrar()
			return nil, ErrPersonalB2V3NoDisponible
		}
		p.emisores[i], e = confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(c.Base.pdp, atestador, verificador, emisorCapacidad)
		if e != nil {
			p.Cerrar()
			return nil, ErrPersonalB2V3NoDisponible
		}
	}
	return p, nil
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

func sondearGobiernoPersonalB2(ctx context.Context, pool *pgxpool.Pool, m MaterialPersonalB2, publica ed25519.PublicKey) error {
	if ctx == nil || ctx.Err() != nil || pool == nil || len(publica) != ed25519.PublicKeySize {
		return ErrPersonalB2V3NoDisponible
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		return ErrPersonalB2V3NoDisponible
	}
	huella := sha256.Sum256(spki)
	type clave struct {
		AudienciaConsumo     string `json:"audiencia_consumo"`
		ClaveID              string `json:"clave_id"`
		Version              uint64 `json:"version"`
		RevisionGobierno     uint64 `json:"revision_gobierno"`
		HuellaGobiernoSHA256 string `json:"huella_gobierno_sha256"`
		HuellaSecretoSHA256  string `json:"huella_secreto_sha256"`
		EmisorID             string `json:"emisor_id"`
	}
	material := struct {
		Claves        []clave `json:"claves"`
		Configuracion struct {
			Revision                  string `json:"revision"`
			Secuencia                 uint64 `json:"secuencia"`
			HuellaConfiguracionSHA256 string `json:"huella_configuracion_sha256"`
		} `json:"configuracion"`
		Raiz struct {
			ClaveID             string `json:"clave_id"`
			Version             uint64 `json:"version"`
			HuellaSPKISHA256    string `json:"huella_spki_sha256"`
			AudienciaDespliegue string `json:"audiencia_despliegue"`
			Suite               string `json:"suite"`
		} `json:"raiz"`
	}{Claves: make([]clave, 0, 8)}
	for _, c := range []struct {
		m         capacidadMaterial
		audiencia string
	}{
		{m.V3.Capacidades.Ficha, personal.AudienciaFichaEmpleadoB2},
		{m.V3.Capacidades.Vacantes, personal.AudienciaVacantesB2},
		{m.V3.Capacidades.Alta, personal.AudienciaAltaEmpleadoB2},
		{m.V3.Capacidades.Hecho, personal.AudienciaHechoEmpleadoB2},
		{m.V3.Capacidades.CatalogoConsultar, personal.AudienciaConsultarCatalogoEmpleadoB2},
		{m.V3.Capacidades.CatalogoPublicar, personal.AudienciaPublicarCatalogoEmpleadoB2},
		{m.V3.Capacidades.CatalogoRetirar, personal.AudienciaRetirarCatalogoEmpleadoB2},
		{m.V3.Capacidades.Empleados, personal.AudienciaEmpleadosB2},
	} {
		material.Claves = append(material.Claves, clave{c.audiencia, c.m.ClaveID, c.m.Version, c.m.RevisionGobierno, c.m.HuellaGobierno, c.m.SHA256, c.m.EmisorID})
	}
	material.Configuracion.Revision = m.V3.ConfiguracionRef
	material.Configuracion.Secuencia = m.V3.ConfiguracionOrden
	material.Configuracion.HuellaConfiguracionSHA256 = m.V3.ConfiguracionSHA256
	material.Raiz.ClaveID = m.V3.ClaveID
	material.Raiz.Version = m.V3.ClaveVersion
	material.Raiz.HuellaSPKISHA256 = hex.EncodeToString(huella[:])
	material.Raiz.AudienciaDespliegue = m.V3.Audiencia
	material.Raiz.Suite = confianza.SuiteAtestacionAutorizacionV3COSEEdDSA
	b, err := json.Marshal(material)
	if err != nil {
		return ErrPersonalB2V3NoDisponible
	}
	defer clear(b)
	sonda, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	var vigente bool
	if err := pool.QueryRow(sonda, `SELECT vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1($1::jsonb)`, b).Scan(&vigente); err != nil || !vigente {
		return ErrPersonalB2V3NoDisponible
	}
	return nil
}
