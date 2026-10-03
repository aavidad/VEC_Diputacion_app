package consultafirmasv2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ResultadoRecuperacion struct {
	Resultado
	Recuperaciones []ports.RecuperacionFirmaV2
}

type ServicioRecuperacion struct {
	fuente      FuenteContexto
	autorizador ports.AutorizadorRecuperacionFirmasV2
	lector      ports.LectorRecuperacionFirmasV2
}

func NuevaRecuperacion(f FuenteContexto, a ports.AutorizadorRecuperacionFirmasV2, l ports.LectorRecuperacionFirmasV2) (*ServicioRecuperacion, error) {
	if nulo(f) || nulo(a) || nulo(l) {
		return nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return &ServicioRecuperacion{f, a, l}, nil
}

// Recuperar exige una decisión nueva para esta operación y conserva los bytes
// históricos que entrega la autoridad. La fuente de contexto es nominal.
func (s *ServicioRecuperacion) Recuperar(ctx context.Context, q Solicitud) (ResultadoRecuperacion, error) {
	var cero ResultadoRecuperacion
	if ctx == nil || s == nil || nulo(s.fuente) || nulo(s.autorizador) || nulo(s.lector) {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := q.Validar(); err != nil {
		return cero, err
	}
	c, err := s.fuente.ResolverContextoConsultaFirmasR5V2(ctx)
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err != nil {
		return cero, err
	}
	if !domain.ReferenciaOpacaValida(c.OrganizacionRef) || !domain.ReferenciaOpacaValida(c.FirmantePrincipalCandidatoRef) ||
		!strings.HasPrefix(c.FirmantePrincipalCandidatoRef, "per_") {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	m := ports.MaterialConsultaFirmasR5V2{MaterialConsultaFirmasR5: ports.MaterialConsultaFirmasR5{
		OrganizacionRef: c.OrganizacionRef, ExpedienteRef: q.ExpedienteRef, VersionExpediente: q.VersionExpediente,
		Documento: q.Documento, FirmantePrincipalCandidatoRef: c.FirmantePrincipalCandidatoRef,
		PasoOrden: q.PasoOrden, ClaveIdempotencia: q.ClaveIdempotencia, CatalogoHuella: q.CatalogoHuella}, Via: q.Via}
	if _, err := m.Canonico(); err != nil {
		return cero, err
	}
	capacidad, err := s.autorizador.AutorizarRecuperacionFirmasV2(ctx, m)
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err != nil {
		return cero, err
	}
	if err := firmaautorizacionv2.ValidarCapacidadRecuperacionFirmasV2(capacidad, m); err != nil {
		return cero, err
	}
	lectura, err := s.lector.RecuperarFirmasAutorizadasV2(ctx, m, capacidad)
	if err != nil {
		return cero, &falloLector{err}
	}
	if err := ctx.Err(); err != nil {
		return cero, &falloLector{err}
	}
	return proyectarRecuperacion(m, lectura)
}

func proyectarRecuperacion(m ports.MaterialConsultaFirmasR5V2, l ports.LecturaRecuperacionFirmasV2) (ResultadoRecuperacion, error) {
	var cero ResultadoRecuperacion
	r, err := proyectar(m, l.LecturaFirmasR5V2)
	if err != nil {
		return cero, err
	}
	if len(l.Recuperaciones) != len(l.RevisionesPDF) || len(l.Recuperaciones) > MaximoFilas {
		return cero, ports.ErrResultadoFirmaDocumentoInvalido
	}
	porRef := make(map[string]Firma, len(r.Firmas))
	for _, f := range r.Firmas {
		porRef[f.FirmaRef] = f
	}
	vistos := make(map[string]bool, len(l.Recuperaciones))
	for _, v := range l.Recuperaciones {
		f, ok := porRef[v.FirmaRef]
		if !ok || vistos[v.FirmaRef] || f.RevisionPDF == nil || !recuperacionValida(m, f, v) {
			return cero, ports.ErrResultadoFirmaDocumentoInvalido
		}
		vistos[v.FirmaRef] = true
	}
	for _, f := range r.Firmas {
		if f.RevisionPDF != nil && !vistos[f.FirmaRef] {
			return cero, ports.ErrResultadoFirmaDocumentoInvalido
		}
	}
	return ResultadoRecuperacion{r, append([]ports.RecuperacionFirmaV2(nil), l.Recuperaciones...)}, nil
}

func recuperacionValida(m ports.MaterialConsultaFirmasR5V2, f Firma, v ports.RecuperacionFirmaV2) bool {
	if !domain.HuellaSHA256FirmaValida(v.MaterialRootSHA256) ||
		!domain.HuellaSHA256FirmaValida(v.CanonNominalSHA256) ||
		!strings.HasPrefix(v.CanonNominalRef, "evidencia:competencia-firmante-ct:") ||
		!domain.HuellaSHA256FirmaValida(strings.TrimPrefix(v.CanonNominalRef, "evidencia:competencia-firmante-ct:")) ||
		len(v.CanonNominal) < 512 || len(v.CanonNominal) > 32768 || !utf8.ValidString(v.CanonNominal) {
		return false
	}
	h := sha256.Sum256([]byte(v.CanonNominal))
	if v.CanonNominalSHA256 != hex.EncodeToString(h[:]) {
		return false
	}
	// AUT35 emitió un objeto cerrado. Comprobamos los vínculos observables con
	// CT172; el adaptador histórico debe acreditar el resto bajo su propia ACL.
	var canon map[string]json.RawMessage
	if !jsonSinClavesDuplicadas(v.CanonNominal) || json.Unmarshal([]byte(v.CanonNominal), &canon) != nil || !clavesExactas(canon,
		"esquema", "identidad", "competencia", "personal", "recurso", "relacion_ct", "accion", "finalidad",
		"motivo", "circuito", "paso_ref", "paso_orden", "fecha_historica") {
		return false
	}
	var compacto bytes.Buffer
	if json.Compact(&compacto, []byte(v.CanonNominal)) != nil || compacto.String() != v.CanonNominal ||
		textoJSON(canon["esquema"]) != "vec.competencia-firmante.historica.v1" ||
		textoJSON(canon["paso_ref"]) != f.PasoRef || numeroJSON(canon["paso_orden"]) != uint64(f.PasoOrden) {
		return false
	}
	if !objetoConClaves(canon["identidad"], "certificado_der_sha256", "persona_ref", "persona", "cuenta",
		"vinculo_cuenta_persona", "cuenta_persona_cuenta_ref", "cuenta_persona_persona_ref",
		"vinculo_certificado", "vinculo_cuenta_ref", "vinculo_persona_ref", "vinculo_der_sha256") ||
		!objetoConClaves(canon["competencia"], "asignacion", "rol", "rol_id", "control_rol", "persona_ref",
			"perfil_esperado_ref", "perfil_activo_ref", "modulo_id", "tipo_recurso", "recurso_ref",
			"ambito_organizacion_ref", "ambito_unidad_ref", "asignacion_rol_ref", "control_rol_ref",
			"vigente_desde", "vigente_hasta") ||
		!objetoConClaves(canon["personal"], "cargo", "enlace_ocupante", "ocupante_persona_ref", "cargo_ref_enlace",
			"cargo_vigente_desde", "cargo_vigente_hasta", "enlace_vigente_desde", "enlace_vigente_hasta", "delegacion") ||
		!objetoConClaves(canon["relacion_ct"], "expediente_ref", "unidad_ref", "origen_ref", "origen_version",
			"prueba_snapshot_sha256", "evento_ref", "evento_huella_sha256", "confirmada_en") ||
		!objetoConClaves(canon["motivo"], "catalogo_id", "catalogo_version", "catalogo_huella_sha256", "entrada_clave") {
		return false
	}
	identidad, competencia, relacion := mapaJSON(canon["identidad"]), mapaJSON(canon["competencia"]), mapaJSON(canon["relacion_ct"])
	if !domain.ReferenciaOpacaValida(textoJSON(canon["accion"])) ||
		!domain.ReferenciaOpacaValida(textoJSON(canon["finalidad"])) ||
		textoJSON(competencia["persona_ref"]) == "" ||
		textoJSON(competencia["persona_ref"]) != textoJSON(identidad["persona_ref"]) ||
		textoJSON(relacion["expediente_ref"]) != m.ExpedienteRef {
		return false
	}
	var recurso map[string]json.RawMessage
	if json.Unmarshal(canon["recurso"], &recurso) != nil || !clavesExactas(recurso,
		"organizacion_ref", "unidad_ref", "expediente_ref", "documento_ref", "recurso_autorizable_ref",
		"modulo_id", "tipo_recurso", "recurso_contexto_sha256", "original", "pdf_raiz_sha256",
		"firmado", "pdf_firmado_sha256", "numero_firmas", "entrada_revision") ||
		textoJSON(recurso["organizacion_ref"]) != m.OrganizacionRef ||
		textoJSON(recurso["expediente_ref"]) != m.ExpedienteRef ||
		textoJSON(recurso["unidad_ref"]) == "" ||
		textoJSON(relacion["unidad_ref"]) != textoJSON(recurso["unidad_ref"]) ||
		textoJSON(competencia["ambito_organizacion_ref"]) != m.OrganizacionRef ||
		textoJSON(competencia["ambito_unidad_ref"]) != textoJSON(recurso["unidad_ref"]) ||
		textoJSON(competencia["modulo_id"]) != ports.ModuloContratacion ||
		textoJSON(competencia["tipo_recurso"]) != textoJSON(recurso["tipo_recurso"]) ||
		textoJSON(competencia["recurso_ref"]) != f.Original.Ref ||
		textoJSON(recurso["documento_ref"]) != f.Original.Ref ||
		textoJSON(recurso["recurso_autorizable_ref"]) != f.Original.Ref ||
		textoJSON(recurso["modulo_id"]) != ports.ModuloContratacion ||
		!domain.HuellaSHA256FirmaValida(textoJSON(recurso["recurso_contexto_sha256"])) ||
		textoJSON(recurso["pdf_raiz_sha256"]) != f.Original.SHA256 ||
		textoJSON(recurso["pdf_firmado_sha256"]) != f.Custodiado.SHA256 ||
		numeroJSON(recurso["numero_firmas"]) != uint64(f.RevisionPDF.OrdenFirma) ||
		!documentoCanonValido(recurso["original"], f.Original) || !documentoCanonValido(recurso["firmado"], *f.Custodiado) {
		return false
	}
	if f.RevisionPDF.OrdenFirma == 1 {
		if string(recurso["entrada_revision"]) != "null" {
			return false
		}
	} else if !documentoCanonValido(recurso["entrada_revision"], f.RevisionPDF.Entrada) {
		return false
	}
	var circuito map[string]json.RawMessage
	return json.Unmarshal(canon["circuito"], &circuito) == nil && clavesExactas(circuito, "referencia", "version", "huella_sha256") &&
		textoJSON(circuito["referencia"]) == f.CatalogoRef && textoJSON(circuito["huella_sha256"]) == f.CatalogoHuella &&
		numeroJSON(circuito["version"]) > 0
}

func clavesExactas(m map[string]json.RawMessage, nombres ...string) bool {
	if len(m) != len(nombres) {
		return false
	}
	for _, nombre := range nombres {
		if _, ok := m[nombre]; !ok {
			return false
		}
	}
	return true
}

func mapaJSON(raw json.RawMessage) map[string]json.RawMessage {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	return m
}

func objetoConClaves(raw json.RawMessage, nombres ...string) bool {
	m := mapaJSON(raw)
	return clavesExactas(m, nombres...)
}

// Un mapa JSON por sí solo perdería una clave repetida antes del cruce.
func jsonSinClavesDuplicadas(valor string) bool {
	d := json.NewDecoder(strings.NewReader(valor))
	if !leerValorJSONSinDuplicados(d) {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}

func leerValorJSONSinDuplicados(d *json.Decoder) bool {
	t, err := d.Token()
	if err != nil {
		return false
	}
	abre, ok := t.(json.Delim)
	if !ok {
		return true
	}
	switch abre {
	case '{':
		vistos := make(map[string]bool)
		for d.More() {
			clave, err := d.Token()
			nombre, esTexto := clave.(string)
			if err != nil || !esTexto || vistos[nombre] || !leerValorJSONSinDuplicados(d) {
				return false
			}
			vistos[nombre] = true
		}
	case '[':
		for d.More() {
			if !leerValorJSONSinDuplicados(d) {
				return false
			}
		}
	default:
		return false
	}
	cierra, err := d.Token()
	return err == nil && cierra == json.Delim(abre+2)
}

func textoJSON(raw json.RawMessage) string {
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

func numeroJSON(raw json.RawMessage) uint64 {
	var n uint64
	_ = json.Unmarshal(raw, &n)
	return n
}

func documentoCanonValido(raw json.RawMessage, d Documento) bool {
	var campos map[string]json.RawMessage
	return json.Unmarshal(raw, &campos) == nil && clavesExactas(campos, "referencia", "version", "huella_sha256") &&
		textoJSON(campos["referencia"]) == d.Ref && numeroJSON(campos["version"]) == d.Version &&
		textoJSON(campos["huella_sha256"]) == d.SHA256
}
