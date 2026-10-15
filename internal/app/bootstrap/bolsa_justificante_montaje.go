package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	bolsapg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/analisiscontenido"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	docaut "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const consumidorJustificantesBolsaExterno = "bolsa_justificantes"
const rutaConfiguracionJustificantesBolsa = "externo/documentos/justificantes-bolsa.json"

type configuracionJustificantesBolsa struct {
	Version int                                  `json:"version"`
	Datos   configuracionSubidaDocumentosExterna `json:"datos"`
	Motivo  vec.ReferenciaEntradaCatalogo        `json:"motivo"`
}

func leerConfiguracionJustificantesBolsa(material string) (configuracionJustificantesBolsa, error) {
	var c configuracionJustificantesBolsa
	b, err := leerFicheroMaterialSeguro(filepath.Join(material, rutaConfiguracionJustificantesBolsa), 128<<10)
	if err != nil {
		return c, err
	}
	defer clear(b)
	if validarClavesJSONUnicas(b) != nil {
		return c, docports.ErrSubidaLocalNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var extra any
	if d.Decode(&c) != nil || !errors.Is(d.Decode(&extra), io.EOF) || c.Version != 1 || c.Datos.Version != 1 ||
		c.Datos.DSNDatos == "" || strings.TrimSpace(c.Datos.DSNDatos) != c.Datos.DSNDatos || c.Datos.DSNFrontera != "" ||
		!directorioLocalSubidaSeguro(c.Datos.Directorio) || c.Datos.RetencionMinimaDias == nil || *c.Datos.RetencionMinimaDias < 0 ||
		!nombreClaveSubidaDocumentosValido(c.Datos.ClaveCorrelacionArchivo) ||
		!nombreClaveSubidaDocumentosValido(c.Datos.ClaveSeudonimoArchivo) ||
		!nombreClaveSubidaDocumentosValido(c.Datos.ClaveSolicitudArchivo) ||
		c.Datos.Analizador.Endpoint == "" || c.Datos.Analizador.NombreServidor == "" ||
		c.Datos.Analizador.RutaCA == "" || c.Datos.Analizador.RutaCertificado == "" || c.Datos.Analizador.RutaClave == "" ||
		c.Datos.Analizador.ConectorID == "" || c.Motivo.CatalogoID != "motivos_autorizacion_justificante_renuncia_bolsa" ||
		!vec.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) {
		return configuracionJustificantesBolsa{}, docports.ErrSubidaLocalNoDisponible
	}
	return c, nil
}

type fabricaJustificantesBolsa struct {
	pool          *pgxpool.Pool
	registro      docports.RegistroSubidaLocal
	emisores      emisoresJustificanteBolsa
	preparacion   autoridadPreparacionJustificanteBolsa
	motivo        vec.ReferenciaEntradaCatalogo
	almacen       vecports.AlmacenObjetos
	analizador    vecports.AnalizadorContenido
	seudonimos    vecports.SeudonimizadorSujetoAlmacen
	sellador      vecports.SelladorSolicitudCargaDocumental
	correlaciones docports.DerivadorCorrelacionSubidaLocal
}

