package application

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// PrepararMaterialBases conserva una propuesta sin crear un borrador gobernado.
// La forma canónica no acredita fuente, vigencia, documentos, firma ni permisos.
func PrepararMaterialBases(ctx context.Context, material ports.MaterialBasesPropuesto, canonizador ports.CanonizadorBases) (ports.PreparacionBases, error) {
	var vacio ports.PreparacionBases
	if ctx == nil || canonizador == nil {
		return vacio, ports.ErrPreparadorBasesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if material.Alcance != "preparacion_sintetica" || !identidadMaterialValida(material.IdentidadMaterial) || material.VersionMaterial < 1 || material.VersionMaterial > 1_000_000 {
		return vacio, ports.ErrMaterialBasesInvalido
	}
	material.Contenido = copiarContenidoPropuesto(material.Contenido)
	evaluacion, err := canonizador.EvaluarMaterialBases(ctx, material)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return vacio, cancelacion
	}
	if errors.Is(err, ports.ErrMaterialBasesInvalido) {
		return vacio, ports.ErrMaterialBasesInvalido
	}
	if err != nil {
		return vacio, ports.ErrPreparadorBasesNoDisponible
	}
	resultado := ports.PreparacionBases{Estado: "pendiente", MaterialPropuesto: material,
		ContenidoCanonicoBolsa: evaluacion.ContenidoCanonicoBolsa, Pendientes: evaluacion.Pendientes}
	return resultado, nil
}

func identidadMaterialValida(s string) bool {
	if s == "" || len(s) > 180 || strings.TrimSpace(s) != s || !utf8.ValidString(s) {
		return false
	}
	return strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789:_./-") == ""
}

func copiarContenidoPropuesto(c bolsa.ContenidoPublicableConvocatoria) bolsa.ContenidoPublicableConvocatoria {
	c.Categorias = append([]string(nil), c.Categorias...)
	c.Plazos = append([]bolsa.PlazoConvocatoria(nil), c.Plazos...)
	c.Requisitos = append([]bolsa.RequisitoConvocatoria(nil), c.Requisitos...)
	c.Documentos = append([]bolsa.DocumentoPublicableConvocatoria(nil), c.Documentos...)
	c.Ayuda = append([]bolsa.AyudaConvocatoria(nil), c.Ayuda...)
	return c
}
