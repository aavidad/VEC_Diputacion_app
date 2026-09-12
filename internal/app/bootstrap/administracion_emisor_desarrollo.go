package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/veraison/go-cose"
	"vec-diputacion-granada/config"
	admin "vec-diputacion-granada/internal/modules/administracion"
	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	appvec "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const audienciaAtestacionAdministracionDesarrollo = "vec:desarrollo:administracion:atestacion:v3"
const catalogoMotivosAdministracionDesarrollo = "motivos_autorizacion_administracion"
const organizacionConfiguracionCorreoAdministracionV3 = "organizacion:dipgra"

// Sólo material previamente preparado fuera de Git. La factoría no crea
// identidades, claves, concesiones ni publicaciones de gobierno PostgreSQL.
type archivoEmisorAdministracionDesarrollo struct {
	Version            int                            `json:"version"`
	Autoridad          string                         `json:"autoridad"`
	CuentaRef          string                         `json:"cuenta_ref"`
	CuentaOrdinariaRef string                         `json:"cuenta_ordinaria_ref"`
	PerfilRef          string                         `json:"perfil_ref"`
	PersonaRef         string                         `json:"persona_ref"`
	Motivo             core.ReferenciaEntradaCatalogo `json:"motivo"`
	FuenteDSNFile      string                         `json:"fuente_autorizacion_dsn_file"`
	MotivosDSNFile     string                         `json:"motivos_autorizacion_dsn_file"`
	Firma              struct {
		File       string    `json:"file"`
		ClaveID    string    `json:"clave_id"`
		Version    uint64    `json:"version"`
		SPKISHA256 string    `json:"spki_sha256"`
		Desde      time.Time `json:"desde"`
		Hasta      time.Time `json:"hasta"`
	} `json:"firma"`
	Confianza struct {
		Referencia   string    `json:"referencia"`
		Orden        uint64    `json:"orden"`
		PublicadaEn  time.Time `json:"publicada_en"`
		ExpiraEn     time.Time `json:"expira_en"`
		HuellaSHA256 string    `json:"huella_sha256"`
	} `json:"confianza"`
	Capacidad         archivoCapacidadIncorporacionV2 `json:"capacidad"`
	CapacidadConsulta archivoCapacidadIncorporacionV2 `json:"capacidad_consulta"`
}

type dependenciasEmisorAdministracionDesarrollo struct {
	emisor         *confianza.EmisorMaterialAutorizacionAtestadaV3
	emisorConsulta *confianza.EmisorMaterialAutorizacionAtestadaV3
	pdp            vp.AutorizadorSolicitudLigadaV3
	motivo         core.ReferenciaEntradaCatalogo
	cerrar         func()
}

