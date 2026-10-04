package ejecucioncopias

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cs03 "vec-diputacion-granada/internal/modules/administracion/adapters/destinocopias"
	cs03app "vec-diputacion-granada/internal/modules/administracion/application/destinocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	ej "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
)

type fuentePrueba struct{ contenido map[string][]byte }

func (f fuentePrueba) Leer(_ context.Context, _ string, a copias.Artefacto) ([]byte, error) {
	return append([]byte(nil), f.contenido[a.ID]...), nil
}

func TestDestinoConjuntoAutenticadoRecuperaTrasReabrirYRechazaTamper(t *testing.T) {
	ctx := context.Background()
	var fixture struct {
		Manifiesto copias.Manifiesto `json:"manifiesto"`
	}
	b, e := os.ReadFile("../../../../../cmd/vec-copias-comprobar/testdata/compatible.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &fixture); e != nil {
		t.Fatal(e)
	}
	m := fixture.Manifiesto
	m.Verificacion = copias.Verificacion{Estado: "pendiente_verificacion"}
	m.ConjuntoRef = "test-conjunto-1"
	m.OperacionRef = "test-operacion-1"
	m.Proteccion = copias.Proteccion{Formato: "jwe-json", Algoritmo: "A256KW_A256GCM", ClaveRef: "clave-prueba", ClaveVersion: "v1", AutenticacionRef: "indice:" + m.ConjuntoRef, CifradoSHA256: strings.Repeat("0", 64)}
	m.TamanoBytes = 0
	material := make(map[string][]byte)
	for n := range m.Componentes {
		a := &m.Componentes[n]
		contenido := []byte{byte(n + 1)}
		if n == 0 {
			contenido = []byte{}
		}
		a.SHA256 = cs03huella(contenido)
		a.TamanoBytes = int64(len(contenido))
		m.TamanoBytes += a.TamanoBytes
		material[a.ID] = contenido
	}
	if r := copias.ValidarManifiesto(m); len(r) != 0 {
		t.Fatalf("fixture manifiesto: %v", r)
	}
	base := t.TempDir()
	almac := filepath.Join(base, "almacen")
	cat := filepath.Join(base, "catalogo")
	restaurada := filepath.Join(base, "restaurada")
	for _, dir := range []string{almac, cat, restaurada} {
		if e := os.Mkdir(dir, 0700); e != nil {
			t.Fatal(e)
		}
	}
	var clave [32]byte
	for n := range clave {
		clave[n] = byte(n + 1)
	}
	p, e := cs03.NuevoProtectorJWE(clave, "clave-prueba", "v1", 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	storage, e := cs03.NuevoFilesystem(cs03.ConfiguracionFilesystem{Raiz: almac, MaximoClaroBytes: 1 << 20}, p)
	if e != nil {
		t.Fatal(e)
	}
	catalogo, e := AbrirCatalogo(cat, []string{restaurada})
	if e != nil {
		t.Fatal(e)
	}
	d := DestinoCS03{Destino: storage, Fuente: fuentePrueba{material}, Catalogo: catalogo, ProteccionEsperada: m.Proteccion}
	origen := fixture.Manifiesto.Verificacion.Fisica
	// La medida de origen no es un ensayo de arranque del binario archivado.
	origen.ArranqueRef = ""
	publicado, e := d.Publicar(ctx, ej.Captura{Manifiesto: m, Origen: origen})
	if e != nil {
		t.Fatal(e)
	}
	if publicado.ManifiestoSHA256 == "" {
		t.Fatal("falta huella del manifiesto")
	}
	if publicado.ManifiestoBaseSHA256 != publicado.ManifiestoSHA256 || publicado.EjecucionVerificacionRef != publicado.IndiceAutenticadoRef {
		t.Fatal("vínculo pendiente inicial incompleto")
	}
	if e := storage.Close(); e != nil {
		t.Fatal(e)
	}
	if e := catalogo.Close(); e != nil {
		t.Fatal(e)
	}
	storage, e = cs03.NuevoFilesystem(cs03.ConfiguracionFilesystem{Raiz: almac, MaximoClaroBytes: 1 << 20}, p)
	if e != nil {
		t.Fatal(e)
	}
	defer storage.Close()
	catalogo, e = AbrirCatalogo(cat, []string{restaurada})
	if e != nil {
		t.Fatal(e)
	}
	defer catalogo.Close()
	d = DestinoCS03{Destino: storage, Fuente: fuentePrueba{material}, Catalogo: catalogo, ProteccionEsperada: m.Proteccion}
	recuperado, e := d.Recuperar(ctx, m.ConjuntoRef)
	if e != nil || recuperado.IndiceAutenticadoRef != publicado.IndiceAutenticadoRef || recuperado.Origen != origen {
		t.Fatalf("recuperación: %v", e)
	}
	contenido, e := d.LeerComponente(ctx, m.ConjuntoRef, m.Componentes[0].ID)
	if e != nil || string(contenido) != string(material[m.Componentes[0].ID]) {
		t.Fatalf("material: %v", e)
	}
	clear(contenido)
	v := copias.Verificacion{Estado: "valida", VerificadorVersion: "v1", Fecha: m.Fin.Add(time.Second), Fisica: fixture.Manifiesto.Verificacion.Fisica, Logica: fixture.Manifiesto.Verificacion.Logica}
	final, e := d.CerrarVerificacion(ctx, recuperado, v)
	if e != nil {
		t.Fatal(e)
	}
	if final.IndiceAutenticadoRef == publicado.IndiceAutenticadoRef {
		t.Fatal("índice final igual al pendiente")
	}
	if final.ManifiestoBaseSHA256 != publicado.ManifiestoBaseSHA256 || final.EjecucionVerificacionRef != publicado.EjecucionVerificacionRef || final.ManifiestoSHA256 == publicado.ManifiestoSHA256 {
		t.Fatal("vínculo base no conservado")
	}
	if final.Origen != origen {
		t.Fatal("origen cambiado en cierre")
	}
	if e = storage.Close(); e != nil {
		t.Fatal(e)
	}
	if e = catalogo.Close(); e != nil {
		t.Fatal(e)
	}
	storage, e = cs03.NuevoFilesystem(cs03.ConfiguracionFilesystem{Raiz: almac, MaximoClaroBytes: 1 << 20}, p)
	if e != nil {
		t.Fatal(e)
	}
	defer storage.Close()
	catalogo, e = AbrirCatalogo(cat, []string{restaurada})
	if e != nil {
		t.Fatal(e)
	}
	defer catalogo.Close()
	d = DestinoCS03{Destino: storage, Fuente: fuentePrueba{material}, Catalogo: catalogo, ProteccionEsperada: m.Proteccion}
	conservado, e := d.Recuperar(ctx, m.ConjuntoRef)
	if e != nil || conservado.ManifiestoBaseSHA256 != publicado.ManifiestoBaseSHA256 || conservado.EjecucionVerificacionRef != publicado.EjecucionVerificacionRef {
		t.Fatalf("base tras reabrir: %v", e)
	}
	otraClave := d
	otraClave.ProteccionEsperada.ClaveRef = "clave-distinta"
	if _, e = otraClave.Recuperar(ctx, m.ConjuntoRef); e == nil {
		t.Fatal("metadatos de clave distinta aceptados")
	}
	idx, _, e := d.leerIndice(ctx, m.ConjuntoRef)
	if e != nil {
		t.Fatal(e)
	}
	if idx.IndiceInicial == nil {
		t.Fatal("falta referencia inicial sellada")
	}
	inicialPath := filepath.Join(almac, idx.IndiceInicial.ObjetoRef)
	inicialCifrado, e := os.ReadFile(inicialPath)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(inicialPath, []byte("indice-inicial-alterado"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = d.Recuperar(ctx, m.ConjuntoRef); e == nil {
		t.Fatal("índice inicial alterado aceptado")
	}
	if e = os.WriteFile(inicialPath, inicialCifrado, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = d.Recuperar(ctx, m.ConjuntoRef); e != nil {
		t.Fatal(e)
	}
	componentePath := filepath.Join(almac, idx.Componentes[0].ObjetoRef)
	original, e := os.ReadFile(componentePath)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(componentePath, []byte("componente-alterado"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = d.Recuperar(ctx, m.ConjuntoRef); e == nil {
		t.Fatal("componente alterado aceptado")
	}
	if e = os.WriteFile(componentePath, original, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = d.Recuperar(ctx, m.ConjuntoRef); e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(componentePath); e != nil {
		t.Fatal(e)
	}
	if _, e = d.Recuperar(ctx, m.ConjuntoRef); e == nil {
		t.Fatal("componente ausente aceptado")
	}
	if e = os.WriteFile(componentePath, original, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = d.Recuperar(ctx, m.ConjuntoRef); e != nil {
		t.Fatal(e)
	}
	indice, e := catalogo.Leer(ctx, m.ConjuntoRef)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(almac, indice.ObjetoRef), []byte("cifrado-alterado"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = d.Recuperar(ctx, m.ConjuntoRef); e == nil {
		t.Fatal("índice alterado aceptado")
	}
}

func cs03huella(b []byte) string { return cs03app.Huella(b) }
