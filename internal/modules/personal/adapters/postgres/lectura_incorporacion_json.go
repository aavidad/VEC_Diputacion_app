package postgres

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"time"
	"unicode/utf8"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

// 64KiB de material en base64 más la envoltura tipada, sin límites V3 mezclados.
const maximoRespuestaLecturaIncorporacion = 128 << 10

var instanteLecturaSQL = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,6})?Z$`)

type registroLecturaSQL struct {
	Solicitud              ct.SolicitudAltaPersonalRPT `json:"solicitud"`
	Resultado              ct.ResultadoAltaPersonalRPT `json:"resultado"`
	MaterialCanonico       string                      `json:"material_canonico"`
	MaterialSHA256         string                      `json:"material_sha256"`
	RegistradoEn           string                      `json:"registrado_en"`
	DecisionOriginalRef    string                      `json:"decision_original_ref"`
	AuditoriaRef           string                      `json:"auditoria_ref"`
	OutboxRef              string                      `json:"outbox_ref"`
	EjercicioSintetico     bool                        `json:"ejercicio_sintetico"`
	FirmaOficial           bool                        `json:"firma_oficial"`
	EficaciaAdministrativa bool                        `json:"eficacia_administrativa"`
}
type respuestaLecturaSQL struct {
	Registro            registroLecturaSQL `json:"registro"`
	DecisionLecturaRef  string             `json:"decision_lectura_ref"`
	ConsumoHuellaSHA256 string             `json:"consumo_huella_sha256"`
	AuditoriaLecturaRef string             `json:"auditoria_lectura_ref"`
	LeidaEn             string             `json:"leida_en"`
}

func decodificarLecturaIncorporacion(b []byte) (lector.Resultado, error) {
	var cero lector.Resultado
	if len(b) == 0 || len(b) > maximoRespuestaLecturaIncorporacion || !utf8.Valid(b) {
		return cero, lector.ErrNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(b))
	if valorLecturaJSON(d, 0) != nil || exigirFinJSONAlta(d) != nil {
		return cero, lector.ErrNoDisponible
	}
	root, e := objetoLecturaJSON(b, "registro", "decision_lectura_ref", "consumo_huella_sha256", "auditoria_lectura_ref", "leida_en")
	if e != nil {
		return cero, e
	}
	registro, e := objetoLecturaJSON(root["registro"], "solicitud", "resultado", "material_canonico", "material_sha256", "registrado_en", "decision_original_ref", "auditoria_ref", "outbox_ref", "ejercicio_sintetico", "firma_oficial", "eficacia_administrativa")
	if e != nil {
		return cero, e
	}
	solicitud, e := objetoLecturaJSON(registro["solicitud"], "esquema", "contrato_version", "solicitud_ref", "expediente_ref", "version_expediente", "capacidad_ref", "correlacion_ref", "idempotencia_ref", "fuente_rpt", "puesto_ref", "plaza_ref")
	if e != nil {
		return cero, e
	}
	if _, e = objetoLecturaJSON(solicitud["fuente_rpt"], "referencia", "version", "huella_sha256"); e != nil {
		return cero, e
	}
	if _, e = objetoLecturaJSON(registro["resultado"], "esquema", "contrato_version", "resultado_ref", "recibo_ref", "solicitud_ref", "correlacion_ref", "idempotencia_ref", "huella_solicitud_sha256", "estado", "relacion_ref", "ocupacion_ref"); e != nil {
		return cero, e
	}
	var w respuestaLecturaSQL
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&w) != nil || exigirFinJSONAlta(d) != nil {
		return cero, lector.ErrNoDisponible
	}
	if len(w.Registro.MaterialCanonico) == 0 || len(w.Registro.MaterialCanonico) > base64.StdEncoding.EncodedLen(64<<10) {
		return cero, lector.ErrNoDisponible
	}
	material, e := base64.StdEncoding.Strict().DecodeString(w.Registro.MaterialCanonico)
	if e != nil || len(material) > 64<<10 || base64.StdEncoding.EncodeToString(material) != w.Registro.MaterialCanonico {
		borrarBytesAlta(material)
		return cero, lector.ErrNoDisponible
	}
	ok := false
	defer func() {
		if !ok {
			borrarBytesAlta(material)
		}
	}()
	fecha, e := instanteWireLectura(w.Registro.RegistradoEn)
	if e != nil {
		return cero, e
	}
	leida, e := instanteWireLectura(w.LeidaEn)
	if e != nil {
		return cero, e
	}
	if !ctdomain.ReferenciaOpacaValida(w.DecisionLecturaRef) || !ctdomain.ReferenciaOpacaValida(w.AuditoriaLecturaRef) || !patronHuellaAlta.MatchString(w.ConsumoHuellaSHA256) || !patronHuellaAlta.MatchString(w.Registro.MaterialSHA256) {
		return cero, lector.ErrNoDisponible
	}
	r := lector.Resultado{Registro: ct.RegistroPersonalEjercicio{Solicitud: w.Registro.Solicitud, Resultado: w.Registro.Resultado, MaterialCanonico: material, MaterialSHA256: w.Registro.MaterialSHA256, RegistradoEn: fecha, DecisionOriginalRef: w.Registro.DecisionOriginalRef, AuditoriaRef: w.Registro.AuditoriaRef, OutboxRef: w.Registro.OutboxRef, EjercicioSintetico: w.Registro.EjercicioSintetico, FirmaOficial: w.Registro.FirmaOficial, EficaciaAdministrativa: w.Registro.EficaciaAdministrativa}, DecisionLecturaRef: w.DecisionLecturaRef, ConsumoHuellaSHA256: w.ConsumoHuellaSHA256, AuditoriaLecturaRef: w.AuditoriaLecturaRef, LeidaEn: leida}
	ok = true
	return r, nil
}

func instanteWireLectura(s string) (time.Time, error) {
	// time.Parse admite coma y fracciones sobrantes; SQL no las emite.
	if !instanteLecturaSQL.MatchString(s) {
		return time.Time{}, lector.ErrNoDisponible
	}
	t, e := time.Parse(time.RFC3339Nano, s)
	// SQL produce UTC/microsegundos (puede conservar ceros fraccionarios).
	if e != nil || len(s) == 0 || s[len(s)-1] != 'Z' || !ctdomain.InstanteUTCCanonico(t) {
		return time.Time{}, lector.ErrNoDisponible
	}
	return t, nil
}

func objetoLecturaJSON(b []byte, claves ...string) (map[string]json.RawMessage, error) {
	var m map[string]json.RawMessage
	if json.Unmarshal(b, &m) != nil || !clavesExactasAlta(m, claves...) {
		return nil, lector.ErrNoDisponible
	}
	return m, nil
}

// Antes de unmarshalling: ninguna clave repetida (ni escapada), null, array,
// profundidad excesiva o valor posterior. Los objetos exactos se cotejan luego;
// así encoding/json no puede aplicar case-folding a ninguna clave de autoridad.
func valorLecturaJSON(d *json.Decoder, profundidad int) error {
	if profundidad > 5 {
		return lector.ErrNoDisponible
	}
	t, e := d.Token()
	if e != nil || t == nil {
		return lector.ErrNoDisponible
	}
	delim, compuesto := t.(json.Delim)
	if !compuesto {
		return nil
	}
	if delim != '{' {
		return lector.ErrNoDisponible
	}
	vistas := map[string]bool{}
	for d.More() {
		k, e := d.Token()
		s, ok := k.(string)
		if e != nil || !ok || vistas[s] {
			return lector.ErrNoDisponible
		}
		vistas[s] = true
		if valorLecturaJSON(d, profundidad+1) != nil {
			return lector.ErrNoDisponible
		}
	}
	t, e = d.Token()
	if e != nil || t != json.Delim('}') {
		return lector.ErrNoDisponible
	}
	return nil
}
