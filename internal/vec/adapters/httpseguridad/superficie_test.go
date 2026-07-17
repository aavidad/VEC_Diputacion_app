package httpseguridad

import (
	"errors"
	"net/netip"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestValidarConjuntoSuperficiesSeparado(t *testing.T) {
	configuraciones := configuracionesValidas()
	t.Run("el portal anonimo y el area personal comparten listener exterior", func(t *testing.T) {
		if configuraciones[0].DireccionEscucha != configuraciones[1].DireccionEscucha {
			t.Fatal("la prueba debe representar un unico listener para el portal exterior")
		}
		if err := ValidarConjuntoSuperficies(configuraciones); err != nil {
			t.Fatalf("el conjunto seguro con listener exterior compartido debe ser valido: %v", err)
		}
		if err := ValidarArquitecturaCompleta(configuraciones); err != nil {
			t.Fatalf("la arquitectura completa debe ser valida: %v", err)
		}
	})
	if err := ValidarArquitecturaCompleta(configuraciones[:3]); !errors.Is(err, ErrConfiguracionSuperficie) {
		t.Fatalf("una arquitectura parcial debe denegarse: %v", err)
	}

	pruebas := []struct {
		nombre    string
		modificar func([]ConfiguracionSuperficie)
	}{
		{
			nombre: "audiencia compartida",
			modificar: func(c []ConfiguracionSuperficie) {
				c[3].Audiencia = c[2].Audiencia
			},
		},
		{
			nombre: "listener interno compartido con el exterior",
			modificar: func(c []ConfiguracionSuperficie) {
				c[2].DireccionEscucha = "10.40.0.20:8080"
			},
		},
		{
			nombre: "listener de administracion compartido con el exterior",
			modificar: func(c []ConfiguracionSuperficie) {
				c[3].DireccionEscucha = "10.50.0.20:8080"
			},
		},
		{
			nombre: "clases exteriores en listeners diferentes",
			modificar: func(c []ConfiguracionSuperficie) {
				c[1].DireccionEscucha = ":8081"
			},
		},
		{
			nombre: "aliases de listener exterior ambiguos",
			modificar: func(c []ConfiguracionSuperficie) {
				c[1].DireccionEscucha = "0.0.0.0:8080"
			},
		},
		{
			nombre: "superficie duplicada",
			modificar: func(c []ConfiguracionSuperficie) {
				c[1].Superficie = c[0].Superficie
				c[1].Audiencia = ""
				c[1].EmisorIdentidad = ""
				c[1].HuellasProxyTLSPermitidas = nil
				c[1].IdentidadesSANProxyPermitidas = nil
				c[1].DuracionMaximaAsercion = 0
				c[1].ToleranciaReloj = 0
				c[1].MetodosAdmitidos = nil
				c[1].FactoresRequeridos = nil
				c[1].MinimoFactoresVerificados = 0
				c[1].MinimoGruposCriptograficosDistintos = 0
				c[1].GarantiaMinima = ""
				c[1].PermiteAnonimo = true
			},
		},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			copia := append([]ConfiguracionSuperficie(nil), configuraciones...)
			prueba.modificar(copia)
			if err := ValidarConjuntoSuperficies(copia); !errors.Is(err, ErrSuperficiesCompartidas) {
				t.Fatalf("se esperaba limite compartido, recibido %v", err)
			}
		})
	}
}

