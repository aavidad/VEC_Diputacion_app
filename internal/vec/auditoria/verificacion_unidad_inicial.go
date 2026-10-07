package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const EsquemaVerificacionUnidadInicial = "vec.auditoria.verificacion.unidad-inicial-personal.v1"

// RegistroUnidadInicialPersonalV1 proyecta el acto técnico AD176. No acredita
// personas, perfiles ni el origen del LOGIN o de los recibos agregados.
type RegistroUnidadInicialPersonalV1 struct {
	RegistroEventoAdminV3
	OperadorLogin       string `json:"operador_login"`
	PlanRef             string `json:"plan_ref"`
	PlanSHA256          string `json:"plan_sha256"`
	PreimagenSHA256     string `json:"preimagen_sha256"`
	ConfiguracionSHA256 string `json:"configuracion_sha256"`
	AprobacionRef       string `json:"aprobacion_ref"`
	AlcanceFuente       string `json:"alcance_fuente"`
	ReciboRef           string `json:"recibo_ref"`
	ReciboSHA256        string `json:"recibo_sha256"`
}

type InformeVerificacionUnidadInicial struct {
	InformeVerificacionFuentesIniciales
	MaterialUnidadRecalculado         bool `json:"material_unidad_recalculado"`
	MaterialIntentosUnidadRecalculado bool `json:"material_intentos_unidad_recalculado"`
}

// VerificarCadenaUnidadInicialV1 conserva las familias instaladas hasta AD174,
// añadiendo sólo publicación de unidad AD176; no admite AD172/173 futuras.
func VerificarCadenaUnidadInicialV1(d DocumentoVerificacionMixta, checkpoint CoberturaCadena, maxRegistros uint64) InformeVerificacionUnidadInicial {
	r := InformeVerificacionUnidadInicial{InformeVerificacionFuentesIniciales: InformeVerificacionFuentesIniciales{InformeVerificacion: verificarCadenaMixta(d, checkpoint, maxRegistros, EsquemaVerificacionUnidadInicial)}}
	if r.Estado == "verificada" {
		for _, x := range d.Registros {
			r.MaterialEventoRecalculado = r.MaterialEventoRecalculado || x.Preperfil != nil || x.Bootstrap != nil
			r.MaterialFuentesRecalculado = r.MaterialFuentesRecalculado || x.FuentesIniciales != nil
			r.MaterialIntentosFuentesRecalculado = r.MaterialIntentosFuentesRecalculado || x.IntentoFuentesIniciales != nil
			r.MaterialUnidadRecalculado = r.MaterialUnidadRecalculado || x.UnidadInicial != nil
			r.MaterialIntentosUnidadRecalculado = r.MaterialIntentosUnidadRecalculado || x.IntentoUnidadInicial != nil
		}
	}
	return r
}

func cotejarRegistroUnidadInicialV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var e RegistroEventoAdminV3
	if r.UnidadInicial == nil || r.Consumo != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil || r.IntentoUnidadInicial != nil || r.FuentesIniciales != nil || r.IntentoFuentesIniciales != nil {
		return e, "tipo_invalido", "tipo_registro"
	}
	f := *r.UnidadInicial
	e = f.RegistroEventoAdminV3
	if e.Secuencia != secuencia {
		return e, "secuencia_distinta", "secuencia"
	}
	if len(e.EventoRef) != 39 || !strings.HasPrefix(e.EventoRef, "evento_") || !hex32Valido(e.EventoRef[7:]) || e.AuditoriaRef != "aud_v3_u_"+e.EventoRef[7:] {
		return e, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(e.AnteriorSHA256) || !huellaCadenaValida(e.HuellaSHA256) || !huellaCadenaValida(e.EventoMaterialSHA256) ||
		!huellaCadenaValida(e.FuenteSHA256) || !huellaCadenaValida(f.ReciboSHA256) || !referenciaReciboUnidadValida(f.ReciboRef) || !huellaCadenaValida(f.PlanSHA256) || !huellaCadenaValida(f.PreimagenSHA256) || !huellaCadenaValida(f.ConfiguracionSHA256) ||
		!referenciaPlanUnidadInicialValida(f.PlanRef) || !referenciaFuenteAD171.MatchString(f.AprobacionRef) || !referenciaFuenteAD171.MatchString(e.FuenteRef) ||
		!recursoUnidadInicial.MatchString(e.RecursoRef) || e.MotivoRef != "unidad_registrada" || e.Proceso != "postgresql" ||
		len(e.CorrelacionRef) != 44 || !strings.HasPrefix(e.CorrelacionRef, "correlacion_") || !hex32Valido(e.CorrelacionRef[12:]) ||
		f.OperadorLogin == "" || len(f.OperadorLogin) > 63 || !utf8.ValidString(f.OperadorLogin) || strings.ContainsRune(f.OperadorLogin, 0) ||
		f.AlcanceFuente != "sintetico_declarado" || e.ModuloID != "personal" || e.Accion != "inicializar_unidad_sintetica_admin_v1" ||
		e.Resultado != "permitido" || e.Canal != "operacion_tecnica_privada" || e.FinalidadRef != "inicializar_unidad_sintetica_admin" {
		return e, "registro_invalido", "coordenadas"
	}
	instante, err := time.Parse(time.RFC3339Nano, e.RegistradaEn)
	if err != nil {
		return e, motivoErrorInstanteAuditoria(err), "registrada_en"
	}
	if instante.Year() < 1 || instante.Year() > 9999 || instante.UTC().Format("2006-01-02T15:04:05.000000Z") != e.RegistradaEn {
		return e, "instante_invalido", "registrada_en"
	}
	material := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.unidad-inicial-personal.v1", r.TipoRegistro, e.EventoRef, f.OperadorLogin,
		f.PlanRef, f.PlanSHA256, f.PreimagenSHA256, f.ConfiguracionSHA256, f.AprobacionRef, f.AlcanceFuente,
		e.Accion, e.RecursoRef, e.Resultado, e.MotivoRef, e.Proceso, e.Canal, e.FinalidadRef, e.CorrelacionRef, e.FuenteRef, e.FuenteSHA256, f.ReciboRef, f.ReciboSHA256))
	if hex.EncodeToString(material[:]) != e.EventoMaterialSHA256 {
		return e, "material_distinto", "evento_material_sha256"
	}
	eslabon := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon.unidad-inicial-personal.v1", strconv.FormatUint(secuencia, 10), e.AnteriorSHA256, e.AuditoriaRef, e.EventoMaterialSHA256, e.RegistradaEn))
	if hex.EncodeToString(eslabon[:]) != e.HuellaSHA256 {
		return e, "huella_distinta", "huella_sha256"
	}
	return e, "", ""
}

func referenciaPlanUnidadInicialValida(ref string) bool {
	if len(ref) > 128 || !strings.HasPrefix(ref, "pui_") || len(ref) < 26 {
		return false
	}
	for _, c := range ref[4:] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

var recursoUnidadInicial = regexp.MustCompile(`\Aunidad:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\z`)

func referenciaReciboUnidadValida(ref string) bool {
	return len(ref) == 46 && strings.HasPrefix(ref, "recibo_unidad:") && hex32Valido(ref[14:])
}
