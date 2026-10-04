package main

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	config "vec-diputacion-granada/config/auditoriaperiodica"
	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	"vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type opcionesPeriodicas struct{ Config, Conexion, CapturaRef, Master, TSA string }

// Una invocación sirve para un temporizador externo. SQL gobierna la cadencia;
// este modo no permite coberturas manuales ni extracción de filas personales.
func runPeriodico(o opcionesPeriodicas, out, log io.Writer) int {
	ref := o.CapturaRef
	fallo := func() int {
		_ = json.NewEncoder(out).Encode(map[string]any{"modo": "DESARROLLO", "estado": "rechazado", "codigo": "entrada_o_dependencia_invalida", "captura_ref": ref})
		return 1
	}
	raw, err := leerRegular(o.Config, 16*1024, false)
	if err != nil {
		return fallo()
	}
	cfg, err := config.Decodificar(raw)
	if err != nil {
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
	conexion, err := leerRegular(o.Conexion, 4096, true)
	if err != nil {
		return fallo()
	}
	defer clear(conexion)
	poolCfg, err := pgxpool.ParseConfig(string(conexion))
	if err != nil {
		return fallo()
	}
	poolCfg.MaxConns = 2
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fallo()
	}
	defer pool.Close()
	fuente, err := postgres.NuevaFuenteCheckpointPeriodicoPostgreSQL(pool, cfg.MaxRegistros)
	if err != nil {
		return fallo()
	}
	var c ports.CapturaCheckpointPeriodico
	var previo *domain.ReciboCheckpointDesarrollo
	var confirmacion ports.AcuseCheckpointPeriodico
	if ref != "" {
		c, previo, confirmacion, err = fuente.RecuperarCheckpoint(ctx, ref)
	} else {
		c, err = application.CapturarCheckpointPeriodico(ctx, fuente, cfg.MaxRegistros)
	}
	if err != nil {
		return fallo()
	}
	ref = c.CapturaRef
	if c.Estado == "no_vencido" {
		codigo = domain.ResultadoTecnicoCorrecto
		return escribirResultado(out, map[string]any{"modo": "DESARROLLO", "estado": "no_vencido", "auditoria_ref": c.Acuse.AuditoriaRef})
	}
	falloSello := func() int {
		_ = fuente.RegistrarFalloCheckpoint(ctx, "confirmar_sello_periodico_v1", "error")
		return fallo()
	}
	if c.PinSPKISHA256 != cfg.PinSPKISHA256 {
		return falloSello()
	}
	k, err := leerRegular(o.Master, 32, true)
	if err != nil || len(k) != 32 {
		clear(k)
		return falloSello()
	}
	defer clear(k)
	t, err := leerRegular(o.TSA, 32, true)
	if err != nil || len(t) != 32 {
		clear(t)
		return falloSello()
	}
	defer clear(t)
	var km, ts [32]byte
	copy(km[:], k)
	copy(ts[:], t)
	defer clear(km[:])
	defer clear(ts[:])
	proveedor, err := bootstrap.NuevoProveedorCheckpointDesarrollo(km, ts, c.Checkpoint.Politica)
	if err != nil {
		return falloSello()
	}
	defer proveedor.CerrarCheckpoint()
	if proveedor.PinCheckpoint() != cfg.PinSPKISHA256 {
		return falloSello()
	}
	var r application.ResultadoCheckpointPeriodico
	if previo != nil {
		// Cotejo del artefacto conservado, sin volver a firmar o confirmar.
		b, e := json.Marshal(*previo)
		der, e2 := proveedor.PublicaCheckpointDER()
		v, e3 := bootstrap.NuevoVerificadorCheckpointDesarrollo(der, cfg.PinSPKISHA256, c.Checkpoint.Politica)
		if e != nil || e2 != nil || e3 != nil || previo.Checkpoint != c.Checkpoint || confirmacion.CapturaRef != ref || confirmacion.ReciboHuellaSHA256 != domain.HuellaCheckpoint(b) ||
			application.VerificarCheckpointDesarrollo(ctx, *previo, v, cfg.MaxRegistros).Firma != "verificada_con_pin_externo" {
			return falloSello()
		}
		r = application.ResultadoCheckpointPeriodico{Estado: "recuperado", Captura: c, Recibo: *previo, Acuse: confirmacion}
	} else {
		r, err = application.SellarConfirmarCheckpointPeriodico(ctx, fuente, c, proveedor, proveedor, cfg.MaxRegistros)
		if err != nil {
			return fallo()
		}
	}
	codigo = domain.ResultadoTecnicoCorrecto
	return escribirResultado(out, map[string]any{"modo": "DESARROLLO", "estado": r.Estado, "captura_ref": ref, "auditoria_ref": r.Acuse.AuditoriaRef,
		"recibo": r.Recibo, "tsa": "no_verificada_offline", "tiempo_independiente": false, "firma_legal": false, "integridad_registros": "no_evaluada"})
}
