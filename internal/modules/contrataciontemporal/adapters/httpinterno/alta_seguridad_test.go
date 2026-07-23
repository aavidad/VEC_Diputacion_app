package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestManejadorAltaRechazaContextoAusenteCaducadoYOtraOrganizacion(t *testing.T) {
	casos := []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{"ausente", ErrContextoCanalAusente, http.StatusUnauthorized, "autenticacion_requerida"},
		{"caducado", ErrContextoCanalCaducado, http.StatusUnauthorized, "autenticacion_requerida"},
		{"otra organización", ErrContextoCanalOrganizacionDenegada, http.StatusForbidden, "acceso_denegado"},
		{"autoridad no disponible", ErrContextoCanalNoDisponible, http.StatusServiceUnavailable, "servicio_no_disponible"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
			autoridad.err = errors.Join(caso.err, errors.New("detalle privado de identidad"))
			respuesta := ejecutarPeticionPrueba(t, manejador, nuevaPeticionPrueba(t, cuerpoValidoPrueba()))
			if respuesta.Code != caso.estado || codigoErrorPrueba(t, respuesta) != caso.codigo {
				t.Fatalf("estado/código = %d/%s", respuesta.Code, codigoErrorPrueba(t, respuesta))
			}
			if strings.Contains(respuesta.Body.String(), "detalle privado") {
				t.Fatalf("filtró error privado: %s", respuesta.Body.String())
			}
			if llamadas, _ := ejecutor.instantanea(); llamadas != 0 {
				t.Fatal("ejecutó sin contexto confiable")
			}
			comprobarCabecerasSegurasPrueba(t, respuesta)
		})
	}
}

func TestManejadorAltaNoAceptaContextoFabricadoOConSolicitudInyectada(t *testing.T) {
	casos := []struct {
		nombre    string
		modificar func(*application.SolicitudRegistrarExpediente)
	}{
		{"autenticación inválida", func(c *application.SolicitudRegistrarExpediente) { c.AutenticacionRef = "cliente" }},
		{"sesión inválida", func(c *application.SolicitudRegistrarExpediente) { c.SesionRef = "cliente" }},
		{"perfil inválido", func(c *application.SolicitudRegistrarExpediente) { c.PerfilRef = "cliente" }},
		{"organización inválida", func(c *application.SolicitudRegistrarExpediente) { c.OrganizacionRef = "" }},
		{"clave inyectada por autoridad", func(c *application.SolicitudRegistrarExpediente) {
			c.ClaveIdempotencia = claveIdempotenciaPrueba
		}},
		{"solicitud de autoridad no vacía", func(c *application.SolicitudRegistrarExpediente) {
			var entrada solicitudAltaJSON
			_ = json.Unmarshal(cuerpoValidoPrueba(), &entrada)
			c.Solicitud, _ = entrada.Solicitud.dominio()
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
			caso.modificar(&autoridad.contexto)
			respuesta := ejecutarPeticionPrueba(t, manejador, nuevaPeticionPrueba(t, cuerpoValidoPrueba()))
			if respuesta.Code != http.StatusInternalServerError ||
				codigoErrorPrueba(t, respuesta) != "error_interno" {
				t.Fatalf("contexto fabricado no cerrado: %d %s", respuesta.Code, respuesta.Body.String())
			}
			if llamadas, _ := ejecutor.instantanea(); llamadas != 0 {
				t.Fatal("el contexto fabricado alcanzó el ejecutor")
			}
		})
	}
}

