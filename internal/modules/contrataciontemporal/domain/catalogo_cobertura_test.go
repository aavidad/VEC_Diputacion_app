package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCatalogoCoberturaPublicaOpcionesGobernadasSinListaCompilada(t *testing.T) {
	borrador := borradorCatalogoCoberturaValido()
	borrador.Vias = append(borrador.Vias, DefinicionViaCobertura{
		Clave: "convenio_institucional_futuro",
		Orden: 30,
		Comprobaciones: []ComprobacionExigibleCobertura{{
			Clave: "convenio_vigente", Orden: 1, Obligatoria: true,
			Procedencia: procedencia("convenios", "fuente_convenios_2030"),
		}},
	})

	catalogo, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil {
		t.Fatalf("publicar catálogo: %v", err)
	}
	if catalogo.Version() != 7 || catalogo.Referencia() != "catalogo_cobertura_general" ||
		!huellaCatalogoValida(catalogo.HuellaSHA256()) ||
		catalogo.Canon() != CanonHuellaCatalogoCoberturaV1() {
		t.Fatalf("metadatos de publicación inesperados: %#v", catalogo.Publicacion())
	}
	via, encontrada := catalogo.Via("convenio_institucional_futuro")
	if !encontrada || via.Clave != "convenio_institucional_futuro" {
		t.Fatalf("la nueva vía gobernada no está disponible: %#v", via)
	}
}

func TestCatalogoCoberturaOrdenaYConservaObligatoriedadYProcedencia(t *testing.T) {
	catalogo, err := PublicarCatalogoViasCobertura(borradorCatalogoCoberturaValido())
	if err != nil {
		t.Fatalf("publicar catálogo: %v", err)
	}
	vias := catalogo.Vias()
	if len(vias) != 2 || vias[0].Clave != "bolsa_vigente" ||
		vias[1].Clave != "oferta_sae" {
		t.Fatalf("orden de vías inesperado: %#v", vias)
	}
	comprobaciones := vias[0].Comprobaciones
	if len(comprobaciones) != 2 ||
		comprobaciones[0].Clave != "existe_bolsa_vigente" ||
		!comprobaciones[0].Obligatoria ||
		comprobaciones[0].Procedencia.Clave != "bolsa" ||
		comprobaciones[1].Clave != "hay_candidaturas_disponibles" {
		t.Fatalf("comprobaciones inesperadas: %#v", comprobaciones)
	}
}

func TestCatalogoCoberturaEsInmutableFrenteAEntradasYSalidas(t *testing.T) {
	borrador := borradorCatalogoCoberturaValido()
	catalogo, err := PublicarCatalogoViasCobertura(borrador)
	if err != nil {
		t.Fatalf("publicar catálogo: %v", err)
	}
	huellaOriginal := catalogo.HuellaSHA256()

	borrador.Vias[0].Clave = "alterada_entrada"
	borrador.Vias[0].Comprobaciones[0].Clave = "alterada_comprobacion_entrada"
	vias := catalogo.Vias()
	vias[0].Clave = "alterada_salida"
	vias[0].Comprobaciones[0].Clave = "alterada_comprobacion_salida"
	publicacion := catalogo.Publicacion()
	publicacion.Vias[0].Comprobaciones[0].Procedencia.Clave = "alterada_publicacion"

	via, encontrada := catalogo.Via("bolsa_vigente")
	if !encontrada || via.Comprobaciones[0].Clave != "existe_bolsa_vigente" ||
		catalogo.HuellaSHA256() != huellaOriginal || catalogo.Validar() != nil {
		t.Fatal("el catálogo comparte memoria mutable con una frontera")
	}
	via.Comprobaciones[0].Clave = "alterada_consulta"
	otra, _ := catalogo.Via("bolsa_vigente")
	if otra.Comprobaciones[0].Clave != "existe_bolsa_vigente" {
		t.Fatal("la consulta por vía no devuelve una copia defensiva")
	}
}

