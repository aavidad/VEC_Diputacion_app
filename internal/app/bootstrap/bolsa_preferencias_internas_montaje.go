package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	bolsapg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	mibolsa "vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
)

const envBolsaPreferenciasInternas = "VEC_BOLSA_PREFERENCIAS_INTERNA_ENABLED"
const loginBolsaPreferenciasInternas = "vec_interno_bolsa_preferencias_desarrollo"
const rolBolsaPreferenciasInternas = "vec_bolsa_llamamientos_preferencias_interna"

type configuracionBolsaPreferenciasInternas struct {
	Version           int                                 `json:"version"`
	Superficie        core.SuperficieAutenticacionActorV1 `json:"superficie"`
	DSNBolsa          string                              `json:"dsn_bolsa"`
	CertificadoSHA256 string                              `json:"certificado_sha256"`
	Sujeto            string                              `json:"sujeto"`
	CuentaRef         string                              `json:"cuenta_ref"`
	PerfilRef         string                              `json:"perfil_ref"`
	PersonaRef        string                              `json:"persona_ref"`
	CandidatoRef      string                              `json:"candidato_ref"`
}

func (configuracionBolsaPreferenciasInternas) String() string {
	return "[CONFIGURACION PRIVADA BOLSA]"
}
func (configuracionBolsaPreferenciasInternas) GoString() string {
	return "[CONFIGURACION PRIVADA BOLSA]"
}
func (configuracionBolsaPreferenciasInternas) MarshalJSON() ([]byte, error) {
	return []byte(`{"configuracion":"privada"}`), nil
}
func (configuracionBolsaPreferenciasInternas) LogValue() slog.Value {
	return slog.StringValue("[CONFIGURACION PRIVADA BOLSA]")
}

func leerConfiguracionBolsaPreferenciasInternas(cfg config.Config) (configuracionBolsaPreferenciasInternas, error) {
	vacia := configuracionBolsaPreferenciasInternas{}
	b, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "bolsa-preferencias-interna.json"), 16<<10)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return vacia, errMiBolsaNoDisponible
	}
	defer borrarBytes(b)
	var c configuracionBolsaPreferenciasInternas
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var extra any
	if d.Decode(&c) != nil || !errors.Is(d.Decode(&extra), io.EOF) || c.Version != 1 ||
		c.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 || c.DSNBolsa == "" ||
		!huellaPreimagenMiBolsaPortalExterno.MatchString(c.CertificadoSHA256) ||
		c.Sujeto == "" || c.CuentaRef == "" || c.PerfilRef == "" || c.PersonaRef == "" || c.CandidatoRef == "" {
		return vacia, errMiBolsaNoDisponible
	}
	return c, nil
}

func descriptorMaterialOrdenarPreferenciasBolsasInternaV3() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        bolsapuertos.AudienciaOrdenarPreferenciasBolsasInterna,
		Dominio:          "vec.bolsa.preferencias-bolsas.ordenar.interna.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:bolsa-preferencias-bolsas-ordenar-interna:",
		ProveedorNominal: "proveedor-material-bolsa-preferencias-bolsas-ordenar-interna",
	}
}

// preferenciasBolsasInternasSolicitadas decide si vec-server publica la clave
// de la audiencia interna junto a las demás de su gobierno de desarrollo.
func preferenciasBolsasInternasSolicitadas() bool {
	return os.Getenv(envBolsaPreferenciasInternas) == "true"
}

func comprobarACLPreferenciasBolsasInternas(ctx context.Context, pool *pgxpool.Pool) bool {
	if ctx == nil || pool == nil {
		return false
	}
	var valida bool
	err := pool.QueryRow(ctx, `SELECT session_user=$1 AND current_user=session_user
 AND has_schema_privilege(session_user,'vec_bolsa_llamamientos','USAGE')
 AND NOT has_schema_privilege(session_user,'vec_bolsa_llamamientos','CREATE')
 AND to_regprocedure('vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)') IS NOT NULL
 AND to_regprocedure('vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)') IS NOT NULL
 AND to_regprocedure('vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND to_regprocedure('vec_autorizacion.cotejar_lectura_preferencias_bolsas_actual_interna_v1(jsonb,text,text)') IS NOT NULL
 AND to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 AND to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)') IS NOT NULL
 AND to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_interna_v1(bytea,bytea,jsonb)') IS NOT NULL
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c
  WHERE c.conrelid=to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3')
   AND c.conname='auditoria_lectura_preferencias_bolsas_formato_v2'
   AND c.contype='c' AND c.convalidated
   AND strpos(pg_catalog.pg_get_constraintdef(c.oid,false),'interna_corporativa')>0)
 AND has_function_privilege(session_user,'vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)','EXECUTE')
 AND has_function_privilege(session_user,'vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)','EXECUTE')
 AND has_function_privilege(session_user,'vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('public','vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)','EXECUTE')
 AND NOT has_function_privilege('public','vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)','EXECUTE')
 AND NOT has_function_privilege('public','vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')`,
		loginBolsaPreferenciasInternas).Scan(&valida)
	return err == nil && valida
}

