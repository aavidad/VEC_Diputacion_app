package httpseguridad

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type cuerpoContadoFirmaVec struct {
	lector   io.Reader
	leidos   int
	llamadas int
	cerrado  bool
}

type contextoVinculoPublicoFirmaVec struct {
	context.Context
	vinculo VinculoPeticionPasarela
}

func (c contextoVinculoPublicoFirmaVec) Value(any) any { return c.vinculo }

func (c *cuerpoContadoFirmaVec) Read(destino []byte) (int, error) {
	c.llamadas++
	n, err := c.lector.Read(destino)
	c.leidos += n
	return n, err
}

func (c *cuerpoContadoFirmaVec) Close() error {
	c.cerrado = true
	return nil
}

func TestEmisorRegistroFirmaVecPreparadaNoReleeCuerpoYConservaCanal(t *testing.T) {
	entorno := nuevoEntornoAsercionPasarela(t)
	cuerpo := cuerpoRegistroFirmaVecPasarelaPrueba(LimiteCuerpoRegistroFirmaVecPasarela)
	huella := sha256.Sum256(cuerpo)
	peticion := httptest.NewRequest(http.MethodPost, rutaRegistroFirmaVecPasarela, nil)
	original := &cuerpoContadoFirmaVec{lector: bytes.NewReader(cuerpo)}
	peticion.Body = original
	peticion.ContentLength = int64(len(cuerpo))
	preparada, err := PrepararPeticionAsercionPasarela(peticion, LimiteCuerpoRegistroFirmaVecPasarela)
	if err != nil {
		t.Fatalf("preparar peticion: %v", err)
	}
	defer preparada.Body.Close()
	if !original.cerrado || original.leidos != len(cuerpo) {
		t.Fatalf("lectura inicial leidos=%d cerrado=%v", original.leidos, original.cerrado)
	}
	antesBytes, antesLlamadas := original.leidos, original.llamadas
	protegida, err := entorno.emisor.EmitirRegistroFirmaVecPreparada(preparada.Context(), entorno.identidad)
	if err != nil {
		t.Fatalf("emitir desde contexto preparado: %v", err)
	}
	if original.leidos != antesBytes || original.llamadas != antesLlamadas {
		t.Fatal("el emisor volvio a leer el cuerpo original")
	}
	copia, err := io.ReadAll(preparada.Body)
	if err != nil || len(copia) != len(cuerpo) || sha256.Sum256(copia) != huella {
		t.Fatalf("el cuerpo preparado cambio tras emitir: %v", err)
	}
	preparada.Header.Set(CabeceraAsercionPasarela, base64.RawURLEncoding.EncodeToString(protegida))
	extraida, err := (ExtractorAsercionPasarela{}).ExtraerAsercionProtegida(preparada)
	if err != nil || !bytes.Equal(extraida, protegida) {
		t.Fatalf("extraer asercion: %v", err)
	}
	credencial := debeCredencial(t, extraida, entorno.canal)
	if _, err := entorno.servicio.Resolver(preparada.Context(), credencial); err != nil {
		t.Fatalf("resolver vinculo en el mismo canal: %v", err)
	}
}

func TestEmisorRegistroFirmaVecPreparadaRechazaContextosAjenos(t *testing.T) {
	entorno := nuevoEntornoAsercionPasarela(t)
	if _, err := entorno.emisor.EmitirRegistroFirmaVecPreparada(context.Background(), entorno.identidad); !errors.Is(err, ErrAsercionPasarela) {
		t.Fatalf("contexto sin preparacion: %v", err)
	}
	falso := contextoVinculoPublicoFirmaVec{Context: context.Background(), vinculo: VinculoPeticionPasarela{
		Metodo: http.MethodPost, Ruta: rutaRegistroFirmaVecPasarela, CuerpoSHA256: string(bytes.Repeat([]byte("a"), 64)),
	}}
	if _, err := entorno.emisor.EmitirRegistroFirmaVecPreparada(falso, entorno.identidad); !errors.Is(err, ErrAsercionPasarela) {
		t.Fatalf("vinculo publico sin preparacion: %v", err)
	}
	for _, caso := range []struct{ nombre, metodo, ruta string }{
		{"metodo", http.MethodPut, rutaRegistroFirmaVecPasarela},
		{"ruta", http.MethodPost, "/api/vec/contratacion-temporal/firmas-documento/registro-externo"},
		{"query", http.MethodPost, rutaRegistroFirmaVecPasarela + "?x=1"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(caso.metodo, caso.ruta, nil)
			preparada, err := PrepararPeticionAsercionPasarela(peticion, limiteCuerpoPasarela)
			if err != nil {
				t.Fatalf("preparar ruta ajena para rechazo del emisor: %v", err)
			}
			defer preparada.Body.Close()
			if _, err := entorno.emisor.EmitirRegistroFirmaVecPreparada(preparada.Context(), entorno.identidad); !errors.Is(err, ErrAsercionPasarela) {
				t.Fatalf("emision con %s ajeno: %v", caso.nombre, err)
			}
		})
	}
}
