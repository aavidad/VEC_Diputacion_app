package internactproveedores

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/composicion/gobiernov3lector"
	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	pa "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	pl "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
)

var ErrProveedoresCTNoDisponibles = errors.New("composicion interna: proveedores CT no disponibles")

const audienciaAtestacionCTInterna = "vec:desarrollo:contratacion-temporal:atestacion:v3"

type Configuracion struct {
	Material Material
	Fuente   *internagobierno.FuenteF1
	Reloj    ct.Reloj
}

// Proveedores conserva la propiedad de los pools y la devuelve al montaje.
// No instala SQL, claves, asignaciones ni gobierno al arrancar.
type Proveedores struct {
	Cadena          *inc.CadenaAutorizacionAplicacion
	Detalle         *appct.ServicioConsultaDetalleRRHH
	Planes          []byte
	TernaPlanes     ct.ReferenciaVersionadaPersonalRPT
	FuentePersonal  []byte
	TernaPersonal   fuenteejercicio.TernaEsperada
	pools           []*pgxpool.Pool
	motivos         *pgct.PoolResolucionMotivosRRHHPostgreSQL
	consultas       *pgct.PoolConsultasRRHHPostgreSQL
	firmante        *firmanteV3
	pdp             *app.ServicioAutorizacionSolicitudLigadaV3
	catalogoMotivos string
	gobierno        *gobiernoV3Compartido
}

// gobiernoV3Compartido es la única raíz de atestación del proceso: la carga CT
// desde su material privado y la reutiliza B2. La raíz y sus coordenadas
// quedan fijas; solo la configuración publicada se renueva mediante el lector.
type gobiernoV3Compartido struct {
	raiz   confianza.RaizPublicaAtestacionAutorizacionV3
	coord  raizGobiernoV3
	lector *gobiernov3lector.Lector
}

func (p *Proveedores) Cerrar() {
	if p == nil {
		return
	}
	if p.consultas != nil {
		p.consultas.Cerrar()
		p.consultas = nil
	}
	if p.motivos != nil {
		p.motivos.Cerrar()
		p.motivos = nil
	}
	for _, pool := range p.pools {
		if pool != nil {
			pool.Close()
		}
	}
	p.pools = nil
	if p.firmante != nil {
		clear(p.firmante.privada)
		p.firmante = nil
	}
	p.pdp = nil
	p.gobierno = nil
	p.catalogoMotivos = ""
	clear(p.Planes)
	clear(p.FuentePersonal)
	p.Planes = nil
	p.FuentePersonal = nil
}

