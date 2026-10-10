package bootstrap

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	gobiernoperfiles "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Ficheros del overlay diario de la versión de rol de Bolsa (B1) que se
// escriben en la misma carpeta privada de la publicación. Cada clave va en su
// propio fichero 0600: nunca se comparte con otro overlay de vec-admin.
const (
	ArchivoOverlayVersionBolsa           = "version-bolsa.json"
	ArchivoMaterialVersionBolsaPropuesta = "version-bolsa-propuesta.bin"
	ArchivoMaterialVersionBolsaCierre    = "version-bolsa-cierre.bin"
)

// DestinoOverlayVersionBolsaAdmin procede de la configuración privada del
// operador. Los dos pools son los ficheros DSN de los LOGIN B1 ya
// aprovisionados por el DBA; los motivos, las entradas publicadas del
// catálogo de motivos para las dos audiencias. DirectorioSalida es la ruta
// absoluta de la carpeta abierta como raíz, porque vec-admin exige rutas
// absolutas para los ficheros de material.
type DestinoOverlayVersionBolsaAdmin struct {
	DirectorioSalida           string
	PoolGobierno, PoolCatalogo string
	Motivos                    map[string]domain.ReferenciaEntradaCatalogo
}

// Formato de VEC_ADMIN_VERSION_BOLSA_CONFIG_FILE: las mismas claves JSON que
// lee vec-admin (configuracionGobiernoRolesPrivada y sus metadatos).
type overlayVersionBolsa struct {
	PoolGobierno string                                      `json:"pool_gobierno"`
	PoolCatalogo string                                      `json:"pool_catalogo"`
	Confianza    confianzaOverlayVersionBolsa                `json:"confianza"`
	Motivos      map[string]domain.ReferenciaEntradaCatalogo `json:"motivos"`
}

type confianzaOverlayVersionBolsa struct {
	Cabecera struct {
		FormatoVersion uint16 `json:"formato_version"`
		Suite          string `json:"suite"`
		ClaveID        string `json:"clave_id"`
		Audiencia      string `json:"audiencia"`
	} `json:"cabecera"`
	Raiz struct {
		ClaveID       string    `json:"clave_id"`
		Audiencia     string    `json:"audiencia"`
		Version       uint64    `json:"version"`
		PublicaBase64 string    `json:"publica_base64"`
		Estado        string    `json:"estado"`
		ValidaDesde   time.Time `json:"valida_desde"`
		ValidaHasta   time.Time `json:"valida_hasta"`
		RevocadaEn    time.Time `json:"revocada_en"`
	} `json:"raiz"`
	Gobierno struct {
		Revision     string    `json:"revision"`
		HuellaSHA256 string    `json:"huella_sha256"`
		Secuencia    uint64    `json:"secuencia"`
		PublicadaEn  time.Time `json:"publicada_en"`
		ExpiraEn     time.Time `json:"expira_en"`
	} `json:"gobierno"`
	EntradasCapacidad []entradaOverlayVersionBolsa `json:"entradas_capacidad"`
}

type entradaOverlayVersionBolsa struct {
	Audiencia        string    `json:"audiencia"`
	ClaveID          string    `json:"clave_id"`
	EmisorID         string    `json:"emisor_id"`
	HuellaGobierno   string    `json:"huella_gobierno"`
	Version          uint64    `json:"version"`
	RevisionGobierno uint64    `json:"revision_gobierno"`
	MaterialArchivo  string    `json:"material_archivo"`
	Estado           string    `json:"estado"`
	ValidaDesde      time.Time `json:"valida_desde"`
	ValidaHasta      time.Time `json:"valida_hasta"`
	RevocadaEn       time.Time `json:"revocada_en"`
}

// Una clave tal como la escribe EscribirMaterialPrivado en material.json.
type claveMaterialGobiernoUsuarios struct {
	Audiencia      string    `json:"audiencia"`
	ClaveID        string    `json:"clave_id"`
	Version        uint64    `json:"version"`
	Revision       uint64    `json:"revision_gobierno"`
	HuellaGobierno string    `json:"huella_gobierno_sha256"`
	Secreto        []byte    `json:"secreto_hmac"`
	HuellaSecreto  string    `json:"huella_secreto_sha256"`
	EmisorID       string    `json:"emisor_id"`
	Desde          time.Time `json:"valida_desde"`
	Hasta          time.Time `json:"valida_hasta"`
}

