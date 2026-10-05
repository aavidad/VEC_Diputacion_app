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

const EsquemaVerificacionMantenimientoFijo = "vec.auditoria.verificacion.mantenimiento-perfil-fijo-admin.v1"

//go:embed motivos_intento_mantenimiento.json
var motivosIntentoMantenimientoJSON string

type RegistroOperacionMantenimientoV1 struct {
	AuditoriaRef         string `json:"auditoria_ref"`
	Secuencia            uint64 `json:"secuencia"`
	AnteriorSHA256       string `json:"anterior_sha256"`
	HuellaSHA256         string `json:"huella_sha256"`
	RegistradaEn         string `json:"registrada_en"`
	EventoRef            string `json:"evento_ref"`
	EventoMaterialSHA256 string `json:"evento_material_sha256"`
	OperadorLogin        string `json:"operador_login"`
	Accion               string `json:"accion"`
	ModuloID             string `json:"modulo_id"`
	RecursoRef           string `json:"recurso_ref"`
	Resultado            string `json:"resultado"`
	MotivoRef            string `json:"motivo_ref"`
	Proceso              string `json:"proceso"`
	Canal                string `json:"canal"`
	FinalidadRef         string `json:"finalidad_ref"`
	CorrelacionRef       string `json:"correlacion_ref"`
}
type RegistroMantenimientoFijoV1 struct {
	RegistroOperacionMantenimientoV1
	PlanSHA256               string `json:"plan_sha256"`
	PreimagenSHA256          string `json:"preimagen_sha256"`
	CatalogoSHA256           string `json:"catalogo_sha256"`
	RolOrigenRef             string `json:"rol_origen_ref"`
	RolOrigenSHA256          string `json:"rol_origen_sha256"`
	RolDestinoRef            string `json:"rol_destino_ref"`
	RolDestinoSHA256         string `json:"rol_destino_sha256"`
	Asignacion1OrigenRef     string `json:"asignacion_1_origen_ref"`
	Asignacion1OrigenSHA256  string `json:"asignacion_1_origen_sha256"`
	Asignacion1DestinoRef    string `json:"asignacion_1_destino_ref"`
	Asignacion1DestinoSHA256 string `json:"asignacion_1_destino_sha256"`
	Asignacion2OrigenRef     string `json:"asignacion_2_origen_ref"`
	Asignacion2OrigenSHA256  string `json:"asignacion_2_origen_sha256"`
	Asignacion2DestinoRef    string `json:"asignacion_2_destino_ref"`
	Asignacion2DestinoSHA256 string `json:"asignacion_2_destino_sha256"`
}
type RegistroIntentoMantenimientoFijoV1 struct {
	RegistroOperacionMantenimientoV1
	SolicitudSHA256 string `json:"solicitud_sha256"`
}
type InformeVerificacionMantenimientoFijo struct {
	InformeVerificacionBootstrapCentral
	MaterialMantenimientoRecalculado         bool `json:"material_mantenimiento_recalculado"`
	MaterialIntentosMantenimientoRecalculado bool `json:"material_intentos_mantenimiento_recalculado"`
}

