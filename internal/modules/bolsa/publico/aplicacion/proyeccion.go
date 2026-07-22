package aplicacion

import (
	"crypto/subtle"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	canonicopublico "vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/publico/dominio"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/puertos"
)

func proyectarResumen(c dominiobolsa.ResumenConvocatoria, indice indiceCatalogos, ahora time.Time) (ResumenConvocatoriaPublica, error) {
	if err := c.ValidarPublicacion(); err != nil {
		return ResumenConvocatoriaPublica{}, ErrDatosPublicosNoConfiables
	}
	d := c.DatosPublicos
	tipo, err := indice.resolver(puertosbolsa.CatalogoTiposConvocatoria, d.Tipo)
	if err != nil {
		return ResumenConvocatoriaPublica{}, err
	}
	estado, err := indice.resolver(puertosbolsa.CatalogoEstadosConvocatoria, string(c.Estado))
	if err != nil {
		return ResumenConvocatoriaPublica{}, err
	}
	categorias := make([]ReferenciaCategoriaPublica, 0, len(d.Categorias))
	for _, clave := range d.Categorias {
		valor, err := indice.resolverCategoria(d.CatalogoCategorias, clave)
		if err != nil {
			return ResumenConvocatoriaPublica{}, err
		}
		categorias = append(categorias, ReferenciaCategoriaPublica{Clave: valor.Clave, Version: valor.Version})
	}
	plazos, err := proyectarPlazos(d.Plazos, indice, ahora)
	if err != nil {
		return ResumenConvocatoriaPublica{}, err
	}
	return ResumenConvocatoriaPublica{
		IdentificadorPublico: d.IdentificadorPublico, Version: c.Version, HuellaSHA256: c.HuellaSHA256,
		Titulo: d.Titulo, Resumen: d.Resumen, Tipo: tipo, Estado: estado, Categorias: categorias,
		CatalogoCategorias: ReferenciaCatalogoCategoriasConvocatoriaPublica{
			Referencia:             d.CatalogoCategorias.CatalogoID,
			Version:                d.CatalogoCategorias.CatalogoVersion,
			HuellaSHA256:           d.CatalogoCategorias.CatalogoHuellaSHA256,
			HuellaProyeccionSHA256: d.CatalogoCategorias.CatalogoHuellaProyeccionSHA256,
		},
		PlazoDestacado: plazoDestacado(plazos), NumeroRequisitos: c.NumeroRequisitos,
		NumeroDocumentos: c.NumeroDocumentos, NumeroAyudas: c.NumeroAyudas,
		PublicadaEn: d.PublicadaEn, ActualizadaEn: d.ActualizadaEn,
	}, nil
}

func resumirDetalle(c dominiobolsa.Convocatoria) dominiobolsa.ResumenConvocatoria {
	resultado := dominiobolsa.ResumenConvocatoria{
		Version: c.Version, Estado: c.Estado, HuellaSHA256: c.HuellaSHA256,
	}
	if c.DatosPublicos == nil {
		return resultado
	}
	datos := c.DatosPublicos
	resultado.DatosPublicos = &dominiobolsa.DatosPublicosResumenConvocatoria{
		IdentificadorPublico: datos.IdentificadorPublico,
		Tipo:                 datos.Tipo,
		CatalogoCategorias:   datos.CatalogoCategorias,
		Categorias:           append([]string(nil), datos.Categorias...),
		Titulo:               datos.Titulo,
		Resumen:              datos.Resumen,
		PublicadaEn:          datos.PublicadaEn,
		ActualizadaEn:        datos.ActualizadaEn,
		Plazos:               append([]dominiobolsa.PlazoConvocatoria(nil), datos.Plazos...),
	}
	resultado.NumeroRequisitos = len(datos.Requisitos)
	resultado.NumeroDocumentos = len(datos.Documentos)
	resultado.NumeroAyudas = len(datos.Ayuda)
	return resultado
}

