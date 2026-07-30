package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestPerfilDocumentalComprometeConformidadPoliticaYLimite(t *testing.T) {
	identidad, err := NuevaIdentidadSintacticaDocumental("pdf")
	if err != nil {
		t.Fatal(err)
	}
	referenciaV1, _ := NuevaReferenciaPerfilDocumental("pdfa-4", 1)
	referenciaV2, _ := NuevaReferenciaPerfilDocumental("pdfa-4", 2)
	conformidad := conformidadDocumentalPrueba(t, 1, strings.Repeat("a", 64), strings.Repeat("b", 64))
	capacidades, err := NuevasCapacidadesPerfilFormatoDocumental(
		CapacidadPerfilRenderizar,
		CapacidadPerfilMetadatoInstitucional,
		CapacidadPerfilFirmaElectronica,
		CapacidadPerfilPreservacionLargoPlazo,
	)
	if err != nil {
		t.Fatal(err)
	}

	perfilV1, err := NuevoPerfilFormatoDocumentalConforme(
		referenciaV1, identidad, "application/pdf", "pdf", "binario",
		capacidades, conformidad, 32*1024*1024,
	)
	if err != nil || perfilV1.Validar() != nil || perfilV1.Referencia() != referenciaV1 ||
		perfilV1.Identidad() != identidad || perfilV1.Conformidad() != conformidad ||
		perfilV1.MaximoBytes() != 32*1024*1024 || len(perfilV1.DigestSHA256()) != 64 {
		t.Fatalf("perfil gobernado invalido: %#v, %v", perfilV1, err)
	}
	perfilV2 := perfilDocumentalPrueba(t, referenciaV2, identidad, capacidades, conformidad, 32*1024*1024, "pdf")
	if perfilV1.DigestSHA256() == perfilV2.DigestSHA256() {
		t.Fatal("la version del perfil no quedo comprometida")
	}

	conformidadReglasDistintas := conformidadDocumentalPrueba(
		t, 1, strings.Repeat("c", 64), strings.Repeat("b", 64),
	)
	perfilReglasDistintas := perfilDocumentalPrueba(
		t, referenciaV1, identidad, capacidades, conformidadReglasDistintas, 32*1024*1024, "pdf",
	)
	conformidadPoliticaDistinta := conformidadDocumentalPrueba(
		t, 1, strings.Repeat("a", 64), strings.Repeat("d", 64),
	)
	perfilPoliticaDistinta := perfilDocumentalPrueba(
		t, referenciaV1, identidad, capacidades, conformidadPoliticaDistinta, 32*1024*1024, "pdf",
	)
	perfilLimiteDistinto := perfilDocumentalPrueba(
		t, referenciaV1, identidad, capacidades, conformidad, 16*1024*1024, "pdf",
	)
	for nombre, candidato := range map[string]PerfilFormatoDocumental{
		"reglas":   perfilReglasDistintas,
		"politica": perfilPoliticaDistinta,
		"limite":   perfilLimiteDistinto,
	} {
		if candidato.DigestSHA256() == perfilV1.DigestSHA256() {
			t.Fatalf("%s no quedo comprometido en el digest del perfil", nombre)
		}
	}

	if _, err := NuevoPerfilFormatoDocumentalConforme(
		referenciaV1, identidad, "application/pdf", "pdf", "binario",
		capacidades, conformidad, 0,
	); !errors.Is(err, ErrPerfilFormatoDocumentalInvalido) {
		t.Fatalf("limite cero aceptado: %v", err)
	}
	if _, err := NuevoPerfilFormatoDocumentalConforme(
		referenciaV1, identidad, "application/pdf", "pdf", "binario",
		capacidades, conformidad, maximoBytesPerfilDocumental+1,
	); !errors.Is(err, ErrPerfilFormatoDocumentalInvalido) {
		t.Fatalf("limite no acotado aceptado: %v", err)
	}
	if _, err := NuevoPerfilFormatoDocumental(
		referenciaV1, identidad, "application/pdf", "pdf", "binario",
		EstadoPerfilDocumentalVigente, capacidades,
	); !errors.Is(err, ErrPerfilFormatoDocumentalInvalido) {
		t.Fatalf("constructor sin conformidad ni limite concedio autoridad: %v", err)
	}
}

