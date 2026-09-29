package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type resolutorContextosProvisionCTPrueba struct {
	porPerfil map[string]dominiovec.ResultadoContextoActorRegistradoV2
	err       error
}

func (r resolutorContextosProvisionCTPrueba) ResolverContextoActorRegistradoV2(
	_ context.Context, s dominiovec.SolicitudContextoActor,
) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	if r.err != nil {
		return dominiovec.ResultadoContextoActorRegistradoV2{}, r.err
	}
	return r.porPerfil[s.PerfilActivoRef], nil
}

func manifiestoProvisionPerfilesCTPrueba() manifiestoProvisionPerfilesCT {
	huella := strings.Repeat("a", 64)
	return manifiestoProvisionPerfilesCT{
		Esquema:       esquemaManifiestoProvisionPerfilesCT,
		AprobacionRef: "demo:ct:perfiles:20260929",
		Perfiles: [2]entradaProvisionPerfilCT{
			{Clave: "alta", PerfilRef: "prf_alta_prueba", ContextoRef: "rca_contexto_alta_prueba",
				ContextoHuellaSHA256: huella, ObjetivoAsignacionRef: "asignacion:alta:v1", ObjetivoSHA256: huella},
			{Clave: "cobertura", PerfilRef: "prf_cobertura_prueba", ContextoRef: "rca_contexto_cobertura_prueba",
				ContextoHuellaSHA256: huella, PreimagenAsignacionRef: "asignacion:cobertura:v1",
				PreimagenSHA256: huella, ObjetivoAsignacionRef: "asignacion:cobertura:v2", ObjetivoSHA256: huella},
		},
	}
}

func TestManifiestoPerfilesCTCotejaArchivoAprobadoYPreimagen(t *testing.T) {
	m := manifiestoProvisionPerfilesCTPrueba()
	contenido, huella, err := manifiestoCanonicoProvisionPerfilesCT(m)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "perfiles.json")
	if err := os.WriteFile(ruta, contenido, 0600); err != nil {
		t.Fatal(err)
	}
	leido, obtenido, err := leerManifiestoProvisionPerfilesCT(ruta, huella, m.AprobacionRef)
	if err != nil || obtenido != huella || leido.Perfiles[1].PreimagenAsignacionRef != m.Perfiles[1].PreimagenAsignacionRef {
		t.Fatalf("manifiesto aprobado no recuperado: %v", err)
	}
	if _, _, err := leerManifiestoProvisionPerfilesCT(ruta, strings.Repeat("b", 64), m.AprobacionRef); !errors.Is(err, errProvisionPerfilesCTObsoleta) {
		t.Fatalf("huella cambiada aceptada: %v", err)
	}
	if _, _, err := leerManifiestoProvisionPerfilesCT(ruta, huella, "demo:otra"); !errors.Is(err, errProvisionPerfilesCTEntrada) {
		t.Fatalf("aprobación ajena aceptada: %v", err)
	}
	enlace := filepath.Join(t.TempDir(), "enlace.json")
	if err := os.Symlink(ruta, enlace); err != nil {
		t.Fatal(err)
	}
	if _, _, err := leerManifiestoProvisionPerfilesCT(enlace, huella, m.AprobacionRef); !errors.Is(err, errProvisionPerfilesCTEntrada) {
		t.Fatalf("enlace aceptado: %v", err)
	}
	if err := os.Chmod(ruta, 0664); err != nil {
		t.Fatal(err)
	}
	if _, _, err := leerManifiestoProvisionPerfilesCT(ruta, huella, m.AprobacionRef); !errors.Is(err, errProvisionPerfilesCTEntrada) {
		t.Fatalf("archivo modificable por grupo aceptado: %v", err)
	}
}

