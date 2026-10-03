package composicion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	personalrpt "vec-diputacion-granada/internal/modules/personal/adapters/rpt"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type lectorSeleccionadaCorrelacionPrueba func(context.Context) (personalports.ResultadoRelacionParaRPTV1, error)

func (f lectorSeleccionadaCorrelacionPrueba) ConsultarSeleccionada(ctx context.Context, _ vecdomain.ContextoActor, _ personaldomain.PreparacionRelacionParaRPT, _, _ string) (personalports.ResultadoRelacionParaRPTV1, error) {
	return f(ctx)
}

type emisorCorrelacionRPTPrueba struct{ ref string }

func (e *emisorCorrelacionRPTPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, s vecdomain.SolicitudAutorizacionLigadaV3, _ vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := s.Datos()
	if err == nil {
		e.ref, _ = datos.Correlacion.ValorCanonico()
	}
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3
}

type servicioSeleccionadaRPTPrueba struct{ llamadas int }

func (s *servicioSeleccionadaRPTPrueba) ConsultarRelacionParaRPT(context.Context, personalports.ConsultaRelacionParaRPTV1) (personalports.ResultadoRelacionParaRPTV1, error) {
	s.llamadas++
	return personalports.ResultadoRelacionParaRPTV1{}, nil
}

func TestLectorRPTComparteCorrelacionTecnicaV3EIntentoOriginal(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	id := identidadIntentoRPTVigenciaPrueba(t, ahora, "")
	resolutor := &resolutorIntentoRPTVigenciaPrueba{identidad: id}
	emisor := &emisorCorrelacionRPTPrueba{}
	proveedor, err := NuevoProveedorAutorizacionLectorRelacionRPT(resolutor, emisor, configuracionIntentosRPTPrueba().MotivoDenegado)
	if err != nil {
		t.Fatal(err)
	}
	d := &destinoIntentosRPTPrueba{}
	registro, _ := NuevoRegistroIntentosLectorRelacionRPT(d, configuracionIntentosRPTPrueba())
	m, err := personaldomain.NuevoMaterialLectorRelacionRPT(personaldomain.SolicitudLectorRelacionRPT{Actor: id.Resultado.Contexto, EmpleadoRef: "emp_" + strings.Repeat("e", 24), RelacionRef: "rel_" + strings.Repeat("r", 24), OrganismoRef: "organismo:dipgra", VersionEsperada: 1, Corte: personaldomain.CorteEmpleadoB2{VigenteEn: "2026-10-02", ConocidoEn: ahora}})
	if err != nil {
		t.Fatal(err)
	}
	var registroTecnico bytes.Buffer
	resultadoTecnico := nuevoEmisorTecnicoLectorRPTPrueba(t, &registroTecnico)
	lector := lectorRelacionSeleccionadaCorrelacion{identidad: resolutor, limiteIdentidad: time.Second, emisorTecnico: resultadoTecnico, configuracionResultados: configuracionResultadosLectorRPTPrueba(), siguiente: lectorSeleccionadaCorrelacionPrueba(func(ctx context.Context) (personalports.ResultadoRelacionParaRPTV1, error) {
		resolutor.identidad = IdentidadRegistradaLectorRelacionRPT{}
		if _, err := proveedor.AutorizarRelacionParaRPT(ctx, m); !errors.Is(err, personaldomain.ErrLectorRelacionRPTDenegado) {
			t.Errorf("V3: %v", err)
		}
		if err := registro.RegistrarIntentoRelacionRPT(ctx, personalports.IntentoLectorRelacionRPT{Actor: id.Resultado.Contexto, RelacionRef: m.Recurso().Referencia, Motivo: "denegado"}); err != nil {
			t.Errorf("intento: %v", err)
		}
		return personalports.ResultadoRelacionParaRPTV1{}, personaldomain.ErrLectorRelacionRPTDenegado
	})}
	ctx, _ := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	tecnica, _ := vecports.CorrelacionIncidenciasPeticion(ctx)
	_, err = lector.ConsultarSeleccionada(ctx, id.Resultado.Contexto, personaldomain.PreparacionRelacionParaRPT{}, "", "")
	if !errors.Is(err, personaldomain.ErrLectorRelacionRPTDenegado) || len(d.ordenes) != 1 || resolutor.llamadas != 1 {
		t.Fatal("no mantuvo original", err)
	}
	datos, _ := d.ordenes[0].Datos()
	if emisor.ref != "correlacion_"+tecnica || datos.Datos.CorrelacionRef != emisor.ref || datos.ResultadoContexto.HuellaSHA256 != id.Resultado.HuellaSHA256 {
		t.Fatal("correlación o identidad divergentes")
	}
	cerrarEmisorTecnicoLectorRPTPrueba(t, resultadoTecnico)
	var linea map[string]any
	if json.Unmarshal(bytes.TrimSpace(registroTecnico.Bytes()), &linea) != nil || linea["correlacion_ref"] != emisor.ref || linea["resultado"] != "denegado" {
		t.Fatal("V3, nominal y técnico no comparten correlación")
	}
}