func TestConformidadDocumentalEsVersionadaDeclarativaYCerrada(t *testing.T) {
	conformidad := conformidadDocumentalPrueba(t, 3, strings.Repeat("a", 64), strings.Repeat("b", 64))
	if conformidad.Validar() != nil || conformidad.Version() != 3 ||
		conformidad.EsquemaRef() != "esquema:pdfa4:1" ||
		conformidad.DialectoRef() != "dialecto:pdfa4" ||
		conformidad.CanonicalizacionRef() != "canonicalizacion:pdf:1" ||
		conformidad.ReglasRef() != "reglas:pdfa4:1" ||
		conformidad.PoliticaRef() != "politica:documental:1" ||
		len(conformidad.DigestSHA256()) != 64 {
		t.Fatalf("conformidad invalida: %#v", conformidad)
	}

	casos := []struct {
		nombre                           string
		esquema, dialecto, canon, reglas string
		politica                         string
	}{
		{"url", "https://example.test/esquema", "dialecto:pdfa4", "canonicalizacion:pdf:1", "reglas:pdfa4:1", "politica:documental:1"},
		{"codigo", "esquema:pdfa4:1", "dialecto:pdfa4", "canonicalizacion:pdf:1", "reglas;ejecutar", "politica:documental:1"},
		{"comodin", "esquema:pdfa4:1", "dialecto:*", "canonicalizacion:pdf:1", "reglas:pdfa4:1", "politica:documental:1"},
		{"politica url", "esquema:pdfa4:1", "dialecto:pdfa4", "canonicalizacion:pdf:1", "reglas:pdfa4:1", "https://example.test/politica"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := NuevaReferenciaConformidadDocumental(
				"conformidad:pdfa4", 1, caso.esquema, caso.dialecto, caso.canon,
				caso.reglas, strings.Repeat("a", 64), caso.politica, strings.Repeat("b", 64),
			)
			if !errors.Is(err, ErrConformidadDocumentalInvalida) {
				t.Fatalf("referencia ejecutable o abierta aceptada: %v", err)
			}
		})
	}
	if _, err := NuevaReferenciaConformidadDocumental(
		"conformidad:pdfa4", 1, "esquema:pdfa4:1", "dialecto:pdfa4",
		"canonicalizacion:pdf:1", "reglas:pdfa4:1", "no-es-huella",
		"politica:documental:1", strings.Repeat("b", 64),
	); !errors.Is(err, ErrConformidadDocumentalInvalida) {
		t.Fatalf("huella de reglas invalida aceptada: %v", err)
	}
	if (ReferenciaConformidadDocumental{}).Validar() == nil {
		t.Fatal("conformidad cero obtuvo semantica positiva")
	}
}

