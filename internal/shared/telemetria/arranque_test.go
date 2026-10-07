package telemetria

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestRegistrarArranqueDistingueComposicionDeEscucha(t *testing.T) {
	var destino bytes.Buffer
	RegistrarArranque(&destino, EventoArranque{
		Servicio: "vec-server", Superficie: "interno", Entorno: "desarrollo",
		Fase: "composicion", Resultado: "preparada", Duracion: 45 * time.Millisecond,
	})
	var got map[string]any
	if err := json.Unmarshal(destino.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["msg"] != "vec.process.startup" || got["vec.arranque.fase"] != "composicion" ||
		got["vec.arranque.resultado"] != "preparada" || got["vec.arranque.duracion"] != 0.045 ||
		got["error.type"] != nil {
		t.Fatalf("hito de composición inesperado: %v", got)
	}
}

func TestRegistrarArranqueNoExponeDatosDeErrorNiConfiguracion(t *testing.T) {
	secreto := "/ruta/privada?password=irrepetible"
	var destino bytes.Buffer
	RegistrarArranque(&destino, EventoArranque{
		Servicio: secreto, Superficie: secreto, Entorno: secreto,
		Fase: secreto, Resultado: secreto, Causa: secreto,
	})
	if strings.Contains(destino.String(), secreto) || strings.Contains(destino.String(), "irrepetible") {
		t.Fatalf("dato privado en registro técnico: %s", destino.String())
	}
	var got map[string]any
	if err := json.Unmarshal(destino.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["error.type"] != "otro" || got["vec.arranque.resultado"] != "fallida" {
		t.Fatalf("causa no saneada: %v", got)
	}
	if ClaseErrorArranque(errors.New(secreto)) != "otro" {
		t.Fatal("texto de error aceptado como causa")
	}
}

type falloConfiguracionArranquePrueba struct{ error }

func (falloConfiguracionArranquePrueba) ClaseErrorArranque() string { return "configuracion" }

func TestClaseErrorArranqueAdmiteMarcadorTipadoSinPerderCausasPrevias(t *testing.T) {
	causa := falloConfiguracionArranquePrueba{errors.New("ruta privada sintetica")}
	if got := ClaseErrorArranque(fmt.Errorf("composicion: %w", causa)); got != "configuracion" {
		t.Fatalf("clase de fuente configurada: %q", got)
	}
	if got := ClaseErrorArranque(errors.Join(causa, context.Canceled)); got != "cancelada" {
		t.Fatalf("cancelacion sustituida por configuracion: %q", got)
	}
	if got := MensajeErrorArranque(errors.New("bootstrap: fuente configurada no disponible")); got != "bootstrap: fuente configurada no disponible" {
		t.Fatalf("mensaje fijo rechazado: %q", got)
	}
}

func TestMensajeErrorArranqueConservaDiagnosticosInternosAnidados(t *testing.T) {
	material := errors.New("bootstrap: material criptografico de desarrollo invalido")
	if got := MensajeErrorArranque(fmt.Errorf("composicion: %w", material)); got != "composicion: "+material.Error() {
		t.Fatalf("causa de material perdida: %q", got)
	}
	if got := MensajeErrorArranque(fmt.Errorf("%w: ruta enlazada o no canonica", material)); got != material.Error()+": ruta enlazada o no canonica" {
		t.Fatalf("detalle interno de material perdido: %q", got)
	}
	if got := MensajeErrorArranque(errors.New("auditoria.intentos.configuracion_no_disponible")); got != "auditoria.intentos.configuracion_no_disponible" {
		t.Fatalf("componente de auditoria perdido: %q", got)
	}
	admin := errors.New("administracion: configuracion no valida")
	pg := &pgconn.PgError{Code: "42501", Message: "Juan Perez", Detail: "password=secreto", Hint: "postgres://usuario:clave@host/base", Routine: "Juan_Perez", ConstraintName: "dni_12345678Z"}
	got := MensajeErrorArranque(errors.Join(admin, fmt.Errorf("auditoria_intentos: %w", pg)))
	if !strings.Contains(got, admin.Error()) || !strings.Contains(got, "PostgreSQL bd_42501") {
		t.Fatalf("diagnostico ADMIN y SQLSTATE perdidos: %q", got)
	}
	for _, privado := range []string{"Juan", "Perez", "password", "secreto", "postgres://", "usuario", "clave", "host", "dni_12345678Z"} {
		if strings.Contains(got, privado) {
			t.Fatalf("dato privado %q en diagnostico: %q", privado, got)
		}
	}
}

func TestMensajeErrorArranqueDescomponeSistemaSinRutasNiRed(t *testing.T) {
	privado := "secreto-personal"
	for _, caso := range []struct {
		nombre string
		err    error
		quiere string
	}{
		{"archivo", &os.PathError{Op: "open", Path: "/privado/" + privado + "/clave.pem", Err: syscall.EACCES}, "archivo open: sistema errno=13"},
		{"url", &url.Error{Op: "Get", URL: "https://usuario:clave@" + privado + ".invalid/persona", Err: &net.OpError{Op: "dial", Net: "tcp", Addr: &net.TCPAddr{IP: net.IPv4(10, 2, 3, 4), Port: 443}, Err: syscall.ECONNREFUSED}}, "URL get: red dial: sistema errno=111"},
		{"dns", &net.DNSError{Name: privado + ".invalid", Server: "10.2.3.4", IsNotFound: true}, "DNS: nombre no encontrado"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			got := MensajeErrorArranque(caso.err)
			if !strings.Contains(got, caso.quiere) || strings.Contains(got, privado) || strings.Contains(got, "10.2.3.4") || strings.Contains(got, "usuario") {
				t.Fatalf("diagnostico inseguro o incompleto: %q", got)
			}
		})
	}
}

func TestMensajeErrorArranqueNoPublicaTextoLibreNiControl(t *testing.T) {
	for _, valor := range []string{
		"fallo de Juan Perez con DNI 12345678Z",
		"postgres://usuario:password@interno/base",
		"password=secreto /ruta/privada",
		"otro\nERROR falso: acceso concedido",
	} {
		got := MensajeErrorArranque(errors.New(valor))
		if got != "error interno no catalogado tipo=*errors.errorString" || strings.ContainsAny(got, "\r\n\x00") {
			t.Fatalf("texto libre publicado: %q", got)
		}
	}
}