func TestManejadorAltaRechazaInyeccionPorCabecerasQueryYCuerpo(t *testing.T) {
	cabeceras := []string{
		"Cookie", "Authorization", "Proxy-Authorization", "Remote-User",
		"X-Remote-User", "X-Forwarded-User", "X-Forwarded-For", "X-Auth-User",
		"X-Vec-Actor", "X-Role", "Idempotency-Key", "X-HTTP-Method-Override",
		"Content-Encoding", "X-User", "X-Actor", "X-Profile", "X-Organization",
		"X-Session", "X-Account", "X-Permissions", "X-Usuario", "X-Perfil",
		"X-Organizacion", "X-Sesion", "X-Cuenta", "X-Permisos",
	}
	for _, nombre := range cabeceras {
		t.Run("cabecera "+nombre, func(t *testing.T) {
			manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
			peticion := nuevaPeticionPrueba(t, cuerpoValidoPrueba())
			peticion.Header[nombre] = []string{"valor-inyectado"}
			respuesta := ejecutarPeticionPrueba(t, manejador, peticion)
			if respuesta.Code != http.StatusBadRequest ||
				codigoErrorPrueba(t, respuesta) != "peticion_no_permitida" {
				t.Fatalf("%s aceptada: %d %s", nombre, respuesta.Code, respuesta.Body.String())
			}
			if autoridad.numeroLlamadas() != 0 {
				t.Fatal("la cabecera alcanzó la autoridad")
			}
			if llamadas, _ := ejecutor.instantanea(); llamadas != 0 {
				t.Fatal("la cabecera alcanzó el ejecutor")
			}
		})
	}
	for _, campo := range []string{
		`"actor_ref":"actor:cliente"`,
		`"perfil_ref":"perfil:cliente"`,
		`"organizacion_ref":"organizacion:cliente"`,
		`"rol":"administrador"`,
	} {
		t.Run("cuerpo "+campo, func(t *testing.T) {
			manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
			cuerpo := bytes.TrimSpace(cuerpoValidoPrueba())
			cuerpo = append(append(append([]byte(nil), cuerpo[:len(cuerpo)-1]...), ','), []byte(campo+"}")...)
			respuesta := ejecutarPeticionPrueba(t, manejador, nuevaPeticionPrueba(t, cuerpo))
			if respuesta.Code != http.StatusBadRequest {
				t.Fatalf("campo de autoridad aceptado: %d %s", respuesta.Code, respuesta.Body.String())
			}
			if autoridad.numeroLlamadas() != 0 {
				t.Fatal("el cuerpo alcanzó la autoridad")
			}
			if llamadas, _ := ejecutor.instantanea(); llamadas != 0 {
				t.Fatal("el cuerpo alcanzó el ejecutor")
			}
		})
	}
	for _, clave := range []string{
		"",
		"reintentar",
		"00000000-0000-4000-8000-000000000000",
		"4D36E96E-E325-4F9B-BEBC-291D91D6F732",
	} {
		t.Run("clave de intención inválida "+clave, func(t *testing.T) {
			cuerpo := bytes.Replace(
				cuerpoValidoPrueba(),
				[]byte(claveIdempotenciaPrueba),
				[]byte(clave),
				1,
			)
			exigirRechazoLocalPrueba(t, cuerpo, http.StatusUnprocessableEntity)
		})
	}
}

func TestManejadorAltaPropagaCancelacionYPlazoAntesDelEjecutor(t *testing.T) {
	t.Run("cancelado antes de autoridad", func(t *testing.T) {
		manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		peticion := nuevaPeticionPrueba(t, cuerpoValidoPrueba()).WithContext(ctx)
		respuesta := ejecutarPeticionPrueba(t, manejador, peticion)
		if respuesta.Code != http.StatusRequestTimeout ||
			codigoErrorPrueba(t, respuesta) != "peticion_cancelada" {
			t.Fatalf("cancelación = %d %s", respuesta.Code, respuesta.Body.String())
		}
		if autoridad.numeroLlamadas() != 0 {
			t.Fatal("llamó autoridad con contexto cancelado")
		}
		if llamadas, _ := ejecutor.instantanea(); llamadas != 0 {
			t.Fatal("llamó ejecutor con contexto cancelado")
		}
	})
	t.Run("cancelado por autoridad", func(t *testing.T) {
		manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		autoridad.alLlamar = cancelar
		peticion := nuevaPeticionPrueba(t, cuerpoValidoPrueba()).WithContext(ctx)
		respuesta := ejecutarPeticionPrueba(t, manejador, peticion)
		if respuesta.Code != http.StatusRequestTimeout {
			t.Fatalf("cancelación = %d %s", respuesta.Code, respuesta.Body.String())
		}
		if llamadas, _ := ejecutor.instantanea(); llamadas != 0 {
			t.Fatal("llamó ejecutor tras cancelación")
		}
	})
	t.Run("cancelado durante lectura", func(t *testing.T) {
		manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
		ctx, cancelar := context.WithCancel(context.Background())
		peticion := nuevaPeticionPrueba(t, cuerpoValidoPrueba()).WithContext(ctx)
		peticion.Body = io.NopCloser(&lectorCanceladorPrueba{
			contenido: cuerpoValidoPrueba(),
			cancelar:  cancelar,
		})
		peticion.ContentLength = int64(len(cuerpoValidoPrueba()))
		respuesta := ejecutarPeticionPrueba(t, manejador, peticion)
		if respuesta.Code != http.StatusRequestTimeout ||
			codigoErrorPrueba(t, respuesta) != "peticion_cancelada" {
			t.Fatalf("cancelación durante lectura = %d %s", respuesta.Code, respuesta.Body.String())
		}
		if autoridad.numeroLlamadas() != 0 {
			t.Fatal("llamó autoridad tras cancelación durante lectura")
		}
		if llamadas, _ := ejecutor.instantanea(); llamadas != 0 {
			t.Fatal("llamó ejecutor tras cancelación durante lectura")
		}
	})
	t.Run("plazo agotado", func(t *testing.T) {
		manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
		ctx, cancelar := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancelar()
		peticion := nuevaPeticionPrueba(t, cuerpoValidoPrueba()).WithContext(ctx)
		respuesta := ejecutarPeticionPrueba(t, manejador, peticion)
		if respuesta.Code != http.StatusGatewayTimeout ||
			codigoErrorPrueba(t, respuesta) != "plazo_agotado" {
			t.Fatalf("plazo = %d %s", respuesta.Code, respuesta.Body.String())
		}
		if autoridad.numeroLlamadas() != 0 {
			t.Fatal("llamó autoridad con plazo agotado")
		}
		if llamadas, _ := ejecutor.instantanea(); llamadas != 0 {
			t.Fatal("llamó ejecutor con plazo agotado")
		}
	})
}