func TestManifiestoPerfilesCTRechazaJSONNoCanonico(t *testing.T) {
	m := manifiestoProvisionPerfilesCTPrueba()
	canon, _, err := manifiestoCanonicoProvisionPerfilesCT(m)
	if err != nil {
		t.Fatal(err)
	}
	contenido := append(append([]byte(nil), canon...), '\n')
	ruta := filepath.Join(t.TempDir(), "perfiles.json")
	if err := os.WriteFile(ruta, contenido, 0600); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(contenido)
	if _, _, err := leerManifiestoProvisionPerfilesCT(ruta, hex.EncodeToString(h[:]), m.AprobacionRef); !errors.Is(err, errProvisionPerfilesCTEntrada) {
		t.Fatalf("JSON no canónico aceptado: %v", err)
	}
}

func TestPreimagenPerfilesCTDeniegaRevocacionYRestriccion(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	semilla := soporte.instantaneaCobertura
	ahora := semilla.AsignacionPerfil.VigenteDesde.Add(time.Minute)
	if err := validarPreimagenProvisionPerfilCT(semilla, semilla, ahora); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(*dominiovec.InstantaneaAutorizacion)
	}{
		{"revocada", func(i *dominiovec.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
			i.AsignacionPerfil.RevocadaPor = "autoridad:prueba"
			i.AsignacionPerfil.RevocadaEn = ahora
			i.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
		}},
		{"ambito_restringido", func(i *dominiovec.InstantaneaAutorizacion) {
			i.AsignacionPerfil.Ambitos[0].Valores = []string{"organizacion:otra"}
		}},
		{"control_retirado", func(i *dominiovec.InstantaneaAutorizacion) {
			i.ControlVigenciaVersionRol.Estado = dominiovec.EstadoControlVigenciaVersionRolRetirada
			i.ControlVigenciaVersionRol.ActoRef = "acto:prueba"
			i.ControlVigenciaVersionRol.MotivoCodigo = "motivo:prueba"
		}},
		{"rol_ajeno", func(i *dominiovec.InstantaneaAutorizacion) {
			i.VersionRol.RolID = "rol:ajeno"
			i.AsignacionPerfil.VersionRolRef = i.VersionRol.Referencia()
			i.ControlVigenciaVersionRol.VersionRolRef = i.VersionRol.Referencia()
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			actual := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
			caso.mutar(&actual)
			if err := validarPreimagenProvisionPerfilCT(actual, semilla, ahora); !errors.Is(err, errProvisionPerfilesCTObsoleta) {
				t.Fatalf("preimagen insegura aceptada: %v", err)
			}
		})
	}
}

func TestActoProvisionPerfilesCTLigaManifiestoYPerfil(t *testing.T) {
	primero := actoAsignacionProvisionCT(strings.Repeat("a", 64), "alta")
	if primero == actoAsignacionProvisionCT(strings.Repeat("a", 64), "cobertura") ||
		primero == actoAsignacionProvisionCT(strings.Repeat("b", 64), "alta") ||
		primero != actoAsignacionProvisionCT(strings.Repeat("a", 64), "alta") || len(primero) > 512 {
		t.Fatal("acto no liga el manifiesto y el perfil")
	}
}

func TestHuellaProvisionPerfilesCTNormalizaCatalogoVacio(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	a := soporte.instantaneaCobertura
	b := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(a)
	b.Politicas = []dominiovec.PoliticaRestrictiva{}
	ha, errA := huellaJSONProvisionPerfilesCT(a)
	hb, errB := huellaJSONProvisionPerfilesCT(b)
	if errA != nil || errB != nil || ha != hb {
		t.Fatalf("catálogo vacío cambió la huella: %v/%v", errA, errB)
	}
}

func TestSolicitudProvisionPerfilesCTCierraSinModoValido(t *testing.T) {
	_, err := EjecutarProvisionPerfilesCT(context.Background(), config.Config{}, SolicitudProvisionPerfilesCT{})
	if !errors.Is(err, errProvisionPerfilesCTEntrada) {
		t.Fatalf("solicitud sin modo aceptada: %v", err)
	}
}

