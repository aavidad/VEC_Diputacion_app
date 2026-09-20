package asercionfirmada

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

func TestPeticionRechazaFirmaAlteradaYAlgoritmoNoAdmitido(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	emisor, _ := NuevoEmisor(ConfiguracionEmisor{ClaveID: "k1", Clave: priv})
	verificador, _ := NuevoVerificador(ConfiguracionVerificador{Claves: map[string]ed25519.PublicKey{"k1": pub}})
	p := Peticion{Emisor: "https://auth.example", Audiencia: "vec.personal", Superficie: httpseguridad.SuperficieExternaPersonal, SesionID: "s", Metodo: "GET", Destino: "/x", CuerpoSHA256: HuellaCuerpo(nil), NonceSHA256: HuellaCuerpo([]byte("n")), EmitidaEn: time.Now().UTC().Truncate(time.Microsecond), ExpiraEn: time.Now().UTC().Add(time.Minute).Truncate(time.Microsecond), CanalVinculadoRef: "tls-exportador:sha256:x"}
	material, err := emisor.EmitirPeticion(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = verificador.VerificarPeticion(context.Background(), material); err != nil {
		t.Fatal(err)
	}
	material[len(material)-2] ^= 1
	if _, err = verificador.VerificarPeticion(context.Background(), material); err == nil {
		t.Fatal("firma alterada admitida")
	}
	if _, err = verificador.VerificarPeticion(context.Background(), append(material, '.')); err == nil {
		t.Fatal("segmento extra admitido")
	}
	_, otra, _ := ed25519.GenerateKey(rand.Reader)
	otro, _ := NuevoEmisor(ConfiguracionEmisor{ClaveID: "desconocida", Clave: otra})
	material, _ = otro.EmitirPeticion(context.Background(), p)
	if _, err = verificador.VerificarPeticion(context.Background(), material); err == nil {
		t.Fatal("kid desconocido admitido")
	}
	material, _ = emisor.EmitirInicio(context.Background(), httpseguridad.AsercionProxyIdentidad{})
	if _, err = verificador.VerificarPeticion(context.Background(), material); err == nil {
		t.Fatal("inicio confundido con peticion")
	}

	carga, _ := json.Marshal(peticion{Tipo: "PETICION", Peticion: p})
	firmanteHMAC, _ := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.HS256, Key: []byte("0123456789abcdef0123456789abcdef")},
		(&jose.SignerOptions{}).WithType("VEC-IDENTIDAD-V1").WithHeader("kid", "k1"),
	)
	firmadaHMAC, _ := firmanteHMAC.Sign(carga)
	compactaHMAC, _ := firmadaHMAC.CompactSerialize()
	if _, err = verificador.VerificarPeticion(context.Background(), []byte(compactaHMAC)); err == nil {
		t.Fatal("algoritmo HMAC admitido")
	}

	firmanteTipoErroneo, _ := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.EdDSA, Key: priv},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "k1"),
	)
	firmadaTipoErroneo, _ := firmanteTipoErroneo.Sign(carga)
	compactaTipoErroneo, _ := firmadaTipoErroneo.CompactSerialize()
	if _, err = verificador.VerificarPeticion(context.Background(), []byte(compactaTipoErroneo)); err == nil {
		t.Fatal("typ distinto admitido")
	}

	cargaExtra := append(carga[:len(carga)-1], []byte(`,"extra":true}`)...)
	firmanteExtra, _ := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.EdDSA, Key: priv},
		(&jose.SignerOptions{}).WithType("VEC-IDENTIDAD-V1").WithHeader("kid", "k1"),
	)
	firmadaExtra, _ := firmanteExtra.Sign(cargaExtra)
	compactaExtra, _ := firmadaExtra.CompactSerialize()
	if _, err = verificador.VerificarPeticion(context.Background(), []byte(compactaExtra)); err == nil {
		t.Fatal("campo firmado desconocido admitido")
	}
}