func TestSituacionOperativaEsAppendOnlyYNoContaminaPerfilHistorico(t *testing.T) {
	perfil := perfilPDFGobernadoPrueba(t)
	revision, err := NuevaRevisionCatalogoFormatosDocumentales(7, strings.Repeat("e", 64))
	if err != nil {
		t.Fatal(err)
	}
	vigente, err := NuevaSituacionOperativaPerfilDocumental(
		"publicacion:pdfa4:1", perfil, revision, 1, EstadoPublicacionPerfilVigente,
	)
	if err != nil || vigente.Validar() != nil || vigente.Secuencia() != 1 ||
		!vigente.Coincide(perfil, revision) {
		t.Fatalf("situacion inicial invalida: %#v, %v", vigente, err)
	}
	huellaPerfilAntes := perfil.DigestSHA256()
	revocada, err := NuevaSituacionOperativaPerfilDocumental(
		"publicacion:pdfa4:1", perfil, revision, 2, EstadoPublicacionPerfilRevocada,
	)
	if err != nil || !revocada.EsSucesoraDe(vigente) ||
		revocada.Estado() != EstadoPublicacionPerfilRevocada {
		t.Fatalf("revocacion no encadenada: %#v, %v", revocada, err)
	}
	if perfil.DigestSHA256() != huellaPerfilAntes || perfil.Validar() != nil {
		t.Fatal("la revocacion reescribio el perfil historico")
	}
	reactivada, _ := NuevaSituacionOperativaPerfilDocumental(
		"publicacion:pdfa4:1", perfil, revision, 3, EstadoPublicacionPerfilVigente,
	)
	if reactivada.EsSucesoraDe(revocada) {
		t.Fatal("una publicacion revocada fue reactivada en la misma cadena")
	}
	salto, _ := NuevaSituacionOperativaPerfilDocumental(
		"publicacion:pdfa4:1", perfil, revision, 4, EstadoPublicacionPerfilRetirada,
	)
	if salto.EsSucesoraDe(vigente) {
		t.Fatal("se acepto un salto en la secuencia append-only")
	}
}

func TestPerfilDocumentalRechazaSintaxisAmbiguaYCapacidadesAbiertas(t *testing.T) {
	for _, identificador := range []string{"", "PDF", "pdf*", "https://formatos.example/pdf"} {
		if _, err := NuevaIdentidadSintacticaDocumental(identificador); !errors.Is(err, ErrIdentidadSintacticaDocumentalInvalida) {
			t.Fatalf("identidad no canonica aceptada: %q, %v", identificador, err)
		}
	}
	if _, err := NuevasCapacidadesPerfilFormatoDocumental(
		CapacidadPerfilRenderizar, CapacidadPerfilFormatoDocumental("ejecutar_comando"),
	); !errors.Is(err, ErrPerfilFormatoDocumentalInvalido) {
		t.Fatalf("capacidad no compilada aceptada: %v", err)
	}
	if _, err := NuevasCapacidadesPerfilFormatoDocumental(
		CapacidadPerfilRenderizar, CapacidadPerfilRenderizar,
	); !errors.Is(err, ErrPerfilFormatoDocumentalInvalido) {
		t.Fatalf("capacidad duplicada aceptada: %v", err)
	}

	identidad, _ := NuevaIdentidadSintacticaDocumental("archivo")
	referencia, _ := NuevaReferenciaPerfilDocumental("tar-gzip", 1)
	capacidades, _ := NuevasCapacidadesPerfilFormatoDocumental(CapacidadPerfilRenderizar)
	conformidad := conformidadDocumentalPrueba(t, 1, strings.Repeat("a", 64), strings.Repeat("b", 64))
	perfil := perfilDocumentalPrueba(
		t, referencia, identidad, capacidades, conformidad, 1024, "tar.gz",
	)
	if perfil.Extension() != "tar.gz" {
		t.Fatalf("extension compuesta rechazada: %#v", perfil)
	}
	for _, extension := range []string{".tar.gz", "tar.gz.", "tar..gz", "tar/gz", "tar\\gz"} {
		if _, err := NuevoPerfilFormatoDocumentalConforme(
			referencia, identidad, "application/x-tar", extension, "binario",
			capacidades, conformidad, 1024,
		); !errors.Is(err, ErrPerfilFormatoDocumentalInvalido) {
			t.Fatalf("extension ambigua aceptada: %q, %v", extension, err)
		}
	}
}