func TestAplicarProvisionPerfilesCTRevalidaContextoCambiado(t *testing.T) {
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	alta, err := nuevoContextoAltaFijoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	cobertura, err := nuevoContextoCoberturaContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	perfiles := [2]perfilPreparadoProvisionCT{
		{clave: "alta", contexto: alta, registrado: alta.Resultado},
		{clave: "cobertura", contexto: cobertura, registrado: cobertura.Resultado},
	}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		principalID:       principal.ID,
		certificadoSHA256: principal.Attributes["certificate_sha256"],
	}
	resolver := resolutorContextosProvisionCTPrueba{porPerfil: map[string]dominiovec.ResultadoContextoActorRegistradoV2{
		alta.Resultado.Contexto.PerfilActivoRef:      alta.Resultado,
		cobertura.Resultado.Contexto.PerfilActivoRef: cobertura.Resultado,
	}}
	if err := cotejarContextosActualesProvisionCT(context.Background(), resolver, soporte, &perfiles); err != nil {
		t.Fatalf("contextos vigentes denegados: %v", err)
	}
	resolver.porPerfil[alta.Resultado.Contexto.PerfilActivoRef] = cobertura.Resultado
	if err := cotejarContextosActualesProvisionCT(context.Background(), resolver, soporte, &perfiles); !errors.Is(err, errProvisionPerfilesCTObsoleta) {
		t.Fatalf("contexto sustituido aceptado: %v", err)
	}
	resolver.porPerfil[alta.Resultado.Contexto.PerfilActivoRef] = alta.Resultado
	resolver.err = errors.New("contexto revocado")
	if err := cotejarContextosActualesProvisionCT(context.Background(), resolver, soporte, &perfiles); !errors.Is(err, errProvisionPerfilesCTObsoleta) {
		t.Fatalf("contexto revocado aceptado: %v", err)
	}
}

func TestPrepararPerfilesCTSigueSoloLecturaSinPublicador(t *testing.T) {
	_, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	contexto, err := nuevoContextoAltaFijoContratacionTemporalDesarrollo(principal, ahora)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	semilla, err := nuevaInstantaneaAutorizacionAltaFijaContratacionTemporalDesarrollo(
		vinculo.PrincipalID, vinculo.PerfilActivoRef, ahora, nil)
	if err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteDosPerfilesArranqueCTPrueba{porPerfil: make(map[string]dominiovec.InstantaneaAutorizacion)}
	conexiones := conexionesProvisionPerfilesCT{almacen: fuente}
	p := perfilPreparadoProvisionCT{clave: "alta", contexto: contexto, registrado: contexto.Resultado, semilla: semilla}
	if err := prepararPerfilProvisionCT(context.Background(), conexiones, &p, ahora, true); err != nil ||
		p.existe || p.replay || p.objetivo.AsignacionPerfil.Referencia() != semilla.AsignacionPerfil.Referencia() {
		t.Fatalf("ausencia no preparó objetivo v1 sin escritura: %v", err)
	}
	fuente.porPerfil[vinculo.PerfilActivoRef] = semilla
	p = perfilPreparadoProvisionCT{clave: "alta", contexto: contexto, registrado: contexto.Resultado, semilla: semilla}
	if err := prepararPerfilProvisionCT(context.Background(), conexiones, &p, ahora, true); err != nil || !p.replay {
		t.Fatalf("publicada exacta no se reutilizó: %v", err)
	}
	revocada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	revocada.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
	revocada.AsignacionPerfil.RevocadaPor = "autoridad:prueba"
	revocada.AsignacionPerfil.RevocadaEn = ahora
	revocada.AsignacionPerfil.RevocacionRef = "revocacion:prueba"
	fuente.porPerfil[vinculo.PerfilActivoRef] = revocada
	p = perfilPreparadoProvisionCT{clave: "alta", contexto: contexto, registrado: contexto.Resultado, semilla: semilla}
	if err := prepararPerfilProvisionCT(context.Background(), conexiones, &p, ahora, true); !errors.Is(err, errProvisionPerfilesCTObsoleta) {
		t.Fatalf("revocada propuesta como alta: %v", err)
	}
}