func TestCatalogoCoberturaRestauraSoloSiElResumenEsCoherente(t *testing.T) {
	catalogo, err := PublicarCatalogoViasCobertura(borradorCatalogoCoberturaValido())
	if err != nil {
		t.Fatalf("publicar catálogo: %v", err)
	}
	publicacion := catalogo.Publicacion()
	restaurado, err := RestaurarCatalogoViasCobertura(publicacion)
	if err != nil || restaurado.HuellaSHA256() != catalogo.HuellaSHA256() {
		t.Fatalf("restaurar publicación íntegra: %v", err)
	}

	publicacion.Vias[0].Comprobaciones[0].Obligatoria = false
	if _, err := RestaurarCatalogoViasCobertura(publicacion); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("se aceptó contenido adulterado: %v", err)
	}
	publicacion = catalogo.Publicacion()
	publicacion.HuellaSHA256 = strings.Repeat("0", 64)
	if _, err := RestaurarCatalogoViasCobertura(publicacion); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("se aceptó una huella nula: %v", err)
	}
	publicacion = catalogo.Publicacion()
	publicacion.Canon.VersionEsquema++
	if _, err := RestaurarCatalogoViasCobertura(publicacion); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("se aceptó un canon desconocido: %v", err)
	}
}

func TestCatalogoCoberturaHuellaVersionYContenido(t *testing.T) {
	base := borradorCatalogoCoberturaValido()
	primero, err := PublicarCatalogoViasCobertura(base)
	if err != nil {
		t.Fatalf("publicar catálogo base: %v", err)
	}
	base.Version++
	segundaVersion, err := PublicarCatalogoViasCobertura(base)
	if err != nil {
		t.Fatalf("publicar segunda versión: %v", err)
	}
	base.Version--
	base.Vias[0].Comprobaciones[0].Obligatoria = false
	otroContenido, err := PublicarCatalogoViasCobertura(base)
	if err != nil {
		t.Fatalf("publicar contenido distinto: %v", err)
	}
	if primero.HuellaSHA256() == segundaVersion.HuellaSHA256() ||
		primero.HuellaSHA256() == otroContenido.HuellaSHA256() {
		t.Fatal("la huella no está ligada a versión y contenido")
	}
}

func TestCatalogoCoberturaCanonV1MantieneVectorGolden(t *testing.T) {
	catalogo, err := PublicarCatalogoViasCobertura(borradorCatalogoCoberturaValido())
	if err != nil {
		t.Fatalf("publicar catálogo: %v", err)
	}
	const huellaEsperada = "5d0ec6a06e0e1f3dae7012fde168e998fa78aa1b8d68660ae757cfa340abd7c5"
	if catalogo.HuellaSHA256() != huellaEsperada {
		t.Fatalf(
			"vector golden del canon V1 cambió: obtenido %s",
			catalogo.HuellaSHA256(),
		)
	}
}

func TestCatalogoCoberturaCanonDistingueVigenciaIndefinida(t *testing.T) {
	finito := borradorCatalogoCoberturaValido()
	catalogoFinito, err := PublicarCatalogoViasCobertura(finito)
	if err != nil {
		t.Fatalf("publicar catálogo finito: %v", err)
	}
	indefinido := borradorCatalogoCoberturaValido()
	indefinido.Vigencia.Hasta = time.Time{}
	catalogoIndefinido, err := PublicarCatalogoViasCobertura(indefinido)
	if err != nil {
		t.Fatalf("publicar catálogo indefinido: %v", err)
	}
	if catalogoFinito.HuellaSHA256() == catalogoIndefinido.HuellaSHA256() ||
		!catalogoIndefinido.VigenteEn(
			finito.Vigencia.Hasta.Add(365*24*time.Hour),
		) {
		t.Fatal("el canon no distingue explícitamente la ausencia de Hasta")
	}
}

