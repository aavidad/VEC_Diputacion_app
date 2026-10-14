package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
	meritosports "vec-diputacion-granada/internal/modules/meritos/ports"
	"vec-diputacion-granada/internal/vec/adapters/analisiscontenido"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const rutaMaterialSubidaDocumentosExterna = "externo/documentos/subidas.json"

const audienciaSubirComprobanteM4 = "vec_documentos.subida_local.acto_modulo.v1"
const audienciaDescargarComprobanteM4 = "vec_documentos.subida_local.descarga_propia.v1"

// La cuenta y el perfil son los de Méritos ya acreditados para esta petición.
// Documentos local conserva sus LOGIN de datos y frontera, sin otro PDP.
type configuracionSubidaDocumentosExterna struct {
	Version                 int                                  `json:"version"`
	DSNDatos                string                               `json:"dsn_datos"`
	DSNFrontera             string                               `json:"dsn_frontera"`
	Directorio              string                               `json:"directorio"`
	RetencionMinimaDias     *int64                               `json:"retencion_minima_dias"`
	ClaveCorrelacionArchivo string                               `json:"clave_correlacion_archivo"`
	ClaveSeudonimoArchivo   string                               `json:"clave_seudonimo_archivo"`
	ClaveSolicitudArchivo   string                               `json:"clave_solicitud_archivo"`
	Analizador              configuracionAnalizadorSubidaExterna `json:"analizador"`
}

type configuracionAnalizadorSubidaExterna struct {
	Endpoint        string `json:"endpoint"`
	NombreServidor  string `json:"nombre_servidor"`
	RutaCA          string `json:"ruta_ca"`
	RutaCertificado string `json:"ruta_certificado"`
	RutaClave       string `json:"ruta_clave"`
	ConectorID      string `json:"conector_id"`
}

func (c configuracionAnalizadorSubidaExterna) cliente() analisiscontenido.ConfiguracionCliente {
	return analisiscontenido.ConfiguracionCliente{Endpoint: c.Endpoint, NombreServidor: c.NombreServidor,
		RutaCA: c.RutaCA, RutaCertificado: c.RutaCertificado, RutaClave: c.RutaClave,
		ConectorID: c.ConectorID}
}

