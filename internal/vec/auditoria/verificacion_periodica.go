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

const EsquemaVerificacionPeriodica = "vec.auditoria.verificacion.periodica.v1"
const TipoOperacionPeriodica = "operacion_tecnica_auditoria_periodica"

// RegistroOperacionPeriodicaV1 proyecta metadatos técnicos de AD186. El
// detalle conserva exactamente jsonb::text; no contiene registros personales.
type RegistroOperacionPeriodicaV1 struct {
	RegistroOperacionMantenimientoV1
	DetalleCanonicoBase64 string `json:"detalle_canonico_base64"`
}

type InformeVerificacionPeriodica struct {
	InformeVerificacionMantenimientoFijo
	MaterialPeriodicaRecalculado bool `json:"material_periodica_recalculado"`
}

// VerificarCadenaPeriodicaV1 amplía el cotejo sin autenticar por sí solo el
// LOGIN, la captura o el checkpoint, ni fijar una política de conservación.
func VerificarCadenaPeriodicaV1(d DocumentoVerificacionMixta, c CoberturaCadena, max uint64) InformeVerificacionPeriodica {
	r := InformeVerificacionPeriodica{InformeVerificacionMantenimientoFijo: verificarCadenaMantenimientoFijo(d, c, max, EsquemaVerificacionPeriodica)}
	if r.Estado == "verificada" {
		for _, x := range d.Registros {
			r.MaterialPeriodicaRecalculado = r.MaterialPeriodicaRecalculado || x.Periodica != nil
		}
	}
	return r
}

func cotejarOperacionPeriodicaV1(r RegistroMixtoV2, secuencia uint64) (RegistroEventoAdminV3, string, string) {
	var e RegistroEventoAdminV3
	if r.TipoRegistro != TipoOperacionPeriodica || r.Periodica == nil || r.Consumo != nil || r.ConsumoOrigen != nil ||
		r.ConsumoFecha != nil || r.Intento != nil || r.Preperfil != nil || r.Bootstrap != nil || r.FuentesIniciales != nil ||
		r.IntentoFuentesIniciales != nil || r.UnidadInicial != nil || r.IntentoUnidadInicial != nil ||
		r.IntentoBootstrapCentral != nil || r.MantenimientoFijo != nil || r.IntentoMantenimientoFijo != nil {
		return e, "tipo_invalido", "tipo_registro"
	}
	p := *r.Periodica
	b := p.RegistroOperacionMantenimientoV1
	if b.Secuencia != secuencia || secuencia == 0 || secuencia > maxSecuenciaVerificacion {
		return e, "secuencia_distinta", "secuencia"
	}
	if len(b.EventoRef) != 39 || !strings.HasPrefix(b.EventoRef, "evento_") || !hex32Valido(b.EventoRef[7:]) ||
		b.AuditoriaRef != "aud_v3_per_"+b.EventoRef[7:] {
		return e, "referencia_distinta", "auditoria_ref"
	}
	if !huellaCadenaValida(b.AnteriorSHA256) || !huellaCadenaValida(b.HuellaSHA256) || !huellaCadenaValida(b.EventoMaterialSHA256) ||
		b.OperadorLogin == "" || len(b.OperadorLogin) > 63 || !utf8.ValidString(b.OperadorLogin) || strings.ContainsRune(b.OperadorLogin, 0) ||
		b.ModuloID != "auditoria" || b.RecursoRef != "auditoria_periodica:comun_interna" || b.FinalidadRef != "integridad_auditoria_periodica" ||
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
	if len(detalle) > 8192 || base64.StdEncoding.EncodeToString(detalle) != normalizado || !detallePeriodicaValido(detalle, b) {
		return e, "registro_invalido", "detalle_canonico_base64"
	}
	material := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.periodica.material.v1", TipoOperacionPeriodica,
		b.EventoRef, b.OperadorLogin, b.Accion, b.ModuloID, b.RecursoRef, b.FinalidadRef, b.Resultado, b.MotivoRef,
		b.Proceso, b.Canal, b.CorrelacionRef, string(detalle)))
	if hex.EncodeToString(material[:]) != b.EventoMaterialSHA256 {
		return e, "material_distinto", "evento_material_sha256"
	}
	huella := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.periodica.eslabon.v1", strconv.FormatUint(secuencia, 10),
		b.AnteriorSHA256, b.AuditoriaRef, b.EventoMaterialSHA256, b.RegistradaEn))
	if hex.EncodeToString(huella[:]) != b.HuellaSHA256 {
		return e, "huella_distinta", "huella_sha256"
	}
	e = RegistroEventoAdminV3{AuditoriaRef: b.AuditoriaRef, Secuencia: secuencia, AnteriorSHA256: b.AnteriorSHA256,
		HuellaSHA256: b.HuellaSHA256, EventoRef: b.EventoRef}
	return e, "", ""
}

