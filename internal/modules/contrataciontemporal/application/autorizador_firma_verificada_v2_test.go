package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type planBindingFirmaPrueba struct {
	descriptor ports.DescriptorConstructorFirmaV2
	llamadas   int
}

func (p *planBindingFirmaPrueba) DescriptorFirmaV2(context.Context, ports.MaterialFirmaVerificadaV2) (ports.DescriptorConstructorFirmaV2, error) {
	p.llamadas++
	return p.descriptor, nil
}

type emisorBindingFirmaPrueba struct {
	perfil  string
	ambitos ports.AmbitosOperadorFirmaV2
	visto   vd.RecursoAutorizable
	antes   func()
}

func (e *emisorBindingFirmaPrueba) ObtenerPerfilActivoOperadorFirmaV2(context.Context) (string, error) {
	return e.perfil, nil
}
func (e *emisorBindingFirmaPrueba) ObtenerAmbitosOperadorFirmaV2(context.Context) (ports.AmbitosOperadorFirmaV2, error) {
	return e.ambitos, nil
}
func (e *emisorBindingFirmaPrueba) AutorizarMaterialFirmaVerificadaV2(_ context.Context, m ports.MaterialFirmaVerificadaV2, r vd.RecursoAutorizable) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.visto = r
	if e.antes != nil {
		e.antes()
	}
	accion, audiencia := ports.AccionRegistrarFirmaVec, ports.AudienciaFirmaVecV2
	if m.Via == ports.ViaFirmaExternaPortafirmas {
		accion, audiencia = ports.AccionRegistrarFirmaExterna, ports.AudienciaFirmaExternaV2
	}
	return capacidadMultiplePrueba("binding", r, accion, audiencia)
}

func materialBindingFirmaPrueba(t *testing.T) ports.MaterialFirmaVerificadaV2 {
	t.Helper()
	s, d, _, o := prepararServicioMultipleConsumidor(t)
	if _, err := registrarPasoMultipleConsumidor(t, s, d, o, 1); err != nil {
		t.Fatal(err)
	}
	return d.materiales[0]
}

func TestAutorizadorFirmaV2ConservaDescriptorAnteriorAlPDP(t *testing.T) {
	m := materialBindingFirmaPrueba(t)
	p := &planBindingFirmaPrueba{descriptor: descriptorMultiplePrueba(m)}
	antes, err := CanonicoDescriptorFirmaVerificadaV2(m, p.descriptor)
	if err != nil {
		t.Fatal(err)
	}
	emisor := &emisorBindingFirmaPrueba{perfil: m.PerfilActivoOperadorRef, ambitos: ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef}}
	// Cambiar el plan durante PDP no puede sustituir lo que se autorizó.
	emisor.antes = func() { p.descriptor.Seleccion.CargoRef = "cargo:posterior" }
	a, err := NuevoAutorizadorNominalFirmaV2(p, emisor)
	if err != nil {
		t.Fatal(err)
	}
	c, err := a.AutorizarFirmaVerificadaV2(context.Background(), m)
	if err != nil || ValidarCapacidadFirmaVerificadaV2(c, m) != nil || p.llamadas != 1 {
		t.Fatalf("binding no acreditado: %v", err)
	}
	descriptor, h := c.ExportarDescriptorParaConsumidor()
	sha := sha256.Sum256(antes)
	if !bytes.Equal(antes, descriptor) || h != hex.EncodeToString(sha[:]) || emisor.visto.Atributos["descriptor_firma_sha256"] != h || len(emisor.visto.Atributos) != 2 {
		t.Fatal("descriptor resuelto de nuevo o no ligado al PDP")
	}
	descriptor[0] = 'X'
	segunda, _ := c.ExportarDescriptorParaConsumidor()
	if !bytes.Equal(segunda, antes) {
		t.Fatal("la exportación modifica la capacidad original")
	}
	legacy := ports.TransportarMaterialFirmaVerificadaV2(c.ExportarMaterialParaConsumidor())
	if !errors.Is(ValidarCapacidadFirmaVerificadaV2(legacy, m), ports.ErrFirmaDocumentoDenegada) {
		t.Fatal("capacidad legada escribió sin descriptor")
	}
	posterior, err := CanonicoDescriptorFirmaVerificadaV2(m, p.descriptor)
	if err != nil {
		t.Fatal(err)
	}
	shaPosterior := sha256.Sum256(posterior)
	alterada := ports.TransportarMaterialFirmaVerificadaV2ConDescriptor(c.ExportarMaterialParaConsumidor(), posterior, hex.EncodeToString(shaPosterior[:]))
	if !errors.Is(ValidarCapacidadFirmaVerificadaV2(alterada, m), ports.ErrFirmaDocumentoDenegada) {
		t.Fatal("el mismo PDP autorizó otro cargo nominal")
	}
}