func Construir(ctx context.Context, c Configuracion) (Proveedores, error) {
	var vacio Proveedores
	if ctx == nil || ctx.Err() != nil || c.Fuente == nil || c.Reloj == nil || c.Material.raiz == nil {
		return vacio, ErrProveedoresCTNoDisponibles
	}
	m := c.Material
	privada, err := leerConHuella(m.raiz, m.V3.ClaveArchivo, m.V3.ClaveSHA256, ed25519.PrivateKeySize)
	if err != nil || len(privada) != ed25519.PrivateKeySize {
		clear(privada)
		return vacio, ErrProveedoresCTNoDisponibles
	}
	defer clear(privada)
	clave := ed25519.PrivateKey(privada)
	if subtle.ConstantTimeCompare(clave.Public().(ed25519.PublicKey), privada[32:]) != 1 {
		return vacio, ErrProveedoresCTNoDisponibles
	}
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(m.V3.ClaveID, m.V3.ClaveVersion, clave.Public().(ed25519.PublicKey), m.V3.Audiencia, confianza.EstadoClaveAtestacionAutorizacionV3Activa, m.V3.RaizDesde, m.V3.RaizHasta, time.Time{})
	if err != nil || m.V3.Audiencia != audienciaAtestacionCTInterna {
		return vacio, ErrProveedoresCTNoDisponibles
	}
	firmante := &firmanteV3{claveID: m.V3.ClaveID, audiencia: m.V3.Audiencia, privada: append(ed25519.PrivateKey(nil), privada...), reloj: c.Reloj}
	atestador, err := app.NuevoServicioAtestacionesAutorizacionV3(core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: m.V3.ClaveID, Audiencia: m.V3.Audiencia}, firmante)
	if err != nil {
		clear(firmante.privada)
		return vacio, ErrProveedoresCTNoDisponibles
	}
	emisiones, err := crearEmisiones(m, c.Reloj, raiz)
	if err != nil {
		clear(firmante.privada)
		return vacio, ErrProveedoresCTNoDisponibles
	}
	perfiles := []perfilPool{
		{"fuente_autorizacion", "vec_autorizacion_fuente", "vec_autorizacion.obtener_instantanea(text,text)"},
		{"registro_autorizacion", "vec_autorizacion_registro", "vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)"},
		{"motivos_autorizacion", "vec_autorizacion_motivos_evaluador", "vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)"},
		{"gobierno_v3", "vec_autorizacion_atestada_v3_preflight_interno", "vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)"},
	}
	var salida Proveedores
	salida.firmante = firmante
	fallo := func() (Proveedores, error) {
		salida.Cerrar()
		return vacio, ErrProveedoresCTNoDisponibles
	}
	for _, perfil := range perfiles {
		pool, e := abrirPool(ctx, m.Pools[perfil.nombre], perfil)
		if e != nil {
			return fallo()
		}
		salida.pools = append(salida.pools, pool)
	}
	coord, err := coordenadasRaizV3(m.V3.ClaveID, m.V3.ClaveVersion, m.V3.Audiencia, clave.Public().(ed25519.PublicKey))
	if err != nil {
		return fallo()
	}
	anterior := gobiernov3lector.Publicacion{Revision: m.V3.ConfiguracionRef, Secuencia: m.V3.ConfiguracionOrden,
		HuellaSHA256: m.V3.ConfiguracionSHA256, PublicadaEn: m.V3.ConfiguracionPublicada, ExpiraEn: m.V3.ConfiguracionExpira}
	// CT lee con leer_configuracion_interna_v2('ct') (AD3-69): mismo contrato
	// de cinco audiencias que v1, sin la comparación entre escalas
	// revision_gobierno/checkpoint que rechazaba claves recién publicadas.
	// v1 sigue instalada y en el manifiesto hasta su retirada.
	verificador, err := nuevoLectorGobiernoV3(ctx, anterior, raiz, c.Reloj,
		fuenteGobiernoV3(consultaLecturaGobiernoV3(salida.pools[3], sqlLeerConfiguracionCT), clavesGobiernoCT(m), coord))
	if err != nil {
		return fallo()
	}
	salida.gobierno = &gobiernoV3Compartido{raiz: raiz, coord: coord, lector: verificador}
	for _, perfil := range []perfilPool{
		{"motivos_rrhh", "vec_autorizacion_motivos_rrhh_resolutor", "vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)"},
		{"consulta_rrhh", "vec_contratacion_temporal_consultor_rrhh", "vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"},
	} {
		sonda, e := abrirPool(ctx, m.Pools[perfil.nombre], perfil)
		if e != nil {
			return fallo()
		}
		sonda.Close()
	}
	motivosPool, e := pgct.NuevoPoolResolucionMotivosRRHHPostgreSQL(ctx, m.Pools["motivos_rrhh"].DSN, m.Pools["motivos_rrhh"].Login)
	if e != nil {
		return fallo()
	}
	salida.motivos = motivosPool
	consultaPool, e := pgct.NuevoPoolConsultasRRHHPostgreSQL(ctx, m.Pools["consulta_rrhh"].DSN, m.Pools["consulta_rrhh"].Login)
	if e != nil {
		return fallo()
	}
	salida.consultas = consultaPool
	fuente, e := pgvec.NuevoAlmacenAutorizacion(salida.pools[0])
	if e != nil {
		return fallo()
	}
	registro, e := pgvec.NuevoAlmacenAutorizacion(salida.pools[1])
	if e != nil {
		return fallo()
	}
	validador, e := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(salida.pools[2], m.CatalogoMotivos)
	if e != nil {
		return fallo()
	}
	pdp, e := app.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro, validador, c.Reloj, seg.GeneradorReferenciasCriptograficas{}, app.ConfiguracionServicioAutorizacion{VigenciaDecision: 5 * time.Second})
	if e != nil {
		return fallo()
	}
	salida.pdp = pdp
	salida.catalogoMotivos = m.CatalogoMotivos
	salida.Cadena, e = inc.NuevaCadenaAutorizacionAplicacion(pdp, atestador, verificador, emisiones)
	if e != nil {
		return fallo()
	}
	resolutorMotivos, e := pgct.NuevoResolutorMotivoConsultaRRHHPostgreSQL(motivosPool)
	if e != nil {
		return fallo()
	}
	// Las consultas RRHH usan su propio catálogo gobernado de motivos, distinto
	// del de alta/lectura. Como en vec-server, cada PDP valida un único catálogo
	// cerrado: el de la cadena de incorporación y el de cuadro/detalle.
	instanteMotivos := c.Reloj.Ahora()
	motivoCuadro, e := resolutorMotivos.ResolverMotivoCuadroRRHH(ctx, instanteMotivos)
	if e != nil || !core.ReferenciaMotivoAutorizacionV2Valida(motivoCuadro) {
		return fallo()
	}
	motivoDetalle, e := resolutorMotivos.ResolverMotivoDetalleRRHH(ctx, instanteMotivos)
	if e != nil || !core.ReferenciaMotivoAutorizacionV2Valida(motivoDetalle) ||
		motivoDetalle.CatalogoID != motivoCuadro.CatalogoID {
		return fallo()
	}
	validadorConsulta, e := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(salida.pools[2], motivoDetalle.CatalogoID)
	if e != nil {
		return fallo()
	}
	pdpConsulta, e := app.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro, validadorConsulta, c.Reloj, seg.GeneradorReferenciasCriptograficas{}, app.ConfiguracionServicioAutorizacion{VigenciaDecision: 5 * time.Second})
	if e != nil {
		return fallo()
	}
	sesion, e := pgct.NuevaSesionConsultaRRHHPostgreSQL(consultaPool)
	if e != nil {
		return fallo()
	}
	emisorCapacidadCuadro, e := crearCapacidad(m.raiz, m.V3.Capacidades.Cuadro, ct.AudienciaConsumoConsultaCuadroRRHHV3, c.Reloj)
	if e != nil {
		return fallo()
	}
	emisorCapacidadDetalle, e := crearCapacidad(m.raiz, m.V3.Capacidades.Detalle, ct.AudienciaConsumoConsultaDetalleRRHHV3, c.Reloj)
	if e != nil {
		return fallo()
	}
	emisorCuadro := &emisorMaterialRenovable{lector: verificador, pdp: pdpConsulta, atestador: atestador, capacidad: emisorCapacidadCuadro}
	emisorDetalle := &emisorMaterialRenovable{lector: verificador, pdp: pdpConsulta, atestador: atestador, capacidad: emisorCapacidadDetalle}
	emisor, e := ct.NuevoEmisorMaterialConsultaRRHH(resolutorMotivos, seg.GeneradorReferenciasCriptograficas{}, c.Reloj, emisorCuadro, emisorDetalle)
	if e != nil {
		return fallo()
	}
	salida.Detalle, e = appct.NuevoServicioConsultaDetalleRRHH(AutoridadDetalle{Fuente: c.Fuente, Reloj: c.Reloj}, emisor, sesion, c.Reloj)
	if e != nil {
		return fallo()
	}
	salida.Planes, e = leerArchivoPrivado(m.raiz, m.PlanesArchivo, 2<<20)
	if e != nil {
		return fallo()
	}
	salida.FuentePersonal, e = leerArchivoPrivado(m.raiz, m.PersonalArchivo, 2<<20)
	if e != nil {
		return fallo()
	}
	salida.TernaPlanes = m.TernaPlanes
	salida.TernaPersonal = fuenteejercicio.TernaEsperada{Referencia: m.TernaPersonal.Referencia, Version: m.TernaPersonal.Version, HuellaSHA256: m.TernaPersonal.HuellaSHA256}
	if _, e := inc.NuevaFuentePlanesPreparacionV2(salida.Planes, salida.TernaPlanes); e != nil {
		return fallo()
	}
	if _, e := fuenteejercicio.NuevaFuenteEjercicio(salida.FuentePersonal, salida.TernaPersonal); e != nil {
		return fallo()
	}
	// La clave privada retenida sólo sirve al firmante ya construido; el
	// material de arranque original se borra al salir. Nunca se registra.
	return salida, nil
}

