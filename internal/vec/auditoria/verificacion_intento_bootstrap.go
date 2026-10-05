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

const EsquemaVerificacionBootstrapCentral = "vec.auditoria.verificacion.bootstrap-central-admin.v1"

// El checkpoint y las fuentes se cotejan sin autenticar su origen privado.
type InformeVerificacionBootstrapCentral struct {
	InformeVerificacionUnidadInicial
	MaterialIntentosBootstrapRecalculado bool `json:"material_intentos_bootstrap_recalculado"`
}

func VerificarCadenaBootstrapCentralV1(d DocumentoVerificacionMixta, checkpoint CoberturaCadena, maxRegistros uint64) InformeVerificacionBootstrapCentral {
	r := InformeVerificacionBootstrapCentral{InformeVerificacionUnidadInicial: InformeVerificacionUnidadInicial{InformeVerificacionFuentesIniciales: InformeVerificacionFuentesIniciales{InformeVerificacion: verificarCadenaMixta(d, checkpoint, maxRegistros, EsquemaVerificacionBootstrapCentral)}}}
	if r.Estado == "verificada" {
		for _, x := range d.Registros {
			r.MaterialEventoRecalculado = r.MaterialEventoRecalculado || x.Preperfil != nil || x.Bootstrap != nil
			r.MaterialFuentesRecalculado = r.MaterialFuentesRecalculado || x.FuentesIniciales != nil
			r.MaterialIntentosFuentesRecalculado = r.MaterialIntentosFuentesRecalculado || x.IntentoFuentesIniciales != nil
			r.MaterialUnidadRecalculado = r.MaterialUnidadRecalculado || x.UnidadInicial != nil
			r.MaterialIntentosUnidadRecalculado = r.MaterialIntentosUnidadRecalculado || x.IntentoUnidadInicial != nil
			r.MaterialIntentosBootstrapRecalculado = r.MaterialIntentosBootstrapRecalculado || x.IntentoBootstrapCentral != nil
		}
	}
	return r
}

//go:embed motivos_intento_bootstrap.json
var motivosIntentoBootstrapJSON string

// RegistroIntentoBootstrapCentralV1 no contiene campos de fuente confirmada,
// aprobación, preimagen de efecto, persona, perfil ni concesión V3.
type RegistroIntentoBootstrapCentralV1 struct {
	AuditoriaRef         string `json:"auditoria_ref"`
	Secuencia            uint64 `json:"secuencia"`
	AnteriorSHA256       string `json:"anterior_sha256"`
	HuellaSHA256         string `json:"huella_sha256"`
	RegistradaEn         string `json:"registrada_en"`
	EventoRef            string `json:"evento_ref"`
	EventoMaterialSHA256 string `json:"evento_material_sha256"`
	ModuloID             string `json:"modulo_id"`
	OperadorLogin        string `json:"operador_login"`
	SolicitudSHA256      string `json:"solicitud_sha256"`
	Accion               string `json:"accion"`
	RecursoRef           string `json:"recurso_ref"`
	Resultado            string `json:"resultado"`
	MotivoRef            string `json:"motivo_ref"`
	Proceso              string `json:"proceso"`
	Canal                string `json:"canal"`
	FinalidadRef         string `json:"finalidad_ref"`
	CorrelacionRef       string `json:"correlacion_ref"`
}

func motivoIntentoBootstrapValido(resultado, motivo string) bool {
	var motivos []struct {
		Resultado string `json:"resultado"`
		MotivoRef string `json:"motivo_ref"`
	}
	if json.Unmarshal([]byte(motivosIntentoBootstrapJSON), &motivos) != nil {
		return false
	}
	for _, m := range motivos {
		if m.Resultado == resultado && m.MotivoRef == motivo {
			return true
		}
	}
	return false
}

func cotejarIntentoBootstrapCentralV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var e RegistroEventoAdminV3
	if r.IntentoBootstrapCentral == nil || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil || r.Consumo != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil || r.FuentesIniciales != nil || r.IntentoFuentesIniciales != nil {
		return e, "tipo_invalido", "tipo_registro"
	}
	f := *r.IntentoBootstrapCentral
	if f.Secuencia != secuencia {
		return e, "secuencia_distinta", "secuencia"
	}
	if len(f.EventoRef) != 39 || !strings.HasPrefix(f.EventoRef, "evento_") || !hex32Valido(f.EventoRef[7:]) || f.AuditoriaRef != "aud_v3_bi_"+f.EventoRef[7:] {
		return e, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(f.AnteriorSHA256) || !huellaCadenaValida(f.HuellaSHA256) || !huellaCadenaValida(f.EventoMaterialSHA256) || !huellaCadenaValida(f.SolicitudSHA256) ||
		f.OperadorLogin == "" || len(f.OperadorLogin) > 63 || !utf8.ValidString(f.OperadorLogin) || strings.ContainsRune(f.OperadorLogin, 0) ||
		len(f.RecursoRef) != 52 || !strings.HasPrefix(f.RecursoRef, "solicitud_bootstrap:") || !hex32Valido(f.RecursoRef[20:]) ||
		len(f.CorrelacionRef) != 44 || !strings.HasPrefix(f.CorrelacionRef, "correlacion_") || !hex32Valido(f.CorrelacionRef[12:]) ||
		f.ModuloID != "administracion" || f.Accion != "registrar_bootstrap_central_admin_v3" || f.Proceso != "postgresql" ||
		f.Canal != "operacion_tecnica_privada" || f.FinalidadRef != "bootstrap_admin" || !motivoIntentoBootstrapValido(f.Resultado, f.MotivoRef) {
		return e, "registro_invalido", "coordenadas"
	}
	instante, err := time.Parse(time.RFC3339Nano, f.RegistradaEn)
	if err != nil {
		return e, motivoErrorInstanteAuditoria(err), "registrada_en"
	}
	if instante.Year() < 1 || instante.Year() > 9999 || instante.UTC().Format("2006-01-02T15:04:05.000000Z") != f.RegistradaEn {
		return e, "instante_invalido", "registrada_en"
	}
	material := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.intento-bootstrap-central-admin.v1", r.TipoRegistro, f.EventoRef, f.OperadorLogin, f.SolicitudSHA256,
		f.Accion, f.RecursoRef, f.Resultado, f.MotivoRef, f.Proceso, f.Canal, f.FinalidadRef, f.CorrelacionRef))
	if hex.EncodeToString(material[:]) != f.EventoMaterialSHA256 {
		return e, "material_distinto", "evento_material_sha256"
	}
	eslabon := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon.intento-bootstrap-central-admin.v1", strconv.FormatUint(secuencia, 10), f.AnteriorSHA256, f.AuditoriaRef, f.EventoMaterialSHA256, f.RegistradaEn))
	if hex.EncodeToString(eslabon[:]) != f.HuellaSHA256 {
		return e, "huella_distinta", "huella_sha256"
	}
	e = RegistroEventoAdminV3{AuditoriaRef: f.AuditoriaRef, Secuencia: f.Secuencia, AnteriorSHA256: f.AnteriorSHA256, HuellaSHA256: f.HuellaSHA256, EventoRef: f.EventoRef}
	return e, "", ""
}
