package auditoria

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const EsquemaVerificacionContextoAdminPreV2 = "vec.auditoria.verificacion.contexto-admin-pre-v2.v1"

//go:embed verificacion_contexto_admin_prev2_motivos.json
var motivosContextoAdminPreV2 []byte

type RegistroContextoAdminPreV2 struct {
	AuditoriaRef         string  `json:"auditoria_ref"`
	Secuencia            uint64  `json:"secuencia"`
	AnteriorSHA256       string  `json:"anterior_sha256"`
	HuellaSHA256         string  `json:"huella_sha256"`
	RegistradaEn         string  `json:"registrada_en"`
	TipoRegistro         string  `json:"tipo_registro"`
	EventoRef            string  `json:"evento_ref"`
	EventoMaterialSHA256 string  `json:"evento_material_sha256"`
	OperadorLogin        string  `json:"operador_login"`
	ActorRef             *string `json:"actor_ref"`
	PerfilActivoRef      *string `json:"perfil_activo_ref"`
	Accion               string  `json:"accion"`
	ModuloID             string  `json:"modulo_id"`
	RecursoRef           string  `json:"recurso_ref"`
	Resultado            string  `json:"resultado"`
	MotivoRef            string  `json:"motivo_ref"`
	Proceso              string  `json:"proceso"`
	Canal                string  `json:"canal"`
	FinalidadRef         string  `json:"finalidad_ref"`
	CorrelacionRef       string  `json:"correlacion_ref"`
	FuenteRef            *string `json:"fuente_ref"`
	FuenteSHA256         *string `json:"fuente_sha256"`
}

var referenciaFuentePreV2 = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,159}$`)
var personaPreV2 = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,124}$`)
var perfilPreV2 = regexp.MustCompile(`^prf_[A-Za-z0-9_-]{22,124}$`)

func valorNullablePreV2(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
func motivoContextoAdminPreV2Valido(accion, resultado, motivo string) bool {
	var catalogo []struct {
		Accion    string `json:"accion"`
		Resultado string `json:"resultado"`
		Motivo    string `json:"motivo_ref"`
	}
	if json.Unmarshal(motivosContextoAdminPreV2, &catalogo) != nil {
		return false
	}
	for _, c := range catalogo {
		if c.Accion == accion && c.Resultado == resultado && c.Motivo == motivo {
			return true
		}
	}
	return false
}
func fuenteContextoAdminPreV2Valida(r RegistroContextoAdminPreV2) bool {
	if (r.FuenteRef == nil) != (r.FuenteSHA256 == nil) {
		return false
	}
	if r.FuenteRef != nil && (!referenciaFuentePreV2.MatchString(*r.FuenteRef) || !huellaCadenaValida(*r.FuenteSHA256)) {
		return false
	}
	if r.ActorRef != nil && (!personaPreV2.MatchString(*r.ActorRef) || r.FuenteRef == nil) {
		return false
	}
	if r.PerfilActivoRef != nil && (r.ActorRef == nil || !perfilPreV2.MatchString(*r.PerfilActivoRef)) {
		return false
	}
	return r.Resultado != "permitido" || (r.ActorRef != nil && r.PerfilActivoRef != nil && r.FuenteRef != nil)
}
func cotejarContextoAdminPreV2(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var cero RegistroEventoAdminV3
	if r.TipoRegistro != "contexto_admin_pre_v2" || r.ContextoAdminPreV2 == nil || r.Consumo != nil || r.ConsumoOrigen != nil || r.ConsumoFecha != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil || r.FronteraAdminTecnica != nil || r.GobiernoUsuarios != nil || r.IntentoGobiernoUsuarios != nil || r.Preservacion != nil || r.Periodica != nil || r.MantenimientoFijo != nil || r.IntentoMantenimientoFijo != nil || r.IntentoBootstrapCentral != nil || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil || r.FuentesIniciales != nil || r.IntentoFuentesIniciales != nil {
		return cero, "tipo_invalido", "tipo_registro"
	}
	a := *r.ContextoAdminPreV2
	if a.TipoRegistro != "contexto_admin_pre_v2" || a.Secuencia != secuencia || secuencia == 0 || secuencia > 1<<53-1 {
		return cero, "secuencia_distinta", "secuencia"
	}
	if len(a.EventoRef) != 39 || !strings.HasPrefix(a.EventoRef, "evento_") || !hex32Valido(a.EventoRef[7:]) || a.AuditoriaRef != "aud_v3_ap2_"+a.EventoRef[7:] {
		return cero, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(a.AnteriorSHA256) || !huellaCadenaValida(a.HuellaSHA256) || !huellaCadenaValida(a.EventoMaterialSHA256) || a.OperadorLogin == "" || len(a.OperadorLogin) > 63 || !utf8.ValidString(a.OperadorLogin) || strings.ContainsRune(a.OperadorLogin, 0) || a.ModuloID != "administracion" || a.Canal != "administracion_privilegiada" || a.FinalidadRef != "establecer_contexto_admin" || !procesoFronteraAdminValido(a.Proceso) || !referenciaFuentePreV2.MatchString(a.RecursoRef) || len(a.CorrelacionRef) != 44 || !strings.HasPrefix(a.CorrelacionRef, "correlacion_") || !hex32Valido(a.CorrelacionRef[12:]) || !motivoContextoAdminPreV2Valido(a.Accion, a.Resultado, a.MotivoRef) || !fuenteContextoAdminPreV2Valida(a) {
		return cero, "registro_invalido", "coordenadas"
	}
	t, err := time.Parse(time.RFC3339Nano, a.RegistradaEn)
	if err != nil {
		return cero, motivoErrorInstanteAuditoria(err), "registrada_en"
	}
	if t.Year() < 1 || t.Year() > 9999 || t.UTC().Format("2006-01-02T15:04:05.000000Z") != a.RegistradaEn {
		return cero, "instante_invalido", "registrada_en"
	}
	fields := []string{"vec.auditoria.contexto-admin-pre-v2.v1", a.TipoRegistro, a.EventoRef, a.OperadorLogin, valorNullablePreV2(a.ActorRef), valorNullablePreV2(a.PerfilActivoRef), a.Accion, a.RecursoRef, a.Resultado, a.MotivoRef, a.Proceso, a.Canal, a.FinalidadRef, a.CorrelacionRef, valorNullablePreV2(a.FuenteRef), valorNullablePreV2(a.FuenteSHA256)}
	h := sha256.Sum256(huellaEncuadradaIntento(fields...))
	if hex.EncodeToString(h[:]) != a.EventoMaterialSHA256 {
		return cero, "material_distinto", "evento_material_sha256"
	}
	h = sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon.contexto-admin-pre-v2.v1", strconv.FormatUint(secuencia, 10), a.AnteriorSHA256, a.AuditoriaRef, a.EventoMaterialSHA256, a.RegistradaEn))
	if hex.EncodeToString(h[:]) != a.HuellaSHA256 {
		return cero, "huella_distinta", "huella_sha256"
	}
	return RegistroEventoAdminV3{AuditoriaRef: a.AuditoriaRef, Secuencia: a.Secuencia, AnteriorSHA256: a.AnteriorSHA256, HuellaSHA256: a.HuellaSHA256, EventoRef: a.EventoRef}, "", ""
}
func VerificarCadenaContextoAdminPreV2(d DocumentoVerificacionMixta, c CoberturaCadena, max uint64) InformeVerificacion {
	return verificarCadenaMixta(d, c, max, EsquemaVerificacionContextoAdminPreV2)
}
