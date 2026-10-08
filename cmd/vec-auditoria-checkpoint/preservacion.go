package main

import (
	"context"
	"crypto/tls"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"strings"
	"time"
	config "vec-diputacion-granada/config/auditoriapreservacion"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	"vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type opcionesPreservacion struct {
	Operacion, Config, Conexion, Entrada string
	Version                              uint64
}

func runPreservacion(o opcionesPreservacion, out, log io.Writer) int {
	fallo := func() int {
		return escribirResultadoPreservacion(out, map[string]string{"modo": "DESARROLLO", "estado": "rechazado", "codigo": "entrada_o_dependencia_invalida"}, 1)
	}
	raw, err := leerRegular(o.Config, 4096, false)
	if err != nil {
		return fallo()
	}
	var cfg config.Ejecutor
	if decodificar(raw, &cfg) != nil || cfg.Validar() != nil {
		return fallo()
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout())
	defer cancel()
	ctx, err = ports.ConCorrelacionIncidenciasPeticion(ctx)
	if err != nil {
		return fallo()
	}
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino: log, Entorno: "desarrollo", VersionBinario: cfg.VersionBinario, Capacidad: 8})
	if err != nil {
		return fallo()
	}
	codigo := domain.ResultadoTecnicoNoDisponible
	defer func() {
		emisor.EmitirResultadoConContexto(ctx, domain.SolicitudResultadoTecnico{Resultado: codigo, Componente: domain.ComponenteIncidenciaAuditoria, Etapa: domain.EtapaIncidenciaEmision})
		c, cancelar := context.WithTimeout(context.Background(), time.Second)
		defer cancelar()
		_ = emisor.Cerrar(c)
	}()
	pool, err := poolPreservacion(ctx, o)
	if err != nil {
		return fallo()
	}
	defer pool.Close()
	configurar := o.Operacion == "configurar-preservacion"
	fuente, err := postgres.NuevaFuentePreservacionAuditoriaPostgreSQL(pool, configurar)
	if err != nil {
		return fallo()
	}
	var r domain.ResultadoPreservacionAuditoria
	if configurar {
		raw, err = leerRegular(o.Entrada, 4096, false)
		var s domain.SolicitudPreservacionAuditoria
		if err != nil || decodificar(raw, &s) != nil || s.Validar() != nil {
			_ = fuente.RegistrarFalloPreservacion(ctx, "configurar_preservacion_auditoria_v1", "error")
			return fallo()
		}
		r, err = application.PublicarPreservacionAuditoria(ctx, fuente, s)
	} else {
		if !versionConsultaPreservacionValida(ctx, o.Version, fuente.RegistrarFalloPreservacion) {
			return fallo()
		}
		r, err = application.ConsultarPreservacionAuditoria(ctx, fuente, o.Version)
	}
	if err != nil {
		return fallo()
	}
	codigo = domain.ResultadoTecnicoCorrecto
	return escribirResultado(out, map[string]any{"modo": "DESARROLLO", "resultado": r, "resolucion_documental": false, "expurgo_autorizado": false})
}

func versionConsultaPreservacionValida(ctx context.Context, version uint64, registrar func(context.Context, string, string) error) bool {
	if version <= domain.MaxVersionPreservacionAuditoria {
		return true
	}
	_ = registrar(ctx, "consultar_preservacion_auditoria_v1", "error")
	return false
}
func escribirResultadoPreservacion(out io.Writer, r any, codigo int) int {
	if escribirResultado(out, r) != 0 {
		return 1
	}
	return codigo
}
func poolPreservacion(ctx context.Context, o opcionesPreservacion) (*pgxpool.Pool, error) {
	conexion, err := leerRegular(o.Conexion, 4096, true)
	if err != nil {
		return nil, errEntrada
	}
	defer clear(conexion)
	poolCfg, err := pgxpool.ParseConfig(string(conexion))
	if err != nil {
		return nil, errEntrada
	}
	// Las conexiones TCP verifican la identidad del servidor y nunca caen a
	// texto claro. Los sockets locales sirven al ensayo aislado de desarrollo.
	if !strings.HasPrefix(poolCfg.ConnConfig.Host, "/") {
		if poolCfg.ConnConfig.TLSConfig == nil || poolCfg.ConnConfig.TLSConfig.InsecureSkipVerify {
			return nil, errEntrada
		}
		if poolCfg.ConnConfig.TLSConfig.MinVersion < tls.VersionTLS12 {
			poolCfg.ConnConfig.TLSConfig.MinVersion = tls.VersionTLS12
		}
	}
	for _, alternativa := range poolCfg.ConnConfig.Fallbacks {
		if !strings.HasPrefix(alternativa.Host, "/") {
			if alternativa.TLSConfig == nil || alternativa.TLSConfig.InsecureSkipVerify {
				return nil, errEntrada
			}
			if alternativa.TLSConfig.MinVersion < tls.VersionTLS12 {
				alternativa.TLSConfig.MinVersion = tls.VersionTLS12
			}
		}
	}
	poolCfg.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, errEntrada
	}
	return pool, nil

}
