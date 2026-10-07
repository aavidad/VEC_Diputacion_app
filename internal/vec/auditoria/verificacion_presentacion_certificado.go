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

// AD221 registra la presentación técnica anterior a F1. Este esquema no
// atribuye perfil activo, decisión V3 ni firma documental al certificado.
const EsquemaVerificacionPresentacionCertificado = "vec.auditoria.verificacion.presentacion-certificado.v1"

var fechaJSONBPresentacion = regexp.MustCompile(`\A[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]{1,6})?(?:Z|[+-][0-9]{2}:[0-9]{2})\z`)

type RegistroPresentacionCertificadoV1 struct {
	RegistroOperacionMantenimientoV1
	Fase                      string  `json:"fase"`
	OperacionRef              string  `json:"operacion_ref"`
	PresentacionRef           string  `json:"presentacion_ref"`
	AutenticacionOriginalRef  string  `json:"autenticacion_original_ref"`
	SesionOriginalRef         string  `json:"sesion_original_ref"`
	CuentaRef                 string  `json:"cuenta_ref"`
	ActorCuentaRef            *string `json:"actor_cuenta_ref"`
	ActorAutenticacionRef     *string `json:"actor_autenticacion_ref"`
	ActorSesionRef            *string `json:"actor_sesion_ref"`
	SujetoIDHMAC              string  `json:"sujeto_id_hmac"`
	Superficie                string  `json:"superficie"`
	Tipo                      string  `json:"tipo"`
	ReciboSHA256              string  `json:"recibo_sha256"`
	ACROriginal               string  `json:"acr_original"`
	ACRActual                 *string `json:"acr_actual"`
	PoliticaOriginalRef       string  `json:"politica_original_ref"`
	PoliticaOriginalSHA256    string  `json:"politica_original_sha256"`
	PoliticaActualRef         *string `json:"politica_actual_ref"`
	PoliticaActualSHA256      *string `json:"politica_actual_sha256"`
	CanalSHA256               *string `json:"canal_sha256"`
	AsercionActualSHA256      *string `json:"asercion_actual_sha256"`
	PresentacionValidaHasta   *string `json:"presentacion_valida_hasta"`
	ControlSesionRef          *string `json:"control_sesion_ref"`
	ControlSesionRevision     *string `json:"control_sesion_revision"`
	ControlOrigenOperacionRef *string `json:"control_origen_operacion_ref"`
}

func VerificarCadenaPresentacionCertificadoV1(d DocumentoVerificacionMixta, c CoberturaCadena, max uint64) InformeVerificacion {
	return verificarCadenaMixta(d, c, max, EsquemaVerificacionPresentacionCertificado)
}

func esquemaAdmiteFamiliasPreviasOPresentacion(esquema string) bool {
	return esquemaAdmiteFamiliasPreviasOCatalogo(esquema) || esquema == EsquemaVerificacionPresentacionCertificado
}

