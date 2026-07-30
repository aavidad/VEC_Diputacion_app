package httpseguridad

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type entornoCapsulaIdentidad struct {
	servicio *ServicioIdentidad
	canal    CanalProxyAutenticado
	capsula  CapsulaIdentidadPeticion
	registro *registroMemoria
	reloj    *relojFijo
}

func nuevoEntornoCapsulaIdentidad(t *testing.T) entornoCapsulaIdentidad {
	t.Helper()
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	registro := nuevoRegistroMemoria()
	reloj := &relojFijo{ahora: ahora}
	servicio := debeServicio(
		t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh),
		registro, reloj,
	)
	canal := debeCanalTLS(t, servicio, configuracion)
	verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))
	identidad, err := servicio.Resolver(
		context.Background(), debeCredencial(t, []byte("capsula-opaca"), canal),
	)
	if err != nil {
		t.Fatalf("resolver identidad: %v", err)
	}
	capsula, err := servicio.ProyectarCapsulaIdentidadPeticion(
		context.Background(), identidad, canal,
	)
	if err != nil {
		t.Fatalf("proyectar cápsula: %v", err)
	}
	return entornoCapsulaIdentidad{
		servicio: servicio, canal: canal, capsula: capsula,
		registro: registro, reloj: reloj,
	}
}

func TestCapsulaIdentidadPeticionRevalidaCanalYServicio(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	configuracion := configuracionInternaValida()
	verificador := &verificadorFalso{}
	servicio := debeServicio(t, configuracion, verificador, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, servicio, configuracion)
	verificador.fijarAsercion(asercionInternaValida(ahora, configuracion, canal))
	identidad, err := servicio.Resolver(context.Background(), debeCredencial(t, []byte("capsula"), canal))
	if err != nil {
		t.Fatal(err)
	}
	capsula, err := servicio.ProyectarCapsulaIdentidadPeticion(context.Background(), identidad, canal)
	cuenta, auditoria, errDatos := capsula.datos(
		context.Background(), servicio, canal.ReferenciaVinculacion(),
	)
	if err != nil || errDatos != nil || cuenta.Validar() != nil || auditoria.CanalVinculadoRef() != canal.ReferenciaVinculacion() {
		t.Fatal("cápsula válida rechazada")
	}
	vinculado, err := servicio.VincularCapsulaIdentidadPeticion(context.Background(), capsula, canal)
	if err != nil {
		t.Fatal("vincular cápsula")
	}
	if _, _, err = servicio.ExtraerCapsulaIdentidadPeticion(vinculado); err != nil {
		t.Fatal("extraer cápsula")
	}
	canal.evidenciaRef = "tls-exportador:sha256:cruzado"
	if _, err := servicio.ProyectarCapsulaIdentidadPeticion(context.Background(), identidad, canal); !errors.Is(err, ErrSesionNoValida) {
		t.Fatal("canal cruzado admitido")
	}
}

