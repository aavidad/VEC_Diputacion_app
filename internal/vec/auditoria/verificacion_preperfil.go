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

const EsquemaVerificacionPreperfil = "vec.auditoria.verificacion.v3"

// MotivoInstanteAD171Invalido conserva el diagnóstico cerrado que recibe la
// CLI cuando el parseo o la representación canónica de la fecha no se admiten.
const MotivoInstanteAD171Invalido = "instante_invalido"

// RegistroEventoAdminV3 proyecta únicamente el material y las coordenadas
// AD171. La huella de fuente no autentica la evidencia privada de origen.
type RegistroEventoAdminV3 struct {
	AuditoriaRef         string `json:"auditoria_ref"`
	Secuencia            uint64 `json:"secuencia"`
	AnteriorSHA256       string `json:"anterior_sha256"`
	HuellaSHA256         string `json:"huella_sha256"`
	RegistradaEn         string `json:"registrada_en"`
	EventoRef            string `json:"evento_ref"`
	EventoMaterialSHA256 string `json:"evento_material_sha256"`
	ModuloID             string `json:"modulo_id"`
	Accion               string `json:"accion"`
	RecursoRef           string `json:"recurso_ref"`
	Resultado            string `json:"resultado"`
	MotivoRef            string `json:"motivo_ref"`
	Proceso              string `json:"proceso"`
	Canal                string `json:"canal"`
	FinalidadRef         string `json:"finalidad_ref"`
	CorrelacionRef       string `json:"correlacion_ref"`
	FuenteRef            string `json:"fuente_ref"`
	FuenteSHA256         string `json:"fuente_sha256"`
}

type RegistroPreperfilV3 struct {
	RegistroEventoAdminV3
	ActorRef string `json:"actor_ref"`
}

type RegistroBootstrapV3 struct {
	RegistroEventoAdminV3
	OperadorLogin string `json:"operador_login"`
	PlanSHA256    string `json:"plan_sha256"`
	AprobacionRef string `json:"aprobacion_ref"`
}

// InformeVerificacionV3 añade el alcance del material AD171. No autentica
// actor, LOGIN, fuente, aprobación ni checkpoint suministrados por archivos.
type InformeVerificacionV3 struct {
	InformeVerificacion
	MaterialEventoRecalculado bool `json:"material_evento_recalculado"`
}

func VerificarCadenaMixtaV3(d DocumentoVerificacionMixta, checkpoint CoberturaCadena, maxRegistros uint64) InformeVerificacionV3 {
	r := InformeVerificacionV3{InformeVerificacion: verificarCadenaMixta(d, checkpoint, maxRegistros, EsquemaVerificacionPreperfil)}
	if r.Estado == "verificada" {
		for _, registro := range d.Registros {
			if registro.Preperfil != nil || registro.Bootstrap != nil {
				r.MaterialEventoRecalculado = true
				break
			}
		}
	}
	return r
}

var (
	referenciaFuenteAD171 = regexp.MustCompile(`\A[A-Za-z0-9][A-Za-z0-9._:-]{0,127}\z`)
	recursoAD171          = regexp.MustCompile(`\A[A-Za-z0-9][A-Za-z0-9._:-]{0,199}\z`)
	actorAD171            = regexp.MustCompile(`\Aper_[A-Za-z0-9_-]{22,128}\z`)
)