func referenciaPresentacionValida(valor, prefijo string) bool {
	if !strings.HasPrefix(valor, prefijo) || len(valor) < len(prefijo)+22 || len(valor) > len(prefijo)+128 {
		return false
	}
	for _, c := range valor[len(prefijo):] {
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func valorPresentacionOpcional(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func revisionControlPresentacionValida(v string) bool {
	if len(v) == 0 || len(v) > 20 || v[0] < '1' || v[0] > '9' {
		return false
	}
	for _, c := range v[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func coordenadasPresentacion(m *RegistroPresentacionCertificadoV1) (accion, motivo, canal string) {
	switch m.Tipo {
	case "apertura":
		return "presentar_certificado_v1", "presentacion_abierta", "certificado_mtls_atestado_por_frontera"
	case "reanudacion":
		return "presentar_certificado_v1", "presentacion_reanudada", "certificado_mtls_atestado_por_frontera"
	case "renovacion":
		return "presentar_certificado_v1", "presentacion_renovada", "certificado_mtls_atestado_por_frontera"
	case "revocacion":
		return "revocar_vinculo_certificado_v1", "vinculo_revocado", "identidad_sesion_vigente"
	case "revocacion_control_is2_sin_actor_humano_nominal":
		return "revocar_sesion_v1", "control_sesion_revocado", "control_sesion_is2"
	}
	return "", "", ""
}

func cotejarPresentacionCertificadoV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var evento RegistroEventoAdminV3
	if r.TipoRegistro != "presentacion_certificado_v1" || r.PresentacionCertificado == nil ||
		otrasFamiliasPresentesPerfilesAsignables(r) || r.PerfilesAsignables != nil || r.IntentoPerfilesAsignables != nil ||
		r.ProvisionIdentidadInterna != nil || r.IntentoIdentidadInterna != nil ||
		r.CatalogoAcciones != nil || r.IntentoCatalogoAcciones != nil {
		return evento, "tipo_invalido", "tipo_registro"
	}
	m := r.PresentacionCertificado
	base := m.RegistroOperacionMantenimientoV1
	if secuencia == 0 || secuencia > 1<<53-1 || base.Secuencia != secuencia {
		return evento, "secuencia_distinta", "secuencia"
	}
	if !referenciaPresentacionValida(m.OperacionRef, "opr_") || !referenciaPresentacionValida(m.PresentacionRef, "prs_") ||
		!referenciaPresentacionValida(m.AutenticacionOriginalRef, "aut_") || !referenciaPresentacionValida(m.SesionOriginalRef, "ses_") ||
		!referenciaPresentacionValida(m.CuentaRef, "cta_") ||
		!referenciaPresentacionValida(m.PoliticaOriginalRef, "pga_") || !huellaCadenaValida(m.SujetoIDHMAC) ||
		!huellaCadenaValida(m.ReciboSHA256) || m.ReciboSHA256 == strings.Repeat("0", 64) ||
		!huellaCadenaValida(m.PoliticaOriginalSHA256) || m.PoliticaOriginalSHA256 == strings.Repeat("0", 64) ||
		m.ACROriginal != "urn:vec:acr:certificado-desarrollo-protegido" ||
		(m.Superficie != "externa_personal" && m.Superficie != "interna_corporativa" && m.Superficie != "administracion_privilegiada") {
		return evento, "registro_invalido", "presentacion"
	}
	tecnica := m.Tipo == "revocacion_control_is2_sin_actor_humano_nominal"
	fase := "pre_f1_sin_perfil_activo"
	if tecnica {
		fase = "sin_actor_humano_nominal"
	}
	if m.Fase != fase {
		return evento, "registro_invalido", "fase"
	}
	if tecnica {
		if m.ActorCuentaRef != nil || m.ActorAutenticacionRef != nil || m.ActorSesionRef != nil ||
			m.ControlSesionRef == nil || !referenciaPresentacionValida(*m.ControlSesionRef, "cse_") ||
			m.ControlSesionRevision == nil || !revisionControlPresentacionValida(*m.ControlSesionRevision) ||
			m.ControlOrigenOperacionRef == nil || *m.ControlOrigenOperacionRef != m.OperacionRef {
			return evento, "registro_invalido", "control_sesion_sin_actor"
		}
	} else if m.ActorCuentaRef == nil || !referenciaPresentacionValida(*m.ActorCuentaRef, "cta_") ||
		m.ActorAutenticacionRef == nil || !referenciaPresentacionValida(*m.ActorAutenticacionRef, "aut_") ||
		m.ActorSesionRef == nil || !referenciaPresentacionValida(*m.ActorSesionRef, "ses_") ||
		m.ControlSesionRef != nil || m.ControlSesionRevision != nil || m.ControlOrigenOperacionRef != nil {
		return evento, "registro_invalido", "actor_y_control"
	}
	accion, motivo, canal := coordenadasPresentacion(m)
	if accion == "" || base.Accion != accion || base.MotivoRef != motivo || base.Canal != canal ||
		base.ModuloID != "identidad" || base.FinalidadRef != "autenticacion_temporal_pre_f1" ||
		base.Resultado != "permitido" || base.Proceso != "postgresql" ||
		base.RecursoRef != "presentacion_certificado:"+m.PresentacionRef {
		return evento, "registro_invalido", "coordenadas"
	}
	instante, err := time.Parse(time.RFC3339Nano, base.RegistradaEn)
	if err != nil {
		return evento, motivoErrorInstanteAuditoria(err), "registrada_en"
	}
	if instante.Year() < 1 || instante.Year() > 9999 || instante.UTC().Format("2006-01-02T15:04:05.000000Z") != base.RegistradaEn {
		return evento, "instante_invalido", "registrada_en"
	}
	validaHastaCanonica := ""
	if m.Tipo == "revocacion" || tecnica {
		if m.ACRActual != nil || m.PoliticaActualRef != nil || m.PoliticaActualSHA256 != nil ||
			m.CanalSHA256 != nil || m.AsercionActualSHA256 != nil || m.PresentacionValidaHasta != nil {
			return evento, "registro_invalido", "actual_revocacion"
		}
	} else {
		if m.ACRActual == nil || *m.ACRActual != m.ACROriginal || m.PoliticaActualRef == nil ||
			*m.PoliticaActualRef != m.PoliticaOriginalRef || m.PoliticaActualSHA256 == nil ||
			*m.PoliticaActualSHA256 != m.PoliticaOriginalSHA256 || m.CanalSHA256 == nil ||
			!huellaCadenaValida(*m.CanalSHA256) || m.AsercionActualSHA256 == nil ||
			!huellaCadenaValida(*m.AsercionActualSHA256) || m.PresentacionValidaHasta == nil {
			return evento, "registro_invalido", "actual_presentacion"
		}
		if !fechaJSONBPresentacion.MatchString(*m.PresentacionValidaHasta) {
			return evento, "instante_invalido", "presentacion_valida_hasta"
		}
		hasta, err := time.Parse(time.RFC3339Nano, *m.PresentacionValidaHasta)
		if err != nil {
			return evento, motivoErrorInstanteAuditoria(err), "presentacion_valida_hasta"
		}
		if hasta.Year() < 1 || hasta.Year() > 9999 || hasta.Nanosecond()%1000 != 0 || !hasta.After(instante) {
			return evento, "instante_invalido", "presentacion_valida_hasta"
		}
		validaHastaCanonica = hasta.UTC().Format("2006-01-02T15:04:05.000000Z")
	}
	eventoSuma := sha256.Sum256([]byte(m.OperacionRef))
	eventoEsperado := "evento_" + hex.EncodeToString(eventoSuma[:])[:32]
	correlacionSuma := sha256.Sum256([]byte(m.OperacionRef + ":" + m.PresentacionRef))
	correlacionEsperada := "correlacion_" + hex.EncodeToString(correlacionSuma[:])[:32]
	if base.EventoRef != eventoEsperado || base.CorrelacionRef != correlacionEsperada ||
		base.AuditoriaRef != "aud_v3_pc_"+eventoEsperado[7:] {
		return evento, "referencia_distinta", "evento_correlacion_auditoria"
	}
	if !huellaCadenaValida(base.AnteriorSHA256) || !huellaCadenaValida(base.HuellaSHA256) ||
		!huellaCadenaValida(base.EventoMaterialSHA256) || base.OperadorLogin == "" || len(base.OperadorLogin) > 63 ||
		!utf8.ValidString(base.OperadorLogin) || strings.ContainsRune(base.OperadorLogin, 0) {
		return evento, "registro_invalido", "material"
	}
	material := []string{base.EventoRef, base.OperadorLogin, m.OperacionRef, m.PresentacionRef,
		m.AutenticacionOriginalRef, m.SesionOriginalRef, m.CuentaRef, valorPresentacionOpcional(m.ActorCuentaRef),
		valorPresentacionOpcional(m.ActorAutenticacionRef), valorPresentacionOpcional(m.ActorSesionRef),
		m.SujetoIDHMAC, m.Superficie, m.Tipo,
		m.ReciboSHA256, m.ACROriginal, valorPresentacionOpcional(m.ACRActual), m.PoliticaOriginalRef,
		m.PoliticaOriginalSHA256, valorPresentacionOpcional(m.PoliticaActualRef),
		valorPresentacionOpcional(m.PoliticaActualSHA256), valorPresentacionOpcional(m.CanalSHA256),
		valorPresentacionOpcional(m.AsercionActualSHA256), validaHastaCanonica,
		base.RegistradaEn, valorPresentacionOpcional(m.ControlSesionRef),
		valorPresentacionOpcional(m.ControlSesionRevision), valorPresentacionOpcional(m.ControlOrigenOperacionRef),
		base.Accion, base.MotivoRef, base.Canal, base.CorrelacionRef}
	for _, campo := range material {
		if len(campo) > 512 || !utf8.ValidString(campo) {
			return evento, "registro_invalido", "material"
		}
	}
	suma := sha256.Sum256(huellaEncuadradaIntento(append([]string{"vec.auditoria.presentacion-certificado.v1"}, material...)...))
	if hex.EncodeToString(suma[:]) != base.EventoMaterialSHA256 {
		return evento, "material_distinto", "evento_material_sha256"
	}
	suma = sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon.presentacion-certificado.v1", strconv.FormatUint(secuencia, 10),
		base.AnteriorSHA256, base.AuditoriaRef, base.EventoMaterialSHA256, base.RegistradaEn))
	if hex.EncodeToString(suma[:]) != base.HuellaSHA256 {
		return evento, "huella_distinta", "huella_sha256"
	}
	return RegistroEventoAdminV3{AuditoriaRef: base.AuditoriaRef, Secuencia: secuencia,
		AnteriorSHA256: base.AnteriorSHA256, HuellaSHA256: base.HuellaSHA256, EventoRef: base.EventoRef}, "", ""
}