// EscribirOverlayVersionBolsaAdmin genera el overlay B1 del día a partir de
// una publicación ya confirmada: exige el acuse de aplicar con estado
// permitido, ligado a los mismos plan y material de la carpeta, y un conjunto
// que incluya las dos audiencias B1. Escribe primero los dos ficheros de
// clave y por último version-bolsa.json; nada se sobrescribe. No conecta con
// la base ni publica nada.
func EscribirOverlayVersionBolsaAdmin(salida *os.Root, d DestinoOverlayVersionBolsaAdmin, nombreAcuse string, reloj ports.Reloj) error {
	if salida == nil || dependenciaBootstrapNula(reloj) || !nombreAcuseGobiernoUsuariosValido(nombreAcuse) || validarDestinoOverlayVersionBolsa(d) != nil {
		return ErrGobiernoUsuariosAdmin
	}
	data, err := leerPrivadoGobiernoUsuarios(salida, ArchivoConfiguracionGobiernoUsuarios)
	defer borrarBytes(data)
	var cfg ConfiguracionMaterialUsuariosAdmin
	if err != nil || decodificarGobiernoUsuarios(data, &cfg) != nil {
		return ErrGobiernoUsuariosAdmin
	}
	conjunto, ok := AudienciasConjuntoCapacidadesAdmin(cfg.ConjuntoVersion)
	if !ok || cfg.ConjuntoVersion == 0 {
		return ErrGobiernoUsuariosAdmin
	}
	segmentos := map[string]string{}
	for _, a := range conjunto {
		segmentos[a.Audiencia] = a.Segmento
	}
	audiencias := []string{gobiernoperfiles.AudienciaVersionarRolBolsaProponer, gobiernoperfiles.AudienciaVersionarRolBolsaAprobar}
	for _, a := range audiencias {
		if segmentos[a] == "" {
			return ErrGobiernoUsuariosAdmin
		}
	}
	plan, err := leerPrivadoGobiernoUsuarios(salida, ArchivoPlanGobiernoUsuarios)
	if err != nil {
		return ErrGobiernoUsuariosAdmin
	}
	material, err := leerPrivadoGobiernoUsuarios(salida, ArchivoMaterialGobiernoUsuarios)
	defer borrarBytes(material)
	if err != nil {
		return ErrGobiernoUsuariosAdmin
	}
	acuse, err := leerPrivadoGobiernoUsuarios(salida, nombreAcuse)
	if err != nil {
		return ErrGobiernoUsuariosAdmin
	}
	// El acuse prueba que la base confirmó COMMIT de este plan y material.
	h := sha256.Sum256(plan)
	r, err := validarAcuseGobiernoUsuariosAdmin(acuse, string(plan), hex.EncodeToString(h[:]), material)
	if err != nil || r.Estado != "permitido" {
		return ErrGobiernoUsuariosAdmin
	}
	var p planGobiernoUsuariosAdmin
	if decodificarGobiernoUsuarios(plan, &p) != nil || p.ConjuntoVersion != cfg.ConjuntoVersion ||
		p.Configuracion.Revision != cfg.Gobierno.Revision || p.Configuracion.Huella != cfg.Gobierno.HuellaSHA256 ||
		p.Configuracion.Secuencia != cfg.Gobierno.Secuencia {
		return ErrGobiernoUsuariosAdmin
	}
	var m struct {
		Claves []claveMaterialGobiernoUsuarios `json:"claves"`
	}
	if decodificarGobiernoUsuarios(material, &m) != nil || len(m.Claves) != len(conjunto) {
		return ErrGobiernoUsuariosAdmin
	}
	defer func() {
		for i := range m.Claves {
			borrarBytes(m.Claves[i].Secreto)
		}
	}()
	ahora := reloj.Ahora().UTC()
	if !ahora.Before(cfg.Gobierno.ExpiraEn) || cfg.Raiz.Estado != confianza.EstadoClaveAtestacionAutorizacionV3Activa || !cfg.Raiz.RevocadaEn.IsZero() || len(cfg.Raiz.Publica) != 32 {
		return ErrGobiernoUsuariosAdmin
	}
	var o overlayVersionBolsa
	o.PoolGobierno, o.PoolCatalogo, o.Motivos = d.PoolGobierno, d.PoolCatalogo, d.Motivos
	c := &o.Confianza
	c.Cabecera.FormatoVersion = domain.VersionFormatoAtestacionAutorizacionV3
	c.Cabecera.Suite = confianza.SuiteAtestacionAutorizacionV3COSEEdDSA
	c.Cabecera.ClaveID, c.Cabecera.Audiencia = cfg.Raiz.ClaveID, cfg.Raiz.Audiencia
	c.Raiz.ClaveID, c.Raiz.Audiencia, c.Raiz.Version = cfg.Raiz.ClaveID, cfg.Raiz.Audiencia, cfg.Raiz.Version
	c.Raiz.PublicaBase64 = base64.StdEncoding.EncodeToString(cfg.Raiz.Publica)
	c.Raiz.Estado = string(cfg.Raiz.Estado)
	c.Raiz.ValidaDesde, c.Raiz.ValidaHasta = cfg.Raiz.ValidaDesde.UTC(), cfg.Raiz.ValidaHasta.UTC()
	c.Raiz.RevocadaEn = time.Time{}.UTC()
	c.Gobierno.Revision, c.Gobierno.HuellaSHA256, c.Gobierno.Secuencia = cfg.Gobierno.Revision, cfg.Gobierno.HuellaSHA256, cfg.Gobierno.Secuencia
	c.Gobierno.PublicadaEn, c.Gobierno.ExpiraEn = cfg.Gobierno.PublicadaEn.UTC(), cfg.Gobierno.ExpiraEn.UTC()
	archivos := map[string]string{
		gobiernoperfiles.AudienciaVersionarRolBolsaProponer: ArchivoMaterialVersionBolsaPropuesta,
		gobiernoperfiles.AudienciaVersionarRolBolsaAprobar:  ArchivoMaterialVersionBolsaCierre,
	}
	secretos := map[string][]byte{}
	for _, a := range audiencias {
		i := slices.IndexFunc(m.Claves, func(k claveMaterialGobiernoUsuarios) bool { return k.Audiencia == a })
		if i < 0 || m.Claves[i].Audiencia != conjunto[i].Audiencia {
			return ErrGobiernoUsuariosAdmin
		}
		k := m.Claves[i]
		hs := sha256.Sum256(k.Secreto)
		if len(k.Secreto) != 32 || hex.EncodeToString(hs[:]) != k.HuellaSecreto || k.EmisorID != conjunto[i].EmisorID ||
			!strings.HasPrefix(k.ClaveID, "clave:capacidad:admin:"+segmentos[a]+":") || k.Version == 0 || k.Revision == 0 ||
			!shaGobiernoUsuarios.MatchString(k.HuellaGobierno) || !ahora.Before(k.Hasta) || !k.Hasta.After(k.Desde) {
			return ErrGobiernoUsuariosAdmin
		}
		secretos[a] = k.Secreto
		c.EntradasCapacidad = append(c.EntradasCapacidad, entradaOverlayVersionBolsa{
			Audiencia: a, ClaveID: k.ClaveID, EmisorID: k.EmisorID, HuellaGobierno: k.HuellaGobierno,
			Version: k.Version, RevisionGobierno: k.Revision, MaterialArchivo: filepath.Join(d.DirectorioSalida, archivos[a]),
			Estado: string(confianza.EstadoClaveHMACCapacidadAtestacionV3Emision), ValidaDesde: k.Desde.UTC(), ValidaHasta: k.Hasta.UTC(),
			RevocadaEn: time.Time{}.UTC()})
	}
	b, err := json.MarshalIndent(o, "", " ")
	if err != nil {
		return ErrGobiernoUsuariosAdmin
	}
	// Las claves primero; el overlay, que las referencia, el último. Si algo
	// falla a medias la carpeta queda sin overlay y no se reutiliza.
	for _, a := range audiencias {
		if escribirPrivadoGobiernoUsuarios(salida, archivos[a], secretos[a]) != nil {
			return ErrGobiernoUsuariosAdmin
		}
	}
	return escribirPrivadoGobiernoUsuarios(salida, ArchivoOverlayVersionBolsa, append(b, '\n'))
}

