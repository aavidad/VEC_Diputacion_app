package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const EsquemaVerificacionIdentidadInternaSintetica = "vec.auditoria.verificacion.identidad-interna-sintetica.v1"

type RegistroProvisionIdentidadInternaSinteticaV1 struct {
	RegistroOperacionMantenimientoV1
	OperacionRef        string `json:"operacion_ref"`
	PlanRef             string `json:"plan_ref"`
	PlanSHA256          string `json:"plan_sha256"`
	PreimagenSHA256     string `json:"preimagen_sha256"`
	ConfiguracionSHA256 string `json:"configuracion_sha256"`
	AprobacionRef       string `json:"aprobacion_ref"`
	AlcanceFuente       string `json:"alcance_fuente"`
	FuenteRef           string `json:"fuente_ref"`
	FuenteSHA256        string `json:"fuente_sha256"`
}

type RegistroIntentoIdentidadInternaSinteticaV1 struct {
	RegistroOperacionMantenimientoV1
	SolicitudSHA256 string `json:"solicitud_sha256"`
}

func VerificarCadenaIdentidadInternaSinteticaV1(d DocumentoVerificacionMixta, c CoberturaCadena, max uint64) InformeVerificacion {
	return verificarCadenaMixta(d, c, max, EsquemaVerificacionIdentidadInternaSintetica)
}

func esquemaAdmiteFamiliasAdminOIdentidad(esquema string) bool {
	return esquemaAdmiteFamiliasAdmin(esquema) || esquema == EsquemaVerificacionIdentidadInternaSintetica
}

