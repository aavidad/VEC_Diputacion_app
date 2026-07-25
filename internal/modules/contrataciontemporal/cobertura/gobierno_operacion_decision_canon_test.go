package cobertura

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestPoliticaActuacionCoberturaCanonComprometeCadaCampo(t *testing.T) {
	solicitud := solicitudGobiernoOperacionCoberturaPrueba(t, false, 2)
	base := publicacionGobiernoOperacionCoberturaPrueba(
		t,
		solicitud,
		instanteOperacionDecisionCoberturaPrueba,
	).PoliticaActuacion
	if base.Validar() != nil {
		t.Fatal("la política base no es válida")
	}
	casos := []struct {
		nombre    string
		adulterar func(*PublicacionPoliticaActuacionCobertura)
	}{
		{"canon dominio", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Canon.Dominio += ".otro"
		}},
		{"canon versión", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Canon.VersionEsquema++
		}},
		{"canon algoritmo", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Canon.Algoritmo = "sha-512"
		}},
		{"referencia", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Referencia += "_otra"
		}},
		{"versión", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Version++
		}},
		{"organización", func(p *PublicacionPoliticaActuacionCobertura) {
			p.OrganizacionRef += "_otra"
		}},
		{"acción", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Accion = domain.AccionRectificarCoberturaGobernada
		}},
		{"catálogo referencia", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Catalogo.Referencia += "_otro"
		}},
		{"catálogo versión", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Catalogo.Version++
		}},
		{"catálogo huella", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Catalogo.HuellaSHA256 = strings.Repeat("c", 64)
		}},
		{"política referencia", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Politica.Referencia += "_otra"
		}},
		{"política versión", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Politica.Version++
		}},
		{"política huella", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Politica.HuellaSHA256 = strings.Repeat("d", 64)
		}},
		{"finalidad CT clave", func(p *PublicacionPoliticaActuacionCobertura) {
			p.FinalidadContratacionClave = "otra_finalidad_ct"
		}},
		{"finalidad CT ref", func(p *PublicacionPoliticaActuacionCobertura) {
			p.FinalidadContratacionRef += "_otra"
		}},
		{"finalidad VEC", func(p *PublicacionPoliticaActuacionCobertura) {
			p.FinalidadAutorizacionVEC = "otra_finalidad_vec"
		}},
		{"unidad", func(p *PublicacionPoliticaActuacionCobertura) {
			p.UnidadEjecutoraRef += "_otra"
		}},
		{"fase", func(p *PublicacionPoliticaActuacionCobertura) {
			p.FaseDestino = "otra_fase"
		}},
		{"estado", func(p *PublicacionPoliticaActuacionCobertura) {
			p.EstadoDestino = domain.EstadoCompletado
		}},
		{"motivo decidir catálogo", func(p *PublicacionPoliticaActuacionCobertura) {
			p.MotivoAutorizacionDecidir.CatalogoID += "_otro"
		}},
		{"motivo decidir versión", func(p *PublicacionPoliticaActuacionCobertura) {
			p.MotivoAutorizacionDecidir.CatalogoVersion++
		}},
		{"motivo decidir huella", func(p *PublicacionPoliticaActuacionCobertura) {
			p.MotivoAutorizacionDecidir.CatalogoHuellaSHA256 =
				strings.Repeat("e", 64)
		}},
		{"motivo decidir clave", func(p *PublicacionPoliticaActuacionCobertura) {
			p.MotivoAutorizacionDecidir.EntradaClave =
				"motivo_33333333333333333333333333333333"
		}},
		{"motivo rectificar catálogo", func(p *PublicacionPoliticaActuacionCobertura) {
			p.MotivoAutorizacionRectificar.CatalogoID += "_otro"
		}},
		{"motivo rectificar versión", func(p *PublicacionPoliticaActuacionCobertura) {
			p.MotivoAutorizacionRectificar.CatalogoVersion++
		}},
		{"motivo rectificar huella", func(p *PublicacionPoliticaActuacionCobertura) {
			p.MotivoAutorizacionRectificar.CatalogoHuellaSHA256 =
				strings.Repeat("f", 64)
		}},
		{"motivo rectificar clave", func(p *PublicacionPoliticaActuacionCobertura) {
			p.MotivoAutorizacionRectificar.EntradaClave =
				"motivo_44444444444444444444444444444444"
		}},
		{"equivalencia", func(p *PublicacionPoliticaActuacionCobertura) {
			p.EquivalenciaMotivosRef =
				"equivalencia_motivos_cobertura_01"
		}},
		{"publicación", func(p *PublicacionPoliticaActuacionCobertura) {
			p.PublicadaEn = p.PublicadaEn.Add(time.Microsecond)
		}},
		{"vigencia desde", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Vigencia.Desde = p.Vigencia.Desde.Add(time.Microsecond)
		}},
		{"vigencia hasta", func(p *PublicacionPoliticaActuacionCobertura) {
			p.Vigencia.Hasta = p.Vigencia.Hasta.Add(time.Microsecond)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			adulterada := base
			caso.adulterar(&adulterada)
			if adulterada.Validar() == nil {
				t.Fatal("el canon no detectó el campo adulterado")
			}
			huella, err := CalcularHuellaSHA256PoliticaActuacionCobertura(
				adulterada,
			)
			if err == nil && huella == base.HuellaSHA256 {
				t.Fatal("el campo no participa en la huella")
			}
		})
	}
}