func detallePeriodicaValido(raw []byte, b RegistroOperacionMantenimientoV1) bool {
	claves, perfil := camposDetallePeriodica(b.Accion, b.Resultado, b.MotivoRef)
	if len(claves) == 0 {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var detalle map[string]any
	if d.Decode(&detalle) != nil || d.Decode(new(any)) != io.EOF || len(detalle) != len(claves) {
		return false
	}
	for _, clave := range claves {
		v, existe := detalle[clave]
		if !existe {
			return false
		}
		if clave == "version" || clave == "previa_secuencia" {
			n, ok := v.(json.Number)
			if !ok || len(n.String()) == 0 {
				return false
			}
			decimal := n.String()
			if clave == "previa_secuencia" && !previaPeriodicaValida(decimal, b) {
				return false
			}
			if clave == "version" && (len(decimal) > 9 || decimal[0] < '1' || decimal[0] > '9' || strings.ContainsFunc(decimal, func(r rune) bool { return r < '0' || r > '9' })) {
				return false
			}
			continue
		}
		s, ok := v.(string)
		if !ok {
			return false
		}
		switch clave {
		case "perfil_tecnico_ref":
			if s != perfil {
				return false
			}
		case "captura_ref":
			if len(s) != 40 || !strings.HasPrefix(s, "captura_") || !hex32Valido(s[8:]) {
				return false
			}
		default:
			// Tras AD207 la captura fija la cabeza sellada, no el asiento anterior.
			if !huellaCadenaValida(s) || clave == "previa_cabeza_sha256" && b.AnteriorSHA256 != MarcadorSinAnteriorV5 && s != b.AnteriorSHA256 {
				return false
			}
		}
	}
	// PostgreSQL jsonb ordena claves por longitud UTF-8 y luego por bytes.
	// El ABI sólo admite claves ASCII, referencias cerradas y enteros; las
	// cadenas no necesitan los escapes HTML propios de encoding/json.
	sort.Slice(claves, func(i, j int) bool {
		if len(claves[i]) != len(claves[j]) {
			return len(claves[i]) < len(claves[j])
		}
		return claves[i] < claves[j]
	})
	var canon bytes.Buffer
	canon.WriteByte('{')
	for i, clave := range claves {
		if i > 0 {
			canon.WriteString(", ")
		}
		canon.WriteString(strconv.Quote(clave))
		canon.WriteString(": ")
		switch valor := detalle[clave].(type) {
		case string:
			canon.WriteString(strconv.Quote(valor))
		case json.Number:
			canon.WriteString(valor.String())
		default:
			return false
		}
	}
	canon.WriteByte('}')
	return bytes.Equal(raw, canon.Bytes())
}

// previaPeriodicaValida: antes de AD207 la captura bloqueaba la cabeza y su
// propio asiento la seguía; después la cabeza sellada queda por debajo de su
// número de orden.
func previaPeriodicaValida(decimal string, b RegistroOperacionMantenimientoV1) bool {
	if b.AnteriorSHA256 != MarcadorSinAnteriorV5 {
		return decimal == strconv.FormatUint(b.Secuencia-1, 10)
	}
	n, err := strconv.ParseUint(decimal, 10, 64)
	return err == nil && strconv.FormatUint(n, 10) == decimal && n < b.Secuencia
}

func camposDetallePeriodica(accion, resultado, motivo string) ([]string, string) {
	perfil := "vec_auditoria_periodica_sellador"
	switch accion {
	case "configurar_sello_periodico_v1":
		perfil = "vec_auditoria_periodica_configurador"
	case "capturar_sello_periodico_v1", "confirmar_sello_periodico_v1":
	default:
		return nil, ""
	}
	if resultado == "denegado" && motivo == "operacion_denegada" || resultado == "error" && motivo == "operacion_error" {
		return []string{"perfil_tecnico_ref"}, perfil
	}
	if accion == "configurar_sello_periodico_v1" && resultado == "permitido" && motivo == "politica_registrada" {
		return []string{"version", "anterior_sha256", "configuracion_sha256", "perfil_tecnico_ref"}, perfil
	}
	if accion == "capturar_sello_periodico_v1" {
		if resultado == "denegado" && motivo == "politica_inactiva" {
			return []string{"perfil_tecnico_ref"}, perfil
		}
		if resultado != "permitido" {
			return nil, ""
		}
		switch motivo {
		case "captura_registrada":
			return []string{"captura_ref", "configuracion_sha256", "previa_secuencia", "previa_cabeza_sha256", "perfil_tecnico_ref"}, perfil
		case "captura_recuperada":
			return []string{"captura_ref", "configuracion_sha256", "perfil_tecnico_ref"}, perfil
		case "no_vencido":
			return []string{"configuracion_sha256", "perfil_tecnico_ref"}, perfil
		case "recibo_recuperado":
			return []string{"captura_ref", "recibo_sha256", "perfil_tecnico_ref"}, perfil
		}
	}
	if accion == "confirmar_sello_periodico_v1" && resultado == "permitido" && (motivo == "recibo_registrado" || motivo == "recibo_recuperado") {
		return []string{"captura_ref", "recibo_sha256", "perfil_tecnico_ref"}, perfil
	}
	return nil, ""
}
