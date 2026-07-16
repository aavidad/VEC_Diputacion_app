package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type huellaArtefactoFixtureV3 struct {
	SHA256 string `json:"sha256"`
	Tamano int    `json:"tamano"`
}

type fixtureDoradoManifiestoV3 struct {
	Esquema        string                           `json:"esquema_fixture"`
	Manifiesto     manifiestoProbatorioPostgreSQLV3 `json:"manifiesto"`
	Contenido      huellaArtefactoFixtureV3         `json:"contenido_canonico"`
	Representacion huellaArtefactoFixtureV3         `json:"representacion_canonica"`
	Preimagen      huellaArtefactoFixtureV3         `json:"preimagen_hmac"`
}

func TestFixtureDoradoManifiestoV3EsValidoYByteExactoEnGo(t *testing.T) {
	documentoFixture, err := os.ReadFile("../../testdata/manifiesto_probatorio_v3_dorado.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture fixtureDoradoManifiestoV3
	if err = decodificarJSONEstricto(documentoFixture, &fixture); err != nil ||
		fixture.Esquema != "vec.pruebas.bolsa.manifiesto-probatorio-v3-dorado.v1" {
		t.Fatalf("fixture dorado V3 invalido: %v", err)
	}
	manifiesto, err := fixture.Manifiesto.dominio()
	if err != nil {
		t.Fatalf("el fixture no es un manifiesto de dominio valido: %v", err)
	}
	if err = manifiesto.Validar(); err != nil {
		t.Fatalf("el fixture no supera Manifiesto.Validar: %v", err)
	}
	documento, contenido, representacion, preimagen, err :=
		serializarManifiestoProbatorioV3(&manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytesPostgreSQL(documento, contenido, representacion, preimagen)
	documentoEsperado, err := json.Marshal(fixture.Manifiesto)
	if err != nil || !bytes.Equal(documento, documentoEsperado) {
		t.Fatalf("el DTO Go diverge del fixture compartido: %v", err)
	}
	for _, artefacto := range []struct {
		nombre   string
		valor    []byte
		esperado huellaArtefactoFixtureV3
	}{
		{"contenido", contenido, fixture.Contenido},
		{"representacion", representacion, fixture.Representacion},
		{"preimagen", preimagen, fixture.Preimagen},
	} {
		suma := sha256.Sum256(artefacto.valor)
		if len(artefacto.valor) != artefacto.esperado.Tamano ||
			hex.EncodeToString(suma[:]) != artefacto.esperado.SHA256 {
			t.Fatalf("artefacto dorado %s divergente", artefacto.nombre)
		}
	}
}

type verificadorManifiestoPostgreSQLPrueba struct {
	error        error
	cancelar     context.CancelFunc
	invocaciones int
}

func (v *verificadorManifiestoPostgreSQLPrueba) VerificarSelloBaremacion(
	_ context.Context, _ puertosbolsa.SolicitudVerificarSelloBaremacion,
) error {
	v.invocaciones++
	if v.cancelar != nil {
		v.cancelar()
	}
	return v.error
}

func TestSerializacionManifiestoPostgreSQLV3ConservaArtefactosExactos(t *testing.T) {
	manifiesto := manifiestoPostgreSQLV3Prueba(t)
	documento, contenido, representacion, preimagen, err := serializarManifiestoProbatorioV3(&manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytesPostgreSQL(documento, contenido, representacion, preimagen)
	var fila manifiestoProbatorioPostgreSQLV3
	if err = decodificarJSONEstricto(documento, &fila); err != nil {
		t.Fatal(err)
	}
	recuperado, err := fila.dominio()
	if err != nil || recuperado.HuellaManifiestoSHA256 != manifiesto.HuellaManifiestoSHA256 ||
		recuperado.SelloManifiestoHMACSHA256 != manifiesto.SelloManifiestoHMACSHA256 {
		t.Fatalf("roundtrip de manifiesto divergente: %v", err)
	}
	artefactos, err := puertosbolsa.ArtefactosCanonicosManifiestoProbatorioBaremacion(manifiesto)
	if err != nil || !bytes.Equal(contenido, artefactos.ContenidoSinHuella.Revelar()) ||
		!bytes.Equal(representacion, artefactos.RepresentacionSellada.Revelar()) ||
		!bytes.Equal(preimagen, artefactos.PreimagenHMAC.Revelar()) {
		t.Fatalf("artefactos persistidos divergentes: %v", err)
	}
	if bytes.Contains(documento, []byte("VersionEsquema")) ||
		!bytes.Contains(documento, []byte(`"version_esquema":3`)) {
		t.Fatalf("DTO no usa contrato snake_case V3: %s", documento)
	}
}

func TestArchivoPostgreSQLV3RechazaCorrupcionYCamposDesconocidos(t *testing.T) {
	manifiesto := manifiestoPostgreSQLV3Prueba(t)
	fila, err := manifiestoPostgreSQLDesdeDominio(manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	_, contenido, representacion, preimagen, err := serializarManifiestoProbatorioV3(&manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytesPostgreSQL(contenido, representacion, preimagen)
	archivado := manifiestoArchivadoDecodificadoV3{
		Manifiesto: manifiesto, Contenido: contenido,
		Representacion: representacion, PreimagenHMAC: preimagen,
	}
	archivo := archivoProbatorioPostgreSQLV3{
		Esquema:             esquemaArchivoProbatorioPostgreSQLV3,
		BaremacionMeritoRef: manifiesto.BaremacionMeritoRef, NumeroVersion: "2",
		Manifiestos: []manifiestoArchivadoPostgreSQLV3{{
			Manifiesto: fila, ContenidoManifiestoCanonicoHex: hex.EncodeToString(contenido),
			RepresentacionManifiestoCanonicaHex: hex.EncodeToString(representacion),
			PreimagenHMACManifiestoHex:          hex.EncodeToString(preimagen),
		}},
	}
	archivo.HuellaArchivoSHA256 = huellaArchivoProbatorioV3(
		archivo.Esquema, archivo.BaremacionMeritoRef, archivo.NumeroVersion,
		[]manifiestoArchivadoDecodificadoV3{archivado},
	)
	documento, err := json.Marshal(archivo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = decodificarArchivoProbatorioV3(documento); err != nil {
		t.Fatalf("archivo V3 exacto rechazado: %v", err)
	}

	corrupto := archivo
	corrupto.Manifiestos = append([]manifiestoArchivadoPostgreSQLV3(nil), archivo.Manifiestos...)
	prefijoCorrupto := "00"
	if corrupto.Manifiestos[0].PreimagenHMACManifiestoHex[:2] == prefijoCorrupto {
		prefijoCorrupto = "01"
	}
	corrupto.Manifiestos[0].PreimagenHMACManifiestoHex = prefijoCorrupto +
		corrupto.Manifiestos[0].PreimagenHMACManifiestoHex[2:]
	documentoCorrupto, _ := json.Marshal(corrupto)
	if _, err = decodificarArchivoProbatorioV3(documentoCorrupto); !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
		t.Fatalf("corrupcion binaria admitida: %v", err)
	}
	documentoAjeno := append(documento[:len(documento)-1], []byte(`,"campo_ajeno":true}`)...)
	if _, err = decodificarArchivoProbatorioV3(documentoAjeno); !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
		t.Fatalf("campo desconocido admitido: %v", err)
	}
}

func TestVerificacionManifiestoPostgreSQLV3FallaAntesDePersistirYCancela(t *testing.T) {
	manifiesto := manifiestoPostgreSQLV3Prueba(t)
	for _, caso := range []struct {
		nombre   string
		motivo   error
		esperado error
	}{
		{"clave revocada", errors.New("clave revocada"), puertosbolsa.ErrSelloBaremacionNoAutentico},
		{"KMS indisponible", puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible, puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible},
	} {
		verificador := &verificadorManifiestoPostgreSQLPrueba{error: caso.motivo}
		repositorio := &RepositorioBaremaciones{verificador: verificador}
		if err := repositorio.verificarSelloManifiestoProbatorioV3(context.Background(), &manifiesto); !errors.Is(err, caso.esperado) || verificador.invocaciones != 1 {
			t.Fatalf("%s no fallo cerrado: %v", caso.nombre, err)
		}
	}

	ctx, cancelar := context.WithCancel(context.Background())
	verificador := &verificadorManifiestoPostgreSQLPrueba{cancelar: cancelar}
	repositorio := &RepositorioBaremaciones{verificador: verificador}
	if err := repositorio.verificarSelloManifiestoProbatorioV3(ctx, &manifiesto); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion del conector ocultada: %v", err)
	}
}

func TestVerificacionHistoricaPostgreSQLV3ConservaIndisponibilidadKMS(t *testing.T) {
	verificador := &verificadorManifiestoPostgreSQLPrueba{
		error: puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible,
	}
	repositorio := &RepositorioBaremaciones{verificador: verificador}
	representacion, err := puertosbolsa.NuevaCargaProtegida(
		[]byte("representacion-historica-postgresql-v3"),
	)
	if err != nil {
		t.Fatal(err)
	}
	peticion := puertosbolsa.SolicitudVerificarSelloBaremacion{
		Finalidad:              puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3,
		RepresentacionCanonica: representacion,
		SelloHMAC:              "hmac-sha256:historico_postgresql_v3:" + strings.Repeat("a", 64),
	}
	err = repositorio.verificarSelloHistoricoArchivoV3(context.Background(), peticion)
	if !errors.Is(err, puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible) ||
		verificador.invocaciones != 1 {
		t.Fatalf("indisponibilidad historica confundida con corrupcion: %v", err)
	}
}

func TestArchivoPostgreSQLV3VacioLigaVersionUnoYDecimalCanonico(t *testing.T) {
	baremacion := baremacionPostgreSQLPrueba(t)
	huella, err := baremacion.HuellaEstadoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	version := puertosbolsa.VersionBaremacion{
		Referencia: puertosbolsa.ReferenciaVersionBaremacion{
			BaremacionMeritoRef: baremacion.ID, Numero: 1, HuellaEstadoSHA256: huella,
		},
		Agregado: baremacion, ConfirmadaEn: instantePostgreSQLPrueba,
	}
	archivo := archivoProbatorioPostgreSQLV3{
		Esquema:             esquemaArchivoProbatorioPostgreSQLV3,
		BaremacionMeritoRef: baremacion.ID, NumeroVersion: "1",
		Manifiestos: []manifiestoArchivadoPostgreSQLV3{},
	}
	archivo.HuellaArchivoSHA256 = huellaArchivoProbatorioV3(
		archivo.Esquema, archivo.BaremacionMeritoRef, archivo.NumeroVersion, nil,
	)
	documento, err := json.Marshal(archivo)
	if err != nil {
		t.Fatal(err)
	}
	repositorio := &RepositorioBaremaciones{verificador: verificadorPostgreSQLBaremacionPrueba{}}
	manifiestos, partes, err := repositorio.verificarArchivoProbatorioV3(
		context.Background(), version, documento,
	)
	if err != nil || len(manifiestos) != 0 || len(partes) != 1 || partes[0] != "0" {
		t.Fatalf("archivo vacio V3 no verificable: manifiestos=%d partes=%v err=%v", len(manifiestos), partes, err)
	}

	archivo.NumeroVersion = "01"
	archivo.HuellaArchivoSHA256 = huellaArchivoProbatorioV3(
		archivo.Esquema, archivo.BaremacionMeritoRef, archivo.NumeroVersion, nil,
	)
	documento, _ = json.Marshal(archivo)
	if _, _, err = repositorio.verificarArchivoProbatorioV3(context.Background(), version, documento); !errors.Is(err, puertosbolsa.ErrEvidenciaBaremacionNoConfiable) {
		t.Fatalf("decimal no canonico admitido: %v", err)
	}
}

func manifiestoPostgreSQLV3Prueba(t *testing.T) puertosbolsa.ManifiestoProbatorioBaremacion {
	t.Helper()
	const baremacionRef = "baremacion:postgresql:manifiesto-v3"
	const procesoRef = "proceso:postgresql:manifiesto-v3"
	const decisionRef = "decision:postgresql:manifiesto-v3"
	huella := func(valor string) string { return strings.Repeat(valor, 64) }
	acciones := []puertosbolsa.AccionOperacionBaremacion{
		puertosbolsa.AccionConsultarBaremacionVigente,
		puertosbolsa.AccionRecuperarCalculoBaremacion,
		puertosbolsa.AccionConsultarCriterioBaremacion,
		puertosbolsa.AccionConsultarEvidenciaBaremacion,
		puertosbolsa.AccionConsultarRepresentacionBaremacion,
		puertosbolsa.AccionAdoptarDecisionInicialBaremacion,
		puertosbolsa.AccionConsultarPoliticaFirmaBaremacion,
		puertosbolsa.AccionCodificarDecisionBaremacion,
		puertosbolsa.AccionCustodiarDecisionBaremacion,
		puertosbolsa.AccionPrepararFirmaDecisionBaremacion,
		puertosbolsa.AccionConsultarFirmaDecisionBaremacion,
		puertosbolsa.AccionValidarFirmaDecisionBaremacion,
		puertosbolsa.AccionRecuperarBinarioFirmadoBaremacion,
		puertosbolsa.AccionCustodiarDocumentoFirmadoBaremacion,
		puertosbolsa.AccionRetenerDocumentoFirmadoBaremacion,
		puertosbolsa.AccionReservarDecisionBaremacion,
		puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion,
		puertosbolsa.AccionConfirmarDecisionBaremacion,
	}
	recursos := []string{
		baremacionRef, "calculo:manifiesto-v3", procesoRef,
		"documento:manifiesto-v3", "representacion:manifiesto-v3", baremacionRef,
		"politica:manifiesto-v3", decisionRef, decisionRef, decisionRef,
		"sesion:manifiesto-v3", "firma:manifiesto-v3",
		"documento-firmado:manifiesto-v3", "documento-firmado:manifiesto-v3",
		"documento-firmado:manifiesto-v3", baremacionRef, baremacionRef, baremacionRef,
	}
	autorizaciones := make([]puertosbolsa.AutorizacionProbatoriaBaremacion, len(acciones))
	for indice, accion := range acciones {
		clase, existe := puertosbolsa.ClaseRecursoRequeridaOperacionBaremacion(accion)
		if !existe {
			t.Fatalf("accion sin clase: %s", accion)
		}
		autorizaciones[indice] = puertosbolsa.AutorizacionProbatoriaBaremacion{
			Secuencia: uint32(indice + 1), Accion: accion, ClaseRecurso: clase,
			RecursoRef: recursos[indice], AutorizacionRef: "autorizacion:manifiesto-v3:" + strconvItoaPrueba(indice+1),
		}
	}
	tipos := []puertosbolsa.TipoEvidenciaProbatoriaBaremacion{
		puertosbolsa.EvidenciaEstadoBaseBaremacion, puertosbolsa.EvidenciaCalculoOficialBaremacion,
		puertosbolsa.EvidenciaCriterioPublicadoBaremacion, puertosbolsa.EvidenciaDocumentoMeritoBaremacion,
		puertosbolsa.EvidenciaRepresentacionBaremacion, puertosbolsa.EvidenciaContenidoDecisionBaremacion,
		puertosbolsa.EvidenciaPoliticaFirmaBaremacion, puertosbolsa.EvidenciaDocumentoCanonicoBaremacion,
		puertosbolsa.EvidenciaCustodiaFirmableBaremacion, puertosbolsa.EvidenciaPreparacionFirmaBaremacion,
		puertosbolsa.EvidenciaConsultaFirmaBaremacion, puertosbolsa.EvidenciaValidacionInicialBaremacion,
		puertosbolsa.EvidenciaValidacionFinalBaremacion, puertosbolsa.EvidenciaRecuperacionFirmadoBaremacion,
		puertosbolsa.EvidenciaCustodiaFirmadoBaremacion, puertosbolsa.EvidenciaRetencionFirmadoBaremacion,
	}
	referencias := []string{
		baremacionRef, "evidencia-calculo:v3", procesoRef, "documento:manifiesto-v3",
		"representacion:manifiesto-v3", decisionRef, "aprobacion-politica:v3", decisionRef,
		"custodia-firmable:v3", "preparacion-firma:v3", "consulta-firma:v3",
		"validacion-firma:v3", "validacion-firma:v3", "recuperacion-firmado:v3",
		"custodia-firmado:v3", "retencion-firmado:v3",
	}
	evidencias := make([]puertosbolsa.EvidenciaProbatoriaBaremacion, len(tipos))
	for indice, tipo := range tipos {
		huellaEvidencia := huella("a")
		if indice == 0 {
			huellaEvidencia = huella("1")
		}
		if indice == 3 || indice == 4 {
			huellaEvidencia = huella("2")
		}
		if indice == 7 || indice == 8 {
			huellaEvidencia = huella("3")
		}
		if indice == 11 || indice == 12 {
			huellaEvidencia = huella("4")
		}
		if indice == 14 || indice == 15 {
			huellaEvidencia = huella("5")
		}
		evidencias[indice] = puertosbolsa.EvidenciaProbatoriaBaremacion{
			Secuencia: uint32(indice + 1), Tipo: tipo,
			Referencia: referencias[indice], HuellaEvidenciaSHA256: huellaEvidencia,
		}
	}
	base := puertosbolsa.ManifiestoProbatorioBaremacion{
		Esquema:        puertosbolsa.EsquemaManifiestoProbatorioBaremacion,
		Finalidad:      puertosbolsa.FinalidadManifiestoProbatorioBaremacion,
		VersionEsquema: puertosbolsa.VersionManifiestoProbatorioBaremacion,
		Referencia:     "manifiesto:postgresql:v3", ProcesoRef: procesoRef,
		SolicitudRef: "solicitud:postgresql:v3", SujetoRef: "sujeto:postgresql:v3",
		BaremacionMeritoRef: baremacionRef, DecisionRef: decisionRef,
		VersionBase: 1, HuellaVersionBaseSHA256: huella("1"),
		Autorizaciones: autorizaciones, Evidencias: evidencias,
		CreadoEn: time.Date(2026, time.July, 16, 9, 0, 0, 0, time.UTC),
	}
	preparado, _, err := base.PrepararSellado()
	if err != nil {
		t.Fatalf("preparar manifiesto estructural V3: %v", err)
	}
	resultado, err := preparado.IncorporarSello("hmac-sha256:manifiesto_v3:" + huella("f"))
	if err != nil {
		t.Fatalf("incorporar sello estructural V3: %v", err)
	}
	return resultado
}

func strconvItoaPrueba(valor int) string {
	const digitos = "0123456789"
	if valor < 10 {
		return string(digitos[valor])
	}
	return string([]byte{digitos[valor/10], digitos[valor%10]})
}
