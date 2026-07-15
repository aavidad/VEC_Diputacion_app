package httpapi

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIdentidadNoInfiereRolNiGarantia(t *testing.T) {
	politica := politicaCabecerasPrueba(t)
	peticion := httptest.NewRequest("GET", "/api/vec/session", nil)
	peticion.RemoteAddr = "10.20.30.8:4321"
	peticion.Header.Set("X-VEC-Subject", "persona-1")
	peticion.Header.Set("X-VEC-Auth-Mechanism", "kerberos_ad")

	identidad := identityFromRequest(peticion, politica)
	if identidad.subject != "persona-1" || len(identidad.roles) != 0 || identidad.assurance != "" {
		t.Fatalf("se infirio autoridad no acreditada: %+v", identidad)
	}
}

func TestIdentidadDeniegaAliasAutoritativosContradictorios(t *testing.T) {
	politica := politicaCabecerasPrueba(t)
	tests := []struct {
		nombre  string
		alterar func(http.Header)
	}{
		{
			nombre: "sujeto",
			alterar: func(h http.Header) {
				h.Set("X-VEC-Subject", "persona-1")
				h.Set("X-Auth-Subject", "persona-2")
			},
		},
		{
			nombre: "mecanismo",
			alterar: func(h http.Header) {
				h.Set("X-VEC-Auth-Mechanism", "kerberos_ad")
				h.Set("X-Auth-Mechanism", "dnie")
			},
		},
		{
			nombre: "roles",
			alterar: func(h http.Header) {
				h.Set("X-VEC-Roles", "administrativo")
				h.Set("X-Auth-Roles", "administrador")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest("GET", "/api/vec/session", nil)
			peticion.RemoteAddr = "10.20.30.8:4321"
			peticion.Header.Set("X-VEC-Subject", "persona-1")
			peticion.Header.Set("X-VEC-Auth-Mechanism", "kerberos_ad")
			peticion.Header.Set("X-VEC-Roles", "administrativo")
			test.alterar(peticion.Header)

			identidad := identityFromRequest(peticion, politica)
			if identidad.subject != "" || len(identidad.roles) != 0 || identidad.method != "" {
				t.Fatalf("alias contradictorios aceptados: %+v", identidad)
			}
		})
	}
}

func TestIdentidadNoNormalizaAfirmacionesAutoritativas(t *testing.T) {
	politica := politicaCabecerasPrueba(t)
	casos := []struct {
		nombre   string
		cabecera string
		valor    string
	}{
		{nombre: "rol con mayusculas", cabecera: "X-VEC-Roles", valor: "ADMINISTRADOR"},
		{nombre: "rol con espacio", cabecera: "X-VEC-Roles", valor: "administrador "},
		{nombre: "roles duplicados", cabecera: "X-VEC-Roles", valor: "administrador,administrador"},
		{nombre: "separador alternativo", cabecera: "X-VEC-Roles", valor: "administrador;tecnico_rrhh"},
		{nombre: "metodo con mayusculas", cabecera: "X-VEC-Auth-Mechanism", valor: "KERBEROS_AD"},
		{nombre: "metodo con espacio", cabecera: "X-VEC-Auth-Mechanism", valor: "kerberos_ad "},
		{nombre: "garantia con mayusculas", cabecera: "X-Auth-Assurance", valor: "ALTO"},
		{nombre: "garantia con espacio", cabecera: "X-Auth-Assurance", valor: "alto "},
		{nombre: "sujeto con espacio", cabecera: "X-VEC-Subject", valor: "persona-1 "},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
			peticion.RemoteAddr = "10.20.30.8:4321"
			peticion.Header.Set("X-VEC-Subject", "persona-1")
			peticion.Header.Set("X-VEC-Roles", "administrador")
			peticion.Header.Set("X-VEC-Auth-Mechanism", "kerberos_ad")
			peticion.Header.Set("X-Auth-Assurance", "alto")
			peticion.Header.Set(caso.cabecera, caso.valor)

			identidad := identityFromRequest(peticion, politica)
			switch caso.cabecera {
			case "X-VEC-Roles", "X-VEC-Subject":
				if identidad.subject != "" || len(identidad.roles) != 0 {
					t.Fatalf("la asercion no canonica se corrigio: %+v", identidad)
				}
			case "X-VEC-Auth-Mechanism":
				if identidad.method != "" {
					t.Fatalf("el metodo no canonico se corrigio: %+v", identidad)
				}
			case "X-Auth-Assurance":
				if identidad.assurance != "" {
					t.Fatalf("la garantia no canonica se corrigio: %+v", identidad)
				}
			}
		})
	}
}

