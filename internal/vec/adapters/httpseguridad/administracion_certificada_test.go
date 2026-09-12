package httpseguridad

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func configuracionAdministracionCertificada() ConfiguracionSuperficie {
	c := configuracionAdministracionValida()
	c.MetodosAdmitidos = []MetodoAutenticacion{MetodoCertificado, MetodoDNIe}
	c.FactoresRequeridos = nil
	c.MinimoFactoresVerificados = 1
	c.MinimoGruposCriptograficosDistintos = 1
	return c
}

func TestAdministracionCertificadaPoliticaSinKerberos(t *testing.T) {
	for _, metodo := range []MetodoAutenticacion{MetodoCertificado, MetodoDNIe} {
		t.Run(string(metodo), func(t *testing.T) {
			c := configuracionAdministracionCertificada()
			c.MetodosAdmitidos = []MetodoAutenticacion{metodo}
			if err := c.Validar(); err != nil {
				t.Fatalf("certificado administrativo sin Kerberos rechazado: %v", err)
			}
			c.Superficie = SuperficieInternaCorporativa
			c.ZonaRed = ZonaRedInterna
			c.RequiereCuentaPrivilegiada = false
			if !errors.Is(c.Validar(), ErrConfiguracionSuperficie) {
				t.Fatal("se ha eliminado Kerberos de la superficie interna")
			}
		})
	}
	for _, metodo := range []MetodoAutenticacion{MetodoKerberos, MetodoClave, MetodoSSO} {
		c := configuracionAdministracionCertificada()
		c.MetodosAdmitidos = []MetodoAutenticacion{metodo}
		if !errors.Is(c.Validar(), ErrConfiguracionSuperficie) {
			t.Fatalf("administración admite una política sin certificado: %s", metodo)
		}
	}
	c := configuracionAdministracionCertificada()
	c.GarantiaMinima = dominiovec.AuthAssuranceSubstantial
	if !errors.Is(c.Validar(), ErrConfiguracionSuperficie) {
		t.Fatal("administración sin garantía alta")
	}
	c = configuracionAdministracionCertificada()
	c.RequiereCuentaPrivilegiada = false
	if !errors.Is(c.Validar(), ErrConfiguracionSuperficie) {
		t.Fatal("administración sin cuenta privilegiada")
	}
	politica, err := NuevaPoliticaRed(configuracionAdministracionCertificada())
	if err != nil || politica.Autorizar(netip.MustParseAddr("10.50.0.10")) != nil ||
		!errors.Is(politica.Autorizar(netip.MustParseAddr("198.51.100.5")), ErrRedNoAutorizada) {
		t.Fatal("la política administrativa perdió la restricción de red")
	}
}

func TestAdministracionCertificadaResuelveCapsulaYRevoca(t *testing.T) {
	for _, metodo := range []MetodoAutenticacion{MetodoCertificado, MetodoDNIe} {
		t.Run(string(metodo), func(t *testing.T) {
			ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
			c := configuracionAdministracionCertificada()
			v := &verificadorFalso{}
			registro := nuevoRegistroMemoria()
			s := debeServicio(t, c, v, evaluadorValido(dominiovec.AuthAssuranceHigh), registro, &relojFijo{ahora: ahora})
			canal := debeCanalTLS(t, s, c)
			a := asercionAdministracionCertificada(ahora, c, canal, metodo)
			v.fijarAsercion(a)
			identidad, err := s.Resolver(context.Background(), debeCredencial(t, []byte("admin-certificada"), canal))
			if err != nil {
				t.Fatal(err)
			}
			capsula, err := s.ProyectarCapsulaIdentidadPeticion(context.Background(), identidad, canal)
			if err != nil {
				t.Fatal(err)
			}
			ctx, err := s.VincularCapsulaIdentidadPeticion(context.Background(), capsula, canal)
			if err != nil {
				t.Fatal(err)
			}
			cuenta, auditoria, err := s.ExtraerCapsulaIdentidadPeticion(ctx)
			esperado, _ := metodoAutenticacionDominio(metodo)
			if err != nil || cuenta.Metodo != esperado || cuenta.Garantia != dominiovec.AuthAssuranceHigh ||
				!auditoria.CuentaPrivilegiada() || auditoria.Superficie() != SuperficieAdministracionPrivilegiada {
				t.Fatal("la cápsula no conserva la identidad administrativa certificada")
			}
			if _, _, err := s.ExtraerCapsulaIdentidadPeticion(context.Background()); !errors.Is(err, ErrSesionNoValida) {
				t.Fatal("contexto sin cápsula aceptado")
			}
			registro.inactivar(a.Cuenta.CuentaOrdinariaID)
			if _, _, err := s.ExtraerCapsulaIdentidadPeticion(ctx); !errors.Is(err, ErrSesionNoValida) {
				t.Fatal("cuenta administrativa revocada aceptada")
			}
		})
	}
}

