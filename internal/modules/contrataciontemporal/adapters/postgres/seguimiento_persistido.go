// Package postgres contiene adaptadores de persistencia para Contratación Temporal.
package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	maximoBytesSnapshotSeguimientoJSON   = 8 << 20
	maximoBytesSnapshotSeguimientoCanon  = 8 << 20
	maximaProfundidadSnapshotSeguimiento = 32
	maximosElementosSnapshotSeguimiento  = 400_000
)

var ErrSeguimientoPersistidoNoConfiable = errors.New("contratacion temporal: seguimiento persistido no confiable")

// ExpectativaSeguimientoPersistido liga una lectura a la identidad que ya
// conoce el llamador. VersionSeguimiento acepta cero: es la versión inicial.
type ExpectativaSeguimientoPersistido struct {
	Referencia      string
	OrganizacionRef string
	ExpedienteRef   string
	RelacionRef     string
	Definicion      domain.ReferenciaDefinicionSeguimiento
	Version         uint64
}

// SnapshotSeguimientoPersistido separa JSON operativo y binario de integridad.
// Ninguno concede autoridad ni prueba que se haya confirmado un commit.
type SnapshotSeguimientoPersistido struct {
	EstadoJSON           []byte
	EstadoCanonico       []byte
	HuellaCanonicaSHA256 string
}

func (s SnapshotSeguimientoPersistido) clonar() SnapshotSeguimientoPersistido {
	s.EstadoJSON = bytes.Clone(s.EstadoJSON)
	s.EstadoCanonico = bytes.Clone(s.EstadoCanonico)
	return s
}

// PrepararSnapshotSeguimientoPersistido serializa solo un estado ya válido.
func PrepararSnapshotSeguimientoPersistido(
	definicion domain.DefinicionSeguimiento,
	seguimiento domain.Seguimiento,
) (SnapshotSeguimientoPersistido, error) {
	var cero SnapshotSeguimientoPersistido
	estado := seguimiento.Estado()
	if definicion.Validar() != nil || seguimiento.Validar(definicion) != nil {
		return cero, ErrSeguimientoPersistidoNoConfiable
	}
	canon, err := domain.SerializarEstadoSeguimientoCanonico(definicion, estado)
	if err != nil || len(canon) == 0 || len(canon) > maximoBytesSnapshotSeguimientoCanon {
		return cero, ErrSeguimientoPersistidoNoConfiable
	}
	jsonEstado, err := json.Marshal(estado)
	if err != nil || len(jsonEstado) == 0 || len(jsonEstado) > maximoBytesSnapshotSeguimientoJSON {
		return cero, ErrSeguimientoPersistidoNoConfiable
	}
	huella := sha256.Sum256(canon)
	return SnapshotSeguimientoPersistido{
		EstadoJSON: jsonEstado, EstadoCanonico: canon,
		HuellaCanonicaSHA256: hex.EncodeToString(huella[:]),
	}.clonar(), nil
}

// RestaurarSeguimientoPersistido verifica el material persistido antes de
// rehidratar. No sustituye al almacén durable ni a sus comprobaciones de ACL.
func RestaurarSeguimientoPersistido(
	definicion domain.DefinicionSeguimiento,
	expectativa ExpectativaSeguimientoPersistido,
	snapshot SnapshotSeguimientoPersistido,
) (domain.Seguimiento, error) {
	var cero domain.Seguimiento
	if definicion.Validar() != nil || expectativa.Definicion.Validar() != nil ||
		!definicion.Referencia().Coincide(expectativa.Definicion) ||
		len(snapshot.EstadoJSON) == 0 || len(snapshot.EstadoJSON) > maximoBytesSnapshotSeguimientoJSON ||
		len(snapshot.EstadoCanonico) == 0 || len(snapshot.EstadoCanonico) > maximoBytesSnapshotSeguimientoCanon ||
		!huellaSnapshotSeguimientoValida(snapshot.HuellaCanonicaSHA256) ||
		validarJSONSnapshotSeguimiento(snapshot.EstadoJSON) != nil {
		return cero, ErrSeguimientoPersistidoNoConfiable
	}
	entrada := snapshot.clonar()
	suma := sha256.Sum256(entrada.EstadoCanonico)
	if hex.EncodeToString(suma[:]) != entrada.HuellaCanonicaSHA256 {
		return cero, ErrSeguimientoPersistidoNoConfiable
	}
	decodificador := json.NewDecoder(bytes.NewReader(entrada.EstadoJSON))
	decodificador.DisallowUnknownFields()
	var estado domain.EstadoPersistidoSeguimiento
	if err := decodificador.Decode(&estado); err != nil || exigirFinSnapshotSeguimiento(decodificador) != nil ||
		estado.Referencia != expectativa.Referencia ||
		estado.OrganizacionRef != expectativa.OrganizacionRef ||
		estado.ExpedienteRef != expectativa.ExpedienteRef ||
		estado.RelacionRef != expectativa.RelacionRef ||
		!estado.Definicion.Coincide(expectativa.Definicion) || estado.Version != expectativa.Version {
		return cero, ErrSeguimientoPersistidoNoConfiable
	}
	rehidratado, err := domain.RehidratarSeguimiento(definicion, estado)
	if err != nil {
		return cero, ErrSeguimientoPersistidoNoConfiable
	}
	canon, err := domain.SerializarEstadoSeguimientoCanonico(definicion, rehidratado.Estado())
	if err != nil || !bytes.Equal(canon, entrada.EstadoCanonico) {
		return cero, ErrSeguimientoPersistidoNoConfiable
	}
	return rehidratado, nil
}