func TestCatalogoCoberturaIdentidadExactaResuelveColisionDurable(t *testing.T) {
	base := borradorCatalogoCoberturaValido()
	registrado, err := PublicarCatalogoViasCobertura(base)
	if err != nil {
		t.Fatalf("publicar catálogo: %v", err)
	}
	repetido, err := PublicarCatalogoViasCobertura(base)
	if err != nil {
		t.Fatalf("repetir catálogo: %v", err)
	}
	if err := ValidarReintentoPublicacionCatalogoCobertura(
		registrado.Identidad(),
		repetido.Identidad(),
	); err != nil {
		t.Fatalf("se rechazó reintento exacto: %v", err)
	}

	base.Vias[0].Comprobaciones[0].Obligatoria = false
	otroContenido, err := PublicarCatalogoViasCobertura(base)
	if err != nil {
		t.Fatalf("publicar otro contenido: %v", err)
	}
	if err := ValidarReintentoPublicacionCatalogoCobertura(
		registrado.Identidad(),
		otroContenido.Identidad(),
	); !errors.Is(err, ErrPublicacionCatalogoEnConflicto) {
		t.Fatalf("no se detectó contenido diferente con misma clave: %v", err)
	}

	base.Version++
	otraVersion, err := PublicarCatalogoViasCobertura(base)
	if err != nil {
		t.Fatalf("publicar otra versión: %v", err)
	}
	if err := ValidarReintentoPublicacionCatalogoCobertura(
		registrado.Identidad(),
		otraVersion.Identidad(),
	); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("se trató otra clave durable como reintento: %v", err)
	}
}

func TestCatalogoCoberturaAplicaVigenciaSemiabierta(t *testing.T) {
	catalogo, err := PublicarCatalogoViasCobertura(borradorCatalogoCoberturaValido())
	if err != nil {
		t.Fatalf("publicar catálogo: %v", err)
	}
	desde, hasta := catalogo.Vigencia().Desde, catalogo.Vigencia().Hasta
	casos := []struct {
		nombre   string
		instante time.Time
		vigente  bool
	}{
		{"antes", desde.Add(-time.Microsecond), false},
		{"inicio incluido", desde, true},
		{"interior", desde.Add(time.Hour), true},
		{"final excluido", hasta, false},
		{"instante no canónico", desde.Add(time.Nanosecond), false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if obtenido := catalogo.VigenteEn(caso.instante); obtenido != caso.vigente {
				t.Fatalf("VigenteEn() = %v; quiere %v", obtenido, caso.vigente)
			}
		})
	}
}

func TestCatalogoCoberturaRechazaDuplicadosYProcedenciaAmbigua(t *testing.T) {
	casos := []struct {
		nombre    string
		modificar func(*BorradorCatalogoViasCobertura)
	}{
		{
			"clave de vía duplicada",
			func(b *BorradorCatalogoViasCobertura) {
				b.Vias[1].Clave = b.Vias[0].Clave
			},
		},
		{
			"orden de vía duplicado",
			func(b *BorradorCatalogoViasCobertura) {
				b.Vias[1].Orden = b.Vias[0].Orden
			},
		},
		{
			"clave de comprobación duplicada en la vía",
			func(b *BorradorCatalogoViasCobertura) {
				b.Vias[1].Comprobaciones[1].Clave = b.Vias[1].Comprobaciones[0].Clave
			},
		},
		{
			"orden de comprobación duplicado",
			func(b *BorradorCatalogoViasCobertura) {
				b.Vias[1].Comprobaciones[1].Orden = b.Vias[1].Comprobaciones[0].Orden
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			borrador := borradorCatalogoCoberturaValido()
			caso.modificar(&borrador)
			if _, err := PublicarCatalogoViasCobertura(borrador); !errors.Is(err, ErrDatoInvalido) {
				t.Fatalf("se aceptó catálogo inválido: %v", err)
			}
		})
	}
}

