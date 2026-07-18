package gobiernoconvocatorias

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

// Estos valores son parte del contrato durable. No se regeneran desde el
// código probado: cualquier cambio exige revisar y versionar deliberadamente
// el esquema criptográfico correspondiente.
const (
	aadCanonicaBorradorDorada      = "{\"esquema\":\"bolsa.convocatoria.borrador.aad.v1\",\"version_ref\":\"proceso:bolsa:auxiliar-2026-1#1\",\"version_revision\":1,\"huella_version_sha256\":\"c99741e60ea930a7ff55e0ff85fddee6a4d4b547ba84cb2711548d89b0171d50\",\"esquema_material\":\"bolsa.convocatoria.intencion.v2\",\"huella_material_sha256\":\"1d9559ac3a4587ec9e4c71c50008c8e8c60b7debae8a6b5de096fa63f9237190\",\"perfil_cifrado_ref\":\"perfil:cifrado:borradores:v1\",\"perfil_cifrado_version\":1,\"huella_perfil_cifrado_sha256\":\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\"algoritmo_aead\":\"A256GCM\",\"algoritmo_envoltura_clave\":\"A256KW\",\"evidencia_perfil_ref\":\"evidencia:resolucion-perfil-cifrado:001\",\"evidencia_perfil_version\":1,\"huella_evidencia_perfil_sha256\":\"d69bc655b54f22213a2d03811fd92c07da2296c06ad6d534b2d8186cb959e936\",\"decision_politica_ref\":\"decision:politica-cifrado:borradores:001\",\"decision_politica_version\":1,\"huella_decision_politica_sha256\":\"20ca8cc2c31dc1c9e3a8bfa9063109eab1507aa7578a121d4f09b824a0441e8e\",\"localizador_esquema\":2,\"localizador_dominio\":\"localizador\",\"localizador_clave_ref\":\"clave:hmac:convocatorias:localizador:v3\",\"localizador_generacion\":3,\"localizador_hmac_sha256\":\"21a19f3f0f62d2b33d347cf9c6f64db62ef6d4c156ccae13aacab673095823fe\",\"huella_solicitud_esquema\":2,\"huella_solicitud_dominio\":\"huella_solicitud\",\"huella_solicitud_clave_ref\":\"clave:hmac:convocatorias:huella:v3\",\"huella_solicitud_generacion\":3,\"huella_solicitud_hmac_sha256\":\"f456f3f649430b0ce1348db1f3d96a27593c6282c0d562ab8d3620299b2c9927\",\"revision_diario\":1,\"cercado_diario\":1,\"arrendamiento_inicia_en\":\"2026-07-18T09:00:00.007Z\",\"arrendamiento_vence_en\":\"2026-07-18T09:02:00.007Z\",\"atestacion_sellado_ref\":\"atestacion:motivo:001\",\"atestacion_sellado_version\":1,\"huella_atestacion_sellado_sha256\":\"5555555555555555555555555555555555555555555555555555555555555555\",\"token_consumo_sellado_ref\":\"consumo:motivo:001\",\"huella_correlacion_sha256\":\"b7f3f6944cba8392fdc5245ffb3ea2def4fc6e7ca132ffd635e894b3c11eba93\",\"procedencia_esquema\":\"vec.acto.procedencia.v1\",\"perfil_ejecucion\":\"pruebas\",\"autoridad_acto\":\"autoritativo\",\"proveedor_procedencia_ref\":\"proveedor-pruebas\",\"migrable_produccion\":true}"
	preimagenAtestacionKMSDorada   = "{\"Esquema\":\"bolsa.convocatoria.borrador.atestacion-kms.v1\",\"AtestacionRef\":\"atestacion:kms:borrador:001\",\"Estado\":\"vigente\",\"PerfilRef\":\"perfil:cifrado:borradores:v1\",\"HuellaPerfil\":\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\"AlgoritmoAEAD\":\"A256GCM\",\"AlgoritmoEnvoltura\":\"A256KW\",\"ClaveRef\":\"clave:kms:borradores:v1\",\"HuellaAAD\":\"95bffe4d2422c748d85f103430d2e99ce3117445c362a05f83b1e35bef9ff24c\",\"HuellaEnvoltura\":\"b49d7d4067999aec587566fdd08d06c910e2037cfe0fd7b81874b4e954fb650d\",\"HuellaSobre\":\"328f371674c16b2a178f829d9fe01b19aebcff2b168b481b79c3b59a6fc1975c\",\"VerificadorRef\":\"verificador:kms-emisor-prueba:v1\",\"AlgoritmoFirma\":\"Ed25519\",\"HuellaClavePublica\":\"dae7b96f2766f6ec4a82cb806b5de86f8a80bb55b2525fb5899f025fc7ee1453\",\"ProcedenciaEsquema\":\"vec.acto.procedencia.v1\",\"PerfilEjecucion\":\"pruebas\",\"Autoridad\":\"autoritativo\",\"ProveedorProcedenciaRef\":\"proveedor-pruebas\",\"VersionAtestacion\":1,\"PerfilVersion\":1,\"VersionClave\":1,\"MigrableProduccion\":true,\"EmitidaEn\":\"2026-07-18T09:00:00.011Z\",\"ValidaHasta\":\"2026-07-18T09:04:00.011Z\"}"
	preimagenRevalidacionKMSDorada = "{\"Esquema\":\"bolsa.convocatoria.borrador.revalidacion-kms.v1\",\"AtestacionRef\":\"atestacion:kms:borrador:001\",\"Estado\":\"autorizada\",\"HuellaAAD\":\"95bffe4d2422c748d85f103430d2e99ce3117445c362a05f83b1e35bef9ff24c\",\"HuellaCuerpoRecibo\":\"899dffa8f86829470c203320ce37945f2750171732f2c53bda7263e698a7cfd8\",\"ComprobacionRef\":\"comprobacion:kms:persistencia:001\",\"HuellaComprobacion\":\"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee\",\"AlgoritmoFirma\":\"Ed25519\",\"VerificadorRef\":\"verificador:kms-revalidacion-prueba:v1\",\"HuellaClavePublica\":\"b48c7a6250118e1556efed4b0c837f67c4557d835fc0f24bab4d8c5bae50919c\",\"AlgoritmoAtestacion\":\"Ed25519\",\"VerificadorAtestacion\":\"verificador:kms-emisor-prueba:v1\",\"HuellaClaveAtestacion\":\"dae7b96f2766f6ec4a82cb806b5de86f8a80bb55b2525fb5899f025fc7ee1453\",\"HuellaPreimagenAtestacion\":\"19208c645928cf9c9ded5966f2a9d03dab9f13ecff33d4f43cf739de4b5262a0\",\"HuellaFirmaAtestacion\":\"aad85bf858ca95e9b6401ffb9a6f458818a628666ee00120f0fa90ae9158d6d2\",\"VersionAtestacion\":1,\"Revision\":1,\"Cercado\":1,\"Identidad\":{\"localizador\":{\"version_esquema\":2,\"dominio\":\"localizador\",\"clave_ref\":\"clave:hmac:convocatorias:localizador:v3\",\"generacion_clave\":3,\"valor_hmac_sha256\":\"21a19f3f0f62d2b33d347cf9c6f64db62ef6d4c156ccae13aacab673095823fe\"},\"huella_solicitud\":{\"version_esquema\":2,\"dominio\":\"huella_solicitud\",\"clave_ref\":\"clave:hmac:convocatorias:huella:v3\",\"generacion_clave\":3,\"valor_hmac_sha256\":\"f456f3f649430b0ce1348db1f3d96a27593c6282c0d562ab8d3620299b2c9927\"}},\"ArrendamientoVenceEn\":\"2026-07-18T09:02:00.007Z\",\"ConfirmacionSolicitadaEn\":\"2026-07-18T09:00:00.012Z\",\"RevalidacionSolicitadaEn\":\"2026-07-18T09:00:00.012001Z\",\"ComprobadaEn\":\"2026-07-18T09:00:00.012002Z\"}"
)

