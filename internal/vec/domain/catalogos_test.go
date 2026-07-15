package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func catalogoConfigurablePrueba() CatalogoConfigurable {
	fecha := time.Date(2026, time.July, 14, 9, 0, 0, 0, time.UTC)
	return CatalogoConfigurable{
		ID:             "bolsa.estados_participacion",
		Version:        1,
		Revision:       1,
		ModuloID:       "bolsa",
		Nombre:         "Estados de participacion en bolsa",
		Descripcion:    "Catalogo administrado por Seleccion Externa.",
		FuenteRef:      "reglamento-bolsas:2026",
		MotivoCreacion: "Configuracion inicial del ciclo de la bolsa",
		Entradas: []EntradaCatalogoConfigurable{
			{
				Clave:        "disponible",
				Etiqueta:     "Disponible",
				Orden:        10,
				VigenteDesde: fecha,
				Atributos:    map[string]string{"color": "verde", "publicable": "si"},
			},
			{
				Clave:        "trabajando",
				Etiqueta:     "Trabajando",
				Orden:        20,
				VigenteDesde: fecha,
				Atributos:    map[string]string{"color": "azul", "publicable": "si"},
			},
		},
		Estado:    EstadoCatalogoBorrador,
		CreadoPor: "tecnico-configuracion-1",
		CreadoEn:  fecha,
	}
}

func TestCatalogoConfigurableCanonizaYAdmiteNuevasOpcionesSinCodigo(t *testing.T) {
	catalogo := catalogoConfigurablePrueba()
	catalogo.Entradas = append(catalogo.Entradas, EntradaCatalogoConfigurable{
		Clave:        "cosa_cuatro",
		Etiqueta:     "Nueva opcion creada desde administracion",
		Orden:        15,
		VigenteDesde: catalogo.CreadoEn,
		Atributos:    map[string]string{"comportamiento_ref": "regla:bolsa:cosa-cuatro:v1"},
	})
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		t.Fatalf("ClonarCanonico() error = %v", err)
	}
	if len(canonico.Entradas) != 3 || canonico.Entradas[1].Clave != "cosa_cuatro" {
		t.Fatalf("catalogo canonico inesperado: %+v", canonico.Entradas)
	}
	huellaPrimera, err := catalogo.HuellaSHA256()
	if err != nil {
		t.Fatalf("HuellaSHA256() error = %v", err)
	}
	reordenado := catalogo
	reordenado.Entradas = []EntradaCatalogoConfigurable{catalogo.Entradas[2], catalogo.Entradas[1], catalogo.Entradas[0]}
	zona := time.FixedZone("EuropaPrueba", 2*60*60)
	reordenado.CreadoEn = reordenado.CreadoEn.In(zona)
	for indice := range reordenado.Entradas {
		reordenado.Entradas[indice].VigenteDesde = reordenado.Entradas[indice].VigenteDesde.In(zona)
	}
	huellaSegunda, err := reordenado.HuellaSHA256()
	if err != nil || huellaPrimera != huellaSegunda {
		t.Fatalf("huella no canonica: primera=%q segunda=%q error=%v", huellaPrimera, huellaSegunda, err)
	}
	canonico.Entradas[0].Atributos["color"] = "alterado"
	if catalogo.Entradas[0].Atributos["color"] != "verde" {
		t.Fatal("el clon comparte atributos con la fuente")
	}
}