func crearEmisiones(m Material, reloj ct.Reloj, raiz confianza.RaizPublicaAtestacionAutorizacionV3) (inc.EmisionesAutoridad, error) {
	var e inc.EmisionesAutoridad
	var err error
	e.Alta.Emisor, err = crearCapacidad(m.raiz, m.V3.Capacidades.Alta, pa.AudienciaAltaEjercicio, reloj)
	if err != nil {
		return inc.EmisionesAutoridad{}, err
	}
	e.Alta.Raiz = raiz
	e.Lectura.Emisor, err = crearCapacidad(m.raiz, m.V3.Capacidades.Lectura, pl.AudienciaV2, reloj)
	if err != nil {
		return inc.EmisionesAutoridad{}, err
	}
	e.Lectura.Raiz = raiz
	e.CT.Emisor, err = crearCapacidad(m.raiz, m.V3.Capacidades.CT, ct.AudienciaConfirmacionIncorporacionV2, reloj)
	if err != nil {
		return inc.EmisionesAutoridad{}, err
	}
	e.CT.Raiz = raiz
	return e, nil
}

func crearCapacidad(raiz *os.Root, m capacidadMaterial, audiencia string, reloj ct.Reloj) (*confianza.EmisorCapacidadesAtestacionAutorizacionV3, error) {
	if raiz == nil || m.ClaveID == "" || m.EmisorID == "" || m.Version == 0 || m.RevisionGobierno == 0 || m.HuellaGobierno == "" || m.Desde.IsZero() || m.Hasta.IsZero() {
		return nil, ErrProveedoresCTNoDisponibles
	}
	b, err := leerConHuella(raiz, m.Archivo, m.SHA256, 256)
	if err != nil {
		return nil, ErrProveedoresCTNoDisponibles
	}
	defer clear(b)
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(m.ClaveID, m.Version, b, m.EmisorID, audiencia, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, m.Desde, m.Hasta, time.Time{}, m.RevisionGobierno, m.HuellaGobierno)
	if err != nil || reloj.Ahora().Before(m.Desde) || !reloj.Ahora().Before(m.Hasta) {
		return nil, ErrProveedoresCTNoDisponibles
	}
	emisor, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	if err != nil {
		return nil, ErrProveedoresCTNoDisponibles
	}
	return emisor, nil
}

