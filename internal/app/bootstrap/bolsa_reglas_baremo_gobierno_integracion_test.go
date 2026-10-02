package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	bolsapg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// Ensayo directo del servicio y sus dependencias reales. No acredita HTTP ni
// navegador. Las claves persistentes y las entradas sintéticas son externas.
type configuracionEnsayoBaremo struct {
	Version        int                          `json:"version"`
	Entorno        map[string]string            `json:"entorno"`
	RuntimeDSN     string                       `json:"runtime_dsn"`
	EvidenciaDSN   string                       `json:"evidencia_dsn"`
	ResumenSQL     string                       `json:"resumen_sql"`
	Conjunto       string                       `json:"conjunto"`
	ConjuntoAjeno  string                       `json:"conjunto_ajeno"`
	Motivo         vd.ReferenciaEntradaCatalogo `json:"motivo"`
	ClaveOperacion string                       `json:"clave_operacion"`
	Continuidad    string                       `json:"continuidad"`
	Rutas          RutasGobiernoReglasBaremoV3  `json:"rutas"`
}

type resumenEnsayoBaremo struct {
	Versiones, Recibos, Historias, Outbox int64
	Huella                                string
}

type continuidadEnsayoBaremo struct {
	Recibo     bp.ReciboAltaBorradorReglasV3 `json:"recibo"`
	Resumen    resumenEnsayoBaremo           `json:"resumen"`
	Postmaster time.Time                     `json:"postmaster"`
}

type ensayoBaremoReal struct {
	perfil                 *PerfilGobiernoReglasBaremoV3
	broker                 *ProveedorGobiernoReglasBaremoV3
	servicio               *app.ServicioGobiernoV3
	principal              vd.Principal
	fronteras              catalogoFronterasComunDesarrollo
	runtime                *pgxpool.Pool
	certificadoValidoHasta time.Time
}

