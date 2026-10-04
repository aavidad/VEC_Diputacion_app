package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func metadatosDenominacionPrueba() MetadatosDenominacionPersonaDesarrollo {
	return MetadatosDenominacionPersonaDesarrollo{Version: 1, Entorno: "desarrollo", Norma: NormaDenominacionPersonaDesarrollo{Referencia: "norma:denominacion:prueba:v1", Case: "fold", FormaUnicode: "NFC", Separadores: " -'", MaxBytes: 512, MaxTokens: 12}, Cifrado: ClaveDenominacionPersonaDesarrollo{Referencia: "clave:denominacion:cifrado:prueba", Version: 1}, Busqueda: ClaveDenominacionPersonaDesarrollo{Referencia: "clave:denominacion:busqueda:prueba", Version: 1}, CifradoRetenidas: []ClaveDenominacionPersonaDesarrollo{}}
}
func guardarMetadatosDenominacion(t *testing.T, ruta string, m MetadatosDenominacionPersonaDesarrollo) {
	t.Helper()
	if os.Chmod(filepath.Dir(ruta), 0700) != nil {
		t.Fatal("directorio_privado")
	}
	b, e := json.Marshal(m)
	if e != nil || os.WriteFile(ruta, b, 0600) != nil {
		t.Fatal("metadata_fixture")
	}
}
func TestKMSEnvolturaDenominacionDedicadaYRetiroVivo(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "metadata.json")
	m := metadatosDenominacionPrueba()
	guardarMetadatosDenominacion(t, ruta, m)
	f, e := NuevaFuenteConfiguracionPrivadaDenominacionPersona(ruta)
	if e != nil {
		t.Fatal("fuente")
	}
	var master [32]byte
	for i := range master {
		master[i] = byte(i + 1)
	}
	ahora := func() time.Time { return time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC) }
	p, indice, e := NuevoProtectorDenominacionPersonaDesarrollo(master, f, ahora)
	if e != nil {
		t.Fatal("protector")
	}
	nombre := []byte("Elena Márquez")
	prep, e := p.PrepararDenominacionPersona(context.Background(), "per_aaaaaaaaaaaaaaaaaaaaaaaa", 0, "procedencia:sintetica", "ambito:admin", indice, nombre)
	if e != nil {
		t.Fatal("preparar")
	}
	envelope := derivarClaveDesarrollo(master, "vec.kms.desarrollo.envoltura.v1")
	src := &fuenteClavesDenominacionDesarrollo{envoltura: envelope, fuente: f, norma: m.norma()}
	claves, e := src.CargarClavesDenominacionPersona(context.Background())
	if e != nil {
		t.Fatal("claves")
	}
	contacto := derivarClaveDesarrollo(envelope, "vec.kms.desarrollo.contacto-usuario.v1")
	if claves.Cifrado.Material == claves.Busqueda.Material || claves.Cifrado.Material == contacto || claves.Busqueda.Material == contacto {
		t.Fatal("dominios_no_separados")
	}
	old := m.Cifrado
	old.RetenerHasta = ahora().Add(time.Hour)
	m.Cifrado.Version = 2
	m.CifradoRetenidas = []ClaveDenominacionPersonaDesarrollo{old}
	guardarMetadatosDenominacion(t, ruta, m)
	coincide := false
	e = p.ConDenominacionDescifrada(context.Background(), prep.Sobre, func(d domain.DenominacionPersona) error {
		return d.ConNombreMostrar(func(b []byte) error { coincide = bytes.Equal(b, nombre); return nil })
	})
	if e != nil || !coincide {
		t.Fatal("retencion_original")
	}
	m.CifradoRetenidas[0].Revocada = true
	guardarMetadatosDenominacion(t, ruta, m)
	if p.ConDenominacionDescifrada(context.Background(), prep.Sobre, func(domain.DenominacionPersona) error { return nil }) == nil {
		t.Fatal("retenida_revocada_abierta")
	}
	m.Busqueda.Revocada = true
	guardarMetadatosDenominacion(t, ruta, m)
	if _, e = p.PrepararDenominacionPersona(context.Background(), prep.PersonaRef, 0, prep.ProcedenciaRef, "ambito:admin", indice, nombre); e == nil {
		t.Fatal("busqueda_retirada_admitida")
	}
}
func TestKMSDenominacionNormaModificadaCierraProtectorAnterior(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "cfg.json")
	m := metadatosDenominacionPrueba()
	guardarMetadatosDenominacion(t, ruta, m)
	f, e := NuevaFuenteConfiguracionPrivadaDenominacionPersona(ruta)
	if e != nil {
		t.Fatal("fuente")
	}
	var k [32]byte
	k[0] = 1
	p, indice, e := NuevoProtectorDenominacionPersonaDesarrollo(k, f, time.Now)
	if e != nil {
		t.Fatal("protector")
	}
	m.Norma.Case = "exact"
	guardarMetadatosDenominacion(t, ruta, m)
	if _, e = p.PrepararDenominacionPersona(context.Background(), "per_aaaaaaaaaaaaaaaaaaaaaaaa", 0, "procedencia:sintetica", "ambito:admin", indice, []byte("Elena Márquez")); e == nil {
		t.Fatal("norma_sustituida")
	}
}
