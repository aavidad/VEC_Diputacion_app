package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type materialAADDescifradoDesarrolloPrueba struct {
	Esquema                   string `json:"esquema"`
	VersionRef                string `json:"version_ref"`
	VersionRevision           int    `json:"version_revision"`
	HuellaVersionSHA256       string `json:"huella_version_sha256"`
	EsquemaMaterial           string `json:"esquema_material"`
	HuellaMaterialSHA256      string `json:"huella_material_sha256"`
	PerfilCifradoRef          string `json:"perfil_cifrado_ref"`
	PerfilCifradoVersion      uint32 `json:"perfil_cifrado_version"`
	HuellaPerfilSHA256        string `json:"huella_perfil_cifrado_sha256"`
	AlgoritmoAEAD             string `json:"algoritmo_aead"`
	AlgoritmoEnvolturaClave   string `json:"algoritmo_envoltura_clave"`
	EvidenciaPerfilRef        string `json:"evidencia_perfil_ref"`
	EvidenciaPerfilVersion    uint32 `json:"evidencia_perfil_version"`
	HuellaEvidenciaPerfil     string `json:"huella_evidencia_perfil_sha256"`
	DecisionPoliticaRef       string `json:"decision_politica_ref"`
	DecisionPoliticaVersion   uint32 `json:"decision_politica_version"`
	HuellaDecisionPolitica    string `json:"huella_decision_politica_sha256"`
	LocalizadorEsquema        uint16 `json:"localizador_esquema"`
	LocalizadorDominio        string `json:"localizador_dominio"`
	LocalizadorClaveRef       string `json:"localizador_clave_ref"`
	LocalizadorGeneracion     uint32 `json:"localizador_generacion"`
	LocalizadorHMACSHA256     string `json:"localizador_hmac_sha256"`
	HuellaSolicitudEsquema    uint16 `json:"huella_solicitud_esquema"`
	HuellaSolicitudDominio    string `json:"huella_solicitud_dominio"`
	HuellaSolicitudClaveRef   string `json:"huella_solicitud_clave_ref"`
	HuellaSolicitudGeneracion uint32 `json:"huella_solicitud_generacion"`
	HuellaSolicitudHMACSHA256 string `json:"huella_solicitud_hmac_sha256"`
	RevisionDiario            uint64 `json:"revision_diario"`
	CercadoDiario             uint64 `json:"cercado_diario"`
	ArrendamientoIniciaEn     string `json:"arrendamiento_inicia_en"`
	ArrendamientoVenceEn      string `json:"arrendamiento_vence_en"`
	AtestacionSelladoRef      string `json:"atestacion_sellado_ref"`
	AtestacionSelladoVersion  uint32 `json:"atestacion_sellado_version"`
	HuellaAtestacionSellado   string `json:"huella_atestacion_sellado_sha256"`
	TokenConsumoSelladoRef    string `json:"token_consumo_sellado_ref"`
	HuellaCorrelacionSHA256   string `json:"huella_correlacion_sha256"`
	ProcedenciaEsquema        string `json:"procedencia_esquema"`
	PerfilEjecucion           string `json:"perfil_ejecucion"`
	AutoridadActo             string `json:"autoridad_acto"`
	ProveedorProcedenciaRef   string `json:"proveedor_procedencia_ref"`
	MigrableProduccion        bool   `json:"migrable_produccion"`
}

