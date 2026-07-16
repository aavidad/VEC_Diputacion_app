package memory

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestRepositorioFlujoFirmaRechazaArrendamientoConCercadoObsoleto(t *testing.T) {
	ctx := context.Background()
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba}
	sellador := selladorEstadoFlujoMemoriaPrueba{}
	repositorio, err := NuevoRepositorioFlujosFirmaBaremacion(reloj, sellador)
	if err != nil {
		t.Fatal(err)
	}
	if formateado := fmt.Sprintf("%v|%#v|%+v", repositorio, repositorio, repositorio); strings.Count(formateado, "[REPOSITORIO-FLUJOS-FIRMA-BAREMACION-MEMORIA-REDACTADO]") != 3 {
		t.Fatalf("el repositorio filtro su secreto HMAC: %q", formateado)
	}
	tipoRepositorio := reflect.TypeOf(repositorio).Elem()
	if _, existe := tipoRepositorio.FieldByName("claveHMACTokenArrendamiento"); existe {
		t.Fatal("el adaptador conserva la clave HMAC en un contenedor reflectible")
	}
	campoOperacion, existe := tipoRepositorio.FieldByName("operarHMACTokenArrendamiento")
	if !existe {
		t.Fatal("el adaptador no contiene la operacion HMAC privada")
	}
	valorOperacion := reflect.ValueOf(repositorio).Elem().FieldByName("operarHMACTokenArrendamiento")
	if campoOperacion.Type.Kind() != reflect.Func || valorOperacion.CanInterface() || valorOperacion.CanSet() {
		t.Fatalf("operacion HMAC privada insegura: clase=%v accesible=%t mutable=%t",
			campoOperacion.Type.Kind(), valorOperacion.CanInterface(), valorOperacion.CanSet())
	}
	invocable := true
	func() {
		defer func() {
			if recover() != nil {
				invocable = false
			}
		}()
		valorOperacion.Call([]reflect.Value{
			reflect.ValueOf(puertosbolsa.TokenArrendamientoFlujoFirmaBaremacion{}),
			reflect.ValueOf(finalidadSellarTokenArrendamientoMemoria),
			reflect.Zero(valorOperacion.Type().In(2)),
		})
	}()
	if invocable {
		t.Fatal("reflect pudo invocar el cierre HMAC privado del repositorio")
	}
	protector, err := NuevoProtectorEstadoFlujoFirmaBaremacion(
		"clave-estado-flujo-firma-v1",
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatal(err)
	}
	carga, err := puertosbolsa.NuevaCargaProtegida([]byte("estado-inicial-sin-capacidades"))
	if err != nil {
		t.Fatal(err)
	}
	estado, err := protector.ProtegerEstadoFlujoFirmaBaremacion(ctx, carga)
	if err != nil {
		t.Fatal(err)
	}
	expediente := sellarExpedienteFlujoMemoriaPrueba(t, sellador, puertosbolsa.ExpedienteFlujoFirmaBaremacion{
		FlujoRef: "flujo-firma-cercado-001", Version: 1,
		IndiceIdempotenciaHMAC: hmacEstadoFlujoMemoriaPrueba("indice"),
		HuellaSolicitudHMAC:    hmacEstadoFlujoMemoriaPrueba("solicitud"),
		VinculoActorHMAC:       hmacEstadoFlujoMemoriaPrueba("actor"),
		PerfilActorClave:       perfilBaremacionMemoriaPrueba,
		ProcesoRef:             "proceso-001",
		SolicitudRef:           "solicitud-001",
		BaremacionMeritoRef:    "baremacion-001",
		DecisionRef:            "decision-001",
		Estado:                 puertosbolsa.EstadoExpedienteFirmaPreparando,
		EstadoProtegido:        estado,
		CreadoEn:               instanteMemoriaPrueba,
		ActualizadoEn:          instanteMemoriaPrueba,
	})
	if _, err := repositorio.CrearORecuperarFlujoFirmaBaremacion(
		ctx,
		puertosbolsa.SolicitudCrearORecuperarFlujoFirmaBaremacion{Expediente: expediente},
	); err != nil {
		t.Fatal(err)
	}
	consulta := puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion{
		FlujoRef: expediente.FlujoRef, IndiceIdempotenciaHMAC: expediente.IndiceIdempotenciaHMAC,
		VinculoActorHMAC: expediente.VinculoActorHMAC,
	}
	primero, err := repositorio.AdquirirArrendamientoFlujoFirmaBaremacion(
		ctx,
		puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion{
			Consulta: consulta, VersionEsperada: 1,
			PropietarioRef: "propietario-antiguo", Duracion: time.Second,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	reloj.fijar(instanteMemoriaPrueba.Add(2 * time.Second))
	segundo, err := repositorio.AdquirirArrendamientoFlujoFirmaBaremacion(
		ctx,
		puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion{
			Consulta: consulta, VersionEsperada: 1,
			PropietarioRef: "propietario-vigente", Duracion: time.Second,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if segundo.Arrendamiento.SecuenciaCercado <= primero.Arrendamiento.SecuenciaCercado {
		t.Fatalf("cercado no monotono: primero=%d segundo=%d", primero.Arrendamiento.SecuenciaCercado, segundo.Arrendamiento.SecuenciaCercado)
	}
	registroVigente := repositorio.arrendamientos[segundo.Arrendamiento.FlujoRef]
	_, tokenVigente, errorTokenVigente := repositorio.operarHMACTokenArrendamiento(
		segundo.Arrendamiento.Token,
		finalidadVerificarTokenArrendamientoMemoria,
		registroVigente.huellaTokenHMAC,
	)
	if len(registroVigente.huellaTokenHMAC) != sha256.Size || errorTokenVigente != nil || !tokenVigente {
		t.Fatal("el adaptador no conservo una huella HMAC autenticable del token")
	}

	siguiente, err := expediente.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	siguiente.Version = 2
	siguiente.ActualizadoEn = reloj.Ahora()
	siguiente.PuntosControl = append(siguiente.PuntosControl, puertosbolsa.PuntoControlFirmaBaremacion{
		Paso: puertosbolsa.PasoPrepararFirmaBaremacion, Estado: puertosbolsa.EstadoPuntoControlFirmaDeclarado,
		EfectoRef: "efecto-preparacion-001", ClaveIdempotenciaHMAC: hmacEstadoFlujoMemoriaPrueba("efecto"),
		DeclaradoEn: reloj.Ahora(),
	})
	siguiente.SelloEstadoHMAC = ""
	siguiente = sellarExpedienteFlujoMemoriaPrueba(t, sellador, siguiente)

	// Los metadatos visibles conservan valor de cercado y diagnostico, pero no
	// constituyen autoridad sin la capacidad opaca emitida por el repositorio.
	soloMetadatos := puertosbolsa.ArrendamientoFlujoFirmaBaremacion{
		FlujoRef:         segundo.Arrendamiento.FlujoRef,
		PropietarioRef:   segundo.Arrendamiento.PropietarioRef,
		SecuenciaCercado: segundo.Arrendamiento.SecuenciaCercado,
		ExpiraEn:         segundo.Arrendamiento.ExpiraEn,
	}
	if _, err := repositorio.GuardarFlujoFirmaBaremacion(ctx, puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
		VersionEsperada: 1, Arrendamiento: soloMetadatos, Siguiente: siguiente,
	}); err == nil {
		t.Fatal("copiar solo los metadatos autorizo Guardar()")
	}
	if err := repositorio.LiberarArrendamientoFlujoFirmaBaremacion(
		ctx,
		puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion{Arrendamiento: soloMetadatos},
	); err == nil {
		t.Fatal("copiar solo los metadatos autorizo Liberar()")
	}

	tokenAjeno, err := puertosbolsa.NuevoTokenArrendamientoFlujoFirmaBaremacion()
	if err != nil {
		t.Fatal(err)
	}
	conTokenAjeno := segundo.Arrendamiento
	conTokenAjeno.Token = tokenAjeno
	if _, err := repositorio.GuardarFlujoFirmaBaremacion(ctx, puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
		VersionEsperada: 1, Arrendamiento: conTokenAjeno, Siguiente: siguiente,
	}); !errors.Is(err, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido) {
		t.Fatalf("Guardar() con token ajeno error = %v", err)
	}
	if err := repositorio.LiberarArrendamientoFlujoFirmaBaremacion(
		ctx,
		puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion{Arrendamiento: conTokenAjeno},
	); !errors.Is(err, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido) {
		t.Fatalf("Liberar() con token ajeno error = %v", err)
	}

	_, err = repositorio.GuardarFlujoFirmaBaremacion(ctx, puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
		VersionEsperada: 1, Arrendamiento: primero.Arrendamiento, Siguiente: siguiente,
	})
	if !errors.Is(err, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido) {
		t.Fatalf("Guardar() con cercado obsoleto error = %v", err)
	}
	if err := repositorio.LiberarArrendamientoFlujoFirmaBaremacion(
		ctx,
		puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion{Arrendamiento: primero.Arrendamiento},
	); !errors.Is(err, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido) {
		t.Fatalf("Liberar() con token obsoleto error = %v", err)
	}
	guardado, err := repositorio.GuardarFlujoFirmaBaremacion(ctx, puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
		VersionEsperada: 1, Arrendamiento: segundo.Arrendamiento, Siguiente: siguiente,
	})
	if err != nil {
		t.Fatalf("Guardar() con cercado vigente error = %v", err)
	}
	if guardado.Version != 2 || len(guardado.PuntosControl) != 1 {
		t.Fatalf("estado guardado inesperado: version=%d puntos=%d", guardado.Version, len(guardado.PuntosControl))
	}
	liberacion := puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion{
		Arrendamiento: segundo.Arrendamiento,
	}
	if err := repositorio.LiberarArrendamientoFlujoFirmaBaremacion(ctx, liberacion); err != nil {
		t.Fatalf("Liberar() con token vigente error = %v", err)
	}
	if err := repositorio.LiberarArrendamientoFlujoFirmaBaremacion(ctx, liberacion); err != nil {
		t.Fatalf("la liberacion idempotente devolvio error = %v", err)
	}
}

