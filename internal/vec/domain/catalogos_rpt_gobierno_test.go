package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func contenidoPublicarRPTPrueba(t *testing.T) ContenidoGobiernoCategoriaRPT {
	t.Helper()
	borrador := catalogoConfigurablePrueba()
	publicado, err := borrador.Publicar("responsable-configuracion-2", "aprobacion-1", "Revision funcional superada", borrador.CreadoEn.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := json.Marshal(publicado)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(doc)
	return ContenidoGobiernoCategoriaRPT{
		Accion: AccionGobiernoCategoriaRPTPublicar, CatalogoID: publicado.ID,
		ModuloID: publicado.ModuloID, Version: publicado.Version,
		DocumentoCanonico: ptrGobiernoRPT(string(doc)), DocumentoHuellaSHA256: ptrGobiernoRPT(hex.EncodeToString(h[:])),
		PreimagenesControl:      map[string]PreimagenControlGobiernoCategoriaRPT{},
		PreimagenesHuellaSHA256: strings.Repeat("a", 64),
		MotivoRef:               "motivos:1:motivo_0123456789abcdef0123456789abcdef",
		FuenteRef:               publicado.FuenteRef,
	}
}

func ptrGobiernoRPT[T any](v T) *T { return &v }

func TestContenidoGobiernoCategoriaRPTExigeEditorSeparadoYDocumentoIntacto(t *testing.T) {
	c := contenidoPublicarRPTPrueba(t)
	if err := c.ValidarParaEditor("tecnico-configuracion-1"); err != nil {
		t.Fatal(err)
	}
	borrador := c
	borrador.DocumentoHuellaSHA256 = nil
	borrador.PreimagenesHuellaSHA256 = ""
	completado, err := borrador.PrepararBorradorParaEditor("tecnico-configuracion-1")
	if err != nil || completado.PreimagenesHuellaSHA256 != "" ||
		*completado.DocumentoHuellaSHA256 != *c.DocumentoHuellaSHA256 {
		t.Fatalf("borrador preparado incorrectamente: %v", err)
	}
	if err := c.ValidarParaEditor("responsable-configuracion-2"); !errors.Is(err, ErrGobiernoCategoriaRPTInvalido) {
		t.Fatalf("autopublicacion: %v", err)
	}
	alterado := c
	alterado.DocumentoCanonico = ptrGobiernoRPT(*c.DocumentoCanonico + " ")
	if err := alterado.ValidarParaEditor("tecnico-configuracion-1"); !errors.Is(err, ErrGobiernoCategoriaRPTInvalido) {
		t.Fatalf("huella de documento alterada: %v", err)
	}
}

func TestContenidoGobiernoCategoriaRPTDeshabilitarExigePreimagenExacta(t *testing.T) {
	id := "categoria.ejemplo"
	rev := int64(7)
	c := ContenidoGobiernoCategoriaRPT{
		Accion:     AccionGobiernoCategoriaRPTDeshabilitar,
		CatalogoID: "catalogo.ejemplo", ModuloID: "bolsa", Version: 2,
		CategoriaID: &id, RevisionEsperada: &rev,
		PreimagenesControl: map[string]PreimagenControlGobiernoCategoriaRPT{
			id: {Version: 2, Revision: rev, HuellaSHA256: strings.Repeat("a", 64), Estado: "habilitada"},
		},
		PreimagenesHuellaSHA256: strings.Repeat("b", 64),
		MotivoRef:               "motivos:1:motivo_0123456789abcdef0123456789abcdef",
		FuenteRef:               "resolucion-2026-99",
	}
	if err := c.ValidarParaEditor("tecnico-configuracion-1"); err != nil {
		t.Fatal(err)
	}
	c.PreimagenesControl[id] = PreimagenControlGobiernoCategoriaRPT{Version: 2, Revision: rev - 1, HuellaSHA256: strings.Repeat("a", 64), Estado: "habilitada"}
	if err := c.ValidarParaEditor("tecnico-configuracion-1"); !errors.Is(err, ErrGobiernoCategoriaRPTInvalido) {
		t.Fatalf("CAS obsoleto: %v", err)
	}
}