// Funciones SQL del rol de preflight. Los consumidores se fijan aquí como
// literales: el llamante nunca aporta audiencias ni otro discriminador.
const (
	sqlLeerConfiguracionCT    = `SELECT vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('ct',$1::jsonb)`
	sqlComprobarMaterialB2    = `SELECT vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2('personal_b2',$1::jsonb)`
	sqlLeerConfiguracionB2    = `SELECT vec_autorizacion_atestada_v3.leer_configuracion_interna_v2('personal_b2',$1::jsonb)`
	maximoRespuestaGobiernoV3 = 4 << 10
	limiteConsultaGobiernoV3  = 5 * time.Second
)

var errGobiernoV3 = errors.New("composicion interna: gobierno V3 no disponible")

type claveGobiernoV3 struct {
	AudienciaConsumo     string `json:"audiencia_consumo"`
	ClaveID              string `json:"clave_id"`
	Version              uint64 `json:"version"`
	RevisionGobierno     uint64 `json:"revision_gobierno"`
	HuellaGobiernoSHA256 string `json:"huella_gobierno_sha256"`
	HuellaSecretoSHA256  string `json:"huella_secreto_sha256"`
	EmisorID             string `json:"emisor_id"`
}

type raizGobiernoV3 struct {
	ClaveID             string `json:"clave_id"`
	Version             uint64 `json:"version"`
	HuellaSPKISHA256    string `json:"huella_spki_sha256"`
	AudienciaDespliegue string `json:"audiencia_despliegue"`
	Suite               string `json:"suite"`
}

type configuracionGobiernoV3 struct {
	Revision                  string `json:"revision"`
	Secuencia                 uint64 `json:"secuencia"`
	HuellaConfiguracionSHA256 string `json:"huella_configuracion_sha256"`
}

// materialGobiernoV3 es el JSON de tres claves que aceptan las sondas v1 y v2.
type materialGobiernoV3 struct {
	Claves        []claveGobiernoV3       `json:"claves"`
	Configuracion configuracionGobiernoV3 `json:"configuracion"`
	Raiz          raizGobiernoV3          `json:"raiz"`
}