func TestReferenciaGobernadaValidaEquivaleAlContratoHistorico(t *testing.T) {
	t.Parallel()

	referencia := regexp.MustCompile(`^[a-z][a-z0-9._:-]{0,255}$`)
	comprobar := func(valor string) {
		t.Helper()
		if obtenido, esperado := referenciaGobernadaValida(valor), referencia.MatchString(valor); obtenido != esperado {
			t.Fatalf("validacion de referencia no equivalente: %q obtenido=%v esperado=%v",
				valor, obtenido, esperado)
		}
	}

	for _, longitud := range []int{0, 1, 2, 255, 256, 257} {
		base := strings.Repeat("a", longitud)
		comprobar(base)
		if longitud == 0 {
			continue
		}
		for _, posicion := range []int{0, longitud / 2, longitud - 1} {
			for octeto := 0; octeto <= 255; octeto++ {
				candidato := []byte(base)
				candidato[posicion] = byte(octeto)
				comprobar(string(candidato))
			}
		}
	}
	for _, valor := range []string{
		"A", "aB", "1a", "a/b", "a b", "a\n", "á", "aá",
		"Ａ", "aİ", "aK", string([]byte{0xff}), "a" + string([]byte{0xff}),
	} {
		comprobar(valor)
	}
}

func TestHuellaSHA256DocumentalGobernadaEquivaleAlContratoHistorico(t *testing.T) {
	t.Parallel()

	referencia := func(valor string) bool {
		if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) ||
			valor != strings.TrimSpace(valor) {
			return false
		}
		decodificada, err := hex.DecodeString(valor)
		return err == nil && len(decodificada) == sha256.Size
	}
	comprobar := func(valor string) {
		t.Helper()
		obtenido := esHuellaSHA256DocumentalGobernada(valor)
		canonico := esSHA256(valor)
		esperado := referencia(valor)
		if obtenido != canonico || canonico != esperado {
			t.Fatalf("validacion SHA-256 no equivalente: %q documental=%v canonica=%v esperada=%v",
				valor, obtenido, canonico, esperado)
		}
	}

	for longitud := 0; longitud <= sha256.Size*2+2; longitud++ {
		comprobar(strings.Repeat("a", longitud))
	}
	for _, base := range []string{
		strings.Repeat("0", sha256.Size*2),
		strings.Repeat("9", sha256.Size*2),
		strings.Repeat("a", sha256.Size*2),
		strings.Repeat("f", sha256.Size*2),
	} {
		for posicion := range len(base) {
			for octeto := 0; octeto <= 255; octeto++ {
				candidato := []byte(base)
				candidato[posicion] = byte(octeto)
				comprobar(string(candidato))
			}
		}
	}
	for _, valor := range []string{
		strings.Repeat("A", sha256.Size*2),
		strings.Repeat("F", sha256.Size*2),
		" " + strings.Repeat("a", sha256.Size*2-1),
		strings.Repeat("a", sha256.Size*2-1) + "\n",
		"á" + strings.Repeat("a", sha256.Size*2-2),
		"Ａ" + strings.Repeat("a", sha256.Size*2-3),
		"İ" + strings.Repeat("a", sha256.Size*2-2),
		string([]byte{0xff}) + strings.Repeat("a", sha256.Size*2-1),
	} {
		comprobar(valor)
	}
}

