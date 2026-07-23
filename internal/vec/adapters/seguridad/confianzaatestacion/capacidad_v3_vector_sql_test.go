package confianzaatestacion

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	puertoscontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type entradaVectorSQLO205 struct {
	Caso                   string                       `json:"caso"`
	Ahora                  string                       `json:"ahora"`
	DecisionPlantillaB64   string                       `json:"decision_plantilla_b64"`
	MotivoB64              string                       `json:"motivo_b64"`
	ContextoB64            string                       `json:"contexto_b64"`
	ManifiestoB64          string                       `json:"manifiesto_b64"`
	ManifiestoHuellaSHA256 string                       `json:"manifiesto_huella_sha256"`
	AutoridadEfectiva      string                       `json:"autoridad_efectiva"`
	ResueltoEn             string                       `json:"resuelto_en"`
	AltaB64                string                       `json:"alta_b64"`
	SellosB64              string                       `json:"sellos_b64"`
	EfectoHuellaSHA256     string                       `json:"efecto_huella_sha256"`
	ClaveID                string                       `json:"clave_id"`
	ClaveVersion           uint64                       `json:"clave_version"`
	RevisionGobierno       uint64                       `json:"revision_gobierno"`
	HuellaGobiernoSHA256   string                       `json:"huella_gobierno_sha256"`
	EmisorID               string                       `json:"emisor_id"`
	AudienciaConsumo       string                       `json:"audiencia_consumo"`
	ClaveHMACB64           string                       `json:"clave_hmac_b64"`
	ClaveValidaDesde       string                       `json:"clave_valida_desde"`
	ClaveValidaHasta       string                       `json:"clave_valida_hasta"`
	RevisionConfianza      string                       `json:"revision_confianza"`
	SecuenciaConfianza     uint64                       `json:"secuencia_confianza"`
	RaizClaveID            string                       `json:"raiz_clave_id"`
	RaizVersion            uint64                       `json:"raiz_version"`
	AudienciaDespliegue    string                       `json:"audiencia_despliegue"`
	Politicas              []domain.PoliticaRestrictiva `json:"politicas"`
	RevisionCatalogo       uint64                       `json:"revision_catalogo"`
	HuellaCatalogoSHA256   string                       `json:"huella_catalogo_sha256"`
	AsignacionID           string                       `json:"asignacion_id"`
	AsignacionVersion      int                          `json:"asignacion_version"`
	PersonaVersion         uint64                       `json:"persona_version"`
	PerfilVersion          uint64                       `json:"perfil_version"`
}

type decisionPlantillaO205 struct {
	DecisionRef               string                                  `json:"decision_ref"`
	Accion                    string                                  `json:"accion"`
	RecursoRef                string                                  `json:"recurso_ref"`
	ModuloID                  string                                  `json:"modulo_id"`
	TipoRecurso               string                                  `json:"tipo_recurso"`
	Finalidad                 string                                  `json:"finalidad"`
	CorrelacionRef            string                                  `json:"correlacion_ref"`
	VinculoAutenticacionActor domain.DatosVinculoAutenticacionActorV2 `json:"vinculo_autenticacion_actor"`
}

type motivoCanonicoO205 struct {
	Referencia domain.ReferenciaEntradaCatalogo `json:"referencia"`
}

type efectoAltaO205 struct {
	OrganizacionRef string `json:"organizacion_ref"`
	Solicitud       struct {
		CentroRef    string `json:"centro_ref"`
		CategoriaRef string `json:"categoria_ref"`
	} `json:"solicitud"`
	Flujo struct {
		DefinicionRef string `json:"definicion_ref"`
		Version       uint64 `json:"version"`
		HuellaSHA256  string `json:"huella_sha256"`
	} `json:"flujo"`
}

type sellosO205 struct {
	Activo struct {
		HuellaHMAC string `json:"huella_hmac"`
	} `json:"activo"`
}

