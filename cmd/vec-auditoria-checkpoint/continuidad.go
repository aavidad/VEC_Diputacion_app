package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	config "vec-diputacion-granada/config/auditoriacheckpoint"
	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
)

type opcionesContinuidad struct {
	ancla, entrada, spki, pin string
	maxRecibos                int
}

func runContinuidad(ctx context.Context, cfg config.AuditoriaCheckpointOffline, o opcionesContinuidad, out io.Writer) int {
	fallo := func(clave string) int {
		r := application.ResultadoContinuidadCheckpoint{ResultadoCheckpointDesarrollo: application.ResultadoCheckpointDesarrollo{
			Esquema: domain.EsquemaContinuidadCheckpointDesarrollo, Modo: "DESARROLLO", Firma: "rechazada", IntegridadCadena: "no_evaluada",
			OrigenExtraccion: "no_acreditado", TSA: "no_verificada_offline"}, InformeContinuidadCheckpoint: domain.RechazoContinuidadCheckpoint(clave, "valida", "invalida")}
		if escribirResultado(out, r) != 0 {
			return 1
		}
		return 1
	}
	if o.ancla == "" || o.entrada == "" || o.spki == "" || o.pin == "" || o.maxRecibos < 1 || o.maxRecibos > 256 {
		return fallo("argumentos_continuidad")
	}
	anclaBytes, err := leerRegular(o.ancla, cfg.MaxBytes, false)
	if err != nil {
		return fallo("ancla")
	}
	// Ambos documentos comparten el presupuesto, no dos límites independientes.
	if int64(len(anclaBytes)) >= cfg.MaxBytes {
		return fallo("max_bytes")
	}
	loteBytes, err := leerRegular(o.entrada, cfg.MaxBytes-int64(len(anclaBytes)), false)
	if err != nil {
		return fallo("lote_o_max_bytes")
	}
	ancla, err := decodificarReciboContinuidad(anclaBytes)
	if err != nil {
		return fallo("estructura_ancla")
	}
	recibos, err := decodificarLoteContinuidad(loteBytes, o.maxRecibos)
	if err != nil {
		return fallo("estructura_lote")
	}
	der, err := leerRegular(o.spki, 4096, false)
	if err != nil {
		return fallo("spki")
	}
	v, err := bootstrap.NuevoVerificadorCheckpointDesarrollo(der, o.pin, cfg.Politica)
	if err != nil {
		return fallo("raiz_o_politica")
	}
	r := application.VerificarContinuidadCheckpoint(ctx, ancla, recibos, v, cfg.MaxRegistros, o.maxRecibos)
	if escribirResultado(out, r) != 0 || r.Continuidad != "verificada" {
		return 1
	}
	return 0
}

func objetoContinuidad(b []byte, claves ...string) (map[string]json.RawMessage, error) {
	var objeto map[string]json.RawMessage
	if decodificar(b, &objeto) != nil || len(objeto) != len(claves) {
		return nil, errEntrada
	}
	for _, clave := range claves {
		valor, existe := objeto[clave]
		if !existe || bytes.Equal(bytes.TrimSpace(valor), []byte("null")) {
			return nil, errEntrada
		}
	}
	return objeto, nil
}

func decodificarReciboContinuidad(b []byte) (domain.ReciboCheckpointDesarrollo, error) {
	vacio := domain.ReciboCheckpointDesarrollo{}
	r, err := objetoContinuidad(b, "checkpoint", "tsa", "pin_spki_sha256", "firma_base64")
	if err != nil {
		return vacio, err
	}
	c, err := objetoContinuidad(r["checkpoint"], "esquema", "politica", "cobertura")
	if err != nil {
		return vacio, err
	}
	if _, err := objetoContinuidad(c["politica"], "version", "politica_ref", "politica_version", "clave_ref", "clave_version", "proveedor_kms", "proveedor_kms_version", "proveedor_tsa", "proveedor_tsa_version", "operacion_tsa", "modo"); err != nil {
		return vacio, err
	}
	if _, err := objetoContinuidad(c["cobertura"], "cadena_id", "primera_secuencia", "ultima_secuencia", "registros", "anterior_sha256", "cabeza_sha256"); err != nil {
		return vacio, err
	}
	if _, err := objetoContinuidad(r["tsa"], "referencia", "huella_preimagen_sha256", "huella_checkpoint_sha256", "autoridad", "esquema"); err != nil {
		return vacio, err
	}
	var recibo domain.ReciboCheckpointDesarrollo
	if decodificar(b, &recibo) != nil {
		return vacio, errEntrada
	}
	return recibo, nil
}

func decodificarLoteContinuidad(b []byte, maxRecibos int) ([]domain.ReciboCheckpointDesarrollo, error) {
	if maxRecibos < 1 || maxRecibos > 256 {
		return nil, errEntrada
	}
	objeto, err := objetoContinuidad(b, "esquema", "recibos")
	var esquema string
	if err != nil || json.Unmarshal(objeto["esquema"], &esquema) != nil || esquema != domain.EsquemaContinuidadCheckpointDesarrollo {
		return nil, errEntrada
	}
	d := json.NewDecoder(bytes.NewReader(objeto["recibos"]))
	token, err := d.Token()
	if err != nil || token != json.Delim('[') {
		return nil, errEntrada
	}
	recibos := make([]domain.ReciboCheckpointDesarrollo, 0, maxRecibos)
	for d.More() {
		if len(recibos) >= maxRecibos {
			return nil, errEntrada
		}
		var bruto json.RawMessage
		if d.Decode(&bruto) != nil {
			return nil, errEntrada
		}
		r, err := decodificarReciboContinuidad(bruto)
		if err != nil {
			return nil, err
		}
		recibos = append(recibos, r)
	}
	if len(recibos) == 0 {
		return nil, errEntrada
	}
	if token, err = d.Token(); err != nil || token != json.Delim(']') {
		return nil, errEntrada
	}
	if d.Decode(new(any)) != io.EOF {
		return nil, errEntrada
	}
	return recibos, nil
}