func huellasPublicasIguales(a, b string) bool {
	return len(a) == 64 && len(b) == 64 &&
		subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func huellaConvocatoriaPublica(c dominiobolsa.Convocatoria) (string, error) {
	return canonicopublico.HuellaConvocatoriaV2(c)
}

func proyectarPlazos(origen []dominiobolsa.PlazoConvocatoria, indice indiceCatalogos, ahora time.Time) ([]PlazoPublico, error) {
	resultado := make([]PlazoPublico, 0, len(origen))
	for _, p := range origen {
		tipo, err := indice.resolver(puertosbolsa.CatalogoTiposPlazo, p.Tipo)
		if err != nil {
			return nil, err
		}
		situacion, etiqueta, semantica := situacionPlazo(p, ahora)
		resultado = append(resultado, PlazoPublico{Referencia: p.Referencia, Tipo: tipo, Titulo: p.Titulo, Descripcion: p.Descripcion, AbreEn: p.AbreEn, CierraEn: p.CierraEn, Situacion: situacion, Etiqueta: etiqueta, Semantica: semantica})
	}
	sort.Slice(resultado, func(a, b int) bool { return resultado[a].AbreEn.Before(resultado[b].AbreEn) })
	return resultado, nil
}

func situacionPlazo(plazo dominiobolsa.PlazoConvocatoria, ahora time.Time) (string, string, string) {
	if ahora.Before(plazo.AbreEn) {
		return "proximo", "Próximo", "informacion"
	}
	// El instante de cierre forma parte del plazo; se cierra en el primer
	// microsegundo posterior, igual en memoria y en timestamptz(6).
	if !ahora.After(plazo.CierraEn) {
		return "abierto", "Abierto", "exito"
	}
	return "cerrado", "Cerrado", "neutro"
}

func plazoDestacado(plazos []PlazoPublico) *PlazoPublico {
	for indice := range plazos {
		if plazos[indice].Situacion == "abierto" {
			copia := plazos[indice]
			return &copia
		}
	}
	for indice := range plazos {
		if plazos[indice].Situacion == "proximo" {
			copia := plazos[indice]
			return &copia
		}
	}
	if len(plazos) > 0 {
		copia := plazos[len(plazos)-1]
		return &copia
	}
	return nil
}

func proyectarRequisitos(origen []dominiobolsa.RequisitoConvocatoria) []RequisitoPublico {
	resultado := make([]RequisitoPublico, 0, len(origen))
	for _, r := range origen {
		resultado = append(resultado, RequisitoPublico{Referencia: r.Referencia, Orden: r.Orden, Titulo: r.Titulo, Descripcion: r.Descripcion, Obligatorio: r.Obligatorio})
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Orden < resultado[j].Orden })
	return resultado
}

func proyectarDocumentos(origen []dominiobolsa.DocumentoConvocatoria, indice indiceCatalogos) ([]DocumentoPublico, error) {
	resultado := make([]DocumentoPublico, 0, len(origen))
	for _, d := range origen {
		tipo, err := indice.resolver(puertosbolsa.CatalogoTiposDocumento, d.Tipo)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, DocumentoPublico{Referencia: d.Referencia, Tipo: tipo, Orden: d.Orden, Titulo: d.Titulo, Descripcion: d.Descripcion, Formato: d.Formato, URL: d.URL, PublicadoEn: d.PublicadoEn})
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Orden < resultado[j].Orden })
	return resultado, nil
}

func proyectarAyuda(origen []dominiobolsa.AyudaConvocatoria, indice indiceCatalogos) ([]AyudaPublica, error) {
	resultado := make([]AyudaPublica, 0, len(origen))
	for _, a := range origen {
		categoria, err := indice.resolver(puertosbolsa.CatalogoCategoriasAyuda, a.Categoria)
		if err != nil {
			return nil, err
		}
		resultado = append(resultado, AyudaPublica{Referencia: a.Referencia, Categoria: categoria, Orden: a.Orden, Pregunta: a.Pregunta, Respuesta: a.Respuesta})
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Orden < resultado[j].Orden })
	return resultado, nil
}