func claveGobierno(c capacidadMaterial, audiencia string) claveGobiernoV3 {
	return claveGobiernoV3{audiencia, c.ClaveID, c.Version, c.RevisionGobierno, c.HuellaGobierno, c.SHA256, c.EmisorID}
}

// clavesGobiernoCT sigue el orden cerrado de AD3-50a/53a y de v2('ct').
func clavesGobiernoCT(m Material) []claveGobiernoV3 {
	return []claveGobiernoV3{
		claveGobierno(m.V3.Capacidades.Alta, pa.AudienciaAltaEjercicio),
		claveGobierno(m.V3.Capacidades.Lectura, pl.AudienciaV2),
		claveGobierno(m.V3.Capacidades.CT, ct.AudienciaConfirmacionIncorporacionV2),
		claveGobierno(m.V3.Capacidades.Cuadro, ct.AudienciaConsumoConsultaCuadroRRHHV3),
		claveGobierno(m.V3.Capacidades.Detalle, ct.AudienciaConsumoConsultaDetalleRRHHV3),
	}
}

func coordenadasRaizV3(claveID string, version uint64, audiencia string, publica ed25519.PublicKey) (raizGobiernoV3, error) {
	if claveID == "" || version == 0 || audiencia != audienciaAtestacionCTInterna || len(publica) != ed25519.PublicKeySize {
		return raizGobiernoV3{}, errGobiernoV3
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		return raizGobiernoV3{}, errGobiernoV3
	}
	huella := sha256.Sum256(spki)
	return raizGobiernoV3{ClaveID: claveID, Version: version, HuellaSPKISHA256: hex.EncodeToString(huella[:]),
		AudienciaDespliegue: audiencia, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA}, nil
}

func (m materialGobiernoV3) codificar(p gobiernov3lector.Publicacion) ([]byte, error) {
	m.Claves = append([]claveGobiernoV3(nil), m.Claves...)
	m.Configuracion = configuracionGobiernoV3{Revision: p.Revision, Secuencia: p.Secuencia, HuellaConfiguracionSHA256: p.HuellaSHA256}
	return json.Marshal(m)
}

// consultaGobiernoV3 ejecuta una función del preflight sobre el material JSON
// y devuelve su resultado en bruto. Es la frontera que sustituyen las pruebas.
type consultaGobiernoV3 func(context.Context, []byte) ([]byte, error)

// comprobacionGobiernoV3 ejecuta la sonda booleana de material de emisión.
type comprobacionGobiernoV3 func(context.Context, []byte) (bool, error)

func consultaLecturaGobiernoV3(pool *pgxpool.Pool, sql string) consultaGobiernoV3 {
	return func(ctx context.Context, material []byte) ([]byte, error) {
		if pool == nil || ctx == nil {
			return nil, errGobiernoV3
		}
		sonda, cancelar := context.WithTimeout(ctx, limiteConsultaGobiernoV3)
		defer cancelar()
		var respuesta []byte
		if err := pool.QueryRow(sonda, sql, material).Scan(&respuesta); err != nil {
			return nil, errGobiernoV3
		}
		return respuesta, nil
	}
}

func comprobacionMaterialGobiernoV3(pool *pgxpool.Pool, sql string) comprobacionGobiernoV3 {
	return func(ctx context.Context, material []byte) (bool, error) {
		if pool == nil || ctx == nil {
			return false, errGobiernoV3
		}
		sonda, cancelar := context.WithTimeout(ctx, limiteConsultaGobiernoV3)
		defer cancelar()
		var vigente bool
		if err := pool.QueryRow(sonda, sql, material).Scan(&vigente); err != nil {
			return false, errGobiernoV3
		}
		return vigente, nil
	}
}