// Los diagnósticos devuelven claves y estados fijos, nunca campos de entrada.
func cotejarRegistroAdminV3(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var evento RegistroEventoAdminV3
	var orden []string
	if r.Consumo != nil || r.Intento != nil {
		return evento, "tipo_invalido", "tipo_registro"
	}
	switch r.TipoRegistro {
	case "preperfil_autenticado":
		if r.Preperfil == nil || r.Bootstrap != nil {
			return evento, "tipo_invalido", "tipo_registro"
		}
		p := *r.Preperfil
		evento = p.RegistroEventoAdminV3
		if !actorAD171.MatchString(p.ActorRef) || len(p.ActorRef) > 200 ||
			(p.Accion != "listar_perfiles_propios_admin" && p.Accion != "seleccionar_perfil_admin") ||
			p.Canal != "administracion_privilegiada" || p.FinalidadRef != "seleccion_perfil" {
			return evento, "registro_invalido", "preperfil"
		}
		orden = []string{r.TipoRegistro, p.EventoRef, p.ActorRef}
	case "bootstrap_operador":
		if r.Bootstrap == nil || r.Preperfil != nil {
			return evento, "tipo_invalido", "tipo_registro"
		}
		b := *r.Bootstrap
		evento = b.RegistroEventoAdminV3
		if b.OperadorLogin == "" || len(b.OperadorLogin) > 63 || !utf8.ValidString(b.OperadorLogin) || strings.ContainsRune(b.OperadorLogin, 0) ||
			!huellaCadenaValida(b.PlanSHA256) || !referenciaFuenteAD171.MatchString(b.AprobacionRef) ||
			b.Accion != "ejecutar_plan_bootstrap_admin" || b.Canal != "operacion_tecnica_privada" || b.FinalidadRef != "bootstrap_admin" {
			return evento, "registro_invalido", "bootstrap"
		}
		orden = []string{r.TipoRegistro, b.EventoRef, b.OperadorLogin, b.PlanSHA256, b.AprobacionRef}
	default:
		return evento, "tipo_invalido", "tipo_registro"
	}
	if evento.Secuencia != secuencia {
		return evento, "secuencia_distinta", "secuencia"
	}
	if len(evento.EventoRef) != 39 || !strings.HasPrefix(evento.EventoRef, "evento_") || !hex32Valido(evento.EventoRef[7:]) ||
		evento.AuditoriaRef != "aud_v3_p_"+evento.EventoRef[7:] {
		return evento, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(evento.AnteriorSHA256) || !huellaCadenaValida(evento.HuellaSHA256) ||
		!huellaCadenaValida(evento.EventoMaterialSHA256) || !huellaCadenaValida(evento.FuenteSHA256) ||
		evento.ModuloID != "administracion" || !referenciaFuenteAD171.MatchString(evento.FuenteRef) ||
		!recursoAD171.MatchString(evento.RecursoRef) || !codigoOrdenAD169.MatchString(evento.MotivoRef) ||
		!procesoOrdenAD169.MatchString(evento.Proceso) || len(evento.CorrelacionRef) != 44 ||
		!strings.HasPrefix(evento.CorrelacionRef, "correlacion_") || !hex32Valido(evento.CorrelacionRef[12:]) ||
		(evento.Resultado != "permitido" && evento.Resultado != "denegado" && evento.Resultado != "error") {
		return evento, "registro_invalido", "coordenadas"
	}
	instante, err := time.Parse(time.RFC3339Nano, evento.RegistradaEn)
	if err != nil || instante.Year() < 1 || instante.Year() > 9999 || instante.UTC().Format("2006-01-02T15:04:05.000000Z") != evento.RegistradaEn {
		return evento, MotivoInstanteAD171Invalido, "registrada_en"
	}
	orden = append(orden, evento.Accion, evento.RecursoRef, evento.Resultado, evento.MotivoRef, evento.Proceso,
		evento.Canal, evento.FinalidadRef, evento.CorrelacionRef, evento.FuenteRef, evento.FuenteSHA256)
	material := sha256.Sum256(huellaEncuadradaIntento(append([]string{"vec.auditoria.admin-preperfil.v1"}, orden...)...))
	if hex.EncodeToString(material[:]) != evento.EventoMaterialSHA256 {
		return evento, "material_distinto", "evento_material_sha256"
	}
	eslabon := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon.admin-preperfil.v1", strconv.FormatUint(secuencia, 10),
		evento.AnteriorSHA256, evento.AuditoriaRef, evento.EventoMaterialSHA256, evento.RegistradaEn))
	if hex.EncodeToString(eslabon[:]) != evento.HuellaSHA256 {
		return evento, "huella_distinta", "huella_sha256"
	}
	return evento, "", ""
}
