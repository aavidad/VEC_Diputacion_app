package plantillascatalogo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecpruebas "vec-diputacion-granada/internal/vec/pruebas"
)

var fechaPrueba = time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)

type relojPrueba struct{}

func (relojPrueba) Ahora() time.Time { return fechaPrueba }

type repoPrueba struct {
	lectura  Lectura
	material MaterialCambio
	llamadas int
}

func (r *repoPrueba) ComprobarAccion(_ context.Context, _ vecdomain.ContextoActor, accion string, _ Lectura) (bool, error) {
	return accion == "editar" || accion == "publicar", nil
}

func (r *repoPrueba) Consultar(context.Context, vecdomain.ContextoActor) (Lectura, error) {
	return r.lectura, nil
}
func (r *repoPrueba) Cambiar(_ context.Context, _ vecdomain.ContextoActor, m MaterialCambio) (ResultadoCambio, error) {
	r.material = m
	r.llamadas++
	return ResultadoCambio{Recibo: Recibo{EstadoReplay: "registrado"}}, nil
}
func referencia(prefijo, semilla string) string {
	h := sha256.Sum256([]byte(semilla))
	return prefijo + hex.EncodeToString(h[:16])
}
func actorPrueba(t *testing.T, semilla string) vecdomain.ContextoActor {
	t.Helper()
	a, _, err := vecpruebas.NuevoContextoYVinculo(fechaPrueba, referencia("per_", semilla), referencia("prf_", semilla), vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func basePrueba(t *testing.T) vecdomain.CatalogoConfigurable {
	t.Helper()
	base := vecdomain.CatalogoConfigurable{ID: CatalogoID, Version: 1, Revision: 1, ModuloID: ModuloID, Nombre: "Plantillas RRHH", FuenteRef: "fuente:rrhh:ejemplo", MotivoCreacion: "Origen aprobado para pruebas", Estado: vecdomain.EstadoCatalogoBorrador, CreadoPor: referencia("per_", "creador"), CreadoEn: fechaPrueba.Add(-48 * time.Hour), Entradas: []vecdomain.EntradaCatalogoConfigurable{{Clave: "informe_definitivo", Etiqueta: "Informe", Orden: 1, VigenteDesde: fechaPrueba.Add(-72 * time.Hour), Atributos: map[string]string{"titulo": "Informe", "parrafo.01": "Texto"}}}}
	publicado, err := base.Publicar(referencia("per_", "publicador"), "aprobacion:rrhh:v1", "Publicación inicial", fechaPrueba.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return publicado
}
func servicioPrueba(t *testing.T, r *repoPrueba) (*Servicio, vecdomain.ContextoActor) {
	t.Helper()
	b := basePrueba(t)
	s, err := NuevoServicio(r, relojPrueba{}, func(c vecdomain.CatalogoConfigurable) error { return c.Validar() }, &b)
	if err != nil {
		t.Fatal(err)
	}
	return s, actorPrueba(t, "editor")
}
func solicitudPrueba() SolicitudEditar {
	return SolicitudEditar{ClaveIdempotencia: "11111111-1111-4111-8111-111111111111", VersionEsperada: 1, RevisionEsperada: 0, Motivo: "Nueva plantilla acordada por RRHH", FuenteRef: "fuente:rrhh:v2", Entrada: vecdomain.EntradaCatalogoConfigurable{Clave: "documento_nuevo", Etiqueta: "Documento nuevo", Orden: 12, VigenteDesde: fechaPrueba.Add(-time.Hour), Atributos: map[string]string{"titulo": "Nuevo", "parrafo.01": "Contenido"}}}
}

func TestPrimeraEdicionPreservaBasePublicadaEnMaterialDurable(t *testing.T) {
	r := &repoPrueba{}
	s, a := servicioPrueba(t, r)
	peticion := solicitudPrueba()
	peticion.Entrada.VigenteDesde = peticion.Entrada.VigenteDesde.In(time.FixedZone("Madrid", 2*3600))
	if _, err := s.Editar(context.Background(), a, peticion); err != nil {
		t.Fatal(err)
	}
	m := r.material
	if r.llamadas != 1 || m.CatalogoBase == nil || m.Catalogo == nil || m.Catalogo.Version != 2 || m.Catalogo.Revision != 1 || m.Catalogo.Estado != vecdomain.EstadoCatalogoBorrador || len(m.Catalogo.Entradas) != 2 || m.CatalogoBase.Estado != vecdomain.EstadoCatalogoPublicado {
		t.Fatalf("material incompleto: %+v", m)
	}
	if h, _ := m.CatalogoBase.HuellaSHA256(); h != m.BaseHuellaSHA256 {
		t.Fatal("base no ligada a su huella")
	}
	if m.Catalogo.Entradas[0].Clave != "informe_definitivo" {
		t.Fatal("se perdió la plantilla preexistente")
	}
	var canonica SolicitudEditar
	if err := json.Unmarshal(m.Solicitud, &canonica); err != nil || !canonica.Entrada.VigenteDesde.Equal(peticion.Entrada.VigenteDesde) || canonica.Entrada.VigenteDesde.Location() != time.UTC {
		t.Fatalf("fecha de solicitud no canónica: %+v, %v", canonica.Entrada.VigenteDesde, err)
	}
}

func TestPublicarExigePersonaDistintaDeEditor(t *testing.T) {
	r := &repoPrueba{}
	s, a := servicioPrueba(t, r)
	actual := basePrueba(t)
	borrador, err := actual.NuevaVersion(2, a.Principal.ID, "fuente:rrhh:v2", "Edición RRHH", fechaPrueba.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	r.lectura = Lectura{Borrador: &borrador, Publicado: &actual}
	proyeccion, err := s.Consultar(context.Background(), a)
	if err != nil || !proyeccion.PuedeEditar || proyeccion.PuedePublicar {
		t.Fatalf("proyección de separación de funciones: %+v, %v", proyeccion, err)
	}
	_, err = s.Publicar(context.Background(), a, SolicitudPublicar{ClaveIdempotencia: "22222222-2222-4222-8222-222222222222", VersionEsperada: 2, RevisionEsperada: 1, Motivo: "Aprobación", AprobacionRef: "aprobacion:rrhh:v2"})
	if !errors.Is(err, ErrEntradaInvalida) || r.llamadas != 0 {
		t.Fatalf("publicación propia aceptada: %v", err)
	}
}

func TestConflictoDeVersionEntregaSolicitudParaRecuperarClave(t *testing.T) {
	r := &repoPrueba{}
	s, a := servicioPrueba(t, r)
	peticion := solicitudPrueba()
	peticion.VersionEsperada = 99
	if _, err := s.Editar(context.Background(), a, peticion); err != nil {
		t.Fatal(err)
	}
	if r.llamadas != 1 || r.material.Catalogo != nil || len(r.material.Solicitud) == 0 {
		t.Fatalf("no delegó replay a SQL: %+v", r.material)
	}
}