func TestAdministracionCertificadaSinAlternativasDeAcceso(t *testing.T) {
	for _, metodo := range []MetodoAutenticacion{MetodoKerberos, MetodoClave, MetodoSSO} {
		for _, conservaCertificado := range []bool{false, true} {
			t.Run(string(metodo)+map[bool]string{false: "_solo", true: "_primario_ajeno"}[conservaCertificado], func(t *testing.T) {
				ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
				c := configuracionAdministracionCertificada()
				c.MetodosAdmitidos = append(c.MetodosAdmitidos, metodo)
				v := &verificadorFalso{}
				evaluador := evaluadorValido(dominiovec.AuthAssuranceHigh)
				s := debeServicio(t, c, v, evaluador, nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
				canal := debeCanalTLS(t, s, c)
				a := asercionAdministracionCertificada(ahora, c, canal, MetodoCertificado)
				factor := FactorAutenticacion{Metodo: metodo, SujetoVinculadoID: a.SujetoID, Principal: "cuenta@CORPORATIVA.EXAMPLE", EvidenciaRef: "otro:001", GrupoCriptograficoRef: "grupo:otro", VerificadoEn: a.EmitidaEn}
				if !conservaCertificado {
					a.Factores = nil
				}
				a.Factores = append(a.Factores, factor)
				a.MetodoPrimario = metodo
				v.fijarAsercion(a)
				if _, err := s.Resolver(context.Background(), debeCredencial(t, []byte("admin-alternativa"), canal)); !errors.Is(err, ErrAsercionNoValida) {
					t.Fatalf("alternativa sin identidad certificada admitida: %v", err)
				}
				if evaluador.llamadas.Load() != 0 {
					t.Fatal("identidad inválida llegó a evaluación y alta")
				}
			})
		}
	}
}

func TestAdministracionCertificadaRevalidaMetodoPrimario(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	c := configuracionAdministracionCertificada()
	c.MetodosAdmitidos = append(c.MetodosAdmitidos, MetodoKerberos)
	v := &verificadorFalso{}
	s := debeServicio(t, c, v, evaluadorValido(dominiovec.AuthAssuranceHigh), nuevoRegistroMemoria(), &relojFijo{ahora: ahora})
	canal := debeCanalTLS(t, s, c)
	a := asercionAdministracionCertificada(ahora, c, canal, MetodoCertificado)
	a.Factores = append(a.Factores, FactorAutenticacion{Metodo: MetodoKerberos, SujetoVinculadoID: a.SujetoID, Principal: "cuenta@CORPORATIVA.EXAMPLE", EvidenciaRef: "krb:001", GrupoCriptograficoRef: "grupo:krb", VerificadoEn: a.EmitidaEn})
	v.fijarAsercion(a)
	identidad, err := s.Resolver(context.Background(), debeCredencial(t, []byte("admin-revalidacion"), canal))
	if err != nil {
		t.Fatal(err)
	}
	if err := validarEstadoSesion(identidad.estado, c, ahora); err != nil {
		t.Fatalf("precondición de revalidación válida: %v", err)
	}
	// Representa una sesión anterior o corrompida cuyo método primario ya
	// no satisface la política actual. Se conserva el factor certificado
	// para que la comprobación no pase por ausencia de evidencias.
	identidad.estado.metodoPrimario = MetodoKerberos
	identidad.estado.metodoObservado = dominiovec.AuthMethodKerberos
	if !errors.Is(validarEstadoSesion(identidad.estado, c, ahora), ErrSesionNoValida) {
		t.Fatal("revalidación administrativa acepta método primario ajeno")
	}
}

func asercionAdministracionCertificada(ahora time.Time, c ConfiguracionSuperficie, canal CanalProxyAutenticado, metodo MetodoAutenticacion) AsercionProxyIdentidad {
	a := asercionInternaValida(ahora, c, canal)
	a.Cuenta = CuentaAcceso{ID: "admin-tecnica", SujetoVinculadoID: a.SujetoID, CuentaOrdinariaID: "cuenta-tecnica", Privilegiada: true}
	a.Factores = a.Factores[1:]
	a.Factores[0].Metodo = metodo
	a.MetodoPrimario = metodo
	return a
}
