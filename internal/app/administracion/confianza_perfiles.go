package administracion

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	postgres "vec-diputacion-granada/internal/vec/adapters/postgres"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	AudienciaPerfilesOrdinarioV3        = "vec_autorizacion.administracion_perfiles.ordinario.v1"
	AudienciaPerfilesPropuestaV3        = "vec_autorizacion.administracion_perfiles.propuesta.v1"
	AudienciaPerfilesCierreV3           = "vec_autorizacion.administracion_perfiles.cierre.v1"
	AudienciaPerfilesConsultaV3         = "vec_autorizacion.administracion_perfiles.consulta.v1"
	AudienciaPerfilesCapacidadesV3      = "vec_autorizacion.administracion_perfiles.lectura.consultar_capacidades.v1"
	AudienciaPerfilesBuscarPersonasV3   = "vec_autorizacion.administracion_perfiles.lectura.buscar_personas.v1"
	AudienciaPerfilesPersonaV3          = "vec_autorizacion.administracion_perfiles.lectura.consultar_persona.v1"
	AudienciaPerfilesRolesV3            = "vec_autorizacion.administracion_perfiles.lectura.listar_roles.v1"
	AudienciaPerfilesPropuestasV3       = "vec_autorizacion.administracion_perfiles.lectura.listar_propuestas.v1"
	AudienciaPerfilesPropuestaLecturaV3 = "vec_autorizacion.administracion_perfiles.lectura.consultar_propuesta.v1"
	AudienciaPerfilesReciboV3           = "vec_autorizacion.administracion_perfiles.lectura.consultar_recibo.v1"
)

// MaterialCapacidadPerfilesV3 recibe exclusivamente la clave ya aprovisionada
// de una audiencia ADMIN. La fábrica no genera ni deriva claves.
type MaterialCapacidadPerfilesV3 struct {
	Audiencia, ClaveID, EmisorID, HuellaGobierno string
	Version, RevisionGobierno                    uint64
	Material                                     []byte
	ValidaDesde, ValidaHasta                     time.Time
	Estado                                       confianza.EstadoClaveHMACCapacidadAtestacionV3
	RevocadaEn                                   time.Time
}

func (MaterialCapacidadPerfilesV3) String() string {
	return "administracion.MaterialCapacidadPerfilesV3{[MATERIAL-OCULTO]}"
}
func (m MaterialCapacidadPerfilesV3) GoString() string { return m.String() }
func (m MaterialCapacidadPerfilesV3) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, m.String())
}
func (MaterialCapacidadPerfilesV3) MarshalJSON() ([]byte, error) {
	return []byte(`{"material":"oculto"}`), nil
}
func (m MaterialCapacidadPerfilesV3) MarshalText() ([]byte, error) { return []byte(m.String()), nil }
func (m MaterialCapacidadPerfilesV3) LogValue() slog.Value         { return slog.StringValue(m.String()) }

// ConfiguracionConfianzaPerfilesV3 contiene una instantánea del gobierno y
// material privado leído por la composición. No acepta valores de HTTP.
type ConfiguracionConfianzaPerfilesV3 struct {
	Cabecera          domain.CabeceraAtestacionAutorizacionV3
	Raiz              MaterialRaizPerfilesV3
	Gobierno          GobiernoConfianzaPerfilesV3
	EntradasCapacidad []MaterialCapacidadPerfilesV3
}

// MaterialRaizPerfilesV3 contiene sólo coordenadas públicas del firmante.
type MaterialRaizPerfilesV3 struct {
	ClaveID, Audiencia                   string
	Version                              uint64
	Publica                              ed25519.PublicKey
	Estado                               confianza.EstadoClaveAtestacionAutorizacionV3
	ValidaDesde, ValidaHasta, RevocadaEn time.Time
}

type GobiernoConfianzaPerfilesV3 struct {
	Revision, HuellaSHA256 string
	Secuencia              uint64
	PublicadaEn, ExpiraEn  time.Time
}

func (ConfiguracionConfianzaPerfilesV3) String() string {
	return "administracion.ConfiguracionConfianzaPerfilesV3{[MATERIAL-OCULTO]}"
}
func (c ConfiguracionConfianzaPerfilesV3) GoString() string { return c.String() }
func (c ConfiguracionConfianzaPerfilesV3) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, c.String())
}
func (ConfiguracionConfianzaPerfilesV3) MarshalJSON() ([]byte, error) {
	return []byte(`{"material":"oculto"}`), nil
}
func (c ConfiguracionConfianzaPerfilesV3) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}
func (c ConfiguracionConfianzaPerfilesV3) LogValue() slog.Value { return slog.StringValue(c.String()) }