type lectorCanceladorPrueba struct {
	contenido []byte
	cancelar  context.CancelFunc
	leido     bool
}

func (l *lectorCanceladorPrueba) Read(destino []byte) (int, error) {
	if l.leido {
		return 0, io.EOF
	}
	l.leido = true
	l.cancelar()
	return copy(destino, l.contenido), nil
}

func TestManejadorAltaClasificaResultadoPendienteSinInducirReintento(t *testing.T) {
	for _, causa := range []error{
		errors.New("fallo privado posterior a COMMIT"),
		context.Canceled,
		context.DeadlineExceeded,
	} {
		t.Run(causa.Error(), func(t *testing.T) {
			manejador, _, ejecutor := nuevoEscenarioPrueba(t)
			ejecutor.recibo = ports.ReciboAlta{}
			ejecutor.err = errors.Join(ErrResultadoAltaIndeterminado, causa)
			respuesta := ejecutarPeticionPrueba(t, manejador, nuevaPeticionPrueba(t, cuerpoValidoPrueba()))
			if respuesta.Code != http.StatusServiceUnavailable ||
				codigoErrorPrueba(t, respuesta) != "operacion_pendiente" {
				t.Fatalf("resultado pendiente = %d %s", respuesta.Code, respuesta.Body.String())
			}
			if respuesta.Header().Get("Retry-After") != "" ||
				strings.Contains(respuesta.Body.String(), "COMMIT") {
				t.Fatalf("respuesta induce reintento o filtra detalle: %v %s", respuesta.Header(), respuesta.Body.String())
			}
		})
	}
}

func TestManejadorAltaRechazaReciboIncompletoAdulteradoFuturoONoLigado(t *testing.T) {
	casos := []struct {
		nombre string
		recibo ports.ReciboAlta
		err    error
		codigo string
		estado int
	}{
		{"incompleto", ports.ReciboAlta{}, nil, "resultado_no_confiable", http.StatusBadGateway},
		{"sin auditoría interna", func() ports.ReciboAlta {
			r := reciboValidoPrueba()
			r.AuditoriaRef = ""
			return r
		}(), nil, "resultado_no_confiable", http.StatusBadGateway},
		{"versión no interoperable", func() ports.ReciboAlta {
			r := reciboValidoPrueba()
			r.Version = MaximoVersionJSON + 1
			return r
		}(), nil, "resultado_no_confiable", http.StatusBadGateway},
		{"futuro", func() ports.ReciboAlta {
			r := reciboValidoPrueba()
			r.ConfirmadaEn = instantePrueba.Add(time.Microsecond)
			return r
		}(), nil, "resultado_no_confiable", http.StatusBadGateway},
		{"no ligado detectado por aplicación", ports.ReciboAlta{}, application.ErrResultadoRegistroNoConfiable, "resultado_no_confiable", http.StatusBadGateway},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manejador, _, ejecutor := nuevoEscenarioPrueba(t)
			ejecutor.recibo, ejecutor.err = caso.recibo, caso.err
			respuesta := ejecutarPeticionPrueba(t, manejador, nuevaPeticionPrueba(t, cuerpoValidoPrueba()))
			if respuesta.Code != caso.estado || codigoErrorPrueba(t, respuesta) != caso.codigo {
				t.Fatalf("recibo inseguro = %d %s", respuesta.Code, respuesta.Body.String())
			}
			if strings.Contains(respuesta.Body.String(), "auditoria") ||
				strings.Contains(respuesta.Body.String(), "evento") {
				t.Fatalf("filtró evidencia interna: %s", respuesta.Body.String())
			}
		})
	}
}