func TestRecursoFirmaV2CanonCoincideConAD170(t *testing.T) {
	m := materialBindingFirmaPrueba(t)
	d, err := CanonicoDescriptorFirmaVerificadaV2(m, descriptorMultiplePrueba(m))
	if err != nil {
		t.Fatal(err)
	}
	r, err := RecursoFirmaVerificadaV2(m, d)
	if err != nil {
		t.Fatal(err)
	}
	md, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hd := sha256.Sum256(d)
	// AD170 reconstruye esta preimagen mínima, conservando el orden de maps Go.
	canon := []byte(`{"ambitos":{"organizacion_ref":"` + m.OrganizacionRef + `"},"atributos":{"descriptor_firma_sha256":"` + hex.EncodeToString(hd[:]) + `","material_sha256":"` + md + `"}}`)
	esperada := sha256.Sum256(canon)
	obtenida, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || obtenida != hex.EncodeToString(esperada[:]) {
		t.Fatalf("preimagen distinta del helper SQL: %v", err)
	}
}

func TestDescriptorFirmaV2RechazaClavesRepetidasAntesDelPDP(t *testing.T) {
	m := materialBindingFirmaPrueba(t)
	d, err := CanonicoDescriptorFirmaVerificadaV2(m, descriptorMultiplePrueba(m))
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(d, &campos) != nil {
		t.Fatal("fixture no es JSON")
	}
	repetido := append([]byte(`{"esquema":`+string(campos["esquema"])+`,`), d[1:]...)
	if !errors.Is(ValidarDescriptorFirmaVerificadaV2(m, repetido), ports.ErrFirmaDocumentoDenegada) {
		t.Fatal("claves repetidas alteran descriptor")
	}
}

func TestAutorizadorFirmaV2LlevaLaUnidadDeLaAsignacion(t *testing.T) {
	m := materialBindingFirmaPrueba(t)
	p := &planBindingFirmaPrueba{descriptor: descriptorMultiplePrueba(m)}
	emisor := &emisorBindingFirmaPrueba{perfil: m.PerfilActivoOperadorRef,
		ambitos: ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef}}
	a, err := NuevoAutorizadorNominalFirmaV2(p, emisor)
	if err != nil {
		t.Fatal(err)
	}
	c, err := a.AutorizarFirmaVerificadaV2(context.Background(), m)
	if err != nil || ValidarCapacidadFirmaVerificadaV2(c, m) != nil {
		t.Fatalf("capacidad con unidad denegada: %v", err)
	}
	if len(emisor.visto.Ambitos) != 2 || emisor.visto.Ambitos["unidad_ref"] != m.UnidadFirmanteRef || emisor.visto.Ambitos["organizacion_ref"] != m.OrganizacionRef {
		t.Fatalf("el PDP no recibió la unidad de la asignación: %v", emisor.visto.Ambitos)
	}
	// Otra unidad del firmante no puede reutilizar esa capacidad.
	otra := m
	otra.UnidadFirmanteRef = "unidad:otra"
	if !errors.Is(ValidarCapacidadFirmaVerificadaV2(c, otra), ports.ErrFirmaDocumentoDenegada) {
		t.Fatal("capacidad aceptada para otra unidad")
	}
	for nombre, ambitos := range map[string]ports.AmbitosOperadorFirmaV2{
		"unidad_distinta_del_paso": {OrganizacionRef: m.OrganizacionRef, UnidadRef: "unidad:otra"},
		"organizacion_distinta":    {OrganizacionRef: "organizacion:otra"},
	} {
		emisor.ambitos, emisor.visto = ambitos, vd.RecursoAutorizable{}
		if _, err := a.AutorizarFirmaVerificadaV2(context.Background(), m); err == nil || emisor.visto.Referencia != "" {
			t.Fatalf("%s: llegó al PDP", nombre)
		}
	}
}

func TestRecursoFirmaV2ExternaNoAdmiteUnidadTodavia(t *testing.T) {
	m := materialBindingFirmaPrueba(t)
	d, err := CanonicoDescriptorFirmaVerificadaV2(m, descriptorMultiplePrueba(m))
	if err != nil {
		t.Fatal(err)
	}
	m.Via = ports.ViaFirmaExternaPortafirmas
	if _, err := RecursoFirmaVerificadaV2ConAmbitos(m, d, ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef}); err == nil {
		t.Fatal("vía externa con unidad antes de su corte")
	}
}

func TestRecursoFirmaV2ConUnidadCoincideConAD206(t *testing.T) {
	m := materialBindingFirmaPrueba(t)
	d, err := CanonicoDescriptorFirmaVerificadaV2(m, descriptorMultiplePrueba(m))
	if err != nil {
		t.Fatal(err)
	}
	r, err := RecursoFirmaVerificadaV2ConAmbitos(m, d, ports.AmbitosOperadorFirmaV2{OrganizacionRef: m.OrganizacionRef, UnidadRef: m.UnidadFirmanteRef})
	if err != nil {
		t.Fatal(err)
	}
	md, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hd := sha256.Sum256(d)
	// AD206 (huella_recurso_firma_interior_ct_v1) reconstruye esta misma preimagen.
	canon := []byte(`{"ambitos":{"organizacion_ref":"` + m.OrganizacionRef + `","unidad_ref":"` + m.UnidadFirmanteRef + `"},"atributos":{"descriptor_firma_sha256":"` + hex.EncodeToString(hd[:]) + `","material_sha256":"` + md + `"}}`)
	esperada := sha256.Sum256(canon)
	obtenida, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || obtenida != hex.EncodeToString(esperada[:]) {
		t.Fatalf("preimagen distinta del helper SQL: %v", err)
	}
}