func TestPoliticaActuacionCoberturaEsDeterministaYTransportable(t *testing.T) {
	solicitud := solicitudGobiernoOperacionCoberturaPrueba(t, false, 2)
	politica := publicacionGobiernoOperacionCoberturaPrueba(
		t,
		solicitud,
		instanteOperacionDecisionCoberturaPrueba,
	).PoliticaActuacion
	primera, err := CalcularHuellaSHA256PoliticaActuacionCobertura(politica)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := CalcularHuellaSHA256PoliticaActuacionCobertura(politica)
	if err != nil || primera != segunda || primera != politica.HuellaSHA256 {
		t.Fatal("la huella de política no es determinista")
	}
	contenido, err := json.Marshal(politica)
	if err != nil {
		t.Fatal(err)
	}
	var restaurada PublicacionPoliticaActuacionCobertura
	if err := json.Unmarshal(contenido, &restaurada); err != nil ||
		restaurada.Validar() != nil ||
		!reflect.DeepEqual(restaurada, politica) {
		t.Fatalf("la publicación durable no se restauró: %v", err)
	}
}

func TestPoliticaActuacionCoberturaGobiernaEquivalenciaDeMotivos(
	t *testing.T,
) {
	solicitud := solicitudGobiernoOperacionCoberturaPrueba(t, false, 2)
	base := publicacionGobiernoOperacionCoberturaPrueba(
		t,
		solicitud,
		instanteOperacionDecisionCoberturaPrueba,
	).PoliticaActuacion

	t.Run("distintos sin equivalencia", func(t *testing.T) {
		if base.Validar() != nil {
			t.Fatal("se rechazaron motivos distintos sin equivalencia")
		}
	})
	t.Run("distintos con equivalencia", func(t *testing.T) {
		politica := base
		politica.EquivalenciaMotivosRef =
			"equivalencia_motivos_cobertura_01"
		if _, err := CalcularHuellaSHA256PoliticaActuacionCobertura(
			politica,
		); !errors.Is(err, ErrGobiernoOperacionCoberturaNoConfiable) {
			t.Fatalf("se aceptó equivalencia para motivos distintos: %v", err)
		}
	})
	t.Run("iguales sin equivalencia", func(t *testing.T) {
		politica := base
		politica.MotivoAutorizacionRectificar =
			politica.MotivoAutorizacionDecidir
		if _, err := CalcularHuellaSHA256PoliticaActuacionCobertura(
			politica,
		); !errors.Is(err, ErrGobiernoOperacionCoberturaNoConfiable) {
			t.Fatalf("se aceptaron motivos iguales sin equivalencia: %v", err)
		}
	})
	t.Run("iguales con equivalencia gobernada", func(t *testing.T) {
		politica := base
		politica.MotivoAutorizacionRectificar =
			politica.MotivoAutorizacionDecidir
		politica.EquivalenciaMotivosRef =
			"equivalencia_motivos_cobertura_01"
		resellarPoliticaActuacionCoberturaPrueba(t, &politica)
		if politica.Validar() != nil {
			t.Fatal("se rechazó equivalencia explícita válida")
		}
	})
}

func TestPoliticaActuacionCoberturaRechazaLimitesYVigenciaInvalidos(
	t *testing.T,
) {
	solicitud := solicitudGobiernoOperacionCoberturaPrueba(t, false, 2)
	base := publicacionGobiernoOperacionCoberturaPrueba(
		t,
		solicitud,
		instanteOperacionDecisionCoberturaPrueba,
	).PoliticaActuacion
	casos := []func(*PublicacionPoliticaActuacionCobertura){
		func(p *PublicacionPoliticaActuacionCobertura) {
			p.Version = MaximoEnteroSeguroOperacionDecisionCobertura + 1
		},
		func(p *PublicacionPoliticaActuacionCobertura) {
			p.Vigencia.Hasta = time.Time{}
		},
		func(p *PublicacionPoliticaActuacionCobertura) {
			p.PublicadaEn = p.Vigencia.Desde.Add(time.Microsecond)
		},
		func(p *PublicacionPoliticaActuacionCobertura) {
			p.EquivalenciaMotivosRef = "no válida"
			p.MotivoAutorizacionRectificar =
				p.MotivoAutorizacionDecidir
		},
	}
	for indice, adulterar := range casos {
		politica := base
		adulterar(&politica)
		if _, err := CalcularHuellaSHA256PoliticaActuacionCobertura(
			politica,
		); err == nil {
			t.Fatalf("caso inválido %d aceptado", indice)
		}
	}
}