func contieneCategoriaPublica(catalogo puertosbolsa.CatalogoCategoriasPublicas, clave string) bool {
	for _, categoria := range catalogo.Categorias {
		if categoria.Clave == clave {
			return true
		}
	}
	return false
}

func validarConteosCategorias(conteos map[string]puertosbolsa.ConteoCategoriaConvocatorias, indice indiceCatalogos) error {
	for clave, conteo := range conteos {
		entrada, existe := indice.porCatalogo[puertosbolsa.CatalogoCategoriasConvocatoria][clave]
		if !existe || !entrada.Publicable || conteo.NumeroConvocatorias < 1 || conteo.NumeroPlazosAbiertos < 0 {
			return ErrDatosPublicosNoConfiables
		}
	}
	return nil
}

func validarFuente(f puertosbolsa.MetadatosFuenteConvocatorias) error {
	if !patronFiltroCatalogo.MatchString(f.Revision) || f.ActualizadaEn.IsZero() || f.ActualizadaEn.Location() != time.UTC || f.ActualizadaEn.Nanosecond()%1000 != 0 ||
		(f.Demostracion && !textoPublicoCanonico(f.Aviso, 500, true)) || (!f.Demostracion && f.Aviso != "" && !textoPublicoCanonico(f.Aviso, 500, true)) {
		return ErrDatosPublicosNoConfiables
	}
	return nil
}

func validarFuenteCategorias(f puertosbolsa.MetadatosFuenteCategorias) error {
	if !patronFiltroCatalogo.MatchString(f.Revision) || f.ActualizadaEn.IsZero() ||
		f.ActualizadaEn.Location() != time.UTC || f.ActualizadaEn.Nanosecond()%1000 != 0 ||
		(f.Demostracion && !textoPublicoCanonico(f.Aviso, 500, true)) ||
		(!f.Demostracion && f.Aviso != "" && !textoPublicoCanonico(f.Aviso, 500, true)) {
		return ErrDatosPublicosNoConfiables
	}
	return nil
}

func fuentesCoinciden(
	categorias puertosbolsa.MetadatosFuenteCategorias,
	convocatorias puertosbolsa.MetadatosFuenteConvocatorias,
) bool {
	if categorias.Demostracion && convocatorias.Demostracion {
		return true
	}
	return categorias.Revision == convocatorias.Revision &&
		categorias.ActualizadaEn.Equal(convocatorias.ActualizadaEn) &&
		categorias.Demostracion == convocatorias.Demostracion &&
		categorias.Aviso == convocatorias.Aviso
}

func fuentesCategoriasCoinciden(a, b puertosbolsa.MetadatosFuenteCategorias) bool {
	return a.Revision == b.Revision && a.ActualizadaEn.Equal(b.ActualizadaEn) &&
		a.Demostracion == b.Demostracion && a.Aviso == b.Aviso
}

func textoPublicoCanonico(valor string, maximo int, multilinea bool) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len([]rune(valor)) > maximo {
		return false
	}
	for _, caracter := range valor {
		if unicode.Is(unicode.Cf, caracter) ||
			(unicode.IsControl(caracter) && (!multilinea || (caracter != '\n' && caracter != '\r' && caracter != '\t'))) {
			return false
		}
	}
	return true
}

func proyectarFuente(f puertosbolsa.MetadatosFuenteConvocatorias) FuentePublica {
	return FuentePublica{Revision: f.Revision, ActualizadaEn: f.ActualizadaEn, Demostracion: f.Demostracion, Aviso: f.Aviso}
}

func proyectarFuenteCategorias(f puertosbolsa.MetadatosFuenteCategorias) FuentePublica {
	return FuentePublica{Revision: f.Revision, ActualizadaEn: f.ActualizadaEn, Demostracion: f.Demostracion, Aviso: f.Aviso}
}

func instanteCanonico(instante time.Time) (time.Time, error) {
	if instante.IsZero() {
		return time.Time{}, fmt.Errorf("%w: reloj sin instante", ErrServicioConsultaPublicaInvalido)
	}
	return instante.UTC().Truncate(time.Microsecond), nil
}
