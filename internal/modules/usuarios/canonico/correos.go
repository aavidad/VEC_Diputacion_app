// Package canonico define la preimagen única de autorización/persistencia de
// correos. Los adaptadores y el caso de uso comparten estas funciones.
package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func AudienciaCorreos(accion string, superficie vecdomain.SuperficieAutenticacionActorV1) (string, error) {
	switch superficie {
	case vecdomain.SuperficieAutenticacionInternaCorporativaV1:
		switch accion {
		case ports.AccionConsultarCorreos:
			return ports.AudienciaConsultarCorreosInterna, nil
		case ports.AccionAnadirCorreo:
			return ports.AudienciaAnadirCorreoInterna, nil
		case ports.AccionReenviarCorreo:
			return ports.AudienciaReenviarCorreoInterna, nil
		case ports.AccionVerificarCorreo:
			return ports.AudienciaVerificarCorreoInterna, nil
		case ports.AccionActivarCorreo:
			return ports.AudienciaActivarCorreoInterna, nil
		case ports.AccionRetirarCorreo:
			return ports.AudienciaRetirarCorreoInterna, nil
		}
	case vecdomain.SuperficieAutenticacionExternaPersonalV1:
		switch accion {
		case ports.AccionConsultarCorreos:
			return ports.AudienciaConsultarCorreosExterna, nil
		case ports.AccionAnadirCorreo:
			return ports.AudienciaAnadirCorreoExterna, nil
		case ports.AccionReenviarCorreo:
			return ports.AudienciaReenviarCorreoExterna, nil
		case ports.AccionVerificarCorreo:
			return ports.AudienciaVerificarCorreoExterna, nil
		case ports.AccionActivarCorreo:
			return ports.AudienciaActivarCorreoExterna, nil
		case ports.AccionRetirarCorreo:
			return ports.AudienciaRetirarCorreoExterna, nil
		}
	}
	return "", ports.ErrCorreosProhibido
}

func HuellasSemanticasValidas(h ports.HuellasSemanticasCorreo) bool {
	if len(h.Retenidas) > 8 {
		return false
	}
	vistas := make(map[string]bool, len(h.Retenidas)+1)
	for _, sello := range append([]ports.HuellaSemanticaCorreo{h.Activa}, h.Retenidas...) {
		valor, err := hex.DecodeString(sello.Valor)
		if sello.ClaveRef == "" || vistas[sello.ClaveRef] || err != nil || len(valor) != 32 {
			return false
		}
		vistas[sello.ClaveRef] = true
	}
	return true
}

// SerializarMaterialCorreos es la única preimagen V3/SQL p_material text.
// Sólo serializa HMAC versionados, nunca direcciones ni códigos en claro.
func SerializarMaterialCorreos(m ports.MaterialCorreos) ([]byte, error) {
	if _, err := AudienciaCorreos(m.Accion, m.Superficie); err != nil || m.PersonaRef == "" || m.PerfilRef == "" || m.FinalidadRef != ports.FinalidadCorreosPropios {
		return nil, ports.ErrCorreosInvalidos
	}
	if m.Accion == ports.AccionConsultarCorreos {
		if m.ClaveOperacion != "" || m.HuellasPeticion.Activa != (ports.HuellaSemanticaCorreo{}) || len(m.HuellasPeticion.Retenidas) != 0 || m.CorreoRef != "" || m.VersionEsperada != 0 {
			return nil, ports.ErrCorreosInvalidos
		}
	} else if m.ClaveOperacion == "" || !HuellasSemanticasValidas(m.HuellasPeticion) {
		return nil, ports.ErrCorreosInvalidos
	}
	m.HuellasPeticion.Retenidas = append([]ports.HuellaSemanticaCorreo{}, m.HuellasPeticion.Retenidas...)
	sort.Slice(m.HuellasPeticion.Retenidas, func(i, j int) bool {
		return m.HuellasPeticion.Retenidas[i].ClaveRef < m.HuellasPeticion.Retenidas[j].ClaveRef
	})
	huellaJSON := json.RawMessage("{}")
	if m.Accion != ports.AccionConsultarCorreos {
		var err error
		huellaJSON, err = json.Marshal(m.HuellasPeticion)
		if err != nil {
			return nil, ports.ErrCorreosInvalidos
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
		CorreoRef       string                                   `json:"correo_ref"`
	}{m.Superficie, m.PersonaRef, m.PerfilRef, m.Accion, m.FinalidadRef, m.VersionEsperada, m.ClaveOperacion, huellaJSON, m.CorreoRef})
	if err != nil || len(b) > 8192 {
		return nil, ports.ErrCorreosInvalidos
	}
	return b, nil
}

// RecursoCorreos entrega el mismo recurso al emisor V3 y a PostgreSQL. La
// superficie viaja en material/vínculo/audiencia, fuera de los ámbitos.
func RecursoCorreos(m ports.MaterialCorreos) (vecdomain.RecursoAutorizable, error) {
	b, err := SerializarMaterialCorreos(m)
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	r := vecdomain.RecursoAutorizable{
		Referencia: m.PersonaRef, ModuloID: "usuarios", Tipo: ports.TipoRecursoCorreos,
		Ambitos:   map[string]string{"persona_ref": m.PersonaRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}
	if err := r.Validar(); err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrCorreosInvalidos
	}
	return r, nil
}