type bundleVectorO205 struct {
	CapacidadB64        string          `json:"capacidad_b64"`
	DecisionB64         string          `json:"decision_b64"`
	MotivoB64           string          `json:"motivo_b64"`
	ContextoB64         string          `json:"contexto_b64"`
	PayloadB64          string          `json:"payload_b64"`
	COSEB64             string          `json:"cose_b64"`
	EvidenciaB64        string          `json:"evidencia_b64"`
	SPKIB64             string          `json:"spki_b64"`
	AltaB64             string          `json:"alta_b64"`
	SellosB64           string          `json:"sellos_b64"`
	PersonaVersion      uint64          `json:"persona_version"`
	PerfilVersion       uint64          `json:"perfil_version"`
	VersionRolDocumento json.RawMessage `json:"version_rol_documento"`
	ControlRolDocumento json.RawMessage `json:"control_rol_documento"`
	AsignacionDocumento json.RawMessage `json:"asignacion_documento"`
}

func TestVectorCanonicoO205EsComunAGoYSQL(t *testing.T) {
	t.Parallel()
	ruta := filepath.Join("testdata", "capacidad_v3_canonica_o2_05.json")
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("leer vector compartido: %v", err)
	}
	contenido = bytes.TrimSuffix(contenido, []byte{'\n'})
	documento, err := interpretarExportacionCapacidadV3(contenido)
	if err != nil {
		t.Fatalf("Go rechazó el vector compartido: %v", err)
	}
	canonica, err := json.Marshal(documento)
	if err != nil {
		t.Fatalf("serializar vector compartido: %v", err)
	}
	if !bytes.Equal(canonica, contenido) {
		t.Fatal("Go no conserva exactamente los bytes canónicos compartidos")
	}
}

// TestGenerarVectorO205ParaSQL parte de un DTO cerrado del runner y atraviesa
// las superficies productivas públicas: construcción nominal, verificación
// VEC-AD-3 y EmisorCapacidadesAtestacionAutorizacionV3.Emitir. SQL consume sin
// recomponer la capacidad ni recalcular su MAC.
func TestGenerarVectorO205ParaSQL(t *testing.T) {
	rutaEntrada := os.Getenv("VEC_O205_VECTOR_ENTRADA")
	rutaSalida := os.Getenv("VEC_O205_VECTOR_SALIDA")
	if rutaEntrada == "" || rutaSalida == "" {
		t.Skip("solo se ejecuta desde la integración PostgreSQL O2-05")
	}
	contenido, err := os.ReadFile(rutaEntrada)
	if err != nil {
		t.Fatalf("leer DTO efímero SQL: %v", err)
	}
	var entrada entradaVectorSQLO205
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&entrada); err != nil ||
		entrada.Caso == "" || strings.ContainsAny(entrada.Caso, "/\\") {
		t.Fatalf("DTO efímero SQL inválido: %v", err)
	}
	bundle := generarBundleVectorO205(t, entrada)
	salida, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("serializar bundle Go: %v", err)
	}
	if err := os.WriteFile(rutaSalida, salida, 0o600); err != nil {
		t.Fatalf("escribir bundle Go: %v", err)
	}
}

