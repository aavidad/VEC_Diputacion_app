package postgres

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	hist "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var ErrTransporteRegistroIncorporacionV2 = errors.New("contratacion temporal: transporte de incorporacion V2 no confiable")

// Límites operativos del transporte, no cardinalidades nuevas del dominio.
// Los bytes JSON no son pg_column_size(jsonb): SQL conserva su propia guarda.
const (
	MaximoBytesMaterialRegistroIncorporacionV2  = 1 << 20
	MaximoBytesEvidenciaRegistroIncorporacionV2 = 16 << 20
	MaximoBytesSalidaRegistroIncorporacionV2    = 32 << 20
	maxProfundidadTransporteRegistroV2          = 32
	maxNodosTransporteRegistroV2                = 500000
)

// LlamadaRegistroIncorporacionV2 es solo transporte, no permiso ni transacción.
// No serializar ni registrar en logs: contiene las piezas nominales originales.
type LlamadaRegistroIncorporacionV2 struct{ parametros []any }

// Parametros conserva los tipos SQL: numeric CT como decimal, bigint lector
// como int64. Cada []byte devuelto es una copia independiente.
func (l LlamadaRegistroIncorporacionV2) Parametros() []any {
	if len(l.parametros) != 23 {
		return nil
	}
	return copiarParametrosRegistroV2(l.parametros)
}
func copiarParametrosRegistroV2(p []any) []any {
	r := append([]any(nil), p...)
	for i, v := range r {
		if b, ok := v.([]byte); ok {
			r[i] = bytes.Clone(b)
		}
	}
	return r
}
func contextoErrorRegistroV2(ctx context.Context) error {
	if ctx != nil {
		if e := ctx.Err(); e != nil {
			return e
		}
	}
	return ErrTransporteRegistroIncorporacionV2
}
func refSeguimientoRegistroV2(s string) bool {
	if len(s) != 68 || !strings.HasPrefix(s, "ref:") || s == "ref:"+strings.Repeat("0", 64) {
		return false
	}
	for _, c := range s[4:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func exportacionParametrosRegistroV2(x vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, numerico bool) ([]any, error) {
	if x.ValidarEstructura() != nil {
		return nil, ErrTransporteRegistroIncorporacionV2
	}
	var pv, fv any
	if numerico {
		pv = strconv.FormatUint(x.PersonaVersion(), 10)
		fv = strconv.FormatUint(x.PerfilVersion(), 10)
	} else {
		// ValidarEstructura ya impone el máximo exacto común 2^53-1.
		pv = int64(x.PersonaVersion())
		fv = int64(x.PerfilVersion())
	}
	return []any{x.CapacidadCanonica(), x.DecisionCanonica(), x.MotivoCanonico(), x.ContextoActorCanonico(), pv, fv,
		x.PayloadVECAD3(), x.SobreCOSESign1(), x.EvidenciaVerificacion(), x.RaizPublicaSPKI()}, nil
}

func PrepararLlamadaRegistroIncorporacionV2(ctx context.Context, actual ct.OrdenConfirmacionIncorporacionV2, lectura lector.OrdenV2, seguimientoRef string, ahora time.Time) (LlamadaRegistroIncorporacionV2, error) {
	var cero LlamadaRegistroIncorporacionV2
	if ctx == nil || ctx.Err() != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	if !dom.InstanteUTCCanonico(ahora) || !refSeguimientoRegistroV2(seguimientoRef) || actual.ValidarEn(ahora) != nil || lectura.ValidarEn(ahora) != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	d, e := actual.Material().Datos()
	if e != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	lc, e := lectura.Material().Contexto()
	if e != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	vc, ec := d.Contexto.Vinculo.Datos()
	vl, el := lc.Vinculo.Datos()
	c, l := actual.Exportacion(), lectura.Exportacion()
	esperado := lector.Selector{OrganizacionRef: d.Preparacion.OrganizacionRef, SolicitudRef: d.Personal.Solicitud.SolicitudRef,
		ExpedienteRef: d.Personal.Solicitud.ExpedienteRef, VersionExpediente: d.Personal.Solicitud.VersionExpediente,
		ResultadoRef: d.Personal.Resultado.ResultadoRef, ReciboRef: d.Personal.Resultado.ReciboRef, RelacionRef: d.Personal.Resultado.RelacionRef,
		OcupacionRef: d.Personal.Resultado.OcupacionRef, MaterialSHA256: d.Personal.MaterialSHA256}
	if ec != nil || el != nil || vc != vl || lectura.Selector() != esperado || lectura.UnidadRef() != d.Preparacion.UnidadRef ||
		!reflect.DeepEqual(lc.Resultado, d.Contexto.Resultado) ||
		!bytes.Equal(c.ContextoActorCanonico(), l.ContextoActorCanonico()) || c.PersonaVersion() != l.PersonaVersion() || c.PerfilVersion() != l.PerfilVersion() ||
		c.ResumenCapacidad().DecisionRef() == l.ResumenCapacidad().DecisionRef() ||
		c.ResumenCapacidad().DecisionRef() == d.Personal.DecisionOriginalRef || l.ResumenCapacidad().DecisionRef() == d.Personal.DecisionOriginalRef {
		return cero, contextoErrorRegistroV2(ctx)
	}
	evidencia, e := ct.CapturarEvidenciaOrdenOriginalIncorporacionV2(ctx, actual)
	if e != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	material, e := actual.Material().MaterialCanonico()
	if e != nil || len(material) == 0 || len(material) > MaximoBytesMaterialRegistroIncorporacionV2 {
		return cero, contextoErrorRegistroV2(ctx)
	}
	b, e := json.Marshal(evidencia)
	if e != nil || len(b) > MaximoBytesEvidenciaRegistroIncorporacionV2 {
		return cero, contextoErrorRegistroV2(ctx)
	}
	pc, e := exportacionParametrosRegistroV2(c, true)
	if e != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	pl, e := exportacionParametrosRegistroV2(l, false)
	if e != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	if actual.ValidarEn(ahora) != nil || lectura.ValidarEn(ahora) != nil || ctx.Err() != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	p := []any{material, seguimientoRef}
	p = append(p, pc...)
	p = append(p, pl...)
	p = append(p, b)
	return LlamadaRegistroIncorporacionV2{p}, nil
}

type salidaTransporteRegistroV2 struct {
	Recibo     ct.ReciboRegistroIncorporacionV2   `json:"recibo"`
	Historia   hist.Documento                     `json:"historia"`
	Recuperado bool                               `json:"recuperado"`
	Consumos   ct.ConsumosRegistroIncorporacionV2 `json:"consumos_actuales"`
}

// DecodificarRegistroIncorporacionV2 comprueba coherencia, nunca commit.
// original debe proceder del lector histórico propietario. Se admite caducada
// AHORA sólo porque Historia valida su fecha ORIGINAL y exige permiso actual.
func DecodificarRegistroIncorporacionV2(ctx context.Context, b []byte, actual, original ct.OrdenConfirmacionIncorporacionV2, ahora time.Time) (ct.ResultadoRegistroIncorporacionV2, error) {
	var cero ct.ResultadoRegistroIncorporacionV2
	if ctx == nil || ctx.Err() != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	if len(b) == 0 || len(b) > MaximoBytesSalidaRegistroIncorporacionV2 || !dom.InstanteUTCCanonico(ahora) || actual.ValidarEn(ahora) != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	var d salidaTransporteRegistroV2
	if decodificarJSONRegistroV2(ctx, b, &d) != nil || d.Historia.Esquema != hist.EsquemaDocumento {
		return cero, contextoErrorRegistroV2(ctx)
	}
	if d.Historia.Evidencia.CotejarOrden(ctx, original) != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	rb, e := json.Marshal(d.Recibo)
	hb, eh := json.Marshal(d.Historia.Recibo)
	if e != nil || eh != nil || !bytes.Equal(rb, hb) {
		return cero, contextoErrorRegistroV2(ctx)
	}
	h, e := ct.NuevaHistoriaRegistroIncorporacionV2(original, d.Recibo, d.Historia.Publicacion, d.Historia.Anterior, d.Historia.Posterior)
	if e != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	r := ct.ResultadoRegistroIncorporacionV2{Recibo: d.Recibo, Historia: h, Recuperado: d.Recuperado, ConsumosActuales: d.Consumos}
	if r.ValidarPara(actual, ahora) != nil || ctx.Err() != nil {
		return cero, contextoErrorRegistroV2(ctx)
	}
	return r.Copia(), nil
}

// Dos fases: límites/duplicados/trailing UTF8 antes del DTO, luego forma y
// recanon tipado exactos. Sólo DTO públicos; jamás Unmarshal de objetos opacos.
func validarJSONRegistroV2(ctx context.Context, b []byte) error {
	if len(b) == 0 || len(b) > MaximoBytesSalidaRegistroIncorporacionV2 || !utf8.Valid(b) {
		return ErrTransporteRegistroIncorporacionV2
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	n := 0
	var nodo func(int) error
	nodo = func(prof int) error {
		n++
		if prof > maxProfundidadTransporteRegistroV2 || n > maxNodosTransporteRegistroV2 || ctx.Err() != nil {
			return contextoErrorRegistroV2(ctx)
		}
		v, e := dec.Token()
		if e != nil {
			return ErrTransporteRegistroIncorporacionV2
		}
		if x, ok := v.(json.Delim); ok {
			switch x {
			case '{':
				seen := map[string]bool{}
				for dec.More() {
					k, e := dec.Token()
					s, ok := k.(string)
					if e != nil || !ok || seen[s] {
						return ErrTransporteRegistroIncorporacionV2
					}
					seen[s] = true
					if e = nodo(prof + 1); e != nil {
						return e
					}
				}
				v, e = dec.Token()
				if e != nil || v != json.Delim('}') {
					return ErrTransporteRegistroIncorporacionV2
				}
			case '[':
				for dec.More() {
					if e = nodo(prof + 1); e != nil {
						return e
					}
				}
				v, e = dec.Token()
				if e != nil || v != json.Delim(']') {
					return ErrTransporteRegistroIncorporacionV2
				}
			default:
				return ErrTransporteRegistroIncorporacionV2
			}
		}
		return nil
	}
	if e := nodo(0); e != nil {
		return e
	}
	if _, e := dec.Token(); e != io.EOF {
		return ErrTransporteRegistroIncorporacionV2
	}
	return nil
}

// La reflexión sólo inspecciona el esquema de DTO públicos para exigir nombres
// exactos (encoding/json acepta aliases). Nunca accede a privados nominales.
// Tiempos: admite RFC3339 UTC de microsegundos con ceros SQL; no redondea.
func normalizarFormaRegistroV2(v any, t reflect.Type) (any, error) {
	fail := func() (any, error) { return nil, ErrTransporteRegistroIncorporacionV2 }
	if t == reflect.TypeOf(time.Time{}) {
		s, ok := v.(string)
		if !ok {
			return fail()
		}
		dt, e := time.Parse(time.RFC3339Nano, s)
		if e != nil || !strings.HasSuffix(s, "Z") || strings.Contains(s, ",") || !dom.InstanteUTCCanonico(dt) {
			return fail()
		}
		if i := strings.IndexByte(s, '.'); i >= 0 && len(s)-i-2 > 6 {
			return fail()
		}
		return dt.Format(time.RFC3339Nano), nil
	}
	if t.Kind() == reflect.Pointer {
		if v == nil {
			return fail()
		}
		return normalizarFormaRegistroV2(v, t.Elem())
	}
	switch t.Kind() {
	case reflect.Struct:
		m, ok := v.(map[string]any)
		if !ok {
			return fail()
		}
		result := make(map[string]any, len(m))
		seen := 0
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				return fail()
			}
			tag := strings.Split(f.Tag.Get("json"), ",")
			name := tag[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = f.Name
			}
			value, exists := m[name]
			if !exists {
				if len(tag) > 1 && tag[1] == "omitempty" {
					continue
				}
				return fail()
			}
			normalized, e := normalizarFormaRegistroV2(value, f.Type)
			if e != nil {
				return fail()
			}
			result[name] = normalized
			seen++
		}
		if seen != len(m) {
			return fail()
		}
		return result, nil
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			s, ok := v.(string)
			if !ok {
				return fail()
			}
			decoded, e := base64.StdEncoding.Strict().DecodeString(s)
			if e != nil || base64.StdEncoding.EncodeToString(decoded) != s {
				return fail()
			}
			return s, nil
		}
		if v == nil {
			return nil, nil
		} // slices nil legítimos del serializador, nunca objetos/punteros null
		a, ok := v.([]any)
		if !ok {
			return fail()
		}
		result := make([]any, len(a))
		for i, x := range a {
			z, e := normalizarFormaRegistroV2(x, t.Elem())
			if e != nil {
				return fail()
			}
			result[i] = z
		}
		return result, nil
	case reflect.String:
		if _, ok := v.(string); !ok {
			return fail()
		}
	case reflect.Bool:
		if _, ok := v.(bool); !ok {
			return fail()
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, ok := v.(json.Number)
		if !ok {
			return fail()
		}
		x, e := strconv.ParseUint(n.String(), 10, t.Bits())
		if e != nil || strconv.FormatUint(x, 10) != n.String() {
			return fail()
		}
	default:
		return fail()
	}
	return v, nil
}
func arbolJSONRegistroV2(b []byte) (any, error) {
	var v any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	e := d.Decode(&v)
	return v, e
}
func decodificarJSONRegistroV2(ctx context.Context, b []byte, out *salidaTransporteRegistroV2) error {
	if validarJSONRegistroV2(ctx, b) != nil {
		return contextoErrorRegistroV2(ctx)
	}
	tree, e := arbolJSONRegistroV2(b)
	if e != nil {
		return ErrTransporteRegistroIncorporacionV2
	}
	original, e := normalizarFormaRegistroV2(tree, reflect.TypeOf(*out))
	if e != nil {
		return e
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(out) != nil {
		return ErrTransporteRegistroIncorporacionV2
	}
	canon, e := json.Marshal(out)
	if e != nil {
		return ErrTransporteRegistroIncorporacionV2
	}
	tree, e = arbolJSONRegistroV2(canon)
	if e != nil {
		return ErrTransporteRegistroIncorporacionV2
	}
	result, e := normalizarFormaRegistroV2(tree, reflect.TypeOf(*out))
	if e != nil || !reflect.DeepEqual(original, result) || ctx.Err() != nil {
		return contextoErrorRegistroV2(ctx)
	}
	return nil
}