func validarDestinoOverlayVersionBolsa(d DestinoOverlayVersionBolsaAdmin) error {
	rutas := []string{d.DirectorioSalida, d.PoolGobierno, d.PoolCatalogo}
	for _, r := range rutas {
		if r == "" || !filepath.IsAbs(r) || filepath.Clean(r) != r || strings.ContainsRune(r, 0) {
			return ErrGobiernoUsuariosAdmin
		}
	}
	// Los pools son ficheros propios de los LOGIN B1, distintos entre sí y
	// fuera de los nombres que deja esta herramienta en la carpeta del día.
	if d.PoolGobierno == d.PoolCatalogo || filepath.Dir(d.PoolGobierno) == d.DirectorioSalida || filepath.Dir(d.PoolCatalogo) == d.DirectorioSalida {
		return ErrGobiernoUsuariosAdmin
	}
	if len(d.Motivos) != 2 {
		return ErrGobiernoUsuariosAdmin
	}
	catalogo := ""
	for _, a := range []string{gobiernoperfiles.AudienciaVersionarRolBolsaProponer, gobiernoperfiles.AudienciaVersionarRolBolsaAprobar} {
		m, ok := d.Motivos[a]
		if !ok || !domain.ReferenciaMotivoAutorizacionV2Valida(m) || (catalogo != "" && m.CatalogoID != catalogo) {
			return ErrGobiernoUsuariosAdmin
		}
		catalogo = m.CatalogoID
	}
	return nil
}
