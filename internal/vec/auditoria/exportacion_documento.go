package auditoria

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrDocumentoExportacionAuditoriaInvalido = errors.New("documento_exportacion_auditoria_invalido")

const maxBytesDocumentoExportacionAuditoria int64 = 64 * 1024 * 1024

// VerificarDocumentoExportacionAuditoria admite únicamente las proyecciones
// ya verificables. La cobertura se recibe aparte y su origen no se acredita aquí.
func VerificarDocumentoExportacionAuditoria(documento []byte, cobertura domain.CoberturaCheckpoint,
	maxBytes int64, maxRegistros uint64,
) (string, InformeVerificacion, error) {
	fallo := InformeVerificacion{Estado: "rechazada", AutenticidadCheckpoint: "no_comprobada"}
	if maxBytes < 1 || maxBytes > maxBytesDocumentoExportacionAuditoria || len(documento) == 0 ||
		int64(len(documento)) > maxBytes || maxRegistros == 0 || cobertura.Validar() != nil || cobertura.Registros > maxRegistros {
		return "", fallo, ErrDocumentoExportacionAuditoriaInvalido
	}
	if err := clavesJSONExportacionUnicas(json.NewDecoder(bytes.NewReader(documento)), 0); err != nil {
		return "", fallo, ErrDocumentoExportacionAuditoriaInvalido
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(documento, &campos) != nil || len(campos) != 3 ||
		len(campos["esquema"]) == 0 || len(campos["manifiesto"]) == 0 || len(campos["registros"]) == 0 ||
		bytes.Equal(bytes.TrimSpace(campos["registros"]), []byte("null")) {
		return "", fallo, ErrDocumentoExportacionAuditoriaInvalido
	}
	var esquema string
	if json.Unmarshal(campos["esquema"], &esquema) != nil {
		return "", fallo, ErrDocumentoExportacionAuditoriaInvalido
	}
	checkpoint := CoberturaCadena{
		CadenaID: cobertura.CadenaID, PrimeraSecuencia: cobertura.PrimeraSecuencia,
		UltimaSecuencia: cobertura.UltimaSecuencia, AnteriorSHA256: cobertura.AnteriorSHA256,
		CabezaSHA256: cobertura.CabezaSHA256, Registros: cobertura.Registros,
	}
	var destino any
	switch esquema {
	case EsquemaVerificacion:
		var d DocumentoVerificacion
		destino = &d
		if decodificarDocumentoExportacionEstricto(documento, destino) == nil {
			fallo = VerificarCadenaV3(d, checkpoint, maxRegistros)
		}
	case EsquemaVerificacionMixta, EsquemaVerificacionPreperfil,
		EsquemaVerificacionFuentesIniciales, EsquemaVerificacionUnidadInicial, EsquemaVerificacionBootstrapCentral,
		EsquemaVerificacionMantenimientoFijo, EsquemaVerificacionPeriodica, EsquemaVerificacionPreservacionAuditoria,
		EsquemaVerificacionGobiernoUsuarios, EsquemaVerificacionFronteraAdminTecnicaV1:
		var d DocumentoVerificacionMixta
		destino = &d
		if decodificarDocumentoExportacionEstricto(documento, destino) == nil {
			switch esquema {
			case EsquemaVerificacionMixta:
				fallo = VerificarCadenaMixtaV2(d, checkpoint, maxRegistros)
			case EsquemaVerificacionPreperfil:
				fallo = VerificarCadenaMixtaV3(d, checkpoint, maxRegistros).InformeVerificacion
			case EsquemaVerificacionFuentesIniciales:
				fallo = VerificarCadenaFuentesInicialesV1(d, checkpoint, maxRegistros).InformeVerificacion
			case EsquemaVerificacionUnidadInicial:
				fallo = VerificarCadenaUnidadInicialV1(d, checkpoint, maxRegistros).InformeVerificacion
			case EsquemaVerificacionBootstrapCentral:
				fallo = VerificarCadenaBootstrapCentralV1(d, checkpoint, maxRegistros).InformeVerificacion
			case EsquemaVerificacionMantenimientoFijo:
				fallo = VerificarCadenaMantenimientoFijoV1(d, checkpoint, maxRegistros).InformeVerificacion
			case EsquemaVerificacionPreservacionAuditoria:
				fallo = VerificarCadenaPreservacionAuditoriaV1(d, checkpoint, maxRegistros).InformeVerificacion
			case EsquemaVerificacionPeriodica:
				fallo = VerificarCadenaPeriodicaV1(d, checkpoint, maxRegistros).InformeVerificacion
			case EsquemaVerificacionGobiernoUsuarios:
				fallo = VerificarCadenaGobiernoUsuariosV1(d, checkpoint, maxRegistros)
			case EsquemaVerificacionFronteraAdminTecnicaV1:
				fallo = VerificarCadenaFronteraAdminTecnicaV1(d, checkpoint, maxRegistros)
			}
		}
	default:
		return "", fallo, ErrDocumentoExportacionAuditoriaInvalido
	}
	if fallo.Esquema == "" {
		return "", fallo, ErrDocumentoExportacionAuditoriaInvalido
	}
	return esquema, fallo, nil
}

func decodificarDocumentoExportacionEstricto(documento []byte, destino any) error {
	d := json.NewDecoder(bytes.NewReader(documento))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return ErrDocumentoExportacionAuditoriaInvalido
	}
	canonico, err := json.Marshal(destino)
	if err != nil {
		return ErrDocumentoExportacionAuditoriaInvalido
	}
	// El roundtrip detecta campos omitidos, null, alias de encoding/json y
	// campos tolerados por decodificadores históricos personalizados.
	original, err := valorJSONExportacion(documento)
	if err != nil {
		return ErrDocumentoExportacionAuditoriaInvalido
	}
	reconstruido, err := valorJSONExportacion(canonico)
	if err != nil || !reflect.DeepEqual(original, reconstruido) {
		return ErrDocumentoExportacionAuditoriaInvalido
	}
	return nil
}

