package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	config "vec-diputacion-granada/config/auditoriacheckpoint"
	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var errEntrada = errors.New("entrada_invalida")

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
func run(args []string, out, log io.Writer) int {
	fs := flag.NewFlagSet("vec-auditoria-checkpoint", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	modo := fs.String("operacion", "", "")
	conf := fs.String("config", "", "")
	entrada := fs.String("entrada", "", "")
	salida := fs.String("salida", "", "")
	master := fs.String("kms-master", "", "")
	tsa := fs.String("tsa-secret", "", "")
	publica := fs.String("spki", "", "")
	pin := fs.String("pin-spki-sha256", "", "")
	cadena := fs.String("cadena", "", "")
	ancla := fs.String("ancla", "", "")
	maxRecibos := fs.Int("max-recibos", 0, "")
	conexion := fs.String("conexion", "", "")
	capturaRef := fs.String("captura-ref", "", "")
	versionPreservacion := fs.Uint64("version", 0, "")
	fallo := func() int {
		_ = json.NewEncoder(out).Encode(map[string]string{"modo": "DESARROLLO", "estado": "rechazado", "codigo": "entrada_o_dependencia_invalida"})
		return 1
	}
	if fs.Parse(args) != nil || fs.NArg() != 0 || (*modo != "emitir" && *modo != "verificar" && *modo != "verificar-continuidad" && *modo != "verificar-exportacion" && *modo != "ejecutar-periodico" && *modo != "configurar-preservacion" && *modo != "consultar-preservacion") {
		return fallo()
	}
	if *modo == "configurar-preservacion" || *modo == "consultar-preservacion" {
		incompatible := false
		fs.Visit(func(f *flag.Flag) {
			switch f.Name {
			case "operacion", "config", "conexion":
			case "entrada":
				if *modo != "configurar-preservacion" {
					incompatible = true
				}
			case "version":
				if *modo != "consultar-preservacion" {
					incompatible = true
				}
			default:
				incompatible = true
			}
		})
		if incompatible {
			return fallo()
		}
		return runPreservacion(opcionesPreservacion{Operacion: *modo, Config: *conf, Conexion: *conexion, Entrada: *entrada, Version: *versionPreservacion}, out, log)
	}
	if *modo == "ejecutar-periodico" {
		incompatible := false
		fs.Visit(func(f *flag.Flag) {
			switch f.Name {
			case "operacion", "config", "conexion", "captura-ref", "kms-master", "tsa-secret":
			default:
				incompatible = true
			}
		})
		if incompatible {
			return fallo()
		}
		return runPeriodico(opcionesPeriodicas{Config: *conf, Conexion: *conexion, CapturaRef: *capturaRef, Master: *master, TSA: *tsa}, out, log)
	}
	if *conexion != "" || *capturaRef != "" || *versionPreservacion != 0 {
		return fallo()
	}
	incompatible := false
	fs.Visit(func(f *flag.Flag) {
		if (*modo == "verificar-continuidad" || *modo == "verificar-exportacion") && (f.Name == "kms-master" || f.Name == "tsa-secret" || f.Name == "salida" || *modo == "verificar-continuidad" && f.Name == "cadena") ||
			*modo != "verificar-continuidad" && (f.Name == "ancla" || f.Name == "max-recibos") {
			incompatible = true
		}
	})
	if incompatible {
		return fallo()
	}
	var cfg config.AuditoriaCheckpointOffline
	b, err := leerRegular(*conf, 16*1024, false)
	if err != nil || decodificar(b, &cfg) != nil || cfg.Validar() != nil {
		return fallo()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ctx, err = ports.ConCorrelacionIncidenciasPeticion(ctx)
	if err != nil {
		return fallo()
	}
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino: log, Entorno: "desarrollo", VersionBinario: cfg.VersionBinario, Capacidad: 8})
	if err != nil {
		return fallo()
	}
	codigo := domain.ResultadoTecnicoEntradaInvalida
	defer func() {
		emisor.EmitirResultadoConContexto(ctx, domain.SolicitudResultadoTecnico{Resultado: codigo, Componente: domain.ComponenteIncidenciaAuditoria, Etapa: domain.EtapaIncidenciaEmision})
		c, cancelar := context.WithTimeout(context.Background(), time.Second)
		defer cancelar()
		_ = emisor.Cerrar(c)
	}()
	if *modo == "verificar-exportacion" {
		resultado := runExportacion(ctx, cfg, opcionesExportacion{*entrada, *cadena, *publica, *pin}, out)
		if resultado == 0 {
			codigo = domain.ResultadoTecnicoCorrecto
		}
		return resultado
	}
	if *modo == "verificar-continuidad" {
		resultado := runContinuidad(ctx, cfg, opcionesContinuidad{*ancla, *entrada, *publica, *pin, *maxRecibos}, out)
		if resultado == 0 {
			codigo = domain.ResultadoTecnicoCorrecto
		}
		return resultado
	}
	b, err = leerRegular(*entrada, cfg.MaxBytes, false)
	if err != nil {
		return fallo()
	}
	if *modo == "emitir" {
		if *cadena != "" || *pin != "" || *salida == "" || *publica == "" || filepath.Clean(*salida) == filepath.Clean(*publica) {
			return fallo()
		}
		var c domain.CoberturaCheckpoint
		if decodificar(b, &c) != nil {
			return fallo()
		}
		k, err := leerRegular(*master, 32, true)
		if err != nil || len(k) != 32 {
			clear(k)
			return fallo()
		}
		defer clear(k)
		t, err := leerRegular(*tsa, 32, true)
		if err != nil || len(t) != 32 {
			clear(t)
			return fallo()
		}
		defer clear(t)
		var km, ts [32]byte
		copy(km[:], k)
		copy(ts[:], t)
		defer clear(km[:])
		defer clear(ts[:])
		proveedor, err := bootstrap.NuevoProveedorCheckpointDesarrollo(km, ts, cfg.Politica)
		if err != nil {
			return fallo()
		}
		defer proveedor.CerrarCheckpoint()
		r, err := application.EmitirCheckpointDesarrollo(ctx, domain.CheckpointDesarrollo{Esquema: domain.EsquemaCheckpointDesarrollo, Politica: cfg.Politica, Cobertura: c}, proveedor, proveedor, cfg.MaxRegistros)
		if err != nil {
			return fallo()
		}
		der, err := proveedor.PublicaCheckpointDER()
		if err != nil {
			return fallo()
		}
		rb, err := json.Marshal(r)
		if err != nil {
			return fallo()
		}
		// Ambos destinos son nuevos. Un fallo deja un artefacto parcial declarado;
		// nunca se reemplaza ni se elimina material preexistente.
		if escribirNuevo(*publica, der) != nil || escribirNuevo(*salida, append(rb, '\n')) != nil {
			codigo = domain.ResultadoTecnicoNoDisponible
			return fallo()
		}
		codigo = domain.ResultadoTecnicoCorrecto
		return escribirResultado(out, map[string]any{"modo": "DESARROLLO", "estado": "emitido", "pin_spki_sha256": proveedor.PinCheckpoint(), "origen_extraccion": "no_acreditado", "integridad_cadena": "no_evaluada", "tsa": "no_verificada_offline", "tiempo_independiente": false, "firma_legal": false})
	}
	if *master != "" || *tsa != "" || *salida != "" {
		return fallo()
	}
	var r domain.ReciboCheckpointDesarrollo
	if decodificar(b, &r) != nil {
		return fallo()
	}
	der, err := leerRegular(*publica, 4096, false)
	if err != nil {
		return fallo()
	}
	verificador, err := bootstrap.NuevoVerificadorCheckpointDesarrollo(der, *pin, cfg.Politica)
	if err != nil {
		return fallo()
	}
	resultado := application.VerificarCheckpointDesarrollo(ctx, r, verificador, cfg.MaxRegistros)
	if resultado.Firma != "verificada_con_pin_externo" {
		return fallo()
	}
	if *cadena != "" {
		cb, err := leerRegular(*cadena, cfg.MaxBytes, false)
		if err != nil {
			return fallo()
		}
		informe, err := verificarCadenaCheckpoint(cb, r.Checkpoint.Cobertura, cfg.MaxBytes, cfg.MaxRegistros)
		if err != nil {
			resultado.IntegridadCadena = "rechazada"
			_ = escribirResultado(out, resultado)
			return 1
		}
		resultado.IntegridadCadena = informe.Estado
		if informe.Estado != "verificada" {
			_ = escribirResultado(out, resultado)
			return 1
		}
		resultado.ConsumosHistoricosSinFechaLigada = informe.ConsumosHistoricosSinFechaLigada
		resultado.FechaConsumoLigadaCotejada = informe.FechaConsumoLigadaCotejada
	}
	codigo = domain.ResultadoTecnicoCorrecto
	return escribirResultado(out, resultado)
}
func escribirResultado(w io.Writer, r any) int {
	if json.NewEncoder(w).Encode(r) != nil {
		return 1
	}
	return 0
}
func leerRegular(ruta string, limite int64, secreto bool) ([]byte, error) {
	if limite < 1 {
		return nil, errEntrada
	}
	if secreto {
		if !filepath.IsAbs(ruta) {
			return nil, errEntrada
		}
		real, e := filepath.EvalSymlinks(ruta)
		if e != nil || real != filepath.Clean(ruta) {
			return nil, errEntrada
		}
		for p := filepath.Dir(ruta); ; p = filepath.Dir(p) {
			if _, e := os.Lstat(filepath.Join(p, ".git")); e == nil {
				return nil, errEntrada
			}
			if p == filepath.Dir(p) {
				break
			}
		}
	}
	// #nosec G304 -- Ruta explícita del operador local; descriptor regular acotado y secretos externos con propietario y permisos comprobados.
	f, e := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if e != nil {
		return nil, errEntrada
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Size() < 1 || st.Size() > limite {
		return nil, errEntrada
	}
	if secreto {
		s, ok := st.Sys().(*syscall.Stat_t)
		if !ok || s.Nlink != 1 || st.Mode().Perm()&0077 != 0 || int64(s.Uid) != int64(os.Geteuid()) {
			return nil, errEntrada
		}
	}
	b, e := io.ReadAll(io.LimitReader(f, limite+1))
	if e != nil || int64(len(b)) > limite || int64(len(b)) != st.Size() {
		clear(b)
		return nil, errEntrada
	}
	return b, nil
}
func escribirNuevo(ruta string, b []byte) error {
	// #nosec G304 -- Destino explícito del operador local; creación exclusiva sin sobrescribir archivos o enlaces existentes.
	f, e := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		return errEntrada
	}
	n, e := f.Write(b)
	if e != nil || n != len(b) {
		_ = f.Close()
		return errEntrada
	}
	if f.Sync() != nil {
		_ = f.Close()
		return errEntrada
	}
	return f.Close()
}