func huellaSnapshotSeguimientoValida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor == string(bytes.Repeat([]byte("0"), sha256.Size*2)) {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil && valor == string(bytes.ToLower([]byte(valor)))
}

func validarJSONSnapshotSeguimiento(contenido []byte) error {
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.UseNumber()
	elementos := 0
	if err := consumirValorSnapshotSeguimiento(d, 0, &elementos); err != nil {
		return err
	}
	if err := exigirFinSnapshotSeguimiento(d); err != nil {
		return err
	}
	return validarFormaEstadoSnapshotSeguimiento(contenido)
}

func validarFormaEstadoSnapshotSeguimiento(contenido []byte) error {
	var raiz json.RawMessage
	if err := json.Unmarshal(contenido, &raiz); err != nil {
		return ErrSeguimientoPersistidoNoConfiable
	}
	estado, err := objetoSnapshotSeguimiento(raiz,
		[]string{"referencia", "organizacion_ref", "expediente_ref", "relacion_ref", "definicion", "version", "estado_actual", "periodo_previsto", "periodos_resultantes", "creado_en", "actualizado_en", "huella_raiz_sha256", "actuaciones"},
		[]string{"cese_efectivo"})
	if err != nil || !camposTipoSnapshotSeguimiento(estado, "string", "referencia", "organizacion_ref", "expediente_ref", "relacion_ref", "estado_actual", "creado_en", "actualizado_en", "huella_raiz_sha256") || !camposTipoSnapshotSeguimiento(estado, "number", "version") || !camposTipoSnapshotSeguimiento(estado, "object", "definicion", "periodo_previsto") || !camposArrayONuloSnapshotSeguimiento(estado, "periodos_resultantes") || !camposTipoSnapshotSeguimiento(estado, "array", "actuaciones") {
		return ErrSeguimientoPersistidoNoConfiable
	}
	if err := validarReferenciaDefinicionSnapshotSeguimiento(estado["definicion"]); err != nil || validarIntervaloSnapshotSeguimiento(estado["periodo_previsto"]) != nil {
		return ErrSeguimientoPersistidoNoConfiable
	}
	if cese, existe := estado["cese_efectivo"]; existe && validarCeseSnapshotSeguimiento(cese) != nil {
		return ErrSeguimientoPersistidoNoConfiable
	}
	for _, clave := range []string{"periodos_resultantes", "actuaciones"} {
		var elementos []json.RawMessage
		if err := json.Unmarshal(estado[clave], &elementos); err != nil {
			return ErrSeguimientoPersistidoNoConfiable
		}
		for _, elemento := range elementos {
			if (clave == "periodos_resultantes" && validarPeriodoResultanteSnapshotSeguimiento(elemento) != nil) || (clave == "actuaciones" && validarActuacionSnapshotSeguimiento(elemento) != nil) {
				return ErrSeguimientoPersistidoNoConfiable
			}
		}
	}
	return nil
}

