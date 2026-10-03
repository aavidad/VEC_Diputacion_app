package composicion

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	personalrpt "vec-diputacion-granada/internal/modules/personal/adapters/rpt"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func contextoCorrelacionLectorRPTPrueba(t *testing.T) context.Context {
	t.Helper()
	ctx := context.Background()
	ref, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	return context.WithValue(ctx, claveCorrelacionLectorRelacionRPT{}, ref)
}

type lectorSeleccionadaCorrelacionPrueba func(context.Context) (personalports.ResultadoRelacionParaRPTV1, error)

func (f lectorSeleccionadaCorrelacionPrueba) ConsultarSeleccionada(ctx context.Context, _ vecdomain.ContextoActor, _ personaldomain.PreparacionRelacionParaRPT, _, _ string) (personalports.ResultadoRelacionParaRPTV1, error) {
	return f(ctx)
}

type emisorCorrelacionRPTPrueba struct{ ref string }

func (e *emisorCorrelacionRPTPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, solicitud vecdomain.SolicitudAutorizacionLigadaV3, _ vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := solicitud.Datos()
	if err == nil {
		e.ref, _ = datos.Correlacion.ValorCanonico()
	}
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3
}

type identidadCorrelacionRPTPrueba struct {
	valor IdentidadRegistradaLectorRelacionRPT
}

func (i identidadCorrelacionRPTPrueba) ResolverIdentidadLectorRelacionRPT(context.Context) (IdentidadRegistradaLectorRelacionRPT, error) {
	return i.valor, nil
}

func TestLectorRPTComparteCorrelacionConV3EIntento(t *testing.T) {
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	identidad := identidadIntentoRPTVigenciaPrueba(t, ahora, "")
	resolutor := identidadCorrelacionRPTPrueba{identidad}
	emisor := &emisorCorrelacionRPTPrueba{}
	proveedor, err := NuevoProveedorAutorizacionLectorRelacionRPT(resolutor, emisor, vecdomain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion", CatalogoVersion: 3,
		CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("b", 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	destino := &destinoIntentosRPTPrueba{}
	registro, err := NuevoRegistroIntentosLectorRelacionRPT(resolutor, destino, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	material, err := personaldomain.NuevoMaterialLectorRelacionRPT(personaldomain.SolicitudLectorRelacionRPT{
		Actor: identidad.Resultado.Contexto, EmpleadoRef: "emp_" + strings.Repeat("e", 24),
		RelacionRef: "rel_" + strings.Repeat("r", 24), OrganismoRef: "organismo:dipgra", VersionEsperada: 1,
		Corte: personaldomain.CorteEmpleadoB2{VigenteEn: "2026-10-02", ConocidoEn: ahora},
	})
	if err != nil {
		t.Fatal(err)
	}
	lector := lectorRelacionSeleccionadaCorrelacion{siguiente: lectorSeleccionadaCorrelacionPrueba(func(ctx context.Context) (personalports.ResultadoRelacionParaRPTV1, error) {
		if _, err := proveedor.AutorizarRelacionParaRPT(ctx, material); !errors.Is(err, personaldomain.ErrLectorRelacionRPTDenegado) {
			t.Errorf("denegación V3: %v", err)
		}
		auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		if err := registro.RegistrarIntentoRelacionRPT(auditCtx, personalports.IntentoLectorRelacionRPT{Actor: identidad.Resultado.Contexto, RelacionRef: material.Recurso().Referencia, Motivo: "denegado"}); err != nil {
			t.Errorf("registro intento: %v", err)
		}
		return personalports.ResultadoRelacionParaRPTV1{}, personaldomain.ErrLectorRelacionRPTDenegado
	})}
	_, err = lector.ConsultarSeleccionada(context.Background(), vecdomain.ContextoActor{}, personaldomain.PreparacionRelacionParaRPT{}, "", "")
	if !errors.Is(err, personaldomain.ErrLectorRelacionRPTDenegado) || emisor.ref == "" || emisor.ref != destino.evento.CorrelacionRef {
		t.Fatalf("correlación V3/intento divergente: V3=%q intento=%q error=%v", emisor.ref, destino.evento.CorrelacionRef, err)
	}
}

type servicioSeleccionadaRPTPrueba struct{ llamadas int }

func (s *servicioSeleccionadaRPTPrueba) ConsultarRelacionParaRPT(context.Context, personalports.ConsultaRelacionParaRPTV1) (personalports.ResultadoRelacionParaRPTV1, error) {
	s.llamadas++
	return personalports.ResultadoRelacionParaRPTV1{}, nil
}

func TestLectorRPTCorrelacionCubreNegativaPreviaAlServicio(t *testing.T) {
	destino := &destinoIntentosRPTPrueba{}
	registro, err := NuevoRegistroIntentosLectorRelacionRPT(identidadLectorRelacionRPTPrueba{err: errors.New("sin identidad")}, destino, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	servicio := &servicioSeleccionadaRPTPrueba{}
	base, err := personalrpt.NuevoLectorRelacionSeleccionadaRPT(servicio, registro)
	if err != nil {
		t.Fatal(err)
	}
	lector := lectorRelacionSeleccionadaCorrelacion{siguiente: base}
	_, err = lector.ConsultarSeleccionada(context.Background(), vecdomain.ContextoActor{}, personaldomain.PreparacionRelacionParaRPT{}, "", "")
	if !errors.Is(err, personaldomain.ErrLectorRelacionRPTInvalido) || servicio.llamadas != 0 || destino.writes != 1 || destino.evento.ActorRef != "" || !strings.HasPrefix(destino.evento.CorrelacionRef, "correlacion_") {
		t.Fatalf("negativa previa sin traza nominal: %+v, error=%v", destino.evento, err)
	}
}

func TestLectorRPTCorrelacionConservadaTrasCancelacionYSolicitudesAisladas(t *testing.T) {
	refs := make(chan string, 2)
	lector := lectorRelacionSeleccionadaCorrelacion{siguiente: lectorSeleccionadaCorrelacionPrueba(func(ctx context.Context) (personalports.ResultadoRelacionParaRPTV1, error) {
		cancelado, cancel := context.WithCancel(ctx)
		cancel()
		ref, err := correlacionLectorRelacionRPT(context.WithoutCancel(cancelado))
		if err != nil {
			return personalports.ResultadoRelacionParaRPTV1{}, err
		}
		valor, err := ref.ValorCanonico()
		refs <- valor
		return personalports.ResultadoRelacionParaRPTV1{}, err
	})}
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := lector.ConsultarSeleccionada(context.Background(), vecdomain.ContextoActor{}, personaldomain.PreparacionRelacionParaRPT{}, "", ""); err != nil {
				t.Errorf("consulta: %v", err)
			}
		}()
	}
	wg.Wait()
	a, b := <-refs, <-refs
	if a == b || !strings.HasPrefix(a, "correlacion_") || !strings.HasPrefix(b, "correlacion_") {
		t.Fatalf("correlaciones mezcladas: %q, %q", a, b)
	}
}
