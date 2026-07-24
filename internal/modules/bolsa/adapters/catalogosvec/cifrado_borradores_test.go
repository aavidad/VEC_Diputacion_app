package catalogosvec

import (
	"context"
	"errors"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type fuenteCifradoBorradoresPrueba struct {
	catalogo dominiovec.CatalogoConfigurable
	err      error
}

func (f *fuenteCifradoBorradoresPrueba) ObtenerCatalogo(
	ctx context.Context, id string, version int,
) (dominiovec.CatalogoConfigurable, error) {
	if err := ctx.Err(); err != nil {
		return dominiovec.CatalogoConfigurable{}, err
	}
	if f.err != nil {
		return dominiovec.CatalogoConfigurable{}, f.err
	}
	if f.catalogo.ID != id || f.catalogo.Version != version {
		return dominiovec.CatalogoConfigurable{}, puertosvec.ErrCatalogoNoEncontrado
	}
	return f.catalogo.ClonarCanonico()
}

func (f *fuenteCifradoBorradoresPrueba) ListarVersionesCatalogo(
	ctx context.Context, id string,
) ([]dominiovec.CatalogoConfigurable, error) {
	catalogo, err := f.ObtenerCatalogo(ctx, id, f.catalogo.Version)
	if err != nil {
		return nil, err
	}
	return []dominiovec.CatalogoConfigurable{catalogo}, nil
}

func TestCatalogoCifradoResuelveAccionesSinListaCompiladaDeEntradas(t *testing.T) {
	instante := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)
	catalogo := catalogoCifradoBorradoresPrueba(t, instante)
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	autoridad, err := NuevaAutoridadPoliticaCifradoBorradoresCatalogo(
		&fuenteCifradoBorradoresPrueba{catalogo: catalogo},
		configuracionCatalogoCifradoPrueba(huella, "politicas"),
	)
	if err != nil {
		t.Fatal(err)
	}

	alta, err := autoridad.catalogo.resolver(
		context.Background(), puertosbolsa.AccionCrearBorradorConvocatoria, instante,
	)
	if err != nil || alta.perfil.AlgoritmoAEAD != "A256GCM" ||
		alta.decisionRef != "decision:politica-cifrado:borradores:crear:v1" {
		t.Fatalf("resolucion de alta inesperada: %+v / %v", alta, err)
	}
	actualizacion, err := autoridad.catalogo.resolver(
		context.Background(), puertosbolsa.AccionActualizarBorradorConvocatoria, instante,
	)
	if err != nil || actualizacion.entrada.Clave != "actualizar-borrador" ||
		actualizacion.perfil.HuellaContenidoSHA256 != alta.perfil.HuellaContenidoSHA256 {
		t.Fatalf("resolucion de actualizacion inesperada: %+v / %v", actualizacion, err)
	}
	if autoridad.IdentidadAutoridadBorrador().ProveedorRef == "" {
		t.Fatal("la autoridad no conservo su identidad tecnica")
	}
}

func TestCatalogoCifradoFallaCerradoAnteAmbiguedadEsquemaOHuella(t *testing.T) {
	instante := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)
	base := catalogoCifradoBorradoresPrueba(t, instante)
	huella, err := base.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}

	pruebas := []struct {
		nombre string
		mutar  func(*dominiovec.CatalogoConfigurable)
	}{
		{"entrada duplicada para accion", func(c *dominiovec.CatalogoConfigurable) {
			duplicada := c.Entradas[0]
			duplicada.Clave = "crear-borrador-alternativo"
			duplicada.Orden = 3
			c.Entradas = append(c.Entradas, duplicada)
		}},
		{"atributo no contratado", func(c *dominiovec.CatalogoConfigurable) {
			c.Entradas[0].Atributos["algoritmo_reserva"] = "no-aprobado"
		}},
		{"contenido cambiado tras fijar huella", func(c *dominiovec.CatalogoConfigurable) {
			c.Entradas[0].Atributos["algoritmo_aead"] = "A128GCM"
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			alterado, err := base.ClonarCanonico()
			if err != nil {
				t.Fatal(err)
			}
			prueba.mutar(&alterado)
			autoridad, err := NuevaAutoridadPoliticaCifradoBorradoresCatalogo(
				&fuenteCifradoBorradoresPrueba{catalogo: alterado},
				configuracionCatalogoCifradoPrueba(huella, "politicas"),
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = autoridad.catalogo.resolver(
				context.Background(), puertosbolsa.AccionCrearBorradorConvocatoria, instante,
			)
			if !errors.Is(err, ErrCatalogoCifradoBorradoresNoDisponible) {
				t.Fatalf("alteracion aceptada: %v", err)
			}
		})
	}
}

