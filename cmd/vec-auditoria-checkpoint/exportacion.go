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

type opcionesExportacion struct{ recibo, cadena, publica, pin string }

func runExportacion(ctx context.Context, cfg config.AuditoriaCheckpointOffline, o opcionesExportacion, out io.Writer) int {
	fallo := func() int { return escribirFalloExportacion(out) }
	if o.cadena == "" || o.recibo == "" || o.publica == "" {
		return fallo()
	}
	b, err := leerRegular(o.recibo, cfg.MaxBytes, false)
	if err != nil {
		return fallo()
	}
	var r domain.ReciboExportacionAuditoriaDesarrollo
	if decodificar(b, &r) != nil || camposExportacionCompletos(b) != nil {
		return fallo()
	}
	der, err := leerRegular(o.publica, 4096, false)
	if err != nil {
		return fallo()
	}
	v, err := bootstrap.NuevoVerificadorExportacionAuditoriaDesarrollo(der, o.pin, cfg.Politica)
	if err != nil {
		return fallo()
	}
	restante := cfg.MaxBytes - int64(len(b))
	cb, err := leerRegular(o.cadena, restante, false)
	if err != nil {
		return fallo()
	}
	resultado := application.VerificarExportacionAuditoriaDesarrollo(ctx, r, cb, v, restante, cfg.MaxRegistros)
	if escribirResultado(out, resultado) != 0 {
		return 1
	}
	if resultado.Estado != "verificada" {
		return 1
	}
	return 0
}

func escribirFalloExportacion(out io.Writer) int {
	_ = escribirResultado(out, map[string]any{"esquema": domain.EsquemaExportacionAuditoriaDesarrollo, "modo": "DESARROLLO", "estado": "rechazada", "codigo": "entrada_o_dependencia_invalida", "origen_extraccion": "no_acreditado", "firma_legal": false, "tiempo_independiente": false})
	return 1
}

// El recibo cerrado no tiene campos opcionales. Se exigen incluso los ceros y
// el aviso false para distinguir ausencia de una declaración firmada.
func camposExportacionCompletos(b []byte) error {
	arbol := map[string][]string{
		"":           {"manifiesto", "tsa", "pin_spki_sha256", "firma_base64"},
		"manifiesto": {"esquema", "politica", "captura", "documento", "cobertura", "historicos_sin_fecha_ligada"},
		"politica":   {"version", "politica_ref", "politica_version", "clave_ref", "clave_version", "proveedor_kms", "proveedor_kms_version", "proveedor_tsa", "proveedor_tsa_version", "operacion_tsa", "modo"},
		"captura":    {"referencia", "auditoria_ref", "auditoria_sha256", "capturada_en"},
		"documento":  {"esquema", "bytes", "sha256"},
		"cobertura":  {"cadena_id", "primera_secuencia", "ultima_secuencia", "anterior_sha256", "cabeza_sha256", "registros"},
		"tsa":        {"referencia", "huella_preimagen_sha256", "huella_checkpoint_sha256", "autoridad", "esquema"},
	}
	var comprobar func([]byte, string) error
	comprobar = func(raw []byte, nombre string) error {
		var objeto map[string]json.RawMessage
		if json.Unmarshal(raw, &objeto) != nil || objeto == nil || len(objeto) != len(arbol[nombre]) {
			return errEntrada
		}
		for _, clave := range arbol[nombre] {
			valor, ok := objeto[clave]
			if !ok || bytes.Equal(bytes.TrimSpace(valor), []byte("null")) {
				return errEntrada
			}
			if _, esObjeto := arbol[clave]; esObjeto {
				if comprobar(valor, clave) != nil {
					return errEntrada
				}
			}
		}
		return nil
	}
	return comprobar(b, "")
}