func TestReplayProvisionCTExigeActoSiManifiestoTeniaOtraPreimagen(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	actual := soporte.instantaneaCobertura
	p := perfilPreparadoProvisionCT{
		clave: "cobertura", contexto: soporte.contextoCobertura,
		registrado: soporte.contextoCobertura.Resultado,
		semilla:    actual, preimagen: actual, existe: true,
		objetivo: actual, replay: true,
	}
	// El fixture principal de cobertura puede omitir el segundo contexto;
	// sólo la comparación de referencias/huellas participa aquí.
	h, err := huellaJSONProvisionPerfilesCT(actual)
	if err != nil {
		t.Fatal(err)
	}
	e := entradaProvisionPerfilCT{
		PreimagenAsignacionRef: actual.AsignacionPerfil.Referencia(), PreimagenSHA256: h,
		ObjetivoAsignacionRef: actual.AsignacionPerfil.Referencia(), ObjetivoSHA256: h,
	}
	if !replaySinCambioProvisionCT(p, e) {
		t.Fatal("no-op exacto denegado")
	}
	e.PreimagenAsignacionRef, e.PreimagenSHA256 = "", ""
	if replaySinCambioProvisionCT(p, e) {
		t.Fatal("recuperación de provisión parcial tratada como no-op")
	}
	e.PreimagenAsignacionRef = "asignacion:anterior:v1"
	e.PreimagenSHA256 = strings.Repeat("b", 64)
	if replaySinCambioProvisionCT(p, e) {
		t.Fatal("preimagen ajena tratada como no-op")
	}
}

func TestProvisionCTNoEmiteReciboPositivoTrasRevocacionEntrePerfiles(t *testing.T) {
	publicaciones, comprobaciones := 0, 0
	resultado, err := publicarYRevalidarPerfilesCT(ResultadoProvisionPerfilesCT{
		Estado: "confirmada", ManifiestoSHA256: strings.Repeat("a", 64),
		AprobacionRef: "demo:ct:perfiles:20260929",
	}, 2,
		func(i int) (ReciboPerfilProvisionCT, error) {
			publicaciones++
			return ReciboPerfilProvisionCT{Clave: []string{"alta", "cobertura"}[i],
				AsignacionRef: "asignacion:confirmada:v1"}, nil
		},
		func() error {
			comprobaciones++
			return errors.New("contexto revocado")
		})
	if !errors.Is(err, ErrProvisionPerfilesCTIncidenciaContexto) ||
		resultado.Estado != "incidencia" || len(resultado.Perfiles) != 1 ||
		publicaciones != 1 || comprobaciones != 1 {
		t.Fatalf("revocación entregó recibo positivo o publicó segundo perfil: %v/%+v/%d/%d",
			err, resultado, publicaciones, comprobaciones)
	}
}

func TestProvisionCTRevalidaDespuesDeCadaPublicacion(t *testing.T) {
	publicaciones, comprobaciones := 0, 0
	resultado, err := publicarYRevalidarPerfilesCT(ResultadoProvisionPerfilesCT{Estado: "confirmada"}, 2,
		func(i int) (ReciboPerfilProvisionCT, error) {
			publicaciones++
			return ReciboPerfilProvisionCT{Clave: []string{"alta", "cobertura"}[i]}, nil
		},
		func() error {
			comprobaciones++
			if comprobaciones == 2 {
				return errors.New("contexto cambiado")
			}
			return nil
		})
	if !errors.Is(err, ErrProvisionPerfilesCTIncidenciaContexto) ||
		resultado.Estado != "incidencia" || len(resultado.Perfiles) != 2 ||
		publicaciones != 2 || comprobaciones != 2 {
		t.Fatalf("segunda revocación no se detectó: %v/%+v/%d/%d", err, resultado, publicaciones, comprobaciones)
	}
}