func (f *fabricaJustificantesBolsa) ServicioJustificanteRenuncia(ctx context.Context, q bolsahttp.IdentidadJustificanteRenuncia,
	bolsa string) (docports.SubidorLocal, error) {
	if f == nil || ctx == nil || ctx.Err() != nil {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	resolutor, err := bolsapg.NuevoResolutorJustificanteRenunciaConPool(f.pool, bolsa, f.preparacion)
	if err != nil {
		return nil, err
	}
	reloj := relojRutasDietas{}
	autoridad, err := docaut.NuevaAutoridadSubidaLocalBolsa(f.emisores, vinculoJustificanteBolsaFijado{q}, f.motivo, reloj)
	if err != nil {
		return nil, err
	}
	return docapp.NuevoServicioSubidaLocal(docapp.ConfiguracionServicioSubidaLocal{
		ModuloID: "bolsa", TipoRecurso: "justificante_renuncia", Finalidad: "custodiar_justificante_renuncia_bolsa",
		Clasificacion: "confidencial", AccionSubir: docports.AccionBolsaJustificanteRenunciaSubirLocal,
		AccionDescargar: docports.AccionBolsaJustificanteRenunciaDescargarLocal,
	}, docapp.DependenciasServicioSubidaLocal{
		Resolutor: resolutorDocumentalJustificanteBolsa{resolutor, bolsa}, Autoridad: autoridad,
		Registro: f.registro, Almacen: f.almacen, Analizador: f.analizador, Seudonimizador: f.seudonimos,
		SelladorSolicitud: f.sellador, Correlaciones: correlacionJustificanteBolsa{f.correlaciones, bolsa}, Reloj: reloj,
	})
}

// La configuración es opcional y no se publica ningún permiso al arrancar.
// Cada fallo deja cerrada únicamente esta ruta.
func montarJustificantesBolsaExternos(ctx context.Context, cfg config.Config, preflight *pgxpool.Pool,
	topologia topologiaPostgreSQLPreferenciasUsuarios, fuente vecports.FuenteAutorizacion,
	registro *vecpg.AlmacenAutorizacion, motivosPool *pgxpool.Pool) (http.Handler, func()) {
	h, cerrar, err := prepararJustificantesBolsaExternos(ctx, cfg, preflight, topologia, fuente, registro, motivosPool)
	if err != nil {
		return nil, func() {}
	}
	return h, cerrar
}

func prepararJustificantesBolsaExternos(ctx context.Context, cfg config.Config, preflight *pgxpool.Pool,
	topologia topologiaPostgreSQLPreferenciasUsuarios, fuente vecports.FuenteAutorizacion,
	registro *vecpg.AlmacenAutorizacion, motivosPool *pgxpool.Pool) (http.Handler, func(), error) {
	nada := func() {}
	etapa, preparado := "configuracion", false
	defer func() {
		if !preparado {
			slog.Warn("justificante_renuncia_bolsa_no_disponible", "etapa", etapa)
		}
	}()
	privada, err := leerConfiguracionJustificantesBolsa(cfg.DevelopmentMaterialDir)
	if err != nil {
		return nil, nada, err
	}
	etapa = "pool_nominal"
	pool, login, err := abrirPoolNominalPortalExterno(ctx, privada.Datos.DSNDatos, "vec_documentos_bolsa_subida_datos", "vec-bolsa-justificante")
	if err != nil {
		return nil, nada, err
	}
	completo := false
	defer func() {
		if !completo {
			pool.Close()
		}
	}()
	etapa = "topologia"
	if login != "vec_externo_documentos_bolsa_datos" || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil {
		return nil, nada, docports.ErrSubidaLocalNoDisponible
	}
	etapa = "funciones"
	funciones := append(firmasDatosSubidaLocal(), "vec_documentos.resolver_justificante_renuncia_bolsa_v1(text,text,text,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)")
	if acreditarFuncionesCarteraExterna(ctx, pool, funciones) != nil {
		return nil, nada, docports.ErrSubidaLocalNoDisponible
	}
	etapa = "claves"
	claves, err := leerClavesSubidaDocumentosExterna(cfg.DevelopmentMaterialDir, privada.Datos)
	if err != nil {
		return nil, nada, err
	}
	etapa = "claves_aplicacion"
	correlaciones, seudonimos, sellador, err := nuevasClavesAplicacionSubidaLocal(&claves)
	if err != nil {
		return nil, nada, err
	}
	etapa = "motivo"
	validador, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2Externo(motivosPool, privada.Motivo.CatalogoID)
	if err != nil {
		return nil, nada, err
	}
	etapa = "autorizacion"
	autorizador, err := nuevaAutorizacionMiBolsaPortalExterno(fuente, registro, registro,
		motivosMiBolsaPortalExterno{privada.Motivo.CatalogoID: validador})
	if err != nil {
		return nil, nada, err
	}
	etapa = "confianza"
	proveedores, err := nuevosProveedoresV3PortalExterno(ctx, cfg.DevelopmentMaterialDir, preflight,
		consumidorJustificantesBolsaExterno, relojContratacionTemporalDesarrollo{})
	if err != nil {
		return nil, nada, err
	}
	etapa = "emisores"
	emisores := map[string]*emisorMaterialRenovableCTDesarrollo{}
	for _, audiencia := range []string{audienciaJustificanteBolsaSubir, audienciaJustificanteBolsaDescargar, audienciaJustificanteBolsaPreparar} {
		proveedor := proveedores[audiencia]
		if proveedor == nil {
			return nil, nada, docports.ErrSubidaLocalNoDisponible
		}
		e, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, proveedor)
		if err != nil {
			return nil, nada, err
		}
		emisores[audiencia] = e
	}
	reloj := relojRutasDietas{}
	etapa = "almacen"
	almacen, cerrarAlmacen, err := nuevoAlmacenDocumentos(ctx, almacenDocumentosDesarrollo{
		Tipo: "ficheros", Directorio: privada.Datos.Directorio, TamanoMaximo: 16 << 20,
		RetencionMinimaDias: privada.Datos.RetencionMinimaDias}, reloj)
	if err != nil {
		return nil, nada, err
	}
	defer func() {
		if !completo {
			cerrarAlmacen()
		}
	}()
	etapa = "conservacion"
	if admitirCatalogoConservacion(cfg, true, "ficheros", almacen) != nil {
		return nil, nada, docports.ErrSubidaLocalNoDisponible
	}
	etapa = "analizador"
	analizador, err := analisiscontenido.NuevoCliente(privada.Datos.Analizador.cliente())
	if err != nil {
		return nil, nada, err
	}
	etapa = "capacidad_analizador"
	capacidad, err := analizador.Capacidades(ctx)
	if err != nil || capacidad.ConectorID != privada.Datos.Analizador.ConectorID {
		return nil, nada, docports.ErrSubidaLocalNoDisponible
	}
	etapa = "repositorio"
	guardador, err := docpg.NuevoRepositorioSubidaLocalConPool(pool)
	if err != nil {
		return nil, nada, err
	}
	fabrica := &fabricaJustificantesBolsa{pool: pool, registro: guardador,
		emisores:    emisoresJustificanteBolsa{emisores[audienciaJustificanteBolsaSubir], emisores[audienciaJustificanteBolsaDescargar]},
		preparacion: autoridadPreparacionJustificanteBolsa{emisores[audienciaJustificanteBolsaPreparar], privada.Motivo},
		motivo:      privada.Motivo, almacen: almacen, analizador: analizador, seudonimos: seudonimos,
		sellador: sellador, correlaciones: correlaciones}
	etapa = "handler"
	handler, err := bolsahttp.NuevoHandlerJustificanteRenuncia(proveedorIdentidadJustificanteBolsa{}, fabrica)
	if err != nil {
		return nil, nada, err
	}
	var unaVez sync.Once
	completo = true
	preparado = true
	return handler, func() { unaVez.Do(func() { cerrarAlmacen(); pool.Close() }) }, nil
}

// Las tres claves propias se derivan por separado y conservan las coordenadas
// de los consumidores externos publicados con anterioridad.
func descriptoresMaterialJustificantesBolsaExternos() []descriptorMaterialConsumidorV3Desarrollo {
	var salida []descriptorMaterialConsumidorV3Desarrollo
	for i, audiencia := range []string{audienciaJustificanteBolsaSubir, audienciaJustificanteBolsaDescargar, audienciaJustificanteBolsaPreparar} {
		sufijo := []string{"subir", "descargar", "preparar"}[i]
		salida = append(salida, descriptorMaterialConsumidorV3Desarrollo{Audiencia: audiencia,
			Dominio:          "vec.bolsa.justificante-renuncia." + sufijo + ".desarrollo.capacidad-v3",
			Prefijo:          "clave:capacidad:bolsa-justificante-renuncia-" + sufijo + ":",
			ProveedorNominal: "proveedor-material-bolsa-justificante-renuncia-" + sufijo})
	}
	return salida
}