func TestManejadorAltaEliminaCabecerasHeredadasYErroresPrivados(t *testing.T) {
	manejador, _, ejecutor := nuevoEscenarioPrueba(t)
	detallePrivado := "detalle_interno_no_publicable"
	ejecutor.recibo = ports.ReciboAlta{}
	ejecutor.err = errors.New(detallePrivado)
	respuesta := httptest.NewRecorder()
	respuesta.Header().Set("Set-Cookie", "sesion=autoridad")
	respuesta.Header().Set("Access-Control-Allow-Origin", "*")
	respuesta.Header().Set("Access-Control-Allow-Credentials", "true")
	respuesta.Header().Set("Content-Encoding", "gzip")
	respuesta.Header().Set("Location", "https://ejemplo.invalid/privado")
	manejador.ServeHTTP(respuesta, nuevaPeticionPrueba(t, cuerpoValidoPrueba()))
	if respuesta.Code != http.StatusInternalServerError ||
		codigoErrorPrueba(t, respuesta) != "error_interno" {
		t.Fatalf("error privado = %d %s", respuesta.Code, respuesta.Body.String())
	}
	for nombre, valores := range respuesta.Header() {
		if strings.Contains(strings.Join(valores, ","), detallePrivado) {
			t.Fatalf("secreto en cabecera %s", nombre)
		}
	}
	if strings.Contains(respuesta.Body.String(), detallePrivado) {
		t.Fatalf("secreto en cuerpo: %s", respuesta.Body.String())
	}
	comprobarCabecerasSegurasPrueba(t, respuesta)
	if respuesta.Header().Get("Location") != "" {
		t.Fatalf("Location heredada: %q", respuesta.Header().Get("Location"))
	}
}

func TestManejadorAltaDobleEnvioPropagaClaveSinExponerla(t *testing.T) {
	manejador, _, ejecutor := nuevoEscenarioPrueba(t)
	for intento := 0; intento < 2; intento++ {
		respuesta := ejecutarPeticionPrueba(t, manejador, nuevaPeticionPrueba(t, cuerpoValidoPrueba()))
		if respuesta.Code != http.StatusCreated {
			t.Fatalf("intento %d = %d %s", intento, respuesta.Code, respuesta.Body.String())
		}
		if strings.Contains(respuesta.Body.String(), claveIdempotenciaPrueba) {
			t.Fatal("expuso clave de idempotencia")
		}
		for _, valores := range respuesta.Header() {
			if strings.Contains(strings.Join(valores, ","), claveIdempotenciaPrueba) {
				t.Fatal("expuso clave de idempotencia en cabecera")
			}
		}
	}
	llamadas, comandos := ejecutor.instantanea()
	if llamadas != 2 || len(comandos) != 2 ||
		comandos[0].ClaveIdempotencia != comandos[1].ClaveIdempotencia {
		t.Fatalf("doble envío no conservó clave: %+v", comandos)
	}
}

func TestManejadorAltaEsSeguroEnCarreraYNoConservaEstadoGlobal(t *testing.T) {
	manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
	const concurrencia = 64
	var grupo sync.WaitGroup
	errores := make(chan string, concurrencia)
	for indice := 0; indice < concurrencia; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			respuesta := httptest.NewRecorder()
			peticion := httptest.NewRequest(
				http.MethodPost,
				RutaAltaSolicitudes,
				bytes.NewReader(cuerpoValidoPrueba()),
			)
			peticion.Header.Set("Content-Type", "application/json")
			peticion.Header.Set("Accept", "*/*")
			manejador.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusCreated {
				errores <- respuesta.Body.String()
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Errorf("respuesta concurrente: %s", err)
	}
	llamadas, _ := ejecutor.instantanea()
	if autoridad.numeroLlamadas() != concurrencia || llamadas != concurrencia {
		t.Fatalf("llamadas concurrentes autoridad/ejecutor = %d/%d", autoridad.numeroLlamadas(), llamadas)
	}
}
