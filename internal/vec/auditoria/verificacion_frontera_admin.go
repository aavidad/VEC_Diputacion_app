package auditoria

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

//go:embed verificacion_frontera_admin_codigos.json
var codigosFronteraAdminJSON string

const EsquemaVerificacionFronteraAdminTecnicaV1 = "vec.auditoria.verificacion.frontera-admin-tecnica.v1"

type RegistroFronteraAdminTecnicaV1 struct {
	AuditoriaRef         string `json:"auditoria_ref"`
	Secuencia            uint64 `json:"secuencia"`
	AnteriorSHA256       string `json:"anterior_sha256"`
	HuellaSHA256         string `json:"huella_sha256"`
	RegistradaEn         string `json:"registrada_en"`
	TipoRegistro         string `json:"tipo_registro"`
	EventoRef            string `json:"evento_ref"`
	EventoMaterialSHA256 string `json:"evento_material_sha256"`
	OperadorLogin        string `json:"operador_login"`
	Accion               string `json:"accion"`
	ModuloID             string `json:"modulo_id"`
	RecursoRef           string `json:"recurso_ref"`
	Resultado            string `json:"resultado"`
	CodigoRef            string `json:"codigo_ref"`
	Proceso              string `json:"proceso"`
	Canal                string `json:"canal"`
	FinalidadRef         string `json:"finalidad_ref"`
	CorrelacionRef       string `json:"correlacion_ref"`
}

// Verificador puro del eslabón nuevo. La unión con el lector de cadena común
// necesita un turno posterior sobre su despacho; no admite familias futuras.
func CotejarRegistroFronteraAdminTecnicaV1(r RegistroFronteraAdminTecnicaV1, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var vacio RegistroEventoAdminV3
	if r.TipoRegistro != "frontera_admin_tecnica" || r.Secuencia != secuencia || secuencia == 0 || secuencia > 1<<53-1 {
		return vacio, "registro_invalido", "tipo_o_secuencia"
	}
	if len(r.EventoRef) != 39 || !strings.HasPrefix(r.EventoRef, "evento_") || !hex32Valido(r.EventoRef[7:]) || r.AuditoriaRef != "aud_v3_fat_"+r.EventoRef[7:] {
		return vacio, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(r.AnteriorSHA256) || !huellaCadenaValida(r.HuellaSHA256) || !huellaCadenaValida(r.EventoMaterialSHA256) || r.OperadorLogin == "" || len(r.OperadorLogin) > 63 || !utf8.ValidString(r.OperadorLogin) || strings.ContainsRune(r.OperadorLogin, 0) || r.Accion != "controlar_frontera_admin_v1" || r.ModuloID != "administracion" || r.Canal != "administracion_privilegiada" || r.FinalidadRef != "control_frontera_admin" || len(r.CorrelacionRef) != 44 || !strings.HasPrefix(r.CorrelacionRef, "correlacion_") || !hex32Valido(r.CorrelacionRef[12:]) || !procesoFronteraAdminValido(r.Proceso) || !codigoFronteraAdminValido(r.Resultado, r.CodigoRef) {
		return vacio, "registro_invalido", "coordenadas"
	}
	recurso := sha256.Sum256([]byte("vec.admin.frontera.solicitud.v1\n" + r.CorrelacionRef))
	if r.RecursoRef != "solicitud_admin:"+hex.EncodeToString(recurso[:16]) {
		return vacio, "referencia_distinta", "recurso_ref"
	}
	t, err := time.Parse(time.RFC3339Nano, r.RegistradaEn)
	if err != nil {
		return vacio, motivoErrorInstanteAuditoria(err), "registrada_en"
	}
	if t.Year() < 1 || t.Year() > 9999 || t.UTC().Format("2006-01-02T15:04:05.000000Z") != r.RegistradaEn {
		return vacio, "instante_invalido", "registrada_en"
	}
	material := []string{"vec.auditoria.frontera-admin-tecnica.v1", r.TipoRegistro, r.EventoRef, r.OperadorLogin, r.Accion, r.RecursoRef, r.Resultado, r.CodigoRef, r.Proceso, r.Canal, r.FinalidadRef, r.CorrelacionRef}
	h := sha256.Sum256(huellaEncuadradaIntento(material...))
	if hex.EncodeToString(h[:]) != r.EventoMaterialSHA256 {
		return vacio, "material_distinto", "evento_material_sha256"
	}
	h = sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon.frontera-admin-tecnica.v1", strconv.FormatUint(secuencia, 10), r.AnteriorSHA256, r.AuditoriaRef, r.EventoMaterialSHA256, r.RegistradaEn))
	if hex.EncodeToString(h[:]) != r.HuellaSHA256 {
		return vacio, "huella_distinta", "huella_sha256"
	}
	return RegistroEventoAdminV3{AuditoriaRef: r.AuditoriaRef, Secuencia: r.Secuencia, AnteriorSHA256: r.AnteriorSHA256, HuellaSHA256: r.HuellaSHA256, EventoRef: r.EventoRef}, "", ""
}
func codigoFronteraAdminValido(resultado, codigo string) bool {
	var datos []struct {
		Codigo    string `json:"codigo_ref"`
		Resultado string `json:"resultado"`
	}
	if json.Unmarshal([]byte(codigosFronteraAdminJSON), &datos) != nil {
		return false
	}
	for _, d := range datos {
		if d.Codigo == codigo && d.Resultado == resultado {
			return true
		}
	}
	return false
}
func procesoFronteraAdminValido(v string) bool {
	if len(v) < 2 || len(v) > 80 || v[0] < 'a' || v[0] > 'z' {
		return false
	}
	for _, c := range v[1:] {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' && c != '-' && c != '.' {
			return false
		}
	}
	return true
}