func TestLectorRPTCorrelacionCubreNegativaPreviaAlServicio(t *testing.T) {
	id := identidadIntentoRPTVigenciaPrueba(t, time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC), "")
	resolutor := &resolutorIntentoRPTVigenciaPrueba{identidad: id}
	d := &destinoIntentosRPTPrueba{}
	registro, _ := NuevoRegistroIntentosLectorRelacionRPT(d, configuracionIntentosRPTPrueba())
	servicio := &servicioSeleccionadaRPTPrueba{}
	base, _ := personalrpt.NuevoLectorRelacionSeleccionadaRPT(servicio, registro)
	lector := lectorRelacionSeleccionadaCorrelacion{siguiente: base, identidad: resolutor, limiteIdentidad: time.Second, emisorTecnico: emisorTecnicoLectorRPTPrueba(t), configuracionResultados: configuracionResultadosLectorRPTPrueba()}
	ctx, _ := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	_, err := lector.ConsultarSeleccionada(ctx, vecdomain.ContextoActor{}, personaldomain.PreparacionRelacionParaRPT{}, "", "")
	if !errors.Is(err, personaldomain.ErrLectorRelacionRPTInvalido) || servicio.llamadas != 0 || len(d.ordenes) != 1 {
		t.Fatal("negativa sin auditoría", err)
	}
}

func TestLectorRPTNoAcuñaCorrelacionNiIdentidadAusentes(t *testing.T) {
	llamadas := 0
	lector := lectorRelacionSeleccionadaCorrelacion{limiteIdentidad: time.Second, emisorTecnico: emisorTecnicoLectorRPTPrueba(t), configuracionResultados: configuracionResultadosLectorRPTPrueba(), identidad: identidadLectorRelacionRPTPrueba{err: errors.New("sin identidad")}, siguiente: lectorSeleccionadaCorrelacionPrueba(func(context.Context) (personalports.ResultadoRelacionParaRPTV1, error) {
		llamadas++
		return personalports.ResultadoRelacionParaRPTV1{}, nil
	})}
	ctx, _ := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	for _, c := range []context.Context{context.Background(), ctx} {
		if _, err := lector.ConsultarSeleccionada(c, vecdomain.ContextoActor{}, personaldomain.PreparacionRelacionParaRPT{}, "", ""); !errors.Is(err, personaldomain.ErrLectorRelacionRPTNoDisponible) {
			t.Fatal("abrió sin frontera", err)
		}
	}
	if llamadas != 0 {
		t.Fatal("delegó sin frontera")
	}
}

func TestLectorRPTFronteraRechazaIdentidadPlazoOLectorAusentes(t *testing.T) {
	lector := lectorSeleccionadaCorrelacionPrueba(func(context.Context) (personalports.ResultadoRelacionParaRPTV1, error) {
		return personalports.ResultadoRelacionParaRPTV1{}, nil
	})
	identidad := identidadLectorRelacionRPTPrueba{}
	for _, plazo := range []time.Duration{0, -time.Second, 31 * time.Second} {
		if _, err := NuevoLectorRelacionSeleccionadaRPTConIdentidad(lector, identidad, plazo, emisorTecnicoLectorRPTPrueba(t), configuracionResultadosLectorRPTPrueba()); !errors.Is(err, personaldomain.ErrLectorRelacionRPTNoDisponible) {
			t.Fatal("plazo inválido admitido", err)
		}
	}
	if _, err := NuevoLectorRelacionSeleccionadaRPTConIdentidad(nil, identidad, time.Second, emisorTecnicoLectorRPTPrueba(t), configuracionResultadosLectorRPTPrueba()); err == nil {
		t.Fatal("lector ausente admitido")
	}
	if _, err := NuevoLectorRelacionSeleccionadaRPTConIdentidad(lector, nil, time.Second, emisorTecnicoLectorRPTPrueba(t), configuracionResultadosLectorRPTPrueba()); err == nil {
		t.Fatal("identidad ausente admitida")
	}
}