func TestCatalogoCoberturaRechazaProcedenciaContradictoriaEntreVias(t *testing.T) {
	borrador := borradorCatalogoCoberturaValido()
	borrador.Vias[0].Comprobaciones[0] = ComprobacionExigibleCobertura{
		Clave: "comprobacion_compartida", Orden: 1, Obligatoria: true,
		Procedencia: procedencia("sae", "fuente_sae_publicada"),
	}
	borrador.Vias[1].Comprobaciones[0] = ComprobacionExigibleCobertura{
		Clave: "comprobacion_compartida", Orden: 20, Obligatoria: false,
		Procedencia: procedencia("bolsa", "fuente_bolsa_publicada"),
	}
	if borrador.Vias[0].Validar() != nil || borrador.Vias[1].Validar() != nil {
		t.Fatal("el caso no alcanza la comprobación de coherencia entre vías")
	}
	if _, err := PublicarCatalogoViasCobertura(borrador); !errors.Is(err, ErrDatoInvalido) {
		t.Fatalf("se aceptó procedencia contradictoria entre vías: %v", err)
	}
}

func TestCatalogoCoberturaAdmiteReutilizarComprobacionConMismaProcedencia(t *testing.T) {
	borrador := borradorCatalogoCoberturaValido()
	borrador.Vias[0].Comprobaciones[0] = ComprobacionExigibleCobertura{
		Clave: "existe_bolsa_vigente", Orden: 1, Obligatoria: false,
		Procedencia: procedencia("bolsa", "fuente_bolsa_publicada"),
	}
	if _, err := PublicarCatalogoViasCobertura(borrador); err != nil {
		t.Fatalf("se rechazó reutilización coherente: %v", err)
	}
}

func TestCatalogoCoberturaRechazaLimitesYMetadatosInvalidos(t *testing.T) {
	casos := []struct {
		nombre    string
		modificar func(*BorradorCatalogoViasCobertura)
	}{
		{"sin referencia", func(b *BorradorCatalogoViasCobertura) {
			b.Referencia = ""
		}},
		{"sin versión", func(b *BorradorCatalogoViasCobertura) {
			b.Version = 0
		}},
		{"publicación no canónica", func(b *BorradorCatalogoViasCobertura) {
			b.PublicadoEn = b.PublicadoEn.Add(time.Nanosecond)
		}},
		{"vigencia invertida", func(b *BorradorCatalogoViasCobertura) {
			b.Vigencia.Hasta = b.Vigencia.Desde
		}},
		{"sin procedencia", func(b *BorradorCatalogoViasCobertura) {
			b.ProcedenciaRef = ""
		}},
		{"sin vías", func(b *BorradorCatalogoViasCobertura) {
			b.Vias = nil
		}},
		{"sin comprobaciones", func(b *BorradorCatalogoViasCobertura) {
			b.Vias[0].Comprobaciones = nil
		}},
		{"orden de comprobación nulo", func(b *BorradorCatalogoViasCobertura) {
			b.Vias[0].Comprobaciones[0].Orden = 0
		}},
		{"procedencia de comprobación incompleta", func(b *BorradorCatalogoViasCobertura) {
			b.Vias[0].Comprobaciones[0].Procedencia.DefinicionFuenteRef = ""
		}},
		{"demasiadas comprobaciones por vía", func(b *BorradorCatalogoViasCobertura) {
			b.Vias[0].Comprobaciones = comprobacionesNumeradas(
				maximoComprobacionesPorViaCobertura + 1,
			)
		}},
		{"demasiadas comprobaciones totales", func(b *BorradorCatalogoViasCobertura) {
			b.Vias = viasConComprobacionesNumeradas(17, 32)
		}},
		{"demasiadas vías", func(b *BorradorCatalogoViasCobertura) {
			b.Vias = viasNumeradas(maximoViasCobertura + 1)
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			borrador := borradorCatalogoCoberturaValido()
			caso.modificar(&borrador)
			if _, err := PublicarCatalogoViasCobertura(borrador); !errors.Is(err, ErrDatoInvalido) {
				t.Fatalf("se aceptó catálogo inválido: %v", err)
			}
		})
	}
}

