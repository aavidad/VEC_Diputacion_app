// Package canonico define la preimagen única de autorización y persistencia
// de la ficha. La aplicación, el emisor V3 y PostgreSQL usan estos bytes.
package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"vec-diputacion-granada/internal/modules/aspirantes/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func hexMinuscula(s string, largo int) bool {
	if len(s) != largo {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func claveRefValida(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != ':' && r != '.' && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

// HuellasValidas exige clave y HMAC hexadecimal de 64 caracteres, sin claves
// repetidas y con ocho retenidas como mucho.
func HuellasValidas(h ports.HuellasSemanticas) bool {
	if len(h.Retenidas) > 8 {
		return false
	}
	vistas := map[string]bool{}
	for _, s := range append([]ports.HuellaSemantica{h.Activa}, h.Retenidas...) {
		if !claveRefValida(s.ClaveRef) || vistas[s.ClaveRef] || !hexMinuscula(s.Valor, 64) {
			return false
		}
		vistas[s.ClaveRef] = true
	}
	return true
}

// IndiceValido comprueba la forma del índice ciego.
func IndiceValido(i ports.IndiceDocumento) bool {
	return claveRefValida(i.ClaveRef) && hexMinuscula(i.Valor, 64)
}

// SerializarMaterial es la única preimagen del parámetro p_material text. El
// orden de las claves es fijo y las huellas retenidas van ordenadas.
func SerializarMaterial(m ports.MaterialFicha) ([]byte, error) {
	if _, err := ports.Audiencia(m.Accion); err != nil || m.Superficie != vecdomain.SuperficieAutenticacionExternaPersonalV1 ||
		m.PersonaRef == "" || m.PerfilRef == "" || m.FinalidadRef != ports.FinalidadFicha || !IndiceValido(m.IndiceDocumento) {
		return nil, ports.ErrInvalida
	}
	huellas := json.RawMessage("{}")
	switch m.Accion {
	case ports.AccionConsultar:
		if m.ClaveOperacion != "" || m.VersionEsperada != 0 || m.HuellasPeticion.Activa != (ports.HuellaSemantica{}) || len(m.HuellasPeticion.Retenidas) != 0 {
			return nil, ports.ErrInvalida
		}
	default:
		if m.ClaveOperacion == "" || !HuellasValidas(m.HuellasPeticion) ||
			(m.Accion == ports.AccionAlta && m.VersionEsperada != 0) || (m.Accion == ports.AccionRectificar && m.VersionEsperada == 0) {
			return nil, ports.ErrInvalida
		}
		h := m.HuellasPeticion
		h.Retenidas = append([]ports.HuellaSemantica{}, h.Retenidas...)
		sort.Slice(h.Retenidas, func(i, j int) bool { return h.Retenidas[i].ClaveRef < h.Retenidas[j].ClaveRef })
		var err error
		if huellas, err = json.Marshal(h); err != nil {
			return nil, ports.ErrInvalida
		}
	}
	b, err := json.Marshal(struct {
		Superficie      vecdomain.SuperficieAutenticacionActorV1 `json:"superficie"`
		PersonaRef      string                                   `json:"persona_ref"`
		PerfilRef       string                                   `json:"perfil_ref"`
		Accion          string                                   `json:"accion"`
		FinalidadRef    string                                   `json:"finalidad_ref"`
		VersionEsperada uint64                                   `json:"version_esperada"`
		ClaveOperacion  string                                   `json:"clave_operacion"`
		HuellasPeticion json.RawMessage                          `json:"huellas_peticion"`
		IndiceDocumento ports.IndiceDocumento                    `json:"indice_documento"`
	}{m.Superficie, m.PersonaRef, m.PerfilRef, m.Accion, m.FinalidadRef, m.VersionEsperada, m.ClaveOperacion, huellas, m.IndiceDocumento})
	if err != nil || len(b) > 8192 {
		return nil, ports.ErrInvalida
	}
	return b, nil
}

// Recurso entrega el mismo recurso al emisor V3 y a PostgreSQL: la persona
// que actúa y la huella SHA-256 del material exacto.
func Recurso(m ports.MaterialFicha) (vecdomain.RecursoAutorizable, error) {
	b, err := SerializarMaterial(m)
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	r := vecdomain.RecursoAutorizable{
		Referencia: m.PersonaRef, ModuloID: ports.ModuloID, Tipo: ports.TipoRecursoFicha,
		Ambitos:   map[string]string{"persona_ref": m.PersonaRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}
	if r.Validar() != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrInvalida
	}
	return r, nil
}
