package composicion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	observabilidad "vec-diputacion-granada/internal/vec/adapters/observabilidad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func configuracionResultadosLectorRPTPrueba() ConfiguracionResultadosTecnicosLectorRPT {
	return ConfiguracionResultadosTecnicosLectorRPT{Componente: vecdomain.ComponenteIncidenciaComposicion, Etapa: vecdomain.EtapaIncidenciaConsulta}
}

func nuevoEmisorTecnicoLectorRPTPrueba(t *testing.T, destino io.Writer) *observabilidad.EmisorJSONLines {
	t.Helper()
	e, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino: destino, Entorno: "pruebas", VersionBinario: "3d109adec", Reloj: func() time.Time { return time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cerrarEmisorTecnicoLectorRPTPrueba(t, e) })
	return e
}
func emisorTecnicoLectorRPTPrueba(t *testing.T) *observabilidad.EmisorJSONLines {
	return nuevoEmisorTecnicoLectorRPTPrueba(t, io.Discard)
}
func cerrarEmisorTecnicoLectorRPTPrueba(t *testing.T, e *observabilidad.EmisorJSONLines) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := e.Cerrar(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestLectorRPTResultadosTecnicosRequierenEmisorYCatalogoCerrado(t *testing.T) {
	l := lectorSeleccionadaCorrelacionPrueba(func(context.Context) (personalports.ResultadoRelacionParaRPTV1, error) {
		return personalports.ResultadoRelacionParaRPTV1{}, nil
	})
	i := identidadLectorRelacionRPTPrueba{}
	var nulo *observabilidad.EmisorJSONLines
	for _, e := range []vecports.EmisorResultadosTecnicosConContexto{nil, nulo} {
		if _, err := NuevoLectorRelacionSeleccionadaRPTConIdentidad(l, i, time.Second, e, configuracionResultadosLectorRPTPrueba()); !errors.Is(err, personaldomain.ErrLectorRelacionRPTNoDisponible) {
			t.Fatal("emisor ausente admitido", err)
		}
	}
	for _, c := range []ConfiguracionResultadosTecnicosLectorRPT{{}, {Componente: "personal:secreto", Etapa: vecdomain.EtapaIncidenciaConsulta}, {Componente: vecdomain.ComponenteIncidenciaComposicion, Etapa: "consulta libre"}} {
		if _, err := NuevoLectorRelacionSeleccionadaRPTConIdentidad(l, i, time.Second, emisorTecnicoLectorRPTPrueba(t), c); !errors.Is(err, personaldomain.ErrLectorRelacionRPTNoDisponible) {
			t.Fatal("catálogo no cerrado", err)
		}
	}
}

func TestLectorRPTResultadosTecnicosPipelineComunPorCadaRespuesta(t *testing.T) {
	id := identidadIntentoRPTVigenciaPrueba(t, time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), "")
	for _, caso := range []struct {
		codigo vecdomain.CodigoResultadoTecnico
		err    error
	}{
		{vecdomain.ResultadoTecnicoCorrecto, nil},
		{vecdomain.ResultadoTecnicoDenegado, personaldomain.ErrLectorRelacionRPTDenegado},
		{vecdomain.ResultadoTecnicoEntradaInvalida, personaldomain.ErrLectorRelacionRPTInvalido},
		{vecdomain.ResultadoTecnicoCancelado, context.Canceled},
		{vecdomain.ResultadoTecnicoCancelado, context.DeadlineExceeded},
		{vecdomain.ResultadoTecnicoNoDisponible, personaldomain.ErrLectorRelacionRPTNoDisponible},
		{vecdomain.ResultadoTecnicoNoDisponible, errors.New("error privado de persona y recurso")},
	} {
		t.Run(string(caso.codigo)+":"+func() string {
			if caso.err == context.DeadlineExceeded {
				return "plazo"
			}
			return "respuesta"
		}(), func(t *testing.T) {
			var b bytes.Buffer
			e := nuevoEmisorTecnicoLectorRPTPrueba(t, &b)
			llamadas := 0
			siguiente := lectorSeleccionadaCorrelacionPrueba(func(context.Context) (personalports.ResultadoRelacionParaRPTV1, error) {
				llamadas++
				return personalports.ResultadoRelacionParaRPTV1{}, caso.err
			})
			l, err := NuevoLectorRelacionSeleccionadaRPTConIdentidad(siguiente, &resolutorIntentoRPTVigenciaPrueba{identidad: id}, time.Second, e, configuracionResultadosLectorRPTPrueba())
			if err != nil {
				t.Fatal(err)
			}
			ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			ctx, marcada := vecports.ConMarcaIncidenciasPeticion(ctx)
			if caso.codigo == vecdomain.ResultadoTecnicoCancelado {
				cancelado, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelado
			}
			correlacion, _ := vecports.CorrelacionIncidenciasPeticion(ctx)
			_, err = l.ConsultarSeleccionada(ctx, id.Resultado.Contexto, personaldomain.PreparacionRelacionParaRPT{}, "organismo:privado", "rel_"+strings.Repeat("r", 24))
			if err != caso.err || llamadas != 1 || marcada() {
				t.Fatal("técnico cambió respuesta o marca HTTP", err, llamadas, marcada())
			}
			cerrarEmisorTecnicoLectorRPTPrueba(t, e)
			var linea map[string]any
			if json.Unmarshal(bytes.TrimSpace(b.Bytes()), &linea) != nil || len(linea) != 10 || linea["esquema"] != vecdomain.EsquemaResultadoTecnico || linea["resultado"] != string(caso.codigo) || linea["correlacion"] != correlacion || linea["correlacion_ref"] != "correlacion_"+correlacion {
				t.Fatal("pipeline sin resultado ligado")
			}
			if strings.Contains(b.String(), "privado") || strings.Contains(b.String(), "per_") || strings.Contains(b.String(), "rel_") || linea["mensaje"] != nil || linea["codigo"] != nil {
				t.Fatal("PII o incidencia en resultado")
			}
			if m := e.MetricasResultadosTecnicos(); m.Aceptados != 1 || m.Escritos != 1 || m.Descartados != 0 || m.FallosEscritura != 0 {
				t.Fatal("no emitió exactamente un resultado", m)
			}
		})
	}
}

