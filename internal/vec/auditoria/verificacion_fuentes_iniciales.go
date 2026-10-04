package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const EsquemaVerificacionFuentesIniciales = "vec.auditoria.verificacion.fuentes-iniciales.v1"

// RegistroFuentesInicialesV1 proyecta el acto técnico AD174. No acredita
// personas, perfiles ni el origen del LOGIN o de los recibos agregados.
type RegistroFuentesInicialesV1 struct {
	RegistroEventoAdminV3
	OperadorLogin       string `json:"operador_login"`
	PlanRef             string `json:"plan_ref"`
	PlanSHA256          string `json:"plan_sha256"`
	PreimagenSHA256     string `json:"preimagen_sha256"`
	ConfiguracionSHA256 string `json:"configuracion_sha256"`
	AprobacionRef       string `json:"aprobacion_ref"`
	AlcanceFuente       string `json:"alcance_fuente"`
}

type InformeVerificacionFuentesIniciales struct {
	InformeVerificacion
	MaterialEventoRecalculado          bool `json:"material_evento_recalculado"`
	MaterialFuentesRecalculado         bool `json:"material_fuentes_recalculado"`
	MaterialIntentosFuentesRecalculado bool `json:"material_intentos_fuentes_recalculado"`
}

// VerificarCadenaFuentesInicialesV1 admite consumo histórico v1, AD169, AD171
// y AD174; no admite las familias AD172/173 pendientes de integración.
func VerificarCadenaFuentesInicialesV1(d DocumentoVerificacionMixta, checkpoint CoberturaCadena, maxRegistros uint64) InformeVerificacionFuentesIniciales {
	r := InformeVerificacionFuentesIniciales{InformeVerificacion: verificarCadenaMixta(d, checkpoint, maxRegistros, EsquemaVerificacionFuentesIniciales)}
	if r.Estado == "verificada" {
		for _, registro := range d.Registros {
			r.MaterialEventoRecalculado = r.MaterialEventoRecalculado || registro.Preperfil != nil || registro.Bootstrap != nil
			r.MaterialFuentesRecalculado = r.MaterialFuentesRecalculado || registro.FuentesIniciales != nil
			r.MaterialIntentosFuentesRecalculado = r.MaterialIntentosFuentesRecalculado || registro.IntentoFuentesIniciales != nil
		}
	}
	return r
}

func cotejarRegistroFuentesInicialesV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var e RegistroEventoAdminV3
	if r.FuentesIniciales == nil || r.Consumo != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil || r.IntentoFuentesIniciales != nil {
		return e, "tipo_invalido", "tipo_registro"
	}
	f := *r.FuentesIniciales
	e = f.RegistroEventoAdminV3
	if e.Secuencia != secuencia {
		return e, "secuencia_distinta", "secuencia"
	}
	if len(e.EventoRef) != 39 || !strings.HasPrefix(e.EventoRef, "evento_") || !hex32Valido(e.EventoRef[7:]) || e.AuditoriaRef != "aud_v3_f_"+e.EventoRef[7:] {
		return e, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(e.AnteriorSHA256) || !huellaCadenaValida(e.HuellaSHA256) || !huellaCadenaValida(e.EventoMaterialSHA256) ||
		!huellaCadenaValida(e.FuenteSHA256) || !huellaCadenaValida(f.PlanSHA256) || !huellaCadenaValida(f.PreimagenSHA256) || !huellaCadenaValida(f.ConfiguracionSHA256) ||
		!referenciaFuenteAD171.MatchString(f.PlanRef) || !referenciaFuenteAD171.MatchString(f.AprobacionRef) || !referenciaFuenteAD171.MatchString(e.FuenteRef) ||
		!recursoAD171.MatchString(e.RecursoRef) || !codigoOrdenAD169.MatchString(e.MotivoRef) || !procesoOrdenAD169.MatchString(e.Proceso) ||
		len(e.CorrelacionRef) != 44 || !strings.HasPrefix(e.CorrelacionRef, "correlacion_") || !hex32Valido(e.CorrelacionRef[12:]) ||
		f.OperadorLogin == "" || len(f.OperadorLogin) > 63 || !utf8.ValidString(f.OperadorLogin) || strings.ContainsRune(f.OperadorLogin, 0) ||
		f.AlcanceFuente != "sintetico_declarado" || e.ModuloID != "administracion" || e.Accion != "provisionar_fuentes_iniciales_admin_v1" ||
		e.Resultado != "permitido" || e.Canal != "operacion_tecnica_privada" || e.FinalidadRef != "provision_fuentes_iniciales_admin" {
		return e, "registro_invalido", "coordenadas"
	}
	instante, err := time.Parse(time.RFC3339Nano, e.RegistradaEn)
	if err != nil {
		return e, motivoErrorInstanteAuditoria(err), "registrada_en"
	}
	if instante.Year() < 1 || instante.Year() > 9999 || instante.UTC().Format("2006-01-02T15:04:05.000000Z") != e.RegistradaEn {
		return e, "instante_invalido", "registrada_en"
	}
	material := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.fuentes-iniciales.v1", r.TipoRegistro, e.EventoRef, f.OperadorLogin,
		f.PlanRef, f.PlanSHA256, f.PreimagenSHA256, f.ConfiguracionSHA256, f.AprobacionRef, f.AlcanceFuente,
		e.Accion, e.RecursoRef, e.Resultado, e.MotivoRef, e.Proceso, e.Canal, e.FinalidadRef, e.CorrelacionRef, e.FuenteRef, e.FuenteSHA256))
	if hex.EncodeToString(material[:]) != e.EventoMaterialSHA256 {
		return e, "material_distinto", "evento_material_sha256"
	}
	eslabon := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon.fuentes-iniciales.v1", strconv.FormatUint(secuencia, 10), e.AnteriorSHA256, e.AuditoriaRef, e.EventoMaterialSHA256, e.RegistradaEn))
	if hex.EncodeToString(eslabon[:]) != e.HuellaSHA256 {
		return e, "huella_distinta", "huella_sha256"
	}
	return e, "", ""
}
