package lecturaincorporacion

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	alta "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// No reconstruye ni refecha el alta. La implementación inyectada debe demostrar
// origen y commit: un JSON autoconsistente no es acreditación de persistencia.
func resultadoParaOrden(r Resultado, o Orden, ahora time.Time) bool {
	return resultadoParaLectura(r, o.Selector(), o.evaluadaEn, o.material.preparadoEn, o.Exportacion(), ahora)
}

// Un único validador de resultado/canon para ambos protocolos de lectura.
func resultadoParaLectura(r Resultado, s Selector, evaluadaEn, preparadoEn time.Time, exportacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, ahora time.Time) bool {
	x := exportacion.ResumenCapacidad()
	if !resultadoValido(r, s, ahora) || r.DecisionLecturaRef != x.DecisionRef() ||
		r.LeidaEn.Before(evaluadaEn) || r.LeidaEn.Before(x.EmitidaEn()) || !r.LeidaEn.Before(x.ExpiraEn()) ||
		r.Registro.RegistradoEn.After(preparadoEn) || r.Registro.DecisionOriginalRef == r.DecisionLecturaRef {
		return false
	}
	// El único serializador del material original es la API propietaria de alta.
	// El recanon exacto cierra duplicados, alias, null, campos extra y whitespace.
	var original struct {
		Esquema  string
		Material alta.MaterialAlta
	}
	d := json.NewDecoder(bytes.NewReader(r.Registro.MaterialCanonico))
	d.DisallowUnknownFields()
	if d.Decode(&original) != nil || d.Decode(new(any)) != io.EOF || original.Esquema != alta.EsquemaMaterialAlta ||
		original.Material.Validar() != nil || original.Material.OrganizacionRef != s.OrganizacionRef || original.Material.Preparacion.Solicitud != r.Registro.Solicitud {
		return false
	}
	canon, err := json.Marshal(original)
	h, eh := original.Material.HuellaSHA256()
	return err == nil && eh == nil && h == s.MaterialSHA256 && bytes.Equal(canon, r.Registro.MaterialCanonico)
}