func generarBundleVectorO205(t *testing.T, e entradaVectorSQLO205) bundleVectorO205 {
	t.Helper()
	ahora := exigirInstanteSQLO205(t, e.Ahora)
	resueltoEn := exigirInstanteSQLO205(t, e.ResueltoEn)
	claveDesde := exigirInstanteSQLO205(t, e.ClaveValidaDesde)
	claveHasta := exigirInstanteSQLO205(t, e.ClaveValidaHasta)
	decisionPlantillaBytes := exigirBase64O205(t, e.DecisionPlantillaB64)
	motivoBytes := exigirBase64O205(t, e.MotivoB64)
	contextoBytes := exigirBase64O205(t, e.ContextoB64)
	manifiestoBytes := exigirBase64O205(t, e.ManifiestoB64)
	altaBytes := exigirBase64O205(t, e.AltaB64)
	sellosBytes := exigirBase64O205(t, e.SellosB64)
	materialHMAC := exigirBase64O205(t, e.ClaveHMACB64)
	defer borrarBytesConfianzaAtestacion(materialHMAC)

	var plantilla decisionPlantillaO205
	var motivo motivoCanonicoO205
	var alta efectoAltaO205
	var sellos sellosO205
	for _, destino := range []struct {
		valor []byte
		en    any
	}{
		{decisionPlantillaBytes, &plantilla},
		{motivoBytes, &motivo},
		{altaBytes, &alta},
		{sellosBytes, &sellos},
	} {
		if err := json.Unmarshal(destino.valor, destino.en); err != nil {
			t.Fatalf("entrada JSON SQL inválida: %v", err)
		}
	}
	motivoCanonico, err := domain.RepresentacionCanonicaMotivoAutorizacionV2(
		motivo.Referencia,
	)
	if err != nil || !bytes.Equal(motivoCanonico, motivoBytes) {
		t.Fatalf("motivo SQL no canónico: %v", err)
	}
	actor, err := domain.RehidratarContextoActorVinculadoV2(contextoBytes)
	if err != nil {
		t.Fatalf("rehidratar contexto SQL: %v", err)
	}
	resultado := domain.ResultadoContextoActorRegistradoV2{
		RegistroContextoRef: plantilla.VinculoAutenticacionActor.RegistroContextoRef,
		Contexto:            actor, RepresentacionCanonica: contextoBytes,
		HuellaSHA256:                      plantilla.VinculoAutenticacionActor.ContextoActorHuellaSHA256,
		ManifiestoProcedenciaCanonico:     manifiestoBytes,
		ManifiestoProcedenciaHuellaSHA256: e.ManifiestoHuellaSHA256,
		AutoridadEfectiva: domain.AutoridadProcedenciaContextoActorV1(
			e.AutoridadEfectiva,
		),
		ResueltoEnAutoritativo: resueltoEn,
	}
	if err := resultado.Validar(); err != nil {
		t.Fatalf("resultado de contexto SQL inválido: %v", err)
	}
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: actor.Instantanea.CuentaRef,
		Metodo:    actor.Principal.AuthMethod, Garantia: actor.Principal.AuthAssurance,
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV2(
		context.Background(),
		revalidadorConfianzaAtestacionV3Prueba{
			resultado: plantilla.VinculoAutenticacionActor.Autenticacion(),
		},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: plantilla.VinculoAutenticacionActor.AutenticacionRef,
			SesionRef:        plantilla.VinculoAutenticacionActor.SesionRef,
		},
		resolutorConfianzaAtestacionV3Prueba{resultado: resultado},
		domain.SolicitudContextoActor{
			Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef,
		},
		&relojConfianzaAtestacionV3Prueba{ahora: ahora},
	)
	if err != nil {
		t.Fatalf("crear vínculo desde autoridades SQL: %v", err)
	}
	correlacion, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generadorCorrelacionConfianzaAtestacionV3Prueba{
			valor: plantilla.CorrelacionRef,
		},
	)
	if err != nil {
		t.Fatalf("crear correlación nominal: %v", err)
	}
	recurso := domain.RecursoAutorizable{
		Referencia: plantilla.RecursoRef,
		ModuloID:   plantilla.ModuloID, Tipo: plantilla.TipoRecurso,
		Ambitos: map[string]string{
			"organizacion_ref": alta.OrganizacionRef,
			"centro_ref":       alta.Solicitud.CentroRef,
			"categoria_ref":    alta.Solicitud.CategoriaRef,
		},
		Atributos: map[string]string{
			"efecto_huella_sha256": e.EfectoHuellaSHA256,
			"flujo_ref":            alta.Flujo.DefinicionRef,
			"flujo_version":        strconv.FormatUint(alta.Flujo.Version, 10),
			"flujo_huella_sha256":  alta.Flujo.HuellaSHA256,
			puertoscontratacion.AtributoHuellaPeticionHMACActiva: sellos.Activo.HuellaHMAC,
		},
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(
		domain.DatosSolicitudAutorizacionLigadaV3{
			VinculoAutenticacionActor: vinculo,
			ReferenciaMotivo:          motivo.Referencia,
			Accion:                    plantilla.Accion, Recurso: recurso,
			Finalidad: plantilla.Finalidad, Correlacion: correlacion,
		},
	)
	if err != nil {
		t.Fatalf("crear solicitud nominal: %v", err)
	}
	instantanea, rol, control, asignacion := instantaneaVectorO205(t, e, ahora, actor)
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, plantilla.DecisionRef,
		ahora, ahora.Add(90*time.Second),
	)
	if err != nil {
		t.Fatalf("evaluar autorización nominal: %v", err)
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatalf("crear decisión nominal: %v", err)
	}
	return emitirBundleVectorO205(
		t, e, ahora, claveDesde, claveHasta, solicitud, decision,
		motivo.Referencia, resultado, rol, control, asignacion,
		altaBytes, sellosBytes,
	)
}

