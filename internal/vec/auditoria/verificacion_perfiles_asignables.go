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

// EsquemaVerificacionPerfilesAsignables admite todas las familias que ya
// admite el esquema de contexto ADMIN previo a V2 y, además, las dos de AD196:
// registro técnico de perfiles asignables (AUT49) y su intento.
const EsquemaVerificacionPerfilesAsignables = "vec.auditoria.verificacion.perfiles-asignables-admin.v1"

//go:embed motivos_intento_perfiles_asignables.json
var motivosIntentoPerfilesAsignablesJSON string

var (
	operacionPerfilesAsignables = regexp.MustCompile(`\Arpa_[A-Za-z0-9_-]{22,124}\z`)
	// De 1 a 32, sin ceros a la izquierda, igual que AD196.
	numeroPerfilesAsignables = regexp.MustCompile(`\A([1-9]|[12][0-9]|3[0-2])\z`)
)

// RegistroPerfilesAsignablesV1 replica las columnas que AD196 compromete en el
// material del evento. La lista registrada queda en AUT49; aquí sólo su huella
// y la de la aprobación externa del plan.
type RegistroPerfilesAsignablesV1 struct {
	RegistroOperacionMantenimientoV1
	PlanSHA256       string `json:"plan_sha256"`
	OperacionRef     string `json:"operacion_ref"`
	PerfilesSHA256   string `json:"perfiles_sha256"`
	PerfilesNumero   string `json:"perfiles_numero"`
	AprobacionSHA256 string `json:"aprobacion_sha256"`
}
type RegistroIntentoPerfilesAsignablesV1 struct {
	RegistroOperacionMantenimientoV1
	SolicitudSHA256 string `json:"solicitud_sha256"`
}

func VerificarCadenaPerfilesAsignablesV1(d DocumentoVerificacionMixta, c CoberturaCadena, max uint64) InformeVerificacion {
	return verificarCadenaMixta(d, c, max, EsquemaVerificacionPerfilesAsignables)
}

// esquemaAdmiteFamiliasAdmin indica los esquemas que aceptan todas las
// familias técnicas y de frontera ADMIN existentes en la cadena común.
func esquemaAdmiteFamiliasAdmin(esquema string) bool {
	return esquema == EsquemaVerificacionContextoAdminPreV2 || esquema == EsquemaVerificacionPerfilesAsignables
}

func otrasFamiliasPresentesPerfilesAsignables(r RegistroMixtoV2) bool {
	return r.Consumo != nil || r.ConsumoOrigen != nil || r.ConsumoFecha != nil || r.ConsumoTransaccion != nil ||
		r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil || r.ContextoAdminPreV2 != nil ||
		r.FronteraAdminTecnica != nil || r.GobiernoUsuarios != nil || r.IntentoGobiernoUsuarios != nil ||
		r.Preservacion != nil || r.Periodica != nil || r.MantenimientoFijo != nil || r.IntentoMantenimientoFijo != nil ||
		r.IntentoBootstrapCentral != nil || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil ||
		r.FuentesIniciales != nil || r.IntentoFuentesIniciales != nil
}