func TestCatalogoConfigurablePublicaConDobleControlYResuelveVigencia(t *testing.T) {
	borrador := catalogoConfigurablePrueba()
	if _, err := borrador.Publicar(borrador.CreadoPor, "aprobacion-1", "Revisado", borrador.CreadoEn.Add(time.Hour)); !errors.Is(err, ErrTransicionCatalogoInvalida) {
		t.Fatalf("autopublicacion: error = %v", err)
	}
	publicado, err := borrador.Publicar("responsable-configuracion-2", "aprobacion-1", "Revision funcional superada", borrador.CreadoEn.Add(time.Hour))
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	if publicado.Estado != EstadoCatalogoPublicado || publicado.PublicadoPor == borrador.CreadoPor {
		t.Fatalf("publicacion inesperada: %+v", publicado)
	}
	entrada, err := publicado.ObtenerEntradaVigente("disponible", borrador.CreadoEn.Add(2*time.Hour))
	if err != nil || entrada.Etiqueta != "Disponible" {
		t.Fatalf("ObtenerEntradaVigente() = %+v, %v", entrada, err)
	}
	if _, err := borrador.ObtenerEntradaVigente("disponible", borrador.CreadoEn); !errors.Is(err, ErrCatalogoNoPublicado) {
		t.Fatalf("consulta sobre borrador: error = %v", err)
	}

	limitada := borrador
	limitada.Entradas[0].VigenteDesde = borrador.CreadoEn.Add(2 * time.Hour)
	limitada.Entradas[0].VigenteHasta = borrador.CreadoEn.Add(3 * time.Hour)
	limitadaPublicada, err := limitada.Publicar("responsable-configuracion-2", "aprobacion-2", "Vigencia programada", borrador.CreadoEn.Add(time.Hour))
	if err != nil {
		t.Fatalf("publicar vigencia programada: %v", err)
	}
	if _, err := limitadaPublicada.ObtenerEntradaVigente("disponible", borrador.CreadoEn.Add(90*time.Minute)); !errors.Is(err, ErrEntradaCatalogoNoVigente) {
		t.Fatalf("entrada antes de plazo: error = %v", err)
	}
	if _, err := limitadaPublicada.ObtenerEntradaVigente("disponible", borrador.CreadoEn.Add(2*time.Hour)); err != nil {
		t.Fatalf("entrada al abrir el plazo: %v", err)
	}
	if _, err := limitadaPublicada.ObtenerEntradaVigente("disponible", borrador.CreadoEn.Add(3*time.Hour)); !errors.Is(err, ErrEntradaCatalogoNoVigente) {
		t.Fatalf("el fin debe ser exclusivo: error = %v", err)
	}
}

func TestCatalogoConfigurableNuevaVersionConservaHistoria(t *testing.T) {
	borrador := catalogoConfigurablePrueba()
	publicado, err := borrador.Publicar("responsable-configuracion-2", "aprobacion-1", "Revision superada", borrador.CreadoEn.Add(time.Hour))
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	nueva, err := publicado.NuevaVersion(2, "tecnico-configuracion-3", "resolucion:2026-99", "Incorporar un nuevo estado", publicado.PublicadoEn.Add(time.Hour))
	if err != nil {
		t.Fatalf("NuevaVersion() error = %v", err)
	}
	nueva.Entradas = append(nueva.Entradas, EntradaCatalogoConfigurable{
		Clave:        "disponible_desde_fecha",
		Etiqueta:     "Disponible desde una fecha",
		Orden:        30,
		VigenteDesde: nueva.CreadoEn,
	})
	if err := nueva.Validar(); err != nil {
		t.Fatalf("nueva version editable: %v", err)
	}
	if nueva.VersionAnteriorRef != publicado.Referencia() || len(publicado.Entradas) != 2 || len(nueva.Entradas) != 3 {
		t.Fatalf("versionado incorrecto: anterior=%+v nueva=%+v", publicado, nueva)
	}
	if _, err := publicado.NuevaVersion(4, "tecnico-configuracion-3", "resolucion:2026-99", "Salto", publicado.PublicadoEn.Add(time.Hour)); !errors.Is(err, ErrTransicionCatalogoInvalida) {
		t.Fatalf("salto de version: error = %v", err)
	}
}

func TestCatalogoConfigurableActualizaBorradorConRevisionOptimista(t *testing.T) {
	borrador := catalogoConfigurablePrueba()
	entradas := append([]EntradaCatalogoConfigurable(nil), borrador.Entradas...)
	entradas = append(entradas, EntradaCatalogoConfigurable{
		Clave:        "no_disponible",
		Etiqueta:     "No disponible",
		Orden:        30,
		VigenteDesde: borrador.CreadoEn,
	})
	actualizado, err := borrador.ActualizarBorrador(
		1,
		"tecnico-configuracion-2",
		borrador.Nombre,
		"Catalogo ampliado durante la revision.",
		"reglamento-bolsas:2026",
		"Incorporar estado previsto en el reglamento",
		entradas,
		borrador.CreadoEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("ActualizarBorrador() error = %v", err)
	}
	if actualizado.Revision != 2 || len(actualizado.Entradas) != 3 || actualizado.UltimaModificacionPor == "" {
		t.Fatalf("actualizacion inesperada: %+v", actualizado)
	}
	if _, err := actualizado.ActualizarBorrador(
		1,
		"tecnico-configuracion-3",
		actualizado.Nombre,
		actualizado.Descripcion,
		actualizado.FuenteRef,
		"Escritura con revision obsoleta",
		actualizado.Entradas,
		actualizado.UltimaModificacionEn.Add(time.Minute),
	); !errors.Is(err, ErrTransicionCatalogoInvalida) {
		t.Fatalf("revision obsoleta: error = %v", err)
	}
	if _, err := actualizado.Publicar(
		actualizado.UltimaModificacionPor,
		"aprobacion-actualizada",
		"Autopublicacion del ultimo modificador",
		actualizado.UltimaModificacionEn.Add(time.Hour),
	); !errors.Is(err, ErrTransicionCatalogoInvalida) {
		t.Fatalf("publicacion por el ultimo modificador: error = %v", err)
	}
}

