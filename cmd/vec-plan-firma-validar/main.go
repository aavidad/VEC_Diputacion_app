// Valida un fichero local del kit de gobierno del plan de firma, sin efectos.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const maxMaterial = 4 << 20

var (
	errMaterial    = errors.New("material no valido")
	shaValido      = regexp.MustCompile(`^[0-9a-f]{64}$`)
	clavePlan      = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,127}$`)
	claveOperacion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{15,127}$`)
)

type material struct {
	Esquema          string  `json:"esquema"`
	Operacion        string  `json:"operacion"`
	CatalogoID       string  `json:"catalogo_id"`
	Version          int     `json:"version"`
	RevisionEsperada int     `json:"revision_esperada"`
	HuellaEsperada   *string `json:"huella_esperada"`
	ClaveOperacion   string  `json:"clave_operacion"`
	CatalogoBase64   string  `json:"catalogo_canonico_base64"`
	CatalogoSHA      string  `json:"catalogo_sha256"`
	TrazaBase64      string  `json:"traza_canonica_base64"`
	TrazaSHA         string  `json:"traza_sha256"`
	EventoBase64     string  `json:"evento_canonico_base64"`
	EventoSHA        string  `json:"evento_sha256"`
}

type resumen struct {
	SHA256     string `json:"sha256"`
	Operacion  string `json:"operacion"`
	CatalogoID string `json:"catalogo_id"`
	Version    int    `json:"version"`
	Revision   int    `json:"revision"`
	Estado     string `json:"estado"`
}

func main() {
	codigo := ejecutar(os.Args[1:], os.Stdout)
	if codigo != 0 {
		_, _ = os.Stderr.WriteString("material_rechazado\n")
	}
	os.Exit(codigo)
}

func ejecutar(args []string, salida io.Writer) int {
	if len(args) != 2 || !shaValido.MatchString(args[1]) {
		return 2
	}
	// #nosec G703 -- el operador elige explícitamente este fichero local.
	inicial, err := os.Lstat(args[0])
	if err != nil || !inicial.Mode().IsRegular() || inicial.Size() > maxMaterial {
		return 1
	}
	// #nosec G703 -- ruta local explícita; se coteja el mismo fichero regular tras abrirlo.
	f, err := os.Open(args[0])
	if err != nil {
		return 1
	}
	b, err := io.ReadAll(io.LimitReader(f, maxMaterial+1))
	actual, errStat := f.Stat()
	errClose := f.Close()
	if err != nil || errStat != nil || errClose != nil || !os.SameFile(inicial, actual) || len(b) > maxMaterial {
		return 1
	}
	r, err := validar(b, args[1])
	if err != nil {
		return 1
	}
	if json.NewEncoder(salida).Encode(r) != nil {
		return 2
	}
	return 0
}

