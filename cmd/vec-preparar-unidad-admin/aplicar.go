package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/shared/i18n"
)

type aprobacionPrivada struct {
	HuellaPlanSHA256 string `json:"huella_plan_sha256"`
}
type diagnosticoAplicacion struct {
	Codigo        string `json:"codigo"`
	Mensaje       string `json:"mensaje"`
	Limite        string `json:"limite"`
	Estado        string `json:"estado"`
	Confirmado    bool   `json:"confirmado"`
	AcuseGuardado bool   `json:"acuse_guardado"`
	Replay        bool   `json:"replay"`
}

func ejecutarAplicacion(p domain.PlanUnidadInicialAdminV1, sha string, fuente []byte, conexion, aprobacion, acuse, timeout string, textos *i18n.Catalog, idioma string, salida, errores io.Writer, abrir abrirTransaccion) int {
	emitir := func(w io.Writer, d diagnosticoAplicacion, exit int) int {
		d.Mensaje = textos.T(idioma, d.Codigo)
		d.Limite = textos.T(idioma, "limite")
		if json.NewEncoder(w).Encode(d) != nil {
			return 2
		}
		return exit
	}
	fallo := func(codigo string) int {
		return emitir(errores, diagnosticoAplicacion{Codigo: codigo, Estado: "sin_confirmar"}, 1)
	}
	duracion, err := time.ParseDuration(timeout)
	if err != nil || duracion <= 0 || abrir == nil {
		return fallo("uso_invalido")
	}
	cb, err := leerPrivado(conexion)
	if err != nil {
		return fallo("fuente_insegura")
	}
	defer clear(cb)
	ab, err := leerPrivado(aprobacion)
	if err != nil {
		return fallo("fuente_insegura")
	}
	defer clear(ab)
	var cfg conexionPrivada
	var aprobado aprobacionPrivada
	if decodificarEstricto(cb, &cfg) != nil || decodificarEstricto(ab, &aprobado) != nil || cfg.DSN == "" || !hashValido(aprobado.HuellaPlanSHA256) {
		return fallo("aplicacion_entrada_invalida")
	}
	raiz, err := abrirRaizPrivada(acuse)
	if err != nil {
		return fallo("acuse_inseguro")
	}
	defer raiz.Close()
	f, err := raiz.OpenFile(filepath.Base(acuse), os.O_WRONLY|os.O_CREATE|os.O_EXCL|noSeguirEnlaces, 0600)
	if err != nil {
		return fallo("acuse_inseguro")
	}
	guardado := false
	defer func() {
		_ = f.Close()
		if !guardado {
			_ = raiz.Remove(filepath.Base(acuse))
		}
	}()
	if f.Chmod(0600) != nil {
		return fallo("acuse_inseguro")
	}
	canon, _, err := p.CanonicoYHuella()
	if err != nil {
		return fallo("fuente_invalida")
	}
	defer clear(canon)
	ctx, cancelar := context.WithTimeout(context.Background(), duracion)
	defer cancelar()
	b, e, err := ejecutarOperacion(ctx, cfg, duracion, canon, aprobado.HuellaPlanSHA256, fuente, documento{p, sha}, abrir)
	if err == errCommit {
		return emitir(errores, diagnosticoAplicacion{Codigo: "commit_no_confirmado", Estado: "indeterminado"}, 2)
	}
	if err != nil {
		return fallo("operacion_no_confirmada")
	}
	defer clear(b)
	b = append(b, '\n')
	if _, err = f.Write(b); err != nil {
		return emitir(errores, diagnosticoAplicacion{Codigo: "acuse_no_guardado", Estado: e.Estado, Confirmado: true, Replay: e.Replay}, 2)
	}
	if f.Sync() != nil || f.Close() != nil {
		return emitir(errores, diagnosticoAplicacion{Codigo: "acuse_no_guardado", Estado: e.Estado, Confirmado: true, Replay: e.Replay}, 2)
	}
	guardado = true
	codigo, exit := "unidad_confirmada", 0
	if e.Estado == "denegado" {
		codigo, exit = "unidad_rechazada", 1
	}
	if e.Estado == "error" {
		codigo, exit = "unidad_no_disponible", 1
	}
	return emitir(salida, diagnosticoAplicacion{Codigo: codigo, Estado: e.Estado, Confirmado: true, AcuseGuardado: true, Replay: e.Replay}, exit)
}
func rutasAplicacionDistintas(previas []string, rutas ...string) bool {
	vistas := map[string]bool{}
	for _, ruta := range previas {
		vistas[ruta] = true
	}
	for _, ruta := range rutas {
		if ruta == "" || vistas[ruta] {
			return false
		}
		vistas[ruta] = true
	}
	return true
}
