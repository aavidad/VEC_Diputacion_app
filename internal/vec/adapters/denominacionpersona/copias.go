package denominacionpersona

import "vec-diputacion-granada/internal/vec/ports"

func copiarMapa(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	c := make(map[string]string, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

// Sólo se usa después de validar la evidencia V2. El contexto privado de la
// invocación conserva los bytes originales aunque otro puerto use sus copias.
func clonarAcceso(a ports.AccesoDenominacionPersona) ports.AccesoDenominacionPersona {
	a.ResultadoContexto, _ = a.ResultadoContexto.Clonar()
	a.Contexto = a.ResultadoContexto.Contexto
	a.Recurso.Ambitos = copiarMapa(a.Recurso.Ambitos)
	a.Recurso.Atributos = copiarMapa(a.Recurso.Atributos)
	a.Auditoria.Metadata = copiarMapa(a.Auditoria.Metadata)
	a.Auditoria.ActorRoles = append([]string(nil), a.Auditoria.ActorRoles...)
	return a
}