// La capacidad es opcional. Un material o una dependencia incompletos dejan
// únicamente esta ruta en 503 y permiten que el resto de la app arranque.
func nuevaRutaPreferenciasBolsasInternas(cfg config.Config,
	identidad *resolvedorIdentidadDesarrollo, usuarios *composicionPreferenciasUsuarios,
	material *proveedorMaterialAltaContratacionTemporalDesarrollo,
) (*vechttp.RutaExacta, func()) {
	seleccion := os.Getenv(envBolsaPreferenciasInternas)
	if seleccion == "" || seleccion == "false" {
		return nil, func() {}
	}
	ruta := &vechttp.RutaExacta{Ruta: bolsahttp.RutaPreferenciasBolsas, Manejador: bolsahttp.NuevoHandlerPreferenciasBolsasNoDisponible()}
	noDisponible := func() (*vechttp.RutaExacta, func()) {
		slog.Warn("preferencias de Mi Bolsa interna no disponibles")
		return ruta, func() {}
	}
	if seleccion != "true" || !cfg.DevelopmentEnabledByDoubleKey() || identidad == nil || identidad.candidatoBolsa == nil ||
		usuarios == nil || usuarios.interna == nil || usuarios.interna.base == nil ||
		usuarios.interna.superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		usuarios.interna.fuente == nil || usuarios.interna.registroAutorizacion == nil || usuarios.interna.motivosPool == nil ||
		material == nil || miBolsaExternaSeleccionadaEnCombinado(cfg) {
		return noDisponible()
	}
	c, err := leerConfiguracionBolsaPreferenciasInternas(cfg)
	if err != nil {
		return noDisponible()
	}
	candidato := identidad.candidatoBolsa
	if candidato.identidad.principal.ID != c.Sujeto ||
		candidato.identidad.principal.Attributes["certificate_sha256"] != c.CertificadoSHA256 ||
		candidato.personaRef != c.PersonaRef || candidato.candidatoRef != c.CandidatoRef ||
		candidato.cuentaRef == c.CuentaRef || candidato.perfilRef == c.PerfilRef {
		return noDisponible()
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	pool, login, err := abrirPoolNominalPortalExterno(ctx, c.DSNBolsa, rolBolsaPreferenciasInternas, "vec-bolsa-preferencias-internas")
	if err != nil {
		return noDisponible()
	}
	cerrar := func() { pool.Close() }
	fallo := func() (*vechttp.RutaExacta, func()) { cerrar(); return noDisponible() }
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, usuarios.interna.motivosPool)
	if err != nil || login != loginBolsaPreferenciasInternas ||
		cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil ||
		usuarios.interna.logins[login] || !comprobarACLPreferenciasBolsasInternas(ctx, pool) {
		return fallo()
	}
	motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(usuarios.interna.motivosPool,
		motivoPortalMiBolsaDesarrollo().CatalogoID)
	if err != nil {
		return fallo()
	}
	reloj := relojContratacionTemporalDesarrollo{}
	autorizador, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(usuarios.interna.fuente,
		usuarios.interna.registroAutorizacion, usuarios.interna.registroAutorizacion, motivos,
		reloj, seguridadvec.GeneradorReferenciasCriptograficas{},
		vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return fallo()
	}
	lecturaPG, err := bolsapg.NuevaLecturaPreferenciasBolsasInternaPostgreSQL(pool)
	if err != nil {
		return fallo()
	}
	auditoriaPG, err := bolsapg.NuevaAuditoriaIntentosPreferenciasBolsasInternaPostgreSQL(pool)
	if err != nil {
		return fallo()
	}
	lectura, err := mibolsa.NuevaLecturaPreferenciasBolsasInterna(lecturaPG, auditoriaPG, usuarios.interna.fuente, reloj)
	if err != nil {
		return fallo()
	}
	escrituraPG, err := bolsapg.NuevaEscrituraPreferenciasBolsasInternaPostgreSQL(pool)
	if err != nil {
		return fallo()
	}
	escritura, err := mibolsa.NuevoServicioPreferenciasBolsasInterna(escrituraPG, autorizador,
		proveedorPreferenciasBolsasInterna{delegado: material}, reloj)
	if err != nil {
		return fallo()
	}
	preparador := &preparadorMiBolsaPortalExterno{preferencias: usuarios.interna,
		identidad: candidato, superficie: core.SuperficieAutenticacionInternaCorporativaV1,
		cuentaInterna: cuentaUsuariosPreferenciasDesarrollo{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{
			CertificadoSHA256: c.CertificadoSHA256, Sujeto: c.Sujeto, CuentaRef: c.CuentaRef, PerfilRef: c.PerfilRef}},
	}
	manejador, err := bolsahttp.NuevoHandlerPreferenciasBolsasConLectura(preparador, escritura, lectura)
	if err != nil {
		return fallo()
	}
	ruta.Manejador = manejador
	return ruta, cerrar
}
