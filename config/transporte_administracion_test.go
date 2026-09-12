package config

import (
	"errors"
	"testing"
)

func transporteAdministracionPrueba() ConfiguracionTransporteAdministracion {
	return ConfiguracionTransporteAdministracion{
		Address:          "127.0.0.1:9443",
		HTTPAllowedCIDRs: []string{"127.0.0.1/32"},
		AllowedOrigins:   []string{"https://admin.test"},
		TLSCertFile:      "/estado/administracion/servidor.pem",
		TLSKeyFile:       "/estado/administracion/servidor.key",
		TLSClientCAFile:  "/estado/administracion/clientes-ca.pem",
	}
}

func TestTransporteAdministracionAusenteParcialYSeparadoDeRRHH(t *testing.T) {
	base := Config{Address: "127.0.0.1:8443", HTTPAllowedCIDRs: []string{"127.0.0.1/32"}, TLSCertFile: "/rrhh/cert.pem", TLSKeyFile: "/rrhh/key.pem"}
	if _, err := (ConfiguracionTransporteAdministracion{}).ConfiguracionServidor(base); !errors.Is(err, ErrTransporteAdministracionDesactivado) {
		t.Fatalf("transporte ausente = %v; se esperaba desactivado", err)
	}
	parcial := transporteAdministracionPrueba()
	parcial.TLSClientCAFile = ""
	if _, err := parcial.ConfiguracionServidor(base); !errors.Is(err, ErrTransporteAdministracionIncompleto) {
		t.Fatalf("transporte parcial = %v; se esperaba incompleto", err)
	}
	misma := transporteAdministracionPrueba()
	misma.Address = base.Address
	if _, err := misma.ConfiguracionServidor(base); !errors.Is(err, ErrTransporteAdministracionInvalido) {
		t.Fatalf("listener compartido = %v; se esperaba denegado", err)
	}
	proyeccion, err := transporteAdministracionPrueba().ConfiguracionServidor(base)
	if err != nil {
		t.Fatal(err)
	}
	if proyeccion.Address != "127.0.0.1:9443" || proyeccion.TLSCertFile != "/estado/administracion/servidor.pem" || proyeccion.TLSKeyFile != "/estado/administracion/servidor.key" || len(proyeccion.HTTPAllowedCIDRs) != 1 || proyeccion.HTTPAllowedCIDRs[0] != "127.0.0.1/32" || len(proyeccion.HTTPAllowedOrigins) != 1 || proyeccion.HTTPAllowedOrigins[0] != "https://admin.test" {
		t.Fatalf("la proyeccion ADMIN heredo o perdio valores: %+v", proyeccion)
	}
	if proyeccion.AdministracionPostgreSQL != base.AdministracionPostgreSQL {
		t.Fatal("la proyeccion no puede borrar las conexiones ADMIN existentes")
	}
}

func TestTransporteAdministracionDesarrolloExigeLoopback(t *testing.T) {
	base := Config{Address: "127.0.0.1:8443", ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}
	for nombre, mutar := range map[string]func(*ConfiguracionTransporteAdministracion){
		"direccion no local": func(c *ConfiguracionTransporteAdministracion) { c.Address = "10.0.0.8:9443" },
		"red no local":       func(c *ConfiguracionTransporteAdministracion) { c.HTTPAllowedCIDRs = []string{"10.0.0.0/8"} },
	} {
		t.Run(nombre, func(t *testing.T) {
			transporte := transporteAdministracionPrueba()
			mutar(&transporte)
			if _, err := transporte.ConfiguracionServidor(base); !errors.Is(err, ErrTransporteAdministracionInvalido) {
				t.Fatalf("configuracion de desarrollo = %v; se esperaba denegada", err)
			}
		})
	}
}