func TestReferenciaComponenteDocumentalFijaRolHomologacionYArtefacto(t *testing.T) {
	marcador, err := NuevaReferenciaComponenteDocumental(
		RolComponenteMarcador, "marcador-pdfa", 3, "homologacion:marcador-pdfa:3",
		strings.Repeat("a", 64), strings.Repeat("b", 64),
	)
	if err != nil || marcador.Validar() != nil || marcador.Rol() != RolComponenteMarcador ||
		marcador.HuellaArtefactoSHA256() != strings.Repeat("b", 64) {
		t.Fatalf("componente atestado invalido: %#v, %v", marcador, err)
	}
	otroArtefacto, err := NuevaReferenciaComponenteDocumental(
		RolComponenteMarcador, "marcador-pdfa", 3, "homologacion:marcador-pdfa:3",
		strings.Repeat("a", 64), strings.Repeat("c", 64),
	)
	if err != nil || otroArtefacto == marcador {
		t.Fatal("un binario distinto compartio la referencia exacta del componente")
	}
	extractor, _ := NuevaReferenciaComponenteDocumental(
		RolComponenteExtractorMetadatos, "marcador-pdfa", 3, "homologacion:marcador-pdfa:3",
		strings.Repeat("a", 64), strings.Repeat("b", 64),
	)
	if extractor == marcador {
		t.Fatal("el rol no forma parte de la referencia del componente")
	}
	if _, err := NuevaReferenciaComponenteDocumental(
		RolComponenteDocumental("libre"), "componente", 1, "homologacion:componente:1",
		strings.Repeat("a", 64), strings.Repeat("b", 64),
	); !errors.Is(err, ErrReferenciaComponenteDocumentalInvalida) {
		t.Fatalf("rol abierto aceptado: %v", err)
	}
	if (ReferenciaComponenteDocumental{}).Validar() == nil {
		t.Fatal("componente cero obtuvo semantica positiva")
	}
}

func TestRolesVerificacionDocumentalSeparanEstructuraYSemantica(t *testing.T) {
	if RolComponenteValidadorEstructural != RolComponenteVerificador {
		t.Fatal("el alias estructural cambio el valor historico del protocolo")
	}
	if !RolComponenteValidadorEstructural.Valido() ||
		!RolComponenteVerificadorSemantico.Valido() ||
		RolComponenteValidadorEstructural == RolComponenteVerificadorSemantico {
		t.Fatal("estructura y semantica no quedaron como roles cerrados distintos")
	}
	semantico, err := NuevaReferenciaComponenteDocumental(
		RolComponenteVerificadorSemantico, "verificador-semantico-pdfa", 1,
		"homologacion:verificador-semantico-pdfa:1", strings.Repeat("c", 64),
		strings.Repeat("d", 64),
	)
	if err != nil || semantico.Validar() != nil ||
		semantico.Rol() != RolComponenteVerificadorSemantico {
		t.Fatalf("rol semantico no publicable: %#v, %v", semantico, err)
	}
}

