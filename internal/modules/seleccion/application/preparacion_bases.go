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
	resultado := ports.PreparacionBases{Estado: "pendiente", MaterialPropuesto: material}
	pendiente := func(campo, codigo string) {
		resultado.Pendientes = append(resultado.Pendientes, ports.PendientePreparacionBases{Campo: campo, Codigo: codigo})
	}
	r := material.Referencias
	for _, campo := range []struct {
		clave string
		ref   bolsa.ReferenciaConfiguracionConvocatoria
	}{
		{"fuente_bases", r.FuenteBases}, {"catalogos", r.Catalogos}, {"calendario", r.Calendario},
		{"reglas_baremacion", r.ReglasBaremacion}, {"flujo_proceso", r.FlujoProceso},
		{"flujo_solicitud", r.FlujoSolicitud}, {"plantilla", r.Plantilla}, {"plaza", r.Plaza}, {"oep", r.OEP}, {"rpt", r.RPT},
	} {
		switch {
		case campo.ref == (bolsa.ReferenciaConfiguracionConvocatoria{}):
			pendiente(campo.clave, "referencia_ausente")
		case canonizador.ComprobarReferencia(campo.ref) != nil:
			pendiente(campo.clave, "referencia_invalida")
		default:
			pendiente(campo.clave, "referencia_no_verificada")
		}
	}
	c := material.Contenido
	for _, campo := range []struct {
		clave   string
		ausente bool
	}{
		{"identificador_publico", c.IdentificadorPublico == ""}, {"tipo", c.Tipo == ""},
		{"titulo", c.Titulo == ""}, {"resumen", c.Resumen == ""},
		{"catalogo_categorias", c.CatalogoCategorias == (bolsa.ReferenciaCatalogoCategorias{})},
		{"categorias", len(c.Categorias) == 0}, {"plazos", len(c.Plazos) == 0}, {"documentos_propuestos", len(c.Documentos) == 0},
	} {
		if campo.ausente {
			pendiente(campo.clave, "material_ausente")
		}
	}
	canonico, err := canonizador.CanonizarContenido(ctx, copiarContenidoPropuesto(c))
	if cancelacion := ctx.Err(); cancelacion != nil {
		return vacio, cancelacion
	}
	if errors.Is(err, bolsa.ErrVersionConvocatoriaGobernadaInvalida) {
		pendiente("contenido", "contenido_no_validado")
	} else if err != nil {
		return vacio, ports.ErrPreparadorBasesNoDisponible
	} else {
		resultado.ContenidoCanonicoBolsa = &canonico
	}
	// Este corte no tiene autoridades que puedan acreditar estas dependencias.
	for _, campo := range []string{"documentos_admitidos", "firma_y_custodia", "acto_aprobacion", "publicacion_oficial"} {
		pendiente(campo, "circuito_pendiente")
	}
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
