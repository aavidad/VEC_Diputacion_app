package autorizacion

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

const (
	personaLecturaV3 = "per_lectura0123456789abcde"
	perfilLecturaV3  = "prf_lectura0123456789abcde"
)

// emisorLecturaV3Prueba obtiene concesiones V3 reales para la solicitud que
// recibe; alterar permite simular un PDP que evalúa otra cosa.
type emisorLecturaV3Prueba struct {
	ahora       time.Time
	persona     string
	perfil      string
	accion      string // concedida por el rol; vacía: la pedida
	alterar     func(*pruebas.DatosConcesionV3Prueba)
	errSeud     error
	llamadas    int
	peticion    SolicitudConcesionAlmacenV3
	sinRegistro bool
}

func (e *emisorLecturaV3Prueba) SeudonimosLecturaOriginal(context.Context, docports.AutorizacionV3) (DatosSeudonimosLectura, error) {
	return DatosSeudonimosLectura{
		SujetoHMAC:    "hmac-sha256:sujeto_v1:" + strings.Repeat("a", 64),
		SolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("b", 64),
	}, e.errSeud
}

func (e *emisorLecturaV3Prueba) EmitirConcesionAlmacenV3(_ context.Context, s SolicitudConcesionAlmacenV3) (ConcesionAlmacenV3, error) {
	e.llamadas++
	e.peticion = s
	datos := pruebas.DatosConcesionV3Prueba{
		Instante: e.ahora, PersonaRef: e.persona, PerfilRef: e.perfil, Accion: s.Accion, AccionConcedida: e.accion,
		Recurso: s.Recurso, Finalidad: s.Finalidad, Campos: []string{"contenido", "documento"},
		DecisionRef: "dec_lectura56789abcdef0123456789abcdef",
	}
	if e.alterar != nil {
		e.alterar(&datos)
	}
	c, err := pruebas.NuevaConcesionV3Prueba(datos)
	if err != nil {
		return ConcesionAlmacenV3{}, err
	}
	salida := ConcesionAlmacenV3{Solicitud: c.Solicitud, Decision: c.Decision, Confirmacion: c.Confirmacion}
	if e.sinRegistro {
		salida.Confirmacion = vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	}
	return salida, nil
}

// actorLecturaV3 devuelve principal y perfil que produce el vínculo de prueba.
func actorLecturaV3(t *testing.T, ahora time.Time) (string, string) {
	t.Helper()
	_, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora, personaLecturaV3, perfilLecturaV3,
		vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	return datos.PrincipalID, datos.PerfilActivoRef
}