func TestIdentidadDeniegaCabecerasAutoritativasRepetidas(t *testing.T) {
	politica := politicaCabecerasPrueba(t)
	for _, cabecera := range []string{
		"X-VEC-Subject",
		"X-VEC-Roles",
		"X-VEC-Auth-Mechanism",
		"X-Auth-Assurance",
	} {
		cabecera := cabecera
		t.Run(cabecera, func(t *testing.T) {
			peticion := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
			peticion.RemoteAddr = "10.20.30.8:4321"
			peticion.Header.Set("X-VEC-Subject", "persona-1")
			peticion.Header.Set("X-VEC-Roles", "administrador")
			peticion.Header.Set("X-VEC-Auth-Mechanism", "kerberos_ad")
			peticion.Header.Set("X-Auth-Assurance", "alto")
			peticion.Header.Add(cabecera, peticion.Header.Get(cabecera))

			identidad := identityFromRequest(peticion, politica)
			switch cabecera {
			case "X-Auth-Assurance":
				if identidad.assurance != "" {
					t.Fatalf("la garantia repetida se acepto: %+v", identidad)
				}
			default:
				if identidad.subject != "" || len(identidad.roles) != 0 || identidad.method != "" {
					t.Fatalf("la asercion repetida se acepto: %+v", identidad)
				}
			}
		})
	}
}

func TestIdentidadDeniegaCabecerasAuxiliaresAmbiguas(t *testing.T) {
	politica := politicaCabecerasPrueba(t)
	casos := []struct {
		nombre  string
		alterar func(http.Header)
	}{
		{
			nombre: "nombre contradictorio",
			alterar: func(h http.Header) {
				h.Set("X-VEC-Display-Name", "Persona Uno")
				h.Set("X-Auth-Display-Name", "Persona Dos")
			},
		},
		{
			nombre: "correo repetido",
			alterar: func(h http.Header) {
				h.Set("X-VEC-Email", "persona@example.invalid")
				h.Add("X-VEC-Email", "persona@example.invalid")
			},
		},
		{
			nombre: "dni contradictorio",
			alterar: func(h http.Header) {
				h.Set("X-VEC-DNI", "00000000T")
				h.Set("X-Auth-NIF", "11111111H")
			},
		},
		{
			nombre: "alias de sujeto explicitamente vacio",
			alterar: func(h http.Header) {
				h["X-Auth-Subject"] = []string{""}
			},
		},
	}

	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(http.MethodGet, "/api/vec/session", nil)
			peticion.RemoteAddr = "10.20.30.8:4321"
			peticion.Header.Set("X-VEC-Subject", "persona-1")
			peticion.Header.Set("X-VEC-Roles", "administrativo")
			peticion.Header.Set("X-VEC-Auth-Mechanism", "kerberos_ad")
			peticion.Header.Set("X-Auth-Assurance", "alto")
			caso.alterar(peticion.Header)

			identidad := identityFromRequest(peticion, politica)
			if identidad.subject != "" || identidad.displayName != "" || identidad.email != "" ||
				len(identidad.roles) != 0 || identidad.method != "" || len(identidad.attributes) != 0 {
				t.Fatalf("la asercion auxiliar ambigua se acepto: %+v", identidad)
			}
		})
	}
}

func TestCertificadoReenviadoSinAtestacionCriptograficaNoCuenta(t *testing.T) {
	politica := politicaCabecerasPrueba(t)
	peticion := httptest.NewRequest("GET", "/api/vec/session", nil)
	peticion.RemoteAddr = "10.20.30.8:4321"
	peticion.Header.Set("X-VEC-Subject", "persona-1")
	peticion.Header.Set("X-VEC-Auth-Mechanism", "kerberos_ad")
	peticion.Header.Set("X-Auth-Assurance", "sustancial")
	peticion.Header.Set("X-SSL-Client-Verify", "SUCCESS")
	peticion.Header.Set("X-SSL-Client-Cert", "-----BEGIN CERTIFICATE-----\\nfalso\\n-----END CERTIFICATE-----")

	identidad := identityFromRequest(peticion, politica)
	if identidad.subject != "persona-1" || identidad.attributes["auth_source"] != "" ||
		identidad.attributes["certificate_ref"] != "" {
		t.Fatalf("el certificado reenviado adquirio autoridad: %+v", identidad)
	}
}

func politicaCabecerasPrueba(t *testing.T) identityPolicy {
	t.Helper()
	_, red, err := net.ParseCIDR("10.20.30.0/24")
	if err != nil {
		t.Fatal(err)
	}
	return identityPolicy{
		trustHeaders: true, trustedProxies: []*net.IPNet{red},
		subjectHeader: "X-VEC-Subject", rolesHeader: "X-VEC-Roles", mechanismHeader: "X-VEC-Auth-Mechanism",
	}
}