func objetoSnapshotSeguimiento(contenido json.RawMessage, obligatorias, opcionales []string) (map[string]json.RawMessage, error) {
	var objeto map[string]json.RawMessage
	if json.Unmarshal(contenido, &objeto) != nil || objeto == nil || len(objeto) < len(obligatorias) || len(objeto) > len(obligatorias)+len(opcionales) {
		return nil, ErrSeguimientoPersistidoNoConfiable
	}
	permitidas := make(map[string]struct{}, len(obligatorias)+len(opcionales))
	opcionalesPermitidas := make(map[string]struct{}, len(opcionales))
	for _, clave := range obligatorias {
		if _, existe := objeto[clave]; !existe {
			return nil, ErrSeguimientoPersistidoNoConfiable
		}
		permitidas[clave] = struct{}{}
	}
	for _, clave := range opcionales {
		permitidas[clave] = struct{}{}
		opcionalesPermitidas[clave] = struct{}{}
	}
	for clave, valor := range objeto {
		if _, permitida := permitidas[clave]; !permitida {
			return nil, ErrSeguimientoPersistidoNoConfiable
		}
		if _, opcional := opcionalesPermitidas[clave]; opcional && bytes.Equal(bytes.TrimSpace(valor), []byte("null")) {
			return nil, ErrSeguimientoPersistidoNoConfiable
		}
	}
	return objeto, nil
}

func camposTipoSnapshotSeguimiento(objeto map[string]json.RawMessage, tipo string, claves ...string) bool {
	for _, clave := range claves {
		var valor any
		if json.Unmarshal(objeto[clave], &valor) != nil || tipoJSONSnapshotSeguimiento(valor) != tipo {
			return false
		}
	}
	return true
}

func camposArrayONuloSnapshotSeguimiento(objeto map[string]json.RawMessage, claves ...string) bool {
	for _, clave := range claves {
		if bytes.Equal(bytes.TrimSpace(objeto[clave]), []byte("null")) {
			continue
		}
		if !camposTipoSnapshotSeguimiento(objeto, "array", clave) {
			return false
		}
	}
	return true
}

func tipoJSONSnapshotSeguimiento(valor any) string {
	switch valor.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return ""
	}
}

func validarReferenciaDefinicionSnapshotSeguimiento(contenido json.RawMessage) error {
	o, err := objetoSnapshotSeguimiento(contenido, []string{"referencia", "version", "huella_sha256"}, nil)
	if err != nil || !camposTipoSnapshotSeguimiento(o, "string", "referencia", "huella_sha256") || !camposTipoSnapshotSeguimiento(o, "number", "version") {
		return ErrSeguimientoPersistidoNoConfiable
	}
	return nil
}

func validarIntervaloSnapshotSeguimiento(contenido json.RawMessage) error {
	o, err := objetoSnapshotSeguimiento(contenido, []string{"desde", "hasta"}, nil)
	if err != nil || !camposTipoSnapshotSeguimiento(o, "string", "desde", "hasta") {
		return ErrSeguimientoPersistidoNoConfiable
	}
	return nil
}

func validarPeriodoResultanteSnapshotSeguimiento(contenido json.RawMessage) error {
	o, err := objetoSnapshotSeguimiento(contenido, []string{"intervalo", "actuacion_ref"}, nil)
	if err != nil || !camposTipoSnapshotSeguimiento(o, "object", "intervalo") || !camposTipoSnapshotSeguimiento(o, "string", "actuacion_ref") {
		return ErrSeguimientoPersistidoNoConfiable
	}
	return validarIntervaloSnapshotSeguimiento(o["intervalo"])
}

func validarCeseSnapshotSeguimiento(contenido json.RawMessage) error {
	o, err := objetoSnapshotSeguimiento(contenido, []string{"efectivo_en", "actuacion_ref"}, nil)
	if err != nil || !camposTipoSnapshotSeguimiento(o, "string", "efectivo_en", "actuacion_ref") {
		return ErrSeguimientoPersistidoNoConfiable
	}
	return nil
}