// fuenteGobiernoV3 envía siempre las mismas claves y la misma raíz, y como
// configuración la última publicación validada. La base comprueba puntero,
// revocaciones, checkpoint y que la raíz no haya cambiado; el lector comprueba
// después la huella contra la raíz fija. Una rotación de raíz o de HMAC no se
// adopta: exige material nuevo.
func fuenteGobiernoV3(consulta consultaGobiernoV3, claves []claveGobiernoV3, raiz raizGobiernoV3) gobiernov3lector.Fuente {
	base := materialGobiernoV3{Claves: append([]claveGobiernoV3(nil), claves...), Raiz: raiz}
	return func(ctx context.Context, previa gobiernov3lector.Publicacion) (gobiernov3lector.Publicacion, error) {
		if consulta == nil || ctx == nil || len(base.Claves) == 0 {
			return gobiernov3lector.Publicacion{}, errGobiernoV3
		}
		b, err := base.codificar(previa)
		if err != nil {
			return gobiernov3lector.Publicacion{}, errGobiernoV3
		}
		defer clear(b)
		respuesta, err := consulta(ctx, b)
		if err != nil || len(respuesta) == 0 || len(respuesta) > maximoRespuestaGobiernoV3 {
			return gobiernov3lector.Publicacion{}, errGobiernoV3
		}
		defer clear(respuesta)
		var publicada struct {
			Revision     string    `json:"revision"`
			Secuencia    uint64    `json:"secuencia"`
			HuellaSHA256 string    `json:"huella_configuracion_sha256"`
			PublicadaEn  time.Time `json:"publicada_en"`
			ExpiraEn     time.Time `json:"expira_en"`
		}
		d := json.NewDecoder(bytes.NewReader(respuesta))
		d.DisallowUnknownFields()
		if d.Decode(&publicada) != nil || d.Decode(new(any)) != io.EOF {
			return gobiernov3lector.Publicacion{}, errGobiernoV3
		}
		return gobiernov3lector.Publicacion{Revision: publicada.Revision, Secuencia: publicada.Secuencia,
			HuellaSHA256: publicada.HuellaSHA256, PublicadaEn: publicada.PublicadaEn, ExpiraEn: publicada.ExpiraEn}, nil
	}
}

// nuevoLectorGobiernoV3 exige una primera lectura válida antes de devolver el
// lector: un material que el gobierno ya no reconoce no arranca.
func nuevoLectorGobiernoV3(ctx context.Context, anterior gobiernov3lector.Publicacion, raiz confianza.RaizPublicaAtestacionAutorizacionV3, reloj ct.Reloj, fuente gobiernov3lector.Fuente) (*gobiernov3lector.Lector, error) {
	if ctx == nil || ctx.Err() != nil || reloj == nil || fuente == nil {
		return nil, errGobiernoV3
	}
	lector, err := gobiernov3lector.Nuevo(anterior, raiz, reloj, fuente)
	if err != nil {
		return nil, errGobiernoV3
	}
	if _, err := lector.Instantanea(ctx); err != nil {
		return nil, errGobiernoV3
	}
	return lector, nil
}

type perfilPool struct{ nombre, rol, funcion string }