func cotejarPerfilesAsignablesV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var base RegistroOperacionMantenimientoV1
	var e RegistroEventoAdminV3
	var campos []string
	var dominio, prefijo string
	if otrasFamiliasPresentesPerfilesAsignables(r) {
		return e, "tipo_invalido", "tipo_registro"
	}
	if r.TipoRegistro == "perfiles_asignables_admin" {
		if r.PerfilesAsignables == nil || r.IntentoPerfilesAsignables != nil {
			return e, "tipo_invalido", "tipo_registro"
		}
		m := *r.PerfilesAsignables
		base = m.RegistroOperacionMantenimientoV1
		dominio, prefijo = "perfiles-asignables-admin.v1", "aud_v3_pa_"
		if !huellaCadenaValida(m.PlanSHA256) {
			return e, "registro_invalido", "plan_sha256"
		}
		if base.Resultado != "permitido" || base.MotivoRef != "perfiles_asignables_registrado" || base.RecursoRef != "perfiles_asignables:"+m.PlanSHA256[:32] {
			return e, "registro_invalido", "perfiles_asignables"
		}
		if !operacionPerfilesAsignables.MatchString(m.OperacionRef) || !huellaCadenaValida(m.PerfilesSHA256) || !huellaCadenaValida(m.AprobacionSHA256) ||
			!numeroPerfilesAsignables.MatchString(m.PerfilesNumero) {
			return e, "registro_invalido", "detalle_perfiles_asignables"
		}
		campos = []string{r.TipoRegistro, base.EventoRef, base.OperadorLogin, m.PlanSHA256, m.OperacionRef, m.PerfilesSHA256, m.PerfilesNumero, m.AprobacionSHA256, base.Proceso, base.Canal, base.FinalidadRef, base.CorrelacionRef}
	} else {
		if r.IntentoPerfilesAsignables == nil || r.PerfilesAsignables != nil {
			return e, "tipo_invalido", "tipo_registro"
		}
		m := *r.IntentoPerfilesAsignables
		base = m.RegistroOperacionMantenimientoV1
		dominio, prefijo = "intento-perfiles-asignables-admin.v1", "aud_v3_pai_"
		const recurso = "solicitud_perfiles_asignables:"
		if !huellaCadenaValida(m.SolicitudSHA256) || len(base.RecursoRef) != len(recurso)+32 || !strings.HasPrefix(base.RecursoRef, recurso) ||
			!hex32Valido(strings.TrimPrefix(base.RecursoRef, recurso)) || !motivoIntentoPerfilesAsignablesValido(base.Resultado, base.MotivoRef) {
			return e, "registro_invalido", "intento_perfiles_asignables"
		}
		campos = []string{r.TipoRegistro, base.EventoRef, base.OperadorLogin, m.SolicitudSHA256, base.Accion, base.RecursoRef, base.Resultado, base.MotivoRef, base.Proceso, base.Canal, base.FinalidadRef, base.CorrelacionRef}
	}
	if base.Secuencia != secuencia || secuencia == 0 || secuencia > 1<<53-1 {
		return e, "secuencia_distinta", "secuencia"
	}
	if len(base.EventoRef) != 39 || !strings.HasPrefix(base.EventoRef, "evento_") || !hex32Valido(base.EventoRef[7:]) || base.AuditoriaRef != prefijo+base.EventoRef[7:] {
		return e, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(base.AnteriorSHA256) || !huellaCadenaValida(base.HuellaSHA256) || !huellaCadenaValida(base.EventoMaterialSHA256) ||
		base.OperadorLogin == "" || len(base.OperadorLogin) > 63 || !utf8.ValidString(base.OperadorLogin) || strings.ContainsRune(base.OperadorLogin, 0) ||
		base.Accion != "registrar_perfiles_asignables_admin_v1" || base.ModuloID != "administracion" || base.Proceso != "postgresql" ||
		base.Canal != "operacion_tecnica_privada" || base.FinalidadRef != "perfiles_asignables_admin" ||
		len(base.CorrelacionRef) != 44 || !strings.HasPrefix(base.CorrelacionRef, "correlacion_") || !hex32Valido(base.CorrelacionRef[12:]) {
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
	return RegistroEventoAdminV3{AuditoriaRef: base.AuditoriaRef, Secuencia: secuencia, AnteriorSHA256: base.AnteriorSHA256, HuellaSHA256: base.HuellaSHA256, EventoRef: base.EventoRef}, "", ""
}

func motivoIntentoPerfilesAsignablesValido(resultado, motivo string) bool {
	var ms []struct {
		Resultado string `json:"resultado"`
		MotivoRef string `json:"motivo_ref"`
	}
	if json.Unmarshal([]byte(motivosIntentoPerfilesAsignablesJSON), &ms) != nil {
		return false
	}
	for _, m := range ms {
		if m.Resultado == resultado && m.MotivoRef == motivo {
			return true
		}
	}
	return false
}
