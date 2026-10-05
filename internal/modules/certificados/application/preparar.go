package application

import (
	"context"
	"reflect"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/certificados/domain"
	"vec-diputacion-granada/internal/modules/certificados/ports"
)

type Preparador struct {
	Fuente       ports.FuenteServicios
	Catalogo     ports.CatalogoPlantillas
	Renderizador ports.Renderizador
}
type Orden struct {
	PlantillaID string
	Version     int
	Idioma      string
}
type Resultado struct {
	Borrador domain.Borrador
	PDF      []byte
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice:
		return r.IsNil()
	}
	return false
}

// PrepararEnsayo prepares review material only. It performs no issuance,
// recognition, authorisation, signature, registration or notification.
func (p *Preparador) PrepararEnsayo(ctx context.Context, o Orden) (Resultado, error) {
	var vacio Resultado
	if p == nil || ctx == nil || nulo(p.Fuente) || nulo(p.Catalogo) || nulo(p.Renderizador) {
		return vacio, domain.ErrNoDisponible
	}
	if e := ctx.Err(); e != nil {
		return vacio, e
	}
	if o.PlantillaID != "servicios" || o.Version < 1 || !domain.IdiomaValido(o.Idioma) {
		return vacio, domain.ErrEntrada
	}
	f, e := p.Fuente.Obtener(ctx)
	if e != nil {
		return vacio, domain.ErrEntrada
	}
	if f.ValidarEnsayo() != nil {
		return vacio, domain.ErrEntrada
	}
	plantilla, textos, e := p.Catalogo.Obtener(ctx, o.PlantillaID, o.Version, o.Idioma)
	if e != nil || plantilla.Validar(o.PlantillaID, o.Version) != nil || textos.Validar() != nil || textos.Idioma != o.Idioma {
		return vacio, domain.ErrCatalogo
	}
	f.Servicios = append([]domain.Servicio{}, f.Servicios...)
	for i := range f.Servicios {
		if d := f.Servicios[i].Dias; d != nil {
			copia := *d
			f.Servicios[i].Dias = &copia
		}
	}
	b := domain.Borrador{Esquema: "vec.certificados.borrador-servicios.v1", Estado: "borrador", Modo: "ensayo_sintetico", Idioma: o.Idioma, Fuente: f}
	b.FuenteHuellaSHA256, e = domain.Huella(f)
	if e != nil {
		return vacio, e
	}
	h, e := domain.Huella(plantilla)
	if e != nil {
		return vacio, e
	}
	b.Plantilla = domain.VersionPlantilla{ID: plantilla.ID, Version: plantilla.Version, Estado: plantilla.Estado, HuellaSHA256: h}
	b.TextosHuellaSHA256, e = domain.Huella(textos)
	if e != nil {
		return vacio, e
	}
	b.Contenido.Titulo = textos.Mensaje("titulo")
	fecha := func(s string) string {
		t, _ := time.Parse("2006-01-02", s)
		return t.Format(textos.FormatoFecha)
	}
	instante, _ := time.Parse(time.RFC3339Nano, f.Corte.ConocidoEn)
	for _, k := range plantilla.Bloques {
		if k == "criterio" && f.ConFormaPersonalV1() {
			k = "criterio_v1" // la fuente V1 no trae días: el criterio de transcribirlos no aplica
		}
		b.Contenido.Parrafos = append(b.Contenido.Parrafos, textos.Mensaje(k,
			"nombre", f.Nombre, "vigente", fecha(f.Corte.VigenteEn), "conocido", instante.Format(textos.FormatoInstante),
			"fuente", f.ProcedenciaRef, "fuente_huella", b.FuenteHuellaSHA256,
			"plantilla", plantilla.ID, "version", strconv.Itoa(plantilla.Version), "plantilla_huella", h))
	}
	v1 := f.ConFormaPersonalV1()
	if v1 {
		b.Contenido.Parrafos = append(b.Contenido.Parrafos, textos.Mensaje("cobertura_"+f.Cobertura))
	}
	for _, estado := range []string{"declarado", "comprobado", "reconocido"} {
		g := domain.Grupo{Estado: estado, Servicios: []domain.Servicio{}}
		b.Contenido.Parrafos = append(b.Contenido.Parrafos, textos.Mensaje(estado))
		for _, s := range f.Servicios {
			if s.Estado != estado {
				continue
			}
			g.Servicios = append(g.Servicios, s)
			b.Contenido.Parrafos = append(b.Contenido.Parrafos, parrafoServicio(textos, fecha, s, v1))
		}
		if len(g.Servicios) == 0 {
			b.Contenido.Parrafos = append(b.Contenido.Parrafos, textos.Mensaje("vacio"))
		}
		b.Grupos = append(b.Grupos, g)
	}
	if len(f.Servicios) == 0 {
		b.Contenido.Parrafos = append(b.Contenido.Parrafos, textos.Mensaje("sin_servicios"))
	}
	b.Contenido.Parrafos = append(b.Contenido.Parrafos, textos.Mensaje("revision"))
	if e = ctx.Err(); e != nil {
		return vacio, e
	}
	pdf, e := p.Renderizador.Renderizar(ctx, b.Contenido)
	if e != nil || len(pdf) < 5 || len(pdf) > 16<<20 || string(pdf[:5]) != "%PDF-" {
		return vacio, domain.ErrNoDisponible
	}
	if e = ctx.Err(); e != nil {
		return vacio, e
	}
	return Resultado{Borrador: b, PDF: pdf}, nil
}

// parrafoServicio presenta un periodo. Con la forma V1 añade acto, certeza y si
// puede sustentar un certificado; nunca muestra días que la fuente no trae.
func parrafoServicio(textos domain.Textos, fecha func(string) string, s domain.Servicio, v1 bool) string {
	if !v1 {
		return textos.Mensaje("servicio", "inicio", fecha(s.Inicio), "fin", fecha(s.Fin), "clase", s.Clase, "dias", strconv.FormatInt(*s.Dias, 10))
	}
	plantillaTexto, fin := "servicio_v1_abierto", ""
	if s.Fin != "" {
		plantillaTexto, fin = "servicio_v1", fecha(s.Fin)
	}
	if s.EnCursoAlCorte {
		plantillaTexto = "servicio_v1_al_corte"
	}
	sustenta := "sustenta_no"
	if s.SustentaCertificacion() {
		sustenta = "sustenta_si"
	}
	return textos.Mensaje(plantillaTexto, "inicio", fecha(s.Inicio), "fin", fin, "clase", s.Clase,
		"clase_version", strconv.FormatInt(s.ClaseVersion, 10), "acto", s.ActoRef,
		"certeza", textos.Mensaje("certeza_"+s.Certeza), "sustenta", textos.Mensaje(sustenta))
}
