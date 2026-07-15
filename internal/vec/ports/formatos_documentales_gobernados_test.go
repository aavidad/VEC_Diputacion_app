package ports

import (
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestDescriptorPerfilYConsultaOperativaSeparanCatalogoDeEjecucion(t *testing.T) {
	perfil, revision := valoresPerfilDocumentalGobernadoPrueba(t)
	descriptor, err := NuevoDescriptorPerfilDocumental(
		"descriptor:pdfa4:2", "publicacion:pdfa4:2", perfil, revision,
	)
	consulta := ConsultaFormatoDocumental{
		Identidad: perfil.Identidad(), PerfilRef: perfil.Referencia(),
		DigestPerfilSHA256: perfil.DigestSHA256(), RevisionCatalogo: revision,
	}
	if err != nil || descriptor.Validar() != nil || !descriptor.Coincide(consulta) ||
		descriptor.PublicacionRef() != "publicacion:pdfa4:2" || descriptor.Perfil() != perfil ||
		descriptor.Revision() != revision {
		t.Fatalf("descriptor de perfil invalido: %#v / %v", descriptor, err)
	}

	consultaActual := ConsultaSituacionOperativaActual{
		PublicacionRef: descriptor.PublicacionRef(), PerfilRef: perfil.Referencia(),
		DigestPerfil: perfil.DigestSHA256(), RevisionCatalogo: revision,
	}
	vigente, _ := domain.NuevaSituacionOperativaPerfilDocumental(
		descriptor.PublicacionRef(), perfil, revision, 1, domain.EstadoPublicacionPerfilVigente,
	)
	revocada, _ := domain.NuevaSituacionOperativaPerfilDocumental(
		descriptor.PublicacionRef(), perfil, revision, 2, domain.EstadoPublicacionPerfilRevocada,
	)
	if consultaActual.Validar() != nil || !consultaActual.Coincide(vigente) ||
		!consultaActual.Coincide(revocada) {
		t.Fatal("la consulta no fija la cadena operativa exacta")
	}
	// Coincidir fija identidad; autorizar exige ademas que el registro haya
	// releido la proyeccion actual y esta sea vigente. La revocacion no muta el
	// descriptor historico.
	if revocada.Estado() == domain.EstadoPublicacionPerfilVigente || descriptor.Perfil() != perfil {
		t.Fatal("estado operativo mezclado con el perfil historico")
	}

	revisionAjena, _ := domain.NuevaRevisionCatalogoFormatosDocumentales(13, strings.Repeat("f", 64))
	discrepante := consulta
	discrepante.RevisionCatalogo = revisionAjena
	if descriptor.Coincide(discrepante) {
		t.Fatal("revision de catalogo discrepante aceptada")
	}
	if (DescriptorPerfilDocumental{}).Validar() == nil ||
		(ConsultaSituacionOperativaActual{}).Validar() == nil {
		t.Fatal("valor cero obtuvo autoridad")
	}
	if _, err := NuevoDescriptorPerfilDocumental(
		"descriptor:*", "publicacion:pdfa4:2", perfil, revision,
	); !errors.Is(err, ErrDescriptorFormatoDocumentalInvalido) {
		t.Fatalf("comodin en descriptor aceptado: %v", err)
	}
}

func TestDescriptorComponenteEsAtestadoPorRolLimiteYDominioDeConfianza(t *testing.T) {
	perfil, revision := valoresPerfilDocumentalGobernadoPrueba(t)
	consultaMarcador := consultaComponenteDocumentalPrueba(
		perfil, revision, domain.RolComponenteMarcador,
	)
	marcador := descriptorComponenteDocumentalPrueba(
		t, consultaMarcador, "marcador-pdfa", 1, "dominio:marcador", "a", 8*1024*1024,
	)
	if marcador.Validar() != nil || !marcador.Coincide(consultaMarcador) ||
		marcador.Componente().Rol() != domain.RolComponenteMarcador ||
		marcador.MaximoBytes() != 8*1024*1024 || len(marcador.DigestDeclaracionSHA256()) != 64 {
		t.Fatalf("descriptor atestado invalido: %#v", marcador)
	}

	extractor := descriptorComponenteDocumentalPrueba(
		t, consultaComponenteDocumentalPrueba(perfil, revision, domain.RolComponenteExtractorMetadatos),
		"extractor-pdfa", 2, "dominio:extractor", "b", 8*1024*1024,
	)
	verificador := descriptorComponenteDocumentalPrueba(
		t, consultaComponenteDocumentalPrueba(perfil, revision, domain.RolComponenteVerificador),
		"verificador-pdfa", 3, "dominio:verificador", "c", 8*1024*1024,
	)
	if !marcador.IndependienteDe(extractor) || !marcador.IndependienteDe(verificador) ||
		!extractor.IndependienteDe(verificador) {
		t.Fatal("componentes realmente segregados no se reconocieron")
	}
	mismoDominio := descriptorComponenteDocumentalPrueba(
		t, consultaComponenteDocumentalPrueba(perfil, revision, domain.RolComponenteExtractorMetadatos),
		"extractor-otro", 4, marcador.DominioConfianzaRef(), "d", 8*1024*1024,
	)
	if marcador.IndependienteDe(mismoDominio) {
		t.Fatal("dos componentes bajo el mismo dominio de confianza parecieron independientes")
	}
	mismoArtefactoRef, _ := domain.NuevaReferenciaComponenteDocumental(
		domain.RolComponenteExtractorMetadatos, "extractor-copia", 5,
		"homologacion:extractor-copia:5", strings.Repeat("d", 64),
		marcador.Componente().HuellaArtefactoSHA256(),
	)
	mismoArtefacto, err := NuevoDescriptorComponenteDocumentalAtestado(
		"atestacion:extractor-copia:5",
		consultaComponenteDocumentalPrueba(perfil, revision, domain.RolComponenteExtractorMetadatos),
		mismoArtefactoRef, "dominio:extractor-copia", "broker:documental",
		"prueba:extractor-copia:5", strings.Repeat("e", 64), 8*1024*1024,
	)
	if err != nil || marcador.IndependienteDe(mismoArtefacto) {
		t.Fatal("el mismo artefacto renombrado parecio una barrera independiente")
	}

	componenteRolAjeno, _ := domain.NuevaReferenciaComponenteDocumental(
		domain.RolComponenteRenderizador, "renderizador-pdfa", 1,
		"homologacion:renderizador-pdfa:1", strings.Repeat("a", 64), strings.Repeat("b", 64),
	)
	if _, err := NuevoDescriptorComponenteDocumentalAtestado(
		"atestacion:rol-ajeno", consultaMarcador, componenteRolAjeno,
		"dominio:renderizador", "broker:documental", "prueba:rol-ajeno",
		strings.Repeat("c", 64), 1024,
	); !errors.Is(err, ErrComponenteDocumentalAtestadoInvalido) {
		t.Fatalf("componente autoclasificado con rol ajeno aceptado: %v", err)
	}
	if _, err := NuevoDescriptorComponenteDocumentalAtestado(
		"atestacion:sin-limite", consultaMarcador, marcador.Componente(),
		"dominio:marcador", "broker:documental", "prueba:sin-limite",
		strings.Repeat("c", 64), 0,
	); !errors.Is(err, ErrComponenteDocumentalAtestadoInvalido) {
		t.Fatalf("componente sin limite homologado aceptado: %v", err)
	}
	if (DescriptorComponenteDocumentalAtestado{}).Validar() == nil {
		t.Fatal("descriptor de componente cero valido")
	}
}

func TestPoliticaInstitucionalPositivaConstruyeURIYNoLaAceptaDelUsuario(t *testing.T) {
	perfil, _ := valoresPerfilDocumentalGobernadoPrueba(t)
	institucion, _ := domain.NuevaReferenciaInstitucionalDocumento(
		"entidad:diputacion_granada", "organo:servicio_seleccion",
	)
	consulta := ConsultaPoliticaInstitucionalDocumental{
		Institucion: institucion, PerfilRef: perfil.Referencia(),
		ManifiestoRef: "manifiesto:documento:001", RequiereURIPublica: true,
	}
	politica, err := NuevaPoliticaInstitucionalDocumentalAtestada(
		"politica:institucional:1", 3, consulta, "endpoint:sede:documentos",
		"https://sede.dipgra.es/documentos", strings.Repeat("a", 64),
	)
	if err != nil || politica.Validar() != nil || !politica.Coincide(consulta) {
		t.Fatalf("politica positiva invalida: %#v / %v", politica, err)
	}
	uuid := "018f6f2a-6d92-4cc3-8c5e-3fb3e8e8a123"
	fecha := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	marca, err := politica.ConstruirMarca(uuid, fecha)
	if err != nil || marca.URIPublica() != "https://sede.dipgra.es/documentos/"+uuid ||
		marca.Institucion() != institucion || marca.ManifiestoRef() != consulta.ManifiestoRef {
		t.Fatalf("URI institucional no construida desde allowlist: %#v / %v", marca, err)
	}

	for nombre, base := range map[string]string{
		"http":       "http://sede.dipgra.es/documentos",
		"host local": "https://servidor.local/documentos",
		"credencial": "https://usuario:clave@sede.dipgra.es/documentos",
		"consulta":   "https://sede.dipgra.es/documentos?token=secreto",
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := NuevaPoliticaInstitucionalDocumentalAtestada(
				"politica:institucional:1", 3, consulta, "endpoint:sede:documentos",
				base, strings.Repeat("a", 64),
			); !errors.Is(err, ErrPoliticaInstitucionalDocumentalInvalida) {
				t.Fatalf("base no pasiva aceptada: %v", err)
			}
		})
	}
	consultaPrivada := consulta
	consultaPrivada.RequiereURIPublica = false
	privada, err := NuevaPoliticaInstitucionalDocumentalAtestada(
		"politica:institucional:privada", 1, consultaPrivada, "", "", strings.Repeat("b", 64),
	)
	marcaPrivada, errMarca := privada.ConstruirMarca(uuid, fecha)
	if err != nil || errMarca != nil || marcaPrivada.URIPublica() != "" {
		t.Fatalf("documento privado obtuvo URI: %#v / %v / %v", marcaPrivada, err, errMarca)
	}
	if (PoliticaInstitucionalDocumentalAtestada{}).Validar() == nil {
		t.Fatal("politica cero obtuvo autoridad")
	}
}

