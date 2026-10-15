package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

const archivoActivacionOfertaCTDesarrollo = "identidad/ofertas-ct.json"
const esquemaActivacionOfertaCTDesarrollo = "vec.ct.ofertas.activacion.v1"

var errActivacionOfertaCTDesarrollo = errors.New("oferta CT: activacion privada no disponible")
var huellaRolOfertaCTDesarrollo = regexp.MustCompile(`^[a-f0-9]{64}$`)

type referenciaRolOfertaCTDesarrollo struct {
	RolID        string `json:"rol_id"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type archivoActivacionOfertaCT struct {
	Esquema                    string                          `json:"esquema"`
	PerfilRef                  string                          `json:"perfil_ref"`
	Rol                        referenciaRolOfertaCTDesarrollo `json:"rol"`
	Motivo                     core.ReferenciaEntradaCatalogo  `json:"motivo"`
	FuenteAutorizacionDSNFile  string                          `json:"fuente_autorizacion_dsn_file"`
	MotivosAutorizacionDSNFile string                          `json:"motivos_autorizacion_dsn_file"`
}

func leerActivacionOfertaCTDesarrollo(cfg config.Config) (archivoActivacionOfertaCT, bool, error) {
	var vacia archivoActivacionOfertaCT
	if !cfg.DevelopmentEnabledByDoubleKey() || !filepath.IsAbs(cfg.DevelopmentMaterialDir) ||
		validarArbolMaterialDesarrollo(cfg.DevelopmentMaterialDir) != nil {
		return vacia, false, errActivacionOfertaCTDesarrollo
	}
	raiz, err := os.OpenRoot(cfg.DevelopmentMaterialDir)
	if err != nil {
		return vacia, false, errActivacionOfertaCTDesarrollo
	}
	defer raiz.Close()
	if _, err := raiz.Lstat(archivoActivacionOfertaCTDesarrollo); errors.Is(err, os.ErrNotExist) {
		return vacia, false, nil
	} else if err != nil {
		return vacia, false, errActivacionOfertaCTDesarrollo
	}
	b, err := leerArchivoIncorporacionV2(raiz, archivoActivacionOfertaCTDesarrollo, 16<<10)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return vacia, true, errActivacionOfertaCTDesarrollo
	}
	defer borrarBytes(b)
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var c archivoActivacionOfertaCT
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF ||
		c.Esquema != esquemaActivacionOfertaCTDesarrollo ||
		!domain.ReferenciaOpacaValida(c.PerfilRef) ||
		!domain.ReferenciaOpacaValida(c.Rol.RolID) || c.Rol.Version < 1 ||
		!huellaRolOfertaCTDesarrollo.MatchString(c.Rol.HuellaSHA256) ||
		c.Motivo.Validar() != nil ||
		!filepath.IsLocal(c.FuenteAutorizacionDSNFile) ||
		!filepath.IsLocal(c.MotivosAutorizacionDSNFile) ||
		c.FuenteAutorizacionDSNFile == c.MotivosAutorizacionDSNFile {
		return vacia, true, errActivacionOfertaCTDesarrollo
	}
	return c, true, nil
}

func leerDSNOfertaCTDesarrollo(cfg config.Config, archivo string) (string, error) {
	if !filepath.IsLocal(archivo) || validarArbolMaterialDesarrollo(cfg.DevelopmentMaterialDir) != nil {
		return "", errActivacionOfertaCTDesarrollo
	}
	raiz, err := os.OpenRoot(cfg.DevelopmentMaterialDir)
	if err != nil {
		return "", errActivacionOfertaCTDesarrollo
	}
	defer raiz.Close()
	b, err := leerArchivoIncorporacionV2(raiz, archivo, 16<<10)
	if err != nil {
		return "", errActivacionOfertaCTDesarrollo
	}
	defer borrarBytes(b)
	if len(bytes.TrimSpace(b)) == 0 {
		return "", errActivacionOfertaCTDesarrollo
	}
	return string(bytes.TrimSpace(b)), nil
}

func descriptorMaterialSeleccionOfertaCTDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        ports.AudienciaVincularOfertaBolsaCT,
		Dominio:          "vec.ct.vinculo-oferta-bolsa.capacidad-v3",
		Prefijo:          "clave:capacidad:ct:vinculo-oferta-bolsa:",
		ProveedorNominal: proveedorMaterialContratacionTemporal,
	}
}

// La sonda es de solo lectura. Un clon anterior a cualquiera de las tres
// migraciones conserva CT operativo y deja cerrada esta única capacidad.
func detectarSeleccionOfertaCTDesarrollo(ctx context.Context, ejecutor *pgxpool.Pool) bool {
	if ctx == nil || ejecutor == nil {
		return false
	}
	var disponible bool
	err := ejecutor.QueryRow(ctx, `SELECT
	  COALESCE(pg_catalog.has_function_privilege(session_user,
	    pg_catalog.to_regprocedure('vec_contratacion_temporal.consultar_seleccion_oferta_ct220(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'),false)
	  AND COALESCE(pg_catalog.has_function_privilege(session_user,
	    pg_catalog.to_regprocedure('vec_contratacion_temporal.reservar_seleccion_oferta_ct220(jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'),false)
	  AND COALESCE(pg_catalog.has_function_privilege(session_user,
	    pg_catalog.to_regprocedure('vec_contratacion_temporal.abrir_ventana_seleccion_oferta_ct220(jsonb,text,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'),false)
	  AND COALESCE(pg_catalog.has_function_privilege(session_user,
	    pg_catalog.to_regprocedure('vec_contratacion_temporal.confirmar_vinculo_seleccion_oferta_ct220(jsonb,text,jsonb,jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'),false)
	  AND COALESCE(pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',
	    pg_catalog.to_regprocedure('vec_bolsa_llamamientos.verificar_oferta_ct_v1(text,text,bigint,text,text,text,text,text,text,integer,jsonb,text)'), 'EXECUTE'),false)
	  AND COALESCE(pg_catalog.has_function_privilege('vec_contratacion_temporal_propietario',
	    pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'), 'EXECUTE'),false)`).Scan(&disponible)
	return err == nil && disponible
}