func TestCatalogoCifradoRechazaConfiguracionNulaYPropagaCancelacion(t *testing.T) {
	var fuenteNula *fuenteCifradoBorradoresPrueba
	if autoridad, err := NuevaAutoridadPoliticaCifradoBorradoresCatalogo(
		fuenteNula, configuracionCatalogoCifradoPrueba("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "politicas"),
	); autoridad != nil || !errors.Is(err, ErrCatalogoCifradoBorradoresNoDisponible) {
		t.Fatalf("fuente tipada nula aceptada: %v", err)
	}

	instante := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)
	catalogo := catalogoCifradoBorradoresPrueba(t, instante)
	huella, _ := catalogo.HuellaSHA256()
	resolvedor, err := NuevoResolvedorPerfilCifradoBorradoresCatalogo(
		&fuenteCifradoBorradoresPrueba{catalogo: catalogo},
		configuracionCatalogoCifradoPrueba(huella, "perfiles"),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err = resolvedor.catalogo.resolver(
		ctx, puertosbolsa.AccionCrearBorradorConvocatoria, instante,
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no cerro la consulta: %v", err)
	}
}

func TestCatalogoCifradoExigePublicacionYVigenciaExactas(t *testing.T) {
	instante := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)
	base := catalogoCifradoBorradoresPrueba(t, instante)

	pruebas := []struct {
		nombre string
		mutar  func(*dominiovec.CatalogoConfigurable)
	}{
		{"catalogo aun no publicado", func(c *dominiovec.CatalogoConfigurable) {
			c.PublicadoEn = instante.Add(time.Microsecond)
		}},
		{"entrada aun no vigente", func(c *dominiovec.CatalogoConfigurable) {
			c.Entradas[0].VigenteDesde = instante.Add(time.Microsecond)
		}},
		{"entrada ya vencida", func(c *dominiovec.CatalogoConfigurable) {
			c.Entradas[0].VigenteHasta = instante
		}},
		{"catalogo de otro modulo", func(c *dominiovec.CatalogoConfigurable) {
			c.ModuloID = "personal"
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			alterado, err := base.ClonarCanonico()
			if err != nil {
				t.Fatal(err)
			}
			prueba.mutar(&alterado)
			huella, err := alterado.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			autoridad, err := NuevaAutoridadPoliticaCifradoBorradoresCatalogo(
				&fuenteCifradoBorradoresPrueba{catalogo: alterado},
				configuracionCatalogoCifradoPrueba(huella, "politicas"),
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = autoridad.catalogo.resolver(
				context.Background(), puertosbolsa.AccionCrearBorradorConvocatoria, instante,
			)
			if !errors.Is(err, ErrCatalogoCifradoBorradoresNoDisponible) {
				t.Fatalf("catalogo fuera de gobierno aceptado: %v", err)
			}
		})
	}
}

func TestCatalogoCifradoNoComparteInstantaneaMutableConLaFuente(t *testing.T) {
	instante := time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC)
	catalogo := catalogoCifradoBorradoresPrueba(t, instante)
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteCifradoBorradoresPrueba{catalogo: catalogo}
	autoridad, err := NuevaAutoridadPoliticaCifradoBorradoresCatalogo(
		fuente, configuracionCatalogoCifradoPrueba(huella, "politicas"),
	)
	if err != nil {
		t.Fatal(err)
	}
	primera, err := autoridad.catalogo.resolver(
		context.Background(), puertosbolsa.AccionCrearBorradorConvocatoria, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	primera.entrada.Atributos["algoritmo_aead"] = "ALTERADO"
	segunda, err := autoridad.catalogo.resolver(
		context.Background(), puertosbolsa.AccionCrearBorradorConvocatoria, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	if segunda.perfil.AlgoritmoAEAD != "A256GCM" ||
		segunda.entrada.Atributos["algoritmo_aead"] != "A256GCM" {
		t.Fatal("una salida pudo mutar la instantanea gobernada de la fuente")
	}
}

func catalogoCifradoBorradoresPrueba(
	t *testing.T, instante time.Time,
) dominiovec.CatalogoConfigurable {
	t.Helper()
	entrada := func(clave, accion, decision string, orden int) dominiovec.EntradaCatalogoConfigurable {
		return dominiovec.EntradaCatalogoConfigurable{
			Clave: clave, Etiqueta: clave, Orden: orden, VigenteDesde: instante.Add(-time.Hour),
			Atributos: map[string]string{
				"accion": accion, "algoritmo_aead": "A256GCM",
				"algoritmo_envoltura_clave": "A256KW",
				"autoridad_politica_ref":    "autoridad:politicas-cifrado:borradores:v1",
				"decision_politica_ref":     decision, "decision_politica_version": "1",
				"evidencia_perfil_ref":     "evidencia:perfil-cifrado:borradores:v1",
				"evidencia_perfil_version": "1",
				"perfil_referencia":        "perfil:cifrado:borradores:a256gcm:v1", "perfil_version": "1",
				"verificador_perfil_ref": "verificador:perfil-cifrado:borradores:v1",
			},
		}
	}
	catalogo := dominiovec.CatalogoConfigurable{
		ID: "politicas-cifrado-borradores", Version: 1, Revision: 1, ModuloID: "bolsa",
		Nombre: "Politicas de cifrado de borradores", FuenteRef: "fuente:seguridad:vec:v1",
		MotivoCreacion: "Gobernar algoritmos y perfiles sin recompilar la aplicacion.",
		Entradas: []dominiovec.EntradaCatalogoConfigurable{
			entrada("crear-borrador", puertosbolsa.AccionCrearBorradorConvocatoria,
				"decision:politica-cifrado:borradores:crear:v1", 1),
			entrada("actualizar-borrador", puertosbolsa.AccionActualizarBorradorConvocatoria,
				"decision:politica-cifrado:borradores:actualizar:v1", 2),
		},
		Estado: dominiovec.EstadoCatalogoPublicado, CreadoPor: "sistema:seguridad:catalogos",
		CreadoEn: instante.Add(-2 * time.Hour), PublicadoPor: "sistema:seguridad:aprobador",
		PublicadoEn: instante.Add(-time.Hour), AprobacionRef: "aprobacion:seguridad:catalogo:v1",
		MotivoPublicacion: "Activar el perfil revisado para borradores de convocatorias.",
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	return canonico
}

func configuracionCatalogoCifradoPrueba(
	huella, responsabilidad string,
) ConfiguracionCatalogoCifradoBorradores {
	return ConfiguracionCatalogoCifradoBorradores{
		CatalogoID: "politicas-cifrado-borradores", Version: 1, HuellaSHA256: huella,
		ProveedorRef:  "proveedor:catalogos:" + responsabilidad,
		InstanciaRef:  "instancia:catalogos:" + responsabilidad,
		CredencialRef: "credencial:catalogos:" + responsabilidad,
		RolRef:        "rol:catalogos:" + responsabilidad,
	}
}
