package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
	"unicode"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var (
	patronCandidatoAvisos   = regexp.MustCompile(`^can_[A-Za-z0-9_-]{22,128}$`)
	patronLlamamientoAvisos = regexp.MustCompile(`^llamamiento:[0-9a-f]{64}$`)
	patronCorreoRefAvisos   = regexp.MustCompile(`^correo:[0-9a-f]{32}$`)
)

// CorreoRefValida comprueba la forma de la referencia opaca de un correo.
func CorreoRefValida(v string) bool { return patronCorreoRefAvisos.MatchString(v) }

// referenciaAvisosSegura rechaza lo que Go y PostgreSQL escaparían de forma
// distinta al serializar JSON (<, >, &, comillas, barra invertida, controles
// y separadores U+2028/U+2029). Así la huella del recurso coincide en SQL.
func referenciaAvisosSegura(v string) bool {
	if len(v) == 0 || len(v) > 512 || strings.TrimSpace(v) != v {
		return false
	}
	for _, r := range v {
		if r == '<' || r == '>' || r == '&' || r == '"' || r == '\\' || r == ' ' || r == ' ' || unicode.IsControl(r) || r == unicode.ReplacementChar {
			return false
		}
	}
	return true
}

// MaterialCorreoAvisos construye el material de la lectura. La superficie es
// siempre la interna: la lectura la hace RRHH al emitir el llamamiento.
func MaterialCorreoAvisos(s ports.SolicitudCorreoAvisos) (ports.MaterialCorreoAvisos, error) {
	m := ports.MaterialCorreoAvisos{
		Esquema: ports.EsquemaMaterialCorreoAvisos, Superficie: string(vecdomain.SuperficieAutenticacionInternaCorporativaV1),
		FinalidadRef: ports.FinalidadCorreoAvisosLlamamiento, BolsaRef: s.BolsaRef, UnidadRef: s.UnidadRef, AmbitoRef: s.AmbitoRef,
		LlamamientoRef: s.LlamamientoRef, CandidatoRef: s.CandidatoRef,
	}
	if _, err := SerializarMaterialCorreoAvisos(m); err != nil {
		return ports.MaterialCorreoAvisos{}, err
	}
	return m, nil
}

// SerializarMaterialCorreoAvisos es la única preimagen V3/SQL (p_material).
func SerializarMaterialCorreoAvisos(m ports.MaterialCorreoAvisos) ([]byte, error) {
	for _, v := range []string{m.BolsaRef, m.UnidadRef, m.AmbitoRef, m.LlamamientoRef, m.CandidatoRef} {
		if !referenciaAvisosSegura(v) {
			return nil, ports.ErrCorreosInvalidos
		}
	}
	if m.Esquema != ports.EsquemaMaterialCorreoAvisos || m.Superficie != string(vecdomain.SuperficieAutenticacionInternaCorporativaV1) ||
		m.FinalidadRef != ports.FinalidadCorreoAvisosLlamamiento || !patronCandidatoAvisos.MatchString(m.CandidatoRef) ||
		!patronLlamamientoAvisos.MatchString(m.LlamamientoRef) {
		return nil, ports.ErrCorreosInvalidos
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > 4096 {
		return nil, ports.ErrCorreosInvalidos
	}
	return b, nil
}

// RecursoCorreoAvisos es el recurso que autoriza la V3: la bolsa constituida
// con su unidad y ámbito, como en la emisión, y la huella del material, que
// fija el llamamiento y la persona candidata. PostgreSQL recalcula la huella.
func RecursoCorreoAvisos(m ports.MaterialCorreoAvisos) (vecdomain.RecursoAutorizable, []byte, error) {
	b, err := SerializarMaterialCorreoAvisos(m)
	if err != nil {
		return vecdomain.RecursoAutorizable{}, nil, err
	}
	h := sha256.Sum256(b)
	r := vecdomain.RecursoAutorizable{
		Referencia: m.BolsaRef, ModuloID: ports.ModuloV3CorreoAvisosLlamamiento, Tipo: ports.TipoRecursoV3CorreoAvisosLlamamiento,
		Ambitos:   map[string]string{"unidad_ref": m.UnidadRef, "ambito_ref": m.AmbitoRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}
	if err := r.Validar(); err != nil {
		return vecdomain.RecursoAutorizable{}, nil, ports.ErrCorreosInvalidos
	}
	return r, b, nil
}
