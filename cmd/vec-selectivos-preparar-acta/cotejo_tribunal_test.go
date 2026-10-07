package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

func TestCLICotejaSalidaExactaDelTribunal(t *testing.T) {
	material, err := os.ReadFile(filepath.Join("..", "vec-selectivos-preparar-tribunal", "testdata", "material.json"))
	if err != nil {
		t.Fatal(err)
	}
	var propuesto domain.MaterialTribunalPropuesto
	if err := json.Unmarshal(material, &propuesto); err != nil {
		t.Fatal(err)
	}
	preparacion, err := domain.PrepararTribunal(propuesto)
	if err != nil {
		t.Fatal(err)
	}
	var salidaTribunal bytes.Buffer
	enc := json.NewEncoder(&salidaTribunal)
	enc.SetIndent("", "  ")
	if err := enc.Encode(struct {
		Preparacion domain.PreparacionTribunal `json:"preparacion"`
		Limite      string                     `json:"limite"`
		Mensajes    []pendienteVisible         `json:"mensajes"`
	}{preparacion, "pendiente", []pendienteVisible{}}); err != nil {
		t.Fatal(err)
	}
	archivo := filepath.Join(t.TempDir(), "tribunal.json")
	if err := os.WriteFile(archivo, salidaTribunal.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	acta, err := os.ReadFile(filepath.Join("testdata", "material-cotejo.json"))
	if err != nil {
		t.Fatal(err)
	}
	var salida, errores bytes.Buffer
	args := []string{"-catalogos-dir", "../../web/static/textos", "-tribunal-salida", archivo}
	if codigo := ejecutar(context.Background(), args, bytes.NewReader(acta), &salida, &errores); codigo != 0 {
		t.Fatalf("código %d, diagnóstico %s", codigo, errores.String())
	}
	huella := sha256.Sum256(salidaTribunal.Bytes())
	huellaHex := hex.EncodeToString(huella[:])
	if !bytes.Contains(salida.Bytes(), []byte(huellaHex)) || !bytes.Contains(salida.Bytes(), []byte(`"cotejo_local": "salida_tribunal_sha256"`)) {
		t.Fatalf("falta huella de la salida exacta: %s", salida.String())
	}
	var cotejada struct {
		Limite      string                 `json:"limite"`
		Preparacion domain.PreparacionActa `json:"preparacion"`
		Mensajes    []pendienteVisible     `json:"mensajes"`
	}
	if err := json.Unmarshal(salida.Bytes(), &cotejada); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cotejada.Limite, "no se han cotejado") || len(cotejada.Mensajes) != len(cotejada.Preparacion.Pendientes) {
		t.Fatalf("límite o mensajes contradictorios: %s", salida.String())
	}
	for i, pendiente := range cotejada.Preparacion.Pendientes {
		if pendiente != (domain.PendienteActa{Campo: cotejada.Mensajes[i].Campo, Codigo: cotejada.Mensajes[i].Codigo}) ||
			pendiente.Codigo == "antecedente_no_cotejado" || pendiente.Codigo == "pertenencia_no_verificada" ||
			strings.Contains(cotejada.Mensajes[i].Mensaje, "pendiente cotejarlas") {
			t.Fatalf("aviso contradictorio: %+v / %+v", pendiente, cotejada.Mensajes[i])
		}
	}
	for _, idioma := range []string{"es", "en"} {
		catalogo, err := cargarCatalogo("../../web/static/textos", idioma)
		if err != nil {
			t.Fatal(err)
		}
		var salidaIdioma, erroresIdioma bytes.Buffer
		argsIdioma := []string{"-catalogos-dir", "../../web/static/textos", "-idioma", idioma, "-tribunal-salida", archivo}
		if codigo := ejecutar(context.Background(), argsIdioma, bytes.NewReader(acta), &salidaIdioma, &erroresIdioma); codigo != 0 {
			t.Fatalf("cotejo %s: %d %s", idioma, codigo, erroresIdioma.String())
		}
		var modoCon struct {
			Limite      string                 `json:"limite"`
			Preparacion domain.PreparacionActa `json:"preparacion"`
			Mensajes    []pendienteVisible     `json:"mensajes"`
			CotejoLocal string                 `json:"cotejo_local"`
		}
		if json.Unmarshal(salidaIdioma.Bytes(), &modoCon) != nil || modoCon.Limite != catalogo.T(idioma, "limite_cotejo_local") ||
			modoCon.CotejoLocal != "salida_tribunal_sha256" || len(modoCon.Mensajes) < 2 || len(modoCon.Mensajes) != len(modoCon.Preparacion.Pendientes) ||
			modoCon.Mensajes[0].Mensaje != catalogo.T(idioma, "campo_antecedente_tribunal")+": "+catalogo.T(idioma, "antecedente_cotejado_local") ||
			modoCon.Mensajes[1].Mensaje != catalogo.T(idioma, "campo_fase_propuesta")+": "+catalogo.T(idioma, "fase_cotejada_local") {
			t.Fatalf("modo cotejado %s incoherente: %s", idioma, salidaIdioma.String())
		}
		var sinCotejo, sinCotejoError bytes.Buffer
		materialOriginal, err := os.ReadFile(filepath.Join("testdata", "material.json"))
		if err != nil {
			t.Fatal(err)
		}
		if codigo := ejecutar(context.Background(), []string{"-catalogos-dir", "../../web/static/textos", "-idioma", idioma}, bytes.NewReader(materialOriginal), &sinCotejo, &sinCotejoError); codigo != 0 {
			t.Fatalf("modo anterior %s: %d %s", idioma, codigo, sinCotejoError.String())
		}
		var modoSin struct {
			Limite      string                 `json:"limite"`
			Preparacion domain.PreparacionActa `json:"preparacion"`
			Mensajes    []pendienteVisible     `json:"mensajes"`
			CotejoLocal string                 `json:"cotejo_local"`
		}
		if json.Unmarshal(sinCotejo.Bytes(), &modoSin) != nil || modoSin.Limite != catalogo.T(idioma, "limite") ||
			modoSin.CotejoLocal != "" || bytes.Contains(sinCotejo.Bytes(), []byte(`"cotejo_local"`)) || len(modoSin.Mensajes) < 2 || len(modoSin.Preparacion.Pendientes) < 2 ||
			modoSin.Preparacion.Pendientes[0].Codigo != "antecedente_no_cotejado" || modoSin.Preparacion.Pendientes[1].Codigo != "pertenencia_no_verificada" ||
			modoSin.Mensajes[0].Mensaje != catalogo.T(idioma, "campo_antecedente_tribunal")+": "+catalogo.T(idioma, "antecedente_no_cotejado") ||
			modoSin.Mensajes[1].Mensaje != catalogo.T(idioma, "campo_fase_propuesta")+": "+catalogo.T(idioma, "pertenencia_no_verificada") {
			t.Fatalf("modo anterior %s alterado: %s", idioma, sinCotejo.String())
		}
	}
	acta = bytes.Replace(acta, []byte("fase:ejemplo-sintetico"), []byte("fase:ajena"), 1)
	salida.Reset()
	errores.Reset()
	if codigo := ejecutar(context.Background(), args, bytes.NewReader(acta), &salida, &errores); codigo != 1 || salida.Len() != 0 || !strings.Contains(errores.String(), "cotejo_invalido") {
		t.Fatalf("fase ajena: código %d, salida %s, error %s", codigo, salida.String(), errores.String())
	}
}