func nuevasDependenciasEmisorAdministracionDesarrollo(ctx context.Context, cfg config.Config, identidad *identidadAdministracionDesarrollo, registro *pgxpool.Pool, reloj vp.Reloj) (*dependenciasEmisorAdministracionDesarrollo, error) {
	f := ErrConfiguracionCorreoAdministracionNoDisponible
	if !cfg.AdministracionPostgreSQL.Configurada() && identidad == nil {
		return nil, nil
	}
	if ctx == nil || ctx.Err() != nil || !cfg.DevelopmentEnabledByDoubleKey() || validarRedLocalDesarrollo(cfg) != nil || identidad == nil || registro == nil || dependenciaAdministracionNula(reloj) || cfg.AdministracionPostgreSQL.Validar() != nil {
		return nil, f
	}
	c, raiz, err := leerEmisorAdministracionDesarrollo(cfg.DevelopmentMaterialDir)
	if err != nil {
		return nil, f
	}
	defer raiz.Close()
	if c.CuentaRef != identidad.cuentaRef || c.CuentaOrdinariaRef != identidad.cuentaOrdinariaRef || c.PerfilRef != identidad.perfilRef || c.PersonaRef != identidad.personaRef {
		return nil, f
	}
	semilla, err := leerArchivoIncorporacionV2(raiz, c.Firma.File, 32)
	if err != nil {
		return nil, f
	}
	defer borrarBytes(semilla)
	secreto, err := leerArchivoIncorporacionV2(raiz, c.Capacidad.File, 32)
	if err != nil {
		return nil, f
	}
	defer borrarBytes(secreto)
	secretoConsulta, err := leerArchivoIncorporacionV2(raiz, c.CapacidadConsulta.File, 32)
	if err != nil {
		return nil, f
	}
	defer borrarBytes(secretoConsulta)
	capacidadesConsulta, err := materialCapacidadConsultaAdministracionDesarrollo(c, semilla, secreto, secretoConsulta, reloj)
	if err != nil {
		return nil, f
	}
	atestador, verificador, capacidades, firmante, err := materialEmisorAdministracionDesarrollo(c, semilla, secreto, reloj)
	if err != nil {
		return nil, f
	}
	completa := false
	defer func() {
		if !completa {
			firmante.cerrar()
		}
	}()
	dsnCorreo, _ := cfg.AdministracionPostgreSQL.DSNCorreo()
	dsnRegistro, _ := cfg.AdministracionPostgreSQL.DSNRegistroAutorizacion()
	dsnIdentidad, dsnRevalidacion, dsnContexto, _ := cfg.AdministracionPostgreSQL.DSNIdentidad()
	usuarios := map[string]bool{}
	for _, dsn := range []string{dsnCorreo, dsnRegistro, dsnIdentidad, dsnRevalidacion, dsnContexto} {
		pc, e := pgxpool.ParseConfig(dsn)
		if e != nil || pc.ConnConfig.User == "" || usuarios[pc.ConnConfig.User] {
			return nil, f
		}
		usuarios[pc.ConnConfig.User] = true
	}
	pcRegistro, _ := pgxpool.ParseConfig(dsnRegistro)
	real := registro.Config().ConnConfig
	if real.User != pcRegistro.ConnConfig.User || real.Host != pcRegistro.ConnConfig.Host || real.Port != pcRegistro.ConnConfig.Port || real.Database != pcRegistro.ConnConfig.Database {
		return nil, f
	}
	pools := make([]*pgxpool.Pool, 0, 2)
	var una sync.Once
	cerrar := func() {
		una.Do(func() {
			firmante.cerrar()
			for _, p := range pools {
				p.Close()
			}
		})
	}
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	for n, archivo := range []string{c.FuenteDSNFile, c.MotivosDSNFile} {
		b, e := leerArchivoIncorporacionV2(raiz, archivo, 16<<10)
		if e != nil {
			return nil, f
		}
		dsn := strings.TrimSpace(string(b))
		borrarBytes(b)
		pc, e := pgxpool.ParseConfig(dsn)
		if e != nil || pc.ConnConfig.User == "" || usuarios[pc.ConnConfig.User] {
			return nil, f
		}
		rol := []string{"vec_autorizacion_fuente", "vec_autorizacion_motivos_evaluador"}[n]
		pool, usuario, e := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, dsn, "vec-admin-desarrollo-emisor-"+rol, rol)
		if e != nil {
			return nil, f
		}
		pools = append(pools, pool)
		if usuarios[usuario] {
			return nil, f
		}
		usuarios[usuario] = true
	}
	fuente, err := pgvec.NuevoAlmacenAutorizacion(pools[0])
	if err != nil {
		return nil, f
	}
	cas, err := pgvec.NuevoAlmacenAutorizacion(registro)
	if err != nil {
		return nil, f
	}
	motivos, err := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools[1], c.Motivo.CatalogoID)
	if err != nil {
		return nil, f
	}
	pdp, err := appvec.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, cas, cas, motivos, reloj, seg.GeneradorReferenciasCriptograficas{}, appvec.ConfiguracionServicioAutorizacion{})
	if err != nil {
		return nil, f
	}
	nominal := *identidad
	nominal.identidad.principal = clonarPrincipalDesarrollo(identidad.identidad.principal)
	autoridad := &autoridadEmisorAdministracionDesarrollo{delegado: pdp, identidad: nominal, motivo: c.Motivo, reloj: reloj}
	emisor, emisorConsulta, err := nuevosEmisoresMaterialAdministracionDesarrollo(autoridad, atestador, verificador, capacidades, capacidadesConsulta)
	if err != nil {
		return nil, f
	}
	completa = true
	return &dependenciasEmisorAdministracionDesarrollo{emisor: emisor, emisorConsulta: emisorConsulta, pdp: autoridad, motivo: c.Motivo, cerrar: cerrar}, nil
}