func TestCatalogoConfigurableRechazaDuplicadosExcesoYRetiradaSinDobleControl(t *testing.T) {
	catalogo := catalogoConfigurablePrueba()
	catalogo.Entradas = append(catalogo.Entradas, catalogo.Entradas[0])
	if err := catalogo.Validar(); !errors.Is(err, ErrEntradaCatalogoDuplicada) {
		t.Fatalf("entrada duplicada: error = %v", err)
	}
	catalogo = catalogoConfigurablePrueba()
	catalogo.Entradas[0].Atributos["texto"] = strings.Repeat("x", maximoCaracteresAtributo+1)
	if err := catalogo.Validar(); !errors.Is(err, ErrEntradaCatalogoInvalida) {
		t.Fatalf("atributo excesivo: error = %v", err)
	}

	publicado, err := catalogoConfigurablePrueba().Publicar("responsable-configuracion-2", "aprobacion-1", "Revision superada", catalogo.CreadoEn.Add(time.Hour))
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	if _, err := publicado.Retirar(publicado.PublicadoPor, "retirada-1", "Sustituido", publicado.PublicadoEn.Add(time.Hour)); !errors.Is(err, ErrTransicionCatalogoInvalida) {
		t.Fatalf("autorretirada: error = %v", err)
	}
	retirado, err := publicado.Retirar("responsable-configuracion-4", "retirada-1", "Sustituido por nueva version", publicado.PublicadoEn.Add(time.Hour))
	if err != nil || retirado.Estado != EstadoCatalogoRetirado {
		t.Fatalf("Retirar() = %+v, %v", retirado, err)
	}
	if _, err := retirado.ObtenerEntradaVigente("disponible", retirado.RetiradoEn); !errors.Is(err, ErrCatalogoNoPublicado) {
		t.Fatalf("catalogo retirado consultable: error = %v", err)
	}
}

func TestHuellaContenidoCatalogoPermaneceDuranteGobiernoYCambiaConSemantica(t *testing.T) {
	borrador := catalogoConfigurablePrueba()
	huellaContenidoBorrador, err := borrador.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella de contenido borrador: %v", err)
	}
	huellaGobiernoBorrador, err := borrador.HuellaSHA256()
	if err != nil {
		t.Fatalf("huella de gobierno borrador: %v", err)
	}
	publicado, err := borrador.Publicar(
		"responsable-configuracion-2",
		"aprobacion-huella-1",
		"Catalogo revisado",
		borrador.CreadoEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	huellaContenidoPublicado, _ := publicado.HuellaContenidoSHA256()
	huellaGobiernoPublicado, _ := publicado.HuellaSHA256()
	retirado, err := publicado.Retirar(
		"responsable-configuracion-3",
		"aprobacion-huella-2",
		"Sustituido por una nueva version",
		publicado.PublicadoEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("Retirar() error = %v", err)
	}
	huellaContenidoRetirado, _ := retirado.HuellaContenidoSHA256()
	huellaGobiernoRetirado, _ := retirado.HuellaSHA256()
	if huellaContenidoBorrador != huellaContenidoPublicado || huellaContenidoPublicado != huellaContenidoRetirado {
		t.Fatalf("la huella semantica cambio con el gobierno: %q %q %q",
			huellaContenidoBorrador, huellaContenidoPublicado, huellaContenidoRetirado)
	}
	if huellaGobiernoBorrador == huellaGobiernoPublicado || huellaGobiernoPublicado == huellaGobiernoRetirado {
		t.Fatalf("la huella de gobierno no cambio: %q %q %q",
			huellaGobiernoBorrador, huellaGobiernoPublicado, huellaGobiernoRetirado)
	}

	actualizado, err := borrador.ActualizarBorrador(
		1,
		"tecnico-configuracion-4",
		borrador.Nombre,
		"Descripcion semanticamente distinta.",
		borrador.FuenteRef,
		"Cambiar el significado documentado",
		borrador.Entradas,
		borrador.CreadoEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("ActualizarBorrador() error = %v", err)
	}
	huellaContenidoActualizado, err := actualizado.HuellaContenidoSHA256()
	if err != nil || huellaContenidoActualizado == huellaContenidoBorrador {
		t.Fatalf("el cambio semantico no altero la huella: %q, %v", huellaContenidoActualizado, err)
	}
}
