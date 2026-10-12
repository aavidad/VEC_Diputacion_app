package internactproveedores

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/composicion/documentosbase"
	core "vec-diputacion-granada/internal/vec/domain"
)

var ErrMaterialFirmaR5NoDisponible = errors.New("composicion interna: material original R5 no disponible")

// MaterialFirmaR5 es el contrato privado de ct_firma_r5.json (versión 1):
// catalogo_motivos, motivo_original, capacidad_documentos, documentos_archivo,
// documentos_sha256, plantillas_archivo, plantillas_sha256, plantillas_id,
// plantillas_version, plantillas_huella_sha256, organizacion_archivo,
// organizacion_sha256, rpt_archivo, rpt_sha256, seudonimos_archivo y
// seudonimos_sha256. El fichero 0600 vive junto a ct_v3.json, pero no modifica
// sus seis pools. documentos_archivo es identidad/documentos.json del mismo
// DevelopmentMaterialDir que consume bootstrap; su SHA cubre DSN y conector.
// plantillas_archivo coincide con CTPlantillasSourcePath y la terna publicada
// identifica la versión más reciente del catálogo. seudonimos_archivo apunta
// a 32 bytes ya aprovisionados en el directorio privado; nunca se generan al
// arrancar. No contiene otra raíz, PDP, identidad ni clave de atestación V3.
// La asignación administrativa del perfil F1 debe conceder por separado
// documentos.original.descargar, documentos.original_firmable.reservar,
// documentos.original_firmable.confirmar y
// documentos.original_firmable.almacen.escribir para la organización exacta.
// El runtime conserva la garantía real F1; una guarda high deniega antes del
// efecto. La vía de registro de firma conserva su requisito high independiente.
type MaterialFirmaR5 struct {
	Version                int                            `json:"version"`
	CatalogoMotivos        string                         `json:"catalogo_motivos"`
	MotivoOriginal         core.ReferenciaEntradaCatalogo `json:"motivo_original"`
	CapacidadDocumentos    capacidadMaterial              `json:"capacidad_documentos"`
	DocumentoArchivo       string                         `json:"documentos_archivo"`
	DocumentoSHA256        string                         `json:"documentos_sha256"`
	PlantillasArchivo      string                         `json:"plantillas_archivo"`
	PlantillasSHA256       string                         `json:"plantillas_sha256"`
	PlantillasID           string                         `json:"plantillas_id"`
	PlantillasVersion      uint64                         `json:"plantillas_version"`
	PlantillasHuellaSHA256 string                         `json:"plantillas_huella_sha256"`
	OrganizacionArchivo    string                         `json:"organizacion_archivo"`
	OrganizacionSHA256     string                         `json:"organizacion_sha256"`
	RPTArchivo             string                         `json:"rpt_archivo"`
	RPTSHA256              string                         `json:"rpt_sha256"`
	SeudonimosArchivo      string                         `json:"seudonimos_archivo"`
	SeudonimosSHA256       string                         `json:"seudonimos_sha256"`
}

func AbrirPoolDocumentosFirmaR5(ctx context.Context, d DocumentoFirmaR5) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(d.Pool.DSN)
	if err != nil || cfg.ConnConfig == nil || cfg.ConnConfig.User == "" {
		return nil, ErrMaterialFirmaR5NoDisponible
	}
	d.Pool.Login = cfg.ConnConfig.User
	return abrirPool(ctx, d.Pool, perfilPool{"documentos_original_r5", "vec_documentos_ejecutor",
		"vec_documentos.reservar_original_firmable_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"})
}

func (MaterialFirmaR5) String() string               { return "[material firma R5 privado]" }
func (m MaterialFirmaR5) GoString() string           { return m.String() }
func (m MaterialFirmaR5) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(m.String())) }
func (m MaterialFirmaR5) LogValue() slog.Value       { return slog.StringValue(m.String()) }
func (MaterialFirmaR5) MarshalJSON() ([]byte, error) {
	return []byte(`"[material firma R5 privado]"`), nil
}

