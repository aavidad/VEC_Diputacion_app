package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"vec-diputacion-granada/internal/vec/adapters/analisiscontenido"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
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