func versionDescifradoDesarrolloPrueba(t *testing.T) dominiobolsa.VersionConvocatoriaGobernada {
	t.Helper()
	base := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	ambito, err := dominiobolsa.NuevoAmbitoOrganizativoConvocatoria(
		"org_diputaciongranada", "uni_seleccionexterna",
	)
	if err != nil {
		t.Fatal(err)
	}
	huella := func(marca string) string { return strings.Repeat(marca, sha256.Size*2) }
	contenido := dominiobolsa.ContenidoPublicableConvocatoria{
		IdentificadorPublico: "auxiliar-administrativo-2026", Tipo: "bolsa_temporal",
		CatalogoCategorias: dominiobolsa.ReferenciaCatalogoCategorias{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256: huella("a"),
		},
		Categorias: []string{"auxiliar_administrativo"}, Titulo: "Bolsa temporal de desarrollo",
		Resumen: "Resumen público de desarrollo.", Descripcion: "Descripción pública de desarrollo.",
		Plazos: []dominiobolsa.PlazoConvocatoria{{
			Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripción",
			Descripcion: "Plazo público.", AbreEn: base, CierraEn: base.Add(30 * 24 * time.Hour),
		}},
		Documentos: []dominiobolsa.DocumentoPublicableConvocatoria{{
			Referencia: "doc:bases", Tipo: "bases", Orden: 1, Titulo: "Bases",
			Descripcion: "Documento de desarrollo.", Formato: "html",
			URL: "/bolsa/documentos/bases-desarrollo.html",
		}},
	}
	referencia := func(id string, version int, marca string) dominiobolsa.ReferenciaConfiguracionConvocatoria {
		return dominiobolsa.ReferenciaConfiguracionConvocatoria{
			ID: id, Version: version, HuellaContenidoSHA256: huella(marca),
		}
	}
	configuracion := dominiobolsa.ConfiguracionFijadaConvocatoria{
		Catalogos:        referencia("catalogos:bolsa", 1, "1"),
		Calendario:       referencia("calendario:bolsa", 1, "2"),
		ReglasBaremacion: referencia("baremo:bolsa", 1, "3"),
		FlujoProceso:     referencia("convocatoria-bolsa", 1, "4"),
		FlujoSolicitud:   referencia("solicitud-bolsa", 1, "5"),
		Plantilla:        referencia("plantilla:bolsa:desarrollo", 1, "8"),
		Documentos: []dominiobolsa.ReferenciaDocumentoOficialConvocatoria{{
			Rol: "bases", PublicacionRef: "doc:bases", DocumentoRef: "documento:logico:bases:001",
			VersionDocumento: 1, RepresentacionRef: "representacion:html:bases:001",
			HuellaContenidoSHA256: huella("6"), FirmaValidadaRef: "firma:validada:bases:001",
			ReciboCustodiaRef: "custodia:bases:001",
		}},
	}
	version, err := dominiobolsa.NuevaVersionConvocatoriaGobernada(
		dominiobolsa.DatosNuevaVersionConvocatoriaGobernada{
			ID: "proceso:bolsa:desarrollo-2026", CodigoVersionPublica: "v1",
			InstanciaFlujoRef:  "instancia:flujo:convocatoria:desarrollo:001",
			AmbitoOrganizativo: ambito, Contenido: contenido, Configuracion: configuracion,
			ExpedienteRef: "expediente:seleccion:desarrollo-2026-001",
			Motivo:        "Creación controlada para probar el KMS de desarrollo.",
			ActorID:       "persona:tecnica:desarrollo:001", Instante: base,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return version
}

func solicitudDescifradoKMSDesarrolloPrueba(
	t *testing.T,
	kms *emisorKMSDesarrollo,
	claroAlternativo []byte,
) (gobiernoconvocatorias.SolicitudDescifradoBorradorDurable, []byte) {
	t.Helper()
	version := versionDescifradoDesarrolloPrueba(t)
	claroCanonico, err := version.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	claroCifrado := claroCanonico
	if claroAlternativo != nil {
		claroCifrado = claroAlternativo
	}
	estado, err := puertosbolsa.EstadoVersionConvocatoria(version)
	if err != nil {
		t.Fatal(err)
	}
	perfil, err := gobiernoconvocatorias.NuevoPerfilCifradoBorrador(
		"perfil:cifrado:desarrollo:v1", 1, strings.Repeat("a", sha256.Size*2),
		algoritmoContenidoDesarrollo, algoritmoEnvolturaDesarrollo,
	)
	if err != nil {
		t.Fatal(err)
	}
	procedencia, err := gobiernoconvocatorias.NuevaProcedenciaActoBorrador(
		config.ExecutionProfileDevelopment, gobiernoconvocatorias.AutoridadActoNoAutoritativa,
		proveedorSeguridadDesarrolloRef, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	materialAAD := materialAADDescifradoDesarrolloPrueba{
		Esquema: "bolsa.convocatoria.borrador.aad.v1", VersionRef: estado.Referencia,
		VersionRevision: estado.Revision, HuellaVersionSHA256: estado.HuellaEstadoSHA256,
		EsquemaMaterial:      puertosbolsa.EsquemaMaterialIntencionGobiernoConvocatoriaV2,
		HuellaMaterialSHA256: strings.Repeat("b", sha256.Size*2),
		PerfilCifradoRef:     perfil.Referencia, PerfilCifradoVersion: perfil.Version,
		HuellaPerfilSHA256: perfil.HuellaContenidoSHA256, AlgoritmoAEAD: perfil.AlgoritmoAEAD,
		AlgoritmoEnvolturaClave: perfil.AlgoritmoEnvolturaClave,
		EvidenciaPerfilRef:      "evidencia:perfil:desarrollo:001", EvidenciaPerfilVersion: 1,
		HuellaEvidenciaPerfil: strings.Repeat("c", sha256.Size*2),
		DecisionPoliticaRef:   "decision:politica:desarrollo:001", DecisionPoliticaVersion: 1,
		HuellaDecisionPolitica: strings.Repeat("d", sha256.Size*2),
		LocalizadorEsquema:     1, LocalizadorDominio: "localizador",
		LocalizadorClaveRef: "clave:hmac:localizador:desarrollo:v1", LocalizadorGeneracion: 1,
		LocalizadorHMACSHA256:  strings.Repeat("e", sha256.Size*2),
		HuellaSolicitudEsquema: 1, HuellaSolicitudDominio: "huella_solicitud",
		HuellaSolicitudClaveRef: "clave:hmac:solicitud:desarrollo:v1", HuellaSolicitudGeneracion: 1,
		HuellaSolicitudHMACSHA256: strings.Repeat("f", sha256.Size*2),
		RevisionDiario:            1, CercadoDiario: 1,
		ArrendamientoIniciaEn: base.Format(time.RFC3339Nano),
		ArrendamientoVenceEn:  base.Add(5 * time.Minute).Format(time.RFC3339Nano),
		AtestacionSelladoRef:  "atestacion:motivo:desarrollo:001", AtestacionSelladoVersion: 1,
		HuellaAtestacionSellado: strings.Repeat("7", sha256.Size*2),
		TokenConsumoSelladoRef:  "consumo:motivo:desarrollo:001",
		HuellaCorrelacionSHA256: strings.Repeat("8", sha256.Size*2),
		ProcedenciaEsquema:      procedencia.Esquema, PerfilEjecucion: procedencia.PerfilEjecucion,
		AutoridadActo: procedencia.Autoridad, ProveedorProcedenciaRef: procedencia.ProveedorRef,
		MigrableProduccion: procedencia.MigrableProduccion,
	}
	aadBytes, err := json.Marshal(materialAAD)
	if err != nil {
		t.Fatal(err)
	}
	huellaAADBytes := sha256.Sum256(aadBytes)
	huellaAAD := hex.EncodeToString(huellaAADBytes[:])
	aad, err := gobiernoconvocatorias.RestaurarAADCanonicaCifradoBorrador(aadBytes, huellaAAD)
	if err != nil {
		t.Fatal(err)
	}
	kms.aleatorio = bytes.NewReader(bytes.Repeat([]byte{0x5a}, 44))
	nonce, cifrado, envuelta, err := kms.cifrarContenido(claroCifrado, aadBytes)
	if err != nil {
		t.Fatal(err)
	}
	envoltura, err := gobiernoconvocatorias.NuevaEnvolturaClaveKMSBorrador(
		perfil, claveMaestraDesarrolloRef, 1, envuelta, huellaAAD,
	)
	if err != nil {
		t.Fatal(err)
	}
	sobre, err := gobiernoconvocatorias.NuevoSobreCifradoAEADBorrador(perfil, nonce, cifrado, huellaAAD)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, _, huellaEnvoltura, err := envoltura.DatosParaPersistencia()
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, huellaSobre, err := sobre.DatosParaPersistencia()
	if err != nil {
		t.Fatal(err)
	}
	atestacion, err := gobiernoconvocatorias.NuevaAtestacionKMSBorrador(
		"atestacion:kms:desarrollo:descifrado:001", 1, perfil, claveMaestraDesarrolloRef, 1,
		huellaAAD, huellaEnvoltura, huellaSobre, verificadorAtestacionDesarrolloRef,
		procedencia, algoritmoFirmaKMSDesarrollo, hex.EncodeToString(kms.huellaPublicaAtestacion[:]),
		base, base.Add(4*time.Minute), func(preimagen []byte) ([]byte, error) {
			return ed25519.Sign(kms.firmaAtestacion, preimagen), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := gobiernoconvocatorias.NuevaSolicitudDescifradoBorradorDurable(
		estado, aad, perfil, envoltura, sobre, atestacion, procedencia,
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, claroCanonico
}

func TestKMSDesarrolloDescifraBorradorDurableConcurrente(t *testing.T) {
	kms, _, _ := nuevosProveedoresKMSPrueba(t)
	solicitud, claro := solicitudDescifradoKMSDesarrolloPrueba(t, kms, nil)
	const trabajadores = 32
	errores := make(chan error, trabajadores)
	var grupo sync.WaitGroup
	grupo.Add(trabajadores)
	for range trabajadores {
		go func() {
			defer grupo.Done()
			resultado, err := kms.DescifrarBorrador(context.Background(), solicitud)
			if err != nil {
				errores <- err
				return
			}
			version, err := resultado.VersionConvocatoria()
			if err != nil {
				errores <- err
				return
			}
			canonico, err := version.RepresentacionCanonica()
			if err != nil || !bytes.Equal(canonico, claro) {
				errores <- errors.New("versión concurrente distinta")
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatal(err)
	}
}

func TestKMSDesarrolloDescifradoFallaConFirmaClaveYClaroAjenos(t *testing.T) {
	kms, _, _ := nuevosProveedoresKMSPrueba(t)
	solicitud, claro := solicitudDescifradoKMSDesarrolloPrueba(t, kms, nil)

	atestacionAlterada := solicitud.AtestacionKMS
	preimagen, algoritmo, verificador, huellaPublica, firma, err :=
		atestacionAlterada.DatosParaVerificacionFirma()
	if err != nil {
		t.Fatal(err)
	}
	firma[0] ^= 0x80
	firmaAlterada, err := gobiernoconvocatorias.RestaurarFirmaEvidenciaBorrador(
		algoritmo, verificador, huellaPublica, preimagen,
		base64.RawURLEncoding.EncodeToString(firma),
	)
	if err != nil {
		t.Fatal(err)
	}
	atestacionAlterada.Firma = firmaAlterada
	solicitudFirma := solicitud
	solicitudFirma.AtestacionKMS = atestacionAlterada
	if solicitudFirma.Validar() != nil {
		t.Fatal("la firma alterada debía alcanzar la verificación criptográfica")
	}
	if _, err := kms.DescifrarBorrador(context.Background(), solicitudFirma); err == nil {
		t.Fatal("se aceptó atestación A con firma alterada")
	}
	_, privadaAjena, err := ed25519.GenerateKey(
		bytes.NewReader(bytes.Repeat([]byte{0x77}, ed25519.SeedSize)),
	)
	if err != nil {
		t.Fatal(err)
	}
	firmaDeClaveAjena, err := gobiernoconvocatorias.RestaurarFirmaEvidenciaBorrador(
		algoritmo, verificador, huellaPublica, preimagen,
		base64.RawURLEncoding.EncodeToString(ed25519.Sign(privadaAjena, preimagen)),
	)
	if err != nil {
		t.Fatal(err)
	}
	atestacionClaveAjena := solicitud.AtestacionKMS
	atestacionClaveAjena.Firma = firmaDeClaveAjena
	solicitudFirmaAjena := solicitud
	solicitudFirmaAjena.AtestacionKMS = atestacionClaveAjena
	if solicitudFirmaAjena.Validar() != nil {
		t.Fatal("la firma ajena debía alcanzar la verificación criptográfica")
	}
	if _, err := kms.DescifrarBorrador(context.Background(), solicitudFirmaAjena); err == nil {
		t.Fatal("se aceptó atestación A firmada por una clave Ed25519 ajena")
	}

	claveAjena := *kms
	claveAjena.claveEnvoltura[0] ^= 0x01
	if _, err := claveAjena.DescifrarBorrador(context.Background(), solicitud); err == nil {
		t.Fatal("se aceptó una clave maestra de envoltura ajena")
	}

	noCanonico := append(append([]byte(nil), claro...), '\n')
	solicitudNoCanonica, _ := solicitudDescifradoKMSDesarrolloPrueba(t, kms, noCanonico)
	if _, err := kms.DescifrarBorrador(context.Background(), solicitudNoCanonica); err == nil {
		t.Fatal("se devolvió texto claro no canónico")
	}
}
