package cobertura

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestVectorGoldenHuellaPoliticaActuacionCoberturaV1(t *testing.T) {
	// Este vector solo fija la representación canónica. No construye una
	// capacidad ni sustituye al resolutor/BD autoritativos.
	publicacion := PublicacionPoliticaActuacionCobertura{
		Referencia:      "politica_actuacion_vector_golden_01",
		Version:         17,
		Canon:           CanonHuellaPoliticaActuacionCoberturaV1(),
		OrganizacionRef: "organizacion_vector_golden_01",
		Accion:          domain.AccionRectificarCoberturaGobernada,
		Catalogo: domain.IdentidadCatalogoViasCobertura{
			Referencia:   "catalogo_cobertura_vector_golden_01",
			Version:      23,
			HuellaSHA256: strings.Repeat("c", 64),
		},
		Politica: domain.IdentidadPoliticaDecisionCobertura{
			Referencia:   "politica_decision_vector_golden_01",
			Version:      11,
			HuellaSHA256: strings.Repeat("d", 64),
		},
		FinalidadContratacionClave: "gestionar_vector_cobertura",
		FinalidadContratacionRef:   "finalidad_ct_vector_golden_01",
		FinalidadAutorizacionVEC:   "autorizar_vector_cobertura",
		UnidadEjecutoraRef:         "unidad_vector_golden_01",
		FaseDestino:                "fase_vector_golden",
		EstadoDestino:              domain.EstadoEnCurso,
		MotivoAutorizacionDecidir: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           "motivos_vector_golden",
			CatalogoVersion:      31,
			CatalogoHuellaSHA256: strings.Repeat("e", 64),
			EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
		},
		MotivoAutorizacionRectificar: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           "motivos_vector_golden",
			CatalogoVersion:      31,
			CatalogoHuellaSHA256: strings.Repeat("e", 64),
			EntradaClave:         "motivo_fedcba9876543210fedcba9876543210",
		},
		EquivalenciaMotivosRef: "",
		PublicadaEn: time.Date(
			2026, 7, 25, 7, 1, 2, 345000000, time.UTC,
		),
		Vigencia: domain.VigenciaCatalogoCobertura{
			Desde: time.Date(
				2026, 7, 25, 8, 2, 3, 456000000, time.UTC,
			),
			Hasta: time.Date(
				2026, 7, 25, 9, 3, 4, 567000000, time.UTC,
			),
		},
	}
	const esperada = "3ffdec894892481a12b3909033e8fc05d49ff25aa3460d25490d1f1d2203b13e"
	obtenida, err := CalcularHuellaSHA256PoliticaActuacionCobertura(
		publicacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	if obtenida != esperada {
		t.Fatalf("vector golden V1 alterado: obtenida=%s", obtenida)
	}
}

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