func TestGobiernoReglasBaremoIntegracionPostgreSQL(t *testing.T) {
	ruta := os.Getenv("VEC_BAREMO_PG_CONFIG")
	if ruta == "" {
		t.Skip("ensayo PostgreSQL real no configurado")
	}
	if os.Getenv("VEC_BAREMO_PG_DESECHABLE") != "si" {
		t.Fatal("se requiere activación explícita de clon desechable")
	}
	fase := os.Getenv("VEC_BAREMO_PG_FASE")
	if fase != "preparar" && fase != "alta" && fase != "recuperar" && fase != "reinicio" {
		t.Fatal("fase de ensayo desconocida")
	}
	var cfg configuracionEnsayoBaremo
	leerJSONEnsayoBaremo(t, ruta, &cfg)
	if cfg.Version != 1 || !rutaPrivadaEnsayoBaremo(cfg.Continuidad) || !cfg.Rutas.valida() ||
		!claveOperacionGobiernoV3(cfg.ClaveOperacion) || !vd.ReferenciaMotivoAutorizacionV2Valida(cfg.Motivo) || cfg.ResumenSQL == "" {
		t.Fatal("configuración privada incompleta")
	}
	cargarEntornoEnsayoBaremo(t, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	conjunto, err := reglas.RestaurarConjuntoReglasBaremo(ficheroPrivadoEnsayoBaremo(t, cfg.Conjunto))
	if err != nil {
		t.Fatal("conjunto canónico privado rechazado")
	}
	e := componerEnsayoBaremoReal(t, ctx, cfg, conjunto, fase == "preparar")
	if fase == "preparar" {
		return
	}
	evidencia := poolEnsayoBaremo(t, ctx, cfg.EvidenciaDSN)
	if evidencia.Config().ConnConfig.User == e.runtime.Config().ConnConfig.User {
		t.Fatal("evidencia y runtime comparten LOGIN")
	}
	peticion := peticionAltaEnsayoBaremo(t, cfg, conjunto)
	var guardado continuidadEnsayoBaremo
	_, errorContinuidad := os.Lstat(cfg.Continuidad)
	if fase == "alta" || (fase == "recuperar" && os.IsNotExist(errorContinuidad)) {
		if !os.IsNotExist(errorContinuidad) {
			t.Fatal("continuidad existente: recuperar la intención original")
		}
		antes, _ := resumenPGEnsayoBaremo(t, ctx, evidencia, cfg)
		esperados := int64(0)
		if fase == "recuperar" {
			esperados = 1
		}
		if antes.Versiones != esperados || antes.Recibos != esperados || antes.Historias != esperados || antes.Outbox != esperados {
			t.Fatal("la intención no está vacía")
		}
		pedido, credenciales := e.credenciales(t, ctx, cfg.Rutas.Alta)
		alta, err := e.servicio.GuardarAltaBorrador(pedido, credenciales, peticion)
		if err != nil || alta.Replay != (fase == "recuperar") {
			t.Fatal("alta o recuperación de la intención original no confirmada")
		}
		resumen, postmaster := resumenPGEnsayoBaremo(t, ctx, evidencia, cfg)
		if resumen.Versiones != 1 || resumen.Recibos != 1 || resumen.Historias != 1 || resumen.Outbox != 1 || !shaHexGobiernoV3(resumen.Huella) {
			t.Fatal("alta sin efectos durables únicos")
		}
		if fase == "recuperar" && resumen != antes {
			t.Fatal("recuperar la respuesta perdida modificó el negocio")
		}
		guardado = continuidadEnsayoBaremo{alta.Recibo, resumen, postmaster}
		guardarContinuidadEnsayoBaremo(t, cfg.Continuidad, guardado)
	} else {
		leerJSONEnsayoBaremo(t, cfg.Continuidad, &guardado)
		original, err := reglas.RestaurarVersionGobernadaReglasBaremo(guardado.Recibo.VersionCanonica)
		if err != nil {
			t.Fatal("canon conservado inválido")
		}
		guardado.Recibo.Estado, err = original.VinculoEstado()
		if err != nil {
			t.Fatal("estado conservado inválido")
		}
		actual, arranque := resumenPGEnsayoBaremo(t, ctx, evidencia, cfg)
		if actual != guardado.Resumen || (fase == "reinicio" && !arranque.After(guardado.Postmaster)) {
			t.Fatal("estado conservado o reinicio PostgreSQL no acreditados")
		}
	}
	pedido, credenciales := e.credenciales(t, ctx, cfg.Rutas.Alta)
	replay, err := e.servicio.GuardarAltaBorrador(pedido, credenciales, peticion)
	if err != nil || !replay.Replay || !reflect.DeepEqual(replay.Recibo, guardado.Recibo) || replay.Acceso.DecisionRef == guardado.Recibo.ConsumoOriginal.DecisionRef || replay.Acceso.AuditoriaRef == guardado.Recibo.AuditoriaRef {
		t.Fatal("replay sin recibo original o sin acceso nuevo separado")
	}
	selector := bp.SelectorGobiernoReglasV3{Identidad: conjunto.Identidad(), Estado: guardado.Recibo.Estado}
	pedido, credenciales = e.credenciales(t, ctx, cfg.Rutas.Consulta)
	consulta, err := e.servicio.ConsultarExacta(pedido, credenciales, app.PeticionConsultaExactaV3{Selector: selector, Motivo: cfg.Motivo})
	if err != nil || !bytes.Equal(consulta.VersionCanonica, guardado.Recibo.VersionCanonica) {
		t.Fatal("consulta exacta no conservó el canon original")
	}
	pedido, credenciales = e.credenciales(t, ctx, cfg.Rutas.Recuperar)
	recuperado, err := e.servicio.RecuperarRecibo(pedido, credenciales, app.PeticionRecuperarReciboV3{Selector: selector, Motivo: cfg.Motivo, ClaveOperacion: cfg.ClaveOperacion, HuellaSolicitudSHA256: guardado.Recibo.HuellaSolicitudSHA256})
	if err != nil || !recuperado.Existe || !reflect.DeepEqual(recuperado.Recibo, guardado.Recibo) || recuperado.Acceso.DecisionRef == guardado.Recibo.ConsumoOriginal.DecisionRef {
		t.Fatal("recuperación sin recibo original o sin acceso nuevo")
	}
	camposCruzadosEnsayoBaremo(t, ctx, cfg, e, selector, guardado.Recibo)
	negativosEnsayoBaremo(t, ctx, cfg, e, peticion, selector)
	final, _ := resumenPGEnsayoBaremo(t, ctx, evidencia, cfg)
	if final != guardado.Resumen {
		t.Fatal("replay, consultas o rechazos modificaron el negocio durable")
	}
}

func componerEnsayoBaremoReal(t *testing.T, ctx context.Context, cfg configuracionEnsayoBaremo, conjunto reglas.ConjuntoReglasBaremo, preparar bool) *ensayoBaremoReal {
	t.Helper()
	c := config.Load()
	composicion, err := NuevaComposicionSeguridadDesarrollo(c, io.Discard)
	if err != nil {
		t.Fatal("composición de seguridad existente rechazada")
	}
	t.Cleanup(composicion.derivadorIdempotencia.borrar)
	dependencias, err := nuevasDependenciasCT(c, composicion.identidad, composicion.derivadorIdempotencia, composicion.emisorKMS, io.Discard)
	if err != nil {
		t.Fatal("dependencias CT existentes rechazadas")
	}
	t.Cleanup(dependencias.Cerrar)
	// Baremo no consume el caso de alta CT ni su lector de cobertura O4-05.
	// Se componen sólo su soporte y las autoridades necesarias, con las
	// fábricas existentes de identidad/contexto/material.
	alta, material := baseNominalEnsayoBaremo(t, ctx, c, dependencias)
	defer material.borrarCopiasEfimeras()
	id := conjunto.Identidad()
	perfil, err := NuevoPerfilGobiernoReglasBaremoV3(alta.soporte, id.ConvocatoriaRef(), id.ExpedienteRef(), dependencias.reloj.Ahora())
	if err != nil {
		t.Fatal("perfil nominal de baremo rechazado")
	}
	fronteras := fronterasEnsayoBaremo(t, perfil, cfg.Rutas)
	if preparar {
		operacion := referenciaAltaContratacionTemporalDesarrollo("oca_", perfil.soporte.principalID+"\x00"+perfil.soporte.certificadoSHA256+"\x00registro-contexto-baremo-v3")
		if publicarResultadoContextoPostgreSQLDesarrollo(ctx, alta.postgresql.gobierno, perfil.soporte.contexto.Resultado, operacion) != nil ||
			AsegurarPerfilGobiernoReglasBaremoV3(ctx, alta.postgresql.gobierno, perfil, "", "", dependencias.reloj.Ahora()) != nil {
			t.Fatal("provisión nominal real no confirmada")
		}
		desde, _, _ := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(dependencias.reloj.Ahora())
		if publicarCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo(ctx, alta.postgresql.gobierno, []vd.ReferenciaEntradaCatalogo{cfg.Motivo}, desde) != nil {
			t.Fatal("motivo gobernado no publicado")
		}
	}
	identidad, cerrar, err := nuevasDependenciasIdentidadConsultasDesarrollo(ctx, c.ContratacionTemporalPostgreSQL, &alta, composicion.derivadorIdempotencia, dependencias.reloj, perfil.soporte, fronteras)
	if err != nil {
		t.Fatal("sesión nominal PostgreSQL no disponible")
	}
	t.Cleanup(cerrar)
	fuenteDSN, motivosDSN, err := c.DSNAutoridadesAutorizacionRRHH()
	if err != nil {
		t.Fatal("autoridades nominales RRHH incompletas")
	}
	fuentePool, err := abrirPoolAutorizacionRRHHDesarrollo(ctx, fuenteDSN, config.RolAutorizacionFuenteRRHH, "vec-ensayo-baremo-fuente")
	if err != nil {
		t.Fatal("fuente nominal rechazada")
	}
	t.Cleanup(fuentePool.Close)
	motivosPool, err := abrirPoolAutorizacionRRHHDesarrollo(ctx, motivosDSN, config.RolAutorizacionMotivosEvaluadorRRHH, "vec-ensayo-baremo-motivos")
	if err != nil {
		t.Fatal("evaluador nominal rechazado")
	}
	t.Cleanup(motivosPool.Close)
	fuente, err := postgresvec.NuevoAlmacenAutorizacion(fuentePool)
	if err != nil {
		t.Fatal("fuente V3 no disponible")
	}
	registro, err := postgresvec.NuevoAlmacenAutorizacion(alta.postgresql.registroAutorizacion)
	if err != nil {
		t.Fatal("registro V3 no disponible")
	}
	validador, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(motivosPool, cfg.Motivo.CatalogoID)
	if err != nil {
		t.Fatal("validador de motivos no disponible")
	}
	pdp, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro, validador, dependencias.reloj, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second})
	if err != nil {
		t.Fatal("PDP común no disponible")
	}
	// El constructor de arranque común conserva clave/raíz y ajusta las
	// coordenadas a la versión realmente gobernada en PostgreSQL. Se repite
	// idempotentemente al recomponer; no publica por operación del servicio.
	proveedor, err := nuevoProveedorMaterialConsumidorConDescriptorDesarrollo(ctx, alta.postgresql.gobierno, material, perfil.soporte, dependencias.reloj, DescriptorMaterialGobiernoReglasBaremoV3())
	if err != nil {
		t.Fatal("proveedor común no disponible")
	}
	broker, err := NuevoProveedorGobiernoReglasBaremoV3(perfil, identidad, pdp, proveedor, fuentePool, cfg.Rutas, dependencias.reloj)
	if err != nil {
		t.Fatal("broker nominal no disponible")
	}
	runtime := poolEnsayoBaremo(t, ctx, cfg.RuntimeDSN)
	repo, err := bolsapg.NuevoRepositorioGobiernoReglasBaremoV3PostgreSQL(ctx, runtime, "vec_bolsa_reglas_baremo_ejecutor_gobierno")
	if err != nil {
		t.Fatal("preflight runtime AD144/BR4 rechazado")
	}
	servicio, err := app.NuevoServicioGobiernoV3(repo, repo, broker, dependencias.reloj.Ahora)
	if err != nil {
		t.Fatal("servicio V3 no compuesto")
	}
	principal, ok := dependencias.resolvedor.principalConRolUnico(rolTecnicoRRHHContratacionTemporalDesarrollo)
	if !ok {
		t.Fatal("identidad nominal RRHH ausente")
	}
	certificado, err := decodificarCertificadoUnico(ficheroPrivadoEnsayoBaremo(t, c.DevelopmentPaths().ClientCertificate))
	if err != nil {
		t.Fatal("certificado RRHH privado rechazado")
	}
	huella := sha256.Sum256(certificado.Raw)
	if hex.EncodeToString(huella[:]) != principal.Attributes["certificate_sha256"] {
		t.Fatal("certificado RRHH distinto de la identidad compuesta")
	}
	return &ensayoBaremoReal{perfil, broker, servicio, principal, fronteras, runtime, certificado.NotAfter.UTC().Truncate(time.Microsecond)}
}

