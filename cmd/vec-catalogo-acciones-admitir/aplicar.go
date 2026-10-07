package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var errRespuestaAdmision = errors.New("respuesta_admision_invalida")
var errCommitAdmision = errors.New("commit_admision_indeterminado")

type transaccionAdmision interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Commit(context.Context) error
	Rollback(context.Context) error
}

type respuestaAdmision struct {
	Estado string
	Replay bool
	Recibo json.RawMessage
}

// La fachada devuelve un intento auditado en cualquier estado. El efecto se
// confirma únicamente después de cotejar el recibo con el plan enviado.
func aplicarPlanEnTransaccion(ctx context.Context, tx transaccionAdmision, plan []byte, huella string, esperado planAdmision) (respuestaAdmision, error) {
	defer func() {
		limite, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelar()
		_ = tx.Rollback(limite)
	}()
	if _, err := tx.Exec(ctx, `SET LOCAL TIME ZONE 'UTC'`); err != nil {
		return respuestaAdmision{}, err
	}
	var b []byte
	if err := tx.QueryRow(ctx, `SELECT vec_autorizacion.registrar_catalogo_acciones_admin_v1($1::text,$2::text)`, string(plan), huella).Scan(&b); err != nil {
		return respuestaAdmision{}, err
	}
	r, err := validarRespuestaAdmision(b, esperado, huella)
	if err != nil {
		return respuestaAdmision{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return respuestaAdmision{}, errCommitAdmision
	}
	return r, nil
}

func validarRespuestaAdmision(b []byte, plan planAdmision, planSHA string) (respuestaAdmision, error) {
	ahora := time.Now().UTC()
	m, err := objetoExacto(b, "estado", "codigo", "recibo", "replay", "auditoria_intento")
	if err != nil {
		return respuestaAdmision{}, err
	}
	estado, ok := cadenaJSON(m["estado"])
	if !ok {
		return respuestaAdmision{}, errRespuestaAdmision
	}
	var replay *bool
	if json.Unmarshal(m["replay"], &replay) != nil || replay == nil {
		return respuestaAdmision{}, errRespuestaAdmision
	}
	if validarAuditoriaIntento(m["auditoria_intento"], ahora) != nil {
		return respuestaAdmision{}, errRespuestaAdmision
	}
	if estado == "permitido" {
		if !esNulo(m["codigo"]) || esNulo(m["recibo"]) ||
			validarReciboAdmision(m["recibo"], plan, planSHA, ahora) != nil ||
			!intentoPosteriorAlRecibo(m["recibo"], m["auditoria_intento"]) {
			return respuestaAdmision{}, errRespuestaAdmision
		}
		return respuestaAdmision{Estado: estado, Replay: *replay, Recibo: m["recibo"]}, nil
	}
	codigo, ok := cadenaJSON(m["codigo"])
	if !ok || !esNulo(m["recibo"]) || *replay ||
		(estado == "denegado" && codigo != "catalogo_acciones_rechazado") ||
		(estado == "error" && codigo != "catalogo_acciones_no_disponible") ||
		(estado != "denegado" && estado != "error") {
		return respuestaAdmision{}, errRespuestaAdmision
	}
	return respuestaAdmision{Estado: estado}, nil
}

func validarReciboAdmision(b []byte, p planAdmision, planSHA string, ahora time.Time) error {
	m, err := objetoExacto(b, "esquema", "operacion_ref", "plan_sha256", "catalogo_ref",
		"catalogo_version", "catalogo_sha256", "paquete_ref", "paquete_version", "paquete_sha256",
		"censo_sha256", "aprobacion_ref", "aprobacion_sha256", "aprobador_ref",
		"auditoria_ref", "auditoria_secuencia", "auditoria_huella_sha256", "confirmado_en")
	if err != nil {
		return err
	}
	comparaciones := map[string]string{
		"esquema":           "vec.admin.catalogo-acciones.recibo.v1",
		"operacion_ref":     p.OperacionRef,
		"plan_sha256":       planSHA,
		"catalogo_ref":      p.CatalogoRef,
		"catalogo_version":  p.CatalogoVersion,
		"catalogo_sha256":   p.CatalogoSHA256,
		"paquete_ref":       p.PaqueteRef,
		"paquete_sha256":    p.PaqueteSHA256,
		"aprobacion_ref":    p.AprobacionRef,
		"aprobacion_sha256": p.AprobacionSHA256,
	}
	for campo, esperado := range comparaciones {
		valor, ok := cadenaJSON(m[campo])
		if !ok || valor != esperado {
			return errRespuestaAdmision
		}
	}
	var paqueteVersion int
	if json.Unmarshal(m["paquete_version"], &paqueteVersion) != nil || strconv.Itoa(paqueteVersion) != p.PaqueteVersion ||
		!huellaValidaValor(m["censo_sha256"]) || !huellaValidaValor(m["auditoria_huella_sha256"]) {
		return errRespuestaAdmision
	}
	aprobador, ok := cadenaJSON(m["aprobador_ref"])
	if !ok || aprobador == "" || len(aprobador) > 128 {
		return errRespuestaAdmision
	}
	auditoria, ok := cadenaJSON(m["auditoria_ref"])
	if !ok || !referenciaAuditoria(auditoria, "aud_v3_caa_") ||
		!secuenciaPositiva(m["auditoria_secuencia"]) || !fechaUTC(m["confirmado_en"], ahora) {
		return errRespuestaAdmision
	}
	return nil
}

func validarAuditoriaIntento(b []byte, ahora time.Time) error {
	m, err := objetoExacto(b, "auditoria_ref", "secuencia", "huella_sha256", "correlacion_ref", "registrada_en")
	if err != nil {
		return err
	}
	ref, ok := cadenaJSON(m["auditoria_ref"])
	if !ok || !referenciaAuditoria(ref, "aud_v3_caai_") ||
		!secuenciaPositiva(m["secuencia"]) || !huellaValidaValor(m["huella_sha256"]) ||
		!fechaUTC(m["registrada_en"], ahora) {
		return errRespuestaAdmision
	}
	corr, ok := cadenaJSON(m["correlacion_ref"])
	if !ok || !referenciaAuditoria(corr, "correlacion_") {
		return errRespuestaAdmision
	}
	return nil
}

func objetoExacto(b []byte, claves ...string) (map[string]json.RawMessage, error) {
	lector := json.NewDecoder(bytes.NewReader(b))
	inicio, err := lector.Token()
	if err != nil || inicio != json.Delim('{') {
		return nil, errRespuestaAdmision
	}
	m := make(map[string]json.RawMessage, len(claves))
	for lector.More() {
		clave, err := lector.Token()
		nombre, ok := clave.(string)
		if err != nil || !ok {
			return nil, errRespuestaAdmision
		}
		if _, repetida := m[nombre]; repetida {
			return nil, errRespuestaAdmision
		}
		var valor json.RawMessage
		if lector.Decode(&valor) != nil {
			return nil, errRespuestaAdmision
		}
		m[nombre] = valor
	}
	fin, err := lector.Token()
	if err != nil || fin != json.Delim('}') || !errors.Is(lector.Decode(new(any)), io.EOF) || len(m) != len(claves) {
		return nil, errRespuestaAdmision
	}
	for _, clave := range claves {
		if _, ok := m[clave]; !ok {
			return nil, errRespuestaAdmision
		}
	}
	return m, nil
}

func cadenaJSON(b []byte) (string, bool) {
	var p *string
	if json.Unmarshal(b, &p) != nil || p == nil {
		return "", false
	}
	return *p, true
}

func esNulo(b []byte) bool { return bytes.Equal(bytes.TrimSpace(b), []byte("null")) }

func huellaValidaValor(b []byte) bool {
	s, ok := cadenaJSON(b)
	return ok && huellaValida(s)
}

func referenciaAuditoria(s, prefijo string) bool {
	if !strings.HasPrefix(s, prefijo) || len(s) != len(prefijo)+32 {
		return false
	}
	_, err := hex.DecodeString(s[len(prefijo):])
	return err == nil
}

func secuenciaPositiva(b []byte) bool {
	var numero json.Number
	if json.Unmarshal(b, &numero) != nil {
		return false
	}
	n, err := numero.Int64()
	return err == nil && n > 0
}

func fechaUTC(b []byte, ahora time.Time) bool {
	s, ok := cadenaJSON(b)
	if !ok {
		return false
	}
	instante, err := time.Parse(time.RFC3339Nano, s)
	if err != nil || instante.Nanosecond()%1000 != 0 {
		return false
	}
	_, desplazamiento := instante.Zone()
	return desplazamiento == 0 && !instante.After(ahora)
}

func intentoPosteriorAlRecibo(recibo, intento []byte) bool {
	var r struct {
		Secuencia int64     `json:"auditoria_secuencia"`
		Fecha     time.Time `json:"confirmado_en"`
	}
	var a struct {
		Secuencia int64     `json:"secuencia"`
		Fecha     time.Time `json:"registrada_en"`
	}
	return json.Unmarshal(recibo, &r) == nil && json.Unmarshal(intento, &a) == nil &&
		a.Secuencia > r.Secuencia && !a.Fecha.Before(r.Fecha)
}
