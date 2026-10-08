package composicion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

func TestExportacionServiciosFormatosUsanCatalogoExactoYSinIdiomaPredeterminado(t *testing.T) {
	b := []byte(`{"formato":{"referencia":"personal:servicios_propios:csv","version":"1","nombre_archivo":"servicios.csv"},"csv":{"fecha_inicio":"c1","fecha_fin":"c2","clase":"c3","dias":"c4","estado":"c5"},"general":{},"estados":{"declarado":"e1","comprobado":"e2","reconocido":"e3"}}`)
	p, e := NuevoProveedorFormatosExportacionServiciosPropios(map[string][]byte{"xx": b})
	if e != nil {
		t.Fatal(e)
	}
	f, e := p.FormatoParaIdioma(context.Background(), "xx")
	h := sha256.Sum256(b)
	if e != nil || f.Datos().CatalogoSHA256 != hex.EncodeToString(h[:]) || f.Datos().Cabeceras[2] != "c3" {
		t.Fatal("formato sin catálogo", e)
	}
	if _, e := p.FormatoParaIdioma(context.Background(), "yy"); !errors.Is(e, domain.ErrExportacionServiciosPropiosInvalida) {
		t.Fatal("idioma inventado")
	}
}
func TestExportacionServiciosIntentoRegistraAccionOriginalNuevaYRecuperaMismaOrden(t *testing.T) {
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC), "")
	ctx, resolver := contextoFichaCapturadaPrueba(t, id)
	d := &destinoIntentosFichaPrueba{err: errors.New("commit ambiguo")}
	c := ConfiguracionIntentosExportacionServiciosPropios(configuracionIntentosFichaPrueba())
	r, e := NuevoRegistroIntentosExportacionServiciosPropios(d, c)
	if e != nil {
		t.Fatal(e)
	}
	if e = r.RegistrarIntentoExportacionServiciosPropios(ctx, ports.IntentoFichaPropia{Motivo: "denegado"}); !errors.Is(e, domain.ErrExportacionServiciosPropiosNoDisponible) || len(d.ordenes) != 2 {
		t.Fatal(e)
	}
	d.err = nil
	if e = r.RegistrarIntentoExportacionServiciosPropios(ctx, ports.IntentoFichaPropia{Motivo: "denegado"}); e != nil {
		t.Fatal(e)
	}
	o, _ := d.ordenes[0].Datos()
	ultimo, _ := d.ordenes[2].Datos()
	if o.Datos.Accion != domain.AccionExportacionServiciosPropios || o.Datos.FinalidadRef != domain.FinalidadExportacionServiciosPropios || o.IntentoRef != ultimo.IntentoRef || o.ResultadoContexto.HuellaSHA256 != id.Resultado.HuellaSHA256 || resolver.llamadas != 1 {
		t.Fatal("actor/operación sustituidos")
	}
}
func TestExportacionServiciosProveedorNoReutilizaCorrelacionDeConsulta(t *testing.T) {
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC), "")
	ctx, resolver := contextoFichaCapturadaPrueba(t, id)
	f, e := domain.NuevoFormatoExportacionServiciosPropios(domain.DatosFormatoExportacionServiciosPropios{Referencia: "personal:servicios_propios:csv", Version: 1, Idioma: "xx", CatalogoSHA256: configuracionIntentosFichaPrueba().MotivoDenegado.CatalogoHuellaSHA256, NombreArchivo: "servicios.csv", Cabeceras: []string{"c1", "c2", "c3", "c4", "c5"}, Estados: map[string]string{"declarado": "e1", "comprobado": "e2", "reconocido": "e3"}})
	if e != nil {
		t.Fatal(e)
	}
	m, e := domain.NuevoMaterialExportacionServiciosPropios(domain.SolicitudExportacionServiciosPropios{Actor: id.Resultado.Contexto, Corte: domain.CorteEmpleadoB2{VigenteEn: "2026-10-03", ConocidoEn: time.Date(2026, 10, 3, 11, 59, 59, 0, time.UTC)}, ReciboRef: "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100", Idioma: "xx"}, f)
	if e != nil {
		t.Fatal(e)
	}
	emisor := &emisorFichaCapturadaPrueba{}
	p, e := NuevoProveedorAutorizacionExportacionServiciosPropios(resolver, emisor, configuracionIntentosFichaPrueba().MotivoDenegado)
	if e != nil {
		t.Fatal(e)
	}
	_, e = p.AutorizarExportacionServiciosPropios(ctx, m)
	if !errors.Is(e, domain.ErrExportacionServiciosPropiosDenegada) || emisor.llamadas != 1 || resolver.llamadas != 1 {
		t.Fatal(e)
	}
	datos, e := emisor.solicitud.Datos()
	if e != nil || datos.Accion != domain.AccionExportacionServiciosPropios || datos.Finalidad != domain.FinalidadExportacionServiciosPropios {
		t.Fatal("permiso de consulta reaprovechado", e)
	}
}

type exportadorCapturaPrueba struct {
	llamadas  int
	identidad IdentidadRegistradaFichaPropia
}

func (e *exportadorCapturaPrueba) Exportar(ctx context.Context, _ domain.SolicitudExportacionServiciosPropios) (ports.ResultadoExportacionServiciosPropios, error) {
	e.llamadas++
	e.identidad, _ = IdentidadOriginalFichaPropia(ctx)
	return ports.ResultadoExportacionServiciosPropios{}, nil
}
func TestExportacionServiciosWrapperReutilizaCapturaHTTPYNoInventaContexto(t *testing.T) {
	id := identidadIntentoFichaPropiaVigenciaPrueba(t, time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC), "")
	ctx, resolver := contextoFichaCapturadaPrueba(t, id)
	exportador := &exportadorCapturaPrueba{}
	wrapper, e := NuevoExportadorServiciosPropiosConIdentidad(exportador, resolver, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = wrapper.Exportar(ctx, domain.SolicitudExportacionServiciosPropios{}); e != nil || resolver.llamadas != 1 || exportador.llamadas != 1 || exportador.identidad.Resultado.HuellaSHA256 != id.Resultado.HuellaSHA256 {
		t.Fatal("captura sustituida", e)
	}
	if _, e = wrapper.Exportar(context.Background(), domain.SolicitudExportacionServiciosPropios{}); !errors.Is(e, domain.ErrExportacionServiciosPropiosNoDisponible) || exportador.llamadas != 1 {
		t.Fatal("generó contexto ficticio", e)
	}
}