func peticionAltaEnsayoBaremo(t *testing.T, cfg configuracionEnsayoBaremo, conjunto reglas.ConjuntoReglasBaremo) app.PeticionAltaBorradorV3 {
	t.Helper()
	ref, err := reglas.NuevaReferenciaVersionada(cfg.Motivo.CatalogoID, uint64(cfg.Motivo.CatalogoVersion), cfg.Motivo.CatalogoHuellaSHA256)
	if err != nil {
		t.Fatal("referencia de catálogo inválida")
	}
	motivo, err := reglas.NuevoMotivoCatalogadoReglasBaremo(ref, cfg.Motivo.EntradaClave)
	if err != nil {
		t.Fatal("motivo de dominio inválido")
	}
	return app.PeticionAltaBorradorV3{Conjunto: conjunto, Motivo: motivo, ClaveOperacion: cfg.ClaveOperacion}
}

func negativosEnsayoBaremo(t *testing.T, ctx context.Context, cfg configuracionEnsayoBaremo, e *ensayoBaremoReal, peticion app.PeticionAltaBorradorV3, selector bp.SelectorGobiernoReglasV3) {
	t.Helper()
	ajeno, err := reglas.RestaurarConjuntoReglasBaremo(ficheroPrivadoEnsayoBaremo(t, cfg.ConjuntoAjeno))
	if err != nil || (ajeno.Identidad().ConvocatoriaRef() == peticion.Conjunto.Identidad().ConvocatoriaRef() && ajeno.Identidad().ExpedienteRef() == peticion.Conjunto.Identidad().ExpedienteRef()) {
		t.Fatal("negativo privado sin ámbito distinto")
	}
	pedido, credenciales := e.credenciales(t, ctx, cfg.Rutas.Alta)
	otra := peticion
	otra.Conjunto = ajeno
	if _, err := e.servicio.GuardarAltaBorrador(pedido, credenciales, otra); !errors.Is(err, app.ErrGobiernoV3Prohibido) {
		t.Fatal("ámbito ajeno no produjo denegación nominal (403 en transporte)")
	}
	for _, ruta := range []string{cfg.Rutas.Consulta, cfg.Rutas.Recuperar} {
		pedido, credenciales = e.credenciales(t, ctx, ruta)
		if _, err := e.servicio.GuardarAltaBorrador(pedido, credenciales, peticion); !errors.Is(err, app.ErrGobiernoV3NoAutenticado) {
			t.Fatal("operación cruzada entre fronteras admitida")
		}
	}
	// Caída real del pool local del consumidor: no se sustituye el repositorio.
	pedido, credenciales = e.credenciales(t, ctx, cfg.Rutas.Consulta)
	e.runtime.Close()
	if _, err := e.servicio.ConsultarExacta(pedido, credenciales, app.PeticionConsultaExactaV3{Selector: selector, Motivo: cfg.Motivo}); !errors.Is(err, app.ErrGobiernoV3NoDisponible) {
		t.Fatal("pool cerrado no produjo indisponibilidad nominal (503 en transporte)")
	}
}

func guardarContinuidadEnsayoBaremo(t *testing.T, ruta string, guardado continuidadEnsayoBaremo) {
	t.Helper()
	b, err := json.Marshal(guardado)
	if err != nil {
		t.Fatal("continuidad no serializable")
	}
	f, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal("continuidad privada no creada")
	}
	_, err = f.Write(b)
	syncErr, closeErr := f.Sync(), f.Close()
	if err != nil || syncErr != nil || closeErr != nil {
		t.Fatal("continuidad privada no confirmada")
	}
}
