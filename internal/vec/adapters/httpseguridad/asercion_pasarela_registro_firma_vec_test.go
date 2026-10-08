package httpseguridad

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func cuerpoRegistroFirmaVecPasarelaPrueba(longitud int) []byte {
	cuerpo := make([]byte, longitud)
	for i := range cuerpo {
		cuerpo[i] = byte(i*37 + 11)
	}
	return cuerpo
}

func TestRegistroFirmaVecPasarelaConservaBytesHuellaYCanal(t *testing.T) {
	cuerpo := cuerpoRegistroFirmaVecPasarelaPrueba(LimiteCuerpoRegistroFirmaVecPasarela)
	huella := sha256.Sum256(cuerpo)
	peticion := httptest.NewRequest(http.MethodPost, rutaRegistroFirmaVecPasarela, bytes.NewReader(cuerpo))
	preparada, err := PrepararPeticionAsercionPasarela(peticion, LimiteCuerpoRegistroFirmaVecPasarela)
	if err != nil {
		t.Fatalf("preparar cuerpo en el limite: %v", err)
	}
	defer preparada.Body.Close()
	vinculo, ok := preparada.Context().Value(claveContextoPeticionPasarela{}).(VinculoPeticionPasarela)
	if !ok || vinculo.Metodo != http.MethodPost || vinculo.Ruta != rutaRegistroFirmaVecPasarela ||
		vinculo.CuerpoSHA256 != hex.EncodeToString(huella[:]) {
		t.Fatal("vinculo no corresponde a los bytes y ruta originales")
	}
	clear(cuerpo)
	repuesto, err := io.ReadAll(preparada.Body)
	if err != nil {
		t.Fatalf("leer copia privada: %v", err)
	}
	if len(repuesto) != LimiteCuerpoRegistroFirmaVecPasarela || sha256.Sum256(repuesto) != huella {
		t.Fatal("la copia privada altero el cuerpo del registro")
	}

	entorno := nuevoEntornoAsercionPasarela(t)
	protegida := entorno.emitirPara(t, preparada)
	preparada.Header.Set(CabeceraAsercionPasarela, base64.RawURLEncoding.EncodeToString(protegida))
	extraida, err := (ExtractorAsercionPasarela{}).ExtraerAsercionProtegida(preparada)
	if err != nil || !bytes.Equal(extraida, protegida) {
		t.Fatalf("extraer asercion vinculada: %v", err)
	}
	credencial := debeCredencial(t, extraida, entorno.canal)
	if _, err := entorno.servicio.Resolver(preparada.Context(), credencial); err != nil {
		t.Fatalf("resolver asercion en el mismo canal: %v", err)
	}
}

func TestRegistroFirmaVecPasarelaAcotaRutaMetodoYLongitud(t *testing.T) {
	justoSobreGeneral := cuerpoRegistroFirmaVecPasarelaPrueba(limiteCuerpoPasarela + 1)
	masUno := cuerpoRegistroFirmaVecPasarelaPrueba(LimiteCuerpoRegistroFirmaVecPasarela + 1)
	for _, caso := range []struct {
		nombre, metodo, ruta string
		cuerpo               []byte
		permitido            bool
	}{
		{"exacto", http.MethodPost, rutaRegistroFirmaVecPasarela, justoSobreGeneral, true},
		{"un_byte_sobre_limite", http.MethodPost, rutaRegistroFirmaVecPasarela, masUno, false},
		{"otro_metodo", http.MethodPut, rutaRegistroFirmaVecPasarela, justoSobreGeneral, false},
		{"otra_ruta", http.MethodPost, rutaRegistroFirmaVecPasarela + "/", justoSobreGeneral, false},
		{"consulta", http.MethodPost, rutaRegistroFirmaVecPasarela + "?x=1", justoSobreGeneral, false},
		{"consulta_vacia", http.MethodPost, rutaRegistroFirmaVecPasarela + "?", justoSobreGeneral, false},
		{"ruta_escapada", http.MethodPost, "/api/vec/contratacion-temporal/firmas-documento/%72egistro-vec", justoSobreGeneral, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			vinculo, err := NuevoVinculoPeticionPasarela(caso.metodo, caso.ruta, caso.cuerpo)
			if (err == nil) != caso.permitido || (!caso.permitido && !errors.Is(err, ErrAsercionPasarela)) {
				t.Fatalf("vinculo permitido=%v, err=%v", caso.permitido, err)
			}
			if caso.permitido && vinculo.Ruta != rutaRegistroFirmaVecPasarela {
				t.Fatal("vinculo con ruta distinta")
			}
			peticion := httptest.NewRequest(caso.metodo, caso.ruta, bytes.NewReader(caso.cuerpo))
			if caso.nombre == "un_byte_sobre_limite" {
				peticion.ContentLength = -1 // El limite rige tambien sin longitud declarada.
			}
			preparada, err := PrepararPeticionAsercionPasarela(peticion, LimiteCuerpoRegistroFirmaVecPasarela)
			if (err == nil) != caso.permitido || (!caso.permitido && !errors.Is(err, ErrAsercionPasarela)) {
				t.Fatalf("preparacion permitida=%v, err=%v", caso.permitido, err)
			}
			if preparada != nil {
				_ = preparada.Body.Close()
			}
		})
	}
	peticion := httptest.NewRequest(http.MethodPost, rutaRegistroFirmaVecPasarela, bytes.NewReader(justoSobreGeneral))
	if _, err := PrepararPeticionAsercionPasarela(peticion, limiteCuerpoPasarela); !errors.Is(err, ErrAsercionPasarela) {
		t.Fatalf("el limite general de la pasarela se aumento sin optar por el contrato: %v", err)
	}
}
