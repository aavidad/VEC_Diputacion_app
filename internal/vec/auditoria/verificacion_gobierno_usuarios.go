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

const EsquemaVerificacionGobiernoUsuarios = "vec.auditoria.verificacion.gobierno-usuarios-admin.v1"

//go:embed motivos_intento_gobierno_usuarios.json
var motivosIntentoGobiernoUsuariosJSON string

type RegistroGobiernoUsuariosV1 struct {
	RegistroOperacionMantenimientoV1
	PlanSHA256              string `json:"plan_sha256"`
	PreimagenSHA256         string `json:"preimagen_sha256"`
	ConfiguracionOrigenRef  string `json:"configuracion_origen_ref"`
	ConfiguracionDestinoRef string `json:"configuracion_destino_ref"`
	ClavesSHA256            string `json:"claves_sha256"`
}
type RegistroIntentoGobiernoUsuariosV1 struct {
	RegistroOperacionMantenimientoV1
	SolicitudSHA256 string `json:"solicitud_sha256"`
}

func VerificarCadenaGobiernoUsuariosV1(d DocumentoVerificacionMixta, c CoberturaCadena, max uint64) InformeVerificacion {
	return verificarCadenaMixta(d, c, max, EsquemaVerificacionGobiernoUsuarios)
}
func cotejarGobiernoUsuariosV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var base RegistroOperacionMantenimientoV1
	var e RegistroEventoAdminV3
	var campos []string
	var dominio, prefijo string
	if r.Consumo != nil || r.ConsumoOrigen != nil || r.ConsumoFecha != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil || r.FuentesIniciales != nil || r.IntentoFuentesIniciales != nil || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil || r.IntentoBootstrapCentral != nil || r.MantenimientoFijo != nil || r.IntentoMantenimientoFijo != nil || r.Periodica != nil || r.Preservacion != nil {
		return e, "tipo_invalido", "tipo_registro"
	}
	if r.TipoRegistro == "gobierno_usuarios_admin" {
		if r.GobiernoUsuarios == nil || r.IntentoGobiernoUsuarios != nil {
			return e, "tipo_invalido", "tipo_registro"
		}
		m := *r.GobiernoUsuarios
		base = m.RegistroOperacionMantenimientoV1
		dominio = "gobierno-usuarios-admin.v1"
		prefijo = "aud_v3_gu_"
		if base.Resultado != "permitido" || base.MotivoRef != "gobierno_usuarios_registrado" || base.RecursoRef != "gobierno_usuarios:"+m.PlanSHA256[:min(32, len(m.PlanSHA256))] {
			return e, "registro_invalido", "gobierno_usuarios"
		}
		if !huellaCadenaValida(m.PlanSHA256) {
			return e, "registro_invalido", "plan_sha256"
		}
		if !huellaCadenaValida(m.PreimagenSHA256) {
			return e, "registro_invalido", "preimagen_sha256"
		}
		if !huellaCadenaValida(m.ClavesSHA256) || !recursoAD171.MatchString(m.ConfiguracionOrigenRef) || !recursoAD171.MatchString(m.ConfiguracionDestinoRef) {
			return e, "registro_invalido", "detalle_gobierno"
		}
		campos = []string{r.TipoRegistro, base.EventoRef, base.OperadorLogin, m.PlanSHA256, m.PreimagenSHA256, m.ConfiguracionOrigenRef, m.ConfiguracionDestinoRef, m.ClavesSHA256, base.Proceso, base.Canal, base.FinalidadRef, base.CorrelacionRef}
	} else {
		if r.IntentoGobiernoUsuarios == nil || r.GobiernoUsuarios != nil {
			return e, "tipo_invalido", "tipo_registro"
		}
		m := *r.IntentoGobiernoUsuarios
		base = m.RegistroOperacionMantenimientoV1
		dominio = "intento-gobierno-usuarios-admin.v1"
		prefijo = "aud_v3_gui_"
		if !huellaCadenaValida(m.SolicitudSHA256) || len(base.RecursoRef) != len("solicitud_gobierno_usuarios:")+32 || !strings.HasPrefix(base.RecursoRef, "solicitud_gobierno_usuarios:") || !hex32Valido(strings.TrimPrefix(base.RecursoRef, "solicitud_gobierno_usuarios:")) || !motivoIntentoGobiernoUsuariosValido(base.Resultado, base.MotivoRef) {
			return e, "registro_invalido", "intento_gobierno_usuarios"
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
		base.Accion != "aprovisionar_gobierno_usuarios_admin_v1" || base.ModuloID != "administracion" || base.Proceso != "postgresql" || base.Canal != "operacion_tecnica_privada" || base.FinalidadRef != "gobierno_usuarios_admin" || len(base.CorrelacionRef) != 44 || !strings.HasPrefix(base.CorrelacionRef, "correlacion_") || !hex32Valido(base.CorrelacionRef[12:]) {
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
func motivoIntentoGobiernoUsuariosValido(resultado, motivo string) bool {
	var ms []struct {
		Resultado string `json:"resultado"`
		MotivoRef string `json:"motivo_ref"`
	}
	if json.Unmarshal([]byte(motivosIntentoGobiernoUsuariosJSON), &ms) != nil {
		return false
	}
	for _, m := range ms {
		if m.Resultado == resultado && m.MotivoRef == motivo {
			return true
		}
	}
	return false
}
