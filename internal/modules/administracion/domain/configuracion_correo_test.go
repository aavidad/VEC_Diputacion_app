package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestVistaConfiguracionCorreoValidaParametrosSMTP(t *testing.T) {
	if err := (VistaConfiguracionCorreo{}).Validar(); err != nil {
		t.Fatalf("la vista inicial no configurada debe ser valida: %v", err)
	}
	vista := vistaConfiguracionCorreoPrueba()
	if err := vista.Validar(); err != nil {
		t.Fatalf("la configuracion SMTP valida se rechazo: %v", err)
	}

	for nombre, alterar := range map[string]func(*VistaConfiguracionCorreo){
		"host vacio":                        func(v *VistaConfiguracionCorreo) { v.Host = "" },
		"puerto cero":                       func(v *VistaConfiguracionCorreo) { v.Puerto = 0 },
		"server name no canonico":           func(v *VistaConfiguracionCorreo) { v.NombreServidor = "smtp..intranet.local" },
		"referencia CA vacia":               func(v *VistaConfiguracionCorreo) { v.ReferenciaCA = "" },
		"remitente invalido":                func(v *VistaConfiguracionCorreo) { v.RemitenteFijo = "rrhh" },
		"TLS desconocido":                   func(v *VistaConfiguracionCorreo) { v.ModoTLS = "oportunista" },
		"tiempo no positivo":                func(v *VistaConfiguracionCorreo) { v.TiempoMaximoMillis = 0 },
		"tiempo fuera del limite SQL":       func(v *VistaConfiguracionCorreo) { v.TiempoMaximoMillis = 3600001 },
		"version ausente":                   func(v *VistaConfiguracionCorreo) { v.Version = 0 },
		"modo de autenticacion desconocido": func(v *VistaConfiguracionCorreo) { v.ModoAutenticacion = "oportunista" },
		"xoauth2 sin usuario":               func(v *VistaConfiguracionCorreo) { v.Usuario = "" },
		"xoauth2 sin secreto":               func(v *VistaConfiguracionCorreo) { v.SecretoConfigurado = false },
		"ninguna con usuario": func(v *VistaConfiguracionCorreo) {
			v.ModoAutenticacion = ModoAutenticacionCorreoNinguna
		},
		"plain sin usuario": func(v *VistaConfiguracionCorreo) {
			v.ModoAutenticacion, v.Usuario = ModoAutenticacionCorreoPlain, ""
		},
		"plain sin secreto": func(v *VistaConfiguracionCorreo) {
			v.ModoAutenticacion, v.SecretoConfigurado = ModoAutenticacionCorreoPlain, false
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			invalida := vistaConfiguracionCorreoPrueba()
			alterar(&invalida)
			if err := invalida.Validar(); err != ErrConfiguracionCorreoInvalida {
				t.Fatalf("se esperaba configuracion invalida; recibido %v", err)
			}
		})
	}
}

func TestActualizacionConfiguracionCorreoRespetaLimitesPersistencia(t *testing.T) {
	valida := actualizacionConfiguracionCorreoPrueba(nil, uint64(1<<63-2))
	if err := valida.Validar(); err != nil {
		t.Fatalf("version incrementable rechazada: %v", err)
	}
	valida.VersionEsperada++
	if valida.Validar() == nil {
		t.Fatal("se admite una version que desborda bigint al guardar")
	}
	secreto, err := NuevoSecretoCorreo([]byte("secreto-sintetico"))
	if err != nil {
		t.Fatal(err)
	}
	sinAutenticacion := actualizacionConfiguracionCorreoPrueba(&secreto, 0)
	sinAutenticacion.ModoAutenticacion = ModoAutenticacionCorreoNinguna
	sinAutenticacion.Usuario = ""
	sinAutenticacion.SecretoConfigurado = false
	if sinAutenticacion.Validar() == nil {
		t.Fatal("se admite guardar un secreto para autenticacion desactivada")
	}
	sinAutenticacion.SecretoNuevo = nil
	if err := sinAutenticacion.Validar(); err != nil {
		t.Fatalf("autenticacion desactivada sin secreto rechazada: %v", err)
	}
}

func TestVistaConfiguracionCorreoAceptaLosTresModosDeAutenticacion(t *testing.T) {
	for nombre, preparar := range map[string]func(*VistaConfiguracionCorreo){
		"ninguna": func(v *VistaConfiguracionCorreo) {
			v.ModoAutenticacion, v.Usuario, v.SecretoConfigurado = ModoAutenticacionCorreoNinguna, "", false
		},
		"plain":   func(v *VistaConfiguracionCorreo) { v.ModoAutenticacion = ModoAutenticacionCorreoPlain },
		"xoauth2": func(v *VistaConfiguracionCorreo) { v.ModoAutenticacion = ModoAutenticacionCorreoXOAUTH2 },
	} {
		t.Run(nombre, func(t *testing.T) {
			vista := vistaConfiguracionCorreoPrueba()
			preparar(&vista)
			if err := vista.Validar(); err != nil {
				t.Fatalf("modo de autenticacion valido rechazado: %v", err)
			}
		})
	}
}