func valorJSONExportacion(b []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var valor any
	if err := d.Decode(&valor); err != nil {
		return nil, err
	}
	if d.Decode(new(any)) != io.EOF {
		return nil, ErrDocumentoExportacionAuditoriaInvalido
	}
	return valor, nil
}

func clavesJSONExportacionUnicas(d *json.Decoder, profundidad int) error {
	if profundidad > 32 {
		return ErrDocumentoExportacionAuditoriaInvalido
	}
	t, err := d.Token()
	if err != nil || t == nil {
		return ErrDocumentoExportacionAuditoriaInvalido
	}
	delim, compuesto := t.(json.Delim)
	if !compuesto {
		return nil
	}
	if delim != '{' && delim != '[' {
		return ErrDocumentoExportacionAuditoriaInvalido
	}
	vistas := map[string]bool{}
	for d.More() {
		if delim == '{' {
			k, e := d.Token()
			clave, ok := k.(string)
			if e != nil || !ok || vistas[clave] || strings.ContainsFunc(clave, func(r rune) bool {
				return r > 127 || r >= 'A' && r <= 'Z'
			}) {
				return ErrDocumentoExportacionAuditoriaInvalido
			}
			vistas[clave] = true
		}
		if err := clavesJSONExportacionUnicas(d, profundidad+1); err != nil {
			return err
		}
	}
	cierre, err := d.Token()
	if err != nil || delim == '{' && cierre != json.Delim('}') || delim == '[' && cierre != json.Delim(']') {
		return ErrDocumentoExportacionAuditoriaInvalido
	}
	if profundidad == 0 {
		if _, err := d.Token(); err != io.EOF {
			return ErrDocumentoExportacionAuditoriaInvalido
		}
	}
	return nil
}
