package administracionperfiles

import (
	"bytes"
	"encoding/json"
	"io"
)

// El modo inmediato omite vigente_desde. La ausencia y JSON null no son una
// fecha ni permiten al cliente escoger el instante efectivo de SQL.
func (c *CambioPerfil) UnmarshalJSON(b []byte) error {
	var campos map[string]json.RawMessage
	if json.Unmarshal(b, &campos) != nil || len(campos) < 3 || len(campos) > 4 {
		return ErrConfiguracionIncompleta
	}
	for _, clave := range []string{"operacion", "rol_version_ref", "objetivo"} {
		if _, existe := campos[clave]; !existe {
			return ErrConfiguracionIncompleta
		}
	}
	if len(campos) == 4 {
		if _, existe := campos["inicio_vigencia"]; !existe {
			return ErrConfiguracionIncompleta
		}
	}
	var objetivo map[string]json.RawMessage
	if json.Unmarshal(campos["objetivo"], &objetivo) != nil || objetivo == nil {
		return ErrConfiguracionIncompleta
	}
	if raw, existe := objetivo["vigente_desde"]; existe && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return ErrConfiguracionIncompleta
	}
	if raw, existe := objetivo["vigente_hasta"]; existe && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return ErrConfiguracionIncompleta
	}
	type sinMetodo CambioPerfil
	var dato sinMetodo
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&dato) != nil || dec.Decode(new(any)) != io.EOF {
		return ErrConfiguracionIncompleta
	}
	_, desdePresente := objetivo["vigente_desde"]
	_, hastaPresente := objetivo["vigente_hasta"]
	_, modoPresente := campos["inicio_vigencia"]
	switch dato.Operacion {
	case "otorgar":
		if !modoPresente || !hastaPresente || bytes.Equal(bytes.TrimSpace(campos["inicio_vigencia"]), []byte("null")) {
			return ErrConfiguracionIncompleta
		}
		if dato.InicioVigencia == "inmediato" && desdePresente ||
			dato.InicioVigencia == "programado" && !desdePresente ||
			(dato.InicioVigencia != "inmediato" && dato.InicioVigencia != "programado") {
			return ErrConfiguracionIncompleta
		}
	case "revocar":
		if modoPresente || desdePresente || hastaPresente {
			return ErrConfiguracionIncompleta
		}
	default:
		return ErrConfiguracionIncompleta
	}
	*c = CambioPerfil(dato)
	return nil
}

func (c CambioPerfil) MarshalJSON() ([]byte, error) {
	type sinMetodo CambioPerfil
	b, err := json.Marshal(sinMetodo(c))
	if err != nil {
		return nil, err
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(b, &campos); err != nil {
		return nil, err
	}
	var objetivo map[string]json.RawMessage
	if err := json.Unmarshal(campos["objetivo"], &objetivo); err != nil {
		return nil, err
	}
	if c.Operacion == "revocar" {
		delete(campos, "inicio_vigencia")
		delete(objetivo, "vigente_desde")
		delete(objetivo, "vigente_hasta")
	} else if c.InicioVigencia == "inmediato" {
		delete(objetivo, "vigente_desde")
	}
	campos["objetivo"], err = json.Marshal(objetivo)
	if err != nil {
		return nil, err
	}
	return json.Marshal(campos)
}