func TestSecretoYActualizacionConfiguracionCorreoNuncaExponenElValor(t *testing.T) {
	const valor = "secreto-smtp-sintetico"
	secreto, err := NuevoSecretoCorreo([]byte(valor))
	if err != nil {
		t.Fatalf("no se creo secreto de prueba: %v", err)
	}
	actualizacion := actualizacionConfiguracionCorreoPrueba(&secreto, 4)
	if err := actualizacion.Validar(); err != nil {
		t.Fatalf("la sustitucion explicita se rechazo: %v", err)
	}
	jsonActualizacion := jsonPrueba(t, actualizacion)
	if !strings.Contains(jsonActualizacion, `"modo_autenticacion":"xoauth2"`) || strings.Contains(jsonActualizacion, "modo_oauth") {
		t.Fatalf("JSON de autenticacion no canonico: %s", jsonActualizacion)
	}

	for nombre, representacion := range map[string]string{
		"String secreto":         fmt.Sprint(secreto),
		"GoString secreto":       fmt.Sprintf("%#v", secreto),
		"JSON secreto":           jsonPrueba(t, secreto),
		"String actualizacion":   fmt.Sprint(actualizacion),
		"GoString actualizacion": fmt.Sprintf("%#v", actualizacion),
		"JSON actualizacion":     jsonActualizacion,
	} {
		t.Run(nombre, func(t *testing.T) {
			if strings.Contains(representacion, valor) || strings.Contains(representacion, "SecretoNuevo") {
				t.Fatalf("la representacion expone secreto: %q", representacion)
			}
		})
	}
}

func TestActualizacionConfiguracionCorreoNilConservaSecretoYValorExplicitoLoSustituye(t *testing.T) {
	for _, modo := range []ModoAutenticacionCorreo{ModoAutenticacionCorreoPlain, ModoAutenticacionCorreoXOAUTH2} {
		conservar := actualizacionConfiguracionCorreoPrueba(nil, 4)
		conservar.ModoAutenticacion = modo
		if err := conservar.Validar(); err != nil {
			t.Fatalf("nil debe conservar el secreto existente en %s: %v", modo, err)
		}
	}

	secreto, err := NuevoSecretoCorreo([]byte("reemplazo-smtp-sintetico"))
	if err != nil {
		t.Fatalf("no se creo secreto de sustitucion: %v", err)
	}
	reemplazar := actualizacionConfiguracionCorreoPrueba(nil, 4)
	reemplazar.SecretoNuevo = &secreto
	if err := reemplazar.Validar(); err != nil {
		t.Fatalf("el secreto explicito debe poder sustituir el existente: %v", err)
	}
}

func TestActualizacionConfiguracionCorreoCASPrimeraAltaYNoAceptaVersionDeVista(t *testing.T) {
	secreto, err := NuevoSecretoCorreo([]byte("alta-smtp-sintetica"))
	if err != nil {
		t.Fatalf("no se creo secreto de primera alta: %v", err)
	}
	primeraAlta := actualizacionConfiguracionCorreoPrueba(&secreto, 0)
	if err := primeraAlta.Validar(); err != nil {
		t.Fatalf("la primera alta CAS v0 se rechazo: %v", err)
	}

	versionEnVista := primeraAlta
	versionEnVista.Version = 1
	if err := versionEnVista.Validar(); err != ErrConfiguracionCorreoInvalida {
		t.Fatalf("la actualizacion no debe aceptar version de vista: %v", err)
	}
}

func vistaConfiguracionCorreoPrueba() VistaConfiguracionCorreo {
	return VistaConfiguracionCorreo{
		Configurada: true, Host: "smtp.intranet.local", Puerto: 465, NombreServidor: "smtp.intranet.local",
		ReferenciaCA: "ca:correo-interno:v1", RemitenteFijo: "rrhh@diputacion.example", Usuario: "rrhh-smtp",
		ModoTLS: ModoTLSCorreoImplicito, ModoAutenticacion: ModoAutenticacionCorreoXOAUTH2, TiempoMaximoMillis: 5000, SecretoConfigurado: true, Version: 4,
	}
}

func actualizacionConfiguracionCorreoPrueba(secreto *SecretoCorreo, versionEsperada uint64) ActualizacionConfiguracionCorreo {
	vista := vistaConfiguracionCorreoPrueba()
	vista.Version = 0
	return ActualizacionConfiguracionCorreo{VistaConfiguracionCorreo: vista, VersionEsperada: versionEsperada, SecretoNuevo: secreto}
}

func jsonPrueba(t *testing.T, valor any) string {
	t.Helper()
	datos, err := json.Marshal(valor)
	if err != nil {
		t.Fatalf("no se serializo valor de prueba: %v", err)
	}
	return string(datos)
}