func validarActuacionSnapshotSeguimiento(contenido json.RawMessage) error {
	o, err := objetoSnapshotSeguimiento(contenido,
		[]string{"secuencia", "version_seguimiento", "definicion", "actuacion_ref", "transicion_clave", "clase", "estado_origen", "estado_destino", "actor_ref", "unidad_ref", "efectivo_en", "registrada_en", "documentos", "recibo_ref", "correlacion_ref", "huella_peticion_sha256", "huella_anterior_sha256", "huella_actuacion_sha256"},
		[]string{"motivo_clave", "periodo", "calendario", "rectifica_actuacion_ref"})
	if err != nil || !camposTipoSnapshotSeguimiento(o, "number", "secuencia", "version_seguimiento") || !camposTipoSnapshotSeguimiento(o, "string", "actuacion_ref", "transicion_clave", "clase", "estado_origen", "estado_destino", "actor_ref", "unidad_ref", "efectivo_en", "registrada_en", "recibo_ref", "correlacion_ref", "huella_peticion_sha256", "huella_anterior_sha256", "huella_actuacion_sha256") || !camposTipoSnapshotSeguimiento(o, "object", "definicion") || !camposArrayONuloSnapshotSeguimiento(o, "documentos") || validarReferenciaDefinicionSnapshotSeguimiento(o["definicion"]) != nil {
		return ErrSeguimientoPersistidoNoConfiable
	}
	for _, clave := range []string{"motivo_clave", "rectifica_actuacion_ref"} {
		if valor, existe := o[clave]; existe && !camposTipoSnapshotSeguimiento(map[string]json.RawMessage{clave: valor}, "string", clave) {
			return ErrSeguimientoPersistidoNoConfiable
		}
	}
	if valor, existe := o["periodo"]; existe && (tipoCrudoSnapshotSeguimiento(valor) != "object" || validarIntervaloSnapshotSeguimiento(valor) != nil) {
		return ErrSeguimientoPersistidoNoConfiable
	}
	if valor, existe := o["calendario"]; existe && validarCalendarioSnapshotSeguimiento(valor) != nil {
		return ErrSeguimientoPersistidoNoConfiable
	}
	var documentos []json.RawMessage
	if json.Unmarshal(o["documentos"], &documentos) != nil {
		return ErrSeguimientoPersistidoNoConfiable
	}
	for _, documento := range documentos {
		d, err := objetoSnapshotSeguimiento(documento, []string{"tipo_clave", "referencia"}, nil)
		if err != nil || !camposTipoSnapshotSeguimiento(d, "string", "tipo_clave", "referencia") {
			return ErrSeguimientoPersistidoNoConfiable
		}
	}
	return nil
}

func tipoCrudoSnapshotSeguimiento(contenido json.RawMessage) string {
	var valor any
	if json.Unmarshal(contenido, &valor) != nil {
		return ""
	}
	return tipoJSONSnapshotSeguimiento(valor)
}

func validarCalendarioSnapshotSeguimiento(contenido json.RawMessage) error {
	o, err := objetoSnapshotSeguimiento(contenido, []string{"referencia", "version", "huella_sha256", "ambito_territorial_clave", "resultado_clave", "calculado_en"}, nil)
	if err != nil || !camposTipoSnapshotSeguimiento(o, "string", "referencia", "huella_sha256", "ambito_territorial_clave", "resultado_clave", "calculado_en") || !camposTipoSnapshotSeguimiento(o, "number", "version") {
		return ErrSeguimientoPersistidoNoConfiable
	}
	return nil
}

func consumirValorSnapshotSeguimiento(d *json.Decoder, profundidad int, elementos *int) error {
	if profundidad > maximaProfundidadSnapshotSeguimiento || elementos == nil || *elementos >= maximosElementosSnapshotSeguimiento {
		return ErrSeguimientoPersistidoNoConfiable
	}
	*elementos = *elementos + 1
	token, err := d.Token()
	if err != nil {
		return err
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		claves := map[string]struct{}{}
		for d.More() {
			clave, err := d.Token()
			texto, ok := clave.(string)
			if err != nil || !ok {
				return ErrSeguimientoPersistidoNoConfiable
			}
			if _, repetida := claves[texto]; repetida {
				return ErrSeguimientoPersistidoNoConfiable
			}
			claves[texto] = struct{}{}
			if err := consumirValorSnapshotSeguimiento(d, profundidad+1, elementos); err != nil {
				return err
			}
		}
		fin, err := d.Token()
		if err != nil || fin != json.Delim('}') {
			return ErrSeguimientoPersistidoNoConfiable
		}
	case '[':
		for d.More() {
			if err := consumirValorSnapshotSeguimiento(d, profundidad+1, elementos); err != nil {
				return err
			}
		}
		fin, err := d.Token()
		if err != nil || fin != json.Delim(']') {
			return ErrSeguimientoPersistidoNoConfiable
		}
	default:
		return ErrSeguimientoPersistidoNoConfiable
	}
	return nil
}

func exigirFinSnapshotSeguimiento(d *json.Decoder) error {
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return ErrSeguimientoPersistidoNoConfiable
	}
	return nil
}