// DependenciasConfianzaPerfilesV3 mantiene separados los tres roles centrales.
// El borde abre y verifica los pools y conserva su cierre y el del firmante.
type DependenciasConfianzaPerfilesV3 struct {
	PoolFuente, PoolRegistro, PoolMotivos *pgxpool.Pool
	CatalogoMotivosID                     string
	Firmante                              ports.FirmanteAtestacionesAutorizacionV3
	Reloj                                 ports.Reloj
	Generador                             ports.GeneradorReferenciaDecisionAutorizacion
	VigenciaDecision                      time.Duration
}

type ConfianzaPerfilesV3 struct {
	Emisores map[string]*confianza.EmisorMaterialAutorizacionAtestadaV3
	Fuente   ports.FuenteAutorizacion
}

// NuevaConfianzaPerfilesV3 compone el PDP central durable, la atestación COSE,
// su verificador y las once capacidades nominales. No abre conexiones,
// publica gobierno ni registra decisiones durante la construcción.
func NuevaConfianzaPerfilesV3(cfg ConfiguracionConfianzaPerfilesV3, deps DependenciasConfianzaPerfilesV3) (ConfianzaPerfilesV3, error) {
	return nuevaConfianzaConAudienciasV3(cfg, deps, audienciasConfianzaPerfilesV3())
}

// La selección es privada y sólo procede de las dos factorías de conjuntos
// cerrados. Configuración y HTTP no pueden ampliar sus audiencias.
func nuevaConfianzaConAudienciasV3(cfg ConfiguracionConfianzaPerfilesV3, deps DependenciasConfianzaPerfilesV3, requeridas []string) (ConfianzaPerfilesV3, error) {
	vacia := ConfianzaPerfilesV3{}
	if deps.PoolFuente == nil || deps.PoolRegistro == nil || deps.PoolMotivos == nil ||
		deps.PoolFuente == deps.PoolRegistro || deps.PoolFuente == deps.PoolMotivos || deps.PoolRegistro == deps.PoolMotivos ||
		dependenciaConfianzaPerfilesNula(deps.Reloj) || dependenciaConfianzaPerfilesNula(deps.Generador) || dependenciaConfianzaPerfilesNula(deps.Firmante) ||
		deps.VigenciaDecision <= 0 || deps.VigenciaDecision > domain.VigenciaMaximaDecisionAutorizacion ||
		deps.VigenciaDecision%time.Microsecond != 0 || cfg.Cabecera.Validar() != nil ||
		cfg.Cabecera.Suite != confianza.SuiteAtestacionAutorizacionV3COSEEdDSA {
		return vacia, ErrConfiguracion
	}
	ahora := deps.Reloj.Ahora()
	configuracion, err := configuracionConfianzaPerfilesV3(cfg, ahora)
	if err != nil {
		return vacia, ErrConfiguracion
	}
	claves, err := clavesConfianzaConAudienciasV3(cfg.EntradasCapacidad, cfg.Cabecera.ClaveID, ahora, requeridas)
	if err != nil {
		return vacia, ErrConfiguracion
	}
	fuente, err := postgres.NuevoAlmacenAutorizacion(deps.PoolFuente)
	if err != nil {
		return vacia, ErrConfiguracion
	}
	registro, err := postgres.NuevoAlmacenAutorizacion(deps.PoolRegistro)
	if err != nil {
		return vacia, ErrConfiguracion
	}
	motivos, err := postgres.NuevoValidadorReferenciaMotivoPostgreSQLV2(deps.PoolMotivos, deps.CatalogoMotivosID)
	if err != nil {
		return vacia, ErrConfiguracion
	}
	pdp, err := application.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro, motivos, deps.Reloj, deps.Generador, application.ConfiguracionServicioAutorizacion{VigenciaDecision: deps.VigenciaDecision})
	if err != nil {
		return vacia, ErrConfiguracion
	}
	atestador, err := application.NuevoServicioAtestacionesAutorizacionV3(cfg.Cabecera, deps.Firmante)
	if err != nil {
		return vacia, ErrConfiguracion
	}
	verificador, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(configuracion, deps.Reloj)
	if err != nil {
		return vacia, ErrConfiguracion
	}
	emisores := make(map[string]*confianza.EmisorMaterialAutorizacionAtestadaV3, len(claves))
	for audiencia, clave := range claves {
		capacidad, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, deps.Reloj)
		if err != nil {
			return vacia, ErrConfiguracion
		}
		emisor, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(pdp, atestador, verificador, capacidad)
		if err != nil {
			return vacia, ErrConfiguracion
		}
		emisores[audiencia] = emisor
	}
	return ConfianzaPerfilesV3{Emisores: emisores, Fuente: fuente}, nil
}