func TestResultadoExtraccionExigeMetadatoYConformidadExactos(t *testing.T) {
	perfil, _ := valoresPerfilDocumentalGobernadoPrueba(t)
	institucion, _ := domain.NuevaReferenciaInstitucionalDocumento(
		"entidad:diputacion_granada", "organo:servicio_seleccion",
	)
	uuid := "018f6f2a-6d92-4cc3-8c5e-3fb3e8e8a123"
	metadato, _ := domain.NuevaMarcaInstitucionalDocumento(
		institucion, uuid, perfil.Referencia(),
		time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC),
		"manifiesto:documento:001", "",
	)
	contenido := []byte("%PDF metadato estructurado")
	resultado := ResultadoExtraccionMetadatoInstitucional{
		Metadato: metadato, HuellaContenidoSHA256: huellaBytesFormatoDocumental(contenido),
		DigestConformidadSHA256: perfil.Conformidad().DigestSHA256(),
	}
	if err := resultado.ValidarContra(perfil, contenido); err != nil {
		t.Fatalf("extraccion exacta rechazada: %v", err)
	}
	resultado.DigestConformidadSHA256 = strings.Repeat("f", 64)
	if !errors.Is(resultado.ValidarContra(perfil, contenido), ErrMetadatoInstitucionalDocumentalInvalido) {
		t.Fatal("extractor valido contra otra conformidad")
	}
}