func leerConfiguracionSubidaDocumentosExterna(materialDir string) (configuracionSubidaDocumentosExterna, error) {
	var vacia configuracionSubidaDocumentosExterna
	if materialDir == "" {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	contenido, err := leerFicheroMaterialSeguro(filepath.Join(materialDir, rutaMaterialSubidaDocumentosExterna), 128<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	defer borrarBytes(contenido)
	decoder := json.NewDecoder(bytes.NewReader(contenido))
	decoder.DisallowUnknownFields()
	var c configuracionSubidaDocumentosExterna
	var extra any
	if decoder.Decode(&c) != nil || !errors.Is(decoder.Decode(&extra), io.EOF) ||
		c.Version != 1 || strings.TrimSpace(c.DSNDatos) == "" ||
		strings.TrimSpace(c.DSNDatos) != c.DSNDatos || strings.TrimSpace(c.DSNFrontera) == "" ||
		strings.TrimSpace(c.DSNFrontera) != c.DSNFrontera || !directorioLocalSubidaSeguro(c.Directorio) ||
		c.RetencionMinimaDias == nil || *c.RetencionMinimaDias < 0 ||
		!nombreClaveSubidaDocumentosValido(c.ClaveCorrelacionArchivo) ||
		!nombreClaveSubidaDocumentosValido(c.ClaveSeudonimoArchivo) ||
		!nombreClaveSubidaDocumentosValido(c.ClaveSolicitudArchivo) ||
		c.ClaveCorrelacionArchivo == c.ClaveSeudonimoArchivo ||
		c.ClaveCorrelacionArchivo == c.ClaveSolicitudArchivo ||
		c.ClaveSeudonimoArchivo == c.ClaveSolicitudArchivo ||
		c.Analizador.Endpoint == "" || c.Analizador.NombreServidor == "" ||
		c.Analizador.RutaCA == "" || c.Analizador.RutaCertificado == "" ||
		c.Analizador.RutaClave == "" || c.Analizador.ConectorID == "" {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	return c, nil
}

func directorioLocalSubidaSeguro(ruta string) bool {
	if ruta == "" || !filepath.IsAbs(ruta) || filepath.Clean(ruta) != ruta {
		return false
	}
	info, err := os.Lstat(ruta)
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 && info.Mode().Perm()&0o077 == 0
}

func nombreClaveSubidaDocumentosValido(nombre string) bool {
	return nombre != "" && !filepath.IsAbs(nombre) && filepath.Clean(nombre) == nombre &&
		filepath.Base(nombre) == nombre && !strings.HasPrefix(nombre, "..") &&
		!strings.ContainsRune(nombre, '\\')
}

type clavesSubidaDocumentosExterna struct{ correlacion, seudonimo, solicitud [32]byte }

func (clavesSubidaDocumentosExterna) String() string   { return "claves_subida_local_redactadas" }
func (clavesSubidaDocumentosExterna) GoString() string { return "claves_subida_local_redactadas" }
func (c *clavesSubidaDocumentosExterna) borrar() {
	if c != nil {
		*c = clavesSubidaDocumentosExterna{}
	}
}

func leerClavesSubidaDocumentosExterna(materialDir string,
	c configuracionSubidaDocumentosExterna) (clavesSubidaDocumentosExterna, error) {
	var vacia clavesSubidaDocumentosExterna
	if materialDir == "" || !nombreClaveSubidaDocumentosValido(c.ClaveCorrelacionArchivo) ||
		!nombreClaveSubidaDocumentosValido(c.ClaveSeudonimoArchivo) ||
		!nombreClaveSubidaDocumentosValido(c.ClaveSolicitudArchivo) ||
		c.ClaveCorrelacionArchivo == c.ClaveSeudonimoArchivo ||
		c.ClaveCorrelacionArchivo == c.ClaveSolicitudArchivo ||
		c.ClaveSeudonimoArchivo == c.ClaveSolicitudArchivo {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	base := filepath.Join(materialDir, "externo", "documentos")
	correlacion, err := leerSecreto32Desarrollo(filepath.Join(base, c.ClaveCorrelacionArchivo))
	if err != nil {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	seudonimo, err := leerSecreto32Desarrollo(filepath.Join(base, c.ClaveSeudonimoArchivo))
	if err != nil {
		correlacion = [32]byte{}
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	solicitud, err := leerSecreto32Desarrollo(filepath.Join(base, c.ClaveSolicitudArchivo))
	if err != nil || !seguridad.RaicesHMACSubidaLocalDistintas(correlacion[:], seudonimo[:], solicitud[:]) {
		correlacion, seudonimo, solicitud = [32]byte{}, [32]byte{}, [32]byte{}
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	return clavesSubidaDocumentosExterna{correlacion, seudonimo, solicitud}, nil
}

func nuevasClavesAplicacionSubidaLocal(c *clavesSubidaDocumentosExterna) (
	docports.DerivadorCorrelacionSubidaLocal,
	vecports.SeudonimizadorSujetoAlmacen,
	vecports.SelladorSolicitudCargaDocumental, error) {
	if c == nil {
		return nil, nil, nil, docports.ErrSubidaLocalNoDisponible
	}
	defer c.borrar()
	correlaciones, err := docapp.NuevoDerivadorCorrelacionSubidaLocalHMAC(c.correlacion[:])
	if err != nil {
		return nil, nil, nil, docports.ErrSubidaLocalNoDisponible
	}
	seudonimos, err := seguridad.NuevoSeudonimizadorSujetoSubidaLocal(c.seudonimo[:])
	if err != nil {
		return nil, nil, nil, docports.ErrSubidaLocalNoDisponible
	}
	sellador, err := seguridad.NuevoSelladorSolicitudSubidaLocal(c.solicitud[:])
	if err != nil {
		return nil, nil, nil, docports.ErrSubidaLocalNoDisponible
	}
	return correlaciones, seudonimos, sellador, nil
}

type poolsSubidaDocumentosExterna struct {
	datos, frontera *pgxpool.Pool
	cerrar          func()
}

func abrirPoolsSubidaDocumentosExterna(ctx context.Context, c configuracionSubidaDocumentosExterna,
	topologia topologiaPostgreSQLPreferenciasUsuarios,
	externa *autoridadPreferenciasUsuariosDesarrollo) (poolsSubidaDocumentosExterna, error) {
	var vacio poolsSubidaDocumentosExterna
	if ctx == nil || ctx.Err() != nil || externa == nil || externa.base == nil {
		return vacio, docports.ErrSubidaLocalNoDisponible
	}
	entradas := [...]struct{ dsn, login, rol string }{
		{c.DSNDatos, "vec_externo_documentos_datos", "vec_documentos_subida_datos"},
		{c.DSNFrontera, "vec_externo_documentos_frontera", "vec_documentos_frontera_externa"},
	}
	var abiertos []*pgxpool.Pool
	var unaVez sync.Once
	cerrar := func() {
		unaVez.Do(func() {
			for _, p := range abiertos {
				p.Close()
			}
		})
	}
	exito := false
	defer func() {
		if !exito {
			cerrar()
		}
	}()
	var pools [2]*pgxpool.Pool
	for i, e := range entradas {
		pool, login, err := abrirPoolNominalPortalExterno(ctx, e.dsn, e.rol, "vec-portal-externo-documentos-local")
		if err != nil {
			return vacio, docports.ErrSubidaLocalNoDisponible
		}
		abiertos = append(abiertos, pool)
		if login != e.login || externa.logins[login] ||
			cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil {
			return vacio, docports.ErrSubidaLocalNoDisponible
		}
		if i == 0 && acreditarFuncionesCarteraExterna(ctx, pool, firmasDatosSubidaLocal()) != nil {
			return vacio, docports.ErrSubidaLocalNoDisponible
		}
		pools[i] = pool
	}
	exito = true
	return poolsSubidaDocumentosExterna{datos: pools[0], frontera: pools[1], cerrar: cerrar}, nil
}

func firmasDatosSubidaLocal() []string {
	const argumentos = "(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"
	return []string{
		"vec_documentos.reservar_subida_local_v1" + argumentos,
		"vec_documentos.confirmar_recepcion_subida_local_v1" + argumentos,
		"vec_documentos.iniciar_analisis_subida_local_v1" + argumentos,
		"vec_documentos.registrar_analisis_subida_local_v1" + argumentos,
		"vec_documentos.cerrar_analisis_subida_local_v1" + argumentos,
		"vec_documentos.consultar_subida_local_propia_v1" + argumentos,
	}
}

type claveContextoSubidaDocumentosExterna struct{}

type contextoSubidaDocumentosExterna struct {
	actor       vec.ResultadoContextoActorRegistradoV2
	vinculo     vec.VinculoAutenticacionActorV2
	correlacion string
}

type vinculoCargaDocumentosExterna struct{}

// Cada acción consume su audiencia y su proveedor raíz M4 exactos.
type emisorMaterialSubidaLocalM4 struct {
	subir, descargar *emisorMaterialRenovableCTDesarrollo
}

func (e emisorMaterialSubidaLocalM4) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context,
	s vec.SolicitudAutorizacionLigadaV3, actor vec.ResultadoContextoActorRegistradoV2) (
	vec.DecisionAutorizacionLigadaV3,
	vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := s.Datos()
	if err != nil || ctx == nil || ctx.Err() != nil {
		return vec.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil,
			docports.ErrSubidaLocalDenegada
	}
	var elegido *emisorMaterialRenovableCTDesarrollo
	switch datos.Accion {
	case docports.AccionMeritosComprobanteSubirLocal:
		elegido = e.subir
	case docports.AccionMeritosComprobanteDescargarLocal:
		elegido = e.descargar
	default:
		return vec.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil,
			docports.ErrSubidaLocalDenegada
	}
	if elegido == nil {
		return vec.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil,
			docports.ErrSubidaLocalNoDisponible
	}
	return elegido.EmitirMaterialAutorizacionAtestadaV3(ctx, s, actor)
}

func (vinculoCargaDocumentosExterna) VinculoCargaLocalCertificado(ctx context.Context,
	actor vec.ResultadoContextoActorRegistradoV2) (vec.VinculoAutenticacionActorV2, error) {
	if ctx == nil || actor.Validar() != nil {
		return vec.VinculoAutenticacionActorV2{}, docports.ErrSubidaLocalDenegada
	}
	c, ok := ctx.Value(claveContextoSubidaDocumentosExterna{}).(contextoSubidaDocumentosExterna)
	if !ok || c.actor.Validar() != nil || c.vinculo.ValidarPara(actor) != nil ||
		c.actor.RegistroContextoRef != actor.RegistroContextoRef ||
		c.actor.HuellaSHA256 != actor.HuellaSHA256 ||
		c.actor.Contexto.PersonaRef != actor.Contexto.PersonaRef ||
		c.actor.Contexto.PerfilActivoRef != actor.Contexto.PerfilActivoRef {
		return vec.VinculoAutenticacionActorV2{}, docports.ErrSubidaLocalDenegada
	}
	return c.vinculo, nil
}

type clavePeticionDeclaracionDocumentalExterna struct{}
type claveHechoDeclaracionDocumentalExterna struct{}

// Este lector se invoca solo tras preparar la declaración de Méritos. Usa el
// actor M4 de esa misma petición y exige que el documento custodiado pertenezca
// al hecho determinístico que se va a declarar.
type lectorComprobanteDeclaracionExterna struct {
	autoridad *autoridadPreferenciasUsuariosDesarrollo
	servicio  docports.SubidorLocal
}

func (l *lectorComprobanteDeclaracionExterna) ConsultarVersionConfirmadaPropia(ctx context.Context,
	persona, ref string) (vec.ReferenciaDocumento, error) {
	if l == nil || l.autoridad == nil || l.servicio == nil || ctx == nil || ctx.Err() != nil ||
		persona == "" || ref == "" {
		return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalDenegada
	}
	r, ok := ctx.Value(clavePeticionDeclaracionDocumentalExterna{}).(*http.Request)
	hecho, hechoOK := ctx.Value(claveHechoDeclaracionDocumentalExterna{}).(string)
	if !ok || r == nil || !hechoOK || hecho == "" {
		return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalDenegada
	}
	c, ok := r.Context().Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	if !ok || c.autoridad != l.autoridad || c.resultado.Validar() != nil ||
		c.vinculo.ValidarPara(c.resultado) != nil ||
		!c.vinculo.VigenteEn(l.autoridad.reloj.Ahora(), c.resultado) ||
		c.resultado.Contexto.PersonaRef != persona {
		return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalDenegada
	}
	correlacion, err := vec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalNoDisponible
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalNoDisponible
	}
	q := docports.SolicitudConsultaLocal{ActorRegistradoV2: c.resultado, VinculoV2: c.vinculo,
		DocumentoID: ref, CorrelacionRef: correlacionRef}
	ctx = context.WithValue(ctx, claveContextoSubidaDocumentosExterna{},
		contextoSubidaDocumentosExterna{actor: c.resultado, vinculo: c.vinculo, correlacion: correlacionRef})
	estado, err := l.servicio.Consultar(ctx, q)
	if err != nil {
		if errors.Is(err, docports.ErrSubidaLocalDenegada) || errors.Is(err, docports.ErrSubidaLocalNoEncontrada) {
			return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalDenegada
		}
		return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalNoDisponible
	}
	if estado.ComprobanteRef != ref || estado.RecursoRef != hecho {
		return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalDenegada
	}
	if estado.Estado != docports.EstadoSubidaLocalAdmitida || estado.DocumentoVersion < 1 ||
		estado.RegistradoEn.IsZero() {
		return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalNoDisponible
	}
	original, err := l.servicio.AbrirOriginal(ctx, q)
	if err != nil || original == nil {
		if original != nil {
			_ = original.Close()
		}
		if errors.Is(err, docports.ErrSubidaLocalDenegada) {
			return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalDenegada
		}
		return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalNoDisponible
	}
	leidos, errorLectura := io.Copy(io.Discard, io.LimitReader(original, (16<<20)+1))
	errorCierre := original.Close()
	if errorLectura != nil || errorCierre != nil || leidos < 1 || leidos > 16<<20 || ctx.Err() != nil {
		return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalNoDisponible
	}
	documento := vec.ReferenciaDocumento{ID: ref, Version: estado.DocumentoVersion}
	if documento.Validar() != nil {
		return vec.ReferenciaDocumento{}, docports.ErrSubidaLocalNoDisponible
	}
	return documento, nil
}

// La frontera usa la misma cuenta y perfil de Méritos, fijados desde la hoja
// TLS de esta petición. No conserva estado de actor en el manejador compartido.
type componenteSubidaDocumentosExterna struct {
	autoridad *autoridadPreferenciasUsuariosDesarrollo
	frontera  vecports.RegistradorAuditoriaFronteraRutaExacta
	manejador http.Handler
	lector    *lectorComprobanteDeclaracionExterna
	cerrar    func()
}

func montarSubidaDocumentosCarteraExterna(ctx context.Context, cfg config.Config,
	cartera, declaracion *autoridadPreferenciasUsuariosDesarrollo,
	topologia topologiaPostgreSQLPreferenciasUsuarios,
	autorizador vecports.AutorizadorSolicitudLigadaV3,
	proveedores map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo,
	motivo vec.ReferenciaEntradaCatalogo) (*componenteSubidaDocumentosExterna, error) {
	if ctx == nil || ctx.Err() != nil || cartera == nil || declaracion == nil ||
		cartera.base == nil || cartera.cuentas == nil ||
		motivo.CatalogoID != "motivos_autorizacion_meritos_cartera" ||
		!vec.ReferenciaMotivoAutorizacionV2Valida(motivo) ||
		proveedores[audienciaSubirComprobanteM4] == nil ||
		proveedores[audienciaDescargarComprobanteM4] == nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	privada, err := leerConfiguracionSubidaDocumentosExterna(cfg.DevelopmentMaterialDir)
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	claves, err := leerClavesSubidaDocumentosExterna(cfg.DevelopmentMaterialDir, privada)
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	correlaciones, seudonimos, sellador, err := nuevasClavesAplicacionSubidaLocal(&claves)
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	pools, err := abrirPoolsSubidaDocumentosExterna(ctx, privada, topologia, cartera)
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	completo := false
	defer func() {
		if !completo {
			pools.cerrar()
		}
	}()
	frontera, err := docpg.NuevoRegistradorFronteraSubidaLocal(pools.frontera)
	if err != nil || frontera.Preflight(ctx) != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	registro, err := docpg.NuevoRepositorioSubidaLocalConPool(pools.datos)
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	resolutor, err := docpg.NuevoResolutorActoSubidaLocalMeritosConPool(pools.datos)
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	emisorSubir, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, proveedores[audienciaSubirComprobanteM4])
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	emisorDescargar, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, proveedores[audienciaDescargarComprobanteM4])
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	reloj := relojRutasDietas{}
	autoridad, err := docautorizacion.NuevaAutoridadSubidaLocal(
		emisorMaterialSubidaLocalM4{subir: emisorSubir, descargar: emisorDescargar},
		vinculoCargaDocumentosExterna{}, motivo, reloj)
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	almacen, cerrarAlmacen, err := nuevoAlmacenDocumentos(ctx, almacenDocumentosDesarrollo{
		Tipo: "ficheros", Directorio: privada.Directorio, TamanoMaximo: 16 << 20,
		RetencionMinimaDias: privada.RetencionMinimaDias}, reloj)
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	defer func() {
		if !completo {
			cerrarAlmacen()
		}
	}()
	if admitirCatalogoConservacion(cfg, true, "ficheros", almacen) != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	analizador, err := analisiscontenido.NuevoCliente(privada.Analizador.cliente())
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	capacidades, err := analizador.Capacidades(ctx)
	if err != nil || capacidades.ConectorID != privada.Analizador.ConectorID {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	servicio, err := docapp.NuevoServicioSubidaLocal(docapp.ConfiguracionServicioSubidaLocal{
		ModuloID: "meritos", TipoRecurso: "hecho", Finalidad: "custodiar_evidencia_meritos",
		Clasificacion: "confidencial", AccionSubir: docports.AccionMeritosComprobanteSubirLocal,
		AccionDescargar: docports.AccionMeritosComprobanteDescargarLocal,
	}, docapp.DependenciasServicioSubidaLocal{Resolutor: resolutor, Autoridad: autoridad,
		Registro: registro, Almacen: almacen, Analizador: analizador,
		Seudonimizador: seudonimos, SelladorSolicitud: sellador, Correlaciones: correlaciones,
		Reloj: reloj})
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	handler, err := meritoshttp.NuevoHandlerComprobantesCartera(preparadorComprobanteDocumentosExterno{},
		cargadorComprobanteDocumentosExterno{servicio: servicio})
	if err != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	var unaVez sync.Once
	cerrar := func() { unaVez.Do(func() { cerrarAlmacen(); pools.cerrar() }) }
	componente := &componenteSubidaDocumentosExterna{autoridad: cartera, frontera: frontera,
		manejador: handler, lector: &lectorComprobanteDeclaracionExterna{
			autoridad: declaracion, servicio: servicio}, cerrar: cerrar}
	completo = true
	return componente, nil
}

func (c *componenteSubidaDocumentosExterna) registrarDenegacion(ctx context.Context, rutaSolicitada string) error {
	if c == nil || c.frontera == nil || ctx == nil {
		return docports.ErrSubidaLocalNoDisponible
	}
	correlacion, err := nuevaCorrelacionDenegacionPreferenciasUsuarios()
	if err != nil {
		return err
	}
	ruta := meritoshttp.RutaComprobantesCartera
	if rutaSolicitada == meritoshttp.RutaEstadoComprobantesCartera {
		ruta = meritoshttp.RutaEstadoComprobantesCartera
	} else if rutaSolicitada != meritoshttp.RutaComprobantesCartera {
		ruta += "/{comprobante_ref}/original"
	}
	orden := vecports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: correlacion,
		Motivo:         vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida,
		Superficie:     vec.SuperficieAuditoriaFronteraRutaExactaDocumentosSubidaLocal,
		Ruta:           ruta,
	}
	if orden.Validar() != nil {
		return docports.ErrSubidaLocalNoDisponible
	}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancelar()
	return c.frontera.RegistrarAuditoriaFronteraRutaExacta(ctxAuditoria, orden)
}

func (c *componenteSubidaDocumentosExterna) proteger(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Path != meritoshttp.RutaComprobantesCartera &&
			!strings.HasPrefix(r.URL.Path, meritoshttp.RutaComprobantesCartera+"/") {
			siguiente.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		fallo := func(estado int, codigo string) {
			w.WriteHeader(estado)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"codigo": codigo}})
		}
		if c == nil || c.autoridad == nil || c.frontera == nil || c.manejador == nil || c.autoridad.base == nil {
			fallo(http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
			return
		}
		consultaEstado := r.Method == http.MethodGet && r.URL.Path == meritoshttp.RutaEstadoComprobantesCartera
		if r.URL.RawQuery != "" && !consultaEstado || r.URL.ForceQuery ||
			len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
			fallo(http.StatusBadRequest, "meritos.error.solicitud_invalida")
			return
		}
		segura, cuenta, _, ahora, err := c.autoridad.identificarCuentaCertificada(r)
		if err != nil {
			if errors.Is(err, ErrMaterialDesarrolloInvalido) &&
				c.registrarDenegacion(r.Context(), r.URL.Path) == nil {
				fallo(http.StatusUnauthorized, "meritos.error.sesion_necesaria")
				return
			}
			fallo(http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
			return
		}
		vinculo, actor, err := c.autoridad.resolverSesion(segura, cuenta, ahora)
		if err != nil || vinculo.ValidarPara(actor) != nil ||
			!vinculo.VigenteEn(c.autoridad.reloj.Ahora(), actor) {
			fallo(http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
			return
		}
		correlacion, err := vec.GenerarReferenciaCorrelacionAutorizacionV2(r.Context(), seguridad.GeneradorReferenciasCriptograficas{})
		if err != nil {
			fallo(http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
			return
		}
		ref, err := correlacion.ValorCanonico()
		if err != nil {
			fallo(http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
			return
		}
		ctx := context.WithValue(segura.Context(), claveContextoSubidaDocumentosExterna{},
			contextoSubidaDocumentosExterna{actor: actor, vinculo: vinculo, correlacion: ref})
		c.manejador.ServeHTTP(w, segura.WithContext(ctx))
	})
}

type preparadorComprobanteDocumentosExterno struct{}

var _ meritoshttp.ProveedorComprobanteCartera = preparadorComprobanteDocumentosExterno{}

func (preparadorComprobanteDocumentosExterno) PrepararComprobanteCartera(r *http.Request) (meritoshttp.IdentidadComprobanteCartera, error) {
	if r == nil || r.Context() == nil {
		return meritoshttp.IdentidadComprobanteCartera{}, meritosports.ErrCarteraSinSesion
	}
	c, ok := r.Context().Value(claveContextoSubidaDocumentosExterna{}).(contextoSubidaDocumentosExterna)
	if !ok || c.actor.Validar() != nil || c.vinculo.ValidarPara(c.actor) != nil ||
		!vec.ReferenciaCorrelacionAutorizacionV2Valida(c.correlacion) {
		return meritoshttp.IdentidadComprobanteCartera{}, meritosports.ErrCarteraSinSesion
	}
	return meritoshttp.IdentidadComprobanteCartera{ActorRegistradoV2: c.actor, VinculoV2: c.vinculo,
		CorrelacionRef: c.correlacion}, nil
}

// El puerto HTTP delega cada efecto al servicio común de custodia local.
// El recibo solo se transforma en confirmación pública cuando está admitido.
type cargadorComprobanteDocumentosExterno struct{ servicio docports.SubidorLocal }

var _ meritoshttp.CargadorComprobanteCartera = cargadorComprobanteDocumentosExterno{}

func (c cargadorComprobanteDocumentosExterno) SubirComprobante(ctx context.Context,
	q meritoshttp.IdentidadComprobanteCartera, clave string, contenido []byte) (meritoshttp.ResultadoCargaComprobanteCartera, error) {
	if c.servicio == nil || ctx == nil || ctx.Err() != nil {
		return meritoshttp.ResultadoCargaComprobanteCartera{}, docports.ErrSubidaLocalNoDisponible
	}
	r, err := c.servicio.Subir(ctx, docports.SolicitudSubidaLocal{
		ActorRegistradoV2: q.ActorRegistradoV2, VinculoV2: q.VinculoV2,
		ClaveIdempotencia: clave}, contenido)
	salida := meritoshttp.ResultadoCargaComprobanteCartera{
		ComprobanteRef: r.ComprobanteRef, CargaID: r.CargaID,
		RegistradoEn: r.RegistradoEn, Estado: string(r.Estado)}
	return salida, err
}

func (c cargadorComprobanteDocumentosExterno) ConsultarEstadoComprobante(ctx context.Context,
	q meritoshttp.IdentidadComprobanteCartera, clave string) (meritoshttp.EstadoComprobanteCartera, error) {
	if c.servicio == nil || ctx == nil || ctx.Err() != nil {
		return meritoshttp.EstadoComprobanteCartera{}, docports.ErrSubidaLocalNoDisponible
	}
	r, err := c.servicio.Consultar(ctx, docports.SolicitudConsultaLocal{
		ActorRegistradoV2: q.ActorRegistradoV2, VinculoV2: q.VinculoV2,
		ClaveIdempotencia: clave, CorrelacionRef: q.CorrelacionRef})
	if errors.Is(err, docports.ErrSubidaLocalNoEncontrada) {
		return meritoshttp.EstadoComprobanteCartera{}, meritoshttp.ErrEstadoComprobanteNoEncontrado
	}
	if err != nil {
		return meritoshttp.EstadoComprobanteCartera{}, err
	}
	salida := meritoshttp.EstadoComprobanteCartera{
		CargaID: r.CargaID, Estado: string(r.Estado), ReciboRef: r.ReciboRef,
		ComprobanteRef: r.ComprobanteRef, CodigoRechazo: r.CodigoRechazo}
	if !r.RegistradoEn.IsZero() {
		instante := r.RegistradoEn
		salida.RegistradoEn = &instante
	}
	return salida, nil
}

func (c cargadorComprobanteDocumentosExterno) AbrirOriginalComprobante(ctx context.Context,
	q meritoshttp.IdentidadComprobanteCartera, ref string) (io.ReadCloser, error) {
	if c.servicio == nil || ctx == nil || ctx.Err() != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	return c.servicio.AbrirOriginal(ctx, docports.SolicitudConsultaLocal{
		ActorRegistradoV2: q.ActorRegistradoV2, VinculoV2: q.VinculoV2,
		DocumentoID: ref, CorrelacionRef: q.CorrelacionRef})
}