func instantaneaVectorO205(
	t *testing.T,
	e entradaVectorSQLO205,
	ahora time.Time,
	actor domain.ContextoActor,
) (domain.InstantaneaAutorizacion, domain.VersionRol, domain.ControlVigenciaVersionRol, domain.AsignacionPerfil) {
	t.Helper()
	id := strings.ReplaceAll(e.Caso, "-", "_")
	rol := domain.VersionRol{
		RolID: "o205_go_" + id, Version: 1, Nombre: "Vector Go O2-05",
		Estado: domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{
			Accion:           puertoscontratacion.AccionCrearSolicitud,
			ModuloID:         puertoscontratacion.ModuloContratacion,
			TipoRecurso:      puertoscontratacion.TipoRecursoExpediente,
			Finalidades:      []string{puertoscontratacion.FinalidadCrearSolicitud},
			GarantiaMinima:   domain.AuthAssuranceSubstantial,
			CamposPermitidos: []string{"estado"},
			Obligaciones:     []string{"auditar"},
		}},
		PublicadaPor: "autoridad-o205-go", PublicadaEn: ahora.Add(-10 * time.Minute),
	}
	control := domain.ControlVigenciaVersionRol{
		VersionRolRef: rol.Referencia(), Revision: 1,
		Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
		ActualizadoPor: rol.PublicadaPor, ActualizadoEn: ahora.Add(-5 * time.Minute),
	}
	asignacion := domain.AsignacionPerfil{
		AsignacionID: e.AsignacionID, Version: e.AsignacionVersion,
		PerfilActivoRef: actor.PerfilActivoRef, PrincipalID: actor.Principal.ID,
		VersionRolRef: rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
		Ambitos: []domain.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{"organizacion:dipgra"}},
			{Clave: "centro_ref", Valores: []string{"centro:seleccion"}},
			{Clave: "categoria_ref", Valores: []string{"categoria:auxiliar"}},
		},
		EmitidaPor: "autoridad-o205-go", EmitidaEn: ahora.Add(-10 * time.Minute),
		VigenteDesde: ahora.Add(-5 * time.Minute), VigenteHasta: ahora.Add(30 * time.Minute),
	}
	instantanea := domain.InstantaneaAutorizacion{
		AsignacionPerfil: asignacion, VersionRol: rol,
		ControlVigenciaVersionRol: control, Politicas: e.Politicas,
		RevisionCatalogoPoliticas:     e.RevisionCatalogo,
		CatalogoPoliticasHuellaSHA256: e.HuellaCatalogoSHA256,
	}
	if err := instantanea.Validar(); err != nil {
		t.Fatalf("instantánea SQL/Go inválida: %v", err)
	}
	return instantanea, rol, control, asignacion
}

