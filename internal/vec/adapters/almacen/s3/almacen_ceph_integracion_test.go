//go:build integracion_ceph

package s3

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestIntegracionCephPerfilFuerte ejecuta la sonda destructiva real del
// contrato. Solo usa claves tecnicas opacas, pero deja durante su retencion las
// dos versiones WORM de la sonda. Nunca debe ejecutarse contra buckets ajenos.
func TestIntegracionCephPerfilFuerte(t *testing.T) {
	if os.Getenv("VEC_CEPH_INTEGRACION") != "1" {
		t.Skip("integracion Ceph no habilitada")
	}
	valores := map[string]string{
		"conector_id":                "ceph_integracion",
		"endpoint":                   os.Getenv("VEC_CEPH_ENDPOINT"),
		"region":                     os.Getenv("VEC_CEPH_REGION"),
		"bucket_cuarentena":          os.Getenv("VEC_CEPH_BUCKET_CUARENTENA"),
		"bucket_admitida":            os.Getenv("VEC_CEPH_BUCKET_ADMITIDA"),
		"access_key_id":              os.Getenv("VEC_CEPH_ACCESS_KEY_ID"),
		"secret_access_key":          os.Getenv("VEC_CEPH_SECRET_ACCESS_KEY"),
		"session_token":              os.Getenv("VEC_CEPH_SESSION_TOKEN"),
		"ruta_ca":                    os.Getenv("VEC_CEPH_RUTA_CA"),
		"redes_permitidas":           os.Getenv("VEC_CEPH_REDES_PERMITIDAS"),
		"path_style":                 valorEntornoO("VEC_CEPH_PATH_STYLE", "true"),
		"tamano_maximo_bytes":        valorEntornoO("VEC_CEPH_TAMANO_MAXIMO_BYTES", "268435456"),
		"duracion_carga_directa":     valorEntornoO("VEC_CEPH_DURACION_CARGA_DIRECTA", "5m"),
		"retencion_minima_admitida":  valorEntornoO("VEC_CEPH_RETENCION_MINIMA_ADMITIDA", "1h"),
		"clave_derivacion_base64url": os.Getenv("VEC_CEPH_CLAVE_DERIVACION_BASE64URL"),
		"cifrado":                    valorEntornoO("VEC_CEPH_CIFRADO", "AES256"),
		"clave_kms":                  os.Getenv("VEC_CEPH_CLAVE_KMS"),
		"usar_bucket_key_kms":        valorEntornoO("VEC_CEPH_USAR_BUCKET_KEY_KMS", "false"),
		"perfil_fuerte":              "true",
		"probar_capacidades":         "true",
		"permitir_eliminacion":       "false",
		"modo_retencion":             "COMPLIANCE",
	}
	configuracion, err := ConfiguracionDesdeMapa(valores)
	if err != nil {
		t.Fatal("configuracion Ceph de integracion no valida")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelar()
	almacen, err := Nuevo(ctx, configuracion)
	if err != nil {
		t.Fatal("Ceph no supera el contrato fuerte S3")
	}
	capacidad, err := almacen.Capacidades(ctx)
	if err != nil || !capacidad.Versionado || !capacidad.IntegridadSHA256 || !capacidad.Retencion ||
		!capacidad.BloqueoLegal || !capacidad.PromocionAtomica || !capacidad.RetencionAtomicaEnPromocion ||
		!capacidad.PreservaObjetoOriginal || !capacidad.CargaDirectaTemporal || !capacidad.CifradoPorObjeto {
		t.Fatal("Ceph anuncio un perfil inferior al contrato fuerte")
	}
}

func valorEntornoO(nombre, predeterminado string) string {
	if valor := os.Getenv(nombre); valor != "" {
		return valor
	}
	return predeterminado
}