// CargarMaterialFirmaR5 usa la raiz privada ya acreditada del inventario CT.
// La ausencia de este fichero mantiene apagada la ruta del original.
func CargarMaterialFirmaR5(base Material) (MaterialFirmaR5, error) {
	var m MaterialFirmaR5
	if base.raiz == nil {
		return m, ErrMaterialFirmaR5NoDisponible
	}
	b, err := leerArchivoPrivado(base.raiz, "ct_firma_r5.json", maximoInventarioCT)
	if err != nil {
		return m, ErrMaterialFirmaR5NoDisponible
	}
	defer clear(b)
	if clavesJSONDuplicadas(b) {
		return m, ErrMaterialFirmaR5NoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil || d.Decode(new(any)) != io.EOF || m.Version != 1 ||
		!core.ReferenciaMotivoAutorizacionV2Valida(m.MotivoOriginal) ||
		m.CatalogoMotivos == "" || m.MotivoOriginal.CatalogoID != m.CatalogoMotivos ||
		m.DocumentoArchivo != "identidad/documentos.json" ||
		!huellaFirmaR5Valida(m.DocumentoSHA256) || !huellaFirmaR5Valida(m.PlantillasSHA256) ||
		!huellaFirmaR5Valida(m.PlantillasHuellaSHA256) ||
		!huellaFirmaR5Valida(m.OrganizacionSHA256) || !huellaFirmaR5Valida(m.RPTSHA256) ||
		m.PlantillasID == "" || m.PlantillasVersion == 0 || m.PlantillasArchivo == "" ||
		filepath.Clean(m.PlantillasArchivo) != m.PlantillasArchivo ||
		m.OrganizacionArchivo == "" || filepath.Clean(m.OrganizacionArchivo) != m.OrganizacionArchivo ||
		m.RPTArchivo == "" || filepath.Clean(m.RPTArchivo) != m.RPTArchivo ||
		!filepath.IsLocal(m.SeudonimosArchivo) || m.SeudonimosArchivo == "." ||
		!huellaFirmaR5Valida(m.SeudonimosSHA256) ||
		!filepath.IsLocal(m.CapacidadDocumentos.Archivo) || m.CapacidadDocumentos.Archivo == "." {
		return MaterialFirmaR5{}, ErrMaterialFirmaR5NoDisponible
	}
	return m, nil
}

func huellaFirmaR5Valida(v string) bool {
	if len(v) != 64 || strings.Trim(v, "0") == "" {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil && v == strings.ToLower(v)
}

// CargarClaveSeudonimosFirmaR5 exige una clave ya aprovisionada, nunca la
// genera al arrancar. El llamante la borra tras construir la autoridad.
func CargarClaveSeudonimosFirmaR5(base Material, m MaterialFirmaR5) ([]byte, error) {
	if base.raiz == nil || !filepath.IsLocal(m.SeudonimosArchivo) || !huellaFirmaR5Valida(m.SeudonimosSHA256) {
		return nil, ErrMaterialFirmaR5NoDisponible
	}
	clave, err := leerConHuella(base.raiz, m.SeudonimosArchivo, m.SeudonimosSHA256, sha256.Size)
	if err != nil || len(clave) != sha256.Size {
		clear(clave)
		return nil, ErrMaterialFirmaR5NoDisponible
	}
	return clave, nil
}

type DocumentoFirmaR5 struct {
	Pool    PoolMaterial
	Almacen documentosbase.ConfiguracionAlmacen
}

func (DocumentoFirmaR5) String() string               { return "[configuración Documentos privada]" }
func (d DocumentoFirmaR5) GoString() string           { return d.String() }
func (d DocumentoFirmaR5) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(d.String())) }
func (d DocumentoFirmaR5) LogValue() slog.Value       { return slog.StringValue(d.String()) }
func (DocumentoFirmaR5) MarshalJSON() ([]byte, error) {
	return []byte(`"[configuración Documentos privada]"`), nil
}

// ExtraerDocumentoFirmaR5 exige la huella del fichero canonico de bootstrap.
// Solo toma el LOGIN ejecutor y la configuracion del conector; no interpreta
// cuentas ni concede permiso a la ruta.
func ExtraerDocumentoFirmaR5(contenido []byte, esperado string) (DocumentoFirmaR5, error) {
	var vacio DocumentoFirmaR5
	if len(contenido) == 0 || len(contenido) > 256<<10 || !huellaFirmaR5Valida(esperado) {
		return vacio, ErrMaterialFirmaR5NoDisponible
	}
	suma := sha256.Sum256(contenido)
	if hex.EncodeToString(suma[:]) != esperado || clavesJSONDuplicadas(contenido) {
		return vacio, ErrMaterialFirmaR5NoDisponible
	}
	var d struct {
		Version   int                                 `json:"version"`
		Autoridad string                              `json:"autoridad"`
		DSN       string                              `json:"dsn_documentos"`
		Almacen   documentosbase.ConfiguracionAlmacen `json:"almacen"`
	}
	if json.Unmarshal(contenido, &d) != nil || d.Version != 1 || d.Autoridad != "no_autoritativo" || d.DSN == "" ||
		strings.TrimSpace(d.DSN) != d.DSN {
		return vacio, ErrMaterialFirmaR5NoDisponible
	}
	return DocumentoFirmaR5{Pool: PoolMaterial{DSN: d.DSN}, Almacen: d.Almacen}, nil
}