func TestCapsulaIdentidadPeticionCierraCancelacionYConcurrencia(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	c := configuracionInternaValida()
	v := &verificadorFalso{}
	s := debeServicio(t, c, v, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, s, c)
	v.fijarAsercion(asercionInternaValida(ahora, c, canal))
	identidad, err := s.Resolver(context.Background(), debeCredencial(t, []byte("concurrente"), canal))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err = s.ProyectarCapsulaIdentidadPeticion(ctx, identidad, canal); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación perdida")
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.ProyectarCapsulaIdentidadPeticion(context.Background(), identidad, canal); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestCapsulaIdentidadPeticionCierraCeroContextoYCancelacion(t *testing.T) {
	var cero CapsulaIdentidadPeticion
	if _, _, err := cero.datos(context.Background(), nil, ""); !errors.Is(err, ErrSesionNoValida) {
		t.Fatal("cero admitido")
	}
	if _, _, err := (*ServicioIdentidad)(nil).ExtraerCapsulaIdentidadPeticion(context.Background()); !errors.Is(err, ErrSesionNoValida) {
		t.Fatal("ausencia admitida")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := (*ServicioIdentidad)(nil).VincularCapsulaIdentidadPeticion(ctx, cero, CanalProxyAutenticado{}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación de vinculación perdida")
	}
	if _, _, err := (*ServicioIdentidad)(nil).ExtraerCapsulaIdentidadPeticion(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelación de extracción perdida")
	}
}

func TestCapsulaIdentidadPeticionRechazaCrucesYAdulteraciones(t *testing.T) {
	entorno := nuevoEntornoCapsulaIdentidad(t)
	otro := nuevoEntornoCapsulaIdentidad(t)
	canalCruzado := entorno.canal
	canalCruzado.evidenciaRef = "tls-exportador:sha256:canal-cruzado"

	if _, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), entorno.capsula, canalCruzado,
	); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("vinculación con canal cruzado admitida: %v", err)
	}
	vinculado, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), entorno.capsula, entorno.canal,
	)
	if err != nil {
		t.Fatalf("vincular cápsula válida: %v", err)
	}
	contextoAlterado := context.WithValue(
		context.Background(),
		claveCapsulaIdentidad{},
		capsulaIdentidadVinculada{
			capsula: entorno.capsula, canalVinculadoRef: canalCruzado.ReferenciaVinculacion(),
		},
	)
	if _, _, err = entorno.servicio.ExtraerCapsulaIdentidadPeticion(
		contextoAlterado,
	); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("vínculo de canal alterado admitido: %v", err)
	}
	if _, err = otro.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), entorno.capsula, otro.canal,
	); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("servicio cruzado admitido: %v", err)
	}
	if _, _, err = otro.servicio.ExtraerCapsulaIdentidadPeticion(
		vinculado,
	); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("extracción por otro servicio admitida: %v", err)
	}

	casos := map[string]func(*CapsulaIdentidadPeticion){
		"instancia": func(c *CapsulaIdentidadPeticion) {
			c.instancia[0] ^= 1
		},
		"canal": func(c *CapsulaIdentidadPeticion) {
			c.canal[0] ^= 1
		},
		"servicio identidad": func(c *CapsulaIdentidadPeticion) {
			c.identidad.servicio = otro.servicio
		},
		"instancia identidad": func(c *CapsulaIdentidadPeticion) {
			c.identidad.instanciaRef[0] ^= 1
		},
		"configuración identidad": func(c *CapsulaIdentidadPeticion) {
			c.identidad.huellaConfiguracion[0] ^= 1
		},
		"canal identidad": func(c *CapsulaIdentidadPeticion) {
			c.identidad.estado.canalVinculadoRef = "tls-exportador:sha256:alterado"
		},
		"estado de vinculación": func(c *CapsulaIdentidadPeticion) {
			c.estado = nil
		},
	}
	for nombre, adulterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			capsula := entorno.capsula
			adulterar(&capsula)
			if _, _, err := capsula.datos(
				context.Background(), entorno.servicio,
				entorno.canal.ReferenciaVinculacion(),
			); !errors.Is(err, ErrSesionNoValida) {
				t.Fatalf("cápsula adulterada admitida: %v", err)
			}
		})
	}
}

func TestCapsulaIdentidadPeticionRevalidaRevocacionYCaducidad(t *testing.T) {
	t.Run("revocada antes de vincular", func(t *testing.T) {
		entorno := nuevoEntornoCapsulaIdentidad(t)
		entorno.registro.revocar("sesion-001")
		if _, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
			context.Background(), entorno.capsula, entorno.canal,
		); !errors.Is(err, ErrSesionNoValida) {
			t.Fatalf("sesión revocada admitida al vincular: %v", err)
		}
	})
	t.Run("revocada antes de extraer", func(t *testing.T) {
		entorno := nuevoEntornoCapsulaIdentidad(t)
		vinculado, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
			context.Background(), entorno.capsula, entorno.canal,
		)
		if err != nil {
			t.Fatalf("vincular: %v", err)
		}
		entorno.registro.revocar("sesion-001")
		if _, _, err = entorno.servicio.ExtraerCapsulaIdentidadPeticion(
			vinculado,
		); !errors.Is(err, ErrSesionNoValida) {
			t.Fatalf("sesión revocada admitida al extraer: %v", err)
		}
	})
	t.Run("caducada antes de vincular", func(t *testing.T) {
		entorno := nuevoEntornoCapsulaIdentidad(t)
		entorno.reloj.fijar(entorno.reloj.Ahora().Add(2 * time.Minute))
		if _, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
			context.Background(), entorno.capsula, entorno.canal,
		); !errors.Is(err, ErrSesionNoValida) {
			t.Fatalf("sesión caducada admitida al vincular: %v", err)
		}
	})
	t.Run("caducada antes de extraer", func(t *testing.T) {
		entorno := nuevoEntornoCapsulaIdentidad(t)
		vinculado, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
			context.Background(), entorno.capsula, entorno.canal,
		)
		if err != nil {
			t.Fatalf("vincular: %v", err)
		}
		entorno.reloj.fijar(entorno.reloj.Ahora().Add(2 * time.Minute))
		if _, _, err = entorno.servicio.ExtraerCapsulaIdentidadPeticion(
			vinculado,
		); !errors.Is(err, ErrSesionNoValida) {
			t.Fatalf("sesión caducada admitida al extraer: %v", err)
		}
	})
}