// ACL efectiva de los seis perfiles de este corte. Incluye las concesiones
// historicas propias del grupo, aunque esta ruta solo use una de ellas.
// Cualquier funcion nueva exige versionar este manifiesto y revisar su uso.
func funcionesEsperadasPerfil(p perfilPool) []string {
	const alcance = "vec_contratacion_temporal.alcance_consulta_rrhh_v1"
	const cuadro = "vec_contratacion_temporal.consulta_cuadro_rrhh_v1"
	const detalle = "vec_contratacion_temporal.consulta_detalle_rrhh_v1"
	const firma = "bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea"
	switch p.rol {
	case "vec_autorizacion_fuente":
		return []string{"vec_autorizacion.obtener_instantanea(text,text)"}
	case "vec_autorizacion_registro":
		return []string{
			"vec_autorizacion.registrar_decision_si_vigente(jsonb)",
			"vec_autorizacion.registrar_decision_contexto_actor_v3(bytea,bytea,numeric,numeric)",
			"vec_autorizacion.leer_concesion_historica_contexto_actor_v3(bytea,bytea,numeric,numeric)",
		}
	case "vec_autorizacion_motivos_evaluador":
		return []string{
			"vec_autorizacion.resolver_motivo_autorizacion_v2_historico(text,integer,text,text,timestamptz)",
			"vec_autorizacion.resolver_motivo_cobertura_historico_v1(text,integer,text,text,text,timestamptz)",
		}
	case "vec_autorizacion_atestada_v3_preflight_interno":
		// Estado exacto tras AD3-69: v1 (AD3-50a/53a) sigue instalada, sin
		// consumidor de lectura (solo acredita el pool) hasta su retirada; CT
		// y B2 leen con v2 por consumidor cerrado. Sin AD3-69 falta v2 y el
		// preflight, CT incluida, falla cerrado; no se admite el estado anterior.
		return []string{
			"vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)",
			"vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)",
			"vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(text,jsonb)",
			"vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(text,jsonb)",
		}
	case "vec_autorizacion_motivos_rrhh_resolutor":
		return []string{
			"vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamptz)",
			"vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamptz)",
		}
	case "vec_contratacion_temporal_consultor_rrhh":
		return []string{
			"vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v1(" + alcance + "," + cuadro + "," + firma + ")",
			"vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(" + alcance + "," + detalle + "," + firma + ")",
			"vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(" + alcance + "," + detalle + ",text," + firma + ")",
			"vec_contratacion_temporal.consultar_preparacion_resolucion_v1(" + alcance + "," + detalle + "," + firma + ")",
			"vec_contratacion_temporal.consultar_original_propuesta_rrhh_atestado_v1(" + alcance + "," + detalle + "," + firma + ")",
			"vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v2(" + alcance + "," + cuadro + "," + firma + ")",
			"vec_contratacion_temporal.consultar_cuadro_rrhh_atestado_v3(" + alcance + "," + cuadro + "," + firma + ")",
			"vec_contratacion_temporal.consultar_estadisticas_rrhh_v1(" + alcance + ",text,date,date)",
		}
	}
	return nil
}

const consultaPerfilEfectivo = `
WITH esperadas AS (
  SELECT pg_catalog.to_regprocedure(firma) AS oid
    FROM pg_catalog.unnest($2::text[]) AS firma
), grupo AS (
  SELECT oid FROM pg_catalog.pg_roles WHERE rolname=$1
)
SELECT
  (SELECT count(*)=$3 AND count(oid)=$3 AND
          bool_and(pg_catalog.has_function_privilege(session_user,oid,'EXECUTE')) AND
          bool_and(EXISTS(SELECT 1 FROM pg_catalog.pg_proc p,
                         LATERAL pg_catalog.aclexplode(p.proacl) a, grupo g
                          WHERE p.oid=esperadas.oid AND a.grantee=g.oid
                            AND a.privilege_type='EXECUTE' AND NOT a.is_grantable))
     FROM esperadas)
  AND NOT EXISTS(
     SELECT 1 FROM pg_catalog.pg_proc p
       JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast')
        AND n.nspname NOT LIKE 'pg_temp_%'
        AND n.nspname NOT LIKE 'pg_toast_temp_%'
        AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
        AND NOT EXISTS(SELECT 1 FROM esperadas e WHERE e.oid=p.oid))
  AND NOT EXISTS(
     SELECT 1 FROM pg_catalog.pg_class c
       JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast')
        AND n.nspname NOT LIKE 'pg_temp_%'
        AND n.nspname NOT LIKE 'pg_toast_temp_%'
        AND ((c.relkind IN ('r','p','v','m','f') AND
              (pg_catalog.has_table_privilege(session_user,c.oid,
                 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR
               pg_catalog.has_any_column_privilege(session_user,c.oid,
                 'SELECT,INSERT,UPDATE,REFERENCES')))
             OR (c.relkind='S' AND pg_catalog.has_sequence_privilege(
                    session_user,c.oid,'USAGE,SELECT,UPDATE'))))
  AND NOT EXISTS(
     SELECT 1 FROM pg_catalog.pg_namespace n
      WHERE n.nspname NOT IN ('pg_catalog','information_schema','pg_toast')
        AND n.nspname NOT LIKE 'pg_temp_%'
        AND n.nspname NOT LIKE 'pg_toast_temp_%'
        AND pg_catalog.has_schema_privilege(session_user,n.oid,'CREATE'))
  AND NOT EXISTS(
     SELECT 1 FROM pg_catalog.pg_largeobject_metadata l
      WHERE pg_catalog.has_largeobject_privilege(session_user,l.oid,'SELECT')
         OR pg_catalog.has_largeobject_privilege(session_user,l.oid,'UPDATE'))
  AND NOT pg_catalog.has_database_privilege(session_user,
      pg_catalog.current_database(),'CREATE,TEMPORARY')`

