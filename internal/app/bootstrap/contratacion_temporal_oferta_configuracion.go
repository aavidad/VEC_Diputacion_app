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

type perfilActivacionOfertaCT struct {
	PerfilRef string                          `json:"perfil_ref"`
	Rol       referenciaRolOfertaCTDesarrollo `json:"rol"`
	Motivo    core.ReferenciaEntradaCatalogo  `json:"motivo"`
}

type archivoActivacionOfertaCT struct {
	Esquema                    string                   `json:"esquema"`
	Vinculo                    perfilActivacionOfertaCT `json:"vinculo"`
	CierreBolsa                perfilActivacionOfertaCT `json:"cierre_bolsa"`
	DescarteCT                 perfilActivacionOfertaCT `json:"descarte_ct"`
	FuenteAutorizacionDSNFile  string                   `json:"fuente_autorizacion_dsn_file"`
	MotivosAutorizacionDSNFile string                   `json:"motivos_autorizacion_dsn_file"`
}

func perfilActivacionOfertaCTValido(p perfilActivacionOfertaCT) bool {
	return domain.ReferenciaOpacaValida(p.PerfilRef) &&
		domain.ReferenciaOpacaValida(p.Rol.RolID) && p.Rol.Version >= 1 &&
		huellaRolOfertaCTDesarrollo.MatchString(p.Rol.HuellaSHA256) && p.Motivo.Validar() == nil
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
		!perfilActivacionOfertaCTValido(c.Vinculo) ||
		!perfilActivacionOfertaCTValido(c.CierreBolsa) ||
		!perfilActivacionOfertaCTValido(c.DescarteCT) ||
		c.Vinculo.PerfilRef == c.CierreBolsa.PerfilRef ||
		c.Vinculo.PerfilRef == c.DescarteCT.PerfilRef ||
		c.CierreBolsa.PerfilRef == c.DescarteCT.PerfilRef ||
		c.Vinculo.Rol.RolID == c.CierreBolsa.Rol.RolID ||
		c.Vinculo.Rol.RolID == c.DescarteCT.Rol.RolID ||
		c.CierreBolsa.Rol.RolID == c.DescarteCT.Rol.RolID ||
		c.Vinculo.Motivo == c.CierreBolsa.Motivo ||
		c.Vinculo.Motivo == c.DescarteCT.Motivo ||
		c.CierreBolsa.Motivo == c.DescarteCT.Motivo ||
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

func descriptorMaterialCierreIntencionOfertaCTDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        "vec_bolsa_llamamientos.oferta_ct.intencion_cerrar.v1",
		Dominio:          "vec.bolsa.oferta-ct.cierre-intencion.capacidad-v3",
		Prefijo:          "clave:capacidad:ct:bolsa-oferta-cerrar-intencion:",
		ProveedorNominal: proveedorMaterialContratacionTemporal,
	}
}

func descriptorMaterialDescarteSeleccionOfertaCTDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        "vec_contratacion_temporal.seleccion_oferta.descartar.v1",
		Dominio:          "vec.ct.seleccion-oferta.descartar.capacidad-v3",
		Prefijo:          "clave:capacidad:ct:seleccion-oferta-descartar:",
		ProveedorNominal: proveedorMaterialContratacionTemporal,
	}
}