func validar(b []byte, esperado string) (resumen, error) {
	var cero resumen
	if len(b) < 2 || len(b) > maxMaterial || !shaValido.MatchString(esperado) || huella(b) != esperado || !jsonUnico(b) {
		return cero, errMaterial
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(b, &raw) != nil || !clavesMaterialExactas(raw) {
		return cero, errMaterial
	}
	var m material
	if decodificarEstricto(b, &m) != nil || m.Esquema != "vec.catalogos.plan-firma.gobierno.v1" || !clavePlan.MatchString(m.CatalogoID) ||
		m.Version < 1 || m.Version > 2147483647 || m.RevisionEsperada < 0 || m.RevisionEsperada > 2147483647 || !claveOperacion.MatchString(m.ClaveOperacion) {
		return cero, errMaterial
	}
	if (m.Operacion == "crear" && m.HuellaEsperada != nil) || (m.Operacion != "crear" && (m.HuellaEsperada == nil || !shaValido.MatchString(*m.HuellaEsperada))) {
		return cero, errMaterial
	}
	canon, ok := decodificarBloque(m.CatalogoBase64, m.CatalogoSHA, 2<<20)
	if !ok || !jsonUnico(canon) {
		return cero, errMaterial
	}
	traza, ok := decodificarBloque(m.TrazaBase64, m.TrazaSHA, 65536)
	if !ok || !jsonUnico(traza) {
		return cero, errMaterial
	}
	evento, ok := decodificarBloque(m.EventoBase64, m.EventoSHA, 65536)
	if !ok || !jsonUnico(evento) {
		return cero, errMaterial
	}
	var c vd.CatalogoConfigurable
	if decodificarEstricto(canon, &c) != nil {
		return cero, errMaterial
	}
	clon, err := c.ClonarCanonico()
	if err != nil || c.ModuloID != "contratacion_temporal" || c.FuenteRef == "paquete:ejemplo:vec:v1" ||
		!clavePlan.MatchString(c.ID) || c.ID != m.CatalogoID || c.Version != m.Version || len(c.Entradas) > 64 {
		return cero, errMaterial
	}
	canonGo, err := json.Marshal(clon)
	if err != nil || !bytes.Equal(canon, canonGo) || huella(canon) != m.CatalogoSHA {
		return cero, errMaterial
	}
	estado, actor, fecha, accion := estadoOperacion(c, m.Operacion, m.RevisionEsperada)
	if estado == "" || c.Estado != vd.EstadoCatalogoConfigurable(estado) {
		return cero, errMaterial
	}
	if !validarPasos(c) {
		return cero, errMaterial
	}
	if m.Operacion != "crear" && *m.HuellaEsperada == m.CatalogoSHA {
		return cero, errMaterial
	}
	var a vd.AuditEntry
	var e vd.Event
	if decodificarEstricto(traza, &a) != nil || decodificarEstricto(evento, &e) != nil {
		return cero, errMaterial
	}
	trazaCanon, err := json.Marshal(a)
	if err != nil || !bytes.Equal(traza, trazaCanon) {
		return cero, errMaterial
	}
	eventoCanon, err := json.Marshal(e)
	if err != nil || !bytes.Equal(evento, eventoCanon) {
		return cero, errMaterial
	}
	ref := c.Referencia()
	antes := ""
	if m.HuellaEsperada != nil {
		antes = *m.HuellaEsperada
	}
	regla, motivo := c.FuenteRef, c.MotivoCreacion
	switch m.Operacion {
	case "actualizar":
		motivo = c.MotivoModificacion
	case "publicar":
		regla, motivo = c.AprobacionRef, c.MotivoPublicacion
	case "retirar":
		regla, motivo = c.RetiradaAprobacionRef, c.MotivoRetirada
	}
	if actor == "" || a.ActorID != actor || e.ActorID != actor || a.Action != accion || e.Type != accion ||
		a.ModuleID != "contratacion_temporal" || e.ModuleID != "contratacion_temporal" || a.SubjectRef != ref || e.SubjectRef != ref ||
		a.ActorProfile == "" || a.AuthorizationRef == "" || a.Purpose == "" || a.CorrelationRef == "" ||
		a.ObjectVersion != c.Version || a.RuleRef != regla || a.Reason != motivo || a.Result != "correcto" || a.BeforeHash != antes || a.AfterHash != m.CatalogoSHA ||
		len(a.Metadata) != 4 || a.Metadata["catalogo_id"] != c.ID || a.Metadata["catalogo_version"] != strconv.Itoa(c.Version) ||
		a.Metadata["estado"] != estado || a.Metadata["revision"] != strconv.Itoa(c.Revision) || len(e.Payload) != 5 ||
		e.Payload["catalogo_id"] != c.ID || e.Payload["catalogo_version"] != strconv.Itoa(c.Version) ||
		e.Payload["catalogo_revision"] != strconv.Itoa(c.Revision) || e.Payload["estado"] != estado || e.Payload["huella_sha256"] != m.CatalogoSHA ||
		!a.OccurredAt.Equal(fecha) || !e.OccurredAt.Equal(fecha) {
		return cero, errMaterial
	}
	return resumen{esperado, m.Operacion, c.ID, c.Version, c.Revision, "material_validado_sin_autorizacion"}, nil
}

func clavesMaterialExactas(raw map[string]json.RawMessage) bool {
	if len(raw) != 13 {
		return false
	}
	for _, nombre := range []string{"esquema", "operacion", "catalogo_id", "version", "revision_esperada", "huella_esperada",
		"clave_operacion", "catalogo_canonico_base64", "catalogo_sha256", "traza_canonica_base64",
		"traza_sha256", "evento_canonico_base64", "evento_sha256"} {
		v, ok := raw[nombre]
		if !ok {
			return false
		}
		v = bytes.TrimSpace(v)
		if nombre == "huella_esperada" && bytes.Equal(v, []byte("null")) {
			continue
		}
		if nombre == "version" || nombre == "revision_esperada" {
			if len(v) == 0 || (v[0] < '0' || v[0] > '9') {
				return false
			}
		} else if len(v) == 0 || v[0] != '"' {
			return false
		}
	}
	return true
}

func estadoOperacion(c vd.CatalogoConfigurable, op string, esperada int) (string, string, time.Time, string) {
	switch op {
	case "crear":
		if esperada == 0 && c.Revision == 1 {
			return "borrador", c.CreadoPor, c.CreadoEn, vd.AccionCatalogoBorradorCreado
		}
	case "actualizar":
		if esperada > 0 && c.Revision == esperada+1 {
			return "borrador", c.UltimaModificacionPor, c.UltimaModificacionEn, vd.AccionCatalogoBorradorActualizado
		}
	case "publicar":
		if esperada > 0 && c.Revision == esperada {
			return "publicado", c.PublicadoPor, c.PublicadoEn, vd.AccionCatalogoPublicado
		}
	case "retirar":
		if esperada > 0 && c.Revision == esperada {
			return "retirado", c.RetiradoPor, c.RetiradoEn, vd.AccionCatalogoRetirado
		}
	}
	return "", "", time.Time{}, ""
}

func validarPasos(c vd.CatalogoConfigurable) bool {
	if len(c.Entradas) == 0 {
		return false
	}
	sha, err := c.HuellaSHA256()
	if err != nil {
		return false
	}
	version, err := strconv.ParseUint(strconv.Itoa(c.Version), 10, 64)
	if err != nil {
		return false
	}
	plan := ct.PlanCompetenciaFirmaV2{Version: ct.VersionPlanFirmaV2{Referencia: c.ID, Version: version, HuellaSHA256: sha}, FuenteRef: c.FuenteRef}
	for _, e := range c.Entradas {
		if !clavePlan.MatchString(e.Clave) || len(e.Atributos) != 18 || e.Atributos["esquema"] != ct.EsquemaPlanCompetenciaFirmaV2 {
			return false
		}
		a := e.Atributos
		cv, x := strconv.ParseUint(a["circuito_version"], 10, 64)
		po, y := strconv.ParseUint(a["paso_orden"], 10, 64)
		mv, z := strconv.ParseUint(a["mapeo_version"], 10, 64)
		if x != nil || y != nil || z != nil || strconv.FormatUint(cv, 10) != a["circuito_version"] || strconv.FormatUint(po, 10) != a["paso_orden"] || strconv.FormatUint(mv, 10) != a["mapeo_version"] {
			return false
		}
		plan.Pasos = append(plan.Pasos, ct.CompetenciaPasoFirmaV2{EntradaClave: e.Clave,
			Circuito:  ct.VersionPlanFirmaV2{Referencia: a["circuito_ref"], Version: cv, HuellaSHA256: a["circuito_sha256"]},
			Documento: a["documento"], PasoRef: a["paso_ref"], PasoOrden: po, PerfilEsperadoRef: a["perfil_esperado_ref"],
			RolID: a["rol_id"], CargoRef: a["cargo_ref"], OrganizacionRef: a["organizacion_ref"], UnidadRef: a["unidad_ref"],
			Accion: a["accion_competencial"], Finalidad: a["finalidad"], TipoRecurso: a["tipo_recurso"], EsquemaContexto: a["esquema_contexto"],
			MapeoVersion: mv, MapeoFuenteRef: a["mapeo_fuente_ref"]})
	}
	if plan.Validar() != nil {
		return false
	}
	if c.Estado == vd.EstadoCatalogoBorrador {
		return true
	}
	_, err = plannominal.DesdeCatalogo(c, c.PublicadoEn)
	return err == nil
}

func decodificarBloque(s, h string, max int) ([]byte, bool) {
	if !shaValido.MatchString(h) || len(s) > base64.StdEncoding.EncodedLen(max)+4 {
		return nil, false
	}
	b, err := base64.StdEncoding.Strict().DecodeString(s)
	return b, err == nil && len(b) >= 2 && len(b) <= max && base64.StdEncoding.EncodeToString(b) == s && huella(b) == h
}
func huella(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func decodificarEstricto(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return errMaterial
	}
	if _, err := d.Token(); err != io.EOF {
		return errMaterial
	}
	return nil
}
func jsonUnico(b []byte) bool {
	if !utf8.Valid(b) {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(b))
	if !valorUnico(d, 0) {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}
func valorUnico(d *json.Decoder, n int) bool {
	if n > 32 {
		return false
	}
	t, err := d.Token()
	if err != nil {
		return false
	}
	v, ok := t.(json.Delim)
	if !ok {
		return true
	}
	switch v {
	case '{':
		visto := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			s, ok := k.(string)
			if e != nil || !ok || visto[s] || !valorUnico(d, n+1) {
				return false
			}
			visto[s] = true
		}
		fin, err := d.Token()
		return err == nil && fin == json.Delim('}')
	case '[':
		for d.More() {
			if !valorUnico(d, n+1) {
				return false
			}
		}
		fin, err := d.Token()
		return err == nil && fin == json.Delim(']')
	}
	return false
}
