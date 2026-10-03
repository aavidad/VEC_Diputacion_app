package main

import (
	"crypto/tls"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type conexionPrivada struct {
	DSN             string `json:"dsn"`
	TimeoutSegundos uint64 `json:"timeout_segundos"`
}
type aprobacionPrivada struct {
	HuellaPlanSHA256 string `json:"huella_plan_sha256"`
}

// aplicarPlan sólo existe en este comando de operador. Ni el proceso ADMIN
// ni su API reciben el pool o la aprobación privada de bootstrap.
func aplicarPlan(plan material, rutaConexion, rutaAprobacion string) (reciboOperador, error) {
	var cero reciboOperador
	b, err := leerPrivado(rutaConexion)
	if err != nil {
		return cero, err
	}
	defer clear(b)
	var cfg conexionPrivada
	if decodificarEstricto(b, &cfg) != nil || cfg.DSN == "" || cfg.TimeoutSegundos == 0 || cfg.TimeoutSegundos > 60 {
		return cero, errors.New("conexion_invalida")
	}
	a, err := leerPrivado(rutaAprobacion)
	if err != nil {
		return cero, err
	}
	defer clear(a)
	var aprobado aprobacionPrivada
	if decodificarEstricto(a, &aprobado) != nil || !domain.HuellaAdministracionPerfilesValida(aprobado.HuellaPlanSHA256) {
		return cero, errors.New("aprobacion_invalida")
	}
	preimagen, err := plan.Preimagen()
	if err != nil || preimagen.HuellaPlanSHA256 != aprobado.HuellaPlanSHA256 {
		return cero, errors.New("aprobacion_divergente")
	}
	pc, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return cero, errors.New("conexion_invalida")
	}
	// Socket local o TLS con verificación; no se admite una conexión remota
	// sin protección ni degradación a un fallback sin verificar el servidor.
	if !conexionBootstrapValida(&pc.ConnConfig.Config) {
		return cero, errors.New("canal_invalido")
	}
	// El proveedor durable y sus migraciones no están compuestos en este corte.
	// Cotejar material privado nunca produce una concesión ni abre una conexión.
	return cero, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}

func canalBootstrapValido(host string, tlsCfg *tls.Config) bool {
	return strings.HasPrefix(host, "/") || tlsCfg != nil && !tlsCfg.InsecureSkipVerify
}

func conexionBootstrapValida(cfg *pgconn.Config) bool {
	if cfg == nil || !canalBootstrapValido(cfg.Host, cfg.TLSConfig) {
		return false
	}
	for _, f := range cfg.Fallbacks {
		if f == nil || !canalBootstrapValido(f.Host, f.TLSConfig) {
			return false
		}
	}
	return true
}

type reciboOperador struct {
	ActoRef           string    `json:"acto_ref"`
	ReciboRef         string    `json:"recibo_ref"`
	HuellaPlanSHA256  string    `json:"huella_plan_sha256"`
	PrimeraPersonaRef string    `json:"primera_persona_ref"`
	SegundaPersonaRef string    `json:"segunda_persona_ref"`
	ConfirmadoEn      time.Time `json:"confirmado_en"`
}
