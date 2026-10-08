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

// Este esquema añade AD219 a la cadena común sin reinterpretar los esquemas
// anteriores ni acreditar por sí mismo la aprobación externa del catálogo.
const EsquemaVerificacionCatalogoAcciones = "vec.auditoria.verificacion.catalogo-acciones-admin.v1"

var (
	operacionCatalogoAcciones  = regexp.MustCompile(`\Acaa_[A-Za-z0-9_-]{22,123}\z`)
	referenciaCatalogoAcciones = regexp.MustCompile(`\A[A-Za-z0-9][A-Za-z0-9._:-]{0,127}\z`)
)

// Los campos siguen el material de AD219. El catálogo completo permanece en
// AUT58; la cadena común conserva referencias, cantidades y huellas.
type RegistroCatalogoAccionesV1 struct {
	RegistroOperacionMantenimientoV1
	OperacionRef     string `json:"operacion_ref"`
	PlanSHA256       string `json:"plan_sha256"`
	CatalogoRef      string `json:"catalogo_ref"`
	CatalogoVersion  string `json:"catalogo_version"`
	CatalogoSHA256   string `json:"catalogo_sha256"`
	PaqueteRef       string `json:"paquete_ref"`
	PaqueteVersion   string `json:"paquete_version"`
	PaqueteSHA256    string `json:"paquete_sha256"`
	CensoSHA256      string `json:"censo_sha256"`
	EntradasNumero   string `json:"entradas_numero"`
	PerfilesNumero   string `json:"perfiles_numero"`
	AprobacionRef    string `json:"aprobacion_ref"`
	AprobacionSHA256 string `json:"aprobacion_sha256"`
}

type RegistroIntentoCatalogoAccionesV1 struct {
	RegistroOperacionMantenimientoV1
	SolicitudSHA256 string `json:"solicitud_sha256"`
}

func VerificarCadenaCatalogoAccionesV1(d DocumentoVerificacionMixta, c CoberturaCadena, max uint64) InformeVerificacion {
	return verificarCadenaMixta(d, c, max, EsquemaVerificacionCatalogoAcciones)
}

func esquemaAdmiteFamiliasPreviasOCatalogo(esquema string) bool {
	return esquemaAdmiteFamiliasAdminOIdentidad(esquema) || esquema == EsquemaVerificacionCatalogoAcciones
}

func numeroCatalogoAccionesValido(valor string, max uint64, maxDigitos int) bool {
	if len(valor) == 0 || len(valor) > maxDigitos || valor[0] < '1' || valor[0] > '9' {
		return false
	}
	n, err := strconv.ParseUint(valor, 10, 64)
	return err == nil && n > 0 && n <= max
}

func motivoIntentoCatalogoAccionesValido(resultado, motivo string) bool {
	switch resultado + "/" + motivo {
	case "permitido/catalogo_acciones_registrado", "permitido/catalogo_acciones_replay",
		"denegado/catalogo_acciones_denegado", "error/catalogo_acciones_error":
		return true
	}
	return false
}