func acreditarPerfilEfectivo(ctx context.Context, con *pgx.Conn, p perfilPool) error {
	funciones := funcionesEsperadasPerfil(p)
	if ctx == nil || con == nil || len(funciones) == 0 {
		return ErrProveedoresCTNoDisponibles
	}
	var exacto bool
	if err := con.QueryRow(ctx, consultaPerfilEfectivo, p.rol, funciones, len(funciones)).Scan(&exacto); err != nil || !exacto {
		return ErrProveedoresCTNoDisponibles
	}
	return nil
}

func abrirPool(ctx context.Context, m PoolMaterial, p perfilPool) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(m.DSN)
	if err != nil || cfg.ConnConfig == nil || cfg.ConnConfig.User != m.Login || m.Login == p.rol || !tlsVerificado(&cfg.ConnConfig.Config) {
		return nil, ErrProveedoresCTNoDisponibles
	}
	cfg.MaxConns = 2
	cfg.MinConns = 0
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.AfterConnect = func(ctx context.Context, con *pgx.Conn) error {
		conexionTLS, ok := con.PgConn().Conn().(*tls.Conn)
		if !ok {
			return ErrProveedoresCTNoDisponibles
		}
		estado := conexionTLS.ConnectionState()
		if !estado.HandshakeComplete || len(estado.VerifiedChains) == 0 || estado.Version < tls.VersionTLS12 {
			return ErrProveedoresCTNoDisponibles
		}
		var login, actual string
		var rol, acl bool
		err := con.QueryRow(ctx, `SELECT session_user::text,current_user::text,
		 COALESCE((SELECT l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb
		   AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
		   AND NOT r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole
		   AND NOT r.rolreplication AND NOT r.rolbypassrls
		   AND (SELECT count(*)=1 AND bool_and(m.roleid=r.oid AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
		          FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)
		   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)
		   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc x WHERE x.proowner=l.oid)
		   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class x WHERE x.relowner=l.oid)
		   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace x WHERE x.nspowner=l.oid)
		   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc x,LATERAL pg_catalog.aclexplode(x.proacl) a WHERE a.grantee=l.oid)
		   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_class x,LATERAL pg_catalog.aclexplode(x.relacl) a WHERE a.grantee=l.oid)
		   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace x,LATERAL pg_catalog.aclexplode(x.nspacl) a WHERE a.grantee=l.oid)
		  FROM pg_catalog.pg_roles l CROSS JOIN pg_catalog.pg_roles r
		  WHERE l.rolname=session_user AND r.rolname=$1),false),
		 COALESCE(pg_catalog.has_function_privilege(session_user,pg_catalog.to_regprocedure($2),'EXECUTE'),false)`, p.rol, p.funcion).Scan(&login, &actual, &rol, &acl)
		if err != nil || login != m.Login || actual != login || !rol || !acl {
			return ErrProveedoresCTNoDisponibles
		}
		return acreditarPerfilEfectivo(ctx, con, p)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, ErrProveedoresCTNoDisponibles
	}
	sonda, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	if pool.Ping(sonda) != nil {
		pool.Close()
		return nil, ErrProveedoresCTNoDisponibles
	}
	return pool, nil
}

func tlsVerificado(cfg *pgconn.Config) bool {
	if cfg == nil || !destinoTLSVerificado(cfg.Host, cfg.TLSConfig) {
		return false
	}
	for _, f := range cfg.Fallbacks {
		if f == nil || !destinoTLSVerificado(f.Host, f.TLSConfig) {
			return false
		}
	}
	return true
}

func destinoTLSVerificado(host string, c *tls.Config) bool {
	if c == nil || c.InsecureSkipVerify || strings.TrimSpace(host) == "" || c.ServerName != host {
		return false
	}
	if c.MinVersion != 0 && c.MinVersion < tls.VersionTLS12 {
		return false
	}
	if c.MaxVersion != 0 && c.MaxVersion < tls.VersionTLS12 {
		return false
	}
	if c.MinVersion != 0 && c.MaxVersion != 0 && c.MinVersion > c.MaxVersion {
		return false
	}
	return c.RootCAs == nil || len(c.RootCAs.Subjects()) > 0
}