func materialDescargaPrueba(t *testing.T, d domain.Documento, ahora time.Time) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	preimagen, err := (docports.ConsultaDocumento{DocumentoID: d.ID, Version: d.Version}).PreimagenDescargar()
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:descarga:prueba", h, h,
		"contexto:prueba", h, docports.AccionDescargar, d.ID, docports.HuellaEfectoV3(preimagen), docports.AudienciaV3,
		ahora, ahora.Add(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	publica, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte("c"), vecports.TamanoMinimoCapacidadCanonicaV3), resumen,
		[]byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("p"), []byte("s"), []byte("e"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

type escenarioLecturaV3 struct {
	ahora   time.Time
	d       domain.Documento
	a       docports.AutorizacionV3
	emisor  *emisorLecturaV3Prueba
	fabrica *FabricaContextoLecturaOriginalV3
}

func nuevoEscenarioLecturaV3(t *testing.T) escenarioLecturaV3 {
	t.Helper()
	ahora := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	d := documentoValidoPrueba(ahora)
	principal, perfil := actorLecturaV3(t, ahora)
	a := docports.AutorizacionV3{
		Material: materialDescargaPrueba(t, d, ahora), Accion: docports.AccionDescargar,
		Finalidad: "descargar_documento_original", RecursoRef: d.ID, AmbitoRef: d.ExpedienteRef,
		PrincipalID: principal, PerfilActivoRef: perfil, CorrelacionRef: "correlacion:descarga:0001",
	}
	emisor := &emisorLecturaV3Prueba{ahora: ahora, persona: personaLecturaV3, perfil: perfilLecturaV3}
	// La lectura se usa después del registro (un segundo tras emitirla).
	fabrica, err := NuevaFabricaContextoLecturaOriginalV3(emisor, relojFijo{ahora.Add(2 * time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	return escenarioLecturaV3{ahora: ahora, d: d, a: a, emisor: emisor, fabrica: fabrica}
}

func TestLecturaV3AbreSoloElObjetoExactoConConcesionRegistrada(t *testing.T) {
	e := nuevoEscenarioLecturaV3(t)
	contexto, err := e.fabrica.ContextoLecturaOriginal(context.Background(), e.d, e.a)
	if err != nil {
		t.Fatalf("lectura V3: %v", err)
	}
	instante := e.ahora.Add(2 * time.Second)
	if contexto.ValidarParaEn(vecports.AccionAlmacenLeer, instante) != nil ||
		contexto.ValidarParaEn(vecports.AccionAlmacenEscribir, instante) == nil {
		t.Fatal("la lectura V3 debe permitir solo leer")
	}
	proyeccion, err := contexto.Proyeccion()
	if err != nil || proyeccion.ObjetoVinculado != (vecports.ReferenciaObjetoAlmacen{Referencia: e.d.ObjetoRef, Version: e.d.ObjetoVersion}) ||
		proyeccion.AccionNegocio != vecports.AccionNegocioLeerOriginalDocumentoGenerado || proyeccion.RecursoRef != e.d.ID {
		t.Fatalf("proyección: %+v %v", proyeccion, err)
	}
	p := e.emisor.peticion
	if p.Accion != vecports.AccionNegocioLeerOriginalDocumentoGenerado || p.Finalidad != e.a.Finalidad ||
		p.Recurso.ModuloID != "documentos" || p.Recurso.Tipo != "documento_original" ||
		p.Recurso.Ambitos["organizacion_ref"] != docports.OrganizacionRefV3 || len(p.Recurso.Ambitos) != 1 ||
		p.Recurso.Atributos[AtributoCorrelacionDescarga] != e.a.CorrelacionRef {
		t.Fatalf("petición al PDP inesperada: %+v", p)
	}
}

func TestLecturaV3NoAbreSinConcesionValida(t *testing.T) {
	casos := map[string]func(*escenarioLecturaV3){
		"sin registro": func(e *escenarioLecturaV3) { e.emisor.sinRegistro = true },
		"denegada":     func(e *escenarioLecturaV3) { e.emisor.accion = docports.AccionListar },
		"otro actor":   func(e *escenarioLecturaV3) { e.emisor.persona = "per_otro000123456789abcdef" },
		"otro perfil":  func(e *escenarioLecturaV3) { e.emisor.perfil = "prf_otro000123456789abcdef" },
		"otra finalidad": func(e *escenarioLecturaV3) {
			e.emisor.alterar = func(d *pruebas.DatosConcesionV3Prueba) { d.Finalidad = "otra" }
		},
		"otra acción": func(e *escenarioLecturaV3) {
			e.emisor.alterar = func(d *pruebas.DatosConcesionV3Prueba) { d.Accion = docports.AccionListar }
		},
		"campos de más": func(e *escenarioLecturaV3) {
			e.emisor.alterar = func(d *pruebas.DatosConcesionV3Prueba) { d.Campos = []string{"contenido", "documento", "recibo"} }
		},
		"con obligación": func(e *escenarioLecturaV3) {
			e.emisor.alterar = func(d *pruebas.DatosConcesionV3Prueba) { d.Obligaciones = []string{"revisar"} }
		},
		"otro recurso": func(e *escenarioLecturaV3) {
			e.emisor.alterar = func(d *pruebas.DatosConcesionV3Prueba) {
				r := d.Recurso
				r.Atributos = map[string]string{}
				for k, v := range d.Recurso.Atributos {
					r.Atributos[k] = v
				}
				r.Atributos[vecports.AtributoAlmacenObjetoVersion] = "objeto:version:otra"
				d.Recurso = r
			}
		},
		"caducada": func(e *escenarioLecturaV3) {
			f, err := NuevaFabricaContextoLecturaOriginalV3(e.emisor, relojFijo{e.ahora.Add(2 * time.Minute)})
			if err != nil {
				panic(err)
			}
			e.fabrica = f
		},
		"sin seudónimos": func(e *escenarioLecturaV3) { e.emisor.errSeud = errors.New("seudonimización caída") },
	}
	for nombre, alterar := range casos {
		e := nuevoEscenarioLecturaV3(t)
		alterar(&e)
		if _, err := e.fabrica.ContextoLecturaOriginal(context.Background(), e.d, e.a); !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) {
			t.Errorf("%s: el almacén se abrió: %v", nombre, err)
		}
	}
}

func TestLecturaV3NoPideConcesionSinDescargaLigada(t *testing.T) {
	e := nuevoEscenarioLecturaV3(t)
	otro := e.d
	otro.Version = 4 // la autorización de descarga es de la versión 3
	casos := map[string]func() (domain.Documento, docports.AutorizacionV3){
		"otra versión": func() (domain.Documento, docports.AutorizacionV3) { return otro, e.a },
		"otro expediente": func() (domain.Documento, docports.AutorizacionV3) {
			a := e.a
			a.AmbitoRef = "ref:" + strings.Repeat("9", 64)
			return e.d, a
		},
		"sin material": func() (domain.Documento, docports.AutorizacionV3) {
			a := e.a
			a.Material = vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
			return e.d, a
		},
		"no descargable": func() (domain.Documento, docports.AutorizacionV3) {
			d := e.d
			d.Custodia = domain.CustodiaExterna
			return d, e.a
		},
	}
	for nombre, caso := range casos {
		d, a := caso()
		if _, err := e.fabrica.ContextoLecturaOriginal(context.Background(), d, a); !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
	if e.emisor.llamadas != 0 {
		t.Fatalf("se pidió concesión al PDP sin descarga ligada: %d", e.emisor.llamadas)
	}
	if _, err := NuevaFabricaContextoLecturaOriginalV3(nil, relojFijo{e.ahora}); !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("sin emisor: %v", err)
	}
}
