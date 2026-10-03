package main

import (
	"context"
	"encoding/json"
	"io"

	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/seleccion/adapters/bolsa"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/i18n"
)

// Los punteros distinguen una preimagen de alta explícita de un campo ausente.
type entradaSolicitudS2 struct {
	MaterialPropuesto *ports.MaterialBasesPropuesto `json:"material_propuesto"`
	Esperada          *preimagenS2JSON              `json:"esperada"`
	ClaveOperacion    *string                       `json:"clave_operacion"`
}

type preimagenS2JSON struct {
	PreparacionRef       *string `json:"preparacion_ref"`
	Revision             *int    `json:"revision"`
	HuellaMaterialSHA256 *string `json:"huella_material_sha256"`
}

type solicitudS2JSON struct {
	Esperada       prep.Esperada `json:"esperada"`
	Material       prep.Material `json:"material"`
	ClaveOperacion string        `json:"clave_operacion"`
}

func ejecutarSolicitudS2(ctx context.Context, entrada io.Reader, salida, errores io.Writer, catalogo *i18n.Catalog, idioma string) int {
	var propuesta entradaSolicitudS2
	if leerJSON(entrada, &propuesta) != nil || propuesta.MaterialPropuesto == nil || propuesta.Esperada == nil || propuesta.ClaveOperacion == nil {
		return informarError(errores, catalogo, idioma, errEntradaJSON.Error())
	}
	p := propuesta.Esperada
	if p.PreparacionRef == nil || p.Revision == nil || p.HuellaMaterialSHA256 == nil {
		return informarError(errores, catalogo, idioma, errEntradaJSON.Error())
	}
	esperada := prep.Esperada{PreparacionRef: *p.PreparacionRef, Revision: *p.Revision, HuellaMaterialSHA256: *p.HuellaMaterialSHA256}
	s, err := application.PrepararSolicitudGuardadoBases(ctx, *propuesta.MaterialPropuesto, esperada, *propuesta.ClaveOperacion, bolsa.CanonizadorBases{})
	if err != nil {
		return informarError(errores, catalogo, idioma, err.Error())
	}
	enc := json.NewEncoder(salida)
	enc.SetIndent("", "  ")
	if enc.Encode(solicitudS2JSON{s.Esperada, s.Material, s.ClaveOperacion}) != nil {
		return informarError(errores, catalogo, idioma, "seleccion.preparacion.salida_no_disponible")
	}
	return 0
}
