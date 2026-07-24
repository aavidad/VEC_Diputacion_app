package gobiernoconvocatorias

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

const directorioVectoresKMSBorradorV1 = "testdata/vectores_kms_confirmacion_borrador_v1"

// manifiestoVectoresKMSBorradorV1 es el ancla compartida por los tests Go y
// PostgreSQL. Las migraciones de producción no leen este fixture. Cambiar un
// valor exige regenerar y revisar deliberadamente el contrato criptográfico.
type manifiestoVectoresKMSBorradorV1 struct {
	Esquema                              string `json:"esquema"`
	HuellaVersionSHA256                  string `json:"huella_version_sha256"`
	HuellaMaterialSHA256                 string `json:"huella_material_sha256"`
	HuellaDecisionPoliticaSHA256         string `json:"huella_decision_politica_sha256"`
	HuellaEvidenciaPerfilSHA256          string `json:"huella_evidencia_perfil_sha256"`
	HuellaAADSHA256                      string `json:"huella_aad_sha256"`
	HuellaEnvolturaSHA256                string `json:"huella_envoltura_sha256"`
	HuellaSobreSHA256                    string `json:"huella_sobre_sha256"`
	HuellaPreimagenAtestacionSHA256      string `json:"huella_preimagen_atestacion_sha256"`
	FirmaAtestacionBase64URLSinRelleno   string `json:"firma_atestacion_base64url_sin_relleno"`
	HuellaFirmaAtestacionSHA256          string `json:"huella_firma_atestacion_sha256"`
	HuellaCuerpoReciboSHA256             string `json:"huella_cuerpo_recibo_sha256"`
	HuellaPreimagenRevalidacionSHA256    string `json:"huella_preimagen_revalidacion_sha256"`
	FirmaRevalidacionBase64URLSinRelleno string `json:"firma_revalidacion_base64url_sin_relleno"`
	HuellaFirmaRevalidacionSHA256        string `json:"huella_firma_revalidacion_sha256"`
	HuellaAcreditacionSHA256             string `json:"huella_acreditacion_sha256"`
}

func leerVectorKMSBorradorV1(t *testing.T, nombre string) []byte {
	t.Helper()
	contenido, err := os.ReadFile(filepath.Join(directorioVectoresKMSBorradorV1, nombre))
	if err != nil {
		t.Fatal(err)
	}
	if len(contenido) < 2 || contenido[len(contenido)-1] != '\n' ||
		contenido[len(contenido)-2] == '\n' || bytes.ContainsRune(contenido, '\r') {
		t.Fatalf("fixture %s debe tener exactamente un salto LF terminal", nombre)
	}
	return append([]byte(nil), contenido[:len(contenido)-1]...)
}

func cargarManifiestoVectoresKMSBorradorV1(t *testing.T) manifiestoVectoresKMSBorradorV1 {
	t.Helper()
	contenido := leerVectorKMSBorradorV1(t, "manifest.json")
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var manifiesto manifiestoVectoresKMSBorradorV1
	if err := decodificador.Decode(&manifiesto); err != nil {
		t.Fatal(err)
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		t.Fatalf("manifesto de vectores KMS contiene datos sobrantes: %v", err)
	}
	if manifiesto.Esquema != "vec.pruebas.bolsa.vectores-kms-confirmacion-borrador.v1" {
		t.Fatalf("esquema de fixture KMS no reconocido: %q", manifiesto.Esquema)
	}
	return manifiesto
}