func borradorCatalogoCoberturaValido() BorradorCatalogoViasCobertura {
	return BorradorCatalogoViasCobertura{
		Referencia: "catalogo_cobertura_general", Version: 7,
		PublicadoEn: time.Date(2026, 7, 23, 9, 0, 0, 0, time.UTC),
		Vigencia: VigenciaCatalogoCobertura{
			Desde: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			Hasta: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		ProcedenciaRef: "resolucion_catalogo_cobertura_2026_07",
		Vias: []DefinicionViaCobertura{
			{
				Clave: "oferta_sae", Orden: 20,
				Comprobaciones: []ComprobacionExigibleCobertura{{
					Clave: "oferta_sae_disponible", Orden: 1, Obligatoria: true,
					Procedencia: procedencia("sae", "fuente_sae_publicada"),
				}},
			},
			{
				Clave: "bolsa_vigente", Orden: 10,
				Comprobaciones: []ComprobacionExigibleCobertura{
					{
						Clave: "hay_candidaturas_disponibles", Orden: 20,
						Obligatoria: true,
						Procedencia: procedencia("bolsa", "fuente_bolsa_publicada"),
					},
					{
						Clave: "existe_bolsa_vigente", Orden: 10,
						Obligatoria: true,
						Procedencia: procedencia("bolsa", "fuente_bolsa_publicada"),
					},
				},
			},
		},
	}
}

func procedencia(
	clave ClaveCatalogo,
	definicion string,
) ProcedenciaComprobacionCobertura {
	return ProcedenciaComprobacionCobertura{
		Clave: clave, DefinicionFuenteRef: definicion,
	}
}

func comprobacionesNumeradas(cantidad int) []ComprobacionExigibleCobertura {
	resultado := make([]ComprobacionExigibleCobertura, cantidad)
	for indice := range resultado {
		resultado[indice] = ComprobacionExigibleCobertura{
			Clave: ClaveCatalogo("comprobacion_" + numeroTresCifras(indice+1)),
			Orden: uint16(indice + 1), Obligatoria: true,
			Procedencia: procedencia("fuente_generica", "definicion_fuente_generica"),
		}
	}
	return resultado
}

func viasNumeradas(cantidad int) []DefinicionViaCobertura {
	resultado := make([]DefinicionViaCobertura, cantidad)
	for indice := range resultado {
		resultado[indice] = DefinicionViaCobertura{
			Clave: ClaveCatalogo("via_" + numeroTresCifras(indice+1)),
			Orden: uint16(indice + 1),
			Comprobaciones: []ComprobacionExigibleCobertura{{
				Clave: ClaveCatalogo("comprobacion_" + numeroTresCifras(indice+1)),
				Orden: 1, Obligatoria: true,
				Procedencia: procedencia("fuente_generica", "definicion_fuente_generica"),
			}},
		}
	}
	return resultado
}

func viasConComprobacionesNumeradas(
	cantidadVias int,
	comprobacionesPorVia int,
) []DefinicionViaCobertura {
	resultado := make([]DefinicionViaCobertura, cantidadVias)
	for indiceVia := range resultado {
		comprobaciones := make(
			[]ComprobacionExigibleCobertura,
			comprobacionesPorVia,
		)
		for indiceComprobacion := range comprobaciones {
			comprobaciones[indiceComprobacion] = ComprobacionExigibleCobertura{
				Clave: ClaveCatalogo(
					"comprobacion_" + numeroTresCifras(indiceVia+1) +
						"_" + numeroTresCifras(indiceComprobacion+1),
				),
				Orden: uint16(indiceComprobacion + 1), Obligatoria: true,
				Procedencia: procedencia(
					"fuente_generica",
					"definicion_fuente_generica",
				),
			}
		}
		resultado[indiceVia] = DefinicionViaCobertura{
			Clave: ClaveCatalogo("via_" + numeroTresCifras(indiceVia+1)),
			Orden: uint16(indiceVia + 1), Comprobaciones: comprobaciones,
		}
	}
	return resultado
}

func numeroTresCifras(valor int) string {
	const digitos = "0123456789"
	return string([]byte{
		digitos[(valor/100)%10],
		digitos[(valor/10)%10],
		digitos[valor%10],
	})
}