func TestConfiguracionSuperficieFallaCerrada(t *testing.T) {
	basePersonal := configuracionPersonalValida()
	baseInterna := configuracionInternaValida()
	baseAdmin := configuracionAdministracionValida()
	basePublica := configuracionPublicaValida()

	pruebas := []struct {
		nombre string
		valor  ConfiguracionSuperficie
	}{
		{"superficie desconocida", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.Superficie = "otra" })},
		{"zona desconocida", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.ZonaRed = "otra" })},
		{"zona interna expuesta como publica", cambiar(baseInterna, func(c *ConfiguracionSuperficie) { c.ZonaRed = ZonaRedPublica })},
		{"direccion ausente", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.DireccionEscucha = "" })},
		{"direccion mal formada", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.DireccionEscucha = "8081" })},
		{"listener con nombre resoluble", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.DireccionEscucha = "localhost:8080" })},
		{"listener con servicio en vez de puerto", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.DireccionEscucha = ":https" })},
		{"listener con puerto cero", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.DireccionEscucha = ":0" })},
		{"listener interno en todas las interfaces", cambiar(baseInterna, func(c *ConfiguracionSuperficie) { c.DireccionEscucha = "0.0.0.0:8443" })},
		{"listener de administracion en todas las interfaces", cambiar(baseAdmin, func(c *ConfiguracionSuperficie) { c.DireccionEscucha = "[::]:9443" })},
		{"audiencia ausente", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.Audiencia = "" })},
		{"audiencia no canonica", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.Audiencia = " vec-personal " })},
		{"redes ausentes", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.RedesPermitidas = nil })},
		{"red mal formada", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.RedesPermitidas = []string{"10.0.0.0/no"} })},
		{"interna con IPv4 universal", cambiar(baseInterna, func(c *ConfiguracionSuperficie) { c.RedesPermitidas = []string{"0.0.0.0/0"} })},
		{"interna con IPv4 universal mapeada", cambiar(baseInterna, func(c *ConfiguracionSuperficie) { c.RedesPermitidas = []string{"::ffff:0:0/96"} })},
		{"administracion con IPv6 universal", cambiar(baseAdmin, func(c *ConfiguracionSuperficie) { c.RedesPermitidas = []string{"::/0"} })},
		{"tolerancia negativa", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.ToleranciaReloj = -time.Second })},
		{"tolerancia superior a dos minutos", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.ToleranciaReloj = 2*time.Minute + time.Second })},
		{"duracion superior a cinco minutos", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.DuracionMaximaAsercion = 5*time.Minute + time.Second })},
		{"anonima con audiencia", cambiar(basePublica, func(c *ConfiguracionSuperficie) { c.Audiencia = "vec-publico" })},
		{"anonima con garantia", cambiar(basePublica, func(c *ConfiguracionSuperficie) { c.GarantiaMinima = dominiovec.AuthAssuranceHigh })},
		{"anonima con emisor", cambiar(basePublica, func(c *ConfiguracionSuperficie) { c.EmisorIdentidad = "idp" })},
		{"anonima con tolerancia de sesion", cambiar(basePublica, func(c *ConfiguracionSuperficie) { c.ToleranciaReloj = time.Second })},
		{"personal permite anonimo", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.PermiteAnonimo = true })},
		{"personal sin emisor", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.EmisorIdentidad = "" })},
		{"personal con emisor sin TLS", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.EmisorIdentidad = "http://identidad.example.test" })},
		{"personal con emisor ambiguo", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.EmisorIdentidad += "?otro=1" })},
		{"personal sin duracion", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.DuracionMaximaAsercion = 0 })},
		{"personal sin garantia", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.GarantiaMinima = "" })},
		{"personal con garantia baja", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.GarantiaMinima = dominiovec.AuthAssuranceLow })},
		{"personal sin identidad proxy TLS", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.IdentidadesSANProxyPermitidas = nil })},
		{"SAN proxy libre", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.IdentidadesSANProxyPermitidas = []string{"proxy.example"} })},
		{"huella proxy mal formada", cambiar(basePersonal, func(c *ConfiguracionSuperficie) {
			c.IdentidadesSANProxyPermitidas = nil
			c.HuellasProxyTLSPermitidas = []string{"sha256:corta"}
		})},
		{"metodos admitidos ausentes", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.MetodosAdmitidos = nil })},
		{"factor desconocido", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.MetodosAdmitidos = []MetodoAutenticacion{"magico"} })},
		{"metodo admitido duplicado", cambiar(basePersonal, func(c *ConfiguracionSuperficie) {
			c.MetodosAdmitidos = []MetodoAutenticacion{MetodoClave, MetodoClave}
		})},
		{"factor requerido no admitido", cambiar(basePersonal, func(c *ConfiguracionSuperficie) {
			c.FactoresRequeridos = []MetodoAutenticacion{MetodoKerberos}
		})},
		{"factor requerido duplicado", cambiar(basePersonal, func(c *ConfiguracionSuperficie) {
			c.FactoresRequeridos = []MetodoAutenticacion{MetodoClave, MetodoClave}
			c.MinimoFactoresVerificados = 2
		})},
		{"minimo de factores ausente", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.MinimoFactoresVerificados = 0 })},
		{"grupos superiores a factores", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.MinimoGruposCriptograficosDistintos = 2 })},
		{"interna sin certificado", cambiar(baseInterna, func(c *ConfiguracionSuperficie) {
			c.FactoresRequeridos = []MetodoAutenticacion{MetodoKerberos}
			c.MinimoFactoresVerificados = 1
			c.MinimoGruposCriptograficosDistintos = 1
		})},
		{"interna comparte grupo", cambiar(baseInterna, func(c *ConfiguracionSuperficie) { c.MinimoGruposCriptograficosDistintos = 1 })},
		{"interna sin garantia alta", cambiar(baseInterna, func(c *ConfiguracionSuperficie) { c.GarantiaMinima = dominiovec.AuthAssuranceSubstantial })},
		{"admin sin cuenta privilegiada", cambiar(baseAdmin, func(c *ConfiguracionSuperficie) { c.RequiereCuentaPrivilegiada = false })},
		{"cuenta privilegiada en personal", cambiar(basePersonal, func(c *ConfiguracionSuperficie) { c.RequiereCuentaPrivilegiada = true })},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			if err := prueba.valor.Validar(); !errors.Is(err, ErrConfiguracionSuperficie) {
				t.Fatalf("se esperaba configuracion no valida, recibido %v", err)
			}
		})
	}
}

