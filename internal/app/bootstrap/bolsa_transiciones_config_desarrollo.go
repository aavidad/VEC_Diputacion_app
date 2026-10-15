package bootstrap

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const envGobiernoTransicionesBolsa = "VEC_BOLSA_TRANSICIONES_GOBIERNO_CONFIG_JSON"

// La variable privada selecciona referencias ya publicadas. Ningún campo
// del navegador ni este fichero publica perfil, permiso o límite normativo.
type seleccionGobiernoTransicionesBolsa struct {
	CatalogoVersion int                                  `json:"catalogo_version"`
	CatalogoSHA256  string                               `json:"catalogo_sha256"`
	PerfilRef       string                               `json:"perfil_ref"`
	Motivo          dominiovec.ReferenciaEntradaCatalogo `json:"motivo"`
}

func leerSeleccionGobiernoTransicionesBolsa() (seleccionGobiernoTransicionesBolsa, bool) {
	texto, existe := os.LookupEnv(envGobiernoTransicionesBolsa)
	if !existe || texto == "" || len(texto) > 4096 {
		return seleccionGobiernoTransicionesBolsa{}, false
	}
	d := json.NewDecoder(strings.NewReader(texto))
	d.DisallowUnknownFields()
	var s seleccionGobiernoTransicionesBolsa
	if d.Decode(&s) != nil || d.Decode(&struct{}{}) != io.EOF || s.CatalogoVersion < 1 ||
		!huellaCatalogoTransicionesValida(s.CatalogoSHA256) || !perfilActivoSeguridadComunValido(s.PerfilRef) ||
		s.Motivo.Validar() != nil {
		return seleccionGobiernoTransicionesBolsa{}, false
	}
	return s, true
}

func huellaCatalogoTransicionesValida(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func descriptorMaterialGobiernoTransicionesBolsaDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        puertosbolsa.AudienciaPublicarTransicionesSituacion,
		Dominio:          "vec.bolsa.politica-transiciones.publicar.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:bolsa-politica-transiciones-publicar:",
		ProveedorNominal: "proveedor-material-politica-transiciones-bolsa",
	}
}

// componerRutaGobiernoTransicionesBolsaDesarrollo es la única llamada que
// necesita el montaje raíz. Toda dependencia optativa se cierra en esta ruta
// con 503, sin alterar el arranque ni el servicio operativo de situación.
func componerRutaGobiernoTransicionesBolsaDesarrollo(ctx context.Context,
	dependenciasCT *DependenciasCT, alta *dependenciasAltaContratacionTemporalDesarrollo,
	base *preparadorBorradorLlamamientoDesarrollo, pdp *autorizadorComunDesarrollo,
	situacion *aplicacionbolsa.ServicioSituacionParticipacion,
	registrador puertosvec.RegistradorIntentosAuditoria, proceso string,
) vechttp.RutaExacta {
	_, selectorPresente := os.LookupEnv(envGobiernoTransicionesBolsa)
	desactivada := func(etapa string) vechttp.RutaExacta {
		if selectorPresente {
			slog.Warn("bolsa transiciones: edición no disponible", "etapa", etapa)
		}
		return nuevaRutaGobiernoTransicionesBolsaDesarrollo(nil, nil, nil, nil, 0, "", dominiovec.ReferenciaEntradaCatalogo{}, nil, "")
	}
	s, activa := leerSeleccionGobiernoTransicionesBolsa()
	if !activa {
		return desactivada("selector")
	}
	if ctx == nil || dependenciasCT == nil || dependenciasCT.derivador == nil ||
		alta == nil || alta.soporte == nil || alta.postgresql.bolsa == nil || alta.postgresql.gobierno == nil ||
		alta.postgresql.proveedorMaterial == nil || base == nil || base.soporte == nil ||
		base.soporte.soporteCanal == nil || base.soporte.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef != s.PerfilRef ||
		pdp == nil || situacion == nil || registrador == nil || proceso != "vec-rrhh" {
		return desactivada("dependencias")
	}
	sonda, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	var instalada bool
	if err := alta.postgresql.bolsa.QueryRow(sonda, `SELECT
	 EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
	  WHERE n.nspname='vec_catalogos_configurables'
	    AND p.proname='leer_limites_transiciones_situacion_bolsa_v1' AND p.pronargs=2
	    AND p.proargtypes[0]='integer'::regtype AND p.proargtypes[1]='text'::regtype
	    AND pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',p.oid,'EXECUTE'))
	 AND EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
	  WHERE n.nspname='vec_autorizacion_atestada_v3'
	    AND p.proname='registrar_y_consumir_politica_transiciones_bolsa_v3_atestada' AND p.pronargs=10
	    AND pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',p.oid,'EXECUTE'))
	 AND pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_politica_transiciones_gobierno_v1(integer,text)') IS NOT NULL
	 AND pg_catalog.to_regprocedure('vec_bolsa_llamamientos.publicar_politica_transiciones_gobierno_v1(bigint,text[],text,text,text,text,integer,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL`).Scan(&instalada); err != nil || !instalada {
		return desactivada("migraciones")
	}
	var lectura []byte
	if err := alta.postgresql.bolsa.QueryRow(sonda,
		`SELECT vec_bolsa_llamamientos.leer_politica_transiciones_gobierno_v1($1,$2)`,
		s.CatalogoVersion, s.CatalogoSHA256).Scan(&lectura); err != nil {
		return desactivada("catalogo")
	}
	var perfil struct {
		Limites struct {
			UnidadRef string `json:"unidad_ref"`
			AmbitoRef string `json:"ambito_ref"`
		} `json:"limites"`
	}
	if json.Unmarshal(lectura, &perfil) != nil || perfil.Limites.UnidadRef != base.soporte.unidadRef ||
		perfil.Limites.AmbitoRef != base.soporte.ambitoRef {
		return desactivada("ambito")
	}
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(dependenciasCT.derivador, dependenciasCT.reloj.Ahora())
	if err != nil {
		return desactivada("material")
	}
	defer material.borrarCopiasEfimeras()
	material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo([]descriptorMaterialConsumidorV3Desarrollo{
		descriptorMaterialGobiernoTransicionesBolsaDesarrollo()})
	if err != nil {
		return desactivada("descriptor")
	}
	proveedor, err := nuevoProveedorMaterialConsumidorDesarrollo(sonda, alta.postgresql.gobierno,
		material, alta.soporte, dependenciasCT.reloj, catalogo, puertosbolsa.AudienciaPublicarTransicionesSituacion)
	if err != nil {
		return desactivada("proveedor")
	}
	emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(pdp, proveedor)
	if err != nil {
		return desactivada("emisor")
	}
	return nuevaRutaGobiernoTransicionesBolsaDesarrollo(alta.postgresql.bolsa, base, emisor, situacion,
		s.CatalogoVersion, s.CatalogoSHA256, s.Motivo, registrador, proceso)
}
