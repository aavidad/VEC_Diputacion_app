package application

import (
	"context"

	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// SolicitudGuardadoBases contiene una intención preparada sin identidad ni
// autorización. El servidor conserva la validación CAS y el guardado durable.
type SolicitudGuardadoBases struct {
	Esperada       prep.Esperada
	Material       prep.Material
	ClaveOperacion string
}

// PrepararSolicitudGuardadoBases traduce el material local al contrato S2.
// La revisión local nunca sustituye la preimagen aportada por el operador.
func PrepararSolicitudGuardadoBases(ctx context.Context, material ports.MaterialBasesPropuesto,
	esperada prep.Esperada, clave string, canonizador ports.CanonizadorBases,
) (SolicitudGuardadoBases, error) {
	var cero SolicitudGuardadoBases
	// Mismos límites del guardado vigente: HuellaIntencion impide agotar la
	// revisión y el material V3/broker limita clave y recurso a 128 bytes.
	if esperada.Validar(true) != nil || esperada.Revision == 1_000_000 || len(esperada.PreparacionRef) > 128 ||
		!prep.IdentificadorValido(clave) || len(clave) > 128 {
		return cero, ports.ErrMaterialBasesInvalido
	}
	preparacion, err := PrepararMaterialBases(ctx, material, canonizador)
	if err != nil {
		return cero, err
	}
	m := preparacion.MaterialPropuesto
	r := m.Referencias
	propuesta := prep.Material{Contenido: m.Contenido, Referencias: []prep.ReferenciaPropuesta{
		{Campo: "fuente_bases", Referencia: r.FuenteBases}, {Campo: "catalogos", Referencia: r.Catalogos},
		{Campo: "calendario", Referencia: r.Calendario}, {Campo: "reglas_baremacion", Referencia: r.ReglasBaremacion},
		{Campo: "flujo_proceso", Referencia: r.FlujoProceso}, {Campo: "flujo_solicitud", Referencia: r.FlujoSolicitud},
		{Campo: "plantilla", Referencia: r.Plantilla}, {Campo: "plaza", Referencia: r.Plaza},
		{Campo: "oep", Referencia: r.OEP}, {Campo: "rpt", Referencia: r.RPT},
	}}
	canonico, err := propuesta.Canonico()
	if err != nil {
		return cero, ports.ErrMaterialBasesInvalido
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return SolicitudGuardadoBases{Esperada: esperada, Material: canonico, ClaveOperacion: clave}, nil
}