func TestVectoresDoradosCriptograficosBorradores(t *testing.T) {
	manifiesto := cargarManifiestoVectoresKMSBorradorV1(t)
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	recibo, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion := *e.confirmador.ultima
	aad, err := confirmacion.Cifrado.AAD.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	huellaAAD, err := confirmacion.Cifrado.AAD.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaVersion, err := confirmacion.Version.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaMaterial, err := confirmacion.Material.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	atestacion, solicitud, revalidacion, err := recibo.EvidenciasKMSParaVerificacion()
	if err != nil {
		t.Fatal(err)
	}
	if !solicitud.SolicitadaEn.Before(recibo.ConfirmadaEn) {
		t.Fatal("el vector confundió la preparación real con el not-before de confirmación")
	}
	preimagenAtestacion, _, _, _, firmaAtestacion, err := atestacion.DatosParaVerificacionFirma()
	if err != nil {
		t.Fatal(err)
	}
	preimagenRevalidacion, _, _, _, firmaRevalidacion, err :=
		revalidacion.DatosParaVerificacionFirma(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	huellaPreimagenAtestacion := sha256.Sum256(preimagenAtestacion)
	huellaFirmaAtestacion := sha256.Sum256(firmaAtestacion)
	huellaPreimagenRevalidacion := sha256.Sum256(preimagenRevalidacion)
	huellaFirmaRevalidacion := sha256.Sum256(firmaRevalidacion)

	vectores := []struct {
		nombre, obtenido, esperado string
	}{
		{"AAD canónica", string(aad), string(leerVectorKMSBorradorV1(t, "aad_canonica.json"))},
		{"huella versión", huellaVersion, manifiesto.HuellaVersionSHA256},
		{"huella material", huellaMaterial, manifiesto.HuellaMaterialSHA256},
		{"huella política", confirmacion.ResolucionPerfilCifrado.Evidencia.HuellaDecisionPoliticaSHA256, manifiesto.HuellaDecisionPoliticaSHA256},
		{"huella evidencia perfil", confirmacion.ResolucionPerfilCifrado.Evidencia.HuellaEvidenciaSHA256, manifiesto.HuellaEvidenciaPerfilSHA256},
		{"huella AAD", huellaAAD, manifiesto.HuellaAADSHA256},
		{"huella envoltura", atestacion.HuellaEnvolturaSHA256, manifiesto.HuellaEnvolturaSHA256},
		{"huella sobre", atestacion.HuellaSobreSHA256, manifiesto.HuellaSobreSHA256},
		{"preimagen atestación KMS", string(preimagenAtestacion), string(leerVectorKMSBorradorV1(t, "preimagen_atestacion.json"))},
		{"huella preimagen atestación", hex.EncodeToString(huellaPreimagenAtestacion[:]), manifiesto.HuellaPreimagenAtestacionSHA256},
		{"firma atestación KMS", base64.RawURLEncoding.EncodeToString(firmaAtestacion), manifiesto.FirmaAtestacionBase64URLSinRelleno},
		{"huella firma atestación", hex.EncodeToString(huellaFirmaAtestacion[:]), manifiesto.HuellaFirmaAtestacionSHA256},
		{"huella cuerpo recibo", huellaCuerpoReciboBorrador(recibo), manifiesto.HuellaCuerpoReciboSHA256},
		{"preimagen revalidación KMS", string(preimagenRevalidacion), string(leerVectorKMSBorradorV1(t, "preimagen_revalidacion.json"))},
		{"huella preimagen revalidación", hex.EncodeToString(huellaPreimagenRevalidacion[:]), manifiesto.HuellaPreimagenRevalidacionSHA256},
		{"firma revalidación KMS", base64.RawURLEncoding.EncodeToString(firmaRevalidacion), manifiesto.FirmaRevalidacionBase64URLSinRelleno},
		{"huella firma revalidación", hex.EncodeToString(huellaFirmaRevalidacion[:]), manifiesto.HuellaFirmaRevalidacionSHA256},
		{"huella acreditación", recibo.AcreditacionKMS.HuellaAcreditacionSHA256, manifiesto.HuellaAcreditacionSHA256},
	}
	for _, vector := range vectores {
		t.Run(vector.nombre, func(t *testing.T) {
			if vector.obtenido != vector.esperado {
				t.Fatalf("vector durable cambió\nobtenido: %q\nesperado: %q", vector.obtenido, vector.esperado)
			}
		})
	}
}
