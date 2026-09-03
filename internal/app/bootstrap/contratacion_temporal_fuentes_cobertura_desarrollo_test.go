package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type resolutorGobiernoCoberturaDesarrolloNoUsado struct{}

func (resolutorGobiernoCoberturaDesarrolloNoUsado) ResolverGobiernoOperacionCobertura(
	context.Context,
	cobertura.SolicitudResolucionGobiernoOperacionCobertura,
) (cobertura.PublicacionGobiernoOperacionCobertura, error) {
	return cobertura.PublicacionGobiernoOperacionCobertura{},
		errors.New("no usado por esta prueba")
}

func TestFuentesCoberturaDesarrolloFirmanVerificanYSeRecuperanTrasReinicio(
	t *testing.T,
) {
	primera := nuevasDependenciasFuentesCoberturaPrueba(t)
	identidadesPrimera := autenticarFuentesCoberturaDesarrolloPrueba(t, primera)
	solicitud := solicitudFuenteCoberturaDesarrolloPrueba(t)
	resultado, err := primera.fuente.ConsultarCobertura(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatalf("consultar fuente firmada: %v", err)
	}
	datos, err := resultado.Datos()
	if err != nil || datos.Comprobacion.Resultado != domain.ComprobacionAfirmativa ||
		datos.Comprobacion.FuenteRef != autoridadFuenteCoberturaDesarrollo {
		t.Fatalf("resultado sintético inesperado: %#v, %v", datos, err)
	}
	peticionVerificacion, err := resultado.SolicitudVerificacion()
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := primera.verificador.VerificarRespuestaCobertura(
		context.Background(),
		peticionVerificacion,
	)
	if err != nil {
		t.Fatalf("verificar respuesta firmada: %v", err)
	}
	identidadVerificador := identidadesPrimera[ports.RolVerificadorCobertura]
	if confirmacion.ValidarPara(
		peticionVerificacion,
		time.Now().UTC().Truncate(time.Microsecond),
		identidadVerificador.ClavePruebaEd25519(),
	) != nil {
		t.Fatal("la confirmación Ed25519 no quedó vinculada al verificador")
	}
	referencia, err := primera.referencias.NuevaReferenciaComprobacionCobertura(
		context.Background(),
	)
	if err != nil || !strings.HasPrefix(referencia, "peticion:ct-cobertura:") {
		t.Fatalf("referencia de cobertura inválida: %q, %v", referencia, err)
	}

	primera.cerrar()
	primera.cerrar()
	assertSecretosFuentesCoberturaBorrados(t, primera)

	segunda := nuevasDependenciasFuentesCoberturaPrueba(t)
	t.Cleanup(segunda.cerrar)
	identidadesSegunda := autenticarFuentesCoberturaDesarrolloPrueba(t, segunda)
	for rol, identidadPrimera := range identidadesPrimera {
		if !ports.IdentidadesAutoridadFuenteAnalisisIguales(
			identidadPrimera,
			identidadesSegunda[rol],
		) {
			t.Fatalf("la identidad %q cambió tras reiniciar", rol)
		}
	}
}

func TestFuenteCoberturaDesarrolloDeniegaCoordenadasNoDeclaradas(t *testing.T) {
	dependencias := nuevasDependenciasFuentesCoberturaPrueba(t)
	t.Cleanup(dependencias.cerrar)
	casos := map[string]func(*ports.SolicitudConsultarCobertura){
		"categoría": func(s *ports.SolicitudConsultarCobertura) {
			s.CategoriaRef = "categoria:desarrollo:desconocida"
		},
		"orden de comprobación": func(s *ports.SolicitudConsultarCobertura) {
			s.Comprobacion.Orden = 2
		},
		"obligatoriedad": func(s *ports.SolicitudConsultarCobertura) {
			s.Comprobacion.Obligatoria = false
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			solicitud := solicitudFuenteCoberturaDesarrolloPrueba(t)
			alterar(&solicitud)
			if _, err := dependencias.fuente.ConsultarCobertura(
				context.Background(),
				solicitud,
			); !errors.Is(err, ports.ErrPeticionFuenteCoberturaInvalida) {
				t.Fatalf("coordenada desconocida aceptada: %v", err)
			}
		})
	}
}

func nuevasDependenciasFuentesCoberturaPrueba(
	t *testing.T,
) dependenciasFuentesCoberturaDesarrollo {
	t.Helper()
	derivador := nuevoDerivadorIdempotenciaPrueba(t, 2, 1)
	dependencias, err := nuevasDependenciasFuentesCoberturaDesarrollo(
		derivador,
		relojContratacionTemporalDesarrollo{},
		resolutorGobiernoCoberturaDesarrolloNoUsado{},
	)
	if err != nil {
		t.Fatalf("componer fuentes de cobertura: %v", err)
	}
	if dependencias.fuente == nil || dependencias.verificador == nil ||
		dependencias.publicador == nil || dependencias.autenticador == nil ||
		dependencias.referencias == nil || dependencias.cerrar == nil {
		dependencias.cerrar()
		t.Fatal("la composición aceptó una dependencia nula")
	}
	return dependencias
}