type destinoTecnicoLectorRPTFallido struct{}

func (destinoTecnicoLectorRPTFallido) Write([]byte) (int, error) {
	return 0, errors.New("destino técnico privado")
}

func TestLectorRPTFalloDestinoTecnicoNoRepiteConsultaNiInvalidaRecibo(t *testing.T) {
	for _, cerrado := range []bool{false, true} {
		t.Run(map[bool]string{false: "sink", true: "cola_cerrada"}[cerrado], func(t *testing.T) {
			id := identidadIntentoRPTVigenciaPrueba(t, time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), "")
			e := nuevoEmisorTecnicoLectorRPTPrueba(t, destinoTecnicoLectorRPTFallido{})
			if cerrado {
				cerrarEmisorTecnicoLectorRPTPrueba(t, e)
			}
			recibo := personalports.ResultadoRelacionParaRPTV1{}
			recibo.Evidencia.ReciboRef = "recibo:confirmado"
			llamadas := 0
			siguiente := lectorSeleccionadaCorrelacionPrueba(func(context.Context) (personalports.ResultadoRelacionParaRPTV1, error) {
				llamadas++
				return recibo, nil
			})
			l, err := NuevoLectorRelacionSeleccionadaRPTConIdentidad(siguiente, &resolutorIntentoRPTVigenciaPrueba{identidad: id}, time.Second, e, configuracionResultadosLectorRPTPrueba())
			if err != nil {
				t.Fatal(err)
			}
			ctx, _ := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
			r, err := l.ConsultarSeleccionada(ctx, id.Resultado.Contexto, personaldomain.PreparacionRelacionParaRPT{}, "", "")
			cerrarEmisorTecnicoLectorRPTPrueba(t, e)
			if err != nil || llamadas != 1 || r.Evidencia.ReciboRef != recibo.Evidencia.ReciboRef {
				t.Fatal("técnico reintentó consulta o invalidó recibo", err, llamadas)
			}
			m := e.MetricasResultadosTecnicos()
			if (!cerrado && m.FallosEscritura != 1) || (cerrado && m.Descartados != 1) {
				t.Fatal("pérdida no contabilizada", m)
			}
		})
	}
}

func TestLectorRPTResultadosTecnicosCubrenFalloAntesDeDelegarSinInventarCorrelacion(t *testing.T) {
	var b bytes.Buffer
	e := nuevoEmisorTecnicoLectorRPTPrueba(t, &b)
	llamadas := 0
	siguiente := lectorSeleccionadaCorrelacionPrueba(func(context.Context) (personalports.ResultadoRelacionParaRPTV1, error) {
		llamadas++
		return personalports.ResultadoRelacionParaRPTV1{}, nil
	})
	l, err := NuevoLectorRelacionSeleccionadaRPTConIdentidad(siguiente, identidadLectorRelacionRPTPrueba{err: errors.New("identidad privada")}, time.Second, e, configuracionResultadosLectorRPTPrueba())
	if err != nil {
		t.Fatal(err)
	}
	ctx, _ := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	for _, c := range []context.Context{context.Background(), ctx} {
		if _, err := l.ConsultarSeleccionada(c, vecdomain.ContextoActor{}, personaldomain.PreparacionRelacionParaRPT{}, "", ""); !errors.Is(err, personaldomain.ErrLectorRelacionRPTNoDisponible) {
			t.Fatal(err)
		}
	}
	cerrarEmisorTecnicoLectorRPTPrueba(t, e)
	var linea map[string]any
	if json.Unmarshal(bytes.TrimSpace(b.Bytes()), &linea) != nil || linea["resultado"] != "no_disponible" || llamadas != 0 {
		t.Fatal("fallo previo sin traza")
	}
	if m := e.MetricasResultadosTecnicos(); m.SinCorrelacion != 1 || m.Escritos != 1 {
		t.Fatal("correlación inventada", m)
	}
}