func TestPoliticaRedUsaSoloDireccionRealDelPar(t *testing.T) {
	politica, err := NuevaPoliticaRed(configuracionInternaValida())
	if err != nil {
		t.Fatalf("crear politica: %v", err)
	}
	if err := politica.Autorizar(netip.MustParseAddr("10.40.2.9")); err != nil {
		t.Fatalf("la red Mulhacen configurada debe pasar: %v", err)
	}

	denegados := []netip.Addr{
		netip.MustParseAddr("198.51.100.8"),
		{},
	}
	for _, direccion := range denegados {
		if err := politica.Autorizar(direccion); !errors.Is(err, ErrRedNoAutorizada) {
			t.Fatalf("el origen %#v debia denegarse: %v", direccion, err)
		}
	}

	incoherente := configuracionInternaValida()
	incoherente.ZonaRed = ZonaRedPublica
	if _, err := NuevaPoliticaRed(incoherente); !errors.Is(err, ErrConfiguracionSuperficie) {
		t.Fatalf("el constructor directo debe comprobar superficie-zona: %v", err)
	}
}

func configuracionesValidas() []ConfiguracionSuperficie {
	return []ConfiguracionSuperficie{
		configuracionPublicaValida(),
		configuracionPersonalValida(),
		configuracionInternaValida(),
		configuracionAdministracionValida(),
	}
}

func configuracionPublicaValida() ConfiguracionSuperficie {
	return ConfiguracionSuperficie{
		Superficie:       SuperficiePublicaAnonima,
		ZonaRed:          ZonaRedPublica,
		DireccionEscucha: ":8080",
		RedesPermitidas:  []string{"0.0.0.0/0", "::/0"},
		PermiteAnonimo:   true,
	}
}

func configuracionPersonalValida() ConfiguracionSuperficie {
	return ConfiguracionSuperficie{
		Superficie:                          SuperficieExternaPersonal,
		ZonaRed:                             ZonaRedPublica,
		DireccionEscucha:                    ":8080",
		Audiencia:                           "vec-personal",
		EmisorIdentidad:                     "https://identidad.example.test",
		RedesPermitidas:                     []string{"0.0.0.0/0", "::/0"},
		IdentidadesSANProxyPermitidas:       []string{"dns:proxy-personal.vec.test"},
		DuracionMaximaAsercion:              5 * time.Minute,
		ToleranciaReloj:                     30 * time.Second,
		MetodosAdmitidos:                    []MetodoAutenticacion{MetodoClave, MetodoCertificado, MetodoDNIe},
		MinimoFactoresVerificados:           1,
		MinimoGruposCriptograficosDistintos: 1,
		GarantiaMinima:                      dominiovec.AuthAssuranceSubstantial,
	}
}

func configuracionInternaValida() ConfiguracionSuperficie {
	return ConfiguracionSuperficie{
		Superficie:                          SuperficieInternaCorporativa,
		ZonaRed:                             ZonaRedInterna,
		DireccionEscucha:                    "10.40.0.20:8443",
		Audiencia:                           "vec-interna",
		EmisorIdentidad:                     "https://idp.mulhacen.test",
		RedesPermitidas:                     []string{"10.40.0.0/16"},
		IdentidadesSANProxyPermitidas:       []string{"dns:proxy-interno.mulhacen.test"},
		DuracionMaximaAsercion:              3 * time.Minute,
		ToleranciaReloj:                     20 * time.Second,
		MetodosAdmitidos:                    []MetodoAutenticacion{MetodoKerberos, MetodoCertificado},
		FactoresRequeridos:                  []MetodoAutenticacion{MetodoKerberos, MetodoCertificado},
		MinimoFactoresVerificados:           2,
		MinimoGruposCriptograficosDistintos: 2,
		GarantiaMinima:                      dominiovec.AuthAssuranceHigh,
	}
}

func configuracionAdministracionValida() ConfiguracionSuperficie {
	return ConfiguracionSuperficie{
		Superficie:                          SuperficieAdministracionPrivilegiada,
		ZonaRed:                             ZonaRedAdministracion,
		DireccionEscucha:                    "10.50.0.20:9443",
		Audiencia:                           "vec-administracion",
		EmisorIdentidad:                     "https://idp-admin.mulhacen.test",
		RedesPermitidas:                     []string{"10.50.0.0/24"},
		IdentidadesSANProxyPermitidas:       []string{"dns:proxy-administracion.mulhacen.test"},
		DuracionMaximaAsercion:              5 * time.Minute,
		ToleranciaReloj:                     10 * time.Second,
		MetodosAdmitidos:                    []MetodoAutenticacion{MetodoKerberos, MetodoCertificado},
		FactoresRequeridos:                  []MetodoAutenticacion{MetodoKerberos, MetodoCertificado},
		MinimoFactoresVerificados:           2,
		MinimoGruposCriptograficosDistintos: 2,
		GarantiaMinima:                      dominiovec.AuthAssuranceHigh,
		RequiereCuentaPrivilegiada:          true,
	}
}

func cambiar(base ConfiguracionSuperficie, cambio func(*ConfiguracionSuperficie)) ConfiguracionSuperficie {
	cambio(&base)
	return base
}