func configuracionConfianzaPerfilesV3(cfg ConfiguracionConfianzaPerfilesV3, ahora time.Time) (confianza.ConfiguracionConfianzaAtestacionAutorizacionV3, error) {
	vacia := confianza.ConfiguracionConfianzaAtestacionAutorizacionV3{}
	r, g := cfg.Raiz, cfg.Gobierno
	if cfg.Cabecera.ClaveID != r.ClaveID || cfg.Cabecera.Audiencia != r.Audiencia ||
		r.Estado != confianza.EstadoClaveAtestacionAutorizacionV3Activa ||
		ahora.Before(r.ValidaDesde) || !ahora.Before(r.ValidaHasta) ||
		ahora.Before(g.PublicadaEn) || !ahora.Before(g.ExpiraEn) {
		return vacia, ErrConfiguracion
	}
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(r.ClaveID, r.Version, r.Publica, r.Audiencia, r.Estado, r.ValidaDesde, r.ValidaHasta, r.RevocadaEn)
	if err != nil {
		return vacia, ErrConfiguracion
	}
	configuracion, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(g.Revision, g.Secuencia, g.PublicadaEn, g.ExpiraEn, raiz)
	if err != nil || configuracion.ValidarHuellaSHA256Esperada(g.HuellaSHA256) != nil {
		return vacia, ErrConfiguracion
	}
	return configuracion, nil
}

func audienciasConfianzaPerfilesV3() []string {
	return []string{AudienciaPerfilesOrdinarioV3, AudienciaPerfilesPropuestaV3, AudienciaPerfilesCierreV3, AudienciaPerfilesConsultaV3,
		AudienciaPerfilesCapacidadesV3, AudienciaPerfilesBuscarPersonasV3, AudienciaPerfilesPersonaV3,
		AudienciaPerfilesRolesV3, AudienciaPerfilesPropuestasV3, AudienciaPerfilesPropuestaLecturaV3, AudienciaPerfilesReciboV3}
}

func clavesConfianzaPerfilesV3(entradas []MaterialCapacidadPerfilesV3, raizID string, ahora time.Time) (map[string]confianza.ClaveHMACCapacidadAtestacionV3, error) {
	return clavesConfianzaConAudienciasV3(entradas, raizID, ahora, audienciasConfianzaPerfilesV3())
}

func clavesConfianzaConAudienciasV3(entradas []MaterialCapacidadPerfilesV3, raizID string, ahora time.Time, requeridas []string) (map[string]confianza.ClaveHMACCapacidadAtestacionV3, error) {
	if len(entradas) != len(requeridas) || len(requeridas) == 0 {
		return nil, ErrConfiguracion
	}
	admitidas := make(map[string]bool, len(requeridas))
	for _, audiencia := range requeridas {
		if audiencia == "" || admitidas[audiencia] {
			return nil, ErrConfiguracion
		}
		admitidas[audiencia] = true
	}
	claves := make(map[string]confianza.ClaveHMACCapacidadAtestacionV3, len(requeridas))
	identificadores := map[string]bool{raizID: true}
	for i, entrada := range entradas {
		if !admitidas[entrada.Audiencia] {
			return nil, ErrConfiguracion
		}
		if _, existe := claves[entrada.Audiencia]; entrada.Estado != confianza.EstadoClaveHMACCapacidadAtestacionV3Emision || existe || identificadores[entrada.ClaveID] || ahora.Before(entrada.ValidaDesde) || !ahora.Before(entrada.ValidaHasta) {
			return nil, ErrConfiguracion
		}
		for j := 0; j < i; j++ {
			if bytes.Equal(entradas[j].Material, entrada.Material) {
				return nil, ErrConfiguracion
			}
		}
		clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(entrada.ClaveID, entrada.Version, entrada.Material, entrada.EmisorID, entrada.Audiencia, entrada.Estado, entrada.ValidaDesde, entrada.ValidaHasta, entrada.RevocadaEn, entrada.RevisionGobierno, entrada.HuellaGobierno)
		if err != nil {
			return nil, ErrConfiguracion
		}
		claves[entrada.Audiencia] = clave
		identificadores[entrada.ClaveID] = true
	}
	return claves, nil
}

func dependenciaConfianzaPerfilesNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
