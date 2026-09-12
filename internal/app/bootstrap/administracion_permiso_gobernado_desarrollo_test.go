package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	adminhttp "vec-diputacion-granada/internal/modules/administracion/adapters/http"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorPermisoAdministracionPrueba struct {
	mu       sync.Mutex
	llamadas int
}

func (a *autorizadorPermisoAdministracionPrueba) ExigirSolicitudLigadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.llamadas++
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, ErrConfiguracionCorreoAdministracionNoDisponible
}

func (a *autorizadorPermisoAdministracionPrueba) llamadasActuales() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.llamadas
}

type referenciasPermisoAdministracionPrueba struct{}

func (*referenciasPermisoAdministracionPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_0123456789abcdef0123456789abcdef", nil
}
func (*referenciasPermisoAdministracionPrueba) NuevaClaveMotivoAutorizacionV2(context.Context) (string, error) {
	return "motivo_0123456789abcdef0123456789abcdef", nil
}

type relojPermisoAdministracionPrueba struct{ ahora time.Time }

func (r *relojPermisoAdministracionPrueba) Ahora() time.Time { return r.ahora }

func motivoPermisoAdministracionPrueba() vecdomain.ReferenciaEntradaCatalogo {
	return vecdomain.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_administracion_prueba", CatalogoVersion: 1,
		CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
}

func TestNuevaFuentePermisoAdministracionGobernadaV3CierraDependenciasNulas(t *testing.T) {
	motivo := motivoPermisoAdministracionPrueba()
	autorizador := &autorizadorPermisoAdministracionPrueba{}
	referencias := &referenciasPermisoAdministracionPrueba{}
	reloj := &relojPermisoAdministracionPrueba{ahora: time.Now().UTC()}
	var autorizadorTipado *autorizadorPermisoAdministracionPrueba
	var referenciasTipadas *referenciasPermisoAdministracionPrueba
	var relojTipado *relojPermisoAdministracionPrueba
	for nombre, datos := range map[string]struct {
		autorizador vecports.AutorizadorSolicitudLigadaV3
		referencias vecports.GeneradorReferenciasAutorizacionV2
		reloj       vecports.Reloj
	}{
		"autorizador nil":       {nil, referencias, reloj},
		"autorizador typed nil": {autorizadorTipado, referencias, reloj},
		"referencias nil":       {autorizador, nil, reloj},
		"referencias typed nil": {autorizador, referenciasTipadas, reloj},
		"reloj nil":             {autorizador, referencias, nil},
		"reloj typed nil":       {autorizador, referencias, relojTipado},
	} {
		t.Run(nombre, func(t *testing.T) {
			fuente, err := nuevaFuentePermisoAdministracionGobernadaV3(datos.autorizador, motivo, datos.referencias, datos.reloj)
			if fuente != nil || err != ErrConfiguracionCorreoAdministracionNoDisponible {
				t.Fatalf("fuente=%v err=%v", fuente, err)
			}
		})
	}
}

func TestFuentePermisoAdministracionGobernadaV3NiegaAntesDelPDP(t *testing.T) {
	autorizador := &autorizadorPermisoAdministracionPrueba{}
	fuente, err := nuevaFuentePermisoAdministracionGobernadaV3(autorizador, motivoPermisoAdministracionPrueba(), &referenciasPermisoAdministracionPrueba{}, &relojPermisoAdministracionPrueba{ahora: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	for nombre, ctx := range map[string]context.Context{"ausente": context.Background(), "cancelado": cancelado} {
		t.Run(nombre, func(t *testing.T) {
			principal, err := fuente.ResolverPrincipalAdministracionCorreoV3(ctx, vecdomain.VinculoAutenticacionActorV2{}, vecdomain.ResultadoContextoActorRegistradoV2{})
			if err != ErrConfiguracionCorreoAdministracionNoDisponible || !reflect.DeepEqual(principal, vecdomain.Principal{}) || autorizador.llamadasActuales() != 0 {
				t.Fatalf("principal=%+v err=%v llamadas=%d", principal, err, autorizador.llamadasActuales())
			}
		})
	}
}

func TestFuentePermisoAdministracionGobernadaV3NiegaSesionCeroConCapacidadTLS(t *testing.T) {
	m := nuevoMaterialTLSAdministracionPrueba(t)
	a, _, cfg := nuevaAutoridadAdministracionPrueba(t, m, true)
	autorizador := &autorizadorPermisoAdministracionPrueba{}
	fuente, err := nuevaFuentePermisoAdministracionGobernadaV3(autorizador, motivoPermisoAdministracionPrueba(), &referenciasPermisoAdministracionPrueba{}, &relojPermisoAdministracionPrueba{ahora: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	resultado := make(chan error, 1)
	url := iniciarServidorAdministracionPrueba(t, a, cfg, m, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := fuente.ResolverPrincipalAdministracionCorreoV3(r.Context(), vecdomain.VinculoAutenticacionActorV2{}, vecdomain.ResultadoContextoActorRegistradoV2{})
		if !reflect.DeepEqual(principal, vecdomain.Principal{}) || err != ErrConfiguracionCorreoAdministracionNoDisponible {
			resultado <- fmt.Errorf("principal=%+v err=%v", principal, err)
		} else {
			resultado <- nil
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	respuesta, err := clienteAdministracionPrueba(m, &m.admin).Get(url + adminhttp.RutaConfiguracionCorreo)
	if err != nil {
		t.Fatal(err)
	}
	respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusNoContent {
		t.Fatalf("estado TLS=%d", respuesta.StatusCode)
	}
	if err := <-resultado; err != nil {
		t.Fatal(err)
	}
	if llamadas := autorizador.llamadasActuales(); llamadas != 0 {
		t.Fatalf("una sesión cero alcanzó PDP: %d", llamadas)
	}
}
