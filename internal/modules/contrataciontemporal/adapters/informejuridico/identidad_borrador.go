package informejuridico

import (
	"context"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var documentoIdentidadBorrador = regexp.MustCompile(`^\*{3}[0-9]{4}\*{2}$`)

func plantillaUsaIdentidad(p *PlantillasBorrador, tipo ports.TipoBorradorRRHH) bool {
	plantilla, ok := p.Plantilla(tipo)
	if !ok {
		return false
	}
	campos := map[string]bool{}
	for _, texto := range append([]string{plantilla.Titulo}, plantilla.Parrafos...) {
		for _, marca := range marcaPlantilla.FindAllStringSubmatch(texto, -1) {
			campos[marca[2]] = true
		}
	}
	return campos["persona_nombre"] && campos["persona_apellidos"] && campos["persona_documento_enmascarado"]
}

func identidadBorrador(ctx context.Context, tipo ports.TipoBorradorRRHH,
	d ports.DetalleExpedienteRRHH, fuente ports.FuenteIdentidadDocumentoRRHH,
	p *PlantillasBorrador, exigida bool,
) (*ports.IdentidadDocumentoRRHH, error) {
	if tipo == ports.BorradorInformeDefinitivo {
		return nil, nil
	}
	usaIdentidad := plantillaUsaIdentidad(p, tipo)
	if !usaIdentidad || fuente == nil {
		if exigida {
			return nil, ports.ErrBorradorRRHHNoDisponible
		}
		return nil, nil
	}
	i, err := fuente(ctx, d.Resumen.ExpedienteRef)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		return nil, err
	}
	if i == nil && !exigida {
		return nil, nil
	}
	if i == nil || i.VersionFuente == 0 || !textoIdentidadBorradorValido(i.Nombre) ||
		!textoIdentidadBorradorValido(i.Apellidos) || !documentoIdentidadBorrador.MatchString(i.DocumentoEnmascarado) {
		return nil, ports.ErrBorradorRRHHNoDisponible
	}
	copia := *i
	return &copia, nil
}

func textoIdentidadBorradorValido(s string) bool {
	return s != "" && s == strings.TrimSpace(s) && len(s) <= 256 && utf8.ValidString(s) &&
		strings.IndexFunc(s, unicode.IsControl) < 0
}
