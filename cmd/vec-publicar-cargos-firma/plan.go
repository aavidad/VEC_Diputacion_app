package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

// Contrato del fichero de cargos y del plan de AUT53. El fichero lo escribe
// quien prepara la operación a partir del plan nominal de firma; el plan es el
// texto canónico que aprueba dirección y ejecuta la base. Las reglas repiten
// las de AUT53 para fallar antes de pedir la aprobación; la base las vuelve a
// comprobar y es la autoridad.
const (
	esquemaCargos = "vec.admin.cargos-firma.cargos.v1"
	esquemaPlan   = "vec.admin.cargos-firma.plan.v1"
	finalidadV2   = "gestionar_contratacion_temporal"
	maximoCargos  = 16
	maximoCaduca  = 24 * time.Hour
)

var errCargos = errors.New("cargos_invalidos")

var (
	reRolID       = regexp.MustCompile(`^[a-z][a-z0-9_]{2,63}$`)
	reHuella      = regexp.MustCompile(`^[0-9a-f]{64}$`)
	reRegla       = regexp.MustCompile(`^[a-z][a-z0-9_]{2,63}$`)
	reOrg         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$`)
	reAccion      = regexp.MustCompile(`^contratacion_temporal\.documento(\.[a-z][a-z0-9_]{1,63}){1,4}$`)
	reTipo        = regexp.MustCompile(`^documento_([a-z0-9_]{1,80}_)?contratacion_temporal$`)
	reFinalidad   = regexp.MustCompile(`^[a-z][a-z0-9_]{2,127}$`)
	reIntervencio = regexp.MustCompile(`(?i)(fiscaliz|intervenc)`)
	reOperacion   = regexp.MustCompile(`^rpa_cf_[A-Za-z0-9_-]{19,121}$`)
)

type competencia struct {
	Accion      string `json:"accion"`
	TipoRecurso string `json:"tipo_recurso"`
	Finalidad   string `json:"finalidad"`
}

type cargoEntrada struct {
	RolID                     string        `json:"rol_id"`
	Version                   uint64        `json:"version"`
	Nombre                    string        `json:"nombre"`
	VersionAnteriorSHA256     string        `json:"version_anterior_sha256"`
	OperacionesV2             []string      `json:"operaciones_v2"`
	Competencias              []competencia `json:"competencias"`
	ReglaAsignacion           string        `json:"regla_asignacion"`
	OrganizacionRef           string        `json:"organizacion_ref"`
	VigenteDesde              time.Time     `json:"vigente_desde"`
	VigenteHasta              time.Time     `json:"vigente_hasta"`
	DuracionPropuestaSegundos uint64        `json:"duracion_propuesta_segundos"`
}

type ficheroCargos struct {
	Esquema string         `json:"esquema"`
	Cargos  []cargoEntrada `json:"cargos"`
}

// cargoPlan es la entrada del plan: la del fichero más la huella del rol.
// Los campos se repiten (sin incrustar) para que la lectura estricta exija
// exactamente las doce claves.
type cargoPlan struct {
	RolID                     string        `json:"rol_id"`
	Version                   uint64        `json:"version"`
	Nombre                    string        `json:"nombre"`
	VersionAnteriorSHA256     string        `json:"version_anterior_sha256"`
	OperacionesV2             []string      `json:"operaciones_v2"`
	Competencias              []competencia `json:"competencias"`
	VersionRolSHA256          string        `json:"version_rol_sha256"`
	ReglaAsignacion           string        `json:"regla_asignacion"`
	OrganizacionRef           string        `json:"organizacion_ref"`
	VigenteDesde              time.Time     `json:"vigente_desde"`
	VigenteHasta              time.Time     `json:"vigente_hasta"`
	DuracionPropuestaSegundos uint64        `json:"duracion_propuesta_segundos"`
}

func (c cargoPlan) entrada() cargoEntrada {
	return cargoEntrada{RolID: c.RolID, Version: c.Version, Nombre: c.Nombre, VersionAnteriorSHA256: c.VersionAnteriorSHA256,
		OperacionesV2: c.OperacionesV2, Competencias: c.Competencias, ReglaAsignacion: c.ReglaAsignacion, OrganizacionRef: c.OrganizacionRef,
		VigenteDesde: c.VigenteDesde, VigenteHasta: c.VigenteHasta, DuracionPropuestaSegundos: c.DuracionPropuestaSegundos}
}

type documentoPlan struct {
	Esquema      string      `json:"esquema"`
	OperacionRef string      `json:"operacion_ref"`
	PreparadoEn  time.Time   `json:"preparado_en"`
	CaducaEn     time.Time   `json:"caduca_en"`
	Cargos       []cargoPlan `json:"cargos"`
}

// concesionV2 devuelve las dos concesiones de la firma V2 tal como las
// comprueba AD162. Es la misma lista que fija AUT53 en SQL.
func concesionV2(op string) (vd.ConcesionRol, bool) {
	switch op {
	case "firmar_vec":
		return vd.ConcesionRol{Accion: ctports.AccionRegistrarFirmaVec, ModuloID: ctports.ModuloContratacion, TipoRecurso: ctports.TipoRecursoFirmaVec,
			Finalidades: []string{finalidadV2}, GarantiaMinima: vd.AuthAssuranceHigh}, true
	case "consultar_r5":
		return vd.ConcesionRol{Accion: ctports.AccionConsultarFirmasR5V2, ModuloID: ctports.ModuloContratacion, TipoRecurso: ctports.TipoRecursoConsultaFirmasR5,
			Finalidades: []string{finalidadV2}, GarantiaMinima: vd.AuthAssuranceHigh, CamposPermitidos: ctports.CamposConsultaFirmasR5V2()}, true
	}
	return vd.ConcesionRol{}, false
}

func validarCargo(c cargoEntrada) error {
	if !reRolID.MatchString(c.RolID) || c.Version < 1 || c.Version > 999999999 ||
		utf8.RuneCountInString(c.Nombre) < 3 || utf8.RuneCountInString(c.Nombre) > 200 || reIntervencio.MatchString(c.Nombre) ||
		(c.Version == 1) != (c.VersionAnteriorSHA256 == "") || (c.Version > 1 && !reHuella.MatchString(c.VersionAnteriorSHA256)) ||
		len(c.OperacionesV2) > 2 || len(c.Competencias) < 1 || len(c.Competencias) > 8 ||
		!reRegla.MatchString(c.ReglaAsignacion) || !reOrg.MatchString(c.OrganizacionRef) ||
		!segundos(c.VigenteDesde) || !segundos(c.VigenteHasta) || !c.VigenteHasta.After(c.VigenteDesde) ||
		c.DuracionPropuestaSegundos < 60 || c.DuracionPropuestaSegundos > 31536000 {
		return errCargos
	}
	anterior := ""
	for _, op := range c.OperacionesV2 {
		if _, ok := concesionV2(op); !ok || op <= anterior {
			return errCargos
		}
		anterior = op
	}
	for _, x := range c.Competencias {
		if !reAccion.MatchString(x.Accion) || !reTipo.MatchString(x.TipoRecurso) || !reFinalidad.MatchString(x.Finalidad) || reIntervencio.MatchString(x.Finalidad) {
			return errCargos
		}
	}
	return nil
}

func segundos(t time.Time) bool {
	_, off := t.Zone()
	return !t.IsZero() && off == 0 && t.Nanosecond() == 0
}

// versionRol construye el mismo documento que documento_rol_cargo_firma_v1:
// V2 en el orden del plan y después las competencias, todo de CT y alto.
func versionRol(c cargoEntrada, operacionRef string, preparado time.Time) (vd.VersionRol, error) {
	if c.Version > uint64(int(^uint(0)>>1)) {
		return vd.VersionRol{}, errCargos
	}
	v := vd.VersionRol{RolID: c.RolID, Version: int(c.Version), Nombre: c.Nombre, Estado: vd.EstadoVersionRolPublicada,
		PublicadaPor: "operacion:cargos_firma:" + operacionRef, PublicadaEn: preparado}
	for _, op := range c.OperacionesV2 {
		x, _ := concesionV2(op)
		v.Concesiones = append(v.Concesiones, x)
	}
	for _, x := range c.Competencias {
		v.Concesiones = append(v.Concesiones, vd.ConcesionRol{Accion: x.Accion, ModuloID: ctports.ModuloContratacion, TipoRecurso: x.TipoRecurso,
			Finalidades: []string{x.Finalidad}, GarantiaMinima: vd.AuthAssuranceHigh})
	}
	return v, v.Validar()
}

// prepararPlan valida el fichero de cargos y devuelve el plan con sus huellas.
// No consulta la base: la CAS de versión y las exclusiones de AUT49 las
// comprueba la operación al aplicar.
func prepararPlan(f ficheroCargos, operacionRef string, preparado time.Time, caduca time.Duration) (documentoPlan, error) {
	if f.Esquema != esquemaCargos || len(f.Cargos) < 1 || len(f.Cargos) > maximoCargos || !reOperacion.MatchString(operacionRef) ||
		!segundos(preparado) || caduca < time.Minute || caduca > maximoCaduca || caduca%time.Second != 0 {
		return documentoPlan{}, errCargos
	}
	p := documentoPlan{Esquema: esquemaPlan, OperacionRef: operacionRef, PreparadoEn: preparado, CaducaEn: preparado.Add(caduca)}
	vistos := map[string]bool{}
	for _, c := range f.Cargos {
		if validarCargo(c) != nil || vistos[c.RolID] || !c.VigenteHasta.After(preparado) {
			return documentoPlan{}, errCargos
		}
		vistos[c.RolID] = true
		v, err := versionRol(c, operacionRef, preparado)
		if err != nil {
			return documentoPlan{}, errCargos
		}
		h, err := v.HuellaSHA256()
		if err != nil {
			return documentoPlan{}, errCargos
		}
		p.Cargos = append(p.Cargos, cargoPlan{RolID: c.RolID, Version: c.Version, Nombre: c.Nombre, VersionAnteriorSHA256: c.VersionAnteriorSHA256,
			OperacionesV2: slices.Clone(c.OperacionesV2), Competencias: slices.Clone(c.Competencias), VersionRolSHA256: h,
			ReglaAsignacion: c.ReglaAsignacion, OrganizacionRef: c.OrganizacionRef, VigenteDesde: c.VigenteDesde, VigenteHasta: c.VigenteHasta,
			DuracionPropuestaSegundos: c.DuracionPropuestaSegundos})
	}
	return p, nil
}

func nuevaOperacionRef() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "rpa_cf_" + hex.EncodeToString(b), nil
}

func instante(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05Z") }

// textoPlan escribe el plan como lo imprime PostgreSQL para jsonb (claves por
// longitud y bytes, ", " y ": "). AUT53 exige que el texto aprobado sea
// exactamente plan::jsonb::text; si no, se deniega.
func textoPlan(p documentoPlan) []byte {
	cargos := make([]any, 0, len(p.Cargos))
	for _, c := range p.Cargos {
		ops := make([]any, 0, len(c.OperacionesV2))
		for _, op := range c.OperacionesV2 {
			ops = append(ops, op)
		}
		comps := make([]any, 0, len(c.Competencias))
		for _, x := range c.Competencias {
			comps = append(comps, map[string]any{"accion": x.Accion, "tipo_recurso": x.TipoRecurso, "finalidad": x.Finalidad})
		}
		cargos = append(cargos, map[string]any{
			"rol_id": c.RolID, "version": c.Version, "nombre": c.Nombre, "version_anterior_sha256": c.VersionAnteriorSHA256,
			"operaciones_v2": ops, "competencias": comps, "version_rol_sha256": c.VersionRolSHA256,
			"regla_asignacion": c.ReglaAsignacion, "organizacion_ref": c.OrganizacionRef,
			"vigente_desde": instante(c.VigenteDesde), "vigente_hasta": instante(c.VigenteHasta),
			"duracion_propuesta_segundos": c.DuracionPropuestaSegundos,
		})
	}
	return jsonbTexto(map[string]any{"esquema": p.Esquema, "operacion_ref": p.OperacionRef,
		"preparado_en": instante(p.PreparadoEn), "caduca_en": instante(p.CaducaEn), "cargos": cargos}, nil)
}

func jsonbTexto(v any, b []byte) []byte {
	switch x := v.(type) {
	case map[string]any:
		claves := make([]string, 0, len(x))
		for k := range x {
			claves = append(claves, k)
		}
		// jsonb ordena las claves por longitud y, a igual longitud, por bytes.
		slices.SortFunc(claves, func(a, c string) int {
			if len(a) != len(c) {
				return len(a) - len(c)
			}
			if a < c {
				return -1
			}
			if a > c {
				return 1
			}
			return 0
		})
		b = append(b, '{')
		for i, k := range claves {
			if i > 0 {
				b = append(b, ", "...)
			}
			b = cadenaJSONB(k, b)
			b = append(b, ": "...)
			b = jsonbTexto(x[k], b)
		}
		return append(b, '}')
	case []any:
		b = append(b, '[')
		for i, e := range x {
			if i > 0 {
				b = append(b, ", "...)
			}
			b = jsonbTexto(e, b)
		}
		return append(b, ']')
	case string:
		return cadenaJSONB(x, b)
	case uint64:
		return strconv.AppendUint(b, x, 10)
	}
	panic("tipo no admitido en el plan")
}

// cadenaJSONB reproduce escape_json de PostgreSQL: sólo se escapan comillas,
// barra invertida y caracteres de control; el resto va en UTF-8.
func cadenaJSONB(s string, b []byte) []byte {
	b = append(b, '"')
	for _, r := range s {
		switch r {
		case '"':
			b = append(b, `\"`...)
		case '\\':
			b = append(b, `\\`...)
		case '\b':
			b = append(b, `\b`...)
		case '\f':
			b = append(b, `\f`...)
		case '\n':
			b = append(b, `\n`...)
		case '\r':
			b = append(b, `\r`...)
		case '\t':
			b = append(b, `\t`...)
		default:
			if r < 0x20 {
				b = append(b, `\u00`...)
				b = append(b, "0123456789abcdef"[r>>4], "0123456789abcdef"[r&0xf])
			} else {
				b = utf8.AppendRune(b, r)
			}
		}
	}
	return append(b, '"')
}