func valoresPerfilDocumentalGobernadoPrueba(
	t *testing.T,
) (domain.PerfilFormatoDocumental, domain.RevisionCatalogoFormatosDocumentales) {
	t.Helper()
	identidad, _ := domain.NuevaIdentidadSintacticaDocumental("pdf")
	perfilRef, _ := domain.NuevaReferenciaPerfilDocumental("pdfa-4", 2)
	capacidades, _ := domain.NuevasCapacidadesPerfilFormatoDocumental(
		domain.CapacidadPerfilRenderizar, domain.CapacidadPerfilMetadatoInstitucional,
		domain.CapacidadPerfilFirmaElectronica,
	)
	conformidad, err := domain.NuevaReferenciaConformidadDocumental(
		"conformidad:pdfa4", 1, "esquema:pdfa4:1", "dialecto:pdfa4",
		"canonicalizacion:pdf:1", "reglas:pdfa4:1", strings.Repeat("a", 64),
		"politica:documental:1", strings.Repeat("b", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	perfil, err := domain.NuevoPerfilFormatoDocumentalConforme(
		perfilRef, identidad, "application/pdf", "pdf", "binario",
		capacidades, conformidad, 32*1024*1024,
	)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := domain.NuevaRevisionCatalogoFormatosDocumentales(12, strings.Repeat("c", 64))
	if err != nil {
		t.Fatal(err)
	}
	return perfil, revision
}

func consultaComponenteDocumentalPrueba(
	perfil domain.PerfilFormatoDocumental,
	revision domain.RevisionCatalogoFormatosDocumentales,
	rol domain.RolComponenteDocumental,
) ConsultaComponenteDocumentalAtestado {
	return ConsultaComponenteDocumentalAtestado{
		Rol: rol, DescriptorPerfilRef: "descriptor:pdfa4:2", PublicacionRef: "publicacion:pdfa4:2",
		PerfilRef: perfil.Referencia(), DigestPerfil: perfil.DigestSHA256(), RevisionCatalogo: revision,
	}
}

func descriptorComponenteDocumentalPrueba(
	t *testing.T,
	consulta ConsultaComponenteDocumentalAtestado,
	identificador string,
	version uint64,
	dominio, semilla string,
	maximoBytes uint64,
) DescriptorComponenteDocumentalAtestado {
	t.Helper()
	caracter := semilla[0]
	huella := func(desplazamiento byte) string {
		valor := caracter + desplazamiento
		if valor > 'f' {
			valor = '0' + (valor - 'g')
		}
		return strings.Repeat(string(valor), 64)
	}
	componente, err := domain.NuevaReferenciaComponenteDocumental(
		consulta.Rol, identificador, version,
		"homologacion:"+identificador+":"+string(rune('0'+version)),
		huella(0), huella(1),
	)
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := NuevoDescriptorComponenteDocumentalAtestado(
		"atestacion:"+identificador, consulta, componente, dominio,
		"broker:documental", "prueba:"+identificador, huella(2), maximoBytes,
	)
	if err != nil {
		t.Fatal(err)
	}
	return descriptor
}