func cotejarCatalogoAccionesV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var evento RegistroEventoAdminV3
	if otrasFamiliasPresentesPerfilesAsignables(r) || r.PerfilesAsignables != nil || r.IntentoPerfilesAsignables != nil ||
		r.ProvisionIdentidadInterna != nil || r.IntentoIdentidadInterna != nil ||
		(r.CatalogoAcciones != nil && r.IntentoCatalogoAcciones != nil) {
		return evento, "tipo_invalido", "tipo_registro"
	}
	var base RegistroOperacionMantenimientoV1
	var material []string
	var dominio, prefijo string
	switch r.TipoRegistro {
	case "catalogo_acciones_admin":
		if r.CatalogoAcciones == nil {
			return evento, "tipo_invalido", "tipo_registro"
		}
		m := r.CatalogoAcciones
		base = m.RegistroOperacionMantenimientoV1
		dominio, prefijo = "catalogo-acciones-admin.v1", "aud_v3_caa_"
		if !operacionCatalogoAcciones.MatchString(m.OperacionRef) || !huellaCadenaValida(m.PlanSHA256) ||
			!referenciaCatalogoAcciones.MatchString(m.CatalogoRef) || !numeroCatalogoAccionesValido(m.CatalogoVersion, 2147483647, 10) ||
			!huellaCadenaValida(m.CatalogoSHA256) || !referenciaCatalogoAcciones.MatchString(m.PaqueteRef) ||
			!numeroCatalogoAccionesValido(m.PaqueteVersion, 2147483647, 10) || !huellaCadenaValida(m.PaqueteSHA256) ||
			!huellaCadenaValida(m.CensoSHA256) || !numeroCatalogoAccionesValido(m.EntradasNumero, 512, 3) ||
			!numeroCatalogoAccionesValido(m.PerfilesNumero, 512, 3) || !referenciaCatalogoAcciones.MatchString(m.AprobacionRef) ||
			!huellaCadenaValida(m.AprobacionSHA256) || base.RecursoRef != "catalogo_acciones_admin:"+m.OperacionRef ||
			base.Resultado != "permitido" || base.MotivoRef != "catalogo_acciones_registrado" {
			return evento, "registro_invalido", "catalogo_acciones"
		}
		material = []string{r.TipoRegistro, base.EventoRef, base.OperadorLogin, m.OperacionRef, m.PlanSHA256,
			m.CatalogoRef, m.CatalogoVersion, m.CatalogoSHA256, m.PaqueteRef, m.PaqueteVersion, m.PaqueteSHA256,
			m.CensoSHA256, m.EntradasNumero, m.PerfilesNumero, m.AprobacionRef, m.AprobacionSHA256,
			base.Proceso, base.Canal, base.FinalidadRef, base.CorrelacionRef}
	case "intento_catalogo_acciones_admin":
		if r.IntentoCatalogoAcciones == nil {
			return evento, "tipo_invalido", "tipo_registro"
		}
		m := r.IntentoCatalogoAcciones
		base = m.RegistroOperacionMantenimientoV1
		dominio, prefijo = "intento-catalogo-acciones-admin.v1", "aud_v3_caai_"
		const recurso = "solicitud_catalogo_acciones_admin:"
		if !huellaCadenaValida(m.SolicitudSHA256) || len(base.RecursoRef) != len(recurso)+32 ||
			!strings.HasPrefix(base.RecursoRef, recurso) || !hex32Valido(strings.TrimPrefix(base.RecursoRef, recurso)) ||
			!motivoIntentoCatalogoAccionesValido(base.Resultado, base.MotivoRef) {
			return evento, "registro_invalido", "intento_catalogo_acciones"
		}
		material = []string{r.TipoRegistro, base.EventoRef, base.OperadorLogin, m.SolicitudSHA256,
			base.Accion, base.RecursoRef, base.Resultado, base.MotivoRef, base.Proceso,
			base.Canal, base.FinalidadRef, base.CorrelacionRef}
	default:
		return evento, "tipo_invalido", "tipo_registro"
	}
	if secuencia == 0 || secuencia > 1<<53-1 || base.Secuencia != secuencia {
		return evento, "secuencia_distinta", "secuencia"
	}
	if len(base.EventoRef) != 39 || !strings.HasPrefix(base.EventoRef, "evento_") || !hex32Valido(base.EventoRef[7:]) ||
		base.AuditoriaRef != prefijo+base.EventoRef[7:] {
		return evento, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(base.AnteriorSHA256) || !huellaCadenaValida(base.HuellaSHA256) ||
		!huellaCadenaValida(base.EventoMaterialSHA256) || base.OperadorLogin == "" || len(base.OperadorLogin) > 63 ||
		!utf8.ValidString(base.OperadorLogin) || strings.ContainsRune(base.OperadorLogin, 0) ||
		base.Accion != "registrar_catalogo_acciones_admin_v1" || base.ModuloID != "administracion" ||
		base.Proceso != "postgresql" || base.Canal != "operacion_tecnica_privada" ||
		base.FinalidadRef != "catalogo_acciones_admin" || len(base.CorrelacionRef) != 44 ||
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
	return RegistroEventoAdminV3{AuditoriaRef: base.AuditoriaRef, Secuencia: secuencia,
		AnteriorSHA256: base.AnteriorSHA256, HuellaSHA256: base.HuellaSHA256, EventoRef: base.EventoRef}, "", ""
}