func cotejarIdentidadInternaSinteticaV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var evento RegistroEventoAdminV3
	if otrasFamiliasPresentesPerfilesAsignables(r) || r.PerfilesAsignables != nil || r.IntentoPerfilesAsignables != nil ||
		(r.ProvisionIdentidadInterna != nil && r.IntentoIdentidadInterna != nil) {
		return evento, "tipo_invalido", "tipo_registro"
	}
	var base RegistroOperacionMantenimientoV1
	var material []string
	var dominio, prefijo string
	switch r.TipoRegistro {
	case "provision_identidad_interna_sintetica":
		if r.ProvisionIdentidadInterna == nil {
			return evento, "tipo_invalido", "tipo_registro"
		}
		p := r.ProvisionIdentidadInterna
		base = p.RegistroOperacionMantenimientoV1
		dominio, prefijo = "identidad-interna-sintetica.v1", "aud_v3_ii_"
		if !strings.HasPrefix(p.OperacionRef, "piis_") || len(p.OperacionRef) < 27 || len(p.OperacionRef) > 128 ||
			!referenciaFuenteAD171.MatchString(p.PlanRef) || !huellaCadenaValida(p.PlanSHA256) ||
			!huellaCadenaValida(p.PreimagenSHA256) || !huellaCadenaValida(p.ConfiguracionSHA256) ||
			!referenciaFuenteAD171.MatchString(p.AprobacionRef) || !referenciaFuenteAD171.MatchString(p.FuenteRef) ||
			!huellaCadenaValida(p.FuenteSHA256) || p.AlcanceFuente != "sintetico_declarado" ||
			base.Accion != "provisionar_identidad_interna_sintetica_v1" || base.Resultado != "permitido" ||
			base.MotivoRef != "identidad_interna_registrada" || base.RecursoRef != "identidad_interna_sintetica:"+p.OperacionRef {
			return evento, "registro_invalido", "provision_identidad"
		}
		for _, c := range p.OperacionRef[len("piis_"):] {
			if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
				return evento, "registro_invalido", "operacion_ref"
			}
		}
		material = []string{r.TipoRegistro, base.EventoRef, base.OperadorLogin, p.OperacionRef, p.PlanRef, p.PlanSHA256,
			p.PreimagenSHA256, p.ConfiguracionSHA256, p.AprobacionRef, p.AlcanceFuente, base.Accion, base.RecursoRef,
			base.Resultado, base.MotivoRef, base.Proceso, base.Canal, base.FinalidadRef, base.CorrelacionRef, p.FuenteRef, p.FuenteSHA256}
	case "intento_identidad_interna_sintetica":
		if r.IntentoIdentidadInterna == nil {
			return evento, "tipo_invalido", "tipo_registro"
		}
		i := r.IntentoIdentidadInterna
		base = i.RegistroOperacionMantenimientoV1
		dominio, prefijo = "intento-identidad-interna-sintetica.v1", "aud_v3_iii_"
		if !huellaCadenaValida(i.SolicitudSHA256) || len(base.RecursoRef) != len("solicitud_identidad_interna:")+32 ||
			!strings.HasPrefix(base.RecursoRef, "solicitud_identidad_interna:") || !hex32Valido(strings.TrimPrefix(base.RecursoRef, "solicitud_identidad_interna:")) ||
			!motivoIntentoIdentidadInternaValido(base.Accion, base.Resultado, base.MotivoRef) {
			return evento, "registro_invalido", "intento_identidad"
		}
		material = []string{r.TipoRegistro, base.EventoRef, base.OperadorLogin, i.SolicitudSHA256, base.Accion,
			base.RecursoRef, base.Resultado, base.MotivoRef, base.Proceso, base.Canal, base.FinalidadRef, base.CorrelacionRef}
	default:
		return evento, "tipo_invalido", "tipo_registro"
	}
	if secuencia == 0 || secuencia > 1<<53-1 || base.Secuencia != secuencia {
		return evento, "secuencia_distinta", "secuencia"
	}
	if len(base.EventoRef) != 39 || !strings.HasPrefix(base.EventoRef, "evento_") || !hex32Valido(base.EventoRef[7:]) || base.AuditoriaRef != prefijo+base.EventoRef[7:] {
		return evento, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(base.AnteriorSHA256) || !huellaCadenaValida(base.HuellaSHA256) || !huellaCadenaValida(base.EventoMaterialSHA256) ||
		base.OperadorLogin == "" || len(base.OperadorLogin) > 63 || !utf8.ValidString(base.OperadorLogin) || strings.ContainsRune(base.OperadorLogin, 0) ||
		base.ModuloID != "administracion" || base.Proceso != "postgresql" || base.Canal != "operacion_tecnica_privada" ||
		base.FinalidadRef != "identidad_interna_sintetica" || len(base.CorrelacionRef) != 44 ||
		!strings.HasPrefix(base.CorrelacionRef, "correlacion_") || !hex32Valido(base.CorrelacionRef[12:]) {
		return evento, "registro_invalido", "coordenadas"
	}
	instante, err := time.Parse(time.RFC3339Nano, base.RegistradaEn)
	if err != nil {
		return evento, motivoErrorInstanteAuditoria(err), "registrada_en"
	}
	if instante.Year() < 1 || instante.Year() > 9999 || instante.UTC().Format("2006-01-02T15:04:05.000000Z") != base.RegistradaEn {
		return evento, "instante_invalido", "registrada_en"
	}
	for _, campo := range material {
		if campo == "" || len(campo) > 200 || !utf8.ValidString(campo) {
			return evento, "registro_invalido", "material"
		}
	}
	suma := sha256.Sum256(huellaEncuadradaIntento(append([]string{"vec.auditoria." + dominio}, material...)...))
	if hex.EncodeToString(suma[:]) != base.EventoMaterialSHA256 {
		return evento, "material_distinto", "evento_material_sha256"
	}
	suma = sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon."+dominio, strconv.FormatUint(secuencia, 10),
		base.AnteriorSHA256, base.AuditoriaRef, base.EventoMaterialSHA256, base.RegistradaEn))
	if hex.EncodeToString(suma[:]) != base.HuellaSHA256 {
		return evento, "huella_distinta", "huella_sha256"
	}
	evento = RegistroEventoAdminV3{AuditoriaRef: base.AuditoriaRef, Secuencia: secuencia, AnteriorSHA256: base.AnteriorSHA256,
		HuellaSHA256: base.HuellaSHA256, EventoRef: base.EventoRef}
	return evento, "", ""
}

func motivoIntentoIdentidadInternaValido(accion, resultado, motivo string) bool {
	switch accion + "/" + resultado + "/" + motivo {
	case "provisionar_identidad_interna_sintetica_v1/permitido/identidad_interna_registrada",
		"provisionar_identidad_interna_sintetica_v1/permitido/identidad_interna_replay",
		"recuperar_identidad_interna_sintetica_v1/permitido/identidad_interna_recuperada",
		"provisionar_identidad_interna_sintetica_v1/denegado/identidad_interna_denegada",
		"recuperar_identidad_interna_sintetica_v1/denegado/identidad_interna_denegada",
		"provisionar_identidad_interna_sintetica_v1/error/identidad_interna_error",
		"recuperar_identidad_interna_sintetica_v1/error/identidad_interna_error":
		return true
	}
	return false
}
