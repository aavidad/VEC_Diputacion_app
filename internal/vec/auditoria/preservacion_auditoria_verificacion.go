package auditoria

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const EsquemaVerificacionPreservacionAuditoria = "vec.auditoria.verificacion.preservacion-provisional.v1"
const TipoOperacionPreservacionAuditoria = "operacion_tecnica_preservacion_auditoria"

type RegistroPreservacionAuditoriaV1 struct {
	RegistroOperacionMantenimientoV1
	DetalleCanonicoBase64 string `json:"detalle_canonico_base64"`
}
type InformePreservacionAuditoriaV1 struct {
	InformeVerificacionPeriodica
	MaterialPreservacionRecalculado bool `json:"material_preservacion_recalculado"`
}

func VerificarCadenaPreservacionAuditoriaV1(d DocumentoVerificacionMixta, c CoberturaCadena, max uint64) InformePreservacionAuditoriaV1 {
	r := InformePreservacionAuditoriaV1{InformeVerificacionPeriodica: InformeVerificacionPeriodica{InformeVerificacionMantenimientoFijo: verificarCadenaMantenimientoFijo(d, c, max, EsquemaVerificacionPreservacionAuditoria)}}
	if r.Estado == "verificada" {
		for _, x := range d.Registros {
			r.MaterialPeriodicaRecalculado = r.MaterialPeriodicaRecalculado || x.Periodica != nil
			r.MaterialPreservacionRecalculado = r.MaterialPreservacionRecalculado || x.Preservacion != nil
		}
	}
	return r
}
func cotejarPreservacionAuditoriaV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var e RegistroEventoAdminV3
	if r.TipoRegistro != TipoOperacionPreservacionAuditoria || r.Preservacion == nil || r.Consumo != nil || r.ConsumoOrigen != nil ||
		r.ConsumoFecha != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil || r.FuentesIniciales != nil ||
		r.IntentoFuentesIniciales != nil || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil ||
		r.IntentoBootstrapCentral != nil || r.Periodica != nil || r.MantenimientoFijo != nil || r.IntentoMantenimientoFijo != nil {
		return e, "tipo_invalido", "tipo_registro"
	}
	p := *r.Preservacion
	b := p.RegistroOperacionMantenimientoV1
	if b.Secuencia != secuencia || secuencia == 0 || secuencia > maxSecuenciaVerificacion {
		return e, "secuencia_distinta", "secuencia"
	}
	if len(b.EventoRef) != 39 || !strings.HasPrefix(b.EventoRef, "evento_") || !hex32Valido(b.EventoRef[7:]) ||
		b.AuditoriaRef != "aud_v3_pre_"+b.EventoRef[7:] {
		return e, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(b.AnteriorSHA256) || !huellaCadenaValida(b.HuellaSHA256) || !huellaCadenaValida(b.EventoMaterialSHA256) ||
		b.OperadorLogin == "" || len(b.OperadorLogin) > 63 || !utf8.ValidString(b.OperadorLogin) || strings.ContainsRune(b.OperadorLogin, 0) ||
		b.ModuloID != "auditoria" || b.RecursoRef != "auditoria_periodica:comun_interna" || b.FinalidadRef != "preservacion_provisional_auditoria" ||
		b.Proceso != "postgresql" || b.Canal != "operacion_tecnica_privada" ||
		len(b.CorrelacionRef) != 44 || !strings.HasPrefix(b.CorrelacionRef, "correlacion_") || !hex32Valido(b.CorrelacionRef[12:]) {
		return e, "registro_invalido", "coordenadas"
	}
	instante, err := time.Parse(time.RFC3339Nano, b.RegistradaEn)
	if err != nil {
		return e, motivoErrorInstanteAuditoria(err), "registrada_en"
	}
	if instante.Year() < 1 || instante.Year() > 9999 || instante.UTC().Format("2006-01-02T15:04:05.000000Z") != b.RegistradaEn {
		return e, "instante_invalido", "registrada_en"
	}
	if len(p.DetalleCanonicoBase64) == 0 || len(p.DetalleCanonicoBase64) > 16*1024 {
		return e, "registro_invalido", "detalle_canonico_base64"
	}
	// encode(...,'base64') de PostgreSQL introduce LF. CR/LF pertenecen al
	// transporte; el material auditado es exclusivamente el detalle decodificado.
	normalizado := strings.ReplaceAll(strings.ReplaceAll(p.DetalleCanonicoBase64, "\r", ""), "\n", "")
	detalle, err := base64.StdEncoding.Strict().DecodeString(normalizado)
	if err != nil {
		var corrupta base64.CorruptInputError
		if errors.As(err, &corrupta) {
			return e, "detalle_base64_invalido", "detalle_canonico_base64"
		}
		return e, "detalle_base64_no_disponible", "detalle_canonico_base64"
	}
	if len(detalle) > 8192 || base64.StdEncoding.EncodeToString(detalle) != normalizado || !detallePreservacionValido(detalle, b) {
		return e, "registro_invalido", "detalle_canonico_base64"
	}
	material := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.preservacion.material.v1", TipoOperacionPreservacionAuditoria,
		b.EventoRef, b.OperadorLogin, b.Accion, b.ModuloID, b.RecursoRef, b.FinalidadRef, b.Resultado, b.MotivoRef,
		b.Proceso, b.Canal, b.CorrelacionRef, string(detalle)))
	if hex.EncodeToString(material[:]) != b.EventoMaterialSHA256 {
		return e, "material_distinto", "evento_material_sha256"
	}
	huella := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.preservacion.eslabon.v1", strconv.FormatUint(secuencia, 10),
		b.AnteriorSHA256, b.AuditoriaRef, b.EventoMaterialSHA256, b.RegistradaEn))
	if hex.EncodeToString(huella[:]) != b.HuellaSHA256 {
		return e, "huella_distinta", "huella_sha256"
	}
	e = RegistroEventoAdminV3{AuditoriaRef: b.AuditoriaRef, Secuencia: secuencia, AnteriorSHA256: b.AnteriorSHA256,
		HuellaSHA256: b.HuellaSHA256, EventoRef: b.EventoRef}
	return e, "", ""
}

