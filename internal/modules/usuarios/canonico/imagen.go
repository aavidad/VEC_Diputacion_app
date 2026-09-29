package canonico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func AudienciaImagen(accion string, superficie vecdomain.SuperficieAutenticacionActorV1) (string, error) {
	switch {
	case accion == ports.AccionConsultarImagen && superficie == vecdomain.SuperficieAutenticacionInternaCorporativaV1:
		return ports.AudienciaConsultarImagenInterna, nil
	case accion == ports.AccionActualizarImagen && superficie == vecdomain.SuperficieAutenticacionInternaCorporativaV1:
		return ports.AudienciaActualizarImagenInterna, nil
	case accion == ports.AccionConsultarImagen && superficie == vecdomain.SuperficieAutenticacionExternaPersonalV1:
		return ports.AudienciaConsultarImagenExterna, nil
	case accion == ports.AccionActualizarImagen && superficie == vecdomain.SuperficieAutenticacionExternaPersonalV1:
		return ports.AudienciaActualizarImagenExterna, nil
	}
	return "", ports.ErrImagenProhibido
}

// HuellaPeticionImagen identifica la petición por persona para reconocer una
// repetición con la misma clave. SQL la recalcula con la misma cadena: todos
// los campos son códigos cerrados o hexadecimales, sin el separador «|».
func HuellaPeticionImagen(persona string, version uint64, catalogo string, e domain.EleccionImagen, fotoSHA256 string) string {
	canon := "usuarios.imagen.peticion.v1|" + persona + "|" + strconv.FormatUint(version, 10) + "|" + catalogo + "|" +
		string(e.Modo) + "|" + e.Paleta + "|" + e.Icono + "|" + fotoSHA256
	h := sha256.Sum256([]byte(canon))
	return hex.EncodeToString(h[:])
}

func hexMinuscula64(s string) bool { return hexadecimalMinuscula(s, 64) }

// SerializarMaterialImagen es la única preimagen V3/SQL p_material text.
func SerializarMaterialImagen(m ports.MaterialImagen) ([]byte, error) {
	if _, err := AudienciaImagen(m.Accion, m.Superficie); err != nil || m.PersonaRef == "" || m.PerfilRef == "" ||
		m.FinalidadRef != ports.FinalidadImagenPropia || m.CatalogoVersionRef == "" || len(m.CatalogoVersionRef) > 96 {
		return nil, ports.ErrImagenPeticionInvalida
	}
	if m.Accion == ports.AccionConsultarImagen {
		if m.ClaveOperacion != "" || m.HuellaPeticion != "" || m.Eleccion != (domain.EleccionImagen{}) || m.FotoSHA256 != "" || m.VersionEsperada != 0 {
			return nil, ports.ErrImagenPeticionInvalida
		}
	} else if m.ClaveOperacion == "" || !hexMinuscula64(m.HuellaPeticion) || m.Eleccion.ValidarCodigos() != nil ||
		(m.FotoSHA256 != "" && (!hexMinuscula64(m.FotoSHA256) || m.Eleccion.Modo != domain.ModoImagenFoto)) ||
		m.HuellaPeticion != HuellaPeticionImagen(m.PersonaRef, m.VersionEsperada, m.CatalogoVersionRef, m.Eleccion, m.FotoSHA256) {
		return nil, ports.ErrImagenPeticionInvalida
	}
	b, err := json.Marshal(m)
	if err != nil || len(b) > 8192 {
		return nil, ports.ErrImagenPeticionInvalida
	}
	return b, nil
}

// RecursoImagen entrega el mismo recurso al emisor V3 y a PostgreSQL.
func RecursoImagen(m ports.MaterialImagen) (vecdomain.RecursoAutorizable, error) {
	b, err := SerializarMaterialImagen(m)
	if err != nil {
		return vecdomain.RecursoAutorizable{}, err
	}
	h := sha256.Sum256(b)
	r := vecdomain.RecursoAutorizable{
		Referencia: m.PersonaRef, ModuloID: "usuarios", Tipo: ports.TipoRecursoImagen,
		Ambitos:   map[string]string{"persona_ref": m.PersonaRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}
	if err := r.Validar(); err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrImagenPeticionInvalida
	}
	return r, nil
}