func autenticarFuentesCoberturaDesarrolloPrueba(
	t *testing.T,
	dependencias dependenciasFuentesCoberturaDesarrollo,
) map[ports.RolAutoridadFuenteAnalisis]ports.IdentidadAutoridadFuenteAnalisis {
	t.Helper()
	presentadores := map[ports.RolAutoridadFuenteAnalisis]ports.PresentadorAutoridadFuenteAnalisis{
		ports.RolFuenteCobertura:             dependencias.fuente,
		ports.RolVerificadorCobertura:        dependencias.verificador,
		ports.RolPublicadorCatalogoCobertura: dependencias.publicador,
	}
	identidades := make(
		map[ports.RolAutoridadFuenteAnalisis]ports.IdentidadAutoridadFuenteAnalisis,
		len(presentadores),
	)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	for rol, presentador := range presentadores {
		desafio, err := ports.NuevoDesafioAutoridadFuenteAnalisis(
			[]byte("peticion:prueba:fuentes-cobertura"),
			dependencias.autenticador.OrganizacionAutoridadFuenteAnalisis(),
			dependencias.autenticador.AudienciaAutoridadFuenteAnalisis(),
			rol,
		)
		if err != nil {
			t.Fatal(err)
		}
		presentacion, err := presentador.PresentarAutoridadFuenteAnalisis(
			context.Background(),
			desafio,
		)
		if err != nil {
			t.Fatal(err)
		}
		evidencia, err := ports.NuevaEvidenciaPublicaAutoridadFuenteAnalisis(
			desafio,
			presentacion,
			rol,
			ahora,
		)
		if err != nil {
			t.Fatal(err)
		}
		identidad, err := dependencias.autenticador.
			VerificarEvidenciaPublicaAutoridadFuenteAnalisis(evidencia)
		if err != nil {
			t.Fatalf("autenticar %q: %v", rol, err)
		}
		identidades[rol] = identidad
	}
	if !ports.AutoridadesFuenteAnalisisSeparadas(
		identidades[ports.RolFuenteCobertura],
		identidades[ports.RolVerificadorCobertura],
		identidades[ports.RolPublicadorCatalogoCobertura],
	) {
		t.Fatal("las tres autoridades no quedaron separadas")
	}
	if identidades[ports.RolFuenteCobertura].BackendRef() !=
		backendFuenteCoberturaDesarrolloRef {
		t.Fatal("la fuente no quedó vinculada a su backend gobernado")
	}
	return identidades
}

func solicitudFuenteCoberturaDesarrolloPrueba(
	t *testing.T,
) ports.SolicitudConsultarCobertura {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	comprobacion := domain.ComprobacionExigibleCobertura{
		Clave:       "existe_bolsa_vigente",
		Orden:       1,
		Obligatoria: true,
		Procedencia: domain.ProcedenciaComprobacionCobertura{
			Clave:               "bolsa",
			DefinicionFuenteRef: backendFuenteCoberturaDesarrolloRef,
		},
	}
	catalogo, err := domain.PublicarCatalogoViasCobertura(
		domain.BorradorCatalogoViasCobertura{
			Referencia:  "catalogo:ct:desarrollo:cobertura",
			Version:     1,
			PublicadoEn: ahora.Add(-time.Hour),
			Vigencia: domain.VigenciaCatalogoCobertura{
				Desde: ahora.Add(-time.Hour),
				Hasta: ahora.Add(time.Hour),
			},
			ProcedenciaRef: "gobierno:ct:desarrollo:cobertura",
			Vias: []domain.DefinicionViaCobertura{{
				Clave:          "bolsa_vigente",
				Orden:          1,
				Comprobaciones: []domain.ComprobacionExigibleCobertura{comprobacion},
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return ports.SolicitudConsultarCobertura{
		PeticionRef:       "peticion:ct:desarrollo:cobertura:prueba",
		OrganizacionRef:   organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef:     "expediente:ct:desarrollo:cobertura:prueba",
		VersionExpediente: 1,
		Catalogo:          catalogo.Identidad(),
		ViaClave:          "bolsa_vigente",
		Comprobacion:      comprobacion,
		CategoriaRef:      categoriaAltaContratacionTemporalDesarrollo,
		Periodo: domain.PeriodoPrevisto{
			Inicio: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
			Fin:    time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		SolicitadaEn: ahora,
	}
}

func assertSecretosFuentesCoberturaBorrados(
	t *testing.T,
	dependencias dependenciasFuentesCoberturaDesarrollo,
) {
	t.Helper()
	fuente := dependencias.fuente.(*fuenteComprobacionCoberturaDesarrollo)
	verificador := dependencias.verificador.(*verificadorRespuestaCoberturaDesarrollo)
	publicador := dependencias.publicador.(*publicadorCatalogoCoberturaDesarrollo)
	secretos := [][]byte{
		fuente.privada,
		fuente.claveRespuesta[:],
		fuente.claveRecibo[:],
		verificador.privada,
		verificador.claveRespuesta[:],
		publicador.privada,
	}
	for indice, secreto := range secretos {
		if !bytes.Equal(secreto, make([]byte, len(secreto))) {
			t.Fatalf("el secreto %d no fue borrado", indice)
		}
	}
}
