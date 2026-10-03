package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func formatoExportacionPrueba(t *testing.T) FormatoExportacionServiciosPropios {
	t.Helper()
	f, e := NuevoFormatoExportacionServiciosPropios(DatosFormatoExportacionServiciosPropios{Referencia: "personal:servicios_propios:csv", Version: 1, Idioma: "xx", CatalogoSHA256: strings.Repeat("a", 64), NombreArchivo: "servicios.csv", Cabeceras: []string{"c1", "c2", "c3", "c4", "c5"}, Estados: map[string]string{"declarado": "e1", "comprobado": "e2", "reconocido": "e3"}})
	if e != nil {
		t.Fatal(e)
	}
	return f
}
func solicitudExportacionPrueba(t *testing.T) SolicitudExportacionServiciosPropios {
	return SolicitudExportacionServiciosPropios{Actor: actorFichaPropiaPrueba(t, vinculoEmpleadoPrueba("pep_", "emp_"+strings.Repeat("b", 24))), ReciboRef: "fichapropia:0f0e0d0c-0b0a-4908-8706-050403020100", Corte: corteFichaPropiaPrueba(), Idioma: "xx"}
}
func TestExportacionServiciosMaterialLigaReciboCorteFormatoActorYHuellaSQL(t *testing.T) {
	s := solicitudExportacionPrueba(t)
	f := formatoExportacionPrueba(t)
	m, e := NuevoMaterialExportacionServiciosPropios(s, f)
	if e != nil {
		t.Fatal(e)
	}
	h := sha256.Sum256(m.Canonico())
	atr := m.Recurso().Atributos
	if atr["material_sha256"] != hex.EncodeToString(h[:]) || atr["operacion"] != "servicios_propios_exportar" || atr["recibo_ref"] != s.ReciboRef || m.Recurso().Tipo != TipoRecursoExportacionServiciosPropios {
		t.Fatal("material no ligado")
	}
	ctx := struct {
		Ambitos   map[string]string `json:"ambitos"`
		Atributos map[string]string `json:"atributos"`
	}{m.Recurso().Ambitos, atr}
	canon, _ := json.Marshal(ctx)
	sha := sha256.Sum256(canon)
	huella, e := m.HuellaSHA256()
	if e != nil || huella != hex.EncodeToString(sha[:]) {
		t.Fatal("canon SQL incompatible")
	}
	alterada := s
	alterada.ReciboRef = "fichapropia:11111111-1111-1111-1111-111111111111"
	otra, _ := NuevoMaterialExportacionServiciosPropios(alterada, f)
	if string(otra.Canonico()) == string(m.Canonico()) {
		t.Fatal("recibo no ligado")
	}
	copia := m.Formato().Datos()
	copia.Cabeceras[0] = "ajena"
	if m.Formato().Datos().Cabeceras[0] == "ajena" {
		t.Fatal("formato mutable")
	}
}
func TestExportacionServiciosDeniegaSinEmpleadoYFormatoAjeno(t *testing.T) {
	s := solicitudExportacionPrueba(t)
	s.Actor = actorFichaPropiaPrueba(t)
	if _, e := NuevoMaterialExportacionServiciosPropios(s, formatoExportacionPrueba(t)); !errors.Is(e, ErrExportacionServiciosPropiosDenegada) {
		t.Fatal(e)
	}
	s = solicitudExportacionPrueba(t)
	s.Idioma = "../xx"
	if _, e := NuevoMaterialExportacionServiciosPropios(s, formatoExportacionPrueba(t)); !errors.Is(e, ErrExportacionServiciosPropiosInvalida) {
		t.Fatal(e)
	}
	d := formatoExportacionPrueba(t).Datos()
	d.NombreArchivo = "../f.csv"
	if _, e := NuevoFormatoExportacionServiciosPropios(d); !errors.Is(e, ErrExportacionServiciosPropiosInvalida) {
		t.Fatal("ruta permitida")
	}
}