func TestVectoresDoradosCriptograficosBorradores(t *testing.T) {
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
	atestacion, solicitud, revalidacion, err := recibo.EvidenciasKMSParaVerificacion()
	if err != nil {
		t.Fatal(err)
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
	huellaPreimagenRevalidacion := sha256.Sum256(preimagenRevalidacion)

	vectores := []struct {
		nombre, obtenido, esperado string
	}{
		{"AAD canónica", string(aad), aadCanonicaBorradorDorada},
		{"huella AAD", huellaAAD, "95bffe4d2422c748d85f103430d2e99ce3117445c362a05f83b1e35bef9ff24c"},
		{"huella política", confirmacion.ResolucionPerfilCifrado.Evidencia.HuellaDecisionPoliticaSHA256, "20ca8cc2c31dc1c9e3a8bfa9063109eab1507aa7578a121d4f09b824a0441e8e"},
		{"huella evidencia perfil", confirmacion.ResolucionPerfilCifrado.Evidencia.HuellaEvidenciaSHA256, "d69bc655b54f22213a2d03811fd92c07da2296c06ad6d534b2d8186cb959e936"},
		{"preimagen atestación KMS", string(preimagenAtestacion), preimagenAtestacionKMSDorada},
		{"huella preimagen atestación", hex.EncodeToString(huellaPreimagenAtestacion[:]), "19208c645928cf9c9ded5966f2a9d03dab9f13ecff33d4f43cf739de4b5262a0"},
		{"firma atestación KMS", base64.RawURLEncoding.EncodeToString(firmaAtestacion), "ejv2we_0n4wzR2t6161HISw99BPQmHutxJ6SfD8fFgBaFJGh7fBitqqNvJno6j7ue8GA-j18GQNuaUvYud5GBQ"},
		{"huella cuerpo recibo", huellaCuerpoReciboBorrador(recibo), "899dffa8f86829470c203320ce37945f2750171732f2c53bda7263e698a7cfd8"},
		{"preimagen revalidación KMS", string(preimagenRevalidacion), preimagenRevalidacionKMSDorada},
		{"huella preimagen revalidación", hex.EncodeToString(huellaPreimagenRevalidacion[:]), "9767b5af7654b5ee9963cef30d5869f3a1beffbe54c3cb18f57efcaecb18bf06"},
		{"firma revalidación KMS", base64.RawURLEncoding.EncodeToString(firmaRevalidacion), "Nk02lywVzSBjWrjbmE-tZx0WWcnRjjAP3mb3fokl6lQr1Ag7foTAW9Pdh_cKYPNmtire-Bgwx_HQD05K3cxaAQ"},
		{"huella acreditación", recibo.AcreditacionKMS.HuellaAcreditacionSHA256, "ca54a859c80cf90b1dbd17214b797d34261f6d8c2427e588cd7d5f5688299b83"},
	}
	for _, vector := range vectores {
		t.Run(vector.nombre, func(t *testing.T) {
			if vector.obtenido != vector.esperado {
				t.Fatalf("vector durable cambió\nobtenido: %q\nesperado: %q", vector.obtenido, vector.esperado)
			}
		})
	}
}