func VerificarCadenaMantenimientoFijoV1(d DocumentoVerificacionMixta, c CoberturaCadena, max uint64) InformeVerificacionMantenimientoFijo {
	return verificarCadenaMantenimientoFijo(d, c, max, EsquemaVerificacionMantenimientoFijo)
}
func verificarCadenaMantenimientoFijo(d DocumentoVerificacionMixta, c CoberturaCadena, max uint64, esquema string) InformeVerificacionMantenimientoFijo {
	r := InformeVerificacionMantenimientoFijo{InformeVerificacionBootstrapCentral: InformeVerificacionBootstrapCentral{InformeVerificacionUnidadInicial: InformeVerificacionUnidadInicial{InformeVerificacionFuentesIniciales: InformeVerificacionFuentesIniciales{InformeVerificacion: verificarCadenaMixta(d, c, max, esquema)}}}}
	if r.Estado == "verificada" {
		for _, x := range d.Registros {
			r.MaterialEventoRecalculado = r.MaterialEventoRecalculado || x.Preperfil != nil || x.Bootstrap != nil
			r.MaterialFuentesRecalculado = r.MaterialFuentesRecalculado || x.FuentesIniciales != nil
			r.MaterialIntentosFuentesRecalculado = r.MaterialIntentosFuentesRecalculado || x.IntentoFuentesIniciales != nil
			r.MaterialUnidadRecalculado = r.MaterialUnidadRecalculado || x.UnidadInicial != nil
			r.MaterialIntentosUnidadRecalculado = r.MaterialIntentosUnidadRecalculado || x.IntentoUnidadInicial != nil
			r.MaterialIntentosBootstrapRecalculado = r.MaterialIntentosBootstrapRecalculado || x.IntentoBootstrapCentral != nil
			r.MaterialMantenimientoRecalculado = r.MaterialMantenimientoRecalculado || x.MantenimientoFijo != nil
			r.MaterialIntentosMantenimientoRecalculado = r.MaterialIntentosMantenimientoRecalculado || x.IntentoMantenimientoFijo != nil
		}
	}
	return r
}
func cotejarMantenimientoFijoV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var base RegistroOperacionMantenimientoV1
	var e RegistroEventoAdminV3
	var campos []string
	var dominio, prefijo string
	if r.Consumo != nil || r.ConsumoOrigen != nil || r.ConsumoFecha != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil || r.FuentesIniciales != nil || r.IntentoFuentesIniciales != nil || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil || r.IntentoBootstrapCentral != nil {
		return e, "tipo_invalido", "tipo_registro"
	}
	if r.TipoRegistro == "mantenimiento_perfil_fijo_admin" {
		if r.MantenimientoFijo == nil || r.IntentoMantenimientoFijo != nil {
			return e, "tipo_invalido", "tipo_registro"
		}
		m := *r.MantenimientoFijo
		base = m.RegistroOperacionMantenimientoV1
		dominio = "mantenimiento-perfil-fijo-admin.v1"
		prefijo = "aud_v3_mf_"
		if base.Resultado != "permitido" || base.MotivoRef != "mantenimiento_registrado" || base.RecursoRef != "mantenimiento_admin:"+m.PlanSHA256[:min(32, len(m.PlanSHA256))] {
			return e, "registro_invalido", "mantenimiento"
		}
		if !huellaCadenaValida(m.PlanSHA256) {
			return e, "registro_invalido", "plan_sha256"
		}
		if !huellaCadenaValida(m.PreimagenSHA256) {
			return e, "registro_invalido", "preimagen_sha256"
		}
		if !huellaCadenaValida(m.CatalogoSHA256) {
			return e, "registro_invalido", "catalogo_sha256"
		}
		if !recursoAD171.MatchString(m.RolOrigenRef) {
			return e, "registro_invalido", "rol_origen_ref"
		}
		if !huellaCadenaValida(m.RolOrigenSHA256) {
			return e, "registro_invalido", "rol_origen_sha256"
		}
		if !recursoAD171.MatchString(m.RolDestinoRef) {
			return e, "registro_invalido", "rol_destino_ref"
		}
		if !huellaCadenaValida(m.RolDestinoSHA256) {
			return e, "registro_invalido", "rol_destino_sha256"
		}
		if !recursoAD171.MatchString(m.Asignacion1OrigenRef) {
			return e, "registro_invalido", "asignacion_1_origen_ref"
		}
		if !huellaCadenaValida(m.Asignacion1OrigenSHA256) {
			return e, "registro_invalido", "asignacion_1_origen_sha256"
		}
		if !recursoAD171.MatchString(m.Asignacion1DestinoRef) {
			return e, "registro_invalido", "asignacion_1_destino_ref"
		}
		if !huellaCadenaValida(m.Asignacion1DestinoSHA256) {
			return e, "registro_invalido", "asignacion_1_destino_sha256"
		}
		if !recursoAD171.MatchString(m.Asignacion2OrigenRef) {
			return e, "registro_invalido", "asignacion_2_origen_ref"
		}
		if !huellaCadenaValida(m.Asignacion2OrigenSHA256) {
			return e, "registro_invalido", "asignacion_2_origen_sha256"
		}
		if !recursoAD171.MatchString(m.Asignacion2DestinoRef) {
			return e, "registro_invalido", "asignacion_2_destino_ref"
		}
		if !huellaCadenaValida(m.Asignacion2DestinoSHA256) {
			return e, "registro_invalido", "asignacion_2_destino_sha256"
		}
		campos = []string{r.TipoRegistro, base.EventoRef, base.OperadorLogin, m.PlanSHA256, m.PreimagenSHA256, m.CatalogoSHA256, m.RolOrigenRef, m.RolOrigenSHA256, m.RolDestinoRef, m.RolDestinoSHA256, m.Asignacion1OrigenRef, m.Asignacion1OrigenSHA256, m.Asignacion1DestinoRef, m.Asignacion1DestinoSHA256, m.Asignacion2OrigenRef, m.Asignacion2OrigenSHA256, m.Asignacion2DestinoRef, m.Asignacion2DestinoSHA256, base.Proceso, base.Canal, base.FinalidadRef, base.CorrelacionRef}
	} else {
		if r.IntentoMantenimientoFijo == nil || r.MantenimientoFijo != nil {
			return e, "tipo_invalido", "tipo_registro"
		}
		m := *r.IntentoMantenimientoFijo
		base = m.RegistroOperacionMantenimientoV1
		dominio = "intento-mantenimiento-perfil-fijo-admin.v1"
		prefijo = "aud_v3_mfi_"
		if !huellaCadenaValida(m.SolicitudSHA256) || len(base.RecursoRef) != len("solicitud_mantenimiento:")+32 || !strings.HasPrefix(base.RecursoRef, "solicitud_mantenimiento:") || !hex32Valido(strings.TrimPrefix(base.RecursoRef, "solicitud_mantenimiento:")) || !motivoIntentoMantenimientoValido(base.Resultado, base.MotivoRef) {
			return e, "registro_invalido", "intento_mantenimiento"
		}
		campos = []string{r.TipoRegistro, base.EventoRef, base.OperadorLogin, m.SolicitudSHA256, base.Accion, base.RecursoRef, base.Resultado, base.MotivoRef, base.Proceso, base.Canal, base.FinalidadRef, base.CorrelacionRef}
	}
	if base.Secuencia != secuencia {
		return e, "secuencia_distinta", "secuencia"
	}
	if len(base.EventoRef) != 39 || !strings.HasPrefix(base.EventoRef, "evento_") || !hex32Valido(base.EventoRef[7:]) || base.AuditoriaRef != prefijo+base.EventoRef[7:] {
		return e, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(base.AnteriorSHA256) || !huellaCadenaValida(base.HuellaSHA256) || !huellaCadenaValida(base.EventoMaterialSHA256) || base.OperadorLogin == "" || len(base.OperadorLogin) > 63 || !utf8.ValidString(base.OperadorLogin) || strings.ContainsRune(base.OperadorLogin, 0) ||
		base.Accion != "mantener_version_perfil_fijo_admin_v1" || base.ModuloID != "administracion" || base.Proceso != "postgresql" || base.Canal != "operacion_tecnica_privada" || base.FinalidadRef != "mantenimiento_perfil_fijo_admin" || len(base.CorrelacionRef) != 44 || !strings.HasPrefix(base.CorrelacionRef, "correlacion_") || !hex32Valido(base.CorrelacionRef[12:]) {
		return e, "registro_invalido", "coordenadas"
	}
	t, err := time.Parse(time.RFC3339Nano, base.RegistradaEn)
	if err != nil {
		return e, motivoErrorInstanteAuditoria(err), "registrada_en"
	}
	if t.Year() < 1 || t.Year() > 9999 || t.UTC().Format("2006-01-02T15:04:05.000000Z") != base.RegistradaEn {
		return e, "instante_invalido", "registrada_en"
	}
	sum := sha256.Sum256(huellaEncuadradaIntento(append([]string{"vec.auditoria." + dominio}, campos...)...))
	if hex.EncodeToString(sum[:]) != base.EventoMaterialSHA256 {
		return e, "material_distinto", "evento_material_sha256"
	}
	sum = sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon."+dominio, strconv.FormatUint(secuencia, 10), base.AnteriorSHA256, base.AuditoriaRef, base.EventoMaterialSHA256, base.RegistradaEn))
	if hex.EncodeToString(sum[:]) != base.HuellaSHA256 {
		return e, "huella_distinta", "huella_sha256"
	}
	e = RegistroEventoAdminV3{AuditoriaRef: base.AuditoriaRef, Secuencia: secuencia, AnteriorSHA256: base.AnteriorSHA256, HuellaSHA256: base.HuellaSHA256, EventoRef: base.EventoRef}
	return e, "", ""
}
func motivoIntentoMantenimientoValido(resultado, motivo string) bool {
	var ms []struct {
		Resultado string `json:"resultado"`
		MotivoRef string `json:"motivo_ref"`
	}
	if json.Unmarshal([]byte(motivosIntentoMantenimientoJSON), &ms) != nil {
		return false
	}
	for _, m := range ms {
		if m.Resultado == resultado && m.MotivoRef == motivo {
			return true
		}
	}
	return false
}