type selladorEstadoFlujoMemoriaPrueba struct{}

func (selladorEstadoFlujoMemoriaPrueba) SellarSolicitudBaremacion(
	ctx context.Context,
	carga puertosbolsa.CargaProtegida,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if carga.Validar() != nil {
		return "", puertosbolsa.ErrCargaProtegidaInvalida
	}
	return hmacEstadoFlujoMemoriaPrueba(string(carga.Revelar())), nil
}

func (s selladorEstadoFlujoMemoriaPrueba) VerificarEstadoFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudVerificarEstadoFlujoFirmaBaremacion,
) error {
	if solicitud.Validar() != nil {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	esperado, err := s.SellarSolicitudBaremacion(ctx, solicitud.RepresentacionCanonica)
	if err != nil || !hmac.Equal([]byte(esperado), []byte(solicitud.SelloHMAC)) {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return nil
}

func sellarExpedienteFlujoMemoriaPrueba(
	t *testing.T,
	sellador selladorEstadoFlujoMemoriaPrueba,
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) puertosbolsa.ExpedienteFlujoFirmaBaremacion {
	t.Helper()
	preparado, canonica, err := expediente.PrepararSellado()
	if err != nil {
		t.Fatal(err)
	}
	sello, err := sellador.SellarSolicitudBaremacion(context.Background(), canonica)
	if err != nil {
		t.Fatal(err)
	}
	sellado, err := preparado.IncorporarSello(sello)
	if err != nil {
		t.Fatal(err)
	}
	return sellado
}

func hmacEstadoFlujoMemoriaPrueba(valor string) string {
	mac := hmac.New(sha256.New, claveHMACMemoriaPrueba)
	_, _ = mac.Write([]byte(valor))
	return "hmac-sha256:flujo_firma_memoria_v1:" + hex.EncodeToString(mac.Sum(nil))
}