func detallePreservacionValido(raw []byte, b RegistroOperacionMantenimientoV1) bool {
	perfil := "vec_auditoria_preservacion_consultor"
	if b.Accion == "configurar_preservacion_auditoria_v1" {
		perfil = "vec_auditoria_preservacion_configurador"
	} else if b.Accion != "consultar_preservacion_auditoria_v1" {
		return false
	}
	var claves []string
	if b.Resultado == "denegado" && b.MotivoRef == "operacion_denegada" || b.Resultado == "error" && b.MotivoRef == "operacion_error" {
		claves = []string{"perfil_tecnico_ref"}
	} else if b.Resultado == "permitido" {
		switch {
		case b.Accion == "configurar_preservacion_auditoria_v1" && (b.MotivoRef == "preservacion_publicada" || b.MotivoRef == "preservacion_replay"):
			claves = []string{"publicacion_ref", "version", "preimagen_sha256", "decision_tecnica_ref", "estado", "medida", "configuracion_sha256", "perfil_tecnico_ref"}
		case b.Accion == "consultar_preservacion_auditoria_v1" && b.MotivoRef == "preservacion_consultada":
			claves = []string{"publicacion_ref", "version", "configuracion_sha256", "perfil_tecnico_ref"}
		case b.Accion == "consultar_preservacion_auditoria_v1" && b.MotivoRef == "preservacion_ausente":
			claves = []string{"version_solicitada", "perfil_tecnico_ref"}
		}
	}
	if len(claves) == 0 {
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if dec.Decode(&m) != nil || dec.Decode(new(any)) != io.EOF || len(m) != len(claves) {
		return false
	}
	for _, k := range claves {
		v, ok := m[k]
		if !ok {
			return false
		}
		if k == "version" || k == "version_solicitada" {
			n, ok := v.(json.Number)
			if !ok {
				return false
			}
			s := n.String()
			if len(s) == 0 || len(s) > 9 || strings.ContainsFunc(s, func(r rune) bool { return r < '0' || r > '9' }) || s[0] == '0' && (len(s) != 1 || k == "version") {
				return false
			}
			continue
		}
		s, ok := v.(string)
		if !ok {
			return false
		}
		switch k {
		case "perfil_tecnico_ref":
			if s != perfil {
				return false
			}
		case "publicacion_ref":
			if len(s) != 45 || !strings.HasPrefix(s, "preservacion_") || !hex32Valido(s[13:]) {
				return false
			}
		case "decision_tecnica_ref":
			if len(s) != 49 || !strings.HasPrefix(s, "decision_tecnica_") || !hex32Valido(s[17:]) {
				return false
			}
		case "estado":
			if s != "provisional" {
				return false
			}
		case "medida":
			if s != "conservar_todo_sin_expurgo" {
				return false
			}
		default:
			if !huellaCadenaValida(s) {
				return false
			}
		}
	}
	return bytes.Equal(raw, canonicoPreservacionPlano(m, claves))
}
func canonicoPreservacionPlano(m map[string]any, claves []string) []byte {
	sort.Slice(claves, func(i, j int) bool {
		if len(claves[i]) != len(claves[j]) {
			return len(claves[i]) < len(claves[j])
		}
		return claves[i] < claves[j]
	})
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range claves {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(strconv.Quote(k))
		b.WriteString(": ")
		switch v := m[k].(type) {
		case string:
			b.WriteString(strconv.Quote(v))
		case json.Number:
			b.WriteString(v.String())
		}
	}
	b.WriteByte('}')
	return b.Bytes()
}