func TestMarcaInstitucionalUsaUUIDV4YReferenciasOpacas(t *testing.T) {
	// El dominio solo conserva referencias opacas. Esta referencia de apariencia
	// personal pasa la sintaxis, pero no queda acreditada: la politica positiva
	// institucional de aplicacion debe rechazarla.
	institucionOpaca, err := NuevaReferenciaInstitucionalDocumento(
		"persona:12345678z", "organo:seleccion",
	)
	if err != nil {
		t.Fatalf("el dominio intento acreditar por prefijo una referencia opaca: %v", err)
	}
	institucion, _ := NuevaReferenciaInstitucionalDocumento(
		"entidad:diputacion_granada", "organo:servicio_seleccion",
	)
	perfil, _ := NuevaReferenciaPerfilDocumental("pdfa-4", 2)
	fecha := time.Date(2026, time.July, 15, 10, 11, 12, 345000000, time.UTC)
	uuidV4 := "018f6f2a-6d92-4cc3-8c5e-3fb3e8e8a123"
	marca, err := NuevaMarcaInstitucionalDocumento(
		institucion, uuidV4, perfil, fecha, "manifiesto:documento:001",
		"https://sede.dipgra.es/documentos/"+uuidV4,
	)
	if err != nil || marca.Validar() != nil || marca.DocumentoUUID() != uuidV4 ||
		marca.Institucion() != institucion || marca.Perfil() != perfil ||
		marca.ManifiestoRef() != "manifiesto:documento:001" {
		t.Fatalf("marca institucional valida rechazada: %#v, %v", marca, err)
	}
	huella, err := marca.HuellaSHA256()
	if err != nil || len(huella) != 64 {
		t.Fatalf("huella de metadato invalida: %q, %v", huella, err)
	}
	if _, err := NuevaMarcaInstitucionalDocumento(
		institucionOpaca, "018f6f2a-6d92-7cc3-8c5e-3fb3e8e8a123", perfil,
		fecha, "manifiesto:documento:001", "",
	); !errors.Is(err, ErrMarcaInstitucionalDocumentoInvalida) {
		t.Fatalf("UUID v7 aceptado donde se exige UUID v4: %v", err)
	}
	// Un host HTTPS publico puede ser sintacticamente pasivo, pero no queda
	// autorizado por ello. La allowlist se comprueba al construirlo en politica.
	if _, err := NuevaMarcaInstitucionalDocumento(
		institucion, uuidV4, perfil, fecha, "manifiesto:documento:001",
		"https://externo.example/documentos/"+uuidV4,
	); err != nil {
		t.Fatalf("el dominio confundio sintaxis pasiva con politica institucional: %v", err)
	}
	for nombre, uri := range map[string]string{
		"http":         "http://sede.dipgra.es/documentos/" + uuidV4,
		"credenciales": "https://usuario:secreto@sede.dipgra.es/documentos/" + uuidV4,
		"consulta":     "https://sede.dipgra.es/documentos/" + uuidV4 + "?token=secreto",
		"host interno": "https://servidor.local/documentos/" + uuidV4,
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := NuevaMarcaInstitucionalDocumento(
				institucion, uuidV4, perfil, fecha, "manifiesto:documento:001", uri,
			); !errors.Is(err, ErrMarcaInstitucionalDocumentoInvalida) {
				t.Fatalf("URI activa o no publica aceptada: %v", err)
			}
		})
	}
	if (MarcaInstitucionalDocumento{}).Validar() == nil {
		t.Fatal("marca cero obtuvo semantica positiva")
	}
}

func conformidadDocumentalPrueba(
	t *testing.T,
	version uint64,
	huellaReglas, huellaPolitica string,
) ReferenciaConformidadDocumental {
	t.Helper()
	conformidad, err := NuevaReferenciaConformidadDocumental(
		"conformidad:pdfa4", version, "esquema:pdfa4:1", "dialecto:pdfa4",
		"canonicalizacion:pdf:1", "reglas:pdfa4:1", huellaReglas,
		"politica:documental:1", huellaPolitica,
	)
	if err != nil {
		t.Fatalf("crear conformidad: %v", err)
	}
	return conformidad
}

func perfilPDFGobernadoPrueba(t *testing.T) PerfilFormatoDocumental {
	t.Helper()
	identidad, _ := NuevaIdentidadSintacticaDocumental("pdf")
	referencia, _ := NuevaReferenciaPerfilDocumental("pdfa-4", 1)
	capacidades, _ := NuevasCapacidadesPerfilFormatoDocumental(
		CapacidadPerfilRenderizar, CapacidadPerfilMetadatoInstitucional,
	)
	return perfilDocumentalPrueba(
		t, referencia, identidad, capacidades,
		conformidadDocumentalPrueba(t, 1, strings.Repeat("a", 64), strings.Repeat("b", 64)),
		32*1024*1024, "pdf",
	)
}

func perfilDocumentalPrueba(
	t *testing.T,
	referencia ReferenciaPerfilDocumental,
	identidad IdentidadSintacticaDocumental,
	capacidades CapacidadesPerfilFormatoDocumental,
	conformidad ReferenciaConformidadDocumental,
	maximoBytes uint64,
	extension string,
) PerfilFormatoDocumental {
	t.Helper()
	mime := "application/pdf"
	if extension == "tar.gz" {
		mime = "application/x-tar"
	}
	perfil, err := NuevoPerfilFormatoDocumentalConforme(
		referencia, identidad, mime, extension, "binario", capacidades, conformidad, maximoBytes,
	)
	if err != nil {
		t.Fatalf("crear perfil: %v", err)
	}
	return perfil
}