// La autoridad común conserva la identidad y el PDP. Cada emisor recibe una
// copia restringida a una única acción y su capacidad de consumo ya gobernada.
func nuevosEmisoresMaterialAdministracionDesarrollo(autoridad *autoridadEmisorAdministracionDesarrollo, atestador *appvec.ServicioAtestacionesAutorizacionV3, verificador *confianza.ServicioConfianzaAtestacionAutorizacionV3, capacidades, capacidadesConsulta *confianza.EmisorCapacidadesAtestacionAutorizacionV3) (*confianza.EmisorMaterialAutorizacionAtestadaV3, *confianza.EmisorMaterialAutorizacionAtestadaV3, error) {
	if autoridad == nil {
		return nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	escritura, lectura := *autoridad, *autoridad
	escritura.accionPermitida = accionConfiguracionCorreoAdministracionV3
	lectura.accionPermitida = adminapp.AccionConsultarConfiguracionCorreo
	emisor, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(&escritura, atestador, verificador, capacidades)
	if err != nil {
		return nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	emisorConsulta, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(&lectura, atestador, verificador, capacidadesConsulta)
	if err != nil {
		return nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	return emisor, emisorConsulta, nil
}

func leerEmisorAdministracionDesarrollo(directorio string) (archivoEmisorAdministracionDesarrollo, *os.Root, error) {
	var c archivoEmisorAdministracionDesarrollo
	f := ErrConfiguracionCorreoAdministracionNoDisponible
	base := filepath.Join(directorio, "administracion")
	if !filepath.IsAbs(directorio) || dentroDeRepositorioGit(base) || validarArbolMaterialDesarrollo(base) != nil {
		return c, nil, f
	}
	antes, err := os.Lstat(base)
	if err != nil || !antes.IsDir() {
		return c, nil, f
	}
	raiz, err := os.OpenRoot(base)
	if err != nil {
		return c, nil, f
	}
	ok := false
	defer func() {
		if !ok {
			raiz.Close()
		}
	}()
	despues, err := raiz.Stat(".")
	if err != nil || !os.SameFile(antes, despues) {
		return c, nil, f
	}
	b, err := leerArchivoIncorporacionV2(raiz, "emisor-v3.json", 32<<10)
	if err != nil {
		return c, nil, f
	}
	defer borrarBytes(b)
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if validarClavesJSONUnicas(b) != nil || d.Decode(&c) != nil || !errors.Is(d.Decode(new(any)), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa || !core.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) || c.Motivo.CatalogoID != catalogoMotivosAdministracionDesarrollo {
		return archivoEmisorAdministracionDesarrollo{}, nil, f
	}
	rutas := map[string]bool{}
	for _, ruta := range []string{c.FuenteDSNFile, c.MotivosDSNFile, c.Firma.File, c.Capacidad.File, c.CapacidadConsulta.File} {
		if !filepath.IsLocal(ruta) || rutas[filepath.Clean(ruta)] {
			return archivoEmisorAdministracionDesarrollo{}, nil, f
		}
		rutas[filepath.Clean(ruta)] = true
	}
	if c.CapacidadConsulta.ClaveID == c.Capacidad.ClaveID || c.CapacidadConsulta.SHA256 == c.Capacidad.SHA256 || c.CapacidadConsulta.HuellaGobierno == c.Capacidad.HuellaGobierno {
		return archivoEmisorAdministracionDesarrollo{}, nil, f
	}
	ok = true
	return c, raiz, nil
}

func materialEmisorAdministracionDesarrollo(c archivoEmisorAdministracionDesarrollo, semilla, secreto []byte, reloj vp.Reloj) (*appvec.ServicioAtestacionesAutorizacionV3, *confianza.ServicioConfianzaAtestacionAutorizacionV3, *confianza.EmisorCapacidadesAtestacionAutorizacionV3, *firmanteEmisorAdministracionDesarrollo, error) {
	fallo := func() (*appvec.ServicioAtestacionesAutorizacionV3, *confianza.ServicioConfianzaAtestacionAutorizacionV3, *confianza.EmisorCapacidadesAtestacionAutorizacionV3, *firmanteEmisorAdministracionDesarrollo, error) {
		return nil, nil, nil, nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	if dependenciaAdministracionNula(reloj) || len(semilla) != 32 || len(secreto) != 32 || bytes.Equal(semilla, make([]byte, 32)) || bytes.Equal(secreto, make([]byte, 32)) || bytes.Equal(semilla, secreto) || !strings.HasPrefix(c.Firma.ClaveID, "clave:atestacion:administracion:") || !strings.HasPrefix(c.Capacidad.ClaveID, "clave:capacidad:administracion:") || !strings.HasPrefix(c.Capacidad.EmisorID, "emisor:administracion:") || !strings.HasPrefix(c.Confianza.Referencia, "confianza:atestacion:administracion:") {
		return fallo()
	}
	ahora := reloj.Ahora()
	if ahora.Before(c.Firma.Desde) || !ahora.Before(c.Firma.Hasta) || ahora.Before(c.Capacidad.Desde) || !ahora.Before(c.Capacidad.Hasta) || ahora.Before(c.Confianza.PublicadaEn) || !ahora.Before(c.Confianza.ExpiraEn) {
		return fallo()
	}
	privada := ed25519.NewKeyFromSeed(semilla)
	ok := false
	defer func() {
		if !ok {
			borrarBytes(privada)
		}
	}()
	publica := privada.Public().(ed25519.PublicKey)
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		return fallo()
	}
	h := sha256.Sum256(spki)
	if hex.EncodeToString(h[:]) != c.Firma.SPKISHA256 {
		return fallo()
	}
	h = sha256.Sum256(secreto)
	if hex.EncodeToString(h[:]) != c.Capacidad.SHA256 {
		return fallo()
	}
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(c.Firma.ClaveID, c.Firma.Version, publica, audienciaAtestacionAdministracionDesarrollo, confianza.EstadoClaveAtestacionAutorizacionV3Activa, c.Firma.Desde, c.Firma.Hasta, time.Time{})
	if err != nil {
		return fallo()
	}
	configuracion, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(c.Confianza.Referencia, c.Confianza.Orden, c.Confianza.PublicadaEn, c.Confianza.ExpiraEn, raiz)
	if err != nil {
		return fallo()
	}
	huella, err := configuracion.HuellaSHA256ParaGobierno()
	if err != nil || huella != c.Confianza.HuellaSHA256 {
		return fallo()
	}
	verificador, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(configuracion, reloj)
	if err != nil {
		return fallo()
	}
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(c.Capacidad.ClaveID, c.Capacidad.Version, secreto, c.Capacidad.EmisorID, audienciaConfiguracionCorreoAdministracionV3, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, c.Capacidad.Desde, c.Capacidad.Hasta, time.Time{}, c.Capacidad.RevisionGobierno, c.Capacidad.HuellaGobierno)
	if err != nil {
		return fallo()
	}
	capacidades, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	if err != nil {
		return fallo()
	}
	firmante := &firmanteEmisorAdministracionDesarrollo{mu: &sync.RWMutex{}, claveID: c.Firma.ClaveID, privada: privada, reloj: reloj}
	atestador, err := appvec.NuevoServicioAtestacionesAutorizacionV3(core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: c.Firma.ClaveID, Audiencia: audienciaAtestacionAdministracionDesarrollo}, firmante)
	if err != nil {
		return fallo()
	}
	ok = true
	return atestador, verificador, capacidades, firmante, nil
}

// Consume material de consulta suministrado expresamente. La revisión y huella
// de gobierno se conservan; esta carga no publica ni deriva una concesión nueva.
func materialCapacidadConsultaAdministracionDesarrollo(c archivoEmisorAdministracionDesarrollo, semilla, secretoEscritura, secretoConsulta []byte, reloj vp.Reloj) (*confianza.EmisorCapacidadesAtestacionAutorizacionV3, error) {
	f := ErrConfiguracionCorreoAdministracionNoDisponible
	capacidad := c.CapacidadConsulta
	if dependenciaAdministracionNula(reloj) || len(secretoConsulta) != 32 || bytes.Equal(secretoConsulta, make([]byte, 32)) || bytes.Equal(secretoConsulta, semilla) || bytes.Equal(secretoConsulta, secretoEscritura) || capacidad.ClaveID == c.Capacidad.ClaveID || capacidad.SHA256 == c.Capacidad.SHA256 || capacidad.HuellaGobierno == c.Capacidad.HuellaGobierno || !strings.HasPrefix(capacidad.ClaveID, "clave:capacidad:administracion:") || !strings.HasPrefix(capacidad.EmisorID, "emisor:administracion:") {
		return nil, f
	}
	h := sha256.Sum256(secretoConsulta)
	ahora := reloj.Ahora()
	if hex.EncodeToString(h[:]) != capacidad.SHA256 || ahora.Before(capacidad.Desde) || !ahora.Before(capacidad.Hasta) {
		return nil, f
	}
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(capacidad.ClaveID, capacidad.Version, secretoConsulta, capacidad.EmisorID, audienciaConsultaConfiguracionCorreoAdministracionV3, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, capacidad.Desde, capacidad.Hasta, time.Time{}, capacidad.RevisionGobierno, capacidad.HuellaGobierno)
	if err != nil {
		return nil, f
	}
	emisor, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	if err != nil {
		return nil, f
	}
	return emisor, nil
}

type autoridadEmisorAdministracionDesarrollo struct {
	delegado        vp.AutorizadorSolicitudLigadaV3
	identidad       identidadAdministracionDesarrollo
	motivo          core.ReferenciaEntradaCatalogo
	reloj           vp.Reloj
	accionPermitida string
}

func (a *autoridadEmisorAdministracionDesarrollo) ExigirSolicitudLigadaV3(ctx context.Context, solicitud core.SolicitudAutorizacionLigadaV3, resultado core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	fallo := func() (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	if a == nil || dependenciaAdministracionNula(a.delegado) || dependenciaAdministracionNula(a.reloj) {
		return fallo()
	}
	d, err := solicitud.Datos()
	if err != nil || !contextoPermisoAdministracionGobernadoValido(ctx, d.VinculoAutenticacionActor, resultado, a.reloj.Ahora()) || d.ReferenciaMotivo != a.motivo || resultado.Contexto.Principal.ID != a.identidad.personaRef || resultado.Contexto.PersonaRef != a.identidad.personaRef || resultado.Contexto.PerfilActivoRef != a.identidad.perfilRef || resultado.Contexto.Instantanea.CuentaRef != a.identidad.cuentaRef || d.Recurso.Referencia != referenciaConfiguracionCorreoAdministracionV3 || d.Recurso.ModuloID != admin.ModuleID || d.Recurso.Tipo != tipoRecursoConfiguracionCorreoAdministracion || d.Finalidad != finalidadConfiguracionCorreoAdministracionV3 || len(d.Recurso.Ambitos) != 1 || d.Recurso.Ambitos["organizacion_ref"] != organizacionConfiguracionCorreoAdministracionV3 {
		return fallo()
	}
	accionPermitida := a.accionPermitida
	if accionPermitida == "" {
		accionPermitida = admin.PermissionIntegrationsManage
	}
	if d.Accion != accionPermitida || (accionPermitida != admin.PermissionIntegrationsManage && accionPermitida != accionConfiguracionCorreoAdministracionV3 && accionPermitida != adminapp.AccionConsultarConfiguracionCorreo) {
		return fallo()
	}
	if d.Accion == admin.PermissionIntegrationsManage && len(d.Recurso.Atributos) != 0 {
		return fallo()
	}
	if d.Accion == accionConfiguracionCorreoAdministracionV3 || d.Accion == adminapp.AccionConsultarConfiguracionCorreo {
		capacidad, ok := capacidadAdministracionDesdeContexto(ctx)
		if !ok || capacidad.ruta != "/api/vec/administracion/configuracion-correo" || (d.Accion == accionConfiguracionCorreoAdministracionV3 && capacidad.metodo != "PUT") || (d.Accion == adminapp.AccionConsultarConfiguracionCorreo && capacidad.metodo != "GET") {
			return fallo()
		}
		h := d.Recurso.Atributos["material_sha256"]
		b, e := hex.DecodeString(h)
		if e != nil || len(b) != 32 || hex.EncodeToString(b) != h || h == strings.Repeat("0", 64) || len(d.Recurso.Atributos) != 1 {
			return fallo()
		}
	}
	return a.delegado.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
}

type firmanteEmisorAdministracionDesarrollo struct {
	mu      *sync.RWMutex
	claveID string
	privada ed25519.PrivateKey
	reloj   vp.Reloj
}

func (f *firmanteEmisorAdministracionDesarrollo) cerrar() {
	if f == nil || f.mu == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	borrarBytes(f.privada)
	f.privada = nil
}
func (firmanteEmisorAdministracionDesarrollo) String() string {
	return "firmanteAdministracion{redactado}"
}
func (firmanteEmisorAdministracionDesarrollo) GoString() string {
	return "firmanteAdministracion{redactado}"
}
func (firmanteEmisorAdministracionDesarrollo) MarshalJSON() ([]byte, error) {
	return []byte(`{"redactado":true}`), nil
}

func (f *firmanteEmisorAdministracionDesarrollo) FirmarAtestacionAutorizacionV3(ctx context.Context, s vp.SolicitudFirmaAtestacionAutorizacionV3) (vp.ResultadoFirmaAtestacionAutorizacionV3, error) {
	if f == nil || f.mu == nil || dependenciaAdministracionNula(f.reloj) || ctx == nil || ctx.Err() != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, vp.ErrFirmaAtestacionNoDisponible
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	c, err := s.Cabecera()
	if err != nil || len(f.privada) != ed25519.PrivateKeySize || c.ClaveID != f.claveID || c.Audiencia != audienciaAtestacionAdministracionDesarrollo {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, vp.ErrFirmaAtestacionNoDisponible
	}
	mensaje, err := s.Mensaje()
	if err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, vp.ErrFirmaAtestacionNoDisponible
	}
	defer borrarBytes(mensaje)
	aad, err := confianza.AADExternoAtestacionAutorizacionV3(c.Audiencia)
	if err != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, vp.ErrFirmaAtestacionNoDisponible
	}
	sobre := cose.NewSign1Message()
	sobre.Headers.Protected.SetAlgorithm(cose.AlgorithmEdDSA)
	sobre.Headers.Protected[cose.HeaderLabelKeyID] = []byte(f.claveID)
	sobre.Payload = mensaje
	signer, err := cose.NewSigner(cose.AlgorithmEdDSA, f.privada)
	if err != nil || sobre.Sign(rand.Reader, aad, signer) != nil {
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, vp.ErrFirmaAtestacionNoDisponible
	}
	sobre.Payload, sobre.Headers.RawProtected, sobre.Headers.RawUnprotected = nil, nil, nil
	firma, err := sobre.MarshalCBOR()
	if err != nil || ctx.Err() != nil {
		borrarBytes(firma)
		return vp.ResultadoFirmaAtestacionAutorizacionV3{}, vp.ErrFirmaAtestacionNoDisponible
	}
	defer borrarBytes(firma)
	h := sha256.Sum256(mensaje)
	return vp.NuevoResultadoFirmaAtestacionAutorizacionV3(s, firma, "evidencia:firma:administracion:desarrollo:"+hex.EncodeToString(h[:8]), f.reloj.Ahora())
}