// La sonda es de solo lectura. Un clon anterior a cualquiera de las tres
// migraciones conserva CT operativo y deja cerrada esta única capacidad.
// pg_catalog permite comprobar firmas de Bolsa y AD278 sin conceder USAGE de
// esos esquemas al login CT. Cada fila fija nombre, tipos y rol consumidor.
const consultaPreflightSeleccionOfertaCTDesarrollo = `WITH tipos AS (
 SELECT 'pg_catalog.jsonb'::regtype::oid AS j,
        'pg_catalog.text'::regtype::oid AS t,
        'pg_catalog.bytea'::regtype::oid AS b,
        'pg_catalog.numeric'::regtype::oid AS n,
        'pg_catalog.int8'::regtype::oid AS g,
        'pg_catalog.int4'::regtype::oid AS i
), requeridas AS (
 SELECT v.rol, v.esquema, v.nombre, v.argumentos
 FROM tipos AS x
 CROSS JOIN LATERAL (VALUES
  (session_user::text, 'vec_contratacion_temporal', 'consultar_seleccion_oferta_ct220',
   ARRAY[x.j,x.t] || ARRAY[x.b,x.b,x.b,x.b,x.n,x.n,x.b,x.b,x.b,x.b]),
  (session_user::text, 'vec_contratacion_temporal', 'reservar_seleccion_oferta_ct220',
   ARRAY[x.j,x.j,x.t] || ARRAY[x.b,x.b,x.b,x.b,x.n,x.n,x.b,x.b,x.b,x.b]),
  (session_user::text, 'vec_contratacion_temporal', 'abrir_ventana_seleccion_oferta_ct220',
   ARRAY[x.j,x.t,x.j,x.t] || ARRAY[x.b,x.b,x.b,x.b,x.n,x.n,x.b,x.b,x.b,x.b]),
  (session_user::text, 'vec_contratacion_temporal', 'confirmar_vinculo_seleccion_oferta_ct220',
   ARRAY[x.j,x.t,x.j,x.j,x.t] || ARRAY[x.b,x.b,x.b,x.b,x.n,x.n,x.b,x.b,x.b,x.b]),
  ('vec_contratacion_temporal_propietario', 'vec_bolsa_llamamientos', 'verificar_oferta_ct_v1',
   ARRAY[x.t,x.t,x.g,x.t,x.t,x.t,x.t,x.t,x.t,x.i,x.j,x.t]),
  ('vec_contratacion_temporal_propietario', 'vec_autorizacion_atestada_v3',
   'registrar_y_consumir_vinculo_oferta_bolsa_ct_v3_atestada',
   ARRAY[x.b,x.b,x.b,x.b,x.n,x.n,x.b,x.b,x.b,x.b]),
  ('vec_bolsa_llamamientos_ejecutor', 'vec_bolsa_llamamientos',
   'recuperar_o_cerrar_intencion_oferta_ct_v1',
   ARRAY[x.j,x.t,x.t,x.t,x.t,x.b,x.b,x.b,x.b,x.n,x.n,x.b,x.b,x.b,x.b]),
  ('vec_bolsa_llamamientos_propietario', 'vec_autorizacion_atestada_v3',
   'registrar_y_consumir_cierre_intencion_oferta_ct_v3_atestada',
   ARRAY[x.b,x.b,x.b,x.b,x.n,x.n,x.b,x.b,x.b,x.b]),
  ('vec_contratacion_temporal_propietario', 'vec_bolsa_llamamientos',
   'verificar_cierre_intencion_oferta_ct_v1', ARRAY[x.j,x.j]),
  (session_user::text, 'vec_contratacion_temporal', 'descartar_seleccion_oferta_ct220',
   ARRAY[x.j,x.t,x.j,x.j,x.t] || ARRAY[x.b,x.b,x.b,x.b,x.n,x.n,x.b,x.b,x.b,x.b]),
  ('vec_contratacion_temporal_propietario', 'vec_autorizacion_atestada_v3',
   'registrar_y_consumir_descarte_seleccion_oferta_ct_v3_atestada',
   ARRAY[x.b,x.b,x.b,x.b,x.n,x.n,x.b,x.b,x.b,x.b])
 ) AS v(rol,esquema,nombre,argumentos)
)
SELECT count(*)=11 AND count(p.oid)=11 AND
 coalesce(bool_and(coalesce(pg_catalog.has_function_privilege(r.rol,p.oid,'EXECUTE'),false)),false)
FROM requeridas AS r
LEFT JOIN pg_catalog.pg_namespace AS n ON n.nspname=r.esquema
LEFT JOIN pg_catalog.pg_proc AS p ON p.pronamespace=n.oid AND p.proname=r.nombre
 AND p.prokind='f' AND p.proargtypes::text=pg_catalog.array_to_string(r.argumentos,' ')`

func detectarSeleccionOfertaCTDesarrollo(ctx context.Context, ejecutor *pgxpool.Pool) bool {
	if ctx == nil || ejecutor == nil {
		return false
	}
	var disponible bool
	err := ejecutor.QueryRow(ctx, consultaPreflightSeleccionOfertaCTDesarrollo).Scan(&disponible)
	return err == nil && disponible
}
