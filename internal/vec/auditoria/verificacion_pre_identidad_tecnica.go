package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// AD222 registra un rechazo técnico anterior a acreditar la identidad. El
// esquema no le atribuye actor, perfil, decisión V3 ni autenticidad externa.
const EsquemaVerificacionPreIdentidadTecnica = "vec.auditoria.verificacion.pre-identidad-tecnica.v1"

type RegistroPreIdentidadTecnicaV1 struct {
	RegistroOperacionMantenimientoV1
	Fase           string `json:"fase"`
	MetodoEsperado string `json:"metodo_esperado"`
	Ruta           string `json:"ruta"`
	Superficie     string `json:"superficie"`
}

func VerificarCadenaPreIdentidadTecnicaV1(d DocumentoVerificacionMixta, c CoberturaCadena, max uint64) InformeVerificacion {
	return verificarCadenaMixta(d, c, max, EsquemaVerificacionPreIdentidadTecnica)
}

func esquemaAdmiteFamiliasPreviasOPreIdentidad(esquema string) bool {
	return esquemaAdmiteFamiliasPreviasOPresentacion(esquema) || esquema == EsquemaVerificacionPreIdentidadTecnica
}

func motivoPreIdentidadTecnicaValido(resultado, motivo string) bool {
	switch resultado + "/" + motivo {
	case "denegado/certificado_requerido", "denegado/autenticacion_requerida",
		"denegado/acceso_denegado", "denegado/metodo_no_permitido",
		"denegado/recurso_no_encontrado", "denegado/solicitud_invalida",
		"error/servicio_no_disponible", "error/respuesta_incompatible":
		return true
	}
	return false
}

func cotejarPreIdentidadTecnicaV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var evento RegistroEventoAdminV3
	if r.TipoRegistro != "pre_identidad_tecnica_v1" || r.PreIdentidadTecnica == nil ||
		otrasFamiliasPresentesPerfilesAsignables(r) || r.PerfilesAsignables != nil || r.IntentoPerfilesAsignables != nil ||
		r.ProvisionIdentidadInterna != nil || r.IntentoIdentidadInterna != nil ||
		r.CatalogoAcciones != nil || r.IntentoCatalogoAcciones != nil || r.PresentacionCertificado != nil {
		return evento, "tipo_invalido", "tipo_registro"
	}
	m := r.PreIdentidadTecnica
	base := m.RegistroOperacionMantenimientoV1
	if secuencia == 0 || secuencia > 1<<53-1 || base.Secuencia != secuencia {
		return evento, "secuencia_distinta", "secuencia"
	}
	if len(base.EventoRef) != 39 || !strings.HasPrefix(base.EventoRef, "evento_") ||
		!hex32Valido(base.EventoRef[7:]) || base.AuditoriaRef != "aud_v3_pit_"+base.EventoRef[7:] {
		return evento, "referencia_distinta", "auditoria_ref"
	}
	if len(base.CorrelacionRef) != 44 || !strings.HasPrefix(base.CorrelacionRef, "correlacion_") ||
		!hex32Valido(base.CorrelacionRef[12:]) {
		return evento, "registro_invalido", "correlacion_ref"
	}
	recurso := sha256.Sum256([]byte("vec.identidad.preacreditacion.solicitud.v1\n" + base.CorrelacionRef))
	if base.RecursoRef != "solicitud_sesion:"+hex.EncodeToString(recurso[:16]) {
		return evento, "referencia_distinta", "recurso_ref"
	}
	if m.Fase != "preacreditacion" || m.Superficie != "interna_corporativa" ||
		(m.MetodoEsperado != "GET" || m.Ruta != "/api/vec/session") &&
			(m.MetodoEsperado != "POST" || m.Ruta != "/api/vec/session/start") ||
		base.Accion != "registrar_pre_identidad_tecnica_v1" || base.ModuloID != "identidad" ||
		base.Canal != "identidad_http_interno_preacreditacion" || base.FinalidadRef != "preacreditacion_identidad" ||
		!procesoOrdenAD169.MatchString(base.Proceso) || !motivoPreIdentidadTecnicaValido(base.Resultado, base.MotivoRef) {
		return evento, "registro_invalido", "coordenadas"
	}
	if base.AnteriorSHA256 != MarcadorSinAnteriorV5 || !huellaCadenaValida(base.HuellaSHA256) ||
		!huellaCadenaValida(base.EventoMaterialSHA256) || base.OperadorLogin == "" || len(base.OperadorLogin) > 63 ||
		!utf8.ValidString(base.OperadorLogin) || strings.ContainsRune(base.OperadorLogin, 0) {
		return evento, "registro_invalido", "material"
	}
	instante, err := time.Parse(time.RFC3339Nano, base.RegistradaEn)
	if err != nil {
		return evento, motivoErrorInstanteAuditoria(err), "registrada_en"
	}
	if instante.Year() < 1 || instante.Year() > 9999 ||
		instante.UTC().Format("2006-01-02T15:04:05.000000Z") != base.RegistradaEn {
		return evento, "instante_invalido", "registrada_en"
	}
	material := []string{r.TipoRegistro, base.EventoRef, base.OperadorLogin, m.Fase, m.MetodoEsperado,
		m.Ruta, base.Accion, base.RecursoRef, base.Resultado, base.MotivoRef,
		base.Proceso, base.Canal, m.Superficie, base.FinalidadRef, base.CorrelacionRef}
	for _, campo := range material {
		if campo == "" || len(campo) > 200 || !utf8.ValidString(campo) {
			return evento, "registro_invalido", "material"
		}
	}
	suma := sha256.Sum256(huellaEncuadradaIntento(append([]string{"vec.auditoria.pre-identidad-tecnica.v1"}, material...)...))
	if hex.EncodeToString(suma[:]) != base.EventoMaterialSHA256 {
		return evento, "material_distinto", "evento_material_sha256"
	}
	suma = sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon.pre-identidad-tecnica.v1",
		strconv.FormatUint(secuencia, 10), base.AnteriorSHA256, base.AuditoriaRef, base.EventoMaterialSHA256, base.RegistradaEn))
	if hex.EncodeToString(suma[:]) != base.HuellaSHA256 {
		return evento, "huella_distinta", "huella_sha256"
	}
	return RegistroEventoAdminV3{AuditoriaRef: base.AuditoriaRef, Secuencia: secuencia,
		AnteriorSHA256: base.AnteriorSHA256, HuellaSHA256: base.HuellaSHA256, EventoRef: base.EventoRef}, "", ""
}