func TestCapsulaIdentidadPeticionCierraSustitucionYContextosTerminados(t *testing.T) {
	entorno := nuevoEntornoCapsulaIdentidad(t)
	vinculado, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), entorno.capsula, entorno.canal,
	)
	if err != nil {
		t.Fatalf("vincular: %v", err)
	}
	if _, err = entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), entorno.capsula, entorno.canal,
	); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("segunda vinculación en otro contexto admitida: %v", err)
	}
	otro := nuevoEntornoCapsulaIdentidad(t)
	if _, err = otro.servicio.VincularCapsulaIdentidadPeticion(
		vinculado, otro.capsula, otro.canal,
	); !errors.Is(err, ErrSesionNoValida) {
		t.Fatalf("sustitución dentro del contexto admitida: %v", err)
	}

	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	ctxVencido, cancelarVencido := context.WithDeadline(
		context.Background(), time.Now().Add(-time.Second),
	)
	defer cancelarVencido()
	for nombre, caso := range map[string]struct {
		ctx    context.Context
		espera error
	}{
		"cancelado": {ctx: ctxCancelado, espera: context.Canceled},
		"vencido":   {ctx: ctxVencido, espera: context.DeadlineExceeded},
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
				caso.ctx, entorno.capsula, entorno.canal,
			); !errors.Is(err, caso.espera) {
				t.Fatalf("vinculación perdió error de contexto: %v", err)
			}
			if _, _, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(
				caso.ctx,
			); !errors.Is(err, caso.espera) {
				t.Fatalf("extracción perdió error de contexto: %v", err)
			}
		})
	}
}