func emitirBundleVectorO205(
	t *testing.T,
	e entradaVectorSQLO205,
	ahora, claveDesde, claveHasta time.Time,
	solicitud domain.SolicitudAutorizacionLigadaV3,
	decision domain.DecisionAutorizacionLigadaV3,
	motivo domain.ReferenciaEntradaCatalogo,
	resultado domain.ResultadoContextoActorRegistradoV2,
	rol domain.VersionRol,
	control domain.ControlVigenciaVersionRol,
	asignacion domain.AsignacionPerfil,
	alta, sellos []byte,
) bundleVectorO205 {
	t.Helper()
	publica, privada, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		e.RaizClaveID, e.RaizVersion, publica, e.AudienciaDespliegue,
		EstadoClaveAtestacionAutorizacionV3Activa,
		ahora.Add(-10*time.Minute), ahora.Add(time.Hour), time.Time{},
	)
	if err != nil {
		t.Fatalf("crear raíz efímera: %v", err)
	}
	configuracion, err := NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		e.RevisionConfianza, e.SecuenciaConfianza,
		ahora.Add(-5*time.Minute), ahora.Add(30*time.Minute), raiz,
	)
	if err != nil {
		t.Fatalf("crear configuración efímera: %v", err)
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          SuiteAtestacionAutorizacionV3COSEEdDSA,
		ClaveID:        e.RaizClaveID, Audiencia: e.AudienciaDespliegue,
	}
	atestacion := atestacionConfianzaAtestacionV3Prueba(
		t, cabecera, decision, motivo, resultado, privada, ahora,
	)
	servicio, err := NuevoServicioConfianzaAtestacionAutorizacionV3(
		configuracion, &relojConfianzaAtestacionV3Prueba{ahora: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	prueba, err := servicio.Verificar(
		context.Background(), solicitud, decision, motivo, resultado, atestacion,
	)
	if err != nil {
		t.Fatalf("verificar VEC-AD-3 real: %v", err)
	}
	materialHMAC := exigirBase64O205(t, e.ClaveHMACB64)
	defer borrarBytesConfianzaAtestacion(materialHMAC)
	clave, err := NuevaClaveHMACCapacidadAtestacionAutorizacionV3(
		e.ClaveID, e.ClaveVersion, materialHMAC, e.EmisorID,
		e.AudienciaConsumo, EstadoClaveHMACCapacidadAtestacionV3Emision,
		claveDesde, claveHasta, time.Time{}, e.RevisionGobierno,
		e.HuellaGobiernoSHA256,
	)
	if err != nil {
		t.Fatalf("crear clave HMAC efímera: %v", err)
	}
	emisor, err := NuevoEmisorCapacidadesAtestacionAutorizacionV3(
		clave, &relojConfianzaAtestacionV3Prueba{ahora: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := emisor.Emitir(
		context.Background(), solicitud, decision, motivo, resultado,
		atestacion, prueba,
	)
	if err != nil {
		t.Fatalf("Emitir capacidad productiva: %v", err)
	}
	capacidadBytes, _ := capacidad.ExportacionCanonicaParaConsumidor()
	decisionBytes, _ := domain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	motivoBytes, _ := domain.RepresentacionCanonicaMotivoAutorizacionV2(motivo)
	solicitudFirma, _ := ports.NuevaSolicitudFirmaAtestacionAutorizacionV3(
		cabecera, decision, motivo, resultado,
	)
	payload, _ := solicitudFirma.Mensaje()
	resultadoFirma, _ := atestacion.Resultado()
	cose, _ := resultadoFirma.Firma()
	evidencia, _ := prueba.ExportacionCanonicaParaConsumidor()
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	return bundleVectorO205{
		CapacidadB64:   base64.StdEncoding.EncodeToString(capacidadBytes),
		DecisionB64:    base64.StdEncoding.EncodeToString(decisionBytes),
		MotivoB64:      base64.StdEncoding.EncodeToString(motivoBytes),
		ContextoB64:    e.ContextoB64,
		PayloadB64:     base64.StdEncoding.EncodeToString(payload),
		COSEB64:        base64.StdEncoding.EncodeToString(cose),
		EvidenciaB64:   base64.StdEncoding.EncodeToString(evidencia),
		SPKIB64:        base64.StdEncoding.EncodeToString(spki),
		AltaB64:        base64.StdEncoding.EncodeToString(alta),
		SellosB64:      base64.StdEncoding.EncodeToString(sellos),
		PersonaVersion: e.PersonaVersion, PerfilVersion: e.PerfilVersion,
		VersionRolDocumento: exigirJSONO205(t, rol),
		ControlRolDocumento: exigirJSONO205(t, control),
		AsignacionDocumento: exigirJSONO205(t, asignacion),
	}
}

func parsearInstanteEntradaSQLO205(valor string) (time.Time, error) {
	const formato = "2006-01-02T15:04:05.000000Z"
	instante, err := time.Parse(formato, valor)
	if err != nil || instante.Format(formato) != valor ||
		!instanteCanonicoConfianza(instante) {
		return time.Time{}, errors.New("instante SQL O2-05 inválido")
	}
	return instante, nil
}

func exigirInstanteSQLO205(t *testing.T, valor string) time.Time {
	t.Helper()
	instante, err := parsearInstanteEntradaSQLO205(valor)
	if err != nil {
		t.Fatal(err)
	}
	return instante
}

func exigirBase64O205(t *testing.T, valor string) []byte {
	t.Helper()
	contenido, err := base64.StdEncoding.DecodeString(valor)
	if err != nil {
		t.Fatalf("base64 SQL inválido: %v", err)
	}
	return contenido
}

func exigirJSONO205(t *testing.T, valor any) json.RawMessage {
	t.Helper()
	contenido, err := json.Marshal(valor)
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}