func TestCapsulaIdentidadPeticionNoFiltraNiSeReconstruye(t *testing.T) {
	entorno := nuevoEntornoCapsulaIdentidad(t)
	capsula := entorno.capsula
	contextoVinculado, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), capsula, entorno.canal,
	)
	if err != nil {
		t.Fatalf("vincular: %v", err)
	}
	vinculada, ok := contextoVinculado.Value(
		claveCapsulaIdentidad{},
	).(capsulaIdentidadVinculada)
	if !ok {
		t.Fatal("transporte privado ausente")
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info(
		"cápsula", "valor", capsula,
	)
	var registroVinculada bytes.Buffer
	slog.New(slog.NewTextHandler(&registroVinculada, nil)).Info(
		"cápsula vinculada", "valor", vinculada,
	)
	representaciones := []string{
		fmt.Sprintf("%s %v %#v %+v", capsula, capsula, capsula, capsula),
		capsula.String(),
		capsula.GoString(),
		capsula.LogValue().String(),
		registro.String(),
		fmt.Sprintf("%s %v %#v %+v", vinculada, vinculada, vinculada, vinculada),
		vinculada.String(),
		vinculada.GoString(),
		vinculada.LogValue().String(),
		registroVinculada.String(),
	}
	prohibidos := []string{
		capsula.identidad.confirmacion.AutenticacionRef,
		capsula.identidad.confirmacion.AsercionRef,
		capsula.identidad.confirmacion.SesionRef,
		capsula.identidad.confirmacion.CuentaRef,
		capsula.identidad.estado.canalVinculadoRef,
		"persona-001", "cuenta-tecnica", "sesion-001",
	}
	for _, representacion := range representaciones {
		for _, prohibido := range prohibidos {
			if strings.Contains(representacion, prohibido) {
				t.Fatalf("cápsula filtrada (%q): %q", prohibido, representacion)
			}
		}
	}
	if _, err := json.Marshal(capsula); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("JSON admitido: %v", err)
	}
	if _, err := capsula.MarshalText(); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("texto admitido: %v", err)
	}
	if _, err := capsula.MarshalBinary(); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("binario admitido: %v", err)
	}
	if _, err := capsula.GobEncode(); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("Gob admitido: %v", err)
	}
	if _, err := capsula.MarshalCBOR(); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("CBOR admitido: %v", err)
	}
	if _, err := capsula.MarshalYAML(); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("YAML admitido: %v", err)
	}
	if err := capsula.MarshalXML(
		xml.NewEncoder(&bytes.Buffer{}), xml.StartElement{},
	); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("XML admitido: %v", err)
	}
	var serializacionGob bytes.Buffer
	if err := gob.NewEncoder(&serializacionGob).Encode(capsula); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("codificador Gob admitió cápsula: %v", err)
	}
	var reconstruida CapsulaIdentidadPeticion
	if err := json.Unmarshal([]byte(`{}`), &reconstruida); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("JSON reconstruyó cápsula: %v", err)
	}
	if err := reconstruida.UnmarshalText([]byte("dato")); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("texto reconstruyó cápsula: %v", err)
	}
	if err := reconstruida.UnmarshalBinary([]byte("dato")); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("binario reconstruyó cápsula: %v", err)
	}
	if err := reconstruida.GobDecode([]byte("dato")); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("Gob reconstruyó cápsula: %v", err)
	}
	if err := reconstruida.UnmarshalCBOR([]byte{0xa0}); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("CBOR reconstruyó cápsula: %v", err)
	}
	if err := reconstruida.UnmarshalYAML(
		func(any) error { return nil },
	); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("YAML reconstruyó cápsula: %v", err)
	}
	if err := reconstruida.UnmarshalXML(
		xml.NewDecoder(bytes.NewReader(nil)), xml.StartElement{},
	); !errors.Is(err, ErrIdentidadNoSerializable) {
		t.Fatalf("XML reconstruyó cápsula: %v", err)
	}
}

func TestCapsulaIdentidadPeticionSoloSeVinculaUnaVezConCarrera(t *testing.T) {
	entorno := nuevoEntornoCapsulaIdentidad(t)
	const intentos = 64
	var grupo sync.WaitGroup
	var exitos atomic.Int64
	var rechazos atomic.Int64
	for range intentos {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			_, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
				context.Background(), entorno.capsula, entorno.canal,
			)
			switch {
			case err == nil:
				exitos.Add(1)
			case errors.Is(err, ErrSesionNoValida):
				rechazos.Add(1)
			default:
				t.Errorf("error inesperado: %v", err)
			}
		}()
	}
	grupo.Wait()
	if exitos.Load() != 1 || rechazos.Load() != intentos-1 {
		t.Fatalf(
			"vinculación no exclusiva: éxitos=%d rechazos=%d",
			exitos.Load(), rechazos.Load(),
		)
	}
}

func TestCapsulaIdentidadPeticionPermiteLecturaConcurrenteRevalidada(t *testing.T) {
	entorno := nuevoEntornoCapsulaIdentidad(t)
	vinculado, err := entorno.servicio.VincularCapsulaIdentidadPeticion(
		context.Background(), entorno.capsula, entorno.canal,
	)
	if err != nil {
		t.Fatalf("vincular: %v", err)
	}
	var grupo sync.WaitGroup
	for range 32 {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			cuenta, auditoria, err := entorno.servicio.ExtraerCapsulaIdentidadPeticion(
				vinculado,
			)
			if err != nil || cuenta.Validar() != nil ||
				auditoria.CanalVinculadoRef() != entorno.canal.ReferenciaVinculacion() {
				t.Errorf("lectura concurrente rechazada: %v", err)
			}
		}()
	}
	grupo.Wait()
}
